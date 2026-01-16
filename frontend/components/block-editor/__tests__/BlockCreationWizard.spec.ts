import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { defineComponent, h } from 'vue'
import BlockCreationWizard from '../BlockCreationWizard.vue'

const messages = {
  blockEditor: {
    createBlockTitle: 'Create Block',
    createBlockSubtitle: 'Choose how you want to create your new block',
    fromScratch: 'From Scratch',
    fromScratchDesc: 'Create a completely new block with custom code',
    inheritBlock: 'Inherit Block',
    inheritBlockDesc: 'Extend an existing block with custom behavior',
    fromTemplate: 'From Template',
    fromTemplateDesc: 'Start with a pre-built template',
    recommended: 'Recommended',
    selectTemplate: 'Select a Template',
    featureFullControl: 'Full control over implementation',
    featureCustomCode: 'Write custom code',
    featureCustomSchema: 'Define custom schemas',
    featureReuseCode: 'Reuse existing code',
    featureOverrideDefaults: 'Override default config',
    featureTransformIO: 'Transform inputs/outputs',
    featureQuickStart: 'Quick start',
    featurePreBuilt: 'Pre-built logic',
    featureCustomizable: 'Fully customizable',
    inherits: 'inherits',
    errors: {
      invalidJson: 'Invalid JSON',
    },
  },
  common: {
    cancel: 'Cancel',
    back: 'Back',
    loading: 'Loading...',
  },
  errors: {
    generic: 'An error occurred',
  },
}

// Create i18n instance for tests
const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: {
    en: messages,
    ja: messages,
  },
})

// Mock useBlocks composable
const mockCreate = vi.fn()
vi.mock('~/composables/useBlocks', () => ({
  useBlocks: () => ({
    create: mockCreate,
  }),
}))

// Mock BlockForm component
const MockBlockForm = defineComponent({
  name: 'BlockForm',
  props: ['creationType', 'templateData'],
  emits: ['submit', 'cancel', 'back'],
  setup(props, { emit }) {
    return () =>
      h('div', { class: 'mock-block-form' }, [
        h('span', { class: 'creation-type' }, props.creationType),
        h('button', { class: 'mock-back-btn', onClick: () => emit('back') }, 'Back'),
        h('button', { class: 'mock-cancel-btn', onClick: () => emit('cancel') }, 'Cancel'),
        h('button', { class: 'mock-submit-btn', onClick: () => emit('submit', { slug: 'test' }) }, 'Submit'),
      ])
  },
})

