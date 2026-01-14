import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import OutputSchemaWidget from './OutputSchemaWidget.vue'

// Mock crypto.randomUUID
let uuidCounter = 0
beforeEach(() => {
  uuidCounter = 0
  vi.stubGlobal('crypto', {
    randomUUID: () => `test-uuid-${++uuidCounter}`
  })
})

describe('OutputSchemaWidget', () => {
  const defaultProps = {
    name: 'output_schema',
    property: {
      type: 'object' as const,
      title: '出力スキーマ',
      description: '出力データのスキーマを定義'
    },
    modelValue: undefined
  }

  it('renders with title and description', () => {
    const wrapper = mount(OutputSchemaWidget, {
      props: defaultProps
    })

    expect(wrapper.find('.widget-label').text()).toContain('出力スキーマ')
    expect(wrapper.find('.widget-description').text()).toBe('出力データのスキーマを定義')
  })

  it('uses name as title when title is not provided', () => {
    const wrapper = mount(OutputSchemaWidget, {
      props: {
        ...defaultProps,
        property: { type: 'object' as const }
      }
    })

    expect(wrapper.find('.widget-label').text()).toContain('output_schema')
  })

  it('shows empty state when modelValue is undefined', () => {
    const wrapper = mount(OutputSchemaWidget, {
      props: defaultProps
    })

    expect(wrapper.find('.empty-state').exists()).toBe(true)
    expect(wrapper.findAll('.field-item')).toHaveLength(0)
  })

  it('initializes fields from modelValue schema', () => {
    const schema = {
      type: 'object',
      properties: {
        name: { type: 'string', title: '名前' },
        age: { type: 'number', title: '年齢' }
      },
      required: ['name']
    }

    const wrapper = mount(OutputSchemaWidget, {
      props: {
        ...defaultProps,
        modelValue: schema
      }
    })

    const fieldItems = wrapper.findAll('.field-item')
    expect(fieldItems).toHaveLength(2)
  })

  it('adds new field when add button is clicked', async () => {
    const wrapper = mount(OutputSchemaWidget, {
      props: defaultProps
    })

    const addButton = wrapper.find('.add-field-button')
    await addButton.trigger('click')

    const fieldItems = wrapper.findAll('.field-item')
    expect(fieldItems).toHaveLength(1)

    // New field is added but not emitted until name is filled
    // This is by design - empty fields are not included in schema
  })

  it('removes field when remove button is clicked', async () => {
    const schema = {
      type: 'object',
      properties: {
        name: { type: 'string' }
      }
    }

    const wrapper = mount(OutputSchemaWidget, {
      props: {
        ...defaultProps,
        modelValue: schema
      }
    })

    const removeButton = wrapper.find('.remove-button')
    await removeButton.trigger('click')

    expect(wrapper.find('.empty-state').exists()).toBe(true)
  })

  it('updates field name and emits change', async () => {
    const wrapper = mount(OutputSchemaWidget, {
      props: defaultProps
    })

    // Add a field first
    const addButton = wrapper.find('.add-field-button')
    await addButton.trigger('click')

    // Find the name input and update it
    const nameInput = wrapper.find('.name-group .field-input')
    await nameInput.setValue('testField')

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeTruthy()
    const lastEmit = emitted![emitted!.length - 1][0] as any
    expect(lastEmit.properties?.testField).toBeDefined()
  })

  it('updates field type and emits change', async () => {
    const schema = {
      type: 'object',
      properties: {
        myField: { type: 'string' }
      }
    }

    const wrapper = mount(OutputSchemaWidget, {
      props: {
        ...defaultProps,
        modelValue: schema
      }
    })

    const typeSelect = wrapper.find('.field-select')
    await typeSelect.setValue('number')

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeTruthy()
  })

  it('toggles required status', async () => {
    const schema = {
      type: 'object',
      properties: {
        myField: { type: 'string' }
      }
    }

    const wrapper = mount(OutputSchemaWidget, {
      props: {
        ...defaultProps,
        modelValue: schema
      }
    })

    const requiredCheckbox = wrapper.find('.required-group input[type="checkbox"]')
    await requiredCheckbox.setValue(true)

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeTruthy()
    const lastEmit = emitted![emitted!.length - 1][0] as any
    expect(lastEmit.required).toContain('myField')
  })

  it('toggles JSON editor view', async () => {
    const wrapper = mount(OutputSchemaWidget, {
      props: defaultProps
    })

    // Initially should show visual editor
    expect(wrapper.find('.visual-editor').exists()).toBe(true)
    expect(wrapper.find('.json-editor').exists()).toBe(false)

    // Find and click the toggle button
    const toggleButton = wrapper.find('.toggle-button')
    await toggleButton.trigger('click')

    expect(wrapper.find('.visual-editor').exists()).toBe(false)
    expect(wrapper.find('.json-editor').exists()).toBe(true)
  })

  it('emits blur event on input blur', async () => {
    const wrapper = mount(OutputSchemaWidget, {
      props: defaultProps
    })

    // Add a field to have an input
    const addButton = wrapper.find('.add-field-button')
    await addButton.trigger('click')

    const input = wrapper.find('.field-input')
    await input.trigger('blur')

    expect(wrapper.emitted('blur')).toBeTruthy()
  })

  it('disables inputs when disabled prop is true', async () => {
    const schema = {
      type: 'object',
      properties: {
        myField: { type: 'string' }
      }
    }

    const wrapper = mount(OutputSchemaWidget, {
      props: {
        ...defaultProps,
        modelValue: schema,
        disabled: true
      }
    })

    const nameInput = wrapper.find('.field-input')
    expect(nameInput.attributes('disabled')).toBeDefined()

    const addButton = wrapper.find('.add-field-button')
    expect(addButton.attributes('disabled')).toBeDefined()
  })

  it('displays all field types in select', async () => {
    const schema = {
      type: 'object',
      properties: {
        myField: { type: 'string' }
      }
    }

    const wrapper = mount(OutputSchemaWidget, {
      props: {
        ...defaultProps,
        modelValue: schema
      }
    })

    const typeSelect = wrapper.find('.field-select')
    const options = typeSelect.findAll('option')

    expect(options.length).toBeGreaterThanOrEqual(5)
    expect(options.some(o => o.text() === '文字列')).toBe(true)
    expect(options.some(o => o.text() === '数値')).toBe(true)
    expect(options.some(o => o.text() === '真偽値')).toBe(true)
    expect(options.some(o => o.text() === 'オブジェクト')).toBe(true)
    expect(options.some(o => o.text() === '配列')).toBe(true)
  })

  it('handles JSON editor input', async () => {
    const wrapper = mount(OutputSchemaWidget, {
      props: defaultProps
    })

    // Switch to JSON editor
    const toggleButton = wrapper.find('.toggle-button')
    await toggleButton.trigger('click')

    // Input JSON
    const textarea = wrapper.find('.json-textarea')
    await textarea.setValue('{"type":"object","properties":{"test":{"type":"string"}}}')
    await textarea.trigger('input')

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeTruthy()
  })

  it('handles empty JSON input', async () => {
    const wrapper = mount(OutputSchemaWidget, {
      props: {
        ...defaultProps,
        modelValue: {
          type: 'object',
          properties: { field1: { type: 'string' } }
        }
      }
    })

    // Switch to JSON editor
    const toggleButton = wrapper.find('.toggle-button')
    await toggleButton.trigger('click')

    // Clear JSON
    const textarea = wrapper.find('.json-textarea')
    await textarea.setValue('')
    await textarea.trigger('input')

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeTruthy()
    const lastEmit = emitted![emitted!.length - 1][0] as any
    expect(Object.keys(lastEmit)).toHaveLength(0)
  })

  it('shows parse error for invalid JSON', async () => {
    const wrapper = mount(OutputSchemaWidget, {
      props: defaultProps
    })

    // Switch to JSON editor
    const toggleButton = wrapper.find('.toggle-button')
    await toggleButton.trigger('click')

    // Input invalid JSON
    const textarea = wrapper.find('.json-textarea')
    await textarea.setValue('invalid json')
    await textarea.trigger('input')

    expect(wrapper.find('.parse-error').exists()).toBe(true)
  })

  it('syncs JSON text when toggling to JSON view', async () => {
    const schema = {
      type: 'object',
      properties: {
        name: { type: 'string' }
      }
    }

    const wrapper = mount(OutputSchemaWidget, {
      props: {
        ...defaultProps,
        modelValue: schema
      }
    })

    // Toggle to JSON view
    const toggleButton = wrapper.find('.toggle-button')
    await toggleButton.trigger('click')

    // Check that JSON textarea has content
    const textarea = wrapper.find('.json-textarea')
    const textareaElement = textarea.element as HTMLTextAreaElement
    expect(textareaElement.value).toContain('"type"')
    expect(textareaElement.value).toContain('"properties"')
  })
})
