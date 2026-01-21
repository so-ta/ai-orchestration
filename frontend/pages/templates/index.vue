<script setup lang="ts">
import type { ProjectTemplate } from '~/types/api'

definePageMeta({
  layout: 'default'
})

const { t } = useI18n()
const router = useRouter()
const toast = useToast()
const { confirm } = useConfirm()

const {
  templates,
  categories,
  loading,
  error,
  fetchTemplates,
  fetchMarketplace,
  fetchCategories,
  useTemplate,
  deleteTemplate
} = useTemplates()

// Tab state
const activeTab = ref<'my' | 'marketplace'>('my')

// Filter state
const searchQuery = ref('')
const selectedCategory = ref<string | null>(null)
const featuredOnly = ref(false)

// Fetch data on mount
onMounted(async () => {
  await fetchCategories()
  await loadTemplates()
})

// Watch tab changes
watch(activeTab, () => {
  loadTemplates()
})

// Watch filters
watch([searchQuery, selectedCategory, featuredOnly], () => {
  loadTemplates()
})

async function loadTemplates() {
  if (activeTab.value === 'my') {
    await fetchTemplates({
      search: searchQuery.value || undefined,
      category: selectedCategory.value || undefined
    })
  } else {
    await fetchMarketplace({
      search: searchQuery.value || undefined,
      category: selectedCategory.value || undefined,
      isFeatured: featuredOnly.value || undefined
    })
  }
}

async function handleUseTemplate(template: ProjectTemplate) {
  const projectName = await prompt('プロジェクト名を入力してください', template.name)
  if (!projectName) return

  try {
    const project = await useTemplate(template.id, projectName)
    router.push(`/projects/${project.id}`)
  } catch {
    toast.error(t('templates.messages.useFailed'))
  }
}

async function handleDeleteTemplate(template: ProjectTemplate) {
  const confirmed = await confirm({
    title: t('common.delete'),
    message: `テンプレート「${template.name}」を削除しますか？`,
    confirmText: t('common.delete'),
    cancelText: t('common.cancel'),
    variant: 'danger'
  })

  if (confirmed) {
    try {
      await deleteTemplate(template.id)
      await loadTemplates()
    } catch {
      toast.error('削除に失敗しました')
    }
  }
}

function formatRating(rating: number | undefined): string {
  if (!rating) return '-'
  return rating.toFixed(1)
}

function getVisibilityBadgeClass(visibility: string): string {
  switch (visibility) {
    case 'public': return 'badge-success'
    case 'tenant': return 'badge-info'
    default: return 'badge-secondary'
  }
}
</script>

