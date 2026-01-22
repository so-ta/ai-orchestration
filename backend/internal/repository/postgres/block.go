package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// BlockDefinitionRepository implements repository.BlockDefinitionRepository
type BlockDefinitionRepository struct {
	pool *pgxpool.Pool
}

// NewBlockDefinitionRepository creates a new BlockDefinitionRepository
func NewBlockDefinitionRepository(pool *pgxpool.Pool) *BlockDefinitionRepository {
	return &BlockDefinitionRepository{pool: pool}
}

// MaxInheritanceDepth is the maximum allowed inheritance depth
const MaxInheritanceDepth = 10

func (r *BlockDefinitionRepository) Create(ctx context.Context, block *domain.BlockDefinition) error {
	errorCodesJSON, err := json.Marshal(block.ErrorCodes)
	if err != nil {
		return fmt.Errorf("failed to marshal error codes: %w", err)
	}

	outputPortsJSON, err := json.Marshal(block.OutputPorts)
	if err != nil {
		return fmt.Errorf("failed to marshal output ports: %w", err)
	}

	internalStepsJSON, err := json.Marshal(block.InternalSteps)
	if err != nil {
		return fmt.Errorf("failed to marshal internal steps: %w", err)
	}

	// Convert empty GroupKind to nil for database
	var groupKind *string
	if block.GroupKind != "" {
		gk := string(block.GroupKind)
		groupKind = &gk
	}

	// Convert empty Subcategory to nil for database
	var subcategory *string
	if block.Subcategory != "" {
		sc := string(block.Subcategory)
		subcategory = &sc
	}

	// Marshal request/response configs
	var requestJSON, responseJSON []byte
	if block.Request != nil {
		requestJSON, err = json.Marshal(block.Request)
		if err != nil {
			return fmt.Errorf("failed to marshal request config: %w", err)
		}
	}
	if block.Response != nil {
		responseJSON, err = json.Marshal(block.Response)
		if err != nil {
			return fmt.Errorf("failed to marshal response config: %w", err)
		}
	}

	query := `
		INSERT INTO block_definitions (
			id, tenant_id, slug, name, description, category, subcategory, icon,
			config_schema, output_schema, output_ports,
			error_codes, required_credentials, is_public,
			code, ui_config, is_system, version,
			parent_block_id, config_defaults, pre_process, post_process, internal_steps,
			group_kind, is_container, request, response,
			enabled, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30)
	`

	_, err = r.pool.Exec(ctx, query,
		block.ID,
		block.TenantID,
		block.Slug,
		block.Name,
		block.Description,
		block.Category,
		subcategory,
		block.Icon,
		block.ConfigSchema,
		block.OutputSchema,
		outputPortsJSON,
		errorCodesJSON,
		block.RequiredCredentials,
		block.IsPublic,
		block.Code,
		block.UIConfig,
		block.IsSystem,
		block.Version,
		block.ParentBlockID,
		block.ConfigDefaults,
		block.PreProcess,
		block.PostProcess,
		internalStepsJSON,
		groupKind,
		block.IsContainer,
		requestJSON,
		responseJSON,
		block.Enabled,
		block.CreatedAt,
		block.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create block definition: %w", err)
	}

	return nil
}

func (r *BlockDefinitionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.BlockDefinition, error) {
	block, err := r.getByIDRaw(ctx, id)
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, nil
	}

	// Resolve inheritance if this block has a parent
	if block.ParentBlockID != nil {
		return r.resolveInheritance(ctx, block)
	}

	return block, nil
}

// GetByIDRaw returns a block without resolving inheritance (for internal use)
func (r *BlockDefinitionRepository) GetByIDRaw(ctx context.Context, id uuid.UUID) (*domain.BlockDefinition, error) {
	return r.getByIDRaw(ctx, id)
}

