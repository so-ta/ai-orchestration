<script setup lang="ts">
import type { OAuth2ProviderWithStatus, CredentialScope, StartAuthorizationRequest } from '~/types/oauth2'

const props = defineProps<{
  show: boolean
  provider: OAuth2ProviderWithStatus | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const { t } = useI18n()
const toast = useToast()
const oauth2Api = useOAuth2()

const loading = ref(false)
const formData = reactive({
  name: '',
  scope: 'personal' as CredentialScope,
  projectId: '',
  selectedScopes: [] as string[],
})

const scopeOptions = computed(() => [
  { value: 'personal', label: t('oauth.scope.personal'), desc: t('oauth.scope.personalDesc') },
  { value: 'project', label: t('oauth.scope.project'), desc: t('oauth.scope.projectDesc') },
  { value: 'organization', label: t('oauth.scope.organization'), desc: t('oauth.scope.organizationDesc') },
])

watch(() => props.show, (isOpen) => {
  if (isOpen && props.provider) {
    formData.name = `${props.provider.name} Connection`
    formData.scope = 'personal'
    formData.projectId = ''
    formData.selectedScopes = [...(props.provider.default_scopes || [])]
  }
})

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

async function startAuthorization() {
  if (!props.provider) return

  loading.value = true
  try {
    const request: StartAuthorizationRequest = {
      provider_slug: props.provider.slug,
      name: formData.name,
      scope: formData.scope,
      scopes: formData.selectedScopes,
    }
    if (formData.scope === 'project' && formData.projectId) {
      request.project_id = formData.projectId
    }

    const response = await oauth2Api.startAuthorization(request)
    window.location.href = response.authorization_url
  } catch {
    toast.error(t('oauth.errors.authorizationFailed'))
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <UiModal
    :show="show"
    :title="t('oauth.connectService')"
    size="md"
    @close="emit('close')"
  >
    <div v-if="provider" class="oauth2-connect-form">
      <div class="provider-header">
        <div class="oauth2-provider-icon large">
          <img
            v-if="getProviderIcon(provider.slug)"
            :src="getProviderIcon(provider.slug)"
            :alt="provider.name"
            class="provider-logo"
          >
          <span v-else class="provider-fallback">{{ provider.name[0] }}</span>
        </div>
        <div class="provider-title">
          <h3>{{ provider.name }}</h3>
          <p v-if="provider.description" class="text-secondary">{{ provider.description }}</p>
        </div>
      </div>

      <form @submit.prevent="startAuthorization">
        <div class="form-group">
          <label class="form-label">{{ t('oauth.form.credentialName') }} *</label>
          <input
            v-model="formData.name"
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
              :class="{ selected: formData.scope === option.value }"
            >
              <input
                v-model="formData.scope"
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

        <div v-if="provider.available_scopes && provider.available_scopes.length > 0" class="form-group">
          <label class="form-label">{{ t('oauth.form.selectPermissions') }}</label>
          <div class="scope-checkboxes">
            <label
              v-for="scope in provider.available_scopes"
              :key="scope"
              class="scope-checkbox"
            >
              <input
                v-model="formData.selectedScopes"
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
      <button class="btn btn-secondary" @click="emit('close')">
        {{ t('common.cancel') }}
      </button>
      <button class="btn btn-primary" :disabled="loading" @click="startAuthorization">
        {{ loading ? t('common.loading') : t('oauth.form.connect') }}
      </button>
    </template>
  </UiModal>
</template>

<style scoped>
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

.provider-title h3 {
  margin: 0;
  font-size: 1.125rem;
  font-weight: 600;
}

.provider-title p {
  margin: 0.25rem 0 0;
  font-size: 0.875rem;
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

.text-secondary {
  color: var(--color-text-secondary);
}
</style>
