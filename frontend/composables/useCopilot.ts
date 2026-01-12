// Copilot API composable for AI-assisted workflow building

// Types for Copilot requests and responses
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

// Session management types
export interface CopilotSession {
  id: string
  tenant_id: string
  user_id: string
  workflow_id: string
  title: string
  is_active: boolean
  created_at: string
  updated_at: string
  messages?: CopilotMessage[]
}

export interface CopilotMessage {
  id: string
  session_id: string
  role: 'user' | 'assistant'
  content: string
  metadata?: Record<string, unknown>
  created_at: string
}

export interface ChatWithSessionResponse {
  response: string
  suggestions?: StepSuggestion[]
  session: CopilotSession
}

// Workflow generation types
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

// Async execution types (meta-workflow architecture)
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

export function useCopilot() {
  const api = useApi()

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

  // Chat with the copilot
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

  // Suggest for a specific step (shortcut)
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

  // Explain a specific step (shortcut)
  async function explainStep(
    workflowId: string,
    stepId: string
  ): Promise<ExplainResponse> {
    return api.post<ExplainResponse>(
      `/workflows/${workflowId}/steps/${stepId}/copilot/explain`,
      {}
    )
  }

  // ========== Session Management ==========

  // Get or create the active session for a workflow
  async function getOrCreateSession(workflowId: string): Promise<CopilotSession> {
    return api.get<CopilotSession>(`/workflows/${workflowId}/copilot/session`)
  }

  // List all sessions for a workflow
  async function listSessions(workflowId: string): Promise<CopilotSession[]> {
    const response = await api.get<{ sessions: CopilotSession[] }>(
      `/workflows/${workflowId}/copilot/sessions`
    )
    return response.sessions || []
  }

  // Start a new chat session
  async function startNewSession(workflowId: string): Promise<CopilotSession> {
    return api.post<CopilotSession>(
      `/workflows/${workflowId}/copilot/sessions/new`,
      {}
    )
  }

  // Get session with all messages
  async function getSessionMessages(
    workflowId: string,
    sessionId: string
  ): Promise<CopilotSession> {
    return api.get<CopilotSession>(
      `/workflows/${workflowId}/copilot/sessions/${sessionId}`
    )
  }

  // Chat with session persistence
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

  // Generate workflow from natural language description
  async function generateWorkflow(
    workflowId: string,
    description: string
  ): Promise<GenerateWorkflowResponse> {
    return api.post<GenerateWorkflowResponse>(
      `/workflows/${workflowId}/copilot/generate`,
      { description }
    )
  }

  // ========== Async Operations with Polling (Meta-Workflow Architecture) ==========

  // Polling interval in milliseconds
  const POLL_INTERVAL = 1000
  const MAX_POLL_DURATION = 120000 // 2 minutes max

  // Get copilot run status
  async function getCopilotRun<T = unknown>(runId: string): Promise<CopilotRunResult<T>> {
    return api.get<CopilotRunResult<T>>(`/copilot/runs/${runId}`)
  }

  // Poll for run completion
  async function pollForCompletion<T = unknown>(
    runId: string,
    onProgress?: (status: CopilotRunStatus) => void
  ): Promise<CopilotRunResult<T>> {
    const startTime = Date.now()

    while (Date.now() - startTime < MAX_POLL_DURATION) {
      const result = await getCopilotRun<T>(runId)

      if (onProgress) {
        onProgress(result.status)
      }

      if (result.status === 'completed' || result.status === 'failed' || result.status === 'cancelled') {
        return result
      }

      // Wait before next poll
      await new Promise(resolve => setTimeout(resolve, POLL_INTERVAL))
    }

    // Timeout
    throw new Error('Copilot operation timed out')
  }

  // Async generate workflow with polling
  async function asyncGenerateWorkflow(
    prompt: string,
    options?: { sessionId?: string; onProgress?: (status: CopilotRunStatus) => void }
  ): Promise<CopilotRunResult<GenerateWorkflowResponse>> {
    // Start async operation
    const { run_id } = await api.post<AsyncRunResponse>('/copilot/async/generate', {
      prompt,
      session_id: options?.sessionId,
    })

    // Poll for completion
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

  return {
    // Legacy sync functions
    suggest,
    diagnose,
    explain,
    optimize,
    chat,
    suggestForStep,
    explainStep,
    // Session management
    getOrCreateSession,
    listSessions,
    startNewSession,
    getSessionMessages,
    chatWithSession,
    // Workflow generation (sync)
    generateWorkflow,
    // Async functions with polling (meta-workflow architecture)
    getCopilotRun,
    asyncGenerateWorkflow,
    asyncSuggest,
    asyncDiagnose,
    asyncOptimize,
    // Start async without waiting (for manual polling)
    startAsyncGenerate,
    startAsyncSuggest,
    startAsyncDiagnose,
    startAsyncOptimize,
  }
}
