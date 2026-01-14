<script setup lang="ts">
import type { BlockDefinition, BlockCategory, Credential, CredentialType, CredentialStatus, CreateCredentialRequest, CredentialData, AuditLog, AuditAction } from '~/types/api'
import type { AuditLogFilter } from '~/composables/useAuditLogs'

const { t } = useI18n()
const toast = useToast()

// Settings page
const activeTab = ref('general')

// General settings
const generalSettings = reactive({
  timezone: 'Asia/Tokyo',
  dateFormat: 'YYYY-MM-DD',
})

// Notification settings
const notificationSettings = reactive({
  emailOnFailure: true,
  emailOnSuccess: false,
  slackWebhook: '',
})

const saving = ref(false)

async function saveSettings() {
  saving.value = true
  // Simulate save - in real implementation, this would call the API
  await new Promise(resolve => setTimeout(resolve, 500))
  toast.success(t('common.success'))
  saving.value = false
}

// ============================================================================
// Blocks Tab
// ============================================================================
const blocks = useBlocks()
const blocksLoading = ref(false)
const blocksList = ref<BlockDefinition[]>([])
const showBlockModal = ref(false)
const showDeleteBlockModal = ref(false)
const selectedBlock = ref<BlockDefinition | null>(null)
const blockFormData = reactive({
  slug: '',
  name: '',
  description: '',
  category: 'integration' as BlockCategory,
  icon: '',
  config_schema: '{}',
  input_schema: '{}',
  output_schema: '{}',
  code: '',
  ui_config: '{}',
})

async function fetchBlocks() {
  blocksLoading.value = true
  try {
    const response = await blocks.list()
    // Filter to show only tenant blocks (exclude system blocks)
    blocksList.value = (response.blocks || []).filter(block => !block.is_system)
  } catch {
    toast.error(t('tenantBlocks.messages.createFailed'))
  } finally {
    blocksLoading.value = false
  }
}

function openAddBlockModal() {
  selectedBlock.value = null
  blockFormData.slug = ''
  blockFormData.name = ''
  blockFormData.description = ''
  blockFormData.category = 'integration'
  blockFormData.icon = ''
  blockFormData.config_schema = '{}'
  blockFormData.input_schema = '{}'
  blockFormData.output_schema = '{}'
  blockFormData.code = ''
  blockFormData.ui_config = '{}'
  showBlockModal.value = true
}

function openEditBlockModal(block: BlockDefinition) {
  selectedBlock.value = block
  blockFormData.slug = block.slug
  blockFormData.name = block.name
  blockFormData.description = block.description || ''
  blockFormData.category = block.category
  blockFormData.icon = block.icon || ''
  blockFormData.config_schema = JSON.stringify(block.config_schema || {}, null, 2)
  blockFormData.input_schema = JSON.stringify(block.input_schema || {}, null, 2)
  blockFormData.output_schema = JSON.stringify(block.output_schema || {}, null, 2)
  blockFormData.code = block.code || ''
  blockFormData.ui_config = JSON.stringify(block.ui_config || {}, null, 2)
  showBlockModal.value = true
}

async function submitBlockForm() {
  if (!blockFormData.name.trim() || !blockFormData.slug.trim()) {
    toast.error(t('common.error'))
    return
  }

  // Validate JSON fields
  let configSchema, inputSchema, outputSchema, uiConfig
  try {
    configSchema = JSON.parse(blockFormData.config_schema)
    inputSchema = JSON.parse(blockFormData.input_schema)
    outputSchema = JSON.parse(blockFormData.output_schema)
    uiConfig = JSON.parse(blockFormData.ui_config)
  } catch {
    toast.error(t('tenantBlocks.messages.invalidJson'))
    return
  }

  blocksLoading.value = true
  try {
    if (selectedBlock.value) {
      await blocks.update(selectedBlock.value.slug, {
        name: blockFormData.name,
        description: blockFormData.description || undefined,
        icon: blockFormData.icon || undefined,
        config_schema: configSchema,
        input_schema: inputSchema,
        output_schema: outputSchema,
        code: blockFormData.code || undefined,
        ui_config: uiConfig,
      })
      toast.success(t('tenantBlocks.messages.updated'))
    } else {
      await blocks.create({
        slug: blockFormData.slug,
        name: blockFormData.name,
        description: blockFormData.description || undefined,
        category: blockFormData.category,
        icon: blockFormData.icon || undefined,
        config_schema: configSchema,
        input_schema: inputSchema,
        output_schema: outputSchema,
        code: blockFormData.code || undefined,
        ui_config: uiConfig,
      })
      toast.success(t('tenantBlocks.messages.created'))
    }
    showBlockModal.value = false
    await fetchBlocks()
  } catch {
    toast.error(selectedBlock.value ? t('tenantBlocks.messages.updateFailed') : t('tenantBlocks.messages.createFailed'))
  } finally {
    blocksLoading.value = false
  }
}

function openDeleteBlockModal(block: BlockDefinition) {
  selectedBlock.value = block
  showDeleteBlockModal.value = true
}

