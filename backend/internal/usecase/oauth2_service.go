package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
	"github.com/souta/ai-orchestration/pkg/crypto"
)

// OAuth2Service handles OAuth2 authorization flows
type OAuth2Service struct {
	providerRepo   repository.OAuth2ProviderRepository
	appRepo        repository.OAuth2AppRepository
	connectionRepo repository.OAuth2ConnectionRepository
	credentialRepo repository.CredentialRepository
	encryptor      *crypto.Encryptor
	baseURL        string // Application base URL for callbacks
	httpClient     *http.Client
}

// NewOAuth2Service creates a new OAuth2Service
func NewOAuth2Service(
	providerRepo repository.OAuth2ProviderRepository,
	appRepo repository.OAuth2AppRepository,
	connectionRepo repository.OAuth2ConnectionRepository,
	credentialRepo repository.CredentialRepository,
	encryptor *crypto.Encryptor,
	baseURL string,
) *OAuth2Service {
	return &OAuth2Service{
		providerRepo:   providerRepo,
		appRepo:        appRepo,
		connectionRepo: connectionRepo,
		credentialRepo: credentialRepo,
		encryptor:      encryptor,
		baseURL:        baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// StartAuthorizationInput contains the input for starting OAuth2 authorization
type StartAuthorizationInput struct {
	TenantID     uuid.UUID
	UserID       uuid.UUID
	ProviderSlug string
	Scope        domain.OwnerScope
	ProjectID    *uuid.UUID
	Name         string
	Scopes       []string // Additional scopes to request
}

// StartAuthorizationOutput contains the result of starting OAuth2 authorization
type StartAuthorizationOutput struct {
	AuthorizationURL string
	State            string
	CredentialID     uuid.UUID
}

// StartAuthorization initiates the OAuth2 authorization flow
func (s *OAuth2Service) StartAuthorization(ctx context.Context, input StartAuthorizationInput) (*StartAuthorizationOutput, error) {
	// 1. Get provider
	provider, err := s.providerRepo.GetBySlug(ctx, input.ProviderSlug)
	if err != nil {
		return nil, fmt.Errorf("get provider: %w", err)
	}

	// 2. Get OAuth2 app for tenant
	app, err := s.appRepo.GetByTenantAndProvider(ctx, input.TenantID, provider.ID)
	if err != nil {
		return nil, fmt.Errorf("get oauth2 app: %w", err)
	}

	// 3. Decrypt client ID
	clientID, err := s.encryptor.DecryptDEK(app.EncryptedClientID, app.ClientIDNonce)
	if err != nil {
		return nil, fmt.Errorf("decrypt client ID: %w", err)
	}

	// 4. Create credential (placeholder for OAuth2)
	var cred *domain.Credential
	switch input.Scope {
	case domain.OwnerScopeOrganization:
		cred = domain.NewCredential(input.TenantID, input.Name, domain.CredentialTypeOAuth2)
	case domain.OwnerScopeProject:
		if input.ProjectID == nil {
			return nil, fmt.Errorf("project_id required for project scope")
		}
		cred = domain.NewProjectCredential(input.TenantID, *input.ProjectID, input.Name, domain.CredentialTypeOAuth2)
	case domain.OwnerScopePersonal:
		cred = domain.NewPersonalCredential(input.TenantID, input.UserID, input.Name, domain.CredentialTypeOAuth2)
	default:
		return nil, fmt.Errorf("invalid scope: %s", input.Scope)
	}

	// Set metadata
	metadata := domain.CredentialMetadata{
		ServiceName: provider.Name,
	}
	metadataJSON, _ := metadata.ToJSON()
	cred.Metadata = metadataJSON

	if err := s.credentialRepo.Create(ctx, cred); err != nil {
		return nil, fmt.Errorf("create credential: %w", err)
	}

	// 5. Generate state and PKCE
	state, err := generateRandomString(32)
	if err != nil {
		return nil, fmt.Errorf("generate state: %w", err)
	}

	var codeVerifier, codeChallenge string
	if provider.PKCERequired {
		codeVerifier, err = generateRandomString(64)
		if err != nil {
			return nil, fmt.Errorf("generate code verifier: %w", err)
		}
		codeChallenge = generateCodeChallenge(codeVerifier)
	}

	// 6. Create OAuth2 connection (pending state)
	conn := domain.NewOAuth2Connection(cred.ID, app.ID)
	conn.State = state
	conn.CodeVerifier = codeVerifier

	if err := s.connectionRepo.Create(ctx, conn); err != nil {
		// Clean up credential
		_ = s.credentialRepo.Delete(ctx, input.TenantID, cred.ID)
		return nil, fmt.Errorf("create connection: %w", err)
	}

	// 7. Build authorization URL
	scopes := app.GetScopes()
	if len(input.Scopes) > 0 {
		scopes = append(scopes, input.Scopes...)
	}

	authURL, err := s.buildAuthorizationURL(provider, string(clientID), state, scopes, codeChallenge)
	if err != nil {
		return nil, fmt.Errorf("build authorization URL: %w", err)
	}

	return &StartAuthorizationOutput{
		AuthorizationURL: authURL,
		State:            state,
		CredentialID:     cred.ID,
	}, nil
}

// HandleCallbackInput contains the input for handling OAuth2 callback
type HandleCallbackInput struct {
	Code  string
	State string
	Error string
}

// HandleCallbackOutput contains the result of handling OAuth2 callback
type HandleCallbackOutput struct {
	CredentialID uuid.UUID
	AccountEmail string
	AccountName  string
}

// HandleCallback processes the OAuth2 callback after user authorization
func (s *OAuth2Service) HandleCallback(ctx context.Context, input HandleCallbackInput) (*HandleCallbackOutput, error) {
	// Handle error from provider
	if input.Error != "" {
		return nil, fmt.Errorf("oauth2 error: %s", input.Error)
	}

	// 1. Find connection by state
	conn, err := s.connectionRepo.GetByState(ctx, input.State)
	if err != nil {
		return nil, fmt.Errorf("invalid state: %w", err)
	}

	// 2. Get OAuth2 app
	app, err := s.appRepo.GetByID(ctx, conn.OAuth2AppID)
	if err != nil {
		return nil, fmt.Errorf("get oauth2 app: %w", err)
	}

	// 3. Decrypt client credentials
	clientID, err := s.encryptor.DecryptDEK(app.EncryptedClientID, app.ClientIDNonce)
	if err != nil {
		return nil, fmt.Errorf("decrypt client ID: %w", err)
	}
	clientSecret, err := s.encryptor.DecryptDEK(app.EncryptedClientSecret, app.ClientSecretNonce)
	if err != nil {
		return nil, fmt.Errorf("decrypt client secret: %w", err)
	}

	// 4. Exchange code for token
	tokenResp, err := s.exchangeCodeForToken(ctx, app.Provider, string(clientID), string(clientSecret), input.Code, conn.CodeVerifier)
	if err != nil {
		conn.MarkAsError(err.Error())
		_ = s.connectionRepo.Update(ctx, conn)
		return nil, fmt.Errorf("exchange code for token: %w", err)
	}

	// 5. Encrypt and store tokens
	accessTokenEnc, accessNonce, err := s.encryptor.EncryptDEK([]byte(tokenResp.AccessToken))
	if err != nil {
		return nil, fmt.Errorf("encrypt access token: %w", err)
	}

	var refreshTokenEnc, refreshNonce []byte
	if tokenResp.RefreshToken != "" {
		refreshTokenEnc, refreshNonce, err = s.encryptor.EncryptDEK([]byte(tokenResp.RefreshToken))
		if err != nil {
			return nil, fmt.Errorf("encrypt refresh token: %w", err)
		}
	}

	// Calculate expiration
	var expiresAt *time.Time
	if tokenResp.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		expiresAt = &t
	}

	conn.EncryptedAccessToken = accessTokenEnc
	conn.AccessTokenNonce = accessNonce
	conn.EncryptedRefreshToken = refreshTokenEnc
	conn.RefreshTokenNonce = refreshNonce
	conn.TokenType = tokenResp.TokenType
	conn.AccessTokenExpiresAt = expiresAt
	conn.State = "" // Clear state after use
	conn.CodeVerifier = ""

	// 6. Fetch user info if available
	if app.Provider.UserinfoURL != "" {
		userInfo, err := s.fetchUserInfo(ctx, app.Provider.UserinfoURL, tokenResp.AccessToken)
		if err == nil {
			conn.AccountID = userInfo.GetAccountID()
			conn.AccountEmail = userInfo.Email
			conn.AccountName = userInfo.GetDisplayName()
			conn.RawUserinfo = userInfo.Raw
		}
	}

	conn.MarkAsConnected()
	if err := s.connectionRepo.Update(ctx, conn); err != nil {
		return nil, fmt.Errorf("update connection: %w", err)
	}

	return &HandleCallbackOutput{
		CredentialID: conn.CredentialID,
		AccountEmail: conn.AccountEmail,
		AccountName:  conn.AccountName,
	}, nil
}

// RefreshToken refreshes the access token for a connection
func (s *OAuth2Service) RefreshToken(ctx context.Context, connectionID uuid.UUID) error {
	// 1. Get connection
	conn, err := s.connectionRepo.GetByID(ctx, connectionID)
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}

	if !conn.CanRefresh() {
		return domain.ErrOAuth2RefreshFailed
	}

	// 2. Get OAuth2 app
	app, err := s.appRepo.GetByID(ctx, conn.OAuth2AppID)
	if err != nil {
		return fmt.Errorf("get oauth2 app: %w", err)
	}

	// 3. Decrypt credentials
	clientID, err := s.encryptor.DecryptDEK(app.EncryptedClientID, app.ClientIDNonce)
	if err != nil {
		return fmt.Errorf("decrypt client ID: %w", err)
	}
	clientSecret, err := s.encryptor.DecryptDEK(app.EncryptedClientSecret, app.ClientSecretNonce)
	if err != nil {
		return fmt.Errorf("decrypt client secret: %w", err)
	}
	refreshToken, err := s.encryptor.DecryptDEK(conn.EncryptedRefreshToken, conn.RefreshTokenNonce)
	if err != nil {
		return fmt.Errorf("decrypt refresh token: %w", err)
	}

	// 4. Refresh token
	tokenResp, err := s.refreshAccessToken(ctx, app.Provider, string(clientID), string(clientSecret), string(refreshToken))
	if err != nil {
		conn.MarkAsError(err.Error())
		_ = s.connectionRepo.Update(ctx, conn)
		return fmt.Errorf("refresh token: %w", err)
	}

	// 5. Encrypt and store new tokens
	accessTokenEnc, accessNonce, err := s.encryptor.EncryptDEK([]byte(tokenResp.AccessToken))
	if err != nil {
		return fmt.Errorf("encrypt access token: %w", err)
	}

	var refreshTokenEnc, refreshNonce []byte
	if tokenResp.RefreshToken != "" {
		refreshTokenEnc, refreshNonce, err = s.encryptor.EncryptDEK([]byte(tokenResp.RefreshToken))
		if err != nil {
			return fmt.Errorf("encrypt refresh token: %w", err)
		}
	}

	var expiresAt *time.Time
	if tokenResp.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		expiresAt = &t
	}

	conn.UpdateTokens(accessTokenEnc, refreshTokenEnc, accessNonce, refreshNonce, expiresAt)
	if err := s.connectionRepo.Update(ctx, conn); err != nil {
		return fmt.Errorf("update connection: %w", err)
	}

	return nil
}

