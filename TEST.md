# Test Documentation: httpcl

This document explains the project’s testing strategy, where tests live, how each test works, and why they are written the way they are.

---

## 1. Testing strategy overview

We use two kinds of tests:

- **Unit tests** – Test a single package in isolation with minimal dependencies. They live in `*_test.go` files **inside** the package they test (e.g. `pkg/netutil/checks_test.go`, `internal/stats/collector_test.go`, `internal/engine/engine_test.go`). They exercise one function or type at a time, often with no network or with a local test server.
- **Integration tests** – Test the **engine’s full run path** (orchestrator, workers, HTTP client, stats) against a real HTTP server. They live in the **`test/`** package and use `httptest.Server` so no external network or real DNS is required (the server URL is `http://127.0.0.1:<port>`).

We do **not** test the CLI (`internal/cli`) or the UI renderer output in this suite; those would require either capturing stdout or running the binary as a subprocess, which we leave for future work.

---

## 2. Where test files live and why

### 2.1 `test/` directory (integration tests)

**Location:** `test/integration_test.go`, `test/noop_renderer.go`  
**Package name:** `test`

**Why a separate top-level `test/` folder?**

- Integration tests need to **wire multiple packages together**: `engine` (orchestrator, config, client), and indirectly `stats`, `netutil`, and the **UI interface** (`ui.Renderer`). Putting them in a dedicated `test` package keeps integration concerns in one place and avoids making the engine or UI depend on test-only types.
- The `test` package imports `engine` and provides its **own** implementation of `ui.Renderer` (the no-op renderer). That way we never start the real TUI or print to the terminal during tests.
- We run the **same code path** as production: `Orchestrator.Run()` → preflight (DNS, ulimit) → workers → HTTP requests → stats → renderer. The only substitution is the renderer implementation and the target URL (our `httptest.Server`).

**Why package name `test`?**

- It’s a clear, conventional name for “tests that exercise the program as a whole.” Running `go test ./test/` runs only these integration tests. The name does not conflict with the standard `testing` package.

### 2.2 `pkg/netutil/checks_test.go` (unit tests)

**Location:** Next to `checks.go` in the same package.  
**Package name:** `netutil` (same as production code).

**Why here?**

- Go idiom: **unit tests live beside the code they test** in the same package. That gives “white-box” access (we could test unexported functions if we had any) and keeps the test file close to the implementation.
- `netutil` has **no dependency** on the rest of the app: it only uses the standard library (`net`, `net/url`, `syscall`). So we can test it in complete isolation. There is no need for a fake server or engine.

### 2.3 `internal/stats/collector_test.go` (unit tests)

**Location:** Next to `collector.go` in the same package.  
**Package name:** `stats`.

**Why here?**

- The collector is a **pure metrics aggregator**: it has no HTTP, no UI, no config. Testing it in-package lets us call `Record()` and `Snapshot()` directly and assert on exact numbers and percentiles.
- We need to verify **thread safety** (multiple goroutines calling `Record` concurrently) and **correctness of derived stats** (latency percentiles, success/error counts, bytes). Doing that in a small unit test is simpler and faster than going through the full engine.

### 2.4 `internal/engine/engine_test.go` (unit tests)

**Location:** Next to `orchestrator.go`, `client.go`, etc., in the same package.  
**Package name:** `engine`.

**Why here?**

- The engine package has **unexported** types and functions (`orchestrator.cfg`, `worker`, `runPipelineSlot`, `newHTTPClient`). Tests in the same package can only call **exported** APIs (`NewOrchestrator`, `Run`, and the constructor we use for the client in tests). We still put tests here to:
  - Use a **no-op renderer** (defined in the test file) that implements `ui.Renderer`, so we don’t pull in the real UI or print anything.
  - Test `NewOrchestrator` with various configs (zero workers, zero duration, empty method) to ensure it **does not panic** and applies defaults.
  - Test **`newHTTPClient`** directly (exported only for tests would require a test-only export; here we can test it because the test file is in the same package and can access the unexported name… Actually in Go, unexported names are only visible in the same package, so `engine_test.go` is in package `engine`, so it **can** call `newHTTPClient`). So we test the client’s shape (non-nil, Transport set, `Timeout == 0`) to document and guard the benchmark behaviour.

