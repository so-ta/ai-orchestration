/**
 * JSON Schemaを解析してUI生成用のフィールド情報に変換するcomposable
 *
 * 標準JSON Schemaの属性からウィジェットタイプを自動推論し、
 * オプショナルなui_configで上書き可能。
 */

import { computed, type Ref } from 'vue';
import type {
  ConfigSchema,
  JSONSchemaProperty,
  UIConfig,
  ParsedField,
  ParsedSchema,
  WidgetType,
  UIGroup,
} from '../types/config-schema';

/**
 * JSON Schemaプロパティからウィジェットタイプを推論
 */
export function inferWidgetType(prop: JSONSchemaProperty): WidgetType {
  const { type, format, enum: enumValues, maxLength, additionalProperties } = prop;

  if (type === 'boolean') {
    return 'checkbox';
  }

  if (type === 'integer' || type === 'number') {
    return 'number';
  }

  if (type === 'string') {
    // Format-based inference
    if (format === 'uri') return 'text'; // URL input
    if (format === 'email') return 'text'; // Email input
    if (format === 'date-time' || format === 'date' || format === 'time') return 'datetime';

    // Enum -> Select
    if (enumValues && enumValues.length > 0) {
      return 'select';
    }

    // Long text -> Textarea
    if (maxLength && maxLength > 200) {
      return 'textarea';
    }

    return 'text';
  }

  if (type === 'array') {
    return 'array';
  }

  if (type === 'object') {
    // Key-value pairs
    if (additionalProperties) {
      return 'key-value';
    }
    return 'json';
  }

  return 'text';
}

/**
 * フィールドの表示順序を決定
 */
function getFieldOrder(
  name: string,
  index: number,
  uiConfig?: UIConfig
): number {
  if (uiConfig?.fieldOrder) {
    const orderIndex = uiConfig.fieldOrder.indexOf(name);
    if (orderIndex !== -1) {
      return orderIndex;
    }
  }
  return index + 1000; // Default: preserve original order
}

/**
 * フィールドのグループを決定
 */
function getFieldGroup(name: string, uiConfig?: UIConfig): string | undefined {
  return uiConfig?.fieldGroups?.[name];
}

/**
 * スキーマをパースしてフィールド情報を生成
 */
export function parseSchema(
  schema: ConfigSchema | null | undefined,
  uiConfig?: UIConfig
): ParsedSchema {
  if (!schema || !schema.properties) {
    return {
      fields: [],
      groups: uiConfig?.groups || [],
      conditionalRules: schema?.allOf || [],
    };
  }

  const requiredFields = new Set(schema.required || []);
  const fields: ParsedField[] = [];

  let index = 0;
  for (const [name, property] of Object.entries(schema.properties)) {
    const override = uiConfig?.fieldOverrides?.[name];
    const inferredWidget = inferWidgetType(property);
    const widget = override?.widget || inferredWidget;

    fields.push({
      name,
      property,
      required: requiredFields.has(name),
      widget,
      group: getFieldGroup(name, uiConfig),
      order: getFieldOrder(name, index, uiConfig),
      override,
      visible: true, // Initial visibility
    });

    index++;
  }

  // Sort by order
  fields.sort((a, b) => a.order - b.order);

  return {
    fields,
    groups: uiConfig?.groups || [],
    conditionalRules: schema.allOf || [],
  };
}

/**
 * 条件付きフィールドの表示状態を評価
 */
export function evaluateConditionalVisibility(
  parsedSchema: ParsedSchema,
  values: Record<string, unknown>
): ParsedField[] {
  const { fields, conditionalRules } = parsedSchema;

  // Collect conditionally required fields
  const conditionallyRequired = new Set<string>();
  const conditionallyHidden = new Set<string>();

  for (const rule of conditionalRules) {
    if (!rule.if?.properties) continue;

    // Check if condition matches
    let matches = true;
    for (const [field, constraint] of Object.entries(rule.if.properties)) {
      if ('const' in constraint && values[field] !== constraint.const) {
        matches = false;
        break;
      }
    }

    // Apply then/else
    const applicable = matches ? rule.then : rule.else;
    if (applicable?.required) {
      for (const field of applicable.required) {
        conditionallyRequired.add(field);
      }
    }
  }

  // Update visibility based on conditional rules
  // Fields that are NOT conditionally required AND have conditional rules are hidden
  return fields.map((field) => {
    // Check if this field has any conditional requirement rule
    const hasConditionalRule = conditionalRules.some(
      (rule) =>
        rule.then?.required?.includes(field.name) ||
        rule.else?.required?.includes(field.name)
    );

    // If field has conditional rules, show only when conditionally required
    // Otherwise, always show
    const visible = hasConditionalRule
      ? conditionallyRequired.has(field.name)
      : true;

    return {
      ...field,
      visible,
      required: field.required || conditionallyRequired.has(field.name),
    };
  });
}

/**
 * useSchemaParser composable
 */
export function useSchemaParser(
  schema: Ref<ConfigSchema | null | undefined>,
  uiConfig: Ref<UIConfig | undefined>,
  values: Ref<Record<string, unknown>>
) {
  const parsedSchema = computed(() => parseSchema(schema.value, uiConfig.value));

  const visibleFields = computed(() =>
    evaluateConditionalVisibility(parsedSchema.value, values.value)
  );

  const groups = computed(() => parsedSchema.value.groups);

  const fieldsByGroup = computed(() => {
    const result: Record<string, ParsedField[]> = { _ungrouped: [] };

    for (const group of groups.value) {
      result[group.id] = [];
    }

    for (const field of visibleFields.value) {
      if (!field.visible) continue;

      if (field.group && result[field.group]) {
        result[field.group].push(field);
      } else {
        result._ungrouped.push(field);
      }
    }

    return result;
  });

  return {
    parsedSchema,
    visibleFields,
    groups,
    fieldsByGroup,
  };
}
