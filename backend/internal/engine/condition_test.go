package engine

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConditionEvaluator_Evaluate_Literals(t *testing.T) {
	eval := NewConditionEvaluator()

	tests := []struct {
		name       string
		expression string
		expected   bool
	}{
		{"empty string is true", "", true},
		{"true literal", "true", true},
		{"false literal", "false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := eval.Evaluate(tt.expression, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConditionEvaluator_Evaluate_Equality(t *testing.T) {
	eval := NewConditionEvaluator()

	tests := []struct {
		name       string
		expression string
		data       string
		expected   bool
	}{
		{
			name:       "string equality - true",
			expression: `$.status == "success"`,
			data:       `{"status": "success"}`,
			expected:   true,
		},
		{
			name:       "string equality - false",
			expression: `$.status == "success"`,
			data:       `{"status": "failed"}`,
			expected:   false,
		},
		{
			name:       "number equality",
			expression: `$.count == 5`,
			data:       `{"count": 5}`,
			expected:   true,
		},
		{
			name:       "inequality",
			expression: `$.status != "failed"`,
			data:       `{"status": "success"}`,
			expected:   true,
		},
		{
			name:       "boolean equality",
			expression: `$.active == true`,
			data:       `{"active": true}`,
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := eval.Evaluate(tt.expression, json.RawMessage(tt.data))
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConditionEvaluator_Evaluate_Comparison(t *testing.T) {
	eval := NewConditionEvaluator()

	tests := []struct {
		name       string
		expression string
		data       string
		expected   bool
	}{
		{
			name:       "greater than - true",
			expression: `$.value > 10`,
			data:       `{"value": 15}`,
			expected:   true,
		},
		{
			name:       "greater than - false",
			expression: `$.value > 10`,
			data:       `{"value": 5}`,
			expected:   false,
		},
		{
			name:       "greater than or equal",
			expression: `$.value >= 10`,
			data:       `{"value": 10}`,
			expected:   true,
		},
		{
			name:       "less than",
			expression: `$.value < 10`,
			data:       `{"value": 5}`,
			expected:   true,
		},
		{
			name:       "less than or equal",
			expression: `$.value <= 10`,
			data:       `{"value": 10}`,
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := eval.Evaluate(tt.expression, json.RawMessage(tt.data))
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConditionEvaluator_Evaluate_NestedPath(t *testing.T) {
	eval := NewConditionEvaluator()

	data := `{
		"user": {
			"profile": {
				"age": 25,
				"name": "Alice"
			}
		}
	}`

	tests := []struct {
		name       string
		expression string
		expected   bool
	}{
		{
			name:       "nested path equality",
			expression: `$.user.profile.name == "Alice"`,
			expected:   true,
		},
		{
			name:       "nested path comparison",
			expression: `$.user.profile.age >= 18`,
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := eval.Evaluate(tt.expression, json.RawMessage(data))
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConditionEvaluator_Evaluate_Truthy(t *testing.T) {
	eval := NewConditionEvaluator()

	tests := []struct {
		name       string
		expression string
		data       string
		expected   bool
	}{
		{
			name:       "truthy string",
			expression: `$.message`,
			data:       `{"message": "hello"}`,
			expected:   true,
		},
		{
			name:       "falsy empty string",
			expression: `$.message`,
			data:       `{"message": ""}`,
			expected:   false,
		},
		{
			name:       "truthy number",
			expression: `$.count`,
			data:       `{"count": 5}`,
			expected:   true,
		},
		{
			name:       "falsy zero",
			expression: `$.count`,
			data:       `{"count": 0}`,
			expected:   false,
		},
		{
			name:       "truthy boolean true",
			expression: `$.active`,
			data:       `{"active": true}`,
			expected:   true,
		},
		{
			name:       "falsy boolean false",
			expression: `$.active`,
			data:       `{"active": false}`,
			expected:   false,
		},
		{
			name:       "missing field is falsy",
			expression: `$.missing`,
			data:       `{"other": "value"}`,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := eval.Evaluate(tt.expression, json.RawMessage(tt.data))
			// Missing field returns false without error
			if tt.expression == "$.missing" {
				assert.False(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestConditionEvaluator_Evaluate_NullComparison(t *testing.T) {
	eval := NewConditionEvaluator()

	tests := []struct {
		name       string
		expression string
		data       string
		expected   bool
	}{
		{
			name:       "null equality",
			expression: `$.value == null`,
			data:       `{"value": null}`,
			expected:   true,
		},
		{
			name:       "null inequality",
			expression: `$.value != null`,
			data:       `{"value": "something"}`,
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := eval.Evaluate(tt.expression, json.RawMessage(tt.data))
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConditionEvaluator_Evaluate_WithOutput(t *testing.T) {
	eval := NewConditionEvaluator()

	// Simulate condition step output
	data := `{
		"result": true,
		"score": 85,
		"category": "A"
	}`

	tests := []struct {
		name       string
		expression string
		expected   bool
	}{
		{
			name:       "result is true",
			expression: `$.result == true`,
			expected:   true,
		},
		{
			name:       "score is high",
			expression: `$.score >= 80`,
			expected:   true,
		},
		{
			name:       "category A",
			expression: `$.category == "A"`,
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := eval.Evaluate(tt.expression, json.RawMessage(data))
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConditionEvaluator_Compare(t *testing.T) {
	tests := []struct {
		name     string
		left     interface{}
		right    interface{}
		expected int
	}{
		{"nil vs nil", nil, nil, 0},
		{"nil vs value", nil, "a", -1},
		{"value vs nil", "a", nil, 1},
		{"equal numbers", 5.0, 5.0, 0},
		{"less than", 3.0, 5.0, -1},
		{"greater than", 7.0, 5.0, 1},
		{"equal strings", "abc", "abc", 0},
		{"string comparison", "abc", "def", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compare(tt.left, tt.right)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConditionEvaluator_IsTruthy(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected bool
	}{
		{"nil is falsy", nil, false},
		{"true is truthy", true, true},
		{"false is falsy", false, false},
		{"empty string is falsy", "", false},
		{"non-empty string is truthy", "hello", true},
		{"zero is falsy", 0.0, false},
		{"non-zero is truthy", 5.0, true},
		{"empty array is falsy", []interface{}{}, false},
		{"non-empty array is truthy", []interface{}{"a"}, true},
		{"empty map is falsy", map[string]interface{}{}, false},
		{"non-empty map is truthy", map[string]interface{}{"a": 1}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTruthy(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}
