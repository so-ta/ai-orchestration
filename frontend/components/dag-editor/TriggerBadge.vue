<script setup lang="ts">
/**
 * TriggerBadge.vue
 * Startブロックのトリガー種別を視覚的に表示するバッジコンポーネント
 */

type StartTriggerType = 'manual' | 'webhook' | 'schedule' | 'slack' | 'email'

const props = withDefaults(defineProps<{
  triggerType?: StartTriggerType
  stepType?: string // ブロックタイプ（schedule_trigger, webhook_trigger等）
  size?: 'sm' | 'md'
}>(), {
  triggerType: 'manual',
  stepType: '',
  size: 'sm',
})

// stepTypeからトリガータイプを判定（専用トリガーブロックの場合）
const effectiveTriggerType = computed((): StartTriggerType => {
  if (props.stepType === 'schedule_trigger') return 'schedule'
  if (props.stepType === 'webhook_trigger') return 'webhook'
  return props.triggerType || 'manual'
})

// トリガー種別ごとの設定
interface TriggerConfig {
  color: string
  bgColor: string
  icon: string
  label: string
}

const triggerConfigs: Record<StartTriggerType, TriggerConfig> = {
  manual: {
    color: '#64748b',
    bgColor: '#f1f5f9',
    icon: '👤',
    label: '手動',
  },
  webhook: {
    color: '#3b82f6',
    bgColor: '#dbeafe',
    icon: '↗',
    label: 'Webhook',
  },
  schedule: {
    color: '#22c55e',
    bgColor: '#dcfce7',
    icon: '⏰',
    label: 'スケジュール',
  },
  slack: {
    color: '#7c3aed',
    bgColor: '#ede9fe',
    icon: '#',
    label: 'Slack',
  },
  email: {
    color: '#f59e0b',
    bgColor: '#fef3c7',
    icon: '✉',
    label: 'メール',
  },
}

const config = computed(() => triggerConfigs[effectiveTriggerType.value] || triggerConfigs.manual)

const sizeClasses = computed(() => {
  return props.size === 'sm'
    ? 'text-[10px] px-1.5 py-0.5 gap-0.5'
    : 'text-xs px-2 py-1 gap-1'
})

const iconSize = computed(() => props.size === 'sm' ? 'text-[10px]' : 'text-sm')
</script>

<template>
  <div
    class="trigger-badge inline-flex items-center rounded-full font-medium whitespace-nowrap"
    :class="sizeClasses"
    :style="{
      backgroundColor: config.bgColor,
      color: config.color,
      border: `1px solid ${config.color}40`,
    }"
    :title="`トリガー: ${config.label}`"
  >
    <span :class="iconSize">{{ config.icon }}</span>
    <span v-if="size === 'md'">{{ config.label }}</span>
  </div>
</template>

<style scoped>
.trigger-badge {
  user-select: none;
  pointer-events: none;
}
</style>
