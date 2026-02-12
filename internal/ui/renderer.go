package ui

import (
	"fmt"
	"math"
	"os"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/thetangentline/httpcl/internal/stats"
)

// humanizeBytes formats bytes as KB, MB, or GB.
func humanizeBytes(b float64) string {
	if math.IsNaN(b) || b < 0 {
		return "0 B"
	}
	switch {
	case b >= 1e9:
		return fmt.Sprintf("%.2f GB", b/1e9)
	case b >= 1e6:
		return fmt.Sprintf("%.2f MB", b/1e6)
	case b >= 1e3:
		return fmt.Sprintf("%.2f KB", b/1e3)
	default:
		return fmt.Sprintf("%.0f B", b)
	}
}

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

// visibleLen returns the rune length of s without ANSI escape sequences.
func visibleLen(s string) int {
	n := 0
	i := 0
	for i < len(s) {
		if s[i] == '\033' && i+1 < len(s) && s[i+1] == '[' {
			i += 2
			for i < len(s) && (s[i] < 0x40 || s[i] == ';') {
				i++
			}
			if i < len(s) {
				i++
			}
			continue
		}
		n++
		i++
	}
	return n
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
		snap.RequestsPerSAvg,
		snap.LatencyP50.Truncate(10*time.Microsecond),
	)

	line = truncateToWidth(line, termWidth())

	fmt.Fprint(os.Stdout, line)
	r.lastLineLen = len(line)
}

