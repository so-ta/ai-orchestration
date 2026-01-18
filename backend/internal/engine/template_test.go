package engine

import (
	"encoding/json"
	"testing"
)

func TestExpandConfigTemplates(t *testing.T) {
	tests := []struct {
		name     string
		config   string
		input    string
		expected string
	}{
		{
			name:     "empty config",
			config:   `{}`,
			input:    `{"name": "test"}`,
			expected: `{}`,
		},
		{
			name:     "no templates",
			config:   `{"collection": "my-docs", "top_k": 5}`,
			input:    `{"query": "hello"}`,
			expected: `{"collection": "my-docs", "top_k": 5}`,
		},
		{
			name:     "simple string template",
			config:   `{"prompt": "Hello {{name}}"}`,
			input:    `{"name": "World"}`,
			expected: `{"prompt": "Hello World"}`,
		},
		{
			name:     "entire value is template - string",
			config:   `{"collection": "{{collection}}"}`,
			input:    `{"collection": "my-docs"}`,
			expected: `{"collection": "my-docs"}`,
		},
		{
			name:     "entire value is template - array",
			config:   `{"documents": "{{documents}}"}`,
			input:    `{"documents": [{"text": "hello"}, {"text": "world"}]}`,
			expected: `{"documents": [{"text": "hello"}, {"text": "world"}]}`,
		},
		{
			name:     "entire value is template - object",
			config:   `{"user": "{{user}}"}`,
			input:    `{"user": {"id": "123", "name": "Alice"}}`,
			expected: `{"user": {"id": "123", "name": "Alice"}}`,
		},
		{
			name:     "entire value is template - number",
			config:   `{"count": "{{count}}"}`,
			input:    `{"count": 42}`,
			expected: `{"count": 42}`,
		},
		{
			name:     "entire value is template - boolean",
			config:   `{"enabled": "{{enabled}}"}`,
			input:    `{"enabled": true}`,
			expected: `{"enabled": true}`,
		},
		{
			name:     "nested path",
			config:   `{"user_id": "{{user.id}}"}`,
			input:    `{"user": {"id": "123", "name": "Alice"}}`,
			expected: `{"user_id": "123"}`,
		},
		{
			name:     "jsonpath syntax with $.",
			config:   `{"collection": "{{$.collection}}"}`,
			input:    `{"collection": "my-docs"}`,
			expected: `{"collection": "my-docs"}`,
		},
		{
			name:     "jsonpath syntax with $ only",
			config:   `{"data": "{{$}}"}`,
			input:    `{"key": "value"}`,
			expected: `{"data": {"key": "value"}}`,
		},
		{
			name:     "multiple templates in one string",
			config:   `{"greeting": "Hello {{first}} {{last}}!"}`,
			input:    `{"first": "John", "last": "Doe"}`,
			expected: `{"greeting": "Hello John Doe!"}`,
		},
		{
			name:     "mixed static and template values",
			config:   `{"collection": "{{collection}}", "top_k": 5, "query": "{{query}}"}`,
			input:    `{"collection": "my-docs", "query": "search term"}`,
			expected: `{"collection": "my-docs", "top_k": 5, "query": "search term"}`,
		},
		{
			name:     "template not found - returns empty string",
			config:   `{"value": "{{missing}}"}`,
			input:    `{"other": "data"}`,
			expected: `{"value": ""}`,
		},
		{
			name:     "nested config object",
			config:   `{"outer": {"inner": "{{value}}"}}`,
			input:    `{"value": "nested"}`,
			expected: `{"outer": {"inner": "nested"}}`,
		},
		{
			name:     "config array with templates",
			config:   `{"items": ["{{first}}", "static", "{{second}}"]}`,
			input:    `{"first": "one", "second": "three"}`,
			expected: `{"items": ["one", "static", "three"]}`,
		},
		{
			name:     "deeply nested path",
			config:   `{"value": "{{a.b.c.d}}"}`,
			input:    `{"a": {"b": {"c": {"d": "deep"}}}}`,
			expected: `{"value": "deep"}`,
		},
		{
			name:     "template with whitespace",
			config:   `{"value": "{{ name }}"}`,
			input:    `{"name": "test"}`,
			expected: `{"value": "test"}`,
		},
		{
			name:     "empty input",
			config:   `{"value": "{{name}}"}`,
			input:    `{}`,
			expected: `{"value": ""}`,
		},
		{
			name:     "null input",
			config:   `{"value": "{{name}}"}`,
			input:    ``,
			expected: `{"value": ""}`,
		},
		{
			name:     "array in string context",
			config:   `{"message": "Items: {{items}}"}`,
			input:    `{"items": [1, 2, 3]}`,
			expected: `{"message": "Items: [1,2,3]"}`,
		},
		{
			name:     "object in string context",
			config:   `{"message": "User: {{user}}"}`,
			input:    `{"user": {"name": "Alice"}}`,
			expected: `{"message": "User: {\"name\":\"Alice\"}"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExpandConfigTemplates(
				json.RawMessage(tt.config),
				json.RawMessage(tt.input),
			)
			if err != nil {
				t.Fatalf("ExpandConfigTemplates() error = %v", err)
			}

			// Normalize JSON for comparison
			var expectedObj, resultObj interface{}
			if err := json.Unmarshal([]byte(tt.expected), &expectedObj); err != nil {
				t.Fatalf("Failed to parse expected JSON: %v", err)
			}
			if err := json.Unmarshal(result, &resultObj); err != nil {
				t.Fatalf("Failed to parse result JSON: %v", err)
			}

			expectedBytes, _ := json.Marshal(expectedObj)
			resultBytes, _ := json.Marshal(resultObj)

			if string(expectedBytes) != string(resultBytes) {
				t.Errorf("ExpandConfigTemplates() = %s, want %s", string(result), tt.expected)
			}
		})
	}
}

func TestExpandString(t *testing.T) {
	inputData := map[string]interface{}{
		"name":    "Alice",
		"age":     30,
		"items":   []interface{}{"a", "b", "c"},
		"user":    map[string]interface{}{"id": "123"},
		"enabled": true,
	}

	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{
			name:     "plain string",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "single template - string",
			input:    "{{name}}",
			expected: "Alice",
		},
		{
			name:     "single template - number",
			input:    "{{age}}",
			expected: float64(30), // JSON numbers are float64
		},
		{
			name:     "single template - array",
			input:    "{{items}}",
			expected: []interface{}{"a", "b", "c"},
		},
		{
			name:     "single template - object",
			input:    "{{user}}",
			expected: map[string]interface{}{"id": "123"},
		},
		{
			name:     "single template - boolean",
			input:    "{{enabled}}",
			expected: true,
		},
		{
			name:     "template in string",
			input:    "Hello {{name}}!",
			expected: "Hello Alice!",
		},
		{
			name:     "array in string context",
			input:    "Items: {{items}}",
			expected: `Items: ["a","b","c"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandString(tt.input, inputData)

			// Compare based on type
			switch expected := tt.expected.(type) {
			case string:
				if str, ok := result.(string); !ok || str != expected {
					t.Errorf("expandString() = %v, want %v", result, expected)
				}
			default:
				// For non-string types, use JSON comparison
				expectedBytes, _ := json.Marshal(expected)
				resultBytes, _ := json.Marshal(result)
				if string(expectedBytes) != string(resultBytes) {
					t.Errorf("expandString() = %v, want %v", result, expected)
				}
			}
		})
	}
}