async function confirmDeleteBlock() {
  if (!selectedBlock.value) return

  blocksLoading.value = true
  try {
    await blocks.remove(selectedBlock.value.slug)
    toast.success(t('tenantBlocks.messages.deleted'))
    showDeleteBlockModal.value = false
    selectedBlock.value = null
    await fetchBlocks()
  } catch {
    toast.error(t('tenantBlocks.messages.deleteFailed'))
  } finally {
    blocksLoading.value = false
  }
}

// ============================================================================
// Credentials Tab
// ============================================================================
const credentialsApi = useCredentials()
const credentialsLoading = ref(false)
const credentialsList = ref<Credential[]>([])
const showCredentialModal = ref(false)
const showDeleteCredentialModal = ref(false)
const selectedCredential = ref<Credential | null>(null)
const credentialFormData = reactive({
  name: '',
  description: '',
  credential_type: 'api_key' as CredentialType,
  provider: '',
  expires_at: '',
  // API Key fields
  api_key: '',
  header_name: 'Authorization',
  header_prefix: 'Bearer ',
  // Basic Auth fields
  username: '',
  password: '',
})

async function fetchCredentials() {
  credentialsLoading.value = true
  try {
    await credentialsApi.fetchCredentials()
    credentialsList.value = credentialsApi.credentials.value
  } catch {
    toast.error(t('credentials.messages.createFailed'))
  } finally {
    credentialsLoading.value = false
  }
}

function openAddCredentialModal() {
  selectedCredential.value = null
  credentialFormData.name = ''
  credentialFormData.description = ''
  credentialFormData.credential_type = 'api_key'
  credentialFormData.provider = ''
  credentialFormData.expires_at = ''
  credentialFormData.api_key = ''
  credentialFormData.header_name = 'Authorization'
  credentialFormData.header_prefix = 'Bearer '
  credentialFormData.username = ''
  credentialFormData.password = ''
  showCredentialModal.value = true
}

function openEditCredentialModal(credential: Credential) {
  selectedCredential.value = credential
  credentialFormData.name = credential.name
  credentialFormData.description = credential.description || ''
  credentialFormData.credential_type = credential.credential_type
  credentialFormData.provider = credential.metadata?.provider || ''
  credentialFormData.expires_at = credential.expires_at || ''
  // Clear secret fields - they need to be re-entered
  credentialFormData.api_key = ''
  credentialFormData.header_name = 'Authorization'
  credentialFormData.header_prefix = 'Bearer '
  credentialFormData.username = ''
  credentialFormData.password = ''
  showCredentialModal.value = true
}

async function submitCredentialForm() {
  if (!credentialFormData.name.trim()) {
    toast.error(t('common.error'))
    return
  }

  credentialsLoading.value = true
  try {
    const credentialData: CredentialData = {}

    if (credentialFormData.credential_type === 'api_key') {
      if (!selectedCredential.value && !credentialFormData.api_key) {
        toast.error(t('common.error'))
        credentialsLoading.value = false
        return
      }
      if (credentialFormData.api_key) {
        credentialData.api_key = credentialFormData.api_key
        credentialData.header_name = credentialFormData.header_name
        credentialData.header_prefix = credentialFormData.header_prefix
      }
    } else if (credentialFormData.credential_type === 'basic_auth') {
      if (!selectedCredential.value && (!credentialFormData.username || !credentialFormData.password)) {
        toast.error(t('common.error'))
        credentialsLoading.value = false
        return
      }
      if (credentialFormData.username) credentialData.username = credentialFormData.username
      if (credentialFormData.password) credentialData.password = credentialFormData.password
    }

    if (selectedCredential.value) {
      await credentialsApi.updateCredential(selectedCredential.value.id, {
        name: credentialFormData.name,
        description: credentialFormData.description || undefined,
        data: Object.keys(credentialData).length > 0 ? credentialData : undefined,
        metadata: credentialFormData.provider ? { provider: credentialFormData.provider } : undefined,
        expires_at: credentialFormData.expires_at || undefined,
      })
      toast.success(t('credentials.messages.updated'))
    } else {
      const request: CreateCredentialRequest = {
        name: credentialFormData.name,
        description: credentialFormData.description || undefined,
        credential_type: credentialFormData.credential_type,
        data: credentialData,
        metadata: credentialFormData.provider ? { provider: credentialFormData.provider } : undefined,
        expires_at: credentialFormData.expires_at || undefined,
      }
      await credentialsApi.createCredential(request)
      toast.success(t('credentials.messages.created'))
    }
    showCredentialModal.value = false
    await fetchCredentials()
  } catch {
    toast.error(selectedCredential.value ? t('credentials.messages.updateFailed') : t('credentials.messages.createFailed'))
  } finally {
    credentialsLoading.value = false
  }
}

function openDeleteCredentialModal(credential: Credential) {
  selectedCredential.value = credential
  showDeleteCredentialModal.value = true
}

async function confirmDeleteCredential() {
  if (!selectedCredential.value) return

  credentialsLoading.value = true
  try {
    await credentialsApi.deleteCredential(selectedCredential.value.id)
    toast.success(t('credentials.messages.deleted'))
    showDeleteCredentialModal.value = false
    selectedCredential.value = null
    await fetchCredentials()
  } catch {
    toast.error(t('credentials.messages.deleteFailed'))
  } finally {
    credentialsLoading.value = false
  }
}

