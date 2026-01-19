package usecase

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// BlockPackageUsecase handles custom block package business logic
type BlockPackageUsecase struct {
	packageRepo repository.CustomBlockPackageRepository
	blockRepo   repository.BlockDefinitionRepository
}

// NewBlockPackageUsecase creates a new BlockPackageUsecase
func NewBlockPackageUsecase(
	packageRepo repository.CustomBlockPackageRepository,
	blockRepo repository.BlockDefinitionRepository,
) *BlockPackageUsecase {
	return &BlockPackageUsecase{
		packageRepo: packageRepo,
		blockRepo:   blockRepo,
	}
}

// CreatePackageInput represents input for creating a block package
type CreatePackageInput struct {
	TenantID     uuid.UUID
	Name         string
	Version      string
	Description  string
	Blocks       []domain.PackageBlockDefinition
	Dependencies []domain.PackageDependency
	CreatedBy    *uuid.UUID
}

// Create creates a new block package
func (u *BlockPackageUsecase) Create(ctx context.Context, input CreatePackageInput) (*domain.CustomBlockPackage, error) {
	// Validate input
	if input.Name == "" {
		return nil, domain.NewValidationError("name", "name is required")
	}
	if input.Version == "" {
		return nil, domain.NewValidationError("version", "version is required")
	}

	// Check if package with same name/version already exists
	existing, _ := u.packageRepo.GetByNameAndVersion(ctx, input.TenantID, input.Name, input.Version)
	if existing != nil {
		return nil, domain.NewValidationError("version", "package with this name and version already exists")
	}

	pkg := domain.NewCustomBlockPackage(input.TenantID, input.Name, input.Version, input.CreatedBy)
	pkg.Description = input.Description

	if input.Blocks != nil {
		if err := pkg.SetBlocks(input.Blocks); err != nil {
			return nil, err
		}
	}
	if input.Dependencies != nil {
		if err := pkg.SetDependencies(input.Dependencies); err != nil {
			return nil, err
		}
	}

	if err := u.packageRepo.Create(ctx, pkg); err != nil {
		return nil, err
	}

	return pkg, nil
}

// GetByID retrieves a block package by ID
func (u *BlockPackageUsecase) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.CustomBlockPackage, error) {
	pkg, err := u.packageRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Verify tenant access
	if pkg.TenantID != tenantID {
		return nil, domain.ErrForbidden
	}

	return pkg, nil
}

// ListPackagesInput represents input for listing block packages
type ListPackagesInput struct {
	TenantID uuid.UUID
	Status   *domain.BlockPackageStatus
	Search   *string
	Page     int
	Limit    int
}

// ListPackagesOutput represents output for listing block packages
type ListPackagesOutput struct {
	Packages []*domain.CustomBlockPackage
	Page     int
	Limit    int
	Total    int
}

// List lists block packages for a tenant
func (u *BlockPackageUsecase) List(ctx context.Context, input ListPackagesInput) (*ListPackagesOutput, error) {
	filter := repository.BlockPackageFilter{
		Status: input.Status,
		Search: input.Search,
		Page:   input.Page,
		Limit:  input.Limit,
	}

	packages, total, err := u.packageRepo.ListByTenant(ctx, input.TenantID, filter)
	if err != nil {
		return nil, err
	}

	return &ListPackagesOutput{
		Packages: packages,
		Page:     input.Page,
		Limit:    input.Limit,
		Total:    total,
	}, nil
}

// UpdatePackageInput represents input for updating a block package
type UpdatePackageInput struct {
	ID           uuid.UUID
	TenantID     uuid.UUID
	Description  *string
	Blocks       []domain.PackageBlockDefinition
	Dependencies []domain.PackageDependency
	BundleURL    *string
}

