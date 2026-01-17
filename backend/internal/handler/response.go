package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/souta/ai-orchestration/internal/domain"
)

// Response represents a standard API response
type Response struct {
	Data interface{} `json:"data,omitempty"`
	Meta *Meta       `json:"meta,omitempty"`
}

// Meta contains metadata about the response
type Meta struct {
	Page  int `json:"page,omitempty"`
	Limit int `json:"limit,omitempty"`
	Total int `json:"total,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error details
type ErrorDetail struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// JSON writes a JSON response
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// WriteHeader has already been called, so we cannot send an error response
		// Log the error instead
		slog.Error("failed to encode JSON response", "error", err)
	}
}

// JSONData writes a data response
func JSONData(w http.ResponseWriter, status int, data interface{}) {
	JSON(w, status, Response{Data: data})
}

// JSONList writes a paginated list response
func JSONList(w http.ResponseWriter, status int, data interface{}, page, limit, total int) {
	JSON(w, status, Response{
		Data: data,
		Meta: &Meta{
			Page:  page,
			Limit: limit,
			Total: total,
		},
	})
}

// Error writes an error response
func Error(w http.ResponseWriter, status int, code, message string, details interface{}) {
	JSON(w, status, ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

// HandleError converts domain errors to HTTP responses
func HandleError(w http.ResponseWriter, err error) {
	// Check for input schema validation errors first
	var inputValidationErrs *domain.InputValidationErrors
	if errors.As(err, &inputValidationErrs) {
		Error(w, http.StatusBadRequest, "SCHEMA_VALIDATION_ERROR", "Input validation failed", map[string]interface{}{
			"errors": inputValidationErrs.Errors,
		})
		return
	}

	var validationErr domain.ValidationError
	if errors.As(err, &validationErr) {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", validationErr.Message, map[string]string{
			"field": validationErr.Field,
		})
		return
	}

	switch {
	case errors.Is(err, domain.ErrProjectNotFound),
		errors.Is(err, domain.ErrProjectVersionNotFound),
		errors.Is(err, domain.ErrStepNotFound),
		errors.Is(err, domain.ErrEdgeNotFound),
		errors.Is(err, domain.ErrRunNotFound),
		errors.Is(err, domain.ErrScheduleNotFound),
		errors.Is(err, domain.ErrWebhookNotFound),
		errors.Is(err, domain.ErrTenantNotFound),
		errors.Is(err, domain.ErrBlockGroupNotFound),
		errors.Is(err, domain.ErrCredentialNotFound),
		errors.Is(err, domain.ErrSystemCredentialNotFound),
		errors.Is(err, domain.ErrBlockDefinitionNotFound),
		errors.Is(err, domain.ErrStepRunNotFound):
		Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)

	case errors.Is(err, domain.ErrRunNotCancellable),
		errors.Is(err, domain.ErrRunNotResumable),
		errors.Is(err, domain.ErrScheduleDisabled):
		Error(w, http.StatusConflict, "INVALID_STATE", err.Error(), nil)

	case errors.Is(err, domain.ErrCredentialExpired),
		errors.Is(err, domain.ErrCredentialRevoked),
		errors.Is(err, domain.ErrSystemCredentialExpired),
		errors.Is(err, domain.ErrSystemCredentialRevoked):
		Error(w, http.StatusForbidden, "CREDENTIAL_UNAVAILABLE", err.Error(), nil)

	case errors.Is(err, domain.ErrBlockDefinitionSlugExists):
		Error(w, http.StatusConflict, "SLUG_EXISTS", err.Error(), nil)

	case errors.Is(err, domain.ErrBlockCodeHidden):
		Error(w, http.StatusForbidden, "CODE_HIDDEN", err.Error(), nil)

	case errors.Is(err, domain.ErrProjectAlreadyPublished),
		errors.Is(err, domain.ErrProjectNotEditable),
		errors.Is(err, domain.ErrEdgeDuplicate):
		Error(w, http.StatusConflict, "CONFLICT", err.Error(), nil)

	case errors.Is(err, domain.ErrProjectHasCycle),
		errors.Is(err, domain.ErrProjectHasUnconnected),
		errors.Is(err, domain.ErrProjectHasUnreachable),
		errors.Is(err, domain.ErrEdgeSelfLoop),
		errors.Is(err, domain.ErrEdgeCreatesCycle),
		errors.Is(err, domain.ErrInvalidStepType),
		errors.Is(err, domain.ErrScheduleInvalidCron),
		errors.Is(err, domain.ErrBlockGroupInvalidType),
		errors.Is(err, domain.ErrStepCannotBeInGroup),
		errors.Is(err, domain.ErrBlockGroupInvalidRole):
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)

	case errors.Is(err, domain.ErrUnauthorized):
		Error(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error(), nil)

	case errors.Is(err, domain.ErrForbidden):
		Error(w, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)

	default:
		slog.Error("internal error", "error", err)
		Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
	}
}
