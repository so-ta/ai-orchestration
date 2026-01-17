<script setup lang="ts">
/**
 * TriggerConfigPanel.vue
 * Startブロックのトリガー設定UIコンポーネント
 *
 * トリガータイプ選択とタイプ別の設定フォームを提供
 */

const { t } = useI18n()

type StartTriggerType = 'manual' | 'webhook' | 'schedule' | 'slack' | 'email'

interface WebhookConfig {
  secret?: string
  input_mapping?: Record<string, string>
  allowed_ips?: string[]
}

interface ScheduleConfig {
  cron_expression?: string
  timezone?: string
  input_data?: Record<string, unknown>
}

interface SlackConfig {
  event_types?: string[]
  channel_filter?: string[]
}

interface EmailConfig {
  trigger_condition?: string
  input_mapping?: Record<string, string>
}

type TriggerConfig = WebhookConfig | ScheduleConfig | SlackConfig | EmailConfig | Record<string, unknown>

const props = defineProps<{
  triggerType?: StartTriggerType
  triggerConfig?: TriggerConfig
  stepId?: string
  readonly?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:trigger', data: { trigger_type: StartTriggerType; trigger_config: TriggerConfig }): void
}>()

// Local state
const localTriggerType = ref<StartTriggerType>(props.triggerType || 'manual')
const localConfig = ref<TriggerConfig>({ ...(props.triggerConfig || {}) })

// Watch for prop changes
watch(() => props.triggerType, (newVal) => {
  if (newVal && newVal !== localTriggerType.value) {
    localTriggerType.value = newVal
  }
}, { immediate: true })

watch(() => props.triggerConfig, (newVal) => {
  if (newVal) {
    localConfig.value = { ...newVal }
  }
}, { immediate: true, deep: true })

// Trigger type options
const triggerTypeOptions = [
  { value: 'manual', label: t('trigger.type.manual'), icon: '👤', description: t('trigger.description.manual') },
  { value: 'webhook', label: t('trigger.type.webhook'), icon: '↗', description: t('trigger.description.webhook') },
  { value: 'schedule', label: t('trigger.type.schedule'), icon: '⏰', description: t('trigger.description.schedule') },
  { value: 'slack', label: t('trigger.type.slack'), icon: '#', description: t('trigger.description.slack') },
  { value: 'email', label: t('trigger.type.email'), icon: '✉', description: t('trigger.description.email') },
]

// Emit changes
function emitUpdate() {
  emit('update:trigger', {
    trigger_type: localTriggerType.value,
    trigger_config: localConfig.value,
  })
}

// Handle trigger type change
function handleTypeChange(newType: StartTriggerType) {
  localTriggerType.value = newType
  // Reset config when type changes
  localConfig.value = getDefaultConfig(newType)
  emitUpdate()
}

// Get default config for trigger type
function getDefaultConfig(type: StartTriggerType): TriggerConfig {
  switch (type) {
    case 'webhook':
      return { secret: '', input_mapping: {}, allowed_ips: [] }
    case 'schedule':
      return { cron_expression: '0 9 * * *', timezone: 'Asia/Tokyo', input_data: {} }
    case 'slack':
      return { event_types: ['message'], channel_filter: [] }
    case 'email':
      return { trigger_condition: '', input_mapping: {} }
    default:
      return {}
  }
}

// Webhook config helpers
const webhookConfig = computed({
  get: () => localConfig.value as WebhookConfig,
  set: (val) => {
    localConfig.value = val
    emitUpdate()
  },
})

// Schedule config helpers
const scheduleConfig = computed({
  get: () => localConfig.value as ScheduleConfig,
  set: (val) => {
    localConfig.value = val
    emitUpdate()
  },
})

// Slack config helpers
const slackConfig = computed({
  get: () => localConfig.value as SlackConfig,
  set: (val) => {
    localConfig.value = val
    emitUpdate()
  },
})

// Email config helpers
const emailConfig = computed({
  get: () => localConfig.value as EmailConfig,
  set: (val) => {
    localConfig.value = val
    emitUpdate()
  },
})

// Update single field
function updateField(field: string, value: unknown) {
  (localConfig.value as Record<string, unknown>)[field] = value
  emitUpdate()
}

// Slack event type options
const slackEventOptions = [
  { value: 'message', label: 'メッセージ' },
  { value: 'reaction_added', label: 'リアクション追加' },
  { value: 'app_mention', label: 'アプリメンション' },
  { value: 'slash_command', label: 'スラッシュコマンド' },
]

// Cron expression examples
const cronExamples = [
  { label: t('schedules.cronExamples.everyMinute'), value: '* * * * *' },
  { label: t('schedules.cronExamples.everyHour'), value: '0 * * * *' },
  { label: t('schedules.cronExamples.everyDay9am'), value: '0 9 * * *' },
  { label: t('schedules.cronExamples.everyMonday'), value: '0 9 * * 1' },
  { label: t('schedules.cronExamples.firstOfMonth'), value: '0 0 1 * *' },
]

