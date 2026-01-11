package handler

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/middleware"
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
