package validation

import (
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
			name:    "async await syntax",
			code:    `const result = await ctx.llm.chat('openai', 'gpt-4', {}); return result;`,
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

func TestJSValidator_ValidateWithExecuteFunction(t *testing.T) {
	validator := NewJSValidator()

	tests := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{
			name: "execute function with async",
			code: `async function execute(input, context) {
				const result = await context.llm.chat('openai', 'gpt-4', {});
				return result;
			}`,
			wantErr: false,
		},
		{
			name: "execute function without async",
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
			code:    `async function execute(input, context) { return { invalid`,
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
