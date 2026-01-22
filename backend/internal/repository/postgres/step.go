package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/souta/ai-orchestration/internal/domain"
)

// StepRepository implements repository.StepRepository
type StepRepository struct {
	pool *pgxpool.Pool
}

// NewStepRepository creates a new StepRepository
func NewStepRepository(pool *pgxpool.Pool) *StepRepository {
	return &StepRepository{pool: pool}
}

// Create creates a new step
func (r *StepRepository) Create(ctx context.Context, s *domain.Step) error {
	query := `
		INSERT INTO steps (id, tenant_id, project_id, name, type, config, block_group_id, group_role, position_x, position_y,
			block_definition_id, credential_bindings, trigger_type, trigger_config, tool_name, tool_description, tool_input_schema, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`
	_, err := r.pool.Exec(ctx, query,
		s.ID, s.TenantID, s.ProjectID, s.Name, s.Type, s.Config,
		s.BlockGroupID, s.GroupRole,
		s.PositionX, s.PositionY,
		s.BlockDefinitionID, s.CredentialBindings,
		s.TriggerType, s.TriggerConfig,
		s.ToolName, s.ToolDescription, s.ToolInputSchema,
		s.CreatedAt, s.UpdatedAt,
	)
	return err
}

// GetByID retrieves a step by ID
func (r *StepRepository) GetByID(ctx context.Context, tenantID, projectID, id uuid.UUID) (*domain.Step, error) {
	query := `
		SELECT id, tenant_id, project_id, name, type, config, block_group_id, group_role, position_x, position_y,
			block_definition_id, credential_bindings, trigger_type, trigger_config, tool_name, tool_description, tool_input_schema, created_at, updated_at
		FROM steps
		WHERE id = $1 AND project_id = $2 AND tenant_id = $3
	`
	var s domain.Step
	var groupRole *string
	var triggerType *string
	err := r.pool.QueryRow(ctx, query, id, projectID, tenantID).Scan(
		&s.ID, &s.TenantID, &s.ProjectID, &s.Name, &s.Type, &s.Config,
		&s.BlockGroupID, &groupRole,
		&s.PositionX, &s.PositionY,
		&s.BlockDefinitionID, &s.CredentialBindings,
		&triggerType, &s.TriggerConfig,
		&s.ToolName, &s.ToolDescription, &s.ToolInputSchema,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrStepNotFound
	}
	if err != nil {
		return nil, err
	}
	if groupRole != nil {
		s.GroupRole = *groupRole
	}
	if triggerType != nil {
		tt := domain.StepTriggerType(*triggerType)
		s.TriggerType = &tt
	}
	return &s, nil
}

// GetByIDOnly retrieves a step by ID only (without tenant/project verification)
// Used for webhook triggers where only step ID is known
func (r *StepRepository) GetByIDOnly(ctx context.Context, id uuid.UUID) (*domain.Step, error) {
	query := `
		SELECT id, tenant_id, project_id, name, type, config, block_group_id, group_role, position_x, position_y,
			block_definition_id, credential_bindings, trigger_type, trigger_config, tool_name, tool_description, tool_input_schema, created_at, updated_at
		FROM steps
		WHERE id = $1
	`
	var s domain.Step
	var groupRole *string
	var triggerType *string
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.TenantID, &s.ProjectID, &s.Name, &s.Type, &s.Config,
		&s.BlockGroupID, &groupRole,
		&s.PositionX, &s.PositionY,
		&s.BlockDefinitionID, &s.CredentialBindings,
		&triggerType, &s.TriggerConfig,
		&s.ToolName, &s.ToolDescription, &s.ToolInputSchema,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrStepNotFound
	}
	if err != nil {
		return nil, err
	}
	if groupRole != nil {
		s.GroupRole = *groupRole
	}
	if triggerType != nil {
		tt := domain.StepTriggerType(*triggerType)
		s.TriggerType = &tt
	}
	return &s, nil
}

