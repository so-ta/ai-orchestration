package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestBlockCategory_IsValid(t *testing.T) {
	validCategories := []BlockCategory{
		BlockCategoryAI,
		BlockCategoryFlow,
		BlockCategoryApps,
		BlockCategoryCustom,
	}

	for _, cat := range validCategories {
		t.Run(string(cat), func(t *testing.T) {
			if !cat.IsValid() {
				t.Errorf("IsValid() = false for valid category %v", cat)
			}
		})
	}

	invalidCategories := []BlockCategory{
		BlockCategory("invalid"),
		BlockCategory(""),
	}

	for _, cat := range invalidCategories {
		t.Run(string(cat), func(t *testing.T) {
			if cat.IsValid() {
				t.Errorf("IsValid() = true for invalid category %v", cat)
			}
		})
	}
}

func TestBlockGroupKind_IsValid(t *testing.T) {
	tests := []struct {
		kind BlockGroupKind
		want bool
	}{
		{BlockGroupKindNone, true},
		{BlockGroupKindParallel, true},
		{BlockGroupKindTryCatch, true},
		{BlockGroupKindForeach, true},
		{BlockGroupKindWhile, true},
		{BlockGroupKindAgent, true},
		{BlockGroupKind("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			if got := tt.kind.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewBlockDefinition(t *testing.T) {
	tenantID := uuid.New()
	slug := "test-block"
	name := "Test Block"
	category := BlockCategoryAI

	block := NewBlockDefinition(&tenantID, slug, name, category)

	if block.ID == uuid.Nil {
		t.Error("NewBlockDefinition() should generate a non-nil UUID")
	}
	if block.TenantID == nil || *block.TenantID != tenantID {
		t.Error("NewBlockDefinition() TenantID mismatch")
	}
	if block.Slug != slug {
		t.Errorf("NewBlockDefinition() Slug = %v, want %v", block.Slug, slug)
	}
	if block.Name != name {
		t.Errorf("NewBlockDefinition() Name = %v, want %v", block.Name, name)
	}
	if block.Category != category {
		t.Errorf("NewBlockDefinition() Category = %v, want %v", block.Category, category)
	}
	if !block.Enabled {
		t.Error("NewBlockDefinition() should be enabled by default")
	}
	if len(block.OutputPorts) != 1 || block.OutputPorts[0].Name != "output" {
		t.Error("NewBlockDefinition() should have default output port")
	}
}

func TestNewBlockDefinition_SystemBlock(t *testing.T) {
	block := NewBlockDefinition(nil, "system-block", "System Block", BlockCategoryFlow)

	if block.TenantID != nil {
		t.Error("NewBlockDefinition() with nil tenantID should have nil TenantID")
	}
	if !block.IsSystemBlock() {
		t.Error("IsSystemBlock() should return true for nil tenantID")
	}
}

func TestBlockDefinition_IsSystemBlock(t *testing.T) {
	tenantID := uuid.New()
	tests := []struct {
		name     string
		tenantID *uuid.UUID
		want     bool
	}{
		{"system block", nil, true},
		{"tenant block", &tenantID, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block := &BlockDefinition{TenantID: tt.tenantID}
			if got := block.IsSystemBlock(); got != tt.want {
				t.Errorf("IsSystemBlock() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlockDefinition_CanBeInherited(t *testing.T) {
	tests := []struct {
		name string
		code string
		want bool
	}{
		{"with code", "console.log('test')", true},
		{"without code", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block := &BlockDefinition{Code: tt.code}
			if got := block.CanBeInherited(); got != tt.want {
				t.Errorf("CanBeInherited() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlockDefinition_HasInheritance(t *testing.T) {
	parentID := uuid.New()
	tests := []struct {
		name          string
		parentBlockID *uuid.UUID
		want          bool
	}{
		{"with parent", &parentID, true},
		{"without parent", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block := &BlockDefinition{ParentBlockID: tt.parentBlockID}
			if got := block.HasInheritance(); got != tt.want {
				t.Errorf("HasInheritance() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlockDefinition_HasInternalSteps(t *testing.T) {
	tests := []struct {
		name          string
		internalSteps []InternalStep
		want          bool
	}{
		{"with steps", []InternalStep{{Type: "llm"}}, true},
		{"without steps", nil, false},
		{"empty steps", []InternalStep{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block := &BlockDefinition{InternalSteps: tt.internalSteps}
			if got := block.HasInternalSteps(); got != tt.want {
				t.Errorf("HasInternalSteps() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlockDefinition_IsGroupBlock(t *testing.T) {
	tests := []struct {
		groupKind BlockGroupKind
		want      bool
	}{
		{BlockGroupKindNone, false},
		{BlockGroupKindParallel, true},
		{BlockGroupKindTryCatch, true},
		{BlockGroupKindForeach, true},
		{BlockGroupKindWhile, true},
		{BlockGroupKindAgent, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.groupKind), func(t *testing.T) {
			block := &BlockDefinition{GroupKind: tt.groupKind}
			if got := block.IsGroupBlock(); got != tt.want {
				t.Errorf("IsGroupBlock() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlockDefinition_GetEffectiveCode(t *testing.T) {
	tests := []struct {
		name         string
		code         string
		resolvedCode string
		want         string
	}{
		{"resolved code", "original", "resolved", "resolved"},
		{"own code", "original", "", "original"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block := &BlockDefinition{Code: tt.code, ResolvedCode: tt.resolvedCode}
			if got := block.GetEffectiveCode(); got != tt.want {
				t.Errorf("GetEffectiveCode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlockDefinition_GetEffectiveConfigDefaults(t *testing.T) {
	ownDefaults := json.RawMessage(`{"key": "own"}`)
	resolvedDefaults := json.RawMessage(`{"key": "resolved"}`)

	tests := []struct {
		name                   string
		configDefaults         json.RawMessage
		resolvedConfigDefaults json.RawMessage
		wantKey                string
	}{
		{"resolved defaults", ownDefaults, resolvedDefaults, "resolved"},
		{"own defaults", ownDefaults, nil, "own"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block := &BlockDefinition{
				ConfigDefaults:         tt.configDefaults,
				ResolvedConfigDefaults: tt.resolvedConfigDefaults,
			}
			got := block.GetEffectiveConfigDefaults()
			var result map[string]string
			json.Unmarshal(got, &result)
			if result["key"] != tt.wantKey {
				t.Errorf("GetEffectiveConfigDefaults() key = %v, want %v", result["key"], tt.wantKey)
			}
		})
	}
}

func TestNewBlockError(t *testing.T) {
	err := NewBlockError("LLM_001", "Rate limit exceeded", true)

	if err.Code != "LLM_001" {
		t.Errorf("NewBlockError() Code = %v, want LLM_001", err.Code)
	}
	if err.Message != "Rate limit exceeded" {
		t.Errorf("NewBlockError() Message = %v, want Rate limit exceeded", err.Message)
	}
	if !err.Retryable {
		t.Error("NewBlockError() Retryable should be true")
	}
}

func TestBlockError_Error(t *testing.T) {
	err := NewBlockError("LLM_001", "Rate limit exceeded", true)
	expected := "[LLM_001] Rate limit exceeded"

	if err.Error() != expected {
		t.Errorf("Error() = %v, want %v", err.Error(), expected)
	}
}

func TestBlockError_WithDetails(t *testing.T) {
	err := NewBlockError("LLM_001", "Error", false)
	details := map[string]string{"retry_after": "60s"}

	err.WithDetails(details)

	if err.Details == nil {
		t.Error("WithDetails() should set Details")
	}
}

func TestBlockError_WithRetryAfter(t *testing.T) {
	err := NewBlockError("LLM_001", "Error", true)
	duration := 30 * time.Second

	err.WithRetryAfter(duration)

	if err.RetryAfter == nil || *err.RetryAfter != duration {
		t.Error("WithRetryAfter() should set RetryAfter")
	}
}

func TestNewBlockVersion(t *testing.T) {
	block := NewBlockDefinition(nil, "test", "Test", BlockCategoryAI)
	block.Version = 5
	block.Code = "test code"
	changedBy := uuid.New()

	version := NewBlockVersion(block, "Initial version", &changedBy)

	if version.ID == uuid.Nil {
		t.Error("NewBlockVersion() should generate a non-nil UUID")
	}
	if version.BlockID != block.ID {
		t.Error("NewBlockVersion() BlockID mismatch")
	}
	if version.Version != 5 {
		t.Errorf("NewBlockVersion() Version = %v, want 5", version.Version)
	}
	if version.Code != "test code" {
		t.Errorf("NewBlockVersion() Code mismatch")
	}
	if version.ChangeSummary != "Initial version" {
		t.Error("NewBlockVersion() ChangeSummary mismatch")
	}
	if version.ChangedBy == nil || *version.ChangedBy != changedBy {
		t.Error("NewBlockVersion() ChangedBy mismatch")
	}
}

func TestValidBlockCategories(t *testing.T) {
	categories := ValidBlockCategories()

	if len(categories) != 4 {
		t.Errorf("ValidBlockCategories() returned %d categories, want 4", len(categories))
	}

	expected := map[BlockCategory]bool{
		BlockCategoryAI:     true,
		BlockCategoryFlow:   true,
		BlockCategoryApps:   true,
		BlockCategoryCustom: true,
	}

	for _, cat := range categories {
		if !expected[cat] {
			t.Errorf("Unexpected category: %v", cat)
		}
	}
}

func TestValidBlockGroupKinds(t *testing.T) {
	kinds := ValidBlockGroupKinds()

	if len(kinds) != 5 {
		t.Errorf("ValidBlockGroupKinds() returned %d kinds, want 5", len(kinds))
	}

	expected := map[BlockGroupKind]bool{
		BlockGroupKindParallel: true,
		BlockGroupKindTryCatch: true,
		BlockGroupKindForeach:  true,
		BlockGroupKindWhile:    true,
		BlockGroupKindAgent:    true,
	}

	for _, kind := range kinds {
		if !expected[kind] {
			t.Errorf("Unexpected group kind: %v", kind)
		}
	}
}
