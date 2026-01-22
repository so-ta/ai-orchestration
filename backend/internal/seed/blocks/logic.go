package blocks

import (
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
		Name:        LText("Condition", "条件分岐"),
		Description: LText("Branch based on expression", "式に基づいて分岐"),
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryBranching,
		Icon:        "git-branch",
		ConfigSchema: LSchema(`{
			"type": "object",
			"properties": {
				"expression": {"type": "string", "title": "Expression", "description": "JSONPath expression"}
			}
		}`, `{
			"type": "object",
			"properties": {
				"expression": {"type": "string", "title": "式", "description": "JSONPath式"}
			}
		}`),
		OutputPorts: []domain.LocalizedOutputPort{
			LPortWithDesc("true", "Yes", "はい", "When condition is true", "条件が真の場合", true),
			LPortWithDesc("false", "No", "いいえ", "When condition is false", "条件が偽の場合", false),
		},
		Code: `
// Helper function to resolve JSONPath-like expressions ($.field.nested)
function resolveValue(expr, data) {
    expr = expr.trim();
    // String literal
    if ((expr.startsWith('"') && expr.endsWith('"')) || (expr.startsWith("'") && expr.endsWith("'"))) {
        return expr.slice(1, -1);
    }
    // Boolean literal
    if (expr === 'true') return true;
    if (expr === 'false') return false;
    // Null literal
    if (expr === 'null') return null;
    // Number literal
    const num = parseFloat(expr);
    if (!isNaN(num)) return num;
    // JSONPath ($.field.nested)
    if (expr.startsWith('$.')) {
        const path = expr.slice(2).split('.');
        let current = data;
        for (const part of path) {
            if (current == null || typeof current !== 'object') return undefined;
            current = current[part];
        }
        return current;
    }
    // Simple field name
    return data[expr];
}

// Helper function to evaluate comparison expressions
function evaluateExpr(expression, data) {
    const operators = ['==', '!=', '>=', '<=', '>', '<'];
    for (const op of operators) {
        const parts = expression.split(op);
        if (parts.length === 2) {
            const left = resolveValue(parts[0], data);
            const right = resolveValue(parts[1], data);
            switch (op) {
                case '==': return left == right;
                case '!=': return left != right;
                case '>=': return left >= right;
                case '<=': return left <= right;
                case '>': return left > right;
                case '<': return left < right;
            }
        }
    }
    // No operator found, check if value is truthy
    return !!resolveValue(expression, data);
}

const result = evaluateExpr(config.expression, input);
return {
    ...input,
    __branch: result ? 'true' : 'false'
};
`,
		UIConfig: LSchema(`{"icon": "git-branch", "color": "#F59E0B"}`, `{"icon": "git-branch", "color": "#F59E0B"}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{
			LError("COND_001", "INVALID_EXPR", "無効な式", "Invalid condition expression", "無効な条件式です", false),
			LError("COND_002", "EVAL_ERROR", "評価エラー", "Expression evaluation error", "式の評価中にエラーが発生しました", false),
		},
		Enabled: true,
	}
}

func SwitchBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "switch",
		Version:     1,
		Name:        LText("Switch", "スイッチ"),
		Description: LText("Multi-branch routing", "複数分岐ルーティング"),
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryBranching,
		Icon:        "shuffle",
		ConfigSchema: LSchema(`{
			"type": "object",
			"properties": {
				"mode": {"enum": ["rules", "expression"], "type": "string", "title": "Mode", "description": "Evaluation mode"},
				"cases": {
					"type": "array",
					"title": "Cases",
					"description": "Case definitions",
					"items": {
						"type": "object",
						"properties": {
							"name": {"type": "string", "title": "Name"},
							"expression": {"type": "string", "title": "Expression"},
							"is_default": {"type": "boolean", "title": "Is Default"}
						}
					}
				}
			}
		}`, `{
			"type": "object",
			"properties": {
				"mode": {"enum": ["rules", "expression"], "type": "string", "title": "モード", "description": "評価モード"},
				"cases": {
					"type": "array",
					"title": "ケース",
					"description": "ケース定義",
					"items": {
						"type": "object",
						"properties": {
							"name": {"type": "string", "title": "名前"},
							"expression": {"type": "string", "title": "式"},
							"is_default": {"type": "boolean", "title": "デフォルト"}
						}
					}
				}
			}
		}`),
		OutputPorts: []domain.LocalizedOutputPort{
			LPortWithDesc("default", "Default", "デフォルト", "When no case matches", "どのケースにもマッチしない場合", true),
			// Generic case branches - users define case meanings in config
			LPortWithDesc("case_1", "Case 1", "ケース1", "First case branch", "1番目のケース分岐", false),
			LPortWithDesc("case_2", "Case 2", "ケース2", "Second case branch", "2番目のケース分岐", false),
			LPortWithDesc("case_3", "Case 3", "ケース3", "Third case branch", "3番目のケース分岐", false),
			LPortWithDesc("case_4", "Case 4", "ケース4", "Fourth case branch", "4番目のケース分岐", false),
			LPortWithDesc("case_5", "Case 5", "ケース5", "Fifth case branch", "5番目のケース分岐", false),
			LPortWithDesc("case_6", "Case 6", "ケース6", "Sixth case branch", "6番目のケース分岐", false),
		},
		Code: `
// Helper function to resolve JSONPath-like expressions ($.field.nested)
function resolveValue(expr, data) {
    expr = expr.trim();
    // String literal
    if ((expr.startsWith('"') && expr.endsWith('"')) || (expr.startsWith("'") && expr.endsWith("'"))) {
        return expr.slice(1, -1);
    }
    // Boolean literal
    if (expr === 'true') return true;
    if (expr === 'false') return false;
    // Null literal
    if (expr === 'null') return null;
    // Number literal
    const num = parseFloat(expr);
    if (!isNaN(num)) return num;
    // JSONPath ($.field.nested)
    if (expr.startsWith('$.')) {
        const path = expr.slice(2).split('.');
        let current = data;
        for (const part of path) {
            if (current == null || typeof current !== 'object') return undefined;
            current = current[part];
        }
        return current;
    }
    // Simple field name
    return data[expr];
}

// Helper function to evaluate comparison expressions
function evaluateExpr(expression, data) {
    const operators = ['==', '!=', '>=', '<=', '>', '<'];
    for (const op of operators) {
        const parts = expression.split(op);
        if (parts.length === 2) {
            const left = resolveValue(parts[0], data);
            const right = resolveValue(parts[1], data);
            switch (op) {
                case '==': return left == right;
                case '!=': return left != right;
                case '>=': return left >= right;
                case '<=': return left <= right;
                case '>': return left > right;
                case '<': return left < right;
            }
        }
    }
    // No operator found, check if value is truthy
    return !!resolveValue(expression, data);
}

let matchedCase = null;
for (const c of config.cases || []) {
    if (c.is_default) {
        matchedCase = matchedCase || c.name;
        continue;
    }
    if (evaluateExpr(c.expression, input)) {
        matchedCase = c.name;
        break;
    }
}
return {
    ...input,
    __branch: matchedCase || 'default'
};
`,
		UIConfig: LSchema(`{"icon": "shuffle", "color": "#F59E0B"}`, `{"icon": "shuffle", "color": "#F59E0B"}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{
			LError("SWITCH_001", "NO_MATCH", "マッチなし", "No matching case", "マッチするケースがありません", false),
		},
		Enabled: true,
	}
}
