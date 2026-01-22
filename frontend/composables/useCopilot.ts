// Copilot API composable for AI-assisted workflow building
// All copilot functionality is now handled by the Agent-based workflow engine.
//
// This file is the main entry point for Copilot functionality.
// Types and constants have been extracted to ./copilot/types.ts for maintainability.

// Import types for internal use
import type {
  AgentSessionResponse,
  AgentMessageResponse,
  CopilotSessionMode,
} from './copilot/types'

// Re-export all types and constants from the centralized types file
export * from './copilot/types'

// ============================================================================
// Main composable
// ============================================================================

export function useCopilot() {
  const api = useApi()

  // ==========================================================================
  // Agent-based Copilot (autonomous tool-calling agent with SSE streaming)
  // All copilot logic is implemented in the Copilot workflow (copilot.go)
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

    // Handle step-level errors with more context
    eventSource.addEventListener('step_error', (event) => {
      if (cancelled) return
      try {
        const parsed = JSON.parse(event.data) as { type: string; data: { step_id: string; step_name: string; error: string } }
        const errorMessage = `Step "${parsed.data.step_name}" failed: ${parsed.data.error}`
        callbacks.onError?.(errorMessage)
      } catch (e) {
        console.error('Failed to parse step_error event', e)
        callbacks.onError?.('Step execution error')
      }
    })

    eventSource.addEventListener('done', () => {
      eventSource.close()
    })

    // Handle heartbeat/ping events (just ignore them - they keep the connection alive)
    eventSource.addEventListener('ping', () => {
      // Heartbeat received - connection is alive
    })

    eventSource.onerror = (_event) => {
      if (!cancelled) {
        // Check if the connection was closed normally (readyState === 2 means CLOSED)
        if (eventSource.readyState === EventSource.CLOSED) {
          // Stream closed unexpectedly - may be due to server error
          callbacks.onError?.('Connection closed unexpectedly. The server may have encountered an error.')
        } else {
          callbacks.onError?.('Connection error. Please check your network connection.')
        }
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

  /**
   * Get agent session by ID (with messages)
   */
  async function getAgentSession(
    projectId: string,
    sessionId: string
  ): Promise<AgentSessionWithMessages> {
    return api.get<AgentSessionWithMessages>(
      `/workflows/${projectId}/copilot/agent/sessions/${sessionId}`
    )
  }

  /**
   * Get active agent session for a project (returns null if no active session)
   */
  async function getActiveAgentSession(
    projectId: string
  ): Promise<{ session: AgentSessionWithMessages | null }> {
    return api.get<{ session: AgentSessionWithMessages | null }>(
      `/workflows/${projectId}/copilot/agent/sessions/active`
    )
  }

  return {
    // Agent-based copilot (autonomous tool-calling agent)
    // All copilot functionality is now handled by the workflow engine
    startAgentSession,
    sendAgentMessage,
    streamAgentMessage,
    cancelAgentStream,
    getAgentTools,
    getAgentSession,
    getActiveAgentSession,

    // Constants
    HEARING_PHASE_LABELS,
    HEARING_PHASES,
    SESSION_MODE_LABELS,
    ASSUMPTION_CATEGORY_LABELS,
  }
}
