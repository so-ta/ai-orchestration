<script setup lang="ts">
import type { SchemaField } from '~/composables/schema'
import { FIELD_TYPES, ARRAY_ITEM_TYPES } from '~/composables/schema'

interface Props {
  field: SchemaField
  index: number
  depth?: number
  disabled?: boolean
  isArrayItem?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  depth: 0,
  disabled: false,
  isArrayItem: false,
})

const emit = defineEmits<{
  remove: []
  update: [key: keyof SchemaField, value: SchemaField[keyof SchemaField]]
  'toggle-expand': []
  'add-child': []
  'add-item-child': []
  'remove-child': [index: number]
  'remove-item-child': [index: number]
  'update-child': [childIndex: number, key: keyof SchemaField, value: SchemaField[keyof SchemaField]]
  'toggle-child-expand': [childIndex: number]
  blur: []
}>()

const typeOptions = computed(() => props.isArrayItem ? ARRAY_ITEM_TYPES : FIELD_TYPES)

function getDepthClass(d: number): string {
  return `depth-${Math.min(d, 3)}`
}

function handleFieldUpdate(key: keyof SchemaField, event: Event) {
  const target = event.target as HTMLInputElement | HTMLSelectElement
  const value = target.type === 'checkbox' ? (target as HTMLInputElement).checked : target.value
  emit('update', key, value)
}
</script>

<template>
  <div class="field-item" :class="getDepthClass(depth)">
    <div class="field-header">
      <div class="field-header-left">
        <button
          v-if="field.type === 'object' || (field.type === 'array' && field.itemType === 'object')"
          type="button"
          class="expand-button"
          @click="emit('toggle-expand')"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="12"
            height="12"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            :class="{ rotated: field.expanded }"
          >
            <polyline points="9 18 15 12 9 6" />
          </svg>
        </button>
        <span class="field-index">#{{ index + 1 }}</span>
        <span v-if="field.type === 'object'" class="type-badge object-badge">{ }</span>
        <span v-else-if="field.type === 'array'" class="type-badge array-badge">[ ]</span>
      </div>
      <button
        type="button"
        class="remove-button"
        :disabled="disabled"
        title="フィールドを削除"
        @click="emit('remove')"
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="18" y1="6" x2="6" y2="18" />
          <line x1="6" y1="6" x2="18" y2="18" />
        </svg>
      </button>
    </div>

    <div class="field-row">
      <div class="field-group name-group">
        <label class="field-label">フィールド名 *</label>
        <input
          type="text"
          class="field-input"
          :value="field.name"
          :disabled="disabled"
          placeholder="field_name"
          @input="handleFieldUpdate('name', $event)"
          @blur="emit('blur')"
        >
      </div>
      <div class="field-group type-group">
        <label class="field-label">型</label>
        <select
          class="field-select"
          :value="field.type"
          :disabled="disabled"
          @change="handleFieldUpdate('type', $event)"
        >
          <option v-for="t in typeOptions" :key="t.value" :value="t.value">
            {{ t.label }}
          </option>
        </select>
      </div>
      <div class="field-group required-group">
        <label class="field-label">必須</label>
        <label class="checkbox-wrapper">
          <input
            type="checkbox"
            :checked="field.required"
            :disabled="disabled"
            @change="handleFieldUpdate('required', $event)"
          >
          <span class="checkbox-mark" />
        </label>
      </div>
    </div>

    <!-- Array item type selector -->
    <div v-if="field.type === 'array' && !isArrayItem" class="field-row">
      <div class="field-group">
        <label class="field-label">配列の要素型</label>
        <select
          class="field-select"
          :value="field.itemType || 'string'"
          :disabled="disabled"
          @change="handleFieldUpdate('itemType', $event)"
        >
          <option v-for="t in ARRAY_ITEM_TYPES" :key="t.value" :value="t.value">
            {{ t.label }}
          </option>
        </select>
      </div>
    </div>

    <div class="field-row">
      <div class="field-group">
        <label class="field-label">表示名</label>
        <input
          type="text"
          class="field-input"
          :value="field.title"
          :disabled="disabled"
          placeholder="日本語の表示名"
          @input="handleFieldUpdate('title', $event)"
          @blur="emit('blur')"
        >
      </div>
    </div>

    <div v-if="!isArrayItem" class="field-row">
      <div class="field-group">
        <label class="field-label">説明</label>
        <input
          type="text"
          class="field-input"
          :value="field.description"
          :disabled="disabled"
          placeholder="このフィールドの説明"
          @input="handleFieldUpdate('description', $event)"
          @blur="emit('blur')"
        >
      </div>
    </div>

    <!-- Nested children for object type -->
    <div v-if="field.type === 'object' && field.expanded" class="nested-fields">
      <div class="nested-header">
        <span class="nested-label">プロパティ</span>
      </div>
      <div v-if="!field.children || field.children.length === 0" class="nested-empty">
        <p>プロパティが定義されていません</p>
      </div>
      <template v-else>
        <OutputSchemaFieldRow
          v-for="(child, childIndex) in field.children"
          :key="child.id"
          :field="child"
          :index="childIndex"
          :depth="depth + 1"
          :disabled="disabled"
          class="nested"
          @remove="emit('remove-child', childIndex)"
          @update="(key, value) => emit('update-child', childIndex, key, value)"
          @toggle-expand="emit('toggle-child-expand', childIndex)"
          @blur="emit('blur')"
        />
      </template>
      <button
        type="button"
        class="add-nested-button"
        :disabled="disabled"
        @click="emit('add-child')"
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="12" y1="5" x2="12" y2="19" />
          <line x1="5" y1="12" x2="19" y2="12" />
        </svg>
        プロパティを追加
      </button>
    </div>

    <!-- Nested children for array of objects -->
    <div v-if="field.type === 'array' && field.itemType === 'object' && field.expanded" class="nested-fields">
      <div class="nested-header">
        <span class="nested-label">配列要素のプロパティ</span>
      </div>
      <div v-if="!field.itemChildren || field.itemChildren.length === 0" class="nested-empty">
        <p>プロパティが定義されていません</p>
      </div>
      <template v-else>
        <OutputSchemaFieldRow
          v-for="(child, childIndex) in field.itemChildren"
          :key="child.id"
          :field="child"
          :index="childIndex"
          :depth="depth + 1"
          :disabled="disabled"
          :is-array-item="true"
          class="nested"
          @remove="emit('remove-item-child', childIndex)"
          @update="(key, value) => emit('update-child', childIndex, key, value)"
          @toggle-expand="emit('toggle-child-expand', childIndex)"
          @blur="emit('blur')"
        />
      </template>
      <button
        type="button"
        class="add-nested-button"
        :disabled="disabled"
        @click="emit('add-item-child')"
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="12" y1="5" x2="12" y2="19" />
          <line x1="5" y1="12" x2="19" y2="12" />
        </svg>
        プロパティを追加
      </button>
    </div>
  </div>
