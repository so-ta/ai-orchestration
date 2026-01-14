package validation

import (
	"fmt"
	"strings"

	"github.com/dop251/goja"
)

// JSValidator validates JavaScript code syntax using Goja VM
type JSValidator struct{}

// NewJSValidator creates a new JavaScript validator
func NewJSValidator() *JSValidator {
	return &JSValidator{}
}

// ValidateSyntax checks if JavaScript code has valid syntax
func (v *JSValidator) ValidateSyntax(code string) error {
	if strings.TrimSpace(code) == "" {
		return nil // Empty code is valid (some blocks may not have code)
	}

	// Wrap code in an async IIFE to support await syntax
	// This matches how the runtime sandbox handles code with await
	wrappedCode := fmt.Sprintf(`
(async function() {
	%s
})();
`, code)

	// Compile without executing to check syntax
	_, err := goja.Compile("validation", wrappedCode, false)
	if err != nil {
		return fmt.Errorf("JavaScript syntax error: %w", err)
	}

	return nil
}

// ValidateWithExecuteFunction validates code that defines an execute function
func (v *JSValidator) ValidateWithExecuteFunction(code string) error {
	if strings.TrimSpace(code) == "" {
		return nil
	}

	// Check if code defines an execute function
	if strings.Contains(code, "function execute") || strings.Contains(code, "async function execute") {
		wrappedCode := fmt.Sprintf(`
%s

(async function() {
	var result = await execute(input, context);
	return result;
})();
`, code)
		_, err := goja.Compile("validation", wrappedCode, false)
		if err != nil {
			return fmt.Errorf("JavaScript syntax error: %w", err)
		}
		return nil
	}

	// Otherwise, treat the code as the body of the execute function
	return v.ValidateSyntax(code)
}
