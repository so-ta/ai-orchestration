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
		INSERT INTO steps (id, workflow_id, name, type, config, block_group_id, group_role, position_x, position_y,
			block_definition_id, credential_bindings, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := r.pool.Exec(ctx, query,
		s.ID, s.WorkflowID, s.Name, s.Type, s.Config,
		s.BlockGroupID, s.GroupRole,
		s.PositionX, s.PositionY,
		s.BlockDefinitionID, s.CredentialBindings,
		s.CreatedAt, s.UpdatedAt,
	)
	return err
}

// GetByID retrieves a step by ID
func (r *StepRepository) GetByID(ctx context.Context, workflowID, id uuid.UUID) (*domain.Step, error) {
	query := `
		SELECT id, workflow_id, name, type, config, block_group_id, group_role, position_x, position_y,
			block_definition_id, credential_bindings, created_at, updated_at
		FROM steps
		WHERE id = $1 AND workflow_id = $2
	`
	var s domain.Step
	err := r.pool.QueryRow(ctx, query, id, workflowID).Scan(
		&s.ID, &s.WorkflowID, &s.Name, &s.Type, &s.Config,
		&s.BlockGroupID, &s.GroupRole,
		&s.PositionX, &s.PositionY,
		&s.BlockDefinitionID, &s.CredentialBindings,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrStepNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// ListByWorkflow retrieves all steps for a workflow
func (r *StepRepository) ListByWorkflow(ctx context.Context, workflowID uuid.UUID) ([]*domain.Step, error) {
	query := `
		SELECT id, workflow_id, name, type, config, block_group_id, group_role, position_x, position_y,
			block_definition_id, credential_bindings, created_at, updated_at
		FROM steps
		WHERE workflow_id = $1
		ORDER BY created_at
	`
	rows, err := r.pool.Query(ctx, query, workflowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []*domain.Step
	for rows.Next() {
		var s domain.Step
		if err := rows.Scan(
			&s.ID, &s.WorkflowID, &s.Name, &s.Type, &s.Config,
			&s.BlockGroupID, &s.GroupRole,
			&s.PositionX, &s.PositionY,
			&s.BlockDefinitionID, &s.CredentialBindings,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		steps = append(steps, &s)
	}

	return steps, nil
}

// ListByBlockGroup retrieves all steps in a block group
func (r *StepRepository) ListByBlockGroup(ctx context.Context, blockGroupID uuid.UUID) ([]*domain.Step, error) {
	query := `
		SELECT id, workflow_id, name, type, config, block_group_id, group_role, position_x, position_y,
			block_definition_id, credential_bindings, created_at, updated_at
		FROM steps
		WHERE block_group_id = $1
		ORDER BY created_at
	`
	rows, err := r.pool.Query(ctx, query, blockGroupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []*domain.Step
	for rows.Next() {
		var s domain.Step
		if err := rows.Scan(
			&s.ID, &s.WorkflowID, &s.Name, &s.Type, &s.Config,
			&s.BlockGroupID, &s.GroupRole,
			&s.PositionX, &s.PositionY,
			&s.BlockDefinitionID, &s.CredentialBindings,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		steps = append(steps, &s)
	}

	return steps, nil
}

// Update updates a step
func (r *StepRepository) Update(ctx context.Context, s *domain.Step) error {
	s.UpdatedAt = time.Now().UTC()
	query := `
		UPDATE steps
		SET name = $1, type = $2, config = $3, block_group_id = $4, group_role = $5, position_x = $6, position_y = $7,
			block_definition_id = $8, credential_bindings = $9, updated_at = $10
		WHERE id = $11 AND workflow_id = $12
	`
	result, err := r.pool.Exec(ctx, query,
		s.Name, s.Type, s.Config, s.BlockGroupID, s.GroupRole, s.PositionX, s.PositionY,
		s.BlockDefinitionID, s.CredentialBindings, s.UpdatedAt,
		s.ID, s.WorkflowID,
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
func (r *StepRepository) Delete(ctx context.Context, workflowID, id uuid.UUID) error {
	query := `DELETE FROM steps WHERE id = $1 AND workflow_id = $2`
	result, err := r.pool.Exec(ctx, query, id, workflowID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrStepNotFound
	}
	return nil
}