</template>

<style scoped>
.field-item {
  padding: 0.75rem;
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: 6px;
}

.field-item.nested {
  margin-left: 0;
  background: var(--color-surface);
}

/* Depth-based styling */
.field-item.depth-0 {
  border-left: 3px solid var(--color-primary);
}

.field-item.depth-1 {
  border-left: 3px solid #10b981;
}

.field-item.depth-2 {
  border-left: 3px solid #f59e0b;
}

.field-item.depth-3 {
  border-left: 3px solid #8b5cf6;
}

.field-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.5rem;
}

.field-header-left {
  display: flex;
  align-items: center;
  gap: 0.375rem;
}

.expand-button {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  padding: 0;
  background: transparent;
  border: none;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: transform 0.15s ease;
}

.expand-button:hover {
  color: var(--color-text);
}

.expand-button svg {
  transition: transform 0.15s ease;
}

.expand-button svg.rotated {
  transform: rotate(90deg);
}

.field-index {
  font-size: 0.625rem;
  font-weight: 600;
  color: var(--color-text-secondary);
}

.type-badge {
  font-size: 0.625rem;
  font-weight: 600;
  padding: 0.125rem 0.375rem;
  border-radius: 3px;
}

.object-badge {
  background: rgba(139, 92, 246, 0.15);
  color: #8b5cf6;
}

.array-badge {
  background: rgba(16, 185, 129, 0.15);
  color: #10b981;
}

.remove-button {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  padding: 0;
  color: var(--color-text-secondary);
  background: transparent;
  border: none;
  border-radius: 3px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.remove-button:hover:not(:disabled) {
  color: var(--color-error);
  background: rgba(239, 68, 68, 0.1);
}

.remove-button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.field-row {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 0.5rem;
}

.field-row:last-child {
  margin-bottom: 0;
}

.field-group {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.name-group {
  flex: 2;
}

.type-group {
  flex: 1;
}

.required-group {
  flex: 0 0 auto;
  width: 50px;
  align-items: center;
}

.field-label {
  font-size: 0.625rem;
  font-weight: 500;
  color: var(--color-text-secondary);
}

.field-input,
.field-select {
  padding: 0.375rem 0.5rem;
  font-size: 0.75rem;
  color: var(--color-text);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 4px;
  outline: none;
  transition: border-color 0.15s ease;
}

.field-input:focus,
.field-select:focus {
  border-color: var(--color-primary);
}

.field-input:disabled,
.field-select:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.field-input::placeholder {
  color: var(--color-text-secondary);
  opacity: 0.5;
}

.checkbox-wrapper {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}

.checkbox-wrapper input {
  position: absolute;
  opacity: 0;
  cursor: pointer;
}

.checkbox-mark {
  width: 18px;
  height: 18px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 4px;
  transition: all 0.15s ease;
}

.checkbox-wrapper input:checked ~ .checkbox-mark {
  background: var(--color-primary);
  border-color: var(--color-primary);
}

.checkbox-wrapper input:checked ~ .checkbox-mark::after {
  content: '';
  position: absolute;
  left: 6px;
  top: 2px;
  width: 5px;
  height: 10px;
  border: solid white;
  border-width: 0 2px 2px 0;
  transform: rotate(45deg);
}

.checkbox-wrapper input:disabled ~ .checkbox-mark {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Nested fields */
.nested-fields {
  margin-top: 0.75rem;
  padding-top: 0.75rem;
  border-top: 1px dashed var(--color-border);
}

.nested-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.5rem;
}

.nested-label {
  font-size: 0.6875rem;
  font-weight: 600;
  color: var(--color-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.nested-empty {
  padding: 0.75rem;
  text-align: center;
  background: var(--color-surface);
  border: 1px dashed var(--color-border);
  border-radius: 4px;
  margin-bottom: 0.5rem;
}

.nested-empty p {
  margin: 0;
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
}

.add-nested-button {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.25rem;
  width: 100%;
  padding: 0.375rem;
  font-size: 0.6875rem;
  font-weight: 500;
  color: var(--color-text-secondary);
  background: transparent;
  border: 1px dashed var(--color-border);
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.15s ease;
  margin-top: 0.5rem;
}

.add-nested-button:hover:not(:disabled) {
  color: var(--color-primary);
  border-color: var(--color-primary);
  background: rgba(59, 130, 246, 0.05);
}

.add-nested-button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
