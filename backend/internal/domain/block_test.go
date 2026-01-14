package domain

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewBlockDefinition(t *testing.T) {
	tenantID := uuid.New()

	block := NewBlockDefinition(&tenantID, "my-block", "My Block", BlockCategoryAI)

	assert.NotEqual(t, uuid.Nil, block.ID)
	assert.Equal(t, &tenantID, block.TenantID)
	assert.Equal(t, "my-block", block.Slug)
	assert.Equal(t, "My Block", block.Name)
	assert.Equal(t, BlockCategoryAI, block.Category)
	assert.True(t, block.Enabled)
	assert.False(t, block.CreatedAt.IsZero())
	assert.False(t, block.UpdatedAt.IsZero())
}

func TestBlockDefinition_CanBeInherited(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name:     "block with code can be inherited",
			code:     "return { result: input.value * 2 }",
			expected: true,
		},
		{
			name:     "block without code cannot be inherited",
			code:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block := &BlockDefinition{Code: tt.code}
			assert.Equal(t, tt.expected, block.CanBeInherited())
		})
	}
}

func TestBlockDefinition_HasInheritance(t *testing.T) {
	tests := []struct {
		name          string
		parentBlockID *uuid.UUID
		expected      bool
	}{
		{
			name:          "block with parent has inheritance",
			parentBlockID: func() *uuid.UUID { id := uuid.New(); return &id }(),
			expected:      true,
		},
		{
			name:          "block without parent has no inheritance",
			parentBlockID: nil,
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block := &BlockDefinition{ParentBlockID: tt.parentBlockID}
			assert.Equal(t, tt.expected, block.HasInheritance())
		})
	}
}

func TestBlockDefinition_HasInternalSteps(t *testing.T) {
	tests := []struct {
		name          string
		internalSteps []InternalStep
		expected      bool
	}{
		{
			name: "block with internal steps",
			internalSteps: []InternalStep{
				{Type: "http", Config: json.RawMessage(`{}`), OutputKey: "step1"},
			},
			expected: true,
		},
		{
			name:          "block without internal steps",
			internalSteps: nil,
			expected:      false,
		},
		{
			name:          "block with empty internal steps",
			internalSteps: []InternalStep{},
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block := &BlockDefinition{InternalSteps: tt.internalSteps}
			assert.Equal(t, tt.expected, block.HasInternalSteps())
		})
	}
}

func TestBlockDefinition_GetEffectiveCode(t *testing.T) {
	tests := []struct {
		name         string
		code         string
		resolvedCode string
		expected     string
	}{
		{
			name:         "returns own code when no resolved code",
			code:         "return { own: true }",
			resolvedCode: "",
			expected:     "return { own: true }",
		},
		{
			name:         "returns resolved code when available",
			code:         "",
			resolvedCode: "return { resolved: true }",
			expected:     "return { resolved: true }",
		},
		{
			name:         "resolved code takes precedence",
			code:         "return { own: true }",
			resolvedCode: "return { resolved: true }",
			expected:     "return { resolved: true }",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block := &BlockDefinition{
				Code:         tt.code,
				ResolvedCode: tt.resolvedCode,
			}
			assert.Equal(t, tt.expected, block.GetEffectiveCode())
		})
	}
}

func TestBlockDefinition_GetEffectiveConfigDefaults(t *testing.T) {
	t.Run("returns own config defaults when no resolved", func(t *testing.T) {
		ownDefaults := json.RawMessage(`{"own": true}`)
		block := &BlockDefinition{
			ConfigDefaults: ownDefaults,
		}
		assert.Equal(t, ownDefaults, block.GetEffectiveConfigDefaults())
	})

	t.Run("returns resolved config defaults when available", func(t *testing.T) {
		ownDefaults := json.RawMessage(`{"own": true}`)
		resolvedDefaults := json.RawMessage(`{"resolved": true}`)
		block := &BlockDefinition{
			ConfigDefaults:         ownDefaults,
			ResolvedConfigDefaults: resolvedDefaults,
		}
		assert.Equal(t, resolvedDefaults, block.GetEffectiveConfigDefaults())
	})
}

func TestBlockDefinition_IsSystemBlock(t *testing.T) {
	tests := []struct {
		name     string
		tenantID *uuid.UUID
		isSystem bool
		expected bool
	}{
		{
			name:     "system block with is_system flag",
			tenantID: nil,
			isSystem: true,
			expected: true,
		},
		{
			name:     "system block without tenant",
			tenantID: nil,
			isSystem: false,
			expected: true,
		},
		{
			name:     "tenant block is not system",
			tenantID: func() *uuid.UUID { id := uuid.New(); return &id }(),
			isSystem: false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block := &BlockDefinition{
				TenantID: tt.tenantID,
				IsSystem: tt.isSystem,
			}
			assert.Equal(t, tt.expected, block.IsSystemBlock())
		})
	}
}

func TestBlockCategory_IsValid(t *testing.T) {
	tests := []struct {
		category BlockCategory
		expected bool
	}{
		{BlockCategoryAI, true},
		{BlockCategoryLogic, true},
		{BlockCategoryIntegration, true},
		{BlockCategoryData, true},
		{BlockCategoryControl, true},
		{BlockCategoryUtility, true},
		{BlockCategory("invalid"), false},
		{BlockCategory(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.category), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.category.IsValid())
		})
	}
}

func TestInternalStep_Structure(t *testing.T) {
	step := InternalStep{
		Type:      "http-call",
		Config:    json.RawMessage(`{"url": "https://api.example.com"}`),
		OutputKey: "api_response",
	}

	assert.Equal(t, "http-call", step.Type)
	assert.Equal(t, "api_response", step.OutputKey)
	assert.Contains(t, string(step.Config), "api.example.com")
}
