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

export interface Project {
  id: string
  tenant_id: string
  name: string
  description?: string
  status: 'draft' | 'published'
  version: number
  variables?: object // Project-level shared variables
  is_system?: boolean
  system_slug?: string
  draft?: object // Draft state (unsaved changes)
  has_draft: boolean // Indicates if current state is from draft
  created_by?: string
  published_at?: string
  created_at: string
  updated_at: string
  steps?: Step[]
  edges?: Edge[]
  block_groups?: BlockGroup[]
}

export interface Step {
  id: string
  project_id: string
  name: string
  type: StepType
  config: object
  block_group_id?: string      // Reference to containing block group
  group_role?: GroupRole       // Role within block group
  trigger_type?: 'manual' | 'webhook' | 'schedule' | 'slack' | 'email' // Start block trigger type
  trigger_config?: object      // Start block trigger configuration
  credential_bindings?: Record<string, string> // Mapping of credential names to credential IDs
  // Agent Group tool definition (for entry point steps within Agent groups)
  tool_name?: string           // Tool name exposed to the agent
  tool_description?: string    // Description of what the tool does
  tool_input_schema?: object   // JSON Schema for tool parameters
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
  | 'subflow'
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
  project_id: string
  source_step_id?: string | null // Source step (null if from group)
  target_step_id?: string | null // Target step (null if to group)
  source_block_group_id?: string | null // Source group (null if from step)
  target_block_group_id?: string | null // Target group (null if to step)
  source_port?: string // Output port name (e.g., "true", "false", "out")
  condition?: string
  created_at: string
}

export type TriggerType = 'manual' | 'schedule' | 'webhook' | 'test' | 'internal'

export interface Run {
  id: string
  tenant_id: string
  project_id: string
  project_version: number
  status: RunStatus
  run_number: number
  input?: object
  output?: object
  error?: string
  triggered_by: TriggerType
  triggered_by_user?: string
  start_step_id?: string // Which Start block triggered this run
  started_at?: string
  completed_at?: string
  created_at: string
  step_runs?: StepRun[]
  project_definition?: ProjectDefinition
}

export interface ProjectVersion {
  id: string
  project_id: string
  version: number
  definition: ProjectDefinition
  saved_by?: string
  saved_at: string
}

export interface ProjectDefinition {
  name: string
  description: string
  variables?: object // Project-level shared variables
  steps: Step[]
  edges: Edge[]
  block_groups?: BlockGroup[]
}

export type RunStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled'

export interface StepRun {
  id: string
  run_id: string
  step_id: string
  step_name: string
  status: StepRunStatus
  attempt: number
  sequence_number: number
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
export type BlockCategory = 'ai' | 'flow' | 'apps' | 'custom'

// Block Subcategories for better organization
export type BlockSubcategory =
  | 'chat'       // AI: chat/conversation
  | 'rag'        // AI: RAG (retrieval augmented generation)
  | 'routing'    // AI: routing/classification
  | 'trigger'    // Flow: start triggers (manual, schedule, webhook)
  | 'branching'  // Flow: conditional branching
  | 'data'       // Flow: data transformation
  | 'control'    // Flow: control flow
  | 'utility'    // Flow: utilities
  | 'slack'      // Apps: Slack
  | 'discord'    // Apps: Discord
  | 'notion'     // Apps: Notion
  | 'github'     // Apps: GitHub
  | 'google'     // Apps: Google (Sheets, etc)
  | 'linear'     // Apps: Linear
  | 'email'      // Apps: Email providers
  | 'web'        // Apps: Web/HTTP

export interface ErrorCodeDef {
  code: string
  name: string
  description: string
  retryable: boolean
}

// Output port for blocks with multiple outputs (e.g., condition: true/false)
export interface OutputPort {
  name: string        // Unique identifier (e.g., "true", "false", "default")
  label: string       // Display label (e.g., "Yes", "No")
  description?: string
  is_default: boolean
  schema?: object     // Output type schema (JSON Schema)
}

// Internal step for composite blocks
export interface InternalStep {
  type: string       // Block slug to execute
  config: object     // Configuration for the step
  output_key: string // Key to store this step's output
}

// Group block kind (for container blocks)
export type BlockGroupKind = 'parallel' | 'try_catch' | 'foreach' | 'while'

export interface BlockDefinition {
  id: string
  tenant_id?: string
  slug: string
  name: string
  description?: string
  category: BlockCategory
  subcategory?: BlockSubcategory
  icon?: string
  config_schema: object
  output_schema?: object
  output_ports: OutputPort[] // Multiple output ports for branching (e.g., condition, switch)
  error_codes: ErrorCodeDef[]
  // Required credentials for this block (optional)
  required_credentials?: string[]
  // Unified Block Model fields
  code?: string              // JavaScript code executed in sandbox
  ui_config?: object         // UI metadata (icon, color, configSchema)
  is_system?: boolean        // System blocks can only be edited by admins
  version?: number           // Version number, incremented on each update