describe('BlockCreationWizard', () => {
  const createWrapper = (props = {}) => {
    return mount(BlockCreationWizard, {
      props: { ...props },
      global: {
        plugins: [i18n],
        stubs: {
          BlockForm: MockBlockForm,
        },
      },
    })
  }

  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('Step 0: Type Selection', () => {
    it('renders initial type selection step', () => {
      const wrapper = createWrapper()

      expect(wrapper.text()).toContain('Create Block')
      expect(wrapper.text()).toContain('Choose how you want to create your new block')
      expect(wrapper.text()).toContain('From Scratch')
      expect(wrapper.text()).toContain('Inherit Block')
      expect(wrapper.text()).toContain('From Template')
    })

    it('renders three type cards', () => {
      const wrapper = createWrapper()

      const typeCards = wrapper.findAll('.type-card')
      expect(typeCards.length).toBe(3)
    })

    it('shows recommended badge on inherit card', () => {
      const wrapper = createWrapper()

      const recommendedBadge = wrapper.find('.recommended-badge')
      expect(recommendedBadge.exists()).toBe(true)
      expect(recommendedBadge.text()).toContain('Recommended')
    })

    it('emits cancel when cancel button clicked', async () => {
      const wrapper = createWrapper()

      const cancelBtn = wrapper.find('.btn-secondary')
      await cancelBtn.trigger('click')

      expect(wrapper.emitted('cancel')).toBeTruthy()
    })

    it('navigates to step 2 when scratch type is selected', async () => {
      const wrapper = createWrapper()

      const typeCards = wrapper.findAll('.type-card')
      await typeCards[0].trigger('click') // scratch

      // Should go directly to form step
      expect(wrapper.find('.mock-block-form').exists()).toBe(true)
    })

    it('navigates to step 2 when inherit type is selected', async () => {
      const wrapper = createWrapper()

      const typeCards = wrapper.findAll('.type-card')
      await typeCards[1].trigger('click') // inherit

      // Should go directly to form step
      expect(wrapper.find('.mock-block-form').exists()).toBe(true)
    })

    it('navigates to step 1 (template selection) when template type is selected', async () => {
      const wrapper = createWrapper()

      const typeCards = wrapper.findAll('.type-card')
      await typeCards[2].trigger('click') // template

      // Should show template selection, not form
      expect(wrapper.find('.template-selection').exists()).toBe(true)
      expect(wrapper.find('.mock-block-form').exists()).toBe(false)
    })
  })

  describe('Step 1: Template Selection', () => {
    it('renders template selection after selecting template type', async () => {
      const wrapper = createWrapper()

      // Select template type
      const typeCards = wrapper.findAll('.type-card')
      await typeCards[2].trigger('click')

      expect(wrapper.text()).toContain('Select a Template')
    })

    it('renders template categories', async () => {
      const wrapper = createWrapper()

      // Select template type
      const typeCards = wrapper.findAll('.type-card')
      await typeCards[2].trigger('click')

      // Check categories exist
      expect(wrapper.findAll('.category-section').length).toBeGreaterThan(0)
      expect(wrapper.findAll('.template-card').length).toBeGreaterThan(0)
    })

    it('renders back button', async () => {
      const wrapper = createWrapper()

      // Select template type
      const typeCards = wrapper.findAll('.type-card')
      await typeCards[2].trigger('click')

      const backBtn = wrapper.find('.back-btn')
      expect(backBtn.exists()).toBe(true)
    })

    it('goes back to step 0 when back button clicked', async () => {
      const wrapper = createWrapper()

      // Select template type
      const typeCards = wrapper.findAll('.type-card')
      await typeCards[2].trigger('click')

      // Click back
      const backBtn = wrapper.find('.back-btn')
      await backBtn.trigger('click')

      // Should be back to type selection
      expect(wrapper.find('.type-selection').exists()).toBe(true)
    })

    it('navigates to step 2 when template is selected', async () => {
      const wrapper = createWrapper()

      // Select template type
      const typeCards = wrapper.findAll('.type-card')
      await typeCards[2].trigger('click')

      // Select a template
      const templateCards = wrapper.findAll('.template-card')
      expect(templateCards.length).toBeGreaterThan(0)
      await templateCards[0].trigger('click')

      // Should now show form
      expect(wrapper.find('.mock-block-form').exists()).toBe(true)
    })
  })

  describe('Step 2: Block Form', () => {
    it('renders BlockForm when creation type is scratch', async () => {
      const wrapper = createWrapper()

      const typeCards = wrapper.findAll('.type-card')
      await typeCards[0].trigger('click')

      expect(wrapper.find('.mock-block-form').exists()).toBe(true)
    })

    it('passes correct creationType prop to BlockForm for scratch', async () => {
      const wrapper = createWrapper()

      const typeCards = wrapper.findAll('.type-card')
      await typeCards[0].trigger('click') // scratch

      expect(wrapper.find('.creation-type').text()).toBe('scratch')
    })

    it('passes correct creationType prop to BlockForm for inherit', async () => {
      const wrapper = createWrapper()

      const typeCards = wrapper.findAll('.type-card')
      await typeCards[1].trigger('click') // inherit

      expect(wrapper.find('.creation-type').text()).toBe('inherit')
    })
  })

  describe('Loading state', () => {
    it('does not show loading overlay initially', () => {
      const wrapper = createWrapper()

      expect(wrapper.find('.loading-overlay').exists()).toBe(false)
    })
  })

  describe('Error state', () => {
    it('does not show error message initially', () => {
      const wrapper = createWrapper()

      expect(wrapper.find('.error-message').exists()).toBe(false)
    })
  })

  describe('Templates', () => {
    it('has predefined templates with required fields', async () => {
      const wrapper = createWrapper()

      // Select template type
      const typeCards = wrapper.findAll('.type-card')
      await typeCards[2].trigger('click')

      // Check that templates are rendered with name and description
      const templateCards = wrapper.findAll('.template-card')
      expect(templateCards.length).toBeGreaterThan(0)

      // Each template should have icon, name, description
      templateCards.forEach((card) => {
        expect(card.find('.template-icon').exists()).toBe(true)
        expect(card.find('.template-name').exists()).toBe(true)
        expect(card.find('.template-description').exists()).toBe(true)
      })
    })

    it('shows inherits badge for templates that inherit from other blocks', async () => {
      const wrapper = createWrapper()

      // Select template type
      const typeCards = wrapper.findAll('.type-card')
      await typeCards[2].trigger('click')

      // Some templates should have inherits badge (e.g., Discord and Slack inherit from HTTP)
      const inheritsBadges = wrapper.findAll('.inherits-badge')
      expect(inheritsBadges.length).toBeGreaterThan(0)
    })
  })

  describe('Navigation flow', () => {
    it('completes full flow: type -> template -> form', async () => {
      const wrapper = createWrapper()

      // Step 0: Select template type
      expect(wrapper.find('.type-selection').exists()).toBe(true)
      const typeCards = wrapper.findAll('.type-card')
      await typeCards[2].trigger('click')

      // Step 1: Template selection
      expect(wrapper.find('.template-selection').exists()).toBe(true)
      const templateCards = wrapper.findAll('.template-card')
      await templateCards[0].trigger('click')

      // Step 2: Form
      expect(wrapper.find('.mock-block-form').exists()).toBe(true)
    })

    it('allows going back from form to template selection', async () => {
      const wrapper = createWrapper()

      // Navigate to form via template
      const typeCards = wrapper.findAll('.type-card')
      await typeCards[2].trigger('click')

      const templateCards = wrapper.findAll('.template-card')
      await templateCards[0].trigger('click')

      // Now in form, trigger back via BlockForm emit
      const mockBackBtn = wrapper.find('.mock-back-btn')
      await mockBackBtn.trigger('click')

      // Should be back to template selection
      expect(wrapper.find('.template-selection').exists()).toBe(true)
    })

    it('emits cancel when BlockForm emits cancel', async () => {
      const wrapper = createWrapper()

      // Navigate to form
      const typeCards = wrapper.findAll('.type-card')
      await typeCards[0].trigger('click')

      // Trigger cancel via BlockForm emit
      const mockCancelBtn = wrapper.find('.mock-cancel-btn')
      await mockCancelBtn.trigger('click')

      expect(wrapper.emitted('cancel')).toBeTruthy()
    })
  })
})
