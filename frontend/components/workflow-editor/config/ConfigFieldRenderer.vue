<script setup lang="ts">
import type { ParsedField } from './types/config-schema';
import TextWidget from './widgets/TextWidget.vue';
import TextareaWidget from './widgets/TextareaWidget.vue';
import NumberWidget from './widgets/NumberWidget.vue';
import SelectWidget from './widgets/SelectWidget.vue';
import CheckboxWidget from './widgets/CheckboxWidget.vue';
import ArrayWidget from './widgets/ArrayWidget.vue';
import KeyValueWidget from './widgets/KeyValueWidget.vue';

const props = defineProps<{
  field: ParsedField;
  modelValue: unknown;
  error?: string;
  disabled?: boolean;
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
    case 'json': // Fallback to key-value for now
      return KeyValueWidget;
    default:
      return TextWidget;
  }
});

function handleUpdate(value: unknown) {
  emit('update:modelValue', value);
}

function handleBlur() {
  emit('blur');
}
</script>

<template>
  <component
    :is="widgetComponent"
    :name="field.name"
    :property="field.property"
    :model-value="(modelValue as any)"
    :override="field.override"
    :error="error"
    :disabled="disabled"
    @update:model-value="handleUpdate"
    @blur="handleBlur"
  />
</template>
