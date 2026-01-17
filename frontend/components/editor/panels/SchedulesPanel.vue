<script setup lang="ts">
/**
 * SchedulesPanel.vue
 * スケジュール管理スライドアウトパネル
 *
 * ProjectSchedulesTab をラップして表示
 */

import type { Step } from '~/types/api'

defineProps<{
  projectId: string
  steps: Step[]
}>()
</script>

<template>
  <div class="schedules-panel">
    <ProjectSchedulesTab
      :project-id="projectId"
      :steps="steps"
    />
  </div>
</template>

<style scoped>
.schedules-panel {
  height: 100%;
  overflow: hidden;
}

.schedules-panel :deep(.schedules-tab) {
  height: 100%;
  overflow-y: auto;
}
</style>
