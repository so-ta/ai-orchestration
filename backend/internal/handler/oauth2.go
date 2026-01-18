package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/usecase"
)

// OAuth2Handler handles HTTP requests for OAuth2 operations
type OAuth2Handler struct {
	service      *usecase.OAuth2Service
	auditService *usecase.AuditService
}

// NewOAuth2Handler creates a new OAuth2Handler
func NewOAuth2Handler(service *usecase.OAuth2Service, auditService *usecase.AuditService) *OAuth2Handler {
	return &OAuth2Handler{
		service:      service,
		auditService: auditService,
	}
}

// ============================================================================
// Provider Endpoints
// ============================================================================

// ListProviders returns all OAuth2 providers
func (h *OAuth2Handler) ListProviders(w http.ResponseWriter, r *http.Request) {
	providers, err := h.service.ListProviders(r.Context())
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, usecase.ToProviderResponses(providers))
}

// GetProvider returns a provider by slug
func (h *OAuth2Handler) GetProvider(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "provider slug is required", nil)
		return
	}

	provider, err := h.service.GetProvider(r.Context(), slug)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, usecase.ToProviderResponse(provider))
}

// ============================================================================
// OAuth2 App Endpoints
// ============================================================================

// CreateAppRequest represents the request body for creating an OAuth2 app
type CreateAppRequest struct {
	ProviderSlug string   `json:"provider_slug"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	CustomScopes []string `json:"custom_scopes,omitempty"`
}

// CreateApp creates an OAuth2 app for a tenant
func (h *OAuth2Handler) CreateApp(w http.ResponseWriter, r *http.Request) {
	var req CreateAppRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	if req.ProviderSlug == "" {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "provider_slug is required", nil)
		return
	}
	if req.ClientID == "" {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "client_id is required", nil)
		return
	}
	if req.ClientSecret == "" {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "client_secret is required", nil)
		return
	}

	tenantID := getTenantID(r)

	app, err := h.service.CreateApp(r.Context(), usecase.CreateAppInput{
		TenantID:     tenantID,
		ProviderSlug: req.ProviderSlug,
		ClientID:     req.ClientID,
		ClientSecret: req.ClientSecret,
		CustomScopes: req.CustomScopes,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	// Log audit event
	logAudit(r.Context(), h.auditService, r, domain.AuditActionOAuth2AppCreate, domain.AuditResourceOAuth2App, &app.ID, map[string]interface{}{
		"provider_slug": req.ProviderSlug,
	})

	JSON(w, http.StatusCreated, usecase.ToAppResponse(app))
}

// ListApps returns all OAuth2 apps for a tenant
func (h *OAuth2Handler) ListApps(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)

	apps, err := h.service.ListApps(r.Context(), tenantID)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, usecase.ToAppResponses(apps))
}

// GetApp returns an OAuth2 app by ID
func (h *OAuth2Handler) GetApp(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "app_id", "app ID")
	if !ok {
		return
	}

	app, err := h.service.GetApp(r.Context(), id)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, usecase.ToAppResponse(app))
}

// DeleteApp deletes an OAuth2 app
func (h *OAuth2Handler) DeleteApp(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "app_id", "app ID")
	if !ok {
		return
	}

	if err := h.service.DeleteApp(r.Context(), id); err != nil {
		HandleError(w, err)
		return
	}

	// Log audit event
	logAudit(r.Context(), h.auditService, r, domain.AuditActionOAuth2AppDelete, domain.AuditResourceOAuth2App, &id, nil)

	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// Authorization Flow Endpoints
// ============================================================================

// StartAuthorizationRequest represents the request body for starting OAuth2 authorization
type StartAuthorizationRequest struct {
	ProviderSlug string            `json:"provider_slug"`
	Name         string            `json:"name"`
	Scope        domain.OwnerScope `json:"scope"`
	ProjectID    *uuid.UUID        `json:"project_id,omitempty"`
	Scopes       []string          `json:"scopes,omitempty"`
}

// StartAuthorization initiates the OAuth2 authorization flow
func (h *OAuth2Handler) StartAuthorization(w http.ResponseWriter, r *http.Request) {
	var req StartAuthorizationRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	if req.ProviderSlug == "" {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "provider_slug is required", nil)
		return
	}
	if req.Name == "" {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "name is required", nil)
		return
	}
	if req.Scope == "" {
		req.Scope = domain.OwnerScopePersonal // Default to personal
	}

	// Validate scope
	switch req.Scope {
	case domain.OwnerScopeOrganization, domain.OwnerScopeProject, domain.OwnerScopePersonal:
		// Valid
	default:
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid scope: must be organization, project, or personal", nil)
		return
	}

	// Project ID required for project scope
	if req.Scope == domain.OwnerScopeProject && req.ProjectID == nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "project_id is required for project scope", nil)
		return
	}

	tenantID := getTenantID(r)
	userID := getUserID(r)

	output, err := h.service.StartAuthorization(r.Context(), usecase.StartAuthorizationInput{
		TenantID:     tenantID,
		UserID:       userID,
		ProviderSlug: req.ProviderSlug,
		Scope:        req.Scope,
		ProjectID:    req.ProjectID,
		Name:         req.Name,
		Scopes:       req.Scopes,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	// Log audit event
	logAudit(r.Context(), h.auditService, r, domain.AuditActionOAuth2Start, domain.AuditResourceCredential, &output.CredentialID, map[string]interface{}{
		"provider_slug": req.ProviderSlug,
		"scope":         string(req.Scope),
	})

	JSON(w, http.StatusOK, map[string]interface{}{
		"authorization_url": output.AuthorizationURL,
		"state":             output.State,
		"credential_id":     output.CredentialID,
	})
}

// HandleCallback processes the OAuth2 callback from the provider
func (h *OAuth2Handler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	errorParam := r.URL.Query().Get("error")

	if state == "" {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "state parameter is required", nil)
		return
	}

	output, err := h.service.HandleCallback(r.Context(), usecase.HandleCallbackInput{
		Code:  code,
		State: state,
		Error: errorParam,
	})

	if err != nil {
		// Redirect to frontend with error
		http.Redirect(w, r, "/oauth/callback?error="+err.Error(), http.StatusFound)
		return
	}

	// Log audit event
	logAudit(r.Context(), h.auditService, r, domain.AuditActionOAuth2Callback, domain.AuditResourceCredential, &output.CredentialID, map[string]interface{}{
		"account_email": output.AccountEmail,
	})

	// Redirect to frontend success page
	http.Redirect(w, r, "/oauth/callback?success=true&credential_id="+output.CredentialID.String(), http.StatusFound)
}

// ============================================================================
// Connection Endpoints
// ============================================================================

// ListConnections returns all connections for an OAuth2 app
func (h *OAuth2Handler) ListConnections(w http.ResponseWriter, r *http.Request) {
	appID, ok := parseUUID(w, r, "app_id", "app ID")
	if !ok {
		return
	}

	connections, err := h.service.ListConnectionsByApp(r.Context(), appID)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, usecase.ToConnectionResponses(connections))
}

// GetConnection returns an OAuth2 connection by ID
func (h *OAuth2Handler) GetConnection(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "connection_id", "connection ID")
	if !ok {
		return
	}

	connection, err := h.service.GetConnection(r.Context(), id)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, usecase.ToConnectionResponse(connection))
}

// GetConnectionByCredential returns an OAuth2 connection by credential ID
func (h *OAuth2Handler) GetConnectionByCredential(w http.ResponseWriter, r *http.Request) {
	credentialID, ok := parseUUID(w, r, "credential_id", "credential ID")
	if !ok {
		return
	}

	connection, err := h.service.GetConnectionByCredential(r.Context(), credentialID)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, usecase.ToConnectionResponse(connection))
}

// RefreshConnection refreshes the access token for a connection
func (h *OAuth2Handler) RefreshConnection(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "connection_id", "connection ID")
	if !ok {
		return
	}

	if err := h.service.RefreshToken(r.Context(), id); err != nil {
		HandleError(w, err)
		return
	}

	// Log audit event
	logAudit(r.Context(), h.auditService, r, domain.AuditActionOAuth2Refresh, domain.AuditResourceCredential, &id, nil)

	// Return updated connection
	connection, err := h.service.GetConnection(r.Context(), id)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, usecase.ToConnectionResponse(connection))
}

// RevokeConnection revokes an OAuth2 connection
func (h *OAuth2Handler) RevokeConnection(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "connection_id", "connection ID")
	if !ok {
		return
	}

	if err := h.service.RevokeConnection(r.Context(), id); err != nil {
		HandleError(w, err)
		return
	}

	// Log audit event
	logAudit(r.Context(), h.auditService, r, domain.AuditActionOAuth2Revoke, domain.AuditResourceCredential, &id, nil)

	w.WriteHeader(http.StatusNoContent)
}

// DeleteConnection deletes an OAuth2 connection
func (h *OAuth2Handler) DeleteConnection(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r, "connection_id", "connection ID")
	if !ok {
		return
	}

	tenantID := getTenantID(r)

	if err := h.service.DeleteConnection(r.Context(), id, tenantID); err != nil {
		HandleError(w, err)
		return
	}

	// Log audit event
	logAudit(r.Context(), h.auditService, r, domain.AuditActionOAuth2Delete, domain.AuditResourceCredential, &id, nil)

	w.WriteHeader(http.StatusNoContent)
}