**Why a no-op renderer in engine_test and another in test/?**

- **engine_test.go** defines a small `noopRender` (value receiver, no pointer) so the engine package doesn’t depend on the `test` package. These tests are quick “does it construct and run without panic” checks.
- **test/noop_renderer.go** defines `noopRenderer` for the **integration** tests, which need to pass a `Renderer` into `NewOrchestrator` from **outside** the engine package. The engine only sees the interface; both no-op implementations satisfy `ui.Renderer`.

---

## 3. How to run tests

```bash
# All tests (unit + integration)
go test ./...

# Only integration tests (test/ package)
go test ./test/...

# Only unit tests for a specific package
go test ./pkg/netutil/...
go test ./internal/stats/...
go test ./internal/engine/...

# With verbose output
go test -v ./...

# Run a single test by name
go test -run TestRun_RequiresURL ./test/...
```

Integration tests start a real HTTP server and run for the configured duration (e.g. 50–200 ms per test), so the whole suite may take around 1–2 seconds.

---

## 4. Integration tests (`test/` package) in depth

### 4.1 No-op renderer (`test/noop_renderer.go`)

**What it does:** Defines a type `noopRenderer` with two methods, `Render(snap stats.Snapshot)` and `RenderFinal(snap stats.Snapshot)`, both with empty bodies. `NewNoopRenderer()` returns a `*noopRenderer`.

**Why it exists:** `Orchestrator.Run()` expects a `ui.Renderer`. In production, that’s the ASCII TUI: it prints a live line and a final report. In tests we must not print to stdout (it would clutter test output and make assertions on stdout fragile). So we pass a renderer that **implements the interface but does nothing**. The engine still calls `Render` and `RenderFinal` at the same points; we just don’t observe the output.

**Why in the test package:** The `test` package is the only place that needs this. The engine only depends on the **interface** `ui.Renderer`; it doesn’t know about `noopRenderer`. So the dependency is one-way: `test` → `engine` and `test` → `internal/stats` (for `stats.Snapshot`). No cycle.

### 4.2 Test server (`testServer()` in integration_test.go)

**What it does:** Builds an `httptest.Server` with a multiplexer that:

- **`/fail404`** → responds with `404 Not Found`.
- **`/fail500`** → responds with `500 Internal Server Error`.
- **`/`** (and any other path) → responds with `200 OK` and body `"ok"`.

**Why this shape:** We need a **deterministic** backend that:

1. Accepts **all HTTP methods** (GET, POST, PUT, PATCH, DELETE) so we can test each method without changing the server.
2. Exposes paths that **force** specific status codes so we can assert that the engine treats 4xx vs 5xx correctly (success vs error in the stats).
3. Binds to **localhost** so `PreflightDNS(o.cfg.URL)` succeeds when the URL is `srv.URL` (e.g. `http://127.0.0.1:xxxxx`). No external DNS or network.

**Why `defer srv.Close()`:** Each test that uses the server must close it when the test ends so the OS releases the port and we don’t leak goroutines.

### 4.3 TestRun_RequiresURL

**What it does:** Builds an `engine.Config` with **empty URL**, creates an orchestrator with the no-op renderer, and calls `orch.Run()`. It then asserts that `Run()` returns a **non-nil error** and that the error message is exactly `"url is required"`.

**Why test this:** The orchestrator is designed to **fail fast** if the URL is missing: it returns before starting DNS preflight, workers, or the renderer loop. This test locks in that contract and the exact error message so callers (CLI) can rely on it.

**Why no server:** We never send a single HTTP request; the error happens at the very start of `Run()`.

### 4.4 TestRun_InvalidURL

**What it does:** Sets `cfg.URL = "http://"` (no host). Then calls `orch.Run()` and asserts that it returns a **non-nil error**.

**Why test this:** The first step of `Run()` after the URL check is **DNS preflight** (`netutil.PreflightDNS`). For `"http://"`, `url.Parse` yields an empty hostname, and `PreflightDNS` returns an error like “missing host in url”. We want to ensure the benchmark **never starts** when the URL is invalid: no workers, no connections, no renderer loop. So we don’t need a server; we only check that `Run()` errors out.

**Why this URL:** It’s a minimal invalid case that doesn’t depend on network or DNS. Other options (e.g. unresolvable hostname) could be flaky or require mocks.

