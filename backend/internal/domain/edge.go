package domain

import (
	"time"

	"github.com/google/uuid"
)

// Edge represents a connection between two steps or block groups in the DAG
// An edge can connect:
// - Step to Step (traditional edge)
// - Step to BlockGroup (entering a group)
// - BlockGroup to Step (exiting a group)
// - BlockGroup to BlockGroup (group to group connection)
type Edge struct {
	ID                 uuid.UUID  `json:"id"`
	TenantID           uuid.UUID  `json:"tenant_id"`
	WorkflowID         uuid.UUID  `json:"workflow_id"`
	SourceStepID       *uuid.UUID `json:"source_step_id,omitempty"`        // Source step (nil if from group)
	TargetStepID       *uuid.UUID `json:"target_step_id,omitempty"`        // Target step (nil if to group)
	SourceBlockGroupID *uuid.UUID `json:"source_block_group_id,omitempty"` // Source group (nil if from step)
	TargetBlockGroupID *uuid.UUID `json:"target_block_group_id,omitempty"` // Target group (nil if to step)
	SourcePort         string     `json:"source_port,omitempty"`           // Output port name (e.g., "true", "false", "out")
	TargetPort         string     `json:"target_port,omitempty"`           // Input port name (e.g., "input", "items", "in")
	Condition          *string    `json:"condition,omitempty"`             // Optional condition expression for edge traversal
	CreatedAt          time.Time  `json:"created_at"`
}

// NewEdge creates a new edge between steps
func NewEdge(tenantID, workflowID, sourceStepID, targetStepID uuid.UUID, condition string) *Edge {
	var cond *string
	if condition != "" {
		cond = &condition
	}
	return &Edge{
		ID:           uuid.New(),
		TenantID:     tenantID,
		WorkflowID:   workflowID,
		SourceStepID: &sourceStepID,
		TargetStepID: &targetStepID,
		SourcePort:   "", // Default: use default output port
		TargetPort:   "", // Default: use default input port
		Condition:    cond,
		CreatedAt:    time.Now().UTC(),
	}
}

// NewEdgeWithPort creates a new edge between steps with specific source and target ports
func NewEdgeWithPort(tenantID, workflowID, sourceStepID, targetStepID uuid.UUID, sourcePort, targetPort, condition string) *Edge {
	var cond *string
	if condition != "" {
		cond = &condition
	}
	return &Edge{
		ID:           uuid.New(),
		TenantID:     tenantID,
		WorkflowID:   workflowID,
		SourceStepID: &sourceStepID,
		TargetStepID: &targetStepID,
		SourcePort:   sourcePort,
		TargetPort:   targetPort,
		Condition:    cond,
		CreatedAt:    time.Now().UTC(),
	}
}

// NewEdgeToGroup creates a new edge from a step to a block group
func NewEdgeToGroup(tenantID, workflowID, sourceStepID, targetGroupID uuid.UUID, sourcePort string) *Edge {
	return &Edge{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		WorkflowID:         workflowID,
		SourceStepID:       &sourceStepID,
		TargetBlockGroupID: &targetGroupID,
		SourcePort:         sourcePort,
		TargetPort:         "in", // Default group input port
		CreatedAt:          time.Now().UTC(),
	}
}

// NewEdgeFromGroup creates a new edge from a block group to a step
func NewEdgeFromGroup(tenantID, workflowID, sourceGroupID, targetStepID uuid.UUID, targetPort string) *Edge {
	return &Edge{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		WorkflowID:         workflowID,
		SourceBlockGroupID: &sourceGroupID,
		TargetStepID:       &targetStepID,
		SourcePort:         "out", // Default group output port
		TargetPort:         targetPort,
		CreatedAt:          time.Now().UTC(),
	}
}

// NewGroupToGroupEdge creates a new edge between two block groups
func NewGroupToGroupEdge(tenantID, workflowID, sourceGroupID, targetGroupID uuid.UUID) *Edge {
	return &Edge{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		WorkflowID:         workflowID,
		SourceBlockGroupID: &sourceGroupID,
		TargetBlockGroupID: &targetGroupID,
		SourcePort:         "out",
		TargetPort:         "in",
		CreatedAt:          time.Now().UTC(),
	}
}

// IsStepToStep returns true if the edge connects two steps
func (e *Edge) IsStepToStep() bool {
	return e.SourceStepID != nil && e.TargetStepID != nil
}

// IsStepToGroup returns true if the edge connects a step to a group
func (e *Edge) IsStepToGroup() bool {
	return e.SourceStepID != nil && e.TargetBlockGroupID != nil
}

// IsGroupToStep returns true if the edge connects a group to a step
func (e *Edge) IsGroupToStep() bool {
	return e.SourceBlockGroupID != nil && e.TargetStepID != nil
}

// IsGroupToGroup returns true if the edge connects two groups
func (e *Edge) IsGroupToGroup() bool {
	return e.SourceBlockGroupID != nil && e.TargetBlockGroupID != nil
}
