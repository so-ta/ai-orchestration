package domain

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestNewBlockGroup(t *testing.T) {
	tenantID := uuid.New()
	projectID := uuid.New()
	name := "Test Group"
	groupType := BlockGroupTypeParallel

	group := NewBlockGroup(tenantID, projectID, name, groupType)

	if group.ID == uuid.Nil {
		t.Error("NewBlockGroup() should generate a non-nil UUID")
	}
	if group.TenantID != tenantID {
		t.Errorf("NewBlockGroup() TenantID = %v, want %v", group.TenantID, tenantID)
	}
	if group.ProjectID != projectID {
		t.Errorf("NewBlockGroup() ProjectID = %v, want %v", group.ProjectID, projectID)
	}
	if group.Name != name {
		t.Errorf("NewBlockGroup() Name = %v, want %v", group.Name, name)
	}
	if group.Type != groupType {
		t.Errorf("NewBlockGroup() Type = %v, want %v", group.Type, groupType)
	}
	if group.Width != 400 {
		t.Errorf("NewBlockGroup() Width = %v, want 400", group.Width)
	}
	if group.Height != 300 {
		t.Errorf("NewBlockGroup() Height = %v, want 300", group.Height)
	}
}

func TestBlockGroupType_IsValid(t *testing.T) {
	validTypes := []BlockGroupType{
		BlockGroupTypeParallel, BlockGroupTypeTryCatch,
		BlockGroupTypeForeach, BlockGroupTypeWhile, BlockGroupTypeAgent,
	}

	for _, gt := range validTypes {
		t.Run(string(gt), func(t *testing.T) {
			if !gt.IsValid() {
				t.Errorf("IsValid() = false for valid type %v", gt)
			}
		})
	}

	invalidTypes := []BlockGroupType{
		BlockGroupType("invalid"),
		BlockGroupType(""),
	}

	for _, gt := range invalidTypes {
		t.Run(string(gt), func(t *testing.T) {
			if gt.IsValid() {
				t.Errorf("IsValid() = true for invalid type %v", gt)
			}
		})
	}
}

func TestBlockGroup_SetConfig(t *testing.T) {
	group := NewBlockGroup(uuid.New(), uuid.New(), "Test", BlockGroupTypeParallel)
	config := map[string]interface{}{
		"max_concurrent": 5,
		"fail_fast":      true,
	}

	err := group.SetConfig(config)
	if err != nil {
		t.Fatalf("SetConfig() error = %v", err)
	}

	got, err := group.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig() error = %v", err)
	}

	if int(got["max_concurrent"].(float64)) != 5 {
		t.Errorf("SetConfig() max_concurrent mismatch")
	}
	if got["fail_fast"] != true {
		t.Errorf("SetConfig() fail_fast mismatch")
	}
}

func TestBlockGroup_GetConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  json.RawMessage
		wantLen int
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "empty config",
			config:  json.RawMessage("{}"),
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "valid config",
			config:  json.RawMessage(`{"max_concurrent": 10}`),
			wantLen: 1,
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			config:  json.RawMessage(`{invalid`),
			wantLen: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			group := &BlockGroup{Config: tt.config}
			got, err := group.GetConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != tt.wantLen {
				t.Errorf("GetConfig() got %d keys, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestBlockGroup_SetPosition(t *testing.T) {
	group := NewBlockGroup(uuid.New(), uuid.New(), "Test", BlockGroupTypeParallel)

	group.SetPosition(100, 200)

	if group.PositionX != 100 {
		t.Errorf("SetPosition() PositionX = %v, want 100", group.PositionX)
	}
	if group.PositionY != 200 {
		t.Errorf("SetPosition() PositionY = %v, want 200", group.PositionY)
	}
}

func TestBlockGroup_SetSize(t *testing.T) {
	group := NewBlockGroup(uuid.New(), uuid.New(), "Test", BlockGroupTypeParallel)

	group.SetSize(500, 400)

	if group.Width != 500 {
		t.Errorf("SetSize() Width = %v, want 500", group.Width)
	}
	if group.Height != 400 {
		t.Errorf("SetSize() Height = %v, want 400", group.Height)
	}
}

func TestBlockGroup_SetParent(t *testing.T) {
	group := NewBlockGroup(uuid.New(), uuid.New(), "Child", BlockGroupTypeParallel)
	parentID := uuid.New()

	group.SetParent(&parentID)

	if group.ParentGroupID == nil || *group.ParentGroupID != parentID {
		t.Error("SetParent() should set ParentGroupID")
	}
}

func TestBlockGroup_ClearParent(t *testing.T) {
	group := NewBlockGroup(uuid.New(), uuid.New(), "Child", BlockGroupTypeParallel)
	parentID := uuid.New()
	group.SetParent(&parentID)

	group.ClearParent()

	if group.ParentGroupID != nil {
		t.Error("ClearParent() should clear ParentGroupID")
	}
}

func TestBlockGroup_SetPreProcess(t *testing.T) {
	group := NewBlockGroup(uuid.New(), uuid.New(), "Test", BlockGroupTypeForeach)
	code := "return { ...input, timestamp: Date.now() };"

	group.SetPreProcess(code)

	if group.PreProcess == nil || *group.PreProcess != code {
		t.Errorf("SetPreProcess() PreProcess = %v, want %v", group.PreProcess, code)
	}
}

func TestBlockGroup_SetPostProcess(t *testing.T) {
	group := NewBlockGroup(uuid.New(), uuid.New(), "Test", BlockGroupTypeForeach)
	code := "return { result: output.data };"

	group.SetPostProcess(code)

	if group.PostProcess == nil || *group.PostProcess != code {
		t.Errorf("SetPostProcess() PostProcess = %v, want %v", group.PostProcess, code)
	}
}

func TestBlockGroup_HasParent(t *testing.T) {
	tests := []struct {
		name          string
		parentGroupID *uuid.UUID
		want          bool
	}{
		{"with parent", func() *uuid.UUID { id := uuid.New(); return &id }(), true},
		{"no parent", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			group := &BlockGroup{ParentGroupID: tt.parentGroupID}
			if got := group.HasParent(); got != tt.want {
				t.Errorf("HasParent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlockGroup_IsLoop(t *testing.T) {
	tests := []struct {
		groupType BlockGroupType
		want      bool
	}{
		{BlockGroupTypeForeach, true},
		{BlockGroupTypeWhile, true},
		{BlockGroupTypeParallel, false},
		{BlockGroupTypeTryCatch, false},
		{BlockGroupTypeAgent, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.groupType), func(t *testing.T) {
			group := &BlockGroup{Type: tt.groupType}
			if got := group.IsLoop(); got != tt.want {
				t.Errorf("IsLoop() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlockGroup_IsErrorHandler(t *testing.T) {
	tests := []struct {
		groupType BlockGroupType
		want      bool
	}{
		{BlockGroupTypeTryCatch, true},
		{BlockGroupTypeParallel, false},
		{BlockGroupTypeForeach, false},
		{BlockGroupTypeWhile, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.groupType), func(t *testing.T) {
			group := &BlockGroup{Type: tt.groupType}
			if got := group.IsErrorHandler(); got != tt.want {
				t.Errorf("IsErrorHandler() = %v, want %v", got, tt.want)
			}
		})
	}
}