// GetValidAccessToken returns a valid access token, refreshing if necessary
func (s *OAuth2Service) GetValidAccessToken(ctx context.Context, credentialID uuid.UUID) (string, error) {
	conn, err := s.connectionRepo.GetByCredentialID(ctx, credentialID)
	if err != nil {
		return "", fmt.Errorf("get connection: %w", err)
	}

	if conn.Status != domain.OAuth2ConnectionStatusConnected {
		return "", domain.ErrOAuth2TokenExpired
	}

	// Check if refresh is needed
	if conn.IsAccessTokenExpired() {
		if !conn.CanRefresh() {
			conn.MarkAsExpired()
			_ = s.connectionRepo.Update(ctx, conn)
			return "", domain.ErrOAuth2TokenExpired
		}

		if err := s.RefreshToken(ctx, conn.ID); err != nil {
			return "", err
		}

		// Re-fetch connection after refresh
		conn, err = s.connectionRepo.GetByCredentialID(ctx, credentialID)
		if err != nil {
			return "", fmt.Errorf("get connection after refresh: %w", err)
		}
	}

	// Decrypt and return access token
	accessToken, err := s.encryptor.DecryptDEK(conn.EncryptedAccessToken, conn.AccessTokenNonce)
	if err != nil {
		return "", fmt.Errorf("decrypt access token: %w", err)
	}

	// Record usage
	conn.RecordUsage()
	_ = s.connectionRepo.Update(ctx, conn)

	return string(accessToken), nil
}

