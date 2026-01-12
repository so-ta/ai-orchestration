<script setup lang="ts">
import type { JSONSchemaProperty, FieldOverride } from '../types/config-schema';

const props = defineProps<{
  name: string;
  property: JSONSchemaProperty;
  modelValue: number | undefined;
  override?: FieldOverride;
  error?: string;
  disabled?: boolean;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: number | undefined): void;
  (e: 'blur'): void;
}>();

const step = computed(() => {
  if (props.override?.step) return props.override.step;
  if (props.property.type === 'integer') return 1;
  return 0.1;
});

const placeholder = computed(() => {
  return props.override?.placeholder || props.property.description || '';
});

const displayValue = computed(() => {
  if (props.modelValue !== undefined) return props.modelValue;
  if (props.property.default !== undefined) return props.property.default as number;
  return '';
});

function handleInput(event: Event) {
  const target = event.target as HTMLInputElement;
  const value = target.value;

  if (value === '') {
    emit('update:modelValue', undefined);
    return;
  }

  const parsed = props.property.type === 'integer'
    ? parseInt(value, 10)
    : parseFloat(value);

  if (!isNaN(parsed)) {
    emit('update:modelValue', parsed);
  }
}

function handleBlur() {
  emit('blur');
}
</script>

<template>
  <div class="number-widget">
    <label :for="name" class="field-label">
      {{ property.title || name }}
      <span v-if="property.minimum !== undefined || property.maximum !== undefined" class="field-range">
        ({{ property.minimum ?? '' }} - {{ property.maximum ?? '' }})
      </span>
    </label>

    <input
      :id="name"
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
    />

    <p v-if="property.description && !error" class="field-description">
      {{ property.description }}
    </p>

    <p v-if="error" class="field-error">
      {{ error }}
    </p>
  </div>
</template>

<style scoped>
.number-widget {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.field-label {
  font-size: 12px;
  font-weight: 500;
  color: var(--color-text-secondary, #6b7280);
}

.field-range {
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
