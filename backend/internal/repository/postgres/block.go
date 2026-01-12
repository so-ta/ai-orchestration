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

func (r *BlockDefinitionRepository) Create(ctx context.Context, block *domain.BlockDefinition) error {
	errorCodesJSON, err := json.Marshal(block.ErrorCodes)
	if err != nil {
		return fmt.Errorf("failed to marshal error codes: %w", err)
	}

	inputPortsJSON, err := json.Marshal(block.InputPorts)
	if err != nil {
		return fmt.Errorf("failed to marshal input ports: %w", err)
	}

	outputPortsJSON, err := json.Marshal(block.OutputPorts)
	if err != nil {
		return fmt.Errorf("failed to marshal output ports: %w", err)
	}

	query := `
		INSERT INTO block_definitions (
			id, tenant_id, slug, name, description, category, icon,
			config_schema, input_schema, output_schema, input_ports, output_ports,
			error_codes, required_credentials, is_public,
			code, ui_config, is_system, version,
			enabled, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
	`

	_, err = r.pool.Exec(ctx, query,
		block.ID,
		block.TenantID,
		block.Slug,
		block.Name,
		block.Description,
		block.Category,
		block.Icon,
		block.ConfigSchema,
		block.InputSchema,
		block.OutputSchema,
		inputPortsJSON,
		outputPortsJSON,
		errorCodesJSON,
		block.RequiredCredentials,
		block.IsPublic,
		block.Code,
		block.UIConfig,
		block.IsSystem,
		block.Version,
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
	query := `
		SELECT id, tenant_id, slug, name, description, category, icon,
			   config_schema, input_schema, output_schema, input_ports, output_ports,
			   error_codes, required_credentials, COALESCE(is_public, false),
			   COALESCE(code, ''), COALESCE(ui_config, '{}'), COALESCE(is_system, false), COALESCE(version, 1),
			   enabled, created_at, updated_at
		FROM block_definitions
		WHERE id = $1
	`

	block := &domain.BlockDefinition{}
	var errorCodesJSON []byte
	var inputPortsJSON []byte
	var outputPortsJSON []byte

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&block.ID,
		&block.TenantID,
		&block.Slug,
		&block.Name,
		&block.Description,
		&block.Category,
		&block.Icon,
		&block.ConfigSchema,
		&block.InputSchema,
		&block.OutputSchema,
		&inputPortsJSON,
		&outputPortsJSON,
		&errorCodesJSON,
		&block.RequiredCredentials,
		&block.IsPublic,
		&block.Code,
		&block.UIConfig,
		&block.IsSystem,
		&block.Version,
		&block.Enabled,
		&block.CreatedAt,
		&block.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get block definition: %w", err)
	}

	if err := json.Unmarshal(errorCodesJSON, &block.ErrorCodes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal error codes: %w", err)
	}

	if len(inputPortsJSON) > 0 {
		if err := json.Unmarshal(inputPortsJSON, &block.InputPorts); err != nil {
			return nil, fmt.Errorf("failed to unmarshal input ports: %w", err)
		}
	}

	if len(outputPortsJSON) > 0 {
		if err := json.Unmarshal(outputPortsJSON, &block.OutputPorts); err != nil {
			return nil, fmt.Errorf("failed to unmarshal output ports: %w", err)
		}
	}

	return block, nil
}

