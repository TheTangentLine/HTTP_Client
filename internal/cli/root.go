package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/thetangentline/httpcl/internal/engine"
	"github.com/thetangentline/httpcl/internal/ui"
)

var rootCmd = &cobra.Command{
	Use:   "httpcl",
	Short: "httpcl is an HTTP benchmarking tool",
}

// Global/direct run flags
var (
	flagMethod      string
	flagURL         string
	flagConnections int
	flagDuration    time.Duration
	flagWorkers     int
	flagPipeline    int
)

func init() {
	// start (interactive) command
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start interactive benchmark wizard",
		RunE: func(cmd *cobra.Command, args []string) error {
			wcfg, err := ui.RunInteractiveWizard()
			if err != nil {
				return err
			}
			cfg := engine.Config{
				Method:      wcfg.Method,
				URL:         wcfg.URL,
				Connections: wcfg.Connections,
				Duration:    wcfg.Duration,
				Workers:     wcfg.Workers,
				Pipeline:    wcfg.Pipeline,
			}
			return runBenchmark(cfg)
		},
	}

	// run (direct) command
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run benchmark with flags",
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagURL == "" {
				return fmt.Errorf("url is required (use -u or --url)")
			}

			cfg := engine.Config{
				Method:      flagMethod,
				URL:         flagURL,
				Connections: flagConnections,
				Duration:    flagDuration,
				Workers:     flagWorkers,
				Pipeline:    flagPipeline,
			}

			return runBenchmark(cfg)
		},
	}

	runCmd.Flags().StringVarP(&flagMethod, "method", "m", "GET", "HTTP method")
	runCmd.Flags().StringVarP(&flagURL, "url", "u", "", "Target URL")
	runCmd.Flags().IntVarP(&flagConnections, "connections", "c", 10, "Number of concurrent persistent connections")
	runCmd.Flags().DurationVarP(&flagDuration, "duration", "d", 10*time.Second, "Total test duration (e.g. 10s, 2m, 1h)")
	runCmd.Flags().IntVarP(&flagWorkers, "workers", "w", 1, "Number of CPU workers/goroutines to spawn")
	runCmd.Flags().IntVarP(&flagPipeline, "pipeline", "p", 1, "Number of pipelined requests per connection")

	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(runCmd)
}

// Execute runs the root cobra command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// runBenchmark is a thin wrapper to wire engine and UI.
func runBenchmark(cfg engine.Config) error {
	renderer := ui.NewRenderer()
	orch := engine.NewOrchestrator(cfg, renderer)
	return orch.Run()
}
