package ui

import (
	"fmt"
	"os"
	"time"

	"github.com/thetangentline/httpcl/internal/stats"
)

// Renderer defines the minimal interface used by the engine.
type Renderer interface {
	Render(snap stats.Snapshot)
	RenderFinal(snap stats.Snapshot)
}

// asciiRenderer is a simple ANSI/ASCII renderer that prints a single-line summary.
type asciiRenderer struct {
	lastLineLen int
}

// NewRenderer creates a new ASCII renderer.
func NewRenderer() Renderer {
	return &asciiRenderer{}
}

// clearLine clears the current line in the terminal using ANSI escape codes.
func (r *asciiRenderer) clearLine() {
	if r.lastLineLen == 0 {
		return
	}
	// Carriage return + clear line.
	fmt.Fprint(os.Stdout, "\r\033[2K")
}

func (r *asciiRenderer) Render(snap stats.Snapshot) {
	r.clearLine()

	line := fmt.Sprintf(
		"[httpcl] total=%d ok=%d err=%d rps=%.1f avg=%s",
		snap.TotalRequests,
		snap.Successes,
		snap.Errors,
		snap.RequestsPerS,
		snap.LatencyP50.Truncate(10*time.Microsecond),
	)

	fmt.Fprint(os.Stdout, line)
	r.lastLineLen = len(line)
}

func (r *asciiRenderer) RenderFinal(snap stats.Snapshot) {
	r.clearLine()
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "========================================")
	fmt.Fprintln(os.Stdout, " httpcl benchmark report")
	fmt.Fprintln(os.Stdout, "========================================")
	fmt.Fprintf(os.Stdout, " Total Requests : %d\n", snap.TotalRequests)
	fmt.Fprintf(os.Stdout, " Successes      : %d\n", snap.Successes)
	fmt.Fprintf(os.Stdout, " Errors         : %d\n", snap.Errors)
	fmt.Fprintf(os.Stdout, " Requests / sec : %.2f\n", snap.RequestsPerS)
	fmt.Fprintf(os.Stdout, " P50 Latency    : %s\n", snap.LatencyP50)
	fmt.Fprintf(os.Stdout, " P95 Latency    : %s\n", snap.LatencyP95)
	fmt.Fprintf(os.Stdout, " P99 Latency    : %s\n", snap.LatencyP99)
	fmt.Fprintln(os.Stdout, "========================================")
}

