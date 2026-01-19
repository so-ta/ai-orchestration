<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import type { BuilderMessage, HearingPhase, BuilderRunStatus } from '~/composables/useBuilder'

const builder = useBuilder()
const toast = useToast()
const router = useRouter()

// State
const sessions = ref<Array<{
  id: string
  status: string
  hearing_phase: string
  hearing_progress: number
  project_id?: string
  created_at: string
  updated_at: string
}>>([])

const currentSessionId = ref<string | null>(null)
const currentSession = ref<{
  id: string
  status: string
  hearing_phase: string
  hearing_progress: number
  project_id?: string
  messages?: BuilderMessage[]
  created_at: string
  updated_at: string
} | null>(null)

const chatHistory = ref<Array<{ role: 'user' | 'assistant'; content: string; suggestedQuestions?: string[] }>>([])
const userInput = ref('')
const isLoading = ref(false)
const runStatus = ref<BuilderRunStatus | null>(null)
const showSessionList = ref(false)

// Computed
const currentPhase = computed(() => currentSession.value?.hearing_phase as HearingPhase || 'analysis')
const currentProgress = computed(() => currentSession.value?.hearing_progress || 0)
const isHearingComplete = computed(() => currentPhase.value === 'completed')
const canConstruct = computed(() => isHearingComplete.value && currentSession.value?.status === 'hearing')
const isBuilding = computed(() => currentSession.value?.status === 'building')
const isReviewing = computed(() => currentSession.value?.status === 'reviewing')
const projectId = computed(() => currentSession.value?.project_id)

// Methods
async function loadSessions() {
  try {
    const response = await builder.listSessions()
    sessions.value = response.sessions
  } catch (error) {
    console.error('Failed to load sessions:', error)
  }
}

async function loadSession(sessionId: string) {
  try {
    isLoading.value = true
    currentSessionId.value = sessionId
    const session = await builder.getSession(sessionId)
    currentSession.value = session

    chatHistory.value = (session.messages || []).map(msg => ({
      role: msg.role,
      content: msg.content,
      suggestedQuestions: msg.suggested_questions,
    }))
  } catch (error) {
    console.error('Failed to load session:', error)
    toast.error('セッションの読み込みに失敗しました')
  } finally {
    isLoading.value = false
  }
}

async function startNewSession() {
  if (!userInput.value.trim()) {
    toast.warning('作りたいワークフローを入力してください')
    return
  }

  try {
    isLoading.value = true
    const response = await builder.startSession(userInput.value)
    currentSessionId.value = response.session_id

    chatHistory.value = [{
      role: 'user',
      content: userInput.value,
    }]

    if (response.message) {
      chatHistory.value.push({
        role: 'assistant',
        content: response.message.content,
        suggestedQuestions: response.message.suggested_questions,
      })
    }

    currentSession.value = {
      id: response.session_id,
      status: response.status,
      hearing_phase: response.phase,
      hearing_progress: response.progress,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }

    userInput.value = ''
    await loadSessions()

  } catch (error) {
    console.error('Failed to start session:', error)
    toast.error('セッションの開始に失敗しました')
  } finally {
    isLoading.value = false
  }
}

async function sendMessage() {
  if (!userInput.value.trim() || !currentSessionId.value) return

  const message = userInput.value
  userInput.value = ''

  chatHistory.value.push({
    role: 'user',
    content: message,
  })

  try {
    isLoading.value = true
    runStatus.value = 'pending'

    const session = await builder.sendMessageAndWait(
      currentSessionId.value,
      message,
      (status) => { runStatus.value = status }
    )

    currentSession.value = session

    // Find the latest assistant message
    if (session.messages) {
      const latestAssistant = session.messages.filter(m => m.role === 'assistant').pop()
      if (latestAssistant) {
        chatHistory.value.push({
          role: 'assistant',
          content: latestAssistant.content,
          suggestedQuestions: latestAssistant.suggested_questions,
        })
      }
    }

  } catch (error) {
    console.error('Failed to send message:', error)
    toast.error('メッセージの送信に失敗しました')
  } finally {
    isLoading.value = false
    runStatus.value = null
  }
}

