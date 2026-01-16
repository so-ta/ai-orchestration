<script setup lang="ts">
/**
 * BlockCreationWizard - ブロック作成ウィザードコンポーネント
 *
 * 新規ブロック作成時に、作成方法（ゼロから/継承/テンプレート）を選択し、
 * 適切なフォームへ誘導するウィザード。
 */
import type { BlockDefinition, BlockCategory } from '~/types/api'
import { useBlocks } from '~/composables/useBlocks'

const emit = defineEmits<{
  complete: [block: BlockDefinition]
  cancel: []
}>()

const { t } = useI18n()
const blocksApi = useBlocks()

// Wizard state
const step = ref(0)
const creationType = ref<'scratch' | 'inherit' | 'template' | null>(null)
const selectedTemplate = ref<TemplateDefinition | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)

// Template definition
// Note: displayCategory is for UI grouping, blockCategory is the actual BlockCategory
interface TemplateDefinition {
  id: string
  name: string
  description: string
  icon: string
  displayCategory: string  // UI grouping (notification, data, utility)
  blockCategory: BlockCategory  // Actual block category
  inheritsFrom?: string
  parentBlockSlug?: string
  configDefaults?: Record<string, unknown>
  preProcess?: string
  postProcess?: string
  code?: string
  configSchema?: Record<string, unknown>
}

// Predefined templates
const templates: TemplateDefinition[] = [
  {
    id: 'discord-notify',
    name: 'Discord通知',
    description: 'Discord Webhookにメッセージを送信',
    icon: '💬',
    displayCategory: 'notification',
    blockCategory: 'apps',
    inheritsFrom: 'HTTP',
    parentBlockSlug: 'http',
    configDefaults: {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
    },
    preProcess: `const webhookUrl = ctx.secrets.DISCORD_WEBHOOK_URL || config.webhook_url;
return {
  url: webhookUrl,
  body: {
    content: input.message,
    embeds: input.embeds || []
  }
};`,
    postProcess: `return {
  success: input.status < 400,
  status: input.status
};`,
  },
  {
    id: 'slack-notify',
    name: 'Slack通知',
    description: 'Slack Webhookにメッセージを送信',
    icon: '📢',
    displayCategory: 'notification',
    blockCategory: 'apps',
    inheritsFrom: 'HTTP',
    parentBlockSlug: 'http',
    configDefaults: {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
    },
    preProcess: `const webhookUrl = ctx.secrets.SLACK_WEBHOOK_URL || config.webhook_url;
return {
  url: webhookUrl,
  body: {
    text: input.message,
    channel: config.channel,
    username: config.username
  }
};`,
  },
  {
    id: 'json-transformer',
    name: 'JSONトランスフォーマー',
    description: 'JSONデータを変換するカスタムブロック',
    icon: '🔄',
    displayCategory: 'data',
    blockCategory: 'flow',
    code: `// input: 変換対象のデータ
// config.mapping: 変換マッピング定義

const mapping = config.mapping || {};
const result = {};

for (const [newKey, sourcePath] of Object.entries(mapping)) {
  result[newKey] = getPath(input, sourcePath);
}

return result;`,
    configSchema: {
      type: 'object',
      properties: {
        mapping: {
          type: 'object',
          title: 'フィールドマッピング',
          description: '変換先キー → 変換元パス',
        },
      },
    },
  },
  {
    id: 'data-validator',
    name: 'データバリデーター',
    description: '入力データの検証を行うブロック',
    icon: '✅',
    displayCategory: 'data',
    blockCategory: 'flow',
    code: `// config.rules: 検証ルール配列
// { field: string, type: string, required?: boolean }

const rules = config.rules || [];
const errors = [];

for (const rule of rules) {
  const value = getPath(input, rule.field);

  if (rule.required && (value === undefined || value === null)) {
    errors.push(rule.field + ' is required');
    continue;
  }

  if (value !== undefined && rule.type) {
    const actualType = typeof value;
    if (actualType !== rule.type) {
      errors.push(rule.field + ' should be ' + rule.type + ', got ' + actualType);
    }
  }
}

return {
  valid: errors.length === 0,
  errors: errors,
  data: input
};`,
    configSchema: {
      type: 'object',
      properties: {
        rules: {
          type: 'array',
          title: '検証ルール',
          items: {
            type: 'object',
            properties: {
              field: { type: 'string', title: 'フィールド' },
              type: { type: 'string', title: 'タイプ', enum: ['string', 'number', 'boolean', 'object', 'array'] },
              required: { type: 'boolean', title: '必須' },
            },
          },
        },
      },
    },
  },
  {
    id: 'error-handler',
    name: 'エラーハンドラー',
    description: 'エラーをフォーマットして通知用に整形',
    icon: '⚠️',
    displayCategory: 'utility',
    blockCategory: 'flow',
    code: `const error = input.error || input;

return {
  error_type: error.code || 'UNKNOWN_ERROR',
  message: error.message || String(error),
  timestamp: new Date().toISOString(),
  context: {
    workflow_id: ctx.workflow?.id,
    run_id: ctx.run?.id
  }
};`,
  },
]

