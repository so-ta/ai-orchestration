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
