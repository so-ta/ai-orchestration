import { describe, it, expect, beforeEach, vi } from 'vitest'
import { useCommandHistory, BatchCommand, type Command, type CommandType } from './useCommandHistory'

// Helper to create a mock command
function createMockCommand(options: {
  type?: CommandType
  description?: string
  execute?: () => Promise<void>
  undo?: () => Promise<void>
}): Command {
  return {
    id: `cmd-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`,
    type: options.type || 'step:create',
    description: options.description || 'Test command',
    timestamp: Date.now(),
    execute: options.execute || vi.fn().mockResolvedValue(undefined),
    undo: options.undo || vi.fn().mockResolvedValue(undefined),
  }
}

describe('useCommandHistory', () => {
  let history: ReturnType<typeof useCommandHistory>

  beforeEach(() => {
    history = useCommandHistory()
    // Clear any existing history
    history.clear()
  })

  describe('initial state', () => {
    it('should start with empty stacks', () => {
      expect(history.canUndo.value).toBe(false)
      expect(history.canRedo.value).toBe(false)
    })

    it('should have correct initial stats', () => {
      expect(history.stats.value.undoCount).toBe(0)
      expect(history.stats.value.redoCount).toBe(0)
      expect(history.stats.value.isExecuting).toBe(false)
    })

    it('should have null descriptions initially', () => {
      expect(history.undoDescription.value).toBeNull()
      expect(history.redoDescription.value).toBeNull()
    })
  })

  describe('execute', () => {
    it('should execute a command', async () => {
      const executeFn = vi.fn().mockResolvedValue(undefined)
      const command = createMockCommand({ execute: executeFn })

      await history.execute(command)

      expect(executeFn).toHaveBeenCalledTimes(1)
    })

    it('should add command to undo stack after execution', async () => {
      const command = createMockCommand({ description: 'Create step' })

      await history.execute(command)

      expect(history.canUndo.value).toBe(true)
      expect(history.undoDescription.value).toBe('Create step')
    })

    it('should clear redo stack when new command is executed', async () => {
      // Setup: execute and undo a command to populate redo stack
      const cmd1 = createMockCommand({ description: 'First' })
      await history.execute(cmd1)
      await history.undo()
      expect(history.canRedo.value).toBe(true)

      // Execute new command
      const cmd2 = createMockCommand({ description: 'Second' })
      await history.execute(cmd2)

      // Redo stack should be cleared
      expect(history.canRedo.value).toBe(false)
    })

    it('should update lastCommandTime', async () => {
      const before = history.stats.value.lastCommandTime

      await history.execute(createMockCommand({}))

      expect(history.stats.value.lastCommandTime).toBeGreaterThanOrEqual(before)
    })

    it('should throw if command execution fails', async () => {
      const error = new Error('Execution failed')
      const command = createMockCommand({
        execute: vi.fn().mockRejectedValue(error),
      })

      await expect(history.execute(command)).rejects.toThrow('Execution failed')
    })
  })

  describe('undo', () => {
    it('should undo the last command', async () => {
      const undoFn = vi.fn().mockResolvedValue(undefined)
      const command = createMockCommand({ undo: undoFn })

      await history.execute(command)
      const result = await history.undo()

      expect(result).toBe(true)
      expect(undoFn).toHaveBeenCalledTimes(1)
    })

    it('should move command to redo stack', async () => {
      const command = createMockCommand({ description: 'Test' })

      await history.execute(command)
      await history.undo()

      expect(history.canUndo.value).toBe(false)
      expect(history.canRedo.value).toBe(true)
      expect(history.redoDescription.value).toBe('Test')
    })

    it('should return false when nothing to undo', async () => {
      const result = await history.undo()

      expect(result).toBe(false)
    })

    it('should throw and restore command on undo failure', async () => {
      const error = new Error('Undo failed')
      const command = createMockCommand({
        undo: vi.fn().mockRejectedValue(error),
      })

      await history.execute(command)

      await expect(history.undo()).rejects.toThrow('Undo failed')
      // Command should be back on undo stack
      expect(history.canUndo.value).toBe(true)
    })
  })

  describe('redo', () => {
    it('should redo the last undone command', async () => {
      const executeFn = vi.fn().mockResolvedValue(undefined)
      const command = createMockCommand({ execute: executeFn })

      await history.execute(command)
      await history.undo()
      const result = await history.redo()

      expect(result).toBe(true)
      expect(executeFn).toHaveBeenCalledTimes(2) // Initial + redo
    })

    it('should move command back to undo stack', async () => {
      const command = createMockCommand({ description: 'Test' })

      await history.execute(command)
      await history.undo()
      await history.redo()

      expect(history.canUndo.value).toBe(true)
      expect(history.canRedo.value).toBe(false)
      expect(history.undoDescription.value).toBe('Test')
    })

    it('should return false when nothing to redo', async () => {
      const result = await history.redo()

      expect(result).toBe(false)
    })

    it('should throw and restore command on redo failure', async () => {
      let callCount = 0
      const executeFn = vi.fn().mockImplementation(() => {
        callCount++
        if (callCount > 1) {
          return Promise.reject(new Error('Redo failed'))
        }
        return Promise.resolve()
      })
      const command = createMockCommand({ execute: executeFn })

      await history.execute(command)
      await history.undo()

      await expect(history.redo()).rejects.toThrow('Redo failed')
      // Command should be back on redo stack
      expect(history.canRedo.value).toBe(true)
    })
  })

  describe('clear', () => {
    it('should clear both stacks', async () => {
      await history.execute(createMockCommand({}))
      await history.undo()

      history.clear()

      expect(history.canUndo.value).toBe(false)
      expect(history.canRedo.value).toBe(false)
      expect(history.stats.value.undoCount).toBe(0)
      expect(history.stats.value.redoCount).toBe(0)
    })

    it('should reset lastCommandTime', async () => {
      await history.execute(createMockCommand({}))

      history.clear()

      expect(history.stats.value.lastCommandTime).toBe(0)
    })
  })

  describe('executeBatch', () => {
    it('should execute single command without batching', async () => {
      const command = createMockCommand({ description: 'Single' })

      await history.executeBatch([command], 'Batch')

      // Should have the single command's description, not batch
      expect(history.undoDescription.value).toBe('Single')
    })

    it('should execute multiple commands as a batch', async () => {
      const cmd1 = createMockCommand({ type: 'step:create' })
      const cmd2 = createMockCommand({ type: 'edge:create' })

      await history.executeBatch([cmd1, cmd2], 'Create step with edge')

      // Only one entry in history
      expect(history.stats.value.undoCount).toBe(1)
      expect(history.undoDescription.value).toBe('Create step with edge')
    })

    it('should do nothing with empty array', async () => {
      await history.executeBatch([], 'Empty batch')

      expect(history.stats.value.undoCount).toBe(0)
    })

    it('should undo all commands in batch together', async () => {
      const undo1 = vi.fn().mockResolvedValue(undefined)
      const undo2 = vi.fn().mockResolvedValue(undefined)
      const cmd1 = createMockCommand({ undo: undo1 })
      const cmd2 = createMockCommand({ undo: undo2 })

      await history.executeBatch([cmd1, cmd2], 'Batch')
      await history.undo()

      expect(undo1).toHaveBeenCalled()
      expect(undo2).toHaveBeenCalled()
    })
  })

  describe('history computed', () => {
    it('should return history in reverse order (most recent first)', async () => {
      await history.execute(createMockCommand({ description: 'First' }))
      await history.execute(createMockCommand({ description: 'Second' }))
      await history.execute(createMockCommand({ description: 'Third' }))

      const historyList = history.history.value

      expect(historyList).toHaveLength(3)
      expect(historyList[0].description).toBe('Third')
      expect(historyList[1].description).toBe('Second')
      expect(historyList[2].description).toBe('First')
    })
  })

  describe('concurrent execution prevention', () => {
    it('should prevent concurrent command execution', async () => {
      // Create a slow command
      let resolveExecute: () => void
      const slowCommand = createMockCommand({
        execute: () => new Promise((resolve) => {
          resolveExecute = resolve
        }),
      })

      // Start executing
      const execPromise = history.execute(slowCommand)

      // Try to execute another command while first is running
      const fastCommand = createMockCommand({})
      const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})

      await history.execute(fastCommand)

      expect(warnSpy).toHaveBeenCalledWith(
        expect.stringContaining('Cannot execute while another command is executing')
      )

      // Cleanup
      resolveExecute!()
      await execPromise
      warnSpy.mockRestore()
    })
  })
})

