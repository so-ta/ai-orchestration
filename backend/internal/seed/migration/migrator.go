package migration

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
	"github.com/souta/ai-orchestration/internal/seed/blocks"
)

// MigrationResult tracks what happened during migration
type MigrationResult struct {
	Created   []string // Slugs of newly created blocks
	Updated   []string // Slugs of updated blocks
	Unchanged []string // Slugs of unchanged blocks
	Errors    []error
}

// DryRunResult shows what would happen during migration without applying
type DryRunResult struct {
	ToCreate  []string      // Slugs of blocks to create
	ToUpdate  []UpdateInfo  // Info about blocks to update
	Unchanged []string      // Slugs of unchanged blocks
}

// UpdateInfo provides details about a block update
type UpdateInfo struct {
	Slug       string
	OldVersion int
	NewVersion int
	Reason     string
}

// Migrator handles block definition migration
type Migrator struct {
	blockRepo   repository.BlockDefinitionRepository
	versionRepo repository.BlockVersionRepository
}

// NewMigrator creates a new migrator
func NewMigrator(
	blockRepo repository.BlockDefinitionRepository,
	versionRepo repository.BlockVersionRepository,
) *Migrator {
	return &Migrator{
		blockRepo:   blockRepo,
		versionRepo: versionRepo,
	}
}

// Migrate performs UPSERT for all blocks in the registry
func (m *Migrator) Migrate(ctx context.Context, registry *blocks.Registry) (*MigrationResult, error) {
	result := &MigrationResult{
		Created:   make([]string, 0),
		Updated:   make([]string, 0),
		Unchanged: make([]string, 0),
		Errors:    make([]error, 0),
	}

	for _, seedBlock := range registry.GetAll() {
		action, err := m.upsertBlock(ctx, seedBlock)
		if err != nil {
			result.Errors = append(result.Errors,
				fmt.Errorf("block %s: %w", seedBlock.Slug, err))
			continue
		}

		switch action {
		case "created":
			result.Created = append(result.Created, seedBlock.Slug)
		case "updated":
			result.Updated = append(result.Updated, seedBlock.Slug)
		case "unchanged":
			result.Unchanged = append(result.Unchanged, seedBlock.Slug)
		}
	}

	return result, nil
}

// upsertBlock creates or updates a single block
func (m *Migrator) upsertBlock(ctx context.Context, seedBlock *blocks.SystemBlockDefinition) (string, error) {
	// Look up existing block by slug (system blocks have tenant_id = NULL)
	existing, err := m.blockRepo.GetBySlug(ctx, nil, seedBlock.Slug)
	if err != nil {
		return "", fmt.Errorf("failed to get existing block: %w", err)
	}

	if existing == nil {
		// CREATE new block
		return m.createBlock(ctx, seedBlock)
	}

	// Check if it's a system block (tenant_id = NULL)
	if existing.TenantID != nil {
		// Skip non-system blocks - don't overwrite tenant blocks
		return "unchanged", nil
	}

	// Check if update is needed
	if m.hasChanges(existing, seedBlock) {
		// UPDATE existing block
		return m.updateBlock(ctx, existing, seedBlock)
	}

	return "unchanged", nil
}

