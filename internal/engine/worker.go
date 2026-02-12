package engine

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/thetangentline/httpcl/internal/stats"
)

// worker runs as one "process": it spawns cfg.Pipeline goroutines (one per pipeline
// slot) so that many requests are in flight concurrently per worker. All share the
// same http.Client and collector. connections is reserved for future per-worker
// connection limits.
func worker(
	ctx context.Context,
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
			runPipelineSlot(ctx, client, cfg, collector)
		}()
	}
	wg.Wait()
}

// runPipelineSlot issues HTTP requests in a loop until the context is done.
// One goroutine per pipeline slot gives concurrent in-flight requests per worker.
func runPipelineSlot(
	ctx context.Context,
	client *http.Client,
	cfg Config,
	collector *stats.Collector,
) {
	req, err := http.NewRequestWithContext(ctx, cfg.Method, cfg.URL, nil)
	if err != nil {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			start := time.Now()
			resp, err := client.Do(req)
			latency := time.Since(start)

			success := err == nil && resp != nil && resp.StatusCode >= 200 && resp.StatusCode < 500
			collector.Record(latency, success)

			if resp != nil && resp.Body != nil {
				_ = resp.Body.Close()
			}
		}
	}
}