// Update updates a block package
func (u *BlockPackageUsecase) Update(ctx context.Context, input UpdatePackageInput) (*domain.CustomBlockPackage, error) {
	pkg, err := u.packageRepo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	// Verify tenant access
	if pkg.TenantID != input.TenantID {
		return nil, domain.ErrForbidden
	}

	// Can only update draft packages
	if pkg.Status != domain.BlockPackageStatusDraft {
		return nil, domain.NewValidationError("status", "only draft packages can be updated")
	}

	if input.Description != nil {
		pkg.Description = *input.Description
	}
	if input.Blocks != nil {
		if err := pkg.SetBlocks(input.Blocks); err != nil {
			return nil, err
		}
	}
	if input.Dependencies != nil {
		if err := pkg.SetDependencies(input.Dependencies); err != nil {
			return nil, err
		}
	}
	if input.BundleURL != nil {
		pkg.BundleURL = *input.BundleURL
	}

	if err := u.packageRepo.Update(ctx, pkg); err != nil {
		return nil, err
	}

	return pkg, nil
}

// Delete deletes a block package
func (u *BlockPackageUsecase) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	pkg, err := u.packageRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Verify tenant access
	if pkg.TenantID != tenantID {
		return domain.ErrForbidden
	}

	// Can only delete draft packages
	if pkg.Status != domain.BlockPackageStatusDraft {
		return domain.NewValidationError("status", "only draft packages can be deleted")
	}

	return u.packageRepo.Delete(ctx, id)
}

// Publish publishes a block package and creates block definitions
func (u *BlockPackageUsecase) Publish(ctx context.Context, tenantID, id uuid.UUID) (*domain.CustomBlockPackage, error) {
	pkg, err := u.packageRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Verify tenant access
	if pkg.TenantID != tenantID {
		return nil, domain.ErrForbidden
	}

	// Can only publish draft packages
	if pkg.Status != domain.BlockPackageStatusDraft {
		return nil, domain.NewValidationError("status", "only draft packages can be published")
	}

	// Get blocks from package
	blocks, err := pkg.GetBlocks()
	if err != nil {
		return nil, err
	}

	// Create block definitions for each block in the package
	for _, blockDef := range blocks {
		category := domain.BlockCategory(blockDef.Category)
		if !category.IsValid() {
			category = domain.BlockCategoryCustom
		}

		configSchema := blockDef.ConfigSchema
		if configSchema == nil {
			configSchema = json.RawMessage(`{}`)
		}

		block := domain.NewBlockDefinition(
			&tenantID,
			blockDef.Slug,
			blockDef.Name,
			category,
		)
		block.Description = blockDef.Description
		block.Icon = blockDef.Icon
		block.Code = blockDef.Code
		block.ConfigSchema = configSchema
		block.OutputSchema = blockDef.OutputSchema
		block.UIConfig = blockDef.UIConfig

		// Check if block already exists
		existing, _ := u.blockRepo.GetBySlug(ctx, &tenantID, blockDef.Slug)
		if existing != nil {
			// Update existing block
			existing.Name = block.Name
			existing.Description = block.Description
			existing.Category = block.Category
			existing.Icon = block.Icon
			existing.Code = block.Code
			existing.ConfigSchema = block.ConfigSchema
			existing.OutputSchema = block.OutputSchema
			existing.UIConfig = block.UIConfig
			if err := u.blockRepo.Update(ctx, existing); err != nil {
				return nil, err
			}
		} else {
			// Create new block
			if err := u.blockRepo.Create(ctx, block); err != nil {
				return nil, err
			}
		}
	}

	// Mark package as published
	if err := u.packageRepo.Publish(ctx, id); err != nil {
		return nil, err
	}

	pkg.Publish()
	return pkg, nil
}

// Deprecate deprecates a block package
func (u *BlockPackageUsecase) Deprecate(ctx context.Context, tenantID, id uuid.UUID) (*domain.CustomBlockPackage, error) {
	pkg, err := u.packageRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Verify tenant access
	if pkg.TenantID != tenantID {
		return nil, domain.ErrForbidden
	}

	// Can only deprecate published packages
	if pkg.Status != domain.BlockPackageStatusPublished {
		return nil, domain.NewValidationError("status", "only published packages can be deprecated")
	}

	if err := u.packageRepo.Deprecate(ctx, id); err != nil {
		return nil, err
	}

	pkg.Deprecate()
	return pkg, nil
}
