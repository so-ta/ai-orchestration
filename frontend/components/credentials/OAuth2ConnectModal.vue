<script setup lang="ts">
/**
 * OAuth2ConnectModal
 * OAuth2認証フローを開始するためのモーダル
 */
import type { OAuth2ProviderWithStatus, CredentialScope } from '~/types/oauth2'

const props = defineProps<{
  show: boolean
  provider: OAuth2ProviderWithStatus | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'connect', data: { name: string; scope: CredentialScope; projectId?: string; scopes?: string[] }): void
}>()

const { t } = useI18n()
const toast = useToast()
const { startAuthorization } = useOAuth2()

// Form state
const credentialName = ref('')
const selectedScope = ref<CredentialScope>('personal')
const selectedProjectId = ref<string | undefined>()
const selectedScopes = ref<string[]>([])

// Loading state
const connecting = ref(false)

// Available scopes from provider
const availableScopes = computed(() => {
  if (!props.provider) return []
  return props.provider.default_scopes || []
})

// Initialize selected scopes when provider changes
watch(() => props.provider, (provider) => {
  if (provider) {
    selectedScopes.value = [...(provider.default_scopes || [])]
    credentialName.value = ''
    selectedScope.value = 'personal'
    selectedProjectId.value = undefined
  }
}, { immediate: true })

// Scope options
const scopeOptions = computed(() => [
  {
    value: 'organization' as CredentialScope,
    label: t('oauth.scope.organization'),
    description: t('oauth.scope.organizationDesc'),
  },
  {
    value: 'project' as CredentialScope,
    label: t('oauth.scope.project'),
    description: t('oauth.scope.projectDesc'),
  },
  {
    value: 'personal' as CredentialScope,
    label: t('oauth.scope.personal'),
    description: t('oauth.scope.personalDesc'),
  },
])

// Validation
const isValid = computed(() => {
  if (!credentialName.value.trim()) return false
  if (selectedScope.value === 'project' && !selectedProjectId.value) return false
  return true
})

// Check if provider is configured
const isConfigured = computed(() => props.provider?.app_configured ?? false)

async function handleConnect() {
  if (!props.provider || !isValid.value) return

  connecting.value = true
  try {
    const response = await startAuthorization({
      provider_slug: props.provider.slug,
      name: credentialName.value.trim(),
      scope: selectedScope.value,
      project_id: selectedProjectId.value,
      scopes: selectedScopes.value,
    })

    // Redirect to authorization URL
    window.location.href = response.authorization_url
  } catch (error) {
    console.error('Failed to start authorization:', error)
    toast.error(t('oauth.errors.authorizationFailed'))
    connecting.value = false
  }
}

function handleClose() {
  if (!connecting.value) {
    emit('close')
  }
}

function toggleScope(scope: string) {
  const index = selectedScopes.value.indexOf(scope)
  if (index === -1) {
    selectedScopes.value.push(scope)
  } else {
    selectedScopes.value.splice(index, 1)
  }
}
</script>

