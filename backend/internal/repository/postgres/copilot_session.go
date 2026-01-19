package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
		INSERT INTO copilot_sessions (
			id, tenant_id, user_id, context_project_id, mode, title,
			status, hearing_phase, hearing_progress,
			spec, project_id, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := r.pool.Exec(ctx, query,
		session.ID,
		session.TenantID,
		session.UserID,
		session.ContextProjectID,
		session.Mode,
		session.Title,
		session.Status,
		session.HearingPhase,
		session.HearingProgress,
		session.Spec,
		session.ProjectID,
		session.CreatedAt,
		session.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create copilot session: %w", err)
	}

	return nil
}

// GetByID retrieves a copilot session by ID
func (r *CopilotSessionRepository) GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.CopilotSession, error) {
	query := `
		SELECT id, tenant_id, user_id, context_project_id, mode, title,
			   status, hearing_phase, hearing_progress,
			   spec, project_id, created_at, updated_at
		FROM copilot_sessions
		WHERE id = $1 AND tenant_id = $2
	`

	session := &domain.CopilotSession{}
	var contextProjectID sql.NullString
	var title sql.NullString
	var spec []byte
	var projectID sql.NullString

	err := r.pool.QueryRow(ctx, query, id, tenantID).Scan(
		&session.ID,
		&session.TenantID,
		&session.UserID,
		&contextProjectID,
		&session.Mode,
		&title,
		&session.Status,
		&session.HearingPhase,
		&session.HearingProgress,
		&spec,
		&projectID,
		&session.CreatedAt,
		&session.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("copilot session not found: %w", err)
		}
		return nil, fmt.Errorf("get copilot session: %w", err)
	}

	if contextProjectID.Valid {
		id, _ := uuid.Parse(contextProjectID.String)
		session.ContextProjectID = &id
	}
	if title.Valid {
		session.Title = title.String
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
func (r *CopilotSessionRepository) GetWithMessages(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.CopilotSession, error) {
	session, err := r.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	// Get messages
	messagesQuery := `
		SELECT id, session_id, role, content, phase, extracted_data, suggested_questions, created_at
		FROM copilot_messages
		WHERE session_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.pool.Query(ctx, messagesQuery, id)
	if err != nil {
		return nil, fmt.Errorf("get copilot messages: %w", err)
	}
	defer rows.Close()

	var messages []domain.CopilotMessage
	for rows.Next() {
		var msg domain.CopilotMessage
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
			return nil, fmt.Errorf("scan copilot message: %w", err)
		}

		if phase.Valid {
			p := domain.CopilotPhase(phase.String)
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
		return nil, fmt.Errorf("iterate copilot messages: %w", err)
	}

	session.Messages = messages
	return session, nil
}

// GetActiveByUser retrieves the most recent active session for a user (global, no project context)
func (r *CopilotSessionRepository) GetActiveByUser(ctx context.Context, tenantID uuid.UUID, userID string) (*domain.CopilotSession, error) {
	query := `
		SELECT id, tenant_id, user_id, context_project_id, mode, title,
			   status, hearing_phase, hearing_progress,
			   spec, project_id, created_at, updated_at
		FROM copilot_sessions
		WHERE tenant_id = $1 AND user_id = $2
		  AND status NOT IN ('completed', 'abandoned')
		ORDER BY created_at DESC
		LIMIT 1
	`

	session := &domain.CopilotSession{}
	var contextProjectID sql.NullString
	var title sql.NullString
	var spec []byte
	var projectID sql.NullString

	err := r.pool.QueryRow(ctx, query, tenantID, userID).Scan(
		&session.ID,
		&session.TenantID,
		&session.UserID,
		&contextProjectID,
		&session.Mode,
		&title,
		&session.Status,
		&session.HearingPhase,
		&session.HearingProgress,
		&spec,
		&projectID,
		&session.CreatedAt,
		&session.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // No active session
		}
		return nil, fmt.Errorf("get active copilot session: %w", err)
	}

	if contextProjectID.Valid {
		id, _ := uuid.Parse(contextProjectID.String)
		session.ContextProjectID = &id
	}
	if title.Valid {
		session.Title = title.String
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

// GetActiveByUserAndProject retrieves the active session for a user and project
func (r *CopilotSessionRepository) GetActiveByUserAndProject(ctx context.Context, tenantID uuid.UUID, userID string, projectID uuid.UUID) (*domain.CopilotSession, error) {
	query := `
		SELECT id, tenant_id, user_id, context_project_id, mode, title,
			   status, hearing_phase, hearing_progress,
			   spec, project_id, created_at, updated_at
		FROM copilot_sessions
		WHERE tenant_id = $1 AND user_id = $2 AND context_project_id = $3
		  AND status NOT IN ('completed', 'abandoned')
		ORDER BY created_at DESC
		LIMIT 1
	`

	session := &domain.CopilotSession{}
	var contextProjectID sql.NullString
	var title sql.NullString
	var spec []byte
	var genProjectID sql.NullString

	err := r.pool.QueryRow(ctx, query, tenantID, userID, projectID).Scan(
		&session.ID,
		&session.TenantID,
		&session.UserID,
		&contextProjectID,
		&session.Mode,
		&title,
		&session.Status,
		&session.HearingPhase,
		&session.HearingProgress,
		&spec,
		&genProjectID,
		&session.CreatedAt,
		&session.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // No active session
		}
		return nil, fmt.Errorf("get active copilot session by project: %w", err)
	}

	if contextProjectID.Valid {
		id, _ := uuid.Parse(contextProjectID.String)
		session.ContextProjectID = &id
	}
	if title.Valid {
		session.Title = title.String
	}
	if spec != nil {
		session.Spec = json.RawMessage(spec)
	}
	if genProjectID.Valid {
		id, _ := uuid.Parse(genProjectID.String)
		session.ProjectID = &id
	}

	return session, nil
}

// ListByUser retrieves all sessions for a user (global, no project context)
func (r *CopilotSessionRepository) ListByUser(ctx context.Context, tenantID uuid.UUID, userID string, filter repository.CopilotSessionFilter) ([]*domain.CopilotSession, int, error) {
	// Count query
	countQuery := `
		SELECT COUNT(*)
		FROM copilot_sessions
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
		return nil, 0, fmt.Errorf("count copilot sessions: %w", err)
	}

	// List query
	query := `
		SELECT id, tenant_id, user_id, context_project_id, mode, title,
			   status, hearing_phase, hearing_progress,
			   spec, project_id, created_at, updated_at
		FROM copilot_sessions
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
		return nil, 0, fmt.Errorf("list copilot sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*domain.CopilotSession
	for rows.Next() {
		session := &domain.CopilotSession{}
		var contextProjectID sql.NullString
		var title sql.NullString
		var spec []byte
		var projectID sql.NullString

		err := rows.Scan(
			&session.ID,
			&session.TenantID,
			&session.UserID,
			&contextProjectID,
			&session.Mode,
			&title,
			&session.Status,
			&session.HearingPhase,
			&session.HearingProgress,
			&spec,
			&projectID,
			&session.CreatedAt,
			&session.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan copilot session: %w", err)
		}

		if contextProjectID.Valid {
			id, _ := uuid.Parse(contextProjectID.String)
			session.ContextProjectID = &id
		}
		if title.Valid {
			session.Title = title.String
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
		return nil, 0, fmt.Errorf("iterate copilot sessions: %w", err)
	}

	return sessions, total, nil
}

// ListByUserAndProject retrieves all sessions for a user and project
func (r *CopilotSessionRepository) ListByUserAndProject(ctx context.Context, tenantID uuid.UUID, userID string, projectID uuid.UUID) ([]*domain.CopilotSession, error) {
	query := `
		SELECT id, tenant_id, user_id, context_project_id, mode, title,
			   status, hearing_phase, hearing_progress,
			   spec, project_id, created_at, updated_at
		FROM copilot_sessions
		WHERE tenant_id = $1 AND user_id = $2 AND context_project_id = $3
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, tenantID, userID, projectID)
	if err != nil {
		return nil, fmt.Errorf("list copilot sessions by project: %w", err)
	}
	defer rows.Close()

	var sessions []*domain.CopilotSession
	for rows.Next() {
		session := &domain.CopilotSession{}
		var contextProjectID sql.NullString
		var title sql.NullString
		var spec []byte
		var genProjectID sql.NullString

		err := rows.Scan(
			&session.ID,
			&session.TenantID,
			&session.UserID,
			&contextProjectID,
			&session.Mode,
			&title,
			&session.Status,
			&session.HearingPhase,
			&session.HearingProgress,
			&spec,
			&genProjectID,
			&session.CreatedAt,
			&session.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan copilot session: %w", err)
		}

		if contextProjectID.Valid {
			id, _ := uuid.Parse(contextProjectID.String)
			session.ContextProjectID = &id
		}
		if title.Valid {
			session.Title = title.String
		}
		if spec != nil {
			session.Spec = json.RawMessage(spec)
		}
		if genProjectID.Valid {
			id, _ := uuid.Parse(genProjectID.String)
			session.ProjectID = &id
		}

		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate copilot sessions: %w", err)
	}

	return sessions, nil
}