func (r *asciiRenderer) RenderFinal(snap stats.Snapshot) {
	r.clearLine()
	fmt.Fprintln(os.Stdout)

	padTo := func(s string, n int) string {
		need := n - visibleLen(s)
		if need <= 0 {
			return s
		}
		return s + strings.Repeat(" ", need)
	}
	cellPad := 2 // left padding inside each cell
	cell := func(s string, w int) string {
		inner := strings.Repeat(" ", cellPad) + s
		return padTo(inner, w)
	}
	latMs := func(d time.Duration) string { return fmt.Sprintf("%d ms", d.Milliseconds()) }

	// Grid column widths: Stat, then 7 metric columns
	cw := []int{12, 12, 12, 12, 12, 12, 12, 12}
	gridTop := func() {
		fmt.Fprintf(os.Stdout, "┌%s┬%s┬%s┬%s┬%s┬%s┬%s┬%s┐\n",
			strings.Repeat("─", cw[0]), strings.Repeat("─", cw[1]), strings.Repeat("─", cw[2]),
			strings.Repeat("─", cw[3]), strings.Repeat("─", cw[4]), strings.Repeat("─", cw[5]),
			strings.Repeat("─", cw[6]), strings.Repeat("─", cw[7]))
	}
	gridMid := func() {
		fmt.Fprintf(os.Stdout, "├%s┼%s┼%s┼%s┼%s┼%s┼%s┼%s┤\n",
			strings.Repeat("─", cw[0]), strings.Repeat("─", cw[1]), strings.Repeat("─", cw[2]),
			strings.Repeat("─", cw[3]), strings.Repeat("─", cw[4]), strings.Repeat("─", cw[5]),
			strings.Repeat("─", cw[6]), strings.Repeat("─", cw[7]))
	}
	gridBot := func() {
		fmt.Fprintf(os.Stdout, "└%s┴%s┴%s┴%s┴%s┴%s┴%s┴%s┘\n",
			strings.Repeat("─", cw[0]), strings.Repeat("─", cw[1]), strings.Repeat("─", cw[2]),
			strings.Repeat("─", cw[3]), strings.Repeat("─", cw[4]), strings.Repeat("─", cw[5]),
			strings.Repeat("─", cw[6]), strings.Repeat("─", cw[7]))
	}
	gridRow := func(a1, a2, a3, a4, a5, a6, a7, a8 string) {
		fmt.Fprintf(os.Stdout, "│%s│%s│%s│%s│%s│%s│%s│%s│\n",
			cell(a1, cw[0]), cell(a2, cw[1]), cell(a3, cw[2]), cell(a4, cw[3]),
			cell(a5, cw[4]), cell(a6, cw[5]), cell(a7, cw[6]), cell(a8, cw[7]))
	}

	fmt.Fprintf(os.Stdout, "%s%s%s\n", colorBold, "Latency (ms)", colorReset)
	gridTop()
	gridRow(colorCyan+"Stat"+colorReset, colorCyan+"2.5%"+colorReset, colorCyan+"50%"+colorReset, colorCyan+"97.5%"+colorReset, colorCyan+"99%"+colorReset, colorCyan+"Avg"+colorReset, colorCyan+"Stdev"+colorReset, colorCyan+"Max"+colorReset)
	gridMid()
	gridRow("Latency", latMs(snap.LatencyP25), latMs(snap.LatencyP50), latMs(snap.LatencyP975), latMs(snap.LatencyP99), latMs(snap.LatencyAvg), latMs(snap.LatencyStdev), latMs(snap.LatencyMax))
	gridBot()
	fmt.Fprintln(os.Stdout)

	fmt.Fprintf(os.Stdout, "%s%s%s\n", colorBold, "Throughput", colorReset)
	gridTop()
	gridRow(colorCyan+"Stat"+colorReset, colorCyan+"1%"+colorReset, colorCyan+"2.5%"+colorReset, colorCyan+"50%"+colorReset, colorCyan+"97.5%"+colorReset, colorCyan+"Avg"+colorReset, colorCyan+"Stdev"+colorReset, colorCyan+"Min"+colorReset)
	gridMid()
	gridRow("Req/Sec", fmt.Sprintf("%.0f", snap.RPSP01), fmt.Sprintf("%.0f", snap.RPSP025), fmt.Sprintf("%.0f", snap.RPSP50), fmt.Sprintf("%.0f", snap.RPSP975), fmt.Sprintf("%.2f", snap.RequestsPerSAvg), fmt.Sprintf("%.0f", snap.RPSStdev), fmt.Sprintf("%.0f", snap.RPSMin))
	gridRow("Bytes/Sec", humanizeBytes(snap.BytesPerSP01), humanizeBytes(snap.BytesPerSP025), humanizeBytes(snap.BytesPerSP50), humanizeBytes(snap.BytesPerSP975), humanizeBytes(snap.BytesPerSAvg), humanizeBytes(snap.BytesPerSStdev), humanizeBytes(snap.BytesPerSMin))
	gridBot()
	fmt.Fprintln(os.Stdout)

	width := termWidth()
	if width <= 0 {
		width = 80
	}
	if width > 72 {
		width = 72
	}
	inner := width - 2
	hLine := strings.Repeat("─", inner)

	fmt.Fprintf(os.Stdout, "┌%s┐\n", hLine)
	fmt.Fprintf(os.Stdout, "│%s│\n", padTo(" "+colorBold+"Summary"+colorReset, inner))
	fmt.Fprintf(os.Stdout, "├%s┤\n", hLine)

	summaryRow := func(label, value string, valueColor string) {
		if valueColor == "" {
			valueColor = colorReset
		}
		s := " " + colorBold + label + colorReset + " : " + valueColor + value + colorReset
		fmt.Fprintf(os.Stdout, "│%s│\n", padTo(s, inner))
	}
	summaryRowColored := func(label, value string, rowColor string) {
		s := " " + rowColor + colorBold + label + colorReset + rowColor + " : " + value + colorReset
		fmt.Fprintf(os.Stdout, "│%s│\n", padTo(s, inner))
	}

	summaryRow("Total Requests", fmt.Sprintf("%d", snap.TotalRequests), "")
	summaryRowColored("Successes", fmt.Sprintf("%d", snap.Successes), colorGreen)
	summaryRowColored("Errors", fmt.Sprintf("%d", snap.Errors), colorRed)
	summaryRow("Duration", snap.Duration.String(), "")
	summaryRow("Data sent", humanizeBytes(float64(snap.TotalBytesSent)), colorCyan)
	summaryRow("Data received", humanizeBytes(float64(snap.TotalBytesRecv)), colorCyan)

	fmt.Fprintf(os.Stdout, "└%s┘\n", hLine)
	fmt.Fprintf(os.Stdout, "%sDone.%s\n", colorDim, colorReset)
}
