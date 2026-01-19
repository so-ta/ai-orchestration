<script setup lang="ts">
import type { GitSyncDirection, Credential } from '~/types/api'

const { t } = useI18n()
const toast = useToast()

const props = defineProps<{
  show: boolean
  projectId: string
}>()

const emit = defineEmits<{
  (e: 'close' | 'updated'): void
}>()

const {
  gitSync,
  loading,
  syncing,
  fetchGitSync,
  configureGitSync,
  updateGitSync,
  deleteGitSync,
  triggerSync
} = useGitSync()

// Credentials
const credentials = useCredentials()
const availableCredentials = ref<Credential[]>([])

// Form state
const isEditing = ref(false)
const formData = ref({
  repository_url: '',
  branch: 'main',
  file_path: '',
  sync_direction: 'push' as GitSyncDirection,
  auto_sync: false,
  credentials_id: ''
})

// Sync direction options
const syncDirectionOptions = [
  { value: 'push', label: t('gitSync.direction.push') },
  { value: 'pull', label: t('gitSync.direction.pull') },
  { value: 'bidirectional', label: t('gitSync.direction.bidirectional') }
]

// Load data when modal opens
watch(() => props.show, async (show) => {
  if (show) {
    await loadData()
  }
}, { immediate: true })

async function loadData() {
  await fetchGitSync(props.projectId)

  // Load credentials for dropdown
  try {
    await credentials.fetchCredentials()
    availableCredentials.value = credentials.credentials.value.filter((c: Credential) =>
      c.credential_type === 'api_key' || c.credential_type === 'oauth2'
    )
  } catch {
    // Ignore credentials error
  }

  if (gitSync.value) {
    formData.value = {
      repository_url: gitSync.value.repository_url,
      branch: gitSync.value.branch,
      file_path: gitSync.value.file_path,
      sync_direction: gitSync.value.sync_direction,
      auto_sync: gitSync.value.auto_sync,
      credentials_id: gitSync.value.credentials_id || ''
    }
    isEditing.value = false
  } else {
    resetForm()
    isEditing.value = true
  }
}

function resetForm() {
  formData.value = {
    repository_url: '',
    branch: 'main',
    file_path: `workflows/${props.projectId}.json`,
    sync_direction: 'push',
    auto_sync: false,
    credentials_id: ''
  }
}

async function handleSave() {
  if (!formData.value.repository_url) {
    toast.error('リポジトリURLは必須です')
    return
  }

  try {
    if (gitSync.value) {
      await updateGitSync(props.projectId, {
        branch: formData.value.branch,
        file_path: formData.value.file_path,
        sync_direction: formData.value.sync_direction,
        auto_sync: formData.value.auto_sync,
        credentials_id: formData.value.credentials_id || undefined
      })
    } else {
      await configureGitSync(props.projectId, {
        repository_url: formData.value.repository_url,
        branch: formData.value.branch,
        file_path: formData.value.file_path,
        sync_direction: formData.value.sync_direction,
        credentials_id: formData.value.credentials_id || undefined
      })
    }
    toast.success(t('gitSync.messages.configured'))
    isEditing.value = false
    emit('updated')
  } catch {
    toast.error(t('gitSync.messages.configFailed'))
  }
}

async function handleSync() {
  try {
    await triggerSync(props.projectId)
    toast.success(t('gitSync.messages.syncSuccess'))
    await fetchGitSync(props.projectId)
  } catch {
    toast.error(t('gitSync.messages.syncFailed'))
  }
}

async function handleDelete() {
  try {
    await deleteGitSync(props.projectId)
    toast.success(t('gitSync.messages.deleted'))
    resetForm()
    isEditing.value = true
    emit('updated')
  } catch {
    toast.error('削除に失敗しました')
  }
}

function formatDate(dateStr: string | undefined): string {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString('ja-JP')
}

function handleClose() {
  emit('close')
}
</script>

