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
)

// CredentialShareRepository implements repository.CredentialShareRepository
type CredentialShareRepository struct {
	pool *pgxpool.Pool
}

// NewCredentialShareRepository creates a new CredentialShareRepository
func NewCredentialShareRepository(pool *pgxpool.Pool) *CredentialShareRepository {
	return &CredentialShareRepository{pool: pool}
}

func (r *CredentialShareRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.CredentialShare, error) {
	query := `
		SELECT id, credential_id, shared_with_user_id, shared_with_project_id,
			   permission, shared_by_user_id, note, created_at, expires_at
		FROM credential_shares
		WHERE id = $1
	`

	var share domain.CredentialShare
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&share.ID,
		&share.CredentialID,
		&share.SharedWithUserID,
		&share.SharedWithProjectID,
		&share.Permission,
		&share.SharedByUserID,
		&share.Note,
		&share.CreatedAt,
		&share.ExpiresAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrCredentialShareNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get credential share by ID: %w", err)
	}

	return &share, nil
}

func (r *CredentialShareRepository) ListByCredential(ctx context.Context, credentialID uuid.UUID) ([]*domain.CredentialShare, error) {
	query := `
		SELECT id, credential_id, shared_with_user_id, shared_with_project_id,
			   permission, shared_by_user_id, note, created_at, expires_at
		FROM credential_shares
		WHERE credential_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, credentialID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shares []*domain.CredentialShare
	for rows.Next() {
		var share domain.CredentialShare
		err := rows.Scan(
			&share.ID,
			&share.CredentialID,
			&share.SharedWithUserID,
			&share.SharedWithProjectID,
			&share.Permission,
			&share.SharedByUserID,
			&share.Note,
			&share.CreatedAt,
			&share.ExpiresAt,
		)
		if err != nil {
			return nil, err
		}
		shares = append(shares, &share)
	}

	return shares, nil
}

func (r *CredentialShareRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.CredentialShare, error) {
	query := `
		SELECT id, credential_id, shared_with_user_id, shared_with_project_id,
			   permission, shared_by_user_id, note, created_at, expires_at
		FROM credential_shares
		WHERE shared_with_user_id = $1
		  AND (expires_at IS NULL OR expires_at > $2)
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID, time.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shares []*domain.CredentialShare
	for rows.Next() {
		var share domain.CredentialShare
		err := rows.Scan(
			&share.ID,
			&share.CredentialID,
			&share.SharedWithUserID,
			&share.SharedWithProjectID,
			&share.Permission,
			&share.SharedByUserID,
			&share.Note,
			&share.CreatedAt,
			&share.ExpiresAt,
		)
		if err != nil {
			return nil, err
		}
		shares = append(shares, &share)
	}

	return shares, nil
}

func (r *CredentialShareRepository) ListByProject(ctx context.Context, projectID uuid.UUID) ([]*domain.CredentialShare, error) {
	query := `
		SELECT id, credential_id, shared_with_user_id, shared_with_project_id,
			   permission, shared_by_user_id, note, created_at, expires_at
		FROM credential_shares
		WHERE shared_with_project_id = $1
		  AND (expires_at IS NULL OR expires_at > $2)
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, projectID, time.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shares []*domain.CredentialShare
	for rows.Next() {
		var share domain.CredentialShare
		err := rows.Scan(
			&share.ID,
			&share.CredentialID,
			&share.SharedWithUserID,
			&share.SharedWithProjectID,
			&share.Permission,
			&share.SharedByUserID,
			&share.Note,
			&share.CreatedAt,
			&share.ExpiresAt,
		)
		if err != nil {
			return nil, err
		}
		shares = append(shares, &share)
	}

	return shares, nil
}

func (r *CredentialShareRepository) GetByCredentialAndUser(ctx context.Context, credentialID, userID uuid.UUID) (*domain.CredentialShare, error) {
	query := `
		SELECT id, credential_id, shared_with_user_id, shared_with_project_id,
			   permission, shared_by_user_id, note, created_at, expires_at
		FROM credential_shares
		WHERE credential_id = $1 AND shared_with_user_id = $2
	`

	var share domain.CredentialShare
	err := r.pool.QueryRow(ctx, query, credentialID, userID).Scan(
		&share.ID,
		&share.CredentialID,
		&share.SharedWithUserID,
		&share.SharedWithProjectID,
		&share.Permission,
		&share.SharedByUserID,
		&share.Note,
		&share.CreatedAt,
		&share.ExpiresAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrCredentialShareNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get credential share by credential and user: %w", err)
	}

	return &share, nil
}

func (r *CredentialShareRepository) GetByCredentialAndProject(ctx context.Context, credentialID, projectID uuid.UUID) (*domain.CredentialShare, error) {
	query := `
		SELECT id, credential_id, shared_with_user_id, shared_with_project_id,
			   permission, shared_by_user_id, note, created_at, expires_at
		FROM credential_shares
		WHERE credential_id = $1 AND shared_with_project_id = $2
	`

	var share domain.CredentialShare
	err := r.pool.QueryRow(ctx, query, credentialID, projectID).Scan(
		&share.ID,
		&share.CredentialID,
		&share.SharedWithUserID,
		&share.SharedWithProjectID,
		&share.Permission,
		&share.SharedByUserID,
		&share.Note,
		&share.CreatedAt,
		&share.ExpiresAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrCredentialShareNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get credential share by credential and project: %w", err)
	}

	return &share, nil
}

func (r *CredentialShareRepository) Create(ctx context.Context, share *domain.CredentialShare) error {
	query := `
		INSERT INTO credential_shares (
			id, credential_id, shared_with_user_id, shared_with_project_id,
			permission, shared_by_user_id, note, created_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.pool.Exec(ctx, query,
		share.ID,
		share.CredentialID,
		share.SharedWithUserID,
		share.SharedWithProjectID,
		share.Permission,
		share.SharedByUserID,
		share.Note,
		share.CreatedAt,
		share.ExpiresAt,
	)

	return err
}

func (r *CredentialShareRepository) Update(ctx context.Context, share *domain.CredentialShare) error {
	query := `
		UPDATE credential_shares SET
			permission = $2,
			note = $3,
			expires_at = $4
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query,
		share.ID,
		share.Permission,
		share.Note,
		share.ExpiresAt,
	)

	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrCredentialShareNotFound
	}

	return nil
}

func (r *CredentialShareRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM credential_shares WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrCredentialShareNotFound
	}

	return nil
}

func (r *CredentialShareRepository) DeleteExpired(ctx context.Context) (int, error) {
	query := `DELETE FROM credential_shares WHERE expires_at IS NOT NULL AND expires_at <= $1`

	result, err := r.pool.Exec(ctx, query, time.Now())
	if err != nil {
		return 0, err
	}

	return int(result.RowsAffected()), nil
}
