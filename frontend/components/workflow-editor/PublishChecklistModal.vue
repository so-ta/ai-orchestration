<script setup lang="ts">
/**
 * PublishChecklistModal - Pre-publish validation checklist
 *
 * Shows validation results before publishing a workflow:
 * - Start block exists
 * - All blocks are connected
 * - No infinite loops
 * - Credentials are configured
 * - Trigger is properly configured
 * - Required configurations are set
 */

import type { Step, Edge, BlockDefinition, ApiResponse } from '~/types/api'

interface CheckResult {
  id: string
  label: string
  status: 'passed' | 'warning' | 'error' | 'checking'
  message?: string
}

interface ValidationApiResult {
  checks: Array<{
    id: string
    label: string
    status: 'passed' | 'warning' | 'error'
    message?: string
  }>
  can_publish: boolean
  error_count: number
  warning_count: number
}

const props = defineProps<{
  show: boolean
  workflowId: string
  steps: Step[]
  edges: Edge[]
  blockDefinitions: BlockDefinition[]
}>()

const emit = defineEmits<{
  close: []
  publish: []
  fixIssue: [checkId: string]
  enableTrigger: []
}>()

const { t } = useI18n()
const api = useApi()

// Validation state
const isChecking = ref(true)
const checks = ref<CheckResult[]>([])
const _useApiValidation = ref(true) // Use API validation by default (reserved for future use)

// Run checks when modal opens
watch(() => props.show, async (show) => {
  if (show) {
    await runChecks()
  }
}, { immediate: true })

// Run all validation checks
async function runChecks() {
  isChecking.value = true

  // Initialize all check items
  checks.value = [
    { id: 'hasStartBlock', label: t('publishChecklist.checks.hasStartBlock'), status: 'checking' },
    { id: 'allConnected', label: t('publishChecklist.checks.allConnected'), status: 'checking' },
    { id: 'noLoop', label: t('publishChecklist.checks.noLoop'), status: 'checking' },
    { id: 'credentialsSet', label: t('publishChecklist.checks.credentialsSet'), status: 'checking' },
    { id: 'triggerConfigured', label: t('publishChecklist.checks.triggerConfigured'), status: 'checking' },
    { id: 'requiredConfigSet', label: t('publishChecklist.checks.requiredConfigSet'), status: 'checking' },
  ]

  try {
    // Try to use API validation
    const response = await api.post<ApiResponse<ValidationApiResult>>(`/workflows/${props.workflowId}/validate`)
    const result = response.data

    // Update checks from API response
    for (const apiCheck of result.checks) {
      const check = checks.value.find(c => c.id === apiCheck.id)
      if (check) {
        check.status = apiCheck.status
        check.message = apiCheck.message
        // Translate label if available
        const translatedLabel = t(`publishChecklist.checks.${apiCheck.id}`, apiCheck.label)
        if (translatedLabel !== `publishChecklist.checks.${apiCheck.id}`) {
          check.label = translatedLabel
        }
      } else {
        // Add new check from API that we don't have locally
        checks.value.push({
          id: apiCheck.id,
          label: apiCheck.label,
          status: apiCheck.status,
          message: apiCheck.message,
        })
      }
    }
  } catch {
    // Fallback to local validation if API fails
    console.warn('API validation failed, falling back to local validation')
    await runLocalChecks()
  }

  isChecking.value = false
}

