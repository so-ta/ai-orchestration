<script setup lang="ts">
/**
 * ReleaseModal.vue
 * リリース（バージョンスナップショット）作成モーダル
 * 公開前チェックリストを統合
 */

import type { Step, Edge, BlockDefinition } from '~/types/api'

interface CheckResult {
  id: string
  label: string
  status: 'passed' | 'warning' | 'error' | 'checking'
  message?: string
}

const { t } = useI18n()

const props = defineProps<{
  show: boolean
  projectName?: string
  steps: Step[]
  edges: Edge[]
  blockDefinitions: BlockDefinition[]
}>()

const emit = defineEmits<{
  close: []
  create: [name: string, description: string]
}>()

const releaseName = ref('')
const releaseDescription = ref('')
const creating = ref(false)

// Validation state
const isChecking = ref(true)
const checks = ref<CheckResult[]>([])

// Reset form and run checks when modal opens
watch(() => props.show, async (show) => {
  if (show) {
    releaseName.value = ''
    releaseDescription.value = ''
    await runChecks()
  }
})

// Re-run checks when steps/edges change while modal is open
watch([() => props.steps, () => props.edges], async () => {
  if (props.show) {
    await runChecks()
  }
}, { deep: true })

// Run all validation checks
async function runChecks() {
  isChecking.value = true
  checks.value = [
    { id: 'hasStartBlock', label: t('publishChecklist.checks.hasStartBlock'), status: 'checking' },
    { id: 'allConnected', label: t('publishChecklist.checks.allConnected'), status: 'checking' },
    { id: 'noLoop', label: t('publishChecklist.checks.noLoop'), status: 'checking' },
    { id: 'credentialsSet', label: t('publishChecklist.checks.credentialsSet'), status: 'checking' },
  ]

  // Simulate async check delay for better UX
  await new Promise(resolve => setTimeout(resolve, 200))

  // Check 1: Start block exists
  const startBlockTypes = ['start', 'manual_trigger', 'schedule_trigger', 'webhook_trigger']
  const hasStartBlock = props.steps.some(step => startBlockTypes.includes(step.type))
  updateCheck('hasStartBlock', hasStartBlock ? 'passed' : 'error',
    hasStartBlock ? undefined : t('publishChecklist.checks.hasStartBlockError'))

  // Check 2: All blocks connected
  const { allConnected, unconnectedCount } = checkAllConnected()
  updateCheck('allConnected',
    allConnected ? 'passed' : 'warning',
    allConnected ? undefined : t('publishChecklist.checks.allConnectedWarning', { count: unconnectedCount }))

  // Check 3: No infinite loops
  const hasCycle = checkForCycles()
  updateCheck('noLoop',
    hasCycle ? 'error' : 'passed',
    hasCycle ? t('publishChecklist.checks.noLoopError') : undefined)

  // Check 4: Credentials configured
  const { allSet, missingCount } = checkCredentials()
  updateCheck('credentialsSet',
    allSet ? 'passed' : 'warning',
    allSet ? undefined : t('publishChecklist.checks.credentialsSetWarning', { count: missingCount }))

  isChecking.value = false
}

function updateCheck(id: string, status: CheckResult['status'], message?: string) {
  const check = checks.value.find(c => c.id === id)
  if (check) {
    check.status = status
    check.message = message
  }
}

// Check if all blocks are connected
function checkAllConnected(): { allConnected: boolean; unconnectedCount: number } {
  if (props.steps.length <= 1) {
    return { allConnected: true, unconnectedCount: 0 }
  }

  const connectedIds = new Set<string>()
  for (const edge of props.edges) {
    if (edge.source_step_id) connectedIds.add(edge.source_step_id)
    if (edge.target_step_id) connectedIds.add(edge.target_step_id)
  }

  const unconnectedCount = props.steps.filter(step => !connectedIds.has(step.id)).length
  return {
    allConnected: unconnectedCount === 0,
    unconnectedCount,
  }
}

