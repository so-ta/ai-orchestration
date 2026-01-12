// Execution-related types

export interface ExecutionLog {
  id: string
  timestamp: Date
  level: 'info' | 'warn' | 'error' | 'success'
  message: string
  stepId?: string
  stepName?: string
  data?: unknown
}

export interface ABTestConfig {
  id: string
  name: string
  provider: string
  model: string
  enabled: boolean
}

export interface PromptVariant {
  id: string
  name: string
  prompt: string
  enabled: boolean
}
