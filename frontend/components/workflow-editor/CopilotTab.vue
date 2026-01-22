<script setup lang="ts">
import { nextTick } from 'vue'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import type { AgentStreamState } from '~/composables/useCopilot'
import { useCopilotDraft } from '~/composables/useCopilotDraft'
import { useToolDisplay, useChatScroll, useProposalStatuses, toolCallToDraftChange, draftChangeToProposalChange, isE2EWorkflowTool, toolResultToChatExtension, type ProposalChange, useCopilotProgress, useCopilotActions } from '~/composables/copilot'
import CopilotProposalCard, { type Proposal } from './CopilotProposalCard.vue'
import CopilotProgressCard from './copilot/CopilotProgressCard.vue'
import CopilotInlineAction from './copilot/CopilotInlineAction.vue'
import CopilotTestResultCard from './copilot/CopilotTestResultCard.vue'
import CopilotWelcomePanel from './CopilotWelcomePanel.vue'
import type { WorkflowProgressStatus, InlineAction, WorkflowTestResult, InlineActionResult, CopilotChatExtension } from './copilot/types'

// Configure marked for safe rendering
marked.setOptions({
  breaks: true,
  gfm: true,
})

function renderMarkdown(content: string): string {
  try {
    const raw = marked.parse(content) as string
    return DOMPurify.sanitize(raw)
  } catch {
    return DOMPurify.sanitize(content)
  }
}

const props = defineProps<{
  workflowId: string
  stepCount?: number
}>()

const emit = defineEmits<{
  'changes:applied': [changes: ProposalChange[]]
  'changes:preview': []
  'workflow:updated': []
}>()

const { t } = useI18n()
const copilot = useCopilot()
const toast = useToast()
const copilotDraft = useCopilotDraft()

// Use composables
const { getToolLabel, getToolDescription, getToolResultSummary } = useToolDisplay()
const { chatMessagesRef, isUserScrolledUp, handleChatScroll, scrollToBottom, autoScrollIfNeeded, resetScrollState } = useChatScroll()
const proposalStatusManager = useProposalStatuses(props.workflowId)

// E2E workflow composables
const copilotProgress = useCopilotProgress({
  workflowId: props.workflowId,
  onPhaseChange: (phase) => {
    console.log('Copilot phase changed:', phase)
  },
})

const copilotActions = useCopilotActions({
  workflowId: props.workflowId,
  onProgress: () => {
    copilotProgress.fetchProgress()
  },
  onTestStart: () => {
    console.log('Test started')
  },
  onTestComplete: (result) => {
    console.log('Test completed:', result.status)
    // Add test result to chat
    if (result) {
      chatHistory.value.push({
        role: 'assistant',
        content: result.status === 'success'
          ? t('copilot.test.success')
          : t('copilot.test.failed'),
        testResult: result,
        messageId: `test-${Date.now()}`,
      })
    }
  },
  onPublishComplete: () => {
    emit('workflow:updated')
  },
})

// Tool execution type
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
  messageId?: string
  // E2E workflow extensions
  progressCard?: WorkflowProgressStatus
  inlineAction?: InlineAction
  testResult?: WorkflowTestResult
}

// State
const isLoading = ref(false)
const isLoadingSession = ref(false)
const chatMessage = ref('')
const chatHistory = ref<ChatMessage[]>([])

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

// Get proposal with local status applied
function getProposalWithStatus(proposal: Proposal): Proposal {
  const localStatus = proposalStatusManager.getStatus(proposal.id)
  if (localStatus && localStatus !== proposal.status) {
    return { ...proposal, status: localStatus }
  }
  return proposal
}