async function revokeCredential(credential: Credential) {
  credentialsLoading.value = true
  try {
    await credentialsApi.revokeCredential(credential.id)
    toast.success(t('credentials.messages.revoked'))
    await fetchCredentials()
  } catch {
    toast.error(t('credentials.messages.revokeFailed'))
  } finally {
    credentialsLoading.value = false
  }
}

async function activateCredential(credential: Credential) {
  credentialsLoading.value = true
  try {
    await credentialsApi.activateCredential(credential.id)
    toast.success(t('credentials.messages.activated'))
    await fetchCredentials()
  } catch {
    toast.error(t('credentials.messages.activateFailed'))
  } finally {
    credentialsLoading.value = false
  }
}

// Format date
function formatDate(date: string | undefined): string {
  if (!date) return '-'
  return new Date(date).toLocaleDateString()
}

// ============================================================================
// Audit Logs Tab
// ============================================================================
const auditLogsApi = useAuditLogs()
const auditLogsLoading = ref(false)
const auditLogsList = ref<AuditLog[]>([])
const auditLogsTotal = ref(0)
const auditLogsPage = ref(1)
const auditLogsLimit = ref(50)

// Audit log filters
const filterResourceType = ref('')
const filterAction = ref<AuditAction | ''>('')
const filterFromDate = ref('')
const filterToDate = ref('')

// Resource types available
const resourceTypes = [
  'workflow',
  'step',
  'edge',
  'run',
  'schedule',
  'webhook',
  'credential',
  'block',
  'tenant',
]

const auditActions: AuditAction[] = ['create', 'update', 'delete', 'publish', 'execute', 'cancel', 'approve', 'reject']

async function fetchAuditLogs() {
  auditLogsLoading.value = true
  try {
    const filter: AuditLogFilter = {
      page: auditLogsPage.value,
      limit: auditLogsLimit.value,
    }
    if (filterResourceType.value) {
      filter.resource_type = filterResourceType.value
    }
    if (filterAction.value) {
      filter.action = filterAction.value
    }
    if (filterFromDate.value) {
      filter.from_date = filterFromDate.value
    }
    if (filterToDate.value) {
      filter.to_date = filterToDate.value
    }

    const result = await auditLogsApi.list(filter)
    auditLogsList.value = result.data
    auditLogsTotal.value = result.total
  } catch {
    toast.error(t('errors.loadFailed'))
  } finally {
    auditLogsLoading.value = false
  }
}

const auditLogsTotalPages = computed(() => Math.ceil(auditLogsTotal.value / auditLogsLimit.value))

function formatDateTime(date: string): string {
  return new Date(date).toLocaleString()
}

function getActionBadgeClass(action: AuditAction): string {
  switch (action) {
    case 'create':
    case 'approve':
      return 'badge-success'
    case 'update':
    case 'publish':
      return 'badge-info'
    case 'delete':
    case 'reject':
      return 'badge-danger'
    case 'execute':
      return 'badge-warning'
    case 'cancel':
      return 'badge-secondary'
    default:
      return 'badge-secondary'
  }
}

function formatChanges(changes: object | undefined): string {
  if (!changes || Object.keys(changes).length === 0) {
    return t('auditLogs.details.noChanges')
  }
  return JSON.stringify(changes, null, 2)
}

function clearAuditFilters() {
  filterResourceType.value = ''
  filterAction.value = ''
  filterFromDate.value = ''
  filterToDate.value = ''
}

function refreshAuditLogs() {
  fetchAuditLogs()
}

// Watch audit log filters
watch([filterResourceType, filterAction, filterFromDate, filterToDate], () => {
  auditLogsPage.value = 1
  fetchAuditLogs()
})

watch(auditLogsPage, () => {
  fetchAuditLogs()
})

// Load data when tab changes
watch(activeTab, (newTab) => {
  if (newTab === 'blocks') {
    fetchBlocks()
  } else if (newTab === 'credentials') {
    fetchCredentials()
  } else if (newTab === 'audit-logs') {
    fetchAuditLogs()
  }
})

const tabs = computed(() => [
  { id: 'general', label: t('settings.general') },
  { id: 'notifications', label: t('settings.notifications') },
  { id: 'blocks', label: t('tenantBlocks.title') },
  { id: 'credentials', label: t('credentials.title') },
  { id: 'audit-logs', label: t('auditLogs.title') },
  { id: 'team', label: t('settings.team') },
])

const categoryOptions = [
  { value: 'ai', label: 'AI' },
  { value: 'logic', label: 'Logic' },
  { value: 'data', label: 'Data' },
  { value: 'integration', label: 'Integration' },
  { value: 'control', label: 'Control' },
  { value: 'utility', label: 'Utility' },
]

const credentialTypes = [
  { value: 'api_key', label: t('credentials.type.api_key') },
  { value: 'basic_auth', label: t('credentials.type.basic_auth') },
  { value: 'oauth2', label: t('credentials.type.oauth2') },
  { value: 'custom', label: t('credentials.type.custom') },
]

function getStatusClass(status: CredentialStatus): string {
  switch (status) {
    case 'active': return 'status-active'
    case 'expired': return 'status-expired'
    case 'revoked': return 'status-revoked'
    default: return ''
  }
}
</script>

