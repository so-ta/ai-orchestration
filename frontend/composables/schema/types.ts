/**
 * Output schema field interface with children support
 */
export interface SchemaField {
  id: string
  name: string
  type: string
  title: string
  description: string
  required: boolean
  // For nested objects
  children?: SchemaField[]
  // For arrays
  itemType?: string
  itemChildren?: SchemaField[] // For array of objects
  // UI state
  expanded?: boolean
}

/**
 * JSON Schema structure for output schema
 */
export interface OutputSchemaObject {
  type?: string
  properties?: Record<string, SchemaProperty>
  required?: string[]
  items?: SchemaProperty
}

/**
 * Schema property definition
 */
export interface SchemaProperty {
  type?: string
  title?: string
  description?: string
  properties?: Record<string, SchemaProperty>
  required?: string[]
  items?: SchemaProperty
}

/**
 * Field type options
 */
export const FIELD_TYPES = [
  { value: 'string', label: '文字列' },
  { value: 'number', label: '数値' },
  { value: 'boolean', label: '真偽値' },
  { value: 'object', label: 'オブジェクト' },
  { value: 'array', label: '配列' },
] as const

/**
 * Array item type options (excludes nested array for simplicity)
 */
export const ARRAY_ITEM_TYPES = [
  { value: 'string', label: '文字列' },
  { value: 'number', label: '数値' },
  { value: 'boolean', label: '真偽値' },
  { value: 'object', label: 'オブジェクト' },
] as const
