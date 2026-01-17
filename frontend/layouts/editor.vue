<script setup lang="ts">
/**
 * editor.vue
 * フルスクリーンエディタ用レイアウト
 *
 * 機能:
 * - ヘッダーなし（EditorHeaderはページ内で管理）
 * - パディングなし（全画面利用）
 * - ToastとConfirmDialogのみ提供
 */
</script>

<template>
  <div class="editor-layout">
    <!-- Main Content (full screen) -->
    <slot />

    <!-- Toast notifications -->
    <ToastContainer />
    <!-- Global confirmation dialog -->
    <UiConfirmDialog />
  </div>
</template>

<style scoped>
.editor-layout {
  display: flex;
  flex-direction: column;
  height: 100vh;
  width: 100vw;
  overflow: hidden;
  background: var(--color-background);
}
</style>
