package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// BlockPackageStatus represents the status of a custom block package
type BlockPackageStatus string

const (
	BlockPackageStatusDraft      BlockPackageStatus = "draft"
	BlockPackageStatusPublished  BlockPackageStatus = "published"
	BlockPackageStatusDeprecated BlockPackageStatus = "deprecated"
)

// CustomBlockPackage represents a package of custom blocks
type CustomBlockPackage struct {
	ID           uuid.UUID          `json:"id"`
	TenantID     uuid.UUID          `json:"tenant_id"`
	Name         string             `json:"name"`
	Version      string             `json:"version"`
	Description  string             `json:"description,omitempty"`
	BundleURL    string             `json:"bundle_url,omitempty"` // CDN URL for the bundle
	Blocks       json.RawMessage    `json:"blocks"`               // Array of block definitions
	Dependencies json.RawMessage    `json:"dependencies"`         // NPM-style dependencies
	Status       BlockPackageStatus `json:"status"`
	CreatedBy    *uuid.UUID         `json:"created_by,omitempty"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
}

// PackageBlockDefinition represents a block definition within a package
type PackageBlockDefinition struct {
	Slug         string          `json:"slug"`
	Name         string          `json:"name"`
	Description  string          `json:"description,omitempty"`
	Category     string          `json:"category"`
	Icon         string          `json:"icon,omitempty"`
	ConfigSchema json.RawMessage `json:"config_schema"`
	OutputSchema json.RawMessage `json:"output_schema,omitempty"`
	Code         string          `json:"code"`
	UIConfig     json.RawMessage `json:"ui_config,omitempty"`
}

// PackageDependency represents a package dependency
type PackageDependency struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// NewCustomBlockPackage creates a new custom block package
func NewCustomBlockPackage(tenantID uuid.UUID, name, version string, createdBy *uuid.UUID) *CustomBlockPackage {
	now := time.Now().UTC()
	return &CustomBlockPackage{
		ID:           uuid.New(),
		TenantID:     tenantID,
		Name:         name,
		Version:      version,
		Blocks:       json.RawMessage(`[]`),
		Dependencies: json.RawMessage(`[]`),
		Status:       BlockPackageStatusDraft,
		CreatedBy:    createdBy,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// GetBlocks returns the block definitions in this package
func (p *CustomBlockPackage) GetBlocks() ([]PackageBlockDefinition, error) {
	var blocks []PackageBlockDefinition
	if err := json.Unmarshal(p.Blocks, &blocks); err != nil {
		return nil, err
	}
	return blocks, nil
}

// SetBlocks sets the block definitions in this package
func (p *CustomBlockPackage) SetBlocks(blocks []PackageBlockDefinition) error {
	data, err := json.Marshal(blocks)
	if err != nil {
		return err
	}
	p.Blocks = data
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// GetDependencies returns the package dependencies
func (p *CustomBlockPackage) GetDependencies() ([]PackageDependency, error) {
	var deps []PackageDependency
	if err := json.Unmarshal(p.Dependencies, &deps); err != nil {
		return nil, err
	}
	return deps, nil
}

// SetDependencies sets the package dependencies
func (p *CustomBlockPackage) SetDependencies(deps []PackageDependency) error {
	data, err := json.Marshal(deps)
	if err != nil {
		return err
	}
	p.Dependencies = data
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// Publish marks the package as published
func (p *CustomBlockPackage) Publish() {
	p.Status = BlockPackageStatusPublished
	p.UpdatedAt = time.Now().UTC()
}

// Deprecate marks the package as deprecated
func (p *CustomBlockPackage) Deprecate() {
	p.Status = BlockPackageStatusDeprecated
	p.UpdatedAt = time.Now().UTC()
}

// IsPublished returns true if the package is published
func (p *CustomBlockPackage) IsPublished() bool {
	return p.Status == BlockPackageStatusPublished
}
