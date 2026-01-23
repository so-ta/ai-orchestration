<script setup lang="ts">
/**
 * CopilotWelcomePanel - Unified Copilot onboarding panel
 *
 * Shared component for WelcomeDialog and CopilotTab empty state.
 * Contains header, tab switcher, templates, and structured prompt input.
 * Focus on guiding user input with inline editing for slots.
 */
import { useSnippetInput } from '~/composables/useSnippetInput'

const props = defineProps<{
  /** Compact mode for smaller spaces like Copilot sidebar */
  compact?: boolean
  /** Show skip to canvas button */
  showSkipButton?: boolean
  /** Auto focus first input on mount and use case change */
  autoFocus?: boolean
}>()

const emit = defineEmits<{
  submit: [prompt: string]
  skip: []
}>()

const { t, locale } = useI18n()

// Slot examples data (matching i18n structure but kept here for reliable access)
const slotExamplesData: Record<string, Record<string, string[]>> = {
  periodic: {
    頻度: ['毎朝9時', '毎週月曜', '毎月1日', '毎時'],
    情報: ['天気予報', '売上サマリー', 'ニュース', 'タスク一覧'],
    通知先: ['Slack', 'メール', 'LINE', 'Discord'],
  },
  event: {
    サービス: ['GitHub', 'Slack', 'Notion', 'Webhook'],
    イベント: ['Issueが作成されたら', 'メッセージが来たら', 'ページが更新されたら', 'データが追加されたら'],
    アクション: ['タスクを作成', '通知を送る', 'データを保存', '分類して振り分け'],
  },
  analysis: {
    データソース: ['売上データ', 'アクセスログ', '顧客データ', 'SNS投稿'],
    出力先: ['Slack', 'Notion', 'Google Sheets', 'メール'],
  },
  sync: {
    ソース: ['Notion', 'Google Sheets', 'Salesforce', 'Airtable'],
    データ: ['タスク', '顧客情報', '商品データ', 'スケジュール'],
    宛先: ['カレンダー', 'CRM', 'Slack', 'データベース'],
  },
  aiGenerate: {
    入力: ['会議メモ', 'キーワード', '要件', 'データ'],
    成果物: ['議事録', '記事', '提案書', 'サマリー'],
  },
}

// Use case keys
const useCaseKeys = ['periodic', 'event', 'analysis', 'sync', 'aiGenerate', 'freeform'] as const
type UseCaseKey = (typeof useCaseKeys)[number]

// Use case state
const selectedUseCase = ref<UseCaseKey>('periodic')

// Snippet input composable
const {
  templateParts,
  slots,
  allSlotsFilled,
  hasSlots,
  finalPrompt,
  setTemplate,
  selectSlot,
  fillSlot,
  getSlotValue,
  isSlotFilled,
  getSlotIndex,
} = useSnippetInput()

// Track focused slot
const focusedSlot = ref<string | null>(null)

// Use case options for dropdown
const useCaseOptions = computed(() =>
  useCaseKeys.map((key) => ({
    key,
    label: t(`welcomeDialog.useCases.${key}.label`),
  }))
)


// Focus first slot helper
function focusFirstSlot() {
  if (!props.autoFocus) return
  nextTick(() => {
    const firstSlot = slots.value[0]
    if (firstSlot) {
      focusedSlot.value = firstSlot.content
      nextTick(() => {
        slotInputRefs.value[firstSlot.content]?.focus()
      })
    }
  })
}

// Watch for use case changes
watch(
  selectedUseCase,
  (newCase) => {
    if (newCase === 'freeform') {
      setTemplate('')
    } else {
      const template = t(`welcomeDialog.useCases.${newCase}.template`) || ''
      setTemplate(template)
      // Focus first slot when use case changes (after template is set)
      focusFirstSlot()
    }
    focusedSlot.value = null
  },
  { immediate: true }
)

// Watch for locale changes
watch(locale, () => {
  if (selectedUseCase.value !== 'freeform') {
    const template = t(`welcomeDialog.useCases.${selectedUseCase.value}.template`) || ''
    setTemplate(template)
  }
})

// Auto focus on mount
onMounted(() => {
  if (props.autoFocus && selectedUseCase.value !== 'freeform') {
    focusFirstSlot()
  }
})

// Get the first unfilled slot label for showing examples
const activeSlotForExamples = computed((): string | null => {
  // If a slot is focused, use that
  if (focusedSlot.value) return focusedSlot.value
  // Otherwise, find the first unfilled slot
  for (const slot of slots.value) {
    if (!isSlotFilled(slot.content)) {
      return slot.content
    }
  }
  return null
})

