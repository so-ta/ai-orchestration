package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewEdge(t *testing.T) {
	tenantID := uuid.New()
	projectID := uuid.New()
	sourceStepID := uuid.New()
	targetStepID := uuid.New()
	condition := "$.success == true"

	edge := NewEdge(tenantID, projectID, sourceStepID, targetStepID, condition)

	if edge.ID == uuid.Nil {
		t.Error("NewEdge() should generate a non-nil UUID")
	}
	if edge.TenantID != tenantID {
		t.Errorf("NewEdge() TenantID = %v, want %v", edge.TenantID, tenantID)
	}
	if edge.ProjectID != projectID {
		t.Errorf("NewEdge() ProjectID = %v, want %v", edge.ProjectID, projectID)
	}
	if edge.SourceStepID == nil || *edge.SourceStepID != sourceStepID {
		t.Error("NewEdge() SourceStepID mismatch")
	}
	if edge.TargetStepID == nil || *edge.TargetStepID != targetStepID {
		t.Error("NewEdge() TargetStepID mismatch")
	}
	if edge.Condition == nil || *edge.Condition != condition {
		t.Error("NewEdge() Condition mismatch")
	}
}

func TestNewEdge_NoCondition(t *testing.T) {
	tenantID := uuid.New()
	projectID := uuid.New()
	sourceStepID := uuid.New()
	targetStepID := uuid.New()

	edge := NewEdge(tenantID, projectID, sourceStepID, targetStepID, "")

	if edge.Condition != nil {
		t.Error("NewEdge() with empty condition should have nil Condition")
	}
}

func TestNewEdgeWithPort(t *testing.T) {
	tenantID := uuid.New()
	projectID := uuid.New()
	sourceStepID := uuid.New()
	targetStepID := uuid.New()

	edge := NewEdgeWithPort(tenantID, projectID, sourceStepID, targetStepID, "true", "")

	if edge.SourcePort != "true" {
		t.Errorf("NewEdgeWithPort() SourcePort = %v, want true", edge.SourcePort)
	}
}

func TestNewEdgeToGroup(t *testing.T) {
	tenantID := uuid.New()
	projectID := uuid.New()
	sourceStepID := uuid.New()
	targetGroupID := uuid.New()

	edge := NewEdgeToGroup(tenantID, projectID, sourceStepID, targetGroupID, "output")

	if edge.SourceStepID == nil || *edge.SourceStepID != sourceStepID {
		t.Error("NewEdgeToGroup() SourceStepID mismatch")
	}
	if edge.TargetBlockGroupID == nil || *edge.TargetBlockGroupID != targetGroupID {
		t.Error("NewEdgeToGroup() TargetBlockGroupID mismatch")
	}
	if edge.SourcePort != "output" {
		t.Errorf("NewEdgeToGroup() SourcePort = %v, want output", edge.SourcePort)
	}
}

func TestNewEdgeFromGroup(t *testing.T) {
	tenantID := uuid.New()
	projectID := uuid.New()
	sourceGroupID := uuid.New()
	targetStepID := uuid.New()

	edge := NewEdgeFromGroup(tenantID, projectID, sourceGroupID, targetStepID)

	if edge.SourceBlockGroupID == nil || *edge.SourceBlockGroupID != sourceGroupID {
		t.Error("NewEdgeFromGroup() SourceBlockGroupID mismatch")
	}
	if edge.TargetStepID == nil || *edge.TargetStepID != targetStepID {
		t.Error("NewEdgeFromGroup() TargetStepID mismatch")
	}
	if edge.SourcePort != "out" {
		t.Errorf("NewEdgeFromGroup() SourcePort = %v, want out", edge.SourcePort)
	}
}

func TestNewGroupToGroupEdge(t *testing.T) {
	tenantID := uuid.New()
	projectID := uuid.New()
	sourceGroupID := uuid.New()
	targetGroupID := uuid.New()

	edge := NewGroupToGroupEdge(tenantID, projectID, sourceGroupID, targetGroupID)

	if edge.SourceBlockGroupID == nil || *edge.SourceBlockGroupID != sourceGroupID {
		t.Error("NewGroupToGroupEdge() SourceBlockGroupID mismatch")
	}
	if edge.TargetBlockGroupID == nil || *edge.TargetBlockGroupID != targetGroupID {
		t.Error("NewGroupToGroupEdge() TargetBlockGroupID mismatch")
	}
	if edge.SourcePort != "out" {
		t.Errorf("NewGroupToGroupEdge() SourcePort = %v, want out", edge.SourcePort)
	}
}

