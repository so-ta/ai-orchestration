<script setup lang="ts">
/**
 * BlockCard - ブロックカード表示コンポーネント
 *
 * ブロック一覧でブロックを表示するためのカードコンポーネント。
 * 編集、複製、バージョン確認、削除のアクションをサポート。
 */
import type { BlockDefinition } from '~/types/api'
import { categoryConfig, getBlockColor } from '~/composables/useBlocks'

const props = defineProps<{
  block: BlockDefinition
}>()

const emit = defineEmits<{
  edit: [block: BlockDefinition]
  duplicate: [block: BlockDefinition]
  viewVersions: [block: BlockDefinition]
  delete: [block: BlockDefinition]
}>()

const { t } = useI18n()

// Get category display name
function getCategoryName(category: string): string {
  const config = categoryConfig[category as keyof typeof categoryConfig]
  return config ? t(config.nameKey) : category
}

// Format date
function formatDate(date: string | undefined): string {
  if (!date) return '-'
  return new Date(date).toLocaleDateString()
}

// Get background color for icon
const iconBgColor = computed(() => {
  return getBlockColor(props.block.slug) + '15' // Add alpha
})

// Check if block has parent (is inherited)
const hasParent = computed(() => !!props.block.parent_block_id)
</script>

<template>
  <div class="block-card" :class="{ 'is-system': block.is_system }">
    <div class="card-header">
      <div class="block-icon" :style="{ background: iconBgColor }">
        {{ block.icon || '&#9632;' }}
      </div>
      <div class="block-info">
        <h4 class="block-name">{{ block.name }}</h4>
        <code class="block-slug">{{ block.slug }}</code>
      </div>
      <div class="block-badges">
        <span v-if="block.is_system" class="badge badge-system">
          {{ t('blockEditor.systemBlock') }}
        </span>
        <span v-if="hasParent" class="badge badge-inherited">
          {{ t('blockEditor.inherited') }}
        </span>
        <span class="badge badge-version">v{{ block.version || 1 }}</span>
      </div>
    </div>

    <p v-if="block.description" class="block-description">
      {{ block.description }}
    </p>

    <!-- Inheritance info -->
    <div v-if="hasParent" class="inheritance-info">
      <span class="inheritance-icon">&#128279;</span>
      <span class="inheritance-text">
        {{ t('blockEditor.inheritedFrom') }}
      </span>
    </div>

    <div class="card-meta">
      <span class="meta-category">
        <span class="category-dot" :style="{ background: categoryConfig[block.category]?.color }"></span>
        {{ getCategoryName(block.category) }}
      </span>
      <span class="meta-updated">
        {{ formatDate(block.updated_at) }}
      </span>
    </div>

    <div class="card-actions">
      <button
        class="action-btn"
        :title="t('common.edit')"
        @click.stop="emit('edit', block)"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
          <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
        </svg>
      </button>
      <button
        class="action-btn"
        :title="t('common.duplicate')"
        @click.stop="emit('duplicate', block)"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
          <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
        </svg>
      </button>
      <button
        class="action-btn"
        :title="t('blockEditor.versionHistory')"
        @click.stop="emit('viewVersions', block)"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10" />
          <polyline points="12 6 12 12 16 14" />
        </svg>
      </button>
      <button
        v-if="!block.is_system"
        class="action-btn action-danger"
        :title="t('common.delete')"
        @click.stop="emit('delete', block)"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="3 6 5 6 21 6" />
          <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
        </svg>
      </button>
    </div>
  </div>
</template>

<style scoped>
.block-card {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 0.5rem;
  padding: 1rem;
  transition: all 0.15s;
}

.block-card:hover {
  border-color: var(--color-primary);
  box-shadow: 0 2px 8px rgba(99, 102, 241, 0.1);
}

.block-card.is-system {
  border-left: 3px solid var(--color-primary);
}

.card-header {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
  margin-bottom: 0.75rem;
}

.block-icon {
  width: 2.5rem;
  height: 2.5rem;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 0.5rem;
  font-size: 1.25rem;
  flex-shrink: 0;
}

.block-info {
  flex: 1;
  min-width: 0;
}

.block-name {
  font-size: 0.9375rem;
  font-weight: 600;
  margin: 0 0 0.25rem 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.block-slug {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
  background: var(--color-background);
  padding: 0.125rem 0.375rem;
  border-radius: 0.25rem;
}

.block-badges {
  display: flex;
  flex-wrap: wrap;
  gap: 0.25rem;
}

.badge {
  padding: 0.125rem 0.375rem;
  border-radius: 0.25rem;
  font-size: 0.625rem;
  font-weight: 600;
  text-transform: uppercase;
}

.badge-system {
  background: rgba(99, 102, 241, 0.1);
  color: var(--color-primary);
}

.badge-inherited {
  background: rgba(34, 197, 94, 0.1);
  color: #16a34a;
}

.badge-version {
  background: var(--color-background);
  color: var(--color-text-secondary);
}

.block-description {
  font-size: 0.8125rem;
  color: var(--color-text-secondary);
  margin: 0 0 0.75rem 0;
  line-height: 1.4;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.inheritance-info {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem;
  background: rgba(34, 197, 94, 0.05);
  border: 1px solid rgba(34, 197, 94, 0.1);
  border-radius: 0.25rem;
  margin-bottom: 0.75rem;
}

.inheritance-icon {
  font-size: 0.875rem;
}

.inheritance-text {
  font-size: 0.75rem;
  color: #16a34a;
}

.card-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  padding-top: 0.75rem;
  border-top: 1px solid var(--color-border);
}

.meta-category {
  display: flex;
  align-items: center;
  gap: 0.375rem;
}

.category-dot {
  width: 0.5rem;
  height: 0.5rem;
  border-radius: 50%;
}

.card-actions {
  display: flex;
  gap: 0.25rem;
  margin-top: 0.75rem;
  padding-top: 0.75rem;
  border-top: 1px solid var(--color-border);
}

.action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 2rem;
  height: 2rem;
  padding: 0;
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: 0.375rem;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.15s;
}

.action-btn:hover {
  background: var(--color-surface);
  color: var(--color-text);
  border-color: var(--color-primary);
}

.action-btn.action-danger:hover {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
  border-color: #ef4444;
}
</style>
