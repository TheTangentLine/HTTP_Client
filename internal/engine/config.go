package engine

import "time"

// Config holds the runtime configuration for a benchmark run.
type Config struct {
	Method      string
	URL         string
	Connections int
	Duration    time.Duration
	Workers     int
	Pipeline    int
}

