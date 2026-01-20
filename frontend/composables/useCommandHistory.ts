/**
 * Command History composable for Undo/Redo functionality
 *
 * Implements the Command pattern for workflow editor operations.
 * Each command encapsulates an action that can be executed and reversed.
 */

export type CommandType =
  | 'step:create'
  | 'step:update'
  | 'step:delete'
  | 'step:move'
  | 'edge:create'
  | 'edge:delete'
  | 'group:create'
  | 'group:update'
  | 'group:delete'
  | 'group:move'
  | 'copilot:batch'

export interface Command {
  /** Unique identifier for this command */
  id: string
  /** Type of command for debugging and filtering */
  type: CommandType
  /** Human-readable description */
  description: string
  /** When the command was created */
  timestamp: number
  /** Execute the command */
  execute(): Promise<void>
  /** Undo the command (reverse the action) */
  undo(): Promise<void>
}

/**
 * Batch command that groups multiple commands into a single undo unit.
 * Used for Copilot changes where multiple operations should be undone together.
 */
export class BatchCommand implements Command {
  readonly id: string
  readonly type: CommandType = 'copilot:batch'
  readonly timestamp: number

  constructor(
    private commands: Command[],
    public description: string
  ) {
    this.id = `batch-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`
    this.timestamp = Date.now()
  }

  async execute(): Promise<void> {
    // Execute commands in order
    for (const cmd of this.commands) {
      await cmd.execute()
    }
  }

  async undo(): Promise<void> {
    // Undo commands in reverse order
    for (let i = this.commands.length - 1; i >= 0; i--) {
      await this.commands[i].undo()
    }
  }

  /** Get the commands in this batch (for inspection) */
  getCommands(): readonly Command[] {
    return this.commands
  }
}

const HISTORY_LIMIT = 50

// Global state for command history (shared across components)
const undoStack = ref<Command[]>([])
const redoStack = ref<Command[]>([])
const isExecuting = ref(false)
const lastCommandTime = ref<number>(0)

/**
 * Command History composable for Undo/Redo functionality
 */
export function useCommandHistory() {
  // Computed states
  const canUndo = computed(() => undoStack.value.length > 0 && !isExecuting.value)
  const canRedo = computed(() => redoStack.value.length > 0 && !isExecuting.value)

  // History for display purposes (most recent first)
  const history = computed(() => [...undoStack.value].reverse())

  // Stats for debugging
  const stats = computed(() => ({
    undoCount: undoStack.value.length,
    redoCount: redoStack.value.length,
    isExecuting: isExecuting.value,
    lastCommandTime: lastCommandTime.value,
  }))

  /**
   * Execute a command and add it to the undo stack
   */
  async function execute(command: Command): Promise<void> {
    if (isExecuting.value) {
      console.warn('[CommandHistory] Cannot execute while another command is executing')
      return
    }

    try {
      isExecuting.value = true
      await command.execute()

      // Add to undo stack
      undoStack.value.push(command)

      // Clear redo stack (new action invalidates redo history)
      redoStack.value = []

      // Enforce history limit
      if (undoStack.value.length > HISTORY_LIMIT) {
        undoStack.value.shift()
      }

      lastCommandTime.value = Date.now()
    } catch (error) {
      console.error('[CommandHistory] Failed to execute command:', command.type, error)
      throw error
    } finally {
      isExecuting.value = false
    }
  }

  /**
   * Execute multiple commands as a single undo unit (batch)
   */
  async function executeBatch(commands: Command[], description: string): Promise<void> {
    if (commands.length === 0) return

    if (commands.length === 1) {
      // Single command doesn't need batching
      await execute(commands[0])
      return
    }

    const batchCmd = new BatchCommand(commands, description)
    await execute(batchCmd)
  }

  /**
   * Undo the last command
   */
  async function undo(): Promise<boolean> {
    if (!canUndo.value) {
      console.warn('[CommandHistory] Nothing to undo')
      return false
    }

    const command = undoStack.value.pop()
    if (!command) return false

    try {
      isExecuting.value = true
      await command.undo()

      // Move to redo stack
      redoStack.value.push(command)

      lastCommandTime.value = Date.now()
      return true
    } catch (error) {
      console.error('[CommandHistory] Failed to undo command:', command.type, error)
      // Put the command back on the undo stack since undo failed
      undoStack.value.push(command)
      throw error
    } finally {
      isExecuting.value = false
    }
  }

  /**
   * Redo the last undone command
   */
  async function redo(): Promise<boolean> {
    if (!canRedo.value) {
      console.warn('[CommandHistory] Nothing to redo')
      return false
    }

    const command = redoStack.value.pop()
    if (!command) return false

    try {
      isExecuting.value = true
      await command.execute()

      // Move back to undo stack
      undoStack.value.push(command)

      lastCommandTime.value = Date.now()
      return true
    } catch (error) {
      console.error('[CommandHistory] Failed to redo command:', command.type, error)
      // Put the command back on the redo stack since redo failed
      redoStack.value.push(command)
      throw error
    } finally {
      isExecuting.value = false
    }
  }

  /**
   * Clear all history (e.g., when switching projects)
   */
  function clear(): void {
    undoStack.value = []
    redoStack.value = []
    lastCommandTime.value = 0
  }

  /**
   * Get the description of the command that would be undone
   */
  const undoDescription = computed(() => {
    const cmd = undoStack.value[undoStack.value.length - 1]
    return cmd?.description || null
  })

  /**
   * Get the description of the command that would be redone
   */
  const redoDescription = computed(() => {
    const cmd = redoStack.value[redoStack.value.length - 1]
    return cmd?.description || null
  })

  return {
    // State
    canUndo,
    canRedo,
    isExecuting: readonly(isExecuting),
    history,
    stats,
    undoDescription,
    redoDescription,

    // Actions
    execute,
    executeBatch,
    undo,
    redo,
    clear,
  }
}
