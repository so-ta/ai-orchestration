<script setup lang="ts">
/**
 * OAuth2 Apps Management Page
 * 管理者向けOAuth2クライアント設定管理ページ
 */
import type { OAuth2ProviderWithStatus, OAuth2App, CreateOAuth2AppRequest } from '~/types/oauth2'

const { t } = useI18n()
const toast = useToast()
const { confirm } = useConfirm()
const {
  providers,
  providersLoading,
  fetchProviders,
  apps,
  appsLoading,
  fetchApps,
  createApp,
  deleteApp
} = useOAuth2()

definePageMeta({
  layout: 'default',
  middleware: ['admin'],
})

// State
const showCreateModal = ref(false)
const selectedProvider = ref<OAuth2ProviderWithStatus | null>(null)
const formData = ref<CreateOAuth2AppRequest>({
  provider_slug: '',
  client_id: '',
  client_secret: '',
  scopes: [],
  custom_authorization_url: '',
  custom_token_url: '',
})
const creating = ref(false)

// Fetch data on mount
onMounted(async () => {
  await Promise.all([fetchProviders(), fetchApps()])
})

// Get apps for a specific provider
function getAppsForProvider(providerSlug: string): OAuth2App[] {
  return apps.value.filter(app => app.provider_slug === providerSlug)
}

// Check if provider has an app configured
function hasAppConfigured(providerSlug: string): boolean {
  return apps.value.some(app => app.provider_slug === providerSlug)
}

// Open create modal for a provider
function openCreateModal(provider: OAuth2ProviderWithStatus) {
  selectedProvider.value = provider
  formData.value = {
    provider_slug: provider.slug,
    client_id: '',
    client_secret: '',
    scopes: [...(provider.default_scopes || [])],
    custom_authorization_url: '',
    custom_token_url: '',
  }
  showCreateModal.value = true
}

// Handle create app
async function handleCreateApp() {
  if (!formData.value.client_id || !formData.value.client_secret) {
    toast.error(t('admin.oauth2Apps.validation.required'))
    return
  }

  creating.value = true
  try {
    await createApp(formData.value)
    toast.success(t('admin.oauth2Apps.createSuccess'))
    showCreateModal.value = false
  } catch {
    toast.error(t('admin.oauth2Apps.createError'))
  } finally {
    creating.value = false
  }
}

// Handle delete app
async function handleDeleteApp(app: OAuth2App) {
  const confirmed = await confirm({
    title: t('admin.oauth2Apps.deleteTitle'),
    message: t('admin.oauth2Apps.deleteMessage', { provider: app.provider_slug }),
    confirmText: t('common.delete'),
    cancelText: t('common.cancel'),
    variant: 'danger',
  })

  if (confirmed) {
    try {
      await deleteApp(app.id)
      toast.success(t('admin.oauth2Apps.deleteSuccess'))
    } catch {
      toast.error(t('admin.oauth2Apps.deleteError'))
    }
  }
}

// Toggle scope
function toggleScope(scope: string) {
  if (!formData.value.scopes) {
    formData.value.scopes = []
  }
  const index = formData.value.scopes.indexOf(scope)
  if (index >= 0) {
    formData.value.scopes.splice(index, 1)
  } else {
    formData.value.scopes.push(scope)
  }
}
</script>

