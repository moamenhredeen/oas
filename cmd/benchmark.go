/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/moamenhredeen/oas/internal/benchmarker"
	"github.com/moamenhredeen/oas/internal/models"
	"github.com/moamenhredeen/oas/internal/output"
	"github.com/moamenhredeen/oas/internal/parser"
	"github.com/spf13/cobra"
)

var (
	// Benchmark-specific flags
	benchIterations   int
	benchConcurrency  int
	benchWarmup       int
	benchRateLimit    float64
	benchTimeout      int
	benchNoKeepAlive  bool
	benchOutputFormat string
	benchOutputFile   string

	// Shared flags (reuse serverURL, filter, tags, verbose from test.go)

	// Color helpers
	cyan   = color.New(color.FgCyan, color.Bold).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
	white  = color.New(color.FgWhite, color.Bold).SprintFunc()
)

// benchmarkCmd represents the benchmark command
var benchmarkCmd = &cobra.Command{
	Use:   "benchmark [openapi-spec-file]",
	Short: "Benchmark API performance",
	Long: `Benchmark API endpoints by measuring response times and throughput.

This command runs multiple iterations of each API request and collects
performance metrics including latency percentiles (p50, p90, p99),
requests per second, and error rates.

Examples:
  # Basic benchmark with defaults (100 iterations, 1 concurrent)
  oas benchmark api-spec.json

  # High-load benchmark with concurrency
  oas benchmark api-spec.json -n 1000 -c 10

  # Rate-limited benchmark
  oas benchmark api-spec.json -n 500 --rate 50

  # Export results to JSON
  oas benchmark api-spec.json -o json --output-file results.json`,
	Args: cobra.ExactArgs(1),
	Run:  runBenchmark,
}

