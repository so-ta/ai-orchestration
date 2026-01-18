/**
 * AI Workflow Builder composable
 * Provides API access for AI-assisted workflow building
 */

interface BuilderSession {
  id: string
  tenant_id: string
  user_id: string
  status: BuilderSessionStatus
  hearing_phase: HearingPhase
  hearing_progress: number
  spec?: Record<string, unknown>
  project_id?: string
  copilot_session_id?: string
  messages?: BuilderMessage[]
  created_at: string
  updated_at: string
}

interface BuilderMessage {
  id: string
  role: 'user' | 'assistant'
  content: string
  suggested_questions?: string[]
  created_at: string
}

type BuilderSessionStatus = 'hearing' | 'building' | 'reviewing' | 'completed' | 'abandoned'
type HearingPhase = 'purpose' | 'conditions' | 'actors' | 'frequency' | 'integrations' | 'pain_points' | 'confirmation' | 'completed'

interface StartSessionResponse {
  session_id: string
  status: string
  phase: string
  progress: number
  message?: {
    id: string
    role: string
    content: string
    suggested_questions?: string[]
  }
}

interface SendMessageResponse {
  run_id: string
  status: string
}

interface GetSessionResponse {
  id: string
  status: string
  hearing_phase: string
  hearing_progress: number
  project_id?: string
  messages?: BuilderMessage[]
  created_at: string
  updated_at: string
}

interface ListSessionsResponse {
  sessions: Array<{
    id: string
    status: string
    hearing_phase: string
    hearing_progress: number
    project_id?: string
    created_at: string
    updated_at: string
  }>
  total: number
}

type BuilderRunStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled'

interface BuilderRunResult {
  run_id: string
  status: BuilderRunStatus
  started_at?: string
  completed_at?: string
  output?: Record<string, unknown>
  error?: string
}

const HEARING_PHASE_LABELS: Record<HearingPhase, string> = {
  purpose: '目的・ゴール',
  conditions: '開始・終了条件',
  actors: '関与者・承認',
  frequency: '実行頻度・期限',
  integrations: 'ツール連携',
  pain_points: '課題・困りごと',
  confirmation: '確認',
  completed: '完了',
}

const HEARING_PHASES: HearingPhase[] = [
  'purpose',
  'conditions',
  'actors',
  'frequency',
  'integrations',
  'pain_points',
  'confirmation',
  'completed',
]

export function useBuilder() {
  const { get, post, delete: del } = useApi()

  /**
   * Start a new builder session
   */
  async function startSession(initialPrompt: string): Promise<StartSessionResponse> {
    return post<StartSessionResponse>('/builder/sessions', {
      initial_prompt: initialPrompt,
    })
  }

  /**
   * Get a builder session by ID
   */
  async function getSession(sessionId: string): Promise<GetSessionResponse> {
    return get<GetSessionResponse>(`/builder/sessions/${sessionId}`)
  }

  /**
   * List all builder sessions
   */
  async function listSessions(): Promise<ListSessionsResponse> {
    return get<ListSessionsResponse>('/builder/sessions')
  }

  /**
   * Send a message to the builder (async)
   */
  async function sendMessage(sessionId: string, content: string): Promise<SendMessageResponse> {
    return post<SendMessageResponse>(`/builder/sessions/${sessionId}/messages`, {
      content,
    })
  }

  /**
   * Start workflow construction (async)
   */
  async function construct(sessionId: string): Promise<SendMessageResponse> {
    return post<SendMessageResponse>(`/builder/sessions/${sessionId}/construct`, {})
  }

  /**
   * Refine the workflow based on feedback (async)
   */
  async function refine(sessionId: string, feedback: string): Promise<SendMessageResponse> {
    return post<SendMessageResponse>(`/builder/sessions/${sessionId}/refine`, {
      feedback,
    })
  }

  /**
   * Finalize the workflow
   */
  async function finalize(sessionId: string): Promise<{ status: string }> {
    return post<{ status: string }>(`/builder/sessions/${sessionId}/finalize`, {})
  }

  /**
   * Delete a builder session
   */
  async function deleteSession(sessionId: string): Promise<void> {
    await del(`/builder/sessions/${sessionId}`)
  }

  /**
   * Get a run result by ID (for polling async operations)
   */
  async function getRun(runId: string): Promise<BuilderRunResult> {
    return get<BuilderRunResult>(`/runs/${runId}`)
  }

  /**
   * Poll for run completion
   */
  async function pollForCompletion(
    runId: string,
    onProgress?: (status: BuilderRunStatus) => void,
    options?: { interval?: number; maxAttempts?: number }
  ): Promise<BuilderRunResult> {
    const interval = options?.interval ?? 1000
    const maxAttempts = options?.maxAttempts ?? 60

    for (let attempt = 0; attempt < maxAttempts; attempt++) {
      const result = await getRun(runId)

      if (onProgress) {
        onProgress(result.status)
      }

      if (result.status === 'completed' || result.status === 'failed' || result.status === 'cancelled') {
        return result
      }

      await new Promise(resolve => setTimeout(resolve, interval))
    }

    throw new Error('Polling timeout: run did not complete in time')
  }

  /**
   * Send message and poll for completion
   */
  async function sendMessageAndWait(
    sessionId: string,
    content: string,
    onProgress?: (status: BuilderRunStatus) => void
  ): Promise<GetSessionResponse> {
    const { run_id } = await sendMessage(sessionId, content)
    await pollForCompletion(run_id, onProgress)
    return getSession(sessionId)
  }

  /**
   * Construct workflow and poll for completion
   */
  async function constructAndWait(
    sessionId: string,
    onProgress?: (status: BuilderRunStatus) => void
  ): Promise<GetSessionResponse> {
    const { run_id } = await construct(sessionId)
    await pollForCompletion(run_id, onProgress)
    return getSession(sessionId)
  }

  /**
   * Refine workflow and poll for completion
   */
  async function refineAndWait(
    sessionId: string,
    feedback: string,
    onProgress?: (status: BuilderRunStatus) => void
  ): Promise<GetSessionResponse> {
    const { run_id } = await refine(sessionId, feedback)
    await pollForCompletion(run_id, onProgress)
    return getSession(sessionId)
  }

  return {
    // Session management
    startSession,
    getSession,
    listSessions,
    deleteSession,

    // Actions
    sendMessage,
    construct,
    refine,
    finalize,

    // Async helpers (with polling)
    getRun,
    pollForCompletion,
    sendMessageAndWait,
    constructAndWait,
    refineAndWait,

    // Constants
    HEARING_PHASE_LABELS,
    HEARING_PHASES,
  }
}

export type {
  BuilderSession,
  BuilderMessage,
  BuilderSessionStatus,
  HearingPhase,
  StartSessionResponse,
  SendMessageResponse,
  GetSessionResponse,
  ListSessionsResponse,
  BuilderRunStatus,
  BuilderRunResult,
}
