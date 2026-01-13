import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import CodeWidget from '../CodeWidget.vue'
import type { JSONSchemaProperty, FieldOverride } from '../../types/config-schema'

describe('CodeWidget', () => {
  const defaultProperty: JSONSchemaProperty = {
    type: 'string',
    title: 'Code',
    description: 'Enter your code here',
  }

  const createWrapper = (props: {
    name?: string
    property?: JSONSchemaProperty
    modelValue?: string
    override?: FieldOverride
    error?: string
    disabled?: boolean
  } = {}) => {
    return mount(CodeWidget, {
      props: {
        name: props.name || 'testCode',
        property: props.property || defaultProperty,
        modelValue: props.modelValue,
        override: props.override,
        error: props.error,
        disabled: props.disabled,
      },
    })
  }

  describe('rendering', () => {
    it('renders with default props', () => {
      const wrapper = createWrapper()

      expect(wrapper.find('.code-widget').exists()).toBe(true)
      expect(wrapper.find('.code-widget-label').text()).toBe('Code')
      expect(wrapper.find('.code-widget-description').text()).toBe('Enter your code here')
    })

    it('renders with title from property', () => {
      const wrapper = createWrapper({
        property: { type: 'string', title: 'Custom Title' },
      })

      expect(wrapper.find('.code-widget-label').text()).toBe('Custom Title')
    })

    it('falls back to name when no title', () => {
      const wrapper = createWrapper({
        name: 'myCodeField',
        property: { type: 'string' },
      })

      expect(wrapper.find('.code-widget-label').text()).toBe('myCodeField')
    })

    it('shows required indicator when x-required is true', () => {
      const wrapper = createWrapper({
        property: { type: 'string', 'x-required': true },
      })

      expect(wrapper.find('.required-indicator').exists()).toBe(true)
    })

    it('hides required indicator when x-required is false', () => {
      const wrapper = createWrapper({
        property: { type: 'string', 'x-required': false },
      })

      expect(wrapper.find('.required-indicator').exists()).toBe(false)
    })

    it('hides description when not provided', () => {
      const wrapper = createWrapper({
        property: { type: 'string' },
      })

      expect(wrapper.find('.code-widget-description').exists()).toBe(false)
    })

    it('shows error message when error prop is set', () => {
      const wrapper = createWrapper({
        error: 'Invalid code syntax',
      })

      expect(wrapper.find('.code-widget-error').text()).toBe('Invalid code syntax')
      expect(wrapper.find('.code-input').classes()).toContain('has-error')
    })

    it('hides error message when error prop is not set', () => {
      const wrapper = createWrapper()

      expect(wrapper.find('.code-widget-error').exists()).toBe(false)
    })
  })

  describe('language detection', () => {
    it('uses javascript as default language', () => {
      const wrapper = createWrapper()

      expect(wrapper.find('.language-badge').text()).toBe('javascript')
    })

    it('uses language from override', () => {
      const wrapper = createWrapper({
        override: { language: 'python' },
      })

      expect(wrapper.find('.language-badge').text()).toBe('python')
    })

    it('uses language from property x-ui-language', () => {
      const wrapper = createWrapper({
        property: { type: 'string', 'x-ui-language': 'typescript' },
      })

      expect(wrapper.find('.language-badge').text()).toBe('typescript')
    })

    it('prefers override language over property', () => {
      const wrapper = createWrapper({
        property: { type: 'string', 'x-ui-language': 'python' },
        override: { language: 'go' },
      })

      expect(wrapper.find('.language-badge').text()).toBe('go')
    })
  })

  describe('code input', () => {
    it('displays modelValue in textarea', () => {
      const code = 'const x = 1;'
      const wrapper = createWrapper({ modelValue: code })

      const textarea = wrapper.find('.code-input')
      expect((textarea.element as HTMLTextAreaElement).value).toBe(code)
    })

    it('emits update:modelValue on input', async () => {
      const wrapper = createWrapper()
      const textarea = wrapper.find('.code-input')

      await textarea.setValue('const y = 2;')

      expect(wrapper.emitted('update:modelValue')).toBeTruthy()
      expect(wrapper.emitted('update:modelValue')![0]).toEqual(['const y = 2;'])
    })

    it('emits blur event on blur', async () => {
      const wrapper = createWrapper()
      const textarea = wrapper.find('.code-input')

      await textarea.trigger('blur')

      expect(wrapper.emitted('blur')).toBeTruthy()
    })

    it('disables textarea when disabled prop is true', () => {
      const wrapper = createWrapper({ disabled: true })

      const textarea = wrapper.find('.code-input')
      expect((textarea.element as HTMLTextAreaElement).disabled).toBe(true)
    })

    it('uses default rows of 10', () => {
      const wrapper = createWrapper()

      const textarea = wrapper.find('.code-input')
      // Use getAttribute for consistent string comparison in jsdom
      expect(textarea.element.getAttribute('rows')).toBe('10')
    })

    it('uses rows from override', () => {
      const wrapper = createWrapper({
        override: { rows: 20 },
      })

      const textarea = wrapper.find('.code-input')
      // Use getAttribute for consistent string comparison in jsdom
      expect(textarea.element.getAttribute('rows')).toBe('20')
    })
  })

  describe('line numbers', () => {
    it('shows correct number of line numbers', () => {
      const code = 'line1\nline2\nline3'
      const wrapper = createWrapper({ modelValue: code })

      const lineNumbers = wrapper.findAll('.line-numbers span')
      expect(lineNumbers).toHaveLength(3)
      expect(lineNumbers[0].text()).toBe('1')
      expect(lineNumbers[1].text()).toBe('2')
      expect(lineNumbers[2].text()).toBe('3')
    })

    it('shows 1 line number for empty code', () => {
      const wrapper = createWrapper({ modelValue: '' })

      const lineNumbers = wrapper.findAll('.line-numbers span')
      expect(lineNumbers).toHaveLength(1)
    })
  })

  describe('syntax highlighting', () => {
    it('highlights JavaScript keywords', () => {
      const code = 'const x = 1;'
      const wrapper = createWrapper({ modelValue: code })

      const highlightedHtml = wrapper.find('.code-highlight').html()
      expect(highlightedHtml).toContain('<span class="keyword">const</span>')
    })

    it('highlights strings', () => {
      const code = 'const msg = "hello";'
      const wrapper = createWrapper({ modelValue: code })

      const highlightedHtml = wrapper.find('.code-highlight').html()
      expect(highlightedHtml).toContain('<span class="string">"hello"</span>')
    })

    it('highlights comments', () => {
      const code = '// this is a comment'
      const wrapper = createWrapper({ modelValue: code })

      const highlightedHtml = wrapper.find('.code-highlight').html()
      expect(highlightedHtml).toContain('<span class="comment">// this is a comment</span>')
    })

    it('highlights numbers', () => {
      const code = 'const x = 42;'
      const wrapper = createWrapper({ modelValue: code })

      const highlightedHtml = wrapper.find('.code-highlight').html()
      expect(highlightedHtml).toContain('<span class="number">42</span>')
    })

    it('escapes HTML in code', () => {
      const code = 'const x = "<script>";'
      const wrapper = createWrapper({ modelValue: code })

      const highlightedHtml = wrapper.find('.code-highlight').html()
      expect(highlightedHtml).toContain('&lt;script&gt;')
      expect(highlightedHtml).not.toContain('<script>')
    })

    it('does not highlight for non-JavaScript language', () => {
      const code = 'const x = 1;'
      const wrapper = createWrapper({
        modelValue: code,
        property: { type: 'string', 'x-ui-language': 'python' },
      })

      const highlightedHtml = wrapper.find('.code-highlight').html()
      // Still escapes HTML but no keyword highlighting
      expect(highlightedHtml).toContain('const x = ')
    })
  })
})
