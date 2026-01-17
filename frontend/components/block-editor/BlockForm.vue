<script setup lang="ts">
/**
 * BlockForm - 統合ブロック編集フォームコンポーネント
 *
 * システムブロック/カスタムブロックの作成・編集に共通で使用する統合フォーム。
 * ステップ形式で基本情報、継承設定、コード・スキーマ、テストを入力可能。
 */
import type { BlockDefinition, BlockCategory } from '~/types/api'
import { categoryConfig } from '~/composables/useBlocks'

// Template data interface (allows string JSON values for schemas)
interface TemplateFormData {
  name?: string
  description?: string
  icon?: string
  category?: BlockCategory
  code?: string
  config_schema?: string | object
  ui_config?: string | object
  config_defaults?: string | object
  pre_process?: string
  post_process?: string
}

const props = withDefaults(defineProps<{
  block?: BlockDefinition | null
  isEdit?: boolean
  creationType?: 'scratch' | 'inherit' | 'template'
  templateData?: TemplateFormData
}>(), {
  block: null,
  isEdit: false,
  creationType: 'scratch',
  templateData: undefined,
})

const emit = defineEmits<{
  submit: [data: BlockFormData]
  cancel: []
  back: []
}>()

const { t } = useI18n()

// Form data interface
export interface BlockFormData {
  slug: string
  name: string
  description: string
  category: BlockCategory
  icon: string
  code: string
  config_schema: string
  ui_config: string
  change_summary: string
  // Inheritance fields
  parent_block_id?: string
  config_defaults?: string
  pre_process?: string
  post_process?: string
}

// Steps
const steps = computed(() => [
  { id: 1, label: t('blockEditor.steps.basicInfo') },
  { id: 2, label: t('blockEditor.steps.inheritance') },
  { id: 3, label: t('blockEditor.steps.implementation') },
  { id: 4, label: t('blockEditor.steps.test') },
])

const currentStep = ref(1)
const useInheritance = ref(props.creationType === 'inherit')

// Parent block (when inheriting)
const parentBlock = ref<BlockDefinition | null>(null)

// Form state
const form = reactive<BlockFormData>({
  slug: '',
  name: '',
  description: '',
  category: 'custom' as BlockCategory,
  icon: '',
  code: '',
  config_schema: '{}',
  ui_config: '{}',
  change_summary: '',
  parent_block_id: undefined,
  config_defaults: '{}',
  pre_process: '',
  post_process: '',
})

// Initialize form from block or template
function initializeForm() {
  if (props.block) {
    form.slug = props.block.slug
    form.name = props.block.name
    form.description = props.block.description || ''
    form.category = props.block.category
    form.icon = props.block.icon || ''
    form.code = props.block.code || ''
    form.config_schema = JSON.stringify(props.block.config_schema || {}, null, 2)
    form.ui_config = JSON.stringify(props.block.ui_config || {}, null, 2)
    form.parent_block_id = props.block.parent_block_id
    form.config_defaults = JSON.stringify(props.block.config_defaults || {}, null, 2)
    form.pre_process = props.block.pre_process || ''
    form.post_process = props.block.post_process || ''
    useInheritance.value = !!props.block.parent_block_id
  } else if (props.templateData) {
    // Helper to convert object to JSON string
    const toJsonString = (val: string | object | undefined): string => {
      if (typeof val === 'string') return val
      return JSON.stringify(val || {}, null, 2)
    }

    Object.assign(form, {
      ...form,
      name: props.templateData.name || '',
      description: props.templateData.description || '',
      icon: props.templateData.icon || '',
      category: props.templateData.category || 'custom',
      code: props.templateData.code || '',
      config_schema: toJsonString(props.templateData.config_schema),
      ui_config: toJsonString(props.templateData.ui_config),
      config_defaults: toJsonString(props.templateData.config_defaults),
      pre_process: props.templateData.pre_process || '',
      post_process: props.templateData.post_process || '',
    })
    useInheritance.value = false
  }
}

onMounted(() => {
  initializeForm()
})

watch(() => props.block, () => {
  initializeForm()
})

// Categories
const categories: BlockCategory[] = ['ai', 'flow', 'apps', 'custom']

function getCategoryName(category: BlockCategory): string {
  const config = categoryConfig[category]
  return config ? t(config.nameKey) : category
}

// Auto-generate slug from name
function autoGenerateSlug() {
  if (form.name && !props.isEdit && !form.slug) {
    form.slug = form.name
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, '_')
      .replace(/^_+|_+$/g, '')
  }
}

// Navigation
function nextStep() {
  if (currentStep.value < 4) {
    // Skip inheritance step if not using inheritance
    if (currentStep.value === 1 && !useInheritance.value) {
      currentStep.value = 3
    } else {
      currentStep.value++
    }
  }
}

