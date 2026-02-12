package stats

import (
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

const maxLatencySamples = 50000
const maxBucketSamples = 600 // ~10 min at 1s buckets

// Snapshot represents a point-in-time view of collected metrics.
type Snapshot struct {
	TotalRequests   uint64
	Successes       uint64
	Errors          uint64
	TotalBytesSent  uint64
	TotalBytesRecv  uint64
	Duration        time.Duration
	RequestsPerSAvg float64
	BytesPerSAvg    float64

	// Latency (ms) – percentiles and stats
	LatencyP25  time.Duration
	LatencyP50  time.Duration
	LatencyP975 time.Duration
	LatencyP99  time.Duration
	LatencyAvg  time.Duration
	LatencyStdev time.Duration
	LatencyMax  time.Duration

	// Throughput (Req/Sec and Bytes/Sec) – percentiles from 1s buckets
	RPSP01   float64
	RPSP025  float64
	RPSP50   float64
	RPSP975  float64
	RPSStdev float64
	RPSMin   float64

	BytesPerSP01   float64
	BytesPerSP025  float64
	BytesPerSP50   float64
	BytesPerSP975  float64
	BytesPerSStdev float64
	BytesPerSMin   float64
}

// Collector aggregates metrics from workers in a thread-safe way.
type Collector struct {
	startTime time.Time

	totalRequests   uint64
	successes      uint64
	errors         uint64
	totalBytesSent uint64
	totalBytesRecv uint64

	mu           sync.Mutex
	latencySamples []time.Duration
	lastBucketTime  time.Time
	lastBucketReqs  uint64
	lastBucketSent  uint64
	lastBucketRecv  uint64
	rpsBuckets      []float64
	bytesPerSBuckets []float64
}

// NewCollector creates a new Collector instance.
func NewCollector() *Collector {
	return &Collector{
		startTime:     time.Now(),
		lastBucketTime: time.Now(),
		latencySamples: make([]time.Duration, 0, maxLatencySamples),
		rpsBuckets:      make([]float64, 0, maxBucketSamples),
		bytesPerSBuckets: make([]float64, 0, maxBucketSamples),
	}
}

// Record records the outcome of a single request and bytes sent/received.
func (c *Collector) Record(latency time.Duration, success bool, bytesSent, bytesRecv uint64) {
	atomic.AddUint64(&c.totalRequests, 1)
	atomic.AddUint64(&c.totalBytesSent, bytesSent)
	atomic.AddUint64(&c.totalBytesRecv, bytesRecv)
	if success {
		atomic.AddUint64(&c.successes, 1)
	} else {
		atomic.AddUint64(&c.errors, 1)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.latencySamples) < maxLatencySamples {
		c.latencySamples = append(c.latencySamples, latency)
	}
}

func percentileDuration(s []time.Duration, p float64) time.Duration {
	if len(s) == 0 {
		return 0
	}
	idx := int(math.Round(p / 100 * float64(len(s)-1)))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(s) {
		idx = len(s) - 1
	}
	return s[idx]
}

func percentileFloat(s []float64, p float64) float64 {
	if len(s) == 0 {
		return 0
	}
	idx := int(math.Round(p / 100 * float64(len(s)-1)))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(s) {
		idx = len(s) - 1
	}
	return s[idx]
}

func avgStdevDuration(s []time.Duration) (avg, stdev time.Duration) {
	if len(s) == 0 {
		return 0, 0
	}
	var sum int64
	for _, d := range s {
		sum += d.Nanoseconds()
	}
	avg = time.Duration(sum / int64(len(s)))
	var varSum float64
	for _, d := range s {
		diff := float64(d.Nanoseconds() - avg.Nanoseconds())
		varSum += diff * diff
	}
	stdev = time.Duration(int64(math.Sqrt(varSum / float64(len(s)))))
	return avg, stdev
}

func avgStdevMinFloat(s []float64) (avg, stdev, min float64) {
	if len(s) == 0 {
		return 0, 0, 0
	}
	var sum float64
	min = s[0]
	for _, v := range s {
		sum += v
		if v < min {
			min = v
		}
	}
	avg = sum / float64(len(s))
	var varSum float64
	for _, v := range s {
		diff := v - avg
		varSum += diff * diff
	}
	stdev = math.Sqrt(varSum / float64(len(s)))
	return avg, stdev, min
}

// Snapshot returns a full snapshot including percentiles and throughput buckets.
func (c *Collector) Snapshot() Snapshot {
	elapsed := time.Since(c.startTime)
	elapsedSec := elapsed.Seconds()
	if elapsedSec < 0.001 {
		elapsedSec = 0.001
	}

	totalReqs := atomic.LoadUint64(&c.totalRequests)
	totalSent := atomic.LoadUint64(&c.totalBytesSent)
	totalRecv := atomic.LoadUint64(&c.totalBytesRecv)

	c.mu.Lock()
	// Flush a 1s bucket if enough time has passed
	now := time.Now()
	if now.Sub(c.lastBucketTime) >= time.Second && totalReqs > c.lastBucketReqs {
		reqDelta := totalReqs - c.lastBucketReqs
		sentDelta := totalSent - c.lastBucketSent
		recvDelta := totalRecv - c.lastBucketRecv
		secs := now.Sub(c.lastBucketTime).Seconds()
		if secs > 0 {
			c.rpsBuckets = append(c.rpsBuckets, float64(reqDelta)/secs)
			bytesPerS := float64(sentDelta+recvDelta) / secs
			c.bytesPerSBuckets = append(c.bytesPerSBuckets, bytesPerS)
			if len(c.rpsBuckets) > maxBucketSamples {
				c.rpsBuckets = c.rpsBuckets[1:]
				c.bytesPerSBuckets = c.bytesPerSBuckets[1:]
			}
		}
		c.lastBucketTime = now
		c.lastBucketReqs = totalReqs
		c.lastBucketSent = totalSent
		c.lastBucketRecv = totalRecv
	}

	latencySamples := make([]time.Duration, len(c.latencySamples))
	copy(latencySamples, c.latencySamples)
	rpsBuckets := make([]float64, len(c.rpsBuckets))
	copy(rpsBuckets, c.rpsBuckets)
	bytesBuckets := make([]float64, len(c.bytesPerSBuckets))
	copy(bytesBuckets, c.bytesPerSBuckets)
	c.mu.Unlock()

	snap := Snapshot{
		TotalRequests:   totalReqs,
		Successes:       atomic.LoadUint64(&c.successes),
		Errors:          atomic.LoadUint64(&c.errors),
		TotalBytesSent:  totalSent,
		TotalBytesRecv:  totalRecv,
		Duration:        elapsed,
		RequestsPerSAvg: float64(totalReqs) / elapsedSec,
		BytesPerSAvg:    float64(totalSent+totalRecv) / elapsedSec,
	}

	if len(latencySamples) > 0 {
		sort.Slice(latencySamples, func(i, j int) bool { return latencySamples[i] < latencySamples[j] })
		snap.LatencyP25 = percentileDuration(latencySamples, 2.5)
		snap.LatencyP50 = percentileDuration(latencySamples, 50)
		snap.LatencyP975 = percentileDuration(latencySamples, 97.5)
		snap.LatencyP99 = percentileDuration(latencySamples, 99)
		snap.LatencyAvg, snap.LatencyStdev = avgStdevDuration(latencySamples)
		snap.LatencyMax = latencySamples[len(latencySamples)-1]
	}

	if len(rpsBuckets) > 0 {
		sort.Float64s(rpsBuckets)
		snap.RPSP01, snap.RPSP025, snap.RPSP50, snap.RPSP975 = percentileFloat(rpsBuckets, 1), percentileFloat(rpsBuckets, 2.5), percentileFloat(rpsBuckets, 50), percentileFloat(rpsBuckets, 97.5)
		snap.RPSStdev = 0
		_, snap.RPSStdev, snap.RPSMin = avgStdevMinFloat(rpsBuckets)
	}
	if len(bytesBuckets) > 0 {
		sort.Float64s(bytesBuckets)
		snap.BytesPerSP01, snap.BytesPerSP025, snap.BytesPerSP50, snap.BytesPerSP975 = percentileFloat(bytesBuckets, 1), percentileFloat(bytesBuckets, 2.5), percentileFloat(bytesBuckets, 50), percentileFloat(bytesBuckets, 97.5)
		_, snap.BytesPerSStdev, snap.BytesPerSMin = avgStdevMinFloat(bytesBuckets)
	}

	return snap
}
