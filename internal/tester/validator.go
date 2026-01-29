package tester

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/moamenhredeen/oas/internal/models"
	"github.com/moamenhredeen/oas/internal/parser"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
)

// Validator validates HTTP responses against OpenAPI specifications
type Validator struct {
}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateResponse validates an HTTP response against the OpenAPI spec
func (v *Validator) ValidateResponse(resp *http.Response, opDetails *parser.OperationDetails) ([]models.ValidationError, error) {
	var errors []models.ValidationError

	if resp == nil {
		return []models.ValidationError{{Field: "response", Message: "response is nil"}}, nil
	}

	if opDetails == nil || opDetails.Responses == nil {
		return errors, nil
	}

	statusCode := resp.StatusCode
	statusCodeStr := fmt.Sprintf("%d", statusCode)

	// Find matching response definition
	var responseDef *v3.Response
	var found bool

	// Check for exact status code match
	if opDetails.Responses.Codes != nil {
		for pair := opDetails.Responses.Codes.First(); pair != nil; pair = pair.Next() {
			if pair.Key() == statusCodeStr {
				responseDef = pair.Value()
				found = true
				break
			}
		}
	}

	// Check for default response
	if !found && opDetails.Responses.Default != nil {
		responseDef = opDetails.Responses.Default
		found = true
	}

	// Check for status code ranges (2xx, 4xx, etc.)
	if !found && opDetails.Responses.Codes != nil {
		statusRange := fmt.Sprintf("%dxx", statusCode/100)
		for pair := opDetails.Responses.Codes.First(); pair != nil; pair = pair.Next() {
			if pair.Key() == statusRange {
				responseDef = pair.Value()
				found = true
				break
			}
		}
	}

	if !found {
		errors = append(errors, models.ValidationError{
			Field:   "status_code",
			Message: fmt.Sprintf("unexpected status code %d, not defined in OpenAPI spec", statusCode),
		})
		return errors, nil
	}

	// Validate headers
	if responseDef.Headers != nil {
		for pair := responseDef.Headers.First(); pair != nil; pair = pair.Next() {
			headerName := pair.Key()
			headerValue := resp.Header.Get(headerName)
			if headerValue == "" {
				// Check if header is required (simplified - assume all defined headers are required)
				errors = append(errors, models.ValidationError{
					Field:   fmt.Sprintf("header.%s", headerName),
					Message: fmt.Sprintf("missing required header: %s", headerName),
				})
			}
		}
	}

	// Validate content type
	contentType := resp.Header.Get("Content-Type")
	if responseDef.Content != nil && responseDef.Content.Len() > 0 {
		// Check if content type matches any defined content type
		contentTypeMatched := false
		for pair := responseDef.Content.First(); pair != nil; pair = pair.Next() {
			definedContentType := pair.Key()
			if strings.Contains(contentType, strings.Split(definedContentType, ";")[0]) {
				contentTypeMatched = true
				break
			}
		}

		if !contentTypeMatched && contentType != "" {
			errors = append(errors, models.ValidationError{
				Field:   "content_type",
				Message: fmt.Sprintf("unexpected content type: %s", contentType),
			})
		}

		// Validate response body schema if JSON
		if strings.Contains(contentType, "json") && responseDef.Content.Len() > 0 {
			var schema *base.Schema
			for pair := responseDef.Content.First(); pair != nil; pair = pair.Next() {
				ct := pair.Key()
				mediaType := pair.Value()
				if strings.Contains(ct, "json") {
					if mediaType.Schema != nil {
						schema = mediaType.Schema.Schema()
					}
					break
				}
			}

			if schema != nil {
				bodyErrors := v.validateJSONSchema(resp, schema)
				errors = append(errors, bodyErrors...)
			}
		}
	}

	return errors, nil
}

// validateJSONSchema validates JSON response body against schema (simplified)
func (v *Validator) validateJSONSchema(resp *http.Response, schema *base.Schema) []models.ValidationError {
	var errors []models.ValidationError

	// Read response body
	var bodyData interface{}
	if err := json.NewDecoder(resp.Body).Decode(&bodyData); err != nil {
		errors = append(errors, models.ValidationError{
			Field:   "body",
			Message: fmt.Sprintf("failed to parse JSON response: %v", err),
		})
		return errors
	}

	// Basic schema validation
	if schema.Type != nil && len(schema.Type) > 0 {
		schemaType := schema.Type[0]
		switch schemaType {
		case "object":
			if _, ok := bodyData.(map[string]interface{}); !ok {
				errors = append(errors, models.ValidationError{
					Field:   "body",
					Message: "expected object type, got different type",
				})
			}
		case "array":
			if _, ok := bodyData.([]interface{}); !ok {
				errors = append(errors, models.ValidationError{
					Field:   "body",
					Message: "expected array type, got different type",
				})
			}
		case "string":
			if _, ok := bodyData.(string); !ok {
				errors = append(errors, models.ValidationError{
					Field:   "body",
					Message: "expected string type, got different type",
				})
			}
		case "integer", "number":
			// Numbers can be float64 in JSON
			if _, ok := bodyData.(float64); !ok {
				errors = append(errors, models.ValidationError{
					Field:   "body",
					Message: "expected number type, got different type",
				})
			}
		case "boolean":
			if _, ok := bodyData.(bool); !ok {
				errors = append(errors, models.ValidationError{
					Field:   "body",
					Message: "expected boolean type, got different type",
				})
			}
		}
	}

	// Validate required fields for objects
	if schema.Type != nil && len(schema.Type) > 0 && schema.Type[0] == "object" {
		if obj, ok := bodyData.(map[string]interface{}); ok {
			if schema.Required != nil {
				for _, requiredField := range schema.Required {
					if _, exists := obj[requiredField]; !exists {
						errors = append(errors, models.ValidationError{
							Field:   fmt.Sprintf("body.%s", requiredField),
							Message: fmt.Sprintf("missing required field: %s", requiredField),
						})
					}
				}
			}
		}
	}

	return errors
}
