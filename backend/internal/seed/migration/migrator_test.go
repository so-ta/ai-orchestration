package migration

import (
	"encoding/json"
	"testing"

	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/seed/blocks"
)

func TestHasChanges(t *testing.T) {
	migrator := &Migrator{}

	tests := []struct {
		name     string
		existing *domain.BlockDefinition
		seed     *blocks.SystemBlockDefinition
		want     bool
	}{
		{
			name: "no changes",
			existing: &domain.BlockDefinition{
				Version:     1,
				Name:        "Test Block",
				Description: "A test block",
				Category:    domain.BlockCategoryFlow,
				Icon:        "test",
				Code:        "return {};",
				Enabled:     true,
			},
			seed: &blocks.SystemBlockDefinition{
				Version:     1,
				Name:        "Test Block",
				Description: "A test block",
				Category:    domain.BlockCategoryFlow,
				Icon:        "test",
				Code:        "return {};",
				Enabled:     true,
			},
			want: false,
		},
		{
			name: "version changed",
			existing: &domain.BlockDefinition{
				Version: 1,
				Name:    "Test Block",
			},
			seed: &blocks.SystemBlockDefinition{
				Version: 2,
				Name:    "Test Block",
			},
			want: true,
		},
		{
			name: "name changed",
			existing: &domain.BlockDefinition{
				Version: 1,
				Name:    "Old Name",
			},
			seed: &blocks.SystemBlockDefinition{
				Version: 1,
				Name:    "New Name",
			},
			want: true,
		},
		{
			name: "code changed",
			existing: &domain.BlockDefinition{
				Version: 1,
				Code:    "return { old: true };",
			},
			seed: &blocks.SystemBlockDefinition{
				Version: 1,
				Code:    "return { new: true };",
			},
			want: true,
		},
		{
			name: "config schema changed",
			existing: &domain.BlockDefinition{
				Version:      1,
				ConfigSchema: json.RawMessage(`{"type": "object"}`),
			},
			seed: &blocks.SystemBlockDefinition{
				Version:      1,
				ConfigSchema: json.RawMessage(`{"type": "object", "properties": {}}`),
			},
			want: true,
		},
		{
			name: "subcategory changed",
			existing: &domain.BlockDefinition{
				Version:     1,
				Name:        "Test Block",
				Category:    domain.BlockCategoryAI,
				Subcategory: domain.BlockSubcategoryChat,
			},
			seed: &blocks.SystemBlockDefinition{
				Version:     1,
				Name:        "Test Block",
				Category:    domain.BlockCategoryAI,
				Subcategory: domain.BlockSubcategoryRAG,
			},
			want: true,
		},
		{
			name: "subcategory added",
			existing: &domain.BlockDefinition{
				Version:  1,
				Name:     "Test Block",
				Category: domain.BlockCategoryAI,
				// No subcategory
			},
			seed: &blocks.SystemBlockDefinition{
				Version:     1,
				Name:        "Test Block",
				Category:    domain.BlockCategoryAI,
				Subcategory: domain.BlockSubcategoryChat,
			},
			want: true,
		},
		{
			name: "subcategory same",
			existing: &domain.BlockDefinition{
				Version:     1,
				Name:        "Test Block",
				Category:    domain.BlockCategoryAI,
				Subcategory: domain.BlockSubcategoryChat,
			},
			seed: &blocks.SystemBlockDefinition{
				Version:     1,
				Name:        "Test Block",
				Category:    domain.BlockCategoryAI,
				Subcategory: domain.BlockSubcategoryChat,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := migrator.hasChanges(tt.existing, tt.seed); got != tt.want {
				t.Errorf("hasChanges() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJSONEqual(t *testing.T) {
	tests := []struct {
		name string
		a    json.RawMessage
		b    json.RawMessage
		want bool
	}{
		{
			name: "both empty",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "one empty",
			a:    json.RawMessage(`{}`),
			b:    nil,
			want: false,
		},
		{
			name: "equal objects",
			a:    json.RawMessage(`{"a": 1, "b": 2}`),
			b:    json.RawMessage(`{"b": 2, "a": 1}`),
			want: true,
		},
		{
			name: "different objects",
			a:    json.RawMessage(`{"a": 1}`),
			b:    json.RawMessage(`{"a": 2}`),
			want: false,
		},
		{
			name: "whitespace difference",
			a:    json.RawMessage(`{"a":1}`),
			b:    json.RawMessage(`{ "a" : 1 }`),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := jsonEqual(tt.a, tt.b); got != tt.want {
				t.Errorf("jsonEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPortsEqual(t *testing.T) {
	tests := []struct {
		name string
		a    []domain.InputPort
		b    []domain.InputPort
		want bool
	}{
		{
			name: "both empty",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "equal ports",
			a: []domain.InputPort{
				{Name: "input", Label: "Input", Required: true},
			},
			b: []domain.InputPort{
				{Name: "input", Label: "Input", Required: true},
			},
			want: true,
		},
		{
			name: "different length",
			a: []domain.InputPort{
				{Name: "input", Label: "Input"},
			},
			b:    nil,
			want: false,
		},
		{
			name: "different name",
			a: []domain.InputPort{
				{Name: "input1", Label: "Input"},
			},
			b: []domain.InputPort{
				{Name: "input2", Label: "Input"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := portsEqual(tt.a, tt.b); got != tt.want {
				t.Errorf("portsEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDescribeChanges(t *testing.T) {
	migrator := &Migrator{}

	existing := &domain.BlockDefinition{
		Version: 1,
		Name:    "Test",
		Code:    "old code",
	}

	seed := &blocks.SystemBlockDefinition{
		Version: 2,
		Name:    "Test Updated",
		Code:    "new code",
	}

	result := migrator.describeChanges(existing, seed)

	// Should contain version, name, and code
	if result == "" {
		t.Error("Expected non-empty description")
	}
	if !contains(result, "version") {
		t.Error("Expected 'version' in description")
	}
	if !contains(result, "name") {
		t.Error("Expected 'name' in description")
	}
	if !contains(result, "code") {
		t.Error("Expected 'code' in description")
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestTopologicalSort(t *testing.T) {
	tests := []struct {
		name        string
		blocks      []*blocks.SystemBlockDefinition
		wantOrder   []string // expected order of slugs
		wantErr     bool
	}{
		{
			name: "no dependencies",
			blocks: []*blocks.SystemBlockDefinition{
				{Slug: "a"},
				{Slug: "b"},
				{Slug: "c"},
			},
			wantOrder: nil, // order doesn't matter, just check all are present
			wantErr:   false,
		},
		{
			name: "simple chain",
			blocks: []*blocks.SystemBlockDefinition{
				{Slug: "child", ParentBlockSlug: "parent"},
				{Slug: "parent"},
			},
			wantOrder: []string{"parent", "child"},
			wantErr:   false,
		},
		{
			name: "multi-level inheritance",
			blocks: []*blocks.SystemBlockDefinition{
				{Slug: "github_create_issue", ParentBlockSlug: "github-api"},
				{Slug: "github-api", ParentBlockSlug: "bearer-api"},
				{Slug: "bearer-api", ParentBlockSlug: "rest-api"},
				{Slug: "rest-api", ParentBlockSlug: "http"},
				{Slug: "http"},
			},
			wantOrder: []string{"http", "rest-api", "bearer-api", "github-api", "github_create_issue"},
			wantErr:   false,
		},
		{
			name: "multiple inheritance branches",
			blocks: []*blocks.SystemBlockDefinition{
				{Slug: "http"},
				{Slug: "webhook", ParentBlockSlug: "http"},
				{Slug: "slack", ParentBlockSlug: "webhook"},
				{Slug: "discord", ParentBlockSlug: "webhook"},
				{Slug: "rest-api", ParentBlockSlug: "http"},
				{Slug: "bearer-api", ParentBlockSlug: "rest-api"},
			},
			wantOrder: nil, // Just verify no error and correct relative ordering
			wantErr:   false,
		},
		{
			name: "circular dependency",
			blocks: []*blocks.SystemBlockDefinition{
				{Slug: "a", ParentBlockSlug: "c"},
				{Slug: "b", ParentBlockSlug: "a"},
				{Slug: "c", ParentBlockSlug: "b"},
			},
			wantErr: true,
		},
		{
			name: "self-reference",
			blocks: []*blocks.SystemBlockDefinition{
				{Slug: "a", ParentBlockSlug: "a"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := topologicalSort(tt.blocks)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Check all blocks are present
			if len(result) != len(tt.blocks) {
				t.Errorf("expected %d blocks, got %d", len(tt.blocks), len(result))
				return
			}

			// Check specific order if provided
			if tt.wantOrder != nil {
				for i, slug := range tt.wantOrder {
					if result[i].Slug != slug {
						t.Errorf("position %d: expected %s, got %s", i, slug, result[i].Slug)
					}
				}
			} else {
				// For cases without specific order, verify parent comes before child
				slugIndex := make(map[string]int)
				for i, block := range result {
					slugIndex[block.Slug] = i
				}

				for _, block := range tt.blocks {
					if block.ParentBlockSlug != "" {
						parentIdx, parentExists := slugIndex[block.ParentBlockSlug]
						childIdx := slugIndex[block.Slug]
						if parentExists && parentIdx >= childIdx {
							t.Errorf("parent %s (index %d) should come before child %s (index %d)",
								block.ParentBlockSlug, parentIdx, block.Slug, childIdx)
						}
					}
				}
			}
		})
	}
}

func TestTopologicalSort_MissingParent(t *testing.T) {
	// Test case where parent is referenced but not in the list
	// This should still work - the child just waits for a parent that never comes
	// which means the child will never be processed (cycle detection catches this)
	blocks := []*blocks.SystemBlockDefinition{
		{Slug: "child", ParentBlockSlug: "missing_parent"},
	}

	_, err := topologicalSort(blocks)
	// This should detect as circular dependency since child can never be processed
	if err == nil {
		t.Errorf("expected error for missing parent, but got none")
	}
}
