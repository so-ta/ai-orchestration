<script setup lang="ts">
import type { Credential, CredentialType, CredentialStatus, CreateCredentialRequest, CredentialData } from '~/types/api'

const props = defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const { t } = useI18n()
const toast = useToast()

const activeTab = ref('general')

// ============================================================================
// General Settings
// ============================================================================
const generalSettings = reactive({
  timezone: 'Asia/Tokyo',
  dateFormat: 'YYYY-MM-DD',
})

// ============================================================================
// Notification Settings
// ============================================================================
const notificationSettings = reactive({
  emailOnFailure: true,
  emailOnSuccess: false,
  slackWebhook: '',
})

const saving = ref(false)

async function saveSettings() {
  saving.value = true
  await new Promise(resolve => setTimeout(resolve, 500))
  toast.success(t('common.success'))
  saving.value = false
}

// ============================================================================
// Credentials Tab
// ============================================================================
const credentialsApi = useCredentials()
const credentialsLoading = ref(false)
const credentialsList = ref<Credential[]>([])
const showCredentialModal = ref(false)
const showDeleteCredentialModal = ref(false)
const selectedCredential = ref<Credential | null>(null)
const credentialFormData = reactive({
  name: '',
  description: '',
  credential_type: 'api_key' as CredentialType,
  provider: '',
  expires_at: '',
  api_key: '',
  header_name: 'Authorization',
  header_prefix: 'Bearer ',
  username: '',
  password: '',
})

async function fetchCredentials() {
  credentialsLoading.value = true
  try {
    await credentialsApi.fetchCredentials()
    credentialsList.value = credentialsApi.credentials.value
  } catch {
    toast.error(t('credentials.messages.createFailed'))
  } finally {
    credentialsLoading.value = false
  }
}

function openAddCredentialModal() {
  selectedCredential.value = null
  credentialFormData.name = ''
  credentialFormData.description = ''
  credentialFormData.credential_type = 'api_key'
  credentialFormData.provider = ''
  credentialFormData.expires_at = ''
  credentialFormData.api_key = ''
  credentialFormData.header_name = 'Authorization'
  credentialFormData.header_prefix = 'Bearer '
  credentialFormData.username = ''
  credentialFormData.password = ''
  showCredentialModal.value = true
}

function openEditCredentialModal(credential: Credential) {
  selectedCredential.value = credential
  credentialFormData.name = credential.name
  credentialFormData.description = credential.description || ''
  credentialFormData.credential_type = credential.credential_type
  credentialFormData.provider = credential.metadata?.provider || ''
  credentialFormData.expires_at = credential.expires_at || ''
  credentialFormData.api_key = ''
  credentialFormData.header_name = 'Authorization'
  credentialFormData.header_prefix = 'Bearer '
  credentialFormData.username = ''
  credentialFormData.password = ''
  showCredentialModal.value = true
}

async function submitCredentialForm() {
  if (!credentialFormData.name.trim()) {
    toast.error(t('common.error'))
    return
  }

  credentialsLoading.value = true
  try {
    const credentialData: CredentialData = {}

    if (credentialFormData.credential_type === 'api_key') {
      if (!selectedCredential.value && !credentialFormData.api_key) {
        toast.error(t('common.error'))
        credentialsLoading.value = false
        return
      }
      if (credentialFormData.api_key) {
        credentialData.api_key = credentialFormData.api_key
        credentialData.header_name = credentialFormData.header_name
        credentialData.header_prefix = credentialFormData.header_prefix
      }
    } else if (credentialFormData.credential_type === 'basic_auth') {
      if (!selectedCredential.value && (!credentialFormData.username || !credentialFormData.password)) {
        toast.error(t('common.error'))
        credentialsLoading.value = false
        return
      }
      if (credentialFormData.username) credentialData.username = credentialFormData.username
      if (credentialFormData.password) credentialData.password = credentialFormData.password
    }

    if (selectedCredential.value) {
      await credentialsApi.updateCredential(selectedCredential.value.id, {
        name: credentialFormData.name,
        description: credentialFormData.description || undefined,
        data: Object.keys(credentialData).length > 0 ? credentialData : undefined,
        metadata: credentialFormData.provider ? { provider: credentialFormData.provider } : undefined,
        expires_at: credentialFormData.expires_at || undefined,
      })
      toast.success(t('credentials.messages.updated'))
    } else {
      const request: CreateCredentialRequest = {
        name: credentialFormData.name,
        description: credentialFormData.description || undefined,
        credential_type: credentialFormData.credential_type,
        data: credentialData,
        metadata: credentialFormData.provider ? { provider: credentialFormData.provider } : undefined,
        expires_at: credentialFormData.expires_at || undefined,
      }
      await credentialsApi.createCredential(request)
      toast.success(t('credentials.messages.created'))
    }
    showCredentialModal.value = false
    await fetchCredentials()
  } catch {
    toast.error(selectedCredential.value ? t('credentials.messages.updateFailed') : t('credentials.messages.createFailed'))
  } finally {
    credentialsLoading.value = false
  }
}