// RevokeConnection revokes an OAuth2 connection
func (s *OAuth2Service) RevokeConnection(ctx context.Context, connectionID uuid.UUID) error {
	conn, err := s.connectionRepo.GetByID(ctx, connectionID)
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}

	app, err := s.appRepo.GetByID(ctx, conn.OAuth2AppID)
	if err != nil {
		return fmt.Errorf("get oauth2 app: %w", err)
	}

	// Try to revoke token at provider if supported
	if app.Provider.RevokeURL != "" && conn.EncryptedAccessToken != nil {
		accessToken, _ := s.encryptor.DecryptDEK(conn.EncryptedAccessToken, conn.AccessTokenNonce)
		_ = s.revokeTokenAtProvider(ctx, app.Provider.RevokeURL, string(accessToken))
	}

	conn.MarkAsRevoked()
	if err := s.connectionRepo.Update(ctx, conn); err != nil {
		return fmt.Errorf("update connection: %w", err)
	}

	return nil
}

// Helper functions

func (s *OAuth2Service) buildAuthorizationURL(provider *domain.OAuth2Provider, clientID, state string, scopes []string, codeChallenge string) (string, error) {
	u, err := url.Parse(provider.AuthorizationURL)
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Set("client_id", clientID)
	q.Set("redirect_uri", s.baseURL+"/api/v1/oauth2/callback")
	q.Set("response_type", "code")
	q.Set("state", state)

	if len(scopes) > 0 {
		q.Set("scope", strings.Join(scopes, " "))
	}

	if codeChallenge != "" {
		q.Set("code_challenge", codeChallenge)
		q.Set("code_challenge_method", "S256")
	}

	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (s *OAuth2Service) exchangeCodeForToken(ctx context.Context, provider *domain.OAuth2Provider, clientID, clientSecret, code, codeVerifier string) (*domain.OAuth2TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("redirect_uri", s.baseURL+"/api/v1/oauth2/callback")

	if codeVerifier != "" {
		data.Set("code_verifier", codeVerifier)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", provider.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tokenResp domain.OAuth2TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}

	if tokenResp.Error != "" {
		return nil, fmt.Errorf("%s: %s", tokenResp.Error, tokenResp.ErrorDescription)
	}

	if tokenResp.TokenType == "" {
		tokenResp.TokenType = "Bearer"
	}

	return &tokenResp, nil
}

func (s *OAuth2Service) refreshAccessToken(ctx context.Context, provider *domain.OAuth2Provider, clientID, clientSecret, refreshToken string) (*domain.OAuth2TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", provider.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tokenResp domain.OAuth2TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}

	if tokenResp.Error != "" {
		return nil, fmt.Errorf("%s: %s", tokenResp.Error, tokenResp.ErrorDescription)
	}

	if tokenResp.TokenType == "" {
		tokenResp.TokenType = "Bearer"
	}

	return &tokenResp, nil
}

func (s *OAuth2Service) fetchUserInfo(ctx context.Context, userinfoURL, accessToken string) (*domain.OAuth2UserinfoResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", userinfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var userInfo domain.OAuth2UserinfoResponse
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, err
	}
	userInfo.Raw = body

	return &userInfo, nil
}

func (s *OAuth2Service) revokeTokenAtProvider(ctx context.Context, revokeURL, token string) error {
	data := url.Values{}
	data.Set("token", token)

	req, err := http.NewRequestWithContext(ctx, "POST", revokeURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// Utility functions

func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes)[:length], nil
}

func generateCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// ============================================================================
// Provider and App Management
// ============================================================================

// ListProviders returns all OAuth2 providers
func (s *OAuth2Service) ListProviders(ctx context.Context) ([]*domain.OAuth2Provider, error) {
	return s.providerRepo.List(ctx)
}

// ListPresetProviders returns all preset OAuth2 providers
func (s *OAuth2Service) ListPresetProviders(ctx context.Context) ([]*domain.OAuth2Provider, error) {
	return s.providerRepo.ListPresets(ctx)
}

// GetProvider returns a provider by slug
func (s *OAuth2Service) GetProvider(ctx context.Context, slug string) (*domain.OAuth2Provider, error) {
	return s.providerRepo.GetBySlug(ctx, slug)
}

// CreateAppInput contains input for creating an OAuth2 app
type CreateAppInput struct {
	TenantID     uuid.UUID
	ProviderSlug string
	ClientID     string
	ClientSecret string
	CustomScopes []string
}

// CreateApp creates an OAuth2 app for a tenant
func (s *OAuth2Service) CreateApp(ctx context.Context, input CreateAppInput) (*domain.OAuth2App, error) {
	// Get provider
	provider, err := s.providerRepo.GetBySlug(ctx, input.ProviderSlug)
	if err != nil {
		return nil, fmt.Errorf("get provider: %w", err)
	}

	// Check if app already exists
	existing, err := s.appRepo.GetByTenantAndProvider(ctx, input.TenantID, provider.ID)
	if err == nil && existing != nil {
		return nil, domain.ErrOAuth2AppAlreadyExists
	}

	// Encrypt client credentials
	clientIDEnc, clientIDNonce, err := s.encryptor.EncryptDEK([]byte(input.ClientID))
	if err != nil {
		return nil, fmt.Errorf("encrypt client ID: %w", err)
	}
	clientSecretEnc, clientSecretNonce, err := s.encryptor.EncryptDEK([]byte(input.ClientSecret))
	if err != nil {
		return nil, fmt.Errorf("encrypt client secret: %w", err)
	}

	app := domain.NewOAuth2App(input.TenantID, provider.ID)
	app.EncryptedClientID = clientIDEnc
	app.ClientIDNonce = clientIDNonce
	app.EncryptedClientSecret = clientSecretEnc
	app.ClientSecretNonce = clientSecretNonce
	app.CustomScopes = input.CustomScopes
	app.Provider = provider

	if err := s.appRepo.Create(ctx, app); err != nil {
		return nil, fmt.Errorf("create app: %w", err)
	}

	return app, nil
}

// ListApps returns all OAuth2 apps for a tenant
func (s *OAuth2Service) ListApps(ctx context.Context, tenantID uuid.UUID) ([]*domain.OAuth2App, error) {
	return s.appRepo.ListByTenant(ctx, tenantID)
}

// GetApp returns an OAuth2 app by ID
func (s *OAuth2Service) GetApp(ctx context.Context, id uuid.UUID) (*domain.OAuth2App, error) {
	return s.appRepo.GetByID(ctx, id)
}

// DeleteApp deletes an OAuth2 app
func (s *OAuth2Service) DeleteApp(ctx context.Context, id uuid.UUID) error {
	return s.appRepo.Delete(ctx, id)
}

// ============================================================================
// Connection Management
// ============================================================================

// ListConnectionsByApp returns all connections for an OAuth2 app
func (s *OAuth2Service) ListConnectionsByApp(ctx context.Context, appID uuid.UUID) ([]*domain.OAuth2Connection, error) {
	return s.connectionRepo.ListByApp(ctx, appID)
}

// GetConnection returns an OAuth2 connection by ID
func (s *OAuth2Service) GetConnection(ctx context.Context, id uuid.UUID) (*domain.OAuth2Connection, error) {
	return s.connectionRepo.GetByID(ctx, id)
}

// GetConnectionByCredential returns an OAuth2 connection by credential ID
func (s *OAuth2Service) GetConnectionByCredential(ctx context.Context, credentialID uuid.UUID) (*domain.OAuth2Connection, error) {
	return s.connectionRepo.GetByCredentialID(ctx, credentialID)
}

// DeleteConnection deletes an OAuth2 connection (and associated credential)
func (s *OAuth2Service) DeleteConnection(ctx context.Context, connectionID uuid.UUID, tenantID uuid.UUID) error {
	conn, err := s.connectionRepo.GetByID(ctx, connectionID)
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}

	// Delete connection
	if err := s.connectionRepo.Delete(ctx, connectionID); err != nil {
		return fmt.Errorf("delete connection: %w", err)
	}

	// Delete associated credential
	if err := s.credentialRepo.Delete(ctx, tenantID, conn.CredentialID); err != nil {
		// Log but don't fail
		return nil
	}

	return nil
}