### 4.5 TestRun_AllMethods

**What it does:** Starts the test server, then for each of **GET, POST, PUT, PATCH, DELETE** runs a subtest that:

1. Builds a `Config` with that method, `URL = srv.URL + "/"`, and short duration (100 ms).
2. For POST, PUT, PATCH, sets `Body: []byte("test-body")`; for GET and DELETE leaves body empty.
3. Creates an orchestrator with the no-op renderer and calls `orch.Run()`.
4. Asserts that `Run()` returns **no error**.

**Why test all methods:** The engine and worker build `http.Request` with `cfg.Method` and optionally `cfg.Body`. We need to ensure every method we claim to support actually runs without panic or error. GET and DELETE with no body, and POST/PUT/PATCH with body, cover the two code paths (reused request vs. new request per iteration when body is present).

**Why 100 ms duration:** Long enough for at least a few requests to complete so the full path (workers, pipeline slots, client, collector) is exercised, but short enough to keep the suite fast.

### 4.6 TestRun_WithBody

**What it does:** Starts a **custom** server that reads the request body into a package-level variable `lastBody`. Then runs the orchestrator with **POST** and `Body: []byte("hello-world")`. After `Run()` succeeds, it asserts that `lastBody` equals the sent body.

**Why test this:** With a body, the worker creates a **new** `*http.Request` per iteration (because the body reader is consumed). This test proves that the body we put in `Config.Body` is actually sent on the wire and received by the server. Without it we could have a bug where body is ignored or sent only once.

**Why a custom server:** The shared `testServer()` doesn’t inspect the body. Here we need to **capture** the body to assert on it, so we use a one-off handler.

### 4.7 TestRun_EdgeCase_ServerReturns404 and TestRun_EdgeCase_ServerReturns500

**What they do:** Point the config at `srv.URL + "/fail404"` and `srv.URL + "/fail500"` respectively, run the orchestrator, and assert **no error from `Run()`** (the benchmark completes).

**Why test 404:** In the engine, “success” is defined as `err == nil && resp != nil && resp.StatusCode >= 200 && resp.StatusCode < 500`. So **404 is counted as success** (we get a valid response; it’s the application’s semantics). The test ensures that hitting 404 doesn’t cause `Run()` to fail or panic; the run finishes and the stats collector records those responses as successes.

**Why test 500:** Status **500** is **outside** the success range, so it’s counted as an **error** in the stats. We don’t assert the exact error count here; we only check that `Run()` still **completes**. That way we know the engine doesn’t crash or hang when the server returns 5xx; errors are recorded and the run shuts down normally after the duration.

### 4.8 TestRun_EdgeCase_ShortDuration_Drain

**What it does:** Runs the benchmark with **50 ms duration**, 2 workers, pipeline 2, 4 connections. Asserts that `Run()` returns **no error**.

**Why test this:** When the duration expires, the orchestrator **closes** the `durationDone` channel. Workers are supposed to **stop starting new requests** but **finish in-flight requests** (drain). If we wrongly cancelled the context or timed out the HTTP client, we’d see a burst of errors or a panic. This test stresses that the drain logic works: 50 ms is short enough that many requests are still in flight when the timer fires, yet the run must complete cleanly. So we’re regression-testing the “duration does not cancel in-flight requests” behaviour.

### 4.9 TestRun_EdgeCase_MultipleWorkersAndPipeline

**What it does:** Runs with 4 workers, pipeline 2, 8 connections, 150 ms duration. Asserts that `Run()` succeeds.

**Why test this:** Confirms that **multiple workers** and **multiple pipeline slots per worker** don’t cause races, deadlocks, or wrong stats. The shared client and shared collector are used from many goroutines; this test gives us confidence that the design holds under concurrency.

### 4.10 TestRun_EdgeCase_EmptyBody_GET

**What it does:** Explicitly sets `Body: nil` and method GET, then runs the benchmark.

**Why test this:** Documents that GET with no body is a valid and common case. The worker uses a **single** request and reuses it for every iteration when there’s no body; this test ensures that path works and doesn’t assume a non-nil body.

### 4.11 TestNewOrchestrator_DefaultConfig

