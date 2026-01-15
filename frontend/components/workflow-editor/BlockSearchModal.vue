<script setup lang="ts">
import type { BlockDefinition, BlockCategory, BlockSubcategory } from '~/types/api'
import {
  categoryConfig,
  subcategoryConfig,
  getBlockColor,
  searchBlocks,
  groupBlocksByCategory,
} from '~/composables/useBlocks'

const { t } = useI18n()
const blockPreferences = useBlockPreferences()

const props = defineProps<{
  modelValue: boolean
  blocks: BlockDefinition[]
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'select', block: BlockDefinition): void
}>()

const searchQuery = ref('')
const selectedIndex = ref(0)
const searchInputRef = ref<HTMLInputElement | null>(null)

// Filter and group blocks based on search query
const filteredBlocks = computed(() => {
  const filtered = searchBlocks(props.blocks, searchQuery.value)
    .filter(b => b.slug !== 'start') // Exclude start block
  return filtered
})

// Recent blocks (only show when no search query)
const recentBlocks = computed(() => {
  if (searchQuery.value) return []
  return blockPreferences.recentBlockSlugs.value
    .map(slug => props.blocks.find(b => b.slug === slug))
    .filter((b): b is BlockDefinition => b !== undefined)
    .slice(0, 5)
})

// Favorite blocks (only show when no search query)
const favoriteBlocks = computed(() => {
  if (searchQuery.value) return []
  return blockPreferences.favoriteBlockSlugs.value
    .map(slug => props.blocks.find(b => b.slug === slug))
    .filter((b): b is BlockDefinition => b !== undefined)
})

// Flat list for keyboard navigation (includes recent and favorites when not searching)
const flatBlockList = computed(() => {
  if (searchQuery.value) {
    return filteredBlocks.value
  }
  // When not searching, show: favorites, recent, then all blocks
  const all: BlockDefinition[] = []
  const seen = new Set<string>()

  // Add favorites first
  for (const block of favoriteBlocks.value) {
    if (!seen.has(block.id)) {
      all.push(block)
      seen.add(block.id)
    }
  }

  // Add recent blocks
  for (const block of recentBlocks.value) {
    if (!seen.has(block.id)) {
      all.push(block)
      seen.add(block.id)
    }
  }

  // Add remaining blocks
  for (const block of filteredBlocks.value) {
    if (!seen.has(block.id)) {
      all.push(block)
      seen.add(block.id)
    }
  }

  return all
})

// Grouped blocks for display
const groupedBlocks = computed(() => {
  return groupBlocksByCategory(filteredBlocks.value)
})

// Reset selection when search changes
watch(searchQuery, () => {
  selectedIndex.value = 0
})

// Focus input when modal opens
watch(() => props.modelValue, (isOpen) => {
  if (isOpen) {
    nextTick(() => {
      searchInputRef.value?.focus()
      searchQuery.value = ''
      selectedIndex.value = 0
    })
  }
})

function closeModal() {
  emit('update:modelValue', false)
}

function selectBlock(block: BlockDefinition) {
  emit('select', block)
  closeModal()
}

function handleKeydown(event: KeyboardEvent) {
  const listLength = flatBlockList.value.length

  switch (event.key) {
    case 'ArrowDown':
      event.preventDefault()
      selectedIndex.value = Math.min(selectedIndex.value + 1, listLength - 1)
      scrollToSelected()
      break
    case 'ArrowUp':
      event.preventDefault()
      selectedIndex.value = Math.max(selectedIndex.value - 1, 0)
      scrollToSelected()
      break
    case 'Enter':
      event.preventDefault()
      if (flatBlockList.value[selectedIndex.value]) {
        selectBlock(flatBlockList.value[selectedIndex.value])
      }
      break
    case 'Escape':
      closeModal()
      break
  }
}

function scrollToSelected() {
  nextTick(() => {
    const selectedElement = document.querySelector('.block-item.selected')
    selectedElement?.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
  })
}

function getCategoryLabel(category: BlockCategory): string {
  return t(categoryConfig[category]?.nameKey || `editor.categories.${category}`)
}

function getSubcategoryLabel(subcategory: BlockSubcategory): string {
  return t(subcategoryConfig[subcategory]?.nameKey || `editor.subcategories.${subcategory}`)
}

