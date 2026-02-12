package test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/thetangentline/httpcl/internal/engine"
)

// testServer returns an httptest.Server that:
// - returns 404 for path /fail404, 500 for /fail500
// - returns 200 for all other paths and methods
func testServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/fail404", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("/fail500", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	return httptest.NewServer(mux)
}

func TestRun_RequiresURL(t *testing.T) {
	cfg := engine.Config{}
	orch := engine.NewOrchestrator(cfg, NewNoopRenderer())
	err := orch.Run()
	if err == nil {
		t.Fatal("expected error when URL is empty")
	}
	if err.Error() != "url is required" {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestRun_InvalidURL verifies Run returns before starting when URL has no host (DNS preflight fails).
func TestRun_InvalidURL(t *testing.T) {
	cfg := engine.Config{
		URL:        "http://",
		Duration:   time.Second,
		Workers:    1,
		Pipeline:   1,
		Connections: 1,
	}
	orch := engine.NewOrchestrator(cfg, NewNoopRenderer())
	err := orch.Run()
	if err == nil {
		t.Fatal("expected error for URL with missing host")
	}
}

func TestRun_AllMethods(t *testing.T) {
	srv := testServer()
	defer srv.Close()

	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE"}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			cfg := engine.Config{
				Method:      method,
				URL:         srv.URL + "/",
				Body:        []byte{},
				Connections: 2,
				Duration:   100 * time.Millisecond,
				Workers:     1,
				Pipeline:   1,
			}
			if method == "POST" || method == "PUT" || method == "PATCH" {
				cfg.Body = []byte("test-body")
			}
			orch := engine.NewOrchestrator(cfg, NewNoopRenderer())
			err := orch.Run()
			if err != nil {
				t.Fatalf("method %s: %v", method, err)
			}
		})
	}
}

func TestRun_WithBody(t *testing.T) {
	var lastBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			b := make([]byte, 20)
			n, _ := r.Body.Read(b)
			lastBody = b[:n]
			_ = r.Body.Close()
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	body := []byte("hello-world")
	cfg := engine.Config{
		Method:      "POST",
		URL:         srv.URL + "/",
		Body:        body,
		Connections: 1,
		Duration:    100 * time.Millisecond,
		Workers:     1,
		Pipeline:    1,
	}
	orch := engine.NewOrchestrator(cfg, NewNoopRenderer())
	err := orch.Run()
	if err != nil {
		t.Fatal(err)
	}
	if len(lastBody) != len(body) || string(lastBody) != string(body) {
		t.Errorf("server received body %q, want %q", lastBody, body)
	}
}

func TestRun_EdgeCase_ServerReturns404(t *testing.T) {
	srv := testServer()
	defer srv.Close()

	cfg := engine.Config{
		Method:      "GET",
		URL:         srv.URL + "/fail404",
		Connections: 2,
		Duration:    200 * time.Millisecond,
		Workers:     1,
		Pipeline:    1,
	}
	orch := engine.NewOrchestrator(cfg, NewNoopRenderer())
	err := orch.Run()
	if err != nil {
		t.Fatal(err)
	}
	// Run completes; 404 is counted as success (engine: 200 <= code < 500).
}

func TestRun_EdgeCase_ServerReturns500(t *testing.T) {
	srv := testServer()
	defer srv.Close()

	cfg := engine.Config{
		Method:      "GET",
		URL:         srv.URL + "/fail500",
		Connections: 2,
		Duration:    200 * time.Millisecond,
		Workers:     1,
		Pipeline:    1,
	}
	orch := engine.NewOrchestrator(cfg, NewNoopRenderer())
	err := orch.Run()
	if err != nil {
		t.Fatal(err)
	}
	// 5xx is counted as error by engine (success = 200 <= code < 500); run still completes.
}

func TestRun_EdgeCase_ShortDuration_Drain(t *testing.T) {
	srv := testServer()
	defer srv.Close()

	cfg := engine.Config{
		Method:      "GET",
		URL:         srv.URL + "/",
		Connections: 4,
		Duration:    50 * time.Millisecond,
		Workers:     2,
		Pipeline:    2,
	}
	orch := engine.NewOrchestrator(cfg, NewNoopRenderer())
	err := orch.Run()
	if err != nil {
		t.Fatalf("short duration run should complete without error: %v", err)
	}
}

func TestRun_EdgeCase_MultipleWorkersAndPipeline(t *testing.T) {
	srv := testServer()
	defer srv.Close()

	cfg := engine.Config{
		Method:      "GET",
		URL:         srv.URL + "/",
		Connections: 8,
		Duration:    150 * time.Millisecond,
		Workers:     4,
		Pipeline:    2,
	}
	orch := engine.NewOrchestrator(cfg, NewNoopRenderer())
	err := orch.Run()
	if err != nil {
		t.Fatal(err)
	}
}

func TestRun_EdgeCase_EmptyBody_GET(t *testing.T) {
	srv := testServer()
	defer srv.Close()

	cfg := engine.Config{
		Method:      "GET",
		URL:         srv.URL + "/",
		Body:        nil,
		Connections: 1,
		Duration:    80 * time.Millisecond,
		Workers:     1,
		Pipeline:    1,
	}
	orch := engine.NewOrchestrator(cfg, NewNoopRenderer())
	err := orch.Run()
	if err != nil {
		t.Fatal(err)
	}
}

func TestNewOrchestrator_DefaultConfig(t *testing.T) {
	cfg := engine.Config{
		URL: "http://127.0.0.1:9999/",
	}
	orch := engine.NewOrchestrator(cfg, NewNoopRenderer())
	if orch == nil {
		t.Fatal("NewOrchestrator returned nil")
	}
	// Defaults are applied inside NewOrchestrator; we can't easily assert private cfg.
	// Just ensure no panic and Run fails fast (e.g. connection refused or DNS) or we use a real server.
	// For a quick test without starting server: Run() will do PreflightDNS("127.0.0.1") which resolves, then try to connect. So it might run. Skip or use a short duration.
	// Let's just verify NewOrchestrator doesn't panic and returns non-nil.
}
