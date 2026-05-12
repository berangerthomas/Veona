package buffer

import (
	"testing"
)

func TestRingBuffer_PushPop(t *testing.T) {
	size := 3
	rb := NewRingBuffer(size)

	// Push 2 items
	rb.Push(MetricPayload{Timestamp: 1})
	rb.Push(MetricPayload{Timestamp: 2})

	if rb.count != 2 {
		t.Errorf("Expected count 2, got %d", rb.count)
	}

	// Pop all
	metrics := rb.PopAll()
	if len(metrics) != 2 {
		t.Errorf("Expected 2 metrics, got %d", len(metrics))
	}
	if rb.count != 0 {
		t.Errorf("Expected count 0 after PopAll, got %d", rb.count)
	}
}

func TestRingBuffer_Overflow(t *testing.T) {
	size := 2
	rb := NewRingBuffer(size)

	rb.Push(MetricPayload{Timestamp: 1})
	rb.Push(MetricPayload{Timestamp: 2})
	rb.Push(MetricPayload{Timestamp: 3}) // Should overwrite 1

	if rb.count != 2 {
		t.Errorf("Expected count 2, got %d", rb.count)
	}

	metrics := rb.PopAll()
	if len(metrics) != 2 {
		t.Errorf("Expected 2 metrics, got %d", len(metrics))
	}
	if metrics[0].Timestamp != 2 {
		t.Errorf("Expected first metric to be 2, got %d", metrics[0].Timestamp)
	}
	if metrics[1].Timestamp != 3 {
		t.Errorf("Expected second metric to be 3, got %d", metrics[1].Timestamp)
	}
}

func TestRingBuffer_Empty(t *testing.T) {
	rb := NewRingBuffer(5)
	metrics := rb.PopAll()
	if metrics == nil {
		t.Errorf("Expected empty slice, got nil when popping from empty buffer")
	}
	if len(metrics) != 0 {
		t.Errorf("Expected length 0 when popping from empty buffer, got %d", len(metrics))
	}
}
