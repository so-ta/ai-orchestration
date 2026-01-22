<script setup lang="ts">
/**
 * WelcomeDialog - Copilot-first onboarding dialog
 *
 * Shows when a new project is created, prompting users to describe
 * what they want to automate. The Copilot sidebar remains closed until
 * the user submits their prompt.
 */

const props = defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  close: []
  submit: [prompt: string]
  skipToCanvas: []
}>()

const { t } = useI18n()

// Local state
const prompt = ref('')
const inputRef = ref<HTMLTextAreaElement | null>(null)

// Example prompts for inspiration
const examplePrompts = computed(() => [
  t('welcomeDialog.examples.slackNotification'),
  t('welcomeDialog.examples.apiToNotion'),
  t('welcomeDialog.examples.githubToSlack'),
])

// Focus input when dialog opens
watch(() => props.show, (show) => {
  if (show) {
    nextTick(() => {
      inputRef.value?.focus()
    })
  }
})

// Handle submit
function handleSubmit() {
  const trimmed = prompt.value.trim()
  if (!trimmed) return

  emit('submit', trimmed)
  prompt.value = ''
}

// Handle skip
function handleSkip() {
  emit('skipToCanvas')
  prompt.value = ''
}

// Handle example click
function handleExampleClick(example: string) {
  prompt.value = example
  inputRef.value?.focus()
}

// Handle keyboard
function handleKeydown(e: KeyboardEvent) {
  // Submit on Cmd/Ctrl + Enter
  if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
    e.preventDefault()
    handleSubmit()
  }
  // Close on Escape
  if (e.key === 'Escape') {
    handleSkip()
  }
}
</script>

<template>
  <Teleport to="body">
    <Transition name="dialog">
      <div v-if="show" class="welcome-dialog-overlay" @click.self="handleSkip">
        <div class="welcome-dialog">
          <!-- Header -->
          <div class="dialog-header">
            <div class="header-icon">
              <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M12 2L2 7l10 5 10-5-10-5z"/>
                <path d="M2 17l10 5 10-5"/>
                <path d="M2 12l10 5 10-5"/>
              </svg>
            </div>
            <h2 class="dialog-title">{{ t('welcomeDialog.title') }}</h2>
            <p class="dialog-subtitle">{{ t('welcomeDialog.subtitle') }}</p>
          </div>

          <!-- Input Section -->
          <div class="input-section">
            <textarea
              ref="inputRef"
              v-model="prompt"
              class="prompt-input"
              :placeholder="t('welcomeDialog.placeholder')"
              rows="3"
              @keydown="handleKeydown"
            />
            <button
              class="submit-btn"
              :disabled="!prompt.trim()"
              @click="handleSubmit"
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="22" y1="2" x2="11" y2="13"/>
                <polygon points="22 2 15 22 11 13 2 9 22 2"/>
              </svg>
              {{ t('welcomeDialog.createWithCopilot') }}
            </button>
          </div>

          <!-- Examples Section -->
          <div class="examples-section">
            <p class="examples-label">{{ t('welcomeDialog.examplesLabel') }}</p>
            <div class="examples-list">
              <button
                v-for="(example, idx) in examplePrompts"
                :key="idx"
                class="example-chip"
                @click="handleExampleClick(example)"
              >
                {{ example }}
              </button>
            </div>
          </div>

          <!-- Divider -->
          <div class="divider">
            <span>{{ t('welcomeDialog.or') }}</span>
          </div>

          <!-- Skip Button -->
          <button class="skip-btn" @click="handleSkip">
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
              <line x1="9" y1="9" x2="15" y2="15"/>
              <line x1="15" y1="9" x2="9" y2="15"/>
            </svg>
            {{ t('welcomeDialog.skipToCanvas') }}
          </button>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.welcome-dialog-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 1rem;
}

.welcome-dialog {
  width: 100%;
  max-width: 520px;
  background: white;
  border-radius: 16px;
  padding: 2rem;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.2);
}

.dialog-header {
  text-align: center;
  margin-bottom: 1.5rem;
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
  margin-bottom: 1rem;
}

.dialog-title {
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--color-text);
  margin: 0 0 0.5rem;
}

.dialog-subtitle {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
  margin: 0;
  line-height: 1.5;
}

.input-section {
  margin-bottom: 1.25rem;
}

.prompt-input {
  width: 100%;
  padding: 0.875rem 1rem;
  font-size: 0.9375rem;
  font-family: inherit;
  line-height: 1.5;
  border: 2px solid var(--color-border);
  border-radius: 12px;
  resize: none;
  transition: border-color 0.15s, box-shadow 0.15s;
  margin-bottom: 0.75rem;
}

.prompt-input:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.prompt-input::placeholder {
  color: var(--color-text-tertiary);
}

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
  transition: transform 0.15s, box-shadow 0.15s, opacity 0.15s;
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

.examples-section {
  margin-bottom: 1.25rem;
}

.examples-label {
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--color-text-secondary);
  margin: 0 0 0.625rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.examples-list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.example-chip {
  padding: 0.5rem 0.875rem;
  font-size: 0.8125rem;
  color: var(--color-text-secondary);
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: 20px;
  cursor: pointer;
  transition: all 0.15s;
  text-align: left;
}

.example-chip:hover {
  background: var(--color-surface);
  border-color: var(--color-primary);
  color: var(--color-primary);
}

.divider {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin-bottom: 1rem;
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

/* Dialog Transition */
.dialog-enter-active,
.dialog-leave-active {
  transition: opacity 0.2s ease;
}

.dialog-enter-active .welcome-dialog,
.dialog-leave-active .welcome-dialog {
  transition: transform 0.2s ease, opacity 0.2s ease;
}

.dialog-enter-from,
.dialog-leave-to {
  opacity: 0;
}

.dialog-enter-from .welcome-dialog,
.dialog-leave-to .welcome-dialog {
  transform: scale(0.95) translateY(10px);
  opacity: 0;
}
</style>
