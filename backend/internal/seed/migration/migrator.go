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

// defaultMigrationLanguage is the language used when storing block definitions in DB
// In future, this could be expanded to store all languages in JSONB columns
const defaultMigrationLanguage = "ja"

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
// Blocks are processed in topological order (parents before children)
// to support multi-level inheritance (e.g., http -> rest-api -> bearer-api -> github-api)
func (m *Migrator) Migrate(ctx context.Context, registry *blocks.Registry) (*MigrationResult, error) {
	result := &MigrationResult{
		Created:   make([]string, 0),
		Updated:   make([]string, 0),
		Unchanged: make([]string, 0),
		Errors:    make([]error, 0),
	}

	allBlocks := registry.GetAll()

	// Build dependency graph and sort topologically
	sortedBlocks, err := topologicalSort(allBlocks)
	if err != nil {
		return nil, fmt.Errorf("failed to sort blocks: %w", err)
	}

	// Process blocks in topological order (parents first)
	for _, seedBlock := range sortedBlocks {
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

// topologicalSort sorts blocks so that parents are processed before children
// Uses Kahn's algorithm for topological sorting
func topologicalSort(allBlocks []*blocks.SystemBlockDefinition) ([]*blocks.SystemBlockDefinition, error) {
	// Build slug -> block map
	blockMap := make(map[string]*blocks.SystemBlockDefinition)
	for _, block := range allBlocks {
		blockMap[block.Slug] = block
	}

	// Calculate in-degree (number of dependencies) for each block
	inDegree := make(map[string]int)
	children := make(map[string][]string) // parent -> children

	for _, block := range allBlocks {
		if _, exists := inDegree[block.Slug]; !exists {
			inDegree[block.Slug] = 0
		}
		if block.ParentBlockSlug != "" {
			inDegree[block.Slug]++
			children[block.ParentBlockSlug] = append(children[block.ParentBlockSlug], block.Slug)
		}
	}

	// Start with blocks that have no dependencies (in-degree = 0)
	var queue []string
	for slug, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, slug)
		}
	}

	var sorted []*blocks.SystemBlockDefinition
	for len(queue) > 0 {
		// Pop from queue
		slug := queue[0]
		queue = queue[1:]

		block := blockMap[slug]
		if block != nil {
			sorted = append(sorted, block)
		}

		// Reduce in-degree for children
		for _, childSlug := range children[slug] {
			inDegree[childSlug]--
			if inDegree[childSlug] == 0 {
				queue = append(queue, childSlug)
			}
		}
	}

	// Check for cycles
	if len(sorted) != len(allBlocks) {
		return nil, fmt.Errorf("circular dependency detected in block inheritance")
	}

	return sorted, nil
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
	lang := defaultMigrationLanguage

	// Resolve parent block slug to ID
	var parentBlockID *uuid.UUID
	if seedBlock.ParentBlockSlug != "" {
		parentBlock, err := m.blockRepo.GetBySlug(ctx, nil, seedBlock.ParentBlockSlug)
		if err != nil {
			return "", fmt.Errorf("failed to get parent block %s: %w", seedBlock.ParentBlockSlug, err)
		}
		if parentBlock == nil {
			return "", fmt.Errorf("parent block %s not found", seedBlock.ParentBlockSlug)
		}
		parentBlockID = &parentBlock.ID
	}

	// Convert localized fields to single language for DB storage
	outputPorts := convertLocalizedOutputPorts(seedBlock.OutputPorts, lang)
	errorCodes := convertLocalizedErrorCodes(seedBlock.ErrorCodes, lang)

	block := &domain.BlockDefinition{
		ID:                  uuid.New(),
		TenantID:            nil, // System block
		Slug:                seedBlock.Slug,
		Name:                seedBlock.Name.Get(lang),
		Description:         seedBlock.Description.Get(lang),
		Category:            seedBlock.Category,
		Subcategory:         seedBlock.Subcategory,
		Icon:                seedBlock.Icon,
		ConfigSchema:        seedBlock.ConfigSchema.Get(lang),
		OutputSchema:        seedBlock.OutputSchema,
		OutputPorts:         outputPorts,
		Code:                seedBlock.Code,
		UIConfig:            seedBlock.UIConfig.Get(lang),
		ErrorCodes:          errorCodes,
		RequiredCredentials: seedBlock.RequiredCredentials,
		IsSystem:            true,
		IsPublic:            false,
		Version:             seedBlock.Version, // Use explicit version from seed
		Enabled:             seedBlock.Enabled,
		GroupKind:           seedBlock.GroupKind,
		IsContainer:         seedBlock.IsContainer,
		// Inheritance fields
		ParentBlockID:  parentBlockID,
		ConfigDefaults: seedBlock.ConfigDefaults,
		PreProcess:     seedBlock.PreProcess,
		PostProcess:    seedBlock.PostProcess,
		InternalSteps:  seedBlock.InternalSteps,
		// Declarative request/response
		Request:   seedBlock.Request,
		Response:  seedBlock.Response,
		CreatedAt: now,
		UpdatedAt: now,
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
	lang := defaultMigrationLanguage

	// Create version snapshot BEFORE updating only if version changes
	if m.versionRepo != nil && existing.Version != seedBlock.Version {
		version := domain.NewBlockVersion(existing, "Migration update", nil)
		if err := m.versionRepo.Create(ctx, version); err != nil {
			return "", fmt.Errorf("failed to create version snapshot: %w", err)
		}
	}

	// Resolve parent block slug to ID
	var parentBlockID *uuid.UUID
	if seedBlock.ParentBlockSlug != "" {
		parentBlock, err := m.blockRepo.GetBySlug(ctx, nil, seedBlock.ParentBlockSlug)
		if err != nil {
			return "", fmt.Errorf("failed to get parent block %s: %w", seedBlock.ParentBlockSlug, err)
		}
		if parentBlock == nil {
			return "", fmt.Errorf("parent block %s not found", seedBlock.ParentBlockSlug)
		}
		parentBlockID = &parentBlock.ID
	}

	// Convert localized fields to single language for DB storage
	outputPorts := convertLocalizedOutputPorts(seedBlock.OutputPorts, lang)
	errorCodes := convertLocalizedErrorCodes(seedBlock.ErrorCodes, lang)

	// Update fields (version is explicitly set from seedBlock)
	existing.Name = seedBlock.Name.Get(lang)
	existing.Description = seedBlock.Description.Get(lang)
	existing.Category = seedBlock.Category
	existing.Subcategory = seedBlock.Subcategory
	existing.Icon = seedBlock.Icon
	existing.ConfigSchema = seedBlock.ConfigSchema.Get(lang)
	existing.OutputSchema = seedBlock.OutputSchema
	existing.OutputPorts = outputPorts
	existing.Code = seedBlock.Code
	existing.UIConfig = seedBlock.UIConfig.Get(lang)
	existing.ErrorCodes = errorCodes
	existing.RequiredCredentials = seedBlock.RequiredCredentials
	existing.Enabled = seedBlock.Enabled
	existing.Version = seedBlock.Version // Use explicit version from seed (no auto-increment)
	existing.GroupKind = seedBlock.GroupKind
	existing.IsContainer = seedBlock.IsContainer
	// Inheritance fields
	existing.ParentBlockID = parentBlockID
	existing.ConfigDefaults = seedBlock.ConfigDefaults
	existing.PreProcess = seedBlock.PreProcess
	existing.PostProcess = seedBlock.PostProcess
	existing.InternalSteps = seedBlock.InternalSteps
	// Declarative request/response
	existing.Request = seedBlock.Request
	existing.Response = seedBlock.Response
	existing.UpdatedAt = time.Now().UTC()

	if err := m.blockRepo.Update(ctx, existing); err != nil {
		return "", fmt.Errorf("failed to update block: %w", err)
	}

	return "updated", nil
}

