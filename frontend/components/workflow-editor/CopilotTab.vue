<script setup lang="ts">
import { marked } from 'marked'
import type { AgentStreamState } from '~/composables/useCopilot'
import { useCopilotDraft, type DraftChange } from '~/composables/useCopilotDraft'
import CopilotProposalCard, { type Proposal, type ProposalChange } from './CopilotProposalCard.vue'

// Configure marked for safe rendering
marked.setOptions({
  breaks: true, // Convert \n to <br>
  gfm: true, // GitHub Flavored Markdown
})

// Render markdown to HTML
function renderMarkdown(content: string): string {
  try {
    return marked.parse(content) as string
  } catch {
    return content
  }
}

const props = defineProps<{
  workflowId: string
}>()

const emit = defineEmits<{
  'changes:applied': [changes: ProposalChange[]]
  'changes:preview': []
}>()

const { t } = useI18n()
const copilot = useCopilot()
const toast = useToast()
const copilotDraft = useCopilotDraft()

// Tool execution type from extracted_data
interface ToolExecution {
  tool_name: string
  arguments: Record<string, unknown>
  result: Record<string, unknown>
  is_error: boolean
  timestamp: string
}

interface ChatMessage {
  role: 'user' | 'assistant' | 'system'
  content: string
  toolExecutions?: ToolExecution[]
  proposal?: Proposal
  messageId?: string // For tracking proposal status
}

// State
const isLoading = ref(false)
const isLoadingSession = ref(false)
const chatMessage = ref('')
const chatHistory = ref<ChatMessage[]>([])

// Scroll tracking
const chatMessagesRef = ref<HTMLElement | null>(null)
const isUserScrolledUp = ref(false)

// Agent state
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

// Proposal status storage (persisted via useCopilotDraft)
const proposalStatuses = ref<Map<string, 'pending' | 'applied' | 'discarded'>>(new Map())

// Load proposal statuses from localStorage
function loadProposalStatuses() {
  try {
    const stored = localStorage.getItem(`copilot-proposal-statuses-${props.workflowId}`)
    if (stored) {
      const parsed = JSON.parse(stored) as Record<string, 'pending' | 'applied' | 'discarded'>
      proposalStatuses.value = new Map(Object.entries(parsed))
    }
  } catch (e) {
    console.warn('Failed to load proposal statuses:', e)
  }
}

// Save proposal statuses to localStorage
function saveProposalStatuses() {
  try {
    const obj = Object.fromEntries(proposalStatuses.value)
    localStorage.setItem(`copilot-proposal-statuses-${props.workflowId}`, JSON.stringify(obj))
  } catch (e) {
    console.warn('Failed to save proposal statuses:', e)
  }
}

// Get proposal with local status applied
function getProposalWithStatus(proposal: Proposal): Proposal {
  const localStatus = proposalStatuses.value.get(proposal.id)
  if (localStatus && localStatus !== proposal.status) {
    return { ...proposal, status: localStatus }
  }
  return proposal
}

// Load active session from DB on mount
async function loadActiveSession() {
  if (isLoadingSession.value) return
  isLoadingSession.value = true

  // Load proposal statuses first
  loadProposalStatuses()

  try {
    const result = await copilot.getActiveAgentSession(props.workflowId)
    if (result.session) {
      agentSessionId.value = result.session.id
      // Convert messages to chat history format with extracted_data
      chatHistory.value = result.session.messages.map((msg, idx) => {
        const chatMsg: ChatMessage = {
          role: msg.role,
          content: msg.content,
          messageId: msg.id || `msg-${idx}`,
        }
        // Extract tool executions and proposal from extracted_data if present
        if (msg.extracted_data && typeof msg.extracted_data === 'object') {
          const data = msg.extracted_data as {
            tool_executions?: ToolExecution[]
            proposal?: Proposal
          }
          if (data.tool_executions && Array.isArray(data.tool_executions)) {
            chatMsg.toolExecutions = data.tool_executions
          }
          if (data.proposal) {
            // Apply local status if available
            chatMsg.proposal = getProposalWithStatus(data.proposal)
          }
        }
        return chatMsg
      })
    }
  } catch (e) {
    console.warn('Failed to load active session from DB:', e)
  } finally {
    isLoadingSession.value = false
    // Scroll to bottom after session is loaded
    nextTick(() => scrollToBottom(false))
  }
}

