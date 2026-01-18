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

// OAuth2ConnectionRepository implements repository.OAuth2ConnectionRepository
type OAuth2ConnectionRepository struct {
	pool *pgxpool.Pool
}

// NewOAuth2ConnectionRepository creates a new OAuth2ConnectionRepository
func NewOAuth2ConnectionRepository(pool *pgxpool.Pool) *OAuth2ConnectionRepository {
	return &OAuth2ConnectionRepository{pool: pool}
}

func (r *OAuth2ConnectionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.OAuth2Connection, error) {
	query := `
		SELECT id, credential_id, oauth2_app_id,
			   encrypted_access_token, encrypted_refresh_token,
			   access_token_nonce, refresh_token_nonce, token_type,
			   access_token_expires_at, refresh_token_expires_at,
			   state, code_verifier,
			   account_id, account_email, account_name, raw_userinfo,
			   status, last_refresh_at, last_used_at, error_message,
			   created_at, updated_at
		FROM oauth2_connections
		WHERE id = $1
	`

	var conn domain.OAuth2Connection
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&conn.ID,
		&conn.CredentialID,
		&conn.OAuth2AppID,
		&conn.EncryptedAccessToken,
		&conn.EncryptedRefreshToken,
		&conn.AccessTokenNonce,
		&conn.RefreshTokenNonce,
		&conn.TokenType,
		&conn.AccessTokenExpiresAt,
		&conn.RefreshTokenExpiresAt,
		&conn.State,
		&conn.CodeVerifier,
		&conn.AccountID,
		&conn.AccountEmail,
		&conn.AccountName,
		&conn.RawUserinfo,
		&conn.Status,
		&conn.LastRefreshAt,
		&conn.LastUsedAt,
		&conn.ErrorMessage,
		&conn.CreatedAt,
		&conn.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrOAuth2ConnectionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get oauth2 connection by ID: %w", err)
	}

	return &conn, nil
}

func (r *OAuth2ConnectionRepository) GetByCredentialID(ctx context.Context, credentialID uuid.UUID) (*domain.OAuth2Connection, error) {
	query := `
		SELECT id, credential_id, oauth2_app_id,
			   encrypted_access_token, encrypted_refresh_token,
			   access_token_nonce, refresh_token_nonce, token_type,
			   access_token_expires_at, refresh_token_expires_at,
			   state, code_verifier,
			   account_id, account_email, account_name, raw_userinfo,
			   status, last_refresh_at, last_used_at, error_message,
			   created_at, updated_at
		FROM oauth2_connections
		WHERE credential_id = $1
	`

	var conn domain.OAuth2Connection
	err := r.pool.QueryRow(ctx, query, credentialID).Scan(
		&conn.ID,
		&conn.CredentialID,
		&conn.OAuth2AppID,
		&conn.EncryptedAccessToken,
		&conn.EncryptedRefreshToken,
		&conn.AccessTokenNonce,
		&conn.RefreshTokenNonce,
		&conn.TokenType,
		&conn.AccessTokenExpiresAt,
		&conn.RefreshTokenExpiresAt,
		&conn.State,
		&conn.CodeVerifier,
		&conn.AccountID,
		&conn.AccountEmail,
		&conn.AccountName,
		&conn.RawUserinfo,
		&conn.Status,
		&conn.LastRefreshAt,
		&conn.LastUsedAt,
		&conn.ErrorMessage,
		&conn.CreatedAt,
		&conn.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrOAuth2ConnectionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get oauth2 connection by credential ID: %w", err)
	}

	return &conn, nil
}

func (r *OAuth2ConnectionRepository) GetByState(ctx context.Context, state string) (*domain.OAuth2Connection, error) {
	query := `
		SELECT id, credential_id, oauth2_app_id,
			   encrypted_access_token, encrypted_refresh_token,
			   access_token_nonce, refresh_token_nonce, token_type,
			   access_token_expires_at, refresh_token_expires_at,
			   state, code_verifier,
			   account_id, account_email, account_name, raw_userinfo,
			   status, last_refresh_at, last_used_at, error_message,
			   created_at, updated_at
		FROM oauth2_connections
		WHERE state = $1 AND status = 'pending'
	`

	var conn domain.OAuth2Connection
	err := r.pool.QueryRow(ctx, query, state).Scan(
		&conn.ID,
		&conn.CredentialID,
		&conn.OAuth2AppID,
		&conn.EncryptedAccessToken,
		&conn.EncryptedRefreshToken,
		&conn.AccessTokenNonce,
		&conn.RefreshTokenNonce,
		&conn.TokenType,
		&conn.AccessTokenExpiresAt,
		&conn.RefreshTokenExpiresAt,
		&conn.State,
		&conn.CodeVerifier,
		&conn.AccountID,
		&conn.AccountEmail,
		&conn.AccountName,
		&conn.RawUserinfo,
		&conn.Status,
		&conn.LastRefreshAt,
		&conn.LastUsedAt,
		&conn.ErrorMessage,
		&conn.CreatedAt,
		&conn.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrOAuth2InvalidState
	}
	if err != nil {
		return nil, fmt.Errorf("get oauth2 connection by state: %w", err)
	}

	return &conn, nil
}

