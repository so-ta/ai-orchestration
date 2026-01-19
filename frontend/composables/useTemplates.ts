// Template management composable
import type {
  ProjectTemplate,
  TemplateReview,
  TemplateCategory,
  TemplateVisibility,
  Project,
} from '~/types/api'

export interface CreateTemplateInput {
  name: string
  description?: string
  category?: string
  tags?: string[]
  definition?: Record<string, unknown>
  variables?: Record<string, unknown>
  author_name?: string
  visibility?: TemplateVisibility
}

export interface CreateFromProjectInput {
  project_id: string
  name?: string
  description?: string
  category?: string
  tags?: string[]
  author_name?: string
  visibility?: TemplateVisibility
}

export interface ListTemplatesInput {
  page?: number
  limit?: number
  category?: string
  search?: string
  scope?: 'my' | 'tenant' | 'public'
  featured?: boolean
  isFeatured?: boolean // alias for featured
}

export function useTemplates() {
  const api = useApi()

  // Reactive state
  const templates = ref<ProjectTemplate[]>([])
  const categories = ref<TemplateCategory[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  // Fetch templates
  async function fetchTemplates(input: ListTemplatesInput = {}): Promise<ProjectTemplate[]> {
    loading.value = true
    error.value = null
    try {
      const params = new URLSearchParams()
      if (input.page) params.set('page', input.page.toString())
      if (input.limit) params.set('limit', input.limit.toString())
      if (input.category) params.set('category', input.category)
      if (input.search) params.set('search', input.search)
      if (input.scope) params.set('scope', input.scope)
      const query = params.toString()
      const result = await api.get<ProjectTemplate[]>(`/api/v1/templates${query ? '?' + query : ''}`)
      templates.value = result
      return result
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'Failed to fetch templates'
      throw e
    } finally {
      loading.value = false
    }
  }

  // Fetch marketplace templates
  async function fetchMarketplace(input: ListTemplatesInput = {}): Promise<ProjectTemplate[]> {
    loading.value = true
    error.value = null
    try {
      const params = new URLSearchParams()
      if (input.page) params.set('page', input.page.toString())
      if (input.limit) params.set('limit', input.limit.toString())
      if (input.category) params.set('category', input.category)
      if (input.search) params.set('search', input.search)
      if (input.featured || input.isFeatured) params.set('featured', 'true')
      const query = params.toString()
      const result = await api.get<ProjectTemplate[]>(`/api/v1/marketplace/templates${query ? '?' + query : ''}`)
      templates.value = result
      return result
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'Failed to fetch marketplace templates'
      throw e
    } finally {
      loading.value = false
    }
  }

  // Fetch categories
  async function fetchCategories(): Promise<TemplateCategory[]> {
    try {
      const result = await api.get<TemplateCategory[]>('/api/v1/templates/categories')
      categories.value = result
      return result
    } catch {
      // Silently fail for categories
      return []
    }
  }

  // Create a new template
  async function createTemplate(input: CreateTemplateInput): Promise<ProjectTemplate> {
    return api.post<ProjectTemplate>('/api/v1/templates', input)
  }

  // Create a template from an existing project
  async function createFromProject(input: CreateFromProjectInput): Promise<ProjectTemplate> {
    return api.post<ProjectTemplate>('/api/v1/templates/from-project', input)
  }

  // Get a template by ID
  async function getTemplate(id: string): Promise<ProjectTemplate> {
    return api.get<ProjectTemplate>(`/api/v1/templates/${id}`)
  }

  // Update a template
  async function updateTemplate(id: string, input: Partial<CreateTemplateInput>): Promise<ProjectTemplate> {
    return api.put<ProjectTemplate>(`/api/v1/templates/${id}`, input)
  }

  // Delete a template
  async function deleteTemplate(id: string): Promise<void> {
    await api.delete(`/api/v1/templates/${id}`)
    templates.value = templates.value.filter(t => t.id !== id)
  }

  // Use a template to create a new project
  async function useTemplate(id: string, projectName?: string): Promise<Project> {
    return api.post<Project>(`/api/v1/templates/${id}/use`, { project_name: projectName })
  }

  // Get template reviews
  async function getReviews(templateId: string): Promise<TemplateReview[]> {
    return api.get<TemplateReview[]>(`/api/v1/templates/${templateId}/reviews`)
  }

  // Add a review to a template
  async function addReview(templateId: string, rating: number, comment?: string): Promise<TemplateReview> {
    return api.post<TemplateReview>(`/api/v1/templates/${templateId}/reviews`, { rating, comment })
  }

  // Get template categories (alias)
  async function getCategories(): Promise<TemplateCategory[]> {
    return fetchCategories()
  }

  return {
    // Reactive state
    templates,
    categories,
    loading,
    error,
    // Methods
    fetchTemplates,
    fetchMarketplace,
    fetchCategories,
    createTemplate,
    createFromProject,
    getTemplate,
    updateTemplate,
    deleteTemplate,
    useTemplate,
    getReviews,
    addReview,
    getCategories,
  }
}
