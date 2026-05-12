package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/veona/agent/internal/buffer"
	"github.com/veona/agent/internal/collector"
	"github.com/veona/agent/internal/config"
	"github.com/veona/agent/internal/dispatcher"
)

func main() {
	// Configure slog to use JSON format
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 1. Configuration parsing
	configPath := flag.String("config", "/etc/veona/config.yaml", "Path to config file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		slog.Error("Failed to load configuration", "path", *configPath, "error", err)
		os.Exit(1)
	}

	if cfg.Server.Token == "" {
		slog.Error("Server token is missing in config file")
		os.Exit(1)
	}

	slog.Info("Starting Veona Agent...", "server", cfg.Server.URL)
	slog.Info("Enabled collectors",
		"cpu", cfg.Collectors.CPU.Interval,
		"mem", cfg.Collectors.Mem.Interval,
		"disk", cfg.Collectors.Disk.Interval,
		"disk_auto", cfg.Collectors.Disk.AutoDiscover)

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 2. Initialize the Ring Buffer
	metricBuffer := buffer.NewRingBuffer(cfg.Buffer.Size)

	// 3. Initialize the HTTP Dispatcher
	httpDispatcher := dispatcher.NewHTTPDispatcher(cfg.Server.URL, cfg.Server.Token)

	// 4. Start Goroutines (Producer / Consumer)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		slog.Info("Starting Dispatcher Goroutine...")
		httpDispatcher.Run(ctx, metricBuffer)
	}()

	// 5. Collector Goroutines setup using global config
	agentCollector := collector.NewCollector(collector.Config{
		Global: *cfg,
	})

	wg.Add(1)
	go func() {
		defer wg.Done()
		slog.Info("Starting Collector Goroutines...")
		agentCollector.Run(ctx, metricBuffer)
	}()

	// Wait for interrupt
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down Veona Agent gracefully...")
	cancel()
	wg.Wait()
	slog.Info("Agent shut down successfully.")
}
