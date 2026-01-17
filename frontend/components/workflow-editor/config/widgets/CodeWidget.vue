<script setup lang="ts">
import { computed } from 'vue'
import type { JSONSchemaProperty, FieldOverride } from '../types/config-schema'

const props = defineProps<{
  name: string
  property: JSONSchemaProperty
  modelValue: string | undefined
  override?: FieldOverride
  error?: string
  disabled?: boolean
  required?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'blur'): void
}>()

// Determine language from property or override
const language = computed(() => {
  // Check override first
  if (props.override?.language) return props.override.language
  // Check property x-ui-language
  if (props.property['x-ui-language']) return String(props.property['x-ui-language'])
  // Default to JavaScript
  return 'javascript'
})

// Get display title
const title = computed(() => {
  return props.property.title || props.name
})

// Get description
const description = computed(() => {
  return props.property.description
})

// Is required? (from prop or x-required)
const isRequired = computed(() => {
  return props.required || props.property['x-required'] === true
})

// Number of rows
const rows = computed(() => {
  return props.override?.rows || 10
})

// Handle input
function handleInput(event: Event) {
  const target = event.target as HTMLTextAreaElement
  emit('update:modelValue', target.value)
}

// Handle blur
function handleBlur() {
  emit('blur')
}

// Format code - simple JavaScript formatter
function formatCode() {
  const code = props.modelValue || ''
  if (!code.trim()) return

  try {
    const formatted = formatJavaScript(code)
    emit('update:modelValue', formatted)
  } catch (e) {
    // If formatting fails, keep original code
    console.warn('Code formatting failed:', e)
  }
}

// Simple JavaScript formatter
function formatJavaScript(code: string): string {
  let result = ''
  let indentLevel = 0
  const indentString = '  ' // 2 spaces
  let inMultilineComment = false

  const lines = code.split('\n')

  for (let lineIndex = 0; lineIndex < lines.length; lineIndex++) {
    const line = lines[lineIndex].trim()

    // Skip empty lines but preserve one
    if (!line) {
      if (result && !result.endsWith('\n\n')) {
        result += '\n'
      }
      continue
    }

    // Check for multiline comment start/end
    if (line.startsWith('/*')) {
      inMultilineComment = true
    }
    if (line.endsWith('*/')) {
      result += indentString.repeat(indentLevel) + line + '\n'
      inMultilineComment = false
      continue
    }
    if (inMultilineComment) {
      result += indentString.repeat(indentLevel) + line + '\n'
      continue
    }

    // Check for single line comment
    if (line.startsWith('//')) {
      result += indentString.repeat(indentLevel) + line + '\n'
      continue
    }

    // Decrease indent for closing braces/brackets at start of line
    if (line.startsWith('}') || line.startsWith(']') || line.startsWith(')')) {
      indentLevel = Math.max(0, indentLevel - 1)
    }

    // Add formatted line
    result += indentString.repeat(indentLevel) + line + '\n'

    // Count braces to adjust indent (ignoring those in strings)
    let tempInString: string | null = null
    for (let i = 0; i < line.length; i++) {
      const char = line[i]
      const prevChar = i > 0 ? line[i - 1] : ''

      // Handle string detection
      if ((char === '"' || char === "'" || char === '`') && prevChar !== '\\') {
        if (tempInString === char) {
          tempInString = null
        } else if (tempInString === null) {
          tempInString = char
        }
        continue
      }

      // Only count braces outside strings
      if (tempInString === null) {
        if (char === '{' || char === '[' || char === '(') {
          // Don't increase if closing brace is on same line
          const remaining = line.slice(i + 1)
          const closingChar = char === '{' ? '}' : char === '[' ? ']' : ')'
          if (!remaining.includes(closingChar)) {
            indentLevel++
          }
        }
      }
    }
  }

  return result.trim() + '\n'
}

// Simple syntax highlighting keywords for JavaScript
// Uses placeholders to avoid replacement conflicts
const highlightedCode = computed(() => {
  const code = props.modelValue || ''
  if (!code) return ''

  // Escape HTML first
  let html = code
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')

  if (language.value === 'javascript' || language.value === 'js') {
    // Use placeholder tokens to avoid nested replacements
    const tokens: { placeholder: string; replacement: string }[] = []
    let tokenIndex = 0

    const createToken = (className: string, content: string): string => {
      const placeholder = `\x00TOKEN${tokenIndex++}\x00`
      tokens.push({
        placeholder,
        replacement: `<span class="${className}">${content}</span>`,
      })
      return placeholder
    }

    // Order matters: strings and comments first (they can contain keywords)
    // Comments
    html = html.replace(/(\/\/.*$)/gm, (match) => createToken('comment', match))

    // Strings (single, double quotes, and template literals)
    html = html.replace(
      /('(?:[^'\\]|\\.)*'|"(?:[^"\\]|\\.)*"|`(?:[^`\\]|\\.)*`)/g,
      (match) => createToken('string', match)
    )

    // Keywords (only match outside of already tokenized areas)
    html = html.replace(
      /\b(const|let|var|function|async|await|return|if|else|for|while|try|catch|throw|new|class|extends|import|export|default|from)\b/g,
      (match) => createToken('keyword', match)
    )

    // Numbers
    html = html.replace(/\b(\d+(?:\.\d+)?)\b/g, (match) => createToken('number', match))

    // Replace all placeholders with actual HTML
    for (const token of tokens) {
      html = html.replace(token.placeholder, token.replacement)
    }
  }

  return html
})
</script>