func runBenchmark(cmd *cobra.Command, args []string) {
	specFile := args[0]

	// Parse OpenAPI spec
	p, err := parser.ParseFile(specFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing OpenAPI file: %v\n", err)
		os.Exit(1)
	}

	// Get server URLs
	serverURLs, err := p.GetServerURLs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting server URLs: %v\n", err)
		os.Exit(1)
	}

	// Use provided server URL or first from spec
	baseURL := serverURL
	if baseURL == "" && len(serverURLs) > 0 {
		baseURL = serverURLs[0]
	}
	if baseURL == "" {
		baseURL = "http://localhost"
	}

	// Get all operations
	operations, err := p.GetOperations(baseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting operations: %v\n", err)
		os.Exit(1)
	}

	// Filter operations (reuse from test command)
	filteredOps := filterOperations(operations, filter, tags)

	if len(filteredOps) == 0 {
		fmt.Println("No operations found matching the criteria")
		os.Exit(0)
	}

	// Create benchmark configuration
	config := benchmarker.Config{
		Iterations:       benchIterations,
		Concurrency:      benchConcurrency,
		WarmupRuns:       benchWarmup,
		RateLimit:        benchRateLimit,
		Timeout:          time.Duration(benchTimeout) * time.Second,
		DisableKeepAlive: benchNoKeepAlive,
	}

	// Print benchmark info
	fmt.Printf("\n%s\n", white("=== Benchmark Configuration ==="))
	fmt.Printf("Endpoints:   %d\n", len(filteredOps))
	fmt.Printf("Iterations:  %d per endpoint\n", config.Iterations)
	fmt.Printf("Concurrency: %d\n", config.Concurrency)
	fmt.Printf("Warmup:      %d iterations\n", config.WarmupRuns)
	if config.RateLimit > 0 {
		fmt.Printf("Rate Limit:  %.0f req/sec\n", config.RateLimit)
	}
	fmt.Printf("Timeout:     %v\n", config.Timeout)
	fmt.Printf("Keep-Alive:  %v\n", !config.DisableKeepAlive)
	fmt.Println()

	// Create benchmarker
	bench := benchmarker.NewBenchmarker(config)

	// Setup context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n\nBenchmark interrupted, generating partial results...")
		cancel()
	}()

	var s *spinner.Spinner
	var currentPhase string
	var phaseStartTime time.Time

	// Create event handler for live output
	onEvent := func(event benchmarker.BenchmarkEvent) {
		switch event.Type {
		case benchmarker.EventWarmupStarting:
			currentPhase = "warmup"
			phaseStartTime = time.Now()
			if isTTY {
				s = spinner.New(spinner.CharSets[14], 100*time.Millisecond)
				s.Suffix = fmt.Sprintf(" [%d/%d] %s %s - Warming up...",
					event.Index+1, event.Total, event.Operation.Method, event.Operation.Path)
				s.Start()
			} else {
				fmt.Printf("[%d/%d] %s %s - Warming up (%d iterations)...\n",
					event.Index+1, event.Total, event.Operation.Method, event.Operation.Path, event.MaxIter)
			}

		case benchmarker.EventWarmupProgress:
			if isTTY && s != nil {
				s.Suffix = fmt.Sprintf(" [%d/%d] %s %s - Warmup %d/%d",
					event.Index+1, event.Total, event.Operation.Method, event.Operation.Path,
					event.Progress, event.MaxIter)
			}

		case benchmarker.EventWarmupCompleted:
			if isTTY && s != nil {
				s.Stop()
			}
			elapsed := time.Since(phaseStartTime)
			fmt.Printf("[%d/%d] %s Warmup completed in %v\n",
				event.Index+1, event.Total, yellow("●"), elapsed.Round(time.Millisecond))

		case benchmarker.EventBenchmarkStarting:
			currentPhase = "benchmark"
			phaseStartTime = time.Now()
			if isTTY {
				s = spinner.New(spinner.CharSets[14], 100*time.Millisecond)
				s.Suffix = fmt.Sprintf(" [%d/%d] %s %s - Benchmarking 0/%d...",
					event.Index+1, event.Total, event.Operation.Method, event.Operation.Path, event.MaxIter)
				s.Start()
			} else {
				fmt.Printf("[%d/%d] %s %s - Running benchmark (%d iterations)...\n",
					event.Index+1, event.Total, event.Operation.Method, event.Operation.Path, event.MaxIter)
			}

		case benchmarker.EventBenchmarkProgress:
			if isTTY && s != nil {
				avgMs := float64(event.RunningAvg.Microseconds()) / 1000
				s.Suffix = fmt.Sprintf(" [%d/%d] %s %s - %d/%d (avg: %.1fms, %.1f req/s, %d errors)",
					event.Index+1, event.Total, event.Operation.Method, event.Operation.Path,
					event.Progress, event.MaxIter, avgMs, event.RunningReqSec, event.ErrorCount)
			}

		case benchmarker.EventBenchmarkCompleted:
			if isTTY && s != nil {
				s.Stop()
			}

			result := event.Result
			elapsed := time.Since(phaseStartTime)
			prefix := fmt.Sprintf("[%d/%d]", event.Index+1, event.Total)

			// Status indicator based on error rate
			var status string
			if result.ErrorRate == 0 {
				status = green("✓")
			} else if result.ErrorRate < 5 {
				status = yellow("●")
			} else {
				status = red("✗")
			}

			fmt.Printf("%s %s %s %s\n", prefix, status, result.Method, result.Path)

			// Always show key metrics
			avgMs := float64(result.AvgTime.Microseconds()) / 1000
			p99Ms := float64(result.P99Time.Microseconds()) / 1000
			fmt.Printf("    %s avg: %.2fms | p99: %.2fms | %.1f req/s | errors: %d (%.1f%%)\n",
				cyan("→"),
				avgMs, p99Ms, result.RequestsPerSec,
				result.ErrorCount, result.ErrorRate)

			// Verbose output: show all details
			if verbose {
				minMs := float64(result.MinTime.Microseconds()) / 1000
				maxMs := float64(result.MaxTime.Microseconds()) / 1000
				p50Ms := float64(result.P50Time.Microseconds()) / 1000
				p90Ms := float64(result.P90Time.Microseconds()) / 1000

				fmt.Printf("    Latency:  min=%.2fms | p50=%.2fms | p90=%.2fms | max=%.2fms\n",
					minMs, p50Ms, p90Ms, maxMs)
				fmt.Printf("    Duration: %v | Success: %d | Errors: %d\n",
					elapsed.Round(time.Millisecond), result.SuccessCount, result.ErrorCount)

				if len(result.StatusCodes) > 0 {
					var codes []string
					for code, count := range result.StatusCodes {
						codes = append(codes, fmt.Sprintf("%d:%d", code, count))
					}
					fmt.Printf("    Status codes: %s\n", strings.Join(codes, ", "))
				}

				if len(result.SampleErrors) > 0 {
					fmt.Printf("    Sample errors:\n")
					for _, e := range result.SampleErrors {
						fmt.Printf("      - %s\n", red(e))
					}
				}
			}
		}
		_ = currentPhase // silence unused warning
	}

	// Run benchmarks
	summary := bench.BenchmarkOperations(ctx, filteredOps, p, onEvent)

	// Handle output format
	if benchOutputFormat != "" {
		format, err := output.ParseFormat(benchOutputFormat)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if err := output.ExportBenchmarkSummary(summary, format, benchOutputFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error exporting results: %v\n", err)
			os.Exit(1)
		}

		// If writing to file, still show summary
		if benchOutputFile != "" {
			fmt.Printf("\nResults exported to: %s\n", benchOutputFile)
			displayBenchmarkSummary(summary)
		}
		// If writing to stdout, skip display (already output)
		return
	}

	// Display summary
	displayBenchmarkSummary(summary)
}

