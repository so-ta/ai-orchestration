// API Types

export interface ApiResponse<T> {
  data: T
}

export interface PaginatedResponse<T> {
  data: T[]
  meta: {
    page: number
    limit: number
    total: number
  }
}

export interface Workflow {
  id: string
  tenant_id: string
  name: string
  description?: string
  status: 'draft' | 'published'
  version: number
  input_schema?: object
  output_schema?: object
  draft?: object // Draft state (unsaved changes)
  has_draft: boolean // Indicates if current state is from draft
  created_by?: string
  published_at?: string
  created_at: string
  updated_at: string
  steps?: Step[]
  edges?: Edge[]
}

export interface Step {
  id: string
  workflow_id: string
  name: string
  type: StepType
  config: object
  block_group_id?: string      // Reference to containing block group
  group_role?: GroupRole       // Role within block group
  position_x: number
  position_y: number
  created_at: string
  updated_at: string
}

export type StepType =
  | 'start'
  | 'llm'
  | 'tool'
  | 'condition'
  | 'switch'
  | 'map'
  | 'join'
  | 'subflow'
  | 'loop'
  | 'wait'
  | 'function'
  | 'router'
  | 'human_in_loop'
  | 'filter'
  | 'split'
  | 'aggregate'
  | 'error'
  | 'note'
  | 'log'

export interface Edge {
  id: string
  workflow_id: string
  source_step_id: string
  target_step_id: string
  source_port?: string // Output port name (e.g., "true", "false")
  target_port?: string // Input port name (e.g., "input", "items")
  condition?: string
  created_at: string
}

export interface Run {
  id: string
  tenant_id: string
  workflow_id: string
  workflow_version: number
  status: RunStatus
  mode: 'test' | 'production'
  input?: object
  output?: object
  error?: string
  triggered_by: 'manual' | 'schedule' | 'webhook'
  triggered_by_user?: string
  started_at?: string
  completed_at?: string
  created_at: string
  step_runs?: StepRun[]
  workflow_definition?: WorkflowDefinition
}

export interface WorkflowVersion {
  id: string
  workflow_id: string
  version: number
  definition: WorkflowDefinition
  saved_by?: string
  saved_at: string
}

export interface WorkflowDefinition {
  name: string
  description: string
  input_schema?: object
  output_schema?: object
  steps: Step[]
  edges: Edge[]
}

export type RunStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled'

export interface StepRun {
  id: string
  run_id: string
  step_id: string
  step_name: string
  status: StepRunStatus
  attempt: number
  input?: object
  output?: object
  error?: string
  started_at?: string
  completed_at?: string
  duration_ms?: number
  created_at: string
}

export type StepRunStatus = 'pending' | 'running' | 'completed' | 'failed' | 'skipped'

// Block Registry Types
export type BlockCategory = 'ai' | 'logic' | 'integration' | 'data' | 'control' | 'utility'
export type ExecutorType = 'builtin' | 'http' | 'function'

export interface ErrorCodeDef {
  code: string
  name: string
  description: string
  retryable: boolean
}

// Input port for blocks with multiple inputs (e.g., join, aggregate)
export interface InputPort {
  name: string        // Unique identifier (e.g., "input", "items", "input_1")
  label: string       // Display label (e.g., "Input", "Items to process")
  description?: string
  required: boolean
  schema?: object     // Input type schema (JSON Schema)
}

// Output port for blocks with multiple outputs (e.g., condition: true/false)
export interface OutputPort {
  name: string        // Unique identifier (e.g., "true", "false", "default")
  label: string       // Display label (e.g., "Yes", "No")
  description?: string
  is_default: boolean
  schema?: object     // Output type schema (JSON Schema)
}