// Check for cycles using DFS
function checkForCycles(): boolean {
  const adj = new Map<string, string[]>()
  for (const edge of props.edges) {
    if (edge.source_step_id && edge.target_step_id) {
      if (!adj.has(edge.source_step_id)) {
        adj.set(edge.source_step_id, [])
      }
      adj.get(edge.source_step_id)!.push(edge.target_step_id)
    }
  }

  const state = new Map<string, number>() // 0=unvisited, 1=visiting, 2=visited
  for (const step of props.steps) {
    state.set(step.id, 0)
  }

  function dfs(id: string): boolean {
    state.set(id, 1)
    for (const neighbor of adj.get(id) || []) {
      if (state.get(neighbor) === 1) return true // back edge = cycle
      if (state.get(neighbor) === 0 && dfs(neighbor)) return true
    }
    state.set(id, 2)
    return false
  }

  for (const step of props.steps) {
    if (state.get(step.id) === 0 && dfs(step.id)) {
      return true
    }
  }
  return false
}

// Check if credentials are configured
function checkCredentials(): { allSet: boolean; missingCount: number } {
  let missingCount = 0

  for (const step of props.steps) {
    const blockDef = props.blockDefinitions.find(b => b.slug === step.type)
    if (!blockDef?.required_credentials?.length) continue

    // Check if step has credential bindings for required credentials
    const bindings = step.credential_bindings || {}
    for (const required of blockDef.required_credentials) {
      if (!bindings[required]) {
        missingCount++
      }
    }
  }

  return {
    allSet: missingCount === 0,
    missingCount,
  }
}

// Computed properties
const hasErrors = computed(() => checks.value.some(c => c.status === 'error'))
const hasWarnings = computed(() => checks.value.some(c => c.status === 'warning'))
const passedCount = computed(() => checks.value.filter(c => c.status === 'passed').length)
const totalCount = computed(() => checks.value.length)

async function handleCreate() {
  if (!releaseName.value.trim()) return

  creating.value = true
  try {
    emit('create', releaseName.value.trim(), releaseDescription.value.trim())
  } finally {
    creating.value = false
  }
}

function handleClose() {
  if (!creating.value) {
    emit('close')
  }
}
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="show" class="modal-overlay" @click.self="handleClose">
        <div class="modal-content">
          <div class="modal-header">
            <h3>{{ t('editor.createReleaseTitle') }}</h3>
            <button class="close-btn" @click="handleClose">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="18" y1="6" x2="6" y2="18" />
                <line x1="6" y1="6" x2="18" y2="18" />
              </svg>
            </button>
          </div>

          <div class="modal-body">
            <p class="modal-description">
              {{ t('editor.createReleaseDescription') }}
            </p>

            <div class="form-group">
              <label for="releaseName">{{ t('editor.releaseName') }}</label>
              <input
                id="releaseName"
                v-model="releaseName"
                type="text"
                class="form-input"
                :placeholder="t('editor.releaseNamePlaceholder')"
                autofocus
                @keyup.enter="handleCreate"
              >
            </div>

            <div class="form-group">
              <label for="releaseDescription">{{ t('editor.releaseDescription') }}</label>
              <textarea
                id="releaseDescription"
                v-model="releaseDescription"
                class="form-textarea"
                :placeholder="t('editor.releaseDescriptionPlaceholder')"
                rows="2"
              />
            </div>

            <!-- Publish Checklist Section -->
            <div class="checklist-section">
              <div class="checklist-header">
                <span class="checklist-title">{{ t('publishChecklist.title') }}</span>
                <span v-if="!isChecking" class="checklist-summary" :class="{ error: hasErrors, warning: hasWarnings && !hasErrors }">
                  {{ passedCount }}/{{ totalCount }}
                </span>
              </div>

              <div class="checklist">
                <div
                  v-for="check in checks"
                  :key="check.id"
                  class="check-item"
                  :class="check.status"
                >
                  <div class="check-icon">
                    <svg v-if="check.status === 'checking'" class="spin" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
                    </svg>
                    <svg v-else-if="check.status === 'passed'" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <polyline points="20 6 9 17 4 12"/>
                    </svg>
                    <svg v-else-if="check.status === 'warning'" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/>
                      <line x1="12" y1="9" x2="12" y2="13"/>
                      <line x1="12" y1="17" x2="12.01" y2="17"/>
                    </svg>
                    <svg v-else xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <circle cx="12" cy="12" r="10"/>
                      <line x1="15" y1="9" x2="9" y2="15"/>
                      <line x1="9" y1="9" x2="15" y2="15"/>
                    </svg>
                  </div>
                  <div class="check-content">
                    <span class="check-label">{{ check.label }}</span>
                    <span v-if="check.message" class="check-message">{{ check.message }}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div class="modal-footer">
            <button class="btn-cancel" @click="handleClose">
              {{ t('common.cancel') }}
            </button>
            <button
              class="btn-create"
              :class="{ warning: hasWarnings && !hasErrors, danger: hasErrors }"
              :disabled="!releaseName.trim() || creating || isChecking"
              @click="handleCreate"
            >
              <svg v-if="creating" class="spinning" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 12a9 9 0 1 1-6.219-8.56" />
              </svg>
              <span v-if="hasErrors || hasWarnings">{{ t('publishChecklist.ignoreAndPublish') }}</span>
              <span v-else>{{ t('editor.createReleaseButton') }}</span>
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.4);
  backdrop-filter: blur(4px);
}

