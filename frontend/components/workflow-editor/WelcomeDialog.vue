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
  selectTemplate: [templateId: string]
}>()

function handleSubmit(prompt: string) {
  emit('submit', prompt)
}

function handleSkip() {
  emit('skipToCanvas')
}

function handleTemplateSelect(templateId: string) {
  emit('selectTemplate', templateId)
}
</script>

<template>
  <Teleport to="body">
    <Transition name="dialog">
      <div v-if="show" class="welcome-dialog-overlay" @click.self="handleSkip">
        <div class="welcome-dialog">
          <CopilotWelcomePanel
            show-skip-button
            @submit="handleSubmit"
            @select-template="handleTemplateSelect"
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
  width: 100%;
  max-width: 580px;
  background: white;
  border-radius: 16px;
  padding: 2rem;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.2);
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
