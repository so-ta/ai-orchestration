package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestBlockGroupType_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		t        BlockGroupType
		expected bool
	}{
		// Valid types (4 types only)
		{"parallel", BlockGroupTypeParallel, true},
		{"try_catch", BlockGroupTypeTryCatch, true},
		{"foreach", BlockGroupTypeForeach, true},
		{"while", BlockGroupTypeWhile, true},

		// Invalid types (removed)
		{"if_else removed", BlockGroupType("if_else"), false},
		{"switch_case removed", BlockGroupType("switch_case"), false},

		// Other invalid types
		{"empty", BlockGroupType(""), false},
		{"random", BlockGroupType("random"), false},
		{"loop", BlockGroupType("loop"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.t.IsValid())
		})
	}
}

func TestValidBlockGroupTypes(t *testing.T) {
	types := ValidBlockGroupTypes()
	assert.Len(t, types, 4) // Only 4 types now
	assert.Contains(t, types, BlockGroupTypeParallel)
	assert.Contains(t, types, BlockGroupTypeTryCatch)
	assert.Contains(t, types, BlockGroupTypeForeach)
	assert.Contains(t, types, BlockGroupTypeWhile)
}

func TestGroupRole_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		role     GroupRole
		expected bool
	}{
		// Valid role (body only)
		{"body", GroupRoleBody, true},

		// Invalid roles (removed)
		{"try removed", GroupRole("try"), false},
		{"catch removed", GroupRole("catch"), false},
		{"finally removed", GroupRole("finally"), false},
		{"then removed", GroupRole("then"), false},
		{"else removed", GroupRole("else"), false},
		{"default removed", GroupRole("default"), false},
		{"case_0 removed", GroupRole("case_0"), false},

		// Other invalid roles
		{"empty", GroupRole(""), false},
		{"random", GroupRole("random"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.role.IsValid())
		})
	}
}

func TestValidGroupRoles(t *testing.T) {
	roles := ValidGroupRoles()
	assert.Len(t, roles, 1) // Only body role now
	assert.Contains(t, roles, GroupRoleBody)
}

func TestNewBlockGroup(t *testing.T) {
	tenantID := uuid.New()
	workflowID := uuid.New()
	name := "Test Group"
	groupType := BlockGroupTypeParallel

	group := NewBlockGroup(tenantID, workflowID, name, groupType)

	assert.NotEqual(t, uuid.Nil, group.ID)
	assert.Equal(t, tenantID, group.TenantID)
	assert.Equal(t, workflowID, group.WorkflowID)
	assert.Equal(t, name, group.Name)
	assert.Equal(t, groupType, group.Type)
	assert.Equal(t, 400, group.Width)  // Default width
	assert.Equal(t, 300, group.Height) // Default height
	assert.Nil(t, group.ParentGroupID)
	assert.Nil(t, group.PreProcess)
	assert.Nil(t, group.PostProcess)
	assert.NotZero(t, group.CreatedAt)
	assert.NotZero(t, group.UpdatedAt)
}

func TestBlockGroup_SetPosition(t *testing.T) {
	group := NewBlockGroup(uuid.New(), uuid.New(), "Test", BlockGroupTypeParallel)
	originalUpdatedAt := group.UpdatedAt

	group.SetPosition(100, 200)

	assert.Equal(t, 100, group.PositionX)
	assert.Equal(t, 200, group.PositionY)
	assert.True(t, group.UpdatedAt.After(originalUpdatedAt) || group.UpdatedAt.Equal(originalUpdatedAt))
}

func TestBlockGroup_SetSize(t *testing.T) {
	group := NewBlockGroup(uuid.New(), uuid.New(), "Test", BlockGroupTypeForeach)
	originalUpdatedAt := group.UpdatedAt

	group.SetSize(500, 400)

	assert.Equal(t, 500, group.Width)
	assert.Equal(t, 400, group.Height)
	assert.True(t, group.UpdatedAt.After(originalUpdatedAt) || group.UpdatedAt.Equal(originalUpdatedAt))
}

func TestBlockGroup_SetParent(t *testing.T) {
	group := NewBlockGroup(uuid.New(), uuid.New(), "Test", BlockGroupTypeWhile)
	parentID := uuid.New()

	group.SetParent(&parentID)

	assert.Equal(t, &parentID, group.ParentGroupID)

	// Test setting to nil
	group.SetParent(nil)
	assert.Nil(t, group.ParentGroupID)
}

func TestNewBlockGroupRun(t *testing.T) {
	tenantID := uuid.New()
	runID := uuid.New()
	blockGroupID := uuid.New()

	run := NewBlockGroupRun(tenantID, runID, blockGroupID)

	assert.NotEqual(t, uuid.Nil, run.ID)
	assert.Equal(t, tenantID, run.TenantID)
	assert.Equal(t, runID, run.RunID)
	assert.Equal(t, blockGroupID, run.BlockGroupID)
	assert.Equal(t, StepRunStatusPending, run.Status)
	assert.NotZero(t, run.CreatedAt)
}

func TestBlockGroup_WithPrePostProcess(t *testing.T) {
	group := NewBlockGroup(uuid.New(), uuid.New(), "Test", BlockGroupTypeTryCatch)

	// Pre/PostProcess should be nil by default
	assert.Nil(t, group.PreProcess)
	assert.Nil(t, group.PostProcess)

	// Set pre_process
	preCode := "return { ...input, processed: true };"
	group.PreProcess = &preCode
	assert.Equal(t, &preCode, group.PreProcess)

	// Set post_process
	postCode := "return { result: output.data };"
	group.PostProcess = &postCode
	assert.Equal(t, &postCode, group.PostProcess)
}

func TestBlockGroupTypes_Constants(t *testing.T) {
	// Verify constant values match expected strings
	assert.Equal(t, BlockGroupType("parallel"), BlockGroupTypeParallel)
	assert.Equal(t, BlockGroupType("try_catch"), BlockGroupTypeTryCatch)
	assert.Equal(t, BlockGroupType("foreach"), BlockGroupTypeForeach)
	assert.Equal(t, BlockGroupType("while"), BlockGroupTypeWhile)
}

func TestGroupRole_Constants(t *testing.T) {
	// Verify constant values match expected strings
	assert.Equal(t, GroupRole("body"), GroupRoleBody)
}
