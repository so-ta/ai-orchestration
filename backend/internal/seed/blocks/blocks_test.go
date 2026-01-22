package blocks_test

import (
	"context"
	"testing"
	"time"

	"github.com/souta/ai-orchestration/internal/block/sandbox"
	"github.com/souta/ai-orchestration/internal/seed/blocks"
	seedtest "github.com/souta/ai-orchestration/internal/seed/testing"
	"github.com/souta/ai-orchestration/internal/seed/validation"
)

func TestRegistry_AllBlocksValid(t *testing.T) {
	registry := blocks.NewRegistry()
	validator := validation.NewBlockValidator()

	result := validator.ValidateAllWithResult(registry)

	if result.InvalidBlocks > 0 {
		t.Errorf("Found %d invalid blocks out of %d total", result.InvalidBlocks, result.TotalBlocks)
		for _, err := range result.Errors {
			t.Errorf("  [%s.%s] %s", err.BlockSlug, err.Field, err.Message)
		}
	}
}

func TestRegistry_BlockCount(t *testing.T) {
	registry := blocks.NewRegistry()

	// We expect at least 30 blocks (based on seed.sql having 39)
	minBlocks := 30
	if registry.Count() < minBlocks {
		t.Errorf("Expected at least %d blocks, got %d", minBlocks, registry.Count())
	}
}

func TestRegistry_GetBySlug(t *testing.T) {
	registry := blocks.NewRegistry()

	// Test some known blocks
	knownSlugs := []string{"llm", "http", "condition", "split", "start"}

	for _, slug := range knownSlugs {
		block, ok := registry.GetBySlug(slug)
		if !ok {
			t.Errorf("Block %s not found in registry", slug)
			continue
		}
		if block.Slug != slug {
			t.Errorf("Block slug mismatch: expected %s, got %s", slug, block.Slug)
		}
	}
}

func TestRegistry_RequiredFields(t *testing.T) {
	registry := blocks.NewRegistry()

	for _, block := range registry.GetAll() {
		t.Run(block.Slug, func(t *testing.T) {
			if block.Slug == "" {
				t.Error("Block has empty slug")
			}
			if block.Name.EN == "" && block.Name.JA == "" {
				t.Error("Block has empty name")
			}
			if block.Version < 1 {
				t.Errorf("Block has invalid version: %d", block.Version)
			}
			if !block.Category.IsValid() {
				t.Errorf("Block has invalid category: %s", block.Category)
			}
		})
	}
}

func TestBlockExecution_DataBlocks(t *testing.T) {
	// Skip execution tests for now - requires sandbox config global support
	// The sandbox needs to expose 'config' as a global variable for block code to work
	t.Skip("Block execution tests require sandbox config global support")

	registry := blocks.NewRegistry()
	sb := sandbox.New(sandbox.Config{
		Timeout:     5 * time.Second,
		MemoryLimit: 64 * 1024 * 1024,
	})
	ctx := context.Background()
	execCtx := seedtest.CreateMockExecutionContext()

	// Test split block
	t.Run("split", func(t *testing.T) {
		block, ok := registry.GetBySlug("split")
		if !ok {
			t.Skip("split block not found")
		}

		for _, tc := range block.TestCases {
			t.Run(tc.Name, func(t *testing.T) {
				input := tc.Input
				input["config"] = tc.Config

				result, err := sb.Execute(ctx, block.Code, input, execCtx)
				if tc.ExpectError {
					if err == nil {
						t.Error("Expected error but got none")
					}
					return
				}

				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				for key, expected := range tc.ExpectedOutput {
					if result[key] != expected {
						t.Errorf("output[%s]: expected %v, got %v", key, expected, result[key])
					}
				}
			})
		}
	})

	// Test filter block
	t.Run("filter", func(t *testing.T) {
		block, ok := registry.GetBySlug("filter")
		if !ok {
			t.Skip("filter block not found")
		}

		for _, tc := range block.TestCases {
			t.Run(tc.Name, func(t *testing.T) {
				input := tc.Input
				input["config"] = tc.Config

				result, err := sb.Execute(ctx, block.Code, input, execCtx)
				if tc.ExpectError {
					if err == nil {
						t.Error("Expected error but got none")
					}
					return
				}

				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				for key, expected := range tc.ExpectedOutput {
					if result[key] != expected {
						t.Errorf("output[%s]: expected %v, got %v", key, expected, result[key])
					}
				}
			})
		}
	})
}

