<script setup lang="ts">
import { computed, ref, watch, nextTick, onMounted } from 'vue'
import type { JSONSchemaProperty, FieldOverride } from '../types/config-schema'

// DOMPurify is loaded dynamically for SSR compatibility
const purify = ref<typeof import('dompurify')['default'] | null>(null)

onMounted(async () => {
  const DOMPurify = (await import('dompurify')).default
  purify.value = DOMPurify
})

const props = defineProps<{
  name: string
  property: JSONSchemaProperty
  modelValue: string | undefined
  override?: FieldOverride
  error?: string
  disabled?: boolean
  required?: boolean
}>()

// Refs for auto-height calculation
const textareaRef = ref<HTMLTextAreaElement | null>(null)
const calculatedHeight = ref<number>(150)

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

// Calculate editor height based on content
function calculateHeight() {
  const textarea = textareaRef.value
  if (!textarea) return

  // Reset height to auto to get scrollHeight
  textarea.style.height = 'auto'
  const scrollHeight = textarea.scrollHeight
  const minHeight = 150
  const maxHeight = 500

  const newHeight = Math.max(minHeight, Math.min(scrollHeight, maxHeight))
  calculatedHeight.value = newHeight
}

// Watch for content changes and recalculate height
watch(
  () => props.modelValue,
  () => {
    nextTick(() => calculateHeight())
  },
  { immediate: true }
)