// ============================================================================
// Response Types
// ============================================================================

// OAuth2ProviderResponse is the API response for an OAuth2 provider
type OAuth2ProviderResponse struct {
	ID               uuid.UUID `json:"id"`
	Slug             string    `json:"slug"`
	Name             string    `json:"name"`
	PKCERequired     bool      `json:"pkce_required"`
	DefaultScopes    []string  `json:"default_scopes"`
	IconURL          string    `json:"icon_url,omitempty"`
	DocumentationURL string    `json:"documentation_url,omitempty"`
	IsPreset         bool      `json:"is_preset"`
}

// ToProviderResponse converts a domain provider to API response
func ToProviderResponse(p *domain.OAuth2Provider) OAuth2ProviderResponse {
	return OAuth2ProviderResponse{
		ID:               p.ID,
		Slug:             p.Slug,
		Name:             p.Name,
		PKCERequired:     p.PKCERequired,
		DefaultScopes:    p.DefaultScopes,
		IconURL:          p.IconURL,
		DocumentationURL: p.DocumentationURL,
		IsPreset:         p.IsPreset,
	}
}

// ToProviderResponses converts domain providers to API responses
func ToProviderResponses(providers []*domain.OAuth2Provider) []OAuth2ProviderResponse {
	responses := make([]OAuth2ProviderResponse, len(providers))
	for i, p := range providers {
		responses[i] = ToProviderResponse(p)
	}
	return responses
}