// ==========================================================================
// Auto-scroll functionality
// ==========================================================================

// Check if user is near bottom (within threshold)
function isNearBottom(threshold = 100): boolean {
  const el = chatMessagesRef.value
  if (!el) return true
  return el.scrollHeight - el.scrollTop - el.clientHeight < threshold
}

// Scroll to bottom smoothly
function scrollToBottom(smooth = true) {
  const el = chatMessagesRef.value
  if (!el) return
  el.scrollTo({
    top: el.scrollHeight,
    behavior: smooth ? 'smooth' : 'instant',
  })
}

// Handle scroll event to track user position
function handleChatScroll() {
  isUserScrolledUp.value = !isNearBottom()
}

// Auto-scroll when chat history changes (new messages)
watch(
  () => chatHistory.value.length,
  () => {
    if (!isUserScrolledUp.value) {
      nextTick(() => scrollToBottom())
    }
  }
)

// Load on mount
onMounted(() => {
  loadActiveSession()
  // Scroll to bottom after session is loaded
  nextTick(() => scrollToBottom(false))
})

// Auto-scroll during streaming (partial response, thinking, tool steps)
watch(
  [
    () => agentStreamState.value.partialResponse,
    () => agentStreamState.value.currentThinking,
    () => agentStreamState.value.toolSteps.length,
  ],
  () => {
    if (!isUserScrolledUp.value && agentStreamState.value.isStreaming) {
      nextTick(() => scrollToBottom(false)) // instant scroll during streaming
    }
  }
)

// ==========================================================================
// Agent Functions (autonomous tool-calling agent with SSE streaming)
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

// Convert tool call to draft change
function toolCallToDraftChange(tool: string, args: Record<string, unknown>): DraftChange | null {
  switch (tool) {
    case 'create_step':
      return {
        type: 'step:create',
        tempId: `temp-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`,
        stepType: (args.step_type || args.type) as string as import('~/types/api').StepType,
        name: (args.name || args.step_name || 'New Step') as string,
        config: (args.config || {}) as object,
        position: {
          x: (args.position_x ?? args.x ?? 200) as number,
          y: (args.position_y ?? args.y ?? 200) as number,
        },
      }
    case 'update_step':
      return {
        type: 'step:update',
        stepId: (args.step_id || args.id) as string,
        patch: args as Partial<import('~/types/api').Step>,
      }
    case 'delete_step':
      return {
        type: 'step:delete',
        stepId: (args.step_id || args.id) as string,
      }
    case 'create_edge':
      return {
        type: 'edge:create',
        sourceId: (args.source_step_id || args.source_id || args.from) as string,
        targetId: (args.target_step_id || args.target_id || args.to) as string,
        sourcePort: args.source_port as string | undefined,
        targetPort: args.target_port as string | undefined,
      }
    case 'delete_edge':
      return {
        type: 'edge:delete',
        edgeId: (args.edge_id || args.id) as string,
      }
    default:
      return null
  }
}