func (r *OAuth2ConnectionRepository) ListByApp(ctx context.Context, oauth2AppID uuid.UUID) ([]*domain.OAuth2Connection, error) {
	query := `
		SELECT id, credential_id, oauth2_app_id,
			   encrypted_access_token, encrypted_refresh_token,
			   access_token_nonce, refresh_token_nonce, token_type,
			   access_token_expires_at, refresh_token_expires_at,
			   state, code_verifier,
			   account_id, account_email, account_name, raw_userinfo,
			   status, last_refresh_at, last_used_at, error_message,
			   created_at, updated_at
		FROM oauth2_connections
		WHERE oauth2_app_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, oauth2AppID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var connections []*domain.OAuth2Connection
	for rows.Next() {
		var conn domain.OAuth2Connection
		err := rows.Scan(
			&conn.ID,
			&conn.CredentialID,
			&conn.OAuth2AppID,
			&conn.EncryptedAccessToken,
			&conn.EncryptedRefreshToken,
			&conn.AccessTokenNonce,
			&conn.RefreshTokenNonce,
			&conn.TokenType,
			&conn.AccessTokenExpiresAt,
			&conn.RefreshTokenExpiresAt,
			&conn.State,
			&conn.CodeVerifier,
			&conn.AccountID,
			&conn.AccountEmail,
			&conn.AccountName,
			&conn.RawUserinfo,
			&conn.Status,
			&conn.LastRefreshAt,
			&conn.LastUsedAt,
			&conn.ErrorMessage,
			&conn.CreatedAt,
			&conn.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		connections = append(connections, &conn)
	}

	return connections, nil
}

func (r *OAuth2ConnectionRepository) ListExpiring(ctx context.Context, within time.Duration) ([]*domain.OAuth2Connection, error) {
	expiresBy := time.Now().Add(within)
	query := `
		SELECT id, credential_id, oauth2_app_id,
			   encrypted_access_token, encrypted_refresh_token,
			   access_token_nonce, refresh_token_nonce, token_type,
			   access_token_expires_at, refresh_token_expires_at,
			   state, code_verifier,
			   account_id, account_email, account_name, raw_userinfo,
			   status, last_refresh_at, last_used_at, error_message,
			   created_at, updated_at
		FROM oauth2_connections
		WHERE status = 'connected'
		  AND access_token_expires_at IS NOT NULL
		  AND access_token_expires_at <= $1
		  AND encrypted_refresh_token IS NOT NULL
		ORDER BY access_token_expires_at ASC
	`

	rows, err := r.pool.Query(ctx, query, expiresBy)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var connections []*domain.OAuth2Connection
	for rows.Next() {
		var conn domain.OAuth2Connection
		err := rows.Scan(
			&conn.ID,
			&conn.CredentialID,
			&conn.OAuth2AppID,
			&conn.EncryptedAccessToken,
			&conn.EncryptedRefreshToken,
			&conn.AccessTokenNonce,
			&conn.RefreshTokenNonce,
			&conn.TokenType,
			&conn.AccessTokenExpiresAt,
			&conn.RefreshTokenExpiresAt,
			&conn.State,
			&conn.CodeVerifier,
			&conn.AccountID,
			&conn.AccountEmail,
			&conn.AccountName,
			&conn.RawUserinfo,
			&conn.Status,
			&conn.LastRefreshAt,
			&conn.LastUsedAt,
			&conn.ErrorMessage,
			&conn.CreatedAt,
			&conn.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		connections = append(connections, &conn)
	}

	return connections, nil
}

func (r *OAuth2ConnectionRepository) Create(ctx context.Context, conn *domain.OAuth2Connection) error {
	query := `
		INSERT INTO oauth2_connections (
			id, credential_id, oauth2_app_id,
			encrypted_access_token, encrypted_refresh_token,
			access_token_nonce, refresh_token_nonce, token_type,
			access_token_expires_at, refresh_token_expires_at,
			state, code_verifier,
			account_id, account_email, account_name, raw_userinfo,
			status, last_refresh_at, last_used_at, error_message,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
	`

	_, err := r.pool.Exec(ctx, query,
		conn.ID,
		conn.CredentialID,
		conn.OAuth2AppID,
		conn.EncryptedAccessToken,
		conn.EncryptedRefreshToken,
		conn.AccessTokenNonce,
		conn.RefreshTokenNonce,
		conn.TokenType,
		conn.AccessTokenExpiresAt,
		conn.RefreshTokenExpiresAt,
		conn.State,
		conn.CodeVerifier,
		conn.AccountID,
		conn.AccountEmail,
		conn.AccountName,
		conn.RawUserinfo,
		conn.Status,
		conn.LastRefreshAt,
		conn.LastUsedAt,
		conn.ErrorMessage,
		conn.CreatedAt,
		conn.UpdatedAt,
	)

	return err
}

func (r *OAuth2ConnectionRepository) Update(ctx context.Context, conn *domain.OAuth2Connection) error {
	query := `
		UPDATE oauth2_connections SET
			encrypted_access_token = $2,
			encrypted_refresh_token = $3,
			access_token_nonce = $4,
			refresh_token_nonce = $5,
			token_type = $6,
			access_token_expires_at = $7,
			refresh_token_expires_at = $8,
			state = $9,
			code_verifier = $10,
			account_id = $11,
			account_email = $12,
			account_name = $13,
			raw_userinfo = $14,
			status = $15,
			last_refresh_at = $16,
			last_used_at = $17,
			error_message = $18,
			updated_at = $19
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query,
		conn.ID,
		conn.EncryptedAccessToken,
		conn.EncryptedRefreshToken,
		conn.AccessTokenNonce,
		conn.RefreshTokenNonce,
		conn.TokenType,
		conn.AccessTokenExpiresAt,
		conn.RefreshTokenExpiresAt,
		conn.State,
		conn.CodeVerifier,
		conn.AccountID,
		conn.AccountEmail,
		conn.AccountName,
		conn.RawUserinfo,
		conn.Status,
		conn.LastRefreshAt,
		conn.LastUsedAt,
		conn.ErrorMessage,
		conn.UpdatedAt,
	)

	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrOAuth2ConnectionNotFound
	}

	return nil
}

func (r *OAuth2ConnectionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM oauth2_connections WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrOAuth2ConnectionNotFound
	}

	return nil
}
