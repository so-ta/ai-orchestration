/**
 * Copilot Type Definitions
 *
 * Central type definitions for all Copilot-related functionality.
 * Split from useCopilot.ts for better maintainability.
 */

// ============================================================================
// Types for Copilot suggestions and diagnostics
// ============================================================================

export interface StepSuggestion {
  type: string
  name: string
  description: string
  config: Record<string, unknown>
  reason: string
}

export interface SuggestResponse {
  suggestions: StepSuggestion[]
}

export interface Diagnosis {
  root_cause: string
  category: 'config_error' | 'input_error' | 'api_error' | 'logic_error' | 'timeout' | 'unknown'
  severity: 'high' | 'medium' | 'low'
}

export interface Fix {
  description: string
  steps: string[]
  config_patch?: Record<string, unknown>
}

export interface DiagnoseResponse {
  diagnosis: Diagnosis
  fixes: Fix[]
  preventions: string[]
}

export interface StepExplanation {
  step_id: string
  step_name: string
  explanation: string
}

export interface ExplainResponse {
  summary: string
  step_details?: StepExplanation[]
}

export interface Optimization {
  category: 'performance' | 'cost' | 'reliability' | 'maintainability'
  title: string
  description: string
  impact: 'high' | 'medium' | 'low'
  effort: 'high' | 'medium' | 'low'
}

export interface OptimizeResponse {
  optimizations: Optimization[]
  summary: string
}

export interface SuggestedAction {
  type: 'add_step' | 'modify_step' | 'delete_step' | 'run_workflow'
  label: string
  description: string
  data?: Record<string, unknown>
}

export interface ChatResponse {
  response: string
  suggestions?: StepSuggestion[]
  actions?: SuggestedAction[]
}

// ============================================================================
// Session management types (Builder integration)
// ============================================================================

export type CopilotSessionMode = 'create' | 'enhance' | 'explain'
export type CopilotSessionStatus = 'hearing' | 'building' | 'reviewing' | 'refining' | 'completed' | 'abandoned'
export type CopilotPhase = 'analysis' | 'proposal' | 'completed'

export interface CopilotSession {
  id: string
  tenant_id: string
  user_id: string
  context_project_id?: string
  mode: CopilotSessionMode
  title?: string
  status: CopilotSessionStatus
  hearing_phase: CopilotPhase
  hearing_progress: number
  spec?: Record<string, unknown>
  project_id?: string
  created_at: string
  updated_at: string
  messages?: CopilotMessage[]
}

export interface CopilotMessage {
  id: string
  session_id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  phase?: CopilotPhase
  extracted_data?: Record<string, unknown>
  suggested_questions?: string[]
  created_at: string
}

// Question that needs user clarification
export interface ClarifyingPoint {
  id: string
  question: string
  options?: string[]
  required: boolean
  answer?: string
}

// Assumption made by AI during analysis
export interface Assumption {
  id: string
  category: 'trigger' | 'actor' | 'step' | 'integration' | 'constraint'
  description: string
  default: string
  confidence: 'high' | 'medium' | 'low'
  confirmed: boolean
}

// ============================================================================
// API Response types
// ============================================================================

export interface StartSessionResponse {
  session_id: string
  status: CopilotSessionStatus
  phase: CopilotPhase
  progress: number
  message?: {
    id: string
    role: string
    content: string
    suggested_questions?: string[]
  }
}

export interface SendMessageResponse {
  run_id: string
  status: string
}

export interface GetSessionResponse {
  id: string
  status: CopilotSessionStatus
  hearing_phase: CopilotPhase
  hearing_progress: number
  mode: CopilotSessionMode
  title?: string
  project_id?: string
  messages?: CopilotMessage[]
  created_at: string
  updated_at: string
}

export interface ListSessionsResponse {
  sessions: Array<{
    id: string
    status: CopilotSessionStatus
    hearing_phase: CopilotPhase
    hearing_progress: number
    mode: CopilotSessionMode
    title?: string
    project_id?: string
    created_at: string
    updated_at: string
  }>
  total: number
}

export interface ChatWithSessionResponse {
  response: string
  suggestions?: StepSuggestion[]
  session: CopilotSession
}

// ============================================================================
// Workflow generation types
// ============================================================================

export interface GeneratedStep {
  temp_id: string
  name: string
  type: string
  description: string
  config: Record<string, unknown>
  position_x: number
  position_y: number
}

export interface GeneratedEdge {
  source_temp_id: string
  target_temp_id: string
  source_port?: string
  condition?: string
}