<template>
  <div class="oauth2-apps-page">
    <!-- Header -->
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t('admin.oauth2Apps.title') }}</h1>
        <p class="page-subtitle">{{ t('admin.oauth2Apps.subtitle') }}</p>
      </div>
    </div>

    <!-- Loading State -->
    <div v-if="providersLoading || appsLoading" class="loading-state">
      <span class="loading-spinner" />
      <span>{{ t('common.loading') }}</span>
    </div>

    <!-- Providers Grid -->
    <div v-else class="providers-grid">
      <div
        v-for="provider in providers"
        :key="provider.slug"
        class="provider-card"
        :class="{ configured: hasAppConfigured(provider.slug) }"
      >
        <div class="provider-header">
          <div class="provider-icon">
            <img
              v-if="provider.icon_url"
              :src="provider.icon_url"
              :alt="provider.name"
              class="provider-logo"
            >
            <span v-else class="provider-letter">{{ provider.name.charAt(0) }}</span>
          </div>
          <div class="provider-info">
            <h3 class="provider-name">{{ provider.name }}</h3>
            <span class="provider-type">{{ provider.authorization_type }}</span>
          </div>
          <div class="provider-status">
            <span v-if="hasAppConfigured(provider.slug)" class="status-badge configured">
              {{ t('admin.oauth2Apps.configured') }}
            </span>
            <span v-else class="status-badge not-configured">
              {{ t('admin.oauth2Apps.notConfigured') }}
            </span>
          </div>
        </div>

        <p class="provider-description">{{ provider.description }}</p>

        <!-- Configured Apps -->
        <div v-if="hasAppConfigured(provider.slug)" class="configured-apps">
          <div
            v-for="app in getAppsForProvider(provider.slug)"
            :key="app.id"
            class="app-item"
          >
            <div class="app-info">
              <span class="app-client-id">{{ t('admin.oauth2Apps.clientId') }}: {{ app.client_id.substring(0, 20) }}...</span>
              <span class="app-scopes">{{ app.scopes?.length || 0 }} {{ t('admin.oauth2Apps.scopes') }}</span>
            </div>
            <button class="btn btn-sm btn-danger-outline" @click="handleDeleteApp(app)">
              {{ t('common.delete') }}
            </button>
          </div>
        </div>

        <!-- Configure Button -->
        <div class="provider-actions">
          <button
            v-if="!hasAppConfigured(provider.slug)"
            class="btn btn-primary"
            @click="openCreateModal(provider)"
          >
            {{ t('admin.oauth2Apps.configure') }}
          </button>
          <button
            v-else
            class="btn btn-outline"
            @click="openCreateModal(provider)"
          >
            {{ t('admin.oauth2Apps.addAnother') }}
          </button>
        </div>
      </div>
    </div>

    <!-- Create Modal -->
    <UiModal :show="showCreateModal" :title="t('admin.oauth2Apps.createTitle')" @close="showCreateModal = false">
      <div v-if="selectedProvider" class="create-form">
        <div class="selected-provider">
          <div class="provider-icon small">
            <img
              v-if="selectedProvider.icon_url"
              :src="selectedProvider.icon_url"
              :alt="selectedProvider.name"
              class="provider-logo"
            >
            <span v-else class="provider-letter">{{ selectedProvider.name.charAt(0) }}</span>
          </div>
          <span class="provider-name">{{ selectedProvider.name }}</span>
        </div>

        <div class="form-group">
          <label class="form-label">{{ t('admin.oauth2Apps.clientId') }} *</label>
          <input
            v-model="formData.client_id"
            type="text"
            class="form-input"
            :placeholder="t('admin.oauth2Apps.clientIdPlaceholder')"
          >
        </div>

        <div class="form-group">
          <label class="form-label">{{ t('admin.oauth2Apps.clientSecret') }} *</label>
          <input
            v-model="formData.client_secret"
            type="password"
            class="form-input"
            :placeholder="t('admin.oauth2Apps.clientSecretPlaceholder')"
          >
        </div>

        <div class="form-group">
          <label class="form-label">{{ t('admin.oauth2Apps.scopesLabel') }}</label>
          <div class="scopes-list">
            <label
              v-for="scope in selectedProvider.available_scopes"
              :key="scope"
              class="scope-item"
            >
              <input
                type="checkbox"
                :checked="formData.scopes?.includes(scope) ?? false"
                @change="toggleScope(scope)"
              >
              <span>{{ scope }}</span>
            </label>
          </div>
        </div>

        <div class="form-group">
          <label class="form-label">{{ t('admin.oauth2Apps.customAuthUrl') }}</label>
          <input
            v-model="formData.custom_authorization_url"
            type="text"
            class="form-input"
            :placeholder="selectedProvider.authorization_url"
          >
          <span class="form-hint">{{ t('admin.oauth2Apps.customUrlHint') }}</span>
        </div>

        <div class="form-group">
          <label class="form-label">{{ t('admin.oauth2Apps.customTokenUrl') }}</label>
          <input
            v-model="formData.custom_token_url"
            type="text"
            class="form-input"
            :placeholder="selectedProvider.token_url"
          >
        </div>
      </div>

      <template #footer>
        <button class="btn btn-outline" @click="showCreateModal = false">
          {{ t('common.cancel') }}
        </button>
        <button class="btn btn-primary" :disabled="creating" @click="handleCreateApp">
          {{ creating ? t('common.saving') : t('common.save') }}
        </button>
      </template>
    </UiModal>
  </div>
