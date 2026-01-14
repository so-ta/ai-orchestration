package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// TenantRepository implements repository.TenantRepository
type TenantRepository struct {
	pool *pgxpool.Pool
}

// NewTenantRepository creates a new TenantRepository
func NewTenantRepository(pool *pgxpool.Pool) *TenantRepository {
	return &TenantRepository{pool: pool}
}

func (r *TenantRepository) Create(ctx context.Context, tenant *domain.Tenant) error {
	query := `
		INSERT INTO tenants (
			id, name, slug, status, plan,
			owner_email, owner_name, billing_email,
			settings, metadata, feature_flags, limits,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	_, err := r.pool.Exec(ctx, query,
		tenant.ID,
		tenant.Name,
		tenant.Slug,
		tenant.Status,
		tenant.Plan,
		tenant.OwnerEmail,
		tenant.OwnerName,
		tenant.BillingEmail,
		tenant.Settings,
		tenant.Metadata,
		tenant.FeatureFlags,
		tenant.Limits,
		tenant.CreatedAt,
		tenant.UpdatedAt,
	)

	return err
}

func (r *TenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	query := `
		SELECT id, name, slug, status, plan,
			   COALESCE(owner_email, ''), COALESCE(owner_name, ''), COALESCE(billing_email, ''),
			   settings, metadata, feature_flags, limits,
			   suspended_at, COALESCE(suspended_reason, ''),
			   created_at, updated_at, deleted_at
		FROM tenants
		WHERE id = $1 AND deleted_at IS NULL
	`

	var tenant domain.Tenant
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&tenant.ID,
		&tenant.Name,
		&tenant.Slug,
		&tenant.Status,
		&tenant.Plan,
		&tenant.OwnerEmail,
		&tenant.OwnerName,
		&tenant.BillingEmail,
		&tenant.Settings,
		&tenant.Metadata,
		&tenant.FeatureFlags,
		&tenant.Limits,
		&tenant.SuspendedAt,
		&tenant.SuspendedReason,
		&tenant.CreatedAt,
		&tenant.UpdatedAt,
		&tenant.DeletedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, repository.ErrTenantNotFound
	}
	if err != nil {
		return nil, err
	}

	return &tenant, nil
}

func (r *TenantRepository) GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	query := `
		SELECT id, name, slug, status, plan,
			   COALESCE(owner_email, ''), COALESCE(owner_name, ''), COALESCE(billing_email, ''),
			   settings, metadata, feature_flags, limits,
			   suspended_at, COALESCE(suspended_reason, ''),
			   created_at, updated_at, deleted_at
		FROM tenants
		WHERE slug = $1 AND deleted_at IS NULL
	`

	var tenant domain.Tenant
	err := r.pool.QueryRow(ctx, query, slug).Scan(
		&tenant.ID,
		&tenant.Name,
		&tenant.Slug,
		&tenant.Status,
		&tenant.Plan,
		&tenant.OwnerEmail,
		&tenant.OwnerName,
		&tenant.BillingEmail,
		&tenant.Settings,
		&tenant.Metadata,
		&tenant.FeatureFlags,
		&tenant.Limits,
		&tenant.SuspendedAt,
		&tenant.SuspendedReason,
		&tenant.CreatedAt,
		&tenant.UpdatedAt,
		&tenant.DeletedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, repository.ErrTenantNotFound
	}
	if err != nil {
		return nil, err
	}

	return &tenant, nil
}

func (r *TenantRepository) List(ctx context.Context, filter repository.TenantFilter) ([]*domain.Tenant, int, error) {
	// Build WHERE clause
	conditions := []string{}
	args := []interface{}{}
	argNum := 1

	if !filter.IncludeDeleted {
		conditions = append(conditions, "deleted_at IS NULL")
	}

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argNum))
		args = append(args, *filter.Status)
		argNum++
	}

	if filter.Plan != nil {
		conditions = append(conditions, fmt.Sprintf("plan = $%d", argNum))
		args = append(args, *filter.Plan)
		argNum++
	}

	if filter.Search != "" {
		searchPattern := "%" + strings.ToLower(filter.Search) + "%"
		conditions = append(conditions, fmt.Sprintf("(LOWER(name) LIKE $%d OR LOWER(slug) LIKE $%d OR LOWER(owner_email) LIKE $%d)", argNum, argNum, argNum))
		args = append(args, searchPattern)
		argNum++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM tenants %s", whereClause)
	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Pagination
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	// Query tenants
	query := fmt.Sprintf(`
		SELECT id, name, slug, status, plan,
			   COALESCE(owner_email, ''), COALESCE(owner_name, ''), COALESCE(billing_email, ''),
			   settings, metadata, feature_flags, limits,
			   suspended_at, COALESCE(suspended_reason, ''),
			   created_at, updated_at, deleted_at
		FROM tenants
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	tenants := []*domain.Tenant{}
	for rows.Next() {
		var t domain.Tenant
		err := rows.Scan(
			&t.ID,
			&t.Name,
			&t.Slug,
			&t.Status,
			&t.Plan,
			&t.OwnerEmail,
			&t.OwnerName,
			&t.BillingEmail,
			&t.Settings,
			&t.Metadata,
			&t.FeatureFlags,
			&t.Limits,
			&t.SuspendedAt,
			&t.SuspendedReason,
			&t.CreatedAt,
			&t.UpdatedAt,
			&t.DeletedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		tenants = append(tenants, &t)
	}

	return tenants, total, nil
}

func (r *TenantRepository) Update(ctx context.Context, tenant *domain.Tenant) error {
	query := `
		UPDATE tenants SET
			name = $2,
			slug = $3,
			status = $4,
			plan = $5,
			owner_email = $6,
			owner_name = $7,
			billing_email = $8,
			settings = $9,
			metadata = $10,
			feature_flags = $11,
			limits = $12,
			suspended_at = $13,
			suspended_reason = $14,
			updated_at = $15
		WHERE id = $1 AND deleted_at IS NULL
	`

	tenant.UpdatedAt = time.Now().UTC()

	result, err := r.pool.Exec(ctx, query,
		tenant.ID,
		tenant.Name,
		tenant.Slug,
		tenant.Status,
		tenant.Plan,
		tenant.OwnerEmail,
		tenant.OwnerName,
		tenant.BillingEmail,
		tenant.Settings,
		tenant.Metadata,
		tenant.FeatureFlags,
		tenant.Limits,
		tenant.SuspendedAt,
		tenant.SuspendedReason,
		tenant.UpdatedAt,
	)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return repository.ErrTenantNotFound
	}

	return nil
}

