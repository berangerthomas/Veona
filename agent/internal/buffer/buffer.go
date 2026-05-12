package buffer

import (
	"sync"
)

// MetricPayload represents the JSON payload to send
type MetricPayload struct {
	Timestamp int64                  `json:"timestamp"`
	Metrics   map[string]interface{} `json:"metrics"`
}

// RingBuffer provides a thread-safe, fixed-size queue that overwrites oldest data when full
type RingBuffer struct {
	data  []MetricPayload
	size  int
	head  int
	tail  int
	count int
	mu    sync.Mutex
}

func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		data: make([]MetricPayload, size),
		size: size,
	}
}

// Push adds a new metric to the buffer. Overwrites the oldest if full.
func (b *RingBuffer) Push(item MetricPayload) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.data[b.head] = item
	b.head = (b.head + 1) % b.size

	if b.count < b.size {
		b.count++
	} else {
		// Overwrite oldest data
		b.tail = (b.tail + 1) % b.size
	}
}

// PopAll extracts all current metrics from the buffer to send them as a batch.
func (b *RingBuffer) PopAll() []MetricPayload {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.count == 0 {
		return []MetricPayload{}
	}

	result := make([]MetricPayload, 0, b.count)
	for i := 0; i < b.count; i++ {
		idx := (b.tail + i) % b.size
		result = append(result, b.data[idx])
	}

	// Reset buffer
	b.head = 0
	b.tail = 0
	b.count = 0

	return result
}
