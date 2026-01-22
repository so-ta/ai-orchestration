package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/souta/ai-orchestration/internal/repository/postgres"
)

// VariablesHandler handles HTTP requests for environment variables
type VariablesHandler struct {
	tenantRepo *postgres.TenantRepository
	userRepo   *postgres.UserRepository
}

// NewVariablesHandler creates a new VariablesHandler
func NewVariablesHandler(pool *pgxpool.Pool) *VariablesHandler {
	return &VariablesHandler{
		tenantRepo: postgres.NewTenantRepository(pool),
		userRepo:   postgres.NewUserRepository(pool),
	}
}

// VariablesResponse represents the response for variables
type VariablesResponse struct {
	Variables map[string]interface{} `json:"variables"`
}

// UpdateVariablesRequest represents the request body for updating variables
type UpdateVariablesRequest struct {
	Variables map[string]interface{} `json:"variables"`
}

// GetTenantVariables retrieves organization-level variables
func (h *VariablesHandler) GetTenantVariables(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	if tenantID == uuid.Nil {
		Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant ID not found", nil)
		return
	}

	variables, err := h.tenantRepo.GetVariables(r.Context(), tenantID)
	if err != nil {
		HandleErrorL(w, r, err)
		return
	}

	JSON(w, http.StatusOK, VariablesResponse{Variables: variables})
}

// UpdateTenantVariables updates organization-level variables
func (h *VariablesHandler) UpdateTenantVariables(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	if tenantID == uuid.Nil {
		Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant ID not found", nil)
		return
	}

	var req UpdateVariablesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body", nil)
		return
	}

	if req.Variables == nil {
		req.Variables = make(map[string]interface{})
	}

	if err := h.tenantRepo.UpdateVariables(r.Context(), tenantID, req.Variables); err != nil {
		HandleErrorL(w, r, err)
		return
	}

	JSON(w, http.StatusOK, VariablesResponse{Variables: req.Variables})
}

// GetUserVariables retrieves personal variables for the current user
func (h *VariablesHandler) GetUserVariables(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	userID := getUserID(r)

	if tenantID == uuid.Nil {
		Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant ID not found", nil)
		return
	}
	if userID == uuid.Nil {
		Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "user ID not found", nil)
		return
	}

	variables, err := h.userRepo.GetVariables(r.Context(), tenantID, userID)
	if err != nil {
		HandleErrorL(w, r, err)
		return
	}

	JSON(w, http.StatusOK, VariablesResponse{Variables: variables})
}

// UpdateUserVariables updates personal variables for the current user
func (h *VariablesHandler) UpdateUserVariables(w http.ResponseWriter, r *http.Request) {
	tenantID := getTenantID(r)
	userID := getUserID(r)

	if tenantID == uuid.Nil {
		Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant ID not found", nil)
		return
	}
	if userID == uuid.Nil {
		Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "user ID not found", nil)
		return
	}

	var req UpdateVariablesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body", nil)
		return
	}

	if req.Variables == nil {
		req.Variables = make(map[string]interface{})
	}

	if err := h.userRepo.UpdateVariables(r.Context(), tenantID, userID, req.Variables); err != nil {
		HandleErrorL(w, r, err)
		return
	}

	JSON(w, http.StatusOK, VariablesResponse{Variables: req.Variables})
}
