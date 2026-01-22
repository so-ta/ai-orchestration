<script setup lang="ts">
/**
 * CopilotWelcomePanel - Unified Copilot onboarding panel
 *
 * Shared component for WelcomeDialog and CopilotTab empty state.
 * Contains header, tab switcher, templates, and structured prompt input.
 * Focus on guiding user input with inline editing for slots.
 */
import { useSnippetInput } from '~/composables/useSnippetInput'

defineProps<{
  /** Compact mode for smaller spaces like Copilot sidebar */
  compact?: boolean
  /** Show skip to canvas button */
  showSkipButton?: boolean
}>()

const emit = defineEmits<{
  submit: [prompt: string]
  selectTemplate: [templateId: string]
  skip: []
}>()

const { t, locale } = useI18n()

// Tab state
const activeTab = ref<'prompt' | 'templates'>('prompt')

// Use case keys
const useCaseKeys = ['periodic', 'event', 'analysis', 'sync', 'aiGenerate', 'freeform'] as const
type UseCaseKey = (typeof useCaseKeys)[number]

// Use case state
const selectedUseCase = ref<UseCaseKey>('periodic')
const slotInputValue = ref('')
const slotInputRef = ref<HTMLInputElement | HTMLInputElement[] | null>(null)

// Snippet input composable
const {
  templateParts,
  slots,
  currentSlotLabel,
  allSlotsFilled,
  hasSlots,
  finalPrompt,
  setTemplate,
  selectSlot,
  fillCurrentSlot,
  getSlotValue,
  isSlotFilled,
  getSlotIndex,
  selectNextSlot,
} = useSnippetInput()

// Pre-defined templates for quick start
const templates = computed(() => [
  {
    id: 'webhook-notification',
    icon: '&#128276;',
    title: t('welcomeDialog.templates.webhookNotification.title'),
    description: t('welcomeDialog.templates.webhookNotification.description'),
  },
  {
    id: 'scheduled-report',
    icon: '&#128197;',
    title: t('welcomeDialog.templates.scheduledReport.title'),
    description: t('welcomeDialog.templates.scheduledReport.description'),
  },
  {
    id: 'api-integration',
    icon: '&#128640;',
    title: t('welcomeDialog.templates.apiIntegration.title'),
    description: t('welcomeDialog.templates.apiIntegration.description'),
  },
  {
    id: 'llm-assistant',
    icon: '&#129302;',
    title: t('welcomeDialog.templates.llmAssistant.title'),
    description: t('welcomeDialog.templates.llmAssistant.description'),
  },
])

// Use case options for dropdown
const useCaseOptions = computed(() =>
  useCaseKeys.map((key) => ({
    key,
    label: t(`welcomeDialog.useCases.${key}.label`),
  }))
)

// Get examples for current slot
const currentExamples = computed((): string[] => {
  if (selectedUseCase.value === 'freeform') return []
  if (!currentSlotLabel.value) return []

  const examplesObj = t(`welcomeDialog.useCases.${selectedUseCase.value}.examples`)
  if (typeof examplesObj !== 'object' || examplesObj === null) return []

  const examples = (examplesObj as Record<string, string[]>)[currentSlotLabel.value]
  return Array.isArray(examples) ? examples : []
})

