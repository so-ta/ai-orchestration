<script setup lang="ts">
import type { Credential, CredentialType, CredentialStatus, CreateCredentialRequest, CredentialData } from '~/types/api'
import type { OAuth2ProviderWithStatus, CredentialScope, StartAuthorizationRequest } from '~/types/oauth2'

const props = defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const { t } = useI18n()
const toast = useToast()
const { isAdmin } = useAuth()

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

// ============================================================================
// OAuth2 Connections Tab
// ============================================================================
const oauth2Api = useOAuth2()
const oauth2Providers = ref<OAuth2ProviderWithStatus[]>([])
const oauth2Loading = ref(false)
const showOAuth2Modal = ref(false)
const selectedProvider = ref<OAuth2ProviderWithStatus | null>(null)
const oauth2FormData = reactive({
  name: '',
  scope: 'personal' as CredentialScope,
  projectId: '',
  selectedScopes: [] as string[],
})

async function fetchOAuth2Providers() {
  oauth2Loading.value = true
  try {
    await oauth2Api.fetchProviders()
    oauth2Providers.value = oauth2Api.providers.value
  } catch {
    toast.error(t('oauth.errors.unknown'))
  } finally {
    oauth2Loading.value = false
  }
}

function openConnectOAuth2Modal(provider: OAuth2ProviderWithStatus) {
  if (!provider.app_configured) {
    toast.error(t('oauth.configureAppDesc'))
    return
  }
  selectedProvider.value = provider
  oauth2FormData.name = `${provider.name} Connection`
  oauth2FormData.scope = 'personal'
  oauth2FormData.projectId = ''
  oauth2FormData.selectedScopes = [...(provider.default_scopes || [])]
  showOAuth2Modal.value = true
}

async function startOAuth2Authorization() {
  if (!selectedProvider.value) return

  oauth2Loading.value = true
  try {
    const request: StartAuthorizationRequest = {
      provider_slug: selectedProvider.value.slug,
      name: oauth2FormData.name,
      scope: oauth2FormData.scope,
      scopes: oauth2FormData.selectedScopes,
    }
    if (oauth2FormData.scope === 'project' && oauth2FormData.projectId) {
      request.project_id = oauth2FormData.projectId
    }

    const response = await oauth2Api.startAuthorization(request)
    // Redirect to OAuth provider
    window.location.href = response.authorization_url
  } catch {
    toast.error(t('oauth.errors.authorizationFailed'))
  } finally {
    oauth2Loading.value = false
  }
}

function getProviderIcon(slug: string): string {
  const icons: Record<string, string> = {
    google: 'https://www.google.com/favicon.ico',
    github: 'https://github.githubassets.com/favicons/favicon.svg',
    slack: 'https://a.slack-edge.com/80588/marketing/img/meta/favicon-32.png',
    microsoft: 'https://www.microsoft.com/favicon.ico',
    notion: 'https://www.notion.so/images/favicon.ico',
    discord: 'https://discord.com/assets/favicon.ico',
    linear: 'https://linear.app/favicon.ico',
    atlassian: 'https://wac-cdn.atlassian.com/assets/img/favicons/atlassian/favicon.png',
  }
  return icons[slug] || ''
}

const scopeOptions = computed(() => [
  { value: 'personal', label: t('oauth.scope.personal'), desc: t('oauth.scope.personalDesc') },
  { value: 'project', label: t('oauth.scope.project'), desc: t('oauth.scope.projectDesc') },
  { value: 'organization', label: t('oauth.scope.organization'), desc: t('oauth.scope.organizationDesc') },
])

const tabs = computed(() => [
  { id: 'general', label: t('settings.general') },
  { id: 'notifications', label: t('settings.notifications') },
  { id: 'credentials', label: t('credentials.title') },
  { id: 'oauth2', label: t('oauth.title') },
])

// Watch for tab change to load credentials or OAuth2 providers
watch(activeTab, (newTab) => {
  if (newTab === 'credentials') {
    fetchCredentials()
  } else if (newTab === 'oauth2') {
    fetchOAuth2Providers()
  }
})

