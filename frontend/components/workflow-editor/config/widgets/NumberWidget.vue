<script setup lang="ts">
import { ref, computed, toRef, type Ref } from 'vue'
import type { JSONSchemaProperty, FieldOverride } from '../types/config-schema'
import { useExpressionMode } from './useExpressionMode'
import { useVariableInsertion } from '../variable-picker/useVariableInsertion'
import VariablePicker from '../variable-picker/VariablePicker.vue'

const props = defineProps<{
  name: string
  property: JSONSchemaProperty
  modelValue: number | string | undefined
  override?: FieldOverride
  error?: string
  disabled?: boolean
  required?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: number | string | undefined): void
  (e: 'blur'): void
}>()

const inputRef = ref<HTMLInputElement | null>(null)
const numberInputRef = ref<HTMLInputElement | null>(null)

// 式モードのセットアップ
const modelValueRef = toRef(props, 'modelValue')
const {
  isExpressionMode,
  expressionText,
  toggleMode,
  updateExpression
} = useExpressionMode({
  modelValue: modelValueRef as Ref<number | string | undefined>,
  emit: (value) => emit('update:modelValue', value as number | string | undefined),
  parseValue: (text) => {
    if (text.trim() === '') return null
    const parsed = props.property.type === 'integer'
      ? parseInt(text, 10)
      : parseFloat(text)
    if (isNaN(parsed)) return null
    return parsed
  },
  formatValue: (value) => value !== undefined ? String(value) : ''
})

// 変数挿入のセットアップ（式モード時のみ使用）
const expressionTextRef = computed({
  get: () => expressionText.value,
  set: (v) => { expressionText.value = v }
})

const {
  pickerVisible,
  pickerPosition,
  isDragOver,
  availableVariables,
  handleInput: handleVariableInput,
  handleKeydown: handleVariableKeydown,
  insertVariable,
  handleDragEnter,
  handleDragOver,
  handleDragLeave,
  handleDrop
} = useVariableInsertion({
  modelValue: expressionTextRef,
  emit: (value) => updateExpression(value),
  inputRef,
  fieldId: `${props.name}-expression`
})

const step = computed(() => {
  if (props.override?.step) return props.override.step
  if (props.property.type === 'integer') return 1
  return 0.1
})

const placeholder = computed(() => {
  return props.override?.placeholder || props.property.description || ''
})

const displayValue = computed(() => {
  if (typeof props.modelValue === 'number') return props.modelValue
  if (props.property.default !== undefined) return props.property.default as number
  return ''
})

function handleInput(event: Event) {
  const target = event.target as HTMLInputElement
  const value = target.value

  if (value === '') {
    emit('update:modelValue', undefined)
    return
  }

  const parsed = props.property.type === 'integer'
    ? parseInt(value, 10)
    : parseFloat(value)

  if (!isNaN(parsed)) {
    emit('update:modelValue', parsed)
  }
}

function handleExpressionInput(event: Event) {
  const target = event.target as HTMLInputElement
  updateExpression(target.value)
  handleVariableInput(event)
}

function handleExpressionKeydown(event: KeyboardEvent) {
  handleVariableKeydown(event)
}

function handleBlur() {
  emit('blur')
}
</script>

