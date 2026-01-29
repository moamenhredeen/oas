package tester

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/moamenhredeen/oas/internal/models"
	"github.com/moamenhredeen/oas/internal/parser"
)

// createMockServer creates a mock HTTP server that implements the pet-store API
func createMockServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.Method {
		case "GET":
			if r.URL.Path == "/pets" {
				// Return a list of pets
				pets := []map[string]interface{}{
					{"id": 1, "name": "Fluffy"},
					{"id": 2, "name": "Spot"},
				}
				w.Header().Set("x-next", "/pets?limit=10")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(pets)
			} else if r.URL.Path == "/pets/test" {
				// Return a single pet
				pet := map[string]interface{}{
					"id":   1,
					"name": "Fluffy",
					"tag":  "cat",
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(pet)
			} else {
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
			}
		case "POST":
			if r.URL.Path == "/pets" {
				// Create a pet
				w.WriteHeader(http.StatusCreated)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
}

func TestIntegrationFullFlow(t *testing.T) {
	// Create mock server
	server := createMockServer()
	defer server.Close()

	// Parse OpenAPI spec
	p, err := parser.ParseFile("../../tests/pet-store.json")
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// Get operations
	operations, err := p.GetOperations(server.URL)
	if err != nil {
		t.Fatalf("Failed to get operations: %v", err)
	}

	if len(operations) == 0 {
		t.Fatal("No operations found")
	}

	// Create tester
	testRunner := NewTester()

	// Test all operations
	summary := testRunner.TestOperations(operations, p)

	// Verify results
	if summary.TotalTests == 0 {
		t.Error("Expected at least one test to run")
	}

	// Check that we got results for the operations
	foundGetPets := false
	for _, result := range summary.Results {
		if result.Path == "/pets" && result.Method == "GET" {
			foundGetPets = true
			if result.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, got %d", result.StatusCode)
			}
			break
		}
	}

	if !foundGetPets {
		t.Error("Expected GET /pets test result")
	}
}

func TestIntegrationSingleOperation(t *testing.T) {
	server := createMockServer()
	defer server.Close()

	p, err := parser.ParseFile("../../tests/pet-store.json")
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	op := models.Operation{
		Path:      "/pets",
		Method:    "GET",
		ServerURL: server.URL,
	}

	testRunner := NewTester()
	result, err := testRunner.TestOperation(op, p)
	if err != nil {
		t.Fatalf("Test operation failed: %v", err)
	}

	if result.Path != "/pets" {
		t.Errorf("Expected path /pets, got %s", result.Path)
	}

	if result.Method != "GET" {
		t.Errorf("Expected method GET, got %s", result.Method)
	}

	if result.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", result.StatusCode)
	}
}

func TestIntegrationPOSTOperation(t *testing.T) {
	server := createMockServer()
	defer server.Close()

	p, err := parser.ParseFile("../../tests/pet-store.json")
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	op := models.Operation{
		Path:      "/pets",
		Method:    "POST",
		ServerURL: server.URL,
	}

	testRunner := NewTester()
	result, err := testRunner.TestOperation(op, p)
	if err != nil {
		t.Fatalf("Test operation failed: %v", err)
	}

	if result.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", result.StatusCode)
	}
}

func TestIntegrationPathParameter(t *testing.T) {
	server := createMockServer()
	defer server.Close()

	p, err := parser.ParseFile("../../tests/pet-store.json")
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	op := models.Operation{
		Path:      "/pets/{petId}",
		Method:    "GET",
		ServerURL: server.URL,
	}

	testRunner := NewTester()
	result, err := testRunner.TestOperation(op, p)
	if err != nil {
		t.Fatalf("Test operation failed: %v", err)
	}

	// The path parameter should be replaced with a generated value
	if result.StatusCode == 0 {
		t.Error("Expected a status code")
	}
}

func TestIntegrationValidationErrors(t *testing.T) {
	// Create a server that returns invalid responses
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return wrong status code
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal error"}`))
	}))
	defer server.Close()

	p, err := parser.ParseFile("../../tests/pet-store.json")
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	op := models.Operation{
		Path:      "/pets",
		Method:    "GET",
		ServerURL: server.URL,
	}

	testRunner := NewTester()
	result, err := testRunner.TestOperation(op, p)
	if err != nil {
		t.Fatalf("Test operation failed: %v", err)
	}

	// Should have validation errors
	if result.Passed {
		t.Error("Expected test to fail due to invalid response")
	}

	if len(result.ValidationErrors) == 0 {
		t.Error("Expected validation errors")
	}
}

func TestIntegrationWithPaginationAPI(t *testing.T) {
	// Create mock server for pagination API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Total-Count", "100")
		w.Header().Set("X-Page", "1")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"items": []map[string]interface{}{
				{"id": 1, "name": "Item 1"},
				{"id": 2, "name": "Item 2"},
			},
			"pagination": map[string]interface{}{
				"page":  1,
				"limit": 20,
				"total": 100,
			},
		})
	}))
	defer server.Close()

	p, err := parser.ParseFile("../../tests/pagination-api.json")
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	operations, err := p.GetOperations(server.URL)
	if err != nil {
		t.Fatalf("Failed to get operations: %v", err)
	}

	testRunner := NewTester()
	summary := testRunner.TestOperations(operations, p)

	if summary.TotalTests == 0 {
		t.Error("Expected at least one test")
	}

	// Check that query parameters were added
	for _, result := range summary.Results {
		if result.Path == "/items" && result.Method == "GET" {
			// Should have executed successfully
			if result.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, got %d", result.StatusCode)
			}
		}
	}
}