// hasChanges compares existing block with seed definition
func (m *Migrator) hasChanges(existing *domain.BlockDefinition, seed *blocks.SystemBlockDefinition) bool {
	lang := defaultMigrationLanguage

	// Compare version first
	if existing.Version != seed.Version {
		return true
	}

	// Compare key fields that would indicate a change (using localized values)
	if existing.Name != seed.Name.Get(lang) {
		return true
	}
	if existing.Description != seed.Description.Get(lang) {
		return true
	}
	if existing.Category != seed.Category {
		return true
	}
	if existing.Subcategory != seed.Subcategory {
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
	if existing.GroupKind != seed.GroupKind {
		return true
	}
	if existing.IsContainer != seed.IsContainer {
		return true
	}

	// Compare inheritance fields
	if existing.PreProcess != seed.PreProcess {
		return true
	}
	if existing.PostProcess != seed.PostProcess {
		return true
	}
	if !jsonEqual(existing.ConfigDefaults, seed.ConfigDefaults) {
		return true
	}
	if !internalStepsEqual(existing.InternalSteps, seed.InternalSteps) {
		return true
	}

	// Compare JSON fields (using localized schema)
	if !jsonEqual(existing.ConfigSchema, seed.ConfigSchema.Get(lang)) {
		return true
	}
	if !jsonEqual(existing.OutputSchema, seed.OutputSchema) {
		return true
	}
	if !jsonEqual(existing.UIConfig, seed.UIConfig.Get(lang)) {
		return true
	}
	if !jsonEqual(existing.RequiredCredentials, seed.RequiredCredentials) {
		return true
	}

	// Compare ports and error codes (using converted values)
	seedOutputPorts := convertLocalizedOutputPorts(seed.OutputPorts, lang)
	seedErrorCodes := convertLocalizedErrorCodes(seed.ErrorCodes, lang)
	if !outputPortsEqual(existing.OutputPorts, seedOutputPorts) {
		return true
	}
	if !errorCodesEqual(existing.ErrorCodes, seedErrorCodes) {
		return true
	}

	return false
}

// internalStepsEqual compares internal steps
func internalStepsEqual(a, b []domain.InternalStep) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Type != b[i].Type ||
			a[i].OutputKey != b[i].OutputKey {
			return false
		}
		if !jsonEqual(a[i].Config, b[i].Config) {
			return false
		}
	}
	return true
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
	lang := defaultMigrationLanguage
	changes := make([]string, 0)

	if existing.Version != seed.Version {
		changes = append(changes, "version")
	}
	if existing.Name != seed.Name.Get(lang) {
		changes = append(changes, "name")
	}
	if existing.Description != seed.Description.Get(lang) {
		changes = append(changes, "description")
	}
	if existing.Code != seed.Code {
		changes = append(changes, "code")
	}
	if !jsonEqual(existing.ConfigSchema, seed.ConfigSchema.Get(lang)) {
		changes = append(changes, "config_schema")
	}
	if !jsonEqual(existing.OutputSchema, seed.OutputSchema) {
		changes = append(changes, "output_schema")
	}
	seedOutputPorts := convertLocalizedOutputPorts(seed.OutputPorts, lang)
	if !outputPortsEqual(existing.OutputPorts, seedOutputPorts) {
		changes = append(changes, "output_ports")
	}
	// Inheritance fields
	if existing.PreProcess != seed.PreProcess {
		changes = append(changes, "pre_process")
	}
	if existing.PostProcess != seed.PostProcess {
		changes = append(changes, "post_process")
	}
	if !jsonEqual(existing.ConfigDefaults, seed.ConfigDefaults) {
		changes = append(changes, "config_defaults")
	}
	if !internalStepsEqual(existing.InternalSteps, seed.InternalSteps) {
		changes = append(changes, "internal_steps")
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

// convertLocalizedOutputPorts converts localized output ports to domain output ports
func convertLocalizedOutputPorts(ports []domain.LocalizedOutputPort, lang string) []domain.OutputPort {
	result := make([]domain.OutputPort, len(ports))
	for i, p := range ports {
		result[i] = domain.OutputPort{
			Name:        p.Name,
			Label:       p.Label.Get(lang),
			Description: p.Description.Get(lang),
			IsDefault:   p.IsDefault,
			Schema:      p.Schema,
		}
	}
	return result
}

// convertLocalizedErrorCodes converts localized error codes to domain error codes
func convertLocalizedErrorCodes(codes []domain.LocalizedErrorCodeDef, lang string) []domain.ErrorCodeDef {
	result := make([]domain.ErrorCodeDef, len(codes))
	for i, c := range codes {
		result[i] = domain.ErrorCodeDef{
			Code:        c.Code,
			Name:        c.Name.Get(lang),
			Description: c.Description.Get(lang),
			Retryable:   c.Retryable,
		}
	}
	return result
}
