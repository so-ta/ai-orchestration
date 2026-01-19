<script setup lang="ts">
import type { Step } from '~/types/api'
import type {
  StepSuggestion,
  ExplainResponse,
  CopilotMessage,
  GenerateWorkflowResponse,
  CopilotRunStatus,
  CopilotSessionMode,
  CopilotPhase,
  CopilotSessionStatus,
  GetSessionResponse,
  AgentStreamState,
} from '~/composables/useCopilot'

const props = defineProps<{
  step: Step | null
  workflowId: string
}>()

const emit = defineEmits<{
  (e: 'apply-suggestion', suggestion: StepSuggestion): void
  (e: 'apply-workflow', workflow: GenerateWorkflowResponse): void
}>()

const { t } = useI18n()
const copilot = useCopilot()
const toast = useToast()

// State
const activeSection = ref<'chat' | 'suggest' | 'explain'>('chat')
const isLoading = ref(false)
const chatMessage = ref('')
const chatHistory = ref<Array<{ role: 'user' | 'assistant' | 'system'; content: string }>>([])
const suggestions = ref<StepSuggestion[]>([])
const explanation = ref<ExplainResponse | null>(null)

// Agent mode state (NEW: autonomous tool-calling agent)
const useAgentMode = ref(true) // Default to agent mode
const agentSessionId = ref<string | null>(null)
const agentStreamState = ref<AgentStreamState>({
  isStreaming: false,
  currentThinking: '',
  toolSteps: [],
  partialResponse: '',
  finalResponse: null,
  error: null,
})
const currentCancelFn = ref<(() => void) | null>(null)

// Session state (Builder integration)
const currentSession = ref<GetSessionResponse | null>(null)
const sessions = ref<Array<{ id: string; title?: string; status: CopilotSessionStatus; hearing_phase: CopilotPhase; hearing_progress: number; created_at: string }>>([])
const showSessionMenu = ref(false)
const isLoadingSession = ref(false)
const pollingStatus = ref<CopilotRunStatus | null>(null)

// Check if step is selected
const hasStep = computed(() => props.step !== null)

// Phase and progress computed properties
const sessionPhase = computed(() => currentSession.value?.hearing_phase ?? 'analysis')
const sessionProgress = computed(() => currentSession.value?.hearing_progress ?? 0)
const sessionStatus = computed(() => currentSession.value?.status ?? 'hearing')

// Is session in an active state (can accept messages)
const canSendMessage = computed(() => {
  if (!currentSession.value) return true // No session yet, will create on first message
  const status = currentSession.value.status
  return status === 'hearing' || status === 'reviewing' || status === 'refining'
})

// Is session ready for construction
const canConstruct = computed(() => {
  if (!currentSession.value) return false
  return currentSession.value.status === 'hearing' && currentSession.value.hearing_phase === 'completed'
})

// Is session in reviewing state (can refine or finalize)
const canRefineOrFinalize = computed(() => {
  if (!currentSession.value) return false
  return currentSession.value.status === 'reviewing' || currentSession.value.status === 'refining'
})

// Load session on mount
onMounted(async () => {
  await loadSession()
})

// Load current session
async function loadSession() {
  isLoadingSession.value = true
  try {
    // Try to load active session or sessions list
    const sessionsResponse = await copilot.listSessions(props.workflowId)
    sessions.value = sessionsResponse.sessions || []

    // If there's an active session, load it
    const activeSession = sessions.value.find(s => s.status !== 'completed' && s.status !== 'abandoned')
    if (activeSession) {
      const session = await copilot.getSession(props.workflowId, activeSession.id)
      currentSession.value = session
      if (session.messages && session.messages.length > 0) {
        chatHistory.value = session.messages.map((msg: CopilotMessage) => ({
          role: msg.role,
          content: msg.content,
        }))
      }
    }
  } catch (error) {
    console.error('Failed to load session:', error)
  } finally {
    isLoadingSession.value = false
  }
}

// Start new session with mode
async function startNewSession(mode: CopilotSessionMode = 'create') {
  if (!chatMessage.value.trim()) {
    toast.error(t('copilot.errors.emptyMessage'))
    return
  }

  isLoadingSession.value = true
  const initialPrompt = chatMessage.value.trim()
  chatMessage.value = ''

  try {
    const response = await copilot.startSession(props.workflowId, initialPrompt, mode)
    chatHistory.value = []

    // Add the initial message to chat
    chatHistory.value.push({ role: 'user', content: initialPrompt })
    if (response.message) {
      chatHistory.value.push({ role: 'assistant', content: response.message.content })
    }

    // Load the full session
    const session = await copilot.getSession(props.workflowId, response.session_id)
    currentSession.value = session

    // Refresh sessions list
    const sessionsResponse = await copilot.listSessions(props.workflowId)
    sessions.value = sessionsResponse.sessions || []

    showSessionMenu.value = false
    toast.success(t('copilot.newSessionStarted'))
  } catch (error) {
    toast.error(t('copilot.errors.sessionFailed'))
    console.error('Failed to start new session:', error)
  } finally {
    isLoadingSession.value = false
  }
}