func displayBenchmarkSummary(summary models.BenchmarkSummary) {
	fmt.Println()
	fmt.Printf("%s\n", white("=== Benchmark Summary ==="))
	fmt.Printf("Total Endpoints:    %d\n", summary.TotalEndpoints)
	fmt.Printf("Total Requests:     %d\n", summary.TotalRequests)
	fmt.Printf("Total Duration:     %v\n", summary.TotalDuration.Round(time.Millisecond))
	fmt.Printf("Overall Throughput: %s\n", cyan(fmt.Sprintf("%.1f req/sec", summary.OverallReqsPerSec)))
	fmt.Println()

	// Latency summary
	fmt.Printf("%s\n", white("Latency Overview:"))
	fmt.Printf("  Min: %.2fms\n", float64(summary.OverallMinTime.Microseconds())/1000)
	fmt.Printf("  Avg: %.2fms\n", float64(summary.OverallAvgTime.Microseconds())/1000)
	fmt.Printf("  Max: %.2fms\n", float64(summary.OverallMaxTime.Microseconds())/1000)
	fmt.Println()

	// Error summary
	if summary.TotalErrors > 0 {
		fmt.Printf("%s\n", white("Error Summary:"))
		fmt.Printf("  Total Errors: %s\n", red(summary.TotalErrors))
		fmt.Printf("  Error Rate:   %s\n", red(fmt.Sprintf("%.2f%%", summary.OverallErrorRate)))
		fmt.Println()
	} else {
		fmt.Printf("Errors: %s\n", green("0"))
		fmt.Println()
	}

	// Per-endpoint table (if verbose or few endpoints)
	if verbose || len(summary.Results) <= 10 {
		fmt.Printf("%s\n", white("Per-Endpoint Results:"))
		fmt.Printf("%-8s %-40s %10s %10s %10s %10s\n",
			"METHOD", "PATH", "AVG(ms)", "P99(ms)", "REQ/S", "ERR%")
		fmt.Println(strings.Repeat("-", 90))

		for _, r := range summary.Results {
			path := r.Path
			if len(path) > 38 {
				path = path[:35] + "..."
			}
			fmt.Printf("%-8s %-40s %10.2f %10.2f %10.1f %10.1f\n",
				r.Method, path,
				float64(r.AvgTime.Microseconds())/1000,
				float64(r.P99Time.Microseconds())/1000,
				r.RequestsPerSec,
				r.ErrorRate)
		}
	}
}

func init() {
	rootCmd.AddCommand(benchmarkCmd)

	// Reuse shared flags from test command
	benchmarkCmd.Flags().StringVar(&serverURL, "server", "", "Override server URL from OpenAPI spec")
	benchmarkCmd.Flags().StringVar(&filter, "filter", "", "Filter endpoints by path pattern or operation ID")
	benchmarkCmd.Flags().StringSliceVar(&tags, "tags", []string{}, "Filter by OpenAPI tags")
	benchmarkCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed output")

	// Benchmark-specific flags
	benchmarkCmd.Flags().IntVarP(&benchIterations, "iterations", "n", 100, "Number of requests per endpoint")
	benchmarkCmd.Flags().IntVarP(&benchConcurrency, "concurrency", "c", 1, "Number of concurrent requests")
	benchmarkCmd.Flags().IntVarP(&benchWarmup, "warmup", "w", 5, "Number of warmup iterations (discarded from stats)")
	benchmarkCmd.Flags().Float64VarP(&benchRateLimit, "rate", "r", 0, "Max requests per second (0 = unlimited)")
	benchmarkCmd.Flags().IntVarP(&benchTimeout, "timeout", "t", 30, "Request timeout in seconds")
	benchmarkCmd.Flags().BoolVar(&benchNoKeepAlive, "no-keepalive", false, "Disable HTTP connection reuse")

	// Output flags
	benchmarkCmd.Flags().StringVarP(&benchOutputFormat, "output", "o", "", "Output format: json, csv")
	benchmarkCmd.Flags().StringVar(&benchOutputFile, "output-file", "", "Write output to file (default: stdout)")
}
