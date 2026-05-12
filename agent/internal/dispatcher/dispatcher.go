package dispatcher

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/veona/agent/internal/buffer"
)

type HTTPDispatcher struct {
	serverURL string
	token     string
	client    *http.Client
}

func NewHTTPDispatcher(serverURL string, token string) *HTTPDispatcher {
	return &HTTPDispatcher{
		serverURL: serverURL,
		token:     token,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Run checks the buffer periodically and flushes it to the server
func (d *HTTPDispatcher) Run(ctx context.Context, buf *buffer.RingBuffer) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Dispatcher stopped.")
			return
		case <-ticker.C:
			metrics := buf.PopAll()
			if len(metrics) > 0 {
				d.sendWithRetry(ctx, metrics, buf)
			}
		}
	}
}

func (d *HTTPDispatcher) sendWithRetry(ctx context.Context, metrics []buffer.MetricPayload, buf *buffer.RingBuffer) {
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		err := d.sendBatch(metrics)
		if err == nil {
			slog.Info("Successfully sent metrics", "count", len(metrics))
			return
		}

		if i == maxRetries-1 {
			slog.Error("Failed to send metrics after max retries, dropping batch to avoid infinite loop", "error", err, "count", len(metrics))
			return
		}

		backoff := time.Duration(1<<i) * time.Second
		slog.Warn("Failed to send metrics, retrying...", "retry", i+1, "backoff", backoff, "error", err)

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
	}
}

func (d *HTTPDispatcher) sendBatch(metrics []buffer.MetricPayload) error {
	jsonData, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	// GZIP Compression
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	if _, err := gz.Write(jsonData); err != nil {
		return fmt.Errorf("gzip write error: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("gzip close error: %w", err)
	}

	req, err := http.NewRequest("POST", d.serverURL, &b)
	if err != nil {
		return fmt.Errorf("request creation error: %w", err)
	}

	// Required Headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Authorization", "Bearer "+d.token)

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("client do error: %w", err)
	}
	defer resp.Body.Close()

	// Drain body to reuse connection
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("server returned error status: %d", resp.StatusCode)
	}

	return nil
}