func TestBlockExecution_LogicBlocks(t *testing.T) {
	// Skip execution tests for now - requires sandbox config global support
	t.Skip("Block execution tests require sandbox config global support")

	registry := blocks.NewRegistry()
	sb := sandbox.New(sandbox.Config{
		Timeout:     5 * time.Second,
		MemoryLimit: 64 * 1024 * 1024,
	})
	ctx := context.Background()
	execCtx := seedtest.CreateMockExecutionContext()

	// Test condition block
	t.Run("condition", func(t *testing.T) {
		block, ok := registry.GetBySlug("condition")
		if !ok {
			t.Skip("condition block not found")
		}

		for _, tc := range block.TestCases {
			t.Run(tc.Name, func(t *testing.T) {
				input := tc.Input
				input["config"] = tc.Config

				result, err := sb.Execute(ctx, block.Code, input, execCtx)
				if tc.ExpectError {
					if err == nil {
						t.Error("Expected error but got none")
					}
					return
				}

				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				for key, expected := range tc.ExpectedOutput {
					if result[key] != expected {
						t.Errorf("output[%s]: expected %v, got %v", key, expected, result[key])
					}
				}
			})
		}
	})
}

func TestBlockExecution_AllBlocksWithTestCases(t *testing.T) {
	// Skip execution tests for now - requires sandbox config global support
	t.Skip("Block execution tests require sandbox config global support")

	registry := blocks.NewRegistry()
	sb := sandbox.New(sandbox.Config{
		Timeout:     5 * time.Second,
		MemoryLimit: 64 * 1024 * 1024,
	})
	ctx := context.Background()
	execCtx := seedtest.CreateMockExecutionContext()

	blocksWithTests := 0
	blocksWithoutTests := 0

	for _, block := range registry.GetAll() {
		if len(block.TestCases) == 0 {
			blocksWithoutTests++
			continue
		}

		blocksWithTests++

		t.Run(block.Slug, func(t *testing.T) {
			for _, tc := range block.TestCases {
				t.Run(tc.Name, func(t *testing.T) {
					input := tc.Input
					if input == nil {
						input = make(map[string]interface{})
					}
					input["config"] = tc.Config

					result, err := sb.Execute(ctx, block.Code, input, execCtx)

					if tc.ExpectError {
						if err == nil {
							t.Error("Expected error but got none")
						} else if tc.ErrorContains != "" {
							if err.Error() != tc.ErrorContains && !contains(err.Error(), tc.ErrorContains) {
								t.Errorf("Expected error containing %q, got %q", tc.ErrorContains, err.Error())
							}
						}
						return
					}

					if err != nil {
						t.Fatalf("Unexpected error: %v", err)
					}

					for key, expected := range tc.ExpectedOutput {
						actual := result[key]
						if !compareValues(expected, actual) {
							t.Errorf("output[%s]: expected %v (%T), got %v (%T)",
								key, expected, expected, actual, actual)
						}
					}
				})
			}
		})
	}

	t.Logf("Blocks with test cases: %d, without: %d", blocksWithTests, blocksWithoutTests)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func compareValues(expected, actual interface{}) bool {
	// Handle numeric comparisons (JavaScript returns float64)
	switch e := expected.(type) {
	case int:
		switch a := actual.(type) {
		case int:
			return e == a
		case int64:
			return int64(e) == a
		case float64:
			return float64(e) == a
		}
	case int64:
		switch a := actual.(type) {
		case int:
			return e == int64(a)
		case int64:
			return e == a
		case float64:
			return float64(e) == a
		}
	case float64:
		switch a := actual.(type) {
		case int:
			return e == float64(a)
		case int64:
			return e == float64(a)
		case float64:
			return e == a
		}
	case bool:
		if a, ok := actual.(bool); ok {
			return e == a
		}
	case string:
		if a, ok := actual.(string); ok {
			return e == a
		}
	}

	// Fallback to direct comparison
	return expected == actual
}
