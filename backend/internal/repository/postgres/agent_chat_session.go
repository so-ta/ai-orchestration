package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// AgentChatSessionRepository handles agent chat session persistence
type AgentChatSessionRepository struct {
	db *pgxpool.Pool
}

// NewAgentChatSessionRepository creates a new AgentChatSessionRepository
func NewAgentChatSessionRepository(db *pgxpool.Pool) *AgentChatSessionRepository {
	return &AgentChatSessionRepository{db: db}
}

// Create creates a new agent chat session
func (r *AgentChatSessionRepository) Create(ctx context.Context, session *domain.AgentChatSession) error {
	query := `
		INSERT INTO agent_chat_sessions (
			id, tenant_id, project_id, start_step_id, user_id,
			status, metadata, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.Exec(ctx, query,
		session.ID,
		session.TenantID,
		session.ProjectID,
		session.StartStepID,
		session.UserID,
		session.Status,
		session.Metadata,
		session.CreatedAt,
		session.UpdatedAt,
	)

	return err
}

// GetByID retrieves an agent chat session by ID
func (r *AgentChatSessionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.AgentChatSession, error) {
	query := `
		SELECT id, tenant_id, project_id, start_step_id, user_id,
		       status, metadata, created_at, updated_at, closed_at
		FROM agent_chat_sessions
		WHERE id = $1
	`

	session := &domain.AgentChatSession{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&session.ID,
		&session.TenantID,
		&session.ProjectID,
		&session.StartStepID,
		&session.UserID,
		&session.Status,
		&session.Metadata,
		&session.CreatedAt,
		&session.UpdatedAt,
		&session.ClosedAt,
	)
	if err != nil {
		return nil, err
	}

	return session, nil
}

// ListByProject retrieves agent chat sessions for a project
func (r *AgentChatSessionRepository) ListByProject(ctx context.Context, projectID uuid.UUID, filter repository.AgentChatSessionFilter) ([]*domain.AgentChatSession, int, error) {
	// Count query
	countQuery := `SELECT COUNT(*) FROM agent_chat_sessions WHERE project_id = $1`
	args := []interface{}{projectID}
	argIndex := 2

	if filter.Status != nil {
		countQuery += ` AND status = $` + string(rune('0'+argIndex))
		args = append(args, *filter.Status)
		argIndex++
	}

	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// List query
	query := `
		SELECT id, tenant_id, project_id, start_step_id, user_id,
		       status, metadata, created_at, updated_at, closed_at
		FROM agent_chat_sessions
		WHERE project_id = $1
	`
	args = []interface{}{projectID}
	argIndex = 2

	if filter.Status != nil {
		query += ` AND status = $` + string(rune('0'+argIndex))
		args = append(args, *filter.Status)
		argIndex++
	}

	query += ` ORDER BY created_at DESC`

	if filter.Limit > 0 {
		query += ` LIMIT $` + string(rune('0'+argIndex))
		args = append(args, filter.Limit)
		argIndex++
	}

	if filter.Page > 0 && filter.Limit > 0 {
		offset := (filter.Page - 1) * filter.Limit
		query += ` OFFSET $` + string(rune('0'+argIndex))
		args = append(args, offset)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var sessions []*domain.AgentChatSession
	for rows.Next() {
		session := &domain.AgentChatSession{}
		err := rows.Scan(
			&session.ID,
			&session.TenantID,
			&session.ProjectID,
			&session.StartStepID,
			&session.UserID,
			&session.Status,
			&session.Metadata,
			&session.CreatedAt,
			&session.UpdatedAt,
			&session.ClosedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		sessions = append(sessions, session)
	}

	return sessions, total, rows.Err()
}

// ListByUser retrieves agent chat sessions for a user
func (r *AgentChatSessionRepository) ListByUser(ctx context.Context, userID string, filter repository.AgentChatSessionFilter) ([]*domain.AgentChatSession, int, error) {
	// Count query
	countQuery := `SELECT COUNT(*) FROM agent_chat_sessions WHERE user_id = $1`
	args := []interface{}{userID}
	argIndex := 2

	if filter.Status != nil {
		countQuery += ` AND status = $` + string(rune('0'+argIndex))
		args = append(args, *filter.Status)
		argIndex++
	}

	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// List query
	query := `
		SELECT id, tenant_id, project_id, start_step_id, user_id,
		       status, metadata, created_at, updated_at, closed_at
		FROM agent_chat_sessions
		WHERE user_id = $1
	`
	args = []interface{}{userID}
	argIndex = 2

	if filter.Status != nil {
		query += ` AND status = $` + string(rune('0'+argIndex))
		args = append(args, *filter.Status)
		argIndex++
	}

	query += ` ORDER BY created_at DESC`

	if filter.Limit > 0 {
		query += ` LIMIT $` + string(rune('0'+argIndex))
		args = append(args, filter.Limit)
		argIndex++
	}

	if filter.Page > 0 && filter.Limit > 0 {
		offset := (filter.Page - 1) * filter.Limit
		query += ` OFFSET $` + string(rune('0'+argIndex))
		args = append(args, offset)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var sessions []*domain.AgentChatSession
	for rows.Next() {
		session := &domain.AgentChatSession{}
		err := rows.Scan(
			&session.ID,
			&session.TenantID,
			&session.ProjectID,
			&session.StartStepID,
			&session.UserID,
			&session.Status,
			&session.Metadata,
			&session.CreatedAt,
			&session.UpdatedAt,
			&session.ClosedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		sessions = append(sessions, session)
	}

	return sessions, total, rows.Err()
}

// Update updates an agent chat session
func (r *AgentChatSessionRepository) Update(ctx context.Context, session *domain.AgentChatSession) error {
	query := `
		UPDATE agent_chat_sessions
		SET status = $2, metadata = $3, updated_at = $4, closed_at = $5
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		session.ID,
		session.Status,
		session.Metadata,
		session.UpdatedAt,
		session.ClosedAt,
	)

	return err
}

// Close closes an agent chat session
func (r *AgentChatSessionRepository) Close(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	query := `
		UPDATE agent_chat_sessions
		SET status = $2, updated_at = $3, closed_at = $3
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, id, domain.AgentChatSessionStatusClosed, now)
	return err
}
