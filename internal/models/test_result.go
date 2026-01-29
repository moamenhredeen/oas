package models

import "time"

// TestResult represents the result of testing a single API endpoint
type TestResult struct {
	// Operation details
	Path        string `json:"path"`
	Method      string `json:"method"`
	OperationID string `json:"operation_id,omitempty"`

	// Test status
	Passed bool   `json:"passed"`
	Error  string `json:"error,omitempty"`

	// Response details
	StatusCode   int           `json:"status_code"`
	ResponseTime time.Duration `json:"response_time_ns"`

	// Validation details
	ValidationErrors []ValidationError `json:"validation_errors,omitempty"`
}

// ValidationError represents a specific validation failure
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// TestSummary represents the overall test results
type TestSummary struct {
	TotalTests int          `json:"total_tests"`
	Passed     int          `json:"passed"`
	Failed     int          `json:"failed"`
	Results    []TestResult `json:"results"`
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
