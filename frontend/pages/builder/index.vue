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
const currentPhase = computed(() => currentSession.value?.hearing_phase as HearingPhase || 'purpose')
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
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    if (currentSessionId.value) {
      if (isReviewing.value) {
        refineWorkflow()
      } else {
        sendMessage()
      }
    } else {
      startNewSession()
    }
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
  <div class="min-h-screen bg-gray-50 dark:bg-gray-900">
    <!-- Header -->
    <header class="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700">
      <div class="max-w-4xl mx-auto px-4 py-4 flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 bg-gradient-to-br from-blue-500 to-purple-600 rounded-xl flex items-center justify-center">
            <svg class="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
            </svg>
          </div>
          <div>
            <h1 class="text-lg font-semibold text-gray-900 dark:text-white">AI Workflow Builder</h1>
            <p class="text-sm text-gray-500 dark:text-gray-400">対話型ワークフロー作成</p>
          </div>
        </div>

        <!-- Session Selector -->
        <div class="relative">
          <button
            class="px-3 py-2 text-sm text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg flex items-center gap-2"
            @click="showSessionList = !showSessionList"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
            </svg>
            履歴
          </button>

          <!-- Session Dropdown -->
          <div
            v-if="showSessionList"
            class="absolute right-0 mt-2 w-72 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 z-50"
          >
            <div class="p-2 border-b border-gray-200 dark:border-gray-700">
              <button
                class="w-full px-3 py-2 text-sm text-left text-blue-600 dark:text-blue-400 hover:bg-blue-50 dark:hover:bg-blue-900/30 rounded-lg"
                @click="currentSessionId = null; currentSession = null; chatHistory = []; showSessionList = false"
              >
                + 新規セッション
              </button>
            </div>
            <div class="max-h-64 overflow-y-auto">
              <button
                v-for="session in sessions"
                :key="session.id"
                class="w-full px-3 py-2 text-sm text-left hover:bg-gray-50 dark:hover:bg-gray-700 flex flex-col"
                :class="{ 'bg-blue-50 dark:bg-blue-900/30': session.id === currentSessionId }"
                @click="loadSession(session.id); showSessionList = false"
              >
                <span class="text-gray-900 dark:text-white truncate">
                  {{ builder.HEARING_PHASE_LABELS[session.hearing_phase as HearingPhase] || session.hearing_phase }}
                </span>
                <span class="text-xs text-gray-500 dark:text-gray-400">
                  {{ formatDate(session.created_at) }}
                </span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </header>

    <!-- Main Content -->
    <main class="max-w-4xl mx-auto px-4 py-6">
      <!-- Progress Bar (when in hearing) -->
      <div v-if="currentSession && currentSession.status === 'hearing'" class="mb-6">
        <div class="flex items-center justify-between mb-2">
          <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ builder.HEARING_PHASE_LABELS[currentPhase] }}
          </span>
          <span class="text-sm text-gray-500 dark:text-gray-400">
            {{ currentProgress }}%
          </span>
        </div>
        <div class="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
          <div
            class="bg-gradient-to-r from-blue-500 to-purple-600 h-2 rounded-full transition-all duration-300"
            :style="{ width: `${currentProgress}%` }"
          />
        </div>
        <div class="flex justify-between mt-2">
          <span
            v-for="(phase, index) in builder.HEARING_PHASES.slice(0, -1)"
            :key="phase"
            class="w-2 h-2 rounded-full"
            :class="builder.HEARING_PHASES.indexOf(currentPhase) >= index ? 'bg-blue-500' : 'bg-gray-300 dark:bg-gray-600'"
          />
        </div>
      </div>

      <!-- Chat Area -->
      <div class="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 min-h-[60vh] flex flex-col">
        <!-- Messages -->
        <div class="flex-1 overflow-y-auto p-4 space-y-4">
          <!-- Welcome Message (when no session) -->
          <div v-if="!currentSession && chatHistory.length === 0" class="text-center py-12">
            <div class="w-16 h-16 bg-gradient-to-br from-blue-100 to-purple-100 dark:from-blue-900/30 dark:to-purple-900/30 rounded-2xl mx-auto mb-4 flex items-center justify-center">
              <svg class="w-8 h-8 text-blue-600 dark:text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z" />
              </svg>
            </div>
            <h2 class="text-xl font-semibold text-gray-900 dark:text-white mb-2">
              ワークフローを作成しましょう
            </h2>
            <p class="text-gray-500 dark:text-gray-400 mb-6 max-w-md mx-auto">
              作りたいワークフローを教えてください。AIが要件をヒアリングし、最適なワークフローを自動生成します。
            </p>
            <div class="flex flex-wrap justify-center gap-2">
              <button
                class="px-3 py-1.5 text-sm text-gray-600 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 rounded-lg"
                @click="userInput = '毎週のレポート作成を自動化したい'"
              >
                毎週のレポート作成を自動化したい
              </button>
              <button
                class="px-3 py-1.5 text-sm text-gray-600 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 rounded-lg"
                @click="userInput = 'Slackの通知をGoogleシートに記録したい'"
              >
                Slackの通知をGoogleシートに記録したい
              </button>
              <button
                class="px-3 py-1.5 text-sm text-gray-600 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 rounded-lg"
                @click="userInput = '承認ワークフローを作りたい'"
              >
                承認ワークフローを作りたい
              </button>
            </div>
          </div>

          <!-- Chat Messages -->
          <template v-for="(msg, index) in chatHistory" :key="index">
            <div :class="msg.role === 'user' ? 'flex justify-end' : 'flex justify-start'">
              <div
                :class="[
                  'max-w-[80%] rounded-2xl px-4 py-3',
                  msg.role === 'user'
                    ? 'bg-blue-600 text-white'
                    : 'bg-gray-100 dark:bg-gray-700 text-gray-900 dark:text-white'
                ]"
              >
                <p class="whitespace-pre-wrap">{{ msg.content }}</p>

                <!-- Suggested Questions -->
                <div
                  v-if="msg.role === 'assistant' && msg.suggestedQuestions && msg.suggestedQuestions.length > 0"
                  class="mt-3 pt-3 border-t border-gray-200 dark:border-gray-600"
                >
                  <p class="text-xs text-gray-500 dark:text-gray-400 mb-2">よく聞かれる質問:</p>
                  <div class="flex flex-wrap gap-2">
                    <button
                      v-for="(q, qIndex) in msg.suggestedQuestions"
                      :key="qIndex"
                      class="text-xs px-2 py-1 bg-white dark:bg-gray-600 hover:bg-gray-50 dark:hover:bg-gray-500 rounded-full text-gray-700 dark:text-gray-200 border border-gray-200 dark:border-gray-500"
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
          <div v-if="isLoading" class="flex justify-start">
            <div class="bg-gray-100 dark:bg-gray-700 rounded-2xl px-4 py-3 flex items-center gap-2">
              <div class="flex gap-1">
                <span class="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style="animation-delay: 0ms"/>
                <span class="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style="animation-delay: 150ms"/>
                <span class="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style="animation-delay: 300ms"/>
              </div>
              <span class="text-sm text-gray-500 dark:text-gray-400">
                {{ runStatus === 'running' ? '処理中...' : '考え中...' }}
              </span>
            </div>
          </div>
        </div>

        <!-- Actions (when hearing complete) -->
        <div v-if="canConstruct" class="p-4 border-t border-gray-200 dark:border-gray-700 bg-gradient-to-r from-blue-50 to-purple-50 dark:from-blue-900/20 dark:to-purple-900/20">
          <div class="flex items-center justify-between">
            <div>
              <p class="text-sm font-medium text-gray-900 dark:text-white">ヒアリング完了!</p>
              <p class="text-xs text-gray-500 dark:text-gray-400">要件をもとにワークフローを構築できます</p>
            </div>
            <button
              :disabled="isLoading"
              class="px-4 py-2 bg-gradient-to-r from-blue-600 to-purple-600 text-white font-medium rounded-lg hover:from-blue-700 hover:to-purple-700 disabled:opacity-50 flex items-center gap-2"
              @click="constructWorkflow"
            >
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
              ワークフローを構築
            </button>
          </div>
        </div>

        <!-- Actions (when reviewing) -->
        <div v-if="isReviewing && projectId" class="p-4 border-t border-gray-200 dark:border-gray-700 bg-gradient-to-r from-green-50 to-emerald-50 dark:from-green-900/20 dark:to-emerald-900/20">
          <div class="flex items-center justify-between">
            <div>
              <p class="text-sm font-medium text-gray-900 dark:text-white">ワークフロー構築完了!</p>
              <p class="text-xs text-gray-500 dark:text-gray-400">プレビューを確認するか、フィードバックを入力して調整できます</p>
            </div>
            <div class="flex gap-2">
              <NuxtLink
                :to="`/projects/${projectId}`"
                class="px-4 py-2 bg-white dark:bg-gray-700 text-gray-700 dark:text-gray-200 font-medium rounded-lg border border-gray-200 dark:border-gray-600 hover:bg-gray-50 dark:hover:bg-gray-600 flex items-center gap-2"
              >
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                </svg>
                プレビュー
              </NuxtLink>
              <button
                :disabled="isLoading"
                class="px-4 py-2 bg-gradient-to-r from-green-600 to-emerald-600 text-white font-medium rounded-lg hover:from-green-700 hover:to-emerald-700 disabled:opacity-50 flex items-center gap-2"
                @click="finalizeWorkflow"
              >
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                </svg>
                確定
              </button>
            </div>
          </div>
        </div>

        <!-- Input Area -->
        <div class="p-4 border-t border-gray-200 dark:border-gray-700">
          <div class="flex gap-2">
            <textarea
              v-model="userInput"
              :placeholder="isReviewing ? 'フィードバックを入力...' : (currentSession ? 'メッセージを入力...' : '作りたいワークフローを教えてください...')"
              :disabled="isLoading || isBuilding"
              rows="1"
              class="flex-1 px-4 py-3 bg-gray-100 dark:bg-gray-700 border-0 rounded-xl resize-none focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-400 text-gray-900 dark:text-white placeholder-gray-500 dark:placeholder-gray-400 disabled:opacity-50"
              @keydown="handleKeyDown"
            />
            <button
              :disabled="isLoading || !userInput.trim() || isBuilding"
              class="px-4 py-3 bg-blue-600 text-white rounded-xl hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
              @click="currentSession ? (isReviewing ? refineWorkflow() : sendMessage()) : startNewSession()"
            >
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
              </svg>
            </button>
          </div>
        </div>
      </div>
    </main>
  </div>
</template>
