package domain

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateInputSchema(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		schema      string
		expectError bool
		errorField  string
	}{
		{
			name:        "nil schema - should pass",
			input:       `{"name": "test"}`,
			schema:      "",
			expectError: false,
		},
		{
			name:        "invalid schema JSON - should skip validation",
			input:       `{"name": "test"}`,
			schema:      `not valid json`,
			expectError: false,
		},
		{
			name:        "empty schema - should pass",
			input:       `{"name": "test"}`,
			schema:      `{}`,
			expectError: false,
		},
		{
			name:        "no properties - should pass",
			input:       `{"name": "test"}`,
			schema:      `{"type": "object"}`,
			expectError: false,
		},
		{
			name:        "valid input with required field",
			input:       `{"name": "test", "value": 42}`,
			schema:      `{"type": "object", "properties": {"name": {"type": "string"}, "value": {"type": "number"}}, "required": ["name"]}`,
			expectError: false,
		},
		{
			name:        "missing required field",
			input:       `{"value": 42}`,
			schema:      `{"type": "object", "properties": {"name": {"type": "string"}, "value": {"type": "number"}}, "required": ["name"]}`,
			expectError: true,
			errorField:  "name",
		},
		{
			name:        "required field is empty string",
			input:       `{"name": ""}`,
			schema:      `{"type": "object", "properties": {"name": {"type": "string"}}, "required": ["name"]}`,
			expectError: true,
			errorField:  "name",
		},
		{
			name:        "required field is null",
			input:       `{"name": null}`,
			schema:      `{"type": "object", "properties": {"name": {"type": "string"}}, "required": ["name"]}`,
			expectError: true,
			errorField:  "name",
		},
		{
			name:        "wrong type - string instead of number",
			input:       `{"value": "not a number"}`,
			schema:      `{"type": "object", "properties": {"value": {"type": "number"}}}`,
			expectError: true,
			errorField:  "value",
		},
		{
			name:        "wrong type - number instead of string",
			input:       `{"name": 123}`,
			schema:      `{"type": "object", "properties": {"name": {"type": "string"}}}`,
			expectError: true,
			errorField:  "name",
		},
		{
			name:        "valid boolean type",
			input:       `{"active": true}`,
			schema:      `{"type": "object", "properties": {"active": {"type": "boolean"}}}`,
			expectError: false,
		},
		{
			name:        "wrong type - string instead of boolean",
			input:       `{"active": "true"}`,
			schema:      `{"type": "object", "properties": {"active": {"type": "boolean"}}}`,
			expectError: true,
			errorField:  "active",
		},
		{
			name:        "valid array type",
			input:       `{"items": [1, 2, 3]}`,
			schema:      `{"type": "object", "properties": {"items": {"type": "array"}}}`,
			expectError: false,
		},
		{
			name:        "valid object type",
			input:       `{"data": {"key": "value"}}`,
			schema:      `{"type": "object", "properties": {"data": {"type": "object"}}}`,
			expectError: false,
		},
		{
			name:        "valid integer type - whole number",
			input:       `{"count": 42}`,
			schema:      `{"type": "object", "properties": {"count": {"type": "integer"}}}`,
			expectError: false,
		},
		{
			name:        "invalid integer type - decimal",
			input:       `{"count": 42.5}`,
			schema:      `{"type": "object", "properties": {"count": {"type": "integer"}}}`,
			expectError: true,
			errorField:  "count",
		},
		{
			name:        "any type - accepts anything",
			input:       `{"data": "string value"}`,
			schema:      `{"type": "object", "properties": {"data": {"type": "any"}}}`,
			expectError: false,
		},
		{
			name:        "invalid JSON input",
			input:       `not valid json`,
			schema:      `{"type": "object", "properties": {"name": {"type": "string"}}}`,
			expectError: true,
			errorField:  "_root",
		},
		{
			name:        "optional field missing - should pass",
			input:       `{"name": "test"}`,
			schema:      `{"type": "object", "properties": {"name": {"type": "string"}, "optional": {"type": "number"}}}`,
			expectError: false,
		},
		{
			name:        "required field with title in error message",
			input:       `{}`,
			schema:      `{"type": "object", "properties": {"name": {"type": "string", "title": "名前"}}, "required": ["name"]}`,
			expectError: true,
			errorField:  "name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var inputJSON json.RawMessage
			if tt.input != "" {
				inputJSON = json.RawMessage(tt.input)
			}

			var schemaJSON json.RawMessage
			if tt.schema != "" {
				schemaJSON = json.RawMessage(tt.schema)
			}

			err := ValidateInputSchema(inputJSON, schemaJSON)

			if tt.expectError {
				require.Error(t, err)
				validationErrors, ok := err.(*InputValidationErrors)
				require.True(t, ok, "error should be InputValidationErrors")
				require.NotEmpty(t, validationErrors.Errors)
				assert.Equal(t, tt.errorField, validationErrors.Errors[0].Field)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFilterOutputBySchema(t *testing.T) {
	tests := []struct {
		name           string
		output         string
		schema         string
		expectedFields []string
	}{
		{
			name:           "nil schema - return original",
			output:         `{"name": "test", "extra": "value"}`,
			schema:         "",
			expectedFields: []string{"name", "extra"},
		},
		{
			name:           "invalid schema JSON - return original",
			output:         `{"name": "test", "extra": "value"}`,
			schema:         `not valid json`,
			expectedFields: []string{"name", "extra"},
		},
		{
			name:           "empty schema - return original",
			output:         `{"name": "test", "extra": "value"}`,
			schema:         `{}`,
			expectedFields: []string{"name", "extra"},
		},
		{
			name:           "filter to defined fields only",
			output:         `{"name": "test", "value": 42, "extra": "should be removed"}`,
			schema:         `{"type": "object", "properties": {"name": {"type": "string"}, "value": {"type": "number"}}}`,
			expectedFields: []string{"name", "value"},
		},
		{
			name:           "schema field not in output - skip it",
			output:         `{"name": "test"}`,
			schema:         `{"type": "object", "properties": {"name": {"type": "string"}, "missing": {"type": "number"}}}`,
			expectedFields: []string{"name"},
		},
		{
			name:           "all output fields match schema",
			output:         `{"name": "test", "value": 42}`,
			schema:         `{"type": "object", "properties": {"name": {"type": "string"}, "value": {"type": "number"}}}`,
			expectedFields: []string{"name", "value"},
		},
		{
			name:           "non-object schema - return original",
			output:         `{"name": "test"}`,
			schema:         `{"type": "array"}`,
			expectedFields: []string{"name"},
		},
		{
			name:           "invalid output JSON - return original",
			output:         `not valid json`,
			schema:         `{"type": "object", "properties": {"name": {"type": "string"}}}`,
			expectedFields: nil, // Will return original string
		},
		{
			name:           "nested objects are passed through",
			output:         `{"data": {"nested": "value"}, "extra": "removed"}`,
			schema:         `{"type": "object", "properties": {"data": {"type": "object"}}}`,
			expectedFields: []string{"data"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var schemaJSON json.RawMessage
			if tt.schema != "" {
				schemaJSON = json.RawMessage(tt.schema)
			}

			result, err := FilterOutputBySchema(json.RawMessage(tt.output), schemaJSON)
			require.NoError(t, err)

			if tt.expectedFields == nil {
				// For invalid JSON, should return original
				assert.Equal(t, tt.output, string(result))
				return
			}

			var resultData map[string]interface{}
			err = json.Unmarshal(result, &resultData)
			require.NoError(t, err)

			// Check that result contains exactly expected fields
			assert.Len(t, resultData, len(tt.expectedFields))
			for _, field := range tt.expectedFields {
				_, exists := resultData[field]
				assert.True(t, exists, "field %s should exist in result", field)
			}
		})
	}
}

func TestInputValidationError(t *testing.T) {
	err := &InputValidationError{
		Field:   "name",
		Message: "is required",
	}
	assert.Equal(t, "name: is required", err.Error())
}

func TestInputValidationErrors(t *testing.T) {
	t.Run("empty errors", func(t *testing.T) {
		err := &InputValidationErrors{}
		assert.Equal(t, "validation failed", err.Error())
	})

	t.Run("with errors", func(t *testing.T) {
		err := &InputValidationErrors{
			Errors: []InputValidationError{
				{Field: "name", Message: "is required"},
				{Field: "value", Message: "must be number"},
			},
		}
		assert.Equal(t, "validation failed: is required", err.Error())
	})
}

func TestValidateType(t *testing.T) {
	tests := []struct {
		name         string
		value        interface{}
		expectedType string
		valid        bool
	}{
		{"empty type - valid", "anything", "", true},
		{"any type - valid", 123, "any", true},
		{"nil value with null type", nil, "null", true},
		{"nil value with string type", nil, "string", false},
		{"string value", "test", "string", true},
		{"number float64", 42.5, "number", true},
		{"number int", 42, "number", true},
		{"boolean true", true, "boolean", true},
		{"boolean false", false, "boolean", true},
		{"array value", []interface{}{1, 2, 3}, "array", true},
		{"object value", map[string]interface{}{"key": "value"}, "object", true},
		{"integer whole number", float64(42), "integer", true},
		{"integer decimal fails", float64(42.5), "integer", false},
		{"unknown type - valid", "anything", "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateType(tt.value, tt.expectedType)
			assert.Equal(t, tt.valid, result)
		})
	}
}
