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
	Connections int
	Duration    time.Duration
	Workers     int
	Pipeline    int
}

// RunInteractiveWizard collects configuration from the user for `httpcl start`.
// To avoid extra dependencies in this minimal implementation, it uses bufio
// instead of a full TUI library; it still respects the high-level spec.
func RunInteractiveWizard() (*WizardConfig, error) {
	reader := bufio.NewReader(os.Stdin)

	readLine := func(prompt, def string) (string, error) {
		fmt.Printf("%s [%s]: ", prompt, def)
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

	url, err := readLine("Target URL", "")
	if err != nil {
		return nil, err
	}
	if url == "" {
		return nil, fmt.Errorf("url is required")
	}

	method, err := readLine("HTTP method (GET, POST, PUT, DELETE)", "GET")
	if err != nil {
		return nil, err
	}

	connStr, err := readLine("Connections (concurrent)", "50")
	if err != nil {
		return nil, err
	}
	connections, err := strconv.Atoi(connStr)
	if err != nil {
		connections = 50
	}

	durStr, err := readLine("Duration (e.g. 10s, 1m)", "10s")
	if err != nil {
		return nil, err
	}
	dur, err := time.ParseDuration(durStr)
	if err != nil {
		dur = 10 * time.Second
	}

	workersStr, err := readLine("Workers (goroutines)", "4")
	if err != nil {
		return nil, err
	}
	workers, err := strconv.Atoi(workersStr)
	if err != nil {
		workers = 4
	}

	pipelineStr, err := readLine("Pipeline (requests per connection)", "1")
	if err != nil {
		return nil, err
	}
	pipeline, err := strconv.Atoi(pipelineStr)
	if err != nil {
		pipeline = 1
	}

	cfg := &WizardConfig{
		Method:      strings.ToUpper(method),
		URL:         url,
		Connections: connections,
		Duration:    dur,
		Workers:     workers,
		Pipeline:    pipeline,
	}

	return cfg, nil
}