// Get examples for the active slot
const activeSlotExamples = computed((): string[] => {
  if (selectedUseCase.value === 'freeform') return []
  if (!activeSlotForExamples.value) return []

  const useCaseExamples = slotExamplesData[selectedUseCase.value]
  if (!useCaseExamples) return []

  return useCaseExamples[activeSlotForExamples.value] || []
})

// Handle submit
function handleSubmit() {
  const prompt = finalPrompt.value.trim()
  if (!prompt) return

  emit('submit', prompt)
  setTemplate('')
  selectedUseCase.value = 'periodic'
  focusedSlot.value = null
}

// Flag to prevent blur from interfering with example click
const isClickingExample = ref(false)

// Handle example click - fill the slot and move to next
function handleExampleClick(example: string) {
  const slotLabel = activeSlotForExamples.value
  if (!slotLabel) return

  isClickingExample.value = true
  fillSlot(slotLabel, example)
  nextTick(() => {
    isClickingExample.value = false
    // Move to next unfilled slot
    const currentIdx = getSlotIndex(slotLabel)
    for (let i = 1; i <= slots.value.length; i++) {
      const nextIdx = (currentIdx + i) % slots.value.length
      const nextSlot = slots.value[nextIdx]
      if (!isSlotFilled(nextSlot.content)) {
        focusedSlot.value = nextSlot.content
        nextTick(() => {
          const inputEl = slotInputRefs.value[nextSlot.content]
          inputEl?.focus()
        })
        return
      }
    }
    // All filled
    focusedSlot.value = null
  })
}

// Handle slot input change
function handleSlotInput(label: string, value: string) {
  fillSlot(label, value)
}

// Handle slot focus
function handleSlotFocus(label: string) {
  focusedSlot.value = label
  const index = getSlotIndex(label)
  if (index >= 0) {
    selectSlot(index)
  }
}

// Handle slot blur
function handleSlotBlur() {
  // Don't clear focused slot if clicking example
  if (isClickingExample.value) return
  focusedSlot.value = null
}

// Refs for all slot inputs
const slotInputRefs = ref<Record<string, HTMLInputElement | null>>({})

// Set input ref for a slot
function setSlotInputRef(el: HTMLInputElement | null, label: string) {
  if (el) {
    slotInputRefs.value[label] = el
  }
}

// Calculate input width based on value or placeholder
// Japanese characters are roughly 2ch wide, ASCII is 1ch
function getSlotInputWidth(label: string): string {
  const value = getSlotValue(label) || ''

  // Calculate width considering character width (Japanese ~2ch, ASCII ~1ch)
  const calcWidth = (str: string) => {
    let width = 0
    for (const char of str) {
      // Check if character is ASCII (single-byte) or not (multi-byte like Japanese)
      width += char.charCodeAt(0) > 127 ? 2 : 1
    }
    return width
  }

  const minWidth = calcWidth(label)
  const contentWidth = calcWidth(value)
  // Use the larger of placeholder width or value width, plus some padding
  const width = Math.max(minWidth, contentWidth) + 1
  return `${width}ch`
}

// Handle input keydown for a specific slot
function handleSlotKeydown(e: KeyboardEvent, slotLabel: string) {
  // Cmd/Ctrl+Enter: always submit
  if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
    e.preventDefault()
    handleSubmit()
    return
  }

  if (e.key === 'Enter') {
    e.preventDefault()
    // If all slots filled, submit
    if (allSlotsFilled.value) {
      handleSubmit()
    } else {
      // Move to next unfilled slot
      const currentIdx = getSlotIndex(slotLabel)
      for (let i = 1; i <= slots.value.length; i++) {
        const nextIdx = (currentIdx + i) % slots.value.length
        const nextSlot = slots.value[nextIdx]
        if (!isSlotFilled(nextSlot.content)) {
          focusedSlot.value = nextSlot.content
          nextTick(() => {
            slotInputRefs.value[nextSlot.content]?.focus()
          })
          return
        }
      }
    }
  } else if (e.key === 'Tab' && !e.shiftKey) {
    e.preventDefault()
    // Move to next slot
    const currentIdx = getSlotIndex(slotLabel)
    const nextIdx = (currentIdx + 1) % slots.value.length
    const nextSlot = slots.value[nextIdx]
    focusedSlot.value = nextSlot.content
    nextTick(() => {
      slotInputRefs.value[nextSlot.content]?.focus()
    })
  } else if (e.key === 'Tab' && e.shiftKey) {
    e.preventDefault()
    // Move to previous slot
    const currentIdx = getSlotIndex(slotLabel)
    const prevIdx = currentIdx <= 0 ? slots.value.length - 1 : currentIdx - 1
    const prevSlot = slots.value[prevIdx]
    focusedSlot.value = prevSlot.content
    nextTick(() => {
      slotInputRefs.value[prevSlot.content]?.focus()
    })
  }
}

