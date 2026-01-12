<script setup lang="ts">
interface Props {
  show: boolean
  title?: string
  size?: 'sm' | 'md' | 'lg' | 'xl'
}

const props = withDefaults(defineProps<Props>(), {
  title: '',
  size: 'md',
})

const emit = defineEmits<{
  close: []
}>()

function handleBackdropClick(e: Event) {
  if (e.target === e.currentTarget) {
    emit('close')
  }
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    emit('close')
  }
}

onMounted(() => {
  document.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown)
})

const sizeClass = computed(() => {
  switch (props.size) {
    case 'sm':
      return 'max-w-sm'
    case 'lg':
      return 'max-w-2xl'
    case 'xl':
      return 'max-w-4xl'
    default:
      return 'max-w-lg'
  }
})
</script>

<template>
  <ClientOnly>
    <Teleport to="body">
      <Transition name="ui-modal">
        <div
          v-if="show"
          class="ui-modal-backdrop"
          @click="handleBackdropClick"
        >
          <div :class="['ui-modal-content', sizeClass]">
            <div v-if="title" class="ui-modal-header">
              <h2 class="ui-modal-title">{{ title }}</h2>
              <button class="ui-modal-close" @click="emit('close')">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M18 6L6 18M6 6l12 12" />
                </svg>
              </button>
            </div>
            <div class="ui-modal-body">
              <slot />
            </div>
            <div v-if="$slots.footer" class="ui-modal-footer">
              <slot name="footer" />
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>
  </ClientOnly>
</template>

<style>
/* Note: Not using 'scoped' because Teleport moves content outside component tree */
.ui-modal-backdrop {
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
  padding: 1rem;
}

.ui-modal-content {
  background: var(--color-surface);
  border-radius: 0.5rem;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
  width: 100%;
  max-height: 90vh;
  overflow-y: auto;
}

.ui-modal-content.max-w-sm {
  max-width: 24rem;
}

.ui-modal-content.max-w-lg {
  max-width: 32rem;
}

.ui-modal-content.max-w-2xl {
  max-width: 42rem;
}

.ui-modal-content.max-w-4xl {
  max-width: 56rem;
}

.ui-modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem 1.5rem;
  border-bottom: 1px solid var(--color-border);
}

.ui-modal-title {
  font-size: 1.125rem;
  font-weight: 600;
  margin: 0;
}

.ui-modal-close {
  background: none;
  border: none;
  color: var(--color-text-secondary);
  cursor: pointer;
  padding: 0.25rem;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 0.25rem;
}

.ui-modal-close:hover {
  color: var(--color-text);
  background: var(--color-bg-hover);
}

.ui-modal-body {
  padding: 1.5rem;
}

.ui-modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  padding: 1rem 1.5rem;
  border-top: 1px solid var(--color-border);
}

/* Transitions */
.ui-modal-enter-active,
.ui-modal-leave-active {
  transition: opacity 0.2s ease;
}

.ui-modal-enter-active .ui-modal-content,
.ui-modal-leave-active .ui-modal-content {
  transition: transform 0.2s ease;
}

.ui-modal-enter-from,
.ui-modal-leave-to {
  opacity: 0;
}

.ui-modal-enter-from .ui-modal-content,
.ui-modal-leave-to .ui-modal-content {
  transform: scale(0.95);
}
</style>