// createBlock creates a new system block
func (m *Migrator) createBlock(ctx context.Context, seedBlock *blocks.SystemBlockDefinition) (string, error) {
	now := time.Now().UTC()

	block := &domain.BlockDefinition{
		ID:                  uuid.New(),
		TenantID:            nil, // System block
		Slug:                seedBlock.Slug,
		Name:                seedBlock.Name,
		Description:         seedBlock.Description,
		Category:            seedBlock.Category,
		Icon:                seedBlock.Icon,
		ConfigSchema:        seedBlock.ConfigSchema,
		InputSchema:         seedBlock.InputSchema,
		OutputSchema:        seedBlock.OutputSchema,
		InputPorts:          seedBlock.InputPorts,
		OutputPorts:         seedBlock.OutputPorts,
		Code:                seedBlock.Code,
		UIConfig:            seedBlock.UIConfig,
		ErrorCodes:          seedBlock.ErrorCodes,
		RequiredCredentials: seedBlock.RequiredCredentials,
		IsSystem:            true,
		IsPublic:            false,
		Version:             seedBlock.Version, // Use explicit version from seed
		Enabled:             seedBlock.Enabled,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	if err := m.blockRepo.Create(ctx, block); err != nil {
		return "", fmt.Errorf("failed to create block: %w", err)
	}

	// Create initial version snapshot
	if m.versionRepo != nil {
		version := domain.NewBlockVersion(block, "Initial seed", nil)
		if err := m.versionRepo.Create(ctx, version); err != nil {
			// Log warning but don't fail the migration
			fmt.Printf("Warning: failed to create initial version for %s: %v\n", seedBlock.Slug, err)
		}
	}

	return "created", nil
}

// updateBlock updates an existing system block
func (m *Migrator) updateBlock(ctx context.Context, existing *domain.BlockDefinition, seedBlock *blocks.SystemBlockDefinition) (string, error) {
	// Create version snapshot BEFORE updating only if version changes
	if m.versionRepo != nil && existing.Version != seedBlock.Version {
		version := domain.NewBlockVersion(existing, "Migration update", nil)
		if err := m.versionRepo.Create(ctx, version); err != nil {
			return "", fmt.Errorf("failed to create version snapshot: %w", err)
		}
	}

	// Update fields (version is explicitly set from seedBlock)
	existing.Name = seedBlock.Name
	existing.Description = seedBlock.Description
	existing.Category = seedBlock.Category
	existing.Icon = seedBlock.Icon
	existing.ConfigSchema = seedBlock.ConfigSchema
	existing.InputSchema = seedBlock.InputSchema
	existing.OutputSchema = seedBlock.OutputSchema
	existing.InputPorts = seedBlock.InputPorts
	existing.OutputPorts = seedBlock.OutputPorts
	existing.Code = seedBlock.Code
	existing.UIConfig = seedBlock.UIConfig
	existing.ErrorCodes = seedBlock.ErrorCodes
	existing.RequiredCredentials = seedBlock.RequiredCredentials
	existing.Enabled = seedBlock.Enabled
	existing.Version = seedBlock.Version // Use explicit version from seed (no auto-increment)
	existing.UpdatedAt = time.Now().UTC()

	if err := m.blockRepo.Update(ctx, existing); err != nil {
		return "", fmt.Errorf("failed to update block: %w", err)
	}

	return "updated", nil
}

// hasChanges compares existing block with seed definition
func (m *Migrator) hasChanges(existing *domain.BlockDefinition, seed *blocks.SystemBlockDefinition) bool {
	// Compare version first
	if existing.Version != seed.Version {
		return true
	}

	// Compare key fields that would indicate a change
	if existing.Name != seed.Name {
		return true
	}
	if existing.Description != seed.Description {
		return true
	}
	if existing.Category != seed.Category {
		return true
	}
	if existing.Icon != seed.Icon {
		return true
	}
	if existing.Code != seed.Code {
		return true
	}
	if existing.Enabled != seed.Enabled {
		return true
	}

	// Compare JSON fields
	if !jsonEqual(existing.ConfigSchema, seed.ConfigSchema) {
		return true
	}
	if !jsonEqual(existing.InputSchema, seed.InputSchema) {
		return true
	}
	if !jsonEqual(existing.OutputSchema, seed.OutputSchema) {
		return true
	}
	if !jsonEqual(existing.UIConfig, seed.UIConfig) {
		return true
	}
	if !jsonEqual(existing.RequiredCredentials, seed.RequiredCredentials) {
		return true
	}

	// Compare ports and error codes
	if !portsEqual(existing.InputPorts, seed.InputPorts) {
		return true
	}
	if !outputPortsEqual(existing.OutputPorts, seed.OutputPorts) {
		return true
	}
	if !errorCodesEqual(existing.ErrorCodes, seed.ErrorCodes) {
		return true
	}

	return false
}

// jsonEqual compares two JSON raw messages
func jsonEqual(a, b json.RawMessage) bool {
	// Handle nil/empty cases
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) == 0 || len(b) == 0 {
		return false
	}

	// Normalize by unmarshaling and remarshaling
	var aVal, bVal interface{}
	if err := json.Unmarshal(a, &aVal); err != nil {
		return false
	}
	if err := json.Unmarshal(b, &bVal); err != nil {
		return false
	}

	aNorm, _ := json.Marshal(aVal)
	bNorm, _ := json.Marshal(bVal)

	return string(aNorm) == string(bNorm)
}

