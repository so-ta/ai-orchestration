package domain

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// BlockCategory represents the category of a block
type BlockCategory string

const (
	BlockCategoryAI          BlockCategory = "ai"
	BlockCategoryLogic       BlockCategory = "logic"
	BlockCategoryIntegration BlockCategory = "integration"
	BlockCategoryData        BlockCategory = "data"
	BlockCategoryControl     BlockCategory = "control"
	BlockCategoryUtility     BlockCategory = "utility"
	BlockCategoryGroup       BlockCategory = "group" // Group blocks (parallel, try_catch, foreach, while)
)

// BlockGroupKind represents the kind of group block
type BlockGroupKind string

const (
	BlockGroupKindNone     BlockGroupKind = ""          // Not a group block
	BlockGroupKindParallel BlockGroupKind = "parallel"  // Parallel execution
	BlockGroupKindTryCatch BlockGroupKind = "try_catch" // Error handling with retry
	BlockGroupKindForeach  BlockGroupKind = "foreach"   // Array iteration
	BlockGroupKindWhile    BlockGroupKind = "while"     // Condition loop
)

// ValidBlockCategories returns all valid block categories
func ValidBlockCategories() []BlockCategory {
	return []BlockCategory{
		BlockCategoryAI,
		BlockCategoryLogic,
		BlockCategoryIntegration,
		BlockCategoryData,
		BlockCategoryControl,
		BlockCategoryUtility,
		BlockCategoryGroup,
	}
}

// ValidBlockGroupKinds returns all valid group block kinds
func ValidBlockGroupKinds() []BlockGroupKind {
	return []BlockGroupKind{
		BlockGroupKindParallel,
		BlockGroupKindTryCatch,
		BlockGroupKindForeach,
		BlockGroupKindWhile,
	}
}

// IsValid checks if the group kind is valid
func (k BlockGroupKind) IsValid() bool {
	if k == BlockGroupKindNone {
		return true // Not a group block
	}
	for _, valid := range ValidBlockGroupKinds() {
		if k == valid {
			return true
		}
	}
	return false
}

// IsValid checks if the category is valid
func (c BlockCategory) IsValid() bool {
	for _, valid := range ValidBlockCategories() {
		if c == valid {
			return true
		}
	}
	return false
}

// ErrorCodeDef defines an error code for a block
type ErrorCodeDef struct {
	Code        string `json:"code"`        // e.g., "LLM_001"
	Name        string `json:"name"`        // e.g., "RATE_LIMIT"
	Description string `json:"description"` // Human-readable description
	Retryable   bool   `json:"retryable"`   // Can this error be retried?
}

// InputPort defines an input connection point for a block
type InputPort struct {
	Name        string          `json:"name"`                  // Unique identifier (e.g., "input", "items", "condition")
	Label       string          `json:"label"`                 // Display label (e.g., "Input", "Items to process")
	Description string          `json:"description,omitempty"` // Human-readable description
	Required    bool            `json:"required"`              // Is this input required?
	Schema      json.RawMessage `json:"schema,omitempty"`      // Input type schema (JSON Schema)
}

// OutputPort defines an output connection point for a block
type OutputPort struct {
	Name        string          `json:"name"`                  // Unique identifier (e.g., "true", "false", "default")
	Label       string          `json:"label"`                 // Display label (e.g., "Yes", "No")
	Description string          `json:"description,omitempty"` // Human-readable description
	IsDefault   bool            `json:"is_default"`            // Is this the default/primary output
	Schema      json.RawMessage `json:"schema,omitempty"`      // Output type schema (JSON Schema)
}

// TypeSchema represents a simplified type for GUI hints
type TypeSchema struct {
	Type       string                 `json:"type"`                 // "string", "number", "boolean", "object", "array", "any"
	Properties map[string]*TypeSchema `json:"properties,omitempty"` // For object type
	Items      *TypeSchema            `json:"items,omitempty"`      // For array type
	Required   []string               `json:"required,omitempty"`   // Required properties
	Enum       []interface{}          `json:"enum,omitempty"`       // Allowed values
}

// UIConfig represents UI metadata for block visualization
type UIConfig struct {
	Icon         string `json:"icon,omitempty"`          // Icon name (e.g., "brain", "play")
	Color        string `json:"color,omitempty"`         // Hex color (e.g., "#8B5CF6")
	ConfigSchema any    `json:"configSchema,omitempty"`  // Schema for block config in workflow editor
}

// InternalStep represents a step inside a composite block
type InternalStep struct {
	Type      string          `json:"type"`       // Block slug to execute
	Config    json.RawMessage `json:"config"`     // Configuration for the step
	OutputKey string          `json:"output_key"` // Key to store this step's output
}

