## httpcl – HTTP Command Line Benchmarking Tool

`httpcl` is a high‑performance HTTP benchmarking CLI written in Go. It supports interactive and direct modes, persistent connections, and real‑time, ASCII‑only terminal UI.

### Requirements

- **Go** 1.21 or newer installed (`go version` to check).
- A POSIX‑like shell (macOS, Linux, WSL).

### Download / Installation

You can either install via `go install` or clone and build from source.

#### Option 1: Install via `go install`

```bash
go install github.com/thetangentline/httpcl/cmd/httpcl@latest
```

Make sure your `$GOPATH/bin` (or Go install `bin` directory) is on your `PATH`, then you can run:

```bash
httpcl --help
```

#### Option 2: Clone and build from source

```bash
git clone https://github.com/thetangentline/httpcl.git
cd httpcl

go build ./cmd/httpcl
./httpcl --help
```

To install the binary into your Go `bin` directory:

```bash
go install ./cmd/httpcl
```

### Usage

`httpcl` has two primary commands: **interactive** `start` and **direct** `run`.

#### Interactive mode (`httpcl start`)

Launches a wizard that asks for URL, method, connections, duration, workers, and pipeline:

```bash
httpcl start
```

Follow the prompts in the terminal; once confirmed, the benchmark runs and a live HUD plus final report are shown.

#### Direct mode (`httpcl run`)

Supply all parameters via flags:

```bash
httpcl run \
  -u https://example.com \
  -m GET \
  -c 100 \
  -d 10s \
  -w 4 \
  -p 1
```

- **`-u, --url`**: Target URL (required).
- **`-m, --method`**: HTTP method (`GET`, `POST`, `PUT`, `DELETE`). Default: `GET`.
- **`-c, --connections`**: Number of concurrent persistent connections.
- **`-d, --duration`**: Total test duration (`10s`, `2m`, `1h`, etc.).
- **`-w, --workers`**: Number of worker goroutines (CPU workers).
- **`-p, --pipeline`**: Requests pipelined per connection.

### Reading the Output

- During the run, a **single‑line HUD** shows total requests, successes, errors, RPS, and average latency.
- At the end, a **boxed report** summarizes:
  - Total requests, successes, errors
  - Requests per second
  - P50, P95, P99 latency

Abort early with **Ctrl+C**; stats collected so far will still be reported.
