<script setup lang="ts">
import type { JSONSchemaProperty, FieldOverride } from '../types/config-schema';

const props = defineProps<{
  name: string;
  property: JSONSchemaProperty;
  modelValue: unknown[] | undefined;
  override?: FieldOverride;
  error?: string;
  disabled?: boolean;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: unknown[]): void;
  (e: 'blur'): void;
}>();

const items = computed(() => props.modelValue || []);

const itemSchema = computed(() => props.property.items);

const canAdd = computed(() => {
  if (props.property.maxItems === undefined) return true;
  return items.value.length < props.property.maxItems;
});

const canRemove = computed(() => {
  if (props.property.minItems === undefined) return true;
  return items.value.length > props.property.minItems;
});

function addItem() {
  if (!canAdd.value) return;

  const newItem = getDefaultValue(itemSchema.value);
  emit('update:modelValue', [...items.value, newItem]);
}

function removeItem(index: number) {
  if (!canRemove.value) return;

  const newItems = [...items.value];
  newItems.splice(index, 1);
  emit('update:modelValue', newItems);
}

function updateItem(index: number, value: unknown) {
  const newItems = [...items.value];
  newItems[index] = value;
  emit('update:modelValue', newItems);
}

function getDefaultValue(schema?: JSONSchemaProperty): unknown {
  if (!schema) return '';

  switch (schema.type) {
    case 'string':
      return schema.default ?? '';
    case 'number':
    case 'integer':
      return schema.default ?? 0;
    case 'boolean':
      return schema.default ?? false;
    case 'object':
      if (schema.properties) {
        const obj: Record<string, unknown> = {};
        for (const [key, prop] of Object.entries(schema.properties)) {
          obj[key] = getDefaultValue(prop);
        }
        return obj;
      }
      return {};
    case 'array':
      return [];
    default:
      return '';
  }
}

function getItemLabel(item: unknown, index: number): string {
  if (typeof item === 'object' && item !== null) {
    // Try to get a display name from common fields
    const obj = item as Record<string, unknown>;
    if (obj.name) return String(obj.name);
    if (obj.label) return String(obj.label);
    if (obj.title) return String(obj.title);
  }
  return `アイテム ${index + 1}`;
}

function isObjectItem(schema?: JSONSchemaProperty): boolean {
  return schema?.type === 'object' && !!schema.properties;
}

function getObjectFields(schema?: JSONSchemaProperty): [string, JSONSchemaProperty][] {
  if (!schema?.properties) return [];
  return Object.entries(schema.properties);
}
</script>

<template>
  <div class="array-widget">
    <div class="array-header">
      <label class="field-label">
        {{ property.title || name }}
        <span v-if="property.minItems || property.maxItems" class="field-count">
          ({{ items.length }}{{ property.maxItems ? ` / ${property.maxItems}` : '' }})
        </span>
      </label>

      <button
        v-if="canAdd"
        type="button"
        class="add-button"
        :disabled="disabled"
        @click="addItem"
      >
        + 追加
      </button>
    </div>

    <div v-if="items.length === 0" class="empty-state">
      アイテムがありません
    </div>

    <div v-else class="items-list">
      <div
        v-for="(item, index) in items"
        :key="index"
        class="item-row"
      >
        <div class="item-content">
          <!-- Object item with nested fields -->
          <template v-if="isObjectItem(itemSchema)">
            <div class="item-header">
              <span class="item-label">{{ getItemLabel(item, index) }}</span>
            </div>
            <div class="item-fields">
              <div
                v-for="[fieldName, fieldSchema] in getObjectFields(itemSchema)"
                :key="fieldName"
                class="item-field"
              >
                <label class="item-field-label">{{ fieldSchema.title || fieldName }}</label>
                <input
                  v-if="fieldSchema.type === 'string'"
                  type="text"
                  :value="(item as Record<string, unknown>)[fieldName] ?? ''"
                  :disabled="disabled"
                  class="item-input"
                  @input="(e) => updateItem(index, { ...(item as Record<string, unknown>), [fieldName]: (e.target as HTMLInputElement).value })"
                />
                <input
                  v-else-if="fieldSchema.type === 'boolean'"
                  type="checkbox"
                  :checked="(item as Record<string, unknown>)[fieldName] as boolean ?? false"
                  :disabled="disabled"
                  class="item-checkbox"
                  @change="(e) => updateItem(index, { ...(item as Record<string, unknown>), [fieldName]: (e.target as HTMLInputElement).checked })"
                />
              </div>
            </div>
          </template>

          <!-- Simple item (string, number, etc.) -->
          <template v-else>
            <input
              v-if="itemSchema?.type === 'string'"
              type="text"
              :value="item as string"
              :disabled="disabled"
              class="item-input full-width"
              @input="(e) => updateItem(index, (e.target as HTMLInputElement).value)"
            />
            <input
              v-else-if="itemSchema?.type === 'number' || itemSchema?.type === 'integer'"
              type="number"
              :value="item as number"
              :disabled="disabled"
              class="item-input full-width"
              @input="(e) => updateItem(index, parseFloat((e.target as HTMLInputElement).value))"
            />
          </template>
        </div>

        <button
          v-if="canRemove"
          type="button"
          class="remove-button"
          :disabled="disabled"
          @click="removeItem(index)"
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
.array-widget {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.array-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.field-label {
  font-size: 12px;
  font-weight: 500;
  color: var(--color-text-secondary, #6b7280);
}

.field-count {
  font-weight: 400;
  color: var(--color-text-muted, #9ca3af);
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

.items-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.item-row {
  display: flex;
  gap: 8px;
  align-items: flex-start;
  padding: 12px;
  background: var(--color-bg-subtle, #f9fafb);
  border-radius: 6px;
  border: 1px solid var(--color-border, #e5e7eb);
}

.item-content {
  flex: 1;
  min-width: 0;
}

.item-header {
  margin-bottom: 8px;
}

.item-label {
  font-size: 12px;
  font-weight: 500;
  color: var(--color-text, #111827);
}

.item-fields {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.item-field {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.item-field-label {
  font-size: 11px;
  color: var(--color-text-muted, #9ca3af);
}

.item-input {
  padding: 6px 10px;
  border: 1px solid var(--color-border, #e5e7eb);
  border-radius: 4px;
  font-size: 13px;
  background: var(--color-bg-input, #fff);
}

.item-input:focus {
  outline: none;
  border-color: var(--color-primary, #3b82f6);
}

.item-input.full-width {
  width: 100%;
}

.item-checkbox {
  width: 16px;
  height: 16px;
}

.remove-button {
  flex-shrink: 0;
  width: 24px;
  height: 24px;
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