// Handle use case selection
function handleUseCaseSelect(useCase: UseCaseKey) {
  selectedUseCase.value = useCase
}

// Handle skip
function handleSkip() {
  emit('skip')
}

// Check if submit is possible
const canSubmit = computed(() => {
  return allSlotsFilled.value
})

</script>

<template>
  <div :class="['copilot-welcome-panel', { compact }]">
    <!-- Header -->
    <div class="panel-header">
      <div class="header-icon">
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="20"
          height="20"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <path d="M12 2L2 7l10 5 10-5-10-5z" />
          <path d="M2 17l10 5 10-5" />
          <path d="M2 12l10 5 10-5" />
        </svg>
      </div>
      <h3 class="header-title">{{ t('welcomeDialog.title') }}</h3>
      <p class="header-subtitle">{{ t('welcomeDialog.subtitle') }}</p>
    </div>

    <!-- Prompt Section -->
    <div class="prompt-section">
      <!-- Use Case Button Group -->
      <div class="usecase-buttons">
        <button
          v-for="option in useCaseOptions"
          :key="option.key"
          :class="['usecase-btn', { active: selectedUseCase === option.key }]"
          @click="handleUseCaseSelect(option.key)"
        >
          {{ option.label }}
        </button>
      </div>

      <!-- Template Builder -->
      <div v-if="selectedUseCase !== 'freeform' && hasSlots" class="template-builder">
        <!-- Template Display -->
        <div class="template-display">
          <template v-for="(part, idx) in templateParts" :key="idx">
            <!-- Fixed Text -->
            <span v-if="part.type === 'text'" class="template-text">
              {{ part.content }}
            </span>

            <!-- Slot: Always show as input -->
            <input
              v-else
              :ref="(el) => setSlotInputRef(el as HTMLInputElement, part.content)"
              type="text"
              :class="[
                'slot-input',
                {
                  'slot-input--filled': isSlotFilled(part.content),
                  'slot-input--focused': focusedSlot === part.content,
                },
              ]"
              :style="{ width: getSlotInputWidth(part.content) }"
              :value="getSlotValue(part.content) || ''"
              :placeholder="focusedSlot === part.content ? '' : part.content"
              @input="handleSlotInput(part.content, ($event.target as HTMLInputElement).value)"
              @focus="handleSlotFocus(part.content)"
              @blur="handleSlotBlur"
              @keydown="handleSlotKeydown($event, part.content)"
            >
          </template>
        </div>

        <!-- Examples for Active Slot -->
        <div v-if="activeSlotForExamples && activeSlotExamples.length > 0" class="examples-section">
          <div class="examples-list">
            <button
              v-for="(example, idx) in activeSlotExamples"
              :key="idx"
              class="example-chip"
              @mousedown.prevent
              @click="handleExampleClick(example)"
            >
              {{ example }}
            </button>
          </div>
        </div>
      </div>

      <!-- Freeform Mode: show message -->
      <div v-else-if="selectedUseCase === 'freeform'" class="freeform-hint">
        <p>{{ t('welcomeDialog.freeformHint') }}</p>
      </div>

      <!-- Submit Button -->
      <button class="submit-btn" :disabled="!canSubmit" @click="handleSubmit">
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="18"
          height="18"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <line x1="22" y1="2" x2="11" y2="13" />
          <polygon points="22 2 15 22 11 13 2 9 22 2" />
        </svg>
        {{ t('welcomeDialog.createWithCopilot') }}
      </button>
    </div>

    <!-- Skip Button (only in WelcomeDialog) -->
    <template v-if="showSkipButton">
      <div class="divider">
        <span>{{ t('welcomeDialog.or') }}</span>
      </div>

      <button class="skip-btn" @click="handleSkip">
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="16"
          height="16"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <rect x="3" y="3" width="18" height="18" rx="2" ry="2" />
          <line x1="9" y1="9" x2="15" y2="15" />
          <line x1="15" y1="9" x2="9" y2="15" />
        </svg>
        {{ t('welcomeDialog.skipToCanvas') }}
      </button>
    </template>
  </div>
</template>

