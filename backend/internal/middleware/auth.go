package middleware

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Context keys
type contextKey string

const (
	TenantIDKey  contextKey = "tenantID"
	UserIDKey    contextKey = "userID"
	UserEmailKey contextKey = "userEmail"
	UserRolesKey contextKey = "userRoles"
)

// AuthConfig holds authentication configuration
type AuthConfig struct {
	KeycloakURL string
	Realm       string
	Enabled     bool
	DevTenantID string // Default tenant for development
}

// AuthMiddleware handles JWT authentication
type AuthMiddleware struct {
	config    *AuthConfig
	publicKey []byte
	keyMutex  sync.RWMutex
	lastFetch time.Time
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(config *AuthConfig) *AuthMiddleware {
	return &AuthMiddleware{
		config: config,
	}
}

// Claims represents JWT claims from Keycloak
type Claims struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	TenantID      string `json:"tenant_id"`
	RealmAccess   struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
	Exp int64 `json:"exp"`
	Iat int64 `json:"iat"`
}

// Handler returns the middleware handler
func (m *AuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth if disabled (development mode)
		if !m.config.Enabled {
			ctx := m.setDevContext(r.Context())
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":{"code":"UNAUTHORIZED","message":"missing authorization header"}}`, http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			http.Error(w, `{"error":{"code":"UNAUTHORIZED","message":"invalid authorization header"}}`, http.StatusUnauthorized)
			return
		}

		token := parts[1]

		// Validate token and extract claims
		claims, err := m.validateToken(token)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":{"code":"UNAUTHORIZED","message":"%s"}}`, err.Error()), http.StatusUnauthorized)
			return
		}

		// Set context values
		ctx := m.setAuthContext(r.Context(), claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// validateToken validates the JWT token and returns claims
func (m *AuthMiddleware) validateToken(token string) (*Claims, error) {
	// Parse JWT without verification for development
	// In production, you should verify the signature with Keycloak's public key
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	// Decode payload (base64)
	payload, err := base64URLDecode(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid token payload")
	}

	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Check expiration
	if claims.Exp < time.Now().Unix() {
		return nil, fmt.Errorf("token expired")
	}

	return &claims, nil
}

// setDevContext sets default context for development
func (m *AuthMiddleware) setDevContext(ctx context.Context) context.Context {
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	if m.config.DevTenantID != "" {
		if id, err := uuid.Parse(m.config.DevTenantID); err == nil {
			tenantID = id
		}
	}

	ctx = context.WithValue(ctx, TenantIDKey, tenantID)
	// Don't set UserID in dev mode - runs will be created without a user reference
	ctx = context.WithValue(ctx, UserIDKey, uuid.Nil)
	ctx = context.WithValue(ctx, UserEmailKey, "dev@example.com")
	ctx = context.WithValue(ctx, UserRolesKey, []string{"tenant_admin"})
	return ctx
}

// setAuthContext sets authentication context from claims
func (m *AuthMiddleware) setAuthContext(ctx context.Context, claims *Claims) context.Context {
	// Parse tenant ID
	var tenantID uuid.UUID
	if claims.TenantID != "" {
		if id, err := uuid.Parse(claims.TenantID); err == nil {
			tenantID = id
		}
	}
	if tenantID == uuid.Nil {
		tenantID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	}

	// Parse user ID (if invalid, userID will be uuid.Nil)
	userID, err := uuid.Parse(claims.Sub)
	if err != nil {
		userID = uuid.Nil
	}

	ctx = context.WithValue(ctx, TenantIDKey, tenantID)
	ctx = context.WithValue(ctx, UserIDKey, userID)
	ctx = context.WithValue(ctx, UserEmailKey, claims.Email)
	ctx = context.WithValue(ctx, UserRolesKey, claims.RealmAccess.Roles)
	return ctx
}

// GetTenantID extracts tenant ID from context
func GetTenantID(ctx context.Context) uuid.UUID {
	if id, ok := ctx.Value(TenantIDKey).(uuid.UUID); ok {
		return id
	}
	return uuid.MustParse("00000000-0000-0000-0000-000000000001")
}

// GetUserID extracts user ID from context
func GetUserID(ctx context.Context) uuid.UUID {
	if id, ok := ctx.Value(UserIDKey).(uuid.UUID); ok {
		return id
	}
	return uuid.Nil
}

// GetUserEmail extracts user email from context
func GetUserEmail(ctx context.Context) string {
	if email, ok := ctx.Value(UserEmailKey).(string); ok {
		return email
	}
	return ""
}

// GetUserRoles extracts user roles from context
func GetUserRoles(ctx context.Context) []string {
	if roles, ok := ctx.Value(UserRolesKey).([]string); ok {
		return roles
	}
	return nil
}

// HasRole checks if user has a specific role
func HasRole(ctx context.Context, role string) bool {
	roles := GetUserRoles(ctx)
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

// base64URLDecode decodes base64url encoded string
func base64URLDecode(s string) ([]byte, error) {
	// Add padding if needed
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}

	// Replace URL-safe characters
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")

	return base64.StdEncoding.DecodeString(s)
}
