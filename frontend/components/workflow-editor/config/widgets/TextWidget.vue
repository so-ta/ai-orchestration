<script setup lang="ts">
import { ref, computed, toRef } from 'vue'
import type { JSONSchemaProperty, FieldOverride } from '../types/config-schema'
import { useVariableInsertion } from '../variable-picker/useVariableInsertion'
import VariablePicker from '../variable-picker/VariablePicker.vue'

const { t } = useI18n()

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

const inputRef = ref<HTMLInputElement | null>(null)
const modelValueRef = toRef(props, 'modelValue')

// Track whether field has been touched for validation
const touched = ref(false)
const isEmpty = computed(() => !props.modelValue || props.modelValue.trim() === '')
const showRequiredWarning = computed(() => props.required && touched.value && isEmpty.value && !props.error)

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
  modelValue: modelValueRef,
  emit: (value) => emit('update:modelValue', value),
  inputRef,
  fieldId: props.name
})

const inputType = computed(() => {
  if (props.property.format === 'uri') return 'url'
  if (props.property.format === 'email') return 'email'
  return 'text'
})

const placeholder = computed(() => {
  return props.override?.placeholder || props.property.description || ''
})

function handleInput(event: Event) {
  const target = event.target as HTMLInputElement
  emit('update:modelValue', target.value)
  handleVariableInput(event)
}

function handleKeydown(event: KeyboardEvent) {
  handleVariableKeydown(event)
}

function handleBlur() {
  touched.value = true
  emit('blur')
}
</script>

<template>
  <div class="text-widget">
    <label :for="name" class="field-label">
      {{ property.title || name }}
      <span v-if="required" class="field-required">*</span>
      <span v-if="props.property.format" class="field-format">({{ props.property.format }})</span>
    </label>

    <input
      :id="name"
      ref="inputRef"
      :type="inputType"
      :value="modelValue ?? property.default ?? ''"
      :placeholder="placeholder"
      :maxlength="property.maxLength"
      :disabled="disabled"
      autocomplete="off"
      :class="['field-input', { 'has-error': error || showRequiredWarning, 'drag-over': isDragOver }]"
      @input="handleInput"
      @keydown="handleKeydown"
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

    <p v-if="showRequiredWarning" class="field-warning">
      {{ t('fieldValidation.required') }}
    </p>

    <VariablePicker
      v-if="availableVariables.length > 0"
      v-model="pickerVisible"
      :variables="availableVariables"
      :position="pickerPosition"
      @select="insertVariable"
    />
  </div>
</template>

<style scoped>
.text-widget {
  display: flex;
  flex-direction: column;
  gap: 4px;
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

.field-format {
  font-weight: 400;
  color: var(--color-text-muted, #9ca3af);
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