// Switch to a different session
async function switchSession(sessionId: string) {
  isLoadingSession.value = true
  try {
    const session = await copilot.getSession(props.workflowId, sessionId)
    currentSession.value = session
    chatHistory.value = session.messages?.map((msg: CopilotMessage) => ({
      role: msg.role,
      content: msg.content,
    })) || []
    showSessionMenu.value = false
  } catch (error) {
    toast.error(t('copilot.errors.sessionFailed'))
    console.error('Failed to switch session:', error)
  } finally {
    isLoadingSession.value = false
  }
}

// Format session title for display
function formatSessionTitle(session: { title?: string; created_at: string }): string {
  if (session.title) return session.title
  const date = new Date(session.created_at)
  return date.toLocaleDateString() + ' ' + date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

// Send message (with session support)
async function sendMessage() {
  if (!chatMessage.value.trim() || isLoading.value) return

  const message = chatMessage.value.trim()

  // If no session exists, start a new one
  if (!currentSession.value) {
    await startNewSession('create')
    return
  }

  chatMessage.value = ''
  chatHistory.value.push({ role: 'user', content: message })

  isLoading.value = true
  pollingStatus.value = 'pending'

  try {
    const session = await copilot.sendMessageAndWait(
      props.workflowId,
      currentSession.value.id,
      message,
      (status) => { pollingStatus.value = status }
    )

    // Update session and chat history
    currentSession.value = session
    if (session.messages && session.messages.length > 0) {
      const lastMsg = session.messages[session.messages.length - 1]
      if (lastMsg.role === 'assistant') {
        chatHistory.value.push({ role: 'assistant', content: lastMsg.content })
      }
    }
  } catch (error: unknown) {
    toast.error(t('copilot.errors.chatFailed'))
    console.error('Chat error:', error)
  } finally {
    isLoading.value = false
    pollingStatus.value = null
  }
}

// Construct workflow
async function handleConstruct() {
  if (!currentSession.value || !canConstruct.value || isLoading.value) return

  isLoading.value = true
  pollingStatus.value = 'pending'
  chatHistory.value.push({ role: 'system', content: t('copilot.startingConstruction') })

  try {
    const session = await copilot.constructAndWait(
      props.workflowId,
      currentSession.value.id,
      (status) => { pollingStatus.value = status }
    )

    currentSession.value = session

    // Add construction result to chat
    if (session.messages && session.messages.length > 0) {
      const lastMsg = session.messages[session.messages.length - 1]
      if (lastMsg.role === 'assistant') {
        chatHistory.value.push({ role: 'assistant', content: lastMsg.content })
      }
    }

    toast.success(t('copilot.workflowConstructed'))
  } catch (error) {
    toast.error(t('copilot.errors.constructFailed'))
    console.error('Construct error:', error)
  } finally {
    isLoading.value = false
    pollingStatus.value = null
  }
}

// Refine workflow
async function handleRefine() {
  if (!currentSession.value || !canRefineOrFinalize.value || !chatMessage.value.trim() || isLoading.value) return

  const feedback = chatMessage.value.trim()
  chatMessage.value = ''
  chatHistory.value.push({ role: 'user', content: `[Refine] ${feedback}` })

  isLoading.value = true
  pollingStatus.value = 'pending'

  try {
    const session = await copilot.refineAndWait(
      props.workflowId,
      currentSession.value.id,
      feedback,
      (status) => { pollingStatus.value = status }
    )

    currentSession.value = session

    // Add refine result to chat
    if (session.messages && session.messages.length > 0) {
      const lastMsg = session.messages[session.messages.length - 1]
      if (lastMsg.role === 'assistant') {
        chatHistory.value.push({ role: 'assistant', content: lastMsg.content })
      }
    }

    toast.success(t('copilot.workflowRefined'))
  } catch (error) {
    toast.error(t('copilot.errors.refineFailed'))
    console.error('Refine error:', error)
  } finally {
    isLoading.value = false
    pollingStatus.value = null
  }
}

// Finalize session
async function handleFinalize() {
  if (!currentSession.value || !canRefineOrFinalize.value || isLoading.value) return

  isLoading.value = true

  try {
    await copilot.finalize(props.workflowId, currentSession.value.id)

    // Reload session
    const session = await copilot.getSession(props.workflowId, currentSession.value.id)
    currentSession.value = session

    chatHistory.value.push({ role: 'system', content: t('copilot.sessionFinalized') })
    toast.success(t('copilot.sessionFinalized'))
  } catch (error) {
    toast.error(t('copilot.errors.finalizeFailed'))
    console.error('Finalize error:', error)
  } finally {
    isLoading.value = false
  }
}

// Delete session
async function handleDeleteSession() {
  if (!currentSession.value) return

  if (!confirm(t('copilot.confirmDelete'))) return

  isLoadingSession.value = true

  try {
    await copilot.deleteSession(props.workflowId, currentSession.value.id)
    currentSession.value = null
    chatHistory.value = []

    // Refresh sessions list
    const sessionsResponse = await copilot.listSessions(props.workflowId)
    sessions.value = sessionsResponse.sessions || []

    toast.success(t('copilot.sessionDeleted'))
  } catch (error) {
    toast.error(t('copilot.errors.deleteFailed'))
    console.error('Delete error:', error)
  } finally {
    isLoadingSession.value = false
  }
}

// Get suggestions for current step
async function fetchSuggestions() {
  if (!props.step) return
  isLoading.value = true
  try {
    const response = await copilot.suggestForStep(
      props.workflowId,
      props.step.id
    )
    suggestions.value = response.suggestions
    activeSection.value = 'suggest'
  } catch (error: unknown) {
    toast.error(t('copilot.errors.suggestFailed'))
    console.error('Suggest error:', error)
  } finally {
    isLoading.value = false
  }
}

// Get explanation for current step
async function fetchExplanation() {
  if (!props.step) return
  isLoading.value = true
  try {
    const response = await copilot.explainStep(props.workflowId, props.step.id)
    explanation.value = response
    activeSection.value = 'explain'
  } catch (error: unknown) {
    toast.error(t('copilot.errors.explainFailed'))
    console.error('Explain error:', error)
  } finally {
    isLoading.value = false
  }
}

// Apply a suggestion
function applySuggestion(suggestion: StepSuggestion) {
  emit('apply-suggestion', suggestion)
  toast.success(t('copilot.suggestionApplied'))
}

// Quick actions
const quickActions = [
  { id: 'suggest', icon: 'lightbulb', label: 'copilot.actions.suggest', action: fetchSuggestions },
  { id: 'explain', icon: 'info', label: 'copilot.actions.explain', action: fetchExplanation },
]

// Status badge color
function getStatusColor(status: CopilotSessionStatus): string {
  switch (status) {
    case 'hearing': return 'var(--color-primary)'
    case 'building': return 'var(--color-warning)'
    case 'reviewing': return 'var(--color-success)'
    case 'refining': return 'var(--color-warning)'
    case 'completed': return 'var(--color-text-secondary)'
    case 'abandoned': return 'var(--color-error)'
    default: return 'var(--color-text-secondary)'
  }
}

// ==========================================================================
// Agent Mode Functions (NEW: autonomous tool-calling agent with SSE streaming)
// ==========================================================================

// Reset agent stream state
function resetAgentState() {
  agentStreamState.value = {
    isStreaming: false,
    currentThinking: '',
    toolSteps: [],
    partialResponse: '',
    finalResponse: null,
    error: null,
  }
  currentCancelFn.value = null
}

// Send message in agent mode (with streaming)
async function sendAgentMessage() {
  if (!chatMessage.value.trim() || agentStreamState.value.isStreaming) return

  const message = chatMessage.value.trim()
  chatMessage.value = ''

  // Add user message to chat
  chatHistory.value.push({ role: 'user', content: message })

  // Reset agent state for new message
  resetAgentState()
  agentStreamState.value.isStreaming = true

  try {
    // Start new session if needed
    if (!agentSessionId.value) {
      const response = await copilot.startAgentSession(props.workflowId, message, 'create')
      agentSessionId.value = response.session_id

      // Add the initial response
      chatHistory.value.push({ role: 'assistant', content: response.response })
      agentStreamState.value.isStreaming = false

      // Show tools used if any
      if (response.tools_used.length > 0) {
        agentStreamState.value.toolSteps = response.tools_used.map((tool, idx) => ({
          id: `init-${idx}`,
          tool,
          status: 'success' as const,
        }))
      }
      return
    }

    // Stream message with SSE
    const { cancel } = copilot.streamAgentMessage(
      props.workflowId,
      agentSessionId.value,
      message,
      {
        onThinking: (content) => {
          agentStreamState.value.currentThinking = content
        },
        onToolCall: (tool, args) => {
          const stepId = `tool-${Date.now()}`
          agentStreamState.value.toolSteps.push({
            id: stepId,
            tool,
            status: 'calling',
            arguments: args,
          })
          agentStreamState.value.currentThinking = ''
        },
        onToolResult: (tool, result, isError) => {
          const step = agentStreamState.value.toolSteps.find(s => s.tool === tool && s.status === 'calling')
          if (step) {
            step.status = isError ? 'error' : 'success'
            step.result = result
            if (isError) step.error = String(result)
          }
        },
        onPartialText: (content) => {
          agentStreamState.value.partialResponse += content
        },
        onComplete: (response, toolsUsed, iterations) => {
          agentStreamState.value.isStreaming = false
          agentStreamState.value.finalResponse = response
          agentStreamState.value.currentThinking = ''
          agentStreamState.value.partialResponse = ''
          currentCancelFn.value = null

          // Add assistant response to chat
          chatHistory.value.push({ role: 'assistant', content: response })

          // Show completion info
          if (toolsUsed.length > 0) {
            console.log(`Agent completed in ${iterations} iterations, used tools: ${toolsUsed.join(', ')}`)
          }
        },
        onError: (error) => {
          agentStreamState.value.isStreaming = false
          agentStreamState.value.error = error
          agentStreamState.value.currentThinking = ''
          currentCancelFn.value = null
          toast.error(t('copilot.errors.agentFailed'))
        },
      }
    )

    currentCancelFn.value = cancel
  } catch (error) {
    agentStreamState.value.isStreaming = false
    agentStreamState.value.error = String(error)
    toast.error(t('copilot.errors.agentFailed'))
  }
}

// Cancel agent streaming
function cancelAgentStream() {
  if (currentCancelFn.value) {
    currentCancelFn.value()
    currentCancelFn.value = null
    agentStreamState.value.isStreaming = false
    agentStreamState.value.currentThinking = ''
    toast.info(t('copilot.agent.cancelled'))
  }
}

// Reset agent session to start fresh
function resetAgentSession() {
  agentSessionId.value = null
  chatHistory.value = []
  resetAgentState()
  toast.info(t('copilot.agent.resetSession'))
}

// Handle send based on mode
async function handleSend() {
  if (useAgentMode.value) {
    await sendAgentMessage()
  } else {
    await sendMessage()
  }
}

// Computed for showing agent streaming UI
const showAgentStreaming = computed(() => {
  return useAgentMode.value && (
    agentStreamState.value.isStreaming ||
    agentStreamState.value.toolSteps.length > 0 ||
    agentStreamState.value.currentThinking ||
    agentStreamState.value.error
  )
})
</script>

<template>
  <div class="copilot-tab">
    <!-- Mode Toggle (Agent vs Legacy) -->
    <div class="mode-toggle">
      <button
        class="mode-btn"
        :class="{ active: useAgentMode }"
        @click="useAgentMode = true"
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="3"/>
          <path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/>
        </svg>
        {{ t('copilot.agent.mode') }}
      </button>
      <button
        class="mode-btn"
        :class="{ active: !useAgentMode }"
        @click="useAgentMode = false"
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
        </svg>
        {{ t('copilot.agent.legacyMode') }}
      </button>
      <button
        v-if="useAgentMode && agentSessionId"
        class="reset-btn"
        :title="t('copilot.agent.resetSession')"
        @click="resetAgentSession"
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8"/>
          <path d="M21 3v5h-5"/>
          <path d="M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16"/>
          <path d="M3 21v-5h5"/>
        </svg>
      </button>
    </div>

    <!-- Session Header (Legacy mode only) -->
    <div v-if="!useAgentMode" class="session-header">
      <div class="session-selector" @click="showSessionMenu = !showSessionMenu">
        <span class="session-label">{{ t('copilot.session') }}:</span>
        <span class="session-title">{{ currentSession ? formatSessionTitle(currentSession) : t('copilot.noActiveSession') }}</span>
        <svg class="session-chevron" :class="{ open: showSessionMenu }" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="6 9 12 15 18 9"/>
        </svg>
      </div>
      <button
        v-if="currentSession"
        class="delete-session-btn"
        :disabled="isLoadingSession"
        :title="t('copilot.deleteSession')"
        @click="handleDeleteSession"
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="3 6 5 6 21 6"/>
          <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
        </svg>
      </button>
      <!-- Session Dropdown -->
      <div v-if="showSessionMenu" class="session-dropdown">
        <div class="session-dropdown-header">{{ t('copilot.sessionHistory') }}</div>
        <div class="session-list">
          <div
            v-for="session in sessions"
            :key="session.id"
            class="session-item"
            :class="{ active: currentSession?.id === session.id }"
            @click="switchSession(session.id)"
          >
            <span class="session-item-title">{{ formatSessionTitle(session) }}</span>
            <span class="session-status-badge" :style="{ background: getStatusColor(session.status) }">
              {{ t(`copilot.status.${session.status}`) }}
            </span>
          </div>
          <div v-if="sessions.length === 0" class="session-empty">{{ t('copilot.noSessions') }}</div>
        </div>
      </div>
    </div>

    <!-- Session Progress (when session exists) -->
    <div v-if="currentSession" class="session-progress">
      <div class="progress-header">
        <span class="progress-phase">{{ t(`copilot.phase.${sessionPhase}`) }}</span>
        <span class="progress-status" :style="{ color: getStatusColor(sessionStatus) }">
          {{ t(`copilot.status.${sessionStatus}`) }}
        </span>
      </div>
      <div class="progress-bar">
        <div class="progress-fill" :style="{ width: `${sessionProgress}%` }" />
      </div>
    </div>

    <!-- Action Buttons (based on session state) -->
    <div v-if="currentSession" class="action-buttons">
      <button
        v-if="canConstruct"
        class="action-btn construct"
        :disabled="isLoading"
        @click="handleConstruct"
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M12 3v3m0 12v3M3 12h3m12 0h3M5.636 5.636l2.122 2.122m8.484 8.484l2.122 2.122M5.636 18.364l2.122-2.122m8.484-8.484l2.122-2.122"/>
        </svg>
        {{ t('copilot.construct') }}
      </button>
      <button
        v-if="canRefineOrFinalize"
        class="action-btn finalize"
        :disabled="isLoading"
        @click="handleFinalize"
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="20 6 9 17 4 12"/>
        </svg>
        {{ t('copilot.finalize') }}
      </button>
    </div>

    <!-- Quick Actions (require step selection) -->
    <div class="quick-actions">
      <button
        v-for="action in quickActions"
        :key="action.id"
        class="quick-action-btn"
        :class="{ active: activeSection === action.id }"
        :disabled="isLoading || !hasStep"
        :title="!hasStep ? t('copilot.selectStepFirst') : ''"
        @click="action.action"
      >
        <span class="action-icon">
          <svg v-if="action.icon === 'lightbulb'" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M9 18h6"/>
            <path d="M10 22h4"/>
            <path d="M15.09 14c.18-.98.65-1.74 1.41-2.5A4.65 4.65 0 0 0 18 8 6 6 0 0 0 6 8c0 1 .23 2.23 1.5 3.5A4.61 4.61 0 0 1 8.91 14"/>
          </svg>
          <svg v-else-if="action.icon === 'info'" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/>
            <path d="M12 16v-4"/>
            <path d="M12 8h.01"/>
          </svg>
        </span>
        {{ t(action.label) }}
      </button>
    </div>

    <!-- Loading Indicator -->
    <div v-if="isLoading" class="loading-indicator">
      <div class="loading-spinner" />
      <span>
        {{ pollingStatus === 'running' ? t('copilot.processing') : t('copilot.thinking') }}
      </span>
    </div>

    <!-- Chat Section -->
    <div v-if="activeSection === 'chat'" class="chat-section">
      <div class="chat-messages">
        <div v-if="chatHistory.length === 0" class="chat-empty">
          <p>{{ useAgentMode ? t('copilot.agent.welcome') : t('copilot.chatWelcome') }}</p>
          <p class="chat-hint">{{ useAgentMode ? t('copilot.agent.hint') : t('copilot.chatHint') }}</p>
        </div>
        <div
          v-for="(msg, idx) in chatHistory"
          :key="idx"
          class="chat-message"
          :class="msg.role"
        >
          <div class="message-content">{{ msg.content }}</div>
        </div>

        <!-- Agent Streaming UI (NEW) -->
        <div v-if="showAgentStreaming" class="agent-streaming">
          <!-- Thinking indicator -->
          <div v-if="agentStreamState.currentThinking" class="thinking-bubble">
            <div class="thinking-icon">
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="3"/>
                <path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/>
              </svg>
            </div>
            <span class="thinking-text">{{ agentStreamState.currentThinking }}</span>
          </div>

          <!-- Tool execution steps -->
          <div v-if="agentStreamState.toolSteps.length > 0" class="tool-steps">
            <div
              v-for="step in agentStreamState.toolSteps"
              :key="step.id"
              class="tool-step"
              :class="step.status"
            >
              <div class="tool-header">
                <span class="tool-icon">
                  <svg v-if="step.status === 'calling'" class="spin" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
                  </svg>
                  <svg v-else-if="step.status === 'success'" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="20 6 9 17 4 12"/>
                  </svg>
                  <svg v-else xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="12" cy="12" r="10"/>
                    <line x1="15" y1="9" x2="9" y2="15"/>
                    <line x1="9" y1="9" x2="15" y2="15"/>
                  </svg>
                </span>
                <span class="tool-name">{{ step.tool }}</span>
                <span class="tool-status">{{ t(`copilot.toolStatus.${step.status}`) }}</span>
              </div>
              <div v-if="step.error" class="tool-error">{{ step.error }}</div>
            </div>
          </div>

          <!-- Partial response (streaming text) -->
          <div v-if="agentStreamState.partialResponse" class="partial-response">
            {{ agentStreamState.partialResponse }}
            <span class="cursor-blink">▊</span>
          </div>

          <!-- Error display -->
          <div v-if="agentStreamState.error" class="agent-error">
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"/>
              <line x1="12" y1="8" x2="12" y2="12"/>
              <line x1="12" y1="16" x2="12.01" y2="16"/>
            </svg>
            <span>{{ agentStreamState.error }}</span>
          </div>
        </div>
      </div>

      <!-- Chat input -->
      <div class="chat-input-container">
        <!-- Cancel button (during streaming) -->
        <button
          v-if="agentStreamState.isStreaming"
          class="cancel-btn"
          @click="cancelAgentStream"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
          </svg>
          {{ t('copilot.agent.cancel') }}
        </button>

        <textarea
          v-model="chatMessage"
          class="chat-input"
          :placeholder="useAgentMode ? t('copilot.agent.placeholder') : (canRefineOrFinalize ? t('copilot.refinePlaceholder') : t('copilot.chatPlaceholder'))"
          :disabled="isLoading || agentStreamState.isStreaming || (!useAgentMode && !canSendMessage)"
          rows="2"
          @keydown.meta.enter="handleSend"
          @keydown.ctrl.enter="handleSend"
        />
        <button
          v-if="!useAgentMode && canRefineOrFinalize"
          class="chat-send-btn refine"
          :disabled="!chatMessage.trim() || isLoading"
          @click="handleRefine"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
            <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
          </svg>
        </button>
        <button
          v-else-if="!agentStreamState.isStreaming"
          class="chat-send-btn"
          :class="{ agent: useAgentMode }"
          :disabled="!chatMessage.trim() || isLoading || agentStreamState.isStreaming"
          @click="handleSend"
        >
          <svg v-if="useAgentMode" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="3"/>
            <path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/>
          </svg>
          <svg v-else xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="22" y1="2" x2="11" y2="13"/>
            <polygon points="22 2 15 22 11 13 2 9 22 2"/>
          </svg>
        </button>
      </div>
    </div>

    <!-- Suggestions Section -->
    <div v-else-if="activeSection === 'suggest'" class="suggest-section">
      <div v-if="suggestions.length === 0" class="empty-suggestions">
        <p>{{ t('copilot.noSuggestions') }}</p>
      </div>
      <div v-else class="suggestions-list">
        <div
          v-for="(suggestion, idx) in suggestions"
          :key="idx"
          class="suggestion-card"
        >
          <div class="suggestion-header">
            <span class="suggestion-type">{{ suggestion.type }}</span>
            <span class="suggestion-name">{{ suggestion.name }}</span>
          </div>
          <p class="suggestion-desc">{{ suggestion.description }}</p>
          <p class="suggestion-reason">{{ suggestion.reason }}</p>
          <button class="btn-apply" @click="applySuggestion(suggestion)">
            {{ t('copilot.applySuggestion') }}
          </button>
        </div>
      </div>
    </div>

    <!-- Explanation Section -->
    <div v-else-if="activeSection === 'explain'" class="explain-section">
      <div v-if="!explanation" class="empty-explanation">
        <p>{{ t('copilot.noExplanation') }}</p>
      </div>
      <div v-else class="explanation-content">
        <h4 class="explanation-title">{{ t('copilot.explanationTitle') }}</h4>
        <p class="explanation-summary">{{ explanation.summary }}</p>
        <div v-if="explanation.step_details && explanation.step_details.length > 0" class="step-details">
          <div
            v-for="detail in explanation.step_details"
            :key="detail.step_id"
            class="step-detail"
          >
            <span class="detail-name">{{ detail.step_name }}</span>
            <p class="detail-explanation">{{ detail.explanation }}</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.copilot-tab {
  display: flex;
  flex-direction: column;
  height: 100%;
  gap: 0.75rem;
}

/* Mode Toggle */
.mode-toggle {
  display: flex;
  gap: 0.25rem;
  padding: 0.25rem;
  background: var(--color-background);
  border-radius: 8px;
}

.mode-btn {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.5rem 0.75rem;
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--color-text-secondary);
  background: transparent;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
}

