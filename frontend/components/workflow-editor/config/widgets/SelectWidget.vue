<script setup lang="ts">
import { ref, computed, toRef, type Ref } from 'vue'
import type { JSONSchemaProperty, FieldOverride } from '../types/config-schema'
import { useExpressionMode } from './useExpressionMode'
import { useVariableInsertion } from '../variable-picker/useVariableInsertion'
import VariablePicker from '../variable-picker/VariablePicker.vue'

const { t } = useI18n()

const props = defineProps<{
  name: string
  property: JSONSchemaProperty
  modelValue: string | number | undefined
  override?: FieldOverride
  error?: string
  disabled?: boolean
  required?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: string | number): void
  (e: 'blur'): void
}>()

const inputRef = ref<HTMLInputElement | null>(null)

// Track whether field has been touched for validation
const touched = ref(false)
const isEmpty = computed(() => props.modelValue === undefined || props.modelValue === '')
const showRequiredWarning = computed(() => props.required && touched.value && isEmpty.value && !props.error)

const options = computed(() => {
  return props.property.enum || []
})

// 式モードのセットアップ
const modelValueRef = toRef(props, 'modelValue')
const {
  isExpressionMode,
  expressionText,
  toggleMode,
  updateExpression
} = useExpressionMode<string | number>({
  modelValue: modelValueRef as Ref<string | number | undefined>,
  emit: (value) => emit('update:modelValue', value as string | number),
  parseValue: (text) => {
    // enumに含まれる値かチェック
    if (options.value.includes(text)) return text
    if (options.value.includes(Number(text))) return Number(text)
    return null
  },
  formatValue: (value) => String(value)
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

const displayValue = computed(() => {
  if (props.modelValue !== undefined) return props.modelValue
  if (props.property.default !== undefined) return props.property.default as string | number
  return ''
})

function handleChange(event: Event) {
  const target = event.target as HTMLSelectElement
  emit('update:modelValue', target.value)
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
  touched.value = true
  emit('blur')
}

function formatOptionLabel(option: string | number): string {
  if (typeof option === 'string') {
    return option
      .replace(/[-_]/g, ' ')
      .replace(/\b\w/g, (char) => char.toUpperCase())
  }
  return String(option)
}
</script>

<template>
  <div class="select-widget">
    <div class="field-header">
      <label :for="name" class="field-label">
        {{ property.title || name }}
        <span v-if="required" class="field-required">*</span>
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
        <span v-if="isExpressionMode" class="toggle-icon">ABC</span>
        <span v-else class="toggle-icon">{ }</span>
      </button>
    </div>

    <!-- 通常モード -->
    <select
      v-if="!isExpressionMode"
      :id="name"
      :value="displayValue"
      :disabled="disabled"
      :class="['field-select', { 'has-error': error || showRequiredWarning }]"
      @change="handleChange"
      @blur="handleBlur"
    >
      <option v-if="!property.default && !modelValue" value="" disabled>
        {{ t('widgets.select.placeholder') }}
      </option>
      <option
        v-for="option in options"
        :key="option"
        :value="option"
      >
        {{ formatOptionLabel(option) }}
      </option>
    </select>

    <!-- 式モード -->
    <input
      v-else
      :id="name"
      ref="inputRef"
      type="text"
      :value="expressionText"
      :disabled="disabled"
      :class="['field-input', { 'has-error': error, 'drag-over': isDragOver }]"
      placeholder="{{$.steps.prev.output}} or value"
      autocomplete="off"
      @input="handleExpressionInput"
      @keydown="handleExpressionKeydown"
      @blur="handleBlur"
      @dragenter="handleDragEnter"
      @dragover="handleDragOver"
      @dragleave="handleDragLeave"
      @drop="handleDrop"
    >

    <p v-if="property.description && !error && !showRequiredWarning" class="field-description">
      {{ property.description }}
    </p>

    <p v-if="error" class="field-error">
      {{ error }}
    </p>

    <p v-else-if="showRequiredWarning" class="field-warning">
      {{ t('fieldValidation.required') }}
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
.select-widget {
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

.field-select {
  padding: 8px 32px 8px 12px;
  border: 1px solid var(--color-border, #e5e7eb);
  border-radius: 6px;
  font-size: 14px;
  background: var(--color-bg-input, #fff);
  color: var(--color-text, #111827);
  cursor: pointer;
  appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 12 12'%3E%3Cpath fill='%236b7280' d='M3 4.5L6 7.5L9 4.5'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 12px center;
  transition: border-color 0.15s, box-shadow 0.15s;
}

.field-select:focus {
  outline: none;
  border-color: var(--color-primary, #3b82f6);
  box-shadow: 0 0 0 3px var(--color-primary-alpha, rgba(59, 130, 246, 0.1));
}

.field-select:disabled {
  background-color: var(--color-bg-disabled, #f3f4f6);
  cursor: not-allowed;
}

.field-select.has-error {
  border-color: var(--color-error, #ef4444);
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

.field-warning {
  font-size: 11px;
  color: var(--color-warning, #f59e0b);
  margin: 0;
}
</style>