// Update updates a copilot session
func (r *CopilotSessionRepository) Update(ctx context.Context, session *domain.CopilotSession) error {
	query := `
		UPDATE copilot_sessions
		SET title = $1, status = $2, hearing_phase = $3, hearing_progress = $4,
			spec = $5, project_id = $6, updated_at = $7
		WHERE id = $8 AND tenant_id = $9
	`

	result, err := r.pool.Exec(ctx, query,
		session.Title,
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
		return fmt.Errorf("update copilot session: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("copilot session not found")
	}

	return nil
}

// AddMessage adds a message to a session
func (r *CopilotSessionRepository) AddMessage(ctx context.Context, message *domain.CopilotMessage) error {
	query := `
		INSERT INTO copilot_messages (
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
		return fmt.Errorf("add copilot message: %w", err)
	}

	return nil
}

// UpdateStatus updates the session status
func (r *CopilotSessionRepository) UpdateStatus(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, status domain.CopilotSessionStatus) error {
	query := `
		UPDATE copilot_sessions
		SET status = $1, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3
	`

	result, err := r.pool.Exec(ctx, query, status, id, tenantID)
	if err != nil {
		return fmt.Errorf("update copilot session status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("copilot session not found")
	}

	return nil
}

// UpdatePhase updates the hearing phase and progress
func (r *CopilotSessionRepository) UpdatePhase(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, phase domain.CopilotPhase, progress int) error {
	query := `
		UPDATE copilot_sessions
		SET hearing_phase = $1, hearing_progress = $2, updated_at = NOW()
		WHERE id = $3 AND tenant_id = $4
	`

	result, err := r.pool.Exec(ctx, query, phase, progress, id, tenantID)
	if err != nil {
		return fmt.Errorf("update copilot session phase: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("copilot session not found")
	}

	return nil
}

// SetSpec sets the workflow spec
func (r *CopilotSessionRepository) SetSpec(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, spec []byte) error {
	query := `
		UPDATE copilot_sessions
		SET spec = $1, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3
	`

	result, err := r.pool.Exec(ctx, query, spec, id, tenantID)
	if err != nil {
		return fmt.Errorf("set copilot session spec: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("copilot session not found")
	}

	return nil
}

// SetProjectID sets the generated project ID
func (r *CopilotSessionRepository) SetProjectID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, projectID uuid.UUID) error {
	query := `
		UPDATE copilot_sessions
		SET project_id = $1, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3
	`

	result, err := r.pool.Exec(ctx, query, projectID, id, tenantID)
	if err != nil {
		return fmt.Errorf("set copilot session project_id: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("copilot session not found")
	}

	return nil
}

// Delete deletes a copilot session
func (r *CopilotSessionRepository) Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
	query := `
		DELETE FROM copilot_sessions
		WHERE id = $1 AND tenant_id = $2
	`

	result, err := r.pool.Exec(ctx, query, id, tenantID)
	if err != nil {
		return fmt.Errorf("delete copilot session: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("copilot session not found")
	}

	return nil
}