<template>
  <UiModal :show="show" :title="t('gitSync.title')" size="md" @close="handleClose">
    <div class="git-sync-modal">
      <!-- Loading -->
      <div v-if="loading" class="loading-state">
        <div class="spinner" />
        <span>{{ t('common.loading') }}</span>
      </div>

      <!-- Content -->
      <div v-else class="modal-content">
        <!-- Status (when configured) -->
        <div v-if="gitSync && !isEditing" class="sync-status">
          <div class="status-header">
            <div class="status-icon success">
              <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M9 19c-5 1.5-5-2.5-7-3m14 6v-3.87a3.37 3.37 0 0 0-.94-2.61c3.14-.35 6.44-1.54 6.44-7A5.44 5.44 0 0 0 20 4.77 5.07 5.07 0 0 0 19.91 1S18.73.65 16 2.48a13.38 13.38 0 0 0-7 0C6.27.65 5.09 1 5.09 1A5.07 5.07 0 0 0 5 4.77a5.44 5.44 0 0 0-1.5 3.78c0 5.42 3.3 6.61 6.44 7A3.37 3.37 0 0 0 9 18.13V22"/>
              </svg>
            </div>
            <div class="status-info">
              <h3>{{ t('gitSync.title') }}</h3>
              <p class="repo-url">{{ gitSync.repository_url }}</p>
            </div>
          </div>

          <div class="status-details">
            <div class="detail-item">
              <span class="detail-label">{{ t('gitSync.form.branch') }}</span>
              <span class="detail-value">{{ gitSync.branch }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-label">{{ t('gitSync.form.filePath') }}</span>
              <code class="detail-value">{{ gitSync.file_path }}</code>
            </div>
            <div class="detail-item">
              <span class="detail-label">{{ t('gitSync.form.syncDirection') }}</span>
              <span class="detail-value">{{ t(`gitSync.direction.${gitSync.sync_direction}`) }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-label">{{ t('gitSync.form.autoSync') }}</span>
              <span class="detail-value">{{ gitSync.auto_sync ? t('common.enabled') : t('common.disabled') }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-label">{{ t('gitSync.lastSync') }}</span>
              <span class="detail-value">{{ formatDate(gitSync.last_sync_at) }}</span>
            </div>
            <div v-if="gitSync.last_commit_sha" class="detail-item">
              <span class="detail-label">{{ t('gitSync.commitSha') }}</span>
              <code class="detail-value sha">{{ gitSync.last_commit_sha.substring(0, 7) }}</code>
            </div>
          </div>

          <div class="status-actions">
            <button class="btn btn-primary" :disabled="syncing" @click="handleSync">
              <svg v-if="syncing" class="spinner-icon" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
              </svg>
              {{ syncing ? t('gitSync.syncing') : t('gitSync.syncNow') }}
            </button>
            <button class="btn btn-secondary" @click="isEditing = true">
              {{ t('common.edit') }}
            </button>
            <button class="btn btn-danger-outline" @click="handleDelete">
              {{ t('gitSync.disable') }}
            </button>
          </div>
        </div>

        <!-- Form (when editing or not configured) -->
        <div v-else class="sync-form">
          <div v-if="!gitSync" class="form-intro">
            <p>{{ t('gitSync.noSyncDesc') }}</p>
          </div>

          <div class="form-group">
            <label class="form-label">{{ t('gitSync.form.repositoryUrl') }} *</label>
            <input
              v-model="formData.repository_url"
              type="text"
              class="form-input"
              :placeholder="t('gitSync.form.repositoryUrlPlaceholder')"
              :disabled="!!gitSync"
            >
          </div>

          <div class="form-row">
            <div class="form-group">
              <label class="form-label">{{ t('gitSync.form.branch') }}</label>
              <input
                v-model="formData.branch"
                type="text"
                class="form-input"
                :placeholder="t('gitSync.form.branchPlaceholder')"
              >
            </div>

            <div class="form-group">
              <label class="form-label">{{ t('gitSync.form.filePath') }}</label>
              <input
                v-model="formData.file_path"
                type="text"
                class="form-input"
                :placeholder="t('gitSync.form.filePathPlaceholder')"
              >
            </div>
          </div>

          <div class="form-group">
            <label class="form-label">{{ t('gitSync.form.syncDirection') }}</label>
            <select v-model="formData.sync_direction" class="form-select">
              <option v-for="opt in syncDirectionOptions" :key="opt.value" :value="opt.value">
                {{ opt.label }}
              </option>
            </select>
          </div>

          <div class="form-group">
            <label class="form-label">{{ t('gitSync.form.credentials') }}</label>
            <select v-model="formData.credentials_id" class="form-select">
              <option value="">なし（公開リポジトリ）</option>
              <option v-for="cred in availableCredentials" :key="cred.id" :value="cred.id">
                {{ cred.name }}
              </option>
            </select>
          </div>

          <div class="form-group checkbox-group">
            <label class="checkbox-label">
              <input v-model="formData.auto_sync" type="checkbox">
              <span>{{ t('gitSync.form.autoSync') }}</span>
            </label>
            <p class="form-hint">{{ t('gitSync.form.autoSyncHint') }}</p>
          </div>

          <div class="form-actions">
            <button v-if="gitSync" class="btn btn-secondary" @click="isEditing = false">
              {{ t('common.cancel') }}
            </button>
            <button class="btn btn-primary" @click="handleSave">
              {{ t('common.save') }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </UiModal>
</template>

<style scoped>
.git-sync-modal {
  min-height: 200px;
}

.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 3rem;
  gap: 1rem;
}

.spinner {
  width: 32px;
  height: 32px;
  border: 3px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

.spinner-icon {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.modal-content {
  padding: 1rem;
}

/* Status View */
.sync-status {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.status-header {
  display: flex;
  align-items: flex-start;
  gap: 1rem;
}

.status-icon {
  width: 40px;
  height: 40px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.status-icon.success {
  background: #dcfce7;
  color: #15803d;
}

.status-info h3 {
  font-size: 1rem;
  font-weight: 600;
  margin: 0 0 0.25rem 0;
}

.repo-url {
  font-size: 0.8125rem;
  color: var(--color-text-secondary);
  margin: 0;
  word-break: break-all;
}

.status-details {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 0.75rem;
  padding: 1rem;
  background: var(--color-surface);
  border-radius: 8px;
}

.detail-item {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.detail-label {
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.detail-value {
  font-size: 0.8125rem;
  color: var(--color-text);
}

.detail-value.sha {
  font-family: 'SF Mono', Monaco, monospace;
  background: white;
  padding: 0.125rem 0.375rem;
  border-radius: 4px;
  font-size: 0.75rem;
}

.status-actions {
  display: flex;
  gap: 0.75rem;
}

/* Form View */
.sync-form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.form-intro {
  padding: 1rem;
  background: var(--color-surface);
  border-radius: 8px;
  margin-bottom: 0.5rem;
}

.form-intro p {
  font-size: 0.8125rem;
  color: var(--color-text-secondary);
  margin: 0;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.form-label {
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--color-text);
}

.form-input,
.form-select {
  padding: 0.5rem 0.75rem;
  font-size: 0.8125rem;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  background: white;
}

.form-input:focus,
.form-select:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.form-input:disabled {
  background: var(--color-surface);
  cursor: not-allowed;
}

.checkbox-group {
  flex-direction: row;
  flex-wrap: wrap;
  align-items: flex-start;
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.8125rem;
  cursor: pointer;
}

.checkbox-group .form-hint {
  width: 100%;
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
  margin: 0.25rem 0 0 0;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 0.5rem;
}

.btn-danger-outline {
  background: white;
  border: 1px solid #fecaca;
  color: var(--color-error);
}

.btn-danger-outline:hover {
  background: #fef2f2;
  border-color: var(--color-error);
}
</style>