// Group templates by displayCategory
const groupedTemplates = computed(() => {
  const groups: Record<string, TemplateDefinition[]> = {}

  for (const template of templates) {
    if (!groups[template.displayCategory]) {
      groups[template.displayCategory] = []
    }
    groups[template.displayCategory].push(template)
  }

  return groups
})

// Category labels
const categoryLabels: Record<string, string> = {
  notification: '通知',
  data: 'データ処理',
  utility: 'ユーティリティ',
}

// Navigation
function selectType(type: 'scratch' | 'inherit' | 'template') {
  creationType.value = type
  step.value = type === 'template' ? 1 : 2
}

function selectTemplate(template: TemplateDefinition) {
  selectedTemplate.value = template
  step.value = 2
}

function goBack() {
  if (step.value === 2 && creationType.value === 'template') {
    step.value = 1
    selectedTemplate.value = null
  } else if (step.value === 1 || step.value === 2) {
    step.value = 0
    creationType.value = null
  }
}

// Handle form submission
async function handleFormSubmit(formData: BlockFormData) {
  try {
    loading.value = true
    error.value = null

    // Parse JSON fields
    let configSchema, uiConfig, configDefaults
    try {
      configSchema = JSON.parse(formData.config_schema || '{}')
      uiConfig = JSON.parse(formData.ui_config || '{}')
      configDefaults = formData.config_defaults ? JSON.parse(formData.config_defaults) : undefined
    } catch {
      error.value = t('blockEditor.errors.invalidJson')
      return
    }

    const response = await blocksApi.create({
      slug: formData.slug,
      name: formData.name,
      description: formData.description || undefined,
      category: formData.category as BlockCategory,
      icon: formData.icon || undefined,
      code: formData.code || undefined,
      config_schema: configSchema,
      ui_config: uiConfig,
      // Note: parent_block_id and other inheritance fields would need backend support
    })

    emit('complete', response.data)
  } catch (err) {
    error.value = err instanceof Error ? err.message : t('errors.generic')
    console.error('Failed to create block:', err)
  } finally {
    loading.value = false
  }
}

// Get template data for form
const templateFormData = computed(() => {
  if (!selectedTemplate.value) return undefined

  return {
    name: selectedTemplate.value.name,
    description: selectedTemplate.value.description,
    icon: selectedTemplate.value.icon,
    category: selectedTemplate.value.blockCategory,
    code: selectedTemplate.value.code || '',
    config_schema: JSON.stringify(selectedTemplate.value.configSchema || {}, null, 2),
    config_defaults: JSON.stringify(selectedTemplate.value.configDefaults || {}, null, 2),
    pre_process: selectedTemplate.value.preProcess || '',
    post_process: selectedTemplate.value.postProcess || '',
  }
})

// Import BlockFormData type
interface BlockFormData {
  slug: string
  name: string
  description: string
  category: string
  icon: string
  code: string
  config_schema: string
  ui_config: string
  change_summary: string
  parent_block_id?: string
  config_defaults?: string
  pre_process?: string
  post_process?: string
}
</script>

