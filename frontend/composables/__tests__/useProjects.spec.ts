import { describe, it, expect, vi, beforeEach } from 'vitest'

// Import after mocking
import { useProjects } from '../useProjects'

// Mock useApi
const mockApi = {
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  delete: vi.fn(),
}

vi.mock('../useApi', () => ({
  useApi: () => mockApi,
}))

describe('useProjects', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('list', () => {
    it('should list projects without params', async () => {
      const mockResponse = { data: [{ id: '1', name: 'Project 1' }] }
      mockApi.get.mockResolvedValue(mockResponse)

      const { list } = useProjects()
      const result = await list()

      expect(mockApi.get).toHaveBeenCalledWith('/workflows')
      expect(result).toEqual(mockResponse)
    })

    it('should list projects with status filter', async () => {
      mockApi.get.mockResolvedValue({ data: [] })

      const { list } = useProjects()
      await list({ status: 'published' })

      expect(mockApi.get).toHaveBeenCalledWith('/workflows?status=published')
    })

    it('should list projects with pagination', async () => {
      mockApi.get.mockResolvedValue({ data: [] })

      const { list } = useProjects()
      await list({ page: 2, limit: 10 })

      expect(mockApi.get).toHaveBeenCalledWith('/workflows?page=2&limit=10')
    })
  })

  describe('get', () => {
    it('should get project by ID', async () => {
      const mockResponse = { data: { id: '1', name: 'Project 1' } }
      mockApi.get.mockResolvedValue(mockResponse)

      const { get } = useProjects()
      const result = await get('1')

      expect(mockApi.get).toHaveBeenCalledWith('/workflows/1')
      expect(result).toEqual(mockResponse)
    })
  })

  describe('create', () => {
    it('should create project', async () => {
      const mockResponse = { data: { id: '1', name: 'New Project' } }
      mockApi.post.mockResolvedValue(mockResponse)

      const { create } = useProjects()
      const result = await create({ name: 'New Project', description: 'Test' })

      expect(mockApi.post).toHaveBeenCalledWith('/workflows', {
        name: 'New Project',
        description: 'Test',
      })
      expect(result).toEqual(mockResponse)
    })
  })

  describe('update', () => {
    it('should update project', async () => {
      const mockResponse = { data: { id: '1', name: 'Updated Project' } }
      mockApi.put.mockResolvedValue(mockResponse)

      const { update } = useProjects()
      const result = await update('1', { name: 'Updated Project' })

      expect(mockApi.put).toHaveBeenCalledWith('/workflows/1', {
        name: 'Updated Project',
      })
      expect(result).toEqual(mockResponse)
    })
  })

  describe('remove', () => {
    it('should delete project', async () => {
      mockApi.delete.mockResolvedValue(undefined)

      const { remove } = useProjects()
      await remove('1')

      expect(mockApi.delete).toHaveBeenCalledWith('/workflows/1')
    })
  })

  describe('save', () => {
    it('should save project with steps and edges', async () => {
      const mockResponse = { data: { id: '1', version: 2 } }
      mockApi.post.mockResolvedValue(mockResponse)

      const { save } = useProjects()
      const saveData = {
        name: 'Project',
        description: 'Test',
        steps: [{ id: 's1', name: 'Step 1', type: 'start', config: {}, position_x: 0, position_y: 0 }],
        edges: [{ id: 'e1', source_step_id: 's1', target_step_id: 's2' }],
      }
      const result = await save('1', saveData)

      expect(mockApi.post).toHaveBeenCalledWith('/workflows/1/save', saveData)
      expect(result).toEqual(mockResponse)
    })
  })

  describe('saveDraft', () => {
    it('should save project as draft', async () => {
      const mockResponse = { data: { id: '1', has_draft: true } }
      mockApi.post.mockResolvedValue(mockResponse)

      const { saveDraft } = useProjects()
      const draftData = {
        name: 'Draft',
        steps: [],
        edges: [],
      }
      const result = await saveDraft('1', draftData)

      expect(mockApi.post).toHaveBeenCalledWith('/workflows/1/draft', draftData)
      expect(result).toEqual(mockResponse)
    })
  })

  describe('discardDraft', () => {
    it('should discard draft', async () => {
      const mockResponse = { data: { id: '1', has_draft: false } }
      mockApi.delete.mockResolvedValue(mockResponse)

      const { discardDraft } = useProjects()
      const result = await discardDraft('1')

      expect(mockApi.delete).toHaveBeenCalledWith('/workflows/1/draft')
      expect(result).toEqual(mockResponse)
    })
  })

  describe('steps', () => {
    it('should list steps', async () => {
      const mockResponse = { data: [{ id: 's1', name: 'Step 1' }] }
      mockApi.get.mockResolvedValue(mockResponse)

      const { listSteps } = useProjects()
      const result = await listSteps('p1')

      expect(mockApi.get).toHaveBeenCalledWith('/workflows/p1/steps')
      expect(result).toEqual(mockResponse)
    })

    it('should create step', async () => {
      const mockResponse = { data: { id: 's1', name: 'New Step' } }
      mockApi.post.mockResolvedValue(mockResponse)

      const { createStep } = useProjects()
      const result = await createStep('p1', { name: 'New Step', type: 'tool' })

      expect(mockApi.post).toHaveBeenCalledWith('/workflows/p1/steps', {
        name: 'New Step',
        type: 'tool',
      })
      expect(result).toEqual(mockResponse)
    })

    it('should update step', async () => {
      const mockResponse = { data: { id: 's1', name: 'Updated Step' } }
      mockApi.put.mockResolvedValue(mockResponse)

      const { updateStep } = useProjects()
      const result = await updateStep('p1', 's1', { name: 'Updated Step' })

      expect(mockApi.put).toHaveBeenCalledWith('/workflows/p1/steps/s1', {
        name: 'Updated Step',
      })
      expect(result).toEqual(mockResponse)
    })

    it('should delete step', async () => {
      mockApi.delete.mockResolvedValue(undefined)

      const { deleteStep } = useProjects()
      await deleteStep('p1', 's1')

      expect(mockApi.delete).toHaveBeenCalledWith('/workflows/p1/steps/s1')
    })
  })

  describe('edges', () => {
    it('should list edges', async () => {
      const mockResponse = { data: [{ id: 'e1' }] }
      mockApi.get.mockResolvedValue(mockResponse)

      const { listEdges } = useProjects()
      const result = await listEdges('p1')

      expect(mockApi.get).toHaveBeenCalledWith('/workflows/p1/edges')
      expect(result).toEqual(mockResponse)
    })

    it('should create edge', async () => {
      const mockResponse = { data: { id: 'e1' } }
      mockApi.post.mockResolvedValue(mockResponse)

      const { createEdge } = useProjects()
      const result = await createEdge('p1', {
        source_step_id: 's1',
        target_step_id: 's2',
      })

      expect(mockApi.post).toHaveBeenCalledWith('/workflows/p1/edges', {
        source_step_id: 's1',
        target_step_id: 's2',
      })
      expect(result).toEqual(mockResponse)
    })

    it('should delete edge', async () => {
      mockApi.delete.mockResolvedValue(undefined)

      const { deleteEdge } = useProjects()
      await deleteEdge('p1', 'e1')

      expect(mockApi.delete).toHaveBeenCalledWith('/workflows/p1/edges/e1')
    })
  })

  describe('versions', () => {
    it('should list versions', async () => {
      const mockResponse = { data: [{ version: 1 }, { version: 2 }] }
      mockApi.get.mockResolvedValue(mockResponse)

      const { listVersions } = useProjects()
      const result = await listVersions('p1')

      expect(mockApi.get).toHaveBeenCalledWith('/workflows/p1/versions')
      expect(result).toEqual(mockResponse)
    })

    it('should get specific version', async () => {
      const mockResponse = { data: { version: 1 } }
      mockApi.get.mockResolvedValue(mockResponse)

      const { getVersion } = useProjects()
      const result = await getVersion('p1', 1)

      expect(mockApi.get).toHaveBeenCalledWith('/workflows/p1/versions/1')
      expect(result).toEqual(mockResponse)
    })
  })
})
