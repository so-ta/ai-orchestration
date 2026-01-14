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
		input       json.RawMessage
		schema      json.RawMessage
		wantErr     bool
		errContains string
	}{
		{
			name:    "nil schema returns no error",
			input:   json.RawMessage(`{"name": "test"}`),
			schema:  nil,
			wantErr: false,
		},
		{
			name:    "empty schema returns no error",
			input:   json.RawMessage(`{"name": "test"}`),
			schema:  json.RawMessage(``),
			wantErr: false,
		},
		{
			name:    "non-object schema type returns no error",
			input:   json.RawMessage(`{"name": "test"}`),
			schema:  json.RawMessage(`{"type": "array"}`),
			wantErr: false,
		},
		{
			name:    "schema without properties returns no error",
			input:   json.RawMessage(`{"name": "test"}`),
			schema:  json.RawMessage(`{"type": "object"}`),
			wantErr: false,
		},
		{
			name:    "valid input passes validation",
			input:   json.RawMessage(`{"name": "test", "age": 25}`),
			schema:  json.RawMessage(`{"type": "object", "properties": {"name": {"type": "string"}, "age": {"type": "integer"}}, "required": ["name"]}`),
			wantErr: false,
		},
		{
			name:        "missing required field fails validation",
			input:       json.RawMessage(`{"age": 25}`),
			schema:      json.RawMessage(`{"type": "object", "properties": {"name": {"type": "string", "title": "Name"}, "age": {"type": "integer"}}, "required": ["name"]}`),
			wantErr:     true,
			errContains: "Name is required",
		},
		{
			name:        "empty string for required field fails validation",
			input:       json.RawMessage(`{"name": "", "age": 25}`),
			schema:      json.RawMessage(`{"type": "object", "properties": {"name": {"type": "string"}, "age": {"type": "integer"}}, "required": ["name"]}`),
			wantErr:     true,
			errContains: "name is required",
		},
		{
			name:        "wrong type fails validation",
			input:       json.RawMessage(`{"name": 123}`),
			schema:      json.RawMessage(`{"type": "object", "properties": {"name": {"type": "string"}}, "required": ["name"]}`),
			wantErr:     true,
			errContains: "name must be of type string",
		},
		{
			name:    "number type accepts float",
			input:   json.RawMessage(`{"value": 3.14}`),
			schema:  json.RawMessage(`{"type": "object", "properties": {"value": {"type": "number"}}}`),
			wantErr: false,
		},
		{
			name:    "integer type accepts whole number as float",
			input:   json.RawMessage(`{"value": 42}`),
			schema:  json.RawMessage(`{"type": "object", "properties": {"value": {"type": "integer"}}}`),
			wantErr: false,
		},
		{
			name:        "integer type rejects float with decimals",
			input:       json.RawMessage(`{"value": 3.14}`),
			schema:      json.RawMessage(`{"type": "object", "properties": {"value": {"type": "integer"}}}`),
			wantErr:     true,
			errContains: "value must be of type integer",
		},
		{
			name:    "boolean type validation",
			input:   json.RawMessage(`{"flag": true}`),
			schema:  json.RawMessage(`{"type": "object", "properties": {"flag": {"type": "boolean"}}}`),
			wantErr: false,
		},
		{
			name:    "array type validation",
			input:   json.RawMessage(`{"items": [1, 2, 3]}`),
			schema:  json.RawMessage(`{"type": "object", "properties": {"items": {"type": "array"}}}`),
			wantErr: false,
		},
		{
			name:    "object type validation",
			input:   json.RawMessage(`{"nested": {"key": "value"}}`),
			schema:  json.RawMessage(`{"type": "object", "properties": {"nested": {"type": "object"}}}`),
			wantErr: false,
		},
		{
			name:        "invalid JSON input fails",
			input:       json.RawMessage(`{invalid json}`),
			schema:      json.RawMessage(`{"type": "object", "properties": {"name": {"type": "string"}}}`),
			wantErr:     true,
			errContains: "invalid JSON input",
		},
		{
			name:    "optional field can be missing",
			input:   json.RawMessage(`{"name": "test"}`),
			schema:  json.RawMessage(`{"type": "object", "properties": {"name": {"type": "string"}, "age": {"type": "integer"}}}`),
			wantErr: false,
		},
		{
			name:    "any type accepts anything",
			input:   json.RawMessage(`{"data": "anything"}`),
			schema:  json.RawMessage(`{"type": "object", "properties": {"data": {"type": "any"}}}`),
			wantErr: false,
		},
		{
			name:    "empty type accepts anything",
			input:   json.RawMessage(`{"data": 123}`),
			schema:  json.RawMessage(`{"type": "object", "properties": {"data": {"type": ""}}}`),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateInputSchema(tt.input, tt.schema)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFilterOutputBySchema(t *testing.T) {
	tests := []struct {
		name           string
		output         json.RawMessage
		schema         json.RawMessage
		expectedOutput map[string]interface{}
		wantOriginal   bool
	}{
		{
			name:         "nil schema returns original",
			output:       json.RawMessage(`{"name": "test", "extra": "data"}`),
			schema:       nil,
			wantOriginal: true,
		},
		{
			name:         "empty schema returns original",
			output:       json.RawMessage(`{"name": "test", "extra": "data"}`),
			schema:       json.RawMessage(``),
			wantOriginal: true,
		},
		{
			name:         "non-object schema returns original",
			output:       json.RawMessage(`{"name": "test", "extra": "data"}`),
			schema:       json.RawMessage(`{"type": "array"}`),
			wantOriginal: true,
		},
		{
			name:         "schema without properties returns original",
			output:       json.RawMessage(`{"name": "test", "extra": "data"}`),
			schema:       json.RawMessage(`{"type": "object"}`),
			wantOriginal: true,
		},
		{
			name:   "filters output to schema properties",
			output: json.RawMessage(`{"name": "test", "age": 25, "extra": "data"}`),
			schema: json.RawMessage(`{"type": "object", "properties": {"name": {"type": "string"}, "age": {"type": "integer"}}}`),
			expectedOutput: map[string]interface{}{
				"name": "test",
				"age":  float64(25),
			},
		},
		{
			name:   "missing schema property results in empty filtered output for that key",
			output: json.RawMessage(`{"extra": "data"}`),
			schema: json.RawMessage(`{"type": "object", "properties": {"name": {"type": "string"}}}`),
			expectedOutput: map[string]interface{}{},
		},
		{
			name:         "invalid JSON output returns original",
			output:       json.RawMessage(`{invalid}`),
			schema:       json.RawMessage(`{"type": "object", "properties": {"name": {"type": "string"}}}`),
			wantOriginal: true,
		},
		{
			name:   "preserves nested objects",
			output: json.RawMessage(`{"data": {"nested": "value"}, "extra": "ignore"}`),
			schema: json.RawMessage(`{"type": "object", "properties": {"data": {"type": "object"}}}`),
			expectedOutput: map[string]interface{}{
				"data": map[string]interface{}{"nested": "value"},
			},
		},
		{
			name:   "preserves arrays",
			output: json.RawMessage(`{"items": [1, 2, 3], "extra": "ignore"}`),
			schema: json.RawMessage(`{"type": "object", "properties": {"items": {"type": "array"}}}`),
			expectedOutput: map[string]interface{}{
				"items": []interface{}{float64(1), float64(2), float64(3)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FilterOutputBySchema(tt.output, tt.schema)
			require.NoError(t, err)

			if tt.wantOriginal {
				assert.Equal(t, tt.output, result)
			} else {
				var resultMap map[string]interface{}
				err := json.Unmarshal(result, &resultMap)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedOutput, resultMap)
			}
		})
	}
}

func TestInputValidationError_Error(t *testing.T) {
	err := &InputValidationError{
		Field:   "name",
		Message: "is required",
	}
	assert.Equal(t, "name: is required", err.Error())
}

func TestInputValidationErrors_Error(t *testing.T) {
	tests := []struct {
		name     string
		errors   []InputValidationError
		expected string
	}{
		{
			name:     "empty errors",
			errors:   []InputValidationError{},
			expected: "validation failed",
		},
		{
			name: "single error",
			errors: []InputValidationError{
				{Field: "name", Message: "is required"},
			},
			expected: "validation failed: is required",
		},
		{
			name: "multiple errors returns first",
			errors: []InputValidationError{
				{Field: "name", Message: "is required"},
				{Field: "age", Message: "must be positive"},
			},
			expected: "validation failed: is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &InputValidationErrors{Errors: tt.errors}
			assert.Equal(t, tt.expected, err.Error())
		})
	}
}

func TestValidateType(t *testing.T) {
	tests := []struct {
		name         string
		value        interface{}
		expectedType string
		want         bool
	}{
		{"nil value with null type", nil, "null", true},
		{"nil value with string type", nil, "string", false},
		{"string value", "hello", "string", true},
		{"string value with wrong type", "hello", "integer", false},
		{"float64 as number", float64(3.14), "number", true},
		{"float64 whole as integer", float64(42), "integer", true},
		{"float64 with decimals as integer", float64(3.14), "integer", false},
		{"bool as boolean", true, "boolean", true},
		{"bool with wrong type", true, "string", false},
		{"slice as array", []interface{}{1, 2}, "array", true},
		{"map as object", map[string]interface{}{"key": "val"}, "object", true},
		{"empty type accepts anything", "anything", "", true},
		{"any type accepts anything", 123, "any", true},
		{"unknown type defaults to true", "value", "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateType(tt.value, tt.expectedType)
			assert.Equal(t, tt.want, result)
		})
	}
}
