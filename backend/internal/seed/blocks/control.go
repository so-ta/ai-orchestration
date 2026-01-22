package blocks

import (
	"encoding/json"

	"github.com/souta/ai-orchestration/internal/domain"
)

func (r *Registry) registerControlBlocks() {
	r.register(StartBlock())           // 継承用（UI非表示）
	r.register(ManualTriggerBlock())   // 手動実行トリガー
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
		Name:        LText("Start", "スタート"),
		Description: LText("Workflow entry point", "ワークフローのエントリーポイント"),
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryControl,
		Icon:        "play",
		ConfigSchema: LSchema(`{
			"type": "object",
			"properties": {
				"input_schema": {
					"type": "object",
					"title": "Input Schema",
					"description": "Define the schema for workflow execution input data",
					"properties": {
						"type": {"type": "string", "default": "object"},
						"required": {"type": "array", "items": {"type": "string"}},
						"properties": {"type": "object"}
					}
				}
			}
		}`, `{
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
		OutputPorts: []domain.LocalizedOutputPort{
			LPortWithDesc("output", "Output", "出力", "Workflow input data", "ワークフローの入力データ", true),
		},
		Code:     `return input;`,
		UIConfig: LSchema(`{"icon": "play", "color": "#10B981"}`, `{"icon": "play", "color": "#10B981"}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{},
		Enabled:    false, // UI非表示化（ManualTriggerBlock等の抽象基底ブロック）
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

// ManualTriggerBlock defines a manual workflow trigger
func ManualTriggerBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:            "manual_trigger",
		Version:         1,
		Name:            LText("Manual Trigger", "手動トリガー"),
		Description:     LText("Trigger workflow manually", "ワークフローを手動で実行するトリガー"),
		Category:        domain.BlockCategoryFlow,
		Subcategory:     domain.BlockSubcategoryControl,
		Icon:            "play",
		ParentBlockSlug: "start",
		ConfigDefaults:  json.RawMessage(`{"trigger_type": "manual"}`),
		ConfigSchema: LSchema(`{
			"type": "object",
			"properties": {
				"input_schema": {
					"type": "object",
					"title": "Input Schema",
					"description": "Define the schema for workflow execution input data",
					"properties": {
						"type": {"type": "string", "default": "object"},
						"required": {"type": "array", "items": {"type": "string"}},
						"properties": {"type": "object"}
					}
				}
			}
		}`, `{
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
		OutputPorts: []domain.LocalizedOutputPort{
			LPortWithDesc("output", "Output", "出力", "Manual execution input", "手動実行の入力", true),
		},
		Code:       `return input;`,
		UIConfig:   LSchema(`{"icon": "play", "color": "#10B981"}`, `{"icon": "play", "color": "#10B981"}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{},
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
		Name:        LText("Wait", "待機"),
		Description: LText("Pause execution", "実行を一時停止"),
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryControl,
		Icon:        "clock",
		ConfigSchema: LSchema(`{
			"type": "object",
			"properties": {
				"until": {"type": "string", "format": "date-time", "title": "Until"},
				"duration_ms": {"type": "integer", "minimum": 0, "title": "Duration (ms)"}
			}
		}`, `{
			"type": "object",
			"properties": {
				"until": {"type": "string", "format": "date-time", "title": "終了時刻"},
				"duration_ms": {"type": "integer", "minimum": 0, "title": "待機時間 (ミリ秒)"}
			}
		}`),
		OutputPorts: []domain.LocalizedOutputPort{
			LPortWithDesc("output", "Output", "出力", "Continues after wait", "待機後に続行", true),
		},
		Code: `
if (config.duration_ms) {
    new Promise(resolve => setTimeout(resolve, config.duration_ms));
}
return input;
`,
		UIConfig:   LSchema(`{"icon": "clock", "color": "#6B7280"}`, `{"icon": "clock", "color": "#6B7280"}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{},
		Enabled:    true,
	}
}

func ErrorBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "error",
		Version:     1,
		Name:        LText("Error", "エラー"),
		Description: LText("Stop workflow with error", "エラーでワークフローを停止"),
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryUtility,
		Icon:        "alert-circle",
		ConfigSchema: LSchema(`{
			"type": "object",
			"properties": {
				"error_code": {"type": "string", "title": "Error Code"},
				"error_type": {"type": "string", "title": "Error Type"},
				"error_message": {"type": "string", "title": "Error Message"}
			}
		}`, `{
			"type": "object",
			"properties": {
				"error_code": {"type": "string", "title": "エラーコード"},
				"error_type": {"type": "string", "title": "エラータイプ"},
				"error_message": {"type": "string", "title": "エラーメッセージ"}
			}
		}`),
		OutputPorts: []domain.LocalizedOutputPort{},
		Code: `
throw new Error(config.error_message || 'Workflow stopped with error');
`,
		UIConfig:   LSchema(`{"icon": "alert-circle", "color": "#EF4444"}`, `{"icon": "alert-circle", "color": "#EF4444"}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{},
		Enabled:    true,
	}
}

func HumanInLoopBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "human_in_loop",
		Version:     1,
		Name:        LText("Human in Loop", "人間承認"),
		Description: LText("Wait for human approval", "人間の承認を待つ"),
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryUtility,
		Icon:        "user-check",
		ConfigSchema: LSchema(`{
			"type": "object",
			"properties": {
				"approval_url": {"type": "boolean", "title": "Generate Approval URL"},
				"instructions": {"type": "string", "title": "Instructions"},
				"timeout_hours": {"type": "integer", "title": "Timeout (hours)"}
			}
		}`, `{
			"type": "object",
			"properties": {
				"approval_url": {"type": "boolean", "title": "承認URLを生成"},
				"instructions": {"type": "string", "title": "指示"},
				"timeout_hours": {"type": "integer", "title": "タイムアウト (時間)"}
			}
		}`),
		OutputPorts: []domain.LocalizedOutputPort{
			LPortWithDesc("approved", "Approved", "承認", "When approved", "承認された場合", true),
			LPortWithDesc("rejected", "Rejected", "却下", "When rejected", "却下された場合", false),
			LPortWithDesc("timeout", "Timeout", "タイムアウト", "When timed out", "タイムアウトした場合", false),
		},
		Code: `
return ctx.human.requestApproval({
    instructions: config.instructions,
    timeout: config.timeout_hours,
    data: input
});
`,
		UIConfig: LSchema(`{"icon": "user-check", "color": "#EC4899"}`, `{"icon": "user-check", "color": "#EC4899"}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{
			LError("HIL_001", "TIMEOUT", "タイムアウト", "Human approval timeout", "人間の承認がタイムアウトしました", false),
			LError("HIL_002", "REJECTED", "却下", "Human rejected", "人間が却下しました", false),
		},
		Enabled: true,
	}
}

// ScheduleTriggerBlock defines a schedule-based workflow trigger
func ScheduleTriggerBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:            "schedule_trigger",
		Version:         1,
		Name:            LText("Schedule Trigger", "スケジュールトリガー"),
		Description:     LText("Trigger workflow on schedule", "ワークフローを定期実行するトリガー"),
		Category:        domain.BlockCategoryFlow,
		Subcategory:     domain.BlockSubcategoryControl,
		Icon:            "clock",
		ParentBlockSlug: "start",
		ConfigDefaults:  json.RawMessage(`{"trigger_type": "schedule"}`),
		ConfigSchema: LSchema(`{
			"type": "object",
			"required": ["cron_expression"],
			"properties": {
				"cron_expression": {
					"type": "string",
					"title": "Cron Expression",
					"description": "Execution schedule (e.g., 0 9 * * *)",
					"default": "0 9 * * *"
				},
				"timezone": {
					"type": "string",
					"title": "Timezone",
					"default": "Asia/Tokyo",
					"enum": ["Asia/Tokyo", "UTC", "America/New_York", "Europe/London"]
				},
				"enabled": {
					"type": "boolean",
					"title": "Enabled",
					"default": true
				},
				"input_schema": {
					"type": "object",
					"title": "Input Schema",
					"description": "Define the schema for workflow execution input data"
				}
			}
		}`, `{
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
		OutputPorts: []domain.LocalizedOutputPort{
			LPortWithDesc("output", "Output", "出力", "Scheduled execution input", "スケジュール実行の入力", true),
		},
		Code: `return input;`,
		PreProcess: `
if (!config.cron_expression) {
    throw new Error('[SCHED_001] Cron expression is required');
}
return input;
`,
		UIConfig: LSchema(`{"icon": "clock", "color": "#22c55e"}`, `{"icon": "clock", "color": "#22c55e"}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{
			LError("SCHED_001", "INVALID_CRON", "無効なCron式", "Invalid cron expression", "Cron式が無効です", false),
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
		Name:            LText("Webhook Trigger", "Webhookトリガー"),
		Description:     LText("Trigger workflow via webhook", "Webhook経由でワークフローをトリガー"),
		Category:        domain.BlockCategoryFlow,
		Subcategory:     domain.BlockSubcategoryControl,
		Icon:            "webhook",
		ParentBlockSlug: "start",
		ConfigDefaults:  json.RawMessage(`{"trigger_type": "webhook"}`),
		ConfigSchema: LSchema(`{
			"type": "object",
			"properties": {
				"secret": {
					"type": "string",
					"title": "Secret",
					"description": "Webhook verification secret"
				},
				"allowed_ips": {
					"type": "array",
					"items": {"type": "string"},
					"title": "Allowed IP Addresses",
					"description": "If empty, all IPs are allowed"
				},
				"input_mapping": {
					"type": "object",
					"title": "Input Mapping",
					"description": "Mapping from webhook payload to input"
				},
				"input_schema": {
					"type": "object",
					"title": "Input Schema",
					"description": "Define the schema for workflow execution input data"
				}
			}
		}`, `{
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
		OutputPorts: []domain.LocalizedOutputPort{
			LPortWithDesc("output", "Output", "出力", "Webhook payload data", "Webhookペイロードデータ", true),
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
		UIConfig: LSchema(`{"icon": "webhook", "color": "#3b82f6"}`, `{"icon": "webhook", "color": "#3b82f6"}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{
			LError("WEBHOOK_001", "INVALID_SIGNATURE", "無効な署名", "Invalid webhook signature", "署名が無効です", false),
			LError("WEBHOOK_002", "IP_NOT_ALLOWED", "IP許可なし", "IP address not allowed", "IPアドレスが許可されていません", false),
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
