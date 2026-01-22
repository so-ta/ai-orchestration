<script setup lang="ts">
import { ref, computed, toRef, type Ref } from 'vue'
import type { JSONSchemaProperty, FieldOverride } from '../types/config-schema'
import { useExpressionMode } from './useExpressionMode'
import { useVariableInsertion } from '../variable-picker/useVariableInsertion'
import VariablePicker from '../variable-picker/VariablePicker.vue'

const props = defineProps<{
  name: string
  property: JSONSchemaProperty
  modelValue: boolean | string | undefined
  override?: FieldOverride
  error?: string
  disabled?: boolean
  required?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean | string): void
  (e: 'blur'): void
}>()

const inputRef = ref<HTMLInputElement | null>(null)

// 式モードのセットアップ
const modelValueRef = toRef(props, 'modelValue')
const {
  isExpressionMode,
  expressionText,
  toggleMode,
  updateExpression
} = useExpressionMode({
  modelValue: modelValueRef as Ref<boolean | string | undefined>,
  emit: (value) => emit('update:modelValue', value as boolean | string),
  parseValue: (text) => {
    const lower = text.toLowerCase().trim()
    if (lower === 'true') return true
    if (lower === 'false') return false
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

const isChecked = computed(() => {
  if (typeof props.modelValue === 'boolean') return props.modelValue
  if (props.property.default !== undefined) return props.property.default as boolean
  return false
})

function handleChange(event: Event) {
  const target = event.target as HTMLInputElement
  emit('update:modelValue', target.checked)
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
  <div class="checkbox-widget">
    <div class="field-header">
      <label v-if="!isExpressionMode" :class="['checkbox-label', { disabled }]">
        <input
          :id="name"
          type="checkbox"
          :checked="isChecked"
          :disabled="disabled"
          class="checkbox-input"
          @change="handleChange"
          @blur="handleBlur"
        >
        <span class="checkbox-box">
          <svg
            v-if="isChecked"
            class="checkbox-icon"
            viewBox="0 0 12 12"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              d="M2.5 6L5 8.5L9.5 4"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>
        </span>
        <span class="checkbox-text">{{ property.title || name }}</span>
        <span v-if="required" class="field-required">*</span>
      </label>

      <label v-else :for="name" class="field-label">
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

    <!-- 式モード -->
    <input
      v-if="isExpressionMode"
      :id="name"
      ref="inputRef"
      type="text"
      :value="expressionText"
      :disabled="disabled"
      :class="['field-input', { 'has-error': error, 'drag-over': isDragOver }]"
      placeholder="{{$.steps.prev.output}} または true/false"
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
.checkbox-widget {
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

.checkbox-label {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  user-select: none;
  flex: 1;
}

.checkbox-label.disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

.checkbox-input {
  position: absolute;
  opacity: 0;
  pointer-events: none;
}

.checkbox-box {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border: 2px solid var(--color-border, #d1d5db);
  border-radius: 4px;
  background: var(--color-bg-input, #fff);
  transition: all 0.15s;
}

.checkbox-input:checked + .checkbox-box {
  background: var(--color-primary, #3b82f6);
  border-color: var(--color-primary, #3b82f6);
}

.checkbox-input:focus + .checkbox-box {
  box-shadow: 0 0 0 3px var(--color-primary-alpha, rgba(59, 130, 246, 0.2));
}

.checkbox-input:disabled + .checkbox-box {
  background: var(--color-bg-disabled, #f3f4f6);
  border-color: var(--color-border-disabled, #e5e7eb);
}

.checkbox-icon {
  width: 12px;
  height: 12px;
  color: white;
}

.checkbox-text {
  font-size: 14px;
  color: var(--color-text, #111827);
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
  flex-shrink: 0;
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

.field-description {
  font-size: 11px;
  color: var(--color-text-muted, #9ca3af);
  margin: 0;
  padding-left: 26px;
}

.checkbox-widget:has(.field-input) .field-description {
  padding-left: 0;
}

.field-error {
  font-size: 11px;
  color: var(--color-error, #ef4444);
  margin: 0;
  padding-left: 26px;
}

.checkbox-widget:has(.field-input) .field-error {
  padding-left: 0;
}
</style>
