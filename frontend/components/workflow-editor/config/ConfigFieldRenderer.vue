<script setup lang="ts">
import { computed } from 'vue';
import type { ParsedField, WidgetValue } from './types/config-schema';
import TextWidget from './widgets/TextWidget.vue';
import TextareaWidget from './widgets/TextareaWidget.vue';
import NumberWidget from './widgets/NumberWidget.vue';
import SelectWidget from './widgets/SelectWidget.vue';
import CheckboxWidget from './widgets/CheckboxWidget.vue';
import ArrayWidget from './widgets/ArrayWidget.vue';
import KeyValueWidget from './widgets/KeyValueWidget.vue';
import CodeWidget from './widgets/CodeWidget.vue';
import JsonWidget from './widgets/JsonWidget.vue';
import OutputSchemaWidget from './widgets/OutputSchemaWidget.vue';
import SecretKeyWidget from './widgets/SecretKeyWidget.vue';

const props = defineProps<{
  field: ParsedField;
  modelValue: unknown;
  error?: string;
  disabled?: boolean;
  required?: boolean;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: unknown): void;
  (e: 'blur'): void;
}>();

const widgetComponent = computed(() => {
  switch (props.field.widget) {
    case 'text':
      return TextWidget;
    case 'textarea':
      return TextareaWidget;
    case 'number':
    case 'slider': // Fallback to number for now
      return NumberWidget;
    case 'select':
    case 'radio': // Fallback to select for now
      return SelectWidget;
    case 'checkbox':
    case 'switch': // Fallback to checkbox for now
      return CheckboxWidget;
    case 'array':
      return ArrayWidget;
    case 'key-value':
      return KeyValueWidget;
    case 'json':
      return JsonWidget;
    case 'output-schema':
      return OutputSchemaWidget;
    case 'code':
      return CodeWidget;
    case 'secret':
      return SecretKeyWidget;
    default:
      return TextWidget;
  }
});

// Normalize value to WidgetValue type for type-safe passing
// Convert null to undefined since widgets don't expect null
const normalizedValue = computed((): WidgetValue => {
  if (props.modelValue === null) {
    return undefined;
  }
  return props.modelValue as WidgetValue;
});

function handleUpdate(value: unknown) {
  emit('update:modelValue', value);
}

function handleBlur() {
  emit('blur');
}
</script>

<template>
  <!--
    Dynamic component with type assertion: Each widget has specific modelValue types
    (string, number, boolean, object, etc.) but WidgetValue is their union type.
    Vue's dynamic component type inference cannot handle this perfectly.
  -->
  <component
    :is="widgetComponent as any"
    :name="field.name"
    :property="field.property"
    :model-value="normalizedValue"
    :override="field.override"
    :error="error"
    :disabled="disabled"
    :required="field.required"
    @update:model-value="handleUpdate"
    @blur="handleBlur"
  />
</template>
