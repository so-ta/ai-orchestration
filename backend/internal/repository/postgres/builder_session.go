package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// BuilderSessionRepository implements repository.BuilderSessionRepository
type BuilderSessionRepository struct {
	pool *pgxpool.Pool
}

// NewBuilderSessionRepository creates a new BuilderSessionRepository
func NewBuilderSessionRepository(pool *pgxpool.Pool) repository.BuilderSessionRepository {
	return &BuilderSessionRepository{pool: pool}
}

// Create creates a new builder session
func (r *BuilderSessionRepository) Create(ctx context.Context, session *domain.BuilderSession) error {
	query := `
		INSERT INTO builder_sessions (
			id, tenant_id, user_id, copilot_session_id,
			status, hearing_phase, hearing_progress,
			spec, project_id, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.pool.Exec(ctx, query,
		session.ID,
		session.TenantID,
		session.UserID,
		session.CopilotSessionID,
		session.Status,
		session.HearingPhase,
		session.HearingProgress,
		session.Spec,
		session.ProjectID,
		session.CreatedAt,
		session.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create builder session: %w", err)
	}

	return nil
}

// GetByID retrieves a builder session by ID
func (r *BuilderSessionRepository) GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.BuilderSession, error) {
	query := `
		SELECT id, tenant_id, user_id, copilot_session_id,
			   status, hearing_phase, hearing_progress,
			   spec, project_id, created_at, updated_at
		FROM builder_sessions
		WHERE id = $1 AND tenant_id = $2
	`

	session := &domain.BuilderSession{}
	var copilotSessionID sql.NullString
	var spec []byte
	var projectID sql.NullString

	err := r.pool.QueryRow(ctx, query, id, tenantID).Scan(
		&session.ID,
		&session.TenantID,
		&session.UserID,
		&copilotSessionID,
		&session.Status,
		&session.HearingPhase,
		&session.HearingProgress,
		&spec,
		&projectID,
		&session.CreatedAt,
		&session.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("builder session not found: %w", err)
		}
		return nil, fmt.Errorf("get builder session: %w", err)
	}

	if copilotSessionID.Valid {
		id, _ := uuid.Parse(copilotSessionID.String)
		session.CopilotSessionID = &id
	}
	if spec != nil {
		session.Spec = json.RawMessage(spec)
	}
	if projectID.Valid {
		id, _ := uuid.Parse(projectID.String)
		session.ProjectID = &id
	}

	return session, nil
}

// GetWithMessages retrieves a session with all its messages
func (r *BuilderSessionRepository) GetWithMessages(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.BuilderSession, error) {
	session, err := r.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	// Get messages
	messagesQuery := `
		SELECT id, session_id, role, content, phase, extracted_data, suggested_questions, created_at
		FROM builder_messages
		WHERE session_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.pool.Query(ctx, messagesQuery, id)
	if err != nil {
		return nil, fmt.Errorf("get builder messages: %w", err)
	}
	defer rows.Close()

	var messages []domain.BuilderMessage
	for rows.Next() {
		var msg domain.BuilderMessage
		var phase sql.NullString
		var extractedData, suggestedQuestions []byte

		err := rows.Scan(
			&msg.ID,
			&msg.SessionID,
			&msg.Role,
			&msg.Content,
			&phase,
			&extractedData,
			&suggestedQuestions,
			&msg.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan builder message: %w", err)
		}

		if phase.Valid {
			p := domain.HearingPhase(phase.String)
			msg.Phase = &p
		}
		if extractedData != nil {
			msg.ExtractedData = json.RawMessage(extractedData)
		}
		if suggestedQuestions != nil {
			msg.SuggestedQuestions = json.RawMessage(suggestedQuestions)
		}

		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate builder messages: %w", err)
	}

	session.Messages = messages
	return session, nil
}

// GetActiveByUser retrieves the most recent active session for a user
func (r *BuilderSessionRepository) GetActiveByUser(ctx context.Context, tenantID uuid.UUID, userID string) (*domain.BuilderSession, error) {
	query := `
		SELECT id, tenant_id, user_id, copilot_session_id,
			   status, hearing_phase, hearing_progress,
			   spec, project_id, created_at, updated_at
		FROM builder_sessions
		WHERE tenant_id = $1 AND user_id = $2
		  AND status NOT IN ('completed', 'abandoned')
		ORDER BY created_at DESC
		LIMIT 1
	`

	session := &domain.BuilderSession{}
	var copilotSessionID sql.NullString
	var spec []byte
	var projectID sql.NullString

	err := r.pool.QueryRow(ctx, query, tenantID, userID).Scan(
		&session.ID,
		&session.TenantID,
		&session.UserID,
		&copilotSessionID,
		&session.Status,
		&session.HearingPhase,
		&session.HearingProgress,
		&spec,
		&projectID,
		&session.CreatedAt,
		&session.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // No active session
		}
		return nil, fmt.Errorf("get active builder session: %w", err)
	}

	if copilotSessionID.Valid {
		id, _ := uuid.Parse(copilotSessionID.String)
		session.CopilotSessionID = &id
	}
	if spec != nil {
		session.Spec = json.RawMessage(spec)
	}
	if projectID.Valid {
		id, _ := uuid.Parse(projectID.String)
		session.ProjectID = &id
	}

	return session, nil
}

