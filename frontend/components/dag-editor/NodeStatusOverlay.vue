<script setup lang="ts">
import type { StepRunStatus } from '~/types/api'

const props = defineProps<{
  status?: StepRunStatus | null
  output?: unknown
}>()

// Compute output count for array/object outputs
const outputCount = computed(() => {
  if (!props.output) return null
  if (Array.isArray(props.output)) {
    return props.output.length
  }
  if (typeof props.output === 'object' && props.output !== null) {
    return Object.keys(props.output).length
  }
  return null
})
</script>

<template>
  <div v-if="status" class="node-status-overlay">
    <!-- Running: Spinner -->
    <div v-if="status === 'running'" class="status-indicator status-running">
      <svg class="spin" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
        <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
      </svg>
    </div>

    <!-- Pending: Clock -->
    <div v-else-if="status === 'pending'" class="status-indicator status-pending">
      <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
        <circle cx="12" cy="12" r="10"/>
        <polyline points="12 6 12 12 16 14"/>
      </svg>
    </div>

    <!-- Completed: Check -->
    <div v-else-if="status === 'completed'" class="status-indicator status-completed">
      <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
        <polyline points="20 6 9 17 4 12"/>
      </svg>
      <span v-if="outputCount !== null" class="output-badge">{{ outputCount }}</span>
    </div>

    <!-- Failed: X -->
    <div v-else-if="status === 'failed'" class="status-indicator status-failed">
      <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
        <line x1="18" y1="6" x2="6" y2="18"/>
        <line x1="6" y1="6" x2="18" y2="18"/>
      </svg>
    </div>

    <!-- Skipped: Skip -->
    <div v-else-if="status === 'skipped'" class="status-indicator status-skipped">
      <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
        <polygon points="5 4 15 12 5 20 5 4"/>
        <line x1="19" y1="5" x2="19" y2="19"/>
      </svg>
    </div>
  </div>
</template>

<style scoped>
.node-status-overlay {
  position: absolute;
  top: -6px;
  right: -6px;
  z-index: 10;
  pointer-events: none;
}

.status-indicator {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  border: 2px solid white;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
}

.status-running {
  background: #3b82f6;
  color: white;
}

.status-pending {
  background: #f59e0b;
  color: white;
}

.status-completed {
  background: #22c55e;
  color: white;
  position: relative;
}

.status-failed {
  background: #ef4444;
  color: white;
}

.status-skipped {
  background: #94a3b8;
  color: white;
}

.output-badge {
  position: absolute;
  top: -8px;
  right: -8px;
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  font-size: 10px;
  font-weight: 700;
  color: #22c55e;
  background: white;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
