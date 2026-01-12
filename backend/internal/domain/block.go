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
	}
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

// ExecutorType represents how a block is executed
type ExecutorType string

const (
	ExecutorTypeBuiltin  ExecutorType = "builtin"  // Go code implementation
	ExecutorTypeHTTP     ExecutorType = "http"     // HTTP request
	ExecutorTypeFunction ExecutorType = "function" // JavaScript function
)

// ValidExecutorTypes returns all valid executor types
func ValidExecutorTypes() []ExecutorType {
	return []ExecutorType{
		ExecutorTypeBuiltin,
		ExecutorTypeHTTP,
		ExecutorTypeFunction,
	}
}

// IsValid checks if the executor type is valid
func (e ExecutorType) IsValid() bool {
	for _, valid := range ValidExecutorTypes() {
		if e == valid {
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

	// Executor (legacy fields, for backward compatibility)
	ExecutorType   ExecutorType    `json:"executor_type"`
	ExecutorConfig json.RawMessage `json:"executor_config,omitempty"`

	// Template-based execution (new)
	TemplateID     *uuid.UUID      `json:"template_id,omitempty"`     // Reference to block_templates
	TemplateConfig json.RawMessage `json:"template_config,omitempty"` // Template configuration

	// Custom code execution (for code-based blocks)
	// Hidden for system blocks when accessed by tenant users
	CustomCode string `json:"custom_code,omitempty"`

	// Required credentials declaration
	// Format: [{"name": "api_key", "type": "api_key", "scope": "system|tenant", "description": "...", "required": true}]
	RequiredCredentials json.RawMessage `json:"required_credentials,omitempty"`

	// Visibility (only applies to tenant blocks; system blocks are always visible)
	IsPublic bool `json:"is_public"`

	// Error handling
	ErrorCodes []ErrorCodeDef `json:"error_codes"`

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
		ExecutorType: ExecutorTypeBuiltin,
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

// HTTPExecutorConfig represents configuration for HTTP executor
type HTTPExecutorConfig struct {
	Method       string            `json:"method"`                  // GET, POST, PUT, DELETE
	URL          string            `json:"url"`                     // URL template
	Headers      map[string]string `json:"headers,omitempty"`       // Request headers
	BodyTemplate string            `json:"body_template,omitempty"` // Request body template
	TimeoutMs    int               `json:"timeout_ms,omitempty"`    // Request timeout
}

// FunctionExecutorConfig represents configuration for function executor
type FunctionExecutorConfig struct {
	Code      string `json:"code"`                 // JavaScript code
	Language  string `json:"language,omitempty"`   // javascript (default)
	TimeoutMs int    `json:"timeout_ms,omitempty"` // Execution timeout
}

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
// BlockTemplate - Reusable patterns for block definitions
// ============================================================================

// TemplateExecutorType represents how a template is executed
type TemplateExecutorType string

const (
	TemplateExecutorBuiltin    TemplateExecutorType = "builtin"    // Go code implementation
	TemplateExecutorJavaScript TemplateExecutorType = "javascript" // JavaScript code
)

// BlockTemplate represents a reusable block pattern
type BlockTemplate struct {
	ID           uuid.UUID            `json:"id"`
	Slug         string               `json:"slug"` // Unique identifier (e.g., "http_api", "graphql")
	Name         string               `json:"name"`
	Description  string               `json:"description,omitempty"`
	ConfigSchema json.RawMessage      `json:"config_schema"`     // What users configure when using this template
	ExecutorType TemplateExecutorType `json:"executor_type"`     // "builtin" or "javascript"
	ExecutorCode string               `json:"executor_code,omitempty"` // For javascript templates
	IsBuiltin    bool                 `json:"is_builtin"`        // Cannot be deleted if true
	CreatedAt    time.Time            `json:"created_at"`
	UpdatedAt    time.Time            `json:"updated_at"`
}

// NewBlockTemplate creates a new block template
func NewBlockTemplate(slug, name string) *BlockTemplate {
	now := time.Now().UTC()
	return &BlockTemplate{
		ID:           uuid.New(),
		Slug:         slug,
		Name:         name,
		ConfigSchema: json.RawMessage("{}"),
		ExecutorType: TemplateExecutorBuiltin,
		IsBuiltin:    false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// BuiltinTemplates returns the slugs of all built-in templates
func BuiltinTemplates() []string {
	return []string{
		"http_api",
		"graphql",
		"transform",
		"llm_call",
	}
}

// IsBuiltinTemplate checks if a slug is a built-in template
func IsBuiltinTemplate(slug string) bool {
	for _, builtin := range BuiltinTemplates() {
		if slug == builtin {
			return true
		}
	}
	return false
}