export interface GenerateWorkflowResponse {
  response: string
  steps: GeneratedStep[]
  edges: GeneratedEdge[]
  start_step_id: string
}

// ============================================================================
// Async execution types (meta-workflow architecture)
// ============================================================================

export type CopilotRunStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled'

export interface AsyncRunResponse {
  run_id: string
  status: CopilotRunStatus
}

export interface CopilotRunResult<T = unknown> {
  run_id: string
  status: CopilotRunStatus
  started_at?: string
  completed_at?: string
  output?: T
  error?: string
}

// ============================================================================
// Agent-based Copilot types (autonomous tool-calling agent)
// ============================================================================

export type AgentEventType = 'thinking' | 'tool_call' | 'tool_result' | 'partial_text' | 'complete' | 'error'

export interface AgentThinkingEvent {
  type: 'thinking'
  data: { content: string }
}

export interface AgentToolCallEvent {
  type: 'tool_call'
  data: { tool_name: string; arguments: Record<string, unknown> }
}

export interface AgentToolResultEvent {
  type: 'tool_result'
  data: { tool_name: string; result: unknown; is_error: boolean }
}

export interface AgentPartialTextEvent {
  type: 'partial_text'
  data: { content: string }
}

export interface AgentCompleteEvent {
  type: 'complete'
  data: { response: string; tools_used: string[]; iterations: number; total_tokens: number }
}

export interface AgentErrorEvent {
  type: 'error'
  data: { error: string }
}

export type AgentEvent =
  | AgentThinkingEvent
  | AgentToolCallEvent
  | AgentToolResultEvent
  | AgentPartialTextEvent
  | AgentCompleteEvent
  | AgentErrorEvent

export interface AgentSessionResponse {
  session_id: string
  response: string
  tools_used: string[]
  status: string
  phase: string
  progress: number
}

export interface AgentMessageResponse {
  session_id: string
  response: string
  tools_used: string[]
  iterations: number
  total_tokens: number
}

export interface AgentToolDefinition {
  name: string
  description: string
  input_schema: Record<string, unknown>
}

export interface AgentSessionMessage {
  id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  extracted_data?: Record<string, unknown>
  created_at: string
}

export interface AgentSessionWithMessages {
  id: string
  status: string
  phase: string
  progress: number
  mode: string
  messages: AgentSessionMessage[]
  created_at: string
  updated_at: string
}

// Streaming state for the agent
export interface AgentStreamState {
  isStreaming: boolean
  currentThinking: string
  toolSteps: Array<{
    id: string
    tool: string
    status: 'calling' | 'success' | 'error'
    arguments?: Record<string, unknown>
    result?: unknown
    error?: string
  }>
  partialResponse: string
  finalResponse: string | null
  error: string | null
}

// ============================================================================
// Constants
// ============================================================================

export const TERMINAL_STATUSES: ReadonlySet<CopilotRunStatus> = new Set(['completed', 'failed', 'cancelled'])

export function isTerminalStatus(status: CopilotRunStatus): boolean {
  return TERMINAL_STATUSES.has(status)
}

// Phase labels for UI
export const HEARING_PHASE_LABELS: Record<CopilotPhase, string> = {
  analysis: '分析中',
  proposal: '提案・確認',
  completed: '完了',
}

export const HEARING_PHASES: CopilotPhase[] = [
  'analysis',
  'proposal',
  'completed',
]

// Session mode labels
export const SESSION_MODE_LABELS: Record<CopilotSessionMode, string> = {
  create: 'ワークフロー構築',
  enhance: '改善',
  explain: '説明',
}

// Assumption category labels
export const ASSUMPTION_CATEGORY_LABELS: Record<Assumption['category'], string> = {
  trigger: 'トリガー',
  actor: '実行者',
  step: 'ステップ',
  integration: '連携',
  constraint: '制約',
}

// ============================================================================
// Type Guards
// ============================================================================

export function isAgentThinkingEvent(event: AgentEvent): event is AgentThinkingEvent {
  return event.type === 'thinking'
}

export function isAgentToolCallEvent(event: AgentEvent): event is AgentToolCallEvent {
  return event.type === 'tool_call'
}

export function isAgentToolResultEvent(event: AgentEvent): event is AgentToolResultEvent {
  return event.type === 'tool_result'
}

export function isAgentPartialTextEvent(event: AgentEvent): event is AgentPartialTextEvent {
  return event.type === 'partial_text'
}

export function isAgentCompleteEvent(event: AgentEvent): event is AgentCompleteEvent {
  return event.type === 'complete'
}

export function isAgentErrorEvent(event: AgentEvent): event is AgentErrorEvent {
  return event.type === 'error'
}
