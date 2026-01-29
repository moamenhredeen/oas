/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/moamenhredeen/oas/internal/models"
	"github.com/moamenhredeen/oas/internal/parser"
	"github.com/moamenhredeen/oas/internal/tester"
	"github.com/spf13/cobra"
)

var (
	serverURL string
	filter    string
	tags      []string
	verbose   bool
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

		// Run tests
		testRunner := tester.NewTester()
		summary := testRunner.TestOperations(filteredOps, p)

		// Display results
		displayResults(summary, verbose)
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

func displayResults(summary models.TestSummary, verbose bool) {
	fmt.Println("\n=== Test Results ===")
	fmt.Printf("Total Tests: %d\n", summary.TotalTests)
	fmt.Printf("Passed: %d\n", summary.Passed)
	fmt.Printf("Failed: %d\n", summary.Failed)
	fmt.Println()

	if verbose {
		for _, result := range summary.Results {
			status := "✓ PASS"
			if !result.Passed {
				status = "✗ FAIL"
			}

			fmt.Printf("%s %s %s\n", status, result.Method, result.Path)
			if result.OperationID != "" {
				fmt.Printf("  Operation ID: %s\n", result.OperationID)
			}
			fmt.Printf("  Status Code: %d\n", result.StatusCode)
			fmt.Printf("  Response Time: %v\n", result.ResponseTime)

			if !result.Passed {
				if result.Error != "" {
					fmt.Printf("  Error: %s\n", result.Error)
				}
				if len(result.ValidationErrors) > 0 {
					fmt.Printf("  Validation Errors:\n")
					for _, ve := range result.ValidationErrors {
						fmt.Printf("    - %s: %s\n", ve.Field, ve.Message)
					}
				}
			}
			fmt.Println()
		}
	} else {
		// Simple output
		for _, result := range summary.Results {
			status := "PASS"
			if !result.Passed {
				status = "FAIL"
			}
			fmt.Printf("%s %s %s", status, result.Method, result.Path)
			if !result.Passed && result.Error != "" {
				fmt.Printf(" - %s", result.Error)
			}
			fmt.Println()
		}
	}

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
}
