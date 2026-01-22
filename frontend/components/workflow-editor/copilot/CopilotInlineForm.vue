<script setup lang="ts">
/**
 * CopilotInlineForm.vue
 *
 * Simple inline form for configuration within chat messages.
 * Supports text, number, select, time, timezone, and cron fields.
 */
import type { FormField } from './types'

const { t } = useI18n()

const props = defineProps<{
  fields: FormField[]
  submitLabel?: string
  disabled?: boolean
}>()

const emit = defineEmits<{
  submit: [values: Record<string, string | number>]
}>()

// Form values
const values = reactive<Record<string, string | number>>({})

// Initialize values with defaults
onMounted(() => {
  for (const field of props.fields) {
    if (field.defaultValue !== undefined) {
      values[field.id] = field.defaultValue
    } else if (field.type === 'number') {
      values[field.id] = 0
    } else {
      values[field.id] = ''
    }
  }
})

// Common timezone options
const timezoneOptions = [
  { value: 'Asia/Tokyo', label: 'Asia/Tokyo (JST)' },
  { value: 'America/New_York', label: 'America/New_York (EST/EDT)' },
  { value: 'America/Los_Angeles', label: 'America/Los_Angeles (PST/PDT)' },
  { value: 'Europe/London', label: 'Europe/London (GMT/BST)' },
  { value: 'Europe/Paris', label: 'Europe/Paris (CET/CEST)' },
  { value: 'UTC', label: 'UTC' },
]

// Validation
const isValid = computed(() => {
  for (const field of props.fields) {
    if (field.required && !values[field.id]) {
      return false
    }
  }
  return true
})

// Handle submit
function handleSubmit() {
  if (isValid.value && !props.disabled) {
    emit('submit', { ...values })
  }
}

// Get options for a field
function getFieldOptions(field: FormField) {
  if (field.type === 'timezone') {
    return timezoneOptions
  }
  return field.options || []
}
</script>

<template>
  <div class="inline-form">
    <div class="form-fields">
      <div v-for="field in fields" :key="field.id" class="form-field">
        <label :for="field.id" class="field-label">
          {{ field.label }}
          <span v-if="field.required" class="required">*</span>
        </label>

        <!-- Text input -->
        <input
          v-if="field.type === 'text'"
          :id="field.id"
          v-model="values[field.id]"
          type="text"
          class="field-input"
          :placeholder="field.placeholder"
          :disabled="disabled"
        >

        <!-- Number input -->
        <input
          v-else-if="field.type === 'number'"
          :id="field.id"
          v-model.number="values[field.id]"
          type="number"
          class="field-input"
          :placeholder="field.placeholder"
          :disabled="disabled"
        >

        <!-- Time input -->
        <input
          v-else-if="field.type === 'time'"
          :id="field.id"
          v-model="values[field.id]"
          type="time"
          class="field-input"
          :disabled="disabled"
        >

        <!-- Select / Timezone -->
        <select
          v-else-if="field.type === 'select' || field.type === 'timezone'"
          :id="field.id"
          v-model="values[field.id]"
          class="field-select"
          :disabled="disabled"
        >
          <option v-if="field.placeholder" value="" disabled>
            {{ field.placeholder }}
          </option>
          <option
            v-for="opt in getFieldOptions(field)"
            :key="opt.value"
            :value="opt.value"
          >
            {{ opt.label }}
          </option>
        </select>

        <!-- Cron input -->
        <input
          v-else-if="field.type === 'cron'"
          :id="field.id"
          v-model="values[field.id]"
          type="text"
          class="field-input cron"
          :placeholder="field.placeholder || '0 9 * * *'"
          :disabled="disabled"
        >
      </div>
    </div>

    <button
      class="submit-btn"
      :disabled="!isValid || disabled"
      @click="handleSubmit"
    >
      {{ submitLabel || t('copilot.action.apply') }}
    </button>
  </div>
</template>

<style scoped>
.inline-form {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.form-fields {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.form-field {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.field-label {
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--color-text-secondary);
}

.required {
  color: var(--color-error);
}

.field-input,
.field-select {
  padding: 0.5rem 0.75rem;
  font-size: 0.8125rem;
  font-family: inherit;
  color: var(--color-text);
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: 6px;
  outline: none;
  transition: border-color 0.15s;
}

.field-input:focus,
.field-select:focus {
  border-color: var(--color-primary);
}

.field-input:disabled,
.field-select:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.field-input.cron {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}

.field-select {
  appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%236b7280' stroke-width='2'%3E%3Cpolyline points='6 9 12 15 18 9'%3E%3C/polyline%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 0.75rem center;
  padding-right: 2rem;
}

.submit-btn {
  padding: 0.5rem 1rem;
  font-size: 0.8125rem;
  font-weight: 500;
  color: white;
  background: var(--color-primary);
  border: none;
  border-radius: 6px;
  cursor: pointer;
  transition: opacity 0.15s;
  align-self: flex-start;
}

.submit-btn:hover:not(:disabled) {
  opacity: 0.9;
}

.submit-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