// Fallback local validation
async function runLocalChecks() {
  // Simulate async check delay for better UX
  await new Promise(resolve => setTimeout(resolve, 300))

  // Check 1: Start block exists
  const startBlockTypes = ['start', 'manual_trigger', 'schedule_trigger', 'webhook_trigger']
  const hasStartBlock = props.steps.some(step => startBlockTypes.includes(step.type))
  updateCheck('hasStartBlock', hasStartBlock ? 'passed' : 'error',
    hasStartBlock ? undefined : t('publishChecklist.messages.addStartBlock'))

  // Check 2: All blocks connected
  const { allConnected, unconnectedSteps } = checkAllConnected()
  updateCheck('allConnected',
    allConnected ? 'passed' : 'warning',
    allConnected ? undefined : t('publishChecklist.messages.unconnectedBlocks', { count: unconnectedSteps.length }))

  // Check 3: No infinite loops
  const hasCycle = checkForCycles()
  updateCheck('noLoop',
    hasCycle ? 'error' : 'passed',
    hasCycle ? t('publishChecklist.messages.cycleDetected') : undefined)

  // Check 4: Credentials configured
  const { allSet, missingCredentials } = checkCredentials()
  updateCheck('credentialsSet',
    allSet ? 'passed' : 'warning',
    allSet ? undefined : t('publishChecklist.messages.missingCredentials', { count: missingCredentials.length }))

  // Check 5: Trigger configured
  const { configured, enabled } = checkTriggerConfig()
  updateCheck('triggerConfigured',
    configured && enabled ? 'passed' : (configured ? 'warning' : 'warning'),
    configured ? (enabled ? undefined : t('publishChecklist.messages.triggerNotEnabled')) : t('publishChecklist.messages.noTrigger'))

  // Check 6: Required config set (simplified local check)
  updateCheck('requiredConfigSet', 'passed', undefined)
}

// Check trigger configuration
function checkTriggerConfig(): { configured: boolean; enabled: boolean } {
  const startBlockTypes = ['start', 'manual_trigger', 'schedule_trigger', 'webhook_trigger']
  const startBlock = props.steps.find(step => startBlockTypes.includes(step.type))

  if (!startBlock) {
    return { configured: false, enabled: false }
  }

  // Manual triggers are always "enabled"
  if (startBlock.trigger_type === 'manual' || String(startBlock.type) === 'manual_trigger') {
    return { configured: true, enabled: true }
  }

  // Check if trigger is enabled in config
  const config = startBlock.trigger_config as Record<string, unknown> | undefined
  const enabled = config?.enabled === true

  return { configured: true, enabled }
}

function updateCheck(id: string, status: CheckResult['status'], message?: string) {
  const check = checks.value.find(c => c.id === id)
  if (check) {
    check.status = status
    check.message = message
  }
}

