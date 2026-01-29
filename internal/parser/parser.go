package parser

import (
	"fmt"
	"os"

	"github.com/moamenhredeen/oas/internal/models"
	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
)

// Parser handles parsing OpenAPI specification files
type Parser struct {
	document libopenapi.Document
}

// ParseFile parses an OpenAPI specification file and returns a Parser instance
func ParseFile(filePath string) (*Parser, error) {
	specBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read OpenAPI file: %w", err)
	}

	document, err := libopenapi.NewDocument(specBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI document: %w", err)
	}

	return &Parser{document: document}, nil
}

// GetServerURLs returns the server URLs from the OpenAPI spec
func (p *Parser) GetServerURLs() ([]string, error) {
	model, errs := p.document.BuildV3Model()
	if errs != nil {
		return nil, fmt.Errorf("failed to build v3 model: %v", errs)
	}

	servers := model.Model.Servers
	if servers == nil || len(servers) == 0 {
		return []string{"http://localhost"}, nil
	}

	urls := make([]string, 0, len(servers))
	for _, server := range servers {
		if server != nil && server.URL != "" {
			urls = append(urls, server.URL)
		}
	}

	return urls, nil
}

// GetOperations extracts all operations from the OpenAPI spec
func (p *Parser) GetOperations(serverURL string) ([]models.Operation, error) {
	model, errs := p.document.BuildV3Model()
	if errs != nil {
		return nil, fmt.Errorf("failed to build v3 model: %v", errs)
	}

	var operations []models.Operation
	paths := model.Model.Paths

	if paths == nil || paths.PathItems == nil {
		return operations, nil
	}

	// Iterate over ordered map
	for pair := paths.PathItems.First(); pair != nil; pair = pair.Next() {
		pathItem := pair.Key()
		pathItemValue := pair.Value()
		if pathItemValue == nil {
			continue
		}

		// Process each HTTP method
		methods := map[string]*v3.Operation{
			"GET":     pathItemValue.Get,
			"POST":    pathItemValue.Post,
			"PUT":     pathItemValue.Put,
			"PATCH":   pathItemValue.Patch,
			"DELETE":  pathItemValue.Delete,
			"HEAD":    pathItemValue.Head,
			"OPTIONS": pathItemValue.Options,
		}

		for method, op := range methods {
			if op == nil {
				continue
			}

			operationID := op.OperationId
			tags := []string{}
			if op.Tags != nil {
				tags = append(tags, op.Tags...)
			}

			operations = append(operations, models.Operation{
				Path:        pathItem,
				Method:      method,
				OperationID: operationID,
				Tags:        tags,
				ServerURL:   serverURL,
				FullPath:    serverURL + pathItem,
			})
		}
	}

	return operations, nil
}

// GetOperationDetails returns detailed information about a specific operation
type OperationDetails struct {
	Operation   *v3.Operation
	Path        string
	Method      string
	Parameters  []*v3.Parameter
	RequestBody *v3.RequestBody
	Responses   *v3.Responses
}

// GetOperationDetails extracts detailed information for a specific operation
func (p *Parser) GetOperationDetails(path, method string) (*OperationDetails, error) {
	model, errs := p.document.BuildV3Model()
	if errs != nil {
		return nil, fmt.Errorf("failed to build v3 model: %v", errs)
	}

	paths := model.Model.Paths
	if paths == nil || paths.PathItems == nil {
		return nil, fmt.Errorf("path not found: %s", path)
	}

	var pathItem *v3.PathItem
	for pair := paths.PathItems.First(); pair != nil; pair = pair.Next() {
		if pair.Key() == path {
			pathItem = pair.Value()
			break
		}
	}

	if pathItem == nil {
		return nil, fmt.Errorf("path not found: %s", path)
	}

	var operation *v3.Operation
	switch method {
	case "GET":
		operation = pathItem.Get
	case "POST":
		operation = pathItem.Post
	case "PUT":
		operation = pathItem.Put
	case "PATCH":
		operation = pathItem.Patch
	case "DELETE":
		operation = pathItem.Delete
	case "HEAD":
		operation = pathItem.Head
	case "OPTIONS":
		operation = pathItem.Options
	default:
		return nil, fmt.Errorf("unsupported method: %s", method)
	}

	if operation == nil {
		return nil, fmt.Errorf("operation not found: %s %s", method, path)
	}

	// Extract parameters
	var parameters []*v3.Parameter
	if operation.Parameters != nil {
		parameters = append(parameters, operation.Parameters...)
	}

	details := &OperationDetails{
		Operation:  operation,
		Path:       path,
		Method:     method,
		Parameters: parameters,
		Responses:  operation.Responses,
	}

	if operation.RequestBody != nil {
		details.RequestBody = operation.RequestBody
	}

	return details, nil
}
