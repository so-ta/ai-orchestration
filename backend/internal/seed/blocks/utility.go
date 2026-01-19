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
	r.register(SetVariablesBlock())
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

// SetVariablesBlock creates a block for setting/transforming variables within a workflow.
// This is useful for:
// - Injecting context (tenant_id, project_id, etc.) into workflow execution
// - Transforming input data before passing to subsequent steps
// - Setting default values
// - Type conversion (string to number, JSON parsing, etc.)
func SetVariablesBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "set-variables",
		Version:     1,
		Name:        "Set Variables",
		Description: "Set or transform variables for use in subsequent steps",
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryUtility,
		Icon:        "variable",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"variables": {
					"type": "array",
					"title": "変数",
					"description": "設定する変数の配列",
					"items": {
						"type": "object",
						"properties": {
							"name": {
								"type": "string",
								"title": "変数名",
								"description": "出力データに追加される変数名"
							},
							"value": {
								"type": "string",
								"title": "値",
								"description": "変数の値（テンプレート式 {{$.field}} に対応）"
							},
							"type": {
								"type": "string",
								"title": "型",
								"enum": ["string", "number", "boolean", "json"],
								"default": "string",
								"description": "変数の型"
							}
						},
						"required": ["name", "value"]
					}
				},
				"merge_input": {
					"type": "boolean",
					"title": "入力をマージ",
					"description": "trueの場合、設定した変数を入力データにマージして出力（デフォルト: true）",
					"default": true
				}
			}
		}`),
		InputPorts: []domain.InputPort{
			{Name: "input", Label: "Input", Schema: json.RawMessage(`{"type": "any"}`), Required: false, Description: "Input data to merge with variables"},
		},
		OutputPorts: []domain.OutputPort{
			{Name: "output", Label: "Output", IsDefault: true, Description: "Input merged with set variables"},
		},
		Code: `
// Set Variables block: sets/transforms variables for subsequent steps
const variables = config.variables || [];
const mergeInput = config.merge_input !== false; // default true

// Helper function to render template expressions
function renderTemplate(template, data) {
    if (typeof template !== 'string') return template;
    return template.replace(/\{\{\s*\$\.([^}]+)\s*\}\}/g, function(match, path) {
        const parts = path.split('.');
        let value = data;
        for (const part of parts) {
            if (value == null) return '';
            value = value[part];
        }
        return value != null ? String(value) : '';
    });
}

// Process each variable
const result = {};
for (const v of variables) {
    if (!v.name) continue;

    const renderedValue = renderTemplate(v.value, input);

    switch (v.type) {
        case 'number':
            result[v.name] = Number(renderedValue);
            break;
        case 'boolean':
            result[v.name] = renderedValue === 'true' || renderedValue === true;
            break;
        case 'json':
            try {
                result[v.name] = JSON.parse(renderedValue);
            } catch (e) {
                result[v.name] = renderedValue;
            }
            break;
        default:
            result[v.name] = renderedValue;
    }
}

// Return merged or new variables
if (mergeInput) {
    return { ...input, ...result };
}
return result;
`,
		UIConfig:   json.RawMessage(`{"icon": "variable", "color": "#10B981"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "VAR_001", Name: "PARSE_ERROR", Description: "Failed to parse JSON value", Retryable: false},
		},
		Enabled: true,
	}
}