// Watch for use case changes
watch(
  selectedUseCase,
  (newCase) => {
    if (newCase === 'freeform') {
      setTemplate('')
    } else {
      const template = t(`welcomeDialog.useCases.${newCase}.template`) || ''
      setTemplate(template)
    }
    slotInputValue.value = ''
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

// Focus input when slot changes
watch(currentSlotLabel, () => {
  slotInputValue.value = ''
  nextTick(() => {
    const inputEl = Array.isArray(slotInputRef.value) ? slotInputRef.value[0] : slotInputRef.value
    inputEl?.focus()
  })
})

// Handle submit
function handleSubmit() {
  const prompt = finalPrompt.value.trim()
  if (!prompt) return

  emit('submit', prompt)
  setTemplate('')
  selectedUseCase.value = 'periodic'
  slotInputValue.value = ''
}

// Flag to prevent blur from interfering with example click
const isClickingExample = ref(false)

// Handle example click - fill slot and move to next
function handleExampleClick(example: string) {
  isClickingExample.value = true
  fillCurrentSlot(example)
  slotInputValue.value = ''
  nextTick(() => {
    isClickingExample.value = false
    getInputEl()?.focus()
  })
}

// Handle input blur - save value if not empty
function handleInputBlur() {
  // Don't process blur if clicking example
  if (isClickingExample.value) return

  if (slotInputValue.value.trim()) {
    fillCurrentSlot(slotInputValue.value.trim())
    slotInputValue.value = ''
  }
}

// Handle slot click - select slot and focus input
function handleSlotClick(label: string) {
  const index = getSlotIndex(label)
  if (index >= 0) {
    selectSlot(index)
    slotInputValue.value = getSlotValue(label) || ''
    nextTick(() => {
      // slotInputRef is an array in v-for, get first element
      const inputEl = Array.isArray(slotInputRef.value) ? slotInputRef.value[0] : slotInputRef.value
      inputEl?.focus()
      inputEl?.select()
    })
  }
}

// Helper to get input element from ref
function getInputEl(): HTMLInputElement | null {
  return Array.isArray(slotInputRef.value) ? slotInputRef.value[0] : slotInputRef.value
}

// Handle input keydown
function handleInputKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && slotInputValue.value.trim()) {
    e.preventDefault()
    fillCurrentSlot(slotInputValue.value.trim())
    slotInputValue.value = ''
    // If all slots filled, submit
    nextTick(() => {
      if (allSlotsFilled.value) {
        handleSubmit()
      } else {
        getInputEl()?.focus()
      }
    })
  } else if (e.key === 'Tab' && !e.shiftKey) {
    e.preventDefault()
    if (slotInputValue.value.trim()) {
      fillCurrentSlot(slotInputValue.value.trim())
      slotInputValue.value = ''
    }
    selectNextSlot()
    nextTick(() => getInputEl()?.focus())
  } else if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
    e.preventDefault()
    if (slotInputValue.value.trim()) {
      fillCurrentSlot(slotInputValue.value.trim())
    }
    nextTick(() => handleSubmit())
  }
}

// Handle use case selection
function handleUseCaseSelect(useCase: UseCaseKey) {
  selectedUseCase.value = useCase
}

// Handle template selection
function handleTemplateSelect(templateId: string) {
  emit('selectTemplate', templateId)
}

// Handle skip
function handleSkip() {
  emit('skip')
}

// Check if submit is possible
const canSubmit = computed(() => {
  return allSlotsFilled.value
})

// Progress indicator
const progress = computed(() => {
  if (!hasSlots.value) return { current: 0, total: 0 }
  const filled = slots.value.filter((s) => isSlotFilled(s.content)).length
  return { current: filled, total: slots.value.length }
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

    <!-- Tab Switcher -->
    <div class="tab-switcher">
      <button :class="['tab-btn', { active: activeTab === 'prompt' }]" @click="activeTab = 'prompt'">
        {{ t('welcomeDialog.tabs.describe') }}
      </button>
      <button :class="['tab-btn', { active: activeTab === 'templates' }]" @click="activeTab = 'templates'">
        {{ t('welcomeDialog.tabs.templates') }}
      </button>
    </div>

    <!-- Templates Section -->
    <div v-if="activeTab === 'templates'" class="templates-section">
      <div :class="['templates-grid', { compact }]">
        <button
          v-for="template in templates"
          :key="template.id"
          class="template-card"
          @click="handleTemplateSelect(template.id)"
        >
          <span class="template-icon" v-html="template.icon" />
          <div class="template-content">
            <span class="template-title">{{ template.title }}</span>
            <span class="template-description">{{ template.description }}</span>
          </div>
        </button>
      </div>
    </div>

    <!-- Prompt Section -->
    <div v-if="activeTab === 'prompt'" class="prompt-section">
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
        <!-- Progress Indicator -->
        <div class="progress-bar">
          <div class="progress-fill" :style="{ width: `${(progress.current / progress.total) * 100}%` }" />
        </div>

        <!-- Template Display -->
        <div class="template-display">
          <template v-for="(part, idx) in templateParts" :key="idx">
            <!-- Fixed Text -->
            <span v-if="part.type === 'text'" class="template-text">
              {{ part.content }}
            </span>

            <!-- Slot: Input when active, Chip when inactive -->
            <template v-else>
              <!-- Active Slot: Show Input -->
              <input
                v-if="currentSlotLabel === part.content"
                ref="slotInputRef"
                v-model="slotInputValue"
                type="text"
                class="slot-inline-input"
                :placeholder="part.content"
                @keydown="handleInputKeydown"
                @blur="handleInputBlur"
              >
              <!-- Inactive Slot: Show Chip -->
              <button
                v-else
                :class="['slot-chip', { 'slot-filled': isSlotFilled(part.content) }]"
                @click="handleSlotClick(part.content)"
              >
                <span class="slot-value">
                  {{ getSlotValue(part.content) || part.content }}
                </span>
              </button>
            </template>
          </template>
        </div>

        <!-- Examples for Active Slot -->
        <div v-if="currentSlotLabel && currentExamples.length > 0" class="examples-section">
          <div class="examples-list">
            <button
              v-for="(example, idx) in currentExamples"
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

/* Tab Switcher */
.tab-switcher {
  display: flex;
  gap: 0.5rem;
  padding: 4px;
  background: var(--color-background);
  border-radius: 10px;
}

.compact .tab-switcher {
  gap: 0.375rem;
  padding: 3px;
  border-radius: 8px;
}

.tab-btn {
  flex: 1;
  padding: 0.625rem 1rem;
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--color-text-secondary);
  background: transparent;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s;
}

.compact .tab-btn {
  padding: 0.5rem 0.75rem;
  font-size: 0.75rem;
  border-radius: 6px;
}

.tab-btn:hover {
  color: var(--color-text);
}

.tab-btn.active {
  background: white;
  color: var(--color-text);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.compact .tab-btn.active {
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.08);
}

/* Templates Section */
.templates-section {
  margin-bottom: 0.25rem;
}

.templates-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 0.75rem;
}

