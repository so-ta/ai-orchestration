<script setup lang="ts">
/**
 * InheritanceChain - 継承チェーン可視化コンポーネント
 *
 * ブロックの継承チェーンを視覚的に表示し、マージされる設定値を確認可能。
 */
import type { BlockDefinition } from '~/types/api'

const props = defineProps<{
  blockId: string
}>()

const { t } = useI18n()
const blocksApi = useBlocks()

// State
const chain = ref<BlockDefinition[]>([])
const loading = ref(false)
const error = ref<string | null>(null)

// Fetch inheritance chain
async function fetchChain() {
  if (!props.blockId) {
    chain.value = []
    return
  }

  try {
    loading.value = true
    error.value = null

    // Fetch all blocks to build chain
    const response = await blocksApi.list()
    const blocks = response.blocks || []

    // Build chain from child to root
    const chainBlocks: BlockDefinition[] = []
    let currentBlock = blocks.find(b => b.id === props.blockId)

    while (currentBlock) {
      chainBlocks.push(currentBlock)

      if (currentBlock.parent_block_id) {
        currentBlock = blocks.find(b => b.id === currentBlock!.parent_block_id)
      } else {
        break
      }

      // Safety limit to prevent infinite loops
      if (chainBlocks.length > 10) {
        error.value = t('blockEditor.chainTooDeep')
        break
      }
    }

    // Reverse to show root first
    chain.value = chainBlocks.reverse()
  } catch (err) {
    error.value = t('blockEditor.chainLoadError')
    console.error('Failed to fetch inheritance chain:', err)
  } finally {
    loading.value = false
  }
}

// Fetch on mount and when blockId changes
onMounted(() => {
  fetchChain()
})

watch(() => props.blockId, () => {
  fetchChain()
})

// Computed: merged config defaults
const mergedDefaults = computed(() => {
  const merged: Record<string, unknown> = {}

  for (const block of chain.value) {
    if (block.config_defaults) {
      Object.assign(merged, block.config_defaults)
    }
  }

  return merged
})

// Check if this is the root (has code)
function isRoot(block: BlockDefinition, index: number): boolean {
  return index === 0 && !!block.code
}

// Check if this is the selected block
function isCurrent(index: number): boolean {
  return index === chain.value.length - 1
}
</script>

<template>
  <div class="inheritance-chain">
    <div class="chain-header">
      <span class="chain-icon">&#128279;</span>
      <span class="chain-title">{{ t('blockEditor.inheritanceChain') }}</span>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="chain-loading">
      {{ t('common.loading') }}
    </div>

    <!-- Error -->
    <div v-else-if="error" class="chain-error">
      {{ error }}
    </div>

    <!-- Empty -->
    <div v-else-if="chain.length === 0" class="chain-empty">
      {{ t('blockEditor.noChain') }}
    </div>

    <!-- Chain visualization -->
    <div v-else class="chain-visual">
      <div
        v-for="(block, index) in chain"
        :key="block.id"
        class="chain-node"
        :class="{
          root: isRoot(block, index),
          current: isCurrent(index)
        }"
      >
        <div class="node-connector" v-if="index > 0">
          <svg width="24" height="24" viewBox="0 0 24 24">
            <path d="M12 4 L12 20" stroke="currentColor" stroke-width="2" fill="none" />
            <path d="M6 14 L12 20 L18 14" stroke="currentColor" stroke-width="2" fill="none" />
          </svg>
        </div>

        <div class="node-content">
          <div class="node-icon">
            {{ block.icon || '&#9632;' }}
          </div>
          <div class="node-info">
            <span class="node-name">{{ block.name }}</span>
            <span class="node-slug">{{ block.slug }}</span>
          </div>
          <div class="node-badges">
            <span v-if="isRoot(block, index)" class="badge badge-root">
              {{ t('blockEditor.root') }}
            </span>
            <span v-if="block.is_system" class="badge badge-system">
              {{ t('blockEditor.systemBlock') }}
            </span>
            <span v-if="isCurrent(index)" class="badge badge-current">
              {{ t('blockEditor.selected') }}
            </span>
          </div>
        </div>

        <!-- Config defaults preview -->
        <div v-if="block.config_defaults && Object.keys(block.config_defaults).length > 0" class="node-defaults">
          <span class="defaults-label">{{ t('blockEditor.configDefaults') }}:</span>
          <code class="defaults-preview">{{ JSON.stringify(block.config_defaults) }}</code>
        </div>
      </div>
    </div>

    <!-- Summary -->
    <div v-if="chain.length > 0" class="chain-summary">
      <div class="summary-item">
        <span class="summary-label">{{ t('blockEditor.chainDepth') }}:</span>
        <span class="summary-value">{{ chain.length }} / 10</span>
      </div>
      <div v-if="Object.keys(mergedDefaults).length > 0" class="summary-item">
        <span class="summary-label">{{ t('blockEditor.mergedDefaults') }}:</span>
        <span class="summary-value">{{ Object.keys(mergedDefaults).length }} {{ t('blockEditor.fields') }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.inheritance-chain {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 0.5rem;
  padding: 1rem;
  margin: 1rem 0;
}

.chain-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 1rem;
}

.chain-icon {
  font-size: 1rem;
}

.chain-title {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--color-text);
}

.chain-loading,
.chain-error,
.chain-empty {
  padding: 1rem;
  text-align: center;
  font-size: 0.875rem;
  color: var(--color-text-secondary);
}

.chain-error {
  color: #ef4444;
  background: rgba(239, 68, 68, 0.1);
  border-radius: 0.375rem;
}

.chain-visual {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.chain-node {
  position: relative;
}

.node-connector {
  display: flex;
  justify-content: center;
  color: var(--color-border);
  margin: 0.25rem 0;
}

.node-content {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem;
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: 0.375rem;
  transition: all 0.15s;
}

.chain-node.root .node-content {
  border-color: var(--color-primary);
  background: rgba(99, 102, 241, 0.05);
}

.chain-node.current .node-content {
  border-color: #22c55e;
  background: rgba(34, 197, 94, 0.05);
}

.node-icon {
  font-size: 1.25rem;
  width: 2rem;
  text-align: center;
}

.node-info {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.node-name {
  font-weight: 500;
  font-size: 0.875rem;
}

.node-slug {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  font-family: 'Monaco', 'Menlo', monospace;
}

.node-badges {
  display: flex;
  gap: 0.375rem;
}

.badge {
  padding: 0.125rem 0.375rem;
  border-radius: 0.25rem;
  font-size: 0.6875rem;
  font-weight: 500;
}

.badge-root {
  background: rgba(99, 102, 241, 0.1);
  color: var(--color-primary);
}

.badge-system {
  background: rgba(107, 114, 128, 0.1);
  color: #6b7280;
}

.badge-current {
  background: rgba(34, 197, 94, 0.1);
  color: #16a34a;
}

.node-defaults {
  margin-top: 0.5rem;
  padding: 0.5rem;
  background: var(--color-background);
  border-radius: 0.25rem;
  font-size: 0.75rem;
}

.defaults-label {
  color: var(--color-text-secondary);
  margin-right: 0.5rem;
}

.defaults-preview {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
  word-break: break-all;
}

.chain-summary {
  margin-top: 1rem;
  padding-top: 1rem;
  border-top: 1px solid var(--color-border);
  display: flex;
  gap: 1.5rem;
}

.summary-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.8125rem;
}

.summary-label {
  color: var(--color-text-secondary);
}

.summary-value {
  font-weight: 500;
}
</style>
