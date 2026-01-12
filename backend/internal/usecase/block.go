package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// BlockUsecase handles block definition business logic
type BlockUsecase struct {
	blockRepo   repository.BlockDefinitionRepository
	versionRepo repository.BlockVersionRepository
}

// NewBlockUsecase creates a new BlockUsecase
func NewBlockUsecase(
	blockRepo repository.BlockDefinitionRepository,
	versionRepo repository.BlockVersionRepository,
) *BlockUsecase {
	return &BlockUsecase{
		blockRepo:   blockRepo,
		versionRepo: versionRepo,
	}
}

// UpdateSystemBlockInput represents input for updating a system block
type UpdateSystemBlockInput struct {
	BlockID       uuid.UUID
	Name          *string
	Description   *string
	Code          *string
	ConfigSchema  json.RawMessage
	InputSchema   json.RawMessage
	OutputSchema  json.RawMessage
	UIConfig      json.RawMessage
	ChangeSummary string
	ChangedBy     *uuid.UUID
}

// UpdateSystemBlock updates a system block and creates a version history entry
func (u *BlockUsecase) UpdateSystemBlock(ctx context.Context, input UpdateSystemBlockInput) (*domain.BlockDefinition, error) {
	// Get existing block
	block, err := u.blockRepo.GetByID(ctx, input.BlockID)
	if err != nil {
		return nil, fmt.Errorf("failed to get block: %w", err)
	}
	if block == nil {
		return nil, domain.ErrBlockDefinitionNotFound
	}

	// Verify it's a system block
	if !block.IsSystem {
		return nil, fmt.Errorf("only system blocks can be updated via this endpoint")
	}

	// Create version snapshot before updating
	version := domain.NewBlockVersion(block, input.ChangeSummary, input.ChangedBy)
	if err := u.versionRepo.Create(ctx, version); err != nil {
		return nil, fmt.Errorf("failed to create version snapshot: %w", err)
	}

	// Update block fields
	if input.Name != nil {
		block.Name = *input.Name
	}
	if input.Description != nil {
		block.Description = *input.Description
	}
	if input.Code != nil {
		block.Code = *input.Code
	}
	if input.ConfigSchema != nil {
		block.ConfigSchema = input.ConfigSchema
	}
	if input.InputSchema != nil {
		block.InputSchema = input.InputSchema
	}
	if input.OutputSchema != nil {
		block.OutputSchema = input.OutputSchema
	}
	if input.UIConfig != nil {
		block.UIConfig = input.UIConfig
	}

	// Increment version
	block.Version++

	// Save updated block
	if err := u.blockRepo.Update(ctx, block); err != nil {
		return nil, fmt.Errorf("failed to update block: %w", err)
	}

	return block, nil
}

// RollbackSystemBlockInput represents input for rolling back a system block
type RollbackSystemBlockInput struct {
	BlockID   uuid.UUID
	Version   int
	ChangedBy *uuid.UUID
}

// RollbackSystemBlock rolls back a system block to a previous version
func (u *BlockUsecase) RollbackSystemBlock(ctx context.Context, input RollbackSystemBlockInput) (*domain.BlockDefinition, error) {
	// Get existing block
	block, err := u.blockRepo.GetByID(ctx, input.BlockID)
	if err != nil {
		return nil, fmt.Errorf("failed to get block: %w", err)
	}
	if block == nil {
		return nil, domain.ErrBlockDefinitionNotFound
	}

	// Verify it's a system block
	if !block.IsSystem {
		return nil, fmt.Errorf("only system blocks can be rolled back via this endpoint")
	}

	// Get the version to rollback to
	targetVersion, err := u.versionRepo.GetByBlockAndVersion(ctx, input.BlockID, input.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to get target version: %w", err)
	}
	if targetVersion == nil {
		return nil, domain.ErrBlockVersionNotFound
	}

	// Create version snapshot before rollback
	version := domain.NewBlockVersion(block, fmt.Sprintf("Rollback to version %d", input.Version), input.ChangedBy)
	if err := u.versionRepo.Create(ctx, version); err != nil {
		return nil, fmt.Errorf("failed to create version snapshot: %w", err)
	}

	// Apply the rollback
	block.Code = targetVersion.Code
	block.ConfigSchema = targetVersion.ConfigSchema
	block.InputSchema = targetVersion.InputSchema
	block.OutputSchema = targetVersion.OutputSchema
	block.UIConfig = targetVersion.UIConfig
	block.Version++

	// Save updated block
	if err := u.blockRepo.Update(ctx, block); err != nil {
		return nil, fmt.Errorf("failed to update block: %w", err)
	}

	return block, nil
}

// GetBlockVersions retrieves all versions of a block
func (u *BlockUsecase) GetBlockVersions(ctx context.Context, blockID uuid.UUID) ([]*domain.BlockVersion, error) {
	return u.versionRepo.ListByBlock(ctx, blockID)
}

// GetBlockVersion retrieves a specific version of a block
func (u *BlockUsecase) GetBlockVersion(ctx context.Context, blockID uuid.UUID, version int) (*domain.BlockVersion, error) {
	return u.versionRepo.GetByBlockAndVersion(ctx, blockID, version)
}

// ListSystemBlocks lists all system blocks
func (u *BlockUsecase) ListSystemBlocks(ctx context.Context) ([]*domain.BlockDefinition, error) {
	isSystem := true
	filter := repository.BlockDefinitionFilter{
		SystemOnly: true,
		IsSystem:   &isSystem,
	}
	return u.blockRepo.List(ctx, nil, filter)
}
