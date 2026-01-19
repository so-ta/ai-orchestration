<script setup lang="ts">
/**
 * OAuth2ProviderSelector
 * OAuth2プロバイダーを選択するためのコンポーネント
 */
import type { OAuth2ProviderWithStatus } from '~/types/oauth2'

defineProps<{
  providers: OAuth2ProviderWithStatus[]
  loading?: boolean
}>()

const emit = defineEmits<{
  (e: 'select', provider: OAuth2ProviderWithStatus): void
}>()

const { t } = useI18n()

// Provider icons mapping
const providerIcons: Record<string, string> = {
  google: 'M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z',
  github: 'M12 2A10 10 0 0 0 2 12c0 4.42 2.87 8.17 6.84 9.5.5.08.66-.23.66-.5v-1.69c-2.77.6-3.36-1.34-3.36-1.34-.46-1.16-1.11-1.47-1.11-1.47-.91-.62.07-.6.07-.6 1 .07 1.53 1.03 1.53 1.03.87 1.52 2.34 1.07 2.91.83.09-.65.35-1.09.63-1.34-2.22-.25-4.55-1.11-4.55-4.92 0-1.11.38-2 1.03-2.71-.1-.25-.45-1.29.1-2.64 0 0 .84-.27 2.75 1.02.79-.22 1.65-.33 2.5-.33.85 0 1.71.11 2.5.33 1.91-1.29 2.75-1.02 2.75-1.02.55 1.35.2 2.39.1 2.64.65.71 1.03 1.6 1.03 2.71 0 3.82-2.34 4.66-4.57 4.91.36.31.69.92.69 1.85V21c0 .27.16.59.67.5C19.14 20.16 22 16.42 22 12A10 10 0 0 0 12 2z',
  slack: 'M5.042 15.165a2.528 2.528 0 0 1-2.52 2.523A2.528 2.528 0 0 1 0 15.165a2.527 2.527 0 0 1 2.522-2.52h2.52v2.52zM6.313 15.165a2.527 2.527 0 0 1 2.521-2.52 2.527 2.527 0 0 1 2.521 2.52v6.313A2.528 2.528 0 0 1 8.834 24a2.528 2.528 0 0 1-2.521-2.522v-6.313zM8.834 5.042a2.528 2.528 0 0 1-2.521-2.52A2.528 2.528 0 0 1 8.834 0a2.528 2.528 0 0 1 2.521 2.522v2.52H8.834zM8.834 6.313a2.528 2.528 0 0 1 2.521 2.521 2.528 2.528 0 0 1-2.521 2.521H2.522A2.528 2.528 0 0 1 0 8.834a2.528 2.528 0 0 1 2.522-2.521h6.312zM18.956 8.834a2.528 2.528 0 0 1 2.522-2.521A2.528 2.528 0 0 1 24 8.834a2.528 2.528 0 0 1-2.522 2.521h-2.522V8.834zM17.688 8.834a2.528 2.528 0 0 1-2.523 2.521 2.527 2.527 0 0 1-2.52-2.521V2.522A2.527 2.527 0 0 1 15.165 0a2.528 2.528 0 0 1 2.523 2.522v6.312zM15.165 18.956a2.528 2.528 0 0 1 2.523 2.522A2.528 2.528 0 0 1 15.165 24a2.527 2.527 0 0 1-2.52-2.522v-2.522h2.52zM15.165 17.688a2.527 2.527 0 0 1-2.52-2.523 2.526 2.526 0 0 1 2.52-2.52h6.313A2.527 2.527 0 0 1 24 15.165a2.528 2.528 0 0 1-2.522 2.523h-6.313z',
  notion: 'M4.459 4.208c.746.606 1.026.56 2.428.466l13.215-.793c.28 0 .047-.28-.046-.326L18.12 2.16c-.466-.373-.98-.559-2.054-.466l-12.8 1.026c-.466.047-.56.28-.373.466l1.566 1.022zm.793 3.172v13.87c0 .746.373 1.026 1.213.98l14.523-.84c.84-.046.933-.56.933-1.166V6.5c0-.606-.233-.886-.746-.84L5.484 6.42c-.56.046-.232.326-.232.98v-.02z',
  linear: 'M12 22C6.477 22 2 17.523 2 12S6.477 2 12 2s10 4.477 10 10-4.477 10-10 10zm0-2a8 8 0 1 0 0-16 8 8 0 0 0 0 16z',
  microsoft: 'M11 11H2V2h9v9zm11 0h-9V2h9v9zm0 11h-9v-9h9v9zm-11 0H2v-9h9v9z',
  discord: 'M20.317 4.37a19.791 19.791 0 0 0-4.885-1.515.074.074 0 0 0-.079.037c-.21.375-.444.864-.608 1.25a18.27 18.27 0 0 0-5.487 0 12.64 12.64 0 0 0-.617-1.25.077.077 0 0 0-.079-.037A19.736 19.736 0 0 0 3.677 4.37a.07.07 0 0 0-.032.027C.533 9.046-.32 13.58.099 18.057a.082.082 0 0 0 .031.057 19.9 19.9 0 0 0 5.993 3.03.078.078 0 0 0 .084-.028 14.09 14.09 0 0 0 1.226-1.994.076.076 0 0 0-.041-.106 13.107 13.107 0 0 1-1.872-.892.077.077 0 0 1-.008-.128 10.2 10.2 0 0 0 .372-.292.074.074 0 0 1 .077-.01c3.928 1.793 8.18 1.793 12.062 0a.074.074 0 0 1 .078.01c.12.098.246.198.373.292a.077.077 0 0 1-.006.127 12.299 12.299 0 0 1-1.873.892.077.077 0 0 0-.041.107c.36.698.772 1.362 1.225 1.993a.076.076 0 0 0 .084.028 19.839 19.839 0 0 0 6.002-3.03.077.077 0 0 0 .032-.054c.5-5.177-.838-9.674-3.549-13.66a.061.061 0 0 0-.031-.03z',
  atlassian: 'M7.127 11.373a.545.545 0 0 0-.926.128l-5.146 10.36a.545.545 0 0 0 .487.789h6.855a.545.545 0 0 0 .487-.3c1.74-3.48.89-8.541-1.757-10.977zm5.2-9.818a.545.545 0 0 0-.983.048C10.257 4.83 10 8.268 10 10.5c0 2.56.91 5.07 2.5 6.913a.545.545 0 0 0 .428.187h6.527a.545.545 0 0 0 .486-.789L12.327 1.555z',
}