// getByIDRaw is the internal method that reads raw block data without inheritance resolution
func (r *BlockDefinitionRepository) getByIDRaw(ctx context.Context, id uuid.UUID) (*domain.BlockDefinition, error) {
	query := `
		SELECT id, tenant_id, slug, name, description, category, subcategory, icon,
			   config_schema, output_schema, output_ports,
			   COALESCE(error_codes, '[]'::jsonb), required_credentials, COALESCE(is_public, false),
			   COALESCE(code, ''), COALESCE(ui_config, '{}'), COALESCE(is_system, false), COALESCE(version, 1),
			   parent_block_id, COALESCE(config_defaults, '{}'), COALESCE(pre_process, ''), COALESCE(post_process, ''), COALESCE(internal_steps, '[]'),
			   group_kind, COALESCE(is_container, false), request, response,
			   enabled, created_at, updated_at
		FROM block_definitions
		WHERE id = $1
	`

	block := &domain.BlockDefinition{}
	var errorCodesJSON []byte
	var outputPortsJSON []byte
	var internalStepsJSON []byte
	var requestJSON []byte
	var responseJSON []byte
	var groupKind *string
	var subcategory *string

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&block.ID,
		&block.TenantID,
		&block.Slug,
		&block.Name,
		&block.Description,
		&block.Category,
		&subcategory,
		&block.Icon,
		&block.ConfigSchema,
		&block.OutputSchema,
		&outputPortsJSON,
		&errorCodesJSON,
		&block.RequiredCredentials,
		&block.IsPublic,
		&block.Code,
		&block.UIConfig,
		&block.IsSystem,
		&block.Version,
		&block.ParentBlockID,
		&block.ConfigDefaults,
		&block.PreProcess,
		&block.PostProcess,
		&internalStepsJSON,
		&groupKind,
		&block.IsContainer,
		&requestJSON,
		&responseJSON,
		&block.Enabled,
		&block.CreatedAt,
		&block.UpdatedAt,
	)
	if groupKind != nil {
		block.GroupKind = domain.BlockGroupKind(*groupKind)
	}
	if subcategory != nil {
		block.Subcategory = domain.BlockSubcategory(*subcategory)
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get block definition: %w", err)
	}

	if len(errorCodesJSON) > 0 {
		if err := json.Unmarshal(errorCodesJSON, &block.ErrorCodes); err != nil {
			return nil, fmt.Errorf("failed to unmarshal error codes: %w", err)
		}
	}

	if len(outputPortsJSON) > 0 {
		if err := json.Unmarshal(outputPortsJSON, &block.OutputPorts); err != nil {
			return nil, fmt.Errorf("failed to unmarshal output ports: %w", err)
		}
	}

	if len(internalStepsJSON) > 0 {
		if err := json.Unmarshal(internalStepsJSON, &block.InternalSteps); err != nil {
			return nil, fmt.Errorf("failed to unmarshal internal steps: %w", err)
		}
	}

	if len(requestJSON) > 0 {
		block.Request = &domain.RequestConfig{}
		if err := json.Unmarshal(requestJSON, block.Request); err != nil {
			return nil, fmt.Errorf("failed to unmarshal request config: %w", err)
		}
	}

	if len(responseJSON) > 0 {
		block.Response = &domain.ResponseConfig{}
		if err := json.Unmarshal(responseJSON, block.Response); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response config: %w", err)
		}
	}

	return block, nil
}

func (r *BlockDefinitionRepository) GetBySlug(ctx context.Context, tenantID *uuid.UUID, slug string) (*domain.BlockDefinition, error) {
	block, err := r.getBySlugRaw(ctx, tenantID, slug)
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, nil
	}

	// Resolve inheritance if this block has a parent
	if block.ParentBlockID != nil {
		return r.resolveInheritance(ctx, block)
	}

	return block, nil
}

