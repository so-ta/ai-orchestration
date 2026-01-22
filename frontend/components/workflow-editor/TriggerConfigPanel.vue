<script setup lang="ts">
/**
 * TriggerConfigPanel.vue
 * Startブロックのトリガー設定UIコンポーネント
 *
 * トリガータイプ選択とタイプ別の設定フォームを提供
 */

const { t } = useI18n()
const toast = useToast()

type StartTriggerType = 'manual' | 'webhook' | 'schedule' | 'slack' | 'email'

interface WebhookConfig {
  secret?: string
  input_mapping?: Record<string, string>
  allowed_ips?: string[]
  enabled?: boolean
}

interface ScheduleConfig {
  cron_expression?: string
  timezone?: string
  input_data?: Record<string, unknown>
  enabled?: boolean
}

interface SlackConfig {
  event_types?: string[]
  channel_filter?: string[]
  enabled?: boolean
}

interface EmailConfig {
  trigger_condition?: string
  input_mapping?: Record<string, string>
  enabled?: boolean
}

type TriggerConfig = WebhookConfig | ScheduleConfig | SlackConfig | EmailConfig | Record<string, unknown>

const props = defineProps<{
  triggerType?: StartTriggerType
  triggerConfig?: TriggerConfig
  stepId?: string
  workflowId?: string
  readonly?: boolean
  /** Block definition's fixed trigger type (from config_defaults) - when set, type selection is disabled */
  fixedTriggerType?: StartTriggerType
}>()

// Steps API for trigger enable/disable
const stepsApi = useSteps()

// Trigger enabled state
const triggerEnabled = ref(false)
const togglingTrigger = ref(false)

// Initialize trigger enabled state from config
watch(() => props.triggerConfig, (config) => {
  if (config && 'enabled' in config) {
    triggerEnabled.value = !!config.enabled
  }
}, { immediate: true })

// Check if trigger can be enabled/disabled (not for manual triggers)
const canToggleTrigger = computed(() => {
  const type = props.fixedTriggerType || props.triggerType
  return type && type !== 'manual'
})

// Handle trigger toggle
async function handleTriggerToggle(enabled: boolean) {
  if (!props.workflowId || !props.stepId) return

  togglingTrigger.value = true
  try {
    await stepsApi.toggleTrigger(props.workflowId, props.stepId, enabled)
    triggerEnabled.value = enabled
    // Also update local config
    updateField('enabled', enabled)
    toast.success(enabled ? t('trigger.enabled.enabledMessage') : t('trigger.enabled.disabledMessage'))
  } catch {
    toast.error(enabled ? t('trigger.enabled.enableFailed') : t('trigger.enabled.disableFailed'))
    // Revert the toggle
    triggerEnabled.value = !enabled
  } finally {
    togglingTrigger.value = false
  }
}

// Runtime config for API base URL
const runtimeConfig = useRuntimeConfig()

// Generate Webhook URL
const webhookUrl = computed(() => {
  if (!props.workflowId || !props.stepId) return null
  const baseUrl = runtimeConfig.public.apiBase || window.location.origin
  return `${baseUrl}/projects/${props.workflowId}/webhook/${props.stepId}`
})

// Copy webhook URL to clipboard
const copied = ref(false)
async function copyWebhookUrl() {
  if (!webhookUrl.value) return
  try {
    await navigator.clipboard.writeText(webhookUrl.value)
    copied.value = true
    setTimeout(() => { copied.value = false }, 2000)
  } catch {
    // Fallback for older browsers
    const textarea = document.createElement('textarea')
    textarea.value = webhookUrl.value
    document.body.appendChild(textarea)
    textarea.select()
    document.execCommand('copy')
    document.body.removeChild(textarea)
    copied.value = true
    setTimeout(() => { copied.value = false }, 2000)
  }
}

const emit = defineEmits<{
  (e: 'update:trigger', data: { trigger_type: StartTriggerType; trigger_config: TriggerConfig }): void
}>()

// Whether the trigger type is fixed by block definition (inherited blocks)
const isTypeFixed = computed(() => !!props.fixedTriggerType)

// Get the effective trigger type (fixed type takes precedence)
const effectiveTriggerType = computed(() => props.fixedTriggerType || props.triggerType || 'manual')

// Local state
const localTriggerType = ref<StartTriggerType>(effectiveTriggerType.value)
const localConfig = ref<TriggerConfig>({ ...(props.triggerConfig || {}) })

