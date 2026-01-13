<script setup lang="ts">
const { state, handleConfirm, handleCancel } = useConfirm()

const confirmButtonClass = computed(() => {
  if (state.value.options?.variant === 'danger') {
    return 'confirm-button danger'
  }
  return 'confirm-button'
})
</script>

<template>
  <UiModal
    :show="state.show"
    :title="state.options?.title || ''"
    size="sm"
    @close="handleCancel"
  >
    <p class="confirm-message">{{ state.options?.message }}</p>
    <template #footer>
      <button
        type="button"
        class="cancel-button"
        @click="handleCancel"
      >
        {{ state.options?.cancelText }}
      </button>
      <button
        type="button"
        :class="confirmButtonClass"
        @click="handleConfirm"
      >
        {{ state.options?.confirmText }}
      </button>
    </template>
  </UiModal>
</template>

<style scoped>
.confirm-message {
  color: var(--color-text-secondary);
  margin: 0;
  line-height: 1.5;
}

.cancel-button {
  padding: 0.5rem 1rem;
  border: 1px solid var(--color-border);
  border-radius: 0.375rem;
  background: var(--color-surface);
  color: var(--color-text);
  cursor: pointer;
  font-size: 0.875rem;
}

.cancel-button:hover {
  background: var(--color-bg-hover);
}

.confirm-button {
  padding: 0.5rem 1rem;
  border: none;
  border-radius: 0.375rem;
  background: var(--color-primary);
  color: white;
  cursor: pointer;
  font-size: 0.875rem;
}

.confirm-button:hover {
  background: var(--color-primary-hover);
}

.confirm-button.danger {
  background: var(--color-error);
}

.confirm-button.danger:hover {
  background: #dc2626;
}
</style>