// OAuth2AppResponse is the API response for an OAuth2 app
type OAuth2AppResponse struct {
	ID           uuid.UUID                 `json:"id"`
	TenantID     uuid.UUID                 `json:"tenant_id"`
	ProviderSlug string                    `json:"provider_slug"`
	ProviderName string                    `json:"provider_name"`
	CustomScopes []string                  `json:"custom_scopes,omitempty"`
	Status       domain.OAuth2AppStatus    `json:"status"`
	CreatedAt    time.Time                 `json:"created_at"`
	Provider     *OAuth2ProviderResponse   `json:"provider,omitempty"`
}

// ToAppResponse converts a domain app to API response
func ToAppResponse(app *domain.OAuth2App) OAuth2AppResponse {
	resp := OAuth2AppResponse{
		ID:           app.ID,
		TenantID:     app.TenantID,
		CustomScopes: app.CustomScopes,
		Status:       app.Status,
		CreatedAt:    app.CreatedAt,
	}
	if app.Provider != nil {
		providerResp := ToProviderResponse(app.Provider)
		resp.Provider = &providerResp
		resp.ProviderSlug = app.Provider.Slug
		resp.ProviderName = app.Provider.Name
	}
	return resp
}

// ToAppResponses converts domain apps to API responses
func ToAppResponses(apps []*domain.OAuth2App) []OAuth2AppResponse {
	responses := make([]OAuth2AppResponse, len(apps))
	for i, app := range apps {
		responses[i] = ToAppResponse(app)
	}
	return responses
}

