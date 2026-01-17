package blocks

import (
	"encoding/json"

	"github.com/souta/ai-orchestration/internal/domain"
)

func (r *Registry) registerControlBlocks() {
	r.register(StartBlock())
	r.register(ScheduleTriggerBlock())
	r.register(WebhookTriggerBlock())
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
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryControl,
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
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryControl,
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
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryUtility,
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
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryUtility,
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

// ScheduleTriggerBlock defines a schedule-based workflow trigger
func ScheduleTriggerBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:            "schedule_trigger",
		Version:         1,
		Name:            "Schedule Trigger",
		Description:     "ワークフローを定期実行するトリガー",
		Category:        domain.BlockCategoryFlow,
		Subcategory:     domain.BlockSubcategoryControl,
		Icon:            "clock",
		ParentBlockSlug: "start",
		ConfigDefaults:  json.RawMessage(`{"trigger_type": "schedule"}`),
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"required": ["cron_expression"],
			"properties": {
				"cron_expression": {
					"type": "string",
					"title": "Cron式",
					"description": "実行スケジュール（例: 0 9 * * *）",
					"default": "0 9 * * *"
				},
				"timezone": {
					"type": "string",
					"title": "タイムゾーン",
					"default": "Asia/Tokyo",
					"enum": ["Asia/Tokyo", "UTC", "America/New_York", "Europe/London"]
				},
				"enabled": {
					"type": "boolean",
					"title": "有効",
					"default": true
				},
				"input_schema": {
					"type": "object",
					"title": "入力スキーマ",
					"description": "ワークフロー実行時の入力データのスキーマを定義"
				}
			}
		}`),
		InputPorts: []domain.InputPort{},
		OutputPorts: []domain.OutputPort{
			{Name: "output", Label: "Output", IsDefault: true, Description: "Scheduled execution input"},
		},
		Code: `return input;`,
		PreProcess: `
if (!config.cron_expression) {
    throw new Error('[SCHED_001] Cron expression is required');
}
return input;
`,
		UIConfig: json.RawMessage(`{"icon": "clock", "color": "#22c55e"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "SCHED_001", Name: "INVALID_CRON", Description: "Cron式が無効です", Retryable: false},
		},
		Enabled: true,
		TestCases: []BlockTestCase{
			{
				Name:           "passthrough with schedule config",
				Input:          map[string]interface{}{"data": "test"},
				Config:         map[string]interface{}{"cron_expression": "0 9 * * *", "timezone": "Asia/Tokyo"},
				ExpectedOutput: map[string]interface{}{"data": "test"},
			},
		},
	}
}

// WebhookTriggerBlock defines a webhook-based workflow trigger
func WebhookTriggerBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:            "webhook_trigger",
		Version:         1,
		Name:            "Webhook Trigger",
		Description:     "Webhook経由でワークフローをトリガー",
		Category:        domain.BlockCategoryFlow,
		Subcategory:     domain.BlockSubcategoryControl,
		Icon:            "webhook",
		ParentBlockSlug: "start",
		ConfigDefaults:  json.RawMessage(`{"trigger_type": "webhook"}`),
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"secret": {
					"type": "string",
					"title": "シークレット",
					"description": "Webhook検証用シークレット"
				},
				"allowed_ips": {
					"type": "array",
					"items": {"type": "string"},
					"title": "許可IPアドレス",
					"description": "空の場合は全IP許可"
				},
				"input_mapping": {
					"type": "object",
					"title": "入力マッピング",
					"description": "Webhookペイロードから入力へのマッピング"
				},
				"input_schema": {
					"type": "object",
					"title": "入力スキーマ",
					"description": "ワークフロー実行時の入力データのスキーマを定義"
				}
			}
		}`),
		InputPorts: []domain.InputPort{},
		OutputPorts: []domain.OutputPort{
			{Name: "output", Label: "Output", IsDefault: true, Description: "Webhook payload data"},
		},
		Code: `return input;`,
		PreProcess: `
// IP address validation
if (config.allowed_ips && config.allowed_ips.length > 0) {
    const clientIP = input.__webhook_client_ip;
    if (clientIP && !config.allowed_ips.includes(clientIP)) {
        throw new Error('[WEBHOOK_002] IP address not in allowlist: ' + clientIP);
    }
}

// HMAC signature validation
if (config.secret) {
    const signature = input.__webhook_signature;
    const payload = input.__webhook_raw_body;
    if (signature && payload) {
        const expectedSignature = ctx.crypto.hmacSha256(config.secret, payload);
        if (signature !== expectedSignature && signature !== 'sha256=' + expectedSignature) {
            throw new Error('[WEBHOOK_001] Invalid webhook signature');
        }
    }
}

// Remove internal webhook metadata before passing to workflow
const cleanInput = { ...input };
delete cleanInput.__webhook_client_ip;
delete cleanInput.__webhook_signature;
delete cleanInput.__webhook_raw_body;
return cleanInput;
`,
		UIConfig: json.RawMessage(`{"icon": "webhook", "color": "#3b82f6"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "WEBHOOK_001", Name: "INVALID_SIGNATURE", Description: "署名が無効です", Retryable: false},
			{Code: "WEBHOOK_002", Name: "IP_NOT_ALLOWED", Description: "IPアドレスが許可されていません", Retryable: false},
		},
		Enabled: true,
		TestCases: []BlockTestCase{
			{
				Name:           "passthrough webhook payload without validation",
				Input:          map[string]interface{}{"event": "push", "data": "payload"},
				Config:         map[string]interface{}{},
				ExpectedOutput: map[string]interface{}{"event": "push", "data": "payload"},
			},
		},
	}
}
