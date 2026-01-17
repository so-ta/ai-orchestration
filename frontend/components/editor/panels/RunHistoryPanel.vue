<script setup lang="ts">
/**
 * RunHistoryPanel.vue
 * 実行履歴スライドアウトパネル
 *
 * WorkflowRunHistory をラップして表示
 */

defineProps<{
  projectId: string
}>()
</script>

<template>
  <div class="run-history-panel">
    <WorkflowRunHistory :workflow-id="projectId" />
  </div>
</template>

<style scoped>
.run-history-panel {
  height: 100%;
  overflow: hidden;
}

.run-history-panel :deep(.run-history) {
  height: 100%;
}
</style>
