package parser

import (
	"testing"
)

func TestParseFile(t *testing.T) {
	p, err := ParseFile("../../tests/pet-store.json")
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	if p == nil {
		t.Fatal("Parser is nil")
	}
}

func TestGetServerURLs(t *testing.T) {
	p, err := ParseFile("../../tests/pet-store.json")
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	urls, err := p.GetServerURLs()
	if err != nil {
		t.Fatalf("Failed to get server URLs: %v", err)
	}

	if len(urls) == 0 {
		t.Error("Expected at least one server URL")
	}

	expectedURL := "http://petstore.swagger.io/v1"
	found := false
	for _, url := range urls {
		if url == expectedURL {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected server URL %s not found. Got: %v", expectedURL, urls)
	}
}

func TestGetOperations(t *testing.T) {
	p, err := ParseFile("../../tests/pet-store.json")
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	operations, err := p.GetOperations("http://petstore.swagger.io/v1")
	if err != nil {
		t.Fatalf("Failed to get operations: %v", err)
	}

	if len(operations) == 0 {
		t.Error("Expected at least one operation")
	}

	// Check for expected operations
	foundGetPets := false
	foundPostPets := false
	foundGetPetById := false

	for _, op := range operations {
		if op.Path == "/pets" && op.Method == "GET" {
			foundGetPets = true
		}
		if op.Path == "/pets" && op.Method == "POST" {
			foundPostPets = true
		}
		if op.Path == "/pets/{petId}" && op.Method == "GET" {
			foundGetPetById = true
		}
	}

	if !foundGetPets {
		t.Error("Expected GET /pets operation not found")
	}
	if !foundPostPets {
		t.Error("Expected POST /pets operation not found")
	}
	if !foundGetPetById {
		t.Error("Expected GET /pets/{petId} operation not found")
	}
}

func TestGetOperationDetails(t *testing.T) {
	p, err := ParseFile("../../tests/pet-store.json")
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	details, err := p.GetOperationDetails("/pets", "GET")
	if err != nil {
		t.Fatalf("Failed to get operation details: %v", err)
	}

	if details == nil {
		t.Fatal("Operation details is nil")
	}

	if details.Path != "/pets" {
		t.Errorf("Expected path /pets, got %s", details.Path)
	}

	if details.Method != "GET" {
		t.Errorf("Expected method GET, got %s", details.Method)
	}

	if details.Operation == nil {
		t.Error("Operation is nil")
	}

	if details.Responses == nil {
		t.Error("Responses is nil")
	}
}

func TestParseFileNotFound(t *testing.T) {
	_, err := ParseFile("nonexistent.json")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestGetOperationsWithAuthAPI(t *testing.T) {
	p, err := ParseFile("../../tests/auth-api.json")
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	operations, err := p.GetOperations("http://api.example.com/v1")
	if err != nil {
		t.Fatalf("Failed to get operations: %v", err)
	}

	if len(operations) < 2 {
		t.Errorf("Expected at least 2 operations, got %d", len(operations))
	}
}