.mode-btn:hover {
  color: var(--color-text);
  background: var(--color-surface);
}

.mode-btn.active {
  color: white;
  background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
}

.reset-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  margin-left: auto;
  color: var(--color-text-secondary);
  background: transparent;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
}

.reset-btn:hover {
  color: var(--color-error);
  background: rgba(239, 68, 68, 0.1);
}

/* Agent Streaming UI */
.agent-streaming {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding: 0.75rem;
  background: var(--color-background);
  border-radius: 8px;
  margin-top: 0.5rem;
}

.thinking-bubble {
  display: flex;
  align-items: flex-start;
  gap: 0.5rem;
  padding: 0.625rem 0.75rem;
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.1) 0%, rgba(139, 92, 246, 0.1) 100%);
  border: 1px dashed rgba(99, 102, 241, 0.3);
  border-radius: 8px;
  color: var(--color-primary);
  font-size: 0.8125rem;
  font-style: italic;
}

.thinking-icon {
  display: flex;
  animation: pulse 2s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.thinking-text {
  flex: 1;
  line-height: 1.5;
}

/* Tool Steps */
.tool-steps {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.tool-step {
  display: flex;
  flex-direction: column;
  padding: 0.5rem 0.75rem;
  background: var(--color-surface);
  border-radius: 6px;
  border-left: 3px solid var(--color-border);
}

.tool-step.calling {
  border-left-color: var(--color-warning);
}

.tool-step.success {
  border-left-color: var(--color-success);
}

.tool-step.error {
  border-left-color: var(--color-error);
}

.tool-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.tool-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
}

.tool-step.calling .tool-icon {
  color: var(--color-warning);
}

.tool-step.success .tool-icon {
  color: var(--color-success);
}

.tool-step.error .tool-icon {
  color: var(--color-error);
}

.tool-icon .spin {
  animation: spin 1s linear infinite;
}

.tool-name {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--color-text);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}

