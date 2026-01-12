import type {
  BlockTemplate,
  CreateBlockTemplateRequest,
  UpdateBlockTemplateRequest,
} from '~/types/api'

interface BlockTemplateListResponse {
  items: BlockTemplate[]
  total: number
}

export function useBlockTemplates() {
  const config = useRuntimeConfig()
  const baseUrl = config.public.apiBaseUrl || 'http://localhost:8080'

  const templates = ref<BlockTemplate[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  const getHeaders = () => {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    }
    return headers
  }

  const fetchTemplates = async () => {
    loading.value = true
    error.value = null
    try {
      const response = await fetch(`${baseUrl}/api/v1/admin/templates`, {
        headers: getHeaders(),
      })
      if (!response.ok) {
        const err = await response.json()
        throw new Error(err.error?.message || 'Failed to fetch templates')
      }
      const data: BlockTemplateListResponse = await response.json()
      templates.value = data.items || []
      return data
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Unknown error'
      throw e
    } finally {
      loading.value = false
    }
  }

  const getTemplate = async (id: string): Promise<BlockTemplate> => {
    const response = await fetch(`${baseUrl}/api/v1/admin/templates/${id}`, {
      headers: getHeaders(),
    })
    if (!response.ok) {
      const err = await response.json()
      throw new Error(err.error?.message || 'Failed to fetch template')
    }
    return response.json()
  }

  const getTemplateBySlug = async (slug: string): Promise<BlockTemplate> => {
    const response = await fetch(`${baseUrl}/api/v1/admin/templates/slug/${slug}`, {
      headers: getHeaders(),
    })
    if (!response.ok) {
      const err = await response.json()
      throw new Error(err.error?.message || 'Failed to fetch template')
    }
    return response.json()
  }

  const createTemplate = async (data: CreateBlockTemplateRequest): Promise<BlockTemplate> => {
    loading.value = true
    error.value = null
    try {
      const response = await fetch(`${baseUrl}/api/v1/admin/templates`, {
        method: 'POST',
        headers: getHeaders(),
        body: JSON.stringify(data),
      })
      if (!response.ok) {
        const err = await response.json()
        throw new Error(err.error?.message || 'Failed to create template')
      }
      const template = await response.json()
      templates.value.push(template)
      return template
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Unknown error'
      throw e
    } finally {
      loading.value = false
    }
  }

  const updateTemplate = async (id: string, data: UpdateBlockTemplateRequest): Promise<BlockTemplate> => {
    loading.value = true
    error.value = null
    try {
      const response = await fetch(`${baseUrl}/api/v1/admin/templates/${id}`, {
        method: 'PUT',
        headers: getHeaders(),
        body: JSON.stringify(data),
      })
      if (!response.ok) {
        const err = await response.json()
        throw new Error(err.error?.message || 'Failed to update template')
      }
      const template = await response.json()
      const index = templates.value.findIndex((t) => t.id === id)
      if (index !== -1) {
        templates.value[index] = template
      }
      return template
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Unknown error'
      throw e
    } finally {
      loading.value = false
    }
  }

  const deleteTemplate = async (id: string): Promise<void> => {
    loading.value = true
    error.value = null
    try {
      const response = await fetch(`${baseUrl}/api/v1/admin/templates/${id}`, {
        method: 'DELETE',
        headers: getHeaders(),
      })
      if (!response.ok) {
        const err = await response.json()
        throw new Error(err.error?.message || 'Failed to delete template')
      }
      templates.value = templates.value.filter((t) => t.id !== id)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Unknown error'
      throw e
    } finally {
      loading.value = false
    }
  }

  // Computed properties
  const builtinTemplates = computed(() => templates.value.filter((t) => t.is_builtin))
  const customTemplates = computed(() => templates.value.filter((t) => !t.is_builtin))

  return {
    templates,
    builtinTemplates,
    customTemplates,
    loading,
    error,
    fetchTemplates,
    getTemplate,
    getTemplateBySlug,
    createTemplate,
    updateTemplate,
    deleteTemplate,
  }
}