<template>
  <div class="creation-wizard">
    <!-- Error message -->
    <div v-if="error" class="error-message">
      {{ error }}
    </div>

    <!-- Step 0: Type Selection -->
    <div v-if="step === 0" class="type-selection">
      <h2 class="wizard-title">{{ t('blockEditor.createBlockTitle') }}</h2>
      <p class="wizard-subtitle">{{ t('blockEditor.createBlockSubtitle') }}</p>

      <div class="type-cards">
        <!-- From Scratch -->
        <div
          class="type-card"
          :class="{ selected: creationType === 'scratch' }"
          @click="selectType('scratch')"
        >
          <div class="card-icon">&#10133;</div>
          <h3 class="card-title">{{ t('blockEditor.fromScratch') }}</h3>
          <p class="card-description">{{ t('blockEditor.fromScratchDesc') }}</p>
          <ul class="card-features">
            <li>{{ t('blockEditor.featureFullControl') }}</li>
            <li>{{ t('blockEditor.featureCustomCode') }}</li>
            <li>{{ t('blockEditor.featureCustomSchema') }}</li>
          </ul>
        </div>

        <!-- Inherit -->
        <div
          class="type-card recommended"
          :class="{ selected: creationType === 'inherit' }"
          @click="selectType('inherit')"
        >
          <div class="recommended-badge">{{ t('blockEditor.recommended') }}</div>
          <div class="card-icon">&#128279;</div>
          <h3 class="card-title">{{ t('blockEditor.inheritBlock') }}</h3>
          <p class="card-description">{{ t('blockEditor.inheritBlockDesc') }}</p>
          <ul class="card-features">
            <li>{{ t('blockEditor.featureReuseCode') }}</li>
            <li>{{ t('blockEditor.featureOverrideDefaults') }}</li>
            <li>{{ t('blockEditor.featureTransformIO') }}</li>
          </ul>
        </div>

        <!-- Template -->
        <div
          class="type-card"
          :class="{ selected: creationType === 'template' }"
          @click="selectType('template')"
        >
          <div class="card-icon">&#128196;</div>
          <h3 class="card-title">{{ t('blockEditor.fromTemplate') }}</h3>
          <p class="card-description">{{ t('blockEditor.fromTemplateDesc') }}</p>
          <ul class="card-features">
            <li>{{ t('blockEditor.featureQuickStart') }}</li>
            <li>{{ t('blockEditor.featurePreBuilt') }}</li>
            <li>{{ t('blockEditor.featureCustomizable') }}</li>
          </ul>
        </div>
      </div>

      <div class="wizard-footer">
        <button class="btn btn-secondary" @click="emit('cancel')">
          {{ t('common.cancel') }}
        </button>
      </div>
    </div>

    <!-- Step 1: Template Selection (only for template type) -->
    <div v-else-if="step === 1 && creationType === 'template'" class="template-selection">
      <div class="selection-header">
        <button class="back-btn" @click="goBack">
          &#8592; {{ t('common.back') }}
        </button>
        <h2 class="wizard-title">{{ t('blockEditor.selectTemplate') }}</h2>
      </div>

      <div class="template-grid">
        <template v-for="(categoryTemplates, category) in groupedTemplates" :key="category">
          <div class="category-section">
            <h3 class="category-title">{{ categoryLabels[category] || category }}</h3>
            <div class="template-list">
              <div
                v-for="template in categoryTemplates"
                :key="template.id"
                class="template-card"
                @click="selectTemplate(template)"
              >
                <div class="template-icon">{{ template.icon }}</div>
                <div class="template-info">
                  <h4 class="template-name">{{ template.name }}</h4>
                  <p class="template-description">{{ template.description }}</p>
                  <span v-if="template.inheritsFrom" class="inherits-badge">
                    {{ template.inheritsFrom }} {{ t('blockEditor.inherits') }}
                  </span>
                </div>
              </div>
            </div>
          </div>
        </template>
      </div>
    </div>

    <!-- Step 2: Block Form -->
    <div v-else-if="step === 2" class="block-form-step">
      <BlockForm
        :creation-type="creationType || 'scratch'"
        :template-data="templateFormData"
        @submit="handleFormSubmit"
        @cancel="emit('cancel')"
        @back="goBack"
      />
    </div>

    <!-- Loading overlay -->
    <div v-if="loading" class="loading-overlay">
      <div class="loading-spinner">{{ t('common.loading') }}</div>
    </div>
  </div>
