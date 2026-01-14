package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/souta/ai-orchestration/internal/repository/postgres"
	"github.com/souta/ai-orchestration/internal/seed/blocks"
	"github.com/souta/ai-orchestration/internal/seed/migration"
	"github.com/souta/ai-orchestration/internal/seed/validation"
	"github.com/souta/ai-orchestration/pkg/database"
)

func main() {
	// Parse flags
	validateOnly := flag.Bool("validate", false, "Only validate blocks without migrating")
	dryRun := flag.Bool("dry-run", false, "Show what would be changed without applying")
	verbose := flag.Bool("verbose", false, "Show detailed output")
	flag.Parse()

	// Create registry with all blocks
	registry := blocks.NewRegistry()

	fmt.Printf("📦 Loaded %d block definitions\n", registry.Count())

	// Validate all blocks
	validator := validation.NewBlockValidator()
	result := validator.ValidateAllWithResult(registry)

	fmt.Printf("\n🔍 Validation Results:\n")
	fmt.Printf("   Total blocks: %d\n", result.TotalBlocks)
	fmt.Printf("   Valid blocks: %d\n", result.ValidBlocks)
	fmt.Printf("   Invalid blocks: %d\n", result.InvalidBlocks)

	if len(result.Errors) > 0 {
		fmt.Printf("\n❌ Validation Errors:\n")
		for _, err := range result.Errors {
			fmt.Printf("   [%s.%s] %s\n", err.BlockSlug, err.Field, err.Message)
		}
		os.Exit(1)
	}

	fmt.Printf("\n✅ All blocks passed validation\n")

	if *validateOnly {
		os.Exit(0)
	}

	// Connect to database
	dbURL := getEnv("DATABASE_URL", "postgres://aio:aio_password@localhost:5432/ai_orchestration?sslmode=disable")
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

	// Create migrator
	migrator := migration.NewMigrator(blockRepo, versionRepo)

	if *dryRun {
		// Dry run mode
		fmt.Printf("\n🔄 Dry Run - Checking what would be changed...\n\n")

		changes, err := migrator.DryRun(ctx, registry)
		if err != nil {
			fmt.Printf("❌ Dry run failed: %v\n", err)
			os.Exit(1)
		}

		if len(changes.ToCreate) > 0 {
			fmt.Printf("📥 Blocks to create (%d):\n", len(changes.ToCreate))
			for _, slug := range changes.ToCreate {
				block, _ := registry.GetBySlug(slug)
				fmt.Printf("   + %s (v%d) - %s\n", slug, block.Version, block.Name)
			}
		}

		if len(changes.ToUpdate) > 0 {
			fmt.Printf("\n📝 Blocks to update (%d):\n", len(changes.ToUpdate))
			for _, change := range changes.ToUpdate {
				if *verbose {
					fmt.Printf("   ~ %s (v%d → v%d) - %s\n",
						change.Slug, change.OldVersion, change.NewVersion, change.Reason)
				} else {
					fmt.Printf("   ~ %s (v%d → v%d)\n",
						change.Slug, change.OldVersion, change.NewVersion)
				}
			}
		}

		if len(changes.Unchanged) > 0 && *verbose {
			fmt.Printf("\n✓ Unchanged blocks (%d):\n", len(changes.Unchanged))
			for _, slug := range changes.Unchanged {
				fmt.Printf("   = %s\n", slug)
			}
		}

		if len(changes.ToCreate) == 0 && len(changes.ToUpdate) == 0 {
			fmt.Printf("\n✅ No changes needed - all blocks are up to date\n")
		} else {
			fmt.Printf("\n📊 Summary: %d to create, %d to update, %d unchanged\n",
				len(changes.ToCreate), len(changes.ToUpdate), len(changes.Unchanged))
		}

		os.Exit(0)
	}

	// Run migration
	fmt.Printf("\n🚀 Running migration...\n\n")

	migrationResult, err := migrator.Migrate(ctx, registry)
	if err != nil {
		fmt.Printf("❌ Migration failed: %v\n", err)
		os.Exit(1)
	}

	if *verbose {
		if len(migrationResult.Created) > 0 {
			fmt.Printf("📥 Created blocks:\n")
			for _, slug := range migrationResult.Created {
				fmt.Printf("   + %s\n", slug)
			}
		}
		if len(migrationResult.Updated) > 0 {
			fmt.Printf("📝 Updated blocks:\n")
			for _, slug := range migrationResult.Updated {
				fmt.Printf("   ~ %s\n", slug)
			}
		}
		if len(migrationResult.Unchanged) > 0 {
			fmt.Printf("✓ Unchanged blocks:\n")
			for _, slug := range migrationResult.Unchanged {
				fmt.Printf("   = %s\n", slug)
			}
		}
		fmt.Println()
	}

	fmt.Printf("✅ Migration completed successfully!\n")
	fmt.Printf("   Created: %d\n", len(migrationResult.Created))
	fmt.Printf("   Updated: %d\n", len(migrationResult.Updated))
	fmt.Printf("   Unchanged: %d\n", len(migrationResult.Unchanged))

	if len(migrationResult.Errors) > 0 {
		fmt.Printf("\n⚠️  Warnings:\n")
		for _, err := range migrationResult.Errors {
			fmt.Printf("   %v\n", err)
		}
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
