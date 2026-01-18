package engine

import (
	"encoding/json"
	"strings"
)

// ScopedVariables holds variables for different scopes
type ScopedVariables struct {
	Org      map[string]interface{} // Organization (tenant) variables - {{$org.xxx}}
	Project  map[string]interface{} // Project variables - {{$project.xxx}}
	Personal map[string]interface{} // Personal (user) variables - {{$personal.xxx}}
}

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
	return ExpandConfigTemplatesWithScopes(config, input, nil)
}

// ExpandConfigTemplatesWithScopes expands template variables including scoped variables.
// Supported syntax:
//   - {{field}} - top-level key from input
//   - {{$.field}} - JSONPath syntax (for compatibility)
//   - {{$org.field}} - organization (tenant) variables
//   - {{$project.field}} - project variables
//   - {{$personal.field}} - personal (user) variables
//   - {{$input.field}} - explicit input reference (same as {{field}})
//   - {{nested.field}} - nested path from input
func ExpandConfigTemplatesWithScopes(config json.RawMessage, input json.RawMessage, scopes *ScopedVariables) (json.RawMessage, error) {
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

	// Ensure scopes is not nil
	if scopes == nil {
		scopes = &ScopedVariables{}
	}

	// Parse config
	var configData interface{}
	if err := json.Unmarshal(config, &configData); err != nil {
		return config, err
	}

	// Expand templates recursively
	expanded := expandValueWithScopes(configData, inputData, scopes)

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

// expandValueWithScopes recursively expands template variables with scope support
func expandValueWithScopes(value interface{}, inputData map[string]interface{}, scopes *ScopedVariables) interface{} {
	switch v := value.(type) {
	case string:
		return expandStringWithScopes(v, inputData, scopes)
	case map[string]interface{}:
		result := make(map[string]interface{}, len(v))
		for key, val := range v {
			result[key] = expandValueWithScopes(val, inputData, scopes)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = expandValueWithScopes(val, inputData, scopes)
		}
		return result
	default:
		// Numbers, booleans, null - return as-is
		return value
	}
}

// expandStringWithScopes expands template variables in a string with scope support.
// Supported scopes: $org, $project, $personal, $input
func expandStringWithScopes(s string, inputData map[string]interface{}, scopes *ScopedVariables) interface{} {
	// Check if the entire string is a single template variable
	trimmed := strings.TrimSpace(s)
	if strings.HasPrefix(trimmed, "{{") && strings.HasSuffix(trimmed, "}}") {
		// Check if there's only one template variable
		inner := trimmed[2 : len(trimmed)-2]
		if !strings.Contains(inner, "{{") && !strings.Contains(inner, "}}") {
			// Single template variable - return actual value
			path := strings.TrimSpace(inner)
			value := extractPathWithScopes(path, inputData, scopes)
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
		value := extractPathWithScopes(path, inputData, scopes)
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

// extractPathWithScopes extracts a value using scope-aware path resolution.
// Supported prefixes: $org., $project., $personal., $input.
// Without prefix, defaults to input data.
func extractPathWithScopes(path string, inputData map[string]interface{}, scopes *ScopedVariables) interface{} {
	// Remove leading $ for standard JSONPath compatibility
	path = strings.TrimPrefix(path, "$.")

	// Check for scoped prefixes
	if strings.HasPrefix(path, "$org.") {
		subPath := strings.TrimPrefix(path, "$org.")
		if scopes != nil && scopes.Org != nil {
			return extractPath(scopes.Org, subPath)
		}
		return nil
	}
	if strings.HasPrefix(path, "$project.") {
		subPath := strings.TrimPrefix(path, "$project.")
		if scopes != nil && scopes.Project != nil {
			return extractPath(scopes.Project, subPath)
		}
		return nil
	}
	if strings.HasPrefix(path, "$personal.") {
		subPath := strings.TrimPrefix(path, "$personal.")
		if scopes != nil && scopes.Personal != nil {
			return extractPath(scopes.Personal, subPath)
		}
		return nil
	}
	if strings.HasPrefix(path, "$input.") {
		subPath := strings.TrimPrefix(path, "$input.")
		return extractPath(inputData, subPath)
	}

	// No scope prefix - default to input data
	path = strings.TrimPrefix(path, "$")
	return extractPath(inputData, path)
}
