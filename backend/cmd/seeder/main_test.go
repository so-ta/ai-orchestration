package main

import (
	"testing"

	"github.com/souta/ai-orchestration/internal/seed/blocks"
	"github.com/souta/ai-orchestration/internal/seed/validation"
)

func TestSeeder_RegistryLoading(t *testing.T) {
	// Test that registry can be created and contains blocks
	registry := blocks.NewRegistry()

	if registry.Count() == 0 {
		t.Error("Expected registry to contain blocks, got 0")
	}

	// Expect at least 30 blocks
	if registry.Count() < 30 {
		t.Errorf("Expected at least 30 blocks, got %d", registry.Count())
	}
}

func TestSeeder_ValidationPasses(t *testing.T) {
	// Test that all blocks pass validation
	registry := blocks.NewRegistry()
	validator := validation.NewBlockValidator()

	result := validator.ValidateAllWithResult(registry)

	if result.InvalidBlocks > 0 {
		t.Errorf("Expected 0 invalid blocks, got %d", result.InvalidBlocks)
		for _, err := range result.Errors {
			t.Errorf("  [%s.%s] %s", err.BlockSlug, err.Field, err.Message)
		}
	}
}

func TestSeeder_GetBySlugExists(t *testing.T) {
	registry := blocks.NewRegistry()

	// Test that key blocks exist
	testSlugs := []string{"llm", "http", "condition", "split", "start"}

	for _, slug := range testSlugs {
		block, ok := registry.GetBySlug(slug)
		if !ok {
			t.Errorf("Expected block '%s' to exist in registry", slug)
			continue
		}
		if block.Slug != slug {
			t.Errorf("Block slug mismatch: expected %s, got %s", slug, block.Slug)
		}
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{
			name:         "returns default when env not set",
			key:          "TEST_SEEDER_NONEXISTENT_KEY",
			defaultValue: "default_value",
			envValue:     "",
			expected:     "default_value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getEnv(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnv(%s, %s) = %s, expected %s", tt.key, tt.defaultValue, result, tt.expected)
			}
		})
	}
}