func TestEdge_IsStepToStep(t *testing.T) {
	stepID1 := uuid.New()
	stepID2 := uuid.New()
	groupID := uuid.New()

	tests := []struct {
		name               string
		sourceStepID       *uuid.UUID
		targetStepID       *uuid.UUID
		sourceBlockGroupID *uuid.UUID
		targetBlockGroupID *uuid.UUID
		want               bool
	}{
		{
			name:         "step to step",
			sourceStepID: &stepID1,
			targetStepID: &stepID2,
			want:         true,
		},
		{
			name:               "step to group",
			sourceStepID:       &stepID1,
			targetBlockGroupID: &groupID,
			want:               false,
		},
		{
			name:               "group to step",
			sourceBlockGroupID: &groupID,
			targetStepID:       &stepID2,
			want:               false,
		},
		{
			name:               "group to group",
			sourceBlockGroupID: &groupID,
			targetBlockGroupID: &groupID,
			want:               false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			edge := &Edge{
				SourceStepID:       tt.sourceStepID,
				TargetStepID:       tt.targetStepID,
				SourceBlockGroupID: tt.sourceBlockGroupID,
				TargetBlockGroupID: tt.targetBlockGroupID,
			}
			if got := edge.IsStepToStep(); got != tt.want {
				t.Errorf("IsStepToStep() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEdge_IsStepToGroup(t *testing.T) {
	stepID := uuid.New()
	groupID := uuid.New()

	tests := []struct {
		name               string
		sourceStepID       *uuid.UUID
		targetBlockGroupID *uuid.UUID
		want               bool
	}{
		{
			name:               "step to group",
			sourceStepID:       &stepID,
			targetBlockGroupID: &groupID,
			want:               true,
		},
		{
			name:         "step to step",
			sourceStepID: &stepID,
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			edge := &Edge{
				SourceStepID:       tt.sourceStepID,
				TargetBlockGroupID: tt.targetBlockGroupID,
			}
			if got := edge.IsStepToGroup(); got != tt.want {
				t.Errorf("IsStepToGroup() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEdge_IsGroupToStep(t *testing.T) {
	stepID := uuid.New()
	groupID := uuid.New()

	tests := []struct {
		name               string
		sourceBlockGroupID *uuid.UUID
		targetStepID       *uuid.UUID
		want               bool
	}{
		{
			name:               "group to step",
			sourceBlockGroupID: &groupID,
			targetStepID:       &stepID,
			want:               true,
		},
		{
			name:               "group to group",
			sourceBlockGroupID: &groupID,
			want:               false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			edge := &Edge{
				SourceBlockGroupID: tt.sourceBlockGroupID,
				TargetStepID:       tt.targetStepID,
			}
			if got := edge.IsGroupToStep(); got != tt.want {
				t.Errorf("IsGroupToStep() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEdge_IsGroupToGroup(t *testing.T) {
	stepID := uuid.New()
	groupID1 := uuid.New()
	groupID2 := uuid.New()

	tests := []struct {
		name               string
		sourceStepID       *uuid.UUID
		targetStepID       *uuid.UUID
		sourceBlockGroupID *uuid.UUID
		targetBlockGroupID *uuid.UUID
		want               bool
	}{
		{
			name:               "group to group",
			sourceBlockGroupID: &groupID1,
			targetBlockGroupID: &groupID2,
			want:               true,
		},
		{
			name:               "step to group",
			sourceStepID:       &stepID,
			targetBlockGroupID: &groupID1,
			want:               false,
		},
		{
			name:               "group to step",
			sourceBlockGroupID: &groupID1,
			targetStepID:       &stepID,
			want:               false,
		},
		{
			name:         "step to step",
			sourceStepID: &stepID,
			targetStepID: &stepID,
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			edge := &Edge{
				SourceStepID:       tt.sourceStepID,
				TargetStepID:       tt.targetStepID,
				SourceBlockGroupID: tt.sourceBlockGroupID,
				TargetBlockGroupID: tt.targetBlockGroupID,
			}
			if got := edge.IsGroupToGroup(); got != tt.want {
				t.Errorf("IsGroupToGroup() = %v, want %v", got, tt.want)
			}
		})
	}
}