function getBlockIndex(block: BlockDefinition): number {
  return flatBlockList.value.findIndex(b => b.id === block.id)
}

function toggleFavorite(event: Event, block: BlockDefinition) {
  event.stopPropagation()
  blockPreferences.toggleFavorite(block.slug)
}

function isFavorite(block: BlockDefinition): boolean {
  return blockPreferences.isFavorite(block.slug)
}
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div
        v-if="modelValue"
        class="modal-overlay"
        @click.self="closeModal"
        @keydown="handleKeydown"
      >
        <div class="modal-content">
          <!-- Search Input -->
          <div class="search-header">
            <div class="search-input-wrapper">
              <svg class="search-icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="11" cy="11" r="8" />
                <path d="m21 21-4.35-4.35" />
              </svg>
              <input
                ref="searchInputRef"
                v-model="searchQuery"
                type="text"
                class="search-input"
                :placeholder="t('editor.searchBlocks')"
                @keydown="handleKeydown"
              >
              <kbd class="keyboard-hint">esc</kbd>
            </div>
          </div>

          <!-- Results -->
          <div class="results-container">
            <template v-if="filteredBlocks.length === 0">
              <div class="no-results">
                <p>{{ t('editor.noBlocksFound') }}</p>
              </div>
            </template>

            <template v-else>
              <!-- Favorites Section (only when not searching) -->
              <div v-if="favoriteBlocks.length > 0 && !searchQuery" class="category-section">
                <div class="category-header">
                  <svg class="section-icon" width="12" height="12" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/>
                  </svg>
                  <span class="category-name">{{ t('editor.favorites') }}</span>
                  <span class="category-count">{{ favoriteBlocks.length }}</span>
                </div>
                <div class="block-list">
                  <button
                    v-for="block in favoriteBlocks"
                    :key="'fav-' + block.id"
                    :class="['block-item', { selected: getBlockIndex(block) === selectedIndex }]"
                    @click="selectBlock(block)"
                    @mouseenter="selectedIndex = getBlockIndex(block)"
                  >
                    <div
                      class="block-color"
                      :style="{ backgroundColor: getBlockColor(block.slug) }"
                    />
                    <div class="block-info">
                      <div class="block-name">{{ block.name }}</div>
                      <div class="block-meta">
                        <span v-if="block.subcategory" class="block-subcategory">
                          {{ getSubcategoryLabel(block.subcategory) }}
                        </span>
                      </div>
                    </div>
                    <button
                      class="favorite-btn favorited"
                      :title="t('editor.removeFromFavorites')"
                      @click="toggleFavorite($event, block)"
                    >
                      <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
                        <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/>
                      </svg>
                    </button>
                    <kbd v-if="getBlockIndex(block) === selectedIndex" class="enter-hint">
                      Enter
                    </kbd>
                  </button>
                </div>
              </div>

              <!-- Recently Used Section (only when not searching) -->
              <div v-if="recentBlocks.length > 0 && !searchQuery" class="category-section">
                <div class="category-header">
                  <svg class="section-icon" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="12" cy="12" r="10"/>
                    <polyline points="12 6 12 12 16 14"/>
                  </svg>
                  <span class="category-name">{{ t('editor.recentlyUsed') }}</span>
                  <span class="category-count">{{ recentBlocks.length }}</span>
                </div>
                <div class="block-list">
                  <button
                    v-for="block in recentBlocks"
                    :key="'recent-' + block.id"
                    :class="['block-item', { selected: getBlockIndex(block) === selectedIndex }]"
                    @click="selectBlock(block)"
                    @mouseenter="selectedIndex = getBlockIndex(block)"
                  >
                    <div
                      class="block-color"
                      :style="{ backgroundColor: getBlockColor(block.slug) }"
                    />
                    <div class="block-info">
                      <div class="block-name">{{ block.name }}</div>
                      <div class="block-meta">
                        <span v-if="block.subcategory" class="block-subcategory">
                          {{ getSubcategoryLabel(block.subcategory) }}
                        </span>
                      </div>
                    </div>
                    <button
                      :class="['favorite-btn', { favorited: isFavorite(block) }]"
                      :title="isFavorite(block) ? t('editor.removeFromFavorites') : t('editor.addToFavorites')"
                      @click="toggleFavorite($event, block)"
                    >
                      <svg width="14" height="14" viewBox="0 0 24 24" :fill="isFavorite(block) ? 'currentColor' : 'none'" stroke="currentColor" stroke-width="2">
                        <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/>
                      </svg>
                    </button>
                    <kbd v-if="getBlockIndex(block) === selectedIndex" class="enter-hint">
                      Enter
                    </kbd>
                  </button>
                </div>
              </div>

              <!-- All Blocks Header (only when not searching and has favorites/recent) -->
              <div v-if="!searchQuery && (favoriteBlocks.length > 0 || recentBlocks.length > 0)" class="all-blocks-divider">
                <span>{{ t('editor.allBlocks') }}</span>
              </div>

              <!-- Grouped by Category -->
              <div
                v-for="(categoryBlocks, category) in groupedBlocks"
                :key="category"
                class="category-section"
              >
                <template v-if="categoryBlocks.length > 0">
                  <div class="category-header">
                    <span
                      class="category-indicator"
                      :style="{ backgroundColor: categoryConfig[category]?.color }"
                    />
                    <span class="category-name">{{ getCategoryLabel(category) }}</span>
                    <span class="category-count">{{ categoryBlocks.length }}</span>
                  </div>

                  <div class="block-list">
                    <button
                      v-for="block in categoryBlocks"
                      :key="block.id"
                      :class="['block-item', { selected: getBlockIndex(block) === selectedIndex }]"
                      @click="selectBlock(block)"
                      @mouseenter="selectedIndex = getBlockIndex(block)"
                    >
                      <div
                        class="block-color"
                        :style="{ backgroundColor: getBlockColor(block.slug) }"
                      />
                      <div class="block-info">
                        <div class="block-name">{{ block.name }}</div>
                        <div class="block-meta">
                          <span v-if="block.subcategory" class="block-subcategory">
                            {{ getSubcategoryLabel(block.subcategory) }}
                          </span>
                          <span v-if="block.description" class="block-desc">
                            {{ block.description }}
                          </span>
                        </div>
                      </div>
                      <button
                        :class="['favorite-btn', { favorited: isFavorite(block) }]"
                        :title="isFavorite(block) ? t('editor.removeFromFavorites') : t('editor.addToFavorites')"
                        @click="toggleFavorite($event, block)"
                      >
                        <svg width="14" height="14" viewBox="0 0 24 24" :fill="isFavorite(block) ? 'currentColor' : 'none'" stroke="currentColor" stroke-width="2">
                          <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/>
                        </svg>
                      </button>
                      <kbd v-if="getBlockIndex(block) === selectedIndex" class="enter-hint">
                        Enter
                      </kbd>
                    </button>
                  </div>
                </template>
              </div>
            </template>
          </div>

          <!-- Footer hint -->
          <div class="modal-footer">
            <span class="hint-item">
              <kbd>↑↓</kbd> {{ t('editor.navigate') }}
            </span>
            <span class="hint-item">
              <kbd>Enter</kbd> {{ t('editor.select') }}
            </span>
            <span class="hint-item">
              <kbd>Esc</kbd> {{ t('editor.close') }}
            </span>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding-top: 10vh;
  z-index: 1000;
}

