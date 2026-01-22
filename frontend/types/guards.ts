/**
 * Type Guards for runtime type checking
 *
 * These functions provide type-safe narrowing for union types
 * and help catch type errors at runtime.
 */

import type {
  DraftChange,
  DraftStepCreate,
  DraftStepUpdate,
  DraftStepDelete,
  DraftEdgeCreate,
  DraftEdgeDelete,
} from '~/composables/useCopilotDraft'
import type {
  Project,
  Step,
  Edge,
  Run,
  StepRun,
  BlockDefinition,
  RunStatus,
  StepRunStatus,
} from './api'

// ============================================================================
// Draft Change Type Guards
// ============================================================================

/**
 * Check if a DraftChange is a step creation
 */
export function isDraftStepCreate(change: DraftChange): change is DraftStepCreate {
  return change.type === 'step:create'
}

/**
 * Check if a DraftChange is a step update
 */
export function isDraftStepUpdate(change: DraftChange): change is DraftStepUpdate {
  return change.type === 'step:update'
}

/**
 * Check if a DraftChange is a step deletion
 */
export function isDraftStepDelete(change: DraftChange): change is DraftStepDelete {
  return change.type === 'step:delete'
}

/**
 * Check if a DraftChange is an edge creation
 */
export function isDraftEdgeCreate(change: DraftChange): change is DraftEdgeCreate {
  return change.type === 'edge:create'
}

/**
 * Check if a DraftChange is an edge deletion
 */
export function isDraftEdgeDelete(change: DraftChange): change is DraftEdgeDelete {
  return change.type === 'edge:delete'
}

/**
 * Check if a DraftChange affects a step (create, update, or delete)
 */
export function isDraftStepChange(
  change: DraftChange
): change is DraftStepCreate | DraftStepUpdate | DraftStepDelete {
  return (
    change.type === 'step:create' ||
    change.type === 'step:update' ||
    change.type === 'step:delete'
  )
}

/**
 * Check if a DraftChange affects an edge (create or delete)
 */
export function isDraftEdgeChange(
  change: DraftChange
): change is DraftEdgeCreate | DraftEdgeDelete {
  return change.type === 'edge:create' || change.type === 'edge:delete'
}

// ============================================================================
// API Type Guards
// ============================================================================

/**
 * Check if an object is a Project
 */
export function isProject(obj: unknown): obj is Project {
  if (typeof obj !== 'object' || obj === null) return false
  const p = obj as Record<string, unknown>
  return (
    typeof p.id === 'string' &&
    typeof p.tenant_id === 'string' &&
    typeof p.name === 'string' &&
    (p.status === 'draft' || p.status === 'published')
  )
}

/**
 * Check if an object is a Step
 */
export function isStep(obj: unknown): obj is Step {
  if (typeof obj !== 'object' || obj === null) return false
  const s = obj as Record<string, unknown>
  return (
    typeof s.id === 'string' &&
    typeof s.project_id === 'string' &&
    typeof s.name === 'string' &&
    typeof s.type === 'string' &&
    typeof s.position_x === 'number' &&
    typeof s.position_y === 'number'
  )
}

/**
 * Check if an object is an Edge
 */
export function isEdge(obj: unknown): obj is Edge {
  if (typeof obj !== 'object' || obj === null) return false
  const e = obj as Record<string, unknown>
  return (
    typeof e.id === 'string' &&
    typeof e.project_id === 'string' &&
    typeof e.created_at === 'string'
  )
}

/**
 * Check if an object is a Run
 */
export function isRun(obj: unknown): obj is Run {
  if (typeof obj !== 'object' || obj === null) return false
  const r = obj as Record<string, unknown>
  return (
    typeof r.id === 'string' &&
    typeof r.tenant_id === 'string' &&
    typeof r.project_id === 'string' &&
    typeof r.status === 'string' &&
    isRunStatus(r.status)
  )
}

/**
 * Check if an object is a StepRun
 */
export function isStepRun(obj: unknown): obj is StepRun {
  if (typeof obj !== 'object' || obj === null) return false
  const sr = obj as Record<string, unknown>
  return (
    typeof sr.id === 'string' &&
    typeof sr.run_id === 'string' &&
    typeof sr.step_id === 'string' &&
    typeof sr.status === 'string' &&
    isStepRunStatus(sr.status)
  )
}

/**
 * Check if a string is a valid RunStatus
 */
export function isRunStatus(status: unknown): status is RunStatus {
  return (
    status === 'pending' ||
    status === 'running' ||
    status === 'completed' ||
    status === 'failed' ||
    status === 'cancelled'
  )
}

/**
 * Check if a string is a valid StepRunStatus
 */
export function isStepRunStatus(status: unknown): status is StepRunStatus {
  return (
    status === 'pending' ||
    status === 'running' ||
    status === 'completed' ||
    status === 'failed' ||
    status === 'skipped'
  )
}

/**
 * Check if an object is a BlockDefinition
 */
export function isBlockDefinition(obj: unknown): obj is BlockDefinition {
  if (typeof obj !== 'object' || obj === null) return false
  const b = obj as Record<string, unknown>
  return (
    typeof b.id === 'string' &&
    typeof b.slug === 'string' &&
    typeof b.name === 'string' &&
    typeof b.category === 'string'
  )
}

// ============================================================================
// Utility Type Guards
// ============================================================================

/**
 * Check if a value is a non-null object
 */
export function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

/**
 * Check if a value is a non-empty string
 */
export function isNonEmptyString(value: unknown): value is string {
  return typeof value === 'string' && value.length > 0
}

/**
 * Check if a value is a valid UUID string
 */
export function isUUID(value: unknown): value is string {
  if (typeof value !== 'string') return false
  const uuidRegex = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
  return uuidRegex.test(value)
}

/**
 * Check if a value is defined (not null or undefined)
 */
export function isDefined<T>(value: T | null | undefined): value is T {
  return value !== null && value !== undefined
}

/**
 * Assert that a value is defined, throwing an error if not
 */
export function assertDefined<T>(
  value: T | null | undefined,
  message = 'Value is null or undefined'
): asserts value is T {
  if (value === null || value === undefined) {
    throw new Error(message)
  }
}

/**
 * Narrow an array to exclude null/undefined values
 */
export function filterDefined<T>(arr: (T | null | undefined)[]): T[] {
  return arr.filter(isDefined)
}

// ============================================================================
// Error Type Guards
// ============================================================================

/**
 * Check if an error is an Error object with a message
 */
export function isErrorWithMessage(error: unknown): error is { message: string } {
  return (
    typeof error === 'object' &&
    error !== null &&
    'message' in error &&
    typeof (error as { message: unknown }).message === 'string'
  )
}

/**
 * Extract error message safely from any error type
 */
export function getErrorMessage(error: unknown): string {
  if (isErrorWithMessage(error)) {
    return error.message
  }
  if (typeof error === 'string') {
    return error
  }
  return 'Unknown error'
}