// Provider colors
const providerColors: Record<string, string> = {
  google: '#4285F4',
  github: '#333333',
  slack: '#4A154B',
  notion: '#000000',
  linear: '#5E6AD2',
  microsoft: '#00A4EF',
  discord: '#5865F2',
  atlassian: '#0052CC',
}

function getProviderIcon(slug: string): string {
  return providerIcons[slug] || ''
}

function getProviderColor(slug: string): string {
  return providerColors[slug] || '#666666'
}

function handleSelect(provider: OAuth2ProviderWithStatus) {
  emit('select', provider)
}
</script>

<template>
  <div class="provider-selector">
    <h3 class="selector-title">{{ t('oauth.selectProvider') }}</h3>

    <!-- Loading -->
    <div v-if="loading" class="loading-state">
      <div class="loading-spinner" />
    </div>

    <!-- Provider List -->
    <div v-else class="provider-grid">
      <button
        v-for="provider in providers"
        :key="provider.id"
        class="provider-card"
        :class="{ configured: provider.app_configured }"
        @click="handleSelect(provider)"
      >
        <div class="provider-icon" :style="{ backgroundColor: getProviderColor(provider.slug) }">
          <svg v-if="getProviderIcon(provider.slug)" viewBox="0 0 24 24" fill="currentColor">
            <path :d="getProviderIcon(provider.slug)" />
          </svg>
          <span v-else class="provider-initial">{{ provider.name[0] }}</span>
        </div>
        <div class="provider-info">
          <span class="provider-name">{{ provider.name }}</span>
          <span v-if="provider.app_configured" class="provider-status configured">
            {{ t('oauth.configured') }}
          </span>
          <span v-else class="provider-status not-configured">
            {{ t('oauth.notConfigured') }}
          </span>
        </div>
        <svg class="provider-arrow" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="9,18 15,12 9,6" />
        </svg>
      </button>
    </div>
  </div>
</template>

<style scoped>
.provider-selector {
  padding: 1rem;
}

.selector-title {
  font-size: 1rem;
  font-weight: 600;
  margin-bottom: 1rem;
  color: var(--color-text-primary);
}

.loading-state {
  display: flex;
  justify-content: center;
  padding: 2rem;
}

.loading-spinner {
  width: 32px;
  height: 32px;
  border: 2px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.provider-grid {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.provider-card {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s ease;
  text-align: left;
  width: 100%;
}

.provider-card:hover {
  border-color: var(--color-primary);
  background: var(--color-surface-hover);
}

.provider-card.configured {
  border-color: var(--color-success);
  background: rgba(var(--color-success-rgb), 0.05);
}

.provider-icon {
  width: 40px;
  height: 40px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  flex-shrink: 0;
}

.provider-icon svg {
  width: 20px;
  height: 20px;
}

.provider-initial {
  font-size: 1.25rem;
  font-weight: 600;
}

.provider-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
}

.provider-name {
  font-weight: 500;
  color: var(--color-text-primary);
}

.provider-status {
  font-size: 0.75rem;
}

.provider-status.configured {
  color: var(--color-success);
}

.provider-status.not-configured {
  color: var(--color-text-tertiary);
}

.provider-arrow {
  color: var(--color-text-tertiary);
  flex-shrink: 0;
}
</style>
