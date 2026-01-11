package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// WorkflowRepository implements repository.WorkflowRepository
type WorkflowRepository struct {
	pool *pgxpool.Pool
}

// NewWorkflowRepository creates a new WorkflowRepository
func NewWorkflowRepository(pool *pgxpool.Pool) *WorkflowRepository {
	return &WorkflowRepository{pool: pool}
}

// Create creates a new workflow
func (r *WorkflowRepository) Create(ctx context.Context, w *domain.Workflow) error {
	query := `
		INSERT INTO workflows (id, tenant_id, name, description, status, version, input_schema, output_schema, draft, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := r.pool.Exec(ctx, query,
		w.ID, w.TenantID, w.Name, w.Description, w.Status, w.Version,
		w.InputSchema, w.OutputSchema, w.Draft, w.CreatedBy, w.CreatedAt, w.UpdatedAt,
	)
	return err
}

// GetByID retrieves a workflow by ID
func (r *WorkflowRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Workflow, error) {
	query := `
		SELECT id, tenant_id, name, description, status, version, input_schema, output_schema, draft,
		       created_by, published_at, created_at, updated_at, deleted_at
		FROM workflows
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`
	var w domain.Workflow
	err := r.pool.QueryRow(ctx, query, id, tenantID).Scan(
		&w.ID, &w.TenantID, &w.Name, &w.Description, &w.Status, &w.Version,
		&w.InputSchema, &w.OutputSchema, &w.Draft, &w.CreatedBy, &w.PublishedAt,
		&w.CreatedAt, &w.UpdatedAt, &w.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrWorkflowNotFound
	}
	if err != nil {
		return nil, err
	}
	// Set HasDraft flag
	w.HasDraft = w.HasUnsavedDraft()
	return &w, nil
}

// List retrieves workflows with pagination
func (r *WorkflowRepository) List(ctx context.Context, tenantID uuid.UUID, filter repository.WorkflowFilter) ([]*domain.Workflow, int, error) {
	// Count query
	countQuery := `SELECT COUNT(*) FROM workflows WHERE tenant_id = $1 AND deleted_at IS NULL`
	args := []interface{}{tenantID}
	argIndex := 2

	if filter.Status != nil {
		countQuery += ` AND status = $` + string(rune('0'+argIndex))
		args = append(args, *filter.Status)
		argIndex++
	}

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// List query
	query := `
		SELECT id, tenant_id, name, description, status, version, input_schema, output_schema, draft,
		       created_by, published_at, created_at, updated_at, deleted_at
		FROM workflows
		WHERE tenant_id = $1 AND deleted_at IS NULL
	`
	args = []interface{}{tenantID}
	argIndex = 2

	if filter.Status != nil {
		query += ` AND status = $2`
		args = append(args, *filter.Status)
		argIndex++
	}

	query += ` ORDER BY updated_at DESC`

	if filter.Limit > 0 {
		offset := (filter.Page - 1) * filter.Limit
		query += ` LIMIT $` + string(rune('0'+argIndex)) + ` OFFSET $` + string(rune('0'+argIndex+1))
		args = append(args, filter.Limit, offset)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var workflows []*domain.Workflow
	for rows.Next() {
		var w domain.Workflow
		if err := rows.Scan(
			&w.ID, &w.TenantID, &w.Name, &w.Description, &w.Status, &w.Version,
			&w.InputSchema, &w.OutputSchema, &w.Draft, &w.CreatedBy, &w.PublishedAt,
			&w.CreatedAt, &w.UpdatedAt, &w.DeletedAt,
		); err != nil {
			return nil, 0, err
		}
		// Set HasDraft flag
		w.HasDraft = w.HasUnsavedDraft()
		workflows = append(workflows, &w)
	}

	return workflows, total, nil
}

// Update updates a workflow
func (r *WorkflowRepository) Update(ctx context.Context, w *domain.Workflow) error {
	w.UpdatedAt = time.Now().UTC()
	query := `
		UPDATE workflows
		SET name = $1, description = $2, status = $3, version = $4,
		    input_schema = $5, output_schema = $6, draft = $7, published_at = $8, updated_at = $9
		WHERE id = $10 AND tenant_id = $11 AND deleted_at IS NULL
	`
	result, err := r.pool.Exec(ctx, query,
		w.Name, w.Description, w.Status, w.Version,
		w.InputSchema, w.OutputSchema, w.Draft, w.PublishedAt, w.UpdatedAt,
		w.ID, w.TenantID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrWorkflowNotFound
	}
	return nil
}

// Delete soft-deletes a workflow
func (r *WorkflowRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	query := `UPDATE workflows SET deleted_at = $1 WHERE id = $2 AND tenant_id = $3 AND deleted_at IS NULL`
	result, err := r.pool.Exec(ctx, query, time.Now().UTC(), id, tenantID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrWorkflowNotFound
	}
	return nil
}

// GetWithStepsAndEdges retrieves a workflow with its steps and edges
// If the workflow has a draft, it returns the draft data instead of the saved data
func (r *WorkflowRepository) GetWithStepsAndEdges(ctx context.Context, tenantID, id uuid.UUID) (*domain.Workflow, error) {
	w, err := r.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	// If workflow has a draft, return the draft data
	if w.HasUnsavedDraft() {
		draft, err := w.GetDraft()
		if err != nil {
			return nil, err
		}
		if draft != nil {
			w.Name = draft.Name
			w.Description = draft.Description
			w.InputSchema = draft.InputSchema
			w.OutputSchema = draft.OutputSchema
			w.Steps = draft.Steps
			w.Edges = draft.Edges
			w.HasDraft = true
			return w, nil
		}
	}

	// Get steps from database
	stepsQuery := `
		SELECT id, workflow_id, name, type, config, block_group_id, group_role, position_x, position_y, created_at, updated_at
		FROM steps
		WHERE workflow_id = $1
		ORDER BY created_at
	`
	rows, err := r.pool.Query(ctx, stepsQuery, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var s domain.Step
		var groupRole *string
		if err := rows.Scan(
			&s.ID, &s.WorkflowID, &s.Name, &s.Type, &s.Config,
			&s.BlockGroupID, &groupRole,
			&s.PositionX, &s.PositionY, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if groupRole != nil {
			s.GroupRole = *groupRole
		}
		w.Steps = append(w.Steps, s)
	}

	// Get edges from database
	edgesQuery := `
		SELECT id, workflow_id, source_step_id, target_step_id, source_port, target_port, condition, created_at
		FROM edges
		WHERE workflow_id = $1
	`
	rows, err = r.pool.Query(ctx, edgesQuery, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var e domain.Edge
		if err := rows.Scan(
			&e.ID, &e.WorkflowID, &e.SourceStepID, &e.TargetStepID, &e.SourcePort, &e.TargetPort, &e.Condition, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		w.Edges = append(w.Edges, e)
	}

	return w, nil
}