**What it does:** Builds a config with only `URL` set (to `http://127.0.0.1:9999/`), calls `NewOrchestrator(cfg, NewNoopRenderer())`, and asserts the result is **non-nil**.

**Why test this:** `NewOrchestrator` applies **defaults** when fields are zero (workers, connections, pipeline, method, duration). We can’t read the private `orch.cfg` from the test package to assert exact values, but we can ensure that **partial config** doesn’t cause a panic and that we get a valid orchestrator back. It’s a smoke test for the constructor.

---

## 5. Unit tests: `pkg/netutil/checks_test.go`

These tests exercise the **preflight** helpers used by the orchestrator before starting a run.

### 5.1 PreflightDNS

**Function behaviour:** `PreflightDNS(rawURL string) error` parses the URL, extracts the hostname, and calls `net.LookupHost(host)`. It returns an error if parsing fails, if the host is empty, or if DNS resolution fails.

- **TestPreflightDNS_InvalidURL:** Passes `"://no-scheme"`. Expects a **non-nil** error whose message contains `"invalid url"`. This checks that malformed URLs are rejected before we ever try DNS.
- **TestPreflightDNS_MissingHost:** Passes `"http://"`. The host is empty after parse. We expect an error containing `"missing host"`. This is the same case we use in `TestRun_InvalidURL` at the integration level; here we test the netutil function in isolation.
- **TestPreflightDNS_ValidResolvableHost:** Passes `http://127.0.0.1/` and `http://localhost/`. Both should resolve on any normal machine. We assert **no error**. This proves that valid, resolvable URLs pass preflight.
- **TestPreflightDNS_UnresolvableHost:** Passes `http://nonexistent.invalid/`. The `.invalid` TLD is reserved (RFC 6761) for “never resolve.” If resolution fails we assert the error message contains `"dns resolution failed"`. If for some reason the environment resolves `.invalid`, we **skip** the test so we don’t fail on exotic setups.

### 5.2 CheckUlimitWarning

**Function behaviour:** On Unix, it reads the process soft limit for open files (`RLIMIT_NOFILE`). If the **requested connection count** is greater than that limit, it returns an error (so the CLI can warn the user). On non-Unix or if `Getrlimit` fails, it returns `nil` (best-effort).

- **TestCheckUlimitWarning_ZeroConnections:** Passes `0`. The function returns early for `requestedConns <= 0` and returns `nil`. We assert no error.
- **TestCheckUlimitWarning_NegativeConnections:** Passes `-1`. Same early exit; we assert no error.
- **TestCheckUlimitWarning_ReturnsErrorWhenOverLimit:** Passes `10_000_000`. On typical systems the soft limit is much lower, so we expect an error whose message contains “requested connections” and “exceed.” On very high-limit systems we accept no error; the test is best-effort.
- **TestCheckUlimitWarning_SmallRequest:** Passes `10`. We only call the function (no assertion). On normal systems it returns `nil`; on very constrained systems it might return an error. We don’t fail either way; the test just ensures the call doesn’t panic.

---

## 6. Unit tests: `internal/stats/collector_test.go`

The collector is the **single source of truth** for request counts, latency samples, and per-second buckets. These tests lock in its behaviour.

### 6.1 TestNewCollector

**What it does:** Calls `NewCollector()`, then immediately calls `Snapshot()`. Asserts the collector is non-nil and that the snapshot has zero total requests, successes, and errors.

**Why:** A fresh collector must not count anything. This also checks that `Snapshot()` is safe to call with no `Record()` calls (no divide-by-zero or nil slice).

### 6.2 TestRecord_SuccessAndError

**What it does:** Creates a collector, calls `Record(10*time.Millisecond, true, 0, 100)` and `Record(20*time.Millisecond, false, 50, 0)`, then takes a snapshot. Asserts: total requests = 2, successes = 1, errors = 1, total bytes sent = 50, total bytes received = 100.

**Why:** This is the **core contract** of `Record`: the first call is a success with 0 sent and 100 received; the second is a failure with 50 sent and 0 received. We verify that atomics and the success/error branch are correct and that bytes are accumulated.

### 6.3 TestRecord_Concurrent

**What it does:** Creates one collector and 100 goroutines; each calls `Record(time.Millisecond, true, 1, 1)`. Waits for all goroutines, then takes a snapshot. Asserts total requests = 100, successes = 100, total bytes sent = 100, total bytes received = 100.