.modal-content {
  width: 100%;
  max-width: 560px;
  max-height: 70vh;
  background: white;
  border-radius: 12px;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.search-header {
  padding: 16px;
  border-bottom: 1px solid var(--color-border, #e5e7eb);
}

.search-input-wrapper {
  display: flex;
  align-items: center;
  gap: 12px;
  background: var(--color-background-secondary, #f9fafb);
  border-radius: 8px;
  padding: 10px 14px;
}

.search-icon {
  color: var(--color-text-secondary, #6b7280);
  flex-shrink: 0;
}

.search-input {
  flex: 1;
  border: none;
  background: none;
  font-size: 15px;
  color: var(--color-text, #111827);
  outline: none;
}

.search-input::placeholder {
  color: var(--color-text-secondary, #9ca3af);
}

.keyboard-hint {
  font-size: 11px;
  padding: 2px 6px;
  background: var(--color-background, white);
  border: 1px solid var(--color-border, #e5e7eb);
  border-radius: 4px;
  color: var(--color-text-secondary, #6b7280);
}

.results-container {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.no-results {
  padding: 32px;
  text-align: center;
  color: var(--color-text-secondary, #6b7280);
}

.category-section {
  margin-bottom: 16px;
}

.category-section:last-child {
  margin-bottom: 0;
}

.category-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px 4px;
}

.section-icon {
  color: var(--color-text-secondary, #6b7280);
}

.category-indicator {
  width: 8px;
  height: 8px;
  border-radius: 2px;
}

.all-blocks-divider {
  display: flex;
  align-items: center;
  padding: 8px 12px;
  margin: 8px 0;
}

.all-blocks-divider::before,
.all-blocks-divider::after {
  content: '';
  flex: 1;
  height: 1px;
  background: var(--color-border, #e5e7eb);
}

.all-blocks-divider span {
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-tertiary, #9ca3af);
  padding: 0 12px;
}

.category-name {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-secondary, #6b7280);
}

.category-count {
  font-size: 10px;
  color: var(--color-text-tertiary, #9ca3af);
  background: var(--color-background-secondary, #f3f4f6);
  padding: 1px 6px;
  border-radius: 10px;
}

.block-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.block-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  background: none;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  text-align: left;
  width: 100%;
  transition: background 0.1s;
}

.block-item:hover,
.block-item.selected {
  background: var(--color-background-secondary, #f3f4f6);
}

.block-item.selected {
  background: var(--color-primary-light, #eff6ff);
}

.block-color {
  width: 4px;
  height: 36px;
  border-radius: 2px;
  flex-shrink: 0;
}

.block-info {
  flex: 1;
  min-width: 0;
}

.block-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text, #111827);
  line-height: 1.3;
}

.block-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 2px;
}

.block-subcategory {
  font-size: 11px;
  color: var(--color-primary, #3b82f6);
  background: var(--color-primary-light, #eff6ff);
  padding: 1px 6px;
  border-radius: 4px;
}

.block-desc {
  font-size: 12px;
  color: var(--color-text-secondary, #6b7280);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.favorite-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  padding: 0;
  background: transparent;
  border: none;
  border-radius: 4px;
  color: var(--color-text-tertiary, #9ca3af);
  cursor: pointer;
  flex-shrink: 0;
  opacity: 0;
  transition: opacity 0.1s, color 0.1s, background 0.1s;
}

.block-item:hover .favorite-btn,
.block-item.selected .favorite-btn,
.favorite-btn.favorited {
  opacity: 1;
}

.favorite-btn:hover {
  background: var(--color-background, white);
  color: var(--color-warning, #f59e0b);
}

.favorite-btn.favorited {
  color: var(--color-warning, #f59e0b);
}

.enter-hint {
  font-size: 10px;
  padding: 2px 6px;
  background: var(--color-primary, #3b82f6);
  color: white;
  border-radius: 4px;
  flex-shrink: 0;
}

.modal-footer {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 24px;
  padding: 12px;
  border-top: 1px solid var(--color-border, #e5e7eb);
  background: var(--color-background-secondary, #f9fafb);
}

.hint-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--color-text-secondary, #6b7280);
}

.hint-item kbd {
  font-size: 10px;
  padding: 2px 5px;
  background: white;
  border: 1px solid var(--color-border, #e5e7eb);
  border-radius: 4px;
  font-family: inherit;
}

/* Transitions */
.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.15s ease;
}

.modal-enter-active .modal-content,
.modal-leave-active .modal-content {
  transition: transform 0.15s ease, opacity 0.15s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-from .modal-content,
.modal-leave-to .modal-content {
  transform: scale(0.95) translateY(-10px);
  opacity: 0;
}

/* Scrollbar */
.results-container::-webkit-scrollbar {
  width: 6px;
}

.results-container::-webkit-scrollbar-track {
  background: transparent;
}

.results-container::-webkit-scrollbar-thumb {
  background: var(--color-border, #d1d5db);
  border-radius: 3px;
}
</style>
