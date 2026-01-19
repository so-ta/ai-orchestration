// Template management composable
import type {
  ProjectTemplate,
  TemplateReview,
  TemplateCategory,
  TemplateVisibility,
  PaginatedList,
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
}

export function useTemplates() {
  const api = useApi()

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

  // List templates
  async function listTemplates(input: ListTemplatesInput = {}): Promise<PaginatedList<ProjectTemplate>> {
    const params = new URLSearchParams()
    if (input.page) params.set('page', input.page.toString())
    if (input.limit) params.set('limit', input.limit.toString())
    if (input.category) params.set('category', input.category)
    if (input.search) params.set('search', input.search)
    if (input.scope) params.set('scope', input.scope)
    const query = params.toString()
    return api.get<PaginatedList<ProjectTemplate>>(`/api/v1/templates${query ? '?' + query : ''}`)
  }

  // List marketplace templates
  async function listMarketplaceTemplates(input: ListTemplatesInput = {}): Promise<PaginatedList<ProjectTemplate>> {
    const params = new URLSearchParams()
    if (input.page) params.set('page', input.page.toString())
    if (input.limit) params.set('limit', input.limit.toString())
    if (input.category) params.set('category', input.category)
    if (input.search) params.set('search', input.search)
    if (input.featured) params.set('featured', 'true')
    const query = params.toString()
    return api.get<PaginatedList<ProjectTemplate>>(`/api/v1/marketplace/templates${query ? '?' + query : ''}`)
  }

  // Update a template
  async function updateTemplate(id: string, input: Partial<CreateTemplateInput>): Promise<ProjectTemplate> {
    return api.put<ProjectTemplate>(`/api/v1/templates/${id}`, input)
  }

  // Delete a template
  async function deleteTemplate(id: string): Promise<void> {
    return api.delete<void>(`/api/v1/templates/${id}`)
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

  // Get template categories
  async function getCategories(): Promise<TemplateCategory[]> {
    return api.get<TemplateCategory[]>('/api/v1/templates/categories')
  }

  return {
    createTemplate,
    createFromProject,
    getTemplate,
    listTemplates,
    listMarketplaceTemplates,
    updateTemplate,
    deleteTemplate,
    useTemplate,
    getReviews,
    addReview,
    getCategories,
  }
}