// Send message (with streaming)
async function sendAgentMessage() {
  if (!chatMessage.value.trim() || agentStreamState.value.isStreaming) return

  const message = chatMessage.value.trim()
  chatMessage.value = ''

  // Reset scroll state when user sends a message (resume auto-scroll)
  isUserScrolledUp.value = false

  // Add user message to chat
  chatHistory.value.push({ role: 'user', content: message })

  // Reset agent state for new message
  resetAgentState()
  agentStreamState.value.isStreaming = true

  // Start draft for collecting changes
  copilotDraft.startDraft(message)

  try {
    // Start new session if needed (this now returns immediately)
    if (!agentSessionId.value) {
      const response = await copilot.startAgentSession(props.workflowId, message, 'create')
      agentSessionId.value = response.session_id
      // Session created, now process the initial message via SSE below
    }

    // Stream message with SSE (used for both initial and subsequent messages)
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

          // Add tool call to draft
          const draftChange = toolCallToDraftChange(tool, args)
          if (draftChange) {
            copilotDraft.addToDraft(draftChange)
          }
        },
        onToolResult: (tool, result, isError) => {
          const toolStep = agentStreamState.value.toolSteps.find(s => s.tool === tool && s.status === 'calling')
          if (toolStep) {
            toolStep.status = isError ? 'error' : 'success'
            toolStep.result = result
            if (isError) {
              if (typeof result === 'object' && result !== null) {
                const errorObj = result as Record<string, unknown>
                toolStep.error = errorObj.error ? String(errorObj.error) : JSON.stringify(result)
              } else {
                toolStep.error = String(result)
              }
            }
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

          // Finalize draft to get proposal data
          const { needsPreview } = copilotDraft.finalizeDraft()

          // Build proposal from draft if available
          let proposal: Proposal | undefined
          const draft = copilotDraft.getDraft()
          if (draft && draft.changes.length > 0) {
            proposal = {
              id: draft.id,
              status: 'pending',
              changes: draft.changes.map(draftChangeToProposalChange),
            }
          }

          // Add assistant response to chat with proposal (inline display)
          const messageId = `msg-${Date.now()}`
          chatHistory.value.push({
            role: 'assistant',
            content: response,
            proposal,
            messageId,
          })

          // Auto-apply small changes (no step creates/deletes)
          if (!needsPreview && copilotDraft.changeSummary.value.total > 0) {
            handleApplyProposal(proposal?.id || '')
            toast.info(t('copilot.preview.autoApplied'))
          }

          // Show completion info
          if (toolsUsed && toolsUsed.length > 0) {
            console.log(`Agent completed in ${iterations ?? '?'} iterations, used tools: ${toolsUsed.join(', ')}`)
          }
        },
        onError: (error) => {
          agentStreamState.value.isStreaming = false
          agentStreamState.value.error = error
          agentStreamState.value.currentThinking = ''
          currentCancelFn.value = null

          // Discard draft on error
          copilotDraft.discardDraft()

          toast.error(t('copilot.errors.agentFailed'))
        },
      }
    )

    currentCancelFn.value = cancel
  } catch (error) {
    agentStreamState.value.isStreaming = false
    copilotDraft.discardDraft()

    // Convert error to user-friendly message
    const errorMessage = error instanceof Error ? error.message : String(error)
    if (errorMessage.includes('Failed to fetch') || errorMessage.includes('NetworkError')) {
      agentStreamState.value.error = t('copilot.errors.networkError')
    } else if (errorMessage.includes('timeout') || errorMessage.includes('Timeout')) {
      agentStreamState.value.error = t('copilot.errors.timeout')
    } else {
      agentStreamState.value.error = errorMessage
    }
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

// Handle send
async function handleSend() {
  await sendAgentMessage()
}

// Computed for showing agent streaming UI
const showAgentStreaming = computed(() => {
  return agentStreamState.value.isStreaming ||
    agentStreamState.value.toolSteps.length > 0 ||
    agentStreamState.value.currentThinking ||
    agentStreamState.value.error
})

// Convert DraftChange to ProposalChange format
function draftChangeToProposalChange(change: DraftChange): ProposalChange {
  switch (change.type) {
    case 'step:create':
      return {
        type: 'step:create',
        temp_id: change.tempId,
        name: change.name,
        step_type: change.stepType,
        config: change.config as Record<string, unknown>,
        position: change.position,
      }
    case 'step:update':
      return {
        type: 'step:update',
        step_id: change.stepId,
        patch: change.patch as Record<string, unknown>,
      }
    case 'step:delete':
      return {
        type: 'step:delete',
        step_id: change.stepId,
      }
    case 'edge:create':
      return {
        type: 'edge:create',
        source_id: change.sourceId,
        target_id: change.targetId,
        source_port: change.sourcePort,
        target_port: change.targetPort,
      }
    case 'edge:delete':
      return {
        type: 'edge:delete',
        edge_id: change.edgeId,
      }
  }
}

// Update proposal status in chat history
function updateProposalStatus(proposalId: string, status: 'applied' | 'discarded') {
  for (const msg of chatHistory.value) {
    if (msg.proposal && msg.proposal.id === proposalId) {
      msg.proposal = { ...msg.proposal, status }
      break
    }
  }
  // Persist status
  proposalStatuses.value.set(proposalId, status)
  saveProposalStatuses()
}

// Get proposal changes by ID from chat history
function getProposalChanges(proposalId: string): ProposalChange[] {
  for (const msg of chatHistory.value) {
    if (msg.proposal && msg.proposal.id === proposalId) {
      return msg.proposal.changes
    }
  }
  return []
}

// Handle apply proposal (from inline card)
function handleApplyProposal(proposalId: string) {
  // Get the proposal changes before updating status
  const changes = getProposalChanges(proposalId)

  updateProposalStatus(proposalId, 'applied')

  // Emit event with changes - parent component handles actual command execution
  if (changes.length > 0) {
    emit('changes:applied', changes)
  }

  copilotDraft.discardDraft()
}

// Handle discard proposal (from inline card)
function handleDiscardProposal(proposalId: string) {
  updateProposalStatus(proposalId, 'discarded')
  copilotDraft.discardDraft()
}

// Handle modify proposal (from inline card)
async function handleModifyProposal(_proposalId: string, feedback: string) {
  copilotDraft.discardDraft()
  // Send the modification request as a new message
  chatMessage.value = feedback
  await sendAgentMessage()
}
</script>

<template>
  <div class="copilot-tab">
    <!-- Chat Section -->
    <div class="chat-section">
      <div ref="chatMessagesRef" class="chat-messages" @scroll="handleChatScroll">
        <div v-if="chatHistory.length === 0" class="chat-empty">
          <p>{{ t('copilot.agent.welcome') }}</p>
          <p class="chat-hint">{{ t('copilot.agent.hint') }}</p>
        </div>
        <div
          v-for="(msg, idx) in chatHistory"
          :key="idx"
          class="chat-message"
          :class="msg.role"
        >
          <!-- Show tool executions before assistant message content -->
          <div v-if="msg.toolExecutions && msg.toolExecutions.length > 0" class="saved-tool-executions">
            <div
              v-for="(exec, execIdx) in msg.toolExecutions"
              :key="execIdx"
              class="tool-step"
              :class="exec.is_error ? 'error' : 'success'"
            >
              <div class="tool-header">
                <span class="tool-icon">
                  <svg v-if="exec.is_error" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="12" cy="12" r="10"/>
                    <line x1="15" y1="9" x2="9" y2="15"/>
                    <line x1="9" y1="9" x2="15" y2="15"/>
                  </svg>
                  <svg v-else xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="20 6 9 17 4 12"/>
                  </svg>
                </span>
                <span class="tool-name">{{ exec.tool_name }}</span>
                <span class="tool-status">{{ exec.is_error ? t('copilot.toolStatus.error') : t('copilot.toolStatus.success') }}</span>
              </div>
            </div>
          </div>
          <!-- User messages: plain text, Assistant messages: markdown -->
          <div
            v-if="msg.role === 'assistant'"
            class="message-content markdown-content"
            v-html="renderMarkdown(msg.content)"
          />
          <div v-else class="message-content">{{ msg.content }}</div>

          <!-- Inline Proposal Card (Claude Code style) -->
          <CopilotProposalCard
            v-if="msg.proposal"
            :proposal="msg.proposal"
            :message-id="msg.messageId"
            @apply="handleApplyProposal"
            @discard="handleDiscardProposal"
            @modify="handleModifyProposal"
          />
        </div>

        <!-- Agent Streaming UI -->
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
              v-for="toolStep in agentStreamState.toolSteps"
              :key="toolStep.id"
              class="tool-step"
              :class="toolStep.status"
            >
              <div class="tool-header">
                <span class="tool-icon">
                  <svg v-if="toolStep.status === 'calling'" class="spin" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
                  </svg>
                  <svg v-else-if="toolStep.status === 'success'" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="20 6 9 17 4 12"/>
                  </svg>
                  <svg v-else xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="12" cy="12" r="10"/>
                    <line x1="15" y1="9" x2="9" y2="15"/>
                    <line x1="9" y1="9" x2="15" y2="15"/>
                  </svg>
                </span>
                <span class="tool-name">{{ toolStep.tool }}</span>
                <span class="tool-status">{{ t(`copilot.toolStatus.${toolStep.status}`) }}</span>
              </div>
              <div v-if="toolStep.error" class="tool-error">{{ toolStep.error }}</div>
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
        <!-- Reset button -->
        <button
          v-if="agentSessionId && !agentStreamState.isStreaming"
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
          :placeholder="t('copilot.agent.placeholder')"
          :disabled="isLoading || agentStreamState.isStreaming"
          rows="2"
          @keydown.meta.enter="handleSend"
          @keydown.ctrl.enter="handleSend"
        />
        <button
          v-if="!agentStreamState.isStreaming"
          class="chat-send-btn"
          :disabled="!chatMessage.trim() || isLoading || agentStreamState.isStreaming"
          @click="handleSend"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="22" y1="2" x2="11" y2="13"/>
            <polygon points="22 2 15 22 11 13 2 9 22 2"/>
          </svg>
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.copilot-tab {
  display: flex;
  flex-direction: column;
  height: 100%;
}

/* Reset Button */
.reset-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  color: var(--color-text-secondary);
  background: transparent;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
  flex-shrink: 0;
}

