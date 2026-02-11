# Project Specification: httpcl

## 1. Project Overview

`httpcl` (HTTP Command Line) is a high-performance, scriptable HTTP benchmarking tool written in Go. It is designed to stress-test endpoints using varying levels of concurrency, pipelining, and duration. It provides real-time feedback through a clean, ASCII-based terminal interface and handles complex networking edge cases.

## 2. Technical Stack

- **Language:** Go 1.21+
- **CLI Framework:** [Cobra](https://github.com/spf13/cobra) (for command routing).
- **Interactive Prompts:** [Survey](https://github.com/AlecAivazis/survey) or [Bubble Tea](https://github.com/charmbracelet/bubbletea).
- **Networking:** Standard `net/http` with customized `http.Transport` for fine-grained connection pooling and keep-alive management.

## 3. Command Structure

| Command        | Description                                                                    | Example                                |
| :------------- | :----------------------------------------------------------------------------- | :------------------------------------- |
| `httpcl start` | **Interactive Mode:** A wizard to set methods, headers, and stress parameters. | `httpcl start`                         |
| `httpcl run`   | **Direct Mode:** Executes a test immediately using flags.                      | `httpcl run -u https://api.com -c 100` |

### Supported Flags (Direct Mode)

- `-m, --method`: HTTP method (GET, POST, PUT, DELETE). Default: `GET`.
- `-u, --url`: Target URL (Required).
- `-c, --connections`: Number of concurrent persistent connections.
- `-d, --duration`: Total test duration (e.g., `10s`, `2m`, `1h`).
- `-w, --workers`: Number of CPU workers/goroutines to spawn.
- `-p, --pipeline`: Number of pipelined requests per connection.

## 4. Edge Case Handling

- **DNS Resolution:** Performs a pre-flight check to validate the URL and resolve the host before spawning workers.
- **Connection Health:** Detects rate-limiting, connection drops, and "Socket Hang-ups," reporting them as error percentages.
- **Payload Management:** Supports raw string bodies or file paths (e.g., `.json`) for `POST`/`PUT` methods.
- **Signal Handling:** Gracefully traps `Ctrl+C` (SIGINT) to stop workers and display the statistics collected up to that point.
- **System Limits:** Checks OS `ulimit` (Open Files) and warns if the requested connection count (`-c`) exceeds system capacity.

## 5. UI Requirements

- **No Emojis/Icons:** Use standard ASCII and box-drawing characters only.
- **Responsive:** Layout must adapt to terminal width.
- **Hierarchy:** Use ANSI colors and text weight (Bold/Dim) for visual structure.