.modal-content {
  width: 100%;
  max-width: 480px;
  background: white;
  border-radius: 12px;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.15);
  overflow: hidden;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid #e5e7eb;
}

.modal-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #111827;
}

.close-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  padding: 0;
  background: transparent;
  border: none;
  border-radius: 6px;
  color: #6b7280;
  cursor: pointer;
  transition: all 0.15s;
}

.close-btn:hover {
  background: #f3f4f6;
  color: #111827;
}

.modal-body {
  padding: 20px;
}

.modal-description {
  margin: 0 0 16px;
  font-size: 13px;
  color: #6b7280;
  line-height: 1.5;
}

.form-group {
  margin-bottom: 16px;
}

.form-group label {
  display: block;
  margin-bottom: 6px;
  font-size: 13px;
  font-weight: 500;
  color: #374151;
}

.form-input,
.form-textarea {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  font-size: 14px;
  color: #111827;
  background: white;
  transition: border-color 0.15s;
}

.form-input:focus,
.form-textarea:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.form-input::placeholder,
.form-textarea::placeholder {
  color: #9ca3af;
}

.form-textarea {
  resize: vertical;
  min-height: 60px;
}

/* Checklist Section */
.checklist-section {
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid #e5e7eb;
}

.checklist-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.checklist-title {
  font-size: 12px;
  font-weight: 600;
  color: #6b7280;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.checklist-summary {
  font-size: 12px;
  font-weight: 600;
  color: #10b981;
}

.checklist-summary.warning {
  color: #f59e0b;
}

.checklist-summary.error {
  color: #ef4444;
}

.checklist {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.check-item {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 8px 10px;
  border-radius: 6px;
  background: #f9fafb;
}

.check-item.passed {
  background: rgba(16, 185, 129, 0.08);
}

.check-item.warning {
  background: rgba(245, 158, 11, 0.08);
}

.check-item.error {
  background: rgba(239, 68, 68, 0.08);
}

.check-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  flex-shrink: 0;
  margin-top: 1px;
}

.check-item.passed .check-icon {
  color: #10b981;
}

.check-item.warning .check-icon {
  color: #f59e0b;
}

.check-item.error .check-icon {
  color: #ef4444;
}

.check-item.checking .check-icon {
  color: #6b7280;
}

.check-content {
  flex: 1;
  min-width: 0;
}

.check-label {
  display: block;
  font-size: 13px;
  font-weight: 500;
  color: #374151;
}

.check-message {
  display: block;
  font-size: 11px;
  color: #6b7280;
  margin-top: 2px;
}

.check-item.warning .check-message {
  color: #b45309;
}

.check-item.error .check-message {
  color: #dc2626;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 16px 20px;
  background: #f9fafb;
  border-top: 1px solid #e5e7eb;
}

.btn-cancel,
.btn-create {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
}

.btn-cancel {
  background: white;
  border: 1px solid #e5e7eb;
  color: #374151;
}

.btn-cancel:hover {
  background: #f9fafb;
  border-color: #d1d5db;
}

.btn-create {
  background: #3b82f6;
  border: none;
  color: white;
}

.btn-create:hover:not(:disabled) {
  background: #2563eb;
}

.btn-create.warning {
  background: #f59e0b;
}

.btn-create.warning:hover:not(:disabled) {
  background: #d97706;
}

.btn-create.danger {
  background: #ef4444;
}

.btn-create.danger:hover:not(:disabled) {
  background: #dc2626;
}

.btn-create:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.spinning,
.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

/* Modal Transition */
.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.2s ease;
}

.modal-enter-active .modal-content,
.modal-leave-active .modal-content {
  transition: transform 0.2s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-from .modal-content,
.modal-leave-to .modal-content {
  transform: scale(0.95) translateY(-10px);
}
</style>