function openDeleteCredentialModal(credential: Credential) {
  selectedCredential.value = credential
  showDeleteCredentialModal.value = true
}

async function confirmDeleteCredential() {
  if (!selectedCredential.value) return

  credentialsLoading.value = true
  try {
    await credentialsApi.deleteCredential(selectedCredential.value.id)
    toast.success(t('credentials.messages.deleted'))
    showDeleteCredentialModal.value = false
    selectedCredential.value = null
    await fetchCredentials()
  } catch {
    toast.error(t('credentials.messages.deleteFailed'))
  } finally {
    credentialsLoading.value = false
  }
}

async function revokeCredential(credential: Credential) {
  credentialsLoading.value = true
  try {
    await credentialsApi.revokeCredential(credential.id)
    toast.success(t('credentials.messages.revoked'))
    await fetchCredentials()
  } catch {
    toast.error(t('credentials.messages.revokeFailed'))
  } finally {
    credentialsLoading.value = false
  }
}

async function activateCredential(credential: Credential) {
  credentialsLoading.value = true
  try {
    await credentialsApi.activateCredential(credential.id)
    toast.success(t('credentials.messages.activated'))
    await fetchCredentials()
  } catch {
    toast.error(t('credentials.messages.activateFailed'))
  } finally {
    credentialsLoading.value = false
  }
}

function getStatusClass(status: CredentialStatus): string {
  switch (status) {
    case 'active': return 'status-active'
    case 'expired': return 'status-expired'
    case 'revoked': return 'status-revoked'
    default: return ''
  }
}

const credentialTypes = computed(() => [
  { value: 'api_key', label: t('credentials.type.api_key') },
  { value: 'basic_auth', label: t('credentials.type.basic_auth') },
  { value: 'oauth2', label: t('credentials.type.oauth2') },
  { value: 'custom', label: t('credentials.type.custom') },
])

const tabs = computed(() => [
  { id: 'general', label: t('settings.general') },
  { id: 'notifications', label: t('settings.notifications') },
  { id: 'credentials', label: t('credentials.title') },
])

// Watch for tab change to load credentials
watch(activeTab, (newTab) => {
  if (newTab === 'credentials') {
    fetchCredentials()
  }
})

// Watch for modal open
watch(() => props.show, (isOpen) => {
  if (isOpen && activeTab.value === 'credentials') {
    fetchCredentials()
  }
})
</script>