// portsEqual compares input ports
func portsEqual(a []domain.InputPort, b []domain.InputPort) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Name != b[i].Name ||
			a[i].Label != b[i].Label ||
			a[i].Description != b[i].Description ||
			a[i].Required != b[i].Required {
			return false
		}
		if !jsonEqual(a[i].Schema, b[i].Schema) {
			return false
		}
	}
	return true
}

// outputPortsEqual compares output ports
func outputPortsEqual(a []domain.OutputPort, b []domain.OutputPort) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Name != b[i].Name ||
			a[i].Label != b[i].Label ||
			a[i].Description != b[i].Description ||
			a[i].IsDefault != b[i].IsDefault {
			return false
		}
		if !jsonEqual(a[i].Schema, b[i].Schema) {
			return false
		}
	}
	return true
}

// errorCodesEqual compares error codes
func errorCodesEqual(a, b []domain.ErrorCodeDef) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Code != b[i].Code ||
			a[i].Name != b[i].Name ||
			a[i].Description != b[i].Description ||
			a[i].Retryable != b[i].Retryable {
			return false
		}
	}
	return true
}

// DryRun checks what would happen during migration without applying
func (m *Migrator) DryRun(ctx context.Context, registry *blocks.Registry) (*DryRunResult, error) {
	result := &DryRunResult{
		ToCreate:  make([]string, 0),
		ToUpdate:  make([]UpdateInfo, 0),
		Unchanged: make([]string, 0),
	}

	for _, seedBlock := range registry.GetAll() {
		existing, err := m.blockRepo.GetBySlug(ctx, nil, seedBlock.Slug)
		if err != nil {
			return nil, fmt.Errorf("failed to get existing block %s: %w", seedBlock.Slug, err)
		}

		if existing == nil {
			result.ToCreate = append(result.ToCreate, seedBlock.Slug)
			continue
		}

		// Skip non-system blocks
		if existing.TenantID != nil {
			result.Unchanged = append(result.Unchanged, seedBlock.Slug)
			continue
		}

		if m.hasChanges(existing, seedBlock) {
			reason := m.describeChanges(existing, seedBlock)
			result.ToUpdate = append(result.ToUpdate, UpdateInfo{
				Slug:       seedBlock.Slug,
				OldVersion: existing.Version,
				NewVersion: seedBlock.Version,
				Reason:     reason,
			})
		} else {
			result.Unchanged = append(result.Unchanged, seedBlock.Slug)
		}
	}

	return result, nil
}

// describeChanges returns a human-readable description of what changed
func (m *Migrator) describeChanges(existing *domain.BlockDefinition, seed *blocks.SystemBlockDefinition) string {
	changes := make([]string, 0)

	if existing.Version != seed.Version {
		changes = append(changes, "version")
	}
	if existing.Name != seed.Name {
		changes = append(changes, "name")
	}
	if existing.Description != seed.Description {
		changes = append(changes, "description")
	}
	if existing.Code != seed.Code {
		changes = append(changes, "code")
	}
	if !jsonEqual(existing.ConfigSchema, seed.ConfigSchema) {
		changes = append(changes, "config_schema")
	}
	if !jsonEqual(existing.InputSchema, seed.InputSchema) {
		changes = append(changes, "input_schema")
	}
	if !jsonEqual(existing.OutputSchema, seed.OutputSchema) {
		changes = append(changes, "output_schema")
	}
	if !portsEqual(existing.InputPorts, seed.InputPorts) {
		changes = append(changes, "input_ports")
	}
	if !outputPortsEqual(existing.OutputPorts, seed.OutputPorts) {
		changes = append(changes, "output_ports")
	}

	if len(changes) == 0 {
		return "no specific changes"
	}

	result := changes[0]
	for i := 1; i < len(changes); i++ {
		result += ", " + changes[i]
	}
	return result
}
