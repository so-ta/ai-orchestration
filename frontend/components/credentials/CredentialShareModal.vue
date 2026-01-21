<script setup lang="ts">
/**
 * CredentialShareModal
 * 認証情報の共有設定モーダル
 */
import type { Credential } from '~/types/api'
import type { CredentialShare, SharePermission } from '~/types/oauth2'

const props = defineProps<{
  modelValue: boolean
  credential: Credential | null
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
}>()

const { t } = useI18n()
const toast = useToast()

// Credential ID ref for composable
const credentialId = computed(() => props.credential?.id)
const {
  shares,
  loading,
  shareWithUser,
  shareWithProject,
  updateShare,
  revokeShare
} = useCredentialShares(credentialId)

// Form state
const shareType = ref<'user' | 'project'>('user')
const targetEmail = ref('')
const targetProjectId = ref('')
const permission = ref<SharePermission>('use')

const permissionOptions: { value: SharePermission; label: string }[] = [
  { value: 'use', label: 'credentialShares.permissions.use' },
  { value: 'edit', label: 'credentialShares.permissions.edit' },
  { value: 'admin', label: 'credentialShares.permissions.admin' },
]

// Close modal
function close() {
  emit('update:modelValue', false)
}

// Handle share
async function handleShare() {
  try {
    if (shareType.value === 'user') {
      if (!targetEmail.value) {
        toast.error(t('credentialShares.validation.emailRequired'))
        return
      }
      await shareWithUser({
        target_user_email: targetEmail.value,
        permission: permission.value,
      })
      targetEmail.value = ''
    } else {
      if (!targetProjectId.value) {
        toast.error(t('credentialShares.validation.projectRequired'))
        return
      }
      await shareWithProject({
        target_project_id: targetProjectId.value,
        permission: permission.value,
      })
      targetProjectId.value = ''
    }
  } catch {
    toast.error(t('credentialShares.shareError'))
  }
}

// Handle update share
async function handleUpdateShare(share: CredentialShare, newPermission: SharePermission) {
  try {
    await updateShare(share.id, { permission: newPermission })
  } catch {
    toast.error(t('credentialShares.updateError'))
  }
}

// Handle revoke share
async function handleRevoke(share: CredentialShare) {
  try {
    await revokeShare(share.id)
  } catch {
    toast.error(t('credentialShares.revokeError'))
  }
}

// Get display name for share target
function getShareTargetName(share: CredentialShare): string {
  if (share.target_user_id) {
    return share.target_user_email || t('credentialShares.unknownUser')
  }
  if (share.target_project_id) {
    return share.target_project_name || t('credentialShares.unknownProject')
  }
  return t('credentialShares.unknown')
}

// Get share type icon
function getShareTypeIcon(share: CredentialShare): string {
  return share.target_user_id ? 'user' : 'project'
}
</script>