async function constructWorkflow() {
  if (!currentSessionId.value) return

  try {
    isLoading.value = true
    runStatus.value = 'pending'

    const session = await builder.constructAndWait(
      currentSessionId.value,
      (status) => { runStatus.value = status }
    )

    currentSession.value = session
    toast.success('ワークフローを構築しました')

    if (session.project_id) {
      chatHistory.value.push({
        role: 'assistant',
        content: `ワークフローを構築しました！プロジェクトID: ${session.project_id}\n\n以下からワークフローを確認・調整できます。`,
      })
    }

  } catch (error) {
    console.error('Failed to construct workflow:', error)
    toast.error('ワークフローの構築に失敗しました')
  } finally {
    isLoading.value = false
    runStatus.value = null
  }
}

async function refineWorkflow() {
  if (!userInput.value.trim() || !currentSessionId.value) return

  const feedback = userInput.value
  userInput.value = ''

  chatHistory.value.push({
    role: 'user',
    content: feedback,
  })

  try {
    isLoading.value = true
    runStatus.value = 'pending'

    const session = await builder.refineAndWait(
      currentSessionId.value,
      feedback,
      (status) => { runStatus.value = status }
    )

    currentSession.value = session

    chatHistory.value.push({
      role: 'assistant',
      content: 'フィードバックを反映しました。',
    })

  } catch (error) {
    console.error('Failed to refine workflow:', error)
    toast.error('ワークフローの調整に失敗しました')
  } finally {
    isLoading.value = false
    runStatus.value = null
  }
}

async function finalizeWorkflow() {
  if (!currentSessionId.value) return

  try {
    isLoading.value = true
    await builder.finalize(currentSessionId.value)

    toast.success('ワークフローを確定しました')

    if (projectId.value) {
      router.push(`/projects/${projectId.value}`)
    }

  } catch (error) {
    console.error('Failed to finalize workflow:', error)
    toast.error('ワークフローの確定に失敗しました')
  } finally {
    isLoading.value = false
  }
}

function useSuggestedQuestion(question: string) {
  userInput.value = question
}

function handleKeyDown(e: KeyboardEvent) {
  if (e.key !== 'Enter' || e.shiftKey) {
    return
  }

  e.preventDefault()

  if (!currentSessionId.value) {
    startNewSession()
    return
  }

  if (isReviewing.value) {
    refineWorkflow()
  } else {
    sendMessage()
  }
}

