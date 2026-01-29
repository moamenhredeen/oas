package models

import "time"

// TestResult represents the result of testing a single API endpoint
type TestResult struct {
	// Operation details
	Path        string
	Method      string
	OperationID string

	// Test status
	Passed bool
	Error  string

	// Response details
	StatusCode   int
	ResponseTime time.Duration

	// Validation details
	ValidationErrors []ValidationError
}

// ValidationError represents a specific validation failure
type ValidationError struct {
	Field   string
	Message string
}

// TestSummary represents the overall test results
type TestSummary struct {
	TotalTests int
	Passed     int
	Failed     int
	Results    []TestResult
}

// AddResult adds a test result to the summary
func (s *TestSummary) AddResult(result TestResult) {
	s.TotalTests++
	s.Results = append(s.Results, result)
	if result.Passed {
		s.Passed++
	} else {
		s.Failed++
	}
}
