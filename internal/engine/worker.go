package engine

import (
	"context"
	"net/http"
	"time"

	"github.com/thetangentline/httpcl/internal/stats"
)

// worker issues HTTP requests in a loop until the context is done.
// It reuses the provided http.Client for connection pooling and keep-alive.
func worker(
	ctx context.Context,
	client *http.Client,
	cfg Config,
	connections int,
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

			success := err == nil && resp.StatusCode >= 200 && resp.StatusCode < 500
			collector.Record(latency, success)

			if resp != nil && resp.Body != nil {
				_ = resp.Body.Close()
			}

			// Simple pipelining emulation via tight loop; throttling can be added later.
			_ = connections // currently unused but kept for future per-worker conn limits
		}
	}
}

