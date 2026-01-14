<script setup lang="ts">
import type { JSONSchemaProperty, FieldOverride } from '../types/config-schema';

const props = defineProps<{
  name: string;
  property: JSONSchemaProperty;
  modelValue: string | undefined;
  override?: FieldOverride;
  error?: string;
  disabled?: boolean;
  required?: boolean;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void;
  (e: 'blur'): void;
}>();

const rows = computed(() => props.override?.rows || 4);

const placeholder = computed(() => {
  return props.override?.placeholder || props.property.description || '';
});

function handleInput(event: Event) {
  const target = event.target as HTMLTextAreaElement;
  emit('update:modelValue', target.value);
}

function handleBlur() {
  emit('blur');
}
</script>

<template>
  <div class="textarea-widget">
    <label :for="name" class="field-label">
      {{ property.title || name }}
      <span v-if="required" class="field-required">*</span>
    </label>

    <textarea
      :id="name"
      :value="modelValue ?? (property.default as string) ?? ''"
      :placeholder="placeholder"
      :rows="rows"
      :maxlength="property.maxLength"
      :disabled="disabled"
      :class="['field-textarea', { 'has-error': error }]"
      @input="handleInput"
      @blur="handleBlur"
    />

    <div class="field-footer">
      <p v-if="property.description && !error" class="field-description">
        {{ property.description }}
      </p>
      <p v-if="error" class="field-error">
        {{ error }}
      </p>
      <span v-if="property.maxLength" class="char-count">
        {{ (modelValue || '').length }} / {{ property.maxLength }}
      </span>
    </div>
  </div>
</template>

<style scoped>
.textarea-widget {
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

.field-textarea {
  padding: 8px 12px;
  border: 1px solid var(--color-border, #e5e7eb);
  border-radius: 6px;
  font-size: 14px;
  font-family: inherit;
  background: var(--color-bg-input, #fff);
  color: var(--color-text, #111827);
  resize: vertical;
  min-height: 80px;
  transition: border-color 0.15s, box-shadow 0.15s;
}

.field-textarea:focus {
  outline: none;
  border-color: var(--color-primary, #3b82f6);
  box-shadow: 0 0 0 3px var(--color-primary-alpha, rgba(59, 130, 246, 0.1));
}

.field-textarea:disabled {
  background: var(--color-bg-disabled, #f3f4f6);
  cursor: not-allowed;
}

.field-textarea.has-error {
  border-color: var(--color-error, #ef4444);
}

.field-footer {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 8px;
}

.field-description {
  font-size: 11px;
  color: var(--color-text-muted, #9ca3af);
  margin: 0;
  flex: 1;
}

.field-error {
  font-size: 11px;
  color: var(--color-error, #ef4444);
  margin: 0;
  flex: 1;
}

.char-count {
  font-size: 11px;
  color: var(--color-text-muted, #9ca3af);
  flex-shrink: 0;
}
</style>