func TestExtractPath(t *testing.T) {
	data := map[string]interface{}{
		"name": "Alice",
		"user": map[string]interface{}{
			"id":   "123",
			"name": "Bob",
			"profile": map[string]interface{}{
				"age": 30,
			},
		},
		"items": []interface{}{"a", "b", "c"},
	}

	tests := []struct {
		name     string
		path     string
		expected interface{}
	}{
		{
			name:     "top level",
			path:     "name",
			expected: "Alice",
		},
		{
			name:     "nested one level",
			path:     "user.id",
			expected: "123",
		},
		{
			name:     "nested two levels",
			path:     "user.profile.age",
			expected: float64(30),
		},
		{
			name:     "array",
			path:     "items",
			expected: []interface{}{"a", "b", "c"},
		},
		{
			name:     "object",
			path:     "user",
			expected: map[string]interface{}{"id": "123", "name": "Bob", "profile": map[string]interface{}{"age": float64(30)}},
		},
		{
			name:     "not found",
			path:     "missing",
			expected: nil,
		},
		{
			name:     "nested not found",
			path:     "user.missing",
			expected: nil,
		},
		{
			name:     "jsonpath with $.",
			path:     "$.name",
			expected: "Alice",
		},
		{
			name:     "jsonpath with $ only",
			path:     "$",
			expected: data,
		},
		{
			name:     "empty path",
			path:     "",
			expected: data,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPath(data, tt.path)

			// Use JSON comparison
			expectedBytes, _ := json.Marshal(tt.expected)
			resultBytes, _ := json.Marshal(result)
			if string(expectedBytes) != string(resultBytes) {
				t.Errorf("extractPath() = %v, want %v", result, tt.expected)
			}
		})
	}
}
