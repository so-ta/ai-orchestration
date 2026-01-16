package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
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

// parseUUID parses a UUID from a URL parameter and writes an error response if invalid.
// Returns the parsed UUID and true if successful, or uuid.Nil and false if parsing failed.
func parseUUID(w http.ResponseWriter, r *http.Request, paramName, resourceName string) (uuid.UUID, bool) {
	idStr := chi.URLParam(r, paramName)
	id, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid "+resourceName, nil)
		return uuid.Nil, false
	}
	return id, true
}

// parseUUIDString parses a UUID from a string and writes an error response if invalid.
// Returns the parsed UUID and true if successful, or uuid.Nil and false if parsing failed.
func parseUUIDString(w http.ResponseWriter, idStr, resourceName string) (uuid.UUID, bool) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid "+resourceName, nil)
		return uuid.Nil, false
	}
	return id, true
}

// decodeJSONBody decodes JSON request body into the provided struct.
// Returns true if successful, or writes an error response and returns false if decoding failed.
func decodeJSONBody(w http.ResponseWriter, r *http.Request, v interface{}) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return false
	}
	return true
}

// StepData represents step data in save request
type StepData struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Type      string          `json:"type"`
	Config    json.RawMessage `json:"config"`
	PositionX int             `json:"position_x"`
	PositionY int             `json:"position_y"`
}

// EdgeData represents edge data in save request
type EdgeData struct {
	ID           string  `json:"id"`
	SourceStepID string  `json:"source_step_id"`
	TargetStepID string  `json:"target_step_id"`
	Condition    *string `json:"condition"`
}

// convertStepData converts StepData slice to domain.Step slice.
// Returns the converted steps and true if successful, or nil and false if any UUID parsing failed.
func convertStepData(w http.ResponseWriter, stepDataList []StepData) ([]domain.Step, bool) {
	steps := make([]domain.Step, len(stepDataList))
	for i, s := range stepDataList {
		stepID, err := uuid.Parse(s.ID)
		if err != nil {
			Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid step ID: "+s.ID, nil)
			return nil, false
		}
		steps[i] = domain.Step{
			ID:        stepID,
			Name:      s.Name,
			Type:      domain.StepType(s.Type),
			Config:    s.Config,
			PositionX: s.PositionX,
			PositionY: s.PositionY,
		}
	}
	return steps, true
}

// convertEdgeData converts EdgeData slice to domain.Edge slice.
// Returns the converted edges and true if successful, or nil and false if any UUID parsing failed.
func convertEdgeData(w http.ResponseWriter, edgeDataList []EdgeData) ([]domain.Edge, bool) {
	edges := make([]domain.Edge, len(edgeDataList))
	for i, e := range edgeDataList {
		edgeID, err := uuid.Parse(e.ID)
		if err != nil {
			Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid edge ID: "+e.ID, nil)
			return nil, false
		}
		sourceID, err := uuid.Parse(e.SourceStepID)
		if err != nil {
			Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid source step ID: "+e.SourceStepID, nil)
			return nil, false
		}
		targetID, err := uuid.Parse(e.TargetStepID)
		if err != nil {
			Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid target step ID: "+e.TargetStepID, nil)
			return nil, false
		}
		edges[i] = domain.Edge{
			ID:           edgeID,
			SourceStepID: &sourceID,
			TargetStepID: &targetID,
			Condition:    e.Condition,
		}
	}
	return edges, true
}

// getClientIP extracts client IP address from request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for reverse proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For may contain multiple IPs, take the first one
		if idx := strings.Index(xff, ","); idx != -1 {
			xff = xff[:idx]
		}
		return strings.TrimSpace(xff)
	}
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	// Fall back to RemoteAddr, stripping port number
	// RemoteAddr format is "IP:port" for IPv4 or "[IP]:port" for IPv6
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// If SplitHostPort fails, return as-is (might already be just an IP)
		return r.RemoteAddr
	}
	return host
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

	// Log asynchronously to not block the response.
	// We intentionally use context.Background() here because:
	// 1. Audit logging should complete even after HTTP response is sent
	// 2. The original request context may be cancelled after response
	// 3. Audit logs are fire-and-forget operations for the request flow
	go func() {
		if err := auditService.Log(context.Background(), input); err != nil {
			slog.Error("Failed to log audit event", "error", err)
		}
	}()
}