function formatDate(dateString: string): string {
  const date = new Date(dateString)
  return date.toLocaleDateString('ja-JP', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

// Lifecycle
onMounted(() => {
  loadSessions()
})
</script>

<template>
  <div class="builder-page">
    <!-- Header -->
    <header class="builder-header">
      <div class="header-left">
        <div class="header-icon">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" />
          </svg>
        </div>
        <div class="header-title">
          <h1>AI Workflow Builder</h1>
          <p>対話型ワークフロー作成</p>
        </div>
      </div>

      <!-- Session Selector -->
      <div class="session-selector">
        <button class="session-btn" @click="showSessionList = !showSessionList">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
          </svg>
          履歴
        </button>

        <!-- Session Dropdown -->
        <div v-if="showSessionList" class="session-dropdown">
          <div class="session-dropdown-header">
            <button
              class="new-session-btn"
              @click="currentSessionId = null; currentSession = null; chatHistory = []; showSessionList = false"
            >
              + 新規セッション
            </button>
          </div>
          <div class="session-list">
            <button
              v-for="session in sessions"
              :key="session.id"
              class="session-item"
              :class="{ active: session.id === currentSessionId }"
              @click="loadSession(session.id); showSessionList = false"
            >
              <span class="session-phase">
                {{ builder.HEARING_PHASE_LABELS[session.hearing_phase as HearingPhase] || session.hearing_phase }}
              </span>
              <span class="session-date">
                {{ formatDate(session.created_at) }}
              </span>
            </button>
          </div>
        </div>
      </div>
    </header>

    <!-- Main Content -->
    <main class="builder-main">
      <!-- Progress Bar (when in hearing) -->
      <div v-if="currentSession && currentSession.status === 'hearing'" class="progress-section">
        <div class="progress-header">
          <span class="progress-label">{{ builder.HEARING_PHASE_LABELS[currentPhase] }}</span>
          <span class="progress-value">{{ currentProgress }}%</span>
        </div>
        <div class="progress-bar">
          <div class="progress-fill" :style="{ width: `${currentProgress}%` }" />
        </div>
        <div class="progress-dots">
          <span
            v-for="(phase, index) in builder.HEARING_PHASES.slice(0, -1)"
            :key="phase"
            class="progress-dot"
            :class="{ active: builder.HEARING_PHASES.indexOf(currentPhase) >= index }"
          />
        </div>
      </div>

      <!-- Chat Area -->
      <div class="chat-container">
        <!-- Messages -->
        <div class="chat-messages">
          <!-- Welcome Message (when no session) -->
          <div v-if="!currentSession && chatHistory.length === 0" class="welcome-section">
            <div class="welcome-icon">
              <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z" />
              </svg>
            </div>
            <h2>ワークフローを作成しましょう</h2>
            <p>作りたいワークフローを教えてください。AIが深く考え、仮定条件を提案し、最小限の質問で最適なワークフローを自動生成します。</p>
            <div class="suggestion-buttons">
              <button class="suggestion-btn" @click="userInput = '毎週のレポート作成を自動化したい'">
                毎週のレポート作成を自動化したい
              </button>
              <button class="suggestion-btn" @click="userInput = 'Slackの通知をGoogleシートに記録したい'">
                Slackの通知をGoogleシートに記録したい
              </button>
              <button class="suggestion-btn" @click="userInput = '承認ワークフローを作りたい'">
                承認ワークフローを作りたい
              </button>
            </div>
          </div>

          <!-- Chat Messages -->
          <template v-for="(msg, index) in chatHistory" :key="index">
            <div class="message" :class="msg.role">
              <div class="message-bubble">
                <p>{{ msg.content }}</p>

                <!-- Suggested Questions -->
                <div
                  v-if="msg.role === 'assistant' && msg.suggestedQuestions && msg.suggestedQuestions.length > 0"
                  class="suggested-questions"
                >
                  <p class="suggested-label">よく聞かれる質問:</p>
                  <div class="suggested-list">
                    <button
                      v-for="(q, qIndex) in msg.suggestedQuestions"
                      :key="qIndex"
                      class="suggested-item"
                      @click="useSuggestedQuestion(q)"
                    >
                      {{ q }}
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </template>

          <!-- Loading Indicator -->
          <div v-if="isLoading" class="message assistant">
            <div class="message-bubble loading">
              <div class="loading-dots">
                <span />
                <span />
                <span />
              </div>
              <span class="loading-text">{{ runStatus === 'running' ? '処理中...' : '考え中...' }}</span>
            </div>
          </div>
        </div>

        <!-- Actions (when hearing complete) -->
        <div v-if="canConstruct" class="action-panel hearing-complete">
          <div class="action-info">
            <p class="action-title">ヒアリング完了!</p>
            <p class="action-subtitle">要件をもとにワークフローを構築できます</p>
          </div>
          <button :disabled="isLoading" class="action-btn primary" @click="constructWorkflow">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" />
            </svg>
            ワークフローを構築
          </button>
        </div>

        <!-- Actions (when reviewing) -->
        <div v-if="isReviewing && projectId" class="action-panel reviewing">
          <div class="action-info">
            <p class="action-title">ワークフロー構築完了!</p>
            <p class="action-subtitle">プレビューを確認するか、フィードバックを入力して調整できます</p>
          </div>
          <div class="action-buttons">
            <NuxtLink :to="`/projects/${projectId}`" class="action-btn secondary">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                <path stroke-linecap="round" stroke-linejoin="round" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
              </svg>
              プレビュー
            </NuxtLink>
            <button :disabled="isLoading" class="action-btn success" @click="finalizeWorkflow">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
              </svg>
              確定
            </button>
          </div>
        </div>

        <!-- Input Area -->
        <div class="input-area">
          <div class="input-wrapper">
            <textarea
              v-model="userInput"
              :placeholder="isReviewing ? 'フィードバックを入力...' : (currentSession ? 'メッセージを入力...' : '作りたいワークフローを教えてください...')"
              :disabled="isLoading || isBuilding"
              rows="1"
              @keydown="handleKeyDown"
            />
            <button
              :disabled="isLoading || !userInput.trim() || isBuilding"
              class="send-btn"
              @click="currentSession ? (isReviewing ? refineWorkflow() : sendMessage()) : startNewSession()"
            >
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
              </svg>
            </button>
          </div>
        </div>
      </div>
    </main>
  </div>
</template>

<style scoped>
.builder-page {
  min-height: calc(100vh - 56px);
  background: var(--color-background);
}

/* Header */
.builder-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem 1.5rem;
  background: var(--color-surface);
  border-bottom: 1px solid var(--color-border);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.header-icon {
  width: 40px;
  height: 40px;
  background: linear-gradient(135deg, #3b82f6, #8b5cf6);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
}

.header-title h1 {
  font-size: 1.125rem;
  font-weight: 600;
  margin: 0;
  color: var(--color-text);
}

.header-title p {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
  margin: 0;
}

/* Session Selector */
.session-selector {
  position: relative;
}

.session-btn {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
  font-size: 0.875rem;
  color: var(--color-text-secondary);
  background: transparent;
  border: none;
  border-radius: var(--radius);
  cursor: pointer;
  transition: all 0.15s;
}

.session-btn:hover {
  background: var(--color-background);
  color: var(--color-text);
}

.session-dropdown {
  position: absolute;
  right: 0;
  top: 100%;
  margin-top: 0.5rem;
  width: 280px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  box-shadow: var(--shadow-lg);
  z-index: 50;
}

.session-dropdown-header {
  padding: 0.5rem;
  border-bottom: 1px solid var(--color-border);
}

.new-session-btn {
  width: 100%;
  padding: 0.5rem 0.75rem;
  font-size: 0.875rem;
  text-align: left;
  color: var(--color-primary);
  background: transparent;
  border: none;
  border-radius: var(--radius);
  cursor: pointer;
}

.new-session-btn:hover {
  background: rgba(59, 130, 246, 0.1);
}

.session-list {
  max-height: 256px;
  overflow-y: auto;
}

.session-item {
  display: flex;
  flex-direction: column;
  width: 100%;
  padding: 0.5rem 0.75rem;
  text-align: left;
  background: transparent;
  border: none;
  cursor: pointer;
  transition: background 0.15s;
}

.session-item:hover {
  background: var(--color-background);
}

.session-item.active {
  background: rgba(59, 130, 246, 0.1);
}

.session-phase {
  font-size: 0.875rem;
  color: var(--color-text);
}

.session-date {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

/* Main Content */
.builder-main {
  max-width: 800px;
  margin: 0 auto;
  padding: 1.5rem;
}

/* Progress Section */
.progress-section {
  margin-bottom: 1.5rem;
}

.progress-header {
  display: flex;
  justify-content: space-between;
  margin-bottom: 0.5rem;
}

.progress-label {
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--color-text);
}

.progress-value {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
}

.progress-bar {
  width: 100%;
  height: 8px;
  background: var(--color-border);
  border-radius: 4px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #3b82f6, #8b5cf6);
  border-radius: 4px;
  transition: width 0.3s;
}

.progress-dots {
  display: flex;
  justify-content: space-between;
  margin-top: 0.5rem;
}

.progress-dot {
  width: 8px;
  height: 8px;
  background: var(--color-border);
  border-radius: 50%;
}

.progress-dot.active {
  background: #3b82f6;
}

/* Chat Container */
.chat-container {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 12px;
  min-height: 60vh;
  display: flex;
  flex-direction: column;
}

.chat-messages {
  flex: 1;
  padding: 1rem;
  overflow-y: auto;
}

/* Welcome Section */
.welcome-section {
  text-align: center;
  padding: 3rem 1rem;
}

.welcome-icon {
  width: 64px;
  height: 64px;
  margin: 0 auto 1rem;
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.1), rgba(139, 92, 246, 0.1));
  border-radius: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #3b82f6;
}

.welcome-section h2 {
  font-size: 1.25rem;
  font-weight: 600;
  color: var(--color-text);
  margin: 0 0 0.5rem;
}

.welcome-section > p {
  color: var(--color-text-secondary);
  margin: 0 0 1.5rem;
  max-width: 400px;
  margin-left: auto;
  margin-right: auto;
}

.suggestion-buttons {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 0.5rem;
}

.suggestion-btn {
  padding: 0.5rem 0.75rem;
  font-size: 0.875rem;
  color: var(--color-text-secondary);
  background: var(--color-background);
  border: none;
  border-radius: var(--radius);
  cursor: pointer;
  transition: all 0.15s;
}

.suggestion-btn:hover {
  background: var(--color-border);
  color: var(--color-text);
}

/* Messages */
.message {
  margin-bottom: 1rem;
}

.message.user {
  display: flex;
  justify-content: flex-end;
}

.message.assistant {
  display: flex;
  justify-content: flex-start;
}

.message-bubble {
  max-width: 80%;
  padding: 0.75rem 1rem;
  border-radius: 16px;
}

.message.user .message-bubble {
  background: #3b82f6;
  color: white;
}

.message.assistant .message-bubble {
  background: var(--color-background);
  color: var(--color-text);
}

.message-bubble p {
  margin: 0;
  white-space: pre-wrap;
}

/* Suggested Questions */
.suggested-questions {
  margin-top: 0.75rem;
  padding-top: 0.75rem;
  border-top: 1px solid var(--color-border);
}

.suggested-label {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  margin: 0 0 0.5rem;
}

.suggested-list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.suggested-item {
  font-size: 0.75rem;
  padding: 0.25rem 0.5rem;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 999px;
  color: var(--color-text);
  cursor: pointer;
  transition: all 0.15s;
}

.suggested-item:hover {
  background: var(--color-background);
}

/* Loading */
.message-bubble.loading {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.loading-dots {
  display: flex;
  gap: 4px;
}

.loading-dots span {
  width: 8px;
  height: 8px;
  background: var(--color-text-secondary);
  border-radius: 50%;
  animation: bounce 1s infinite;
}

.loading-dots span:nth-child(2) {
  animation-delay: 0.15s;
}

.loading-dots span:nth-child(3) {
  animation-delay: 0.3s;
}

@keyframes bounce {
  0%, 60%, 100% {
    transform: translateY(0);
  }
  30% {
    transform: translateY(-4px);
  }
}

.loading-text {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
}

/* Action Panels */
.action-panel {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem;
  border-top: 1px solid var(--color-border);
}

.action-panel.hearing-complete {
  background: linear-gradient(90deg, rgba(59, 130, 246, 0.05), rgba(139, 92, 246, 0.05));
}

.action-panel.reviewing {
  background: linear-gradient(90deg, rgba(34, 197, 94, 0.05), rgba(16, 185, 129, 0.05));
}

.action-info {
  flex: 1;
}

.action-title {
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--color-text);
  margin: 0;
}

.action-subtitle {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  margin: 0;
}

.action-buttons {
  display: flex;
  gap: 0.5rem;
}

.action-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  font-size: 0.875rem;
  font-weight: 500;
  border: none;
  border-radius: var(--radius);
  cursor: pointer;
  text-decoration: none;
  transition: all 0.15s;
}

