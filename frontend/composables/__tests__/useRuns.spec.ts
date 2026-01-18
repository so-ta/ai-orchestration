import { describe, it, expect, vi, beforeEach } from 'vitest'

// Import after mocking
import { useRuns } from '../useRuns'

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

describe('useRuns', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('list', () => {
    it('should list runs for a project without params', async () => {
      const mockResponse = { data: [{ id: 'run-1', status: 'completed' }] }
      mockApi.get.mockResolvedValue(mockResponse)

      const { list } = useRuns()
      const result = await list('project-1')

      expect(mockApi.get).toHaveBeenCalledWith('/workflows/project-1/runs')
      expect(result).toEqual(mockResponse)
    })

    it('should list runs with pagination', async () => {
      mockApi.get.mockResolvedValue({ data: [] })

      const { list } = useRuns()
      await list('project-1', { page: 2, limit: 10 })

      expect(mockApi.get).toHaveBeenCalledWith('/workflows/project-1/runs?page=2&limit=10')
    })

    it('should list runs with only page param', async () => {
      mockApi.get.mockResolvedValue({ data: [] })

      const { list } = useRuns()
      await list('project-1', { page: 3 })

      expect(mockApi.get).toHaveBeenCalledWith('/workflows/project-1/runs?page=3')
    })

    it('should list runs with only limit param', async () => {
      mockApi.get.mockResolvedValue({ data: [] })

      const { list } = useRuns()
      await list('project-1', { limit: 25 })

      expect(mockApi.get).toHaveBeenCalledWith('/workflows/project-1/runs?limit=25')
    })
  })

  describe('get', () => {
    it('should get run by ID', async () => {
      const mockResponse = {
        data: {
          id: 'run-1',
          project_id: 'project-1',
          status: 'completed',
          input: { key: 'value' },
          output: { result: 'success' },
        },
      }
      mockApi.get.mockResolvedValue(mockResponse)

      const { get } = useRuns()
      const result = await get('run-1')

      expect(mockApi.get).toHaveBeenCalledWith('/runs/run-1')
      expect(result).toEqual(mockResponse)
    })
  })

  describe('create', () => {
    it('should create run with minimal data', async () => {
      const mockResponse = { data: { id: 'run-1', status: 'pending' } }
      mockApi.post.mockResolvedValue(mockResponse)

      const { create } = useRuns()
      const result = await create('project-1', { start_step_id: 'start-1' })

      expect(mockApi.post).toHaveBeenCalledWith('/workflows/project-1/runs', { start_step_id: 'start-1' })
      expect(result).toEqual(mockResponse)
    })

    it('should create run with input', async () => {
      mockApi.post.mockResolvedValue({ data: { id: 'run-1' } })

      const { create } = useRuns()
      await create('project-1', { input: { key: 'value' }, start_step_id: 'start-1' })

      expect(mockApi.post).toHaveBeenCalledWith('/workflows/project-1/runs', {
        input: { key: 'value' },
        start_step_id: 'start-1',
      })
    })

    it('should create run with test trigger', async () => {
      mockApi.post.mockResolvedValue({ data: { id: 'run-1' } })

      const { create } = useRuns()
      await create('project-1', { triggered_by: 'test', start_step_id: 'start-1' })

      expect(mockApi.post).toHaveBeenCalledWith('/workflows/project-1/runs', {
        triggered_by: 'test',
        start_step_id: 'start-1',
      })
    })

    it('should create run with manual trigger', async () => {
      mockApi.post.mockResolvedValue({ data: { id: 'run-1' } })

      const { create } = useRuns()
      await create('project-1', { triggered_by: 'manual', start_step_id: 'start-1' })

      expect(mockApi.post).toHaveBeenCalledWith('/workflows/project-1/runs', {
        triggered_by: 'manual',
        start_step_id: 'start-1',
      })
    })

    it('should create run with version', async () => {
      mockApi.post.mockResolvedValue({ data: { id: 'run-1' } })

      const { create } = useRuns()
      await create('project-1', { version: 2, start_step_id: 'start-1' })

      expect(mockApi.post).toHaveBeenCalledWith('/workflows/project-1/runs', {
        version: 2,
        start_step_id: 'start-1',
      })
    })

    it('should create run with all options', async () => {
      mockApi.post.mockResolvedValue({ data: { id: 'run-1' } })

      const { create } = useRuns()
      await create('project-1', {
        input: { data: 'test' },
        triggered_by: 'manual',
        version: 3,
        start_step_id: 'start-1',
      })

      expect(mockApi.post).toHaveBeenCalledWith('/workflows/project-1/runs', {
        input: { data: 'test' },
        triggered_by: 'manual',
        version: 3,
        start_step_id: 'start-1',
      })
    })
  })

  describe('cancel', () => {
    it('should cancel run', async () => {
      const mockResponse = { data: { id: 'run-1', status: 'cancelled' } }
      mockApi.post.mockResolvedValue(mockResponse)

      const { cancel } = useRuns()
      const result = await cancel('run-1')

      expect(mockApi.post).toHaveBeenCalledWith('/runs/run-1/cancel')
      expect(result).toEqual(mockResponse)
    })
  })

  describe('executeSingleStep', () => {
    it('should execute single step without input', async () => {
      const mockResponse = {
        data: { id: 'step-run-1', step_id: 'step-1', status: 'completed' },
      }
      mockApi.post.mockResolvedValue(mockResponse)

      const { executeSingleStep } = useRuns()
      const result = await executeSingleStep('run-1', 'step-1')

      expect(mockApi.post).toHaveBeenCalledWith('/runs/run-1/steps/step-1/execute', {
        input: undefined,
      })
      expect(result).toEqual(mockResponse)
    })

    it('should execute single step with input', async () => {
      const mockResponse = {
        data: { id: 'step-run-1', step_id: 'step-1', status: 'completed' },
      }
      mockApi.post.mockResolvedValue(mockResponse)

      const { executeSingleStep } = useRuns()
      const result = await executeSingleStep('run-1', 'step-1', { override: 'data' })

      expect(mockApi.post).toHaveBeenCalledWith('/runs/run-1/steps/step-1/execute', {
        input: { override: 'data' },
      })
      expect(result).toEqual(mockResponse)
    })
  })

  describe('resumeFromStep', () => {
    it('should resume from step without input override', async () => {
      const mockResponse = {
        data: {
          run_id: 'new-run-1',
          from_step_id: 'step-1',
          steps_to_execute: ['step-1', 'step-2', 'step-3'],
        },
      }
      mockApi.post.mockResolvedValue(mockResponse)

      const { resumeFromStep } = useRuns()
      const result = await resumeFromStep('run-1', 'step-1')

      expect(mockApi.post).toHaveBeenCalledWith('/runs/run-1/resume', {
        from_step_id: 'step-1',
        input_override: undefined,
      })
      expect(result).toEqual(mockResponse)
    })

    it('should resume from step with input override', async () => {
      const mockResponse = {
        data: {
          run_id: 'new-run-1',
          from_step_id: 'step-1',
          steps_to_execute: ['step-1', 'step-2'],
        },
      }
      mockApi.post.mockResolvedValue(mockResponse)

      const { resumeFromStep } = useRuns()
      const result = await resumeFromStep('run-1', 'step-1', { modified: 'input' })

      expect(mockApi.post).toHaveBeenCalledWith('/runs/run-1/resume', {
        from_step_id: 'step-1',
        input_override: { modified: 'input' },
      })
      expect(result).toEqual(mockResponse)
    })
  })

  describe('getStepHistory', () => {
    it('should get step history', async () => {
      const mockResponse = {
        data: [
          { id: 'step-run-1', step_id: 'step-1', status: 'completed', attempt: 1 },
          { id: 'step-run-2', step_id: 'step-1', status: 'failed', attempt: 2 },
          { id: 'step-run-3', step_id: 'step-1', status: 'completed', attempt: 3 },
        ],
      }
      mockApi.get.mockResolvedValue(mockResponse)

      const { getStepHistory } = useRuns()
      const result = await getStepHistory('run-1', 'step-1')

      expect(mockApi.get).toHaveBeenCalledWith('/runs/run-1/steps/step-1/history')
      expect(result).toEqual(mockResponse)
    })

    it('should return empty array when no history', async () => {
      const mockResponse = { data: [] }
      mockApi.get.mockResolvedValue(mockResponse)

      const { getStepHistory } = useRuns()
      const result = await getStepHistory('run-1', 'step-1')

      expect(mockApi.get).toHaveBeenCalledWith('/runs/run-1/steps/step-1/history')
      expect(result).toEqual(mockResponse)
    })
  })
})
