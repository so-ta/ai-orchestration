<script setup lang="ts">
const { t } = useI18n()

// Settings page
const activeTab = ref('general')

// General settings
const generalSettings = reactive({
  timezone: 'Asia/Tokyo',
  dateFormat: 'YYYY-MM-DD',
})

// Notification settings
const notificationSettings = reactive({
  emailOnFailure: true,
  emailOnSuccess: false,
  slackWebhook: '',
})

// API Keys (masked)
const apiKeys = reactive({
  openai: '',
  anthropic: '',
})

const saving = ref(false)
const message = ref<{ type: 'success' | 'error'; text: string } | null>(null)

async function saveSettings() {
  saving.value = true
  message.value = null

  // Simulate save - in real implementation, this would call the API
  await new Promise(resolve => setTimeout(resolve, 500))

  message.value = {
    type: 'success',
    text: t('common.success'),
  }
  saving.value = false

  // Clear message after 3 seconds
  setTimeout(() => {
    message.value = null
  }, 3000)
}

const tabs = computed(() => [
  { id: 'general', label: t('settings.general') },
  { id: 'notifications', label: t('settings.notifications') },
  { id: 'api-keys', label: t('settings.apiKeys') },
  { id: 'team', label: t('settings.team') },
])
</script>

<template>
  <div>
    <div class="flex justify-between items-center mb-4">
      <h1 style="font-size: 1.5rem; font-weight: 600;">
        {{ $t('settings.title') }}
      </h1>
    </div>

    <!-- Success/Error message -->
    <div
      v-if="message"
      :class="['card', message.type === 'success' ? 'bg-success' : 'bg-error']"
      style="padding: 0.75rem 1rem; margin-bottom: 1rem;"
    >
      {{ message.text }}
    </div>

    <div class="card">
      <!-- Tab navigation -->
      <div class="flex gap-4" style="border-bottom: 1px solid var(--color-border); padding: 0 1rem;">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          @click="activeTab = tab.id"
          :class="['tab-button', { active: activeTab === tab.id }]"
        >
          {{ tab.label }}
        </button>
      </div>

      <div style="padding: 1.5rem;">
        <!-- General Settings -->
        <div v-if="activeTab === 'general'">
          <h2 style="font-size: 1.125rem; font-weight: 600; margin-bottom: 1rem;">
            {{ $t('settings.general') }}
          </h2>

          <!-- Language Switcher -->
          <div class="form-group">
            <LanguageSwitcher />
          </div>

          <div class="form-group">
            <label class="form-label">{{ $t('settings.timezone') }}</label>
            <select v-model="generalSettings.timezone" class="form-input">
              <option value="Asia/Tokyo">Asia/Tokyo (JST)</option>
              <option value="UTC">UTC</option>
              <option value="America/New_York">America/New_York (EST)</option>
              <option value="Europe/London">Europe/London (GMT)</option>
            </select>
          </div>

          <div class="form-group">
            <label class="form-label">{{ $t('settings.dateFormat') }}</label>
            <select v-model="generalSettings.dateFormat" class="form-input">
              <option value="YYYY-MM-DD">YYYY-MM-DD</option>
              <option value="MM/DD/YYYY">MM/DD/YYYY</option>
              <option value="DD/MM/YYYY">DD/MM/YYYY</option>
            </select>
          </div>
        </div>

        <!-- Notification Settings -->
        <div v-if="activeTab === 'notifications'">
          <h2 style="font-size: 1.125rem; font-weight: 600; margin-bottom: 1rem;">
            {{ $t('settings.notificationSettings') }}
          </h2>

          <div class="form-group">
            <label class="checkbox-label">
              <input type="checkbox" v-model="notificationSettings.emailOnFailure" />
              {{ $t('settings.emailOnFailure') }}
            </label>
          </div>

          <div class="form-group">
            <label class="checkbox-label">
              <input type="checkbox" v-model="notificationSettings.emailOnSuccess" />
              {{ $t('settings.emailOnSuccess') }}
            </label>
          </div>

          <div class="form-group">
            <label class="form-label">{{ $t('settings.slackWebhook') }}</label>
            <input
              type="text"
              v-model="notificationSettings.slackWebhook"
              class="form-input"
              placeholder="https://hooks.slack.com/services/..."
            />
            <p class="text-secondary" style="font-size: 0.875rem; margin-top: 0.25rem;">
              {{ $t('settings.slackWebhookHint') }}
            </p>
          </div>
        </div>

        <!-- API Keys -->
        <div v-if="activeTab === 'api-keys'">
          <h2 style="font-size: 1.125rem; font-weight: 600; margin-bottom: 1rem;">
            {{ $t('settings.apiKeys') }}
          </h2>
          <p class="text-secondary" style="margin-bottom: 1rem;">
            {{ $t('settings.apiKeysDesc') }}
          </p>

          <div class="form-group">
            <label class="form-label">{{ $t('settings.openaiApiKey') }}</label>
            <input
              type="password"
              v-model="apiKeys.openai"
              class="form-input"
              placeholder="sk-..."
            />
          </div>

          <div class="form-group">
            <label class="form-label">{{ $t('settings.anthropicApiKey') }}</label>
            <input
              type="password"
              v-model="apiKeys.anthropic"
              class="form-input"
              placeholder="sk-ant-..."
            />
          </div>

          <div class="card" style="background: var(--color-warning-bg); padding: 1rem; margin-top: 1rem;">
            <p style="color: var(--color-warning); font-size: 0.875rem;">
              {{ $t('settings.apiKeysDevNote') }}
            </p>
          </div>
        </div>

        <!-- Team Settings -->
        <div v-if="activeTab === 'team'">
          <h2 style="font-size: 1.125rem; font-weight: 600; margin-bottom: 1rem;">
            {{ $t('settings.teamMembers') }}
          </h2>

          <div class="card" style="padding: 2rem; text-align: center;">
            <p class="text-secondary">
              {{ $t('settings.teamComingSoon') }}
            </p>
            <p class="text-secondary" style="margin-top: 0.5rem;">
              {{ $t('settings.teamComingSoonDesc') }}
            </p>
          </div>
        </div>

        <!-- Save button -->
        <div style="margin-top: 1.5rem; padding-top: 1.5rem; border-top: 1px solid var(--color-border);">
          <button
            @click="saveSettings"
            class="btn btn-primary"
            :disabled="saving"
          >
            {{ saving ? $t('common.saving') : $t('common.save') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.tab-button {
  padding: 0.75rem 0;
  background: none;
  border: none;
  color: var(--color-text-secondary);
  cursor: pointer;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
  transition: all 0.2s;
}

.tab-button:hover {
  color: var(--color-text);
}

.tab-button.active {
  color: var(--color-primary);
  border-bottom-color: var(--color-primary);
}

.form-group {
  margin-bottom: 1rem;
}

.form-label {
  display: block;
  font-weight: 500;
  margin-bottom: 0.5rem;
}

.form-input {
  width: 100%;
  max-width: 400px;
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--color-border);
  border-radius: 0.375rem;
  background: var(--color-bg);
  color: var(--color-text);
}

.form-input:focus {
  outline: none;
  border-color: var(--color-primary);
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
}

.checkbox-label input[type="checkbox"] {
  width: 1rem;
  height: 1rem;
}

.bg-success {
  background: rgba(34, 197, 94, 0.1);
  color: #22c55e;
}

.bg-error {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
}
</style>