// getBySlugRaw is the internal method that reads raw block data without inheritance resolution
func (r *BlockDefinitionRepository) getBySlugRaw(ctx context.Context, tenantID *uuid.UUID, slug string) (*domain.BlockDefinition, error) {
	// First try to find tenant-specific block, then fall back to system block
	// Use proper NULL comparison: (tenant_id = $2) OR ($2 IS NULL AND tenant_id IS NULL)
	query := `
		SELECT id, tenant_id, slug, name, description, category, subcategory, icon,
			   config_schema, output_schema, output_ports,
			   COALESCE(error_codes, '[]'::jsonb), required_credentials, COALESCE(is_public, false),
			   COALESCE(code, ''), COALESCE(ui_config, '{}'), COALESCE(is_system, false), COALESCE(version, 1),
			   parent_block_id, COALESCE(config_defaults, '{}'), COALESCE(pre_process, ''), COALESCE(post_process, ''), COALESCE(internal_steps, '[]'),
			   group_kind, COALESCE(is_container, false), request, response,
			   enabled, created_at, updated_at
		FROM block_definitions
		WHERE slug = $1 AND ((tenant_id = $2) OR ($2 IS NULL AND tenant_id IS NULL) OR tenant_id IS NULL)
		ORDER BY tenant_id NULLS LAST
		LIMIT 1
	`

	block := &domain.BlockDefinition{}
	var errorCodesJSON []byte
	var outputPortsJSON []byte
	var internalStepsJSON []byte
	var requestJSON []byte
	var responseJSON []byte
	var groupKind *string
	var subcategory *string

	err := r.pool.QueryRow(ctx, query, slug, tenantID).Scan(
		&block.ID,
		&block.TenantID,
		&block.Slug,
		&block.Name,
		&block.Description,
		&block.Category,
		&subcategory,
		&block.Icon,
		&block.ConfigSchema,
		&block.OutputSchema,
		&outputPortsJSON,
		&errorCodesJSON,
		&block.RequiredCredentials,
		&block.IsPublic,
		&block.Code,
		&block.UIConfig,
		&block.IsSystem,
		&block.Version,
		&block.ParentBlockID,
		&block.ConfigDefaults,
		&block.PreProcess,
		&block.PostProcess,
		&internalStepsJSON,
		&groupKind,
		&block.IsContainer,
		&requestJSON,
		&responseJSON,
		&block.Enabled,
		&block.CreatedAt,
		&block.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get block definition by slug: %w", err)
	}

	if groupKind != nil {
		block.GroupKind = domain.BlockGroupKind(*groupKind)
	}
	if subcategory != nil {
		block.Subcategory = domain.BlockSubcategory(*subcategory)
	}

	if len(errorCodesJSON) > 0 {
		if err := json.Unmarshal(errorCodesJSON, &block.ErrorCodes); err != nil {
			return nil, fmt.Errorf("failed to unmarshal error codes: %w", err)
		}
	}

	if len(outputPortsJSON) > 0 {
		if err := json.Unmarshal(outputPortsJSON, &block.OutputPorts); err != nil {
			return nil, fmt.Errorf("failed to unmarshal output ports: %w", err)
		}
	}

	if len(internalStepsJSON) > 0 {
		if err := json.Unmarshal(internalStepsJSON, &block.InternalSteps); err != nil {
			return nil, fmt.Errorf("failed to unmarshal internal steps: %w", err)
		}
	}

	if len(requestJSON) > 0 {
		block.Request = &domain.RequestConfig{}
		if err := json.Unmarshal(requestJSON, block.Request); err != nil {
			return nil, fmt.Errorf("failed to unmarshal request config: %w", err)
		}
	}

	if len(responseJSON) > 0 {
		block.Response = &domain.ResponseConfig{}
		if err := json.Unmarshal(responseJSON, block.Response); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response config: %w", err)
		}
	}

	return block, nil
}

