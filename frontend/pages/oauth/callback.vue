<script setup lang="ts">
/**
 * OAuth Callback Page
 * OAuth認証フローのコールバックを処理するページ
 *
 * クエリパラメータ:
 * - success=true: 認証成功
 * - error=authorization_failed: 認証失敗
 * - credential_id: 作成された認証情報のID
 */

definePageMeta({
  layout: 'default',
})

const { t } = useI18n()
const router = useRouter()
const route = useRoute()
const toast = useToast()

// Parse query parameters
const success = computed(() => route.query.success === 'true')
const error = computed(() => route.query.error as string | undefined)
const _credentialId = computed(() => route.query.credential_id as string | undefined)

// Loading state for redirect countdown
const redirecting = ref(false)
const countdown = ref(3)

// Error message mapping
const errorMessages: Record<string, string> = {
  authorization_failed: t('oauth.errors.authorizationFailed'),
  access_denied: t('oauth.errors.accessDenied'),
  invalid_state: t('oauth.errors.invalidState'),
  token_exchange_failed: t('oauth.errors.tokenExchangeFailed'),
}

const errorMessage = computed(() => {
  if (!error.value) return null
  return errorMessages[error.value] || t('oauth.errors.unknown')
})

// Redirect to settings after success
function redirectToSettings() {
  redirecting.value = true
  router.push('/settings?tab=credentials')
}

// Auto-redirect on success
onMounted(() => {
  if (success.value) {
    toast.success(t('oauth.connectionSuccess'))

    // Countdown and redirect
    const interval = setInterval(() => {
      countdown.value--
      if (countdown.value <= 0) {
        clearInterval(interval)
        redirectToSettings()
      }
    }, 1000)
  }
})
</script>

<template>
  <div class="oauth-callback">
    <div class="callback-card">
      <!-- Success State -->
      <template v-if="success">
        <div class="icon-success">
          <svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
            <polyline points="22,4 12,14.01 9,11.01" />
          </svg>
        </div>
        <h1>{{ t('oauth.connectionSuccess') }}</h1>
        <p class="message">{{ t('oauth.connectionSuccessMessage') }}</p>
        <p class="redirect-message">
          {{ t('oauth.redirectingIn', { seconds: countdown }) }}
        </p>
        <button class="btn btn-primary" :disabled="redirecting" @click="redirectToSettings">
          {{ t('oauth.goToSettings') }}
        </button>
      </template>

      <!-- Error State -->
      <template v-else-if="error">
        <div class="icon-error">
          <svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10" />
            <line x1="15" y1="9" x2="9" y2="15" />
            <line x1="9" y1="9" x2="15" y2="15" />
          </svg>
        </div>
        <h1>{{ t('oauth.connectionFailed') }}</h1>
        <p class="message error">{{ errorMessage }}</p>
        <div class="button-group">
          <button class="btn btn-secondary" @click="router.back()">
            {{ t('common.back') }}
          </button>
          <button class="btn btn-primary" @click="redirectToSettings">
            {{ t('oauth.goToSettings') }}
          </button>
        </div>
      </template>

      <!-- Loading State (no query params) -->
      <template v-else>
        <div class="loading-spinner" />
        <p>{{ t('oauth.processing') }}</p>
      </template>
    </div>
  </div>
</template>

<style scoped>
.oauth-callback {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: 2rem;
  background: var(--color-background);
}

.callback-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  max-width: 400px;
  padding: 3rem 2rem;
  text-align: center;
  background: var(--color-surface);
  border-radius: 12px;
  box-shadow: var(--shadow-lg);
}

.icon-success {
  color: var(--color-success);
  margin-bottom: 1.5rem;
}

.icon-error {
  color: var(--color-error);
  margin-bottom: 1.5rem;
}

h1 {
  font-size: 1.5rem;
  font-weight: 600;
  margin-bottom: 0.75rem;
  color: var(--color-text-primary);
}

.message {
  color: var(--color-text-secondary);
  margin-bottom: 1.5rem;
  line-height: 1.5;
}

.message.error {
  color: var(--color-error);
}

.redirect-message {
  font-size: 0.875rem;
  color: var(--color-text-tertiary);
  margin-bottom: 1.5rem;
}

.button-group {
  display: flex;
  gap: 0.75rem;
}

.btn {
  padding: 0.75rem 1.5rem;
  border-radius: 6px;
  font-weight: 500;
  cursor: pointer;
  border: none;
  transition: all 0.15s ease;
}

.btn-primary {
  background: var(--color-primary);
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background: var(--color-primary-dark);
}

.btn-primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-secondary {
  background: var(--color-surface-secondary);
  color: var(--color-text-primary);
}

.btn-secondary:hover {
  background: var(--color-surface-hover);
}

.loading-spinner {
  width: 48px;
  height: 48px;
  border: 3px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-bottom: 1.5rem;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
