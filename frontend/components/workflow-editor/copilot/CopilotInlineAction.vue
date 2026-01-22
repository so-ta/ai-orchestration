<script setup lang="ts">
/**
 * CopilotInlineAction.vue
 *
 * Parent component for inline actions within chat messages.
 * Renders the appropriate child component based on action type.
 */
import type {
  InlineAction,
  InlineActionResult,
  ConfirmAction,
  SelectAction,
  FormAction,
  OAuthAction,
  TestAction,
} from './types'
import CopilotOptionCard from './CopilotOptionCard.vue'
import CopilotInlineForm from './CopilotInlineForm.vue'
import CopilotOAuthButton from './CopilotOAuthButton.vue'

const { t } = useI18n()

defineProps<{
  action: InlineAction
  disabled?: boolean
}>()

const emit = defineEmits<{
  'submit': [result: InlineActionResult]
}>()

// Type guards
function isConfirmAction(action: InlineAction): action is ConfirmAction {
  return action.type === 'confirm'
}

function isSelectAction(action: InlineAction): action is SelectAction {
  return action.type === 'select'
}

function isFormAction(action: InlineAction): action is FormAction {
  return action.type === 'form'
}

function isOAuthAction(action: InlineAction): action is OAuthAction {
  return action.type === 'oauth'
}

function isTestAction(action: InlineAction): action is TestAction {
  return action.type === 'test'
}

// Handlers
function handleConfirm(confirmed: boolean) {
  emit('submit', { type: 'confirm', confirmed })
}

function handleSelect(selectedIds: string[]) {
  emit('submit', { type: 'select', selectedIds })
}

function handleFormSubmit(values: Record<string, string | number>) {
  emit('submit', { type: 'form', values })
}

function handleOAuth(credentialId: string, credentialName: string) {
  emit('submit', { type: 'oauth', credentialId, credentialName })
}

function handleTest(skipped: boolean) {
  emit('submit', { type: 'test', skipped })
}
</script>

<template>
  <div class="inline-action" :class="{ disabled }">
    <!-- Title -->
    <div v-if="action.title" class="action-title">
      {{ action.title }}
    </div>

    <!-- Description -->
    <div v-if="action.description" class="action-description">
      {{ action.description }}
    </div>

    <!-- Confirm Action -->
    <div v-if="isConfirmAction(action)" class="confirm-buttons">
      <button
        class="btn-primary"
        :disabled="disabled"
        @click="handleConfirm(true)"
      >
        {{ action.confirmLabel || t('copilot.action.confirm') }}
      </button>
      <button
        class="btn-secondary"
        :disabled="disabled"
        @click="handleConfirm(false)"
      >
        {{ action.cancelLabel || t('copilot.action.cancel') }}
      </button>
    </div>

    <!-- Select Action -->
    <div v-else-if="isSelectAction(action)" class="select-options">
      <CopilotOptionCard
        v-for="option in action.options"
        :key="option.id"
        :option="option"
        :disabled="disabled"
        @select="handleSelect([option.id])"
      />
    </div>

    <!-- Form Action -->
    <CopilotInlineForm
      v-else-if="isFormAction(action)"
      :fields="action.fields"
      :submit-label="action.submitLabel"
      :disabled="disabled"
      @submit="handleFormSubmit"
    />

    <!-- OAuth Action -->
    <CopilotOAuthButton
      v-else-if="isOAuthAction(action)"
      :service="action.service"
      :service-name="action.serviceName"
      :service-icon="action.serviceIcon"
      :existing-credentials="action.existingCredentials"
      :disabled="disabled"
      @connect="handleOAuth"
    />

    <!-- Test Action -->
    <div v-else-if="isTestAction(action)" class="test-buttons">
      <button
        class="btn-primary"
        :disabled="disabled"
        @click="handleTest(false)"
      >
        {{ action.testLabel || t('copilot.action.runTest') }}
      </button>
      <button
        class="btn-ghost"
        :disabled="disabled"
        @click="handleTest(true)"
      >
        {{ action.skipLabel || t('copilot.action.skip') }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.inline-action {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 0.875rem;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 10px;
  margin-top: 0.5rem;
}

.inline-action.disabled {
  opacity: 0.6;
  pointer-events: none;
}

.action-title {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--color-text);
}

.action-description {
  font-size: 0.8125rem;
  color: var(--color-text-secondary);
  line-height: 1.5;
}

/* Button groups */
.confirm-buttons,
.test-buttons {
  display: flex;
  gap: 0.5rem;
}

/* Select options */
.select-options {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

/* Button styles */
.btn-primary,
.btn-secondary,
.btn-ghost {
  padding: 0.5rem 1rem;
  font-size: 0.8125rem;
  font-weight: 500;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
}

.btn-primary {
  background: var(--color-primary);
  color: white;
  border: none;
}

.btn-primary:hover:not(:disabled) {
  opacity: 0.9;
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-secondary {
  background: var(--color-surface);
  color: var(--color-text);
  border: 1px solid var(--color-border);
}

.btn-secondary:hover:not(:disabled) {
  background: var(--color-background);
}

.btn-secondary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-ghost {
  background: transparent;
  color: var(--color-text-secondary);
  border: none;
}

.btn-ghost:hover:not(:disabled) {
  color: var(--color-text);
  background: var(--color-background);
}

.btn-ghost:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