func (r *BlockDefinitionRepository) List(ctx context.Context, tenantID *uuid.UUID, filter repository.BlockDefinitionFilter) ([]*domain.BlockDefinition, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	// Include system blocks (tenant_id IS NULL) and tenant-specific blocks
	if filter.SystemOnly {
		conditions = append(conditions, "tenant_id IS NULL")
	} else if tenantID != nil {
		conditions = append(conditions, fmt.Sprintf("(tenant_id = $%d OR tenant_id IS NULL)", argNum))
		args = append(args, tenantID)
		argNum++
	} else {
		conditions = append(conditions, "tenant_id IS NULL")
	}

	if filter.Category != nil {
		conditions = append(conditions, fmt.Sprintf("category = $%d", argNum))
		args = append(args, *filter.Category)
		argNum++
	}

	if filter.EnabledOnly {
		conditions = append(conditions, "enabled = true")
	}

	if filter.IsSystem != nil {
		conditions = append(conditions, fmt.Sprintf("is_system = $%d", argNum))
		args = append(args, *filter.IsSystem)
		argNum++
	}

	if filter.Search != nil && *filter.Search != "" {
		// Search in name, description, and slug (case-insensitive)
		searchPattern := "%" + *filter.Search + "%"
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d OR slug ILIKE $%d)", argNum, argNum, argNum))
		args = append(args, searchPattern)
		argNum++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, slug, name, description, category, subcategory, icon,
			   config_schema, output_schema, output_ports,
			   COALESCE(error_codes, '[]'::jsonb), required_credentials, COALESCE(is_public, false),
			   COALESCE(code, ''), COALESCE(ui_config, '{}'), COALESCE(is_system, false), COALESCE(version, 1),
			   parent_block_id, COALESCE(config_defaults, '{}'), COALESCE(pre_process, ''), COALESCE(post_process, ''), COALESCE(internal_steps, '[]'),
			   group_kind, COALESCE(is_container, false), request, response,
			   enabled, created_at, updated_at
		FROM block_definitions
		%s
		ORDER BY category, subcategory, name
	`, whereClause)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list block definitions: %w", err)
	}
	defer rows.Close()

	var blocks []*domain.BlockDefinition
	for rows.Next() {
		block := &domain.BlockDefinition{}
		var errorCodesJSON []byte
		var outputPortsJSON []byte
		var internalStepsJSON []byte
		var requestJSON []byte
		var responseJSON []byte
		var groupKind *string
		var subcategory *string

		err := rows.Scan(
			&block.ID,
			&block.TenantID,
			&block.Slug,
			&block.Name,
			&block.Description,
			&block.Category,
			&subcategory,
			&block.Icon,
			&block.ConfigSchema,
			&block.OutputSchema,
			&outputPortsJSON,
			&errorCodesJSON,
			&block.RequiredCredentials,
			&block.IsPublic,
			&block.Code,
			&block.UIConfig,
			&block.IsSystem,
			&block.Version,
			&block.ParentBlockID,
			&block.ConfigDefaults,
			&block.PreProcess,
			&block.PostProcess,
			&internalStepsJSON,
			&groupKind,
			&block.IsContainer,
			&requestJSON,
			&responseJSON,
			&block.Enabled,
			&block.CreatedAt,
			&block.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan block definition: %w", err)
		}

		if groupKind != nil {
			block.GroupKind = domain.BlockGroupKind(*groupKind)
		}
		if subcategory != nil {
			block.Subcategory = domain.BlockSubcategory(*subcategory)
		}

		if len(errorCodesJSON) > 0 {
			if err := json.Unmarshal(errorCodesJSON, &block.ErrorCodes); err != nil {
				return nil, fmt.Errorf("failed to unmarshal error codes: %w", err)
			}
		}

		if len(outputPortsJSON) > 0 {
			if err := json.Unmarshal(outputPortsJSON, &block.OutputPorts); err != nil {
				return nil, fmt.Errorf("failed to unmarshal output ports: %w", err)
			}
		}

		if len(internalStepsJSON) > 0 {
			if err := json.Unmarshal(internalStepsJSON, &block.InternalSteps); err != nil {
				return nil, fmt.Errorf("failed to unmarshal internal steps: %w", err)
			}
		}

		if len(requestJSON) > 0 {
			block.Request = &domain.RequestConfig{}
			if err := json.Unmarshal(requestJSON, block.Request); err != nil {
				return nil, fmt.Errorf("failed to unmarshal request config: %w", err)
			}
		}

		if len(responseJSON) > 0 {
			block.Response = &domain.ResponseConfig{}
			if err := json.Unmarshal(responseJSON, block.Response); err != nil {
				return nil, fmt.Errorf("failed to unmarshal response config: %w", err)
			}
		}

		blocks = append(blocks, block)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating block definitions: %w", err)
	}

	return blocks, nil
}

func (r *BlockDefinitionRepository) Update(ctx context.Context, block *domain.BlockDefinition) error {
	errorCodesJSON, err := json.Marshal(block.ErrorCodes)
	if err != nil {
		return fmt.Errorf("failed to marshal error codes: %w", err)
	}

	outputPortsJSON, err := json.Marshal(block.OutputPorts)
	if err != nil {
		return fmt.Errorf("failed to marshal output ports: %w", err)
	}

	internalStepsJSON, err := json.Marshal(block.InternalSteps)
	if err != nil {
		return fmt.Errorf("failed to marshal internal steps: %w", err)
	}

	// Marshal request/response configs
	var requestJSON, responseJSON []byte
	if block.Request != nil {
		requestJSON, err = json.Marshal(block.Request)
		if err != nil {
			return fmt.Errorf("failed to marshal request config: %w", err)
		}
	}
	if block.Response != nil {
		responseJSON, err = json.Marshal(block.Response)
		if err != nil {
			return fmt.Errorf("failed to marshal response config: %w", err)
		}
	}

	// Convert empty GroupKind to nil for database
	var groupKind *string
	if block.GroupKind != "" {
		gk := string(block.GroupKind)
		groupKind = &gk
	}

	// Convert empty Subcategory to nil for database
	var subcategory *string
	if block.Subcategory != "" {
		sc := string(block.Subcategory)
		subcategory = &sc
	}

	query := `
		UPDATE block_definitions
		SET name = $2, description = $3, category = $4, subcategory = $5, icon = $6,
			config_schema = $7, output_schema = $8, output_ports = $9,
			error_codes = $10, required_credentials = $11, is_public = $12,
			code = $13, ui_config = $14, is_system = $15, version = $16,
			parent_block_id = $17, config_defaults = $18, pre_process = $19, post_process = $20, internal_steps = $21,
			group_kind = $22, is_container = $23, request = $24, response = $25,
			enabled = $26, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query,
		block.ID,
		block.Name,
		block.Description,
		block.Category,
		subcategory,
		block.Icon,
		block.ConfigSchema,
		block.OutputSchema,
		outputPortsJSON,
		errorCodesJSON,
		block.RequiredCredentials,
		block.IsPublic,
		block.Code,
		block.UIConfig,
		block.IsSystem,
		block.Version,
		block.ParentBlockID,
		block.ConfigDefaults,
		block.PreProcess,
		block.PostProcess,
		internalStepsJSON,
		groupKind,
		block.IsContainer,
		requestJSON,
		responseJSON,
		block.Enabled,
	)
	if err != nil {
		return fmt.Errorf("failed to update block definition: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrBlockDefinitionNotFound
	}

	return nil
}

