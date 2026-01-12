package middleware

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewAuthMiddleware(t *testing.T) {
	config := &AuthConfig{
		KeycloakURL: "http://localhost:8180",
		Realm:       "test-realm",
		Enabled:     true,
		DevTenantID: "00000000-0000-0000-0000-000000000001",
	}

	middleware := NewAuthMiddleware(config)

	assert.NotNil(t, middleware)
	assert.Equal(t, config, middleware.config)
}

func TestAuthMiddleware_DevMode(t *testing.T) {
	config := &AuthConfig{
		Enabled:     false,
		DevTenantID: "00000000-0000-0000-0000-000000000002",
	}
	middleware := NewAuthMiddleware(config)

	var capturedCtx context.Context
	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedCtx = r.Context()
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Check context values
	tenantID := GetTenantID(capturedCtx)
	assert.Equal(t, uuid.MustParse("00000000-0000-0000-0000-000000000002"), tenantID)

	userEmail := GetUserEmail(capturedCtx)
	assert.Equal(t, "admin@example.com", userEmail) // Default dev mode email

	roles := GetUserRoles(capturedCtx)
	assert.Contains(t, roles, "admin") // Default dev mode role
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	config := &AuthConfig{
		Enabled: true,
	}
	middleware := NewAuthMiddleware(config)

	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "missing authorization header")
}

func TestAuthMiddleware_InvalidHeaderFormat(t *testing.T) {
	config := &AuthConfig{
		Enabled: true,
	}
	middleware := NewAuthMiddleware(config)

	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name   string
		header string
	}{
		{"invalid format", "InvalidFormat"},
		{"missing token", "Bearer"},
		{"wrong scheme", "Basic token123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", tt.header)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusUnauthorized, rec.Code)
		})
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	config := &AuthConfig{
		Enabled: true,
	}
	middleware := NewAuthMiddleware(config)

	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name  string
		token string
	}{
		{"not jwt format", "not-a-jwt-token"},
		{"invalid base64", "header.!!!invalid!!!.signature"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", "Bearer "+tt.token)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusUnauthorized, rec.Code)
		})
	}
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	config := &AuthConfig{
		Enabled: true,
	}
	middleware := NewAuthMiddleware(config)

	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Create expired token
	claims := Claims{
		Sub:      uuid.New().String(),
		Email:    "test@example.com",
		TenantID: uuid.New().String(),
		Exp:      time.Now().Add(-time.Hour).Unix(), // Expired
		Iat:      time.Now().Add(-2 * time.Hour).Unix(),
	}
	token := createTestJWT(t, claims)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "token expired")
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	config := &AuthConfig{
		Enabled: true,
	}
	middleware := NewAuthMiddleware(config)

	var capturedCtx context.Context
	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedCtx = r.Context()
		w.WriteHeader(http.StatusOK)
	}))

	tenantID := uuid.New()
	userID := uuid.New()
	claims := Claims{
		Sub:      userID.String(),
		Email:    "test@example.com",
		TenantID: tenantID.String(),
		Exp:      time.Now().Add(time.Hour).Unix(),
		Iat:      time.Now().Unix(),
	}
	claims.RealmAccess.Roles = []string{"builder", "operator"}
	token := createTestJWT(t, claims)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Check context values
	assert.Equal(t, tenantID, GetTenantID(capturedCtx))
	assert.Equal(t, userID, GetUserID(capturedCtx))
	assert.Equal(t, "test@example.com", GetUserEmail(capturedCtx))
	assert.Equal(t, []string{"builder", "operator"}, GetUserRoles(capturedCtx))
}

func TestGetTenantID_Default(t *testing.T) {
	ctx := context.Background()
	tenantID := GetTenantID(ctx)
	assert.Equal(t, uuid.MustParse("00000000-0000-0000-0000-000000000001"), tenantID)
}

func TestGetUserID_Nil(t *testing.T) {
	ctx := context.Background()
	userID := GetUserID(ctx)
	assert.Equal(t, uuid.Nil, userID)
}

func TestGetUserEmail_Empty(t *testing.T) {
	ctx := context.Background()
	email := GetUserEmail(ctx)
	assert.Empty(t, email)
}

func TestGetUserRoles_Nil(t *testing.T) {
	ctx := context.Background()
	roles := GetUserRoles(ctx)
	assert.Nil(t, roles)
}

func TestHasRole(t *testing.T) {
	ctx := context.WithValue(context.Background(), UserRolesKey, []string{"admin", "builder"})

	assert.True(t, HasRole(ctx, "admin"))
	assert.True(t, HasRole(ctx, "builder"))
	assert.False(t, HasRole(ctx, "viewer"))
}

func TestHasRole_NoRoles(t *testing.T) {
	ctx := context.Background()
	assert.False(t, HasRole(ctx, "admin"))
}

// createTestJWT creates a test JWT token (without signature verification)
func createTestJWT(t *testing.T, claims Claims) string {
	t.Helper()

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("Failed to marshal claims: %v", err)
	}
	payload := base64.RawURLEncoding.EncodeToString(claimsJSON)

	signature := base64.RawURLEncoding.EncodeToString([]byte("test-signature"))

	return header + "." + payload + "." + signature
}
