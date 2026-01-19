<script setup lang="ts">
/**
 * CredentialSelector
 * ブロック設定で認証情報を選択するためのセレクトコンポーネント
 */
import type { Credential } from '~/types/api'
import type { RequiredCredential } from '~/types/oauth2'

const props = defineProps<{
  modelValue?: string
  requirement: RequiredCredential
  credentials: Credential[]
  loading?: boolean
  disabled?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: string | undefined): void
  (e: 'createNew'): void
}>()

const { t } = useI18n()

// Group credentials by scope
const groupedCredentials = computed(() => {
  const groups: Record<string, Credential[]> = {
    organization: [],
    project: [],
    personal: [],
    shared: [],
  }

  for (const cred of props.credentials) {
    // Filter by matching type if specified
    if (props.requirement.type !== 'custom' && cred.credential_type !== props.requirement.type) {
      continue
    }

    // TODO: Properly categorize by scope when available in Credential type
    // For now, put all in organization
    groups.organization.push(cred)
  }

  return groups
})

// Check if any credentials are available
const hasCredentials = computed(() => {
  return Object.values(groupedCredentials.value).some(group => group.length > 0)
})

// Selected credential info
const selectedCredential = computed(() => {
  if (!props.modelValue) return null
  return props.credentials.find(c => c.id === props.modelValue)
})

function handleChange(event: Event) {
  const target = event.target as HTMLSelectElement
  emit('update:modelValue', target.value || undefined)
}

function handleCreateNew() {
  emit('createNew')
}
</script>

<template>
  <div class="credential-selector">
    <div class="selector-header">
      <label class="selector-label">
        {{ requirement.description || requirement.name }}
        <span v-if="requirement.required" class="required-badge">{{ t('credentialBindings.required') }}</span>
        <span v-else class="optional-badge">{{ t('credentialBindings.optional') }}</span>
      </label>
    </div>

    <div class="selector-body">
      <!-- Loading state -->
      <div v-if="loading" class="loading-state">
        <div class="loading-spinner" />
      </div>

      <!-- No credentials available -->
      <div v-else-if="!hasCredentials" class="empty-state">
        <span class="empty-message">{{ t('credentialBindings.noCredentials') }}</span>
        <button class="create-btn" :disabled="disabled" @click="handleCreateNew">
          {{ t('credentialBindings.createNew') }}
        </button>
      </div>

      <!-- Credential select -->
      <div v-else class="select-wrapper">
        <select
          :value="modelValue || ''"
          class="credential-select"
          :disabled="disabled"
          :class="{ 'has-value': !!modelValue }"
          @change="handleChange"
        >
          <option value="">{{ t('credentialBindings.selectCredential') }}</option>

          <!-- Organization credentials -->
          <optgroup v-if="groupedCredentials.organization.length" :label="t('credentialBindings.scope.organization')">
            <option v-for="cred in groupedCredentials.organization" :key="cred.id" :value="cred.id">
              {{ cred.name }}
            </option>
          </optgroup>

          <!-- Project credentials -->
          <optgroup v-if="groupedCredentials.project.length" :label="t('credentialBindings.scope.project')">
            <option v-for="cred in groupedCredentials.project" :key="cred.id" :value="cred.id">
              {{ cred.name }}
            </option>
          </optgroup>

          <!-- Personal credentials -->
          <optgroup v-if="groupedCredentials.personal.length" :label="t('credentialBindings.scope.personal')">
            <option v-for="cred in groupedCredentials.personal" :key="cred.id" :value="cred.id">
              {{ cred.name }}
            </option>
          </optgroup>

          <!-- Shared credentials -->
          <optgroup v-if="groupedCredentials.shared.length" :label="t('credentialBindings.scope.shared')">
            <option v-for="cred in groupedCredentials.shared" :key="cred.id" :value="cred.id">
              {{ cred.name }}
            </option>
          </optgroup>
        </select>

        <button class="create-btn-inline" :disabled="disabled" @click="handleCreateNew">
          +
        </button>
      </div>
    </div>

    <!-- Selected credential info -->
    <div v-if="selectedCredential" class="selected-info">
      <span class="selected-type">{{ selectedCredential.credential_type }}</span>
      <span v-if="selectedCredential.status !== 'active'" class="selected-status" :class="selectedCredential.status">
        {{ selectedCredential.status }}
      </span>
    </div>
  </div>
</template>

<style scoped>
.credential-selector {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.selector-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.selector-label {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--color-text-primary);
}

.required-badge {
  font-size: 0.6875rem;
  padding: 0.125rem 0.375rem;
  background: rgba(var(--color-error-rgb), 0.1);
  color: var(--color-error);
  border-radius: 3px;
  font-weight: 500;
}

.optional-badge {
  font-size: 0.6875rem;
  padding: 0.125rem 0.375rem;
  background: var(--color-surface-secondary);
  color: var(--color-text-tertiary);
  border-radius: 3px;
  font-weight: 500;
}

.selector-body {
  display: flex;
  align-items: stretch;
}

.loading-state {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 36px;
}

.loading-spinner {
  width: 16px;
  height: 16px;
  border: 2px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.empty-state {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.5rem 0.75rem;
  background: var(--color-surface-secondary);
  border-radius: 6px;
  width: 100%;
}

.empty-message {
  flex: 1;
  font-size: 0.8125rem;
  color: var(--color-text-tertiary);
}

.create-btn {
  padding: 0.375rem 0.75rem;
  background: var(--color-primary);
  color: white;
  border: none;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s ease;
}

.create-btn:hover:not(:disabled) {
  background: var(--color-primary-dark);
}

.create-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.select-wrapper {
  display: flex;
  gap: 0.375rem;
  flex: 1;
}

.credential-select {
  flex: 1;
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  background: var(--color-surface);
  color: var(--color-text-primary);
  font-size: 0.8125rem;
  cursor: pointer;
}

.credential-select:focus {
  outline: none;
  border-color: var(--color-primary);
}

.credential-select:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.credential-select.has-value {
  border-color: var(--color-success);
}

.create-btn-inline {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-surface-secondary);
  border: 1px solid var(--color-border);
  border-radius: 6px;
  color: var(--color-text-secondary);
  font-size: 1.25rem;
  cursor: pointer;
  transition: all 0.15s ease;
}

.create-btn-inline:hover:not(:disabled) {
  background: var(--color-surface-hover);
  border-color: var(--color-primary);
  color: var(--color-primary);
}

.create-btn-inline:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.selected-info {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding-left: 0.125rem;
}

.selected-type {
  font-size: 0.6875rem;
  color: var(--color-text-tertiary);
  text-transform: uppercase;
}

.selected-status {
  font-size: 0.6875rem;
  padding: 0.125rem 0.375rem;
  border-radius: 3px;
  font-weight: 500;
}

.selected-status.expired {
  background: rgba(var(--color-warning-rgb), 0.1);
  color: var(--color-warning);
}

.selected-status.revoked {
  background: rgba(var(--color-error-rgb), 0.1);
  color: var(--color-error);
}
</style>