// Watch for modal open
watch(() => props.show, (isOpen) => {
  if (isOpen) {
    if (activeTab.value === 'credentials') {
      fetchCredentials()
    } else if (activeTab.value === 'oauth2') {
      fetchOAuth2Providers()
    }
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

        <!-- OAuth2 Connections Tab -->
        <div v-if="activeTab === 'oauth2'" class="tab-panel">
          <div class="section-header">
            <div>
              <p class="section-desc">{{ t('oauth.subtitle') }}</p>
            </div>
            <NuxtLink v-if="isAdmin()" to="/admin/oauth2-apps" class="btn btn-secondary btn-sm" @click="emit('close')">
              {{ t('oauth.app.title') }}
            </NuxtLink>
          </div>

          <div v-if="oauth2Loading && oauth2Providers.length === 0" class="empty-state">
            <p>{{ t('common.loading') }}</p>
          </div>

          <div v-else-if="oauth2Providers.length === 0" class="empty-state">
            <p>{{ t('oauth.selectProvider') }}</p>
          </div>

          <div v-else class="oauth2-grid">
            <div
              v-for="provider in oauth2Providers"
              :key="provider.slug"
              class="oauth2-provider-card"
              :class="{ 'not-configured': !provider.app_configured }"
              @click="openConnectOAuth2Modal(provider)"
            >
              <div class="oauth2-provider-icon">
                <img
                  v-if="getProviderIcon(provider.slug)"
                  :src="getProviderIcon(provider.slug)"
                  :alt="provider.name"
                  class="provider-logo"
                >
                <span v-else class="provider-fallback">{{ provider.name[0] }}</span>
              </div>
              <div class="oauth2-provider-info">
                <span class="oauth2-provider-name">{{ provider.name }}</span>
                <span
                  v-if="provider.app_configured"
                  class="oauth2-status configured"
                >{{ t('oauth.configured') }}</span>
                <span v-else class="oauth2-status not-configured">{{ t('oauth.notConfigured') }}</span>
              </div>
              <div class="oauth2-connect-arrow">
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="9 18 15 12 9 6" />
                </svg>
              </div>
            </div>
          </div>

          <!-- Admin hint for unconfigured providers -->
          <div v-if="isAdmin() && oauth2Providers.some(p => !p.app_configured)" class="admin-hint">
            <p>{{ t('oauth.configureAppDesc') }}</p>
            <NuxtLink to="/admin/oauth2-apps" class="admin-link" @click="emit('close')">
              {{ t('oauth.configureApp') }} →
            </NuxtLink>
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

    <!-- OAuth2 Connect Modal -->
    <UiModal
      :show="showOAuth2Modal"
      :title="t('oauth.connectService')"
      size="md"
      @close="showOAuth2Modal = false"
    >
      <div v-if="selectedProvider" class="oauth2-connect-form">
        <div class="provider-header">
          <div class="oauth2-provider-icon large">
            <img
              v-if="getProviderIcon(selectedProvider.slug)"
              :src="getProviderIcon(selectedProvider.slug)"
              :alt="selectedProvider.name"
              class="provider-logo"
            >
            <span v-else class="provider-fallback">{{ selectedProvider.name[0] }}</span>
          </div>
          <div class="provider-title">
            <h3>{{ selectedProvider.name }}</h3>
            <p v-if="selectedProvider.description" class="text-secondary">{{ selectedProvider.description }}</p>
          </div>
        </div>

        <form @submit.prevent="startOAuth2Authorization">
          <div class="form-group">
            <label class="form-label">{{ t('oauth.form.credentialName') }} *</label>
            <input
              v-model="oauth2FormData.name"
              type="text"
              class="form-input"
              :placeholder="t('oauth.form.credentialNamePlaceholder')"
              required
            >
          </div>

          <div class="form-group">
            <label class="form-label">{{ t('oauth.form.selectScope') }}</label>
            <div class="scope-options">
              <label
                v-for="option in scopeOptions"
                :key="option.value"
                class="scope-option"
                :class="{ selected: oauth2FormData.scope === option.value }"
              >
                <input
                  v-model="oauth2FormData.scope"
                  type="radio"
                  :value="option.value"
                  class="scope-radio"
                >
                <div class="scope-content">
                  <span class="scope-label">{{ option.label }}</span>
                  <span class="scope-desc">{{ option.desc }}</span>
                </div>
              </label>
            </div>
          </div>

          <div v-if="selectedProvider.available_scopes && selectedProvider.available_scopes.length > 0" class="form-group">
            <label class="form-label">{{ t('oauth.form.selectPermissions') }}</label>
            <div class="scope-checkboxes">
              <label
                v-for="scope in selectedProvider.available_scopes"
                :key="scope"
                class="scope-checkbox"
              >
                <input
                  v-model="oauth2FormData.selectedScopes"
                  type="checkbox"
                  :value="scope"
                >
                <span>{{ scope }}</span>
              </label>
            </div>
          </div>
        </form>
      </div>

      <template #footer>
        <button class="btn btn-secondary" @click="showOAuth2Modal = false">
          {{ t('common.cancel') }}
        </button>
        <button class="btn btn-primary" :disabled="oauth2Loading" @click="startOAuth2Authorization">
          {{ oauth2Loading ? t('common.loading') : t('oauth.form.connect') }}
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

/* OAuth2 Tab Styles */
.oauth2-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
  gap: 0.75rem;
}

.oauth2-provider-card {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 1rem;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  cursor: pointer;
  transition: all 0.2s;
}

.oauth2-provider-card:hover {
  border-color: var(--color-primary);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
}

.oauth2-provider-card.not-configured {
  opacity: 0.6;
}

.oauth2-provider-card.not-configured:hover {
  border-color: var(--color-border);
  cursor: not-allowed;
}

.oauth2-provider-icon {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-background);
  border-radius: var(--radius);
  flex-shrink: 0;
}

.oauth2-provider-icon.large {
  width: 48px;
  height: 48px;
}

.provider-logo {
  width: 24px;
  height: 24px;
  object-fit: contain;
}

.oauth2-provider-icon.large .provider-logo {
  width: 32px;
  height: 32px;
}

.provider-fallback {
  font-size: 1.25rem;
  font-weight: 600;
  color: var(--color-primary);
}

.oauth2-provider-info {
  flex: 1;
  min-width: 0;
}

.oauth2-provider-name {
  display: block;
  font-weight: 500;
  color: var(--color-text);
}

.oauth2-status {
  display: inline-block;
  font-size: 0.6875rem;
  padding: 0.125rem 0.375rem;
  border-radius: 9999px;
  margin-top: 0.25rem;
}

.oauth2-status.configured {
  background: rgba(34, 197, 94, 0.1);
  color: #22c55e;
}

.oauth2-status.not-configured {
  background: rgba(156, 163, 175, 0.1);
  color: var(--color-text-secondary);
}

.oauth2-connect-arrow {
  color: var(--color-text-secondary);
}

.admin-hint {
  margin-top: 1.5rem;
  padding: 1rem;
  background: #fef3c7;
  border: 1px solid #fcd34d;
  border-radius: var(--radius);
}

.admin-hint p {
  color: #92400e;
  font-size: 0.875rem;
  margin: 0 0 0.5rem;
}

.admin-link {
  color: #92400e;
  font-size: 0.875rem;
  font-weight: 500;
  text-decoration: none;
}

.admin-link:hover {
  text-decoration: underline;
}

/* OAuth2 Connect Modal Styles */
.oauth2-connect-form {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.provider-header {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid var(--color-border);
}

.provider-title h3 {
  margin: 0;
  font-size: 1.125rem;
  font-weight: 600;
}

.provider-title p {
  margin: 0.25rem 0 0;
  font-size: 0.875rem;
}

.scope-options {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.scope-option {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
  padding: 0.75rem;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  cursor: pointer;
  transition: all 0.15s;
}

.scope-option:hover {
  background: var(--color-surface);
}

.scope-option.selected {
  border-color: var(--color-primary);
  background: rgba(59, 130, 246, 0.05);
}

.scope-radio {
  margin-top: 0.125rem;
}

.scope-content {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
}

.scope-label {
  font-weight: 500;
  font-size: 0.875rem;
}

.scope-desc {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

.scope-checkboxes {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.scope-checkbox {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.375rem 0.625rem;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  font-size: 0.75rem;
  cursor: pointer;
}

.scope-checkbox:hover {
  border-color: var(--color-primary);
}

.scope-checkbox input:checked + span {
  color: var(--color-primary);
  font-weight: 500;
}
</style>
