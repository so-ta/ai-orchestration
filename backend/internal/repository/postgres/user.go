package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/souta/ai-orchestration/internal/domain"
)

// UserRepository implements repository.UserRepository
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository creates a new UserRepository
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, tenant_id, email, COALESCE(name, ''), role,
			   COALESCE(variables, '{}'::jsonb),
			   last_login_at, created_at, updated_at
		FROM users
		WHERE id = $1 AND tenant_id = $2
	`

	var user domain.User
	err := r.pool.QueryRow(ctx, query, id, tenantID).Scan(
		&user.ID,
		&user.TenantID,
		&user.Email,
		&user.Name,
		&user.Role,
		&user.Variables,
		&user.LastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by ID: %w", err)
	}

	return &user, nil
}

// GetVariables retrieves only the variables for a user
// Returns empty map if user not found (for graceful degradation)
func (r *UserRepository) GetVariables(ctx context.Context, tenantID, id uuid.UUID) (map[string]interface{}, error) {
	query := `SELECT COALESCE(variables, '{}'::jsonb) FROM users WHERE id = $1 AND tenant_id = $2`

	var variables []byte
	err := r.pool.QueryRow(ctx, query, id, tenantID).Scan(&variables)
	if errors.Is(err, pgx.ErrNoRows) {
		// Return empty map instead of error for graceful degradation
		return make(map[string]interface{}), nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user variables: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(variables, &result); err != nil {
		return nil, fmt.Errorf("unmarshal user variables: %w", err)
	}

	return result, nil
}

// UpdateVariables updates only the variables for a user
func (r *UserRepository) UpdateVariables(ctx context.Context, tenantID, id uuid.UUID, variables map[string]interface{}) error {
	query := `UPDATE users SET variables = $3, updated_at = $4 WHERE id = $1 AND tenant_id = $2`

	varsJSON, err := json.Marshal(variables)
	if err != nil {
		return fmt.Errorf("marshal user variables: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, id, tenantID, varsJSON, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("update user variables: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}

	return nil
}