<template>
  <div class="number-widget">
    <div class="field-header">
      <label :for="name" class="field-label">
        {{ property.title || name }}
        <span v-if="required" class="field-required">*</span>
        <span v-if="property.minimum !== undefined || property.maximum !== undefined" class="field-range">
          ({{ property.minimum ?? '' }} - {{ property.maximum ?? '' }})
        </span>
      </label>
      <button
        v-if="availableVariables.length > 0"
        type="button"
        class="expression-toggle"
        :class="{ active: isExpressionMode }"
        :disabled="disabled"
        title="式モード切り替え"
        @click="toggleMode"
      >
        <span v-if="isExpressionMode" class="toggle-icon">123</span>
        <span v-else class="toggle-icon">{ }</span>
      </button>
    </div>

    <!-- 通常モード -->
    <input
      v-if="!isExpressionMode"
      :id="name"
      ref="numberInputRef"
      type="number"
      :value="displayValue"
      :placeholder="placeholder"
      :min="property.minimum"
      :max="property.maximum"
      :step="step"
      :disabled="disabled"
      :class="['field-input', { 'has-error': error }]"
      @input="handleInput"
      @blur="handleBlur"
    >

    <!-- 式モード -->
    <input
      v-else
      :id="name"
      ref="inputRef"
      type="text"
      :value="expressionText"
      :disabled="disabled"
      :class="['field-input', { 'has-error': error, 'drag-over': isDragOver }]"
      placeholder="{{$.steps.prev.output}} または数値を入力"
      autocomplete="off"
      @input="handleExpressionInput"
      @keydown="handleExpressionKeydown"
      @blur="handleBlur"
      @dragenter="handleDragEnter"
      @dragover="handleDragOver"
      @dragleave="handleDragLeave"
      @drop="handleDrop"
    >

    <p v-if="property.description && !error" class="field-description">
      {{ property.description }}
    </p>

    <p v-if="error" class="field-error">
      {{ error }}
    </p>

    <VariablePicker
      v-if="isExpressionMode && availableVariables.length > 0"
      v-model="pickerVisible"
      :variables="availableVariables"
      :position="pickerPosition"
      @select="insertVariable"
    />
  </div>
</template>

<style scoped>
.number-widget {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.field-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.field-label {
  font-size: 12px;
  font-weight: 500;
  color: var(--color-text-secondary, #6b7280);
}

.field-required {
  color: var(--color-error, #ef4444);
  margin-left: 2px;
}

.field-range {
  font-weight: 400;
  color: var(--color-text-muted, #9ca3af);
}

.expression-toggle {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 20px;
  padding: 0;
  border: 1px solid var(--color-border, #e5e7eb);
  border-radius: 4px;
  background: var(--color-bg-input, #fff);
  cursor: pointer;
  transition: all 0.15s;
}

.expression-toggle:hover:not(:disabled) {
  border-color: var(--color-primary, #3b82f6);
  background: var(--color-primary-alpha, rgba(59, 130, 246, 0.05));
}

.expression-toggle.active {
  border-color: var(--color-primary, #3b82f6);
  background: var(--color-primary, #3b82f6);
  color: white;
}

.expression-toggle:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.toggle-icon {
  font-size: 9px;
  font-weight: 600;
  font-family: 'SF Mono', Monaco, monospace;
}

.field-input {
  padding: 8px 12px;
  border: 1px solid var(--color-border, #e5e7eb);
  border-radius: 6px;
  font-size: 14px;
  background: var(--color-bg-input, #fff);
  color: var(--color-text, #111827);
  transition: border-color 0.15s, box-shadow 0.15s;
}

.field-input:focus {
  outline: none;
  border-color: var(--color-primary, #3b82f6);
  box-shadow: 0 0 0 3px var(--color-primary-alpha, rgba(59, 130, 246, 0.1));
}

.field-input:disabled {
  background: var(--color-bg-disabled, #f3f4f6);
  cursor: not-allowed;
}

.field-input.has-error {
  border-color: var(--color-error, #ef4444);
}

.field-input.drag-over {
  border-color: var(--color-primary, #3b82f6);
  background: var(--color-primary-alpha, rgba(59, 130, 246, 0.05));
}

/* Hide spin buttons */
.field-input::-webkit-outer-spin-button,
.field-input::-webkit-inner-spin-button {
  -webkit-appearance: none;
  margin: 0;
}

.field-input[type='number'] {
  -moz-appearance: textfield;
}

.field-description {
  font-size: 11px;
  color: var(--color-text-muted, #9ca3af);
  margin: 0;
}

.field-error {
  font-size: 11px;
  color: var(--color-error, #ef4444);
  margin: 0;
}
</style>
