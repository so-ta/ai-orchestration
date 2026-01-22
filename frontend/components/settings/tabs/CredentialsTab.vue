<script setup lang="ts">
import type { Credential, CredentialStatus } from '~/types/api'
import CredentialFormModal from './CredentialFormModal.vue'

const { t } = useI18n()
const toast = useToast()
const credentialsApi = useCredentials()

const loading = ref(false)
const credentialsList = ref<Credential[]>([])
const showCredentialModal = ref(false)
const showDeleteModal = ref(false)
const selectedCredential = ref<Credential | null>(null)

async function fetchCredentials() {
  loading.value = true
  try {
    await credentialsApi.fetchCredentials()
    credentialsList.value = credentialsApi.credentials.value
  } catch {
    toast.error(t('credentials.messages.fetchFailed'))
  } finally {
    loading.value = false
  }
}

function openAddModal() {
  selectedCredential.value = null
  showCredentialModal.value = true
}

function openEditModal(credential: Credential) {
  selectedCredential.value = credential
  showCredentialModal.value = true
}

function openDeleteModal(credential: Credential) {
  selectedCredential.value = credential
  showDeleteModal.value = true
}

async function confirmDelete() {
  if (!selectedCredential.value) return

  loading.value = true
  try {
    await credentialsApi.deleteCredential(selectedCredential.value.id)
    showDeleteModal.value = false
    selectedCredential.value = null
    await fetchCredentials()
  } catch {
    toast.error(t('credentials.messages.deleteFailed'))
  } finally {
    loading.value = false
  }
}

async function revokeCredential(credential: Credential) {
  loading.value = true
  try {
    await credentialsApi.revokeCredential(credential.id)
    await fetchCredentials()
  } catch {
    toast.error(t('credentials.messages.revokeFailed'))
  } finally {
    loading.value = false
  }
}

async function activateCredential(credential: Credential) {
  loading.value = true
  try {
    await credentialsApi.activateCredential(credential.id)
    await fetchCredentials()
  } catch {
    toast.error(t('credentials.messages.activateFailed'))
  } finally {
    loading.value = false
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

async function handleSaved() {
  showCredentialModal.value = false
  await fetchCredentials()
}

// Expose fetch for parent to trigger
defineExpose({ fetchCredentials })
</script>

<template>
  <div class="tab-panel">
    <div class="section-header">
      <div>
        <p class="section-desc">{{ t('credentials.subtitle') }}</p>
      </div>
      <button class="btn btn-primary btn-sm" @click="openAddModal">
        + {{ t('credentials.newCredential') }}
      </button>
    </div>

    <div class="info-card">
      <p>{{ t('credentials.securityNote') }}</p>
    </div>

    <div v-if="loading && credentialsList.length === 0" class="empty-state">
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
          <button class="btn btn-sm btn-secondary" @click="openEditModal(credential)">
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
          <button class="btn btn-sm btn-danger" @click="openDeleteModal(credential)">
            {{ t('common.delete') }}
          </button>
        </div>
      </div>
    </div>

    <!-- Credential Form Modal -->
    <CredentialFormModal
      :show="showCredentialModal"
      :credential="selectedCredential"
      @close="showCredentialModal = false"
      @saved="handleSaved"
    />

    <!-- Delete Modal -->
    <UiModal
      :show="showDeleteModal"
      :title="t('credentials.editCredential')"
      size="sm"
      @close="showDeleteModal = false"
    >
      <p>{{ t('credentials.confirmDelete') }}</p>

      <template #footer>
        <button class="btn btn-secondary" @click="showDeleteModal = false">
          {{ t('common.cancel') }}
        </button>
        <button class="btn btn-danger" :disabled="loading" @click="confirmDelete">
          {{ t('common.delete') }}
        </button>
      </template>
    </UiModal>
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
</style>
