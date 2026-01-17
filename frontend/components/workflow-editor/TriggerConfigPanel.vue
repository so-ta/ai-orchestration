<script setup lang="ts">
/**
 * TriggerConfigPanel.vue
 * Startブロックのトリガー設定UIコンポーネント
 *
 * トリガータイプ選択とタイプ別の設定フォームを提供
 */

import TriggerBadge from '~/components/dag-editor/TriggerBadge.vue'

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
    <!-- Current Trigger Badge -->
    <div class="mb-4">
      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
        {{ t('trigger.currentTrigger') }}
      </label>
      <TriggerBadge :trigger-type="localTriggerType" size="md" />
    </div>

    <!-- Trigger Type Selection -->
    <div class="mb-6">
      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
        {{ t('trigger.selectType') }}
      </label>
      <div class="space-y-2">
        <button
          v-for="option in triggerTypeOptions"
          :key="option.value"
          type="button"
          class="w-full flex items-center gap-3 p-3 rounded-lg border transition-colors text-left"
          :class="[
            localTriggerType === option.value
              ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20'
              : 'border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600',
          ]"
          :disabled="readonly"
          @click="handleTypeChange(option.value as StartTriggerType)"
        >
          <span class="text-lg">{{ option.icon }}</span>
          <div class="flex-1">
            <div class="font-medium text-gray-900 dark:text-gray-100">{{ option.label }}</div>
            <div class="text-sm text-gray-500 dark:text-gray-400">{{ option.description }}</div>
          </div>
          <div v-if="localTriggerType === option.value" class="text-primary-500">
            <svg class="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
              <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
            </svg>
          </div>
        </button>
      </div>
    </div>

    <!-- Type-specific Configuration -->
    <div class="trigger-config-form">
      <!-- Manual - No config needed -->
      <div v-if="localTriggerType === 'manual'" class="text-sm text-gray-500 dark:text-gray-400 p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
        <p>{{ t('trigger.manualDescription') }}</p>
      </div>

      <!-- Webhook Config -->
      <div v-else-if="localTriggerType === 'webhook'" class="space-y-4">
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            {{ t('trigger.webhook.secret') }}
          </label>
          <input
            :value="webhookConfig.secret"
            type="text"
            class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
            :placeholder="t('trigger.webhook.secretPlaceholder')"
            :disabled="readonly"
            @input="updateField('secret', ($event.target as HTMLInputElement).value)"
          >
          <p class="mt-1 text-xs text-gray-500">{{ t('trigger.webhook.secretHint') }}</p>
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            {{ t('trigger.webhook.allowedIps') }}
          </label>
          <input
            :value="(webhookConfig.allowed_ips || []).join(', ')"
            type="text"
            class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
            :placeholder="t('trigger.webhook.allowedIpsPlaceholder')"
            :disabled="readonly"
            @input="updateField('allowed_ips', ($event.target as HTMLInputElement).value.split(',').map(s => s.trim()).filter(Boolean))"
          >
          <p class="mt-1 text-xs text-gray-500">{{ t('trigger.webhook.allowedIpsHint') }}</p>
        </div>
      </div>

      <!-- Schedule Config -->
      <div v-else-if="localTriggerType === 'schedule'" class="space-y-4">
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            {{ t('schedules.form.cronExpression') }}
          </label>
          <input
            :value="scheduleConfig.cron_expression"
            type="text"
            class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 font-mono"
            placeholder="0 9 * * *"
            :disabled="readonly"
            @input="updateField('cron_expression', ($event.target as HTMLInputElement).value)"
          >
          <p class="mt-1 text-xs text-gray-500">{{ t('schedules.form.cronHint') }}</p>
        </div>

        <!-- Cron Examples -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            {{ t('schedules.cronExamples.title') }}
          </label>
          <div class="flex flex-wrap gap-2">
            <button
              v-for="example in cronExamples"
              :key="example.value"
              type="button"
              class="px-2 py-1 text-xs border border-gray-200 dark:border-gray-600 rounded hover:bg-gray-100 dark:hover:bg-gray-700"
              :disabled="readonly"
              @click="updateField('cron_expression', example.value)"
            >
              {{ example.label }}
            </button>
          </div>
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            {{ t('schedules.form.timezone') }}
          </label>
          <select
            :value="scheduleConfig.timezone"
            class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
            :disabled="readonly"
            @change="updateField('timezone', ($event.target as HTMLSelectElement).value)"
          >
            <option v-for="tz in timezoneOptions" :key="tz" :value="tz">{{ tz }}</option>
          </select>
        </div>
      </div>

      <!-- Slack Config -->
      <div v-else-if="localTriggerType === 'slack'" class="space-y-4">
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            {{ t('trigger.slack.eventTypes') }}
          </label>
          <div class="space-y-2">
            <label
              v-for="option in slackEventOptions"
              :key="option.value"
              class="flex items-center gap-2"
            >
              <input
                type="checkbox"
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
              <span class="text-sm text-gray-700 dark:text-gray-300">{{ option.label }}</span>
            </label>
          </div>
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            {{ t('trigger.slack.channelFilter') }}
          </label>
          <input
            :value="(slackConfig.channel_filter || []).join(', ')"
            type="text"
            class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
            :placeholder="t('trigger.slack.channelFilterPlaceholder')"
            :disabled="readonly"
            @input="updateField('channel_filter', ($event.target as HTMLInputElement).value.split(',').map(s => s.trim()).filter(Boolean))"
          >
          <p class="mt-1 text-xs text-gray-500">{{ t('trigger.slack.channelFilterHint') }}</p>
        </div>
      </div>

      <!-- Email Config -->
      <div v-else-if="localTriggerType === 'email'" class="space-y-4">
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            {{ t('trigger.email.triggerCondition') }}
          </label>
          <input
            :value="emailConfig.trigger_condition"
            type="text"
            class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
            :placeholder="t('trigger.email.triggerConditionPlaceholder')"
            :disabled="readonly"
            @input="updateField('trigger_condition', ($event.target as HTMLInputElement).value)"
          >
          <p class="mt-1 text-xs text-gray-500">{{ t('trigger.email.triggerConditionHint') }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.trigger-config-panel {
  @apply p-4;
}
</style>