func (r *TenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE tenants SET
			status = $2,
			deleted_at = $3,
			updated_at = $3
		WHERE id = $1 AND deleted_at IS NULL
	`

	now := time.Now().UTC()
	result, err := r.pool.Exec(ctx, query, id, domain.TenantStatusInactive, now)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return repository.ErrTenantNotFound
	}

	return nil
}

func (r *TenantRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.TenantStatus, reason string) error {
	var query string
	var args []interface{}
	now := time.Now().UTC()

	if status == domain.TenantStatusSuspended {
		query = `
			UPDATE tenants SET
				status = $2,
				suspended_at = $3,
				suspended_reason = $4,
				updated_at = $3
			WHERE id = $1 AND deleted_at IS NULL
		`
		args = []interface{}{id, status, now, reason}
	} else {
		query = `
			UPDATE tenants SET
				status = $2,
				suspended_at = NULL,
				suspended_reason = '',
				updated_at = $3
			WHERE id = $1 AND deleted_at IS NULL
		`
		args = []interface{}{id, status, now}
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return repository.ErrTenantNotFound
	}

	return nil
}

func (r *TenantRepository) GetStats(ctx context.Context, id uuid.UUID) (*domain.TenantStats, error) {
	// Get workflow count
	workflowQuery := `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'published') as published
		FROM workflows
		WHERE tenant_id = $1 AND deleted_at IS NULL
	`

	var stats domain.TenantStats
	err := r.pool.QueryRow(ctx, workflowQuery, id).Scan(&stats.WorkflowCount, &stats.PublishedWorkflows)
	if err != nil && err != pgx.ErrNoRows {
		return nil, err
	}

	// Get run count
	runQuery := `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE created_at >= date_trunc('month', CURRENT_DATE)) as this_month
		FROM runs
		WHERE tenant_id = $1
	`
	err = r.pool.QueryRow(ctx, runQuery, id).Scan(&stats.RunCount, &stats.RunsThisMonth)
	if err != nil && err != pgx.ErrNoRows {
		return nil, err
	}

	// Get user count
	userQuery := `SELECT COUNT(*) FROM users WHERE tenant_id = $1`
	err = r.pool.QueryRow(ctx, userQuery, id).Scan(&stats.UserCount)
	if err != nil && err != pgx.ErrNoRows {
		return nil, err
	}

	// Get credential count
	credQuery := `SELECT COUNT(*) FROM credentials WHERE tenant_id = $1`
	err = r.pool.QueryRow(ctx, credQuery, id).Scan(&stats.CredentialCount)
	if err != nil && err != pgx.ErrNoRows {
		return nil, err
	}

	// Get usage cost
	usageQuery := `
		SELECT
			COALESCE(SUM(cost_usd), 0) as total,
			COALESCE(SUM(cost_usd) FILTER (WHERE created_at >= date_trunc('month', CURRENT_DATE)), 0) as this_month
		FROM usage_records
		WHERE tenant_id = $1
	`
	err = r.pool.QueryRow(ctx, usageQuery, id).Scan(&stats.TotalCostUSD, &stats.CostThisMonth)
	if err != nil && err != pgx.ErrNoRows {
		return nil, err
	}

	return &stats, nil
}

func (r *TenantRepository) GetAllStats(ctx context.Context) (map[uuid.UUID]*domain.TenantStats, error) {
	result := make(map[uuid.UUID]*domain.TenantStats)

	// Get all active tenants
	tenantsQuery := `SELECT id FROM tenants WHERE deleted_at IS NULL`
	rows, err := r.pool.Query(ctx, tenantsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tenantIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		tenantIDs = append(tenantIDs, id)
		result[id] = &domain.TenantStats{}
	}

	// Get workflow counts per tenant
	workflowQuery := `
		SELECT tenant_id,
			   COUNT(*) as total,
			   COUNT(*) FILTER (WHERE status = 'published') as published
		FROM workflows
		WHERE deleted_at IS NULL
		GROUP BY tenant_id
	`
	workflowRows, err := r.pool.Query(ctx, workflowQuery)
	if err != nil {
		return nil, err
	}
	defer workflowRows.Close()

	for workflowRows.Next() {
		var tenantID uuid.UUID
		var total, published int
		if err := workflowRows.Scan(&tenantID, &total, &published); err != nil {
			return nil, err
		}
		if stats, ok := result[tenantID]; ok {
			stats.WorkflowCount = total
			stats.PublishedWorkflows = published
		}
	}

	// Get run counts per tenant
	runQuery := `
		SELECT tenant_id,
			   COUNT(*) as total,
			   COUNT(*) FILTER (WHERE created_at >= date_trunc('month', CURRENT_DATE)) as this_month
		FROM runs
		GROUP BY tenant_id
	`
	runRows, err := r.pool.Query(ctx, runQuery)
	if err != nil {
		return nil, err
	}
	defer runRows.Close()

	for runRows.Next() {
		var tenantID uuid.UUID
		var total, thisMonth int
		if err := runRows.Scan(&tenantID, &total, &thisMonth); err != nil {
			return nil, err
		}
		if stats, ok := result[tenantID]; ok {
			stats.RunCount = total
			stats.RunsThisMonth = thisMonth
		}
	}

	// Get usage costs per tenant
	usageQuery := `
		SELECT tenant_id,
			   COALESCE(SUM(cost_usd), 0) as total,
			   COALESCE(SUM(cost_usd) FILTER (WHERE created_at >= date_trunc('month', CURRENT_DATE)), 0) as this_month
		FROM usage_records
		GROUP BY tenant_id
	`
	usageRows, err := r.pool.Query(ctx, usageQuery)
	if err != nil {
		return nil, err
	}
	defer usageRows.Close()

	for usageRows.Next() {
		var tenantID uuid.UUID
		var total, thisMonth float64
		if err := usageRows.Scan(&tenantID, &total, &thisMonth); err != nil {
			return nil, err
		}
		if stats, ok := result[tenantID]; ok {
			stats.TotalCostUSD = total
			stats.CostThisMonth = thisMonth
		}
	}

	return result, nil
}