.templates-grid.compact {
  grid-template-columns: 1fr;
  gap: 0.5rem;
}

.template-card {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
  padding: 1rem;
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: 12px;
  cursor: pointer;
  text-align: left;
  transition: all 0.15s;
}

.compact .template-card {
  gap: 0.625rem;
  padding: 0.75rem;
  border-radius: 10px;
}

.template-card:hover {
  border-color: var(--color-primary);
  background: white;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.05);
}

.compact .template-card:hover {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.template-icon {
  font-size: 1.5rem;
  line-height: 1;
}

.compact .template-icon {
  font-size: 1.25rem;
}

.template-content {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  min-width: 0;
}

.compact .template-content {
  gap: 0.125rem;
}

.template-title {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--color-text);
}

.compact .template-title {
  font-size: 0.8125rem;
}

.template-description {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  line-height: 1.4;
}

.compact .template-description {
  font-size: 0.6875rem;
  line-height: 1.3;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
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

.progress-bar {
  height: 4px;
  background: var(--color-border);
  border-radius: 2px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, var(--color-primary), #8b5cf6);
  border-radius: 2px;
  transition: width 0.3s ease;
}

.template-display {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.25rem;
  padding: 1rem;
  background: var(--color-background);
  border-radius: 12px;
  font-size: 0.9375rem;
  line-height: 2;
}

.compact .template-display {
  padding: 0.75rem;
  font-size: 0.8125rem;
  line-height: 1.8;
}

.template-text {
  color: var(--color-text);
  white-space: pre;
}

.slot-chip {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  padding: 0.375rem 0.75rem;
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--color-primary);
  background: white;
  border: 2px dashed var(--color-primary);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s;
}

.compact .slot-chip {
  padding: 0.25rem 0.5rem;
  font-size: 0.75rem;
}

.slot-chip:hover {
  background: rgba(59, 130, 246, 0.05);
}

.slot-chip.slot-filled {
  color: var(--color-text);
  background: white;
  border: 2px solid var(--color-success, #10b981);
}

.slot-value {
  max-width: 150px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.compact .slot-value {
  max-width: 100px;
}

/* Inline Slot Input - matches slot-chip size */
.slot-inline-input {
  padding: 0.375rem 0.75rem;
  font-size: 0.875rem;
  font-weight: 500;
  font-family: inherit;
  color: var(--color-primary);
  background: white;
  border: 2px solid var(--color-primary);
  border-radius: 8px;
  outline: none;
  min-width: 80px;
  max-width: 200px;
  box-sizing: border-box;
  transition: box-shadow 0.15s;
}

.compact .slot-inline-input {
  padding: 0.25rem 0.5rem;
  font-size: 0.75rem;
  min-width: 60px;
  max-width: 150px;
}

.slot-inline-input:focus {
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.15);
}

.slot-inline-input::placeholder {
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