// Handle input
function handleInput(event: Event) {
  const target = event.target as HTMLTextAreaElement
  emit('update:modelValue', target.value)
  nextTick(() => calculateHeight())
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

// Advanced JavaScript formatter - statement-aware
function formatJavaScript(code: string): string {
  const indentString = '  ' // 2 spaces

  // Tokenize the code to identify strings, comments, and code
  interface Token {
    type: 'code' | 'string' | 'comment' | 'multiline-comment' | 'template'
    value: string
  }

  function tokenize(src: string): Token[] {
    const tokens: Token[] = []
    let i = 0

    while (i < src.length) {
      // Multi-line comment
      if (src[i] === '/' && src[i + 1] === '*') {
        let end = src.indexOf('*/', i + 2)
        if (end === -1) end = src.length
        else end += 2
        tokens.push({ type: 'multiline-comment', value: src.slice(i, end) })
        i = end
        continue
      }

      // Single-line comment
      if (src[i] === '/' && src[i + 1] === '/') {
        let end = src.indexOf('\n', i)
        if (end === -1) end = src.length
        tokens.push({ type: 'comment', value: src.slice(i, end) })
        i = end
        continue
      }

      // Template literal
      if (src[i] === '`') {
        let j = i + 1
        while (j < src.length) {
          if (src[j] === '\\') {
            j += 2
            continue
          }
          if (src[j] === '`') {
            j++
            break
          }
          j++
        }
        tokens.push({ type: 'template', value: src.slice(i, j) })
        i = j
        continue
      }

      // String (single or double quote)
      if (src[i] === '"' || src[i] === "'") {
        const quote = src[i]
        let j = i + 1
        while (j < src.length) {
          if (src[j] === '\\') {
            j += 2
            continue
          }
          if (src[j] === quote) {
            j++
            break
          }
          j++
        }
        tokens.push({ type: 'string', value: src.slice(i, j) })
        i = j
        continue
      }

      // Regular code - collect until next special token
      let j = i
      while (j < src.length) {
        if (
          src[j] === '"' ||
          src[j] === "'" ||
          src[j] === '`' ||
          (src[j] === '/' && (src[j + 1] === '/' || src[j + 1] === '*'))
        ) {
          break
        }
        j++
      }
      if (j > i) {
        tokens.push({ type: 'code', value: src.slice(i, j) })
        i = j
      } else {
        // Fallback: single character
        tokens.push({ type: 'code', value: src[i] })
        i++
      }
    }

    return tokens
  }

  // Format code tokens - split statements properly
  function formatCodePart(codeStr: string, currentIndent: number): { formatted: string; indent: number } {
    let result = ''
    let indent = currentIndent
    let buffer = ''

    const chars = codeStr.split('')
    let parenDepth = 0 // ()
    let bracketDepth = 0 // []

    for (let i = 0; i < chars.length; i++) {
      const ch = chars[i]

      // Track parentheses and brackets (for multi-line expressions)
      if (ch === '(') parenDepth++
      if (ch === ')') parenDepth = Math.max(0, parenDepth - 1)
      if (ch === '[') bracketDepth++
      if (ch === ']') bracketDepth = Math.max(0, bracketDepth - 1)

      // Handle opening braces
      if (ch === '{') {
        buffer += ch
        result += buffer.trim()
        result += '\n'
        indent++
        buffer = ''
        continue
      }

      // Handle closing braces
      if (ch === '}') {
        if (buffer.trim()) {
          result += indentString.repeat(indent) + buffer.trim() + '\n'
        }
        indent = Math.max(0, indent - 1)
        result += indentString.repeat(indent) + ch
        buffer = ''

        // Check if next non-whitespace is else, catch, finally, while (do-while)
        let lookahead = ''
        for (let j = i + 1; j < chars.length; j++) {
          if (chars[j] !== ' ' && chars[j] !== '\n' && chars[j] !== '\t') {
            lookahead = codeStr.slice(j, j + 10)
            break
          }
        }

        if (
          lookahead.startsWith('else') ||
          lookahead.startsWith('catch') ||
          lookahead.startsWith('finally') ||
          lookahead.startsWith('while')
        ) {
          result += ' '
        } else {
          result += '\n'
        }
        continue
      }

      // Handle semicolons (statement end) - only at top level (not inside parens/brackets)
      if (ch === ';' && parenDepth === 0 && bracketDepth === 0) {
        buffer += ch
        result += indentString.repeat(indent) + buffer.trim() + '\n'
        buffer = ''
        continue
      }

      // Handle newlines in original code
      if (ch === '\n') {
        if (buffer.trim()) {
          // If buffer doesn't end with { or ; at top level, it's a continuation
          const trimmed = buffer.trim()
          if (
            parenDepth > 0 ||
            bracketDepth > 0 ||
            trimmed.endsWith(',') ||
            trimmed.endsWith('&&') ||
            trimmed.endsWith('||') ||
            trimmed.endsWith('+') ||
            trimmed.endsWith('?') ||
            trimmed.endsWith(':')
          ) {
            // Continuation line
            buffer += ' '
          } else {
            result += indentString.repeat(indent) + trimmed + '\n'
            buffer = ''
          }
        }
        continue
      }

      // Collapse multiple spaces
      if (ch === ' ' || ch === '\t') {
        if (buffer.length > 0 && !buffer.endsWith(' ')) {
          buffer += ' '
        }
        continue
      }

      buffer += ch
    }

    // Flush remaining buffer
    if (buffer.trim()) {
      result += indentString.repeat(indent) + buffer.trim() + '\n'
    }

    return { formatted: result, indent }
  }

  // Process all tokens
  const tokens = tokenize(code)
  let result = ''
  let currentIndent = 0

  for (const token of tokens) {
    if (token.type === 'code') {
      const { formatted, indent } = formatCodePart(token.value, currentIndent)
      result += formatted
      currentIndent = indent
    } else if (token.type === 'comment') {
      // Single-line comment: add on its own line
      if (result.endsWith('\n') || result === '') {
        result += indentString.repeat(currentIndent) + token.value.trim() + '\n'
      } else {
        result += ' ' + token.value.trim() + '\n'
      }
    } else if (token.type === 'multiline-comment') {
      // Preserve multiline comment formatting
      const lines = token.value.split('\n')
      for (let i = 0; i < lines.length; i++) {
        if (i === 0) {
          result += indentString.repeat(currentIndent) + lines[i].trim()
        } else {
          result += '\n' + indentString.repeat(currentIndent) + ' ' + lines[i].trim()
        }
      }
      result += '\n'
    } else {
      // Strings and template literals: preserve as-is, append to last line
      if (result.endsWith('\n')) {
        result = result.slice(0, -1) + token.value + '\n'
      } else {
        result += token.value
      }
    }
  }

  // Clean up: remove multiple blank lines, ensure single trailing newline
  result = result.replace(/\n{3,}/g, '\n\n').trim() + '\n'

  return result
}

// Syntax highlighting for JavaScript
// SECURITY: HTML is escaped FIRST to prevent XSS, then syntax highlighting spans are added.
// The v-html directive is safe here because user input is always escaped before processing.
const highlightedCode = computed(() => {
  const code = props.modelValue || ''
  if (!code) return ''

  // IMPORTANT: Escape HTML first to prevent XSS attacks
  // All user input (<, >, &, etc.) is converted to HTML entities
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

  // Additional sanitization with DOMPurify for defense in depth
  // Only allow span tags with class attribute for syntax highlighting
  if (purify.value) {
    return purify.value.sanitize(html, {
      ALLOWED_TAGS: ['span'],
      ALLOWED_ATTR: ['class'],
    })
  }
  // Fallback: return HTML-escaped content (safe but without syntax highlighting spans)
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
      <div class="code-editor" :style="{ height: `${calculatedHeight}px` }">
        <div class="line-numbers">
          <span v-for="i in (modelValue || '').split('\n').length" :key="i">{{ i }}</span>
        </div>
        <div class="code-area">
          <!-- Highlighted code display (read-only visual layer) -->
          <pre class="code-highlight" v-html="highlightedCode"/>
          <!-- Actual textarea for input -->
          <textarea
            :id="name"
            ref="textareaRef"
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
  max-height: 500px;
  overflow: auto;
  transition: height 0.15s ease;
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
  min-width: 0;
  overflow: hidden;
}

.code-highlight,
.code-input {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  min-height: 100%;
  margin: 0;
  padding: 0.75rem;
  font-family: 'SF Mono', Monaco, 'Cascadia Code', monospace;
  font-size: 0.75rem;
  line-height: 1.5;
  white-space: pre-wrap;
  word-wrap: break-word;
  overflow-wrap: break-word;
  overflow: hidden;
  box-sizing: border-box;
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
