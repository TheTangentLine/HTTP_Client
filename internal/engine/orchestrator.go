package engine

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/thetangentline/httpcl/internal/stats"
	"github.com/thetangentline/httpcl/internal/ui"
	"github.com/thetangentline/httpcl/pkg/netutil"
)

// Orchestrator coordinates workers, stats collection and UI rendering.
type Orchestrator struct {
	cfg      Config
	renderer ui.Renderer
}

// NewOrchestrator constructs a new Orchestrator.
func NewOrchestrator(cfg Config, renderer ui.Renderer) *Orchestrator {
	// Sensible defaults if not provided
	if cfg.Workers <= 0 {
		cfg.Workers = runtime.NumCPU()
	}
	if cfg.Connections <= 0 {
		cfg.Connections = cfg.Workers * 10
	}
	if cfg.Pipeline <= 0 {
		cfg.Pipeline = 1
	}
	if cfg.Method == "" {
		cfg.Method = "GET"
	}
	if cfg.Duration <= 0 {
		cfg.Duration = 10 * time.Second
	}

	return &Orchestrator{
		cfg:      cfg,
		renderer: renderer,
	}
}

// Run executes a full benchmark session.
func (o *Orchestrator) Run() error {
	if o.cfg.URL == "" {
		return fmt.Errorf("url is required")
	}

	// Basic DNS preflight.
	if err := netutil.PreflightDNS(o.cfg.URL); err != nil {
		return err
	}

	// Basic ulimit warning (best-effort, *nix only).
	if err := netutil.CheckUlimitWarning(o.cfg.Connections); err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
	}

	ui.PrintRunHeader(
		o.cfg.URL,
		o.cfg.Workers,
		o.cfg.Connections,
		o.cfg.Pipeline,
		o.cfg.Duration.String(),
	)

	// Context for total duration and signal handling.
	ctx, cancel := context.WithTimeout(context.Background(), o.cfg.Duration)
	defer cancel()

	// Trap SIGINT for graceful shutdown.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	collector := stats.NewCollector()
	client := newHTTPClient(o.cfg.Connections)

	// Start renderer loop.
	doneRendering := make(chan struct{})
	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				snap := collector.Snapshot()
				o.renderer.Render(snap)
			case <-ctx.Done():
				snap := collector.Snapshot()
				o.renderer.RenderFinal(snap)
				close(doneRendering)
				return
			}
		}
	}()

	var wg sync.WaitGroup
	reqsPerWorker := o.cfg.Connections / o.cfg.Workers
	if reqsPerWorker == 0 {
		reqsPerWorker = 1
	}

	for i := 0; i < o.cfg.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			worker(ctx, client, o.cfg, reqsPerWorker, collector)
		}()
	}

	// Watch for interrupt.
	go func() {
		select {
		case <-sigCh:
			cancel()
		case <-ctx.Done():
		}
	}()

	wg.Wait()
	cancel()
	<-doneRendering

	return nil
}
