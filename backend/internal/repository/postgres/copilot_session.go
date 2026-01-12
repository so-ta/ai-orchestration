package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// CopilotSessionRepository implements repository.CopilotSessionRepository
type CopilotSessionRepository struct {
	pool *pgxpool.Pool
}

// NewCopilotSessionRepository creates a new CopilotSessionRepository
func NewCopilotSessionRepository(pool *pgxpool.Pool) repository.CopilotSessionRepository {
	return &CopilotSessionRepository{pool: pool}
}

// Create creates a new copilot session
func (r *CopilotSessionRepository) Create(ctx context.Context, session *domain.CopilotSession) error {
	query := `
		INSERT INTO copilot_sessions (id, tenant_id, user_id, workflow_id, title, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.pool.Exec(ctx, query,
		session.ID,
		session.TenantID,
		session.UserID,
		session.WorkflowID,
		session.Title,
		session.IsActive,
		session.CreatedAt,
		session.UpdatedAt,
	)
	return err
}

// GetByID retrieves a copilot session by ID
func (r *CopilotSessionRepository) GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.CopilotSession, error) {
	query := `
		SELECT id, tenant_id, user_id, workflow_id, title, is_active, created_at, updated_at
		FROM copilot_sessions
		WHERE id = $1 AND tenant_id = $2
	`
	var session domain.CopilotSession
	var title sql.NullString
	err := r.pool.QueryRow(ctx, query, id, tenantID).Scan(
		&session.ID,
		&session.TenantID,
		&session.UserID,
		&session.WorkflowID,
		&title,
		&session.IsActive,
		&session.CreatedAt,
		&session.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrCopilotSessionNotFound
		}
		return nil, err
	}
	if title.Valid {
		session.Title = title.String
	}
	return &session, nil
}

// GetActiveByUserAndWorkflow retrieves the active session for a user and workflow
func (r *CopilotSessionRepository) GetActiveByUserAndWorkflow(ctx context.Context, tenantID uuid.UUID, userID string, workflowID uuid.UUID) (*domain.CopilotSession, error) {
	query := `
		SELECT id, tenant_id, user_id, workflow_id, title, is_active, created_at, updated_at
		FROM copilot_sessions
		WHERE tenant_id = $1 AND user_id = $2 AND workflow_id = $3 AND is_active = true
		ORDER BY created_at DESC
		LIMIT 1
	`
	var session domain.CopilotSession
	var title sql.NullString
	err := r.pool.QueryRow(ctx, query, tenantID, userID, workflowID).Scan(
		&session.ID,
		&session.TenantID,
		&session.UserID,
		&session.WorkflowID,
		&title,
		&session.IsActive,
		&session.CreatedAt,
		&session.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // No active session found
		}
		return nil, err
	}
	if title.Valid {
		session.Title = title.String
	}
	return &session, nil
}

// GetWithMessages retrieves a session with all its messages
func (r *CopilotSessionRepository) GetWithMessages(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.CopilotSession, error) {
	// Get session
	session, err := r.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	// Get messages
	query := `
		SELECT id, session_id, role, content, metadata, created_at
		FROM copilot_messages
		WHERE session_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []domain.CopilotMessage
	for rows.Next() {
		var msg domain.CopilotMessage
		if err := rows.Scan(
			&msg.ID,
			&msg.SessionID,
			&msg.Role,
			&msg.Content,
			&msg.Metadata,
			&msg.CreatedAt,
		); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	session.Messages = messages
	return session, nil
}

// ListByUserAndWorkflow retrieves all sessions for a user and workflow
func (r *CopilotSessionRepository) ListByUserAndWorkflow(ctx context.Context, tenantID uuid.UUID, userID string, workflowID uuid.UUID) ([]*domain.CopilotSession, error) {
	query := `
		SELECT id, tenant_id, user_id, workflow_id, title, is_active, created_at, updated_at
		FROM copilot_sessions
		WHERE tenant_id = $1 AND user_id = $2 AND workflow_id = $3
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, tenantID, userID, workflowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*domain.CopilotSession
	for rows.Next() {
		var session domain.CopilotSession
		var title sql.NullString
		if err := rows.Scan(
			&session.ID,
			&session.TenantID,
			&session.UserID,
			&session.WorkflowID,
			&title,
			&session.IsActive,
			&session.CreatedAt,
			&session.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if title.Valid {
			session.Title = title.String
		}
		sessions = append(sessions, &session)
	}
	return sessions, nil
}

// Update updates a copilot session
func (r *CopilotSessionRepository) Update(ctx context.Context, session *domain.CopilotSession) error {
	query := `
		UPDATE copilot_sessions
		SET title = $1, is_active = $2, updated_at = $3
		WHERE id = $4 AND tenant_id = $5
	`
	_, err := r.pool.Exec(ctx, query,
		session.Title,
		session.IsActive,
		session.UpdatedAt,
		session.ID,
		session.TenantID,
	)
	return err
}

// AddMessage adds a message to a session
func (r *CopilotSessionRepository) AddMessage(ctx context.Context, message *domain.CopilotMessage) error {
	query := `
		INSERT INTO copilot_messages (id, session_id, role, content, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.pool.Exec(ctx, query,
		message.ID,
		message.SessionID,
		message.Role,
		message.Content,
		message.Metadata,
		message.CreatedAt,
	)
	return err
}

// CloseSession marks a session as inactive
func (r *CopilotSessionRepository) CloseSession(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
	query := `
		UPDATE copilot_sessions
		SET is_active = false, updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2
	`
	_, err := r.pool.Exec(ctx, query, id, tenantID)
	return err
}