func (r *BlockDefinitionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Only allow deleting custom blocks (tenant_id IS NOT NULL)
	query := `
		DELETE FROM block_definitions
		WHERE id = $1 AND tenant_id IS NOT NULL
	`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete block definition: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("block definition not found or is a system block")
	}

	return nil
}

// resolveInheritance resolves the inheritance chain for a block and populates resolved fields
func (r *BlockDefinitionRepository) resolveInheritance(ctx context.Context, block *domain.BlockDefinition) (*domain.BlockDefinition, error) {
	if block.ParentBlockID == nil {
		return block, nil // No inheritance
	}

	// Build inheritance chain: child -> parent -> ... -> root
	chain := []*domain.BlockDefinition{block}
	visited := map[uuid.UUID]bool{block.ID: true}
	current := block

	for i := 0; i < MaxInheritanceDepth && current.ParentBlockID != nil; i++ {
		parent, err := r.getByIDRaw(ctx, *current.ParentBlockID)
		if err != nil {
			return nil, fmt.Errorf("failed to get parent block: %w", err)
		}
		if parent == nil {
			return nil, domain.ErrParentBlockNotFound
		}

		// Check for circular inheritance
		if visited[parent.ID] {
			return nil, domain.ErrCircularInheritance
		}
		visited[parent.ID] = true

		chain = append(chain, parent)
		current = parent
	}

	// Check if we exceeded max depth
	if current.ParentBlockID != nil {
		return nil, domain.ErrInheritanceDepthExceeded
	}

	// Get the root block (last in chain)
	root := chain[len(chain)-1]

	// Verify root block can be inherited (has code)
	if !root.CanBeInherited() {
		return nil, domain.ErrBlockNotInheritable
	}

	// Build resolved block (copy child's basic info)
	resolved := &domain.BlockDefinition{
		// Basic info from child
		ID:          block.ID,
		TenantID:    block.TenantID,
		Slug:        block.Slug,
		Name:        block.Name,
		Description: block.Description,
		Category:    block.Category,
		Subcategory: block.Subcategory,
		Icon:        block.Icon,
		IsPublic:    block.IsPublic,
		IsSystem:    block.IsSystem,
		Version:     block.Version,
		Enabled:     block.Enabled,
		CreatedAt:   block.CreatedAt,
		UpdatedAt:   block.UpdatedAt,

		// Inheritance fields from child
		ParentBlockID:  block.ParentBlockID,
		ConfigDefaults: block.ConfigDefaults,
		PreProcess:     block.PreProcess,
		PostProcess:    block.PostProcess,
		InternalSteps:  block.InternalSteps,

		// Declarative request/response from child (merged later)
		Request:  block.Request,
		Response: block.Response,

		// Schemas - use child's if set, otherwise inherit from parent chain
		ConfigSchema: block.ConfigSchema,
		OutputSchema: block.OutputSchema,
		OutputPorts:  block.OutputPorts,
		ErrorCodes:   block.ErrorCodes,
		UIConfig:     block.UIConfig,

		// Code from root
		Code:         root.Code,
		ResolvedCode: root.Code,

		// RequiredCredentials - merge from chain
		RequiredCredentials: root.RequiredCredentials,
	}

	// Merge Request configs from chain (root -> child, child overrides)
	resolved.Request = mergeRequestConfigs(chain)

	// Merge Response configs from chain (root -> child, child overrides)
	resolved.Response = mergeResponseConfigs(chain)

	// Build PreProcessChain: child -> ... -> root (child's preProcess runs first)
	preProcessChain := make([]string, 0)
	for _, b := range chain {
		if b.PreProcess != "" {
			preProcessChain = append(preProcessChain, b.PreProcess)
		}
	}
	resolved.PreProcessChain = preProcessChain

	// Build PostProcessChain: root -> ... -> child (root's postProcess runs first)
	postProcessChain := make([]string, 0)
	for i := len(chain) - 1; i >= 0; i-- {
		if chain[i].PostProcess != "" {
			postProcessChain = append(postProcessChain, chain[i].PostProcess)
		}
	}
	resolved.PostProcessChain = postProcessChain

	// Merge config defaults (root -> ... -> child, child overrides)
	mergedDefaults := mergeConfigDefaults(chain)
	resolved.ResolvedConfigDefaults = mergedDefaults

	// Inherit schemas from parent if not set
	if len(resolved.OutputSchema) == 0 || string(resolved.OutputSchema) == "{}" || string(resolved.OutputSchema) == "null" {
		for i := 1; i < len(chain); i++ {
			if len(chain[i].OutputSchema) > 0 && string(chain[i].OutputSchema) != "{}" && string(chain[i].OutputSchema) != "null" {
				resolved.OutputSchema = chain[i].OutputSchema
				break
			}
		}
	}

	// Merge ConfigSchema properties from inheritance chain (parent properties first, child overrides)
	resolved.ConfigSchema = mergeConfigSchemas(chain)

	// Merge UIConfig from inheritance chain (parent first, child overrides)
	resolved.UIConfig = mergeUIConfigs(chain)

	return resolved, nil
}

