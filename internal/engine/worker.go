package engine

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/thetangentline/httpcl/internal/stats"
)

// worker runs as one "process": it spawns cfg.Pipeline goroutines (one per pipeline
// slot) so that many requests are in flight concurrently per worker. durationDone
// is closed when the benchmark duration ends; workers stop starting new requests
// but let in-flight requests complete. ctx is cancelled on SIGINT to abort immediately.
func worker(
	ctx context.Context,
	durationDone <-chan struct{},
	client *http.Client,
	cfg Config,
	connections int,
	collector *stats.Collector,
) {
	_ = connections

	pipeline := cfg.Pipeline
	if pipeline <= 0 {
		pipeline = 1
	}

	var wg sync.WaitGroup
	for i := 0; i < pipeline; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runPipelineSlot(ctx, durationDone, client, cfg, collector)
		}()
	}
	wg.Wait()
}

// runPipelineSlot issues HTTP requests in a loop until ctx is cancelled or durationDone
// is closed. When durationDone closes, we stop after the current request completes
// so in-flight requests are not aborted by a timeout.
func runPipelineSlot(
	ctx context.Context,
	durationDone <-chan struct{},
	client *http.Client,
	cfg Config,
	collector *stats.Collector,
) {
	var bodyReader io.Reader
	if len(cfg.Body) > 0 {
		bodyReader = bytes.NewReader(cfg.Body)
	}
	req, err := http.NewRequestWithContext(ctx, cfg.Method, cfg.URL, bodyReader)
	if err != nil {
		return
	}
	if len(cfg.Body) > 0 {
		req.ContentLength = int64(len(cfg.Body))
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-durationDone:
			return
		default:
			// With a body we must create a new request each time (reader is consumed).
			r := req
			if len(cfg.Body) > 0 {
				r, err = http.NewRequestWithContext(ctx, cfg.Method, cfg.URL, bytes.NewReader(cfg.Body))
				if err != nil {
					return
				}
				r.ContentLength = int64(len(cfg.Body))
			}

			bytesSent := uint64(len(cfg.Body))

			start := time.Now()
			resp, err := client.Do(r)
			latency := time.Since(start)

			var bytesRecv uint64
			if resp != nil && resp.Body != nil {
				n, _ := io.Copy(io.Discard, resp.Body)
				bytesRecv = uint64(n)
				_ = resp.Body.Close()
			}

			success := err == nil && resp != nil && resp.StatusCode >= 200 && resp.StatusCode < 500
			collector.Record(latency, success, bytesSent, bytesRecv)
		}
	}
}

