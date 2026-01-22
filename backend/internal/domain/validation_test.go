package domain

import (
	"encoding/json"
	"testing"
)

func TestValidateInputSchema(t *testing.T) {
	tests := []struct {
		name    string
		input   json.RawMessage
		schema  json.RawMessage
		wantErr bool
	}{
		{
			name:    "nil schema",
			input:   json.RawMessage(`{"key": "value"}`),
			schema:  nil,
			wantErr: false,
		},
		{
			name:    "empty schema",
			input:   json.RawMessage(`{"key": "value"}`),
			schema:  json.RawMessage(""),
			wantErr: false,
		},
		{
			name:    "valid input with required field",
			input:   json.RawMessage(`{"name": "test"}`),
			schema:  json.RawMessage(`{"type": "object", "properties": {"name": {"type": "string"}}, "required": ["name"]}`),
			wantErr: false,
		},
		{
			name:    "missing required field",
			input:   json.RawMessage(`{}`),
			schema:  json.RawMessage(`{"type": "object", "properties": {"name": {"type": "string"}}, "required": ["name"]}`),
			wantErr: true,
		},
		{
			name:    "wrong type",
			input:   json.RawMessage(`{"count": "not a number"}`),
			schema:  json.RawMessage(`{"type": "object", "properties": {"count": {"type": "number"}}}`),
			wantErr: true,
		},
		{
			name:    "correct number type",
			input:   json.RawMessage(`{"count": 42}`),
			schema:  json.RawMessage(`{"type": "object", "properties": {"count": {"type": "number"}}}`),
			wantErr: false,
		},
		{
			name:    "boolean type",
			input:   json.RawMessage(`{"enabled": true}`),
			schema:  json.RawMessage(`{"type": "object", "properties": {"enabled": {"type": "boolean"}}}`),
			wantErr: false,
		},
		{
			name:    "array type",
			input:   json.RawMessage(`{"items": [1, 2, 3]}`),
			schema:  json.RawMessage(`{"type": "object", "properties": {"items": {"type": "array"}}}`),
			wantErr: false,
		},
		{
			name:    "object type",
			input:   json.RawMessage(`{"data": {"nested": "value"}}`),
			schema:  json.RawMessage(`{"type": "object", "properties": {"data": {"type": "object"}}}`),
			wantErr: false,
		},
		{
			name:    "invalid json input",
			input:   json.RawMessage(`invalid`),
			schema:  json.RawMessage(`{"type": "object", "properties": {"name": {"type": "string"}}}`),
			wantErr: true,
		},
		{
			name:    "non-object schema",
			input:   json.RawMessage(`{"key": "value"}`),
			schema:  json.RawMessage(`{"type": "string"}`),
			wantErr: false,
		},
		{
			name:    "integer type valid",
			input:   json.RawMessage(`{"count": 5}`),
			schema:  json.RawMessage(`{"type": "object", "properties": {"count": {"type": "integer"}}}`),
			wantErr: false,
		},
		{
			name:    "integer type invalid (float)",
			input:   json.RawMessage(`{"count": 5.5}`),
			schema:  json.RawMessage(`{"type": "object", "properties": {"count": {"type": "integer"}}}`),
			wantErr: true,
		},
		{
			name:    "empty required string",
			input:   json.RawMessage(`{"name": ""}`),
			schema:  json.RawMessage(`{"type": "object", "properties": {"name": {"type": "string"}}, "required": ["name"]}`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateInputSchema(tt.input, tt.schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateInputSchema() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInputValidationError_Error(t *testing.T) {
	err := &InputValidationError{
		Field:   "name",
		Message: "is required",
	}

	expected := "name: is required"
	if err.Error() != expected {
		t.Errorf("Error() = %v, want %v", err.Error(), expected)
	}
}

func TestInputValidationErrors_Error(t *testing.T) {
	tests := []struct {
		name   string
		errors []InputValidationError
		want   string
	}{
		{
			name:   "empty errors",
			errors: []InputValidationError{},
			want:   "validation failed",
		},
		{
			name: "single error",
			errors: []InputValidationError{
				{Field: "name", Message: "is required"},
			},
			want: "validation failed: is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := &InputValidationErrors{Errors: tt.errors}
			if errs.Error() != tt.want {
				t.Errorf("Error() = %v, want %v", errs.Error(), tt.want)
			}
		})
	}
}

func TestFilterOutputBySchema(t *testing.T) {
	tests := []struct {
		name       string
		output     json.RawMessage
		schema     json.RawMessage
		wantFields []string
	}{
		{
			name:       "nil schema",
			output:     json.RawMessage(`{"a": 1, "b": 2}`),
			schema:     nil,
			wantFields: []string{"a", "b"},
		},
		{
			name:       "empty schema",
			output:     json.RawMessage(`{"a": 1, "b": 2}`),
			schema:     json.RawMessage(""),
			wantFields: []string{"a", "b"},
		},
		{
			name:       "filter to schema fields",
			output:     json.RawMessage(`{"a": 1, "b": 2, "c": 3}`),
			schema:     json.RawMessage(`{"type": "object", "properties": {"a": {"type": "number"}, "c": {"type": "number"}}}`),
			wantFields: []string{"a", "c"},
		},
		{
			name:       "non-object schema",
			output:     json.RawMessage(`{"a": 1}`),
			schema:     json.RawMessage(`{"type": "string"}`),
			wantFields: []string{"a"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FilterOutputBySchema(tt.output, tt.schema)
			if err != nil {
				t.Fatalf("FilterOutputBySchema() error = %v", err)
			}

			var data map[string]interface{}
			if err := json.Unmarshal(result, &data); err != nil {
				t.Fatalf("Unmarshal error = %v", err)
			}

			if len(data) != len(tt.wantFields) {
				t.Errorf("FilterOutputBySchema() returned %d fields, want %d", len(data), len(tt.wantFields))
			}

			for _, field := range tt.wantFields {
				if _, ok := data[field]; !ok {
					t.Errorf("FilterOutputBySchema() missing field %v", field)
				}
			}
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
		{"string valid", "hello", "string", true},
		{"string invalid", 123, "string", false},
		{"number float64", 3.14, "number", true},
		{"number int", 42, "number", true},
		{"number string invalid", "42", "number", false},
		{"integer valid", float64(5), "integer", true},
		{"integer invalid", 5.5, "integer", false},
		{"boolean valid", true, "boolean", true},
		{"boolean invalid", "true", "boolean", false},
		{"array valid", []interface{}{1, 2, 3}, "array", true},
		{"array invalid", "array", "array", false},
		{"object valid", map[string]interface{}{"key": "value"}, "object", true},
		{"object invalid", "object", "object", false},
		{"null valid", nil, "null", true},
		{"any type", "anything", "any", true},
		{"empty type", "anything", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateType(tt.value, tt.expectedType); got != tt.want {
				t.Errorf("validateType() = %v, want %v", got, tt.want)
			}
		})
	}
}