.tool-status {
  margin-left: auto;
  font-size: 0.6875rem;
  font-weight: 500;
  text-transform: uppercase;
}

.tool-step.calling .tool-status {
  color: var(--color-warning);
}

.tool-step.success .tool-status {
  color: var(--color-success);
}

.tool-step.error .tool-status {
  color: var(--color-error);
}

.tool-error {
  margin-top: 0.25rem;
  font-size: 0.75rem;
  color: var(--color-error);
}

/* Partial Response */
.partial-response {
  padding: 0.625rem 0.75rem;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 8px;
  font-size: 0.8125rem;
  line-height: 1.6;
  color: var(--color-text);
  white-space: pre-wrap;
}

.cursor-blink {
  animation: blink 1s step-end infinite;
  color: var(--color-primary);
}

@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0; }
}

/* Agent Error */
.agent-error {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.625rem 0.75rem;
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid var(--color-error);
  border-radius: 8px;
  font-size: 0.8125rem;
  color: var(--color-error);
}

/* Cancel Button */
.cancel-btn {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.5rem 0.75rem;
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--color-error);
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid var(--color-error);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
}

.cancel-btn:hover {
  background: var(--color-error);
  color: white;
}

/* Agent Send Button Style */
.chat-send-btn.agent {
  background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
}

