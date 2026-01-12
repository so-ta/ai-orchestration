<script setup lang="ts">
import type { JSONSchemaProperty, FieldOverride } from '../types/config-schema';

const props = defineProps<{
  name: string;
  property: JSONSchemaProperty;
  modelValue: boolean | undefined;
  override?: FieldOverride;
  error?: string;
  disabled?: boolean;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void;
  (e: 'blur'): void;
}>();

const isChecked = computed(() => {
  if (props.modelValue !== undefined) return props.modelValue;
  if (props.property.default !== undefined) return props.property.default as boolean;
  return false;
});

function handleChange(event: Event) {
  const target = event.target as HTMLInputElement;
  emit('update:modelValue', target.checked);
}

function handleBlur() {
  emit('blur');
}
</script>

<template>
  <div class="checkbox-widget">
    <label :class="['checkbox-label', { disabled }]">
      <input
        type="checkbox"
        :id="name"
        :checked="isChecked"
        :disabled="disabled"
        class="checkbox-input"
        @change="handleChange"
        @blur="handleBlur"
      />
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
    </label>

    <p v-if="property.description && !error" class="field-description">
      {{ property.description }}
    </p>

    <p v-if="error" class="field-error">
      {{ error }}
    </p>
  </div>
</template>

<style scoped>
.checkbox-widget {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.checkbox-label {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  user-select: none;
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

.field-description {
  font-size: 11px;
  color: var(--color-text-muted, #9ca3af);
  margin: 0;
  padding-left: 26px;
}

.field-error {
  font-size: 11px;
  color: var(--color-error, #ef4444);
  margin: 0;
  padding-left: 26px;
}
</style>
