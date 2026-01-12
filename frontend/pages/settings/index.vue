<script setup lang="ts">
const { t } = useI18n()

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

// Environment variable types
type EnvVarScope = 'system_block' | 'tenant_block'

interface EnvVar {
  id: string
  key: string
  scope: EnvVarScope
  created_at: string
  updated_at: string
}

// Environment variables state
const envVarsLoading = ref(false)
const systemBlockVars = ref<EnvVar[]>([])
const tenantBlockVars = ref<EnvVar[]>([])
const showEnvVarModal = ref(false)
const showDeleteEnvVarModal = ref(false)
const selectedEnvVar = ref<EnvVar | null>(null)
const currentEnvVarScope = ref<EnvVarScope>('system_block')
const envVarFormData = reactive({
  key: '',
  value: '',
})

// Fetch environment variables
async function fetchEnvVars() {
  envVarsLoading.value = true
  try {
    // TODO: Implement API call
    systemBlockVars.value = []
    tenantBlockVars.value = []
  } finally {
    envVarsLoading.value = false
  }
}

// Open add env var modal
function openAddEnvVarModal(scope: EnvVarScope) {
  envVarFormData.key = ''
  envVarFormData.value = ''
  currentEnvVarScope.value = scope
  selectedEnvVar.value = null
  showEnvVarModal.value = true
}

// Open edit env var modal
function openEditEnvVarModal(envVar: EnvVar) {
  selectedEnvVar.value = envVar
  currentEnvVarScope.value = envVar.scope
  envVarFormData.key = envVar.key
  envVarFormData.value = ''
  showEnvVarModal.value = true
}

// Submit env var form
async function submitEnvVarForm() {
  if (!envVarFormData.key.trim()) {
    showMessage('error', t('credentials.form.keyRequired'))
    return
  }
  if (!/^[A-Z0-9_]+$/.test(envVarFormData.key)) {
    showMessage('error', t('credentials.form.keyHint'))
    return
  }

  envVarsLoading.value = true
  try {
    if (selectedEnvVar.value) {
      // TODO: Implement API call
      showMessage('success', t('credentials.messages.updated'))
    } else {
      // TODO: Implement API call
      showMessage('success', t('credentials.messages.created'))
    }
    showEnvVarModal.value = false
    await fetchEnvVars()
  } catch {
    showMessage('error', selectedEnvVar.value ? t('credentials.messages.updateFailed') : t('credentials.messages.createFailed'))
  } finally {
    envVarsLoading.value = false
  }
}

// Open delete confirmation
function openDeleteEnvVarModal(envVar: EnvVar) {
  selectedEnvVar.value = envVar
  showDeleteEnvVarModal.value = true
}

// Confirm delete
async function confirmDeleteEnvVar() {
  if (!selectedEnvVar.value) return

  envVarsLoading.value = true
  try {
    // TODO: Implement API call
    showMessage('success', t('credentials.messages.deleted'))
    showDeleteEnvVarModal.value = false
    selectedEnvVar.value = null
    await fetchEnvVars()
  } catch {
    showMessage('error', t('credentials.messages.deleteFailed'))
  } finally {
    envVarsLoading.value = false
  }
}

// Format date
function formatDate(date: string | undefined): string {
  if (!date) return '-'
  return new Date(date).toLocaleDateString()
}

const saving = ref(false)
const message = ref<{ type: 'success' | 'error'; text: string } | null>(null)

function showMessage(type: 'success' | 'error', text: string) {
  message.value = { type, text }
  setTimeout(() => {
    message.value = null
  }, 3000)
}

async function saveSettings() {
  saving.value = true
  message.value = null

  // Simulate save - in real implementation, this would call the API
  await new Promise(resolve => setTimeout(resolve, 500))

  message.value = {
    type: 'success',
    text: t('common.success'),
  }
  saving.value = false

  // Clear message after 3 seconds
  setTimeout(() => {
    message.value = null
  }, 3000)
}

// Load env vars when credentials tab is active
watch(activeTab, (newTab) => {
  if (newTab === 'credentials') {
    fetchEnvVars()
  }
})

const tabs = computed(() => [
  { id: 'general', label: t('settings.general') },
  { id: 'notifications', label: t('settings.notifications') },
  { id: 'credentials', label: t('credentials.title') },
  { id: 'team', label: t('settings.team') },
])
</script>