function prevStep() {
  if (currentStep.value > 1) {
    // Skip inheritance step if not using inheritance
    if (currentStep.value === 3 && !useInheritance.value) {
      currentStep.value = 1
    } else {
      currentStep.value--
    }
  }
}

// Handle parent block selection
function onParentSelect(block: BlockDefinition) {
  parentBlock.value = block
  form.parent_block_id = block.id
}

// Local computed for pre/post process (ensures non-undefined)
const localPreProcess = computed({
  get: () => form.pre_process || '',
  set: (value: string) => { form.pre_process = value },
})

const localPostProcess = computed({
  get: () => form.post_process || '',
  set: (value: string) => { form.post_process = value },
})

// Validation
const formErrors = reactive<Record<string, string>>({})

function validateStep(step: number): boolean {
  formErrors.slug = ''
  formErrors.name = ''
  formErrors.category = ''
  formErrors.config_schema = ''
  formErrors.ui_config = ''

  if (step === 1) {
    if (!form.name.trim()) {
      formErrors.name = t('blockEditor.errors.nameRequired')
      return false
    }
    if (!form.slug.trim()) {
      formErrors.slug = t('blockEditor.errors.slugRequired')
      return false
    }
    if (!/^[a-z0-9_-]+$/.test(form.slug)) {
      formErrors.slug = t('blockEditor.errors.slugInvalid')
      return false
    }
  }

  if (step === 3) {
    try {
      JSON.parse(form.config_schema)
    } catch {
      formErrors.config_schema = t('blockEditor.errors.invalidJson')
      return false
    }
    try {
      JSON.parse(form.ui_config)
    } catch {
      formErrors.ui_config = t('blockEditor.errors.invalidJson')
      return false
    }
  }

  return true
}

// Submit
function handleSubmit() {
  if (!validateStep(currentStep.value)) return
  emit('submit', { ...form })
}

// Computed: resolved code for inherited blocks
const resolvedCode = computed(() => {
  if (parentBlock.value?.code) {
    return parentBlock.value.code
  }
  return ''
})

// Pre/Post process templates
const preProcessTemplate = `// 入力変換: input と config を使用して親ブロックへの入力を作成
// Example:
// const webhookUrl = ctx.secrets.DISCORD_WEBHOOK_URL || config.webhook_url;
// return {
//   url: webhookUrl,
//   body: { content: input.message }
// };

return input;`

const postProcessTemplate = `// 出力変換: 親ブロックの出力を変換
// Example:
// return {
//   success: input.status < 400,
//   data: input.body
// };

return input;`
</script>

