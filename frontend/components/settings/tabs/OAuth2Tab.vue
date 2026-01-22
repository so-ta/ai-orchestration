<script setup lang="ts">
import type { OAuth2ProviderWithStatus } from '~/types/oauth2'
import SettingsOAuth2ConnectModal from './SettingsOAuth2ConnectModal.vue'

const emit = defineEmits<{
  (e: 'close-parent'): void
}>()

const { t } = useI18n()
const toast = useToast()
const { isAdmin } = useAuth()
const oauth2Api = useOAuth2()

const providers = ref<OAuth2ProviderWithStatus[]>([])
const loading = ref(false)
const showConnectModal = ref(false)
const selectedProvider = ref<OAuth2ProviderWithStatus | null>(null)

async function fetchProviders() {
  loading.value = true
  try {
    await oauth2Api.fetchProviders()
    providers.value = oauth2Api.providers.value
  } catch {
    toast.error(t('oauth.errors.unknown'))
  } finally {
    loading.value = false
  }
}

function openConnectModal(provider: OAuth2ProviderWithStatus) {
  if (!provider.app_configured) {
    toast.error(t('oauth.configureAppDesc'))
    return
  }
  selectedProvider.value = provider
  showConnectModal.value = true
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

// Expose fetch for parent to trigger
defineExpose({ fetchProviders })
</script>

<template>
  <div class="tab-panel">
    <div class="section-header">
      <div>
        <p class="section-desc">{{ t('oauth.subtitle') }}</p>
      </div>
      <NuxtLink v-if="isAdmin()" to="/admin/oauth2-apps" class="btn btn-secondary btn-sm" @click="emit('close-parent')">
        {{ t('oauth.app.title') }}
      </NuxtLink>
    </div>

    <div v-if="loading && providers.length === 0" class="empty-state">
      <p>{{ t('common.loading') }}</p>
    </div>

    <div v-else-if="providers.length === 0" class="empty-state">
      <p>{{ t('oauth.selectProvider') }}</p>
    </div>

    <div v-else class="oauth2-grid">
      <div
        v-for="provider in providers"
        :key="provider.slug"
        class="oauth2-provider-card"
        :class="{ 'not-configured': !provider.app_configured }"
        @click="openConnectModal(provider)"
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
    <div v-if="isAdmin() && providers.some(p => !p.app_configured)" class="admin-hint">
      <p>{{ t('oauth.configureAppDesc') }}</p>
      <NuxtLink to="/admin/oauth2-apps" class="admin-link" @click="emit('close-parent')">
        {{ t('oauth.configureApp') }} →
      </NuxtLink>
    </div>

    <!-- OAuth2 Connect Modal -->
    <SettingsOAuth2ConnectModal
      :show="showConnectModal"
      :provider="selectedProvider"
      @close="showConnectModal = false"
    />
  </div>
</template>

<style scoped>
.tab-panel {
  padding: 0.5rem 0;
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

.provider-logo {
  width: 24px;
  height: 24px;
  object-fit: contain;
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

.btn-sm {
  padding: 0.375rem 0.625rem;
  font-size: 0.75rem;
}
</style>
