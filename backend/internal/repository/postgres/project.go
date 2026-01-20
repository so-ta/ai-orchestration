package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// ProjectRepository implements repository.ProjectRepository
type ProjectRepository struct {
	db DB
}

// NewProjectRepository creates a new ProjectRepository
func NewProjectRepository(pool *pgxpool.Pool) *ProjectRepository {
	return &ProjectRepository{db: pool}
}

// NewProjectRepositoryWithDB creates a new ProjectRepository with a custom DB implementation
// This is primarily used for testing with mock databases
func NewProjectRepositoryWithDB(db DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

// Create creates a new project
func (r *ProjectRepository) Create(ctx context.Context, p *domain.Project) error {
	query := `
		INSERT INTO projects (id, tenant_id, name, description, status, version, variables, draft, created_by, created_at, updated_at, is_system, system_slug)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := r.db.Exec(ctx, query,
		p.ID, p.TenantID, p.Name, p.Description, p.Status, p.Version,
		p.Variables, p.Draft, p.CreatedBy, p.CreatedAt, p.UpdatedAt,
		p.IsSystem, p.SystemSlug,
	)
	if err != nil {
		return fmt.Errorf("create project: %w", err)
	}
	return nil
}

// GetByID retrieves a project by ID
// This also supports retrieving system projects (is_system = TRUE) regardless of tenant
func (r *ProjectRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Project, error) {
	query := `
		SELECT id, tenant_id, name, description, status, version, variables, draft,
		       created_by, published_at, created_at, updated_at, deleted_at, is_system, system_slug
		FROM projects
		WHERE id = $1 AND deleted_at IS NULL
		  AND (tenant_id = $2 OR is_system = TRUE)
	`
	var p domain.Project
	err := r.db.QueryRow(ctx, query, id, tenantID).Scan(
		&p.ID, &p.TenantID, &p.Name, &p.Description, &p.Status, &p.Version,
		&p.Variables, &p.Draft, &p.CreatedBy, &p.PublishedAt,
		&p.CreatedAt, &p.UpdatedAt, &p.DeletedAt, &p.IsSystem, &p.SystemSlug,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrProjectNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get project by ID: %w", err)
	}
	// Set HasDraft flag
	p.HasDraft = p.HasUnsavedDraft()
	return &p, nil
}

// List retrieves projects with pagination
func (r *ProjectRepository) List(ctx context.Context, tenantID uuid.UUID, filter repository.ProjectFilter) ([]*domain.Project, int, error) {
	// Count query
	countQuery := `SELECT COUNT(*) FROM projects WHERE tenant_id = $1 AND deleted_at IS NULL`
	args := []interface{}{tenantID}
	argIndex := 2

	if filter.Status != nil {
		countQuery += fmt.Sprintf(` AND status = $%d`, argIndex)
		args = append(args, *filter.Status)
		argIndex++
	}

	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count projects: %w", err)
	}

	// List query
	query := `
		SELECT id, tenant_id, name, description, status, version, variables, draft,
		       created_by, published_at, created_at, updated_at, deleted_at, is_system, system_slug
		FROM projects
		WHERE tenant_id = $1 AND deleted_at IS NULL
	`
	args = []interface{}{tenantID}
	argIndex = 2

	if filter.Status != nil {
		query += fmt.Sprintf(` AND status = $%d`, argIndex)
		args = append(args, *filter.Status)
		argIndex++
	}

	query += ` ORDER BY updated_at DESC`

	if filter.Limit > 0 {
		page := filter.Page
		if page < 1 {
			page = 1
		}
		offset := (page - 1) * filter.Limit
		query += fmt.Sprintf(` LIMIT $%d OFFSET $%d`, argIndex, argIndex+1)
		args = append(args, filter.Limit, offset)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	var projects []*domain.Project
	for rows.Next() {
		var p domain.Project
		if err := rows.Scan(
			&p.ID, &p.TenantID, &p.Name, &p.Description, &p.Status, &p.Version,
			&p.Variables, &p.Draft, &p.CreatedBy, &p.PublishedAt,
			&p.CreatedAt, &p.UpdatedAt, &p.DeletedAt, &p.IsSystem, &p.SystemSlug,
		); err != nil {
			return nil, 0, fmt.Errorf("scan project: %w", err)
		}
		// Set HasDraft flag
		p.HasDraft = p.HasUnsavedDraft()
		projects = append(projects, &p)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate projects: %w", err)
	}

	return projects, total, nil
}

// Update updates a project
func (r *ProjectRepository) Update(ctx context.Context, p *domain.Project) error {
	p.UpdatedAt = time.Now().UTC()
	query := `
		UPDATE projects
		SET name = $1, description = $2, status = $3, version = $4,
		    variables = $5, draft = $6, published_at = $7, updated_at = $8
		WHERE id = $9 AND tenant_id = $10 AND deleted_at IS NULL
	`
	result, err := r.db.Exec(ctx, query,
		p.Name, p.Description, p.Status, p.Version,
		p.Variables, p.Draft, p.PublishedAt, p.UpdatedAt,
		p.ID, p.TenantID,
	)
	if err != nil {
		return fmt.Errorf("update project: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrProjectNotFound
	}
	return nil
}

// Delete soft-deletes a project
func (r *ProjectRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	query := `UPDATE projects SET deleted_at = $1 WHERE id = $2 AND tenant_id = $3 AND deleted_at IS NULL`
	result, err := r.db.Exec(ctx, query, time.Now().UTC(), id, tenantID)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrProjectNotFound
	}
	return nil
}

// GetWithStepsAndEdges retrieves a project with its steps and edges
// If the project has a draft, it returns the draft data instead of the saved data
func (r *ProjectRepository) GetWithStepsAndEdges(ctx context.Context, tenantID, id uuid.UUID) (*domain.Project, error) {
	p, err := r.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	// If project has a draft, return the draft data
	if p.HasUnsavedDraft() {
		draft, err := p.GetDraft()
		if err != nil {
			return nil, fmt.Errorf("get draft: %w", err)
		}
		if draft != nil {
			p.Name = draft.Name
			p.Description = draft.Description
			p.Variables = draft.Variables
			p.Steps = draft.Steps
			p.Edges = draft.Edges
			p.BlockGroups = draft.BlockGroups
			p.HasDraft = true
			return p, nil
		}
	}

	// Get steps from database
	stepsQuery := `
		SELECT id, project_id, name, type, config, trigger_type, trigger_config,
		       block_group_id, group_role, position_x, position_y, created_at, updated_at
		FROM steps
		WHERE project_id = $1
		ORDER BY created_at
	`
	rows, err := r.db.Query(ctx, stepsQuery, id)
	if err != nil {
		return nil, fmt.Errorf("query steps: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var s domain.Step
		var groupRole *string
		if err := rows.Scan(
			&s.ID, &s.ProjectID, &s.Name, &s.Type, &s.Config, &s.TriggerType, &s.TriggerConfig,
			&s.BlockGroupID, &groupRole,
			&s.PositionX, &s.PositionY, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan step: %w", err)
		}
		if groupRole != nil {
			s.GroupRole = *groupRole
		}
		s.TenantID = tenantID
		p.Steps = append(p.Steps, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate steps: %w", err)
	}

	// Get edges from database
	edgesQuery := `
		SELECT id, project_id, source_step_id, target_step_id, source_block_group_id, target_block_group_id, source_port, target_port, condition, created_at
		FROM edges
		WHERE project_id = $1
	`
	rows, err = r.db.Query(ctx, edgesQuery, id)
	if err != nil {
		return nil, fmt.Errorf("query edges: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var e domain.Edge
		if err := rows.Scan(
			&e.ID, &e.ProjectID, &e.SourceStepID, &e.TargetStepID, &e.SourceBlockGroupID, &e.TargetBlockGroupID, &e.SourcePort, &e.TargetPort, &e.Condition, &e.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan edge: %w", err)
		}
		e.TenantID = tenantID
		p.Edges = append(p.Edges, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate edges: %w", err)
	}

	// Get block groups from database
	blockGroupsQuery := `
		SELECT id, tenant_id, project_id, name, type, parent_group_id, position_x, position_y, width, height,
		       pre_process, post_process, config, created_at, updated_at
		FROM block_groups
		WHERE project_id = $1
		ORDER BY created_at
	`
	rows, err = r.db.Query(ctx, blockGroupsQuery, id)
	if err != nil {
		return nil, fmt.Errorf("query block groups: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var bg domain.BlockGroup
		if err := rows.Scan(
			&bg.ID, &bg.TenantID, &bg.ProjectID, &bg.Name, &bg.Type,
			&bg.ParentGroupID, &bg.PositionX, &bg.PositionY, &bg.Width, &bg.Height,
			&bg.PreProcess, &bg.PostProcess, &bg.Config,
			&bg.CreatedAt, &bg.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan block group: %w", err)
		}
		p.BlockGroups = append(p.BlockGroups, bg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate block groups: %w", err)
	}

	return p, nil
}

// GetSystemBySlug retrieves a system project by its slug
// System projects are accessible across all tenants
func (r *ProjectRepository) GetSystemBySlug(ctx context.Context, slug string) (*domain.Project, error) {
	query := `
		SELECT id, tenant_id, name, description, status, version, variables, draft,
		       created_by, published_at, created_at, updated_at, deleted_at, is_system, system_slug
		FROM projects
		WHERE system_slug = $1 AND is_system = TRUE AND deleted_at IS NULL
	`
	var p domain.Project
	err := r.db.QueryRow(ctx, query, slug).Scan(
		&p.ID, &p.TenantID, &p.Name, &p.Description, &p.Status, &p.Version,
		&p.Variables, &p.Draft, &p.CreatedBy, &p.PublishedAt,
		&p.CreatedAt, &p.UpdatedAt, &p.DeletedAt, &p.IsSystem, &p.SystemSlug,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrProjectNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get system project by slug: %w", err)
	}
	// Set HasDraft flag
	p.HasDraft = p.HasUnsavedDraft()
	return &p, nil
}
