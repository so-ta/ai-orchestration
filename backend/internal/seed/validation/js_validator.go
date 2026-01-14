package validation

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/dop251/goja"
)

// GojaForbiddenPatterns contains patterns that are not supported by goja runtime
var GojaForbiddenPatterns = []struct {
	Pattern *regexp.Regexp
	Name    string
	Message string
}{
	{
		Pattern: regexp.MustCompile(`\bawait\s+`),
		Name:    "await",
		Message: "goja does not support 'await' keyword. Use synchronous ctx.* methods instead (they are blocking)",
	},
	{
		Pattern: regexp.MustCompile(`\basync\s+function\b`),
		Name:    "async function",
		Message: "goja does not support 'async function'. Use regular functions with synchronous ctx.* methods",
	},
	{
		Pattern: regexp.MustCompile(`\basync\s*\(`),
		Name:    "async arrow function",
		Message: "goja does not support async arrow functions. Use regular functions with synchronous ctx.* methods",
	},
}

// JSValidator validates JavaScript code syntax using Goja VM
type JSValidator struct {
	// StrictMode enables goja compatibility checks (disallow await/async)
	StrictMode bool
}

// NewJSValidator creates a new JavaScript validator with strict mode enabled by default
func NewJSValidator() *JSValidator {
	return &JSValidator{
		StrictMode: true,
	}
}

// NewJSValidatorWithOptions creates a validator with custom options
func NewJSValidatorWithOptions(strictMode bool) *JSValidator {
	return &JSValidator{
		StrictMode: strictMode,
	}
}

// ValidateSyntax checks if JavaScript code has valid syntax
func (v *JSValidator) ValidateSyntax(code string) error {
	if strings.TrimSpace(code) == "" {
		return nil // Empty code is valid (some blocks may not have code)
	}

	// First check for goja-incompatible patterns if strict mode is enabled
	if v.StrictMode {
		if err := v.ValidateGojaCompatibility(code); err != nil {
			return err
		}
	}

	// Wrap code in a function to validate as function body
	var wrappedCode string
	if v.StrictMode {
		// In strict mode, use regular function (goja doesn't support async/await)
		wrappedCode = fmt.Sprintf(`
(function() {
	%s
})();
`, code)
	} else {
		// In non-strict mode, wrap in async for syntax checking only
		// (allows async/await syntax to pass, even though runtime won't support it)
		wrappedCode = fmt.Sprintf(`
(async function() {
	%s
})();
`, code)
	}

	// Compile without executing to check syntax
	_, err := goja.Compile("validation", wrappedCode, false)
	if err != nil {
		return fmt.Errorf("JavaScript syntax error: %w", err)
	}

	return nil
}

// ValidateGojaCompatibility checks for patterns that goja does not support at runtime
func (v *JSValidator) ValidateGojaCompatibility(code string) error {
	for _, pattern := range GojaForbiddenPatterns {
		if pattern.Pattern.MatchString(code) {
			return fmt.Errorf("goja runtime incompatibility: %s", pattern.Message)
		}
	}
	return nil
}

// ValidateWithExecuteFunction validates code that defines an execute function
func (v *JSValidator) ValidateWithExecuteFunction(code string) error {
	if strings.TrimSpace(code) == "" {
		return nil
	}

	// First check for goja-incompatible patterns if strict mode is enabled
	if v.StrictMode {
		if err := v.ValidateGojaCompatibility(code); err != nil {
			return err
		}
	}

	// Check if code defines an execute function
	if strings.Contains(code, "function execute") {
		wrappedCode := fmt.Sprintf(`
%s

(function() {
	var result = execute(input, context);
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
