package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// WebhookRepository implements repository.WebhookRepository
type WebhookRepository struct {
	pool *pgxpool.Pool
}

// NewWebhookRepository creates a new WebhookRepository
func NewWebhookRepository(pool *pgxpool.Pool) *WebhookRepository {
	return &WebhookRepository{pool: pool}
}

func (r *WebhookRepository) Create(ctx context.Context, webhook *domain.Webhook) error {
	query := `
		INSERT INTO webhooks (
			id, tenant_id, workflow_id, workflow_version, name, secret, input_mapping,
			enabled, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.pool.Exec(ctx, query,
		webhook.ID,
		webhook.TenantID,
		webhook.WorkflowID,
		webhook.WorkflowVersion,
		webhook.Name,
		webhook.Secret,
		webhook.InputMapping,
		webhook.Enabled,
		webhook.CreatedAt,
		webhook.UpdatedAt,
	)

	return err
}

func (r *WebhookRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Webhook, error) {
	query := `
		SELECT id, tenant_id, workflow_id, workflow_version, name, secret, input_mapping,
			   enabled, last_triggered_at, trigger_count, created_at, updated_at
		FROM webhooks
		WHERE tenant_id = $1 AND id = $2
	`

	var webhook domain.Webhook
	err := r.pool.QueryRow(ctx, query, tenantID, id).Scan(
		&webhook.ID,
		&webhook.TenantID,
		&webhook.WorkflowID,
		&webhook.WorkflowVersion,
		&webhook.Name,
		&webhook.Secret,
		&webhook.InputMapping,
		&webhook.Enabled,
		&webhook.LastTriggeredAt,
		&webhook.TriggerCount,
		&webhook.CreatedAt,
		&webhook.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, domain.ErrWebhookNotFound
	}
	if err != nil {
		return nil, err
	}

	return &webhook, nil
}

func (r *WebhookRepository) GetByIDForTrigger(ctx context.Context, id uuid.UUID) (*domain.Webhook, error) {
	query := `
		SELECT w.id, w.tenant_id, w.workflow_id, w.name, w.secret, w.input_mapping,
			   w.enabled, w.last_triggered_at, w.trigger_count, w.created_at, w.updated_at,
			   wf.version
		FROM webhooks w
		JOIN workflows wf ON w.workflow_id = wf.id
		WHERE w.id = $1
	`

	var webhook domain.Webhook
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&webhook.ID,
		&webhook.TenantID,
		&webhook.WorkflowID,
		&webhook.Name,
		&webhook.Secret,
		&webhook.InputMapping,
		&webhook.Enabled,
		&webhook.LastTriggeredAt,
		&webhook.TriggerCount,
		&webhook.CreatedAt,
		&webhook.UpdatedAt,
		&webhook.WorkflowVersion,
	)

	if err == pgx.ErrNoRows {
		return nil, domain.ErrWebhookNotFound
	}
	if err != nil {
		return nil, err
	}

	return &webhook, nil
}

func (r *WebhookRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, filter repository.WebhookFilter) ([]*domain.Webhook, int, error) {
	// Count query
	countQuery := `SELECT COUNT(*) FROM webhooks WHERE tenant_id = $1`
	countArgs := []interface{}{tenantID}
	argIndex := 2

	if filter.WorkflowID != nil {
		countQuery += fmt.Sprintf(` AND workflow_id = $%d`, argIndex)
		countArgs = append(countArgs, *filter.WorkflowID)
		argIndex++
	}
	if filter.Enabled != nil {
		countQuery += fmt.Sprintf(` AND enabled = $%d`, argIndex)
		countArgs = append(countArgs, *filter.Enabled)
	}

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// List query
	query := `
		SELECT id, tenant_id, workflow_id, workflow_version, name, secret, input_mapping,
			   enabled, last_triggered_at, trigger_count, created_at, updated_at
		FROM webhooks
		WHERE tenant_id = $1
	`
	args := []interface{}{tenantID}
	argIdx := 2

	if filter.WorkflowID != nil {
		query += fmt.Sprintf(` AND workflow_id = $%d`, argIdx)
		args = append(args, *filter.WorkflowID)
		argIdx++
	}
	if filter.Enabled != nil {
		query += fmt.Sprintf(` AND enabled = $%d`, argIdx)
		args = append(args, *filter.Enabled)
		argIdx++
	}

	query += fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, argIdx, argIdx+1)
	offset := (filter.Page - 1) * filter.Limit
	args = append(args, filter.Limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var webhooks []*domain.Webhook
	for rows.Next() {
		var w domain.Webhook
		err := rows.Scan(
			&w.ID, &w.TenantID, &w.WorkflowID, &w.WorkflowVersion, &w.Name, &w.Secret,
			&w.InputMapping, &w.Enabled, &w.LastTriggeredAt,
			&w.TriggerCount, &w.CreatedAt, &w.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		webhooks = append(webhooks, &w)
	}

	return webhooks, total, nil
}

func (r *WebhookRepository) ListByWorkflow(ctx context.Context, tenantID, workflowID uuid.UUID) ([]*domain.Webhook, error) {
	query := `
		SELECT id, tenant_id, workflow_id, workflow_version, name, secret, input_mapping,
			   enabled, last_triggered_at, trigger_count, created_at, updated_at
		FROM webhooks
		WHERE tenant_id = $1 AND workflow_id = $2
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, tenantID, workflowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var webhooks []*domain.Webhook
	for rows.Next() {
		var w domain.Webhook
		err := rows.Scan(
			&w.ID, &w.TenantID, &w.WorkflowID, &w.WorkflowVersion, &w.Name, &w.Secret,
			&w.InputMapping, &w.Enabled, &w.LastTriggeredAt,
			&w.TriggerCount, &w.CreatedAt, &w.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		webhooks = append(webhooks, &w)
	}

	return webhooks, nil
}

func (r *WebhookRepository) Update(ctx context.Context, webhook *domain.Webhook) error {
	query := `
		UPDATE webhooks SET
			name = $3,
			workflow_version = $4,
			secret = $5,
			input_mapping = $6,
			enabled = $7,
			last_triggered_at = $8,
			trigger_count = $9,
			updated_at = $10
		WHERE tenant_id = $1 AND id = $2
	`

	result, err := r.pool.Exec(ctx, query,
		webhook.TenantID,
		webhook.ID,
		webhook.Name,
		webhook.WorkflowVersion,
		webhook.Secret,
		webhook.InputMapping,
		webhook.Enabled,
		webhook.LastTriggeredAt,
		webhook.TriggerCount,
		webhook.UpdatedAt,
	)

	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrWebhookNotFound
	}

	return nil
}

func (r *WebhookRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	query := `DELETE FROM webhooks WHERE tenant_id = $1 AND id = $2`

	result, err := r.pool.Exec(ctx, query, tenantID, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrWebhookNotFound
	}

	return nil
}
