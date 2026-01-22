<script setup lang="ts">
/**
 * SecretKeyWidget - シークレットキー選択ウィジェット
 *
 * テナントまたはシステムに登録されているシークレットキーの一覧から選択できる。
 * ブロックの required_credentials に基づいて利用可能なキーをフィルタリング。
 */
import type { JSONSchemaProperty, FieldOverride } from '../types/config-schema';

const props = defineProps<{
  name: string;
  property: JSONSchemaProperty;
  modelValue: string | undefined;
  override?: FieldOverride;
  error?: string;
  disabled?: boolean;
  required?: boolean;
  // Optional: filter by credential type
  credentialType?: string;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void;
  (e: 'blur'): void;
}>();

// Predefined secret key options
// TODO: In the future, fetch from API based on tenant's registered credentials
const secretKeyOptions = computed(() => {
  // Common secret keys by category
  const commonKeys = [
    { value: '', label: '選択してください', group: '' },
    // AI Providers
    { value: 'OPENAI_API_KEY', label: 'OpenAI API Key', group: 'AI' },
    { value: 'ANTHROPIC_API_KEY', label: 'Anthropic API Key', group: 'AI' },
    { value: 'COHERE_API_KEY', label: 'Cohere API Key', group: 'AI' },
    // Communication
    { value: 'SLACK_WEBHOOK_URL', label: 'Slack Webhook URL', group: 'Communication' },
    { value: 'DISCORD_WEBHOOK_URL', label: 'Discord Webhook URL', group: 'Communication' },
    { value: 'SENDGRID_API_KEY', label: 'SendGrid API Key', group: 'Communication' },
    // Version Control
    { value: 'GITHUB_TOKEN', label: 'GitHub Token', group: 'Version Control' },
    { value: 'GITLAB_TOKEN', label: 'GitLab Token', group: 'Version Control' },
    // Project Management
    { value: 'NOTION_API_KEY', label: 'Notion API Key', group: 'Project Management' },
    { value: 'LINEAR_API_KEY', label: 'Linear API Key', group: 'Project Management' },
    // Google
    { value: 'GOOGLE_API_KEY', label: 'Google API Key', group: 'Google' },
    { value: 'GOOGLE_SHEETS_API_KEY', label: 'Google Sheets API Key', group: 'Google' },
    // Search
    { value: 'TAVILY_API_KEY', label: 'Tavily API Key', group: 'Search' },
    // Custom
    { value: '_custom', label: 'カスタムキー名を入力...', group: '' },
  ];

  // Filter by credential type if specified
  if (props.credentialType) {
    return commonKeys.filter(
      (k) => k.group === props.credentialType || k.value === '' || k.value === '_custom'
    );
  }

  return commonKeys;
});

const isCustomMode = ref(false);
const customValue = ref('');

const displayValue = computed(() => {
  if (isCustomMode.value) return '_custom';
  if (props.modelValue !== undefined) return props.modelValue;
  if (props.property.default !== undefined) return props.property.default as string;
  return '';
});

const groupedOptions = computed(() => {
  const groups: Record<string, typeof secretKeyOptions.value> = {};
  const ungrouped: typeof secretKeyOptions.value = [];

  for (const option of secretKeyOptions.value) {
    if (option.group) {
      if (!groups[option.group]) {
        groups[option.group] = [];
      }
      groups[option.group].push(option);
    } else {
      ungrouped.push(option);
    }
  }

  return { groups, ungrouped };
});

function handleChange(event: Event) {
  const target = event.target as HTMLSelectElement;
  const value = target.value;

  if (value === '_custom') {
    isCustomMode.value = true;
    customValue.value = props.modelValue || '';
  } else {
    isCustomMode.value = false;
    emit('update:modelValue', value);
  }
}

function handleCustomInput(event: Event) {
  const target = event.target as HTMLInputElement;
  customValue.value = target.value;
  emit('update:modelValue', target.value);
}

function handleBlur() {
  emit('blur');
}

function exitCustomMode() {
  isCustomMode.value = false;
  customValue.value = '';
}

// Check if current value matches a predefined option
onMounted(() => {
  if (props.modelValue) {
    const isPredefined = secretKeyOptions.value.some(
      (opt) => opt.value === props.modelValue && opt.value !== '_custom'
    );
    if (!isPredefined) {
      isCustomMode.value = true;
      customValue.value = props.modelValue;
    }
  }
});
</script>

