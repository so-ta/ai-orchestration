// Block Package management composable (Custom Block SDK)
import type {
  CustomBlockPackage,
  PackageBlockDefinition,
  PackageDependency,
  BlockPackageStatus,
} from '~/types/api'

export interface CreateBlockPackageInput {
  name: string
  version: string
  description?: string
  blocks?: PackageBlockDefinition[]
  dependencies?: PackageDependency[]
}

export interface UpdateBlockPackageInput {
  description?: string
  blocks?: PackageBlockDefinition[]
  dependencies?: PackageDependency[]
  bundle_url?: string
}

export interface ListBlockPackagesInput {
  page?: number
  limit?: number
  status?: BlockPackageStatus
  search?: string
}

export function useBlockPackages() {
  const api = useApi()

  // Reactive state
  const packages = ref<CustomBlockPackage[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  // Fetch packages
  async function fetchPackages(input: ListBlockPackagesInput = {}): Promise<CustomBlockPackage[]> {
    loading.value = true
    error.value = null
    try {
      const params = new URLSearchParams()
      if (input.page) params.set('page', input.page.toString())
      if (input.limit) params.set('limit', input.limit.toString())
      if (input.status) params.set('status', input.status)
      if (input.search) params.set('search', input.search)
      const query = params.toString()
      const result = await api.get<CustomBlockPackage[]>(`/api/v1/block-packages${query ? '?' + query : ''}`)
      packages.value = result
      return result
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'Failed to fetch packages'
      throw e
    } finally {
      loading.value = false
    }
  }

  // Create a new block package
  async function createPackage(input: CreateBlockPackageInput): Promise<CustomBlockPackage> {
    const result = await api.post<CustomBlockPackage>('/api/v1/block-packages', input)
    packages.value = [...packages.value, result]
    return result
  }

  // Get a block package by ID
  async function getPackage(id: string): Promise<CustomBlockPackage> {
    return api.get<CustomBlockPackage>(`/api/v1/block-packages/${id}`)
  }

  // Update a block package
  async function updatePackage(id: string, input: UpdateBlockPackageInput): Promise<CustomBlockPackage> {
    const result = await api.put<CustomBlockPackage>(`/api/v1/block-packages/${id}`, input)
    packages.value = packages.value.map(p => p.id === id ? result : p)
    return result
  }

  // Delete a block package
  async function deletePackage(id: string): Promise<void> {
    await api.delete(`/api/v1/block-packages/${id}`)
    packages.value = packages.value.filter(p => p.id !== id)
  }

  // Publish a block package
  async function publishPackage(id: string): Promise<CustomBlockPackage> {
    const result = await api.post<CustomBlockPackage>(`/api/v1/block-packages/${id}/publish`)
    packages.value = packages.value.map(p => p.id === id ? result : p)
    return result
  }

  // Deprecate a block package
  async function deprecatePackage(id: string): Promise<CustomBlockPackage> {
    const result = await api.post<CustomBlockPackage>(`/api/v1/block-packages/${id}/deprecate`)
    packages.value = packages.value.map(p => p.id === id ? result : p)
    return result
  }

  return {
    // Reactive state
    packages,
    loading,
    error,
    // Methods
    fetchPackages,
    createPackage,
    getPackage,
    updatePackage,
    deletePackage,
    publishPackage,
    deprecatePackage,
  }
}