<template>
  <div class="templates-page">
    <!-- Header -->
    <div class="page-header">
      <div class="header-content">
        <h1 class="page-title">{{ t('templates.title') }}</h1>
        <p class="page-subtitle">{{ t('templates.subtitle') }}</p>
      </div>
      <div class="header-actions">
        <NuxtLink to="/templates/create" class="btn btn-primary">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
          </svg>
          {{ t('templates.createTemplate') }}
        </NuxtLink>
      </div>
    </div>

    <!-- Tabs -->
    <div class="tabs">
      <button
        :class="['tab', { active: activeTab === 'my' }]"
        @click="activeTab = 'my'"
      >
        {{ t('templates.myTemplates') }}
      </button>
      <button
        :class="['tab', { active: activeTab === 'marketplace' }]"
        @click="activeTab = 'marketplace'"
      >
        {{ t('templates.marketplace') }}
      </button>
    </div>

    <!-- Filters -->
    <div class="filters">
      <div class="search-box">
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/>
        </svg>
        <input
          v-model="searchQuery"
          type="text"
          :placeholder="t('templates.searchPlaceholder')"
          class="search-input"
        >
      </div>

      <select v-model="selectedCategory" class="filter-select">
        <option :value="null">{{ t('templates.filters.all') }}</option>
        <option v-for="cat in categories" :key="cat.slug" :value="cat.slug">
          {{ cat.name }}
        </option>
      </select>

      <label v-if="activeTab === 'marketplace'" class="filter-checkbox">
        <input v-model="featuredOnly" type="checkbox">
        <span>{{ t('templates.featured') }}</span>
      </label>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="loading-state">
      <div class="spinner" />
      <span>{{ t('common.loading') }}</span>
    </div>

    <!-- Error -->
    <div v-else-if="error" class="error-state">
      <p>{{ error }}</p>
      <button class="btn btn-primary" @click="loadTemplates">{{ t('common.retry') }}</button>
    </div>

    <!-- Empty State -->
    <div v-else-if="templates.length === 0" class="empty-state">
      <div class="empty-icon">
        <svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
          <polyline points="14 2 14 8 20 8"/>
          <line x1="12" y1="18" x2="12" y2="12"/>
          <line x1="9" y1="15" x2="15" y2="15"/>
        </svg>
      </div>
      <h3>{{ t('templates.noTemplates') }}</h3>
      <p>{{ t('templates.noTemplatesDesc') }}</p>
      <NuxtLink to="/templates/create" class="btn btn-primary">
        {{ t('templates.createTemplate') }}
      </NuxtLink>
    </div>

    <!-- Template Grid -->
    <div v-else class="templates-grid">
      <div
        v-for="template in templates"
        :key="template.id"
        class="template-card"
      >
        <div class="card-header">
          <h3 class="template-name">{{ template.name }}</h3>
          <span :class="['visibility-badge', getVisibilityBadgeClass(template.visibility)]">
            {{ t(`templates.visibility.${template.visibility}`) }}
          </span>
        </div>

        <p v-if="template.description" class="template-description">
          {{ template.description }}
        </p>

        <div class="template-meta">
          <div v-if="template.category" class="meta-item">
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M3 3h18v18H3z"/>
            </svg>
            <span>{{ template.category }}</span>
          </div>
          <div class="meta-item">
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
              <polyline points="7 10 12 15 17 10"/>
              <line x1="12" y1="15" x2="12" y2="3"/>
            </svg>
            <span>{{ template.download_count }}</span>
          </div>
          <div class="meta-item">
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/>
            </svg>
            <span>{{ formatRating(template.rating) }}</span>
          </div>
        </div>

        <div v-if="template.tags && template.tags.length > 0" class="template-tags">
          <span v-for="tag in template.tags.slice(0, 3)" :key="tag" class="tag">
            {{ tag }}
          </span>
          <span v-if="template.tags.length > 3" class="tag tag-more">
            +{{ template.tags.length - 3 }}
          </span>
        </div>

        <div class="card-actions">
          <button class="btn btn-primary btn-sm" @click="handleUseTemplate(template)">
            {{ t('templates.useTemplate') }}
          </button>
          <NuxtLink :to="`/templates/${template.id}`" class="btn btn-secondary btn-sm">
            {{ t('common.edit') }}
          </NuxtLink>
          <button v-if="activeTab === 'my'" class="btn btn-danger-outline btn-sm" @click="handleDeleteTemplate(template)">
            {{ t('common.delete') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.templates-page {
  max-width: 1200px;
  margin: 0 auto;
  padding: 2rem;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 2rem;
}

.page-title {
  font-size: 1.75rem;
  font-weight: 700;
  margin: 0 0 0.25rem 0;
  color: var(--color-text);
}

.page-subtitle {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
  margin: 0;
}

.header-actions .btn {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.tabs {
  display: flex;
  gap: 0;
  margin-bottom: 1.5rem;
  border-bottom: 1px solid var(--color-border);
}

.tab {
  padding: 0.75rem 1.25rem;
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--color-text-secondary);
  background: transparent;
  border: none;
  border-bottom: 2px solid transparent;
  cursor: pointer;
  transition: all 0.15s;
  margin-bottom: -1px;
}

.tab:hover {
  color: var(--color-text);
}

.tab.active {
  color: var(--color-primary);
  border-bottom-color: var(--color-primary);
}

.filters {
  display: flex;
  gap: 1rem;
  margin-bottom: 1.5rem;
  flex-wrap: wrap;
  align-items: center;
}

.search-box {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
  background: white;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  flex: 1;
  max-width: 300px;
}

.search-box svg {
  color: var(--color-text-secondary);
}

.search-input {
  border: none;
  outline: none;
  font-size: 0.875rem;
  width: 100%;
}

.filter-select {
  padding: 0.5rem 0.75rem;
  font-size: 0.875rem;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  background: white;
}

.filter-checkbox {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.875rem;
  cursor: pointer;
}

.loading-state,
.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 4rem 2rem;
  gap: 1rem;
}

.spinner {
  width: 32px;
  height: 32px;
  border: 3px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 4rem 2rem;
  text-align: center;
}

.empty-icon {
  color: var(--color-primary);
  opacity: 0.5;
  margin-bottom: 1rem;
}

.empty-state h3 {
  font-size: 1.125rem;
  margin: 0 0 0.5rem 0;
}

.empty-state p {
  color: var(--color-text-secondary);
  margin: 0 0 1.5rem 0;
}

.templates-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 1.5rem;
}

.template-card {
  background: white;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  padding: 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  transition: box-shadow 0.15s;
}

.template-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 0.5rem;
}

.template-name {
  font-size: 1rem;
  font-weight: 600;
  margin: 0;
  color: var(--color-text);
}

.visibility-badge {
  font-size: 0.6875rem;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  font-weight: 500;
  white-space: nowrap;
}

.badge-success {
  background: #dcfce7;
  color: #15803d;
}

.badge-info {
  background: #dbeafe;
  color: #1d4ed8;
}

.badge-secondary {
  background: #f1f5f9;
  color: #64748b;
}

.template-description {
  font-size: 0.8125rem;
  color: var(--color-text-secondary);
  margin: 0;
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.template-meta {
  display: flex;
  gap: 1rem;
  flex-wrap: wrap;
}

.meta-item {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

.template-tags {
  display: flex;
  gap: 0.375rem;
  flex-wrap: wrap;
}

.tag {
  font-size: 0.6875rem;
  padding: 0.25rem 0.5rem;
  background: var(--color-surface);
  border-radius: 4px;
  color: var(--color-text-secondary);
}

.tag-more {
  background: var(--color-primary);
  color: white;
}

.card-actions {
  display: flex;
  gap: 0.5rem;
  margin-top: auto;
  padding-top: 0.5rem;
}

.btn-sm {
  padding: 0.375rem 0.75rem;
  font-size: 0.8125rem;
}

.btn-danger-outline {
  background: white;
  border: 1px solid #fecaca;
  color: var(--color-error);
}

.btn-danger-outline:hover {
  background: #fef2f2;
  border-color: var(--color-error);
}
</style>
