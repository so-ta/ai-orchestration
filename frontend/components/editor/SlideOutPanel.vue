<script setup lang="ts">
/**
 * SlideOutPanel.vue
 * 右端からスライドインするパネルコンポーネント
 *
 * 機能:
 * - 右端からスライドイン
 * - 半透明バックドロップ（クリックで閉じる）
 * - ヘッダーにタイトルと閉じるボタン
 * - コンテンツはslotで注入
 */

const { t } = useI18n()

const props = defineProps<{
  show: boolean
  title: string
  width?: number
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const panelWidth = computed(() => props.width || 400)

function handleBackdropClick() {
  emit('close')
}

function handleClose() {
  emit('close')
}

// ESC key to close
function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && props.show) {
    emit('close')
  }
}

onMounted(() => {
  document.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown)
})
</script>

<template>
  <Teleport to="body">
    <Transition name="slideout">
      <div v-if="show" class="slideout-container">
        <!-- Backdrop -->
        <div class="slideout-backdrop" @click="handleBackdropClick" />

        <!-- Panel -->
        <aside
          class="slideout-panel"
          :style="{ width: panelWidth + 'px' }"
        >
          <!-- Header -->
          <div class="slideout-header">
            <h3 class="slideout-title">{{ title }}</h3>
            <button class="close-btn" :title="t('common.close')" @click="handleClose">
              <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="18" y1="6" x2="6" y2="18" />
                <line x1="6" y1="6" x2="18" y2="18" />
              </svg>
            </button>
          </div>

          <!-- Content -->
          <div class="slideout-content">
            <slot />
          </div>
        </aside>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.slideout-container {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 100;
  display: flex;
  justify-content: flex-end;
}

.slideout-backdrop {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.3);
}

.slideout-panel {
  position: relative;
  height: 100%;
  background: var(--color-surface);
  box-shadow: -4px 0 20px rgba(0, 0, 0, 0.15);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.slideout-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem 1.25rem;
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}

.slideout-title {
  font-size: 1rem;
  font-weight: 600;
  color: var(--color-text);
  margin: 0;
}

.close-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  padding: 0;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.15s;
}

.close-btn:hover {
  background: var(--color-bg-hover);
  color: var(--color-text);
}

.slideout-content {
  flex: 1;
  overflow-y: auto;
}

/* Transition */
.slideout-enter-active,
.slideout-leave-active {
  transition: all 0.25s ease;
}

.slideout-enter-active .slideout-backdrop,
.slideout-leave-active .slideout-backdrop {
  transition: opacity 0.25s ease;
}

.slideout-enter-active .slideout-panel,
.slideout-leave-active .slideout-panel {
  transition: transform 0.25s ease;
}

.slideout-enter-from .slideout-backdrop,
.slideout-leave-to .slideout-backdrop {
  opacity: 0;
}

.slideout-enter-from .slideout-panel,
.slideout-leave-to .slideout-panel {
  transform: translateX(100%);
}
</style>