func (r *BlockDefinitionRepository) GetBySlug(ctx context.Context, tenantID *uuid.UUID, slug string) (*domain.BlockDefinition, error) {
	// First try to find tenant-specific block, then fall back to system block
	query := `
		SELECT id, tenant_id, slug, name, description, category, icon,
			   config_schema, input_schema, output_schema, input_ports, output_ports,
			   error_codes, required_credentials, COALESCE(is_public, false),
			   COALESCE(code, ''), COALESCE(ui_config, '{}'), COALESCE(is_system, false), COALESCE(version, 1),
			   enabled, created_at, updated_at
		FROM block_definitions
		WHERE slug = $1 AND (tenant_id = $2 OR tenant_id IS NULL)
		ORDER BY tenant_id NULLS LAST
		LIMIT 1
	`

	block := &domain.BlockDefinition{}
	var errorCodesJSON []byte
	var inputPortsJSON []byte
	var outputPortsJSON []byte

	err := r.pool.QueryRow(ctx, query, slug, tenantID).Scan(
		&block.ID,
		&block.TenantID,
		&block.Slug,
		&block.Name,
		&block.Description,
		&block.Category,
		&block.Icon,
		&block.ConfigSchema,
		&block.InputSchema,
		&block.OutputSchema,
		&inputPortsJSON,
		&outputPortsJSON,
		&errorCodesJSON,
		&block.RequiredCredentials,
		&block.IsPublic,
		&block.Code,
		&block.UIConfig,
		&block.IsSystem,
		&block.Version,
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

	if err := json.Unmarshal(errorCodesJSON, &block.ErrorCodes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal error codes: %w", err)
	}

	if len(inputPortsJSON) > 0 {
		if err := json.Unmarshal(inputPortsJSON, &block.InputPorts); err != nil {
			return nil, fmt.Errorf("failed to unmarshal input ports: %w", err)
		}
	}

	if len(outputPortsJSON) > 0 {
		if err := json.Unmarshal(outputPortsJSON, &block.OutputPorts); err != nil {
			return nil, fmt.Errorf("failed to unmarshal output ports: %w", err)
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

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, slug, name, description, category, icon,
			   config_schema, input_schema, output_schema, input_ports, output_ports,
			   error_codes, required_credentials, COALESCE(is_public, false),
			   COALESCE(code, ''), COALESCE(ui_config, '{}'), COALESCE(is_system, false), COALESCE(version, 1),
			   enabled, created_at, updated_at
		FROM block_definitions
		%s
		ORDER BY category, name
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
		var inputPortsJSON []byte
		var outputPortsJSON []byte

		err := rows.Scan(
			&block.ID,
			&block.TenantID,
			&block.Slug,
			&block.Name,
			&block.Description,
			&block.Category,
			&block.Icon,
			&block.ConfigSchema,
			&block.InputSchema,
			&block.OutputSchema,
			&inputPortsJSON,
			&outputPortsJSON,
			&errorCodesJSON,
			&block.RequiredCredentials,
			&block.IsPublic,
			&block.Code,
			&block.UIConfig,
			&block.IsSystem,
			&block.Version,
			&block.Enabled,
			&block.CreatedAt,
			&block.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan block definition: %w", err)
		}

		if err := json.Unmarshal(errorCodesJSON, &block.ErrorCodes); err != nil {
			return nil, fmt.Errorf("failed to unmarshal error codes: %w", err)
		}

		if len(inputPortsJSON) > 0 {
			if err := json.Unmarshal(inputPortsJSON, &block.InputPorts); err != nil {
				return nil, fmt.Errorf("failed to unmarshal input ports: %w", err)
			}
		}

		if len(outputPortsJSON) > 0 {
			if err := json.Unmarshal(outputPortsJSON, &block.OutputPorts); err != nil {
				return nil, fmt.Errorf("failed to unmarshal output ports: %w", err)
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

	inputPortsJSON, err := json.Marshal(block.InputPorts)
	if err != nil {
		return fmt.Errorf("failed to marshal input ports: %w", err)
	}

	outputPortsJSON, err := json.Marshal(block.OutputPorts)
	if err != nil {
		return fmt.Errorf("failed to marshal output ports: %w", err)
	}

	query := `
		UPDATE block_definitions
		SET name = $2, description = $3, category = $4, icon = $5,
			config_schema = $6, input_schema = $7, output_schema = $8, input_ports = $9, output_ports = $10,
			error_codes = $11, required_credentials = $12, is_public = $13,
			code = $14, ui_config = $15, is_system = $16, version = $17,
			enabled = $18, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query,
		block.ID,
		block.Name,
		block.Description,
		block.Category,
		block.Icon,
		block.ConfigSchema,
		block.InputSchema,
		block.OutputSchema,
		inputPortsJSON,
		outputPortsJSON,
		errorCodesJSON,
		block.RequiredCredentials,
		block.IsPublic,
		block.Code,
		block.UIConfig,
		block.IsSystem,
		block.Version,
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
