package engine

import (
	"encoding/json"
	"strings"
)

// ExpandConfigTemplates expands all template variables in config using values from input.
// Supported syntax:
//   - {{field}} - top-level key
//   - {{$.field}} - JSONPath syntax (for compatibility)
//   - {{nested.field}} - nested path
//
// If a template variable is the entire string value (e.g., "{{documents}}"),
// the value is replaced with the actual type from input (array, object, etc.).
// If the template is part of a larger string (e.g., "Hello {{name}}"),
// it is converted to a string representation.
func ExpandConfigTemplates(config json.RawMessage, input json.RawMessage) (json.RawMessage, error) {
	if len(config) == 0 {
		return config, nil
	}

	// Parse input data
	var inputData map[string]interface{}
	if len(input) > 0 {
		if err := json.Unmarshal(input, &inputData); err != nil {
			// If input is not a valid JSON object, use empty map
			inputData = make(map[string]interface{})
		}
	} else {
		inputData = make(map[string]interface{})
	}

	// Parse config
	var configData interface{}
	if err := json.Unmarshal(config, &configData); err != nil {
		return config, err
	}

	// Expand templates recursively
	expanded := expandValue(configData, inputData)

	// Marshal back to JSON
	return json.Marshal(expanded)
}

// expandValue recursively expands template variables in any value
func expandValue(value interface{}, inputData map[string]interface{}) interface{} {
	switch v := value.(type) {
	case string:
		return expandString(v, inputData)
	case map[string]interface{}:
		result := make(map[string]interface{}, len(v))
		for key, val := range v {
			result[key] = expandValue(val, inputData)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = expandValue(val, inputData)
		}
		return result
	default:
		// Numbers, booleans, null - return as-is
		return value
	}
}

// expandString expands template variables in a string.
// If the entire string is a single template variable (e.g., "{{field}}"),
// returns the actual value from input (preserving type).
// Otherwise, performs string substitution.
func expandString(s string, inputData map[string]interface{}) interface{} {
	// Check if the entire string is a single template variable
	trimmed := strings.TrimSpace(s)
	if strings.HasPrefix(trimmed, "{{") && strings.HasSuffix(trimmed, "}}") {
		// Check if there's only one template variable
		inner := trimmed[2 : len(trimmed)-2]
		if !strings.Contains(inner, "{{") && !strings.Contains(inner, "}}") {
			// Single template variable - return actual value
			path := strings.TrimSpace(inner)
			value := extractPath(inputData, path)
			if value != nil {
				return value
			}
			// If not found, return empty string
			return ""
		}
	}

	// Multiple variables or mixed content - do string substitution
	result := s
	for {
		start := strings.Index(result, "{{")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], "}}")
		if end == -1 {
			break
		}
		end += start + 2

		path := strings.TrimSpace(result[start+2 : end-2])
		value := extractPath(inputData, path)
		var replacement string
		if value != nil {
			switch v := value.(type) {
			case string:
				replacement = v
			default:
				// Convert non-string values to JSON
				if jsonBytes, err := json.Marshal(v); err == nil {
					replacement = string(jsonBytes)
				}
			}
		}
		result = result[:start] + replacement + result[end:]
	}

	return result
}

// extractPath extracts a value from data using a path like "field" or "nested.field"
// Also supports JSONPath-style "$.field" for compatibility
func extractPath(data interface{}, path string) interface{} {
	// Remove leading $. if present (JSONPath compatibility)
	path = strings.TrimPrefix(path, "$.")
	path = strings.TrimPrefix(path, "$")

	if path == "" {
		return data
	}

	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		if part == "" {
			continue
		}

		switch v := current.(type) {
		case map[string]interface{}:
			var ok bool
			current, ok = v[part]
			if !ok {
				return nil
			}
		default:
			return nil
		}
	}

	return current
}
