<script setup lang="ts">
/**
 * VariablesPanel.vue
 * 変数管理スライドアウトパネル
 *
 * ProjectVariablesTab をラップして表示
 */

defineProps<{
  projectId: string
  variables: Record<string, unknown>
  readonly?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:variables', variables: Record<string, unknown>): void
}>()
</script>

<template>
  <div class="variables-panel">
    <ProjectVariablesTab
      :project-id="projectId"
      :variables="variables"
      :readonly="readonly"
      @update:variables="emit('update:variables', $event)"
    />
  </div>
</template>

<style scoped>
.variables-panel {
  height: 100%;
  overflow: hidden;
}

.variables-panel :deep(.variables-tab) {
  height: 100%;
  overflow-y: auto;
}
</style>