<template>
  <UiModal :show="modelValue" :title="t('credentialShares.title')" @close="close">
    <div v-if="credential" class="share-modal">
      <!-- Credential Info -->
      <div class="credential-info">
        <span class="credential-name">{{ credential.name }}</span>
        <span class="credential-type">{{ credential.credential_type }}</span>
      </div>

      <!-- Share Form -->
      <div class="share-form">
        <h4 class="section-title">{{ t('credentialShares.addShare') }}</h4>

        <!-- Share Type Tabs -->
        <div class="share-type-tabs">
          <button
            class="type-tab"
            :class="{ active: shareType === 'user' }"
            @click="shareType = 'user'"
          >
            {{ t('credentialShares.shareWithUser') }}
          </button>
          <button
            class="type-tab"
            :class="{ active: shareType === 'project' }"
            @click="shareType = 'project'"
          >
            {{ t('credentialShares.shareWithProject') }}
          </button>
        </div>

        <!-- User Email Input -->
        <div v-if="shareType === 'user'" class="form-group">
          <label class="form-label">{{ t('credentialShares.userEmail') }}</label>
          <input
            v-model="targetEmail"
            type="email"
            class="form-input"
            :placeholder="t('credentialShares.userEmailPlaceholder')"
          >
        </div>

        <!-- Project ID Input -->
        <div v-if="shareType === 'project'" class="form-group">
          <label class="form-label">{{ t('credentialShares.projectId') }}</label>
          <input
            v-model="targetProjectId"
            type="text"
            class="form-input"
            :placeholder="t('credentialShares.projectIdPlaceholder')"
          >
        </div>

        <!-- Permission Select -->
        <div class="form-group">
          <label class="form-label">{{ t('credentialShares.permission') }}</label>
          <select v-model="permission" class="form-select">
            <option v-for="opt in permissionOptions" :key="opt.value" :value="opt.value">
              {{ t(opt.label) }}
            </option>
          </select>
        </div>

        <button class="btn btn-primary" @click="handleShare">
          {{ t('credentialShares.share') }}
        </button>
      </div>

      <!-- Existing Shares -->
      <div class="shares-list">
        <h4 class="section-title">{{ t('credentialShares.existingShares') }}</h4>

        <div v-if="loading" class="loading-state">
          <span class="loading-spinner" />
        </div>

        <div v-else-if="shares.length === 0" class="empty-state">
          {{ t('credentialShares.noShares') }}
        </div>

        <div v-else class="shares-items">
          <div v-for="share in shares" :key="share.id" class="share-item">
            <div class="share-target">
              <span class="share-icon">
                <svg v-if="getShareTypeIcon(share) === 'user'" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
                  <circle cx="12" cy="7" r="4"/>
                </svg>
                <svg v-else xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
                </svg>
              </span>
              <span class="share-name">{{ getShareTargetName(share) }}</span>
            </div>
            <div class="share-actions">
              <select
                :value="share.permission"
                class="form-select small"
                @change="handleUpdateShare(share, ($event.target as HTMLSelectElement).value as SharePermission)"
              >
                <option v-for="opt in permissionOptions" :key="opt.value" :value="opt.value">
                  {{ t(opt.label) }}
                </option>
              </select>
              <button class="btn-icon danger" @click="handleRevoke(share)">
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="3 6 5 6 21 6"/>
                  <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                </svg>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <button class="btn btn-outline" @click="close">
        {{ t('common.close') }}
      </button>
    </template>
  </UiModal>
</template>

<style scoped>
.share-modal {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.credential-info {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem;
  background: var(--color-background);
  border-radius: 8px;
}

.credential-name {
  font-weight: 600;
}

.credential-type {
  font-size: 0.75rem;
  color: var(--color-text-tertiary);
  background: var(--color-surface);
  padding: 0.125rem 0.5rem;
  border-radius: 4px;
}

.share-form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  padding: 1rem;
  background: var(--color-surface-secondary);
  border-radius: 8px;
  border: 1px solid var(--color-border);
}

.section-title {
  font-size: 0.8125rem;
  font-weight: 600;
  margin: 0;
  color: var(--color-text);
}

.share-type-tabs {
  display: flex;
  gap: 0.5rem;
}

.type-tab {
  flex: 1;
  padding: 0.5rem;
  font-size: 0.8125rem;
  border: 1px solid var(--color-border);
  background: var(--color-surface);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
}

.type-tab:hover {
  border-color: var(--color-primary);
}

.type-tab.active {
  background: var(--color-primary);
  border-color: var(--color-primary);
  color: white;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.form-label {
  font-size: 0.8125rem;
  font-weight: 500;
}

.form-input,
.form-select {
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  font-size: 0.875rem;
  transition: border-color 0.15s;
}

.form-input:focus,
.form-select:focus {
  outline: none;
  border-color: var(--color-primary);
}

.form-select.small {
  padding: 0.25rem 0.5rem;
  font-size: 0.75rem;
}

.shares-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.loading-state {
  display: flex;
  justify-content: center;
  padding: 1rem;
}

.loading-spinner {
  width: 20px;
  height: 20px;
  border: 2px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.empty-state {
  text-align: center;
  color: var(--color-text-secondary);
  font-size: 0.875rem;
  padding: 1rem;
}

.shares-items {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.share-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.75rem;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 8px;
}

.share-target {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.share-icon {
  color: var(--color-text-secondary);
}

.share-name {
  font-size: 0.875rem;
}

.share-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.btn-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  background: transparent;
  border-radius: 4px;
  cursor: pointer;
  color: var(--color-text-secondary);
  transition: all 0.15s;
}

.btn-icon:hover {
  background: var(--color-background);
}

.btn-icon.danger:hover {
  background: #fef2f2;
  color: var(--color-error);
}

.btn-outline {
  background: transparent;
  border: 1px solid var(--color-border);
  color: var(--color-text);
}

.btn-outline:hover {
  background: var(--color-background);
}
</style>
