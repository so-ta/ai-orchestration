<script setup lang="ts">
import type { BlockDefinition } from '~/types/api'

const { t } = useI18n()

const props = defineProps<{
  block: BlockDefinition
  visible: boolean
  position?: { x: number; y: number }
}>()

// Parse config schema to get field info
const fieldInfo = computed(() => {
  if (!props.block.config_schema) {
    return { total: 0, required: 0 }
  }

  try {
    const schema = typeof props.block.config_schema === 'string'
      ? JSON.parse(props.block.config_schema)
      : props.block.config_schema

    const properties = schema.properties || {}
    const required = schema.required || []

    return {
      total: Object.keys(properties).length,
      required: required.length,
    }
  } catch {
    return { total: 0, required: 0 }
  }
})

// Format category name
const categoryLabel = computed(() => {
  const category = props.block.category
  const subcategory = props.block.subcategory

  if (subcategory) {
    return `${category} / ${subcategory}`
  }
  return category
})

// Tooltip positioning with boundary checks
const tooltipStyle = computed(() => {
  if (!props.position) {
    return { left: '0px', top: '0px' }
  }

  const tooltipWidth = 240
  const tooltipHeight = 180 // Approximate
  const padding = 10

  let x = props.position.x
  let y = props.position.y

  // Check right boundary - if tooltip would go off-screen, show on left side
  if (typeof window !== 'undefined') {
    if (x + tooltipWidth > window.innerWidth - padding) {
      // Position to the left of the element instead
      x = props.position.x - tooltipWidth - 20
    }

    // Check bottom boundary
    if (y + tooltipHeight > window.innerHeight - padding) {
      y = window.innerHeight - tooltipHeight - padding
    }

    // Ensure not negative
    if (x < padding) x = padding
    if (y < padding) y = padding
  }

  return {
    left: `${x}px`,
    top: `${y}px`,
  }
})
</script>

<template>
  <ClientOnly>
    <Teleport to="body">
      <Transition name="tooltip-fade">
        <div
          v-if="visible"
          class="block-tooltip"
          :style="tooltipStyle"
        >
        <div class="tooltip-header">
          <span class="tooltip-name">{{ block.name }}</span>
          <span v-if="block.category" class="tooltip-category">{{ categoryLabel }}</span>
        </div>

        <p v-if="block.description" class="tooltip-description">
          {{ block.description }}
        </p>

        <div class="tooltip-stats">
          <div class="stat-item">
            <span class="stat-label">{{ t('blockTooltip.fields') }}</span>
            <span class="stat-value">{{ fieldInfo.total }}</span>
          </div>
          <div class="stat-item">
            <span class="stat-label">{{ t('blockTooltip.required') }}</span>
            <span class="stat-value">{{ fieldInfo.required }}</span>
          </div>
        </div>

        <!-- Drag hint -->
        <div class="tooltip-hint">
          <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M5 9l-3 3 3 3M9 5l3-3 3 3M15 19l-3 3-3-3M19 9l3 3-3 3M2 12h20M12 2v20"/>
          </svg>
          <span>{{ t('blockTooltip.dragToCanvas') }}</span>
        </div>
      </div>
    </Transition>
  </Teleport>
  </ClientOnly>
</template>

<style scoped>
.block-tooltip {
  position: fixed;
  z-index: 9999;
  width: 240px;
  padding: 12px;
  background: var(--color-surface, #fff);
  border: 1px solid var(--color-border, #e5e7eb);
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  pointer-events: none;
}

.tooltip-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 8px;
}

.tooltip-name {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--color-text, #1f2937);
  line-height: 1.3;
}

.tooltip-category {
  font-size: 0.6875rem;
  color: var(--color-text-muted, #9ca3af);
  background: var(--color-background, #f9fafb);
  padding: 2px 6px;
  border-radius: 4px;
  white-space: nowrap;
  flex-shrink: 0;
}

.tooltip-description {
  font-size: 0.75rem;
  color: var(--color-text-secondary, #6b7280);
  line-height: 1.5;
  margin: 0 0 10px 0;
}

.tooltip-stats {
  display: flex;
  gap: 16px;
  padding: 8px 0;
  border-top: 1px solid var(--color-border, #e5e7eb);
  border-bottom: 1px solid var(--color-border, #e5e7eb);
  margin-bottom: 8px;
}

.stat-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.stat-label {
  font-size: 0.625rem;
  color: var(--color-text-muted, #9ca3af);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.stat-value {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--color-text, #1f2937);
}

.tooltip-hint {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.6875rem;
  color: var(--color-text-muted, #9ca3af);
}

.tooltip-hint svg {
  opacity: 0.7;
}

/* Fade animation */
.tooltip-fade-enter-active,
.tooltip-fade-leave-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}

.tooltip-fade-enter-from,
.tooltip-fade-leave-to {
  opacity: 0;
  transform: translateY(4px);
}
</style>
