// Block Package management composable (Custom Block SDK)
import type {
  CustomBlockPackage,
  PackageBlockDefinition,
  PackageDependency,
  BlockPackageStatus,
  PaginatedList,
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

  // Create a new block package
  async function createPackage(input: CreateBlockPackageInput): Promise<CustomBlockPackage> {
    return api.post<CustomBlockPackage>('/api/v1/block-packages', input)
  }

  // Get a block package by ID
  async function getPackage(id: string): Promise<CustomBlockPackage> {
    return api.get<CustomBlockPackage>(`/api/v1/block-packages/${id}`)
  }

  // List block packages
  async function listPackages(input: ListBlockPackagesInput = {}): Promise<PaginatedList<CustomBlockPackage>> {
    const params = new URLSearchParams()
    if (input.page) params.set('page', input.page.toString())
    if (input.limit) params.set('limit', input.limit.toString())
    if (input.status) params.set('status', input.status)
    if (input.search) params.set('search', input.search)
    const query = params.toString()
    return api.get<PaginatedList<CustomBlockPackage>>(`/api/v1/block-packages${query ? '?' + query : ''}`)
  }

  // Update a block package
  async function updatePackage(id: string, input: UpdateBlockPackageInput): Promise<CustomBlockPackage> {
    return api.put<CustomBlockPackage>(`/api/v1/block-packages/${id}`, input)
  }

  // Delete a block package
  async function deletePackage(id: string): Promise<void> {
    return api.delete<void>(`/api/v1/block-packages/${id}`)
  }

  // Publish a block package
  async function publishPackage(id: string): Promise<CustomBlockPackage> {
    return api.post<CustomBlockPackage>(`/api/v1/block-packages/${id}/publish`)
  }

  // Deprecate a block package
  async function deprecatePackage(id: string): Promise<CustomBlockPackage> {
    return api.post<CustomBlockPackage>(`/api/v1/block-packages/${id}/deprecate`)
  }

  return {
    createPackage,
    getPackage,
    listPackages,
    updatePackage,
    deletePackage,
    publishPackage,
    deprecatePackage,
  }
}
