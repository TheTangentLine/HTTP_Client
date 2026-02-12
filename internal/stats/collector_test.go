package stats

import (
	"sync"
	"testing"
	"time"
)

func TestNewCollector(t *testing.T) {
	c := NewCollector()
	if c == nil {
		t.Fatal("NewCollector returned nil")
	}
	snap := c.Snapshot()
	if snap.TotalRequests != 0 || snap.Successes != 0 || snap.Errors != 0 {
		t.Errorf("fresh snapshot should be zero: %+v", snap)
	}
}

func TestRecord_SuccessAndError(t *testing.T) {
	c := NewCollector()
	c.Record(10*time.Millisecond, true, 0, 100)
	c.Record(20*time.Millisecond, false, 50, 0)
	snap := c.Snapshot()
	if snap.TotalRequests != 2 {
		t.Errorf("TotalRequests: got %d, want 2", snap.TotalRequests)
	}
	if snap.Successes != 1 {
		t.Errorf("Successes: got %d, want 1", snap.Successes)
	}
	if snap.Errors != 1 {
		t.Errorf("Errors: got %d, want 1", snap.Errors)
	}
	if snap.TotalBytesSent != 50 {
		t.Errorf("TotalBytesSent: got %d, want 50", snap.TotalBytesSent)
	}
	if snap.TotalBytesRecv != 100 {
		t.Errorf("TotalBytesRecv: got %d, want 100", snap.TotalBytesRecv)
	}
}

func TestRecord_Concurrent(t *testing.T) {
	c := NewCollector()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Record(time.Millisecond, true, 1, 1)
		}()
	}
	wg.Wait()
	snap := c.Snapshot()
	if snap.TotalRequests != 100 {
		t.Errorf("TotalRequests: got %d, want 100", snap.TotalRequests)
	}
	if snap.Successes != 100 {
		t.Errorf("Successes: got %d, want 100", snap.Successes)
	}
	if snap.TotalBytesSent != 100 || snap.TotalBytesRecv != 100 {
		t.Errorf("bytes: sent=%d recv=%d", snap.TotalBytesSent, snap.TotalBytesRecv)
	}
}

func TestSnapshot_LatencyPercentiles(t *testing.T) {
	c := NewCollector()
	durations := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		30 * time.Millisecond,
		40 * time.Millisecond,
		50 * time.Millisecond,
	}
	for _, d := range durations {
		c.Record(d, true, 0, 0)
	}
	snap := c.Snapshot()
	if snap.LatencyP50 != 30*time.Millisecond {
		t.Errorf("LatencyP50: got %v, want 30ms", snap.LatencyP50)
	}
	if snap.LatencyMax != 50*time.Millisecond {
		t.Errorf("LatencyMax: got %v, want 50ms", snap.LatencyMax)
	}
	if snap.LatencyAvg == 0 {
		t.Error("LatencyAvg should be non-zero")
	}
}

func TestSnapshot_EmptyCollector(t *testing.T) {
	c := NewCollector()
	snap := c.Snapshot()
	if snap.TotalRequests != 0 {
		t.Errorf("TotalRequests: got %d", snap.TotalRequests)
	}
	if snap.LatencyP50 != 0 || snap.LatencyMax != 0 {
		t.Errorf("latency should be zero: P50=%v Max=%v", snap.LatencyP50, snap.LatencyMax)
	}
}
