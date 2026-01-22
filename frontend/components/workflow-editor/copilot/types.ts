/**
 * Copilot E2E Workflow Types
 *
 * Types for the Copilot progressive experience that guides users through:
 * workflow creation → trigger configuration → credential setup → testing → publishing
 */

// ============================================================================
// Workflow Phases
// ============================================================================

export type WorkflowPhase =
  | 'creation'      // Building the workflow DAG
  | 'configuration' // Trigger setup
  | 'setup'         // Credential configuration
  | 'validation'    // Test execution
  | 'deploy'        // Publishing

export const WORKFLOW_PHASES: WorkflowPhase[] = [
  'creation',
  'configuration',
  'setup',
  'validation',
  'deploy',
]

// ============================================================================
// Progress Checklist Types
// ============================================================================

export type ChecklistItemStatus = 'pending' | 'in_progress' | 'completed' | 'skipped' | 'error'

export interface ChecklistItem {
  id: string
  phase: WorkflowPhase
  label: string
  description?: string
  status: ChecklistItemStatus
  children?: ChecklistItem[]
  action?: InlineAction // Optional inline action for this item
}

export interface WorkflowProgressStatus {
  currentPhase: WorkflowPhase
  items: ChecklistItem[]
  completedCount: number
  totalCount: number
  isComplete: boolean
}

// ============================================================================
// Inline Action Types
// ============================================================================

export type InlineActionType =
  | 'confirm'   // Yes/No confirmation
  | 'select'    // Single selection from options
  | 'form'      // Simple form input
  | 'oauth'     // OAuth connection button
  | 'test'      // Test execution button

// Base action interface
export interface BaseInlineAction {
  id: string
  type: InlineActionType
  title?: string
  description?: string
}

// Confirm action: Yes/No buttons
export interface ConfirmAction extends BaseInlineAction {
  type: 'confirm'
  confirmLabel?: string
  cancelLabel?: string
}

// Select action: Choose from options
export interface SelectOption {
  id: string
  label: string
  description?: string
  icon?: string
  recommended?: boolean
}

export interface SelectAction extends BaseInlineAction {
  type: 'select'
  options: SelectOption[]
  allowMultiple?: boolean
}

// Form action: Simple input form
export interface FormField {
  id: string
  type: 'text' | 'number' | 'select' | 'time' | 'timezone' | 'cron'
  label: string
  placeholder?: string
  defaultValue?: string | number
  options?: Array<{ value: string; label: string }>
  required?: boolean
}

export interface FormAction extends BaseInlineAction {
  type: 'form'
  fields: FormField[]
  submitLabel?: string
}

// OAuth action: Connect external service
export interface OAuthAction extends BaseInlineAction {
  type: 'oauth'
  service: string
  serviceName: string
  serviceIcon?: string
  existingCredentials?: Array<{ id: string; name: string }>
}

// Test action: Run workflow test
export interface TestAction extends BaseInlineAction {
  type: 'test'
  testLabel?: string
  skipLabel?: string
}

export type InlineAction =
  | ConfirmAction
  | SelectAction
  | FormAction
  | OAuthAction
  | TestAction

// ============================================================================
// Action Results
// ============================================================================

export interface ConfirmResult {
  type: 'confirm'
  confirmed: boolean
}

export interface SelectResult {
  type: 'select'
  selectedIds: string[]
}

export interface FormResult {
  type: 'form'
  values: Record<string, string | number>
}

export interface OAuthResult {
  type: 'oauth'
  credentialId: string
  credentialName: string
}

export interface TestResult {
  type: 'test'
  skipped: boolean
}

export type InlineActionResult =
  | ConfirmResult
  | SelectResult
  | FormResult
  | OAuthResult
  | TestResult

// ============================================================================
// Test Result Types
// ============================================================================

export type TestStepStatus = 'pending' | 'running' | 'success' | 'error'

export interface TestStepResult {
  stepId: string
  stepName: string
  status: TestStepStatus
  durationMs?: number
  error?: string
}

export interface WorkflowTestResult {
  runId: string
  status: 'success' | 'failed' | 'running'
  steps: TestStepResult[]
  totalDurationMs?: number
  startedAt: string
  completedAt?: string
}

// ============================================================================
// Trigger Configuration Types
// ============================================================================

export type TriggerType = 'schedule' | 'webhook' | 'manual' | 'slack_event'

export interface ScheduleTriggerConfig {
  type: 'schedule'
  time: string       // HH:mm format
  timezone: string   // e.g., 'Asia/Tokyo'
  cron?: string      // Optional custom cron expression
  days?: string[]    // Optional specific days ['monday', 'tuesday', ...]
}

export interface WebhookTriggerConfig {
  type: 'webhook'
  path?: string
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE'
}

export interface ManualTriggerConfig {
  type: 'manual'
}

export interface SlackEventTriggerConfig {
  type: 'slack_event'
  eventType: string
  channel?: string
}

export type TriggerConfig =
  | ScheduleTriggerConfig
  | WebhookTriggerConfig
  | ManualTriggerConfig
  | SlackEventTriggerConfig

// ============================================================================
// Credential Types
// ============================================================================

export interface RequiredCredential {
  id: string
  stepId: string
  stepName: string
  service: string
  serviceName: string
  serviceIcon?: string
  isConfigured: boolean
  credentialId?: string
  credentialName?: string
}

export interface CredentialLinkResult {
  stepId: string
  credentialId: string
  success: boolean
  error?: string
}

// ============================================================================
// Backend API Types
// ============================================================================

export interface WorkflowCopilotStatusResponse {
  workflowId: string
  currentPhase: WorkflowPhase
  progress: WorkflowProgressStatus
  requiredCredentials: RequiredCredential[]
  trigger?: {
    type: TriggerType
    isConfigured: boolean
    config?: TriggerConfig
  }
  validation?: {
    isValid: boolean
    errors: string[]
    warnings: string[]
  }
  isPublished: boolean
  canPublish: boolean
}

// ============================================================================
// Chat Message Extension Types
// ============================================================================

export interface CopilotChatExtension {
  progressCard?: WorkflowProgressStatus
  inlineAction?: InlineAction
  testResult?: WorkflowTestResult
}
