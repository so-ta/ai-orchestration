package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/souta/ai-orchestration/internal/domain"
)

// OAuth2AppRepository implements repository.OAuth2AppRepository
type OAuth2AppRepository struct {
	pool *pgxpool.Pool
}

// NewOAuth2AppRepository creates a new OAuth2AppRepository
func NewOAuth2AppRepository(pool *pgxpool.Pool) *OAuth2AppRepository {
	return &OAuth2AppRepository{pool: pool}
}

func (r *OAuth2AppRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.OAuth2App, error) {
	query := `
		SELECT a.id, a.tenant_id, a.provider_id,
			   a.encrypted_client_id, a.encrypted_client_secret,
			   a.client_id_nonce, a.client_secret_nonce,
			   a.custom_scopes, a.redirect_uri, a.status,
			   a.created_at, a.updated_at,
			   p.id, p.slug, p.name, p.authorization_url, p.token_url,
			   p.revoke_url, p.userinfo_url, p.pkce_required, p.default_scopes,
			   p.icon_url, p.documentation_url, p.is_preset, p.created_at, p.updated_at
		FROM oauth2_apps a
		JOIN oauth2_providers p ON a.provider_id = p.id
		WHERE a.id = $1
	`

	var app domain.OAuth2App
	var provider domain.OAuth2Provider
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&app.ID,
		&app.TenantID,
		&app.ProviderID,
		&app.EncryptedClientID,
		&app.EncryptedClientSecret,
		&app.ClientIDNonce,
		&app.ClientSecretNonce,
		&app.CustomScopes,
		&app.RedirectURI,
		&app.Status,
		&app.CreatedAt,
		&app.UpdatedAt,
		&provider.ID,
		&provider.Slug,
		&provider.Name,
		&provider.AuthorizationURL,
		&provider.TokenURL,
		&provider.RevokeURL,
		&provider.UserinfoURL,
		&provider.PKCERequired,
		&provider.DefaultScopes,
		&provider.IconURL,
		&provider.DocumentationURL,
		&provider.IsPreset,
		&provider.CreatedAt,
		&provider.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrOAuth2AppNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get oauth2 app by ID: %w", err)
	}

	app.Provider = &provider
	return &app, nil
}

func (r *OAuth2AppRepository) GetByTenantAndProvider(ctx context.Context, tenantID, providerID uuid.UUID) (*domain.OAuth2App, error) {
	query := `
		SELECT a.id, a.tenant_id, a.provider_id,
			   a.encrypted_client_id, a.encrypted_client_secret,
			   a.client_id_nonce, a.client_secret_nonce,
			   a.custom_scopes, a.redirect_uri, a.status,
			   a.created_at, a.updated_at,
			   p.id, p.slug, p.name, p.authorization_url, p.token_url,
			   p.revoke_url, p.userinfo_url, p.pkce_required, p.default_scopes,
			   p.icon_url, p.documentation_url, p.is_preset, p.created_at, p.updated_at
		FROM oauth2_apps a
		JOIN oauth2_providers p ON a.provider_id = p.id
		WHERE a.tenant_id = $1 AND a.provider_id = $2
	`

	var app domain.OAuth2App
	var provider domain.OAuth2Provider
	err := r.pool.QueryRow(ctx, query, tenantID, providerID).Scan(
		&app.ID,
		&app.TenantID,
		&app.ProviderID,
		&app.EncryptedClientID,
		&app.EncryptedClientSecret,
		&app.ClientIDNonce,
		&app.ClientSecretNonce,
		&app.CustomScopes,
		&app.RedirectURI,
		&app.Status,
		&app.CreatedAt,
		&app.UpdatedAt,
		&provider.ID,
		&provider.Slug,
		&provider.Name,
		&provider.AuthorizationURL,
		&provider.TokenURL,
		&provider.RevokeURL,
		&provider.UserinfoURL,
		&provider.PKCERequired,
		&provider.DefaultScopes,
		&provider.IconURL,
		&provider.DocumentationURL,
		&provider.IsPreset,
		&provider.CreatedAt,
		&provider.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrOAuth2AppNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get oauth2 app by tenant and provider: %w", err)
	}

	app.Provider = &provider
	return &app, nil
}

func (r *OAuth2AppRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.OAuth2App, error) {
	query := `
		SELECT a.id, a.tenant_id, a.provider_id,
			   a.encrypted_client_id, a.encrypted_client_secret,
			   a.client_id_nonce, a.client_secret_nonce,
			   a.custom_scopes, a.redirect_uri, a.status,
			   a.created_at, a.updated_at,
			   p.id, p.slug, p.name, p.authorization_url, p.token_url,
			   p.revoke_url, p.userinfo_url, p.pkce_required, p.default_scopes,
			   p.icon_url, p.documentation_url, p.is_preset, p.created_at, p.updated_at
		FROM oauth2_apps a
		JOIN oauth2_providers p ON a.provider_id = p.id
		WHERE a.tenant_id = $1
		ORDER BY p.name ASC
	`

	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []*domain.OAuth2App
	for rows.Next() {
		var app domain.OAuth2App
		var provider domain.OAuth2Provider
		err := rows.Scan(
			&app.ID,
			&app.TenantID,
			&app.ProviderID,
			&app.EncryptedClientID,
			&app.EncryptedClientSecret,
			&app.ClientIDNonce,
			&app.ClientSecretNonce,
			&app.CustomScopes,
			&app.RedirectURI,
			&app.Status,
			&app.CreatedAt,
			&app.UpdatedAt,
			&provider.ID,
			&provider.Slug,
			&provider.Name,
			&provider.AuthorizationURL,
			&provider.TokenURL,
			&provider.RevokeURL,
			&provider.UserinfoURL,
			&provider.PKCERequired,
			&provider.DefaultScopes,
			&provider.IconURL,
			&provider.DocumentationURL,
			&provider.IsPreset,
			&provider.CreatedAt,
			&provider.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		app.Provider = &provider
		apps = append(apps, &app)
	}

	return apps, nil
}

func (r *OAuth2AppRepository) Create(ctx context.Context, app *domain.OAuth2App) error {
	query := `
		INSERT INTO oauth2_apps (
			id, tenant_id, provider_id,
			encrypted_client_id, encrypted_client_secret,
			client_id_nonce, client_secret_nonce,
			custom_scopes, redirect_uri, status,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := r.pool.Exec(ctx, query,
		app.ID,
		app.TenantID,
		app.ProviderID,
		app.EncryptedClientID,
		app.EncryptedClientSecret,
		app.ClientIDNonce,
		app.ClientSecretNonce,
		app.CustomScopes,
		app.RedirectURI,
		app.Status,
		app.CreatedAt,
		app.UpdatedAt,
	)

	return err
}

func (r *OAuth2AppRepository) Update(ctx context.Context, app *domain.OAuth2App) error {
	query := `
		UPDATE oauth2_apps SET
			encrypted_client_id = $2,
			encrypted_client_secret = $3,
			client_id_nonce = $4,
			client_secret_nonce = $5,
			custom_scopes = $6,
			redirect_uri = $7,
			status = $8,
			updated_at = $9
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query,
		app.ID,
		app.EncryptedClientID,
		app.EncryptedClientSecret,
		app.ClientIDNonce,
		app.ClientSecretNonce,
		app.CustomScopes,
		app.RedirectURI,
		app.Status,
		app.UpdatedAt,
	)

	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrOAuth2AppNotFound
	}

	return nil
}

func (r *OAuth2AppRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM oauth2_apps WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrOAuth2AppNotFound
	}

	return nil
}
