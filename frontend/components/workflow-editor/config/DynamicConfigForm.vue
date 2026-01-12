<script setup lang="ts">
/**
 * DynamicConfigForm - JSON Schemaからフォームを動的生成するコンポーネント
 *
 * 標準JSON Schemaからウィジェットを自動推論し、バリデーション付きのフォームを生成。
 * オプショナルなui_configでウィジェットの上書きやグループ化が可能。
 */

import { computed, watch, toRef } from 'vue';
import type { ConfigSchema, UIConfig } from './types/config-schema';
import { useSchemaParser } from './composables/useSchemaParser';
import { useValidation } from './composables/useValidation';
import ConfigFieldRenderer from './ConfigFieldRenderer.vue';

const props = withDefaults(
  defineProps<{
    schema: ConfigSchema | null | undefined;
    uiConfig?: UIConfig;
    modelValue: Record<string, unknown>;
    disabled?: boolean;
  }>(),
  {
    disabled: false,
  }
);

const emit = defineEmits<{
  (e: 'update:modelValue', value: Record<string, unknown>): void;
  (e: 'validation-change', valid: boolean): void;
}>();

// Refs for composables
const schemaRef = toRef(props, 'schema');
const uiConfigRef = toRef(props, 'uiConfig');
const valuesRef = computed(() => props.modelValue);

// Schema parsing
const { visibleFields, groups, fieldsByGroup } = useSchemaParser(
  schemaRef,
  uiConfigRef,
  valuesRef
);

// Validation
const {
  isValid,
  touch,
  touchAll,
  getFieldError,
  hasFieldError,
  validate,
} = useValidation(schemaRef, valuesRef);

// Watch validation state
watch(isValid, (valid) => {
  emit('validation-change', valid);
});

// Update field value
function updateFieldValue(fieldName: string, value: unknown) {
  const newValues = { ...props.modelValue, [fieldName]: value };
  emit('update:modelValue', newValues);
}

// Handle field blur (mark as touched)
function handleFieldBlur(fieldName: string) {
  touch(fieldName);
}

// Check if there are any groups
const hasGroups = computed(() => groups.value.length > 0);

// Get ungrouped fields
const ungroupedFields = computed(() => fieldsByGroup.value._ungrouped || []);

// Expose validation methods
defineExpose({
  validate,
  touchAll,
  isValid,
});
</script>

<template>
  <div class="dynamic-config-form">
    <!-- No schema message -->
    <div v-if="!schema" class="no-schema">
      設定スキーマが定義されていません
    </div>

    <template v-else>
      <!-- Grouped fields -->
      <template v-if="hasGroups">
        <details
          v-for="group in groups"
          :key="group.id"
          :open="!group.collapsed"
          class="field-group"
        >
          <summary class="group-header">
            <span class="group-icon">{{ group.icon || '▸' }}</span>
            <span class="group-title">{{ group.title }}</span>
          </summary>

          <div class="group-content">
            <div
              v-for="field in fieldsByGroup[group.id]"
              :key="field.name"
              class="field-wrapper"
            >
              <ConfigFieldRenderer
                :field="field"
                :model-value="modelValue[field.name]"
                :error="getFieldError(field.name)"
                :disabled="disabled"
                @update:model-value="(v) => updateFieldValue(field.name, v)"
                @blur="() => handleFieldBlur(field.name)"
              />
            </div>
          </div>
        </details>

        <!-- Ungrouped fields (if any) -->
        <div
          v-for="field in ungroupedFields"
          :key="field.name"
          class="field-wrapper"
        >
          <ConfigFieldRenderer
            :field="field"
            :model-value="modelValue[field.name]"
            :error="getFieldError(field.name)"
            :disabled="disabled"
            @update:model-value="(v) => updateFieldValue(field.name, v)"
            @blur="() => handleFieldBlur(field.name)"
          />
        </div>
      </template>

      <!-- No groups - flat list -->
      <template v-else>
        <div
          v-for="field in visibleFields"
          :key="field.name"
          class="field-wrapper"
        >
          <ConfigFieldRenderer
            :field="field"
            :model-value="modelValue[field.name]"
            :error="getFieldError(field.name)"
            :disabled="disabled"
            @update:model-value="(v) => updateFieldValue(field.name, v)"
            @blur="() => handleFieldBlur(field.name)"
          />
        </div>
      </template>
    </template>
  </div>
</template>

<style scoped>
.dynamic-config-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.no-schema {
  padding: 24px;
  text-align: center;
  font-size: 13px;
  color: var(--color-text-muted, #9ca3af);
  background: var(--color-bg-subtle, #f9fafb);
  border-radius: 8px;
  border: 1px dashed var(--color-border, #e5e7eb);
}

.field-wrapper {
  /* Individual field spacing handled by widgets */
}

.field-group {
  border: 1px solid var(--color-border, #e5e7eb);
  border-radius: 8px;
  overflow: hidden;
}

.field-group[open] > .group-header .group-icon {
  transform: rotate(90deg);
}

.group-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background: var(--color-bg-subtle, #f9fafb);
  cursor: pointer;
  user-select: none;
  list-style: none;
}

.group-header::-webkit-details-marker {
  display: none;
}

.group-header:hover {
  background: var(--color-bg-hover, #f3f4f6);
}

.group-icon {
  font-size: 10px;
  color: var(--color-text-muted, #9ca3af);
  transition: transform 0.15s;
}

.group-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text, #111827);
}

.group-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 16px;
}
</style>
