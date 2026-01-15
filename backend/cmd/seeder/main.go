package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/repository/postgres"
	"github.com/souta/ai-orchestration/internal/seed/blocks"
	"github.com/souta/ai-orchestration/internal/seed/migration"
	"github.com/souta/ai-orchestration/internal/seed/validation"
	"github.com/souta/ai-orchestration/internal/seed/workflows"
	"github.com/souta/ai-orchestration/pkg/database"
)

// Default tenant ID for system workflows
const defaultTenantID = "00000000-0000-0000-0000-000000000001"

func main() {
	// Parse flags
	validateOnly := flag.Bool("validate", false, "Only validate blocks without migrating")
	dryRun := flag.Bool("dry-run", false, "Show what would be changed without applying")
	verbose := flag.Bool("verbose", false, "Show detailed output")
	blocksOnly := flag.Bool("blocks-only", false, "Only migrate blocks, skip workflows")
	workflowsOnly := flag.Bool("workflows-only", false, "Only migrate workflows, skip blocks")
	tenantIDStr := flag.String("tenant-id", defaultTenantID, "Tenant ID for workflow migration")
	flag.Parse()

	// Create registries
	blockRegistry := blocks.NewRegistry()
	workflowRegistry := workflows.NewRegistry()

	fmt.Printf("📦 Loaded %d block definitions\n", blockRegistry.Count())
	fmt.Printf("📦 Loaded %d workflow definitions\n", workflowRegistry.Count())

	// Validate all blocks
	validator := validation.NewBlockValidator()
	blockValidationResult := validator.ValidateAllWithResult(blockRegistry)

	fmt.Printf("\n🔍 Block Validation Results:\n")
	fmt.Printf("   Total blocks: %d\n", blockValidationResult.TotalBlocks)
	fmt.Printf("   Valid blocks: %d\n", blockValidationResult.ValidBlocks)
	fmt.Printf("   Invalid blocks: %d\n", blockValidationResult.InvalidBlocks)

	if len(blockValidationResult.Errors) > 0 {
		fmt.Printf("\n❌ Block Validation Errors:\n")
		for _, err := range blockValidationResult.Errors {
			fmt.Printf("   [%s.%s] %s\n", err.BlockSlug, err.Field, err.Message)
		}
		os.Exit(1)
	}

	fmt.Printf("\n✅ All blocks passed validation\n")

	// Validate all workflows
	workflowErrors := validateWorkflows(workflowRegistry)
	if len(workflowErrors) > 0 {
		fmt.Printf("\n❌ Workflow Validation Errors:\n")
		for _, err := range workflowErrors {
			fmt.Printf("   %v\n", err)
		}
		os.Exit(1)
	}

	fmt.Printf("✅ All workflows passed validation\n")

	if *validateOnly {
		os.Exit(0)
	}

	// Parse tenant ID
	tenantID, err := uuid.Parse(*tenantIDStr)
	if err != nil {
		fmt.Printf("❌ Invalid tenant ID: %v\n", err)
		os.Exit(1)
	}

	// Connect to database
	dbURL := getEnv("DATABASE_URL", "")
	if dbURL == "" {
		fmt.Println("❌ DATABASE_URL environment variable is required")
		os.Exit(1)
	}
	ctx := context.Background()

	pool, err := database.NewPool(ctx, database.DefaultConfig(dbURL))
	if err != nil {
		fmt.Printf("❌ Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Create repositories
	blockRepo := postgres.NewBlockDefinitionRepository(pool)
	versionRepo := postgres.NewBlockVersionRepository(pool)
	workflowRepo := postgres.NewWorkflowRepository(pool)
	stepRepo := postgres.NewStepRepository(pool)
	edgeRepo := postgres.NewEdgeRepository(pool)
	blockGroupRepo := postgres.NewBlockGroupRepository(pool)

	// Create migrators
	blockMigrator := migration.NewMigrator(blockRepo, versionRepo)
	workflowMigrator := migration.NewWorkflowMigrator(workflowRepo, stepRepo, edgeRepo).
		WithBlockRepo(blockRepo).
		WithBlockGroupRepo(blockGroupRepo)

	if *dryRun {
		// Dry run mode
		fmt.Printf("\n🔄 Dry Run - Checking what would be changed...\n\n")

		// Block dry run
		if !*workflowsOnly {
			blockChanges, err := blockMigrator.DryRun(ctx, blockRegistry)
			if err != nil {
				fmt.Printf("❌ Block dry run failed: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("=== BLOCKS ===\n")
			if len(blockChanges.ToCreate) > 0 {
				fmt.Printf("📥 Blocks to create (%d):\n", len(blockChanges.ToCreate))
				for _, slug := range blockChanges.ToCreate {
					block, ok := blockRegistry.GetBySlug(slug)
					if !ok {
						fmt.Printf("   + %s (unknown block)\n", slug)
						continue
					}
					fmt.Printf("   + %s (v%d) - %s\n", slug, block.Version, block.Name)
				}
			}

			if len(blockChanges.ToUpdate) > 0 {
				fmt.Printf("\n📝 Blocks to update (%d):\n", len(blockChanges.ToUpdate))
				for _, change := range blockChanges.ToUpdate {
					if *verbose {
						fmt.Printf("   ~ %s (v%d → v%d) - %s\n",
							change.Slug, change.OldVersion, change.NewVersion, change.Reason)
					} else {
						fmt.Printf("   ~ %s (v%d → v%d)\n",
							change.Slug, change.OldVersion, change.NewVersion)
					}
				}
			}

			if len(blockChanges.Unchanged) > 0 && *verbose {
				fmt.Printf("\n✓ Unchanged blocks (%d):\n", len(blockChanges.Unchanged))
				for _, slug := range blockChanges.Unchanged {
					fmt.Printf("   = %s\n", slug)
				}
			}

			fmt.Printf("\n📊 Block Summary: %d to create, %d to update, %d unchanged\n",
				len(blockChanges.ToCreate), len(blockChanges.ToUpdate), len(blockChanges.Unchanged))
		}

		// Workflow dry run
		if !*blocksOnly {
			workflowChanges, err := workflowMigrator.DryRun(ctx, workflowRegistry, tenantID)
			if err != nil {
				fmt.Printf("❌ Workflow dry run failed: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("\n=== WORKFLOWS ===\n")
			if len(workflowChanges.ToCreate) > 0 {
				fmt.Printf("📥 Workflows to create (%d):\n", len(workflowChanges.ToCreate))
				for _, slug := range workflowChanges.ToCreate {
					wf, ok := workflowRegistry.GetBySlug(slug)
					if !ok {
						fmt.Printf("   + %s (unknown workflow)\n", slug)
						continue
					}
					fmt.Printf("   + %s (v%d) - %s\n", slug, wf.Version, wf.Name)
				}
			}

			if len(workflowChanges.ToUpdate) > 0 {
				fmt.Printf("\n📝 Workflows to update (%d):\n", len(workflowChanges.ToUpdate))
				for _, change := range workflowChanges.ToUpdate {
					if *verbose {
						fmt.Printf("   ~ %s (v%d → v%d) - %s\n",
							change.SystemSlug, change.OldVersion, change.NewVersion, change.Reason)
					} else {
						fmt.Printf("   ~ %s (v%d → v%d)\n",
							change.SystemSlug, change.OldVersion, change.NewVersion)
					}
				}
			}

			if len(workflowChanges.Unchanged) > 0 && *verbose {
				fmt.Printf("\n✓ Unchanged workflows (%d):\n", len(workflowChanges.Unchanged))
				for _, slug := range workflowChanges.Unchanged {
					fmt.Printf("   = %s\n", slug)
				}
			}

			fmt.Printf("\n📊 Workflow Summary: %d to create, %d to update, %d unchanged\n",
				len(workflowChanges.ToCreate), len(workflowChanges.ToUpdate), len(workflowChanges.Unchanged))
		}

		os.Exit(0)
	}

	// Run migration
	fmt.Printf("\n🚀 Running migration...\n\n")

	// Block migration
	if !*workflowsOnly {
		fmt.Printf("=== BLOCKS ===\n")
		blockResult, err := blockMigrator.Migrate(ctx, blockRegistry)
		if err != nil {
			fmt.Printf("❌ Block migration failed: %v\n", err)
			os.Exit(1)
		}

		if *verbose {
			if len(blockResult.Created) > 0 {
				fmt.Printf("📥 Created blocks:\n")
				for _, slug := range blockResult.Created {
					fmt.Printf("   + %s\n", slug)
				}
			}
			if len(blockResult.Updated) > 0 {
				fmt.Printf("📝 Updated blocks:\n")
				for _, slug := range blockResult.Updated {
					fmt.Printf("   ~ %s\n", slug)
				}
			}
		}

		fmt.Printf("✅ Block migration completed!\n")
		fmt.Printf("   Created: %d, Updated: %d, Unchanged: %d\n",
			len(blockResult.Created), len(blockResult.Updated), len(blockResult.Unchanged))

		if len(blockResult.Errors) > 0 {
			fmt.Printf("\n⚠️  Block Warnings:\n")
			for _, err := range blockResult.Errors {
				fmt.Printf("   %v\n", err)
			}
		}
	}

	// Workflow migration
	if !*blocksOnly {
		fmt.Printf("\n=== WORKFLOWS ===\n")
		workflowResult, err := workflowMigrator.Migrate(ctx, workflowRegistry, tenantID)
		if err != nil {
			fmt.Printf("❌ Workflow migration failed: %v\n", err)
			os.Exit(1)
		}

		if *verbose {
			if len(workflowResult.Created) > 0 {
				fmt.Printf("📥 Created workflows:\n")
				for _, slug := range workflowResult.Created {
					fmt.Printf("   + %s\n", slug)
				}
			}
			if len(workflowResult.Updated) > 0 {
				fmt.Printf("📝 Updated workflows:\n")
				for _, slug := range workflowResult.Updated {
					fmt.Printf("   ~ %s\n", slug)
				}
			}
		}

		fmt.Printf("✅ Workflow migration completed!\n")
		fmt.Printf("   Created: %d, Updated: %d, Unchanged: %d\n",
			len(workflowResult.Created), len(workflowResult.Updated), len(workflowResult.Unchanged))

		if len(workflowResult.Errors) > 0 {
			fmt.Printf("\n⚠️  Workflow Warnings:\n")
			for _, err := range workflowResult.Errors {
				fmt.Printf("   %v\n", err)
			}
		}
	}

	fmt.Printf("\n✅ All migrations completed successfully!\n")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func validateWorkflows(registry *workflows.Registry) []error {
	var errors []error
	for _, wf := range registry.GetAll() {
		if err := wf.Validate(); err != nil {
			errors = append(errors, fmt.Errorf("%s: %w", wf.SystemSlug, err))
		}
	}
	return errors
}