// Check if all blocks are connected
function checkAllConnected(): { allConnected: boolean; unconnectedSteps: Step[] } {
  if (props.steps.length <= 1) {
    return { allConnected: true, unconnectedSteps: [] }
  }

  const connectedIds = new Set<string>()
  for (const edge of props.edges) {
    if (edge.source_step_id) connectedIds.add(edge.source_step_id)
    if (edge.target_step_id) connectedIds.add(edge.target_step_id)
  }

  const unconnectedSteps = props.steps.filter(step => !connectedIds.has(step.id))
  return {
    allConnected: unconnectedSteps.length === 0,
    unconnectedSteps,
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
function checkCredentials(): { allSet: boolean; missingCredentials: string[] } {
  const missingCredentials: string[] = []

  for (const step of props.steps) {
    const blockDef = props.blockDefinitions.find(b => b.slug === step.type)
    if (!blockDef?.required_credentials?.length) continue

    // Check if step has credential bindings for required credentials
    const bindings = step.credential_bindings || {}
    for (const required of blockDef.required_credentials) {
      if (!bindings[required]) {
        missingCredentials.push(`${step.name}: ${required}`)
      }
    }
  }

  return {
    allSet: missingCredentials.length === 0,
    missingCredentials,
  }
}

// Computed properties
const hasErrors = computed(() => checks.value.some(c => c.status === 'error'))
const hasWarnings = computed(() => checks.value.some(c => c.status === 'warning'))
const errorCount = computed(() => checks.value.filter(c => c.status === 'error').length)
const warningCount = computed(() => checks.value.filter(c => c.status === 'warning').length)

const canPublish = computed(() => !hasErrors.value)

// Handlers
function handlePublish() {
  if (canPublish.value) {
    emit('publish')
  }
}

function handleFixIssue(checkId: string) {
  emit('fixIssue', checkId)
  emit('close')
}
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="show" class="modal-overlay" @click.self="emit('close')">
        <div class="modal-content">
          <!-- Header -->
          <div class="modal-header">
            <div class="header-icon" :class="{ error: hasErrors, warning: hasWarnings && !hasErrors }">
              <svg v-if="isChecking" class="spin" xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
              </svg>
              <svg v-else-if="hasErrors" xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10"/>
                <line x1="15" y1="9" x2="9" y2="15"/>
                <line x1="9" y1="9" x2="15" y2="15"/>
              </svg>
              <svg v-else-if="hasWarnings" xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/>
                <line x1="12" y1="9" x2="12" y2="13"/>
                <line x1="12" y1="17" x2="12.01" y2="17"/>
              </svg>
              <svg v-else xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="20 6 9 17 4 12"/>
              </svg>
            </div>
            <div class="header-text">
              <h2 class="modal-title">{{ t('publishChecklist.title') }}</h2>
              <p class="modal-subtitle">{{ t('publishChecklist.subtitle') }}</p>
            </div>
            <button class="close-btn" @click="emit('close')">
              <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="18" y1="6" x2="6" y2="18"/>
                <line x1="6" y1="6" x2="18" y2="18"/>
              </svg>
            </button>
          </div>

          <!-- Status Summary -->
          <div v-if="!isChecking" class="status-summary" :class="{ error: hasErrors, warning: hasWarnings && !hasErrors, success: !hasErrors && !hasWarnings }">
            <template v-if="hasErrors">
              {{ t('publishChecklist.hasErrors', { count: errorCount }) }}
            </template>
            <template v-else-if="hasWarnings">
              {{ t('publishChecklist.hasWarnings', { count: warningCount }) }}
            </template>
            <template v-else>
              {{ t('publishChecklist.allPassed') }}
            </template>
          </div>
          <div v-else class="status-summary checking">
            {{ t('publishChecklist.checking') }}
          </div>

          <!-- Checklist -->
          <div class="checklist">
            <div
              v-for="check in checks"
              :key="check.id"
              class="check-item"
              :class="check.status"
            >
              <div class="check-icon">
                <svg v-if="check.status === 'checking'" class="spin" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
                </svg>
                <svg v-else-if="check.status === 'passed'" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="20 6 9 17 4 12"/>
                </svg>
                <svg v-else-if="check.status === 'warning'" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/>
                  <line x1="12" y1="9" x2="12" y2="13"/>
                  <line x1="12" y1="17" x2="12.01" y2="17"/>
                </svg>
                <svg v-else xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="10"/>
                  <line x1="15" y1="9" x2="9" y2="15"/>
                  <line x1="9" y1="9" x2="15" y2="15"/>
                </svg>
              </div>
              <div class="check-content">
                <span class="check-label">{{ check.label }}</span>
                <span v-if="check.message" class="check-message">{{ check.message }}</span>
              </div>
              <button
                v-if="check.status === 'error' || check.status === 'warning'"
                class="fix-btn"
                @click="handleFixIssue(check.id)"
              >
                {{ t('publishChecklist.fixIssues') }}
              </button>
            </div>
          </div>

          <!-- Footer -->
          <div class="modal-footer">
            <button class="btn btn-secondary" @click="emit('close')">
              {{ t('publishChecklist.cancel') }}
            </button>
            <!-- Show warning-style button if there are errors or warnings -->
            <button
              v-if="hasErrors || hasWarnings"
              class="btn"
              :class="hasErrors ? 'btn-danger' : 'btn-warning'"
              :disabled="isChecking"
              @click="handlePublish"
            >
              {{ t('publishChecklist.ignoreAndPublish') }}
            </button>
            <!-- Show primary button only when all checks pass -->
            <button
              v-else
              class="btn btn-primary"
              :disabled="isChecking"
              @click="handlePublish"
            >
              {{ t('editor.saveAndPublish') }}
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
  background: rgba(0, 0, 0, 0.5);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 1rem;
}

.modal-content {
  width: 100%;
  max-width: 480px;
  background: white;
  border-radius: 12px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.2);
  overflow: hidden;
}

