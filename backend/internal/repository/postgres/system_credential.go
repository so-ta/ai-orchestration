package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/souta/ai-orchestration/internal/domain"
)

// SystemCredentialRepository implements repository.SystemCredentialRepository
type SystemCredentialRepository struct {
	pool *pgxpool.Pool
}

// NewSystemCredentialRepository creates a new SystemCredentialRepository
func NewSystemCredentialRepository(pool *pgxpool.Pool) *SystemCredentialRepository {
	return &SystemCredentialRepository{pool: pool}
}

// Create creates a new system credential
func (r *SystemCredentialRepository) Create(ctx context.Context, cred *domain.SystemCredential) error {
	query := `
		INSERT INTO system_credentials (
			id, name, description, credential_type,
			encrypted_data, encrypted_dek, data_nonce, dek_nonce,
			metadata, expires_at, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := r.pool.Exec(ctx, query,
		cred.ID,
		cred.Name,
		cred.Description,
		cred.CredentialType,
		cred.EncryptedData,
		cred.EncryptedDEK,
		cred.DataNonce,
		cred.DEKNonce,
		cred.Metadata,
		cred.ExpiresAt,
		cred.Status,
		cred.CreatedAt,
		cred.UpdatedAt,
	)
	return err
}

// GetByID retrieves a system credential by ID
func (r *SystemCredentialRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.SystemCredential, error) {
	query := `
		SELECT id, name, description, credential_type,
			encrypted_data, encrypted_dek, data_nonce, dek_nonce,
			metadata, expires_at, status, created_at, updated_at
		FROM system_credentials
		WHERE id = $1
	`

	cred := &domain.SystemCredential{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&cred.ID,
		&cred.Name,
		&cred.Description,
		&cred.CredentialType,
		&cred.EncryptedData,
		&cred.EncryptedDEK,
		&cred.DataNonce,
		&cred.DEKNonce,
		&cred.Metadata,
		&cred.ExpiresAt,
		&cred.Status,
		&cred.CreatedAt,
		&cred.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrSystemCredentialNotFound
	}
	if err != nil {
		return nil, err
	}
	return cred, nil
}

// GetByName retrieves a system credential by name
func (r *SystemCredentialRepository) GetByName(ctx context.Context, name string) (*domain.SystemCredential, error) {
	query := `
		SELECT id, name, description, credential_type,
			encrypted_data, encrypted_dek, data_nonce, dek_nonce,
			metadata, expires_at, status, created_at, updated_at
		FROM system_credentials
		WHERE name = $1
	`

	cred := &domain.SystemCredential{}
	err := r.pool.QueryRow(ctx, query, name).Scan(
		&cred.ID,
		&cred.Name,
		&cred.Description,
		&cred.CredentialType,
		&cred.EncryptedData,
		&cred.EncryptedDEK,
		&cred.DataNonce,
		&cred.DEKNonce,
		&cred.Metadata,
		&cred.ExpiresAt,
		&cred.Status,
		&cred.CreatedAt,
		&cred.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrSystemCredentialNotFound
	}
	if err != nil {
		return nil, err
	}
	return cred, nil
}

// List lists all system credentials
func (r *SystemCredentialRepository) List(ctx context.Context) ([]*domain.SystemCredential, error) {
	query := `
		SELECT id, name, description, credential_type,
			encrypted_data, encrypted_dek, data_nonce, dek_nonce,
			metadata, expires_at, status, created_at, updated_at
		FROM system_credentials
		ORDER BY name
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var creds []*domain.SystemCredential
	for rows.Next() {
		cred := &domain.SystemCredential{}
		err := rows.Scan(
			&cred.ID,
			&cred.Name,
			&cred.Description,
			&cred.CredentialType,
			&cred.EncryptedData,
			&cred.EncryptedDEK,
			&cred.DataNonce,
			&cred.DEKNonce,
			&cred.Metadata,
			&cred.ExpiresAt,
			&cred.Status,
			&cred.CreatedAt,
			&cred.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		creds = append(creds, cred)
	}

	return creds, rows.Err()
}

// ListByType lists system credentials by type
func (r *SystemCredentialRepository) ListByType(ctx context.Context, credType domain.CredentialType) ([]*domain.SystemCredential, error) {
	query := `
		SELECT id, name, description, credential_type,
			encrypted_data, encrypted_dek, data_nonce, dek_nonce,
			metadata, expires_at, status, created_at, updated_at
		FROM system_credentials
		WHERE credential_type = $1
		ORDER BY name
	`

	rows, err := r.pool.Query(ctx, query, credType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var creds []*domain.SystemCredential
	for rows.Next() {
		cred := &domain.SystemCredential{}
		err := rows.Scan(
			&cred.ID,
			&cred.Name,
			&cred.Description,
			&cred.CredentialType,
			&cred.EncryptedData,
			&cred.EncryptedDEK,
			&cred.DataNonce,
			&cred.DEKNonce,
			&cred.Metadata,
			&cred.ExpiresAt,
			&cred.Status,
			&cred.CreatedAt,
			&cred.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		creds = append(creds, cred)
	}

	return creds, rows.Err()
}

// Update updates a system credential
func (r *SystemCredentialRepository) Update(ctx context.Context, cred *domain.SystemCredential) error {
	query := `
		UPDATE system_credentials
		SET name = $2, description = $3, credential_type = $4,
			encrypted_data = $5, encrypted_dek = $6, data_nonce = $7, dek_nonce = $8,
			metadata = $9, expires_at = $10, status = $11, updated_at = $12
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query,
		cred.ID,
		cred.Name,
		cred.Description,
		cred.CredentialType,
		cred.EncryptedData,
		cred.EncryptedDEK,
		cred.DataNonce,
		cred.DEKNonce,
		cred.Metadata,
		cred.ExpiresAt,
		cred.Status,
		cred.UpdatedAt,
	)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return domain.ErrSystemCredentialNotFound
	}
	return nil
}

// Delete deletes a system credential
func (r *SystemCredentialRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM system_credentials WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return domain.ErrSystemCredentialNotFound
	}
	return nil
}

// UpdateStatus updates the status of a system credential
func (r *SystemCredentialRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.CredentialStatus) error {
	query := `
		UPDATE system_credentials
		SET status = $2, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, id, status)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return domain.ErrSystemCredentialNotFound
	}
	return nil
}
