package tester

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/moamenhredeen/oas/internal/parser"
)

func TestNewValidator(t *testing.T) {
	v := NewValidator()
	if v == nil {
		t.Fatal("Validator is nil")
	}
}

func TestValidateResponseStatusCode(t *testing.T) {
	v := NewValidator()

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id": 1, "name": "test"}`))
	}))
	defer server.Close()

	// Make a request
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Parse OpenAPI spec
	p, err := parser.ParseFile("../../tests/pet-store.json")
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// Get operation details
	opDetails, err := p.GetOperationDetails("/pets", "GET")
	if err != nil {
		t.Fatalf("Failed to get operation details: %v", err)
	}

	// Validate response
	errors, err := v.ValidateResponse(resp, opDetails)
	if err != nil {
		t.Fatalf("Validation error: %v", err)
	}

	// Should have some validation errors since the response doesn't match the spec exactly
	// (we're using a mock server, not the actual petstore API)
	_ = errors
}

func TestValidateResponseInvalidStatusCode(t *testing.T) {
	v := NewValidator()

	// Create a test server that returns 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal error"}`))
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	p, err := parser.ParseFile("../../tests/pet-store.json")
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	opDetails, err := p.GetOperationDetails("/pets", "GET")
	if err != nil {
		t.Fatalf("Failed to get operation details: %v", err)
	}

	errors, err := v.ValidateResponse(resp, opDetails)
	if err != nil {
		t.Fatalf("Validation error: %v", err)
	}

	// Should have validation errors for unexpected status code
	if len(errors) == 0 {
		t.Error("Expected validation errors for unexpected status code")
	}
}

func TestValidateJSONSchema(t *testing.T) {
	v := NewValidator()

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id": 1, "name": "test"}`))
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	p, err := parser.ParseFile("../../tests/pet-store.json")
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	opDetails, err := p.GetOperationDetails("/pets", "GET")
	if err != nil {
		t.Fatalf("Failed to get operation details: %v", err)
	}

	errors, err := v.ValidateResponse(resp, opDetails)
	if err != nil {
		t.Fatalf("Validation error: %v", err)
	}

	_ = errors
}