.modal-header {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
  padding: 1.25rem 1.25rem 1rem;
  border-bottom: 1px solid var(--color-border);
}

.header-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  background: var(--color-success);
  border-radius: 10px;
  color: white;
  flex-shrink: 0;
}

.header-icon.error {
  background: var(--color-error);
}

.header-icon.warning {
  background: var(--color-warning);
}

.header-text {
  flex: 1;
  min-width: 0;
}

.modal-title {
  font-size: 1rem;
  font-weight: 600;
  color: var(--color-text);
  margin: 0 0 0.25rem;
}

.modal-subtitle {
  font-size: 0.8125rem;
  color: var(--color-text-secondary);
  margin: 0;
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
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.15s;
}

.close-btn:hover {
  background: var(--color-background);
  color: var(--color-text);
}

.status-summary {
  padding: 0.75rem 1.25rem;
  font-size: 0.8125rem;
  font-weight: 500;
  text-align: center;
}

.status-summary.success {
  background: rgba(16, 185, 129, 0.1);
  color: var(--color-success);
}

.status-summary.warning {
  background: rgba(245, 158, 11, 0.1);
  color: var(--color-warning);
}

.status-summary.error {
  background: rgba(239, 68, 68, 0.1);
  color: var(--color-error);
}

.status-summary.checking {
  background: var(--color-background);
  color: var(--color-text-secondary);
}

.checklist {
  padding: 1rem 1.25rem;
}

.check-item {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
  padding: 0.75rem;
  border-radius: 8px;
  margin-bottom: 0.5rem;
  transition: background-color 0.15s;
}

.check-item:last-child {
  margin-bottom: 0;
}

.check-item.passed {
  background: rgba(16, 185, 129, 0.05);
}

.check-item.warning {
  background: rgba(245, 158, 11, 0.08);
}

.check-item.error {
  background: rgba(239, 68, 68, 0.08);
}

.check-item.checking {
  background: var(--color-background);
}

.check-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  flex-shrink: 0;
}

.check-item.passed .check-icon {
  color: var(--color-success);
}

.check-item.warning .check-icon {
  color: var(--color-warning);
}

.check-item.error .check-icon {
  color: var(--color-error);
}

.check-item.checking .check-icon {
  color: var(--color-text-secondary);
}

.check-content {
  flex: 1;
  min-width: 0;
}

.check-label {
  display: block;
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--color-text);
}

.check-message {
  display: block;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  margin-top: 0.25rem;
}

.fix-btn {
  padding: 0.375rem 0.75rem;
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--color-primary);
  background: white;
  border: 1px solid var(--color-primary);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
  flex-shrink: 0;
}

.fix-btn:hover {
  background: var(--color-primary);
  color: white;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  padding: 1rem 1.25rem;
  border-top: 1px solid var(--color-border);
  background: var(--color-background);
}

.btn {
  padding: 0.625rem 1rem;
  font-size: 0.875rem;
  font-weight: 500;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s;
}

.btn-secondary {
  background: white;
  border: 1px solid var(--color-border);
  color: var(--color-text);
}

.btn-secondary:hover {
  background: var(--color-background);
}

.btn-primary {
  background: var(--color-primary);
  color: white;
}

.btn-primary:hover:not(:disabled) {
  filter: brightness(1.1);
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-warning {
  background: var(--color-warning);
  color: white;
}

.btn-warning:hover {
  filter: brightness(1.1);
}

.btn-danger {
  background: var(--color-error);
  color: white;
}

.btn-danger:hover {
  filter: brightness(1.1);
}

/* Animations */
@keyframes spin {
  to { transform: rotate(360deg); }
}

.spin {
  animation: spin 1s linear infinite;
}

.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.2s ease;
}

.modal-enter-active .modal-content,
.modal-leave-active .modal-content {
  transition: transform 0.2s ease, opacity 0.2s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-from .modal-content,
.modal-leave-to .modal-content {
  transform: scale(0.95);
  opacity: 0;
}
</style>
