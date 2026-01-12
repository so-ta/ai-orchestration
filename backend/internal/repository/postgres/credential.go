package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// CredentialRepository implements repository.CredentialRepository
type CredentialRepository struct {
	pool *pgxpool.Pool
}

// NewCredentialRepository creates a new CredentialRepository
func NewCredentialRepository(pool *pgxpool.Pool) *CredentialRepository {
	return &CredentialRepository{pool: pool}
}

func (r *CredentialRepository) Create(ctx context.Context, credential *domain.Credential) error {
	query := `
		INSERT INTO credentials (
			id, tenant_id, name, description, credential_type,
			encrypted_data, encrypted_dek, data_nonce, dek_nonce,
			metadata, expires_at, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	_, err := r.pool.Exec(ctx, query,
		credential.ID,
		credential.TenantID,
		credential.Name,
		credential.Description,
		credential.CredentialType,
		credential.EncryptedData,
		credential.EncryptedDEK,
		credential.DataNonce,
		credential.DEKNonce,
		credential.Metadata,
		credential.ExpiresAt,
		credential.Status,
		credential.CreatedAt,
		credential.UpdatedAt,
	)

	return err
}

func (r *CredentialRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Credential, error) {
	query := `
		SELECT id, tenant_id, name, description, credential_type,
			   encrypted_data, encrypted_dek, data_nonce, dek_nonce,
			   metadata, expires_at, status, created_at, updated_at
		FROM credentials
		WHERE tenant_id = $1 AND id = $2
	`

	var cred domain.Credential
	err := r.pool.QueryRow(ctx, query, tenantID, id).Scan(
		&cred.ID,
		&cred.TenantID,
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

	if err == pgx.ErrNoRows {
		return nil, domain.ErrCredentialNotFound
	}
	if err != nil {
		return nil, err
	}

	return &cred, nil
}

func (r *CredentialRepository) GetByName(ctx context.Context, tenantID uuid.UUID, name string) (*domain.Credential, error) {
	query := `
		SELECT id, tenant_id, name, description, credential_type,
			   encrypted_data, encrypted_dek, data_nonce, dek_nonce,
			   metadata, expires_at, status, created_at, updated_at
		FROM credentials
		WHERE tenant_id = $1 AND name = $2
	`

	var cred domain.Credential
	err := r.pool.QueryRow(ctx, query, tenantID, name).Scan(
		&cred.ID,
		&cred.TenantID,
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

	if err == pgx.ErrNoRows {
		return nil, domain.ErrCredentialNotFound
	}
	if err != nil {
		return nil, err
	}

	return &cred, nil
}

func (r *CredentialRepository) List(ctx context.Context, tenantID uuid.UUID, filter repository.CredentialFilter) ([]*domain.Credential, int, error) {
	// Count query
	countQuery := `SELECT COUNT(*) FROM credentials WHERE tenant_id = $1`
	countArgs := []interface{}{tenantID}
	argIndex := 2

	if filter.CredentialType != nil {
		countQuery += fmt.Sprintf(` AND credential_type = $%d`, argIndex)
		countArgs = append(countArgs, *filter.CredentialType)
		argIndex++
	}
	if filter.Status != nil {
		countQuery += fmt.Sprintf(` AND status = $%d`, argIndex)
		countArgs = append(countArgs, *filter.Status)
	}

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// List query
	query := `
		SELECT id, tenant_id, name, description, credential_type,
			   encrypted_data, encrypted_dek, data_nonce, dek_nonce,
			   metadata, expires_at, status, created_at, updated_at
		FROM credentials
		WHERE tenant_id = $1
	`
	args := []interface{}{tenantID}
	argIdx := 2

	if filter.CredentialType != nil {
		query += fmt.Sprintf(` AND credential_type = $%d`, argIdx)
		args = append(args, *filter.CredentialType)
		argIdx++
	}
	if filter.Status != nil {
		query += fmt.Sprintf(` AND status = $%d`, argIdx)
		args = append(args, *filter.Status)
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

	var credentials []*domain.Credential
	for rows.Next() {
		var cred domain.Credential
		err := rows.Scan(
			&cred.ID,
			&cred.TenantID,
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
			return nil, 0, err
		}
		credentials = append(credentials, &cred)
	}

	return credentials, total, nil
}

func (r *CredentialRepository) Update(ctx context.Context, credential *domain.Credential) error {
	query := `
		UPDATE credentials SET
			name = $3,
			description = $4,
			credential_type = $5,
			encrypted_data = $6,
			encrypted_dek = $7,
			data_nonce = $8,
			dek_nonce = $9,
			metadata = $10,
			expires_at = $11,
			status = $12,
			updated_at = $13
		WHERE tenant_id = $1 AND id = $2
	`

	result, err := r.pool.Exec(ctx, query,
		credential.TenantID,
		credential.ID,
		credential.Name,
		credential.Description,
		credential.CredentialType,
		credential.EncryptedData,
		credential.EncryptedDEK,
		credential.DataNonce,
		credential.DEKNonce,
		credential.Metadata,
		credential.ExpiresAt,
		credential.Status,
		credential.UpdatedAt,
	)

	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrCredentialNotFound
	}

	return nil
}

func (r *CredentialRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	query := `DELETE FROM credentials WHERE tenant_id = $1 AND id = $2`

	result, err := r.pool.Exec(ctx, query, tenantID, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrCredentialNotFound
	}

	return nil
}

func (r *CredentialRepository) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.CredentialStatus) error {
	query := `
		UPDATE credentials SET
			status = $3,
			updated_at = $4
		WHERE tenant_id = $1 AND id = $2
	`

	result, err := r.pool.Exec(ctx, query, tenantID, id, status, time.Now().UTC())
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrCredentialNotFound
	}

	return nil
}