<template>
  <div class="secret-key-widget">
    <label :for="name" class="field-label">
      {{ property.title || name }}
      <span v-if="required" class="field-required">*</span>
    </label>

    <!-- Custom input mode -->
    <div v-if="isCustomMode" class="custom-input-wrapper">
      <input
        :id="name"
        type="text"
        :value="customValue"
        :disabled="disabled"
        :class="['field-input', { 'has-error': error }]"
        placeholder="シークレットキー名を入力"
        @input="handleCustomInput"
        @blur="handleBlur"
      >
      <button
        type="button"
        class="back-button"
        title="リストから選択"
        @click="exitCustomMode"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="16"
          height="16"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <polyline points="15 18 9 12 15 6" />
        </svg>
      </button>
    </div>

    <!-- Select mode -->
    <select
      v-else
      :id="name"
      :value="displayValue"
      :disabled="disabled"
      :class="['field-select', { 'has-error': error }]"
      @change="handleChange"
      @blur="handleBlur"
    >
      <!-- Ungrouped options first (placeholder and custom) -->
      <option
        v-for="option in groupedOptions.ungrouped"
        :key="option.value"
        :value="option.value"
        :disabled="option.value === ''"
      >
        {{ option.label }}
      </option>

      <!-- Grouped options -->
      <optgroup
        v-for="(options, groupName) in groupedOptions.groups"
        :key="groupName"
        :label="groupName"
      >
        <option
          v-for="option in options"
          :key="option.value"
          :value="option.value"
        >
          {{ option.label }}
        </option>
      </optgroup>
    </select>

    <p v-if="property.description && !error" class="field-description">
      {{ property.description }}
    </p>

    <p v-if="error" class="field-error">
      {{ error }}
    </p>

    <!-- Info about secret management -->
    <p class="field-hint">
      <svg
        xmlns="http://www.w3.org/2000/svg"
        width="12"
        height="12"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
      >
        <circle cx="12" cy="12" r="10" />
        <path d="M12 16v-4" />
        <path d="M12 8h.01" />
      </svg>
      シークレットは設定画面で管理されます
    </p>
  </div>
</template>

<style scoped>
.secret-key-widget {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.field-label {
  font-size: 12px;
  font-weight: 500;
  color: var(--color-text-secondary, #6b7280);
}

.field-required {
  color: var(--color-error, #ef4444);
  margin-left: 2px;
}

.field-select {
  padding: 8px 32px 8px 12px;
  border: 1px solid var(--color-border, #e5e7eb);
  border-radius: 6px;
  font-size: 14px;
  background: var(--color-bg-input, #fff);
  color: var(--color-text, #111827);
  cursor: pointer;
  appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 12 12'%3E%3Cpath fill='%236b7280' d='M3 4.5L6 7.5L9 4.5'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 12px center;
  transition: border-color 0.15s, box-shadow 0.15s;
}

.field-select:focus {
  outline: none;
  border-color: var(--color-primary, #3b82f6);
  box-shadow: 0 0 0 3px var(--color-primary-alpha, rgba(59, 130, 246, 0.1));
}

.field-select:disabled {
  background-color: var(--color-bg-disabled, #f3f4f6);
  cursor: not-allowed;
}

.field-select.has-error {
  border-color: var(--color-error, #ef4444);
}

.custom-input-wrapper {
  display: flex;
  gap: 8px;
}

.field-input {
  flex: 1;
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
  background-color: var(--color-bg-disabled, #f3f4f6);
  cursor: not-allowed;
}

.field-input.has-error {
  border-color: var(--color-error, #ef4444);
}

.back-button {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 8px;
  border: 1px solid var(--color-border, #e5e7eb);
  border-radius: 6px;
  background: var(--color-bg-input, #fff);
  color: var(--color-text-secondary, #6b7280);
  cursor: pointer;
  transition: all 0.15s;
}

.back-button:hover {
  background: var(--color-bg-hover, #f3f4f6);
  color: var(--color-text, #111827);
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

.field-hint {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  color: var(--color-text-muted, #9ca3af);
  margin: 4px 0 0 0;
}

.field-hint svg {
  flex-shrink: 0;
}
</style>
