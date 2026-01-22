import type { ConfigSchema } from '~/components/workflow-editor/config/types/config-schema'

export interface SchemaField {
  name: string
  type: string
  description: string
  required: boolean
}

/**
 * Composable for schema field parsing and utilities
 */
export function useSchemaFields() {
  /**
   * Extract schema fields for preview from a schema object
   */
  function getSchemaFields(schema: Record<string, unknown> | undefined | null): SchemaField[] {
    if (!schema) return []
    const properties = schema.properties as Record<string, Record<string, unknown>> | undefined
    if (!properties) return []

    const required = (schema.required as string[]) || []

    return Object.entries(properties).map(([name, prop]) => ({
      name,
      type: String(prop.type || 'any'),
      description: String(prop.description || ''),
      required: required.includes(name),
    }))
  }

  /**
   * Generate example JSON from schema fields
   */
  function generateExampleJson(fields: SchemaField[]): string {
    if (fields.length === 0) return '{}'

    const example: Record<string, unknown> = {}
    for (const field of fields) {
      switch (field.type) {
        case 'string':
          example[field.name] = ''
          break
        case 'number':
        case 'integer':
          example[field.name] = 0
          break
        case 'boolean':
          example[field.name] = false
          break
        case 'array':
          example[field.name] = []
          break
        case 'object':
          example[field.name] = {}
          break
        default:
          example[field.name] = null
      }
    }
    return JSON.stringify(example, null, 2)
  }

  /**
   * Get CSS class for type badge
   */
  function getTypeBadgeClass(type: string): string {
    switch (type) {
      case 'string': return 'type-string'
      case 'number':
      case 'integer': return 'type-number'
      case 'boolean': return 'type-boolean'
      case 'array': return 'type-array'
      case 'object': return 'type-object'
      default: return 'type-any'
    }
  }

  /**
   * Convert block config_schema to ConfigSchema format
   */
  function toConfigSchema(schema: Record<string, unknown> | undefined | null): ConfigSchema | null {
    if (!schema || schema.type !== 'object') return null
    const properties = schema.properties as Record<string, unknown> | undefined
    if (!properties || Object.keys(properties).length === 0) return null
    return {
      type: 'object',
      properties: properties || {},
      required: (schema.required as string[]) || [],
    } as ConfigSchema
  }

  /**
   * Check if schema has any fields
   */
  function hasFields(schema: ConfigSchema | null): boolean {
    if (!schema?.properties) return false
    return Object.keys(schema.properties).length > 0
  }

  return {
    getSchemaFields,
    generateExampleJson,
    getTypeBadgeClass,
    toConfigSchema,
    hasFields,
  }
}
