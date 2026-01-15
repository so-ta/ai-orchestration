package blocks

import (
	"encoding/json"

	"github.com/souta/ai-orchestration/internal/domain"
)

func (r *Registry) registerDataBlocks() {
	r.register(SplitBlock())
	r.register(FilterBlock())
	r.register(JoinBlock())
	r.register(MapBlock())
	r.register(AggregateBlock())
}

func SplitBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "split",
		Version:     1,
		Name:        "Split",
		Description: "Split into batches",
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryData,
		Icon:        "scissors",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"batch_size": {"type": "integer", "minimum": 1},
				"input_path": {"type": "string"}
			}
		}`),
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"items": {"type": "array", "description": "分割対象の配列"}
			},
			"description": "Split処理の入力データ"
		}`),
		InputPorts: []domain.InputPort{
			{Name: "input", Label: "Input", Schema: json.RawMessage(`{"type": "any"}`), Required: true, Description: "Data to split into branches"},
		},
		OutputPorts: []domain.OutputPort{
			{Name: "output", Label: "Output", IsDefault: true, Description: "Split batches"},
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
		UIConfig:   json.RawMessage(`{"icon": "scissors", "color": "#06B6D4"}`),
		ErrorCodes: []domain.ErrorCodeDef{},
		Enabled:    true,
		TestCases: []BlockTestCase{
			{
				Name:   "split array into batches of 2",
				Input:  map[string]interface{}{"items": []interface{}{1, 2, 3, 4, 5}},
				Config: map[string]interface{}{"input_path": "items", "batch_size": 2},
				ExpectedOutput: map[string]interface{}{
					"batch_count":  3,
					"total_items":  5,
				},
			},
			{
				Name:   "empty array",
				Input:  map[string]interface{}{"items": []interface{}{}},
				Config: map[string]interface{}{"input_path": "items", "batch_size": 10},
				ExpectedOutput: map[string]interface{}{
					"batch_count":  0,
					"total_items":  0,
				},
			},
		},
	}
}

func FilterBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "filter",
		Version:     1,
		Name:        "Filter",
		Description: "Filter items by condition",
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryData,
		Icon:        "filter",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"keep_all": {"type": "boolean"},
				"expression": {"type": "string"}
			}
		}`),
		InputSchema: json.RawMessage(`{
			"type": "object",
			"required": ["items"],
			"properties": {
				"items": {"type": "array", "description": "フィルター対象の配列"}
			},
			"description": "Filter処理の入力データ"
		}`),
		InputPorts: []domain.InputPort{
			{Name: "items", Label: "Items", Schema: json.RawMessage(`{"type": "array", "items": {"type": "any"}}`), Required: true, Description: "Array of items to filter"},
		},
		OutputPorts: []domain.OutputPort{
			{Name: "matched", Label: "Matched", IsDefault: true, Description: "Items matching condition"},
			{Name: "unmatched", Label: "Unmatched", IsDefault: false, Description: "Items not matching"},
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
		UIConfig: json.RawMessage(`{"icon": "filter", "color": "#06B6D4"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "FILTER_001", Name: "INVALID_EXPR", Description: "Invalid filter expression", Retryable: false},
		},
		Enabled: true,
	}
}

func JoinBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "join",
		Version:     1,
		Name:        "Join",
		Description: "Merge multiple branches",
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryData,
		Icon:        "git-merge",
		ConfigSchema: json.RawMessage(`{}`),
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"data": {"type": "object", "description": "マージするデータ"}
			},
			"description": "マージするデータ"
		}`),
		InputPorts: []domain.InputPort{
			{Name: "input_1", Label: "Input 1", Schema: json.RawMessage(`{"type": "any"}`), Required: false, Description: "First branch result"},
			{Name: "input_2", Label: "Input 2", Schema: json.RawMessage(`{"type": "any"}`), Required: false, Description: "Second branch result"},
			{Name: "input_3", Label: "Input 3", Schema: json.RawMessage(`{"type": "any"}`), Required: false, Description: "Third branch result"},
			{Name: "input_4", Label: "Input 4", Schema: json.RawMessage(`{"type": "any"}`), Required: false, Description: "Fourth branch result"},
		},
		OutputPorts: []domain.OutputPort{
			{Name: "output", Label: "Output", IsDefault: true, Description: "Merged data"},
		},
		Code:       `return input;`,
		UIConfig:   json.RawMessage(`{"icon": "git-merge", "color": "#06B6D4"}`),
		ErrorCodes: []domain.ErrorCodeDef{},
		Enabled:    true,
	}
}

func MapBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "map",
		Version:     1,
		Name:        "Map",
		Description: "Process array items in parallel",
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryData,
		Icon:        "layers",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"parallel": {"type": "boolean"},
				"input_path": {"type": "string"},
				"max_workers": {"type": "integer"}
			}
		}`),
		InputSchema: json.RawMessage(`{
			"type": "object",
			"required": ["items"],
			"properties": {
				"items": {"type": "array", "description": "処理対象の配列"}
			},
			"description": "Map処理の入力データ"
		}`),
		InputPorts: []domain.InputPort{
			{Name: "items", Label: "Items", Schema: json.RawMessage(`{"type": "array", "items": {"type": "any"}}`), Required: true, Description: "Array of items to process"},
		},
		OutputPorts: []domain.OutputPort{
			{Name: "item", Label: "Item", IsDefault: true, Description: "Each mapped item"},
			{Name: "complete", Label: "Complete", IsDefault: false, Description: "All items processed"},
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
		UIConfig: json.RawMessage(`{"icon": "layers", "color": "#06B6D4"}`),
		ErrorCodes: []domain.ErrorCodeDef{
			{Code: "MAP_001", Name: "INVALID_PATH", Description: "Invalid input path", Retryable: false},
		},
		Enabled: true,
	}
}

func AggregateBlock() *SystemBlockDefinition {
	return &SystemBlockDefinition{
		Slug:        "aggregate",
		Version:     1,
		Name:        "Aggregate",
		Description: "Aggregate data operations",
		Category:    domain.BlockCategoryFlow,
		Subcategory: domain.BlockSubcategoryData,
		Icon:        "database",
		ConfigSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"group_by": {"type": "string"},
				"operations": {
					"type": "array",
					"items": {
						"type": "object",
						"properties": {
							"field": {"type": "string"},
							"operation": {"enum": ["sum", "count", "avg", "min", "max", "first", "last", "concat"], "type": "string"},
							"output_field": {"type": "string"}
						}
					}
				}
			}
		}`),
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"items": {"type": "array", "description": "集計対象の配列"}
			},
			"description": "Aggregate処理の入力データ"
		}`),
		InputPorts: []domain.InputPort{
			{Name: "input_1", Label: "Input 1", Schema: json.RawMessage(`{"type": "any"}`), Required: false, Description: "First data source"},
			{Name: "input_2", Label: "Input 2", Schema: json.RawMessage(`{"type": "any"}`), Required: false, Description: "Second data source"},
			{Name: "input_3", Label: "Input 3", Schema: json.RawMessage(`{"type": "any"}`), Required: false, Description: "Third data source"},
			{Name: "input_4", Label: "Input 4", Schema: json.RawMessage(`{"type": "any"}`), Required: false, Description: "Fourth data source"},
		},
		OutputPorts: []domain.OutputPort{
			{Name: "output", Label: "Output", IsDefault: true, Description: "Aggregated result"},
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
		UIConfig:   json.RawMessage(`{"icon": "database", "color": "#06B6D4"}`),
		ErrorCodes: []domain.ErrorCodeDef{},
		Enabled:    true,
	}
}
