<script setup lang="ts">
defineProps<{
  readonly?: boolean
}>()
</script>

<template>
  <div class="floating-toolbar">
    <StepPalette :readonly="readonly" />
  </div>
</template>

<style scoped>
.floating-toolbar {
  position: fixed;
  top: 68px;
  left: 12px;
  bottom: 12px;
  z-index: 99;

  width: 260px;
  display: flex;
  flex-direction: column;

  background: rgba(255, 255, 255, 0.98);
  backdrop-filter: blur(12px);
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 12px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
  overflow: hidden;
}
</style>