// ListByUser retrieves all sessions for a user
func (r *BuilderSessionRepository) ListByUser(ctx context.Context, tenantID uuid.UUID, userID string, filter repository.BuilderSessionFilter) ([]*domain.BuilderSession, int, error) {
	// Count query
	countQuery := `
		SELECT COUNT(*)
		FROM builder_sessions
		WHERE tenant_id = $1 AND user_id = $2
	`
	countArgs := []interface{}{tenantID, userID}
	argIndex := 3

	if filter.Status != nil {
		countQuery += fmt.Sprintf(" AND status = $%d", argIndex)
		countArgs = append(countArgs, *filter.Status)
		argIndex++
	}

	var total int
	err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count builder sessions: %w", err)
	}

	// List query
	query := `
		SELECT id, tenant_id, user_id, copilot_session_id,
			   status, hearing_phase, hearing_progress,
			   spec, project_id, created_at, updated_at
		FROM builder_sessions
		WHERE tenant_id = $1 AND user_id = $2
	`
	args := []interface{}{tenantID, userID}
	argIndex = 3

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *filter.Status)
		argIndex++
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++
	}
	if filter.Page > 0 && filter.Limit > 0 {
		offset := (filter.Page - 1) * filter.Limit
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, offset)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list builder sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*domain.BuilderSession
	for rows.Next() {
		session := &domain.BuilderSession{}
		var copilotSessionID sql.NullString
		var spec []byte
		var projectID sql.NullString

		err := rows.Scan(
			&session.ID,
			&session.TenantID,
			&session.UserID,
			&copilotSessionID,
			&session.Status,
			&session.HearingPhase,
			&session.HearingProgress,
			&spec,
			&projectID,
			&session.CreatedAt,
			&session.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan builder session: %w", err)
		}

		if copilotSessionID.Valid {
			id, _ := uuid.Parse(copilotSessionID.String)
			session.CopilotSessionID = &id
		}
		if spec != nil {
			session.Spec = json.RawMessage(spec)
		}
		if projectID.Valid {
			id, _ := uuid.Parse(projectID.String)
			session.ProjectID = &id
		}

		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate builder sessions: %w", err)
	}

	return sessions, total, nil
}

// Update updates a builder session
func (r *BuilderSessionRepository) Update(ctx context.Context, session *domain.BuilderSession) error {
	query := `
		UPDATE builder_sessions
		SET status = $1, hearing_phase = $2, hearing_progress = $3,
			spec = $4, project_id = $5, updated_at = $6
		WHERE id = $7 AND tenant_id = $8
	`

	result, err := r.pool.Exec(ctx, query,
		session.Status,
		session.HearingPhase,
		session.HearingProgress,
		session.Spec,
		session.ProjectID,
		session.UpdatedAt,
		session.ID,
		session.TenantID,
	)
	if err != nil {
		return fmt.Errorf("update builder session: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("builder session not found")
	}

	return nil
}

// AddMessage adds a message to a session
func (r *BuilderSessionRepository) AddMessage(ctx context.Context, message *domain.BuilderMessage) error {
	query := `
		INSERT INTO builder_messages (
			id, session_id, role, content, phase, extracted_data, suggested_questions, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	var phase *string
	if message.Phase != nil {
		p := string(*message.Phase)
		phase = &p
	}

	_, err := r.pool.Exec(ctx, query,
		message.ID,
		message.SessionID,
		message.Role,
		message.Content,
		phase,
		message.ExtractedData,
		message.SuggestedQuestions,
		message.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("add builder message: %w", err)
	}

	return nil
}

// UpdateStatus updates the session status
func (r *BuilderSessionRepository) UpdateStatus(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, status domain.BuilderSessionStatus) error {
	query := `
		UPDATE builder_sessions
		SET status = $1, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3
	`

	result, err := r.pool.Exec(ctx, query, status, id, tenantID)
	if err != nil {
		return fmt.Errorf("update builder session status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("builder session not found")
	}

	return nil
}

// UpdatePhase updates the hearing phase and progress
func (r *BuilderSessionRepository) UpdatePhase(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, phase domain.HearingPhase, progress int) error {
	query := `
		UPDATE builder_sessions
		SET hearing_phase = $1, hearing_progress = $2, updated_at = NOW()
		WHERE id = $3 AND tenant_id = $4
	`

	result, err := r.pool.Exec(ctx, query, phase, progress, id, tenantID)
	if err != nil {
		return fmt.Errorf("update builder session phase: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("builder session not found")
	}

	return nil
}

// SetSpec sets the workflow spec
func (r *BuilderSessionRepository) SetSpec(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, spec []byte) error {
	query := `
		UPDATE builder_sessions
		SET spec = $1, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3
	`

	result, err := r.pool.Exec(ctx, query, spec, id, tenantID)
	if err != nil {
		return fmt.Errorf("set builder session spec: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("builder session not found")
	}

	return nil
}

// SetProjectID sets the generated project ID
func (r *BuilderSessionRepository) SetProjectID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, projectID uuid.UUID) error {
	query := `
		UPDATE builder_sessions
		SET project_id = $1, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3
	`

	result, err := r.pool.Exec(ctx, query, projectID, id, tenantID)
	if err != nil {
		return fmt.Errorf("set builder session project_id: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("builder session not found")
	}

	return nil
}

// Delete deletes a builder session
func (r *BuilderSessionRepository) Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
	query := `
		DELETE FROM builder_sessions
		WHERE id = $1 AND tenant_id = $2
	`

	result, err := r.pool.Exec(ctx, query, id, tenantID)
	if err != nil {
		return fmt.Errorf("delete builder session: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("builder session not found")
	}

	return nil
}