<template>
  <div class="block-form">
    <!-- Step Indicator -->
    <div class="step-indicator">
      <div
        v-for="step in steps"
        :key="step.id"
        class="step-item"
        :class="{
          active: currentStep === step.id,
          completed: currentStep > step.id,
          skipped: step.id === 2 && !useInheritance
        }"
      >
        <div class="step-number">
          <span v-if="currentStep > step.id">&#10003;</span>
          <span v-else>{{ step.id }}</span>
        </div>
        <div class="step-label">{{ step.label }}</div>
      </div>
    </div>

    <!-- Step 1: Basic Info -->
    <section v-if="currentStep === 1" class="form-section">
      <h3 class="section-title">{{ t('blockEditor.sections.basicInfo') }}</h3>

      <div class="form-row">
        <div class="form-group">
          <label class="form-label">{{ t('blockEditor.fields.name') }} *</label>
          <input
            v-model="form.name"
            type="text"
            class="form-input"
            :placeholder="t('blockEditor.placeholders.name')"
            @blur="autoGenerateSlug"
          >
          <span v-if="formErrors.name" class="form-error">{{ formErrors.name }}</span>
        </div>
        <div class="form-group">
          <label class="form-label">Slug *</label>
          <input
            v-model="form.slug"
            type="text"
            class="form-input"
            placeholder="e.g., my_custom_block"
            pattern="[a-z0-9_-]+"
            :disabled="isEdit"
          >
          <span class="form-hint">{{ t('blockEditor.hints.slug') }}</span>
          <span v-if="formErrors.slug" class="form-error">{{ formErrors.slug }}</span>
        </div>
      </div>

      <div class="form-group">
        <label class="form-label">{{ t('blockEditor.fields.description') }}</label>
        <textarea
          v-model="form.description"
          class="form-input"
          :placeholder="t('blockEditor.placeholders.description')"
          rows="2"
        />
      </div>

      <div class="form-row">
        <div class="form-group">
          <label class="form-label">{{ t('blockEditor.fields.category') }} *</label>
          <select v-model="form.category" class="form-input">
            <option v-for="cat in categories" :key="cat" :value="cat">
              {{ getCategoryName(cat) }}
            </option>
          </select>
        </div>
        <div class="form-group">
          <label class="form-label">{{ t('blockEditor.fields.icon') }}</label>
          <input
            v-model="form.icon"
            type="text"
            class="form-input"
            placeholder="e.g., message-circle"
          >
        </div>
      </div>

      <!-- Inheritance Toggle (only for new blocks) -->
      <div v-if="!isEdit" class="inheritance-toggle">
        <label class="toggle-label">
          <input v-model="useInheritance" type="checkbox" class="toggle-checkbox" >
          <span class="toggle-text">{{ t('blockEditor.useInheritance') }}</span>
        </label>
        <p class="toggle-hint">{{ t('blockEditor.useInheritanceHint') }}</p>
      </div>
    </section>

    <!-- Step 2: Inheritance Settings -->
    <section v-if="currentStep === 2 && useInheritance" class="form-section">
      <h3 class="section-title">{{ t('blockEditor.sections.inheritance') }}</h3>

      <!-- Parent Block Selector -->
      <div class="form-group">
        <label class="form-label">{{ t('blockEditor.fields.parentBlock') }} *</label>
        <BlockSelector
          v-model="form.parent_block_id"
          @select="onParentSelect"
        />
      </div>

      <!-- Inheritance Chain -->
      <InheritanceChain
        v-if="form.parent_block_id"
        :block-id="form.parent_block_id"
      />

      <!-- Config Defaults -->
      <div v-if="parentBlock" class="form-group">
        <label class="form-label">{{ t('blockEditor.fields.configDefaults') }}</label>
        <p class="form-hint">{{ t('blockEditor.hints.configDefaults') }}</p>
        <textarea
          v-model="form.config_defaults"
          class="form-input code-editor"
          rows="8"
          spellcheck="false"
          :placeholder="JSON.stringify(parentBlock.config_schema, null, 2)"
        />
      </div>

      <!-- Pre/Post Process -->
      <ProcessEditor
        v-model:pre-process="localPreProcess"
        v-model:post-process="localPostProcess"
        :pre-process-template="preProcessTemplate"
        :post-process-template="postProcessTemplate"
      />
    </section>

    <!-- Step 3: Implementation -->
    <section v-if="currentStep === 3" class="form-section">
      <h3 class="section-title">{{ t('blockEditor.sections.implementation') }}</h3>

      <!-- Code (inherited or custom) -->
      <div v-if="useInheritance && parentBlock" class="form-group">
        <label class="form-label">{{ t('blockEditor.fields.codeInherited') }}</label>
        <div class="inherited-code-notice">
          <span class="notice-icon">&#8635;</span>
          {{ t('blockEditor.inheritedCodeNotice', { parent: parentBlock.name }) }}
        </div>
        <textarea
          :value="resolvedCode"
          class="form-input code-editor readonly"
          rows="12"
          readonly
          spellcheck="false"
        />
      </div>
      <div v-else class="form-group">
        <label class="form-label">{{ t('blockEditor.fields.code') }} (JavaScript)</label>
        <textarea
          v-model="form.code"
          class="form-input code-editor"
          rows="12"
          spellcheck="false"
          :placeholder="t('blockEditor.placeholders.code')"
        />
      </div>

      <!-- Schemas -->
      <div class="form-row">
        <div class="form-group">
          <label class="form-label">{{ t('blockEditor.fields.configSchema') }} (JSON)</label>
          <textarea
            v-model="form.config_schema"
            class="form-input code-editor"
            rows="8"
            spellcheck="false"
          />
          <span v-if="formErrors.config_schema" class="form-error">{{ formErrors.config_schema }}</span>
        </div>
        <div class="form-group">
          <label class="form-label">{{ t('blockEditor.fields.uiConfig') }} (JSON)</label>
          <textarea
            v-model="form.ui_config"
            class="form-input code-editor"
            rows="8"
            spellcheck="false"
          />
          <span v-if="formErrors.ui_config" class="form-error">{{ formErrors.ui_config }}</span>
        </div>
      </div>
    </section>

    <!-- Step 4: Test & Confirm -->
    <section v-if="currentStep === 4" class="form-section">
      <h3 class="section-title">{{ t('blockEditor.sections.test') }}</h3>

      <BlockTestRunner
        :block-data="form"
        :parent-block="parentBlock"
        :use-inheritance="useInheritance"
      />

      <div class="form-group">
        <label class="form-label">{{ t('blockEditor.fields.changeSummary') }}</label>
        <input
          v-model="form.change_summary"
          type="text"
          class="form-input"
          :placeholder="t('blockEditor.placeholders.changeSummary')"
        >
      </div>
    </section>

    <!-- Footer -->
    <div class="form-footer">
      <button v-if="currentStep === 1 && !isEdit" class="btn btn-secondary" @click="emit('back')">
        {{ t('common.back') }}
      </button>
      <button v-else-if="currentStep > 1" class="btn btn-secondary" @click="prevStep">
        {{ t('common.previous') }}
      </button>
      <div class="footer-spacer"/>
      <button class="btn btn-secondary" @click="emit('cancel')">
        {{ t('common.cancel') }}
      </button>
      <button v-if="currentStep < 4" class="btn btn-primary" @click="nextStep">
        {{ t('common.next') }}
      </button>
      <button v-else class="btn btn-primary" @click="handleSubmit">
        {{ isEdit ? t('common.save') : t('common.create') }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.block-form {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

/* Step Indicator */
.step-indicator {
  display: flex;
  justify-content: space-between;
  padding: 0 1rem;
  margin-bottom: 1rem;
}

.step-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
  flex: 1;
  position: relative;
}

.step-item:not(:last-child)::after {
  content: '';
  position: absolute;
  top: 1rem;
  left: 60%;
  width: 80%;
  height: 2px;
  background: var(--color-border);
}

.step-item.completed:not(:last-child)::after {
  background: var(--color-primary);
}

.step-number {
  width: 2rem;
  height: 2rem;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.875rem;
  font-weight: 600;
  background: var(--color-surface);
  border: 2px solid var(--color-border);
  color: var(--color-text-secondary);
  z-index: 1;
}

.step-item.active .step-number {
  background: var(--color-primary);
  border-color: var(--color-primary);
  color: white;
}

.step-item.completed .step-number {
  background: var(--color-primary);
  border-color: var(--color-primary);
  color: white;
}

.step-item.skipped .step-number {
  opacity: 0.4;
}

.step-label {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  text-align: center;
}

.step-item.active .step-label {
  color: var(--color-primary);
  font-weight: 500;
}

.step-item.skipped .step-label {
  opacity: 0.4;
}

/* Form Section */
.form-section {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 0.5rem;
  padding: 1.5rem;
}

.section-title {
  font-size: 1rem;
  font-weight: 600;
  margin: 0 0 1rem 0;
  padding-bottom: 0.75rem;
  border-bottom: 1px solid var(--color-border);
}

/* Form Elements */
.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
  margin-bottom: 1rem;
}

