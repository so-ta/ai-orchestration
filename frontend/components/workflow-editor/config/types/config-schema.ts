/**
 * ブロック設定スキーマの型定義
 *
 * 標準JSON Schemaをベースに、UI生成に必要な型を定義。
 * 拡張プロパティは最小限に抑え、標準属性からウィジェットを自動推論する設計。
 */

// =============================================================================
// JSON Schema Property Types
// =============================================================================

export type JSONSchemaType = 'string' | 'number' | 'integer' | 'boolean' | 'array' | 'object';

export type JSONSchemaFormat = 'uri' | 'email' | 'date-time' | 'date' | 'time' | 'uuid';

export interface JSONSchemaProperty {
  type: JSONSchemaType;
  title?: string;
  description?: string;
  default?: unknown;

  // Enum
  enum?: (string | number)[];
  const?: string | number | boolean;

  // Number constraints
  minimum?: number;
  maximum?: number;
  exclusiveMinimum?: number;
  exclusiveMaximum?: number;
  multipleOf?: number;

  // String constraints
  minLength?: number;
  maxLength?: number;
  pattern?: string;
  format?: JSONSchemaFormat;

  // Array constraints
  items?: JSONSchemaProperty;
  minItems?: number;
  maxItems?: number;
  uniqueItems?: boolean;

  // Object constraints
  properties?: Record<string, JSONSchemaProperty>;
  required?: string[];
  additionalProperties?: boolean | JSONSchemaProperty;

  // UI hints (x- prefixed custom attributes)
  'x-ui-widget'?: string;
  'x-ui-language'?: string;
  'x-required'?: boolean;

  // Allow any additional x- properties
  [key: `x-${string}`]: unknown;
}

// =============================================================================
// Config Schema (Root)
// =============================================================================

export interface ConditionalSchema {
  if?: {
    properties: Record<string, { const: unknown }>;
  };
  then?: {
    required?: string[];
    properties?: Record<string, Partial<JSONSchemaProperty>>;
  };
  else?: {
    required?: string[];
    properties?: Record<string, Partial<JSONSchemaProperty>>;
  };
}

export interface ConfigSchema {
  type: 'object';
  properties: Record<string, JSONSchemaProperty>;
  required?: string[];
  allOf?: ConditionalSchema[];
}

// =============================================================================
// UI Config (Optional, stored in ui_config)
// =============================================================================

export type WidgetType =
  | 'text'
  | 'textarea'
  | 'number'
  | 'slider'
  | 'select'
  | 'radio'
  | 'checkbox'
  | 'switch'
  | 'code'
  | 'template-editor'
  | 'json'
  | 'key-value'
  | 'array'
  | 'secret'
  | 'color'
  | 'datetime';

export interface FieldOverride {
  widget?: WidgetType;
  rows?: number;
  language?: string;
  placeholder?: string;
  step?: number;
}

export interface UIGroup {
  id: string;
  title: string;
  collapsed?: boolean;
  icon?: string;
}

export interface UIConfig {
  icon?: string;
  color?: string;
  fieldOverrides?: Record<string, FieldOverride>;
  groups?: UIGroup[];
  fieldGroups?: Record<string, string>;
  fieldOrder?: string[];
}

// =============================================================================
// Parsed Field (for rendering)
// =============================================================================

export interface ParsedField {
  name: string;
  property: JSONSchemaProperty;
  required: boolean;
  widget: WidgetType;
  group?: string;
  order: number;
  override?: FieldOverride;
  visible: boolean;
}

export interface ParsedSchema {
  fields: ParsedField[];
  groups: UIGroup[];
  conditionalRules: ConditionalSchema[];
}

// =============================================================================
// Widget Value Types
// =============================================================================

/**
 * ウィジェットに渡される値の型（型安全なキャスト用）
 * Note: null is converted to undefined at ConfigFieldRenderer level
 */
export type WidgetValue =
  | string
  | number
  | boolean
  | unknown[]
  | Record<string, unknown>
  | undefined;

// =============================================================================
// Validation
// =============================================================================

export interface ValidationError {
  field: string;
  message: string;
  keyword: string;
}

export interface ValidationResult {
  valid: boolean;
  errors: ValidationError[];
}