export interface BlockDefinition {
  id: string
  tenant_id?: string
  slug: string
  name: string
  description?: string
  category: BlockCategory
  icon?: string
  config_schema: object
  input_schema?: object
  output_schema?: object
  input_ports: InputPort[]   // Multiple input ports for merging (e.g., join, aggregate)
  output_ports: OutputPort[] // Multiple output ports for branching (e.g., condition, switch)
  executor_type: ExecutorType
  executor_config?: object
  error_codes: ErrorCodeDef[]
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface BlockListResponse {
  blocks: BlockDefinition[]
}

// Block Group Types (Control Flow Constructs)
export type BlockGroupType =
  | 'parallel'     // Parallel execution group
  | 'try_catch'    // Try-catch-finally error handling
  | 'if_else'      // Conditional branching
  | 'switch_case'  // Multi-branch routing
  | 'foreach'      // Array iteration loop
  | 'while'        // Condition-based loop

export type GroupRole =
  | 'body'         // Main execution body (parallel, foreach, while)
  | 'try'          // Try block (try_catch)
  | 'catch'        // Catch block (try_catch)
  | 'finally'      // Finally block (try_catch)
  | 'then'         // Then branch (if_else)
  | 'else'         // Else branch (if_else)
  | 'default'      // Default case (switch_case)
  | string         // Dynamic case roles like 'case_0', 'case_1', etc.

export interface BlockGroup {
  id: string
  workflow_id: string
  name: string
  type: BlockGroupType
  config: BlockGroupConfig
  parent_group_id?: string      // For nested groups
  position_x: number
  position_y: number
  width: number
  height: number
  created_at: string
  updated_at: string
}

// Type-specific configurations
export interface ParallelConfig {
  max_concurrent?: number       // Max concurrent executions (0 = unlimited)
  fail_fast?: boolean           // Stop all on first failure
}

export interface TryCatchConfig {
  error_types?: string[]        // Error types to catch ("*" = all)
  retry_count?: number          // Number of retries before catch
  retry_delay_ms?: number       // Delay between retries in ms
}

export interface IfElseConfig {
  condition: string             // Condition expression (e.g., "$.status == 'active'")
}

export interface SwitchCaseConfig {
  expression: string            // Expression to evaluate
  cases: string[]               // Case values
  has_default?: boolean         // Whether default case exists
}

export interface ForeachConfig {
  input_path: string            // Path to array (e.g., "$.items")
  parallel?: boolean            // Execute iterations in parallel
  max_workers?: number          // Max parallel workers
}

export interface WhileConfig {
  condition: string             // Condition expression
  max_iterations?: number       // Safety limit (default: 100)
  do_while?: boolean            // Execute at least once (do-while)
}

export type BlockGroupConfig =
  | ParallelConfig
  | TryCatchConfig
  | IfElseConfig
  | SwitchCaseConfig
  | ForeachConfig
  | WhileConfig
  | object

export interface BlockGroupRun {
  id: string
  run_id: string
  block_group_id: string
  status: StepRunStatus
  iteration?: number            // For loop groups
  input?: object
  output?: object
  error?: string
  started_at?: string
  completed_at?: string
  created_at: string
}

// Request/Response types for Block Group API
export interface CreateBlockGroupRequest {
  name: string
  type: BlockGroupType
  config?: BlockGroupConfig
  parent_group_id?: string
  position: { x: number; y: number }
  size: { width: number; height: number }
}

export interface UpdateBlockGroupRequest {
  name?: string
  config?: BlockGroupConfig
  parent_group_id?: string
  position?: { x: number; y: number }
  size?: { width: number; height: number }
}

export interface AddStepToGroupRequest {
  step_id: string
  group_role: GroupRole
}

// Credential Types
export type CredentialType = 'api_key' | 'oauth2' | 'basic_auth' | 'custom'
export type CredentialStatus = 'active' | 'expired' | 'revoked'

export interface CredentialMetadata {
  provider?: string      // e.g., "openai", "anthropic", "github"
  scopes?: string[]      // OAuth2 scopes
  environment?: string   // e.g., "production", "staging"
  tags?: string[]        // Custom tags for filtering
}

export interface Credential {
  id: string
  tenant_id: string
  name: string
  description?: string
  credential_type: CredentialType
  metadata: CredentialMetadata
  expires_at?: string
  status: CredentialStatus
  created_at: string
  updated_at: string
}

export interface CreateCredentialRequest {
  name: string
  description?: string
  credential_type: CredentialType
  data: CredentialData
  metadata?: CredentialMetadata
  expires_at?: string
}

export interface UpdateCredentialRequest {
  name?: string
  description?: string
  data?: CredentialData
  metadata?: CredentialMetadata
  expires_at?: string
}

export interface CredentialData {
  // API Key
  api_key?: string
  header_name?: string    // e.g., "Authorization", "X-API-Key"
  header_prefix?: string  // e.g., "Bearer ", "Token "

  // Basic Auth
  username?: string
  password?: string

  // OAuth2
  access_token?: string
  refresh_token?: string
  token_type?: string
  expires_at?: string
  scopes?: string[]

  // Custom fields
  custom?: Record<string, unknown>
}

// System Credential Types (Operator-managed)
export interface SystemCredential {
  id: string
  name: string
  description?: string
  credential_type: CredentialType
  metadata: CredentialMetadata
  expires_at?: string
  status: CredentialStatus
  created_at: string
  updated_at: string
}

export interface CreateSystemCredentialRequest {
  name: string
  description?: string
  credential_type: CredentialType
  data: CredentialData
  metadata?: CredentialMetadata
  expires_at?: string
}

export interface UpdateSystemCredentialRequest {
  name?: string
  description?: string
  data?: CredentialData
  metadata?: CredentialMetadata
  expires_at?: string
}

// Block Template Types
export type TemplateExecutorType = 'builtin' | 'javascript'

export interface BlockTemplate {
  id: string
  slug: string
  name: string
  description?: string
  config_schema: object
  executor_type: TemplateExecutorType
  executor_code?: string
  is_builtin: boolean
  created_at: string
  updated_at: string
}

export interface CreateBlockTemplateRequest {
  slug: string
  name: string
  description?: string
  config_schema?: object
  executor_type: TemplateExecutorType
  executor_code?: string
}

export interface UpdateBlockTemplateRequest {
  slug?: string
  name?: string
  description?: string
  config_schema?: object
  executor_type?: TemplateExecutorType
  executor_code?: string
}
