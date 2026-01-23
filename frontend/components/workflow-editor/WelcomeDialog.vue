<script setup lang="ts">
/**
 * WelcomeDialog - Copilot-first onboarding dialog
 *
 * Shows when a new project is created or when opening a project with
 * only one block. Wraps CopilotWelcomePanel with a modal overlay.
 */
import CopilotWelcomePanel from './CopilotWelcomePanel.vue'

defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  close: []
  submit: [prompt: string]
  skipToCanvas: []
}>()

function handleSubmit(prompt: string) {
  emit('submit', prompt)
}

function handleSkip() {
  emit('skipToCanvas')
}
</script>

<template>
  <Teleport to="body">
    <Transition name="dialog">
      <div v-if="show" class="welcome-dialog-overlay">
        <div class="welcome-dialog">
          <button class="close-button" type="button" @click="handleSkip">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
          <CopilotWelcomePanel
            show-skip-button
            auto-focus
            @submit="handleSubmit"
            @skip="handleSkip"
          />
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
  position: relative;
  width: 100%;
  max-width: 580px;
  background: white;
  border-radius: 16px;
  padding: 2rem;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.2);
}

.close-button {
  position: absolute;
  top: 1rem;
  right: 1rem;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  border-radius: 8px;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.15s;
}

.close-button:hover {
  background: var(--color-background);
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
