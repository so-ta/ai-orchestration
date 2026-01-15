package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// StepType represents the type of a step
type StepType string

const (
	StepTypeStart       StepType = "start"
	StepTypeLLM         StepType = "llm"
	StepTypeTool        StepType = "tool"
	StepTypeCondition   StepType = "condition"
	StepTypeSwitch      StepType = "switch"      // Multi-branch routing (n8n: Switch)
	StepTypeMap         StepType = "map"
	StepTypeJoin        StepType = "join"
	StepTypeSubflow     StepType = "subflow"
	StepTypeWait        StepType = "wait"
	StepTypeFunction    StepType = "function"
	StepTypeRouter      StepType = "router"
	StepTypeHumanInLoop StepType = "human_in_loop"
	StepTypeFilter      StepType = "filter"      // Filter items (n8n: Filter)
	StepTypeSplit       StepType = "split"       // Split into batches (n8n: Split In Batches)
	StepTypeAggregate   StepType = "aggregate"   // Aggregate data (n8n: Aggregate)
	StepTypeError       StepType = "error"       // Stop and error (n8n: Stop And Error)
	StepTypeNote        StepType = "note"        // Documentation/comment node (n8n: NOOP)
	StepTypeLog         StepType = "log"         // Log output for debugging
	// Note: "loop" step type has been removed. Use BlockGroupTypeWhile or BlockGroupTypeForeach instead.
)

// ValidStepTypes returns all valid step types
func ValidStepTypes() []StepType {
	return []StepType{
		StepTypeStart,
		StepTypeLLM,
		StepTypeTool,
		StepTypeCondition,
		StepTypeSwitch,
		StepTypeMap,
		StepTypeJoin,
		StepTypeSubflow,
		StepTypeWait,
		StepTypeFunction,
		StepTypeRouter,
		StepTypeHumanInLoop,
		StepTypeFilter,
		StepTypeSplit,
		StepTypeAggregate,
		StepTypeError,
		StepTypeNote,
		StepTypeLog,
	}
}

// IsValid checks if the step type is valid
func (t StepType) IsValid() bool {
	for _, valid := range ValidStepTypes() {
		if t == valid {
			return true
		}
	}
	return false
}