.reset-btn:hover {
  color: var(--color-error);
  border-color: var(--color-error);
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
  flex-shrink: 0;
}

.cancel-btn:hover {
  background: var(--color-error);
  color: white;
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

.message-content {
  white-space: pre-wrap;
  word-wrap: break-word;
}

/* Markdown content styles */
.markdown-content {
  white-space: normal;
}

.markdown-content :deep(p) {
  margin: 0 0 0.5em 0;
}

.markdown-content :deep(p:last-child) {
  margin-bottom: 0;
}

.markdown-content :deep(ul),
.markdown-content :deep(ol) {
  margin: 0.5em 0;
  padding-left: 1.5em;
}

.markdown-content :deep(li) {
  margin: 0.25em 0;
}

.markdown-content :deep(code) {
  background: rgba(0, 0, 0, 0.06);
  padding: 0.125em 0.375em;
  border-radius: 4px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.875em;
}

.markdown-content :deep(pre) {
  background: rgba(0, 0, 0, 0.06);
  padding: 0.75em;
  border-radius: 6px;
  overflow-x: auto;
  margin: 0.5em 0;
}

.markdown-content :deep(pre code) {
  background: none;
  padding: 0;
  font-size: 0.8125rem;
}

.markdown-content :deep(strong) {
  font-weight: 600;
}

.markdown-content :deep(em) {
  font-style: italic;
}

.markdown-content :deep(a) {
  color: var(--color-primary);
  text-decoration: none;
}

.markdown-content :deep(a:hover) {
  text-decoration: underline;
}

.markdown-content :deep(blockquote) {
  margin: 0.5em 0;
  padding-left: 1em;
  border-left: 3px solid var(--color-border);
  color: var(--color-text-secondary);
}

.markdown-content :deep(h1),
.markdown-content :deep(h2),
.markdown-content :deep(h3),
.markdown-content :deep(h4) {
  margin: 0.75em 0 0.5em 0;
  font-weight: 600;
  line-height: 1.3;
}

.markdown-content :deep(h1) {
  font-size: 1.25em;
}

.markdown-content :deep(h2) {
  font-size: 1.125em;
}

.markdown-content :deep(h3) {
  font-size: 1em;
}

.markdown-content :deep(hr) {
  border: none;
  border-top: 1px solid var(--color-border);
  margin: 0.75em 0;
}

.markdown-content :deep(table) {
  border-collapse: collapse;
  width: 100%;
  margin: 0.5em 0;
  font-size: 0.875em;
}

.markdown-content :deep(th),
.markdown-content :deep(td) {
  border: 1px solid var(--color-border);
  padding: 0.5em 0.75em;
  text-align: left;
}

.markdown-content :deep(th) {
  background: var(--color-background);
  font-weight: 600;
}

/* Saved tool executions in chat history */
.saved-tool-executions {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  margin-bottom: 0.5rem;
  padding-bottom: 0.5rem;
  border-bottom: 1px dashed var(--color-border);
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
  flex-shrink: 0;
}

.chat-send-btn:hover:not(:disabled) {
  opacity: 0.9;
}

.chat-send-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