// mergeConfigDefaults merges config defaults from inheritance chain
// Order: root -> ... -> child (child's values override parent's)
func mergeConfigDefaults(chain []*domain.BlockDefinition) json.RawMessage {
	merged := make(map[string]interface{})

	// Process from root to child (child overrides parent)
	for i := len(chain) - 1; i >= 0; i-- {
		block := chain[i]
		if len(block.ConfigDefaults) > 0 && string(block.ConfigDefaults) != "{}" {
			var defaults map[string]interface{}
			if err := json.Unmarshal(block.ConfigDefaults, &defaults); err == nil {
				for k, v := range defaults {
					merged[k] = v
				}
			}
		}
	}

	if len(merged) == 0 {
		return json.RawMessage("{}")
	}

	result, err := json.Marshal(merged)
	if err != nil {
		return json.RawMessage("{}")
	}
	return result
}

// mergeConfigSchemas merges config schemas from inheritance chain
// Properties are merged from root to child (child's properties override parent's)
func mergeConfigSchemas(chain []*domain.BlockDefinition) json.RawMessage {
	mergedProperties := make(map[string]interface{})
	mergedRequired := make([]string, 0)
	requiredSet := make(map[string]bool)

	// Process from root to child (child overrides parent)
	for i := len(chain) - 1; i >= 0; i-- {
		block := chain[i]
		if len(block.ConfigSchema) == 0 || string(block.ConfigSchema) == "{}" {
			continue
		}

		var schema map[string]interface{}
		if err := json.Unmarshal(block.ConfigSchema, &schema); err != nil {
			continue
		}

		// Merge properties
		if props, ok := schema["properties"].(map[string]interface{}); ok {
			for k, v := range props {
				mergedProperties[k] = v
			}
		}

		// Merge required fields
		if req, ok := schema["required"].([]interface{}); ok {
			for _, r := range req {
				if s, ok := r.(string); ok && !requiredSet[s] {
					mergedRequired = append(mergedRequired, s)
					requiredSet[s] = true
				}
			}
		}
	}

	if len(mergedProperties) == 0 {
		return json.RawMessage("{}")
	}

	merged := map[string]interface{}{
		"type":       "object",
		"properties": mergedProperties,
	}
	if len(mergedRequired) > 0 {
		merged["required"] = mergedRequired
	}

	result, err := json.Marshal(merged)
	if err != nil {
		return json.RawMessage("{}")
	}
	return result
}

