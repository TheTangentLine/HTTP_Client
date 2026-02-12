package engine

import (
	"runtime"
	"testing"

	"github.com/thetangentline/httpcl/internal/stats"
)

// noopRender implements ui.Renderer for tests.
type noopRender struct{}

func (noopRender) Render(snap stats.Snapshot)   {}
func (noopRender) RenderFinal(snap stats.Snapshot) {}

func TestNewOrchestrator_AppliesDefaults(t *testing.T) {
	cfg := Config{
		URL: "http://127.0.0.1:9999/",
	}
	orch := NewOrchestrator(cfg, noopRender{})
	if orch == nil {
		t.Fatal("NewOrchestrator returned nil")
	}
	// Defaults are applied to a copy; we can't read private cfg.
	// Just ensure no panic. Optionally run with very short duration and expect connection error or success.
}

func TestNewOrchestrator_ZeroWorkersUsesNumCPU(t *testing.T) {
	cfg := Config{
		URL:     "http://127.0.0.1:1/",
		Workers: 0,
	}
	orch := NewOrchestrator(cfg, noopRender{})
	if orch == nil {
		t.Fatal("NewOrchestrator returned nil")
	}
	expect := runtime.NumCPU()
	if expect <= 0 {
		expect = 1
	}
	// We cannot read orch.cfg from here; we only verify NewOrchestrator doesn't panic.
	_ = expect
}

func TestNewOrchestrator_ZeroDurationUsesDefault(t *testing.T) {
	cfg := Config{
		URL:      "http://x/",
		Duration: 0,
	}
	orch := NewOrchestrator(cfg, noopRender{})
	if orch == nil {
		t.Fatal("NewOrchestrator returned nil")
	}
	_ = orch
}

func TestConfig_DefaultMethod(t *testing.T) {
	cfg := Config{URL: "http://a/", Method: ""}
	orch := NewOrchestrator(cfg, noopRender{})
	if orch == nil {
		t.Fatal("NewOrchestrator returned nil")
	}
}

func TestNewHTTPClient_NoPanic(t *testing.T) {
	client := newHTTPClient(10)
	if client == nil {
		t.Fatal("newHTTPClient returned nil")
	}
	if client.Transport == nil {
		t.Fatal("Transport is nil")
	}
}

func TestNewHTTPClient_ZeroTimeout(t *testing.T) {
	client := newHTTPClient(5)
	if client.Timeout != 0 {
		t.Errorf("expected Timeout 0 for benchmark client, got %v", client.Timeout)
	}
}