/* Session Header */
.session-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  position: relative;
}

.session-selector {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.375rem 0.5rem;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: 6px;
  cursor: pointer;
  transition: border-color 0.15s;
  flex: 1;
  min-width: 0;
}

.session-selector:hover {
  border-color: var(--color-primary);
}

.session-label {
  font-weight: 500;
  white-space: nowrap;
}

.session-title {
  color: var(--color-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.session-chevron {
  margin-left: auto;
  flex-shrink: 0;
  transition: transform 0.15s;
}

.session-chevron.open {
  transform: rotate(180deg);
}

.delete-session-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  padding: 0;
  color: var(--color-text-secondary);
  background: transparent;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
}

.delete-session-btn:hover:not(:disabled) {
  background: var(--color-error);
  border-color: var(--color-error);
  color: white;
}

.delete-session-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Session Dropdown */
.session-dropdown {
  position: absolute;
  top: 100%;
  left: 0;
  right: 0;
  margin-top: 0.25rem;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  z-index: 100;
  max-height: 200px;
  overflow: hidden;
}

.session-dropdown-header {
  padding: 0.5rem 0.75rem;
  font-size: 0.6875rem;
  font-weight: 600;
  text-transform: uppercase;
  color: var(--color-text-secondary);
  background: var(--color-background);
  border-bottom: 1px solid var(--color-border);
}

.session-list {
  overflow-y: auto;
  max-height: 160px;
}

.session-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
  font-size: 0.75rem;
  color: var(--color-text);
  cursor: pointer;
  transition: background 0.15s;
}