.form-group:last-child {
  margin-bottom: 0;
}

.form-label {
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--color-text);
}

.form-input {
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--color-border);
  border-radius: 0.375rem;
  background: var(--color-background);
  color: var(--color-text);
  font-size: 0.875rem;
  transition: border-color 0.15s;
}

.form-input:focus {
  outline: none;
  border-color: var(--color-primary);
}

.form-input:disabled {
  background: var(--color-surface);
  color: var(--color-text-secondary);
  cursor: not-allowed;
}

textarea.form-input {
  resize: vertical;
  min-height: 60px;
}

.code-editor {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.8125rem;
  line-height: 1.5;
}

.code-editor.readonly {
  background: var(--color-surface);
  color: var(--color-text-secondary);
}

.form-hint {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

.form-error {
  font-size: 0.75rem;
  color: #ef4444;
}

/* Inheritance Toggle */
.inheritance-toggle {
  padding: 1rem;
  background: rgba(99, 102, 241, 0.05);
  border: 1px solid rgba(99, 102, 241, 0.2);
  border-radius: 0.5rem;
  margin-top: 0.5rem;
}

.toggle-label {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
}

.toggle-checkbox {
  width: 1rem;
  height: 1rem;
}

.toggle-text {
  font-weight: 500;
}

.toggle-hint {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  margin-top: 0.5rem;
  margin-bottom: 0;
}

/* Inherited Code Notice */
.inherited-code-notice {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem;
  background: rgba(34, 197, 94, 0.1);
  border: 1px solid rgba(34, 197, 94, 0.2);
  border-radius: 0.375rem;
  font-size: 0.875rem;
  color: #16a34a;
  margin-bottom: 0.5rem;
}

.notice-icon {
  font-size: 1rem;
}

/* Footer */
.form-footer {
  display: flex;
  gap: 0.75rem;
  padding-top: 1rem;
  border-top: 1px solid var(--color-border);
}

.footer-spacer {
  flex: 1;
}

.btn {
  padding: 0.5rem 1rem;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
}

.btn-primary {
  background: var(--color-primary);
  color: white;
  border: none;
}

.btn-primary:hover {
  opacity: 0.9;
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
