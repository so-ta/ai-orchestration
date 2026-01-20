<script setup lang="ts">
import type { AgentStreamState } from '~/composables/useCopilot'
import { useCopilotDraft, type DraftChange } from '~/composables/useCopilotDraft'
import CopilotPreviewPanel from './CopilotPreviewPanel.vue'

const props = defineProps<{
  workflowId: string
}>()

const emit = defineEmits<{
  'changes:applied': []
  'changes:preview': []
}>()

const { t } = useI18n()
const copilot = useCopilot()
const toast = useToast()
const copilotDraft = useCopilotDraft()

// Storage key for chat history
const CHAT_HISTORY_KEY_PREFIX = 'copilot-chat-history-'
const AGENT_SESSION_KEY_PREFIX = 'copilot-agent-session-'

// State
const isLoading = ref(false)
const chatMessage = ref('')
const chatHistory = ref<Array<{ role: 'user' | 'assistant' | 'system'; content: string }>>([])

// Agent state
const agentSessionId = ref<string | null>(null)

// Load chat history from localStorage on mount
function loadChatHistory() {
  if (typeof window === 'undefined') return
  try {
    const stored = localStorage.getItem(CHAT_HISTORY_KEY_PREFIX + props.workflowId)
    if (stored) {
      chatHistory.value = JSON.parse(stored)
    }
    const storedSession = localStorage.getItem(AGENT_SESSION_KEY_PREFIX + props.workflowId)
    if (storedSession) {
      agentSessionId.value = storedSession
    }
  } catch (e) {
    console.warn('Failed to load chat history from localStorage:', e)
  }
}

// Save chat history to localStorage
function saveChatHistory() {
  if (typeof window === 'undefined') return
  try {
    localStorage.setItem(CHAT_HISTORY_KEY_PREFIX + props.workflowId, JSON.stringify(chatHistory.value))
    if (agentSessionId.value) {
      localStorage.setItem(AGENT_SESSION_KEY_PREFIX + props.workflowId, agentSessionId.value)
    } else {
      localStorage.removeItem(AGENT_SESSION_KEY_PREFIX + props.workflowId)
    }
  } catch (e) {
    console.warn('Failed to save chat history to localStorage:', e)
  }
}

// Watch chat history changes and save
watch(chatHistory, saveChatHistory, { deep: true })
watch(agentSessionId, saveChatHistory)

// Load on mount
onMounted(() => {
  loadChatHistory()
})
const agentStreamState = ref<AgentStreamState>({
  isStreaming: false,
  currentThinking: '',
  toolSteps: [],
  partialResponse: '',
  finalResponse: null,
  error: null,
})
const currentCancelFn = ref<(() => void) | null>(null)

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
            if (isError) toolStep.error = String(result)
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

          // Finalize draft and determine if preview is needed
          const { needsPreview } = copilotDraft.finalizeDraft()

          if (needsPreview) {
            // Show preview panel
            showPreviewPanel.value = true
            emit('changes:preview')
          } else if (copilotDraft.changeSummary.value.total > 0) {
            // Auto-apply small changes
            handleApplyDraft()
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

// Preview panel state
const showPreviewPanel = ref(false)

// Computed for showing agent streaming UI
const showAgentStreaming = computed(() => {
  return agentStreamState.value.isStreaming ||
    agentStreamState.value.toolSteps.length > 0 ||
    agentStreamState.value.currentThinking ||
    agentStreamState.value.error
})

// Handle apply draft
function handleApplyDraft() {
  // Emit event - the parent component will handle actual command execution
  // This is because the parent has access to the project state and API instances
  emit('changes:applied')
  showPreviewPanel.value = false
  copilotDraft.discardDraft() // Clear after apply
}

// Handle discard draft
function handleDiscardDraft() {
  copilotDraft.discardDraft()
  showPreviewPanel.value = false
}

// Handle modify request
async function handleModifyDraft(feedback: string) {
  showPreviewPanel.value = false
  copilotDraft.discardDraft()

  // Send the modification request as a new message
  chatMessage.value = feedback
  await sendAgentMessage()
}
</script>

<template>
  <div class="copilot-tab">
    <!-- Preview Panel (shown when changes need review) -->
    <CopilotPreviewPanel
      v-if="showPreviewPanel && copilotDraft.currentDraft.value"
      :draft="copilotDraft.currentDraft.value"
      @apply="handleApplyDraft"
      @discard="handleDiscardDraft"
      @modify="handleModifyDraft"
    />

    <!-- Chat Section -->
    <div class="chat-section">
      <div class="chat-messages">
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
          <div class="message-content">{{ msg.content }}</div>
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