// ListByProject retrieves all steps for a project
func (r *StepRepository) ListByProject(ctx context.Context, tenantID, projectID uuid.UUID) ([]*domain.Step, error) {
	query := `
		SELECT id, tenant_id, project_id, name, type, config, block_group_id, group_role, position_x, position_y,
			block_definition_id, credential_bindings, trigger_type, trigger_config, tool_name, tool_description, tool_input_schema, created_at, updated_at
		FROM steps
		WHERE project_id = $1 AND tenant_id = $2
		ORDER BY created_at
	`
	rows, err := r.pool.Query(ctx, query, projectID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []*domain.Step
	for rows.Next() {
		var s domain.Step
		var groupRole *string
		var triggerType *string
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.ProjectID, &s.Name, &s.Type, &s.Config,
			&s.BlockGroupID, &groupRole,
			&s.PositionX, &s.PositionY,
			&s.BlockDefinitionID, &s.CredentialBindings,
			&triggerType, &s.TriggerConfig,
			&s.ToolName, &s.ToolDescription, &s.ToolInputSchema,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if groupRole != nil {
			s.GroupRole = *groupRole
		}
		if triggerType != nil {
			tt := domain.StepTriggerType(*triggerType)
			s.TriggerType = &tt
		}
		steps = append(steps, &s)
	}

	return steps, nil
}

// ListByBlockGroup retrieves all steps in a block group
func (r *StepRepository) ListByBlockGroup(ctx context.Context, tenantID, blockGroupID uuid.UUID) ([]*domain.Step, error) {
	query := `
		SELECT id, tenant_id, project_id, name, type, config, block_group_id, group_role, position_x, position_y,
			block_definition_id, credential_bindings, trigger_type, trigger_config, tool_name, tool_description, tool_input_schema, created_at, updated_at
		FROM steps
		WHERE block_group_id = $1 AND tenant_id = $2
		ORDER BY created_at
	`
	rows, err := r.pool.Query(ctx, query, blockGroupID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []*domain.Step
	for rows.Next() {
		var s domain.Step
		var groupRole *string
		var triggerType *string
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.ProjectID, &s.Name, &s.Type, &s.Config,
			&s.BlockGroupID, &groupRole,
			&s.PositionX, &s.PositionY,
			&s.BlockDefinitionID, &s.CredentialBindings,
			&triggerType, &s.TriggerConfig,
			&s.ToolName, &s.ToolDescription, &s.ToolInputSchema,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if groupRole != nil {
			s.GroupRole = *groupRole
		}
		if triggerType != nil {
			tt := domain.StepTriggerType(*triggerType)
			s.TriggerType = &tt
		}
		steps = append(steps, &s)
	}

	return steps, nil
}

// ListStartSteps retrieves all start steps (steps with trigger_type set) for a project
func (r *StepRepository) ListStartSteps(ctx context.Context, tenantID, projectID uuid.UUID) ([]*domain.Step, error) {
	query := `
		SELECT id, tenant_id, project_id, name, type, config, block_group_id, group_role, position_x, position_y,
			block_definition_id, credential_bindings, trigger_type, trigger_config, tool_name, tool_description, tool_input_schema, created_at, updated_at
		FROM steps
		WHERE project_id = $1 AND tenant_id = $2 AND trigger_type IS NOT NULL
		ORDER BY created_at
	`
	rows, err := r.pool.Query(ctx, query, projectID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []*domain.Step
	for rows.Next() {
		var s domain.Step
		var groupRole *string
		var triggerType *string
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.ProjectID, &s.Name, &s.Type, &s.Config,
			&s.BlockGroupID, &groupRole,
			&s.PositionX, &s.PositionY,
			&s.BlockDefinitionID, &s.CredentialBindings,
			&triggerType, &s.TriggerConfig,
			&s.ToolName, &s.ToolDescription, &s.ToolInputSchema,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if groupRole != nil {
			s.GroupRole = *groupRole
		}
		if triggerType != nil {
			tt := domain.StepTriggerType(*triggerType)
			s.TriggerType = &tt
		}
		steps = append(steps, &s)
	}

	return steps, nil
}

// GetStartStepByTriggerType retrieves a start step by its trigger type for a project
func (r *StepRepository) GetStartStepByTriggerType(ctx context.Context, tenantID, projectID uuid.UUID, triggerType domain.StepTriggerType) (*domain.Step, error) {
	query := `
		SELECT id, tenant_id, project_id, name, type, config, block_group_id, group_role, position_x, position_y,
			block_definition_id, credential_bindings, trigger_type, trigger_config, tool_name, tool_description, tool_input_schema, created_at, updated_at
		FROM steps
		WHERE project_id = $1 AND tenant_id = $2 AND trigger_type = $3
		LIMIT 1
	`
	var s domain.Step
	var groupRole *string
	var stepTriggerType *string
	err := r.pool.QueryRow(ctx, query, projectID, tenantID, string(triggerType)).Scan(
		&s.ID, &s.TenantID, &s.ProjectID, &s.Name, &s.Type, &s.Config,
		&s.BlockGroupID, &groupRole,
		&s.PositionX, &s.PositionY,
		&s.BlockDefinitionID, &s.CredentialBindings,
		&stepTriggerType, &s.TriggerConfig,
		&s.ToolName, &s.ToolDescription, &s.ToolInputSchema,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrStepNotFound
	}
	if err != nil {
		return nil, err
	}
	if groupRole != nil {
		s.GroupRole = *groupRole
	}
	if stepTriggerType != nil {
		stt := domain.StepTriggerType(*stepTriggerType)
		s.TriggerType = &stt
	}
	return &s, nil
}

// Update updates a step
func (r *StepRepository) Update(ctx context.Context, s *domain.Step) error {
	s.UpdatedAt = time.Now().UTC()
	query := `
		UPDATE steps
		SET name = $1, type = $2, config = $3, block_group_id = $4, group_role = $5, position_x = $6, position_y = $7,
			block_definition_id = $8, credential_bindings = $9, trigger_type = $10, trigger_config = $11,
			tool_name = $12, tool_description = $13, tool_input_schema = $14, updated_at = $15
		WHERE id = $16 AND project_id = $17 AND tenant_id = $18
	`
	result, err := r.pool.Exec(ctx, query,
		s.Name, s.Type, s.Config, s.BlockGroupID, s.GroupRole, s.PositionX, s.PositionY,
		s.BlockDefinitionID, s.CredentialBindings, s.TriggerType, s.TriggerConfig,
		s.ToolName, s.ToolDescription, s.ToolInputSchema, s.UpdatedAt,
		s.ID, s.ProjectID, s.TenantID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrStepNotFound
	}
	return nil
}

// Delete deletes a step
func (r *StepRepository) Delete(ctx context.Context, tenantID, projectID, id uuid.UUID) error {
	query := `DELETE FROM steps WHERE id = $1 AND project_id = $2 AND tenant_id = $3`
	result, err := r.pool.Exec(ctx, query, id, projectID, tenantID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrStepNotFound
	}
	return nil
}
