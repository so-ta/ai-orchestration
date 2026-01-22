package blocks

import (
	"github.com/souta/ai-orchestration/internal/domain"
)

func (r *Registry) registerDataBlocks() {
	r.register(SplitBlock())
	r.register(FilterBlock())
	r.register(MapBlock())
	r.register(AggregateBlock())
}

func SplitBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "split",
		Version:     1,
		Name:        LText("Split", "分割"),
		Description: LText("Split into batches", "バッチに分割"),
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryData,
		Icon:        "scissors",
		ConfigSchema: LSchema(`{
			"type": "object",
			"properties": {
				"batch_size": {"type": "integer", "minimum": 1, "title": "Batch Size", "description": "Number of items per batch"},
				"input_path": {"type": "string", "title": "Input Path", "description": "JSONPath to the array to split"}
			}
		}`, `{
			"type": "object",
			"properties": {
				"batch_size": {"type": "integer", "minimum": 1, "title": "バッチサイズ", "description": "バッチあたりのアイテム数"},
				"input_path": {"type": "string", "title": "入力パス", "description": "分割する配列へのJSONPath"}
			}
		}`),
		OutputPorts: []domain.LocalizedOutputPort{
			LPortWithDesc("output", "Output", "出力", "Split batches", "分割されたバッチ", true),
		},
		Code: `
const items = getPath(input, config.input_path) || [];
const batchSize = config.batch_size || 10;
const batches = [];
for (let i = 0; i < items.length; i += batchSize) {
    batches.push(items.slice(i, i + batchSize));
}
return {
    ...input,
    batches,
    batch_count: batches.length,
    total_items: items.length
};
`,
		UIConfig:   LSchema(`{"icon": "scissors", "color": "#06B6D4"}`, `{"icon": "scissors", "color": "#06B6D4"}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{},
		Enabled:    true,
		TestCases: []BlockTestCase{
			{
				Name:   "split array into batches of 2",
				Input:  map[string]interface{}{"items": []interface{}{1, 2, 3, 4, 5}},
				Config: map[string]interface{}{"input_path": "items", "batch_size": 2},
				ExpectedOutput: map[string]interface{}{
					"batch_count": 3,
					"total_items": 5,
				},
			},
			{
				Name:   "empty array",
				Input:  map[string]interface{}{"items": []interface{}{}},
				Config: map[string]interface{}{"input_path": "items", "batch_size": 10},
				ExpectedOutput: map[string]interface{}{
					"batch_count": 0,
					"total_items": 0,
				},
			},
		},
	}
}

func FilterBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "filter",
		Version:     1,
		Name:        LText("Filter", "フィルター"),
		Description: LText("Filter items by condition", "条件でアイテムをフィルター"),
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryData,
		Icon:        "filter",
		ConfigSchema: LSchema(`{
			"type": "object",
			"properties": {
				"keep_all": {"type": "boolean", "title": "Keep All", "description": "Keep all items without filtering"},
				"expression": {"type": "string", "title": "Expression", "description": "Filter expression"}
			}
		}`, `{
			"type": "object",
			"properties": {
				"keep_all": {"type": "boolean", "title": "全て保持", "description": "フィルターせずに全てのアイテムを保持"},
				"expression": {"type": "string", "title": "式", "description": "フィルター式"}
			}
		}`),
		OutputPorts: []domain.LocalizedOutputPort{
			LPortWithDesc("matched", "Matched", "マッチ", "Items matching condition", "条件にマッチしたアイテム", true),
			LPortWithDesc("unmatched", "Unmatched", "アンマッチ", "Items not matching", "条件にマッチしなかったアイテム", false),
		},
		Code: `
const items = Array.isArray(input) ? input : (input.items || []);
const filtered = items.filter(item => evaluate(config.expression, item));
return {
    items: filtered,
    original_count: items.length,
    filtered_count: filtered.length,
    removed_count: items.length - filtered.length
};
`,
		UIConfig: LSchema(`{"icon": "filter", "color": "#06B6D4"}`, `{"icon": "filter", "color": "#06B6D4"}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{
			LError("FILTER_001", "INVALID_EXPR", "無効な式", "Invalid filter expression", "無効なフィルター式です", false),
		},
		Enabled: true,
	}
}

func MapBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "map",
		Version:     1,
		Name:        LText("Map", "マップ"),
		Description: LText("Process array items in parallel", "配列アイテムを並列処理"),
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryData,
		Icon:        "layers",
		ConfigSchema: LSchema(`{
			"type": "object",
			"properties": {
				"parallel": {"type": "boolean", "title": "Parallel", "description": "Process items in parallel"},
				"input_path": {"type": "string", "title": "Input Path", "description": "JSONPath to the array"},
				"max_workers": {"type": "integer", "title": "Max Workers", "description": "Maximum parallel workers"}
			}
		}`, `{
			"type": "object",
			"properties": {
				"parallel": {"type": "boolean", "title": "並列処理", "description": "アイテムを並列で処理"},
				"input_path": {"type": "string", "title": "入力パス", "description": "配列へのJSONPath"},
				"max_workers": {"type": "integer", "title": "最大ワーカー数", "description": "最大並列ワーカー数"}
			}
		}`),
		OutputPorts: []domain.LocalizedOutputPort{
			LPortWithDesc("item", "Item", "アイテム", "Each mapped item", "各マップされたアイテム", true),
			LPortWithDesc("complete", "Complete", "完了", "All items processed", "全アイテム処理完了", false),
		},
		Code: `
const items = getPath(input, config.input_path) || [];
const maxWorkers = config.max_workers || 10;
let results;
if (config.parallel) {
    results = Promise.all(
        items.map((item, index) => ({ item, index, processed: true }))
    );
} else {
    results = items.map((item, index) => ({ item, index, processed: true }));
}
return {
    ...input,
    items: results,
    count: results.length,
    success_count: results.length,
    error_count: 0
};
`,
		UIConfig: LSchema(`{"icon": "layers", "color": "#06B6D4"}`, `{"icon": "layers", "color": "#06B6D4"}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{
			LError("MAP_001", "INVALID_PATH", "無効なパス", "Invalid input path", "無効な入力パスです", false),
		},
		Enabled: true,
	}
}

func AggregateBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "aggregate",
		Version:     1,
		Name:        LText("Aggregate", "集計"),
		Description: LText("Aggregate data operations", "データ集計操作"),
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryData,
		Icon:        "database",
		ConfigSchema: LSchema(`{
			"type": "object",
			"properties": {
				"group_by": {"type": "string", "title": "Group By", "description": "Field to group by"},
				"operations": {
					"type": "array",
					"title": "Operations",
					"description": "Aggregation operations to perform",
					"items": {
						"type": "object",
						"properties": {
							"field": {"type": "string", "title": "Field"},
							"operation": {"enum": ["sum", "count", "avg", "min", "max", "first", "last", "concat"], "type": "string", "title": "Operation"},
							"output_field": {"type": "string", "title": "Output Field"}
						}
					}
				}
			}
		}`, `{
			"type": "object",
			"properties": {
				"group_by": {"type": "string", "title": "グループ化フィールド", "description": "グループ化するフィールド"},
				"operations": {
					"type": "array",
					"title": "操作",
					"description": "実行する集計操作",
					"items": {
						"type": "object",
						"properties": {
							"field": {"type": "string", "title": "フィールド"},
							"operation": {"enum": ["sum", "count", "avg", "min", "max", "first", "last", "concat"], "type": "string", "title": "操作"},
							"output_field": {"type": "string", "title": "出力フィールド"}
						}
					}
				}
			}
		}`),
		OutputPorts: []domain.LocalizedOutputPort{
			LPortWithDesc("output", "Output", "出力", "Aggregated result", "集計結果", true),
		},
		Code: `
const items = Array.isArray(input) ? input : (input.items || []);
const result = {};
for (const op of config.operations || []) {
    const values = items.map(item => getPath(item, op.field));
    switch (op.operation) {
        case 'sum': result[op.output_field] = values.reduce((a, b) => a + b, 0); break;
        case 'count': result[op.output_field] = values.length; break;
        case 'avg': result[op.output_field] = values.reduce((a, b) => a + b, 0) / values.length; break;
        case 'min': result[op.output_field] = Math.min(...values); break;
        case 'max': result[op.output_field] = Math.max(...values); break;
        case 'first': result[op.output_field] = values[0]; break;
        case 'last': result[op.output_field] = values[values.length - 1]; break;
        case 'concat': result[op.output_field] = values.join(''); break;
    }
}
return result;
`,
		UIConfig:   LSchema(`{"icon": "database", "color": "#06B6D4"}`, `{"icon": "database", "color": "#06B6D4"}`),
		ErrorCodes: []domain.LocalizedErrorCodeDef{},
		Enabled:    true,
	}
}