<template>
  <UiModal
    :show="show"
    :title="t('settings.title')"
    size="lg"
    @close="emit('close')"
  >
    <div class="settings-modal">
      <!-- Tab navigation -->
      <div class="tabs-nav">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          :class="['tab-button', { active: activeTab === tab.id }]"
          @click="activeTab = tab.id"
        >
          {{ tab.label }}
        </button>
      </div>

      <div class="tab-content">
        <!-- General Settings -->
        <div v-if="activeTab === 'general'" class="tab-panel">
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

        <!-- Notification Settings -->
        <div v-if="activeTab === 'notifications'" class="tab-panel">
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

        <!-- Credentials Tab -->
        <div v-if="activeTab === 'credentials'" class="tab-panel">
          <div class="section-header">
            <div>
              <p class="section-desc">{{ t('credentials.subtitle') }}</p>
            </div>
            <button class="btn btn-primary btn-sm" @click="openAddCredentialModal">
              + {{ t('credentials.newCredential') }}
            </button>
          </div>

          <!-- Security note -->
          <div class="info-card">
            <p>{{ t('credentials.securityNote') }}</p>
          </div>

          <div v-if="credentialsLoading && credentialsList.length === 0" class="empty-state">
            <p>{{ t('common.loading') }}</p>
          </div>

          <div v-else-if="credentialsList.length === 0" class="empty-state">
            <p>{{ t('credentials.noCredentials') }}</p>
            <p class="hint">{{ t('credentials.noCredentialsDesc') }}</p>
          </div>

          <div v-else class="credentials-list">
            <div v-for="credential in credentialsList" :key="credential.id" class="credential-item">
              <div class="credential-info">
                <span class="credential-name">{{ credential.name }}</span>
                <span class="credential-type">{{ t(`credentials.type.${credential.credential_type}`) }}</span>
                <span :class="['status-badge', getStatusClass(credential.status)]">
                  {{ t(`credentials.status.${credential.status}`) }}
                </span>
              </div>
              <div class="credential-actions">
                <button class="btn btn-sm btn-secondary" @click="openEditCredentialModal(credential)">
                  {{ t('common.edit') }}
                </button>
                <button
                  v-if="credential.status === 'active'"
                  class="btn btn-sm btn-warning"
                  @click="revokeCredential(credential)"
                >
                  {{ t('credentials.actions.revoke') }}
                </button>
                <button
                  v-if="credential.status === 'revoked'"
                  class="btn btn-sm btn-success"
                  @click="activateCredential(credential)"
                >
                  {{ t('credentials.actions.activate') }}
                </button>
                <button class="btn btn-sm btn-danger" @click="openDeleteCredentialModal(credential)">
                  {{ t('common.delete') }}
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <button class="btn btn-secondary" @click="emit('close')">
        {{ t('common.close') }}
      </button>
    </template>

    <!-- Credential Modal -->
    <UiModal
      :show="showCredentialModal"
      :title="selectedCredential ? t('credentials.editCredential') : t('credentials.newCredential')"
      size="md"
      @close="showCredentialModal = false"
    >
      <form @submit.prevent="submitCredentialForm">
        <div class="form-group">
          <label class="form-label">{{ t('credentials.form.name') }} *</label>
          <input
            v-model="credentialFormData.name"
            type="text"
            class="form-input"
            :placeholder="t('credentials.form.namePlaceholder')"
            required
          >
        </div>

        <div class="form-group">
          <label class="form-label">{{ t('credentials.form.description') }}</label>
          <textarea
            v-model="credentialFormData.description"
            class="form-input"
            rows="2"
            :placeholder="t('credentials.form.descriptionPlaceholder')"
          />
        </div>

        <div class="form-group">
          <label class="form-label">{{ t('credentials.form.credentialType') }}</label>
          <select v-model="credentialFormData.credential_type" class="form-input" :disabled="!!selectedCredential">
            <option v-for="ct in credentialTypes" :key="ct.value" :value="ct.value">
              {{ ct.label }}
            </option>
          </select>
        </div>

        <div class="form-group">
          <label class="form-label">{{ t('credentials.form.provider') }}</label>
          <input
            v-model="credentialFormData.provider"
            type="text"
            class="form-input"
            :placeholder="t('credentials.form.providerPlaceholder')"
          >
        </div>

        <!-- API Key fields -->
        <template v-if="credentialFormData.credential_type === 'api_key'">
          <div class="form-group">
            <label class="form-label">
              {{ t('credentials.form.apiKey') }} {{ selectedCredential ? '' : '*' }}
            </label>
            <input
              v-model="credentialFormData.api_key"
              type="password"
              class="form-input"
              :placeholder="t('credentials.form.apiKeyPlaceholder')"
              :required="!selectedCredential"
            >
            <p v-if="selectedCredential" class="form-hint">
              {{ t('credentials.form.valueHidden') }}
            </p>
          </div>

          <div class="form-row">
            <div class="form-group">
              <label class="form-label">{{ t('credentials.form.headerName') }}</label>
              <input
                v-model="credentialFormData.header_name"
                type="text"
                class="form-input"
                :placeholder="t('credentials.form.headerNamePlaceholder')"
              >
            </div>
            <div class="form-group">
              <label class="form-label">{{ t('credentials.form.headerPrefix') }}</label>
              <input
                v-model="credentialFormData.header_prefix"
                type="text"
                class="form-input"
                :placeholder="t('credentials.form.headerPrefixPlaceholder')"
              >
            </div>
          </div>
        </template>

        <!-- Basic Auth fields -->
        <template v-if="credentialFormData.credential_type === 'basic_auth'">
          <div class="form-group">
            <label class="form-label">
              {{ t('credentials.form.username') }} {{ selectedCredential ? '' : '*' }}
            </label>
            <input
              v-model="credentialFormData.username"
              type="text"
              class="form-input"
              :placeholder="t('credentials.form.usernamePlaceholder')"
              :required="!selectedCredential"
            >
          </div>

          <div class="form-group">
            <label class="form-label">
              {{ t('credentials.form.password') }} {{ selectedCredential ? '' : '*' }}
            </label>
            <input
              v-model="credentialFormData.password"
              type="password"
              class="form-input"
              :placeholder="t('credentials.form.passwordPlaceholder')"
              :required="!selectedCredential"
            >
            <p v-if="selectedCredential" class="form-hint">
              {{ t('credentials.form.valueHidden') }}
            </p>
          </div>
        </template>

        <div class="form-group">
          <label class="form-label">{{ t('credentials.form.expiresAt') }}</label>
          <input
            v-model="credentialFormData.expires_at"
            type="datetime-local"
            class="form-input"
          >
        </div>
      </form>

      <template #footer>
        <button class="btn btn-secondary" @click="showCredentialModal = false">
          {{ t('common.cancel') }}
        </button>
        <button class="btn btn-primary" :disabled="credentialsLoading" @click="submitCredentialForm">
          {{ credentialsLoading ? t('common.saving') : t('common.save') }}
        </button>
      </template>
    </UiModal>

    <!-- Delete Credential Modal -->
    <UiModal
      :show="showDeleteCredentialModal"
      :title="t('credentials.editCredential')"
      size="sm"
      @close="showDeleteCredentialModal = false"
    >
      <p>{{ t('credentials.confirmDelete') }}</p>

      <template #footer>
        <button class="btn btn-secondary" @click="showDeleteCredentialModal = false">
          {{ t('common.cancel') }}
        </button>
        <button class="btn btn-danger" :disabled="credentialsLoading" @click="confirmDeleteCredential">
          {{ t('common.delete') }}
        </button>
      </template>
    </UiModal>
  </UiModal>