// Watch for prop changes (fixed type takes precedence)
watch([() => props.fixedTriggerType, () => props.triggerType], ([fixedType, propType]) => {
  const newVal = fixedType || propType
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
  { value: 'manual', label: t('trigger.type.manual'), icon: '👤', description: t('trigger.type.manualDesc') },
  { value: 'webhook', label: t('trigger.type.webhook'), icon: '↗', description: t('trigger.type.webhookDesc') },
  { value: 'schedule', label: t('trigger.type.schedule'), icon: '⏰', description: t('trigger.type.scheduleDesc') },
  { value: 'slack', label: t('trigger.type.slack'), icon: '#', description: t('trigger.type.slackDesc') },
  { value: 'email', label: t('trigger.type.email'), icon: '✉', description: t('trigger.type.emailDesc') },
]

// Get current trigger type's option info
const currentTriggerOption = computed(() => {
  return triggerTypeOptions.find(opt => opt.value === localTriggerType.value) || triggerTypeOptions[0]
})

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
  { value: 'message', label: t('trigger.config.slack.eventTypes.message') },
  { value: 'reaction_added', label: t('trigger.config.slack.eventTypes.reaction') },
  { value: 'app_mention', label: t('trigger.config.slack.eventTypes.mention') },
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
    <!-- Trigger Type Display (Fixed) or Selection -->
    <div class="trigger-type-section">
      <label class="trigger-label">
        {{ t('trigger.type.label') }}
      </label>

      <!-- Fixed trigger type display (read-only badge) -->
      <div v-if="isTypeFixed" class="trigger-type-fixed">
        <div class="trigger-type-badge">
          <span class="trigger-icon">{{ currentTriggerOption.icon }}</span>
          <div class="trigger-type-info">
            <div class="trigger-type-name">{{ currentTriggerOption.label }}</div>
            <div class="trigger-type-desc">{{ currentTriggerOption.description }}</div>
          </div>
        </div>
      </div>

      <!-- Selectable trigger type options (for generic start block) -->
      <div v-else class="trigger-options">
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

    <!-- Trigger Enable/Disable Toggle (for non-manual triggers) -->
    <div v-if="canToggleTrigger && stepId && workflowId" class="trigger-enabled-section">
      <div class="trigger-enabled-toggle">
        <label class="toggle-container">
          <input
            type="checkbox"
            :checked="triggerEnabled"
            :disabled="readonly || togglingTrigger"
            @change="handleTriggerToggle(($event.target as HTMLInputElement).checked)"
          >
          <span class="toggle-slider" :class="{ loading: togglingTrigger }"></span>
        </label>
        <div class="toggle-label-content">
          <span class="toggle-label">{{ t('trigger.enabled.label') }}</span>
          <span class="toggle-description">{{ t('trigger.enabled.description') }}</span>
        </div>
        <span v-if="triggerEnabled" class="status-badge enabled">{{ t('common.enabled') }}</span>
        <span v-else class="status-badge disabled">{{ t('common.disabled') }}</span>
      </div>
    </div>

    <!-- Type-specific Configuration -->
    <div class="trigger-config-form">
      <!-- Manual - No config needed -->
      <div v-if="localTriggerType === 'manual'" class="trigger-manual-hint">
        <p>{{ t('trigger.type.manualDesc') }}</p>
      </div>

      <!-- Webhook Config -->
      <div v-else-if="localTriggerType === 'webhook'" class="trigger-form-fields">
        <!-- Webhook URL Display -->
        <div v-if="webhookUrl" class="form-group">
          <label class="form-label">
            Webhook URL
          </label>
          <div class="webhook-url-container">
            <code class="webhook-url">{{ webhookUrl }}</code>
            <button
              type="button"
              class="copy-button"
              :class="{ copied }"
              @click="copyWebhookUrl"
            >
              <svg v-if="!copied" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
                <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
              </svg>
              <svg v-else xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="20 6 9 17 4 12" />
              </svg>
            </button>
          </div>
          <p class="form-hint">POST リクエストでこの URL を呼び出すとワークフローが実行されます</p>
        </div>

        <div class="form-group">
          <label class="form-label">
            {{ t('trigger.config.webhook.secret') }}
          </label>
          <input
            :value="webhookConfig.secret"
            type="text"
            class="form-input"
            :placeholder="t('trigger.config.webhook.secretPlaceholder')"
            :disabled="readonly"
            @input="updateField('secret', ($event.target as HTMLInputElement).value)"
          >
          <p class="form-hint">{{ t('trigger.config.webhook.secretHint') }}</p>
        </div>

        <div class="form-group">
          <label class="form-label">
            {{ t('trigger.config.webhook.allowedIps') }}
          </label>
          <input
            :value="(webhookConfig.allowed_ips || []).join(', ')"
            type="text"
            class="form-input"
            :placeholder="t('trigger.config.webhook.allowedIpsPlaceholder')"
            :disabled="readonly"
            @input="updateField('allowed_ips', ($event.target as HTMLInputElement).value.split(',').map(s => s.trim()).filter(Boolean))"
          >
          <p class="form-hint">{{ t('trigger.config.webhook.allowedIpsHint') }}</p>
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
            {{ t('trigger.config.slack.eventType') }}
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
            {{ t('trigger.config.slack.channelFilter') }}
          </label>
          <input
            :value="(slackConfig.channel_filter || []).join(', ')"
            type="text"
            class="form-input"
            :placeholder="t('trigger.config.slack.channelFilterPlaceholder')"
            :disabled="readonly"
            @input="updateField('channel_filter', ($event.target as HTMLInputElement).value.split(',').map(s => s.trim()).filter(Boolean))"
          >
          <p class="form-hint">{{ t('trigger.config.slack.channelFilterHint') }}</p>
        </div>
      </div>

      <!-- Email Config -->
      <div v-else-if="localTriggerType === 'email'" class="trigger-form-fields">
        <div class="form-group">
          <label class="form-label">
            {{ t('trigger.config.email.triggerCondition') }}
          </label>
          <input
            :value="emailConfig.trigger_condition"
            type="text"
            class="form-input"
            :placeholder="t('trigger.config.email.conditionValuePlaceholder')"
            :disabled="readonly"
            @input="updateField('trigger_condition', ($event.target as HTMLInputElement).value)"
          >
          <p class="form-hint">{{ t('trigger.config.email.inputMappingHint') }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.trigger-config-panel {
  display: flex;
  flex-direction: column;
}

/* Trigger Enable/Disable Toggle */
.trigger-enabled-section {
  margin-bottom: 1rem;
  padding: 0.75rem;
  background: var(--color-surface, #f9fafb);
  border: 1px solid var(--color-border);
  border-radius: 8px;
}

.trigger-enabled-toggle {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.toggle-container {
  position: relative;
  display: inline-block;
  width: 44px;
  height: 24px;
  flex-shrink: 0;
}

.toggle-container input {
  opacity: 0;
  width: 0;
  height: 0;
}

.toggle-slider {
  position: absolute;
  cursor: pointer;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: var(--color-border, #d1d5db);
  transition: 0.2s;
  border-radius: 24px;
}

.toggle-slider::before {
  position: absolute;
  content: "";
  height: 18px;
  width: 18px;
  left: 3px;
  bottom: 3px;
  background-color: white;
  transition: 0.2s;
  border-radius: 50%;
}

.toggle-slider.loading::before {
  animation: pulse 1s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.toggle-container input:checked + .toggle-slider {
  background-color: var(--color-primary, #3b82f6);
}

.toggle-container input:checked + .toggle-slider::before {
  transform: translateX(20px);
}

.toggle-container input:disabled + .toggle-slider {
  opacity: 0.6;
  cursor: not-allowed;
}

.toggle-label-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
}

.toggle-label {
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--color-text);
}

.toggle-description {
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
}

.status-badge {
  padding: 0.25rem 0.5rem;
  font-size: 0.6875rem;
  font-weight: 500;
  border-radius: 4px;
  flex-shrink: 0;
}

.status-badge.enabled {
  background: rgba(16, 185, 129, 0.1);
  color: var(--color-success, #10b981);
}

.status-badge.disabled {
  background: rgba(107, 114, 128, 0.1);
  color: var(--color-text-secondary, #6b7280);
}

/* Trigger Type Section - matches PropertiesPanel .form-group */
.trigger-type-section {
  margin-bottom: 0.875rem;
}

/* Label - matches PropertiesPanel .form-label */
.trigger-label {
  display: block;
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--color-text);
  margin-bottom: 0.375rem;
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

/* Fixed trigger type display (read-only) */
.trigger-type-fixed {
  margin-bottom: 0.5rem;
}

.trigger-type-badge {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.625rem 0.75rem;
  border-radius: 6px;
  border: 1px solid var(--color-border);
  background: var(--color-surface, #f9fafb);
}

.trigger-type-badge .trigger-icon {
  font-size: 1rem;
  flex-shrink: 0;
}

.trigger-type-info {
  flex: 1;
  min-width: 0;
}

.trigger-type-name {
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--color-text);
}

.trigger-type-desc {
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
  margin-top: 0.125rem;
}

/* Webhook URL Display */
.webhook-url-container {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
  background: var(--color-surface, #f9fafb);
  border: 1px solid var(--color-border);
  border-radius: 6px;
}

.webhook-url {
  flex: 1;
  font-family: 'SF Mono', Monaco, 'Cascadia Code', monospace;
  font-size: 0.75rem;
  color: var(--color-text);
  word-break: break-all;
  background: transparent;
  padding: 0;
}

.copy-button {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  padding: 0;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  background: var(--color-background, #fff);
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.15s;
  flex-shrink: 0;
}

.copy-button:hover {
  border-color: var(--color-border-hover, #d1d5db);
  color: var(--color-text);
}

.copy-button.copied {
  border-color: var(--color-success, #10B981);
  color: var(--color-success, #10B981);
  background: rgba(16, 185, 129, 0.05);
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