// Step represents a node in the DAG
type Step struct {
	ID           uuid.UUID       `json:"id"`
	TenantID     uuid.UUID       `json:"tenant_id"`
	WorkflowID   uuid.UUID       `json:"workflow_id"`
	Name         string          `json:"name"`
	Type         StepType        `json:"type"`
	Config       json.RawMessage `json:"config"`
	BlockGroupID *uuid.UUID      `json:"block_group_id,omitempty"` // Reference to containing block group
	GroupRole    string          `json:"group_role,omitempty"`     // Role within block group (body, catch, then, else, etc.)
	PositionX    int             `json:"position_x"`
	PositionY    int             `json:"position_y"`

	// Block definition reference (for registry-based blocks)
	BlockDefinitionID *uuid.UUID `json:"block_definition_id,omitempty"`

	// Credential bindings: maps required credential names to actual credential IDs
	// Format: {"credential_name": "uuid-of-tenant-credential", ...}
	CredentialBindings json.RawMessage `json:"credential_bindings,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GetCredentialBindings parses and returns the credential bindings map
func (s *Step) GetCredentialBindings() (map[string]uuid.UUID, error) {
	return ParseCredentialBindings(s.CredentialBindings)
}

// NewStep creates a new step
func NewStep(tenantID, workflowID uuid.UUID, name string, stepType StepType, config json.RawMessage) *Step {
	now := time.Now().UTC()
	return &Step{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		WorkflowID:         workflowID,
		Name:               name,
		Type:               stepType,
		Config:             config,
		CredentialBindings: json.RawMessage(`{}`),
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

// SetPosition sets the position of the step
func (s *Step) SetPosition(x, y int) {
	s.PositionX = x
	s.PositionY = y
	s.UpdatedAt = time.Now().UTC()
}

// LLMStepConfig represents configuration for an LLM step
type LLMStepConfig struct {
	Model          string `json:"model"`
	PromptTemplate string `json:"prompt_template"`
	MaxTokens      int    `json:"max_tokens,omitempty"`
	Temperature    float64 `json:"temperature,omitempty"`
}

// ToolStepConfig represents configuration for a tool step
type ToolStepConfig struct {
	AdapterID    string          `json:"adapter_id"`
	InputMapping json.RawMessage `json:"input_mapping,omitempty"`
}

// ConditionStepConfig represents configuration for a condition step
type ConditionStepConfig struct {
	Expression string `json:"expression"`
}

// MapStepConfig represents configuration for a map step
type MapStepConfig struct {
	InputPath  string `json:"input_path"`
	Parallel   bool   `json:"parallel"`
	MaxWorkers int    `json:"max_workers,omitempty"`
}

// SubflowStepConfig represents configuration for a subflow step
type SubflowStepConfig struct {
	WorkflowID      uuid.UUID       `json:"workflow_id"`
	WorkflowVersion int             `json:"workflow_version,omitempty"`
	InputMapping    json.RawMessage `json:"input_mapping,omitempty"`
}

// LoopType represents the type of loop
type LoopType string

const (
	LoopTypeFor     LoopType = "for"
	LoopTypeForEach LoopType = "forEach"
	LoopTypeWhile   LoopType = "while"
	LoopTypeDoWhile LoopType = "doWhile"
)

// LoopStepConfig represents configuration for a loop step
type LoopStepConfig struct {
	LoopType      LoopType `json:"loop_type"`                 // for, forEach, while, doWhile
	Count         int      `json:"count,omitempty"`           // for: number of iterations
	InputPath     string   `json:"input_path,omitempty"`      // forEach: path to array
	Condition     string   `json:"condition,omitempty"`       // while/doWhile: condition expression
	MaxIterations int      `json:"max_iterations,omitempty"`  // safety limit (default: 100)
	AdapterID     string   `json:"adapter_id,omitempty"`      // adapter to execute per iteration
}

// WaitStepConfig represents configuration for a wait step
type WaitStepConfig struct {
	DurationMs int64  `json:"duration_ms,omitempty"` // delay in milliseconds
	Until      string `json:"until,omitempty"`       // ISO8601 datetime to wait until
}

// FunctionStepConfig represents configuration for a function step
type FunctionStepConfig struct {
	Code         string          `json:"code"`                    // JavaScript code to execute
	Language     string          `json:"language,omitempty"`      // javascript (default)
	TimeoutMs    int             `json:"timeout_ms,omitempty"`    // execution timeout
	OutputSchema json.RawMessage `json:"output_schema,omitempty"` // JSON Schema for output filtering
}

// RouterRoute represents a route option for the router step
type RouterRoute struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	StepID      string `json:"step_id,omitempty"` // target step to route to
}

// RouterStepConfig represents configuration for a router step
type RouterStepConfig struct {
	Routes   []RouterRoute `json:"routes"`
	Model    string        `json:"model,omitempty"`  // LLM model for classification
	Provider string        `json:"provider,omitempty"`
	Prompt   string        `json:"prompt,omitempty"` // custom classification prompt
}

// HumanInLoopNotification represents notification config
type HumanInLoopNotification struct {
	Type   string `json:"type"`   // email, slack, webhook
	Target string `json:"target"` // email address, channel, URL
}

// HumanInLoopField represents a required input field
type HumanInLoopField struct {
	Name     string `json:"name"`
	Type     string `json:"type"`               // boolean, string, number
	Required bool   `json:"required,omitempty"`
	Label    string `json:"label,omitempty"`
}

// HumanInLoopStepConfig represents configuration for human-in-the-loop step
type HumanInLoopStepConfig struct {
	TimeoutHours   int                      `json:"timeout_hours,omitempty"`
	Notification   *HumanInLoopNotification `json:"notification,omitempty"`
	ApprovalURL    bool                     `json:"approval_url,omitempty"`
	RequiredFields []HumanInLoopField       `json:"required_fields,omitempty"`
	Instructions   string                   `json:"instructions,omitempty"`
}

// StartStepConfig represents configuration for a start step
// Start steps are entry points for workflows and pass through input data
type StartStepConfig struct {
	// No configuration needed - start steps pass through input data
}

// SwitchCase represents a case in the switch step
type SwitchCase struct {
	Name       string `json:"name"`                 // Case identifier
	Expression string `json:"expression"`           // Condition expression (e.g., "$.status == 'active'")
	IsDefault  bool   `json:"is_default,omitempty"` // If true, this is the default case
}

// SwitchStepConfig represents configuration for a switch step (multi-branch routing)
type SwitchStepConfig struct {
	Cases []SwitchCase `json:"cases"` // List of cases to evaluate
	Mode  string       `json:"mode"`  // "rules" or "expression"
}

// FilterStepConfig represents configuration for a filter step
type FilterStepConfig struct {
	Expression string `json:"expression"` // Filter condition (e.g., "$.age > 18")
	KeepAll    bool   `json:"keep_all"`   // If false, filter items; if true, keep/remove all based on condition
}

// SplitStepConfig represents configuration for a split step (batch processing)
type SplitStepConfig struct {
	BatchSize int    `json:"batch_size"` // Number of items per batch (1 = process one at a time)
	InputPath string `json:"input_path"` // Path to array to split (e.g., "$.items")
}

// AggregateOperation represents an aggregation operation
type AggregateOperation struct {
	Operation   string `json:"operation"`             // sum, count, avg, min, max, first, last, concat
	Field       string `json:"field,omitempty"`       // Field to aggregate (for sum, avg, min, max)
	OutputField string `json:"output_field"`          // Name of output field
	Separator   string `json:"separator,omitempty"`   // For concat operation
}

// AggregateStepConfig represents configuration for an aggregate step
type AggregateStepConfig struct {
	GroupBy    string               `json:"group_by,omitempty"` // Field to group by (optional)
	Operations []AggregateOperation `json:"operations"`         // Aggregation operations to perform
}

// ErrorStepConfig represents configuration for an error step (stop and error)
type ErrorStepConfig struct {
	ErrorType    string `json:"error_type"`    // Type of error (e.g., "validation", "business", "system")
	ErrorMessage string `json:"error_message"` // Error message to display
	ErrorCode    string `json:"error_code"`    // Optional error code
}

// NoteStepConfig represents configuration for a note/documentation step
type NoteStepConfig struct {
	Content string `json:"content"` // Note/documentation content (markdown supported)
	Color   string `json:"color"`   // Optional color for the note (hex color)
}

// LogStepConfig represents configuration for a log step
type LogStepConfig struct {
	Message string `json:"message"`          // Log message (supports template variables like {{$.input.field}})
	Level   string `json:"level,omitempty"`  // Log level: debug, info, warn, error (default: info)
	Data    string `json:"data,omitempty"`   // JSON path to data to include in log (e.g., "$.input" or "$.steps.step1.output")
}