.session-item:hover {
  background: var(--color-background);
}

.session-item.active {
  background: rgba(59, 130, 246, 0.1);
}

.session-item-title {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.session-status-badge {
  font-size: 0.625rem;
  font-weight: 600;
  padding: 0.125rem 0.375rem;
  color: white;
  border-radius: 4px;
}

.session-empty {
  padding: 1rem;
  text-align: center;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

/* Session Progress */
.session-progress {
  padding: 0.5rem;
  background: var(--color-background);
  border-radius: 6px;
}

.progress-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.375rem;
}

.progress-phase {
  font-size: 0.6875rem;
  font-weight: 600;
  text-transform: uppercase;
  color: var(--color-text-secondary);
}

.progress-status {
  font-size: 0.6875rem;
  font-weight: 600;
}

.progress-bar {
  height: 4px;
  background: var(--color-border);
  border-radius: 2px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: var(--color-primary);
  border-radius: 2px;
  transition: width 0.3s ease;
}

/* Action Buttons */
.action-buttons {
  display: flex;
  gap: 0.5rem;
}

.action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  flex: 1;
  padding: 0.625rem 1rem;
  font-size: 0.8125rem;
  font-weight: 600;
  color: white;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
}

.action-btn.construct {
  background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
}

.action-btn.finalize {
  background: linear-gradient(135deg, #10b981 0%, #059669 100%);
}

.action-btn:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
}

