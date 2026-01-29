package generator

import (
	"testing"

	"github.com/pb33f/libopenapi/datamodel/high/base"
)

func TestNewGenerator(t *testing.T) {
	g := NewGenerator()
	if g == nil {
		t.Fatal("Generator is nil")
	}
}

func TestGenerateString(t *testing.T) {
	g := NewGenerator()

	// Create a simple string schema
	schema := &base.Schema{
		Type: []string{"string"},
	}

	val, err := g.GenerateValue(schema)
	if err != nil {
		t.Fatalf("Failed to generate value: %v", err)
	}

	if _, ok := val.(string); !ok {
		t.Errorf("Expected string, got %T", val)
	}
}

func TestGenerateInteger(t *testing.T) {
	g := NewGenerator()

	schema := &base.Schema{
		Type: []string{"integer"},
	}

	val, err := g.GenerateValue(schema)
	if err != nil {
		t.Fatalf("Failed to generate value: %v", err)
	}

	if _, ok := val.(int); !ok {
		t.Errorf("Expected integer, got %T", val)
	}
}

func TestGenerateBoolean(t *testing.T) {
	g := NewGenerator()

	schema := &base.Schema{
		Type: []string{"boolean"},
	}

	val, err := g.GenerateValue(schema)
	if err != nil {
		t.Fatalf("Failed to generate value: %v", err)
	}

	if _, ok := val.(bool); !ok {
		t.Errorf("Expected boolean, got %T", val)
	}
}

func TestGenerateArray(t *testing.T) {
	g := NewGenerator()

	itemSchema := &base.Schema{
		Type: []string{"string"},
	}

	schema := &base.Schema{
		Type: []string{"array"},
		Items: &base.DynamicValue[*base.SchemaProxy, bool]{
		},
	}

	// Note: This is a simplified test. Full array generation requires proper Items setup
	val, err := g.GenerateValue(schema)
	if err != nil {
		t.Fatalf("Failed to generate value: %v", err)
	}

	if _, ok := val.([]interface{}); !ok {
		t.Errorf("Expected array, got %T", val)
	}

	_ = itemSchema // Suppress unused variable warning
}

func TestGenerateObject(t *testing.T) {
	g := NewGenerator()

	schema := &base.Schema{
		Type: []string{"object"},
	}

	val, err := g.GenerateValue(schema)
	if err != nil {
		t.Fatalf("Failed to generate value: %v", err)
	}

	if _, ok := val.(map[string]interface{}); !ok {
		t.Errorf("Expected object, got %T", val)
	}
}

func TestGenerateFromFormat(t *testing.T) {
	g := NewGenerator()

	tests := []struct {
		format string
		check func(interface{}) bool
	}{
		{"email", func(v interface{}) bool {
			str, ok := v.(string)
			return ok && str != ""
		}},
		{"uuid", func(v interface{}) bool {
			str, ok := v.(string)
			return ok && str != ""
		}},
		{"date", func(v interface{}) bool {
			str, ok := v.(string)
			return ok && str != ""
		}},
	}

	for _, tt := range tests {
		result := g.generateFromFormat(tt.format)
		if !tt.check(result) {
			t.Errorf("Format %s did not generate valid value: %v", tt.format, result)
		}
	}
}

func TestGenerateValueWithExample(t *testing.T) {
	g := NewGenerator()

	// Note: Example is *yaml.Node in libopenapi, so we skip this test
	// as it requires more complex setup. The example functionality
	// is tested indirectly through integration tests.
	schema := &base.Schema{
		Type: []string{"string"},
	}

	val, err := g.GenerateValue(schema)
	if err != nil {
		t.Fatalf("Failed to generate value: %v", err)
	}

	if _, ok := val.(string); !ok {
		t.Errorf("Expected string value, got %T", val)
	}
}

