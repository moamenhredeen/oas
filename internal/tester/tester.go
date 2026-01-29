package tester

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/moamenhredeen/oas/internal/models"
	"github.com/moamenhredeen/oas/internal/parser"
)

// Tester executes API tests based on OpenAPI specifications
type Tester struct {
	requestBuilder *RequestBuilder
	validator     *Validator
	client        *http.Client
}

// NewTester creates a new tester instance
func NewTester() *Tester {
	return &Tester{
		requestBuilder: NewRequestBuilder(),
		validator:      NewValidator(),
		client: &http.Client{
			Timeout: 30 * time.Second,
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

// TestOperations tests multiple operations
func (t *Tester) TestOperations(operations []models.Operation, parser *parser.Parser) models.TestSummary {
	summary := models.TestSummary{
		Results: make([]models.TestResult, 0, len(operations)),
	}

	for _, op := range operations {
		result, err := t.TestOperation(op, parser)
		if err != nil {
			result.Error = fmt.Sprintf("test execution error: %v", err)
			result.Passed = false
		}
		summary.AddResult(result)
	}

	return summary
}