.action-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

/* Quick Actions */
.quick-actions {
  display: flex;
  gap: 0.5rem;
  padding: 0.5rem 0;
  border-bottom: 1px solid var(--color-border);
}

.quick-action-btn {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.5rem 0.75rem;
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--color-text-secondary);
  background: transparent;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
}

.quick-action-btn:hover:not(:disabled) {
  background: var(--color-background);
  border-color: var(--color-primary);
  color: var(--color-primary);
}

.quick-action-btn.active {
  background: rgba(59, 130, 246, 0.1);
  border-color: var(--color-primary);
  color: var(--color-primary);
}

.quick-action-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.action-icon {
  display: flex;
  align-items: center;
  justify-content: center;
}

/* Loading */
.loading-indicator {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  background: var(--color-background);
  border-radius: 6px;
}

.loading-spinner {
  width: 16px;
  height: 16px;
  border: 2px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Chat Section */
.chat-section {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}

.chat-messages {
  flex: 1;
  overflow-y: auto;
  padding: 0.5rem 0;
}

.chat-empty {
  text-align: center;
  padding: 2rem 1rem;
  color: var(--color-text-secondary);
}

.chat-empty p {
  margin: 0;
  font-size: 0.8125rem;
}

.chat-hint {
  margin-top: 0.5rem !important;
  font-size: 0.75rem !important;
  opacity: 0.7;
}

.chat-message {
  padding: 0.5rem 0.75rem;
  margin-bottom: 0.5rem;
  border-radius: 8px;
  font-size: 0.8125rem;
  line-height: 1.5;
}

.chat-message.user {
  background: var(--color-primary);
  color: white;
  margin-left: 1rem;
}

.chat-message.assistant {
  background: var(--color-background);
  border: 1px solid var(--color-border);
  margin-right: 1rem;
}

.chat-message.system {
  background: rgba(59, 130, 246, 0.1);
  border: 1px dashed var(--color-primary);
  color: var(--color-primary);
  text-align: center;
  font-size: 0.75rem;
  font-style: italic;
}

.chat-input-container {
  display: flex;
  align-items: flex-end;
  gap: 0.5rem;
  padding-top: 0.5rem;
  border-top: 1px solid var(--color-border);
}

.chat-input {
  flex: 1;
  padding: 0.625rem 0.75rem;
  font-size: 0.8125rem;
  font-family: inherit;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  outline: none;
  transition: border-color 0.15s;
  resize: none;
  min-height: 40px;
  max-height: 100px;
  line-height: 1.4;
}

.chat-input:focus {
  border-color: var(--color-primary);
}

.chat-send-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  padding: 0;
  background: var(--color-primary);
  color: white;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  transition: opacity 0.15s;
}

