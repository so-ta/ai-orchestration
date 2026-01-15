package blocks

import (
	"encoding/json"

	"github.com/souta/ai-orchestration/internal/domain"
)

func (r *Registry) registerLogicBlocks() {
	r.register(ConditionBlock())
	r.register(SwitchBlock())
}

func ConditionBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "condition",
		Version:     1,
		Name:        "Condition",
		Description: "Branch based on expression",
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryBranching,
		Icon:        "git-branch",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"expression": {"type": "string", "description": "JSONPath expression"}
			}
		}`),
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"value": {"type": "any", "description": "条件式で評価する値"}
			},
			"description": "条件式で評価されるデータ"
		}`),
		InputPorts: []domain.InputPort{
			{Name: "input", Label: "Input", Schema: json.RawMessage(`{"type": "any"}`), Required: true, Description: "Data to evaluate condition against"},
		},
		OutputPorts: []domain.OutputPort{
			{Name: "true", Label: "Yes", IsDefault: true, Description: "When condition is true"},
			{Name: "false", Label: "No", IsDefault: false, Description: "When condition is false"},
		},
		Code: `
const result = evaluate(config.expression, input);
return {
    ...input,
    __branch: result ? 'then' : 'else'
};
`,
		UIConfig: json.RawMessage(`{"icon": "git-branch", "color": "#F59E0B"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "COND_001", Name: "INVALID_EXPR", Description: "Invalid condition expression", Retryable: false},
			{Code: "COND_002", Name: "EVAL_ERROR", Description: "Expression evaluation error", Retryable: false},
		},
		Enabled: true,
	}
}

func SwitchBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "switch",
		Version:     1,
		Name:        "Switch",
		Description: "Multi-branch routing",
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryBranching,
		Icon:        "shuffle",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"mode": {"enum": ["rules", "expression"], "type": "string"},
				"cases": {
					"type": "array",
					"items": {
						"type": "object",
						"properties": {
							"name": {"type": "string"},
							"expression": {"type": "string"},
							"is_default": {"type": "boolean"}
						}
					}
				}
			}
		}`),
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"value": {"type": "any", "description": "分岐条件で評価する値"}
			},
			"description": "分岐条件で評価されるデータ"
		}`),
		InputPorts: []domain.InputPort{
			{Name: "input", Label: "Input", Schema: json.RawMessage(`{"type": "any"}`), Required: true, Description: "Value to switch on"},
		},
		OutputPorts: []domain.OutputPort{
			{Name: "default", Label: "Default", IsDefault: true, Description: "When no case matches"},
		},
		Code: `
let matchedCase = null;
for (const c of config.cases || []) {
    if (c.is_default) {
        matchedCase = matchedCase || c.name;
        continue;
    }
    if (evaluate(c.expression, input)) {
        matchedCase = c.name;
        break;
    }
}
return {
    ...input,
    __branch: matchedCase || 'default'
};
`,
		UIConfig: json.RawMessage(`{"icon": "shuffle", "color": "#F59E0B"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "SWITCH_001", Name: "NO_MATCH", Description: "No matching case", Retryable: false},
		},
		Enabled: true,
	}
}
