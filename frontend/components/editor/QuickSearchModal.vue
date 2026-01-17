<script setup lang="ts">
import type { BlockDefinition, BlockGroupType } from '~/types/api'
import { getBlockColor } from '~/composables/useBlocks'

const { t } = useI18n()

const props = defineProps<{
  open: boolean
  blocks: BlockDefinition[]
}>()

const emit = defineEmits<{
  (e: 'update:open', value: boolean): void
  (e: 'selectBlock', block: BlockDefinition): void
  (e: 'selectGroup', type: BlockGroupType): void
}>()

// Search query
const query = ref('')
const selectedIndex = ref(0)
const inputRef = ref<HTMLInputElement | null>(null)

// Block groups
const blockGroups: Array<{ type: BlockGroupType; name: string; icon: string; description: string }> = [
  { type: 'parallel', name: 'Parallel', icon: '⫲', description: 'Execute steps in parallel' },
  { type: 'foreach', name: 'ForEach', icon: '∀', description: 'Loop over items' },
  { type: 'try_catch', name: 'Try-Catch', icon: '⚡', description: 'Error handling block' },
  { type: 'while', name: 'While', icon: '↻', description: 'Repeat while condition' },
]

// Recent blocks (stored in localStorage)
const recentBlockSlugs = ref<string[]>([])

onMounted(() => {
  const stored = localStorage.getItem('recentBlocks')
  if (stored) {
    try {
      recentBlockSlugs.value = JSON.parse(stored)
    } catch {
      recentBlockSlugs.value = []
    }
  }
})

const recentBlocks = computed(() => {
  return recentBlockSlugs.value
    .map(slug => props.blocks.find(b => b.slug === slug))
    .filter((b): b is BlockDefinition => b !== undefined)
    .slice(0, 5)
})

// Filtered blocks
const filteredBlocks = computed(() => {
  if (!query.value.trim()) {
    return props.blocks.filter(b => b.slug !== 'start')
  }

  const q = query.value.toLowerCase().trim()
  return props.blocks.filter(b =>
    b.slug !== 'start' && (
      b.name.toLowerCase().includes(q) ||
      b.description?.toLowerCase().includes(q) ||
      b.slug.toLowerCase().includes(q)
    )
  )
})

// Filtered block groups
const filteredGroups = computed(() => {
  if (!query.value.trim()) {
    return blockGroups
  }

  const q = query.value.toLowerCase().trim()
  return blockGroups.filter(g =>
    g.name.toLowerCase().includes(q) ||
    g.description.toLowerCase().includes(q)
  )
})

// Total selectable items count
const totalItems = computed(() => {
  const showRecent = !query.value.trim() && recentBlocks.value.length > 0
  return (showRecent ? recentBlocks.value.length : 0) +
         filteredGroups.value.length +
         filteredBlocks.value.length
})

// Reset selection when query changes
watch(query, () => {
  selectedIndex.value = 0
})

// Focus input when opened
watch(() => props.open, (isOpen) => {
  if (isOpen) {
    query.value = ''
    selectedIndex.value = 0
    nextTick(() => {
      inputRef.value?.focus()
    })
  }
})

// Close handler
function close() {
  emit('update:open', false)
}

// Keyboard navigation
function handleKeydown(event: KeyboardEvent) {
  switch (event.key) {
    case 'ArrowDown':
      event.preventDefault()
      selectedIndex.value = Math.min(selectedIndex.value + 1, totalItems.value - 1)
      break
    case 'ArrowUp':
      event.preventDefault()
      selectedIndex.value = Math.max(selectedIndex.value - 1, 0)
      break
    case 'Enter':
      event.preventDefault()
      selectCurrentItem()
      break
    case 'Escape':
      close()
      break
  }
}

// Select current item
function selectCurrentItem() {
  const showRecent = !query.value.trim() && recentBlocks.value.length > 0
  let idx = selectedIndex.value

  // Recent blocks
  if (showRecent) {
    if (idx < recentBlocks.value.length) {
      selectBlock(recentBlocks.value[idx])
      return
    }
    idx -= recentBlocks.value.length
  }

  // Groups
  if (idx < filteredGroups.value.length) {
    selectGroup(filteredGroups.value[idx].type)
    return
  }
  idx -= filteredGroups.value.length

  // Blocks
  if (idx < filteredBlocks.value.length) {
    selectBlock(filteredBlocks.value[idx])
  }
}

