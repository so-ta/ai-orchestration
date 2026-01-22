package blocks

import (
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
		Name:        LText("Note", "ノート"),
		Description: LText("Documentation/comment", "ドキュメント/コメント"),
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryUtility,
		Icon:        "file-text",
		ConfigSchema: LSchema(`{
			"type": "object",
			"properties": {
				"color": {"type": "string", "title": "Color", "description": "Note color"},
				"content": {"type": "string", "title": "Content", "description": "Note content"}
			}
		}`, `{
			"type": "object",
			"properties": {
				"color": {"type": "string", "title": "色", "description": "ノートの色"},
				"content": {"type": "string", "title": "内容", "description": "ノートの内容"}
			}
		}`),
		OutputPorts: []domain.LocalizedOutputPort{},
		Code:        `return input;`,
		UIConfig:    LSchema(`{"icon": "file-text", "color": "#9CA3AF"}`, `{"icon": "file-text", "color": "#9CA3AF"}`),
		ErrorCodes:  []domain.LocalizedErrorCodeDef{},
		Enabled:     true,
	}
}

func CodeBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "code",
		Version:     1,
		Name:        LText("Code", "コード"),
		Description: LText("Execute custom JavaScript code", "カスタムJavaScriptコードを実行"),
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryUtility,
		Icon:        "terminal",
		ConfigSchema: LSchema(`{
			"type": "object",
			"properties": {
				"code": {
					"type": "string",
					"title": "Code",
					"description": "JavaScript code to execute. Use 'return { port: \"portName\", data: {...} }' to specify output port.",
					"x-ui-widget": "code"
				},
				"output_schema": {
					"type": "object",
					"title": "Output Schema",
					"description": "Define the schema for output data (only defined fields are passed to the next step)",
					"x-ui-widget": "output-schema"
				},
				"custom_output_ports": {
					"type": "array",
					"title": "Custom Output Ports",
					"description": "Define output port names that can be specified from code. Use 'return { port: \"portName\", data: {...} }' to specify the output destination.",
					"items": {"type": "string"},
					"default": []
				},
				"enable_error_port": {
					"type": "boolean",
					"title": "Enable Error Port",
					"description": "Output to dedicated error port on error",
					"default": false
				}
			}
		}`, `{
			"type": "object",
			"properties": {
				"code": {
					"type": "string",
					"title": "コード",
					"description": "実行するJavaScriptコード。'return { port: \"portName\", data: {...} }' で出力先を指定できます。",
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
		OutputPorts: []domain.LocalizedOutputPort{
			LPortWithDesc("output", "Output", "出力", "Code execution result", "コード実行結果", true),
		},
		Code:     "// User code is dynamically injected\nreturn input;",
		UIConfig: LSchema(`{"icon": "terminal", "color": "#6366F1"}`, `{"icon": "terminal", "color": "#6366F1"}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{
			LError("CODE_001", "SYNTAX_ERROR", "構文エラー", "JavaScript syntax error", "JavaScriptの構文エラーです", false),
			LError("CODE_002", "RUNTIME_ERROR", "実行時エラー", "JavaScript runtime error", "JavaScriptの実行時エラーです", false),
		},
		Enabled: true,
	}
}

func FunctionBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "function",
		Version:     1,
		Name:        LText("Function", "関数"),
		Description: LText("Execute custom JavaScript", "カスタムJavaScriptを実行"),
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryUtility,
		Icon:        "code",
		ConfigSchema: LSchema(`{
			"type": "object",
			"properties": {
				"code": {
					"type": "string",
					"title": "Code",
					"description": "JavaScript code to execute. Use 'return { port: \"portName\", data: {...} }' to specify output port.",
					"x-ui-widget": "code"
				},
				"language": {
					"enum": ["javascript"],
					"type": "string",
					"title": "Language"
				},
				"timeout_ms": {"type": "integer", "title": "Timeout (ms)", "description": "Execution timeout in milliseconds"},
				"output_schema": {
					"type": "object",
					"title": "Output Schema",
					"description": "Define the schema for output data (only defined fields are passed to the next step)",
					"x-ui-widget": "output-schema"
				},
				"custom_output_ports": {
					"type": "array",
					"title": "Custom Output Ports",
					"description": "Define output port names that can be specified from code. Use 'return { port: \"portName\", data: {...} }' to specify the output destination.",
					"items": {"type": "string"},
					"default": []
				},
				"enable_error_port": {
					"type": "boolean",
					"title": "Enable Error Port",
					"description": "Output to dedicated error port on error",
					"default": false
				}
			}
		}`, `{
			"type": "object",
			"properties": {
				"code": {
					"type": "string",
					"title": "コード",
					"description": "実行するJavaScriptコード。'return { port: \"portName\", data: {...} }' で出力先を指定できます。",
					"x-ui-widget": "code"
				},
				"language": {
					"enum": ["javascript"],
					"type": "string",
					"title": "言語"
				},
				"timeout_ms": {"type": "integer", "title": "タイムアウト (ミリ秒)", "description": "実行タイムアウト（ミリ秒）"},
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
		OutputPorts: []domain.LocalizedOutputPort{
			LPortWithDesc("output", "Output", "出力", "Function result", "関数の実行結果", true),
		},
		Code: `
// This block executes user-defined code
// The user's code is stored in config.code and evaluated dynamically
return input;
`,
		UIConfig: LSchema(`{"icon": "code", "color": "#6366F1"}`, `{"icon": "code", "color": "#6366F1"}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{
			LError("FUNC_001", "SYNTAX_ERROR", "構文エラー", "JavaScript syntax error", "JavaScriptの構文エラーです", false),
			LError("FUNC_002", "RUNTIME_ERROR", "実行時エラー", "JavaScript runtime error", "JavaScriptの実行時エラーです", false),
			LError("FUNC_003", "TIMEOUT", "タイムアウト", "Function execution timeout", "関数の実行がタイムアウトしました", false),
		},
		Enabled: true,
	}
}

func LogBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "log",
		Version:     1,
		Name:        LText("Log", "ログ"),
		Description: LText("Output log messages for debugging", "デバッグ用にログメッセージを出力"),
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryUtility,
		Icon:        "terminal",
		ConfigSchema: LSchema(`{
			"type": "object",
			"properties": {
				"data": {
					"type": "string",
					"title": "Data",
					"description": "JSON path to include additional data (e.g. $.input)"
				},
				"level": {
					"enum": ["debug", "info", "warn", "error"],
					"type": "string",
					"title": "Level",
					"default": "info",
					"description": "Log level"
				},
				"message": {
					"type": "string",
					"title": "Message",
					"description": "Log message (supports {{$.field}} template variables)"
				}
			}
		}`, `{
			"type": "object",
			"properties": {
				"data": {
					"type": "string",
					"title": "データ",
					"description": "追加データのJSONパス（例: $.input）"
				},
				"level": {
					"enum": ["debug", "info", "warn", "error"],
					"type": "string",
					"title": "レベル",
					"default": "info",
					"description": "ログレベル"
				},
				"message": {
					"type": "string",
					"title": "メッセージ",
					"description": "ログメッセージ（{{$.field}} テンプレート変数に対応）"
				}
			}
		}`),
		OutputPorts: []domain.LocalizedOutputPort{
			LPortWithDesc("output", "Output", "出力", "Pass-through output", "パススルー出力", true),
		},
		Code: `
// Log block: outputs to console and passes input through
const level = config.level || 'info';
const message = config.message || JSON.stringify(input);
ctx.log(level, message, input);
return input;
`,
		UIConfig:   LSchema(`{"icon": "terminal", "color": "#6B7280"}`, `{"icon": "terminal", "color": "#6B7280"}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{},
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
		Name:        LText("Set Variables", "変数設定"),
		Description: LText("Set or transform variables for use in subsequent steps", "後続のステップで使用する変数を設定または変換"),
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryUtility,
		Icon:        "variable",
		ConfigSchema: LSchema(`{
			"type": "object",
			"properties": {
				"variables": {
					"type": "array",
					"title": "Variables",
					"description": "Array of variables to set",
					"items": {
						"type": "object",
						"properties": {
							"name": {
								"type": "string",
								"title": "Variable Name",
								"description": "Variable name to add to output data"
							},
							"value": {
								"type": "string",
								"title": "Value",
								"description": "Variable value (supports template expressions {{$.field}})"
							},
							"type": {
								"type": "string",
								"title": "Type",
								"enum": ["string", "number", "boolean", "json"],
								"default": "string",
								"description": "Variable type"
							}
						},
						"required": ["name", "value"]
					}
				},
				"merge_input": {
					"type": "boolean",
					"title": "Merge Input",
					"description": "If true, merge set variables with input data for output (default: true)",
					"default": true
				}
			}
		}`, `{
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
		OutputPorts: []domain.LocalizedOutputPort{
			LPortWithDesc("output", "Output", "出力", "Input merged with set variables", "設定した変数とマージした入力", true),
		},
		Code: `
// Set Variables block: sets/transforms variables for subsequent steps
const variables = config.variables || [];
const mergeInput = config.merge_input !== false; // default true

// Helper function to render template expressions in strings
// Supports both {{$.path}} and {{path}} patterns
function renderTemplateString(template, data) {
    if (typeof template !== 'string') return template;
    // First, replace {{$.path}} patterns
    let result = template.replace(/\{\{\s*\$\.([^}]+)\s*\}\}/g, function(match, path) {
        const parts = path.split('.');
        let value = data;
        for (const part of parts) {
            if (value == null) return '';
            value = value[part];
        }
        return value != null ? String(value) : '';
    });
    // Then, replace {{path}} patterns (without $.)
    result = result.replace(/\{\{\s*([^$}][^}]*)\s*\}\}/g, function(match, path) {
        const parts = path.trim().split('.');
        let value = data;
        for (const part of parts) {
            if (value == null) return '';
            value = value[part];
        }
        return value != null ? String(value) : '';
    });
    return result;
}

// Deep render template expressions in nested objects/arrays
function renderTemplate(value, data) {
    if (typeof value === 'string') {
        return renderTemplateString(value, data);
    }
    if (Array.isArray(value)) {
        return value.map(item => renderTemplate(item, data));
    }
    if (value && typeof value === 'object') {
        const result = {};
        for (const key in value) {
            result[key] = renderTemplate(value[key], data);
        }
        return result;
    }
    return value;
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
		UIConfig: LSchema(`{"icon": "variable", "color": "#10B981"}`, `{"icon": "variable", "color": "#10B981"}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{
			LError("VAR_001", "PARSE_ERROR", "パースエラー", "Failed to parse JSON value", "JSON値のパースに失敗しました", false),
		},
		Enabled: true,
	}
}
