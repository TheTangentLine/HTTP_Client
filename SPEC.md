# Project Specification: httpcl

## 1. Project Overview

`httpcl` (HTTP Command Line) is a high-performance, scriptable HTTP benchmarking tool written in Go. It stress-tests endpoints using configurable concurrency, pipelining, and duration. It provides real-time feedback through a clean, ASCII-based terminal interface and handles DNS, system limits, and graceful shutdown so that in-flight requests complete when the duration ends.

## 2. Technical Stack

- **Language:** Go 1.21+
- **CLI:** [Cobra](https://github.com/spf13/cobra) for command routing and flags.
- **Interactive mode:** Bufio-based prompts (no external survey library); produces a config that the CLI maps into the engine.
- **Networking:** Standard `net/http` with a custom `http.Transport`: connection pooling, keep-alive, HTTP/2, no `Client.Timeout` (lifecycle controlled by context and duration).

## 3. Command Structure

| Command        | Description                                                                    | Example                                |
| :------------- | :----------------------------------------------------------------------------- | :------------------------------------- |
| `httpcl start` | **Interactive mode:** Wizard to set method, URL, body (if applicable), and stress parameters. | `httpcl start`                         |
| `httpcl run`   | **Direct mode:** Run a benchmark using flags only.                              | `httpcl run -u https://api.example.com -c 100 -d 10s` |

### Flags (Direct mode: `run`)

| Flag | Short | Description | Default |
|------|--------|-------------|--------|
| `--method` | `-m` | HTTP method (GET, POST, PUT, PATCH, DELETE). | GET |
| `--url` | `-u` | Target URL. Required for `run`. | (required) |
| `--body` | `-b` | Request body for POST/PUT/PATCH (raw string). | (empty) |
| `--connections` | `-c` | Number of concurrent persistent connections (pool size). | 10 |
| `--duration` | `-d` | Total test duration (e.g. `10s`, `2m`, `1h`). After this time, no new requests are started; in-flight requests complete. | 10s |
| `--workers` | `-w` | Number of worker goroutines. Each worker runs `--pipeline` concurrent request loops. | 1 |
| `--pipeline` | `-p` | Pipelined requests per worker (concurrent in-flight requests per worker). | 1 |

## 4. Edge Case Handling

- **DNS resolution:** Pre-flight check (`netutil.PreflightDNS`) validates and resolves the URL host before any workers start. On failure, the benchmark does not run.
- **System limits:** Best-effort `ulimit` check (`netutil.CheckUlimitWarning`) warns if the requested connection count exceeds the process soft open-files limit; the benchmark still runs.
- **Duration vs. in-flight requests:** When the duration expires, a `durationDone` channel is closed. Workers stop starting new requests but let every request already sent finish. The context is cancelled only on SIGINT or after all workers have returned, so the duration timer does not abort in-flight HTTP calls (avoiding a spike of errors at the end of the run).
- **Signal handling:** SIGINT and SIGTERM cancel the context so workers and the renderer exit promptly; the final report is still printed from the last snapshot.
- **Payload:** For POST/PUT/PATCH, a body can be provided via `-b/--body` (direct) or the wizard (interactive). Each request uses the same body; the client re-builds the request per call when a body is set.
- **Connection health:** Errors (e.g. connection refused, timeouts, 5xx) are counted and reported as errors; success is defined as no error and status in [200, 500).

## 5. UI Requirements

- **No emojis/icons:** ASCII and box-drawing characters only (e.g. `┌`, `─`, `│`, `└`).
- **Responsive:** Layout adapts to terminal width where applicable.
- **Hierarchy:** ANSI colors (e.g. cyan, green, red, dim) and bold for structure; progress/throughput can use characters like `[#####-----]` for bars.