</template>

<style scoped>
.creation-wizard {
  position: relative;
  min-height: 400px;
}

.error-message {
  padding: 0.75rem 1rem;
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.2);
  border-radius: 0.375rem;
  color: #ef4444;
  margin-bottom: 1rem;
}

.wizard-title {
  font-size: 1.25rem;
  font-weight: 600;
  margin: 0 0 0.5rem 0;
  text-align: center;
}

.wizard-subtitle {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
  text-align: center;
  margin-bottom: 2rem;
}

/* Type Selection */
.type-selection {
  padding: 1rem;
}

.type-cards {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.type-card {
  position: relative;
  padding: 1.5rem;
  background: var(--color-surface);
  border: 2px solid var(--color-border);
  border-radius: 0.75rem;
  cursor: pointer;
  transition: all 0.15s;
}

.type-card:hover {
  border-color: var(--color-primary);
  box-shadow: 0 4px 12px rgba(99, 102, 241, 0.15);
}

.type-card.selected {
  border-color: var(--color-primary);
  background: rgba(99, 102, 241, 0.05);
}

.type-card.recommended {
  border-color: rgba(99, 102, 241, 0.3);
}

.recommended-badge {
  position: absolute;
  top: -0.5rem;
  right: 1rem;
  padding: 0.25rem 0.5rem;
  background: var(--color-primary);
  color: white;
  font-size: 0.6875rem;
  font-weight: 600;
  border-radius: 0.25rem;
}

.card-icon {
  font-size: 2rem;
  margin-bottom: 0.75rem;
}

.card-title {
  font-size: 1rem;
  font-weight: 600;
  margin: 0 0 0.5rem 0;
}

.card-description {
  font-size: 0.8125rem;
  color: var(--color-text-secondary);
  margin-bottom: 1rem;
}

.card-features {
  margin: 0;
  padding-left: 1.25rem;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

.card-features li {
  margin-bottom: 0.25rem;
}

/* Template Selection */
.template-selection {
  padding: 1rem;
}

.selection-header {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.back-btn {
  padding: 0.5rem 0.75rem;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 0.375rem;
  font-size: 0.875rem;
  cursor: pointer;
}

.back-btn:hover {
  background: var(--color-background);
}

.template-grid {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.category-section {
  margin-bottom: 0.5rem;
}

.category-title {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--color-text-secondary);
  margin: 0 0 0.75rem 0;
  text-transform: uppercase;
}

.template-list {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 0.75rem;
}

.template-card {
  display: flex;
  gap: 0.75rem;
  padding: 1rem;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 0.5rem;
  cursor: pointer;
  transition: all 0.15s;
}

.template-card:hover {
  border-color: var(--color-primary);
  box-shadow: 0 2px 8px rgba(99, 102, 241, 0.1);
}

.template-icon {
  font-size: 1.5rem;
}

.template-info {
  flex: 1;
}

.template-name {
  font-size: 0.875rem;
  font-weight: 600;
  margin: 0 0 0.25rem 0;
}

.template-description {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  margin: 0;
}

.inherits-badge {
  display: inline-block;
  margin-top: 0.5rem;
  padding: 0.125rem 0.375rem;
  background: rgba(34, 197, 94, 0.1);
  color: #16a34a;
  font-size: 0.6875rem;
  border-radius: 0.25rem;
}

/* Footer */
.wizard-footer {
  display: flex;
  justify-content: flex-end;
  padding-top: 1rem;
  border-top: 1px solid var(--color-border);
}

/* Loading */
.loading-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(255, 255, 255, 0.8);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10;
}

.loading-spinner {
  padding: 1rem 2rem;
  background: var(--color-surface);
  border-radius: 0.5rem;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

/* Buttons */
.btn {
  padding: 0.5rem 1rem;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
}

.btn-secondary {
  background: var(--color-surface);
  color: var(--color-text);
  border: 1px solid var(--color-border);
}

.btn-secondary:hover {
  background: var(--color-background);
}
</style>
