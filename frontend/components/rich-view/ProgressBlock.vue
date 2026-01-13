<script setup lang="ts">
import type { ProgressConfig } from '~/types/rich-view'

const props = defineProps<{
  config: ProgressConfig
}>()

// Clamp value between 0 and 100
const clampedValue = computed(() => {
  return Math.max(0, Math.min(100, props.config.value))
})

// Get color based on config
const progressColor = computed(() => {
  if (props.config.color === 'auto') {
    // Green for high values, yellow for medium, red for low
    if (clampedValue.value >= 70) return '#10b981' // green
    if (clampedValue.value >= 40) return '#f59e0b' // amber
    return '#ef4444' // red
  }
  return props.config.color || '#3b82f6' // default blue
})

// Size classes
const sizeClass = computed(() => {
  switch (props.config.size) {
    case 'sm':
      return 'progress-sm'
    case 'lg':
      return 'progress-lg'
    default:
      return 'progress-md'
  }
})
</script>

<template>
  <div class="progress-block" :class="sizeClass">
    <div v-if="config.label" class="progress-label">
      {{ config.label }}
    </div>
    <div class="progress-bar-container">
      <div
        class="progress-bar"
        :style="{
          width: `${clampedValue}%`,
          backgroundColor: progressColor,
        }"
      />
    </div>
    <div class="progress-value">
      {{ clampedValue }}%
    </div>
  </div>
</template>

<style scoped>
.progress-block {
  margin: 1rem 0;
  padding: 0.75rem 1rem;
  background: white;
  border: 1px solid var(--color-border);
  border-radius: 8px;
}

.progress-label {
  font-weight: 500;
  margin-bottom: 0.5rem;
  color: var(--color-text);
}

.progress-bar-container {
  background: rgba(0, 0, 0, 0.05);
  border-radius: 9999px;
  overflow: hidden;
}

.progress-bar {
  transition: width 0.3s ease;
  border-radius: 9999px;
}

.progress-value {
  margin-top: 0.25rem;
  text-align: right;
  font-weight: 500;
  color: var(--color-text-secondary);
}

/* Size variants */
.progress-sm .progress-bar-container {
  height: 6px;
}

.progress-sm .progress-label {
  font-size: 0.75rem;
}

.progress-sm .progress-value {
  font-size: 0.75rem;
}

.progress-md .progress-bar-container {
  height: 10px;
}

.progress-md .progress-label {
  font-size: 0.875rem;
}

.progress-md .progress-value {
  font-size: 0.875rem;
}

.progress-lg .progress-bar-container {
  height: 16px;
}

.progress-lg .progress-label {
  font-size: 1rem;
}

.progress-lg .progress-value {
  font-size: 1rem;
}
</style>