</template>

<style scoped>
.oauth2-apps-page {
  padding: 1.5rem;
}

.page-header {
  margin-bottom: 2rem;
}

.page-title {
  font-size: 1.5rem;
  font-weight: 600;
  margin: 0;
}

.page-subtitle {
  color: var(--color-text-secondary);
  margin: 0.25rem 0 0;
}

.loading-state {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
  padding: 3rem;
  color: var(--color-text-secondary);
}

.loading-spinner {
  width: 24px;
  height: 24px;
  border: 2px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.providers-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(400px, 1fr));
  gap: 1.5rem;
}

.provider-card {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 12px;
  padding: 1.5rem;
  transition: border-color 0.2s;
}

.provider-card.configured {
  border-color: var(--color-success);
}

.provider-header {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin-bottom: 1rem;
}

.provider-icon {
  width: 48px;
  height: 48px;
  border-radius: 10px;
  background: var(--color-background);
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.provider-icon.small {
  width: 32px;
  height: 32px;
  border-radius: 6px;
}

.provider-logo {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.provider-letter {
  font-size: 1.25rem;
  font-weight: 600;
  color: var(--color-primary);
}

.provider-info {
  flex: 1;
}

.provider-name {
  font-size: 1rem;
  font-weight: 600;
  margin: 0;
}

.provider-type {
  font-size: 0.75rem;
  color: var(--color-text-tertiary);
  text-transform: uppercase;
}

.provider-status {
  flex-shrink: 0;
}

.status-badge {
  display: inline-block;
  padding: 0.25rem 0.75rem;
  border-radius: 9999px;
  font-size: 0.75rem;
  font-weight: 500;
}

.status-badge.configured {
  background: #d1fae5;
  color: #065f46;
}

.status-badge.not-configured {
  background: var(--color-background);
  color: var(--color-text-secondary);
}

.provider-description {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
  margin: 0 0 1rem;
  line-height: 1.5;
}

.configured-apps {
  background: var(--color-background);
  border-radius: 8px;
  padding: 0.75rem;
  margin-bottom: 1rem;
}

.app-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.5rem 0;
}

.app-item:not(:last-child) {
  border-bottom: 1px solid var(--color-border);
}

.app-info {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
}

.app-client-id {
  font-size: 0.8125rem;
  font-family: monospace;
}

.app-scopes {
  font-size: 0.75rem;
  color: var(--color-text-tertiary);
}

.provider-actions {
  display: flex;
  gap: 0.5rem;
}

.create-form {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.selected-provider {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem;
  background: var(--color-background);
  border-radius: 8px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.form-label {
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--color-text);
}

.form-input {
  padding: 0.625rem 0.75rem;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  font-size: 0.875rem;
  transition: border-color 0.15s;
}

.form-input:focus {
  outline: none;
  border-color: var(--color-primary);
}

.form-hint {
  font-size: 0.75rem;
  color: var(--color-text-tertiary);
}

.scopes-list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  padding: 0.5rem;
  background: var(--color-background);
  border-radius: 6px;
}

.scope-item {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  font-size: 0.8125rem;
  cursor: pointer;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  transition: background 0.15s;
}

.scope-item:hover {
  background: var(--color-surface);
}

.btn-sm {
  padding: 0.375rem 0.75rem;
  font-size: 0.75rem;
}

.btn-outline {
  background: transparent;
  border: 1px solid var(--color-border);
  color: var(--color-text);
}

.btn-outline:hover {
  background: var(--color-background);
}

.btn-danger-outline {
  background: transparent;
  border: 1px solid #fecaca;
  color: var(--color-error);
}

.btn-danger-outline:hover {
  background: #fef2f2;
}
</style>
