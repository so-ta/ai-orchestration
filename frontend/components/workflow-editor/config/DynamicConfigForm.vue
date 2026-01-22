<script setup lang="ts">
/**
 * DynamicConfigForm - JSON Schemaからフォームを動的生成するコンポーネント
 *
 * 標準JSON Schemaからウィジェットを自動推論し、バリデーション付きのフォームを生成。
 * オプショナルなui_configでウィジェットの上書きやグループ化が可能。
 */

import { computed, watch, toRef, onMounted } from 'vue';
import type { ConfigSchema, UIConfig, JSONSchemaProperty } from './types/config-schema';
import { useSchemaParser } from './composables/useSchemaParser';
import { useValidation } from './composables/useValidation';
import ConfigFieldRenderer from './ConfigFieldRenderer.vue';

const { t } = useI18n();

// Icon name to SVG path mapping for common icons
const iconSvgPaths: Record<string, string> = {
  robot: 'M12 2a2 2 0 0 1 2 2c0 .74-.4 1.39-1 1.73V7h1a7 7 0 0 1 7 7h1a1 1 0 0 1 1 1v3a1 1 0 0 1-1 1h-1v1a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-1H2a1 1 0 0 1-1-1v-3a1 1 0 0 1 1-1h1a7 7 0 0 1 7-7h1V5.73c-.6-.34-1-.99-1-1.73a2 2 0 0 1 2-2M7.5 13A2.5 2.5 0 0 0 5 15.5A2.5 2.5 0 0 0 7.5 18a2.5 2.5 0 0 0 2.5-2.5A2.5 2.5 0 0 0 7.5 13m9 0a2.5 2.5 0 0 0-2.5 2.5a2.5 2.5 0 0 0 2.5 2.5a2.5 2.5 0 0 0 2.5-2.5a2.5 2.5 0 0 0-2.5-2.5',
  message: 'M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z',
  braces: 'M4 4a2 2 0 0 0-2 2v3a2 2 0 0 1-2 2 2 2 0 0 1 2 2v3a2 2 0 0 0 2 2m16-12a2 2 0 0 1 2 2v3a2 2 0 0 0 2 2 2 2 0 0 0-2 2v3a2 2 0 0 1-2 2',
  'layout-template': 'M3 3h7v9H3zM14 3h7v5h-7zM14 12h7v9h-7zM3 16h7v5H3z',
  settings: 'M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z M12 8a4 4 0 1 0 0 8 4 4 0 0 0 0-8z',
};

function getIconPath(iconName: string | undefined): string | undefined {
  if (!iconName) return undefined;
  return iconSvgPaths[iconName] || undefined;
}

const props = withDefaults(
  defineProps<{
    schema: ConfigSchema | null | undefined;
    uiConfig?: UIConfig;
    modelValue: Record<string, unknown>;
    disabled?: boolean;
    applyDefaults?: boolean; // Whether to auto-apply default values
  }>(),
  {
    uiConfig: undefined,
    disabled: false,
    applyDefaults: true,
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
  validate,
} = useValidation(schemaRef, valuesRef);

// Watch validation state - emit immediately to set initial state
watch(isValid, (valid) => {
  emit('validation-change', valid);
}, { immediate: true });

/**
 * Get default value from schema property
 */
function getDefaultValue(prop: JSONSchemaProperty): unknown {
  if (prop.default !== undefined) {
    return prop.default;
  }

  // Type-based defaults
  switch (prop.type) {
    case 'string':
      return '';
    case 'number':
    case 'integer':
      return undefined; // Don't set default for numbers (optional)
    case 'boolean':
      return false;
    case 'array':
      return [];
    case 'object':
      if (prop.properties) {
        const obj: Record<string, unknown> = {};
        for (const [key, subProp] of Object.entries(prop.properties)) {
          const value = getDefaultValue(subProp);
          if (value !== undefined) {
            obj[key] = value;
          }
        }
        return Object.keys(obj).length > 0 ? obj : undefined;
      }
      return {};
    default:
      return undefined;
  }
}

/**
 * Initialize values with defaults from schema
 */
function initializeDefaults() {
  if (!props.schema?.properties || !props.applyDefaults) return;

  const currentValues = props.modelValue;
  const updates: Record<string, unknown> = {};
  let hasUpdates = false;

  for (const [name, prop] of Object.entries(props.schema.properties)) {
    // Only set default if current value is undefined
    if (currentValues[name] === undefined) {
      const defaultValue = getDefaultValue(prop);
      if (defaultValue !== undefined) {
        updates[name] = defaultValue;
        hasUpdates = true;
      }
    }
  }

  if (hasUpdates) {
    emit('update:modelValue', { ...currentValues, ...updates });
  }
}

// Initialize defaults on mount and when schema changes
onMounted(() => {
  initializeDefaults();
});

watch(schemaRef, () => {
  initializeDefaults();
}, { immediate: false });

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

// Expose validation and initialization methods
defineExpose({
  validate,
  touchAll,
  isValid,
  initializeDefaults,
});
</script>

<template>
  <div class="dynamic-config-form">
    <!-- No schema message -->
    <div v-if="!schema" class="no-schema">
      {{ t('widgets.dynamicConfig.noSchema') }}
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
            <span class="group-icon">
              <svg v-if="getIconPath(group.icon)" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path :d="getIconPath(group.icon) ?? ''" />
              </svg>
              <span v-else>▸</span>
            </span>
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
