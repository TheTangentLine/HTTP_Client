package ui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// WizardConfig is a minimal configuration struct produced by the interactive
// wizard. It is intentionally decoupled from the engine package to avoid
// import cycles; the CLI layer adapts it into engine.Config.
type WizardConfig struct {
	Method      string
	URL         string
	Body        []byte
	Connections int
	Duration    time.Duration
	Workers     int
	Pipeline    int
}

// RunInteractiveWizard collects configuration from the user for `httpcl start`.
func RunInteractiveWizard() (*WizardConfig, error) {
	reader := bufio.NewReader(os.Stdin)

	printWizardHeader()

	// Prompt: bold label, then dim "(required)" or "[default: X]", then ": "
	promptWithDefault := func(label, def string, required bool) (string, error) {
		if required {
			fmt.Printf("%s%s%s %s(required)%s: ", colorBold, label, colorReset, colorDim, colorReset)
		} else {
			fmt.Printf("%s%s%s %s[default: %s]%s: ", colorBold, label, colorReset, colorDim, def, colorReset)
		}
		text, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		text = strings.TrimSpace(text)
		if text == "" {
			text = def
		}
		return text, nil
	}

	url, err := promptWithDefault("Target URL", "", true)
	if err != nil {
		return nil, err
	}
	if url == "" {
		return nil, fmt.Errorf("url is required")
	}

	method, err := promptWithDefault("HTTP method (GET, POST, PUT, DELETE)", "GET", false)
	if err != nil {
		return nil, err
	}
	method = strings.ToUpper(method)

	connStr, err := promptWithDefault("Connections (concurrent)", "50", false)
	if err != nil {
		return nil, err
	}
	connections, _ := strconv.Atoi(connStr)
	if connections <= 0 {
		connections = 50
	}

	durStr, err := promptWithDefault("Duration (e.g. 10s, 1m)", "10s", false)
	if err != nil {
		return nil, err
	}
	dur, err := time.ParseDuration(durStr)
	if err != nil {
		dur = 10 * time.Second
	}

	workersStr, err := promptWithDefault("Workers (goroutines)", "4", false)
	if err != nil {
		return nil, err
	}
	workers, _ := strconv.Atoi(workersStr)
	if workers <= 0 {
		workers = 4
	}

	pipelineStr, err := promptWithDefault("Pipeline (requests per connection)", "1", false)
	if err != nil {
		return nil, err
	}
	pipeline, _ := strconv.Atoi(pipelineStr)
	if pipeline <= 0 {
		pipeline = 1
	}

	var body []byte
	if methodHasBody(method) {
		bodyStr, err := promptWithDefault("Request body (optional, for POST/PUT/PATCH)", "", false)
		if err != nil {
			return nil, err
		}
		if bodyStr != "" {
			body = []byte(bodyStr)
		}
	}

	cfg := &WizardConfig{
		Method:      method,
		URL:         url,
		Body:        body,
		Connections: connections,
		Duration:    dur,
		Workers:     workers,
		Pipeline:    pipeline,
	}

	return cfg, nil
}

func methodHasBody(m string) bool {
	switch m {
	case "POST", "PUT", "PATCH":
		return true
	default:
		return false
	}
}

// printWizardHeader renders a simple, responsive ASCII header for the wizard.
func printWizardHeader() {
	width := 80
	if w := os.Getenv("COLUMNS"); w != "" {
		if v, err := strconv.Atoi(w); err == nil && v > 20 {
			width = v
		}
	}
	if width > 64 {
		width = 64
	}
	inner := width - 2
	hLine := strings.Repeat("─", inner)

	// Pad so each line content (visible) + padding = inner for aligned right border.
	pad := func(n int) int {
		if n < 0 {
			return 0
		}
		return n
	}
	line1Len := 1 + 24 // " " + "httpcl interactive setup"
	line2Len := 1 + 49 // " " + "Answer the following to configure your benchmark."
	fmt.Println()
	fmt.Printf("┌%s┐\n", hLine)
	fmt.Printf("│ %shttpcl interactive setup%s%s│\n", colorBold, colorReset, strings.Repeat(" ", pad(inner-line1Len)))
	fmt.Printf("├%s┤\n", hLine)
	fmt.Printf("│ %sAnswer the following to configure your benchmark.%s%s│\n", colorDim, colorReset, strings.Repeat(" ", pad(inner-line2Len)))
	fmt.Printf("└%s┘\n", hLine)
	fmt.Println()
}