// Select block
function selectBlock(block: BlockDefinition) {
  // Add to recent
  const slugs = recentBlockSlugs.value.filter(s => s !== block.slug)
  slugs.unshift(block.slug)
  recentBlockSlugs.value = slugs.slice(0, 10)
  localStorage.setItem('recentBlocks', JSON.stringify(recentBlockSlugs.value))

  emit('selectBlock', block)
  close()
}

// Select group
function selectGroup(type: BlockGroupType) {
  emit('selectGroup', type)
  close()
}

// Calculate item index for highlighting
function getItemIndex(section: 'recent' | 'groups' | 'blocks', index: number): number {
  const showRecent = !query.value.trim() && recentBlocks.value.length > 0

  if (section === 'recent') {
    return index
  }

  if (section === 'groups') {
    return (showRecent ? recentBlocks.value.length : 0) + index
  }

  return (showRecent ? recentBlocks.value.length : 0) +
         filteredGroups.value.length +
         index
}
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="open" class="quick-search-overlay" @click.self="close">
        <div class="quick-search-modal" @keydown="handleKeydown">
          <!-- Search Input -->
          <div class="search-header">
            <svg class="search-icon" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="11" cy="11" r="8" />
              <path d="m21 21-4.35-4.35" />
            </svg>
            <input
              ref="inputRef"
              v-model="query"
              type="text"
              class="search-input"
              :placeholder="t('editor.searchBlocksPlaceholder')"
            >
            <kbd class="search-shortcut">ESC</kbd>
          </div>

          <!-- Results -->
          <div class="search-results">
            <!-- Recent Blocks -->
            <div v-if="!query.trim() && recentBlocks.length > 0" class="result-section">
              <div class="section-title">{{ t('editor.recentBlocks') }}</div>
              <div
                v-for="(block, index) in recentBlocks"
                :key="`recent-${block.slug}`"
                :class="['result-item', { selected: selectedIndex === getItemIndex('recent', index) }]"
                @click="selectBlock(block)"
                @mouseenter="selectedIndex = getItemIndex('recent', index)"
              >
                <span class="block-indicator" :style="{ background: getBlockColor(block.slug) }" />
                <div class="block-info">
                  <span class="block-name">{{ block.name }}</span>
                  <span class="block-desc">{{ block.description }}</span>
                </div>
              </div>
            </div>

            <!-- Groups -->
            <div v-if="filteredGroups.length > 0" class="result-section">
              <div class="section-title">{{ t('editor.controlFlowGroups') }}</div>
              <div
                v-for="(group, index) in filteredGroups"
                :key="group.type"
                :class="['result-item', { selected: selectedIndex === getItemIndex('groups', index) }]"
                @click="selectGroup(group.type)"
                @mouseenter="selectedIndex = getItemIndex('groups', index)"
              >
                <span class="group-icon">{{ group.icon }}</span>
                <div class="block-info">
                  <span class="block-name">{{ group.name }}</span>
                  <span class="block-desc">{{ group.description }}</span>
                </div>
              </div>
            </div>

            <!-- Blocks -->
            <div v-if="filteredBlocks.length > 0" class="result-section">
              <div v-if="query.trim()" class="section-title">{{ t('editor.searchResults') }}</div>
              <div v-else class="section-title">{{ t('editor.allBlocks') }}</div>
              <div
                v-for="(block, index) in filteredBlocks.slice(0, 20)"
                :key="block.slug"
                :class="['result-item', { selected: selectedIndex === getItemIndex('blocks', index) }]"
                @click="selectBlock(block)"
                @mouseenter="selectedIndex = getItemIndex('blocks', index)"
              >
                <span class="block-indicator" :style="{ background: getBlockColor(block.slug) }" />
                <div class="block-info">
                  <span class="block-name">{{ block.name }}</span>
                  <span class="block-desc">{{ block.description }}</span>
                </div>
                <span class="block-category">{{ block.category }}</span>
              </div>
              <div v-if="filteredBlocks.length > 20" class="more-results">
                {{ t('editor.moreResults', { count: filteredBlocks.length - 20 }) }}
              </div>
            </div>

            <!-- Empty State -->
            <div v-if="query.trim() && filteredBlocks.length === 0 && filteredGroups.length === 0" class="empty-state">
              {{ t('editor.noBlocksFound', { query }) }}
            </div>
          </div>

          <!-- Footer -->
          <div class="search-footer">
            <span><kbd>↑↓</kbd> {{ t('editor.navigate') }}</span>
            <span><kbd>Enter</kbd> {{ t('editor.select') }}</span>
            <span><kbd>ESC</kbd> {{ t('editor.close') }}</span>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.quick-search-overlay {
  position: fixed;
  inset: 0;
  z-index: 200;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding-top: 12vh;
  background: rgba(0, 0, 0, 0.4);
  backdrop-filter: blur(4px);
}