// BlockDefinition represents a block type definition
type BlockDefinition struct {
	ID          uuid.UUID       `json:"id"`
	TenantID    *uuid.UUID      `json:"tenant_id,omitempty"` // NULL = system block
	Slug        string          `json:"slug"`                // Unique identifier
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Category    BlockCategory   `json:"category"`
	Icon        string          `json:"icon,omitempty"`

	// Schemas (JSON Schema format)
	ConfigSchema json.RawMessage `json:"config_schema"`
	InputSchema  json.RawMessage `json:"input_schema,omitempty"`
	OutputSchema json.RawMessage `json:"output_schema,omitempty"`

	// Input ports (for blocks with multiple inputs like join, aggregate)
	InputPorts []InputPort `json:"input_ports"`

	// Output ports (for blocks with multiple outputs like condition, switch)
	OutputPorts []OutputPort `json:"output_ports"`

	// === Unified Block Model fields ===
	// Code: JavaScript code executed in sandbox (all blocks are code-based)
	Code string `json:"code,omitempty"`
	// UIConfig: UI metadata for workflow editor (icon, color, configSchema)
	UIConfig json.RawMessage `json:"ui_config,omitempty"`
	// IsSystem: System blocks can only be edited by admins
	IsSystem bool `json:"is_system"`
	// Version: Version number, incremented on each update
	Version int `json:"version"`

	// Required credentials declaration
	// Format: [{"name": "api_key", "type": "api_key", "scope": "system|tenant", "description": "...", "required": true}]
	RequiredCredentials json.RawMessage `json:"required_credentials,omitempty"`

	// Visibility (only applies to tenant blocks; system blocks are always visible)
	IsPublic bool `json:"is_public"`

	// Error handling
	ErrorCodes []ErrorCodeDef `json:"error_codes"`

	// === Block Inheritance/Extension fields ===
	// ParentBlockID: Reference to parent block for inheritance (only blocks with code can be inherited)
	ParentBlockID *uuid.UUID `json:"parent_block_id,omitempty"`
	// ConfigDefaults: Default values for parent's config_schema
	ConfigDefaults json.RawMessage `json:"config_defaults,omitempty"`
	// PreProcess: JavaScript code executed before main code (input transformation)
	PreProcess string `json:"pre_process,omitempty"`
	// PostProcess: JavaScript code executed after main code (output transformation)
	PostProcess string `json:"post_process,omitempty"`
	// InternalSteps: Array of steps to execute sequentially inside the block
	InternalSteps []InternalStep `json:"internal_steps,omitempty"`

	// === Group Block fields (Phase B: unified block model for groups) ===
	// GroupKind: Type of group block (parallel, try_catch, foreach, while). Empty for non-group blocks.
	GroupKind BlockGroupKind `json:"group_kind,omitempty"`
	// IsContainer: Whether this block can contain other steps (group blocks are containers)
	IsContainer bool `json:"is_container,omitempty"`

	// Resolved fields (populated by resolveInheritance, not stored in DB)
	// PreProcessChain: Chain of preProcess code from child to root (child -> ... -> root)
	PreProcessChain []string `json:"pre_process_chain,omitempty"`
	// PostProcessChain: Chain of postProcess code from root to child (root -> ... -> child)
	PostProcessChain []string `json:"post_process_chain,omitempty"`
	// ResolvedCode: Code from the root ancestor (for inherited blocks)
	ResolvedCode string `json:"resolved_code,omitempty"`
	// ResolvedConfigDefaults: Merged config defaults from entire inheritance chain
	ResolvedConfigDefaults json.RawMessage `json:"resolved_config_defaults,omitempty"`

	// Metadata
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GetRequiredCredentials returns parsed required credentials
func (b *BlockDefinition) GetRequiredCredentials() ([]RequiredCredential, error) {
	return ParseRequiredCredentials(b.RequiredCredentials)
}

// NewBlockDefinition creates a new block definition
func NewBlockDefinition(tenantID *uuid.UUID, slug, name string, category BlockCategory) *BlockDefinition {
	now := time.Now().UTC()
	return &BlockDefinition{
		ID:           uuid.New(),
		TenantID:     tenantID,
		Slug:         slug,
		Name:         name,
		Category:     category,
		ConfigSchema: json.RawMessage("{}"),
		InputPorts:   []InputPort{{Name: "input", Label: "Input", Required: true}}, // Default single input
		OutputPorts:  []OutputPort{{Name: "output", Label: "Output", IsDefault: true}}, // Default single output
		ErrorCodes:   []ErrorCodeDef{},
		Enabled:      true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// IsSystemBlock returns true if this is a system-defined block (not tenant-specific)
func (b *BlockDefinition) IsSystemBlock() bool {
	return b.TenantID == nil
}

// CanBeInherited checks if this block can be inherited
// Only blocks with code can be inherited (system control blocks like if, foreach cannot be inherited)
func (b *BlockDefinition) CanBeInherited() bool {
	return b.Code != ""
}

// HasInheritance returns true if this block inherits from another block
func (b *BlockDefinition) HasInheritance() bool {
	return b.ParentBlockID != nil
}

// HasInternalSteps returns true if this block has internal steps
func (b *BlockDefinition) HasInternalSteps() bool {
	return len(b.InternalSteps) > 0
}

// IsGroupBlock returns true if this block is a group block (container)
func (b *BlockDefinition) IsGroupBlock() bool {
	return b.GroupKind != BlockGroupKindNone && b.GroupKind != ""
}

// GetEffectiveCode returns the code to execute (resolved code for inherited blocks, own code otherwise)
func (b *BlockDefinition) GetEffectiveCode() string {
	if b.ResolvedCode != "" {
		return b.ResolvedCode
	}
	return b.Code
}

// GetEffectiveConfigDefaults returns the config defaults to use (resolved for inherited blocks)
func (b *BlockDefinition) GetEffectiveConfigDefaults() json.RawMessage {
	if b.ResolvedConfigDefaults != nil && len(b.ResolvedConfigDefaults) > 0 {
		return b.ResolvedConfigDefaults
	}
	return b.ConfigDefaults
}

// BlockError represents an error from block execution with error code
type BlockError struct {
	Code       string          `json:"code"`                  // Error code (e.g., "LLM_001")
	Message    string          `json:"message"`               // Human-readable message
	Details    json.RawMessage `json:"details,omitempty"`     // Additional error details
	Retryable  bool            `json:"retryable"`             // Can this be retried?
	RetryAfter *time.Duration  `json:"retry_after,omitempty"` // Suggested retry delay
}

// Error implements the error interface
func (e *BlockError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// NewBlockError creates a new block error
func NewBlockError(code, message string, retryable bool) *BlockError {
	return &BlockError{
		Code:      code,
		Message:   message,
		Retryable: retryable,
	}
}

// WithDetails adds details to the error
func (e *BlockError) WithDetails(details interface{}) *BlockError {
	if data, err := json.Marshal(details); err == nil {
		e.Details = data
	}
	return e
}

// WithRetryAfter sets the retry delay
func (e *BlockError) WithRetryAfter(d time.Duration) *BlockError {
	e.RetryAfter = &d
	return e
}

// Common error codes
const (
	// System errors (000-099)
	ErrCodeSystemInternal = "SYS_001"
	ErrCodeSystemTimeout  = "SYS_002"

	// Config errors (100-199)
	ErrCodeConfigInvalid  = "CFG_001"
	ErrCodeConfigMissing  = "CFG_002"

	// Input errors (200-299)
	ErrCodeInputInvalid   = "INP_001"
	ErrCodeInputMissing   = "INP_002"

	// Execution errors (300-399)
	ErrCodeExecFailed     = "EXEC_001"
	ErrCodeExecCancelled  = "EXEC_002"

	// Auth errors (500-599)
	ErrCodeAuthFailed     = "AUTH_001"
	ErrCodeAuthExpired    = "AUTH_002"

	// Rate limit errors (600-699)
	ErrCodeRateLimit      = "RATE_001"
)

// BlockExecutionRequest represents a request to execute a block
type BlockExecutionRequest struct {
	BlockSlug     string          `json:"block_slug"`
	Input         json.RawMessage `json:"input"`
	Config        json.RawMessage `json:"config"`
	CorrelationID string          `json:"correlation_id"`
	TenantID      uuid.UUID       `json:"tenant_id"`
}

// BlockExecutionResponse represents the response from block execution
type BlockExecutionResponse struct {
	Output     json.RawMessage   `json:"output"`
	DurationMs int               `json:"duration_ms"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// ============================================================================
// BlockVersion - Version history for block definitions
// ============================================================================

// BlockVersion represents a snapshot of a block definition at a specific version
type BlockVersion struct {
	ID           uuid.UUID       `json:"id"`
	BlockID      uuid.UUID       `json:"block_id"`
	Version      int             `json:"version"`

	// Snapshot of block at this version
	Code         string          `json:"code"`
	ConfigSchema json.RawMessage `json:"config_schema"`
	InputSchema  json.RawMessage `json:"input_schema,omitempty"`
	OutputSchema json.RawMessage `json:"output_schema,omitempty"`
	UIConfig     json.RawMessage `json:"ui_config"`

	// Change tracking
	ChangeSummary string     `json:"change_summary,omitempty"`
	ChangedBy     *uuid.UUID `json:"changed_by,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}

// NewBlockVersion creates a new block version from a block definition
func NewBlockVersion(block *BlockDefinition, changeSummary string, changedBy *uuid.UUID) *BlockVersion {
	return &BlockVersion{
		ID:            uuid.New(),
		BlockID:       block.ID,
		Version:       block.Version,
		Code:          block.Code,
		ConfigSchema:  block.ConfigSchema,
		InputSchema:   block.InputSchema,
		OutputSchema:  block.OutputSchema,
		UIConfig:      block.UIConfig,
		ChangeSummary: changeSummary,
		ChangedBy:     changedBy,
		CreatedAt:     time.Now().UTC(),
	}
}

// ErrBlockVersionNotFound is returned when a block version is not found
var ErrBlockVersionNotFound = fmt.Errorf("block version not found")
