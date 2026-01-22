import type { Ref } from 'vue'
import type { SchemaField, OutputSchemaObject } from './types'
import { useOutputSchema } from './useOutputSchema'

/**
 * Composable for field CRUD operations
 */
export function useFieldOperations(
  fields: Ref<SchemaField[]>,
  emit: (e: 'update:modelValue', value: OutputSchemaObject) => void,
) {
  const { fieldsToSchema, schemaToJsonText } = useOutputSchema()

  /**
   * Emit changes to parent
   */
  function emitChanges(): string {
    const schema = fieldsToSchema(fields.value)
    emit('update:modelValue', schema)
    return schemaToJsonText(schema)
  }

  /**
   * Add new field at root level
   */
  function addField() {
    fields.value.push({
      id: crypto.randomUUID(),
      name: '',
      type: 'string',
      title: '',
      description: '',
      required: false,
      expanded: true,
    })
  }

  /**
   * Add child field to a parent (for nested objects)
   */
  function addChildField(parentField: SchemaField) {
    if (!parentField.children) {
      parentField.children = []
    }
    parentField.children.push({
      id: crypto.randomUUID(),
      name: '',
      type: 'string',
      title: '',
      description: '',
      required: false,
      expanded: true,
    })
    emitChanges()
  }

  /**
   * Add item child field (for array of objects)
   */
  function addItemChildField(parentField: SchemaField) {
    if (!parentField.itemChildren) {
      parentField.itemChildren = []
    }
    parentField.itemChildren.push({
      id: crypto.randomUUID(),
      name: '',
      type: 'string',
      title: '',
      description: '',
      required: false,
      expanded: true,
    })
    emitChanges()
  }

  /**
   * Remove field from array
   */
  function removeFieldFromArray(array: SchemaField[], index: number) {
    array.splice(index, 1)
    emitChanges()
  }

  /**
   * Update field property
   */
  function updateField(field: SchemaField, key: keyof SchemaField, value: SchemaField[keyof SchemaField]) {
    // Handle type change - reset children/itemChildren appropriately
    if (key === 'type') {
      if (value === 'object') {
        field.children = field.children || []
        field.itemType = undefined
        field.itemChildren = undefined
      } else if (value === 'array') {
        field.itemType = field.itemType || 'string'
        field.children = undefined
        if (field.itemType === 'object') {
          field.itemChildren = field.itemChildren || []
        }
      } else {
        field.children = undefined
        field.itemType = undefined
        field.itemChildren = undefined
      }
    }

    // Handle itemType change
    if (key === 'itemType') {
      if (value === 'object') {
        field.itemChildren = field.itemChildren || []
      } else {
        field.itemChildren = undefined
      }
    }

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (field as any)[key] = value
    emitChanges()
  }

  /**
   * Toggle field expansion
   */
  function toggleExpand(field: SchemaField) {
    field.expanded = !field.expanded
  }

  return {
    emitChanges,
    addField,
    addChildField,
    addItemChildField,
    removeFieldFromArray,
    updateField,
    toggleExpand,
  }
}
