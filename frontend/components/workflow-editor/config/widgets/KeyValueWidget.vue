<script setup lang="ts">
import type { JSONSchemaProperty, FieldOverride } from '../types/config-schema';

const props = defineProps<{
  name: string;
  property: JSONSchemaProperty;
  modelValue: Record<string, string> | undefined;
  override?: FieldOverride;
  error?: string;
  disabled?: boolean;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: Record<string, string>): void;
  (e: 'blur'): void;
}>();

interface KeyValuePair {
  key: string;
  value: string;
}

const pairs = computed<KeyValuePair[]>(() => {
  const obj = props.modelValue || {};
  return Object.entries(obj).map(([key, value]) => ({ key, value }));
});

function addPair() {
  const currentObj = props.modelValue || {};
  // Find a unique key
  let newKey = '';
  let counter = 1;
  while (newKey === '' || newKey in currentObj) {
    newKey = `key${counter}`;
    counter++;
  }
  emit('update:modelValue', { ...currentObj, [newKey]: '' });
}

function removePair(keyToRemove: string) {
  const currentObj = props.modelValue || {};
  const newObj = { ...currentObj };
  delete newObj[keyToRemove];
  emit('update:modelValue', newObj);
}

function updatePairKey(oldKey: string, newKey: string) {
  if (oldKey === newKey) return;

  const currentObj = props.modelValue || {};

  // Check if new key already exists
  if (newKey in currentObj && oldKey !== newKey) {
    return; // Prevent duplicate keys
  }

  const newObj: Record<string, string> = {};

  // Preserve order while renaming key
  for (const [key, value] of Object.entries(currentObj)) {
    if (key === oldKey) {
      newObj[newKey] = value;
    } else {
      newObj[key] = value;
    }
  }

  emit('update:modelValue', newObj);
}

function updatePairValue(key: string, value: string) {
  const currentObj = props.modelValue || {};
  emit('update:modelValue', { ...currentObj, [key]: value });
}
</script>

<template>
  <div class="keyvalue-widget">
    <div class="widget-header">
      <label class="field-label">
        {{ property.title || name }}
      </label>

      <button
        type="button"
        class="add-button"
        :disabled="disabled"
        @click="addPair"
      >
        + 追加
      </button>
    </div>

    <div v-if="pairs.length === 0" class="empty-state">
      キー・バリューペアがありません
    </div>

    <div v-else class="pairs-list">
      <div class="pairs-header">
        <span class="header-key">キー</span>
        <span class="header-value">値</span>
        <span class="header-action"></span>
      </div>

      <div
        v-for="pair in pairs"
        :key="pair.key"
        class="pair-row"
      >
        <input
          type="text"
          :value="pair.key"
          :disabled="disabled"
          class="key-input"
          placeholder="キー"
          @blur="(e) => updatePairKey(pair.key, (e.target as HTMLInputElement).value)"
        />

        <input
          type="text"
          :value="pair.value"
          :disabled="disabled"
          class="value-input"
          placeholder="値"
          @input="(e) => updatePairValue(pair.key, (e.target as HTMLInputElement).value)"
        />

        <button
          type="button"
          class="remove-button"
          :disabled="disabled"
          @click="removePair(pair.key)"
          title="削除"
        >
          ×
        </button>
      </div>
    </div>

    <p v-if="property.description && !error" class="field-description">
      {{ property.description }}
    </p>

    <p v-if="error" class="field-error">
      {{ error }}
    </p>
  </div>
</template>

<style scoped>
.keyvalue-widget {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.widget-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.field-label {
  font-size: 12px;
  font-weight: 500;
  color: var(--color-text-secondary, #6b7280);
}

.add-button {
  padding: 4px 12px;
  font-size: 12px;
  color: var(--color-primary, #3b82f6);
  background: transparent;
  border: 1px solid var(--color-primary, #3b82f6);
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.15s;
}

.add-button:hover:not(:disabled) {
  background: var(--color-primary, #3b82f6);
  color: white;
}

.add-button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.empty-state {
  padding: 16px;
  text-align: center;
  font-size: 12px;
  color: var(--color-text-muted, #9ca3af);
  background: var(--color-bg-subtle, #f9fafb);
  border-radius: 6px;
  border: 1px dashed var(--color-border, #e5e7eb);
}

.pairs-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.pairs-header {
  display: grid;
  grid-template-columns: 1fr 1fr 32px;
  gap: 8px;
  padding: 0 4px 4px;
  font-size: 11px;
  color: var(--color-text-muted, #9ca3af);
}

.pair-row {
  display: grid;
  grid-template-columns: 1fr 1fr 32px;
  gap: 8px;
  align-items: center;
}

.key-input,
.value-input {
  padding: 8px 10px;
  border: 1px solid var(--color-border, #e5e7eb);
  border-radius: 4px;
  font-size: 13px;
  background: var(--color-bg-input, #fff);
  transition: border-color 0.15s;
}

.key-input:focus,
.value-input:focus {
  outline: none;
  border-color: var(--color-primary, #3b82f6);
}

.key-input:disabled,
.value-input:disabled {
  background: var(--color-bg-disabled, #f3f4f6);
  cursor: not-allowed;
}

.remove-button {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  color: var(--color-text-muted, #9ca3af);
  background: transparent;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.15s;
}

.remove-button:hover:not(:disabled) {
  background: var(--color-error-bg, #fef2f2);
  color: var(--color-error, #ef4444);
}

.remove-button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
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
