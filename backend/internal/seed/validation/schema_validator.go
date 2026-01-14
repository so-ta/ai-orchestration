package validation

import (
	"encoding/json"
	"fmt"
)

// SchemaValidator validates JSON schemas
type SchemaValidator struct{}

// NewSchemaValidator creates a new schema validator
func NewSchemaValidator() *SchemaValidator {
	return &SchemaValidator{}
}

// ValidateSchema checks if a JSON schema is syntactically valid
func (v *SchemaValidator) ValidateSchema(schema json.RawMessage) error {
	if len(schema) == 0 {
		return nil // Empty schema is valid
	}

	// Check if it's null
	if string(schema) == "null" {
		return nil
	}

	// Check if it's empty object
	if string(schema) == "{}" {
		return nil
	}

	// Parse as JSON first
	var schemaMap map[string]interface{}
	if err := json.Unmarshal(schema, &schemaMap); err != nil {
		return fmt.Errorf("invalid JSON in schema: %w", err)
	}

	// Basic JSON Schema validation
	// Check for valid type if present
	if typeVal, ok := schemaMap["type"]; ok {
		validTypes := map[string]bool{
			"string":  true,
			"number":  true,
			"integer": true,
			"boolean": true,
			"object":  true,
			"array":   true,
			"null":    true,
			"any":     true, // Custom type used in this project
		}
		switch t := typeVal.(type) {
		case string:
			if !validTypes[t] {
				return fmt.Errorf("invalid type: %s", t)
			}
		case []interface{}:
			// Type can be an array of types
			for _, item := range t {
				if s, ok := item.(string); ok {
					if !validTypes[s] {
						return fmt.Errorf("invalid type in array: %s", s)
					}
				}
			}
		}
	}

	// Validate properties if present
	if props, ok := schemaMap["properties"]; ok {
		propsMap, ok := props.(map[string]interface{})
		if !ok {
			return fmt.Errorf("properties must be an object")
		}
		for propName, propSchema := range propsMap {
			propSchemaMap, ok := propSchema.(map[string]interface{})
			if !ok {
				return fmt.Errorf("property %s schema must be an object", propName)
			}
			// Recursively validate nested schema
			propSchemaJSON, err := json.Marshal(propSchemaMap)
			if err != nil {
				return fmt.Errorf("failed to marshal property %s schema: %w", propName, err)
			}
			if err := v.ValidateSchema(propSchemaJSON); err != nil {
				return fmt.Errorf("invalid schema for property %s: %w", propName, err)
			}
		}
	}

	// Validate items if present (for array type)
	if items, ok := schemaMap["items"]; ok {
		itemsMap, ok := items.(map[string]interface{})
		if ok {
			itemsJSON, err := json.Marshal(itemsMap)
			if err != nil {
				return fmt.Errorf("failed to marshal items schema: %w", err)
			}
			if err := v.ValidateSchema(itemsJSON); err != nil {
				return fmt.Errorf("invalid items schema: %w", err)
			}
		}
	}

	return nil
}

// ValidateJSONArray checks if a JSON array is valid
func (v *SchemaValidator) ValidateJSONArray(data json.RawMessage) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}

	var arr []interface{}
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("invalid JSON array: %w", err)
	}

	return nil
}