// OAuth2ConnectionResponse is the API response for an OAuth2 connection
type OAuth2ConnectionResponse struct {
	ID                   uuid.UUID                      `json:"id"`
	CredentialID         uuid.UUID                      `json:"credential_id"`
	OAuth2AppID          uuid.UUID                      `json:"oauth2_app_id"`
	Status               domain.OAuth2ConnectionStatus  `json:"status"`
	AccountID            string                         `json:"account_id,omitempty"`
	AccountEmail         string                         `json:"account_email,omitempty"`
	AccountName          string                         `json:"account_name,omitempty"`
	AccessTokenExpiresAt *time.Time                     `json:"access_token_expires_at,omitempty"`
	LastRefreshAt        *time.Time                     `json:"last_refresh_at,omitempty"`
	LastUsedAt           *time.Time                     `json:"last_used_at,omitempty"`
	ErrorMessage         string                         `json:"error_message,omitempty"`
	CreatedAt            time.Time                      `json:"created_at"`
}

// ToConnectionResponse converts a domain connection to API response
func ToConnectionResponse(conn *domain.OAuth2Connection) OAuth2ConnectionResponse {
	return OAuth2ConnectionResponse{
		ID:                   conn.ID,
		CredentialID:         conn.CredentialID,
		OAuth2AppID:          conn.OAuth2AppID,
		Status:               conn.Status,
		AccountID:            conn.AccountID,
		AccountEmail:         conn.AccountEmail,
		AccountName:          conn.AccountName,
		AccessTokenExpiresAt: conn.AccessTokenExpiresAt,
		LastRefreshAt:        conn.LastRefreshAt,
		LastUsedAt:           conn.LastUsedAt,
		ErrorMessage:         conn.ErrorMessage,
		CreatedAt:            conn.CreatedAt,
	}
}

// ToConnectionResponses converts domain connections to API responses
func ToConnectionResponses(conns []*domain.OAuth2Connection) []OAuth2ConnectionResponse {
	responses := make([]OAuth2ConnectionResponse, len(conns))
	for i, conn := range conns {
		responses[i] = ToConnectionResponse(conn)
	}
	return responses
}