.action-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.action-btn.primary {
  background: linear-gradient(135deg, #3b82f6, #8b5cf6);
  color: white;
}

.action-btn.primary:hover:not(:disabled) {
  filter: brightness(1.1);
}

.action-btn.secondary {
  background: var(--color-surface);
  color: var(--color-text);
  border: 1px solid var(--color-border);
}

.action-btn.secondary:hover {
  background: var(--color-background);
}

.action-btn.success {
  background: linear-gradient(135deg, #22c55e, #10b981);
  color: white;
}

.action-btn.success:hover:not(:disabled) {
  filter: brightness(1.1);
}

/* Input Area */
.input-area {
  padding: 1rem;
  border-top: 1px solid var(--color-border);
}

.input-wrapper {
  display: flex;
  gap: 0.5rem;
}

.input-wrapper textarea {
  flex: 1;
  padding: 0.75rem 1rem;
  font-size: 1rem;
  background: var(--color-background);
  border: none;
  border-radius: 12px;
  resize: none;
  color: var(--color-text);
}

.input-wrapper textarea:focus {
  outline: none;
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.3);
}

.input-wrapper textarea:disabled {
  opacity: 0.5;
}

.input-wrapper textarea::placeholder {
  color: var(--color-text-secondary);
}

.send-btn {
  padding: 0.75rem;
  background: #3b82f6;
  color: white;
  border: none;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.15s;
}

.send-btn:hover:not(:disabled) {
  background: #2563eb;
}

.send-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
