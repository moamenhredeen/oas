package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/moamenhredeen/oas/internal/models"
)

// Format represents the output format type
type Format string

const (
	FormatJSON Format = "json"
	FormatCSV  Format = "csv"
)

// ExportTestSummary exports test results to the specified format
func ExportTestSummary(summary models.TestSummary, format Format, filePath string) error {
	w, closer, err := getWriter(filePath)
	if err != nil {
		return err
	}
	if closer != nil {
		defer closer.Close()
	}

	switch format {
	case FormatJSON:
		return exportTestJSON(w, summary)
	case FormatCSV:
		return exportTestCSV(w, summary)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// ExportBenchmarkSummary exports benchmark results to the specified format
func ExportBenchmarkSummary(summary models.BenchmarkSummary, format Format, filePath string) error {
	w, closer, err := getWriter(filePath)
	if err != nil {
		return err
	}
	if closer != nil {
		defer closer.Close()
	}

	switch format {
	case FormatJSON:
		return exportBenchmarkJSON(w, summary)
	case FormatCSV:
		return exportBenchmarkCSV(w, summary)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// getWriter returns an io.Writer for output (stdout or file)
func getWriter(filePath string) (io.Writer, io.Closer, error) {
	if filePath == "" {
		return os.Stdout, nil, nil
	}

	f, err := os.Create(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create output file: %w", err)
	}
	return f, f, nil
}

// exportTestJSON exports test results as JSON
func exportTestJSON(w io.Writer, summary models.TestSummary) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(summary)
}

// exportTestCSV exports test results as CSV
func exportTestCSV(w io.Writer, summary models.TestSummary) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	// Write header
	header := []string{
		"method", "path", "operation_id", "passed", "status_code",
		"response_time_ms", "error",
	}
	if err := cw.Write(header); err != nil {
		return err
	}

	// Write rows
	for _, r := range summary.Results {
		row := []string{
			r.Method,
			r.Path,
			r.OperationID,
			strconv.FormatBool(r.Passed),
			strconv.Itoa(r.StatusCode),
			fmt.Sprintf("%.2f", float64(r.ResponseTime.Milliseconds())),
			r.Error,
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}

	return cw.Error()
}

// exportBenchmarkJSON exports benchmark results as JSON
func exportBenchmarkJSON(w io.Writer, summary models.BenchmarkSummary) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(summary)
}

// exportBenchmarkCSV exports benchmark results as CSV
func exportBenchmarkCSV(w io.Writer, summary models.BenchmarkSummary) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	// Write header
	header := []string{
		"method", "path", "operation_id", "iterations", "concurrency",
		"min_ms", "max_ms", "avg_ms", "p50_ms", "p90_ms", "p99_ms",
		"requests_per_sec", "success_count", "error_count", "error_rate",
	}
	if err := cw.Write(header); err != nil {
		return err
	}

	// Write rows
	for _, r := range summary.Results {
		row := []string{
			r.Method,
			r.Path,
			r.OperationID,
			strconv.Itoa(r.Iterations),
			strconv.Itoa(r.Concurrency),
			fmt.Sprintf("%.2f", float64(r.MinTime.Microseconds())/1000),
			fmt.Sprintf("%.2f", float64(r.MaxTime.Microseconds())/1000),
			fmt.Sprintf("%.2f", float64(r.AvgTime.Microseconds())/1000),
			fmt.Sprintf("%.2f", float64(r.P50Time.Microseconds())/1000),
			fmt.Sprintf("%.2f", float64(r.P90Time.Microseconds())/1000),
			fmt.Sprintf("%.2f", float64(r.P99Time.Microseconds())/1000),
			fmt.Sprintf("%.2f", r.RequestsPerSec),
			strconv.Itoa(r.SuccessCount),
			strconv.Itoa(r.ErrorCount),
			fmt.Sprintf("%.2f", r.ErrorRate),
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}

	return cw.Error()
}

// ParseFormat parses a string into a Format, returning error if invalid
func ParseFormat(s string) (Format, error) {
	switch s {
	case "json":
		return FormatJSON, nil
	case "csv":
		return FormatCSV, nil
	default:
		return "", fmt.Errorf("invalid format '%s': must be 'json' or 'csv'", s)
	}
}
