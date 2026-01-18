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
			scope, project_id, owner_user_id,
			encrypted_data, encrypted_dek, data_nonce, dek_nonce,
			metadata, expires_at, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`

	_, err := r.pool.Exec(ctx, query,
		credential.ID,
		credential.TenantID,
		credential.Name,
		credential.Description,
		credential.CredentialType,
		credential.Scope,
		credential.ProjectID,
		credential.OwnerUserID,
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
			   scope, project_id, owner_user_id,
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
		&cred.Scope,
		&cred.ProjectID,
		&cred.OwnerUserID,
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
		return nil, domain.ErrCredentialNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get credential by ID: %w", err)
	}

	return &cred, nil
}

func (r *CredentialRepository) GetByName(ctx context.Context, tenantID uuid.UUID, name string) (*domain.Credential, error) {
	query := `
		SELECT id, tenant_id, name, description, credential_type,
			   scope, project_id, owner_user_id,
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
		&cred.Scope,
		&cred.ProjectID,
		&cred.OwnerUserID,
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
		return nil, domain.ErrCredentialNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get credential by name: %w", err)
	}

	return &cred, nil
}

func (r *CredentialRepository) List(ctx context.Context, tenantID uuid.UUID, filter repository.CredentialFilter) ([]*domain.Credential, int, error) {
	// Build WHERE conditions
	conditions := []string{"tenant_id = $1"}
	args := []interface{}{tenantID}
	argIdx := 2

	if filter.CredentialType != nil {
		conditions = append(conditions, fmt.Sprintf("credential_type = $%d", argIdx))
		args = append(args, *filter.CredentialType)
		argIdx++
	}
	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}
	if filter.Scope != nil {
		conditions = append(conditions, fmt.Sprintf("scope = $%d", argIdx))
		args = append(args, *filter.Scope)
		argIdx++
	}
	if filter.ProjectID != nil {
		conditions = append(conditions, fmt.Sprintf("project_id = $%d", argIdx))
		args = append(args, *filter.ProjectID)
		argIdx++
	}
	if filter.OwnerUserID != nil {
		conditions = append(conditions, fmt.Sprintf("owner_user_id = $%d", argIdx))
		args = append(args, *filter.OwnerUserID)
		argIdx++
	}

	whereClause := " WHERE " + conditions[0]
	for i := 1; i < len(conditions); i++ {
		whereClause += " AND " + conditions[i]
	}

	// Count query
	countQuery := "SELECT COUNT(*) FROM credentials" + whereClause
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// List query
	query := `
		SELECT id, tenant_id, name, description, credential_type,
			   scope, project_id, owner_user_id,
			   encrypted_data, encrypted_dek, data_nonce, dek_nonce,
			   metadata, expires_at, status, created_at, updated_at
		FROM credentials` + whereClause

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
			&cred.Scope,
			&cred.ProjectID,
			&cred.OwnerUserID,
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
			scope = $6,
			project_id = $7,
			owner_user_id = $8,
			encrypted_data = $9,
			encrypted_dek = $10,
			data_nonce = $11,
			dek_nonce = $12,
			metadata = $13,
			expires_at = $14,
			status = $15,
			updated_at = $16
		WHERE tenant_id = $1 AND id = $2
	`

	result, err := r.pool.Exec(ctx, query,
		credential.TenantID,
		credential.ID,
		credential.Name,
		credential.Description,
		credential.CredentialType,
		credential.Scope,
		credential.ProjectID,
		credential.OwnerUserID,
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
