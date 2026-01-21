package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// BlockGroupType represents the type of a block group (control flow construct)
// Redesigned to 4 types only: parallel, try_catch, foreach, while
// Removed: if_else (use condition block), switch_case (use switch block)
type BlockGroupType string

const (
	BlockGroupTypeParallel BlockGroupType = "parallel"  // Parallel execution of different flows
	BlockGroupTypeTryCatch BlockGroupType = "try_catch" // Error handling with retry support
	BlockGroupTypeForeach  BlockGroupType = "foreach"   // Array iteration (same process for each element)
	BlockGroupTypeWhile    BlockGroupType = "while"     // Condition-based loop
	BlockGroupTypeAgent    BlockGroupType = "agent"     // AI Agent with tool calling (child steps = tools)
)

// ValidBlockGroupTypes returns all valid block group types (5 types: parallel, try_catch, foreach, while, agent)
func ValidBlockGroupTypes() []BlockGroupType {
	return []BlockGroupType{
		BlockGroupTypeParallel,
		BlockGroupTypeTryCatch,
		BlockGroupTypeForeach,
		BlockGroupTypeWhile,
		BlockGroupTypeAgent,
	}
}

// IsValid checks if the block group type is valid
func (t BlockGroupType) IsValid() bool {
	for _, valid := range ValidBlockGroupTypes() {
		if t == valid {
			return true
		}
	}
	return false
}

// GroupRole represents the role of a step within a block group
// Simplified: all groups now only have "body" role
// Removed: try, catch, finally, then, else, default, case_N
// Error handling is done via output ports (out, error)
type GroupRole string

const (
	GroupRoleBody GroupRole = "body" // Main execution body (all group types)
)

// ValidGroupRoles returns all valid group roles (body only)
func ValidGroupRoles() []GroupRole {
	return []GroupRole{
		GroupRoleBody,
	}
}

// IsValid checks if the group role is valid
func (r GroupRole) IsValid() bool {
	return r == GroupRoleBody
}

// BlockGroup represents a control flow construct that groups multiple steps
// Redesigned with pre_process/post_process for input/output transformation
// Similar to regular blocks, providing unified interface
type BlockGroup struct {
	ID            uuid.UUID       `json:"id"`
	TenantID      uuid.UUID       `json:"tenant_id"`
	ProjectID     uuid.UUID       `json:"project_id"`
	Name          string          `json:"name"`
	Type          BlockGroupType  `json:"type"`
	Config        json.RawMessage `json:"config"`
	ParentGroupID *uuid.UUID      `json:"parent_group_id,omitempty"` // For nested groups

	// Input/Output transformation (same as regular blocks)
	PreProcess  *string `json:"pre_process,omitempty"`  // JS: external IN -> internal IN
	PostProcess *string `json:"post_process,omitempty"` // JS: internal OUT -> external OUT

	// UI positioning
	PositionX int `json:"position_x"`
	PositionY int `json:"position_y"`
	Width     int `json:"width"`
	Height    int `json:"height"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewBlockGroup creates a new block group
func NewBlockGroup(tenantID, projectID uuid.UUID, name string, groupType BlockGroupType) *BlockGroup {
	now := time.Now().UTC()
	return &BlockGroup{
		ID:        uuid.New(),
		TenantID:  tenantID,
		ProjectID: projectID,
		Name:      name,
		Type:      groupType,
		Config:    json.RawMessage("{}"),
		Width:     400,
		Height:    300,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// SetPosition sets the position of the block group
func (g *BlockGroup) SetPosition(x, y int) {
	g.PositionX = x
	g.PositionY = y
	g.UpdatedAt = time.Now().UTC()
}

// SetSize sets the size of the block group
func (g *BlockGroup) SetSize(width, height int) {
	g.Width = width
	g.Height = height
	g.UpdatedAt = time.Now().UTC()
}

// SetParent sets the parent group for nesting
func (g *BlockGroup) SetParent(parentID *uuid.UUID) {
	g.ParentGroupID = parentID
	g.UpdatedAt = time.Now().UTC()
}

// ParallelConfig represents configuration for parallel block group
// Executes multiple independent flows concurrently within the group
type ParallelConfig struct {
	MaxConcurrent int  `json:"max_concurrent,omitempty"` // Max concurrent executions (0 = unlimited)
	FailFast      bool `json:"fail_fast,omitempty"`      // Stop all on first failure
}

// TryCatchConfig represents configuration for try-catch block group
// Simplified: catch logic is handled via error output port to external blocks
type TryCatchConfig struct {
	RetryCount int `json:"retry_count,omitempty"`    // Number of retries before error (default: 0)
	RetryDelay int `json:"retry_delay_ms,omitempty"` // Delay between retries in ms
}

// ForeachConfig represents configuration for foreach block group
// Applies the same body process to each element in an array
type ForeachConfig struct {
	InputPath  string `json:"input_path,omitempty"`  // Path to array (default: "$.items")
	Parallel   bool   `json:"parallel,omitempty"`    // Execute iterations in parallel
	MaxWorkers int    `json:"max_workers,omitempty"` // Max parallel workers (0 = unlimited)
}

// WhileConfig represents configuration for while block group
// Repeats body execution while condition is true
type WhileConfig struct {
	Condition     string `json:"condition"`                // Condition expression (e.g., "$.counter < $.target")
	MaxIterations int    `json:"max_iterations,omitempty"` // Safety limit (default: 100)
	DoWhile       bool   `json:"do_while,omitempty"`       // Execute at least once before checking condition
}

// AgentConfig represents configuration for agent block group
// Implements ReAct loop where child steps become callable tools
type AgentConfig struct {
	Provider      string  `json:"provider"`                   // LLM provider: "openai", "anthropic"
	Model         string  `json:"model"`                      // Model ID (e.g., "claude-sonnet-4-20250514")
	SystemPrompt  string  `json:"system_prompt"`              // System prompt defining agent behavior
	MaxIterations int     `json:"max_iterations,omitempty"`   // ReAct loop max iterations (default: 30)
	Temperature   float64 `json:"temperature,omitempty"`      // LLM temperature (default: 0.7)
	ToolChoice    string  `json:"tool_choice,omitempty"`      // "auto", "none", "required" (default: "auto")
	EnableMemory  bool    `json:"enable_memory,omitempty"`    // Enable conversation memory
	MemoryWindow  int     `json:"memory_window,omitempty"`    // Number of messages to keep in memory (default: 20)
}