  // Block Inheritance/Extension fields
  parent_block_id?: string         // Reference to parent block for inheritance
  config_defaults?: object         // Default values for parent's config_schema
  pre_process?: string             // JavaScript code executed before main code
  post_process?: string            // JavaScript code executed after main code
  internal_steps?: InternalStep[]  // Array of steps to execute sequentially inside the block

  // Resolved fields (populated by backend for inherited blocks)
  pre_process_chain?: string[]         // Chain of preProcess code (child -> root)
  post_process_chain?: string[]        // Chain of postProcess code (root -> child)
  resolved_code?: string               // Code from root ancestor
  resolved_config_defaults?: object    // Merged config defaults from inheritance chain

  // Group block fields (Phase B: unified model for groups)
  group_kind?: BlockGroupKind  // Type of group block (parallel, try_catch, foreach, while)
  is_container?: boolean       // Whether this block can contain other steps

  enabled: boolean
  created_at: string
  updated_at: string
}

export interface BlockListResponse {
  blocks: BlockDefinition[]
}

// Block Group Types (Control Flow Constructs)
export type BlockGroupType =
  | 'parallel'     // Parallel execution of different flows
  | 'try_catch'    // Error handling with retry support
  | 'foreach'      // Array iteration (same process for each element)
  | 'while'        // Condition-based loop
  | 'agent'        // AI Agent with tool calling (child steps = tools)

// Error handling is done via output ports (out, error)
export type GroupRole = 'body'

export interface BlockGroup {
  id: string
  project_id: string
  name: string
  type: BlockGroupType
  config: BlockGroupConfig
  parent_group_id?: string      // For nested groups
  pre_process?: string          // JS: external IN -> internal IN
  post_process?: string         // JS: internal OUT -> external OUT
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

// Simplified: catch logic is handled via error output port to external blocks
export interface TryCatchConfig {
  retry_count?: number          // Number of retries before error (default: 0)
  retry_delay_ms?: number       // Delay between retries in ms
}

// NOTE: IfElseConfig removed - use 'condition' system block instead
// NOTE: SwitchCaseConfig removed - use 'switch' system block instead

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

// Agent group config - child steps become callable tools
export interface AgentConfig {
  provider: string              // LLM provider: "openai", "anthropic"
  model: string                 // Model ID (e.g., "claude-sonnet-4-20250514")
  system_prompt: string         // System prompt defining agent behavior
  max_iterations?: number       // ReAct loop max iterations (default: 10)
  temperature?: number          // LLM temperature (default: 0.7)
  tool_choice?: 'auto' | 'none' | 'required'  // Tool calling mode (default: "auto")
  enable_memory?: boolean       // Enable conversation memory
  memory_window?: number        // Number of messages to keep (default: 20)
}

// 5 types: parallel, try_catch, foreach, while, agent
export type BlockGroupConfig =
  | ParallelConfig
  | TryCatchConfig
  | ForeachConfig
  | WhileConfig
  | AgentConfig
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
  pre_process?: string          // JS: external IN -> internal IN
  post_process?: string         // JS: internal OUT -> external OUT
  position: { x: number; y: number }
  size: { width: number; height: number }
}

export interface UpdateBlockGroupRequest {
  name?: string
  config?: BlockGroupConfig
  parent_group_id?: string
  pre_process?: string          // JS: external IN -> internal IN
  post_process?: string         // JS: internal OUT -> external OUT
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

// Tenant Management Types
export type TenantStatus = 'active' | 'suspended' | 'pending' | 'inactive'
export type TenantPlan = 'free' | 'starter' | 'professional' | 'enterprise'

export interface TenantFeatureFlags {
  copilot_enabled: boolean
  advanced_analytics: boolean
  custom_blocks: boolean
  api_access: boolean
  sso_enabled: boolean
  audit_logs: boolean
  max_concurrent_runs: number
}

export interface TenantLimits {
  max_projects: number
  max_runs_per_day: number
  max_users: number
  max_credentials: number
  max_storage_mb: number
  retention_days: number
}

export interface TenantMetadata {
  industry?: string
  company_size?: string
  website?: string
  country?: string
  notes?: string
}

export interface TenantStats {
  project_count: number
  published_projects: number
  run_count: number
  runs_this_month: number
  user_count: number
  credential_count: number
  total_cost_usd: number
  cost_this_month: number
}

export interface Tenant {
  id: string
  name: string
  slug: string
  status: TenantStatus
  plan: TenantPlan
  owner_email?: string
  owner_name?: string
  billing_email?: string
  settings: object
  metadata: TenantMetadata
  feature_flags: TenantFeatureFlags
  limits: TenantLimits
  suspended_at?: string
  suspended_reason?: string
  created_at: string
  updated_at: string
  stats?: TenantStats
}

export interface CreateTenantRequest {
  name: string
  slug: string
  plan?: TenantPlan
  owner_email?: string
  owner_name?: string
  billing_email?: string
  metadata?: TenantMetadata
  feature_flags?: Partial<TenantFeatureFlags>
  limits?: Partial<TenantLimits>
}

export interface UpdateTenantRequest {
  name?: string
  slug?: string
  plan?: TenantPlan
  owner_email?: string
  owner_name?: string
  billing_email?: string
  metadata?: TenantMetadata
  feature_flags?: Partial<TenantFeatureFlags>
  limits?: Partial<TenantLimits>
}

export interface SuspendTenantRequest {
  reason: string
}

export interface TenantOverviewStats {
  total_tenants: number
  status_counts: Record<TenantStatus, number>
  plan_counts: Record<TenantPlan, number>
  total_projects: number
  total_runs: number
  total_runs_this_month: number
  total_cost_usd: number
  cost_this_month: number
}

// Schedule Types
export type ScheduleStatus = 'active' | 'paused' | 'disabled'

export interface Schedule {
  id: string
  tenant_id: string
  project_id: string
  project_version: number
  start_step_id: string // Which Start block to trigger (required)
  name: string
  description?: string
  cron_expression: string
  timezone: string
  input?: object
  status: ScheduleStatus
  next_run_at?: string
  last_run_at?: string
  last_run_id?: string
  run_count: number
  created_by?: string
  created_at: string
  updated_at: string
}

export interface CreateScheduleRequest {
  project_id: string
  start_step_id: string // Which Start block to trigger (required)
  name: string
  description?: string
  cron_expression: string
  timezone?: string
  input?: object
}

export interface UpdateScheduleRequest {
  name?: string
  description?: string
  cron_expression?: string
  timezone?: string
  input?: object
}

// Audit Log Types
export type AuditAction = 'create' | 'update' | 'delete' | 'publish' | 'execute' | 'cancel' | 'approve' | 'reject'

export interface AuditLog {
  id: string
  tenant_id: string
  user_id?: string
  resource_type: string
  resource_id: string
  action: AuditAction
  changes?: object
  metadata?: object
  ip_address?: string
  user_agent?: string
  created_at: string
}

// ============================================================================
// N8N-style Features: Error Workflow, Retry, Templates, etc.
// ============================================================================

// Error Workflow Configuration
export interface ErrorWorkflowConfig {
  trigger_on: ('failed' | 'cancelled' | 'timeout')[]
  input_mapping?: Record<string, string>
  enabled: boolean
}

// Retry Configuration for Steps
export interface RetryConfig {
  max_retries: number           // Maximum retries (default: 0)
  delay_ms: number              // Initial delay (default: 1000)
  exponential_backoff: boolean  // Use exponential backoff
  max_delay_ms: number          // Max delay for backoff (default: 30000)
  retry_on_errors?: string[]    // Error codes to retry on (empty = all)
}

// Streaming Output Chunk
export interface StreamingChunk {
  chunk: string
  timestamp: string
  type: 'text' | 'json' | 'error'
}

// Extended Step with retry config
export interface StepWithRetry extends Step {
  retry_config?: RetryConfig
}

// Extended StepRun with debug features
export interface StepRunWithDebug extends StepRun {
  pinned_input?: object
  streaming_output?: StreamingChunk[]
}

// Extended Run with error workflow tracking
export interface RunWithErrorWorkflow extends Run {
  parent_run_id?: string
  error_trigger_source?: {
    original_run_id: string
    original_project: string
    error_step_id?: string
    error_step_name?: string
    error_message: string
    triggered_at: string
  }
}

// Extended Project with error workflow
export interface ProjectWithErrorWorkflow extends Project {
  error_workflow_id?: string
  error_workflow_config?: ErrorWorkflowConfig
}

// ============================================================================
// Project Templates
// ============================================================================

export type TemplateVisibility = 'private' | 'tenant' | 'public'
export type TemplateReviewStatus = 'pending' | 'approved' | 'rejected'

export interface ProjectTemplate {
  id: string
  tenant_id?: string
  name: string
  description?: string
  category?: string
  tags?: string[]
  definition: ProjectDefinition
  variables?: object
  thumbnail_url?: string
  author_name?: string
  download_count: number
  is_featured: boolean
  visibility: TemplateVisibility
  review_status?: TemplateReviewStatus
  price_usd: number
  rating?: number
  review_count: number
  created_at: string
  updated_at: string
}

export interface CreateTemplateRequest {
  name: string
  description?: string
  category?: string
  tags?: string[]
  definition: ProjectDefinition
  variables?: object
}

export interface InstantiateTemplateRequest {
  name: string
  variables?: object
}

export interface TemplateReview {
  id: string
  template_id: string
  user_id: string
  rating: number
  comment?: string
  created_at: string
}

export interface CreateTemplateReviewRequest {
  rating: number
  comment?: string
}

export interface TemplateCategory {
  slug: string
  name: string
  description?: string
  icon?: string
}

// ============================================================================
// Agent Chat Sessions
// ============================================================================

export type AgentChatSessionStatus = 'active' | 'closed'

export interface AgentChatSession {
  id: string
  tenant_id: string
  project_id: string
  start_step_id: string
  user_id: string
  status: AgentChatSessionStatus
  metadata?: object
  created_at: string
  updated_at: string
  runs?: Run[]
}

export interface CreateAgentChatSessionRequest {
  project_id: string
  start_step_id: string
}

export interface AgentChatMessage {
  id: string
  session_id: string
  run_id?: string
  role: 'user' | 'assistant' | 'system'
  content: string
  metadata?: object
  created_at: string
}

export interface SendAgentChatMessageRequest {
  content: string
  metadata?: object
}

// ============================================================================
// Git Sync
// ============================================================================

export type GitSyncDirection = 'push' | 'pull' | 'bidirectional'

export interface ProjectGitSync {
  id: string
  tenant_id: string
  project_id: string
  repository_url: string
  branch: string
  file_path: string
  sync_direction: GitSyncDirection
  auto_sync: boolean
  last_sync_at?: string
  last_commit_sha?: string
  credentials_id?: string
  created_at: string
  updated_at: string
}

export interface CreateGitSyncRequest {
  repository_url: string
  branch?: string
  file_path?: string
  sync_direction?: GitSyncDirection
  credentials_id?: string
}

export interface UpdateGitSyncRequest {
  branch?: string
  file_path?: string
  sync_direction?: GitSyncDirection
  auto_sync?: boolean
  credentials_id?: string
}

// ============================================================================
// Custom Block Packages (SDK)
// ============================================================================

export type BlockPackageStatus = 'draft' | 'published' | 'deprecated'

export interface PackageBlockDefinition {
  slug: string
  name: string
  description?: string
  category: string
  icon?: string
  config_schema: object
  output_schema?: object
  code: string
  ui_config?: object
}

export interface PackageDependency {
  name: string
  version: string
}

export interface CustomBlockPackage {
  id: string
  tenant_id: string
  name: string
  version: string
  description?: string
  bundle_url?: string
  blocks: PackageBlockDefinition[]
  dependencies: PackageDependency[]
  status: BlockPackageStatus
  created_by?: string
  created_at: string
  updated_at: string
}

export interface CreateBlockPackageRequest {
  name: string
  version: string
  description?: string
  blocks: PackageBlockDefinition[]
  dependencies?: PackageDependency[]
}

export interface UpdateBlockPackageRequest {
  description?: string
  blocks?: PackageBlockDefinition[]
  dependencies?: PackageDependency[]
}

// ============================================================================
// Expression Debugger
// ============================================================================

export interface ExpressionDebugRequest {
  expression: string
  context: object
}

export interface ExpressionDebugResponse {
  result: unknown
  resolved_variables: Array<{
    path: string
    value: unknown
  }>
  errors: string[]
}

// ============================================================================
// Input Pinning
// ============================================================================

export interface PinInputRequest {
  input: object
}

export interface SingleStepExecuteRequest {
  input?: object
  use_pinned_input?: boolean
}
