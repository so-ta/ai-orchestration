package domain

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// InputSchema represents a JSON Schema for workflow input validation
type InputSchema struct {
	Type       string                    `json:"type"`
	Properties map[string]PropertySchema `json:"properties,omitempty"`
	Required   []string                  `json:"required,omitempty"`
}

// PropertySchema represents a property in the input schema
type PropertySchema struct {
	Type        string `json:"type"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
}

// InputValidationError represents a validation error for input data
type InputValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *InputValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// InputValidationErrors represents multiple validation errors
type InputValidationErrors struct {
	Errors []InputValidationError `json:"errors"`
}

func (e *InputValidationErrors) Error() string {
	if len(e.Errors) == 0 {
		return "validation failed"
	}
	return fmt.Sprintf("validation failed: %s", e.Errors[0].Message)
}

// ValidateInputSchema validates input data against an input schema
// Returns nil if validation passes, or InputValidationErrors if validation fails
func ValidateInputSchema(input json.RawMessage, schemaJSON json.RawMessage) error {
	if schemaJSON == nil || len(schemaJSON) == 0 {
		return nil // No schema defined, skip validation
	}

	var schema InputSchema
	if err := json.Unmarshal(schemaJSON, &schema); err != nil {
		return nil // Invalid schema format, skip validation
	}

	if schema.Type != "object" {
		return nil // Only object schemas are supported
	}

	if schema.Properties == nil || len(schema.Properties) == 0 {
		return nil // No properties defined, skip validation
	}

	var inputData map[string]interface{}
	if err := json.Unmarshal(input, &inputData); err != nil {
		return &InputValidationErrors{
			Errors: []InputValidationError{
				{Field: "_root", Message: "invalid JSON input"},
			},
		}
	}

	var errors []InputValidationError

	// Check required fields
	for _, fieldName := range schema.Required {
		value, exists := inputData[fieldName]
		if !exists || value == nil || value == "" {
			propSchema, hasSchema := schema.Properties[fieldName]
			title := fieldName
			if hasSchema && propSchema.Title != "" {
				title = propSchema.Title
			}
			errors = append(errors, InputValidationError{
				Field:   fieldName,
				Message: fmt.Sprintf("%s is required", title),
			})
		}
	}

	// Check types
	for fieldName, propSchema := range schema.Properties {
		value, exists := inputData[fieldName]
		if !exists || value == nil {
			continue // Skip missing optional fields
		}

		if !validateType(value, propSchema.Type) {
			errors = append(errors, InputValidationError{
				Field:   fieldName,
				Message: fmt.Sprintf("%s must be of type %s", fieldName, propSchema.Type),
			})
		}
	}

	if len(errors) > 0 {
		return &InputValidationErrors{Errors: errors}
	}

	return nil
}

// validateType checks if a value matches the expected JSON Schema type
func validateType(value interface{}, expectedType string) bool {
	if expectedType == "" || expectedType == "any" {
		return true
	}

	actualType := reflect.TypeOf(value)
	if actualType == nil {
		return expectedType == "null"
	}

	switch expectedType {
	case "string":
		_, ok := value.(string)
		return ok
	case "number":
		switch value.(type) {
		case float64, float32, int, int64, int32:
			return true
		}
		return false
	case "integer":
		switch v := value.(type) {
		case float64:
			return v == float64(int64(v))
		case float32:
			return v == float32(int32(v))
		case int, int64, int32:
			return true
		}
		return false
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "array":
		_, ok := value.([]interface{})
		return ok
	case "object":
		_, ok := value.(map[string]interface{})
		return ok
	case "null":
		return value == nil
	}

	return true
}

// FilterOutputBySchema filters output data to only include fields defined in the schema
// If schema is nil or empty, returns the original output unchanged
func FilterOutputBySchema(output json.RawMessage, schemaJSON json.RawMessage) (json.RawMessage, error) {
	if schemaJSON == nil || len(schemaJSON) == 0 {
		return output, nil // No schema defined, return original
	}

	var schema InputSchema
	if err := json.Unmarshal(schemaJSON, &schema); err != nil {
		return output, nil // Invalid schema format, return original
	}

	if schema.Type != "object" || schema.Properties == nil || len(schema.Properties) == 0 {
		return output, nil // Only object schemas with properties are supported
	}

	var outputData map[string]interface{}
	if err := json.Unmarshal(output, &outputData); err != nil {
		return output, nil // Not a valid JSON object, return original
	}

	// Filter to only include properties defined in schema
	filtered := make(map[string]interface{})
	for fieldName := range schema.Properties {
		if value, exists := outputData[fieldName]; exists {
			filtered[fieldName] = value
		}
	}

	return json.Marshal(filtered)
}
