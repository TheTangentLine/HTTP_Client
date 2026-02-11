package stats

import (
	"sync/atomic"
	"time"
)

// Snapshot represents a point-in-time view of collected metrics.
type Snapshot struct {
	TotalRequests uint64
	Successes     uint64
	Errors        uint64
	LatencyP50    time.Duration
	LatencyP95    time.Duration
	LatencyP99    time.Duration
	RequestsPerS  float64
}

// Collector aggregates metrics from workers in a thread-safe way.
type Collector struct {
	startTime time.Time

	totalRequests uint64
	successes     uint64
	errors        uint64
	// simple rolling latency stats (could be made more precise later)
	totalLatencyNanos uint64
}

// NewCollector creates a new Collector instance.
func NewCollector() *Collector {
	return &Collector{
		startTime: time.Now(),
	}
}

// Record records the outcome of a single request.
func (c *Collector) Record(latency time.Duration, success bool) {
	atomic.AddUint64(&c.totalRequests, 1)
	atomic.AddUint64(&c.totalLatencyNanos, uint64(latency.Nanoseconds()))
	if success {
		atomic.AddUint64(&c.successes, 1)
	} else {
		atomic.AddUint64(&c.errors, 1)
	}
}

// Snapshot returns a coarse snapshot of metrics.
func (c *Collector) Snapshot() Snapshot {
	elapsed := time.Since(c.startTime).Seconds()
	if elapsed == 0 {
		elapsed = 1
	}

	total := atomic.LoadUint64(&c.totalRequests)
	totalLatency := atomic.LoadUint64(&c.totalLatencyNanos)

	avgLatency := time.Duration(0)
	if total > 0 {
		avgLatency = time.Duration(int64(totalLatency / total))
	}

	return Snapshot{
		TotalRequests: total,
		Successes:     atomic.LoadUint64(&c.successes),
		Errors:        atomic.LoadUint64(&c.errors),
		// For now we approximate all percentiles by the average latency.
		LatencyP50:   avgLatency,
		LatencyP95:   avgLatency,
		LatencyP99:   avgLatency,
		RequestsPerS: float64(total) / elapsed,
	}
}

