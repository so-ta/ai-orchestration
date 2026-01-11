package engine

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ConditionEvaluator evaluates condition expressions
type ConditionEvaluator struct{}

// NewConditionEvaluator creates a new condition evaluator
func NewConditionEvaluator() *ConditionEvaluator {
	return &ConditionEvaluator{}
}

// Evaluate evaluates a condition expression against data
// Supported expressions:
//   - "true" / "false" - literal boolean
//   - "$.field" - check if field exists and is truthy
//   - "$.field == value" - equality check
//   - "$.field != value" - inequality check
//   - "$.field > value" - greater than (numeric)
//   - "$.field >= value" - greater than or equal
//   - "$.field < value" - less than
//   - "$.field <= value" - less than or equal
//   - "$.field.nested" - nested field access
func (e *ConditionEvaluator) Evaluate(expression string, data json.RawMessage) (bool, error) {
	if expression == "" || expression == "true" {
		return true, nil
	}
	if expression == "false" {
		return false, nil
	}

	// Parse the expression
	expr := strings.TrimSpace(expression)

	// Parse data
	var dataMap map[string]interface{}
	if data != nil {
		if err := json.Unmarshal(data, &dataMap); err != nil {
			// Try to parse as a simple value
			var simpleValue interface{}
			if jsonErr := json.Unmarshal(data, &simpleValue); jsonErr != nil {
				return false, fmt.Errorf("failed to parse data: %w", err)
			}
			dataMap = map[string]interface{}{"value": simpleValue}
		}
	} else {
		dataMap = make(map[string]interface{})
	}

	// Check for comparison operators
	operators := []struct {
		op   string
		eval func(left, right interface{}) bool
	}{
		{"==", func(l, r interface{}) bool { return compare(l, r) == 0 }},
		{"!=", func(l, r interface{}) bool { return compare(l, r) != 0 }},
		{">=", func(l, r interface{}) bool { return compare(l, r) >= 0 }},
		{"<=", func(l, r interface{}) bool { return compare(l, r) <= 0 }},
		{">", func(l, r interface{}) bool { return compare(l, r) > 0 }},
		{"<", func(l, r interface{}) bool { return compare(l, r) < 0 }},
	}

	for _, op := range operators {
		if parts := strings.SplitN(expr, op.op, 2); len(parts) == 2 {
			left := strings.TrimSpace(parts[0])
			right := strings.TrimSpace(parts[1])

			leftValue, err := e.ResolveValue(left, dataMap)
			if err != nil {
				return false, err
			}

			rightValue, err := e.ResolveValue(right, dataMap)
			if err != nil {
				return false, err
			}

			return op.eval(leftValue, rightValue), nil
		}
	}

	// No operator found, check if field is truthy
	value, err := e.ResolveValue(expr, dataMap)
	if err != nil {
		return false, nil // Field doesn't exist, return false
	}

	return isTruthy(value), nil
}

// ResolveValue resolves a value from expression
// Supports:
//   - $.field.nested - JSON path
//   - "string" - string literal
//   - 123 - number literal
//   - true/false - boolean literal
//   - null - null literal
func (e *ConditionEvaluator) ResolveValue(expr string, data map[string]interface{}) (interface{}, error) {
	expr = strings.TrimSpace(expr)

	// String literal
	if (strings.HasPrefix(expr, "\"") && strings.HasSuffix(expr, "\"")) ||
		(strings.HasPrefix(expr, "'") && strings.HasSuffix(expr, "'")) {
		return expr[1 : len(expr)-1], nil
	}

	// Boolean literal
	if expr == "true" {
		return true, nil
	}
	if expr == "false" {
		return false, nil
	}

	// Null literal
	if expr == "null" {
		return nil, nil
	}

	// Number literal
	if num, err := strconv.ParseFloat(expr, 64); err == nil {
		return num, nil
	}

	// JSON path ($.field.nested)
	if strings.HasPrefix(expr, "$.") {
		return e.resolvePath(expr[2:], data)
	}

	// Simple field name
	if isIdentifier(expr) {
		return e.resolvePath(expr, data)
	}

	return nil, fmt.Errorf("invalid expression: %s", expr)
}

// resolvePath resolves a dotted path in data
func (e *ConditionEvaluator) resolvePath(path string, data map[string]interface{}) (interface{}, error) {
	parts := strings.Split(path, ".")
	var current interface{} = data

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			val, ok := v[part]
			if !ok {
				return nil, fmt.Errorf("field not found: %s", part)
			}
			current = val
		default:
			return nil, fmt.Errorf("cannot access %s on non-object", part)
		}
	}

	return current, nil
}

// compare compares two values
// Returns: -1 if left < right, 0 if equal, 1 if left > right
func compare(left, right interface{}) int {
	// Handle nil
	if left == nil && right == nil {
		return 0
	}
	if left == nil {
		return -1
	}
	if right == nil {
		return 1
	}

	// Convert to comparable types
	leftNum, leftIsNum := toFloat(left)
	rightNum, rightIsNum := toFloat(right)

	if leftIsNum && rightIsNum {
		if leftNum < rightNum {
			return -1
		} else if leftNum > rightNum {
			return 1
		}
		return 0
	}

	// String comparison
	leftStr := toString(left)
	rightStr := toString(right)

	if leftStr < rightStr {
		return -1
	} else if leftStr > rightStr {
		return 1
	}
	return 0
}

// toFloat converts value to float64
func toFloat(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case int32:
		return float64(val), true
	case string:
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

// toString converts value to string
func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	case nil:
		return ""
	default:
		if b, err := json.Marshal(v); err == nil {
			return string(b)
		}
		return fmt.Sprintf("%v", v)
	}
}

// isTruthy checks if a value is truthy
func isTruthy(v interface{}) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val != ""
	case float64:
		return val != 0
	case int:
		return val != 0
	case []interface{}:
		return len(val) > 0
	case map[string]interface{}:
		return len(val) > 0
	default:
		return true
	}
}

// isIdentifier checks if a string is a valid identifier
func isIdentifier(s string) bool {
	match, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_]*$`, s)
	return match
}