// Timezone options (common ones)
const timezoneOptions = [
  'Asia/Tokyo',
  'America/New_York',
  'America/Los_Angeles',
  'Europe/London',
  'Europe/Paris',
  'UTC',
]
</script>

<template>
  <div class="trigger-config-panel">
    <!-- Trigger Type Selection -->
    <div class="trigger-type-section">
      <label class="trigger-label">
        {{ t('trigger.selectType') }}
      </label>
      <div class="trigger-options">
        <button
          v-for="option in triggerTypeOptions"
          :key="option.value"
          type="button"
          class="trigger-option"
          :class="{ selected: localTriggerType === option.value }"
          :disabled="readonly"
          @click="handleTypeChange(option.value as StartTriggerType)"
        >
          <span class="trigger-icon">{{ option.icon }}</span>
          <div class="trigger-option-content">
            <div class="trigger-option-label">{{ option.label }}</div>
            <div class="trigger-option-desc">{{ option.description }}</div>
          </div>
          <div v-if="localTriggerType === option.value" class="trigger-check">
            <svg width="16" height="16" fill="currentColor" viewBox="0 0 20 20">
              <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
            </svg>
          </div>
        </button>
      </div>
    </div>

    <!-- Type-specific Configuration -->
    <div class="trigger-config-form">
      <!-- Manual - No config needed -->
      <div v-if="localTriggerType === 'manual'" class="trigger-manual-hint">
        <p>{{ t('trigger.manualDescription') }}</p>
      </div>

      <!-- Webhook Config -->
      <div v-else-if="localTriggerType === 'webhook'" class="trigger-form-fields">
        <div class="form-group">
          <label class="form-label">
            {{ t('trigger.webhook.secret') }}
          </label>
          <input
            :value="webhookConfig.secret"
            type="text"
            class="form-input"
            :placeholder="t('trigger.webhook.secretPlaceholder')"
            :disabled="readonly"
            @input="updateField('secret', ($event.target as HTMLInputElement).value)"
          >
          <p class="form-hint">{{ t('trigger.webhook.secretHint') }}</p>
        </div>

        <div class="form-group">
          <label class="form-label">
            {{ t('trigger.webhook.allowedIps') }}
          </label>
          <input
            :value="(webhookConfig.allowed_ips || []).join(', ')"
            type="text"
            class="form-input"
            :placeholder="t('trigger.webhook.allowedIpsPlaceholder')"
            :disabled="readonly"
            @input="updateField('allowed_ips', ($event.target as HTMLInputElement).value.split(',').map(s => s.trim()).filter(Boolean))"
          >
          <p class="form-hint">{{ t('trigger.webhook.allowedIpsHint') }}</p>
        </div>
      </div>

      <!-- Schedule Config -->
      <div v-else-if="localTriggerType === 'schedule'" class="trigger-form-fields">
        <div class="form-group">
          <label class="form-label">
            {{ t('schedules.form.cronExpression') }}
          </label>
          <input
            :value="scheduleConfig.cron_expression"
            type="text"
            class="form-input code-input"
            placeholder="0 9 * * *"
            :disabled="readonly"
            @input="updateField('cron_expression', ($event.target as HTMLInputElement).value)"
          >
          <p class="form-hint">{{ t('schedules.form.cronHint') }}</p>
        </div>

        <!-- Cron Examples -->
        <div class="form-group">
          <label class="form-label">
            {{ t('schedules.cronExamples.title') }}
          </label>
          <div class="cron-examples">
            <button
              v-for="example in cronExamples"
              :key="example.value"
              type="button"
              class="cron-example-chip"
              :disabled="readonly"
              @click="updateField('cron_expression', example.value)"
            >
              {{ example.label }}
            </button>
          </div>
        </div>

        <div class="form-group">
          <label class="form-label">
            {{ t('schedules.form.timezone') }}
          </label>
          <select
            :value="scheduleConfig.timezone"
            class="form-input"
            :disabled="readonly"
            @change="updateField('timezone', ($event.target as HTMLSelectElement).value)"
          >
            <option v-for="tz in timezoneOptions" :key="tz" :value="tz">{{ tz }}</option>
          </select>
        </div>
      </div>

      <!-- Slack Config -->
      <div v-else-if="localTriggerType === 'slack'" class="trigger-form-fields">
        <div class="form-group">
          <label class="form-label">
            {{ t('trigger.slack.eventTypes') }}
          </label>
          <div class="checkbox-group">
            <label
              v-for="option in slackEventOptions"
              :key="option.value"
              class="checkbox-item"
            >
              <input
                type="checkbox"
                class="checkbox-input"
                :checked="(slackConfig.event_types || []).includes(option.value)"
                :disabled="readonly"
                @change="() => {
                  const current = slackConfig.event_types || []
                  const newValue = current.includes(option.value)
                    ? current.filter(v => v !== option.value)
                    : [...current, option.value]
                  updateField('event_types', newValue)
                }"
              >
              <span class="checkbox-label">{{ option.label }}</span>
            </label>
          </div>
        </div>

        <div class="form-group">
          <label class="form-label">
            {{ t('trigger.slack.channelFilter') }}
          </label>
          <input
            :value="(slackConfig.channel_filter || []).join(', ')"
            type="text"
            class="form-input"
            :placeholder="t('trigger.slack.channelFilterPlaceholder')"
            :disabled="readonly"
            @input="updateField('channel_filter', ($event.target as HTMLInputElement).value.split(',').map(s => s.trim()).filter(Boolean))"
          >
          <p class="form-hint">{{ t('trigger.slack.channelFilterHint') }}</p>
        </div>
      </div>

      <!-- Email Config -->
      <div v-else-if="localTriggerType === 'email'" class="trigger-form-fields">
        <div class="form-group">
          <label class="form-label">
            {{ t('trigger.email.triggerCondition') }}
          </label>
          <input
            :value="emailConfig.trigger_condition"
            type="text"
            class="form-input"
            :placeholder="t('trigger.email.triggerConditionPlaceholder')"
            :disabled="readonly"
            @input="updateField('trigger_condition', ($event.target as HTMLInputElement).value)"
          >
          <p class="form-hint">{{ t('trigger.email.triggerConditionHint') }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.trigger-config-panel {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

/* Trigger Type Section */
.trigger-type-section {
  margin-bottom: 0.5rem;
}

.trigger-label {
  display: block;
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--color-text);
  margin-bottom: 0.5rem;
}

.trigger-options {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.trigger-option {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  width: 100%;
  padding: 0.625rem 0.75rem;
  border-radius: 6px;
  border: 1px solid var(--color-border);
  background: var(--color-background, #fff);
  cursor: pointer;
  transition: all 0.15s;
  text-align: left;
}

.trigger-option:hover:not(:disabled) {
  border-color: var(--color-border-hover, #d1d5db);
  background: var(--color-surface-hover, #f9fafb);
}

.trigger-option.selected {
  border-color: var(--color-primary);
  background: rgba(59, 130, 246, 0.05);
}

.trigger-option:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.trigger-icon {
  font-size: 1rem;
  flex-shrink: 0;
}

.trigger-option-content {
  flex: 1;
  min-width: 0;
}

.trigger-option-label {
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--color-text);
}

.trigger-option-desc {
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
  margin-top: 0.125rem;
}

.trigger-check {
  color: var(--color-primary);
  flex-shrink: 0;
}

/* Form Fields */
.trigger-form-fields {
  display: flex;
  flex-direction: column;
  gap: 0.875rem;
}

.trigger-manual-hint {
  padding: 0.75rem;
  background: var(--color-surface);
  border-radius: 6px;
  font-size: 0.8125rem;
  color: var(--color-text-secondary);
}

.trigger-manual-hint p {
  margin: 0;
}

/* Form Elements - matching PropertiesPanel */
.form-group {
  display: flex;
  flex-direction: column;
}

.form-label {
  display: block;
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--color-text);
  margin-bottom: 0.375rem;
}

.form-input {
  width: 100%;
  padding: 0.5rem 0.75rem;
  font-size: 0.8125rem;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  background: var(--color-background, #fff);
  color: var(--color-text);
  transition: border-color 0.15s;
}

.form-input:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.form-input:disabled {
  background: var(--color-surface);
  cursor: not-allowed;
}

.code-input {
  font-family: 'SF Mono', Monaco, 'Cascadia Code', monospace;
  font-size: 0.75rem;
}

.form-hint {
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
  margin-top: 0.25rem;
  margin-bottom: 0;
}

/* Cron Examples */
.cron-examples {
  display: flex;
  flex-wrap: wrap;
  gap: 0.375rem;
}

.cron-example-chip {
  padding: 0.25rem 0.5rem;
  font-size: 0.6875rem;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  background: var(--color-background, #fff);
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.15s;
}

.cron-example-chip:hover:not(:disabled) {
  background: var(--color-surface);
  border-color: var(--color-border-hover, #d1d5db);
  color: var(--color-text);
}

.cron-example-chip:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

/* Checkbox Group */
.checkbox-group {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.checkbox-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
}

.checkbox-input {
  width: 1rem;
  height: 1rem;
  border-radius: 4px;
  accent-color: var(--color-primary);
}

.checkbox-label {
  font-size: 0.8125rem;
  color: var(--color-text);
}
</style>