</template>

<style scoped>
.settings-modal {
  min-height: 400px;
}

.tabs-nav {
  display: flex;
  gap: 0.5rem;
  border-bottom: 1px solid var(--color-border);
  margin-bottom: 1.5rem;
}

.tab-button {
  padding: 0.75rem 1rem;
  background: none;
  border: none;
  color: var(--color-text-secondary);
  cursor: pointer;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
  transition: all 0.2s;
  font-size: 0.875rem;
}

.tab-button:hover {
  color: var(--color-text);
}

.tab-button.active {
  color: var(--color-primary);
  border-bottom-color: var(--color-primary);
}

.tab-panel {
  padding: 0.5rem 0;
}

.form-group {
  margin-bottom: 1rem;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
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

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.section-desc {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
  margin: 0;
}

.info-card {
  background: #dbeafe;
  border: 1px solid #93c5fd;
  border-radius: var(--radius);
  padding: 0.75rem 1rem;
  margin-bottom: 1rem;
}

.info-card p {
  color: #1e40af;
  font-size: 0.8125rem;
  margin: 0;
}

.empty-state {
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  padding: 2rem;
  text-align: center;
}

.empty-state p {
  margin: 0;
  color: var(--color-text-secondary);
}

.empty-state .hint {
  font-size: 0.875rem;
  margin-top: 0.5rem;
}

.credentials-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.credential-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.75rem 1rem;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
}

.credential-info {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.credential-name {
  font-weight: 500;
}

.credential-type {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

.credential-actions {
  display: flex;
  gap: 0.5rem;
}

.status-badge {
  display: inline-flex;
  align-items: center;
  padding: 0.125rem 0.5rem;
  border-radius: 9999px;
  font-size: 0.6875rem;
  font-weight: 500;
}

.status-active {
  background: rgba(34, 197, 94, 0.1);
  color: #22c55e;
}

.status-revoked {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
}

.status-expired {
  background: rgba(245, 158, 11, 0.1);
  color: #f59e0b;
}

.btn-sm {
  padding: 0.375rem 0.625rem;
  font-size: 0.75rem;
}

.btn-danger {
  background: #ef4444;
  color: white;
}

.btn-danger:hover {
  background: #dc2626;
}

.btn-warning {
  background: #f59e0b;
  color: white;
}

.btn-warning:hover {
  background: #d97706;
}

.btn-success {
  background: #22c55e;
  color: white;
}

.btn-success:hover {
  background: #16a34a;
}
</style>
