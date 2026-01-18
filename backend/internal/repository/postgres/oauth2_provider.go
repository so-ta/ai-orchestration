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

// OAuth2ProviderRepository implements repository.OAuth2ProviderRepository
type OAuth2ProviderRepository struct {
	pool *pgxpool.Pool
}

// NewOAuth2ProviderRepository creates a new OAuth2ProviderRepository
func NewOAuth2ProviderRepository(pool *pgxpool.Pool) *OAuth2ProviderRepository {
	return &OAuth2ProviderRepository{pool: pool}
}

func (r *OAuth2ProviderRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.OAuth2Provider, error) {
	query := `
		SELECT id, slug, name, authorization_url, token_url, revoke_url, userinfo_url,
			   pkce_required, default_scopes, icon_url, documentation_url, is_preset,
			   created_at, updated_at
		FROM oauth2_providers
		WHERE id = $1
	`

	var provider domain.OAuth2Provider
	err := r.pool.QueryRow(ctx, query, id).Scan(
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
		return nil, domain.ErrOAuth2ProviderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get oauth2 provider by ID: %w", err)
	}

	return &provider, nil
}

func (r *OAuth2ProviderRepository) GetBySlug(ctx context.Context, slug string) (*domain.OAuth2Provider, error) {
	query := `
		SELECT id, slug, name, authorization_url, token_url, revoke_url, userinfo_url,
			   pkce_required, default_scopes, icon_url, documentation_url, is_preset,
			   created_at, updated_at
		FROM oauth2_providers
		WHERE slug = $1
	`

	var provider domain.OAuth2Provider
	err := r.pool.QueryRow(ctx, query, slug).Scan(
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
		return nil, domain.ErrOAuth2ProviderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get oauth2 provider by slug: %w", err)
	}

	return &provider, nil
}

func (r *OAuth2ProviderRepository) List(ctx context.Context) ([]*domain.OAuth2Provider, error) {
	query := `
		SELECT id, slug, name, authorization_url, token_url, revoke_url, userinfo_url,
			   pkce_required, default_scopes, icon_url, documentation_url, is_preset,
			   created_at, updated_at
		FROM oauth2_providers
		ORDER BY is_preset DESC, name ASC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var providers []*domain.OAuth2Provider
	for rows.Next() {
		var provider domain.OAuth2Provider
		err := rows.Scan(
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
		providers = append(providers, &provider)
	}

	return providers, nil
}

func (r *OAuth2ProviderRepository) ListPresets(ctx context.Context) ([]*domain.OAuth2Provider, error) {
	query := `
		SELECT id, slug, name, authorization_url, token_url, revoke_url, userinfo_url,
			   pkce_required, default_scopes, icon_url, documentation_url, is_preset,
			   created_at, updated_at
		FROM oauth2_providers
		WHERE is_preset = true
		ORDER BY name ASC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var providers []*domain.OAuth2Provider
	for rows.Next() {
		var provider domain.OAuth2Provider
		err := rows.Scan(
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
		providers = append(providers, &provider)
	}

	return providers, nil
}

func (r *OAuth2ProviderRepository) Create(ctx context.Context, provider *domain.OAuth2Provider) error {
	query := `
		INSERT INTO oauth2_providers (
			id, slug, name, authorization_url, token_url, revoke_url, userinfo_url,
			pkce_required, default_scopes, icon_url, documentation_url, is_preset,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	_, err := r.pool.Exec(ctx, query,
		provider.ID,
		provider.Slug,
		provider.Name,
		provider.AuthorizationURL,
		provider.TokenURL,
		provider.RevokeURL,
		provider.UserinfoURL,
		provider.PKCERequired,
		provider.DefaultScopes,
		provider.IconURL,
		provider.DocumentationURL,
		provider.IsPreset,
		provider.CreatedAt,
		provider.UpdatedAt,
	)

	return err
}

func (r *OAuth2ProviderRepository) Update(ctx context.Context, provider *domain.OAuth2Provider) error {
	query := `
		UPDATE oauth2_providers SET
			name = $2,
			authorization_url = $3,
			token_url = $4,
			revoke_url = $5,
			userinfo_url = $6,
			pkce_required = $7,
			default_scopes = $8,
			icon_url = $9,
			documentation_url = $10,
			updated_at = $11
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query,
		provider.ID,
		provider.Name,
		provider.AuthorizationURL,
		provider.TokenURL,
		provider.RevokeURL,
		provider.UserinfoURL,
		provider.PKCERequired,
		provider.DefaultScopes,
		provider.IconURL,
		provider.DocumentationURL,
		provider.UpdatedAt,
	)

	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrOAuth2ProviderNotFound
	}

	return nil
}

func (r *OAuth2ProviderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM oauth2_providers WHERE id = $1 AND is_preset = false`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrOAuth2ProviderNotFound
	}

	return nil
}
