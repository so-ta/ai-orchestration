<script setup lang="ts">
import { computed } from 'vue'
import { getIconComponent } from '~/composables/useBlockIcons'

const props = defineProps<{
  icon: string
  color: string
  size?: number
}>()

const IconComponent = computed(() => getIconComponent(props.icon))

const iconSize = computed(() => props.size || 24)

// Generate a lighter background color from the main color
const backgroundColor = computed(() => {
  // Convert hex to RGB and create a very light version
  const hex = props.color.replace('#', '')
  const r = parseInt(hex.substring(0, 2), 16)
  const g = parseInt(hex.substring(2, 4), 16)
  const b = parseInt(hex.substring(4, 6), 16)
  // Create a light version (mix with white at 90%)
  const lightR = Math.round(r + (255 - r) * 0.9)
  const lightG = Math.round(g + (255 - g) * 0.9)
  const lightB = Math.round(b + (255 - b) * 0.9)
  return `rgb(${lightR}, ${lightG}, ${lightB})`
})
</script>

<template>
  <div
    class="node-icon-wrapper"
    :style="{
      '--icon-color': color,
      '--icon-bg': backgroundColor,
    }"
  >
    <component :is="IconComponent" :size="iconSize" stroke-width="2" />
  </div>
</template>

<style scoped>
.node-icon-wrapper {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  background: var(--icon-bg, #f8fafc);
  border: 2px solid var(--icon-color, #64748b);
  border-radius: 10px;
  color: var(--icon-color, #64748b);
  transition: border-color 0.15s, box-shadow 0.15s;
}
</style>