<template>
  <div class="code-widget">
    <label :for="name" class="code-widget-label">
      {{ title }}
      <span v-if="isRequired" class="required-indicator">*</span>
    </label>

    <p v-if="description" class="code-widget-description">
      {{ description }}
    </p>

    <div class="code-widget-container">
      <div class="code-editor">
        <div class="line-numbers">
          <span v-for="i in (modelValue || '').split('\n').length" :key="i">{{ i }}</span>
        </div>
        <div class="code-area">
          <!-- Highlighted code display (read-only visual layer) -->
          <pre class="code-highlight" v-html="highlightedCode"/>
          <!-- Actual textarea for input -->
          <textarea
            :id="name"
            :value="modelValue || ''"
            :rows="rows"
            :disabled="disabled"
            :class="{ 'has-error': !!error }"
            class="code-input"
            spellcheck="false"
            @input="handleInput"
            @blur="handleBlur"
          />
        </div>
      </div>
      <div class="code-footer">
        <button
          type="button"
          class="format-button"
          :disabled="disabled || !modelValue"
          @click="formatCode"
        >
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 10H3M21 6H3M21 14H3M21 18H3"/>
          </svg>
          Format
        </button>
        <span class="language-badge">{{ language }}</span>
      </div>
    </div>

    <p v-if="error" class="code-widget-error">
      {{ error }}
    </p>
  </div>
</template>

<style scoped>
.code-widget {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.code-widget-label {
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--color-text);
}

.required-indicator {
  color: var(--color-error);
  margin-left: 0.125rem;
}

.code-widget-description {
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
  margin: 0;
  line-height: 1.4;
}

.code-widget-container {
  border: 1px solid var(--color-border);
  border-radius: 6px;
  overflow: hidden;
  background: #1e1e1e;
}

.code-editor {
  display: flex;
  min-height: 150px;
  max-height: 400px;
  overflow: auto;
}

.line-numbers {
  display: flex;
  flex-direction: column;
  padding: 0.75rem 0.5rem;
  background: #2d2d2d;
  color: #6e7681;
  font-family: 'SF Mono', Monaco, 'Cascadia Code', monospace;
  font-size: 0.75rem;
  line-height: 1.5;
  text-align: right;
  user-select: none;
  min-width: 2.5rem;
  border-right: 1px solid #404040;
}

.code-area {
  position: relative;
  flex: 1;
  overflow: auto;
}

.code-highlight,
.code-input {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  margin: 0;
  padding: 0.75rem;
  font-family: 'SF Mono', Monaco, 'Cascadia Code', monospace;
  font-size: 0.75rem;
  line-height: 1.5;
  white-space: pre-wrap;
  word-wrap: break-word;
  overflow: hidden;
}

.code-highlight {
  color: #d4d4d4;
  pointer-events: none;
  z-index: 1;
}

.code-input {
  background: transparent;
  color: transparent;
  caret-color: #d4d4d4;
  border: none;
  outline: none;
  resize: none;
  z-index: 2;
}

.code-input:focus {
  outline: none;
}

.code-input.has-error {
  border-color: var(--color-error);
}

.code-input:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.code-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.375rem 0.75rem;
  background: #2d2d2d;
  border-top: 1px solid #404040;
}

.format-button {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  padding: 0.25rem 0.5rem;
  font-size: 0.625rem;
  font-weight: 500;
  color: #9ca3af;
  background: #404040;
  border: none;
  border-radius: 3px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.format-button:hover:not(:disabled) {
  background: #4a4a4a;
  color: #d4d4d4;
}

.format-button:active:not(:disabled) {
  background: #3a3a3a;
}

.format-button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.language-badge {
  font-size: 0.625rem;
  font-weight: 500;
  text-transform: uppercase;
  color: #9ca3af;
  background: #404040;
  padding: 0.125rem 0.375rem;
  border-radius: 3px;
}

.code-widget-error {
  font-size: 0.6875rem;
  color: var(--color-error);
  margin: 0;
}

/* Syntax highlighting */
:deep(.keyword) {
  color: #569cd6;
}

:deep(.string) {
  color: #ce9178;
}

:deep(.comment) {
  color: #6a9955;
}

:deep(.number) {
  color: #b5cea8;
}
</style>