<style scoped>
.copilot-welcome-panel {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

/* Header */
.panel-header {
  text-align: center;
}

.header-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  background: linear-gradient(135deg, var(--color-primary) 0%, #8b5cf6 100%);
  border-radius: 12px;
  color: white;
  margin-bottom: 0.75rem;
}

.compact .header-icon {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  margin-bottom: 0.5rem;
}

.header-title {
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--color-text);
  margin: 0 0 0.5rem;
}

.compact .header-title {
  font-size: 1rem;
  margin-bottom: 0.25rem;
}

.header-subtitle {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
  margin: 0;
  line-height: 1.5;
}

.compact .header-subtitle {
  font-size: 0.75rem;
  line-height: 1.4;
}

/* Prompt Section */
.prompt-section {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.compact .prompt-section {
  gap: 0.75rem;
}

/* Use Case Button Group */
.usecase-buttons {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.usecase-btn {
  padding: 0.5rem 0.875rem;
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--color-text-secondary);
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: 20px;
  cursor: pointer;
  transition: all 0.15s;
  white-space: nowrap;
}

.compact .usecase-btn {
  padding: 0.375rem 0.625rem;
  font-size: 0.75rem;
}

.usecase-btn:hover {
  border-color: var(--color-primary);
  color: var(--color-text);
  background: white;
}

.usecase-btn.active {
  background: var(--color-primary);
  border-color: var(--color-primary);
  color: white;
}

/* Template Builder */
.template-builder {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.template-display {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 0.25rem;
  padding: 1rem;
  background: var(--color-background);
  border-radius: 12px;
  font-size: 0.9375rem;
  line-height: 2.2;
}

.compact .template-display {
  padding: 0.75rem;
  font-size: 0.8125rem;
  line-height: 2;
}

.template-text {
  color: var(--color-text);
  white-space: pre;
}

/* Slot Input - unified input style for all states */
.slot-input {
  padding: 0.375rem 0.625rem;
  font-size: inherit;
  font-weight: 500;
  font-family: inherit;
  color: var(--color-primary);
  background: white;
  border: 1.5px dashed var(--color-primary);
  border-radius: 6px;
  outline: none;
  box-sizing: border-box;
  transition:
    border 0.15s,
    box-shadow 0.15s,
    color 0.15s,
    width 0.1s;
}

.compact .slot-input {
  padding: 0.125rem 0.375rem;
}

/* Filled: solid border */
.slot-input--filled {
  color: var(--color-text);
  border: 1.5px solid var(--color-primary);
}

/* Focused: solid primary border with shadow */
.slot-input--focused {
  border: 1.5px solid var(--color-primary);
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.15);
}

.slot-input::placeholder {
  color: var(--color-text-tertiary);
  font-weight: 400;
}

/* Examples Section */
.examples-section {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.examples-list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.375rem;
}

.example-chip {
  padding: 0.375rem 0.625rem;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  background: white;
  border: 1px solid var(--color-border);
  border-radius: 16px;
  cursor: pointer;
  transition: all 0.15s;
  text-align: left;
}

.example-chip:hover {
  background: var(--color-primary);
  border-color: var(--color-primary);
  color: white;
}

/* Freeform Hint */
.freeform-hint {
  padding: 1rem;
  background: var(--color-background);
  border-radius: 12px;
  text-align: center;
}

.freeform-hint p {
  margin: 0;
  font-size: 0.875rem;
  color: var(--color-text-secondary);
}

/* Submit Button */
.submit-btn {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 0.875rem 1.5rem;
  font-size: 0.9375rem;
  font-weight: 600;
  color: white;
  background: linear-gradient(135deg, var(--color-primary) 0%, #8b5cf6 100%);
  border: none;
  border-radius: 10px;
  cursor: pointer;
  transition:
    transform 0.15s,
    box-shadow 0.15s,
    opacity 0.15s;
}

.compact .submit-btn {
  padding: 0.625rem 1rem;
  font-size: 0.8125rem;
  border-radius: 8px;
}

.submit-btn:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.4);
}

.submit-btn:active:not(:disabled) {
  transform: translateY(0);
}

.submit-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Divider */
.divider {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin: 0.25rem 0;
}

.divider::before,
.divider::after {
  content: '';
  flex: 1;
  height: 1px;
  background: var(--color-border);
}

.divider span {
  font-size: 0.75rem;
  color: var(--color-text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

/* Skip Button */
.skip-btn {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 0.75rem 1rem;
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--color-text-secondary);
  background: transparent;
  border: 1px solid var(--color-border);
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.15s;
}

.skip-btn:hover {
  background: var(--color-background);
  border-color: var(--color-text-secondary);
  color: var(--color-text);
}
</style>