<template>
  <UiModal :show="show" :title="provider?.name ? `${t('oauth.connectService')}: ${provider.name}` : t('oauth.connectService')" size="md" @close="handleClose">
    <div v-if="provider" class="connect-form">
      <!-- Not Configured Warning -->
      <div v-if="!isConfigured" class="warning-banner">
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
          <line x1="12" y1="9" x2="12" y2="13" />
          <line x1="12" y1="17" x2="12.01" y2="17" />
        </svg>
        <div class="warning-content">
          <strong>{{ t('oauth.configureApp') }}</strong>
          <p>{{ t('oauth.configureAppDesc') }}</p>
        </div>
      </div>

      <!-- Step 1: Credential Name -->
      <div class="form-group">
        <label class="form-label">{{ t('oauth.form.credentialName') }}</label>
        <input
          v-model="credentialName"
          type="text"
          class="form-input"
          :placeholder="t('oauth.form.credentialNamePlaceholder')"
          :disabled="!isConfigured || connecting"
        />
      </div>

      <!-- Step 2: Scope Selection -->
      <div class="form-group">
        <label class="form-label">{{ t('oauth.form.selectScope') }}</label>
        <div class="scope-options">
          <label
            v-for="option in scopeOptions"
            :key="option.value"
            class="scope-option"
            :class="{ selected: selectedScope === option.value, disabled: !isConfigured }"
          >
            <input
              v-model="selectedScope"
              type="radio"
              :value="option.value"
              :disabled="!isConfigured || connecting"
            />
            <div class="scope-content">
              <span class="scope-label">{{ option.label }}</span>
              <span class="scope-desc">{{ option.description }}</span>
            </div>
          </label>
        </div>
      </div>

      <!-- Project Selection (for project scope) -->
      <div v-if="selectedScope === 'project'" class="form-group">
        <label class="form-label">{{ t('oauth.form.selectProject') }}</label>
        <select
          v-model="selectedProjectId"
          class="form-select"
          :disabled="!isConfigured || connecting"
        >
          <option value="">{{ t('oauth.form.selectProject') }}...</option>
          <!-- TODO: Load projects dynamically -->
        </select>
      </div>

      <!-- Step 3: Permission Scopes (if available) -->
      <div v-if="availableScopes.length > 0" class="form-group">
        <label class="form-label">{{ t('oauth.form.selectPermissions') }}</label>
        <div class="permission-list">
          <label
            v-for="scope in availableScopes"
            :key="scope"
            class="permission-item"
            :class="{ disabled: !isConfigured }"
          >
            <input
              type="checkbox"
              :checked="selectedScopes.includes(scope)"
              :disabled="!isConfigured || connecting"
              @change="toggleScope(scope)"
            />
            <span class="permission-name">{{ scope }}</span>
          </label>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="modal-footer">
        <button class="btn btn-secondary" :disabled="connecting" @click="handleClose">
          {{ t('common.cancel') }}
        </button>
        <button
          class="btn btn-primary"
          :disabled="!isValid || !isConfigured || connecting"
          @click="handleConnect"
        >
          <span v-if="connecting" class="btn-spinner" />
          {{ connecting ? t('oauth.processing') : t('oauth.form.connect') }}
        </button>
      </div>
    </template>
  </UiModal>
</template>

<style scoped>
.connect-form {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.warning-banner {
  display: flex;
  gap: 0.75rem;
  padding: 1rem;
  background: rgba(var(--color-warning-rgb), 0.1);
  border: 1px solid var(--color-warning);
  border-radius: 8px;
  color: var(--color-warning);
}

.warning-banner svg {
  flex-shrink: 0;
  margin-top: 0.125rem;
}

.warning-content {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.warning-content strong {
  font-size: 0.875rem;
}

.warning-content p {
  font-size: 0.8125rem;
  margin: 0;
  opacity: 0.9;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.form-label {
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--color-text-primary);
}

.form-input,
.form-select {
  padding: 0.625rem 0.75rem;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  background: var(--color-surface);
  color: var(--color-text-primary);
  font-size: 0.875rem;
}

.form-input:focus,
.form-select:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(var(--color-primary-rgb), 0.1);
}

.form-input:disabled,
.form-select:disabled {
  opacity: 0.5;
  cursor: not-allowed;
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
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.scope-option:hover:not(.disabled) {
  border-color: var(--color-primary);
}

.scope-option.selected {
  border-color: var(--color-primary);
  background: rgba(var(--color-primary-rgb), 0.05);
}

.scope-option.disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.scope-option input[type="radio"] {
  margin-top: 0.125rem;
}

.scope-content {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
}

.scope-label {
  font-weight: 500;
  color: var(--color-text-primary);
}

.scope-desc {
  font-size: 0.8125rem;
  color: var(--color-text-secondary);
}

.permission-list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.permission-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.375rem 0.75rem;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  font-size: 0.8125rem;
  cursor: pointer;
  transition: all 0.15s ease;
}

.permission-item:hover:not(.disabled) {
  border-color: var(--color-primary);
}

.permission-item.disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.permission-name {
  color: var(--color-text-primary);
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 0.625rem 1rem;
  border-radius: 6px;
  font-weight: 500;
  font-size: 0.875rem;
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

.btn-secondary:hover:not(:disabled) {
  background: var(--color-surface-hover);
}

.btn-spinner {
  width: 14px;
  height: 14px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