<template>
  <div>
    <div class="flex justify-between items-center mb-4">
      <h1 style="font-size: 1.5rem; font-weight: 600;">
        {{ $t('settings.title') }}
      </h1>
    </div>

    <!-- Success/Error message -->
    <div
      v-if="message"
      :class="['card', message.type === 'success' ? 'bg-success' : 'bg-error']"
      style="padding: 0.75rem 1rem; margin-bottom: 1rem;"
    >
      {{ message.text }}
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

          <!-- Language Switcher -->
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
        </div>

        <!-- Environment Variables -->
        <div v-if="activeTab === 'credentials'">
          <h2 style="font-size: 1.125rem; font-weight: 600; margin-bottom: 0.5rem;">
            {{ $t('credentials.title') }}
          </h2>
          <p class="text-secondary" style="margin-bottom: 1rem;">
            {{ $t('credentials.subtitle') }}
          </p>

          <!-- Security note -->
          <div class="info-card" style="margin-bottom: 1.5rem;">
            <p>{{ $t('credentials.securityNote') }}</p>
          </div>

          <!-- System Block Section -->
          <div class="env-section">
            <div class="env-section-header">
              <div>
                <h3 class="env-section-title">{{ $t('credentials.systemBlockSection') }}</h3>
                <p class="text-secondary" style="font-size: 0.875rem;">{{ $t('credentials.systemBlockDesc') }}</p>
              </div>
              <button class="btn btn-primary btn-sm" @click="openAddEnvVarModal('system_block')">
                {{ $t('credentials.addVariable') }}
              </button>
            </div>

            <div v-if="envVarsLoading && systemBlockVars.length === 0" class="env-empty-state">
              <p class="text-secondary">{{ $t('common.loading') }}</p>
            </div>

            <div v-else-if="systemBlockVars.length === 0" class="env-empty-state">
              <p class="text-secondary">{{ $t('credentials.noCredentials') }}</p>
            </div>

            <table v-else class="env-table">
              <thead>
                <tr>
                  <th>{{ $t('credentials.table.key') }}</th>
                  <th>{{ $t('credentials.table.updatedAt') }}</th>
                  <th>{{ $t('credentials.table.actions') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="envVar in systemBlockVars" :key="envVar.id">
                  <td><code>{{ envVar.key }}</code></td>
                  <td>{{ formatDate(envVar.updated_at) }}</td>
                  <td>
                    <div class="flex gap-2">
                      <button class="btn btn-sm btn-secondary" @click="openEditEnvVarModal(envVar)">
                        {{ $t('common.edit') }}
                      </button>
                      <button class="btn btn-sm btn-danger" @click="openDeleteEnvVarModal(envVar)">
                        {{ $t('common.delete') }}
                      </button>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <!-- Tenant Block Section -->
          <div class="env-section" style="margin-top: 2rem;">
            <div class="env-section-header">
              <div>
                <h3 class="env-section-title">{{ $t('credentials.tenantBlockSection') }}</h3>
                <p class="text-secondary" style="font-size: 0.875rem;">{{ $t('credentials.tenantBlockDesc') }}</p>
              </div>
              <button class="btn btn-primary btn-sm" @click="openAddEnvVarModal('tenant_block')">
                {{ $t('credentials.addVariable') }}
              </button>
            </div>

            <div v-if="envVarsLoading && tenantBlockVars.length === 0" class="env-empty-state">
              <p class="text-secondary">{{ $t('common.loading') }}</p>
            </div>

            <div v-else-if="tenantBlockVars.length === 0" class="env-empty-state">
              <p class="text-secondary">{{ $t('credentials.noCredentials') }}</p>
            </div>

            <table v-else class="env-table">
              <thead>
                <tr>
                  <th>{{ $t('credentials.table.key') }}</th>
                  <th>{{ $t('credentials.table.updatedAt') }}</th>
                  <th>{{ $t('credentials.table.actions') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="envVar in tenantBlockVars" :key="envVar.id">
                  <td><code>{{ envVar.key }}</code></td>
                  <td>{{ formatDate(envVar.updated_at) }}</td>
                  <td>
                    <div class="flex gap-2">
                      <button class="btn btn-sm btn-secondary" @click="openEditEnvVarModal(envVar)">
                        {{ $t('common.edit') }}
                      </button>
                      <button class="btn btn-sm btn-danger" @click="openDeleteEnvVarModal(envVar)">
                        {{ $t('common.delete') }}
                      </button>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
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

        <!-- Save button (only for general/notifications tabs) -->
        <div v-if="activeTab === 'general' || activeTab === 'notifications'" style="margin-top: 1.5rem; padding-top: 1.5rem; border-top: 1px solid var(--color-border);">
          <button
            @click="saveSettings"
            class="btn btn-primary"
            :disabled="saving"
          >
            {{ saving ? $t('common.saving') : $t('common.save') }}
          </button>
        </div>
      </div>
    </div>

    <!-- Add/Edit Environment Variable Modal -->
    <UiModal
      :show="showEnvVarModal"
      :title="selectedEnvVar ? $t('credentials.editVariable') : $t('credentials.addVariable')"
      size="md"
      @close="showEnvVarModal = false"
    >
      <form @submit.prevent="submitEnvVarForm">
        <div class="form-group">
          <label class="form-label">
            {{ currentEnvVarScope === 'system_block' ? $t('credentials.systemBlockSection') : $t('credentials.tenantBlockSection') }}
          </label>
          <p class="text-secondary" style="font-size: 0.75rem;">
            {{ currentEnvVarScope === 'system_block' ? $t('credentials.systemBlockDesc') : $t('credentials.tenantBlockDesc') }}
          </p>
        </div>

        <div class="form-group">
          <label class="form-label">{{ $t('credentials.form.key') }} *</label>
          <input
            v-model="envVarFormData.key"
            type="text"
            class="form-input"
            style="max-width: 100%;"
            :placeholder="$t('credentials.form.keyPlaceholder')"
            required
            :disabled="!!selectedEnvVar"
          />
          <p class="text-secondary" style="font-size: 0.75rem; margin-top: 0.25rem;">
            {{ $t('credentials.form.keyHint') }}
          </p>
        </div>

        <div class="form-group">
          <label class="form-label">
            {{ $t('credentials.form.value') }} {{ selectedEnvVar ? '' : '*' }}
          </label>
          <input
            v-model="envVarFormData.value"
            type="password"
            class="form-input"
            style="max-width: 100%;"
            :placeholder="$t('credentials.form.valuePlaceholder')"
            :required="!selectedEnvVar"
          />
          <p v-if="selectedEnvVar" class="text-secondary" style="font-size: 0.75rem; margin-top: 0.25rem;">
            {{ $t('credentials.form.valueHidden') }}
          </p>
        </div>
      </form>

      <template #footer>
        <button class="btn btn-secondary" @click="showEnvVarModal = false">
          {{ $t('common.cancel') }}
        </button>
        <button class="btn btn-primary" :disabled="envVarsLoading" @click="submitEnvVarForm">
          {{ envVarsLoading ? $t('common.saving') : $t('common.save') }}
        </button>
      </template>
    </UiModal>

    <!-- Delete Confirmation Modal -->
    <UiModal
      :show="showDeleteEnvVarModal"
      :title="$t('credentials.deleteVariable')"
      size="sm"
      @close="showDeleteEnvVarModal = false"
    >
      <p>{{ $t('credentials.actions.confirmDelete') }}</p>

      <template #footer>
        <button class="btn btn-secondary" @click="showDeleteEnvVarModal = false">
          {{ $t('common.cancel') }}
        </button>
        <button class="btn btn-danger" :disabled="envVarsLoading" @click="confirmDeleteEnvVar">
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

.form-label {
  display: block;
  font-weight: 500;
  margin-bottom: 0.5rem;
}

.form-input {
  width: 100%;
  max-width: 400px;
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

.bg-success {
  background: rgba(34, 197, 94, 0.1);
  color: #22c55e;
}

.bg-error {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
}

/* Environment variables styles */
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

.env-section {
  margin-bottom: 1.5rem;
}

.env-section-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 0.75rem;
}

.env-section-title {
  font-size: 1rem;
  font-weight: 600;
  margin: 0 0 0.25rem;
}

.env-empty-state {
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  padding: 1.5rem;
  text-align: center;
}

.env-table {
  width: 100%;
  border-collapse: collapse;
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
}

.env-table th,
.env-table td {
  padding: 0.75rem 1rem;
  text-align: left;
  border-bottom: 1px solid var(--color-border);
}

.env-table th {
  font-weight: 600;
  font-size: 0.875rem;
  color: var(--color-text-secondary);
}

.env-table tr:last-child td {
  border-bottom: none;
}

.env-table code {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.8125rem;
  background: var(--color-surface);
  padding: 0.125rem 0.375rem;
  border-radius: 0.25rem;
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
</style>