<template>
  <div>
    <div class="flex justify-between items-center mb-4">
      <h1 style="font-size: 1.5rem; font-weight: 600;">
        {{ $t('settings.title') }}
      </h1>
    </div>

    <div class="card">
      <!-- Tab navigation -->
      <div class="flex gap-4" style="border-bottom: 1px solid var(--color-border); padding: 0 1rem;">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          @click="activeTab = tab.id"
          :class="['tab-button', { active: activeTab === tab.id }]"
        >
          {{ tab.label }}
        </button>
      </div>

      <div style="padding: 1.5rem;">
        <!-- General Settings -->
        <div v-if="activeTab === 'general'">
          <h2 style="font-size: 1.125rem; font-weight: 600; margin-bottom: 1rem;">
            {{ $t('settings.general') }}
          </h2>

          <div class="form-group">
            <LanguageSwitcher />
          </div>

          <div class="form-group">
            <label class="form-label">{{ $t('settings.timezone') }}</label>
            <select v-model="generalSettings.timezone" class="form-input">
              <option value="Asia/Tokyo">Asia/Tokyo (JST)</option>
              <option value="UTC">UTC</option>
              <option value="America/New_York">America/New_York (EST)</option>
              <option value="Europe/London">Europe/London (GMT)</option>
            </select>
          </div>

          <div class="form-group">
            <label class="form-label">{{ $t('settings.dateFormat') }}</label>
            <select v-model="generalSettings.dateFormat" class="form-input">
              <option value="YYYY-MM-DD">YYYY-MM-DD</option>
              <option value="MM/DD/YYYY">MM/DD/YYYY</option>
              <option value="DD/MM/YYYY">DD/MM/YYYY</option>
            </select>
          </div>

          <div style="margin-top: 1.5rem; padding-top: 1.5rem; border-top: 1px solid var(--color-border);">
            <button @click="saveSettings" class="btn btn-primary" :disabled="saving">
              {{ saving ? $t('common.saving') : $t('common.save') }}
            </button>
          </div>
        </div>

        <!-- Notification Settings -->
        <div v-if="activeTab === 'notifications'">
          <h2 style="font-size: 1.125rem; font-weight: 600; margin-bottom: 1rem;">
            {{ $t('settings.notificationSettings') }}
          </h2>

          <div class="form-group">
            <label class="checkbox-label">
              <input type="checkbox" v-model="notificationSettings.emailOnFailure" />
              {{ $t('settings.emailOnFailure') }}
            </label>
          </div>

          <div class="form-group">
            <label class="checkbox-label">
              <input type="checkbox" v-model="notificationSettings.emailOnSuccess" />
              {{ $t('settings.emailOnSuccess') }}
            </label>
          </div>

          <div class="form-group">
            <label class="form-label">{{ $t('settings.slackWebhook') }}</label>
            <input
              type="text"
              v-model="notificationSettings.slackWebhook"
              class="form-input"
              placeholder="https://hooks.slack.com/services/..."
            />
            <p class="text-secondary" style="font-size: 0.875rem; margin-top: 0.25rem;">
              {{ $t('settings.slackWebhookHint') }}
            </p>
          </div>

          <div style="margin-top: 1.5rem; padding-top: 1.5rem; border-top: 1px solid var(--color-border);">
            <button @click="saveSettings" class="btn btn-primary" :disabled="saving">
              {{ saving ? $t('common.saving') : $t('common.save') }}
            </button>
          </div>
        </div>

        <!-- Blocks Tab -->
        <div v-if="activeTab === 'blocks'">
          <div class="flex justify-between items-center mb-4">
            <div>
              <h2 style="font-size: 1.125rem; font-weight: 600; margin-bottom: 0.25rem;">
                {{ $t('tenantBlocks.title') }}
              </h2>
              <p class="text-secondary">{{ $t('tenantBlocks.subtitle') }}</p>
            </div>
            <button class="btn btn-primary" @click="openAddBlockModal">
              + {{ $t('tenantBlocks.newBlock') }}
            </button>
          </div>

          <div v-if="blocksLoading && blocksList.length === 0" class="empty-state">
            <p class="text-secondary">{{ $t('common.loading') }}</p>
          </div>

          <div v-else-if="blocksList.length === 0" class="empty-state">
            <p class="text-secondary">{{ $t('tenantBlocks.noBlocks') }}</p>
            <p class="text-secondary" style="font-size: 0.875rem;">{{ $t('tenantBlocks.noBlocksDesc') }}</p>
          </div>

          <table v-else class="data-table">
            <thead>
              <tr>
                <th>{{ $t('tenantBlocks.table.name') }}</th>
                <th>{{ $t('tenantBlocks.table.slug') }}</th>
                <th>{{ $t('tenantBlocks.table.category') }}</th>
                <th>{{ $t('tenantBlocks.table.enabled') }}</th>
                <th>{{ $t('tenantBlocks.table.updatedAt') }}</th>
                <th>{{ $t('tenantBlocks.table.actions') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="block in blocksList" :key="block.id">
                <td>{{ block.name }}</td>
                <td><code>{{ block.slug }}</code></td>
                <td>{{ block.category }}</td>
                <td>
                  <span :class="['status-badge', block.enabled ? 'status-active' : 'status-inactive']">
                    {{ block.enabled ? $t('common.enabled') : $t('common.disabled') }}
                  </span>
                </td>
                <td>{{ formatDate(block.updated_at) }}</td>
                <td>
                  <div class="flex gap-2">
                    <button class="btn btn-sm btn-secondary" @click="openEditBlockModal(block)">
                      {{ $t('common.edit') }}
                    </button>
                    <button class="btn btn-sm btn-danger" @click="openDeleteBlockModal(block)">
                      {{ $t('common.delete') }}
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Credentials Tab -->
        <div v-if="activeTab === 'credentials'">
          <div class="flex justify-between items-center mb-4">
            <div>
              <h2 style="font-size: 1.125rem; font-weight: 600; margin-bottom: 0.25rem;">
                {{ $t('credentials.title') }}
              </h2>
              <p class="text-secondary">{{ $t('credentials.subtitle') }}</p>
            </div>
            <button class="btn btn-primary" @click="openAddCredentialModal">
              + {{ $t('credentials.newCredential') }}
            </button>
          </div>

          <!-- Security note -->
          <div class="info-card" style="margin-bottom: 1.5rem;">
            <p>{{ $t('credentials.securityNote') }}</p>
          </div>

          <div v-if="credentialsLoading && credentialsList.length === 0" class="empty-state">
            <p class="text-secondary">{{ $t('common.loading') }}</p>
          </div>

          <div v-else-if="credentialsList.length === 0" class="empty-state">
            <p class="text-secondary">{{ $t('credentials.noCredentials') }}</p>
            <p class="text-secondary" style="font-size: 0.875rem;">{{ $t('credentials.noCredentialsDesc') }}</p>
          </div>

          <table v-else class="data-table">
            <thead>
              <tr>
                <th>{{ $t('credentials.table.name') }}</th>
                <th>{{ $t('credentials.table.type') }}</th>
                <th>{{ $t('credentials.table.provider') }}</th>
                <th>{{ $t('credentials.table.status') }}</th>
                <th>{{ $t('credentials.table.expiresAt') }}</th>
                <th>{{ $t('credentials.table.actions') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="credential in credentialsList" :key="credential.id">
                <td>{{ credential.name }}</td>
                <td>{{ $t(`credentials.type.${credential.credential_type}`) }}</td>
                <td>{{ credential.metadata?.provider || '-' }}</td>
                <td>
                  <span :class="['status-badge', getStatusClass(credential.status)]">
                    {{ $t(`credentials.status.${credential.status}`) }}
                  </span>
                </td>
                <td>{{ formatDate(credential.expires_at) }}</td>
                <td>
                  <div class="flex gap-2">
                    <button class="btn btn-sm btn-secondary" @click="openEditCredentialModal(credential)">
                      {{ $t('common.edit') }}
                    </button>
                    <button
                      v-if="credential.status === 'active'"
                      class="btn btn-sm btn-warning"
                      @click="revokeCredential(credential)"
                    >
                      {{ $t('credentials.actions.revoke') }}
                    </button>
                    <button
                      v-if="credential.status === 'revoked'"
                      class="btn btn-sm btn-success"
                      @click="activateCredential(credential)"
                    >
                      {{ $t('credentials.actions.activate') }}
                    </button>
                    <button class="btn btn-sm btn-danger" @click="openDeleteCredentialModal(credential)">
                      {{ $t('common.delete') }}
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Audit Logs Tab -->
        <div v-if="activeTab === 'audit-logs'">
          <div class="flex justify-between items-center mb-4">
            <div>
              <h2 style="font-size: 1.125rem; font-weight: 600; margin-bottom: 0.25rem;">
                {{ $t('auditLogs.title') }}
              </h2>
              <p class="text-secondary">{{ $t('auditLogs.subtitle') }}</p>
            </div>
            <button class="btn btn-secondary" @click="refreshAuditLogs">
              {{ $t('auditLogs.refresh') }}
            </button>
          </div>

          <!-- Filters -->
          <div class="filters-bar">
            <select v-model="filterResourceType" class="filter-select">
              <option value="">{{ $t('auditLogs.allResources') }}</option>
              <option v-for="type in resourceTypes" :key="type" :value="type">
                {{ $t(`auditLogs.resourceType.${type}`) }}
              </option>
            </select>
            <select v-model="filterAction" class="filter-select">
              <option value="">{{ $t('auditLogs.allActions') }}</option>
              <option v-for="action in auditActions" :key="action" :value="action">
                {{ $t(`auditLogs.action.${action}`) }}
              </option>
            </select>
            <div class="date-filter">
              <label>{{ $t('auditLogs.from') }}:</label>
              <input v-model="filterFromDate" type="date" class="filter-input" />
            </div>
            <div class="date-filter">
              <label>{{ $t('auditLogs.to') }}:</label>
              <input v-model="filterToDate" type="date" class="filter-input" />
            </div>
            <button v-if="filterResourceType || filterAction || filterFromDate || filterToDate" class="btn btn-secondary btn-sm" @click="clearAuditFilters">
              {{ $t('workflows.clearFilters') }}
            </button>
          </div>

          <!-- Loading -->
          <div v-if="auditLogsLoading && auditLogsList.length === 0" class="empty-state">
            <p class="text-secondary">{{ $t('common.loading') }}</p>
          </div>

          <!-- Empty State -->
          <div v-else-if="auditLogsList.length === 0" class="empty-state">
            <p class="text-secondary">{{ $t('auditLogs.noLogs') }}</p>
            <p class="text-secondary" style="font-size: 0.875rem;">{{ $t('auditLogs.noLogsDesc') }}</p>
          </div>

          <!-- Audit Logs Table -->
          <div v-else>
            <table class="data-table">
              <thead>
                <tr>
                  <th>{{ $t('auditLogs.table.timestamp') }}</th>
                  <th>{{ $t('auditLogs.table.action') }}</th>
                  <th>{{ $t('auditLogs.table.resource') }}</th>
                  <th>{{ $t('auditLogs.table.user') }}</th>
                  <th>{{ $t('auditLogs.table.ipAddress') }}</th>
                  <th>{{ $t('auditLogs.table.changes') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="log in auditLogsList" :key="log.id">
                  <td>{{ formatDateTime(log.created_at) }}</td>
                  <td>
                    <span :class="['action-badge', getActionBadgeClass(log.action)]">
                      {{ $t(`auditLogs.action.${log.action}`) }}
                    </span>
                  </td>
                  <td>
                    <div class="resource-info">
                      <span class="resource-type">{{ $t(`auditLogs.resourceType.${log.resource_type}`) }}</span>
                      <code class="resource-id">{{ log.resource_id.substring(0, 8) }}...</code>
                    </div>
                  </td>
                  <td>{{ log.user_id || '-' }}</td>
                  <td>{{ log.ip_address || '-' }}</td>
                  <td>
                    <details v-if="log.changes && Object.keys(log.changes).length > 0" class="changes-details">
                      <summary>{{ $t('auditLogs.details.title') }}</summary>
                      <pre class="changes-content">{{ formatChanges(log.changes) }}</pre>
                    </details>
                    <span v-else class="no-changes">{{ $t('auditLogs.details.noChanges') }}</span>
                  </td>
                </tr>
              </tbody>
            </table>

            <!-- Pagination -->
            <div v-if="auditLogsTotalPages > 1" class="pagination">
              <button class="btn btn-sm btn-secondary" :disabled="auditLogsPage === 1" @click="auditLogsPage--">
                &lt;
              </button>
              <span class="page-info">{{ auditLogsPage }} / {{ auditLogsTotalPages }}</span>
              <button class="btn btn-sm btn-secondary" :disabled="auditLogsPage >= auditLogsTotalPages" @click="auditLogsPage++">
                &gt;
              </button>
            </div>
          </div>
        </div>

        <!-- Team Settings -->
        <div v-if="activeTab === 'team'">
          <h2 style="font-size: 1.125rem; font-weight: 600; margin-bottom: 1rem;">
            {{ $t('settings.teamMembers') }}
          </h2>

          <div class="card" style="padding: 2rem; text-align: center;">
            <p class="text-secondary">
              {{ $t('settings.teamComingSoon') }}
            </p>
            <p class="text-secondary" style="margin-top: 0.5rem;">
              {{ $t('settings.teamComingSoonDesc') }}
            </p>
          </div>
        </div>
      </div>
    </div>

    <!-- Block Modal -->
    <UiModal
      :show="showBlockModal"
      :title="selectedBlock ? $t('tenantBlocks.editBlock') : $t('tenantBlocks.newBlock')"
      size="lg"
      @close="showBlockModal = false"
    >
      <form @submit.prevent="submitBlockForm">
        <div class="form-group">
          <label class="form-label">{{ $t('tenantBlocks.form.name') }} *</label>
          <input
            v-model="blockFormData.name"
            type="text"
            class="form-input"
            :placeholder="$t('tenantBlocks.form.namePlaceholder')"
            required
          />
        </div>

        <div class="form-group">
          <label class="form-label">{{ $t('tenantBlocks.form.slug') }} *</label>
          <input
            v-model="blockFormData.slug"
            type="text"
            class="form-input"
            :placeholder="$t('tenantBlocks.form.slugPlaceholder')"
            required
            :disabled="!!selectedBlock"
          />
          <p class="text-secondary" style="font-size: 0.75rem; margin-top: 0.25rem;">
            {{ $t('tenantBlocks.form.slugHint') }}
          </p>
        </div>

        <div class="form-group">
          <label class="form-label">{{ $t('tenantBlocks.form.description') }}</label>
          <textarea
            v-model="blockFormData.description"
            class="form-input"
            rows="2"
            :placeholder="$t('tenantBlocks.form.descriptionPlaceholder')"
          />
        </div>

        <div class="form-row">
          <div class="form-group">
            <label class="form-label">{{ $t('tenantBlocks.form.category') }}</label>
            <select v-model="blockFormData.category" class="form-input" :disabled="!!selectedBlock">
              <option v-for="cat in categoryOptions" :key="cat.value" :value="cat.value">
                {{ cat.label }}
              </option>
            </select>
          </div>

        </div>

        <div class="form-group">
          <label class="form-label">{{ $t('tenantBlocks.form.configSchema') }}</label>
          <textarea
            v-model="blockFormData.config_schema"
            class="form-input code-input"
            rows="4"
          />
        </div>

        <div class="form-group">
          <label class="form-label">{{ $t('tenantBlocks.form.code') }}</label>
          <textarea
            v-model="blockFormData.code"
            class="form-input code-input"
            rows="8"
            :placeholder="$t('tenantBlocks.form.codePlaceholder')"
          />
        </div>

        <div class="form-group">
          <label class="form-label">{{ $t('tenantBlocks.form.uiConfig') }}</label>
          <textarea
            v-model="blockFormData.ui_config"
            class="form-input code-input"
            rows="4"
          />
        </div>
      </form>

      <template #footer>
        <button class="btn btn-secondary" @click="showBlockModal = false">
          {{ $t('common.cancel') }}
        </button>
        <button class="btn btn-primary" :disabled="blocksLoading" @click="submitBlockForm">
          {{ blocksLoading ? $t('common.saving') : $t('common.save') }}
        </button>
      </template>
    </UiModal>

    <!-- Delete Block Modal -->
    <UiModal
      :show="showDeleteBlockModal"
      :title="$t('tenantBlocks.editBlock')"
      size="sm"
      @close="showDeleteBlockModal = false"
    >
      <p>{{ $t('tenantBlocks.confirmDelete') }}</p>

      <template #footer>
        <button class="btn btn-secondary" @click="showDeleteBlockModal = false">
          {{ $t('common.cancel') }}
        </button>
        <button class="btn btn-danger" :disabled="blocksLoading" @click="confirmDeleteBlock">
          {{ $t('common.delete') }}
        </button>
      </template>
    </UiModal>

    <!-- Credential Modal -->
    <UiModal
      :show="showCredentialModal"
      :title="selectedCredential ? $t('credentials.editCredential') : $t('credentials.newCredential')"
      size="md"
      @close="showCredentialModal = false"
    >
      <form @submit.prevent="submitCredentialForm">
        <div class="form-group">
          <label class="form-label">{{ $t('credentials.form.name') }} *</label>
          <input
            v-model="credentialFormData.name"
            type="text"
            class="form-input"
            :placeholder="$t('credentials.form.namePlaceholder')"
            required
          />
        </div>

        <div class="form-group">
          <label class="form-label">{{ $t('credentials.form.description') }}</label>
          <textarea
            v-model="credentialFormData.description"
            class="form-input"
            rows="2"
            :placeholder="$t('credentials.form.descriptionPlaceholder')"
          />
        </div>

        <div class="form-group">
          <label class="form-label">{{ $t('credentials.form.credentialType') }}</label>
          <select v-model="credentialFormData.credential_type" class="form-input" :disabled="!!selectedCredential">
            <option v-for="ct in credentialTypes" :key="ct.value" :value="ct.value">
              {{ ct.label }}
            </option>
          </select>
        </div>

        <div class="form-group">
          <label class="form-label">{{ $t('credentials.form.provider') }}</label>
          <input
            v-model="credentialFormData.provider"
            type="text"
            class="form-input"
            :placeholder="$t('credentials.form.providerPlaceholder')"
          />
        </div>

        <!-- API Key fields -->
        <template v-if="credentialFormData.credential_type === 'api_key'">
          <div class="form-group">
            <label class="form-label">
              {{ $t('credentials.form.apiKey') }} {{ selectedCredential ? '' : '*' }}
            </label>
            <input
              v-model="credentialFormData.api_key"
              type="password"
              class="form-input"
              :placeholder="$t('credentials.form.apiKeyPlaceholder')"
              :required="!selectedCredential"
            />
            <p v-if="selectedCredential" class="text-secondary" style="font-size: 0.75rem; margin-top: 0.25rem;">
              {{ $t('credentials.form.valueHidden') }}
            </p>
          </div>

          <div class="form-row">
            <div class="form-group">
              <label class="form-label">{{ $t('credentials.form.headerName') }}</label>
              <input
                v-model="credentialFormData.header_name"
                type="text"
                class="form-input"
                :placeholder="$t('credentials.form.headerNamePlaceholder')"
              />
            </div>
            <div class="form-group">
              <label class="form-label">{{ $t('credentials.form.headerPrefix') }}</label>
              <input
                v-model="credentialFormData.header_prefix"
                type="text"
                class="form-input"
                :placeholder="$t('credentials.form.headerPrefixPlaceholder')"
              />
            </div>
          </div>
        </template>

        <!-- Basic Auth fields -->
        <template v-if="credentialFormData.credential_type === 'basic_auth'">
          <div class="form-group">
            <label class="form-label">
              {{ $t('credentials.form.username') }} {{ selectedCredential ? '' : '*' }}
            </label>
            <input
              v-model="credentialFormData.username"
              type="text"
              class="form-input"
              :placeholder="$t('credentials.form.usernamePlaceholder')"
              :required="!selectedCredential"
            />
          </div>

          <div class="form-group">
            <label class="form-label">
              {{ $t('credentials.form.password') }} {{ selectedCredential ? '' : '*' }}
            </label>
            <input
              v-model="credentialFormData.password"
              type="password"
              class="form-input"
              :placeholder="$t('credentials.form.passwordPlaceholder')"
              :required="!selectedCredential"
            />
            <p v-if="selectedCredential" class="text-secondary" style="font-size: 0.75rem; margin-top: 0.25rem;">
              {{ $t('credentials.form.valueHidden') }}
            </p>
          </div>
        </template>

        <div class="form-group">
          <label class="form-label">{{ $t('credentials.form.expiresAt') }}</label>
          <input
            v-model="credentialFormData.expires_at"
            type="datetime-local"
            class="form-input"
          />
        </div>
      </form>

      <template #footer>
        <button class="btn btn-secondary" @click="showCredentialModal = false">
          {{ $t('common.cancel') }}
        </button>
        <button class="btn btn-primary" :disabled="credentialsLoading" @click="submitCredentialForm">
          {{ credentialsLoading ? $t('common.saving') : $t('common.save') }}
        </button>
      </template>
    </UiModal>

    <!-- Delete Credential Modal -->
    <UiModal
      :show="showDeleteCredentialModal"
      :title="$t('credentials.editCredential')"
      size="sm"
      @close="showDeleteCredentialModal = false"
    >
      <p>{{ $t('credentials.confirmDelete') }}</p>

      <template #footer>
        <button class="btn btn-secondary" @click="showDeleteCredentialModal = false">
          {{ $t('common.cancel') }}
        </button>
        <button class="btn btn-danger" :disabled="credentialsLoading" @click="confirmDeleteCredential">
          {{ $t('common.delete') }}
        </button>
      </template>
    </UiModal>
  </div>
</template>

<style scoped>
.tab-button {
  padding: 0.75rem 0;
  background: none;
  border: none;
  color: var(--color-text-secondary);
  cursor: pointer;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
  transition: all 0.2s;
}

.tab-button:hover {
  color: var(--color-text);
}

.tab-button.active {
  color: var(--color-primary);
  border-bottom-color: var(--color-primary);
}

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
}

.form-input {
  width: 100%;
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--color-border);
  border-radius: 0.375rem;
  background: var(--color-bg);
  color: var(--color-text);
}

.form-input:focus {
  outline: none;
  border-color: var(--color-primary);
}

.code-input {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.8125rem;
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
}

.checkbox-label input[type="checkbox"] {
  width: 1rem;
  height: 1rem;
}

.empty-state {
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  padding: 2rem;
  text-align: center;
}

.info-card {
  background: #dbeafe;
  border: 1px solid #93c5fd;
  border-radius: var(--radius);
  padding: 0.75rem 1rem;
}

.info-card p {
  color: #1e40af;
  font-size: 0.875rem;
  margin: 0;
}

.data-table {
  width: 100%;
  border-collapse: collapse;
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
}

.data-table th,
.data-table td {
  padding: 0.75rem 1rem;
  text-align: left;
  border-bottom: 1px solid var(--color-border);
}

.data-table th {
  font-weight: 600;
  font-size: 0.875rem;
  color: var(--color-text-secondary);
}

.data-table tr:last-child td {
  border-bottom: none;
}

.data-table code {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.8125rem;
  background: var(--color-surface);
  padding: 0.125rem 0.375rem;
  border-radius: 0.25rem;
}

.status-badge {
  display: inline-flex;
  align-items: center;
  padding: 0.25rem 0.5rem;
  border-radius: 9999px;
  font-size: 0.75rem;
  font-weight: 500;
}

.status-active {
  background: rgba(34, 197, 94, 0.1);
  color: #22c55e;
}

.status-inactive,
.status-revoked {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
}

.status-expired {
  background: rgba(245, 158, 11, 0.1);
  color: #f59e0b;
}

.btn-sm {
  padding: 0.375rem 0.75rem;
  font-size: 0.8125rem;
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

/* Audit Logs styles */
.filters-bar {
  display: flex;
  gap: 1rem;
  margin-bottom: 1rem;
  flex-wrap: wrap;
  align-items: center;
}

.filter-select {
  padding: 0.5rem 1rem;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  background: var(--color-surface);
  min-width: 140px;
}

.date-filter {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.date-filter label {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
}

.filter-input {
  padding: 0.5rem;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  background: var(--color-surface);
}

.action-badge {
  display: inline-block;
  padding: 0.25rem 0.5rem;
  border-radius: var(--radius-sm, 0.25rem);
  font-size: 0.75rem;
  font-weight: 500;
}

.badge-success {
  background: #dcfce7;
  color: #166534;
}

.badge-info {
  background: #dbeafe;
  color: #1e40af;
}

.badge-warning {
  background: #fef3c7;
  color: #92400e;
}

.badge-danger {
  background: #fee2e2;
  color: #991b1b;
}

.badge-secondary {
  background: #f3f4f6;
  color: #374151;
}

.resource-info {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.resource-type {
  font-weight: 500;
}

.resource-id {
  background: var(--color-background);
  padding: 0.125rem 0.25rem;
  border-radius: var(--radius-sm, 0.25rem);
  font-size: 0.75rem;
}

.changes-details {
  cursor: pointer;
}

.changes-details summary {
  color: var(--color-primary);
  font-size: 0.875rem;
}

.changes-content {
  background: var(--color-background);
  padding: 0.5rem;
  border-radius: var(--radius-sm, 0.25rem);
  font-size: 0.75rem;
  max-width: 300px;
  overflow-x: auto;
  margin: 0.5rem 0 0;
}

.no-changes {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

.pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 1rem;
  margin-top: 1rem;
  padding: 1rem;
}

.page-info {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
}
</style>
