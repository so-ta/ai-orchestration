<script setup lang="ts">
/**
 * CopilotErrorGuide.vue
 * Copilotエラー発生時のガイダンスとリトライ機能を提供
 */

const { t } = useI18n()

interface ErrorGuide {
  type: 'connection' | 'timeout' | 'credential' | 'config' | 'unknown'
  message: string
  suggestion?: string
  action?: {
    label: string
    handler: () => void
  }
}

const props = defineProps<{
  error: string
  type?: 'connection' | 'timeout' | 'credential' | 'config' | 'unknown'
}>()

const emit = defineEmits<{
  'retry': []
  'dismiss': []
  'open-credentials': []
  'open-config': []
}>()

// Determine error type from error message
const errorType = computed(() => {
  if (props.type) return props.type

  const msg = props.error.toLowerCase()
  if (msg.includes('network') || msg.includes('connection') || msg.includes('fetch')) {
    return 'connection'
  }
  if (msg.includes('timeout') || msg.includes('timed out')) {
    return 'timeout'
  }
  if (msg.includes('credential') || msg.includes('unauthorized') || msg.includes('api key')) {
    return 'credential'
  }
  if (msg.includes('config') || msg.includes('invalid') || msg.includes('required')) {
    return 'config'
  }
  return 'unknown'
})

// Get guide content based on error type
const guide = computed<ErrorGuide>(() => {
  switch (errorType.value) {
    case 'connection':
      return {
        type: 'connection',
        message: t('copilot.errors.connection'),
        suggestion: t('copilot.errors.connectionHint'),
        action: {
          label: t('copilot.errors.retry'),
          handler: () => emit('retry'),
        },
      }
    case 'timeout':
      return {
        type: 'timeout',
        message: t('copilot.errors.timeout'),
        suggestion: t('copilot.errors.timeoutHint'),
        action: {
          label: t('copilot.errors.retry'),
          handler: () => emit('retry'),
        },
      }
    case 'credential':
      return {
        type: 'credential',
        message: t('copilot.errors.credential'),
        suggestion: t('copilot.errors.credentialHint'),
        action: {
          label: t('copilot.errors.openCredentials'),
          handler: () => emit('open-credentials'),
        },
      }
    case 'config':
      return {
        type: 'config',
        message: t('copilot.errors.config'),
        suggestion: t('copilot.errors.configHint'),
        action: {
          label: t('copilot.errors.checkConfig'),
          handler: () => emit('open-config'),
        },
      }
    default:
      return {
        type: 'unknown',
        message: props.error || t('copilot.errors.unknown'),
        suggestion: t('copilot.errors.unknownHint'),
        action: {
          label: t('copilot.errors.retry'),
          handler: () => emit('retry'),
        },
      }
  }
})

// Get icon for error type
const errorIcon = computed(() => {
  switch (errorType.value) {
    case 'connection': return '&#128268;'
    case 'timeout': return '&#9200;'
    case 'credential': return '&#128274;'
    case 'config': return '&#9881;'
    default: return '&#9888;'
  }
})
</script>

<template>
  <div class="error-guide">
    <div class="error-header">
      <span class="error-icon" v-html="errorIcon" />
      <span class="error-title">{{ guide.message }}</span>
      <button class="error-dismiss" @click="emit('dismiss')" :title="t('common.dismiss')">
        &times;
      </button>
    </div>

    <p v-if="guide.suggestion" class="error-suggestion">
      {{ guide.suggestion }}
    </p>

    <div class="error-actions">
      <button v-if="guide.action" class="error-action-btn" @click="guide.action.handler">
        {{ guide.action.label }}
      </button>
      <button class="error-retry-btn" @click="emit('retry')">
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M21 2v6h-6"/>
          <path d="M3 12a9 9 0 0 1 15-6.7L21 8"/>
          <path d="M3 22v-6h6"/>
          <path d="M21 12a9 9 0 0 1-15 6.7L3 16"/>
        </svg>
        {{ t('copilot.errors.retry') }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.error-guide {
  background: #fef2f2;
  border: 1px solid #fecaca;
  border-radius: 8px;
  padding: 1rem;
  margin: 0.5rem 0;
}

.error-header {
  display: flex;
  align-items: flex-start;
  gap: 0.5rem;
  margin-bottom: 0.5rem;
}

.error-icon {
  font-size: 1.25rem;
  line-height: 1;
}

.error-title {
  flex: 1;
  font-weight: 500;
  color: #991b1b;
  font-size: 0.875rem;
}

.error-dismiss {
  background: none;
  border: none;
  font-size: 1.25rem;
  color: #dc2626;
  cursor: pointer;
  padding: 0;
  line-height: 1;
  opacity: 0.6;
}

.error-dismiss:hover {
  opacity: 1;
}

.error-suggestion {
  font-size: 0.8125rem;
  color: #7f1d1d;
  margin: 0 0 0.75rem;
  padding-left: 1.75rem;
}

.error-actions {
  display: flex;
  gap: 0.5rem;
  padding-left: 1.75rem;
}

.error-action-btn {
  padding: 0.5rem 0.875rem;
  background: #ef4444;
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 0.8125rem;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s;
}

.error-action-btn:hover {
  background: #dc2626;
}

.error-retry-btn {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.5rem 0.875rem;
  background: transparent;
  color: #dc2626;
  border: 1px solid #fecaca;
  border-radius: 6px;
  font-size: 0.8125rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
}

.error-retry-btn:hover {
  background: #fee2e2;
  border-color: #f87171;
}
</style>
