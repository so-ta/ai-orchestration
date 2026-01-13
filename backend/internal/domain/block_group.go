package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// BlockGroupType represents the type of a block group (control flow construct)
type BlockGroupType string

const (
	BlockGroupTypeParallel   BlockGroupType = "parallel"    // Parallel execution group
	BlockGroupTypeTryCatch   BlockGroupType = "try_catch"   // Try-catch-finally error handling
	BlockGroupTypeIfElse     BlockGroupType = "if_else"     // Conditional branching
	BlockGroupTypeSwitchCase BlockGroupType = "switch_case" // Multi-branch routing
	BlockGroupTypeForeach    BlockGroupType = "foreach"     // Array iteration loop
	BlockGroupTypeWhile      BlockGroupType = "while"       // Condition-based loop
)

// ValidBlockGroupTypes returns all valid block group types
func ValidBlockGroupTypes() []BlockGroupType {
	return []BlockGroupType{
		BlockGroupTypeParallel,
		BlockGroupTypeTryCatch,
		BlockGroupTypeIfElse,
		BlockGroupTypeSwitchCase,
		BlockGroupTypeForeach,
		BlockGroupTypeWhile,
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
type GroupRole string

const (
	GroupRoleBody    GroupRole = "body"    // Main execution body (parallel, foreach, while)
	GroupRoleTry     GroupRole = "try"     // Try block (try_catch)
	GroupRoleCatch   GroupRole = "catch"   // Catch block (try_catch)
	GroupRoleFinally GroupRole = "finally" // Finally block (try_catch)
	GroupRoleThen    GroupRole = "then"    // Then branch (if_else)
	GroupRoleElse    GroupRole = "else"    // Else branch (if_else)
	GroupRoleDefault GroupRole = "default" // Default case (switch_case)
)

// ValidGroupRoles returns all valid group roles
func ValidGroupRoles() []GroupRole {
	return []GroupRole{
		GroupRoleBody,
		GroupRoleTry,
		GroupRoleCatch,
		GroupRoleFinally,
		GroupRoleThen,
		GroupRoleElse,
		GroupRoleDefault,
	}
}

// IsValid checks if the group role is valid
func (r GroupRole) IsValid() bool {
	// Allow case_N roles for switch_case
	if len(r) > 5 && r[:5] == "case_" {
		return true
	}
	for _, valid := range ValidGroupRoles() {
		if r == valid {
			return true
		}
	}
	return false
}

// BlockGroup represents a control flow construct that groups multiple steps
type BlockGroup struct {
	ID            uuid.UUID       `json:"id"`
	TenantID      uuid.UUID       `json:"tenant_id"`
	WorkflowID    uuid.UUID       `json:"workflow_id"`
	Name          string          `json:"name"`
	Type          BlockGroupType  `json:"type"`
	Config        json.RawMessage `json:"config"`
	ParentGroupID *uuid.UUID      `json:"parent_group_id,omitempty"` // For nested groups
	PositionX     int             `json:"position_x"`
	PositionY     int             `json:"position_y"`
	Width         int             `json:"width"`
	Height        int             `json:"height"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// NewBlockGroup creates a new block group
func NewBlockGroup(tenantID, workflowID uuid.UUID, name string, groupType BlockGroupType) *BlockGroup {
	now := time.Now().UTC()
	return &BlockGroup{
		ID:         uuid.New(),
		TenantID:   tenantID,
		WorkflowID: workflowID,
		Name:       name,
		Type:       groupType,
		Config:     json.RawMessage("{}"),
		Width:      400,
		Height:     300,
		CreatedAt:  now,
		UpdatedAt:  now,
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
type ParallelConfig struct {
	MaxConcurrent int  `json:"max_concurrent,omitempty"` // Max concurrent executions (0 = unlimited)
	FailFast      bool `json:"fail_fast,omitempty"`      // Stop all on first failure
}

// TryCatchConfig represents configuration for try-catch-finally block group
type TryCatchConfig struct {
	ErrorTypes []string `json:"error_types,omitempty"` // Error types to catch ("*" = all)
	RetryCount int      `json:"retry_count,omitempty"` // Number of retries before catch
	RetryDelay int      `json:"retry_delay_ms,omitempty"` // Delay between retries in ms
}

// IfElseConfig represents configuration for if-else block group
type IfElseConfig struct {
	Condition string `json:"condition"` // Condition expression (e.g., "$.status == 'active'")
}

// SwitchCaseConfig represents configuration for switch-case block group
type SwitchCaseConfig struct {
	Expression string   `json:"expression"`           // Expression to evaluate
	Cases      []string `json:"cases"`                // Case values
	HasDefault bool     `json:"has_default,omitempty"` // Whether default case exists
}

// ForeachConfig represents configuration for foreach block group
type ForeachConfig struct {
	InputPath  string `json:"input_path"`            // Path to array (e.g., "$.items")
	Parallel   bool   `json:"parallel,omitempty"`    // Execute iterations in parallel
	MaxWorkers int    `json:"max_workers,omitempty"` // Max parallel workers
}

// WhileConfig represents configuration for while block group
type WhileConfig struct {
	Condition     string `json:"condition"`               // Condition expression
	MaxIterations int    `json:"max_iterations,omitempty"` // Safety limit (default: 100)
	DoWhile       bool   `json:"do_while,omitempty"`       // Execute at least once (do-while)
}

// BlockGroupRun represents the execution state of a block group
type BlockGroupRun struct {
	ID           uuid.UUID       `json:"id"`
	TenantID     uuid.UUID       `json:"tenant_id"`
	RunID        uuid.UUID       `json:"run_id"`
	BlockGroupID uuid.UUID       `json:"block_group_id"`
	Status       StepRunStatus   `json:"status"`
	Iteration    int             `json:"iteration,omitempty"` // For loop groups
	Input        json.RawMessage `json:"input,omitempty"`
	Output       json.RawMessage `json:"output,omitempty"`
	Error        string          `json:"error,omitempty"`
	StartedAt    *time.Time      `json:"started_at,omitempty"`
	CompletedAt  *time.Time      `json:"completed_at,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

// NewBlockGroupRun creates a new block group run
func NewBlockGroupRun(tenantID, runID, blockGroupID uuid.UUID) *BlockGroupRun {
	now := time.Now().UTC()
	return &BlockGroupRun{
		ID:           uuid.New(),
		TenantID:     tenantID,
		RunID:        runID,
		BlockGroupID: blockGroupID,
		Status:       StepRunStatusPending,
		CreatedAt:    now,
	}
}