// Load active session from DB on mount
async function loadActiveSession() {
  if (isLoadingSession.value) return
  isLoadingSession.value = true
  proposalStatusManager.load()

  try {
    const result = await copilot.getActiveAgentSession(props.workflowId)
    if (result.session) {
      agentSessionId.value = result.session.id
      chatHistory.value = result.session.messages.map((msg, idx) => {
        const chatMsg: ChatMessage = {
          role: msg.role,
          content: msg.content,
          messageId: msg.id || `msg-${idx}`,
        }
        if (msg.extracted_data && typeof msg.extracted_data === 'object') {
          const data = msg.extracted_data as {
            tool_executions?: ToolExecution[]
            proposal?: Proposal
          }
          if (data.tool_executions && Array.isArray(data.tool_executions)) {
            chatMsg.toolExecutions = data.tool_executions
          }
          if (data.proposal) {
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
    nextTick(() => scrollToBottom(false))
  }
}

// Listen for external prompt events
function handleExternalPrompt(event: Event) {
  const customEvent = event as CustomEvent<{ prompt: string }>
  if (customEvent.detail?.prompt) {
    chatMessage.value = customEvent.detail.prompt
    nextTick(() => sendAgentMessage())
  }
}

onMounted(() => {
  loadActiveSession()
  nextTick(() => scrollToBottom(false))
  window.addEventListener('copilot-send-prompt', handleExternalPrompt)
})

onUnmounted(() => {
  window.removeEventListener('copilot-send-prompt', handleExternalPrompt)
})

// Auto-scroll when chat history changes
watch(() => chatHistory.value.length, () => autoScrollIfNeeded())

// Auto-scroll during streaming
watch(
  [() => agentStreamState.value.partialResponse, () => agentStreamState.value.currentThinking, () => agentStreamState.value.toolSteps.length],
  () => {
    if (!isUserScrolledUp.value && agentStreamState.value.isStreaming) {
      nextTick(() => scrollToBottom(false))
    }
  }
)

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

// Send message (with streaming)
async function sendAgentMessage() {
  if (!chatMessage.value.trim() || agentStreamState.value.isStreaming) return

  const message = chatMessage.value.trim()
  chatMessage.value = ''
  resetScrollState()
  chatHistory.value.push({ role: 'user', content: message })
  resetAgentState()
  agentStreamState.value.isStreaming = true
  copilotDraft.startDraft(message)

  try {
    if (!agentSessionId.value) {
      const response = await copilot.startAgentSession(props.workflowId, message, 'create')
      agentSessionId.value = response.session_id
    }

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
          agentStreamState.value.toolSteps.push({ id: stepId, tool, status: 'calling', arguments: args })
          agentStreamState.value.currentThinking = ''

          const draftChange = toolCallToDraftChange(tool, args)
          if (draftChange) {
            if (Array.isArray(draftChange)) {
              for (const change of draftChange) copilotDraft.addToDraft(change)
            } else {
              copilotDraft.addToDraft(draftChange)
            }
          }
        },
        onToolResult: (tool, result, isError) => {
          const toolStep = agentStreamState.value.toolSteps.find(s => s.tool === tool && s.status === 'calling')
          if (toolStep) {
            toolStep.status = isError ? 'error' : 'success'
            toolStep.result = result
            if (isError) {
              toolStep.error = typeof result === 'object' && result !== null
                ? (result as Record<string, unknown>).error ? String((result as Record<string, unknown>).error) : JSON.stringify(result)
                : String(result)
            }

            // Check for E2E workflow tool and create chat extension
            if (!isError && isE2EWorkflowTool(tool)) {
              const extension = toolResultToChatExtension(tool, toolStep.arguments as Record<string, unknown>, result as Record<string, unknown>)
              if (extension) {
                // Store extension for later inclusion in the final message
                (toolStep as { chatExtension?: CopilotChatExtension }).chatExtension = extension
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

          const { needsPreview } = copilotDraft.finalizeDraft()
          let proposal: Proposal | undefined
          const draft = copilotDraft.getDraft()
          if (draft && draft.changes.length > 0) {
            proposal = {
              id: draft.id,
              status: 'pending',
              changes: draft.changes.map(draftChangeToProposalChange),
            }
          }

          // Collect chat extensions from E2E workflow tools
          let progressCard: WorkflowProgressStatus | undefined
          let inlineAction: InlineAction | undefined
          let testResult: WorkflowTestResult | undefined

          for (const toolStep of agentStreamState.value.toolSteps) {
            const ext = (toolStep as { chatExtension?: CopilotChatExtension }).chatExtension
            if (ext) {
              if (ext.progressCard) progressCard = ext.progressCard
              if (ext.inlineAction) inlineAction = ext.inlineAction
              if (ext.testResult) testResult = ext.testResult
            }
          }

          const messageId = `msg-${Date.now()}`
          chatHistory.value.push({
            role: 'assistant',
            content: response,
            proposal,
            messageId,
            progressCard,
            inlineAction,
            testResult,
          })

          if (!needsPreview && copilotDraft.changeSummary.value.total > 0) {
            handleApplyProposal(proposal?.id || '')
            toast.info(t('copilot.preview.autoApplied'))
          }

          if (toolsUsed && toolsUsed.length > 0) {
            console.log(`Agent completed in ${iterations ?? '?'} iterations, used tools: ${toolsUsed.join(', ')}`)
            emit('workflow:updated')
          }
        },
        onError: (error) => {
          agentStreamState.value.isStreaming = false
          agentStreamState.value.error = error
          agentStreamState.value.currentThinking = ''
          currentCancelFn.value = null
          copilotDraft.discardDraft()
          toast.error(t('copilot.errors.agentFailed'))
        },
      }
    )
    currentCancelFn.value = cancel
  } catch (error) {
    agentStreamState.value.isStreaming = false
    copilotDraft.discardDraft()
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

function cancelAgentStream() {
  if (currentCancelFn.value) {
    currentCancelFn.value()
    currentCancelFn.value = null
    agentStreamState.value.isStreaming = false
    agentStreamState.value.currentThinking = ''
    toast.info(t('copilot.agent.cancelled'))
  }
}

function resetAgentSession() {
  agentSessionId.value = null
  chatHistory.value = []
  resetAgentState()
  toast.info(t('copilot.agent.resetSession'))
}

async function handleSend() {
  await sendAgentMessage()
}

// Handle structured prompt submit from empty state
async function handleStructuredPromptSubmit(prompt: string) {
  chatMessage.value = prompt
  await sendAgentMessage()
}

const showAgentStreaming = computed(() => {
  return agentStreamState.value.isStreaming ||
    agentStreamState.value.toolSteps.length > 0 ||
    agentStreamState.value.currentThinking ||
    agentStreamState.value.error
})

// Show welcome panel when chat is empty OR workflow has 1 or fewer blocks
const showWelcomePanel = computed(() => {
  const hasNoChat = chatHistory.value.length === 0
  const hasMinimalWorkflow = (props.stepCount ?? 0) <= 1
  return hasNoChat || hasMinimalWorkflow
})

function updateProposalStatus(proposalId: string, status: 'applied' | 'discarded') {
  for (const msg of chatHistory.value) {
    if (msg.proposal && msg.proposal.id === proposalId) {
      msg.proposal = { ...msg.proposal, status }
      break
    }
  }
  proposalStatusManager.setStatus(proposalId, status)
}

function getProposalChanges(proposalId: string): ProposalChange[] {
  for (const msg of chatHistory.value) {
    if (msg.proposal && msg.proposal.id === proposalId) {
      return msg.proposal.changes
    }
  }
  return []
}

function handleApplyProposal(proposalId: string) {
  const changes = getProposalChanges(proposalId)
  updateProposalStatus(proposalId, 'applied')
  if (changes.length > 0) emit('changes:applied', changes)
  copilotDraft.discardDraft()
}

function handleDiscardProposal(proposalId: string) {
  updateProposalStatus(proposalId, 'discarded')
  copilotDraft.discardDraft()
}

async function handleModifyProposal(_proposalId: string, feedback: string) {
  copilotDraft.discardDraft()
  chatMessage.value = feedback
  await sendAgentMessage()
}

// E2E workflow action handlers
async function handleInlineActionResult(actionId: string, result: InlineActionResult) {
  const success = await copilotActions.handleActionResult(actionId, result)
  if (success) {
    emit('workflow:updated')
  }
}

function handleProgressActionClick(itemId: string, action: InlineAction) {
  // When user clicks action button in progress card, add an inline action message
  console.log('Progress action clicked:', itemId)
  chatHistory.value.push({
    role: 'assistant',
    content: action.title || '',
    inlineAction: action,
    messageId: `action-${Date.now()}`,
  })
  nextTick(() => scrollToBottom(true))
}

function handleTestRerun() {
  copilotActions.runTest()
}

// Template prompt mapping for quick start templates
const templatePrompts: Record<string, string> = {
  'webhook-notification': 'Webhookを受信してSlackに通知するワークフローを作成してください',
  'scheduled-report': '定期的にデータを収集してレポートを生成するワークフローを作成してください',
  'api-integration': '複数のAPIを連携してデータを同期するワークフローを作成してください',
  'llm-assistant': 'LLMを使った問い合わせ対応ワークフローを作成してください',
}

// Handle template selection from CopilotWelcomePanel
async function handleTemplateSelect(templateId: string) {
  const prompt = templatePrompts[templateId] || `${templateId}のワークフローを作成してください`
  chatMessage.value = prompt
  await sendAgentMessage()
}
</script>

<template>
  <div class="copilot-tab">
    <div class="chat-section">
      <div ref="chatMessagesRef" class="chat-messages" @scroll="handleChatScroll">
        <!-- Welcome Panel: Show when chat empty OR workflow has 1 or fewer blocks -->
        <div v-if="showWelcomePanel" class="chat-empty">
          <CopilotWelcomePanel
            compact
            @submit="handleStructuredPromptSubmit"
            @select-template="handleTemplateSelect"
          />
        </div>
        <div v-for="(msg, idx) in chatHistory" :key="idx" class="chat-message" :class="msg.role">
          <div v-if="msg.toolExecutions && msg.toolExecutions.length > 0" class="saved-tool-executions">
            <div v-for="(exec, execIdx) in msg.toolExecutions" :key="execIdx" class="tool-step" :class="exec.is_error ? 'error' : 'success'">
              <div class="tool-header">
                <span class="tool-icon">
                  <svg v-if="exec.is_error" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/></svg>
                  <svg v-else xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="20 6 9 17 4 12"/></svg>
                </span>
                <span class="tool-name">{{ getToolLabel(exec.tool_name) }}</span>
                <span v-if="getToolDescription(exec.tool_name, exec.arguments)" class="tool-desc">{{ getToolDescription(exec.tool_name, exec.arguments) }}</span>
              </div>
              <div v-if="getToolResultSummary(exec.tool_name, exec.result, exec.is_error)" class="tool-result-summary">{{ getToolResultSummary(exec.tool_name, exec.result, exec.is_error) }}</div>
            </div>
          </div>
          <!-- eslint-disable-next-line vue/no-v-html -- sanitize済みMarkdownのレンダリング -->
          <div v-if="msg.role === 'assistant' && msg.content" class="message-content markdown-content" v-html="renderMarkdown(msg.content)"/>
          <div v-else-if="msg.role !== 'assistant' && msg.content" class="message-content">{{ msg.content }}</div>
          <!-- E2E Workflow Components -->
          <CopilotProgressCard v-if="msg.progressCard" :progress="msg.progressCard" @item-action="handleProgressActionClick"/>
          <CopilotInlineAction v-if="msg.inlineAction" :action="msg.inlineAction" @result="(result: InlineActionResult) => handleInlineActionResult(msg.inlineAction!.id, result)"/>
          <CopilotTestResultCard v-if="msg.testResult" :result="msg.testResult" @retest="handleTestRerun" @view-details="() => {}"/>
          <!-- Proposal Card -->
          <CopilotProposalCard v-if="msg.proposal" :proposal="msg.proposal" :message-id="msg.messageId" @apply="handleApplyProposal" @discard="handleDiscardProposal" @modify="handleModifyProposal"/>
        </div>

        <div v-if="showAgentStreaming" class="agent-streaming">
          <div v-if="agentStreamState.currentThinking" class="thinking-bubble">
            <div class="thinking-icon"><svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/></svg></div>
            <span class="thinking-text">{{ agentStreamState.currentThinking }}</span>
          </div>

          <div v-if="agentStreamState.toolSteps.length > 0" class="tool-steps">
            <div v-for="toolStep in agentStreamState.toolSteps" :key="toolStep.id" class="tool-step" :class="toolStep.status">
              <div class="tool-header">
                <span class="tool-icon">
                  <svg v-if="toolStep.status === 'calling'" class="spin" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 12a9 9 0 1 1-6.219-8.56"/></svg>
                  <svg v-else-if="toolStep.status === 'success'" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="20 6 9 17 4 12"/></svg>
                  <svg v-else xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/></svg>
                </span>
                <span class="tool-name">{{ getToolLabel(toolStep.tool) }}</span>
                <span v-if="getToolDescription(toolStep.tool, toolStep.arguments)" class="tool-desc">{{ getToolDescription(toolStep.tool, toolStep.arguments) }}</span>
                <span v-if="toolStep.status === 'calling'" class="tool-status calling">{{ t('copilot.toolStatus.calling') }}</span>
              </div>
              <div v-if="toolStep.status === 'success' && getToolResultSummary(toolStep.tool, toolStep.result, false)" class="tool-result-summary">{{ getToolResultSummary(toolStep.tool, toolStep.result, false) }}</div>
              <div v-if="toolStep.error" class="tool-error">{{ toolStep.error }}</div>
            </div>
          </div>

          <div v-if="agentStreamState.partialResponse" class="partial-response">{{ agentStreamState.partialResponse }}<span class="cursor-blink">▊</span></div>
          <div v-if="agentStreamState.error" class="agent-error"><svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg><span>{{ agentStreamState.error }}</span></div>
        </div>
      </div>

      <div class="chat-input-container">
        <button v-if="agentSessionId && !agentStreamState.isStreaming" class="reset-btn" :title="t('copilot.agent.resetSession')" @click="resetAgentSession">
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8"/><path d="M21 3v5h-5"/><path d="M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16"/><path d="M3 21v-5h5"/></svg>
        </button>
        <button v-if="agentStreamState.isStreaming" class="cancel-btn" @click="cancelAgentStream">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"/></svg>
          {{ t('copilot.agent.cancel') }}
        </button>
        <textarea v-model="chatMessage" class="chat-input" :placeholder="t('copilot.agent.placeholder')" :disabled="isLoading || agentStreamState.isStreaming" rows="2" @keydown.meta.enter="handleSend" @keydown.ctrl.enter="handleSend"/>
        <button v-if="!agentStreamState.isStreaming" class="chat-send-btn" :disabled="!chatMessage.trim() || isLoading || agentStreamState.isStreaming" @click="handleSend">
          <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg>
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.copilot-tab { display: flex; flex-direction: column; height: 100%; }
.reset-btn { display: flex; align-items: center; justify-content: center; width: 32px; height: 32px; color: var(--color-text-secondary); background: transparent; border: 1px solid var(--color-border); border-radius: 6px; cursor: pointer; transition: all 0.15s; flex-shrink: 0; }
.reset-btn:hover { color: var(--color-error); border-color: var(--color-error); background: rgba(239, 68, 68, 0.1); }
.agent-streaming { display: flex; flex-direction: column; gap: 0.5rem; padding: 0.75rem; background: var(--color-background); border-radius: 8px; margin-top: 0.5rem; }
.thinking-bubble { display: flex; align-items: flex-start; gap: 0.5rem; padding: 0.625rem 0.75rem; background: linear-gradient(135deg, rgba(99, 102, 241, 0.1) 0%, rgba(139, 92, 246, 0.1) 100%); border: 1px dashed rgba(99, 102, 241, 0.3); border-radius: 8px; color: var(--color-primary); font-size: 0.8125rem; font-style: italic; }
.thinking-icon { display: flex; animation: pulse 2s ease-in-out infinite; }
@keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.5; } }
.thinking-text { flex: 1; line-height: 1.5; }
.tool-steps { display: flex; flex-direction: column; gap: 0.375rem; }
.tool-step { display: flex; flex-direction: column; padding: 0.5rem 0.75rem; background: var(--color-surface); border-radius: 6px; border-left: 3px solid var(--color-border); }
.tool-step.calling { border-left-color: var(--color-warning); }
.tool-step.success { border-left-color: var(--color-success); }
.tool-step.error { border-left-color: var(--color-error); }
.tool-header { display: flex; align-items: center; gap: 0.5rem; }
.tool-icon { display: flex; align-items: center; justify-content: center; width: 20px; height: 20px; }
.tool-step.calling .tool-icon { color: var(--color-warning); }
.tool-step.success .tool-icon { color: var(--color-success); }
.tool-step.error .tool-icon { color: var(--color-error); }
.tool-icon .spin { animation: spin 1s linear infinite; }
.tool-name { font-size: 0.75rem; font-weight: 600; color: var(--color-text); font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace; }
.tool-status { margin-left: auto; font-size: 0.6875rem; font-weight: 500; text-transform: uppercase; }
.tool-step.calling .tool-status { color: var(--color-warning); }
.tool-error { margin-top: 0.25rem; font-size: 0.75rem; color: var(--color-error); }
.tool-desc { font-size: 0.75rem; color: var(--color-text-secondary); font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace; max-width: 200px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.tool-result-summary { margin-top: 0.25rem; padding-left: 1.75rem; font-size: 0.6875rem; color: var(--color-text-tertiary); }
.tool-step.success .tool-result-summary { color: var(--color-success); }
.partial-response { padding: 0.625rem 0.75rem; background: var(--color-surface); border: 1px solid var(--color-border); border-radius: 8px; font-size: 0.8125rem; line-height: 1.6; color: var(--color-text); white-space: pre-wrap; }
.cursor-blink { animation: blink 1s step-end infinite; color: var(--color-primary); }
@keyframes blink { 0%, 100% { opacity: 1; } 50% { opacity: 0; } }
.agent-error { display: flex; align-items: center; gap: 0.5rem; padding: 0.625rem 0.75rem; background: rgba(239, 68, 68, 0.1); border: 1px solid var(--color-error); border-radius: 8px; font-size: 0.8125rem; color: var(--color-error); }
.cancel-btn { display: flex; align-items: center; gap: 0.375rem; padding: 0.5rem 0.75rem; font-size: 0.75rem; font-weight: 500; color: var(--color-error); background: rgba(239, 68, 68, 0.1); border: 1px solid var(--color-error); border-radius: 6px; cursor: pointer; transition: all 0.15s; flex-shrink: 0; }
.cancel-btn:hover { background: var(--color-error); color: white; }
.chat-section { display: flex; flex-direction: column; flex: 1; min-height: 0; }
.chat-messages { flex: 1; overflow-y: auto; padding: 0.5rem 0; }
.chat-empty { display: flex; flex-direction: column; padding: 0.5rem 0; }
.chat-message { padding: 0.5rem 0.75rem; margin-bottom: 0.5rem; border-radius: 8px; font-size: 0.8125rem; line-height: 1.5; }
.chat-message.user { background: var(--color-primary); color: white; margin-left: 1rem; }
.chat-message.assistant { background: var(--color-background); border: 1px solid var(--color-border); margin-right: 1rem; }
.chat-message.system { background: rgba(59, 130, 246, 0.1); border: 1px dashed var(--color-primary); color: var(--color-primary); text-align: center; font-size: 0.75rem; font-style: italic; }
.message-content { white-space: pre-wrap; word-wrap: break-word; }
.markdown-content { white-space: normal; }
.markdown-content :deep(p) { margin: 0 0 0.5em 0; }
.markdown-content :deep(p:last-child) { margin-bottom: 0; }
.markdown-content :deep(ul), .markdown-content :deep(ol) { margin: 0.5em 0; padding-left: 1.5em; }
.markdown-content :deep(li) { margin: 0.25em 0; }
.markdown-content :deep(code) { background: rgba(0, 0, 0, 0.06); padding: 0.125em 0.375em; border-radius: 4px; font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace; font-size: 0.875em; }
.markdown-content :deep(pre) { background: rgba(0, 0, 0, 0.06); padding: 0.75em; border-radius: 6px; overflow-x: auto; margin: 0.5em 0; }
.markdown-content :deep(pre code) { background: none; padding: 0; font-size: 0.8125rem; }
.markdown-content :deep(strong) { font-weight: 600; }
.markdown-content :deep(em) { font-style: italic; }
.markdown-content :deep(a) { color: var(--color-primary); text-decoration: none; }
.markdown-content :deep(a:hover) { text-decoration: underline; }
.markdown-content :deep(blockquote) { margin: 0.5em 0; padding-left: 1em; border-left: 3px solid var(--color-border); color: var(--color-text-secondary); }
.markdown-content :deep(h1), .markdown-content :deep(h2), .markdown-content :deep(h3), .markdown-content :deep(h4) { margin: 0.75em 0 0.5em 0; font-weight: 600; line-height: 1.3; }
.markdown-content :deep(h1) { font-size: 1.25em; }
.markdown-content :deep(h2) { font-size: 1.125em; }
.markdown-content :deep(h3) { font-size: 1em; }
.markdown-content :deep(hr) { border: none; border-top: 1px solid var(--color-border); margin: 0.75em 0; }
.markdown-content :deep(table) { border-collapse: collapse; width: 100%; margin: 0.5em 0; font-size: 0.875em; }
.markdown-content :deep(th), .markdown-content :deep(td) { border: 1px solid var(--color-border); padding: 0.5em 0.75em; text-align: left; }
.markdown-content :deep(th) { background: var(--color-background); font-weight: 600; }
.saved-tool-executions { display: flex; flex-direction: column; gap: 0.25rem; margin-bottom: 0.5rem; padding-bottom: 0.5rem; border-bottom: 1px dashed var(--color-border); }
.chat-input-container { display: flex; align-items: flex-end; gap: 0.5rem; padding-top: 0.5rem; border-top: 1px solid var(--color-border); }
.chat-input { flex: 1; padding: 0.625rem 0.75rem; font-size: 0.8125rem; font-family: inherit; border: 1px solid var(--color-border); border-radius: 8px; outline: none; transition: border-color 0.15s; resize: none; min-height: 40px; max-height: 100px; line-height: 1.4; }
.chat-input:focus { border-color: var(--color-primary); }
.chat-send-btn { display: flex; align-items: center; justify-content: center; width: 36px; height: 36px; padding: 0; background: var(--color-primary); color: white; border: none; border-radius: 8px; cursor: pointer; transition: opacity 0.15s; flex-shrink: 0; }
.chat-send-btn:hover:not(:disabled) { opacity: 0.9; }
.chat-send-btn:disabled { opacity: 0.5; cursor: not-allowed; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>
