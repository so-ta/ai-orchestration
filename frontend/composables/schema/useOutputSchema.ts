import type { SchemaField, OutputSchemaObject, SchemaProperty } from './types'

/**
 * Composable for output schema parsing and conversion
 */
export function useOutputSchema() {
  /**
   * Parse schema to fields (recursive)
   */
  function parseSchemaToFields(schema: unknown, depth = 0): SchemaField[] {
    if (depth > 5) return [] // Prevent infinite recursion

    const schemaObj = schema as OutputSchemaObject | null | undefined
    if (!schemaObj || schemaObj.type !== 'object' || !schemaObj.properties) {
      return []
    }

    const required = schemaObj.required || []
    return Object.entries(schemaObj.properties).map(([name, prop]) => {
      const field: SchemaField = {
        id: crypto.randomUUID(),
        name,
        type: prop.type || 'string',
        title: prop.title || '',
        description: prop.description || '',
        required: required.includes(name),
        expanded: true,
      }

      // Handle nested objects
      if (prop.type === 'object' && prop.properties) {
        field.children = parseSchemaToFields(prop, depth + 1)
      }

      // Handle arrays
      if (prop.type === 'array' && prop.items) {
        field.itemType = prop.items.type || 'string'
        // Handle array of objects
        if (prop.items.type === 'object' && prop.items.properties) {
          field.itemChildren = parseSchemaToFields(prop.items, depth + 1)
        }
      }

      return field
    })
  }

  /**
   * Convert fields to schema (recursive)
   */
  function fieldsToSchema(fields: SchemaField[]): OutputSchemaObject {
    if (fields.length === 0) {
      return {}
    }

    const properties: Record<string, SchemaProperty> = {}
    const required: string[] = []

    for (const field of fields) {
      if (!field.name.trim()) continue

      const prop: SchemaProperty = {
        type: field.type,
        ...(field.title && { title: field.title }),
        ...(field.description && { description: field.description }),
      }

      // Handle nested objects
      if (field.type === 'object' && field.children && field.children.length > 0) {
        const nestedSchema = fieldsToSchema(field.children)
        if (nestedSchema.properties) {
          prop.properties = nestedSchema.properties
          if (nestedSchema.required && nestedSchema.required.length > 0) {
            prop.required = nestedSchema.required
          }
        }
      }

      // Handle arrays
      if (field.type === 'array') {
        const itemType = field.itemType || 'string'
        prop.items = { type: itemType }

        // Handle array of objects
        if (itemType === 'object' && field.itemChildren && field.itemChildren.length > 0) {
          const nestedSchema = fieldsToSchema(field.itemChildren)
          if (nestedSchema.properties) {
            prop.items.properties = nestedSchema.properties
            if (nestedSchema.required && nestedSchema.required.length > 0) {
              prop.items.required = nestedSchema.required
            }
          }
        }
      }

      properties[field.name] = prop

      if (field.required) {
        required.push(field.name)
      }
    }

    return {
      type: 'object',
      properties,
      ...(required.length > 0 && { required }),
    }
  }

  /**
   * Serialize schema to JSON string
   */
  function schemaToJsonText(schema: OutputSchemaObject): string {
    return Object.keys(schema).length > 0 ? JSON.stringify(schema, null, 2) : ''
  }

  /**
   * Get depth class for styling
   */
  function getDepthClass(depth: number): string {
    return `depth-${Math.min(depth, 3)}`
  }

  return {
    parseSchemaToFields,
    fieldsToSchema,
    schemaToJsonText,
    getDepthClass,
  }
}
