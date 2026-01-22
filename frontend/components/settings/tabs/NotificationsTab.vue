<script setup lang="ts">
const { t } = useI18n()

const emit = defineEmits<{
  (e: 'save'): void
}>()

const notificationSettings = reactive({
  emailOnFailure: true,
  emailOnSuccess: false,
  slackWebhook: '',
})

const saving = ref(false)

async function saveSettings() {
  saving.value = true
  await new Promise(resolve => setTimeout(resolve, 500))
  saving.value = false
  emit('save')
}
</script>

<template>
  <div class="tab-panel">
    <div class="form-group">
      <label class="checkbox-label">
        <input v-model="notificationSettings.emailOnFailure" type="checkbox">
        {{ t('settings.emailOnFailure') }}
      </label>
    </div>

    <div class="form-group">
      <label class="checkbox-label">
        <input v-model="notificationSettings.emailOnSuccess" type="checkbox">
        {{ t('settings.emailOnSuccess') }}
      </label>
    </div>

    <div class="form-group">
      <label class="form-label">{{ t('settings.slackWebhook') }}</label>
      <input
        v-model="notificationSettings.slackWebhook"
        type="text"
        class="form-input"
        placeholder="https://hooks.slack.com/services/..."
      >
      <p class="form-hint">{{ t('settings.slackWebhookHint') }}</p>
    </div>

    <div class="form-actions">
      <button class="btn btn-primary" :disabled="saving" @click="saveSettings">
        {{ saving ? t('common.saving') : t('common.save') }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.tab-panel {
  padding: 0.5rem 0;
}

.form-group {
  margin-bottom: 1rem;
}

.form-label {
  display: block;
  font-weight: 500;
  margin-bottom: 0.5rem;
  font-size: 0.875rem;
}

.form-input {
  width: 100%;
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  background: var(--color-bg);
  color: var(--color-text);
  font-size: 0.875rem;
}

.form-input:focus {
  outline: none;
  border-color: var(--color-primary);
}

.form-hint {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  margin-top: 0.25rem;
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
  font-size: 0.875rem;
}

.checkbox-label input[type="checkbox"] {
  width: 1rem;
  height: 1rem;
}

.form-actions {
  margin-top: 1.5rem;
  padding-top: 1rem;
  border-top: 1px solid var(--color-border);
}
</style>
