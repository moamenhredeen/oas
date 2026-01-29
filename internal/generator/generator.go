package generator

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
)

// Generator generates test data from OpenAPI schemas
type Generator struct {
	rng *rand.Rand
}

// NewGenerator creates a new generator instance
func NewGenerator() *Generator {
	return &Generator{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// GenerateValue generates a test value based on a schema
func (g *Generator) GenerateValue(schema *base.Schema) (interface{}, error) {
	if schema == nil {
		return nil, fmt.Errorf("schema is nil")
	}

	// Check for example value first
	if schema.Example != nil {
		return schema.Example, nil
	}

	// Check for default value
	if schema.Default != nil {
		return schema.Default, nil
	}

	// Handle different schema types
	if schema.Type != nil && len(schema.Type) > 0 {
		schemaType := schema.Type[0]
		switch schemaType {
		case "string":
			return g.generateString(schema), nil
		case "integer", "number":
			return g.generateNumber(schema), nil
		case "boolean":
			return true, nil
		case "array":
			return g.generateArray(schema), nil
		case "object":
			return g.generateObject(schema), nil
		}
	}

	// If no type specified, try to infer from format
	if schema.Format != "" {
		return g.generateFromFormat(schema.Format), nil
	}

	// Default to empty string
	return "", nil
}

// generateString generates a string value based on schema constraints
func (g *Generator) generateString(schema *base.Schema) string {
	// Check format
	if schema.Format != "" {
		formatted := g.generateFromFormat(schema.Format)
		if str, ok := formatted.(string); ok {
			return str
		}
	}

	// Check enum
	if schema.Enum != nil && len(schema.Enum) > 0 {
		// Enum values are yaml.Node, extract value
		enumNode := schema.Enum[0]
		if enumNode != nil {
			return enumNode.Value
		}
	}

	// Check pattern (simplified - just return a basic string)
	if schema.Pattern != "" {
		// For now, return a simple string. Full pattern matching would require regex engine
		return "test-string"
	}

	// Check min/max length
	minLength := 0
	maxLength := 10
	if schema.MinLength != nil {
		minLength = int(*schema.MinLength)
	}
	if schema.MaxLength != nil {
		maxLength = int(*schema.MaxLength)
	}

	length := minLength
	if maxLength > minLength {
		length = minLength + g.rng.Intn(maxLength-minLength+1)
	}
	if length == 0 {
		length = 5
	}

	return strings.Repeat("a", length)
}

// generateNumber generates a number value based on schema constraints
func (g *Generator) generateNumber(schema *base.Schema) interface{} {
	var min, max float64
	var isInt bool

	if schema.Type != nil && len(schema.Type) > 0 && schema.Type[0] == "integer" {
		isInt = true
		min = 0
		max = 100
	} else {
		min = 0.0
		max = 100.0
	}

	if schema.Minimum != nil {
		min = *schema.Minimum
	}
	if schema.Maximum != nil {
		max = *schema.Maximum
	}

	value := min + g.rng.Float64()*(max-min)

	if isInt {
		return int(value)
	}
	return value
}

// generateArray generates an array value
func (g *Generator) generateArray(schema *base.Schema) []interface{} {
	minItems := 0
	maxItems := 3
	if schema.MinItems != nil {
		minItems = int(*schema.MinItems)
	}
	if schema.MaxItems != nil {
		maxItems = int(*schema.MaxItems)
	}

	count := minItems
	if maxItems > minItems {
		count = minItems + g.rng.Intn(maxItems-minItems+1)
	}
	if count == 0 {
		count = 1
	}

	result := make([]interface{}, count)
	if schema.Items != nil {
		// Items is a DynamicValue, need to check if it's a SchemaProxy
		if schema.Items.IsA() {
			itemSchemaProxy := schema.Items.A
			if itemSchemaProxy != nil {
				itemSchema := itemSchemaProxy.Schema()
				if itemSchema != nil {
					for i := 0; i < count; i++ {
						val, _ := g.GenerateValue(itemSchema)
						result[i] = val
					}
				} else {
					// Default to string array
					for i := 0; i < count; i++ {
						result[i] = "item"
					}
				}
			} else {
				// Default to string array
				for i := 0; i < count; i++ {
					result[i] = "item"
				}
			}
		} else {
			// Default to string array
			for i := 0; i < count; i++ {
				result[i] = "item"
			}
		}
	} else {
		// Default to string array
		for i := 0; i < count; i++ {
			result[i] = "item"
		}
	}

	return result
}

// generateObject generates an object value
func (g *Generator) generateObject(schema *base.Schema) map[string]interface{} {
	result := make(map[string]interface{})

	if schema.Properties != nil {
		for pair := schema.Properties.First(); pair != nil; pair = pair.Next() {
			propName := pair.Key()
			propSchemaProxy := pair.Value()

			// Check if property is required
			isRequired := false
			if schema.Required != nil {
				for _, req := range schema.Required {
					if req == propName {
						isRequired = true
						break
					}
				}
			}

			// Generate value for required properties or randomly for optional ones
			if isRequired || g.rng.Float64() > 0.5 {
				propSchema := propSchemaProxy.Schema()
				if propSchema != nil {
					val, _ := g.GenerateValue(propSchema)
					result[propName] = val
				}
			}
		}
	}

	return result
}

// generateFromFormat generates a value based on format
func (g *Generator) generateFromFormat(format string) interface{} {
	switch format {
	case "date":
		return time.Now().Format("2006-01-02")
	case "date-time":
		return time.Now().Format(time.RFC3339)
	case "email":
		return "test@example.com"
	case "uri":
		return "https://example.com"
	case "uuid":
		return "123e4567-e89b-12d3-a456-426614174000"
	case "int32":
		return g.rng.Int31()
	case "int64":
		return g.rng.Int63()
	case "float":
		return g.rng.Float32()
	case "double":
		return g.rng.Float64()
	default:
		return "test-value"
	}
}

// GeneratePathParameter generates a value for a path parameter
func (g *Generator) GeneratePathParameter(param *v3.Parameter) (string, error) {
	if param == nil {
		return "", fmt.Errorf("parameter is nil")
	}

	if param.Schema != nil {
		schema := param.Schema.Schema()
		if schema != nil {
			val, err := g.GenerateValue(schema)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%v", val), nil
		}
	}

	// Default to string
	return "test", nil
}

// GenerateQueryParameter generates a value for a query parameter
func (g *Generator) GenerateQueryParameter(param *v3.Parameter) (string, error) {
	return g.GeneratePathParameter(param)
}

// GenerateRequestBody generates a request body from a schema
func (g *Generator) GenerateRequestBody(requestBody *v3.RequestBody) ([]byte, string, error) {
	if requestBody == nil {
		return nil, "", fmt.Errorf("request body is nil")
	}

	// Get the first content type and schema
	if requestBody.Content == nil || requestBody.Content.Len() == 0 {
		return nil, "", fmt.Errorf("no content defined in request body")
	}

	var contentType string
	var schema *base.Schema

	// Prefer application/json
	for pair := requestBody.Content.First(); pair != nil; pair = pair.Next() {
		ct := pair.Key()
		mediaType := pair.Value()
		if strings.Contains(ct, "json") {
			contentType = ct
			if mediaType.Schema != nil {
				schema = mediaType.Schema.Schema()
			}
			break
		}
	}

	// If no JSON found, use the first one
	if schema == nil {
		for pair := requestBody.Content.First(); pair != nil; pair = pair.Next() {
			contentType = pair.Key()
			mediaType := pair.Value()
			if mediaType.Schema != nil {
				schema = mediaType.Schema.Schema()
			}
			break
		}
	}

	if schema == nil {
		return nil, "", fmt.Errorf("no schema found in request body")
	}

	val, err := g.GenerateValue(schema)
	if err != nil {
		return nil, "", err
	}

	// Convert to JSON
	jsonBytes := []byte(fmt.Sprintf("%v", val))
	if contentType == "" {
		contentType = "application/json"
	}

	return jsonBytes, contentType, nil
}

// ResolveReference resolves a $ref reference (simplified - libopenapi should handle this)
func (g *Generator) ResolveReference(schemaProxy *base.SchemaProxy) (*base.Schema, error) {
	// libopenapi should handle $ref resolution automatically
	if schemaProxy == nil {
		return nil, fmt.Errorf("schema proxy is nil")
	}
	return schemaProxy.Schema(), nil
}
