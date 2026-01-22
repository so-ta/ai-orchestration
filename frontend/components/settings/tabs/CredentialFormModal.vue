<script setup lang="ts">
import type { Credential, CredentialType, CreateCredentialRequest, CredentialData } from '~/types/api'

const props = defineProps<{
  show: boolean
  credential: Credential | null
}>()

const emit = defineEmits<{
  (e: 'close' | 'saved'): void
}>()

const { t } = useI18n()
const toast = useToast()
const credentialsApi = useCredentials()

const loading = ref(false)
const formData = reactive({
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

const credentialTypes = computed(() => [
  { value: 'api_key', label: t('credentials.type.api_key') },
  { value: 'basic_auth', label: t('credentials.type.basic_auth') },
  { value: 'oauth2', label: t('credentials.type.oauth2') },
  { value: 'custom', label: t('credentials.type.custom') },
])

const isEdit = computed(() => !!props.credential)

watch(() => props.show, (isOpen) => {
  if (isOpen) {
    if (props.credential) {
      formData.name = props.credential.name
      formData.description = props.credential.description || ''
      formData.credential_type = props.credential.credential_type
      formData.provider = props.credential.metadata?.provider || ''
      formData.expires_at = props.credential.expires_at || ''
      formData.api_key = ''
      formData.header_name = 'Authorization'
      formData.header_prefix = 'Bearer '
      formData.username = ''
      formData.password = ''
    } else {
      formData.name = ''
      formData.description = ''
      formData.credential_type = 'api_key'
      formData.provider = ''
      formData.expires_at = ''
      formData.api_key = ''
      formData.header_name = 'Authorization'
      formData.header_prefix = 'Bearer '
      formData.username = ''
      formData.password = ''
    }
  }
})

async function submit() {
  if (!formData.name.trim()) {
    toast.error(t('common.error'))
    return
  }

  loading.value = true
  try {
    const credentialData: CredentialData = {}

    if (formData.credential_type === 'api_key') {
      if (!isEdit.value && !formData.api_key) {
        toast.error(t('common.error'))
        loading.value = false
        return
      }
      if (formData.api_key) {
        credentialData.api_key = formData.api_key
        credentialData.header_name = formData.header_name
        credentialData.header_prefix = formData.header_prefix
      }
    } else if (formData.credential_type === 'basic_auth') {
      if (!isEdit.value && (!formData.username || !formData.password)) {
        toast.error(t('common.error'))
        loading.value = false
        return
      }
      if (formData.username) credentialData.username = formData.username
      if (formData.password) credentialData.password = formData.password
    }

    if (isEdit.value && props.credential) {
      await credentialsApi.updateCredential(props.credential.id, {
        name: formData.name,
        description: formData.description || undefined,
        data: Object.keys(credentialData).length > 0 ? credentialData : undefined,
        metadata: formData.provider ? { provider: formData.provider } : undefined,
        expires_at: formData.expires_at || undefined,
      })
    } else {
      const request: CreateCredentialRequest = {
        name: formData.name,
        description: formData.description || undefined,
        credential_type: formData.credential_type,
        data: credentialData,
        metadata: formData.provider ? { provider: formData.provider } : undefined,
        expires_at: formData.expires_at || undefined,
      }
      await credentialsApi.createCredential(request)
    }
    emit('saved')
  } catch {
    toast.error(isEdit.value ? t('credentials.messages.updateFailed') : t('credentials.messages.createFailed'))
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <UiModal
    :show="show"
    :title="isEdit ? t('credentials.editCredential') : t('credentials.newCredential')"
    size="md"
    @close="emit('close')"
  >
    <form @submit.prevent="submit">
      <div class="form-group">
        <label class="form-label">{{ t('credentials.form.name') }} *</label>
        <input
          v-model="formData.name"
          type="text"
          class="form-input"
          :placeholder="t('credentials.form.namePlaceholder')"
          required
        >
      </div>

      <div class="form-group">
        <label class="form-label">{{ t('credentials.form.description') }}</label>
        <textarea
          v-model="formData.description"
          class="form-input"
          rows="2"
          :placeholder="t('credentials.form.descriptionPlaceholder')"
        />
      </div>

      <div class="form-group">
        <label class="form-label">{{ t('credentials.form.credentialType') }}</label>
        <select v-model="formData.credential_type" class="form-input" :disabled="isEdit">
          <option v-for="ct in credentialTypes" :key="ct.value" :value="ct.value">
            {{ ct.label }}
          </option>
        </select>
      </div>

      <div class="form-group">
        <label class="form-label">{{ t('credentials.form.provider') }}</label>
        <input
          v-model="formData.provider"
          type="text"
          class="form-input"
          :placeholder="t('credentials.form.providerPlaceholder')"
        >
      </div>

      <!-- API Key fields -->
      <template v-if="formData.credential_type === 'api_key'">
        <div class="form-group">
          <label class="form-label">
            {{ t('credentials.form.apiKey') }} {{ isEdit ? '' : '*' }}
          </label>
          <input
            v-model="formData.api_key"
            type="password"
            class="form-input"
            :placeholder="t('credentials.form.apiKeyPlaceholder')"
            :required="!isEdit"
          >
          <p v-if="isEdit" class="form-hint">
            {{ t('credentials.form.valueHidden') }}
          </p>
        </div>

        <div class="form-row">
          <div class="form-group">
            <label class="form-label">{{ t('credentials.form.headerName') }}</label>
            <input
              v-model="formData.header_name"
              type="text"
              class="form-input"
              :placeholder="t('credentials.form.headerNamePlaceholder')"
            >
          </div>
          <div class="form-group">
            <label class="form-label">{{ t('credentials.form.headerPrefix') }}</label>
            <input
              v-model="formData.header_prefix"
              type="text"
              class="form-input"
              :placeholder="t('credentials.form.headerPrefixPlaceholder')"
            >
          </div>
        </div>
      </template>

      <!-- Basic Auth fields -->
      <template v-if="formData.credential_type === 'basic_auth'">
        <div class="form-group">
          <label class="form-label">
            {{ t('credentials.form.username') }} {{ isEdit ? '' : '*' }}
          </label>
          <input
            v-model="formData.username"
            type="text"
            class="form-input"
            :placeholder="t('credentials.form.usernamePlaceholder')"
            :required="!isEdit"
          >
        </div>

        <div class="form-group">
          <label class="form-label">
            {{ t('credentials.form.password') }} {{ isEdit ? '' : '*' }}
          </label>
          <input
            v-model="formData.password"
            type="password"
            class="form-input"
            :placeholder="t('credentials.form.passwordPlaceholder')"
            :required="!isEdit"
          >
          <p v-if="isEdit" class="form-hint">
            {{ t('credentials.form.valueHidden') }}
          </p>
        </div>
      </template>

      <div class="form-group">
        <label class="form-label">{{ t('credentials.form.expiresAt') }}</label>
        <input
          v-model="formData.expires_at"
          type="datetime-local"
          class="form-input"
        >
      </div>
    </form>

    <template #footer>
      <button class="btn btn-secondary" @click="emit('close')">
        {{ t('common.cancel') }}
      </button>
      <button class="btn btn-primary" :disabled="loading" @click="submit">
        {{ loading ? t('common.saving') : t('common.save') }}
      </button>
    </template>
  </UiModal>
</template>

<style scoped>
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
</style>
