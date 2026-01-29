/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"github.com/moamenhredeen/oas/internal/models"
	"github.com/moamenhredeen/oas/internal/output"
	"github.com/moamenhredeen/oas/internal/parser"
	"github.com/moamenhredeen/oas/internal/tester"
	"github.com/spf13/cobra"
)

var (
	serverURL    string
	filter       string
	tags         []string
	verbose      bool
	outputFormat string
	outputFile   string
	timeout      int

	// Color helpers for output
	green = color.New(color.FgGreen, color.Bold).SprintFunc()
	red   = color.New(color.FgRed, color.Bold).SprintFunc()

	// Check if stdout is a terminal
	isTTY = isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
)

// testCmd represents the test command
var testCmd = &cobra.Command{
	Use:   "test [openapi-spec-file]",
	Short: "Test the APIs",
	Long:  `Test the APIs by providing the OpenAPI Specification file and the endpoints to test.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
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

		// Filter operations
		filteredOps := filterOperations(operations, filter, tags)

		if len(filteredOps) == 0 {
			fmt.Println("No operations found matching the criteria")
			os.Exit(0)
		}

		// Run tests with live output
		testRunner := tester.NewTester(time.Duration(timeout) * time.Second)
		var s *spinner.Spinner

		// Create event handler for live output
		onEvent := func(event tester.TestEvent) {
			switch event.Type {
			case tester.EventStarting:
				if isTTY {
					// Start spinner for TTY
					s = spinner.New(spinner.CharSets[14], 100*time.Millisecond)
					s.Suffix = fmt.Sprintf(" [%d/%d] Running %s %s...",
						event.Index+1, event.Total, event.Operation.Method, event.Operation.Path)
					s.Start()
				} else {
					// Simple output for non-TTY
					fmt.Printf("[%d/%d] Running %s %s...\n",
						event.Index+1, event.Total, event.Operation.Method, event.Operation.Path)
				}
			case tester.EventCompleted:
				if isTTY && s != nil {
					s.Stop()
				}

				result := event.Result
				prefix := fmt.Sprintf("[%d/%d]", event.Index+1, event.Total)

				if result.Passed {
					fmt.Printf("%s %s %s %s\n", prefix, green("✓ PASS"), result.Method, result.Path)
				} else {
					fmt.Printf("%s %s %s %s\n", prefix, red("✗ FAIL"), result.Method, result.Path)
				}

				// Verbose output: show details inline
				if verbose {
					if result.OperationID != "" {
						fmt.Printf("    Operation ID: %s\n", result.OperationID)
					}
					fmt.Printf("    Status Code: %d\n", result.StatusCode)
					fmt.Printf("    Response Time: %v\n", result.ResponseTime)

					if !result.Passed {
						if result.Error != "" {
							fmt.Printf("    Error: %s\n", red(result.Error))
						}
						if len(result.ValidationErrors) > 0 {
							fmt.Printf("    Validation Errors:\n")
							for _, ve := range result.ValidationErrors {
								fmt.Printf("      - %s: %s\n", ve.Field, red(ve.Message))
							}
						}
					}
				}
			}
		}

		summary := testRunner.TestOperations(filteredOps, p, onEvent)

		// Handle output format
		if outputFormat != "" {
			format, err := output.ParseFormat(outputFormat)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			if err := output.ExportTestSummary(summary, format, outputFile); err != nil {
				fmt.Fprintf(os.Stderr, "Error exporting results: %v\n", err)
				os.Exit(1)
			}

			// If writing to file, still show summary
			if outputFile != "" {
				fmt.Printf("\nResults exported to: %s\n", outputFile)
				displayResults(summary)
			}
			// If writing to stdout, skip display (already output)
			if summary.Failed > 0 {
				os.Exit(1)
			}
			return
		}

		// Display summary
		displayResults(summary)
	},
}

func filterOperations(operations []models.Operation, filterStr string, tagFilters []string) []models.Operation {
	var filtered []models.Operation

	for _, op := range operations {
		// Filter by path pattern or operation ID
		if filterStr != "" {
			if !strings.Contains(op.Path, filterStr) && !strings.Contains(op.OperationID, filterStr) {
				continue
			}
		}

		// Filter by tags
		if len(tagFilters) > 0 {
			found := false
			for _, filterTag := range tagFilters {
				for _, opTag := range op.Tags {
					if opTag == filterTag {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				continue
			}
		}

		filtered = append(filtered, op)
	}

	return filtered
}

func displayResults(summary models.TestSummary) {
	fmt.Println("\n=== Test Summary ===")
	fmt.Printf("Total Tests: %d\n", summary.TotalTests)
	fmt.Printf("Passed: %s\n", green(summary.Passed))
	fmt.Printf("Failed: %s\n", red(summary.Failed))

	// Exit with error code if any tests failed
	if summary.Failed > 0 {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(testCmd)

	testCmd.Flags().StringVar(&serverURL, "server", "", "Override server URL from OpenAPI spec")
	testCmd.Flags().StringVar(&filter, "filter", "", "Filter endpoints by path pattern or operation ID")
	testCmd.Flags().StringSliceVar(&tags, "tags", []string{}, "Filter by OpenAPI tags (can be specified multiple times)")
	testCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed output")
	testCmd.Flags().IntVarP(&timeout, "timeout", "t", 30, "Request timeout in seconds")
	testCmd.Flags().StringVarP(&outputFormat, "output", "o", "", "Output format: json, csv")
	testCmd.Flags().StringVar(&outputFile, "output-file", "", "Write output to file (default: stdout)")
}