// mergeUIConfigs merges UI configs from inheritance chain
// Groups and fieldGroups are merged from root to child (child overrides parent)
func mergeUIConfigs(chain []*domain.BlockDefinition) json.RawMessage {
	mergedGroups := make([]interface{}, 0)
	groupIDs := make(map[string]bool)
	mergedFieldGroups := make(map[string]interface{})
	mergedFieldOverrides := make(map[string]interface{})
	var icon, color string

	// Process from root to child (child overrides parent)
	for i := len(chain) - 1; i >= 0; i-- {
		block := chain[i]
		if len(block.UIConfig) == 0 || string(block.UIConfig) == "{}" || string(block.UIConfig) == "null" {
			continue
		}

		var uiConfig map[string]interface{}
		if err := json.Unmarshal(block.UIConfig, &uiConfig); err != nil {
			continue
		}

		// Take icon and color from child (last wins)
		if ic, ok := uiConfig["icon"].(string); ok && ic != "" {
			icon = ic
		}
		if c, ok := uiConfig["color"].(string); ok && c != "" {
			color = c
		}

		// Merge groups (avoid duplicates by ID)
		if groups, ok := uiConfig["groups"].([]interface{}); ok {
			for _, g := range groups {
				if group, ok := g.(map[string]interface{}); ok {
					if id, ok := group["id"].(string); ok && !groupIDs[id] {
						mergedGroups = append(mergedGroups, group)
						groupIDs[id] = true
					}
				}
			}
		}

		// Merge fieldGroups
		if fg, ok := uiConfig["fieldGroups"].(map[string]interface{}); ok {
			for k, v := range fg {
				mergedFieldGroups[k] = v
			}
		}

		// Merge fieldOverrides
		if fo, ok := uiConfig["fieldOverrides"].(map[string]interface{}); ok {
			for k, v := range fo {
				mergedFieldOverrides[k] = v
			}
		}
	}

	if len(mergedGroups) == 0 && len(mergedFieldGroups) == 0 && icon == "" && color == "" {
		return json.RawMessage("{}")
	}

	merged := make(map[string]interface{})
	if icon != "" {
		merged["icon"] = icon
	}
	if color != "" {
		merged["color"] = color
	}
	if len(mergedGroups) > 0 {
		merged["groups"] = mergedGroups
	}
	if len(mergedFieldGroups) > 0 {
		merged["fieldGroups"] = mergedFieldGroups
	}
	if len(mergedFieldOverrides) > 0 {
		merged["fieldOverrides"] = mergedFieldOverrides
	}

	result, err := json.Marshal(merged)
	if err != nil {
		return json.RawMessage("{}")
	}
	return result
}

