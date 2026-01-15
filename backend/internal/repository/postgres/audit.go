package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// AuditLogRepository implements repository.AuditLogRepository
type AuditLogRepository struct {
	pool *pgxpool.Pool
}

// NewAuditLogRepository creates a new AuditLogRepository
func NewAuditLogRepository(pool *pgxpool.Pool) *AuditLogRepository {
	return &AuditLogRepository{pool: pool}
}

func (r *AuditLogRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	query := `
		INSERT INTO audit_logs (
			id, tenant_id, actor_id, actor_email, action, resource_type,
			resource_id, metadata, ip_address, user_agent, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.pool.Exec(ctx, query,
		log.ID,
		log.TenantID,
		log.ActorID,
		log.ActorEmail,
		log.Action,
		log.ResourceType,
		log.ResourceID,
		log.Metadata,
		log.IPAddress,
		log.UserAgent,
		log.CreatedAt,
	)

	return err
}

func (r *AuditLogRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, filter repository.AuditLogFilter) ([]*domain.AuditLog, int, error) {
	// Build WHERE clause
	conditions := []string{"tenant_id = $1"}
	args := []interface{}{tenantID}
	argIdx := 2

	if filter.ActorID != nil {
		conditions = append(conditions, fmt.Sprintf("actor_id = $%d", argIdx))
		args = append(args, *filter.ActorID)
		argIdx++
	}
	if filter.Action != nil {
		conditions = append(conditions, fmt.Sprintf("action = $%d", argIdx))
		args = append(args, *filter.Action)
		argIdx++
	}
	if filter.ResourceType != nil {
		conditions = append(conditions, fmt.Sprintf("resource_type = $%d", argIdx))
		args = append(args, *filter.ResourceType)
		argIdx++
	}
	if filter.ResourceID != nil {
		conditions = append(conditions, fmt.Sprintf("resource_id = $%d", argIdx))
		args = append(args, *filter.ResourceID)
		argIdx++
	}
	if filter.StartTime != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, *filter.StartTime)
		argIdx++
	}
	if filter.EndTime != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, *filter.EndTime)
		argIdx++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count query
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM audit_logs WHERE %s`, whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// List query
	// Cast inet to text for scanning into string
	query := fmt.Sprintf(`
		SELECT id, tenant_id, actor_id, actor_email, action, resource_type,
			   resource_id, metadata, ip_address::text, user_agent, created_at
		FROM audit_logs
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	offset := (filter.Page - 1) * filter.Limit
	args = append(args, filter.Limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []*domain.AuditLog
	for rows.Next() {
		var log domain.AuditLog
		err := rows.Scan(
			&log.ID, &log.TenantID, &log.ActorID, &log.ActorEmail,
			&log.Action, &log.ResourceType, &log.ResourceID, &log.Metadata,
			&log.IPAddress, &log.UserAgent, &log.CreatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		logs = append(logs, &log)
	}

	return logs, total, nil
}

func (r *AuditLogRepository) ListByResource(ctx context.Context, tenantID uuid.UUID, resourceType domain.AuditResourceType, resourceID uuid.UUID) ([]*domain.AuditLog, error) {
	// Cast inet to text for scanning into string
	query := `
		SELECT id, tenant_id, actor_id, actor_email, action, resource_type,
			   resource_id, metadata, ip_address::text, user_agent, created_at
		FROM audit_logs
		WHERE tenant_id = $1 AND resource_type = $2 AND resource_id = $3
		ORDER BY created_at DESC
		LIMIT 100
	`

	rows, err := r.pool.Query(ctx, query, tenantID, resourceType, resourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*domain.AuditLog
	for rows.Next() {
		var log domain.AuditLog
		err := rows.Scan(
			&log.ID, &log.TenantID, &log.ActorID, &log.ActorEmail,
			&log.Action, &log.ResourceType, &log.ResourceID, &log.Metadata,
			&log.IPAddress, &log.UserAgent, &log.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		logs = append(logs, &log)
	}

	return logs, nil
}
