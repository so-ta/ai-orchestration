<script setup lang="ts">
import type { JSONSchemaProperty, FieldOverride } from '../types/config-schema';

const props = defineProps<{
  name: string;
  property: JSONSchemaProperty;
  modelValue: string | number | undefined;
  override?: FieldOverride;
  error?: string;
  disabled?: boolean;
  required?: boolean;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: string | number): void;
  (e: 'blur'): void;
}>();

const options = computed(() => {
  return props.property.enum || [];
});

const displayValue = computed(() => {
  if (props.modelValue !== undefined) return props.modelValue;
  if (props.property.default !== undefined) return props.property.default as string | number;
  return '';
});

function handleChange(event: Event) {
  const target = event.target as HTMLSelectElement;
  emit('update:modelValue', target.value);
}

function handleBlur() {
  emit('blur');
}

function formatOptionLabel(option: string | number): string {
  if (typeof option === 'string') {
    // Capitalize first letter and replace underscores/hyphens with spaces
    return option
      .replace(/[-_]/g, ' ')
      .replace(/\b\w/g, (char) => char.toUpperCase());
  }
  return String(option);
}
</script>

<template>
  <div class="select-widget">
    <label :for="name" class="field-label">
      {{ property.title || name }}
      <span v-if="required" class="field-required">*</span>
    </label>

    <select
      :id="name"
      :value="displayValue"
      :disabled="disabled"
      :class="['field-select', { 'has-error': error }]"
      @change="handleChange"
      @blur="handleBlur"
    >
      <option v-if="!property.default && !modelValue" value="" disabled>
        選択してください
      </option>
      <option
        v-for="option in options"
        :key="option"
        :value="option"
      >
        {{ formatOptionLabel(option) }}
      </option>
    </select>

    <p v-if="property.description && !error" class="field-description">
      {{ property.description }}
    </p>

    <p v-if="error" class="field-error">
      {{ error }}
    </p>
  </div>
</template>

<style scoped>
.select-widget {
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
