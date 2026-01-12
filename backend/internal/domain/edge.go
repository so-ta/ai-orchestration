package domain

import (
	"time"

	"github.com/google/uuid"
)

// Edge represents a connection between two steps in the DAG
type Edge struct {
	ID           uuid.UUID `json:"id"`
	WorkflowID   uuid.UUID `json:"workflow_id"`
	SourceStepID uuid.UUID `json:"source_step_id"`
	TargetStepID uuid.UUID `json:"target_step_id"`
	SourcePort   string    `json:"source_port,omitempty"` // Output port name (e.g., "true", "false")
	TargetPort   string    `json:"target_port,omitempty"` // Input port name (e.g., "input", "items")
	Condition    string    `json:"condition,omitempty"`   // Optional condition expression for edge traversal
	CreatedAt    time.Time `json:"created_at"`
}

// NewEdge creates a new edge
func NewEdge(workflowID, sourceStepID, targetStepID uuid.UUID, condition string) *Edge {
	return &Edge{
		ID:           uuid.New(),
		WorkflowID:   workflowID,
		SourceStepID: sourceStepID,
		TargetStepID: targetStepID,
		SourcePort:   "", // Default: use default output port
		TargetPort:   "", // Default: use default input port
		Condition:    condition,
		CreatedAt:    time.Now().UTC(),
	}
}

// NewEdgeWithPort creates a new edge with specific source and target ports
func NewEdgeWithPort(workflowID, sourceStepID, targetStepID uuid.UUID, sourcePort, targetPort, condition string) *Edge {
	return &Edge{
		ID:           uuid.New(),
		WorkflowID:   workflowID,
		SourceStepID: sourceStepID,
		TargetStepID: targetStepID,
		SourcePort:   sourcePort,
		TargetPort:   targetPort,
		Condition:    condition,
		CreatedAt:    time.Now().UTC(),
	}
}
