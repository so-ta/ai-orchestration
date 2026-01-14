package blocks

import (
	"encoding/json"

	"github.com/souta/ai-orchestration/internal/domain"
)

func (r *Registry) registerControlBlocks() {
	r.register(StartBlock())
	r.register(WaitBlock())
	r.register(ErrorBlock())
	r.register(HumanInLoopBlock())
}

func StartBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "start",
		Version:     1,
		Name:        "Start",
		Description: "Workflow entry point",
		Category:    domain.BlockCategoryControl,
		Icon:        "play",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"input_schema": {
					"type": "object",
					"title": "入力スキーマ",
					"description": "ワークフロー実行時の入力データのスキーマを定義",
					"properties": {
						"type": {"type": "string", "default": "object"},
						"required": {"type": "array", "items": {"type": "string"}},
						"properties": {"type": "object"}
					}
				}
			}
		}`),
		InputPorts: []domain.InputPort{},
		OutputPorts: []domain.OutputPort{
			{Name: "output", Label: "Output", IsDefault: true, Description: "Workflow input data"},
		},
		Code:       `return input;`,
		UIConfig:   json.RawMessage(`{"icon": "play", "color": "#10B981"}`),
		ErrorCodes: []domain.ErrorCodeDef{},
		Enabled:    true,
		TestCases: []BlockTestCase{
			{
				Name:           "passthrough input",
				Input:          map[string]interface{}{"message": "hello"},
				Config:         map[string]interface{}{},
				ExpectedOutput: map[string]interface{}{"message": "hello"},
			},
		},
	}
}

func WaitBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "wait",
		Version:     1,
		Name:        "Wait",
		Description: "Pause execution",
		Category:    domain.BlockCategoryControl,
		Icon:        "clock",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"until": {"type": "string", "format": "date-time"},
				"duration_ms": {"type": "integer", "minimum": 0}
			}
		}`),
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"data": {"type": "object", "description": "待機後にパススルーするデータ"}
			},
			"description": "待機後にそのまま出力されるデータ"
		}`),
		InputPorts: []domain.InputPort{
			{Name: "input", Label: "Input", Schema: json.RawMessage(`{"type": "any"}`), Required: false, Description: "Data to pass through after wait"},
		},
		OutputPorts: []domain.OutputPort{
			{Name: "output", Label: "Output", IsDefault: true, Description: "Continues after wait"},
		},
		Code: `
if (config.duration_ms) {
    new Promise(resolve => setTimeout(resolve, config.duration_ms));
}
return input;
`,
		UIConfig:   json.RawMessage(`{"icon": "clock", "color": "#6B7280"}`),
		ErrorCodes: []domain.ErrorCodeDef{},
		Enabled:    true,
	}
}

func ErrorBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "error",
		Version:     1,
		Name:        "Error",
		Description: "Stop workflow with error",
		Category:    domain.BlockCategoryControl,
		Icon:        "alert-circle",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"error_code": {"type": "string"},
				"error_type": {"type": "string"},
				"error_message": {"type": "string"}
			}
		}`),
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"error_context": {"type": "object", "description": "エラーに関する追加情報"}
			},
			"description": "エラー処理に渡されるデータ"
		}`),
		InputPorts: []domain.InputPort{
			{Name: "error", Label: "Error", Schema: json.RawMessage(`{"type": "object", "properties": {"code": {"type": "string"}, "message": {"type": "string"}}}`), Required: true, Description: "Error information to handle"},
		},
		OutputPorts: []domain.OutputPort{},
		Code: `
throw new Error(config.error_message || 'Workflow stopped with error');
`,
		UIConfig:   json.RawMessage(`{"icon": "alert-circle", "color": "#EF4444"}`),
		ErrorCodes: []domain.ErrorCodeDef{},
		Enabled:    true,
	}
}

func HumanInLoopBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "human_in_loop",
		Version:     1,
		Name:        "Human in Loop",
		Description: "Wait for human approval",
		Category:    domain.BlockCategoryControl,
		Icon:        "user-check",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"approval_url": {"type": "boolean"},
				"instructions": {"type": "string"},
				"timeout_hours": {"type": "integer"}
			}
		}`),
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"context": {"type": "object", "description": "承認者に表示するコンテキスト"},
				"summary": {"type": "string", "description": "承認リクエストの概要"}
			},
			"description": "承認者に表示されるコンテキストデータ"
		}`),
		InputPorts: []domain.InputPort{
			{Name: "input", Label: "Input", Schema: json.RawMessage(`{"type": "any"}`), Required: true, Description: "Context data for human review"},
		},
		OutputPorts: []domain.OutputPort{
			{Name: "approved", Label: "Approved", IsDefault: true, Description: "When approved"},
			{Name: "rejected", Label: "Rejected", IsDefault: false, Description: "When rejected"},
			{Name: "timeout", Label: "Timeout", IsDefault: false, Description: "When timed out"},
		},
		Code: `
return ctx.human.requestApproval({
    instructions: config.instructions,
    timeout: config.timeout_hours,
    data: input
});
`,
		UIConfig: json.RawMessage(`{"icon": "user-check", "color": "#EC4899"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "HIL_001", Name: "TIMEOUT", Description: "Human approval timeout", Retryable: false},
			{Code: "HIL_002", Name: "REJECTED", Description: "Human rejected", Retryable: false},
		},
		Enabled: true,
	}
}