// ValidateInheritance validates that a block can inherit from the specified parent
func (r *BlockDefinitionRepository) ValidateInheritance(ctx context.Context, blockID uuid.UUID, parentBlockID uuid.UUID) error {
	// Check parent exists
	parent, err := r.getByIDRaw(ctx, parentBlockID)
	if err != nil {
		return err
	}
	if parent == nil {
		return domain.ErrParentBlockNotFound
	}

	// Check parent can be inherited
	if !parent.CanBeInherited() {
		return domain.ErrBlockNotInheritable
	}

	// Check for circular reference
	visited := map[uuid.UUID]bool{blockID: true}
	current := parent

	for i := 0; i < MaxInheritanceDepth && current.ParentBlockID != nil; i++ {
		if visited[*current.ParentBlockID] || *current.ParentBlockID == blockID {
			return domain.ErrCircularInheritance
		}
		visited[*current.ParentBlockID] = true

		nextParent, err := r.getByIDRaw(ctx, *current.ParentBlockID)
		if err != nil {
			return err
		}
		if nextParent == nil {
			return domain.ErrParentBlockNotFound
		}
		current = nextParent
	}

	// Check inheritance depth
	if current.ParentBlockID != nil {
		return domain.ErrInheritanceDepthExceeded
	}

	return nil
}

// mergeRequestConfigs merges request configs from inheritance chain
// Order: root -> ... -> child (child's values override parent's)
func mergeRequestConfigs(chain []*domain.BlockDefinition) *domain.RequestConfig {
	var merged *domain.RequestConfig

	// Process from root to child (child overrides parent)
	for i := len(chain) - 1; i >= 0; i-- {
		req := chain[i].Request
		if req == nil {
			continue
		}

		if merged == nil {
			merged = &domain.RequestConfig{}
		}

		// Override non-empty fields
		if req.URL != "" {
			merged.URL = req.URL
		}
		if req.Method != "" {
			merged.Method = req.Method
		}
		if req.Body != nil {
			if merged.Body == nil {
				merged.Body = make(map[string]interface{})
			}
			for k, v := range req.Body {
				merged.Body[k] = v
			}
		}
		if req.Headers != nil {
			if merged.Headers == nil {
				merged.Headers = make(map[string]string)
			}
			for k, v := range req.Headers {
				merged.Headers[k] = v
			}
		}
		if req.QueryParams != nil {
			if merged.QueryParams == nil {
				merged.QueryParams = make(map[string]string)
			}
			for k, v := range req.QueryParams {
				merged.QueryParams[k] = v
			}
		}
	}

	return merged
}

// mergeResponseConfigs merges response configs from inheritance chain
// Order: root -> ... -> child (child's values override parent's)
func mergeResponseConfigs(chain []*domain.BlockDefinition) *domain.ResponseConfig {
	var merged *domain.ResponseConfig

	// Process from root to child (child overrides parent)
	for i := len(chain) - 1; i >= 0; i-- {
		resp := chain[i].Response
		if resp == nil {
			continue
		}

		if merged == nil {
			merged = &domain.ResponseConfig{}
		}

		// Override/merge output mapping
		if resp.OutputMapping != nil {
			if merged.OutputMapping == nil {
				merged.OutputMapping = make(map[string]string)
			}
			for k, v := range resp.OutputMapping {
				merged.OutputMapping[k] = v
			}
		}

		// Override success status if set
		if len(resp.SuccessStatus) > 0 {
			merged.SuccessStatus = resp.SuccessStatus
		}
	}

	return merged
}
