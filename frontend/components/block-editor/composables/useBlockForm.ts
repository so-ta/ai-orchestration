/**
 * Composable for BlockForm state management and validation
 */
import type { BlockDefinition, BlockCategory } from '~/types/api'
import { categoryConfig } from '~/composables/useBlocks'

// Template data interface (allows string JSON values for schemas)
export interface TemplateFormData {
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

export interface UseBlockFormOptions {
  block: Ref<BlockDefinition | null | undefined>
  isEdit: Ref<boolean>
  creationType: Ref<'scratch' | 'inherit' | 'template'>
  templateData: Ref<TemplateFormData | undefined>
}

// Pre/Post process templates
export const preProcessTemplate = `// 入力変換: input と config を使用して親ブロックへの入力を作成
// Example:
// const webhookUrl = ctx.secrets.DISCORD_WEBHOOK_URL || config.webhook_url;
// return {
//   url: webhookUrl,
//   body: { content: input.message }
// };

return input;`

export const postProcessTemplate = `// 出力変換: 親ブロックの出力を変換
// Example:
// return {
//   success: input.status < 400,
//   data: input.body
// };

return input;`

export function useBlockForm(options: UseBlockFormOptions) {
  const { block, isEdit, creationType, templateData } = options
  const { t } = useI18n()

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

  // Step navigation
  const currentStep = ref(1)
  const useInheritance = ref(creationType.value === 'inherit')

  // Parent block (when inheriting)
  const parentBlock = ref<BlockDefinition | null>(null)

  // Validation errors
  const formErrors = reactive<Record<string, string>>({})

  // Categories
  const categories: BlockCategory[] = ['ai', 'flow', 'apps', 'custom']

  /**
   * Initialize form from block or template
   */
  function initializeForm() {
    if (block.value) {
      form.slug = block.value.slug
      form.name = block.value.name
      form.description = block.value.description || ''
      form.category = block.value.category
      form.icon = block.value.icon || ''
      form.code = block.value.code || ''
      form.config_schema = JSON.stringify(block.value.config_schema || {}, null, 2)
      form.ui_config = JSON.stringify(block.value.ui_config || {}, null, 2)
      form.parent_block_id = block.value.parent_block_id
      form.config_defaults = JSON.stringify(block.value.config_defaults || {}, null, 2)
      form.pre_process = block.value.pre_process || ''
      form.post_process = block.value.post_process || ''
      useInheritance.value = !!block.value.parent_block_id
    } else if (templateData.value) {
      // Helper to convert object to JSON string
      const toJsonString = (val: string | object | undefined): string => {
        if (typeof val === 'string') return val
        return JSON.stringify(val || {}, null, 2)
      }

      Object.assign(form, {
        ...form,
        name: templateData.value.name || '',
        description: templateData.value.description || '',
        icon: templateData.value.icon || '',
        category: templateData.value.category || 'custom',
        code: templateData.value.code || '',
        config_schema: toJsonString(templateData.value.config_schema),
        ui_config: toJsonString(templateData.value.ui_config),
        config_defaults: toJsonString(templateData.value.config_defaults),
        pre_process: templateData.value.pre_process || '',
        post_process: templateData.value.post_process || '',
      })
      useInheritance.value = false
    }
  }

  /**
   * Get category display name
   */
  function getCategoryName(category: BlockCategory): string {
    const config = categoryConfig[category]
    return config ? t(config.nameKey) : category
  }

  /**
   * Auto-generate slug from name
   */
  function autoGenerateSlug() {
    if (form.name && !isEdit.value && !form.slug) {
      form.slug = form.name
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, '_')
        .replace(/^_+|_+$/g, '')
    }
  }

  /**
   * Navigate to next step
   */
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

  /**
   * Navigate to previous step
   */
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

  /**
   * Handle parent block selection
   */
  function onParentSelect(selectedBlock: BlockDefinition) {
    parentBlock.value = selectedBlock
    form.parent_block_id = selectedBlock.id
  }

  /**
   * Validate current step
   */
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

  // Local computed for pre/post process (ensures non-undefined)
  const localPreProcess = computed({
    get: () => form.pre_process || '',
    set: (value: string) => { form.pre_process = value },
  })

  const localPostProcess = computed({
    get: () => form.post_process || '',
    set: (value: string) => { form.post_process = value },
  })

  // Computed: resolved code for inherited blocks
  const resolvedCode = computed(() => {
    if (parentBlock.value?.code) {
      return parentBlock.value.code
    }
    return ''
  })

  // Steps definition
  const steps = computed(() => [
    { id: 1, label: t('blockEditor.steps.basicInfo') },
    { id: 2, label: t('blockEditor.steps.inheritance') },
    { id: 3, label: t('blockEditor.steps.implementation') },
    { id: 4, label: t('blockEditor.steps.test') },
  ])

  return {
    // State
    form,
    formErrors,
    currentStep,
    useInheritance,
    parentBlock,
    steps,
    categories,
    // Computed
    localPreProcess,
    localPostProcess,
    resolvedCode,
    // Actions
    initializeForm,
    getCategoryName,
    autoGenerateSlug,
    nextStep,
    prevStep,
    onParentSelect,
    validateStep,
  }
}
