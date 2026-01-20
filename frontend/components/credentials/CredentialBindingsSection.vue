<script setup lang="ts">
/**
 * CredentialBindingsSection
 * ブロック設定パネル内で使用する認証情報バインディングセクション
 */
import type { BlockDefinition } from '~/types/api'
import type { RequiredCredential } from '~/types/oauth2'

const props = defineProps<{
  blockDefinition?: BlockDefinition
  credentialBindings?: Record<string, string>
  readonly?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:credentialBindings', bindings: Record<string, string>): void
  (e: 'openSettings'): void
}>()

const { t } = useI18n()
const toast = useToast()
const { credentials, loading, fetchCredentials } = useCredentials()

// Parse required credentials from block definition
const requiredCredentials = computed<RequiredCredential[]>(() => {
  if (!props.blockDefinition) return []

  // Check if blockDefinition has required_credentials field
  const raw = (props.blockDefinition as unknown as Record<string, unknown>).required_credentials
  if (!raw) return []

  if (typeof raw === 'string') {
    try {
      return JSON.parse(raw)
    } catch {
      return []
    }
  }

  if (Array.isArray(raw)) {
    return raw as RequiredCredential[]
  }

  return []
})

// Local bindings state
const localBindings = ref<Record<string, string>>({})

// Initialize from props
watch(() => props.credentialBindings, (newBindings) => {
  localBindings.value = { ...newBindings }
}, { immediate: true, deep: true })

// Update binding
function updateBinding(name: string, credentialId: string | undefined) {
  if (credentialId) {
    localBindings.value = { ...localBindings.value, [name]: credentialId }
  } else {
    const { [name]: _, ...rest } = localBindings.value
    void _
    localBindings.value = rest
  }
  emit('update:credentialBindings', { ...localBindings.value })
}

// Handle create new credential
function handleCreateNew() {
  emit('openSettings')
}

// Fetch credentials only when needed
watch(requiredCredentials, async (required) => {
  if (required.length > 0) {
    try {
      await fetchCredentials()
    } catch {
      toast.error(t('credentialBindings.fetchError'))
    }
  }
}, { immediate: true })

// Check if section should be shown
const showSection = computed(() => requiredCredentials.value.length > 0)
</script>

<template>
  <section v-if="showSection" class="credential-bindings-section">
    <div class="section-header">
      <h4 class="section-title">{{ t('credentialBindings.title') }}</h4>
      <span class="section-subtitle">{{ t('credentialBindings.subtitle') }}</span>
    </div>

    <div class="bindings-list">
      <CredentialSelector
        v-for="req in requiredCredentials"
        :key="req.name"
        :model-value="localBindings[req.name]"
        :requirement="req"
        :credentials="credentials"
        :loading="loading"
        :disabled="readonly"
        @update:model-value="updateBinding(req.name, $event)"
        @create-new="handleCreateNew"
      />
    </div>
  </section>
</template>

<style scoped>
.credential-bindings-section {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 1rem;
  background: var(--color-surface-secondary);
  border-radius: 8px;
  border: 1px solid var(--color-border);
}

.section-header {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
}

.section-title {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0;
}

.section-subtitle {
  font-size: 0.75rem;
  color: var(--color-text-tertiary);
}

.bindings-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}
</style>
