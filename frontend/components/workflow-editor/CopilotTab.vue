<script setup lang="ts">
import type { Step } from '~/types/api'
import type { StepSuggestion, ExplainResponse, CopilotSession, CopilotMessage, GenerateWorkflowResponse, CopilotRunStatus } from '~/composables/useCopilot'

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
const chatHistory = ref<Array<{ role: 'user' | 'assistant'; content: string }>>([])
const suggestions = ref<StepSuggestion[]>([])
const explanation = ref<ExplainResponse | null>(null)

// Session state
const currentSession = ref<CopilotSession | null>(null)
const sessions = ref<CopilotSession[]>([])
const showSessionMenu = ref(false)
const isLoadingSession = ref(false)

// Check if step is selected
const hasStep = computed(() => props.step !== null)

// Load session on mount
onMounted(async () => {
  await loadSession()
})

// Load current session
async function loadSession() {
  isLoadingSession.value = true
  try {
    const session = await copilot.getOrCreateSession(props.workflowId)
    currentSession.value = session
    // Load messages if session has them
    if (session.messages && session.messages.length > 0) {
      chatHistory.value = session.messages.map((msg: CopilotMessage) => ({
        role: msg.role,
        content: msg.content,
      }))
    }
    // Also load session list
    sessions.value = await copilot.listSessions(props.workflowId)
  } catch (error) {
    console.error('Failed to load session:', error)
  } finally {
    isLoadingSession.value = false
  }
}

