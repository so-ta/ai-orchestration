<script setup lang="ts">
const { t } = useI18n()

const emit = defineEmits<{
  (e: 'save'): void
}>()

const generalSettings = reactive({
  timezone: 'Asia/Tokyo',
  dateFormat: 'YYYY-MM-DD',
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
      <LanguageSwitcher />
    </div>

    <div class="form-group">
      <label class="form-label">{{ t('settings.timezone') }}</label>
      <select v-model="generalSettings.timezone" class="form-input">
        <option value="Asia/Tokyo">Asia/Tokyo (JST)</option>
        <option value="UTC">UTC</option>
        <option value="America/New_York">America/New_York (EST)</option>
        <option value="Europe/London">Europe/London (GMT)</option>
      </select>
    </div>

    <div class="form-group">
      <label class="form-label">{{ t('settings.dateFormat') }}</label>
      <select v-model="generalSettings.dateFormat" class="form-input">
        <option value="YYYY-MM-DD">YYYY-MM-DD</option>
        <option value="MM/DD/YYYY">MM/DD/YYYY</option>
        <option value="DD/MM/YYYY">DD/MM/YYYY</option>
      </select>
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

.form-actions {
  margin-top: 1.5rem;
  padding-top: 1rem;
  border-top: 1px solid var(--color-border);
}
</style>
