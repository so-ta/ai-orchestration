import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import JsonWidget from './JsonWidget.vue'

describe('JsonWidget', () => {
  const defaultProps = {
    name: 'testJson',
    property: {
      type: 'object' as const,
      title: 'Test JSON',
      description: 'A test JSON field'
    },
    modelValue: undefined
  }

  it('renders with title and description', () => {
    const wrapper = mount(JsonWidget, {
      props: defaultProps
    })

    expect(wrapper.find('.json-widget-label').text()).toBe('Test JSON')
    expect(wrapper.find('.json-widget-description').text()).toBe('A test JSON field')
  })

  it('uses name as title when title is not provided', () => {
    const wrapper = mount(JsonWidget, {
      props: {
        ...defaultProps,
        property: { type: 'object' as const }
      }
    })

    expect(wrapper.find('.json-widget-label').text()).toBe('testJson')
  })

  it('initializes with modelValue as JSON string', () => {
    const wrapper = mount(JsonWidget, {
      props: {
        ...defaultProps,
        modelValue: { key: 'value' }
      }
    })

    const textarea = wrapper.find('textarea')
    expect(textarea.element.value).toContain('"key"')
    expect(textarea.element.value).toContain('"value"')
  })

  it('emits update:modelValue on valid JSON input', async () => {
    const wrapper = mount(JsonWidget, {
      props: defaultProps
    })

    const textarea = wrapper.find('textarea')
    await textarea.setValue('{"name": "test"}')
    await textarea.trigger('input')

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeTruthy()
    expect(emitted![emitted!.length - 1]).toEqual([{ name: 'test' }])
  })

  it('shows parse error on invalid JSON', async () => {
    const wrapper = mount(JsonWidget, {
      props: defaultProps
    })

    const textarea = wrapper.find('textarea')
    await textarea.setValue('invalid json')
    await textarea.trigger('input')

    expect(wrapper.find('.parse-error').exists()).toBe(true)
  })

  it('emits empty object on empty input', async () => {
    const wrapper = mount(JsonWidget, {
      props: {
        ...defaultProps,
        modelValue: { key: 'value' }
      }
    })

    const textarea = wrapper.find('textarea')
    await textarea.setValue('')
    await textarea.trigger('input')

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeTruthy()
    expect(emitted![emitted!.length - 1]).toEqual([{}])
  })

  it('formats JSON when format button is clicked', async () => {
    const wrapper = mount(JsonWidget, {
      props: defaultProps
    })

    const textarea = wrapper.find('textarea')
    await textarea.setValue('{"key":"value"}')
    await textarea.trigger('input')

    const formatButton = wrapper.findAll('.action-button').find(btn => btn.text() === '整形')
    expect(formatButton).toBeTruthy()
    await formatButton!.trigger('click')

    // Should be formatted with indentation
    expect(textarea.element.value).toContain('  ')
  })

  it('generates sample JSON when sample button is clicked', async () => {
    const wrapper = mount(JsonWidget, {
      props: defaultProps
    })

    const sampleButton = wrapper.findAll('.action-button').find(btn => btn.text() === 'サンプル')
    expect(sampleButton).toBeTruthy()
    await sampleButton!.trigger('click')

    const textarea = wrapper.find('textarea')
    expect(textarea.element.value).toContain('"type"')
    expect(textarea.element.value).toContain('"properties"')

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeTruthy()
  })

  it('emits blur event on textarea blur', async () => {
    const wrapper = mount(JsonWidget, {
      props: defaultProps
    })

    const textarea = wrapper.find('textarea')
    await textarea.trigger('blur')

    expect(wrapper.emitted('blur')).toBeTruthy()
  })

  it('disables textarea and buttons when disabled prop is true', () => {
    const wrapper = mount(JsonWidget, {
      props: {
        ...defaultProps,
        disabled: true
      }
    })

    const textarea = wrapper.find('textarea')
    expect(textarea.attributes('disabled')).toBeDefined()

    const sampleButton = wrapper.findAll('.action-button').find(btn => btn.text() === 'サンプル')
    expect(sampleButton!.attributes('disabled')).toBeDefined()
  })

  it('shows error message when error prop is provided', () => {
    const wrapper = mount(JsonWidget, {
      props: {
        ...defaultProps,
        error: 'Test error message'
      }
    })

    expect(wrapper.find('.json-widget-error').text()).toBe('Test error message')
  })

  it('updates when modelValue prop changes externally', async () => {
    const wrapper = mount(JsonWidget, {
      props: {
        ...defaultProps,
        modelValue: { original: 'value' }
      }
    })

    await wrapper.setProps({ modelValue: { updated: 'newValue' } })

    const textarea = wrapper.find('textarea')
    expect(textarea.element.value).toContain('"updated"')
    expect(textarea.element.value).toContain('"newValue"')
  })

  it('shows valid indicator when JSON is valid', async () => {
    const wrapper = mount(JsonWidget, {
      props: {
        ...defaultProps,
        modelValue: { valid: true }
      }
    })

    expect(wrapper.find('.json-valid').exists()).toBe(true)
    expect(wrapper.find('.parse-error').exists()).toBe(false)
  })

  it('calculates rows based on content', async () => {
    const wrapper = mount(JsonWidget, {
      props: {
        ...defaultProps,
        modelValue: {
          line1: 'value1',
          line2: 'value2',
          line3: 'value3',
          line4: 'value4',
          line5: 'value5'
        }
      }
    })

    const textarea = wrapper.find('textarea')
    const rows = parseInt(textarea.attributes('rows') || '0')
    expect(rows).toBeGreaterThanOrEqual(6)
  })
})