.chat-send-btn.refine {
  background: var(--color-warning);
}

.chat-send-btn:hover:not(:disabled) {
  opacity: 0.9;
}

.chat-send-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Suggestions Section */
.suggest-section {
  flex: 1;
  overflow-y: auto;
}

.empty-suggestions,
.empty-explanation {
  text-align: center;
  padding: 2rem 1rem;
  color: var(--color-text-secondary);
  font-size: 0.8125rem;
}

.suggestions-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.suggestion-card {
  padding: 0.75rem;
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: 8px;
}

.suggestion-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.375rem;
}

.suggestion-type {
  font-size: 0.625rem;
  font-weight: 600;
  text-transform: uppercase;
  padding: 0.125rem 0.375rem;
  background: rgba(59, 130, 246, 0.1);
  color: var(--color-primary);
  border-radius: 4px;
}

.suggestion-name {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--color-text);
}

.suggestion-desc {
  font-size: 0.75rem;
  color: var(--color-text);
  margin: 0 0 0.25rem;
}

.suggestion-reason {
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
  margin: 0 0 0.5rem;
  font-style: italic;
}

.btn-apply {
  display: inline-flex;
  align-items: center;
  padding: 0.375rem 0.75rem;
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--color-primary);
  background: transparent;
  border: 1px solid var(--color-primary);
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.15s;
}

.btn-apply:hover {
  background: var(--color-primary);
  color: white;
}

/* Explanation Section */
.explain-section {
  flex: 1;
  overflow-y: auto;
}

.explanation-content {
  padding: 0.5rem 0;
}

.explanation-title {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--color-text);
  margin: 0 0 0.5rem;
}

.explanation-summary {
  font-size: 0.8125rem;
  line-height: 1.6;
  color: var(--color-text);
  margin: 0 0 1rem;
}

.step-details {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.step-detail {
  padding: 0.5rem;
  background: var(--color-background);
  border-radius: 6px;
}

.detail-name {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--color-primary);
}

.detail-explanation {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  margin: 0.25rem 0 0;
}
</style>
