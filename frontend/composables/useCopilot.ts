// Copilot API composable for AI-assisted workflow building
// Integrated with Builder functionality for interactive workflow creation

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
// Agent-based Copilot types (NEW: autonomous tool-calling agent)
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

const TERMINAL_STATUSES: ReadonlySet<CopilotRunStatus> = new Set(['completed', 'failed', 'cancelled'])

function isTerminalStatus(status: CopilotRunStatus): boolean {
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
  create: '新規作成',
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
// Main composable
// ============================================================================

export function useCopilot() {
  const api = useApi()

  // Polling settings
  const POLL_INTERVAL = 2000 // 2 seconds
  const MAX_POLL_ATTEMPTS = 180 // 6 minutes max

  // ==========================================================================
  // Legacy Copilot endpoints (global)
  // ==========================================================================

  // Get step suggestions for a workflow
  async function suggest(
    workflowId: string,
    options?: { stepId?: string; context?: string }
  ): Promise<SuggestResponse> {
    return api.post<SuggestResponse>('/copilot/suggest', {
      workflow_id: workflowId,
      step_id: options?.stepId,
      context: options?.context,
    })
  }

  // Diagnose errors in a run
  async function diagnose(
    runId: string,
    stepRunId?: string
  ): Promise<DiagnoseResponse> {
    return api.post<DiagnoseResponse>('/copilot/diagnose', {
      run_id: runId,
      step_run_id: stepRunId,
    })
  }

  // Explain a workflow or step
  async function explain(
    workflowId: string,
    stepId?: string
  ): Promise<ExplainResponse> {
    return api.post<ExplainResponse>('/copilot/explain', {
      workflow_id: workflowId,
      step_id: stepId,
    })
  }

  // Get optimization suggestions for a workflow
  async function optimize(workflowId: string): Promise<OptimizeResponse> {
    return api.post<OptimizeResponse>('/copilot/optimize', {
      workflow_id: workflowId,
    })
  }

  // Chat with the copilot (global)
  async function chat(
    message: string,
    options?: { workflowId?: string; context?: string }
  ): Promise<ChatResponse> {
    return api.post<ChatResponse>('/copilot/chat', {
      workflow_id: options?.workflowId,
      message,
      context: options?.context,
    })
  }

  // ==========================================================================
  // Step-specific Copilot endpoints
  // ==========================================================================

  // Suggest for a specific step
  async function suggestForStep(
    workflowId: string,
    stepId: string,
    context?: string
  ): Promise<SuggestResponse> {
    return api.post<SuggestResponse>(
      `/workflows/${workflowId}/steps/${stepId}/copilot/suggest`,
      { context }
    )
  }

  // Explain a specific step
  async function explainStep(
    workflowId: string,
    stepId: string
  ): Promise<ExplainResponse> {
    return api.post<ExplainResponse>(
      `/workflows/${workflowId}/steps/${stepId}/copilot/explain`,
      {}
    )
  }

  // ==========================================================================
  // Workflow-scoped Copilot Session management (Builder integration)
  // ==========================================================================

  /**
   * Start a new copilot session for a workflow
   */
  async function startSession(
    projectId: string,
    initialPrompt: string,
    mode: CopilotSessionMode = 'create'
  ): Promise<StartSessionResponse> {
    return api.post<StartSessionResponse>(
      `/workflows/${projectId}/copilot/sessions`,
      {
        initial_prompt: initialPrompt,
        mode,
      }
    )
  }

  /**
   * Get a copilot session by ID
   */
  async function getSession(
    projectId: string,
    sessionId: string
  ): Promise<GetSessionResponse> {
    return api.get<GetSessionResponse>(
      `/workflows/${projectId}/copilot/sessions/${sessionId}`
    )
  }

  /**
   * Get session with messages (alias for compatibility)
   */
  async function getSessionWithMessages(
    projectId: string,
    sessionId: string
  ): Promise<CopilotSession> {
    return api.get<CopilotSession>(
      `/workflows/${projectId}/copilot/sessions/${sessionId}/messages`
    )
  }

  /**
   * List all copilot sessions for a workflow
   */
  async function listSessions(projectId: string): Promise<ListSessionsResponse> {
    return api.get<ListSessionsResponse>(
      `/workflows/${projectId}/copilot/sessions`
    )
  }

  /**
   * Send a message to the copilot session (async)
   */
  async function sendMessage(
    projectId: string,
    sessionId: string,
    content: string
  ): Promise<SendMessageResponse> {
    return api.post<SendMessageResponse>(
      `/workflows/${projectId}/copilot/sessions/${sessionId}/messages`,
      { content }
    )
  }

  /**
   * Start workflow construction (async)
   */
  async function construct(
    projectId: string,
    sessionId: string
  ): Promise<SendMessageResponse> {
    return api.post<SendMessageResponse>(
      `/workflows/${projectId}/copilot/sessions/${sessionId}/construct`,
      {}
    )
  }

  /**
   * Refine the workflow based on feedback (async)
   */
  async function refine(
    projectId: string,
    sessionId: string,
    feedback: string
  ): Promise<SendMessageResponse> {
    return api.post<SendMessageResponse>(
      `/workflows/${projectId}/copilot/sessions/${sessionId}/refine`,
      { feedback }
    )
  }

  /**
   * Finalize the copilot session
   */
  async function finalize(
    projectId: string,
    sessionId: string
  ): Promise<{ status: string }> {
    return api.post<{ status: string }>(
      `/workflows/${projectId}/copilot/sessions/${sessionId}/finalize`,
      {}
    )
  }

  /**
   * Delete a copilot session
   */
  async function deleteSession(
    projectId: string,
    sessionId: string
  ): Promise<void> {
    await api.delete(`/workflows/${projectId}/copilot/sessions/${sessionId}`)
  }

  // ==========================================================================
  // Legacy session endpoints (for backward compatibility)
  // ==========================================================================

  // Get or create the active session for a workflow
  async function getOrCreateSession(workflowId: string): Promise<CopilotSession> {
    return api.get<CopilotSession>(`/workflows/${workflowId}/copilot/session`)
  }

  // Start a new chat session (legacy)
  async function startNewSession(workflowId: string): Promise<CopilotSession> {
    return api.post<CopilotSession>(
      `/workflows/${workflowId}/copilot/sessions/new`,
      {}
    )
  }

  // Get session messages (legacy)
  async function getSessionMessages(
    workflowId: string,
    sessionId: string
  ): Promise<CopilotSession> {
    return api.get<CopilotSession>(
      `/workflows/${workflowId}/copilot/sessions/${sessionId}/messages`
    )
  }

  // Chat with session persistence (legacy)
  async function chatWithSession(
    workflowId: string,
    message: string,
    options?: { sessionId?: string; context?: string }
  ): Promise<ChatWithSessionResponse> {
    return api.post<ChatWithSessionResponse>(
      `/workflows/${workflowId}/copilot/chat`,
      {
        message,
        session_id: options?.sessionId,
        context: options?.context,
      }
    )
  }

  // Generate workflow from natural language description (legacy)
  async function generateWorkflow(
    workflowId: string,
    description: string
  ): Promise<GenerateWorkflowResponse> {
    return api.post<GenerateWorkflowResponse>(
      `/workflows/${workflowId}/copilot/generate`,
      { description }
    )
  }

  // ==========================================================================
  // Async Operations with Polling
  // ==========================================================================

  // Get copilot run status
  async function getCopilotRun<T = unknown>(runId: string): Promise<CopilotRunResult<T>> {
    return api.get<CopilotRunResult<T>>(`/copilot/runs/${runId}`)
  }

  // Get run status (for builder)
  async function getRun(runId: string): Promise<CopilotRunResult> {
    return api.get<CopilotRunResult>(`/runs/${runId}`)
  }

  // Poll for run completion
  async function pollForCompletion<T = unknown>(
    runId: string,
    onProgress?: (status: CopilotRunStatus) => void,
    options?: { interval?: number; maxAttempts?: number }
  ): Promise<CopilotRunResult<T>> {
    const interval = options?.interval ?? POLL_INTERVAL
    const maxAttempts = options?.maxAttempts ?? MAX_POLL_ATTEMPTS

    for (let attempt = 0; attempt < maxAttempts; attempt++) {
      const result = await getRun(runId) as CopilotRunResult<T>

      if (onProgress) {
        onProgress(result.status)
      }

      if (isTerminalStatus(result.status)) {
        return result
      }

      await new Promise(resolve => setTimeout(resolve, interval))
    }

    throw new Error('Polling timeout: run did not complete in time')
  }

  // ==========================================================================
  // Helper functions with polling (Builder integration)
  // ==========================================================================

  /**
   * Send message and poll for completion
   */
  async function sendMessageAndWait(
    projectId: string,
    sessionId: string,
    content: string,
    onProgress?: (status: CopilotRunStatus) => void
  ): Promise<GetSessionResponse> {
    const { run_id } = await sendMessage(projectId, sessionId, content)
    await pollForCompletion(run_id, onProgress)
    return getSession(projectId, sessionId)
  }

  /**
   * Construct workflow and poll for completion
   */
  async function constructAndWait(
    projectId: string,
    sessionId: string,
    onProgress?: (status: CopilotRunStatus) => void
  ): Promise<GetSessionResponse> {
    const { run_id } = await construct(projectId, sessionId)
    await pollForCompletion(run_id, onProgress)
    return getSession(projectId, sessionId)
  }

  /**
   * Refine workflow and poll for completion
   */
  async function refineAndWait(
    projectId: string,
    sessionId: string,
    feedback: string,
    onProgress?: (status: CopilotRunStatus) => void
  ): Promise<GetSessionResponse> {
    const { run_id } = await refine(projectId, sessionId, feedback)
    await pollForCompletion(run_id, onProgress)
    return getSession(projectId, sessionId)
  }

  // ==========================================================================
  // Async operations (global copilot)
  // ==========================================================================

  // Async generate workflow with polling
  async function asyncGenerateWorkflow(
    prompt: string,
    options?: { sessionId?: string; onProgress?: (status: CopilotRunStatus) => void }
  ): Promise<CopilotRunResult<GenerateWorkflowResponse>> {
    const { run_id } = await api.post<AsyncRunResponse>('/copilot/async/generate', {
      prompt,
      session_id: options?.sessionId,
    })

    return pollForCompletion<GenerateWorkflowResponse>(run_id, options?.onProgress)
  }

  // Async suggest with polling
  async function asyncSuggest(
    workflowId: string,
    options?: { context?: string; onProgress?: (status: CopilotRunStatus) => void }
  ): Promise<CopilotRunResult<SuggestResponse>> {
    const { run_id } = await api.post<AsyncRunResponse>('/copilot/async/suggest', {
      workflow_id: workflowId,
      context: options?.context,
    })

    return pollForCompletion<SuggestResponse>(run_id, options?.onProgress)
  }

  // Async diagnose with polling
  async function asyncDiagnose(
    runId: string,
    options?: { onProgress?: (status: CopilotRunStatus) => void }
  ): Promise<CopilotRunResult<DiagnoseResponse>> {
    const { run_id } = await api.post<AsyncRunResponse>('/copilot/async/diagnose', {
      run_id: runId,
    })

    return pollForCompletion<DiagnoseResponse>(run_id, options?.onProgress)
  }

  // Async optimize with polling
  async function asyncOptimize(
    workflowId: string,
    options?: { onProgress?: (status: CopilotRunStatus) => void }
  ): Promise<CopilotRunResult<OptimizeResponse>> {
    const { run_id } = await api.post<AsyncRunResponse>('/copilot/async/optimize', {
      workflow_id: workflowId,
    })

    return pollForCompletion<OptimizeResponse>(run_id, options?.onProgress)
  }

  // Start async operation without waiting (for manual polling)
  async function startAsyncGenerate(
    prompt: string,
    sessionId?: string
  ): Promise<string> {
    const { run_id } = await api.post<AsyncRunResponse>('/copilot/async/generate', {
      prompt,
      session_id: sessionId,
    })
    return run_id
  }

  async function startAsyncSuggest(
    workflowId: string,
    context?: string
  ): Promise<string> {
    const { run_id } = await api.post<AsyncRunResponse>('/copilot/async/suggest', {
      workflow_id: workflowId,
      context,
    })
    return run_id
  }

  async function startAsyncDiagnose(runId: string): Promise<string> {
    const { run_id } = await api.post<AsyncRunResponse>('/copilot/async/diagnose', {
      run_id: runId,
    })
    return run_id
  }

  async function startAsyncOptimize(workflowId: string): Promise<string> {
    const { run_id } = await api.post<AsyncRunResponse>('/copilot/async/optimize', {
      workflow_id: workflowId,
    })
    return run_id
  }

  // ==========================================================================
  // Agent-based Copilot (NEW: autonomous tool-calling agent with SSE streaming)
  // ==========================================================================

  /**
   * Start a new agent session
   */
  async function startAgentSession(
    projectId: string,
    initialPrompt: string,
    mode: CopilotSessionMode = 'create'
  ): Promise<AgentSessionResponse> {
    return api.post<AgentSessionResponse>(
      `/workflows/${projectId}/copilot/agent/sessions`,
      {
        initial_prompt: initialPrompt,
        mode,
      }
    )
  }

  /**
   * Send message to agent (non-streaming, waits for complete response)
   */
  async function sendAgentMessage(
    projectId: string,
    sessionId: string,
    content: string
  ): Promise<AgentMessageResponse> {
    return api.post<AgentMessageResponse>(
      `/workflows/${projectId}/copilot/agent/sessions/${sessionId}/messages`,
      { content }
    )
  }

  /**
   * Stream agent message with SSE (real-time updates)
   * Returns an EventSource connection and a state reactive object
   */
  function streamAgentMessage(
    projectId: string,
    sessionId: string,
    message: string,
    callbacks: {
      onThinking?: (content: string) => void
      onToolCall?: (tool: string, args: Record<string, unknown>) => void
      onToolResult?: (tool: string, result: unknown, isError: boolean) => void
      onPartialText?: (content: string) => void
      onComplete?: (response: string, toolsUsed: string[], iterations: number) => void
      onError?: (error: string) => void
    }
  ): { eventSource: EventSource; cancel: () => void } {
    const baseUrl = api.getBaseUrl?.() || ''
    const url = `${baseUrl}/workflows/${projectId}/copilot/agent/sessions/${sessionId}/stream?message=${encodeURIComponent(message)}`

    const eventSource = new EventSource(url, { withCredentials: true })

    // Track if we've been cancelled
    let cancelled = false

    // Handle events
    eventSource.addEventListener('thinking', (event) => {
      if (cancelled) return
      try {
        const parsed = JSON.parse(event.data) as { type: string; data: { content: string } }
        callbacks.onThinking?.(parsed.data.content)
      } catch (e) {
        console.error('Failed to parse thinking event', e)
      }
    })

    eventSource.addEventListener('tool_call', (event) => {
      if (cancelled) return
      try {
        const parsed = JSON.parse(event.data) as { type: string; data: { tool_name: string; arguments: Record<string, unknown> } }
        callbacks.onToolCall?.(parsed.data.tool_name, parsed.data.arguments)
      } catch (e) {
        console.error('Failed to parse tool_call event', e)
      }
    })

    eventSource.addEventListener('tool_result', (event) => {
      if (cancelled) return
      try {
        const parsed = JSON.parse(event.data) as { type: string; data: { tool_name: string; result: unknown; is_error: boolean } }
        callbacks.onToolResult?.(parsed.data.tool_name, parsed.data.result, parsed.data.is_error)
      } catch (e) {
        console.error('Failed to parse tool_result event', e)
      }
    })

    eventSource.addEventListener('partial_text', (event) => {
      if (cancelled) return
      try {
        const parsed = JSON.parse(event.data) as { type: string; data: { content: string } }
        callbacks.onPartialText?.(parsed.data.content)
      } catch (e) {
        console.error('Failed to parse partial_text event', e)
      }
    })

    eventSource.addEventListener('complete', (event) => {
      if (cancelled) return
      try {
        const parsed = JSON.parse(event.data) as { type: string; data: { response: string; tools_used: string[]; iterations: number; total_tokens: number } }
        callbacks.onComplete?.(parsed.data.response, parsed.data.tools_used, parsed.data.iterations)
      } catch (e) {
        console.error('Failed to parse complete event', e)
      }
      eventSource.close()
    })

    eventSource.addEventListener('error', (event) => {
      if (cancelled) return
      try {
        const parsed = JSON.parse((event as MessageEvent).data) as { type: string; data: { error: string } }
        callbacks.onError?.(parsed.data.error)
      } catch {
        callbacks.onError?.('Streaming error')
      }
      eventSource.close()
    })

    eventSource.addEventListener('done', () => {
      eventSource.close()
    })

    // Handle heartbeat/ping events (just ignore them - they keep the connection alive)
    eventSource.addEventListener('ping', () => {
      // Heartbeat received - connection is alive
    })

    eventSource.onerror = () => {
      if (!cancelled) {
        callbacks.onError?.('Connection error')
      }
      eventSource.close()
    }

    // Cancel function
    const cancel = async () => {
      cancelled = true
      eventSource.close()
      try {
        await api.post(`/workflows/${projectId}/copilot/agent/sessions/${sessionId}/cancel`, {})
      } catch (e) {
        console.warn('Failed to cancel agent stream', e)
      }
    }

    return { eventSource, cancel }
  }

  /**
   * Cancel an active agent stream
   */
  async function cancelAgentStream(
    projectId: string,
    sessionId: string
  ): Promise<{ status: string; message: string }> {
    return api.post<{ status: string; message: string }>(
      `/workflows/${projectId}/copilot/agent/sessions/${sessionId}/cancel`,
      {}
    )
  }

  /**
   * Get available agent tools
   */
  async function getAgentTools(): Promise<{ tools: AgentToolDefinition[]; count: number }> {
    return api.get<{ tools: AgentToolDefinition[]; count: number }>('/copilot/agent/tools')
  }

  return {
    // Legacy sync functions (global copilot)
    suggest,
    diagnose,
    explain,
    optimize,
    chat,
    suggestForStep,
    explainStep,

    // Session management (Builder integration)
    startSession,
    getSession,
    getSessionWithMessages,
    listSessions,
    sendMessage,
    construct,
    refine,
    finalize,
    deleteSession,

    // Helper functions with polling
    sendMessageAndWait,
    constructAndWait,
    refineAndWait,

    // Legacy session endpoints
    getOrCreateSession,
    startNewSession,
    getSessionMessages,
    chatWithSession,
    generateWorkflow,

    // Async functions with polling (meta-workflow architecture)
    getCopilotRun,
    getRun,
    pollForCompletion,
    asyncGenerateWorkflow,
    asyncSuggest,
    asyncDiagnose,
    asyncOptimize,

    // Start async without waiting (for manual polling)
    startAsyncGenerate,
    startAsyncSuggest,
    startAsyncDiagnose,
    startAsyncOptimize,

    // Agent-based copilot (NEW: autonomous tool-calling agent)
    startAgentSession,
    sendAgentMessage,
    streamAgentMessage,
    cancelAgentStream,
    getAgentTools,

    // Constants
    HEARING_PHASE_LABELS,
    HEARING_PHASES,
    SESSION_MODE_LABELS,
    ASSUMPTION_CATEGORY_LABELS,
  }
}
