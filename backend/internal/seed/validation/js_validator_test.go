package validation

import (
	"strings"
	"testing"
)

func TestJSValidator_ValidateSyntax(t *testing.T) {
	validator := NewJSValidator()

	tests := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{
			name:    "empty code is valid",
			code:    "",
			wantErr: false,
		},
		{
			name:    "whitespace only is valid",
			code:    "   \n\t  ",
			wantErr: false,
		},
		{
			name:    "simple return statement",
			code:    `return { result: 1 };`,
			wantErr: false,
		},
		{
			name:    "variable declaration and return",
			code:    `const x = 1; return { value: x };`,
			wantErr: false,
		},
		{
			name:    "synchronous ctx call (goja compatible)",
			code:    `const result = ctx.llm.chat('openai', 'gpt-4', {}); return result;`,
			wantErr: false,
		},
		{
			name:    "template literals",
			code:    "const msg = `Hello ${input.name}`; return { message: msg };",
			wantErr: false,
		},
		{
			name:    "arrow functions",
			code:    `const items = [1, 2, 3]; const doubled = items.map(x => x * 2); return { doubled };`,
			wantErr: false,
		},
		{
			name:    "destructuring",
			code:    `const { name, value } = input; return { name, value };`,
			wantErr: false,
		},
		{
			name:    "spread operator",
			code:    `const merged = { ...input, extra: true }; return merged;`,
			wantErr: false,
		},
		{
			name:    "syntax error - missing bracket",
			code:    `return { result: 1`,
			wantErr: true,
		},
		{
			name:    "syntax error - invalid token",
			code:    `const @ = 1;`,
			wantErr: true,
		},
		{
			name:    "syntax error - unclosed string",
			code:    `const msg = "hello;`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateSyntax(tt.code)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSyntax() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJSValidator_GojaCompatibility(t *testing.T) {
	validator := NewJSValidator()

	tests := []struct {
		name        string
		code        string
		wantErr     bool
		errContains string
	}{
		{
			name:        "await keyword is rejected",
			code:        `const result = await ctx.llm.chat('openai', 'gpt-4', {}); return result;`,
			wantErr:     true,
			errContains: "await",
		},
		{
			name:        "async function is rejected",
			code:        `async function process() { return 1; }`,
			wantErr:     true,
			errContains: "async function",
		},
		{
			name:        "async arrow function is rejected",
			code:        `const fn = async () => { return 1; };`,
			wantErr:     true,
			errContains: "async",
		},
		{
			name:        "async arrow function with space is rejected",
			code:        `const fn = async (x) => x * 2;`,
			wantErr:     true,
			errContains: "async",
		},
		{
			name:    "synchronous code is accepted",
			code:    `const result = ctx.llm.chat('openai', 'gpt-4', {}); return result;`,
			wantErr: false,
		},
		{
			name:    "regular function is accepted",
			code:    `function process() { return 1; } return process();`,
			wantErr: false,
		},
		{
			name:    "arrow function without async is accepted",
			code:    `const fn = (x) => x * 2; return fn(5);`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateSyntax(tt.code)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSyntax() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errContains != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error should contain %q, got %q", tt.errContains, err.Error())
				}
			}
		})
	}
}

func TestJSValidator_NonStrictMode(t *testing.T) {
	// Create validator with strict mode disabled
	validator := NewJSValidatorWithOptions(false)

	// With strict mode off, async/await should pass syntax check
	// (even though it won't work at runtime)
	code := `const result = await ctx.llm.chat('openai', 'gpt-4', {}); return result;`
	err := validator.ValidateSyntax(code)
	if err != nil {
		t.Errorf("Non-strict mode should allow async/await for syntax check, got error: %v", err)
	}
}

func TestJSValidator_ValidateWithExecuteFunction(t *testing.T) {
	validator := NewJSValidator()

	tests := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{
			name: "execute function (sync)",
			code: `function execute(input, context) {
				const result = context.llm.chat('openai', 'gpt-4', {});
				return result;
			}`,
			wantErr: false,
		},
		{
			name: "execute function returning simple value",
			code: `function execute(input, context) {
				return { value: input.x + 1 };
			}`,
			wantErr: false,
		},
		{
			name:    "inline code (no execute function)",
			code:    `const x = input.value; return { result: x * 2 };`,
			wantErr: false,
		},
		{
			name:    "syntax error in execute function",
			code:    `function execute(input, context) { return { invalid`,
			wantErr: true,
		},
		{
			name: "async function execute is rejected",
			code: `async function execute(input, context) {
				const result = await context.llm.chat('openai', 'gpt-4', {});
				return result;
			}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateWithExecuteFunction(tt.code)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateWithExecuteFunction() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