.quick-search-modal {
  width: 100%;
  max-width: 560px;
  background: white;
  border-radius: 16px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.25);
  overflow: hidden;
}

/* Search Header */
.search-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 18px 20px;
  border-bottom: 1px solid #e5e7eb;
}

.search-icon {
  color: #9ca3af;
  flex-shrink: 0;
}

.search-input {
  flex: 1;
  border: none;
  font-size: 17px;
  color: #111827;
  outline: none;
}

.search-input::placeholder {
  color: #9ca3af;
}

.search-shortcut {
  padding: 4px 10px;
  background: #f3f4f6;
  border-radius: 6px;
  font-size: 11px;
  font-weight: 500;
  color: #6b7280;
}

/* Results */
.search-results {
  max-height: 420px;
  overflow-y: auto;
}

.result-section {
  padding: 8px 0;
}

.section-title {
  padding: 8px 20px 6px;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: #9ca3af;
}

.result-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 20px;
  cursor: pointer;
  transition: background 0.1s;
}

.result-item:hover,
.result-item.selected {
  background: #f3f4f6;
}

.block-indicator {
  width: 4px;
  height: 32px;
  border-radius: 2px;
  flex-shrink: 0;
}

.group-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  background: #8b5cf6;
  border-radius: 6px;
  font-size: 14px;
  color: white;
  flex-shrink: 0;
}

.block-info {
  flex: 1;
  min-width: 0;
}

.block-name {
  display: block;
  font-size: 14px;
  font-weight: 500;
  color: #111827;
  line-height: 1.3;
}

.block-desc {
  display: block;
  margin-top: 2px;
  font-size: 12px;
  color: #9ca3af;
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.block-category {
  padding: 2px 8px;
  background: #f3f4f6;
  border-radius: 4px;
  font-size: 11px;
  color: #6b7280;
  text-transform: capitalize;
  flex-shrink: 0;
}

.more-results {
  padding: 12px 20px;
  text-align: center;
  font-size: 12px;
  color: #9ca3af;
}

.empty-state {
  padding: 40px 20px;
  text-align: center;
  color: #9ca3af;
  font-size: 14px;
}

/* Footer */
.search-footer {
  display: flex;
  gap: 20px;
  padding: 12px 20px;
  background: #f9fafb;
  border-top: 1px solid #e5e7eb;
  font-size: 12px;
  color: #6b7280;
}

.search-footer kbd {
  padding: 2px 6px;
  background: white;
  border: 1px solid #e5e7eb;
  border-radius: 4px;
  font-size: 11px;
  font-family: inherit;
}

/* Scrollbar */
.search-results::-webkit-scrollbar {
  width: 6px;
}

.search-results::-webkit-scrollbar-track {
  background: transparent;
}

.search-results::-webkit-scrollbar-thumb {
  background: rgba(0, 0, 0, 0.1);
  border-radius: 3px;
}

/* Modal Transition */
.modal-enter-active,
.modal-leave-active {
  transition: all 0.2s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-from .quick-search-modal,
.modal-leave-to .quick-search-modal {
  transform: scale(0.96) translateY(-20px);
}
</style>
