package blocks

import (
	"encoding/json"

	"github.com/souta/ai-orchestration/internal/domain"
)

func (r *Registry) registerUtilityBlocks() {
	r.register(NoteBlock())
	r.register(CodeBlock())
	r.register(FunctionBlock())
	r.register(LogBlock())
}

func NoteBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "note",
		Version:     1,
		Name:        "Note",
		Description: "Documentation/comment",
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryUtility,
		Icon:        "file-text",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"color": {"type": "string"},
				"content": {"type": "string"}
			}
		}`),
		InputPorts:  []domain.InputPort{},
		OutputPorts: []domain.OutputPort{},
		Code:        `return input;`,
		UIConfig:    json.RawMessage(`{"icon": "file-text", "color": "#9CA3AF"}`),
		ErrorCodes:  []domain.ErrorCodeDef{},
		Enabled:     true,
	}
}

func CodeBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "code",
		Version:     1,
		Name:        "Code",
		Description: "Execute custom JavaScript code",
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryUtility,
		Icon:        "terminal",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"code": {
					"type": "string",
					"title": "コード",
					"description": "JavaScript code to execute. Use 'return { port: \"portName\", data: {...} }' to specify output port.",
					"x-ui-widget": "code"
				},
				"output_schema": {
					"type": "object",
					"title": "出力スキーマ",
					"description": "出力データのスキーマを定義（定義されたフィールドのみが次のステップに渡されます）",
					"x-ui-widget": "output-schema"
				},
				"custom_output_ports": {
					"type": "array",
					"title": "カスタム出力ポート",
					"description": "コードから指定可能な出力ポート名を定義します。return { port: \"portName\", data: {...} } で出力先を指定できます。",
					"items": {"type": "string"},
					"default": []
				},
				"enable_error_port": {
					"type": "boolean",
					"title": "エラーハンドルを有効化",
					"description": "エラー発生時に専用のエラーポートに出力します",
					"default": false
				}
			}
		}`),
		InputPorts: []domain.InputPort{
			{Name: "input", Label: "Input", Schema: json.RawMessage(`{"type": "any"}`), Required: false, Description: "Input data for code execution"},
		},
		OutputPorts: []domain.OutputPort{
			{Name: "output", Label: "Output", IsDefault: true, Description: "Code execution result"},
		},
		Code:        "// User code is dynamically injected\nreturn input;",
		UIConfig:    json.RawMessage(`{"icon": "terminal", "color": "#6366F1"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "CODE_001", Name: "SYNTAX_ERROR", Description: "JavaScript syntax error", Retryable: false},
			{Code: "CODE_002", Name: "RUNTIME_ERROR", Description: "JavaScript runtime error", Retryable: false},
		},
		Enabled: true,
	}
}

func FunctionBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "function",
		Version:     1,
		Name:        "Function",
		Description: "Execute custom JavaScript",
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryUtility,
		Icon:        "code",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"code": {
					"type": "string",
					"title": "コード",
					"description": "JavaScript code to execute. Use 'return { port: \"portName\", data: {...} }' to specify output port.",
					"x-ui-widget": "code"
				},
				"language": {
					"enum": ["javascript"],
					"type": "string"
				},
				"timeout_ms": {"type": "integer"},
				"output_schema": {
					"type": "object",
					"title": "出力スキーマ",
					"description": "出力データのスキーマを定義（定義されたフィールドのみが次のステップに渡されます）",
					"x-ui-widget": "output-schema"
				},
				"custom_output_ports": {
					"type": "array",
					"title": "カスタム出力ポート",
					"description": "コードから指定可能な出力ポート名を定義します。return { port: \"portName\", data: {...} } で出力先を指定できます。",
					"items": {"type": "string"},
					"default": []
				},
				"enable_error_port": {
					"type": "boolean",
					"title": "エラーハンドルを有効化",
					"description": "エラー発生時に専用のエラーポートに出力します",
					"default": false
				}
			}
		}`),
		InputPorts: []domain.InputPort{
			{Name: "input", Label: "Input", Schema: json.RawMessage(`{"type": "any"}`), Required: false, Description: "Input data for function"},
		},
		OutputPorts: []domain.OutputPort{
			{Name: "output", Label: "Output", IsDefault: true, Description: "Function result"},
		},
		Code: `
// This block executes user-defined code
// The user's code is stored in config.code and evaluated dynamically
return input;
`,
		UIConfig: json.RawMessage(`{"icon": "code", "color": "#6366F1"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "FUNC_001", Name: "SYNTAX_ERROR", Description: "JavaScript syntax error", Retryable: false},
			{Code: "FUNC_002", Name: "RUNTIME_ERROR", Description: "JavaScript runtime error", Retryable: false},
			{Code: "FUNC_003", Name: "TIMEOUT", Description: "Function execution timeout", Retryable: false},
		},
		Enabled: true,
	}
}

func LogBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "log",
		Version:     1,
		Name:        "Log",
		Description: "Output log messages for debugging",
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryUtility,
		Icon:        "terminal",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"data": {
					"type": "string",
					"description": "JSON path to include additional data (e.g. $.input)"
				},
				"level": {
					"enum": ["debug", "info", "warn", "error"],
					"type": "string",
					"default": "info",
					"description": "Log level"
				},
				"message": {
					"type": "string",
					"description": "Log message (supports {{$.field}} template variables)"
				}
			}
		}`),
		InputPorts: []domain.InputPort{
			{Name: "input", Label: "Input", Description: "Data to log"},
		},
		OutputPorts: []domain.OutputPort{
			{Name: "output", Label: "Output", IsDefault: true, Description: "Pass-through output"},
		},
		Code: `
// Log block: outputs to console and passes input through
const level = config.level || 'info';
const message = config.message || JSON.stringify(input);
ctx.log(level, message, input);
return input;
`,
		UIConfig:   json.RawMessage(`{"icon": "terminal", "color": "#6B7280"}`),
		ErrorCodes: []domain.ErrorCodeDef{},
		Enabled:    true,
	}
}
