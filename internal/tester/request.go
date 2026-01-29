package tester

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/moamenhredeen/oas/internal/generator"
	"github.com/moamenhredeen/oas/internal/parser"
)

// RequestBuilder builds HTTP requests from OpenAPI operations
type RequestBuilder struct {
	generator *generator.Generator
}

// NewRequestBuilder creates a new request builder
func NewRequestBuilder() *RequestBuilder {
	return &RequestBuilder{
		generator: generator.NewGenerator(),
	}
}

// BuildRequest builds an HTTP request from an OpenAPI operation
func (rb *RequestBuilder) BuildRequest(opDetails *parser.OperationDetails, serverURL string) (*http.Request, error) {
	if opDetails == nil {
		return nil, fmt.Errorf("operation details is nil")
	}

	// Build URL with path parameters
	fullPath := opDetails.Path
	if opDetails.Parameters != nil {
		for _, param := range opDetails.Parameters {
			if param != nil && param.In == "path" {
				val, err := rb.generator.GeneratePathParameter(param)
				if err != nil {
					return nil, fmt.Errorf("failed to generate path parameter %s: %w", param.Name, err)
				}
				// Replace {paramName} with value
				fullPath = strings.ReplaceAll(fullPath, "{"+param.Name+"}", val)
			}
		}
	}

	// Build full URL
	fullURL := serverURL + fullPath

	// Add query parameters
	if opDetails.Parameters != nil {
		queryParams := url.Values{}
		for _, param := range opDetails.Parameters {
			if param != nil && param.In == "query" {
				val, err := rb.generator.GenerateQueryParameter(param)
				if err != nil {
					return nil, fmt.Errorf("failed to generate query parameter %s: %w", param.Name, err)
				}
				queryParams.Add(param.Name, val)
			}
		}
		if len(queryParams) > 0 {
			fullURL += "?" + queryParams.Encode()
		}
	}

	// Create request
	var req *http.Request
	var err error

	// Handle request body for POST, PUT, PATCH
	if opDetails.RequestBody != nil && (opDetails.Method == "POST" || opDetails.Method == "PUT" || opDetails.Method == "PATCH") {
		bodyBytes, contentType, err := rb.generator.GenerateRequestBody(opDetails.RequestBody)
		if err != nil {
			return nil, fmt.Errorf("failed to generate request body: %w", err)
		}
		req, err = http.NewRequest(opDetails.Method, fullURL, bytes.NewBuffer(bodyBytes))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", contentType)
	} else {
		req, err = http.NewRequest(opDetails.Method, fullURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
	}

	// Set default headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "oas-test-tool/1.0")

	// Add header parameters
	if opDetails.Parameters != nil {
		for _, param := range opDetails.Parameters {
			if param != nil && param.In == "header" {
				val, err := rb.generator.GeneratePathParameter(param)
				if err != nil {
					return nil, fmt.Errorf("failed to generate header parameter %s: %w", param.Name, err)
				}
				req.Header.Set(param.Name, val)
			}
		}
	}

	return req, nil
}
