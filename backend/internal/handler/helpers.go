package handler

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/middleware"
	"github.com/souta/ai-orchestration/internal/usecase"
)

// getTenantID extracts tenant ID from request context
func getTenantID(r *http.Request) uuid.UUID {
	return middleware.GetTenantID(r.Context())
}

// getUserID extracts user ID from request context
func getUserID(r *http.Request) uuid.UUID {
	return middleware.GetUserID(r.Context())
}

// getUserEmail extracts user email from request context
func getUserEmail(r *http.Request) string {
	return middleware.GetUserEmail(r.Context())
}

// getUserRoles extracts user roles from request context
func getUserRoles(r *http.Request) []string {
	return middleware.GetUserRoles(r.Context())
}

// hasRole checks if user has a specific role
func hasRole(r *http.Request, role string) bool {
	return middleware.HasRole(r.Context(), role)
}

// parseIntQuery parses an integer query parameter with a default value
func parseIntQuery(r *http.Request, key string, defaultValue int) int {
	str := r.URL.Query().Get(key)
	if str == "" {
		return defaultValue
	}
	val, err := strconv.Atoi(str)
	if err != nil {
		return defaultValue
	}
	return val
}

// parseIntFromString parses an integer from a string with a default value
func parseIntFromString(str string, defaultValue int) int {
	if str == "" {
		return defaultValue
	}
	val, err := strconv.Atoi(str)
	if err != nil {
		return defaultValue
	}
	return val
}

// getClientIP extracts client IP address from request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for reverse proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// logAudit is a helper function to log audit events asynchronously
func logAudit(
	ctx context.Context,
	auditService *usecase.AuditService,
	r *http.Request,
	action domain.AuditAction,
	resourceType domain.AuditResourceType,
	resourceID *uuid.UUID,
	metadata map[string]interface{},
) {
	if auditService == nil {
		return
	}

	tenantID := getTenantID(r)
	userID := getUserID(r)
	userEmail := getUserEmail(r)

	var actorID *uuid.UUID
	if userID != uuid.Nil {
		actorID = &userID
	}

	input := usecase.LogAuditInput{
		TenantID:     tenantID,
		ActorID:      actorID,
		ActorEmail:   userEmail,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Metadata:     metadata,
		IPAddress:    getClientIP(r),
		UserAgent:    r.UserAgent(),
	}

	// Log asynchronously to not block the response
	go func() {
		if err := auditService.Log(context.Background(), input); err != nil {
			log.Printf("Failed to log audit event: %v", err)
		}
	}()
}