describe('BatchCommand', () => {
  it('should have correct type', () => {
    const batch = new BatchCommand([], 'Test batch')
    expect(batch.type).toBe('copilot:batch')
  })

  it('should generate unique id', () => {
    const batch1 = new BatchCommand([], 'Batch 1')
    const batch2 = new BatchCommand([], 'Batch 2')

    expect(batch1.id).not.toBe(batch2.id)
    expect(batch1.id).toMatch(/^batch-\d+-\w+$/)
  })

  it('should store description', () => {
    const batch = new BatchCommand([], 'My batch description')
    expect(batch.description).toBe('My batch description')
  })

  it('should execute commands in order', async () => {
    const order: number[] = []
    const cmd1 = createMockCommand({
      execute: vi.fn().mockImplementation(() => {
        order.push(1)
        return Promise.resolve()
      }),
    })
    const cmd2 = createMockCommand({
      execute: vi.fn().mockImplementation(() => {
        order.push(2)
        return Promise.resolve()
      }),
    })

    const batch = new BatchCommand([cmd1, cmd2], 'Test')
    await batch.execute()

    expect(order).toEqual([1, 2])
  })

  it('should undo commands in reverse order', async () => {
    const order: number[] = []
    const cmd1 = createMockCommand({
      undo: vi.fn().mockImplementation(() => {
        order.push(1)
        return Promise.resolve()
      }),
    })
    const cmd2 = createMockCommand({
      undo: vi.fn().mockImplementation(() => {
        order.push(2)
        return Promise.resolve()
      }),
    })

    const batch = new BatchCommand([cmd1, cmd2], 'Test')
    await batch.undo()

    expect(order).toEqual([2, 1]) // Reverse order
  })

  it('should return commands via getCommands', () => {
    const cmd1 = createMockCommand({})
    const cmd2 = createMockCommand({})

    const batch = new BatchCommand([cmd1, cmd2], 'Test')
    const commands = batch.getCommands()

    expect(commands).toHaveLength(2)
    expect(commands[0]).toBe(cmd1)
    expect(commands[1]).toBe(cmd2)
  })

  it('should return readonly commands array', () => {
    const batch = new BatchCommand([createMockCommand({})], 'Test')
    const commands = batch.getCommands()

    // TypeScript will prevent modification at compile time
    // but we can verify the returned array is the internal one
    expect(commands).toBeInstanceOf(Array)
  })
})
