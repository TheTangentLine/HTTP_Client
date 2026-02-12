package engine

import "time"

// Config holds the runtime configuration for a benchmark run.
type Config struct {
	Method      string
	URL         string
	Body        []byte // optional; used for POST, PUT, PATCH
	Connections int
	Duration    time.Duration
	Workers     int
	Pipeline    int
}

