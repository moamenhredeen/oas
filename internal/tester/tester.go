package tester

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/moamenhredeen/oas/internal/models"
	"github.com/moamenhredeen/oas/internal/parser"
)

// EventType represents the type of test event
type EventType int

const (
	// EventStarting indicates a test is about to start
	EventStarting EventType = iota
	// EventCompleted indicates a test has completed
	EventCompleted
)

// TestEvent represents an event during test execution
type TestEvent struct {
	Type      EventType
	Operation models.Operation
	Result    *models.TestResult // nil for Starting events
	Index     int                // current test index (0-based)
	Total     int                // total number of tests
}

// OnTestEvent is a callback function for test events
type OnTestEvent func(event TestEvent)

// Tester executes API tests based on OpenAPI specifications
type Tester struct {
	requestBuilder *RequestBuilder
	validator      *Validator
	client         *http.Client
}

// NewTester creates a new tester instance with configurable timeout
func NewTester(timeout time.Duration) *Tester {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Tester{
		requestBuilder: NewRequestBuilder(),
		validator:      NewValidator(),
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// TestOperation tests a single API operation
func (t *Tester) TestOperation(op models.Operation, parser *parser.Parser) (models.TestResult, error) {
	result := models.TestResult{
		Path:        op.Path,
		Method:      op.Method,
		OperationID: op.OperationID,
		Passed:      false,
	}

	// Get operation details
	opDetails, err := parser.GetOperationDetails(op.Path, op.Method)
	if err != nil {
		result.Error = fmt.Sprintf("failed to get operation details: %v", err)
		return result, nil
	}

	// Build request
	req, err := t.requestBuilder.BuildRequest(opDetails, op.ServerURL)
	if err != nil {
		result.Error = fmt.Sprintf("failed to build request: %v", err)
		return result, nil
	}

	// Execute request
	startTime := time.Now()
	resp, err := t.client.Do(req)
	result.ResponseTime = time.Since(startTime)

	if err != nil {
		result.Error = fmt.Sprintf("request failed: %v", err)
		return result, nil
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode

	// Validate response
	validationErrors, err := t.validator.ValidateResponse(resp, opDetails)
	if err != nil {
		result.Error = fmt.Sprintf("validation error: %v", err)
		return result, nil
	}

	result.ValidationErrors = validationErrors

	// Check if validation passed
	if len(validationErrors) == 0 {
		result.Passed = true
	} else {
		var errorMsgs []string
		for _, ve := range validationErrors {
			errorMsgs = append(errorMsgs, fmt.Sprintf("%s: %s", ve.Field, ve.Message))
		}
		result.Error = fmt.Sprintf("validation failed: %s", strings.Join(errorMsgs, "; "))
	}

	return result, nil
}

// TestOperations tests multiple operations with optional live event reporting
func (t *Tester) TestOperations(operations []models.Operation, parser *parser.Parser, onEvent OnTestEvent) models.TestSummary {
	summary := models.TestSummary{
		Results: make([]models.TestResult, 0, len(operations)),
	}
	total := len(operations)

	for i, op := range operations {
		// Report: test is starting
		if onEvent != nil {
			onEvent(TestEvent{Type: EventStarting, Operation: op, Index: i, Total: total})
		}

		result, err := t.TestOperation(op, parser)
		if err != nil {
			result.Error = fmt.Sprintf("test execution error: %v", err)
			result.Passed = false
		}
		summary.AddResult(result)

		// Report: test completed
		if onEvent != nil {
			onEvent(TestEvent{Type: EventCompleted, Operation: op, Result: &result, Index: i, Total: total})
		}
	}

	return summary
}