**Why:** The real benchmark has many workers and pipeline slots all calling `Record` at the same time. We need to ensure there are **no lost updates** (no race, no wrong count). Using atomics and a mutex for samples is the intended design; this test would catch regressions (e.g. if someone removed atomics).

### 6.4 TestSnapshot_LatencyPercentiles

**What it does:** Records five latencies: 10, 20, 30, 40, 50 ms. Takes a snapshot and asserts: P50 = 30 ms, max = 50 ms, and average is non-zero.

**Why:** The report shows P50, P99, max, and average. Those are computed in `Snapshot()` from the stored latency samples (sorted, then percentile index). This test ensures the **sorting and percentile math** are correct for a small, known set of values.

### 6.5 TestSnapshot_EmptyCollector

**What it does:** Creates a collector, takes a snapshot without any `Record()`. Asserts total requests = 0 and that latency fields (e.g. P50, max) are zero.

**Why:** Ensures `Snapshot()` doesn’t panic or return garbage when there are no samples. The report code must handle “no data” gracefully.

---

## 7. Unit tests: `internal/engine/engine_test.go`

These tests live **inside** the engine package so they can use the same package’s unexported API where needed.

### 7.1 noopRender type

**What it is:** A small struct with two methods, `Render(snap stats.Snapshot)` and `RenderFinal(snap stats.Snapshot)`, both no-ops. It implements `ui.Renderer`.

**Why in engine_test:** The engine’s `NewOrchestrator` and `Run()` require a `Renderer`. The engine package must not import the `test` package (that would create a dependency from production code to tests). So we define a **local** no-op implementation in the engine test file. The engine only sees the interface; it doesn’t care that it’s a no-op.

### 7.2 TestNewOrchestrator_AppliesDefaults, ZeroWorkers, ZeroDuration, DefaultMethod

**What they do:** Build configs with **only** `URL` set (and in some cases `Workers: 0`, `Duration: 0`, or `Method: ""`), call `NewOrchestrator(cfg, noopRender{})`, and assert the result is **non-nil**.

**Why we don’t assert default values:** The applied config is stored in the private field `orch.cfg`. We don’t export it or add test-only getters. So we **can’t** assert “workers became runtime.NumCPU()” or “duration became 10s” without changing production code. Instead we assert that **partial or zero config doesn’t panic** and that we get a valid orchestrator. The actual default values are documented in the orchestrator code and in ARCHITECTURE.md.

### 7.3 TestNewHTTPClient_NoPanic and TestNewHTTPClient_ZeroTimeout

**What they do:** Call `newHTTPClient(10)` (or `5`), then assert: the client is non-nil, the `Transport` is non-nil, and `Client.Timeout` is **0**.

**Why test the client:** The benchmark is designed to **not** use `http.Client.Timeout`; timeouts are controlled by **context** (SIGINT) and by the **duration** (closing `durationDone`). If someone added a non-zero `Client.Timeout`, long-running requests could be cut off and we’d see spurious errors. So we lock in “Timeout must be 0” and “Transport is set” as part of the engine’s contract.

**Why we can call `newHTTPClient`:** The test file is in package `engine`, so it can call unexported functions like `newHTTPClient`. We use that to test the client’s shape without going through the full `Run()`.

---

## 8. Summary table

| Location              | Package  | Type        | What it validates |
|-----------------------|----------|------------|-------------------|
| `test/`               | test     | Integration| Full run path: URL validation, preflight, all methods, body, 4xx/5xx, drain, concurrency. |
| `pkg/netutil/`        | netutil  | Unit       | PreflightDNS (invalid URL, missing host, resolvable, unresolvable); CheckUlimitWarning (0, negative, over limit). |
| `internal/stats/`     | stats    | Unit       | Collector: Record (success/error, bytes), Snapshot (totals, percentiles, empty), concurrent Record. |
| `internal/engine/`    | engine   | Unit       | NewOrchestrator with partial config (no panic); newHTTPClient (non-nil, Transport, Timeout 0). |

Together, these tests give confidence that the benchmark runs correctly for all supported methods and that preflight, stats, and duration-drain behaviour stay correct as the code changes.
