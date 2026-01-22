package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/middleware"
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

// Error writes an error response (uses message as-is)
func Error(w http.ResponseWriter, status int, code, message string, details interface{}) {
	JSON(w, status, ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

// ErrorL writes a localized error response using the error code
// The message is looked up from domain.ErrorMessages based on the request's language
func ErrorL(w http.ResponseWriter, r *http.Request, status int, code string, details interface{}) {
	lang := middleware.GetLanguage(r.Context())
	message := domain.GetErrorMessage(lang, code)
	JSON(w, status, ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

// HandleErrorL converts domain errors to HTTP responses with localized messages
func HandleErrorL(w http.ResponseWriter, r *http.Request, err error) {
	lang := middleware.GetLanguage(r.Context())

	// Check for input schema validation errors first
	var inputValidationErrs *domain.InputValidationErrors
	if errors.As(err, &inputValidationErrs) {
		Error(w, http.StatusBadRequest, "SCHEMA_VALIDATION_ERROR",
			domain.GetErrorMessage(lang, "SCHEMA_VALIDATION_ERROR"),
			map[string]interface{}{"errors": inputValidationErrs.Errors})
		return
	}

	var validationErr domain.ValidationError
	if errors.As(err, &validationErr) {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", validationErr.Message, map[string]string{
			"field": validationErr.Field,
		})
		return
	}

	// Map domain errors to error codes
	type errorMapping struct {
		err    error
		status int
		code   string
	}

	notFoundErrors := []error{
		domain.ErrProjectNotFound, domain.ErrProjectVersionNotFound,
		domain.ErrStepNotFound, domain.ErrEdgeNotFound,
		domain.ErrRunNotFound, domain.ErrScheduleNotFound,
		domain.ErrTenantNotFound, domain.ErrBlockGroupNotFound,
		domain.ErrCredentialNotFound, domain.ErrSystemCredentialNotFound,
		domain.ErrBlockDefinitionNotFound, domain.ErrStepRunNotFound,
		domain.ErrOAuth2ProviderNotFound, domain.ErrOAuth2AppNotFound,
		domain.ErrOAuth2ConnectionNotFound, domain.ErrCredentialShareNotFound,
	}
	for _, e := range notFoundErrors {
		if errors.Is(err, e) {
			Error(w, http.StatusNotFound, "NOT_FOUND", domain.GetErrorMessage(lang, "NOT_FOUND"), nil)
			return
		}
	}

	switch {
	case errors.Is(err, domain.ErrRunNotCancellable):
		Error(w, http.StatusConflict, "RUN_NOT_CANCELLABLE", domain.GetErrorMessage(lang, "RUN_NOT_CANCELLABLE"), nil)
	case errors.Is(err, domain.ErrRunNotResumable):
		Error(w, http.StatusConflict, "RUN_NOT_RESUMABLE", domain.GetErrorMessage(lang, "RUN_NOT_RESUMABLE"), nil)
	case errors.Is(err, domain.ErrScheduleDisabled):
		Error(w, http.StatusConflict, "SCHEDULE_DISABLED", domain.GetErrorMessage(lang, "SCHEDULE_DISABLED"), nil)

	case errors.Is(err, domain.ErrCredentialExpired),
		errors.Is(err, domain.ErrCredentialRevoked),
		errors.Is(err, domain.ErrSystemCredentialExpired),
		errors.Is(err, domain.ErrSystemCredentialRevoked):
		Error(w, http.StatusForbidden, "CREDENTIAL_UNAVAILABLE", domain.GetErrorMessage(lang, "CREDENTIAL_UNAVAILABLE"), nil)

	case errors.Is(err, domain.ErrOAuth2TokenExpired):
		Error(w, http.StatusForbidden, "OAUTH2_TOKEN_EXPIRED", domain.GetErrorMessage(lang, "OAUTH2_TOKEN_EXPIRED"), nil)
	case errors.Is(err, domain.ErrOAuth2RefreshFailed):
		Error(w, http.StatusForbidden, "OAUTH2_REFRESH_FAILED", domain.GetErrorMessage(lang, "OAUTH2_REFRESH_FAILED"), nil)

	case errors.Is(err, domain.ErrOAuth2AppAlreadyExists):
		Error(w, http.StatusConflict, "OAUTH2_APP_ALREADY_EXISTS", domain.GetErrorMessage(lang, "OAUTH2_APP_ALREADY_EXISTS"), nil)
	case errors.Is(err, domain.ErrCredentialShareDuplicate):
		Error(w, http.StatusConflict, "ALREADY_EXISTS", domain.GetErrorMessage(lang, "ALREADY_EXISTS"), nil)

	case errors.Is(err, domain.ErrOAuth2InvalidState):
		Error(w, http.StatusBadRequest, "OAUTH2_INVALID_STATE", domain.GetErrorMessage(lang, "OAUTH2_INVALID_STATE"), nil)

	case errors.Is(err, domain.ErrCredentialAccessDenied):
		Error(w, http.StatusForbidden, "CREDENTIAL_ACCESS_DENIED", domain.GetErrorMessage(lang, "CREDENTIAL_ACCESS_DENIED"), nil)
	case errors.Is(err, domain.ErrCredentialBindingMissing):
		Error(w, http.StatusForbidden, "CREDENTIAL_BINDING_MISSING", domain.GetErrorMessage(lang, "CREDENTIAL_BINDING_MISSING"), nil)
	case errors.Is(err, domain.ErrCredentialInvalidScope):
		Error(w, http.StatusForbidden, "CREDENTIAL_INVALID_SCOPE", domain.GetErrorMessage(lang, "CREDENTIAL_INVALID_SCOPE"), nil)

	case errors.Is(err, domain.ErrBlockDefinitionSlugExists):
		Error(w, http.StatusConflict, "BLOCK_SLUG_EXISTS", domain.GetErrorMessage(lang, "BLOCK_SLUG_EXISTS"), nil)
	case errors.Is(err, domain.ErrBlockCodeHidden):
		Error(w, http.StatusForbidden, "BLOCK_CODE_HIDDEN", domain.GetErrorMessage(lang, "BLOCK_CODE_HIDDEN"), nil)

	case errors.Is(err, domain.ErrProjectAlreadyPublished):
		Error(w, http.StatusConflict, "PROJECT_ALREADY_PUBLISHED", domain.GetErrorMessage(lang, "PROJECT_ALREADY_PUBLISHED"), nil)
	case errors.Is(err, domain.ErrProjectNotEditable):
		Error(w, http.StatusConflict, "PROJECT_NOT_EDITABLE", domain.GetErrorMessage(lang, "PROJECT_NOT_EDITABLE"), nil)
	case errors.Is(err, domain.ErrEdgeDuplicate):
		Error(w, http.StatusConflict, "EDGE_DUPLICATE", domain.GetErrorMessage(lang, "EDGE_DUPLICATE"), nil)

	case errors.Is(err, domain.ErrProjectHasCycle):
		Error(w, http.StatusBadRequest, "PROJECT_HAS_CYCLE", domain.GetErrorMessage(lang, "PROJECT_HAS_CYCLE"), nil)
	case errors.Is(err, domain.ErrProjectHasUnconnected):
		Error(w, http.StatusBadRequest, "PROJECT_HAS_UNCONNECTED", domain.GetErrorMessage(lang, "PROJECT_HAS_UNCONNECTED"), nil)
	case errors.Is(err, domain.ErrProjectHasUnreachable):
		Error(w, http.StatusBadRequest, "PROJECT_HAS_UNREACHABLE", domain.GetErrorMessage(lang, "PROJECT_HAS_UNREACHABLE"), nil)
	case errors.Is(err, domain.ErrEdgeSelfLoop):
		Error(w, http.StatusBadRequest, "EDGE_SELF_LOOP", domain.GetErrorMessage(lang, "EDGE_SELF_LOOP"), nil)
	case errors.Is(err, domain.ErrEdgeCreatesCycle):
		Error(w, http.StatusBadRequest, "EDGE_CREATES_CYCLE", domain.GetErrorMessage(lang, "EDGE_CREATES_CYCLE"), nil)
	case errors.Is(err, domain.ErrInvalidStepType):
		Error(w, http.StatusBadRequest, "INVALID_STEP_TYPE", domain.GetErrorMessage(lang, "INVALID_STEP_TYPE"), nil)
	case errors.Is(err, domain.ErrScheduleInvalidCron):
		Error(w, http.StatusBadRequest, "SCHEDULE_INVALID_CRON", domain.GetErrorMessage(lang, "SCHEDULE_INVALID_CRON"), nil)
	case errors.Is(err, domain.ErrBlockGroupInvalidType):
		Error(w, http.StatusBadRequest, "BLOCK_GROUP_INVALID_TYPE", domain.GetErrorMessage(lang, "BLOCK_GROUP_INVALID_TYPE"), nil)
	case errors.Is(err, domain.ErrStepCannotBeInGroup):
		Error(w, http.StatusBadRequest, "STEP_CANNOT_BE_IN_GROUP", domain.GetErrorMessage(lang, "STEP_CANNOT_BE_IN_GROUP"), nil)
	case errors.Is(err, domain.ErrBlockGroupInvalidRole):
		Error(w, http.StatusBadRequest, "BLOCK_GROUP_INVALID_ROLE", domain.GetErrorMessage(lang, "BLOCK_GROUP_INVALID_ROLE"), nil)

	case errors.Is(err, domain.ErrUnauthorized):
		Error(w, http.StatusUnauthorized, "UNAUTHORIZED", domain.GetErrorMessage(lang, "UNAUTHORIZED"), nil)
	case errors.Is(err, domain.ErrForbidden):
		Error(w, http.StatusForbidden, "FORBIDDEN", domain.GetErrorMessage(lang, "FORBIDDEN"), nil)

	default:
		slog.Error("internal error", "error", err)
		Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", domain.GetErrorMessage(lang, "INTERNAL_ERROR"), nil)
	}
}