// Start new session
async function startNewSession() {
  isLoadingSession.value = true
  try {
    const session = await copilot.startNewSession(props.workflowId)
    currentSession.value = session
    chatHistory.value = []
    sessions.value = await copilot.listSessions(props.workflowId)
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
    const session = await copilot.getSessionMessages(props.workflowId, sessionId)
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
function formatSessionTitle(session: CopilotSession): string {
  if (session.title) return session.title
  const date = new Date(session.created_at)
  return date.toLocaleDateString() + ' ' + date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

// Chat with copilot (using session persistence)
async function sendMessage() {
  if (!chatMessage.value.trim() || isLoading.value) return

  const message = chatMessage.value.trim()
  chatMessage.value = ''
  chatHistory.value.push({ role: 'user', content: message })

  isLoading.value = true
  try {
    const context = props.step
      ? `Currently editing step: ${props.step.name} (${props.step.type})`
      : 'No step selected - general workflow assistance'
    const response = await copilot.chatWithSession(props.workflowId, message, {
      sessionId: currentSession.value?.id,
      context,
    })
    chatHistory.value.push({ role: 'assistant', content: response.response })
    // Update session if returned
    if (response.session) {
      currentSession.value = response.session
    }

    // If suggestions were returned, update them
    if (response.suggestions && response.suggestions.length > 0) {
      suggestions.value = response.suggestions
    }
  } catch (error: unknown) {
    toast.error(t('copilot.errors.chatFailed'))
    console.error('Chat error:', error)
  } finally {
    isLoading.value = false
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

// State for workflow generation
const isGenerating = ref(false)
const generateDescription = ref('')
const showGenerateModal = ref(false)
const generateStatus = ref<CopilotRunStatus | null>(null)

// Generate workflow from description (async polling)
async function generateWorkflow() {
  if (!generateDescription.value.trim() || isGenerating.value) return

  isGenerating.value = true
  generateStatus.value = 'pending'

  // Add user message to chat history immediately
  chatHistory.value.push({
    role: 'user',
    content: `[Workflow Generation] ${generateDescription.value.trim()}`
  })

  try {
    // Use async polling approach
    const result = await copilot.asyncGenerateWorkflow(
      generateDescription.value.trim(),
      {
        sessionId: currentSession.value?.id,
        onProgress: (status) => {
          generateStatus.value = status
        }
      }
    )

    // Check for errors
    if (result.status === 'failed') {
      throw new Error(result.error || 'Workflow generation failed')
    }

    if (result.output) {
      // Add response to chat history
      chatHistory.value.push({
        role: 'assistant',
        content: result.output.response
      })

      // Emit workflow to parent for canvas application
      emit('apply-workflow', result.output)
      toast.success(t('copilot.workflowGenerated'))
    }

    // Close modal and clear input
    showGenerateModal.value = false
    generateDescription.value = ''
  } catch (error: unknown) {
    // Remove user message if generation failed
    chatHistory.value.pop()
    toast.error(t('copilot.errors.generateFailed'))
    console.error('Generate workflow error:', error)
  } finally {
    isGenerating.value = false
    generateStatus.value = null
  }
}

// Quick actions
const quickActions = [
  { id: 'suggest', icon: 'lightbulb', label: 'copilot.actions.suggest', action: fetchSuggestions },
  { id: 'explain', icon: 'info', label: 'copilot.actions.explain', action: fetchExplanation },
]
</script>

<template>
  <div class="copilot-tab">
    <!-- Session Header -->
    <div class="session-header">
      <div class="session-selector" @click="showSessionMenu = !showSessionMenu">
        <span class="session-label">{{ t('copilot.session') }}:</span>
        <span class="session-title">{{ currentSession ? formatSessionTitle(currentSession) : t('copilot.loading') }}</span>
        <svg class="session-chevron" :class="{ open: showSessionMenu }" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="6 9 12 15 18 9"/>
        </svg>
      </div>
      <button class="new-session-btn" :disabled="isLoadingSession" :title="t('copilot.newSession')" @click="startNewSession">
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="12" y1="5" x2="12" y2="19"/>
          <line x1="5" y1="12" x2="19" y2="12"/>
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
            <span v-if="session.is_active" class="session-active-badge">{{ t('copilot.active') }}</span>
          </div>
          <div v-if="sessions.length === 0" class="session-empty">{{ t('copilot.noSessions') }}</div>
        </div>
      </div>
    </div>

    <!-- Generate Workflow Button (always available) -->
    <button
      class="generate-workflow-btn"
      :disabled="isGenerating"
      @click="showGenerateModal = true"
    >
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M12 3v3m0 12v3M3 12h3m12 0h3M5.636 5.636l2.122 2.122m8.484 8.484l2.122 2.122M5.636 18.364l2.122-2.122m8.484-8.484l2.122-2.122"/>
      </svg>
      {{ t('copilot.generateWorkflow') }}
    </button>

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
      <span>{{ t('copilot.thinking') }}</span>
    </div>

    <!-- Chat Section -->
    <div v-if="activeSection === 'chat'" class="chat-section">
      <div class="chat-messages">
        <div v-if="chatHistory.length === 0" class="chat-empty">
          <p>{{ t('copilot.chatWelcome') }}</p>
          <p class="chat-hint">{{ t('copilot.chatHint') }}</p>
        </div>
        <div
          v-for="(msg, idx) in chatHistory"
          :key="idx"
          class="chat-message"
          :class="msg.role"
        >
          <div class="message-content">{{ msg.content }}</div>
        </div>
      </div>
      <div class="chat-input-container">
        <textarea
          v-model="chatMessage"
          class="chat-input"
          :placeholder="t('copilot.chatPlaceholder')"
          :disabled="isLoading"
          rows="2"
          @keydown.meta.enter="sendMessage"
          @keydown.ctrl.enter="sendMessage"
        />
        <button
          class="chat-send-btn"
          :disabled="!chatMessage.trim() || isLoading"
          @click="sendMessage"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
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

    <!-- Generate Workflow Modal -->
    <Teleport to="body">
      <div v-if="showGenerateModal" class="modal-overlay" @click.self="showGenerateModal = false">
        <div class="generate-modal">
          <div class="modal-header">
            <h3>{{ t('copilot.generateWorkflowTitle') }}</h3>
            <button class="modal-close" @click="showGenerateModal = false">
              <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="18" y1="6" x2="6" y2="18"/>
                <line x1="6" y1="6" x2="18" y2="18"/>
              </svg>
            </button>
          </div>
          <div class="modal-body">
            <p class="modal-description">{{ t('copilot.generateWorkflowDescription') }}</p>
            <textarea
              v-model="generateDescription"
              class="generate-input"
              :placeholder="t('copilot.generatePlaceholder')"
              rows="4"
              :disabled="isGenerating"
            />
          </div>
          <div class="modal-footer">
            <button class="btn-cancel" :disabled="isGenerating" @click="showGenerateModal = false">
              {{ t('common.cancel') }}
            </button>
            <button
              class="btn-generate"
              :disabled="!generateDescription.trim() || isGenerating"
              @click="generateWorkflow"
            >
              <div v-if="isGenerating" class="btn-spinner" />
              <template v-if="isGenerating && generateStatus">
                {{ generateStatus === 'pending' ? t('copilot.statusPending') : generateStatus === 'running' ? t('copilot.statusRunning') : t('copilot.generating') }}
              </template>
              <template v-else>
                {{ t('copilot.generate') }}
              </template>
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.copilot-tab {
  display: flex;
  flex-direction: column;
  height: 100%;
  gap: 1rem;
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

.new-session-btn {
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

.new-session-btn:hover:not(:disabled) {
  background: var(--color-primary);
  border-color: var(--color-primary);
  color: white;
}

.new-session-btn:disabled {
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

.session-active-badge {
  font-size: 0.625rem;
  font-weight: 600;
  padding: 0.125rem 0.375rem;
  background: var(--color-success);
  color: white;
  border-radius: 4px;
}

.session-empty {
  padding: 1rem;
  text-align: center;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
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

/* Generate Workflow Button */
.generate-workflow-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  width: 100%;
  padding: 0.75rem 1rem;
  font-size: 0.875rem;
  font-weight: 600;
  color: white;
  background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
  border: none;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
}

.generate-workflow-btn:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(99, 102, 241, 0.3);
}

.generate-workflow-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

/* Generate Modal */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.generate-modal {
  width: 90%;
  max-width: 500px;
  background: var(--color-surface);
  border-radius: 12px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
  overflow: hidden;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem 1.25rem;
  border-bottom: 1px solid var(--color-border);
}

.modal-header h3 {
  margin: 0;
  font-size: 1rem;
  font-weight: 600;
  color: var(--color-text);
}

.modal-close {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  padding: 0;
  color: var(--color-text-secondary);
  background: transparent;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
}

.modal-close:hover {
  background: var(--color-background);
  color: var(--color-text);
}

.modal-body {
  padding: 1.25rem;
}

.modal-description {
  margin: 0 0 1rem;
  font-size: 0.875rem;
  color: var(--color-text-secondary);
  line-height: 1.5;
}

.generate-input {
  width: 100%;
  padding: 0.75rem;
  font-size: 0.875rem;
  font-family: inherit;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  outline: none;
  resize: vertical;
  min-height: 100px;
  transition: border-color 0.15s;
}

.generate-input:focus {
  border-color: var(--color-primary);
}

.modal-footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 0.75rem;
  padding: 1rem 1.25rem;
  border-top: 1px solid var(--color-border);
  background: var(--color-background);
}

.btn-cancel {
  padding: 0.5rem 1rem;
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--color-text-secondary);
  background: transparent;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
}

.btn-cancel:hover:not(:disabled) {
  background: var(--color-background);
  border-color: var(--color-text-secondary);
}

.btn-generate {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1.25rem;
  font-size: 0.875rem;
  font-weight: 600;
  color: white;
  background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
  border: none;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
}

.btn-generate:hover:not(:disabled) {
  box-shadow: 0 4px 12px rgba(99, 102, 241, 0.3);
}

.btn-generate:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-spinner {
  width: 14px;
  height: 14px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
</style>
