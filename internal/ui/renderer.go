package ui

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/thetangentline/httpcl/internal/stats"
)

// ANSI color helpers (8/16-color safe).
const (
	colorReset = "\033[0m"
	colorBold  = "\033[1m"
	colorDim   = "\033[2m"

	colorCyan   = "\033[36m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
)

// Renderer defines the minimal interface used by the engine.
type Renderer interface {
	Render(snap stats.Snapshot)
	RenderFinal(snap stats.Snapshot)
}

// asciiRenderer is a simple ANSI/ASCII renderer that prints a single-line summary.
type asciiRenderer struct {
	lastLineLen int
	headerShown bool
}

// NewRenderer creates a new ASCII renderer.
func NewRenderer() Renderer {
	return &asciiRenderer{}
}

// winsize mirrors the struct used by TIOCGWINSZ.
type winsize struct {
	rows    uint16
	cols    uint16
	xpixels uint16
	ypixels uint16
}

// termWidth returns the current terminal width, or a sensible default.
func termWidth() int {
	ws := &winsize{}
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(os.Stdout.Fd()),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws)),
	)
	if err != 0 || ws.cols == 0 {
		return 80
	}
	return int(ws.cols)
}

// truncateToWidth ensures the line fits in the current terminal width.
func truncateToWidth(s string, width int) string {
	if width <= 0 || len(s) <= width {
		return s
	}
	if width <= 3 {
		return s[:width]
	}
	return s[:width-3] + "..."
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

	// Show a one-time header and footer hint for controls.
	if !r.headerShown {
		width := termWidth()
		if width <= 0 {
			width = 80
		}
		if width > 72 {
			width = 72
		}
		border := strings.Repeat("─", width)

		title := fmt.Sprintf("%s%sHTTPCL benchmark%s", colorBold, colorCyan, colorReset)
		fmt.Fprintf(os.Stdout, "%s\n%s\n", title, border)
		fmt.Fprintf(os.Stdout, "%sControls:%s Ctrl+C to stop\n\n", colorDim, colorReset)
		r.headerShown = true
	}

	// Color-coded, single-line HUD.
	line := fmt.Sprintf(
		"%s[httpcl]%s total=%d %sok=%d%s %serr=%d%s rps=%.1f avg=%s",
		colorCyan, colorReset,
		snap.TotalRequests,
		colorGreen, snap.Successes, colorReset,
		colorRed, snap.Errors, colorReset,
		snap.RequestsPerS,
		snap.LatencyP50.Truncate(10*time.Microsecond),
	)

	line = truncateToWidth(line, termWidth())

	fmt.Fprint(os.Stdout, line)
	r.lastLineLen = len(line)
}

func (r *asciiRenderer) RenderFinal(snap stats.Snapshot) {
	r.clearLine()
	fmt.Fprintln(os.Stdout)

	width := termWidth()
	if width <= 0 {
		width = 80
	}
	if width > 72 {
		width = 72
	}

	inner := width - 2

	// Simple box around the final report.
	hLine := ""
	for i := 0; i < inner; i++ {
		hLine += "─"
	}

	fmt.Fprintf(os.Stdout, "┌%s┐\n", hLine)
	baseTitle := " HTTPCL BENCHMARK REPORT "
	if len(baseTitle) > inner {
		baseTitle = baseTitle[:inner]
	}
	padding := inner - len(baseTitle)
	if padding < 0 {
		padding = 0
	}
	// Bold, slightly “larger-feel” title while keeping box alignment by
	// measuring width from the uncolored text and only coloring the content.
	coloredTitle := colorBold + baseTitle + colorReset
	fmt.Fprintf(os.Stdout, "│%s%s│\n", coloredTitle, strings.Repeat(" ", padding))
	fmt.Fprintf(os.Stdout, "├%s┤\n", hLine)

	row := func(label, value, color string) {
		// Build uncolored text for width calculations.
		baseLabel := label
		maxValWidth := inner - (len(" ") + len(baseLabel) + len(" : "))
		if maxValWidth < 0 {
			maxValWidth = 0
		}
		if len(value) > maxValWidth {
			value = value[:maxValWidth]
		}

		visible := fmt.Sprintf(" %s : %s", baseLabel, value)
		if len(visible) > inner {
			visible = visible[:inner]
		}
		padding := inner - len(visible)
		if padding < 0 {
			padding = 0
		}

		labelPart := baseLabel
		if color != "" {
			labelPart = color + baseLabel + colorReset
		}
		colored := fmt.Sprintf(" %s : %s", labelPart, value)

		fmt.Fprintf(os.Stdout, "│%s%s│\n", colored, strings.Repeat(" ", padding))
	}

	row("Total Requests", fmt.Sprintf("%d", snap.TotalRequests), "")
	row("Successes", fmt.Sprintf("%d", snap.Successes), colorGreen)
	row("Errors", fmt.Sprintf("%d", snap.Errors), colorRed)
	row("Requests / sec", fmt.Sprintf("%.2f", snap.RequestsPerS), colorCyan)
	row("P50 Latency", snap.LatencyP50.String(), "")
	row("P95 Latency", snap.LatencyP95.String(), "")
	row("P99 Latency", snap.LatencyP99.String(), "")

	fmt.Fprintf(os.Stdout, "└%s┘\n", hLine)
	fmt.Fprintf(os.Stdout, "%sDone.%s\n", colorDim, colorReset)
}
