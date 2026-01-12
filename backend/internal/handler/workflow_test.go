package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to set up test context with tenant ID
func setupTestContext(ctx context.Context, tenantID uuid.UUID) context.Context {
	ctx = context.WithValue(ctx, middleware.TenantIDKey, tenantID)
	ctx = context.WithValue(ctx, middleware.UserIDKey, uuid.New())
	ctx = context.WithValue(ctx, middleware.UserEmailKey, "test@example.com")
	ctx = context.WithValue(ctx, middleware.UserRolesKey, []string{"admin"})
	return ctx
}

// TestWorkflowHandler_Create_InvalidJSON tests Create handler with invalid JSON
func TestWorkflowHandler_Create_InvalidJSON(t *testing.T) {
	tenantID := uuid.New()
	handler := NewWorkflowHandler(nil) // usecase is nil but won't be called due to early validation failure

	tests := []struct {
		name           string
		body           string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "invalid JSON",
			body:           `{invalid json`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "malformed JSON",
			body:           `{"name": "test",}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			ctx := setupTestContext(req.Context(), tenantID)
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()
			handler.Create(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)

			var resp ErrorResponse
			err := json.Unmarshal(rec.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedError, resp.Error.Code)
		})
	}
}

// TestWorkflowHandler_Get_InvalidUUID tests Get handler with invalid UUID
func TestWorkflowHandler_Get_InvalidUUID(t *testing.T) {
	tenantID := uuid.New()
	handler := NewWorkflowHandler(nil)

	tests := []struct {
		name           string
		workflowID     string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "invalid UUID format",
			workflowID:     "not-a-uuid",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "partial UUID",
			workflowID:     "12345678",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Get("/api/v1/workflows/{id}", handler.Get)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/"+tt.workflowID, nil)
			ctx := setupTestContext(req.Context(), tenantID)
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)

			var resp ErrorResponse
			err := json.Unmarshal(rec.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedError, resp.Error.Code)
		})
	}
}

// TestWorkflowHandler_Update_Validation tests Update handler validation
func TestWorkflowHandler_Update_Validation(t *testing.T) {
	tenantID := uuid.New()
	workflowID := uuid.New()
	handler := NewWorkflowHandler(nil)

	tests := []struct {
		name           string
		workflowID     string
		body           string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "invalid UUID",
			workflowID:     "invalid-uuid",
			body:           `{"name": "Updated"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "invalid JSON body",
			workflowID:     workflowID.String(),
			body:           `{invalid`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Put("/api/v1/workflows/{id}", handler.Update)

			req := httptest.NewRequest(http.MethodPut, "/api/v1/workflows/"+tt.workflowID, bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			ctx := setupTestContext(req.Context(), tenantID)
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)

			var resp ErrorResponse
			err := json.Unmarshal(rec.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedError, resp.Error.Code)
		})
	}
}

// TestWorkflowHandler_Delete_InvalidUUID tests Delete handler with invalid UUID
func TestWorkflowHandler_Delete_InvalidUUID(t *testing.T) {
	tenantID := uuid.New()
	handler := NewWorkflowHandler(nil)

	r := chi.NewRouter()
	r.Delete("/api/v1/workflows/{id}", handler.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/workflows/invalid-uuid", nil)
	ctx := setupTestContext(req.Context(), tenantID)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp ErrorResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

// TestWorkflowHandler_Save_Validation tests Save handler validation
func TestWorkflowHandler_Save_Validation(t *testing.T) {
	tenantID := uuid.New()
	workflowID := uuid.New()
	stepID := uuid.New()
	edgeID := uuid.New()
	handler := NewWorkflowHandler(nil)

	tests := []struct {
		name           string
		workflowID     string
		body           string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "invalid workflow UUID",
			workflowID:     "invalid-uuid",
			body:           `{"name": "Test"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "invalid JSON body",
			workflowID:     workflowID.String(),
			body:           `{invalid`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:       "invalid step ID",
			workflowID: workflowID.String(),
			body: `{
				"name": "Test",
				"steps": [{"id": "invalid-step-id", "name": "Step 1", "type": "start"}],
				"edges": []
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:       "invalid edge ID",
			workflowID: workflowID.String(),
			body: `{
				"name": "Test",
				"steps": [{"id": "` + stepID.String() + `", "name": "Step 1", "type": "start"}],
				"edges": [{"id": "invalid-edge-id", "source_step_id": "` + stepID.String() + `", "target_step_id": "` + stepID.String() + `"}]
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:       "invalid source step ID in edge",
			workflowID: workflowID.String(),
			body: `{
				"name": "Test",
				"steps": [{"id": "` + stepID.String() + `", "name": "Step 1", "type": "start"}],
				"edges": [{"id": "` + edgeID.String() + `", "source_step_id": "invalid", "target_step_id": "` + stepID.String() + `"}]
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:       "invalid target step ID in edge",
			workflowID: workflowID.String(),
			body: `{
				"name": "Test",
				"steps": [{"id": "` + stepID.String() + `", "name": "Step 1", "type": "start"}],
				"edges": [{"id": "` + edgeID.String() + `", "source_step_id": "` + stepID.String() + `", "target_step_id": "invalid"}]
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Post("/api/v1/workflows/{id}/save", handler.Save)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/"+tt.workflowID+"/save", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			ctx := setupTestContext(req.Context(), tenantID)
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)

			var resp ErrorResponse
			err := json.Unmarshal(rec.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedError, resp.Error.Code)
		})
	}
}

// TestWorkflowHandler_SaveDraft_Validation tests SaveDraft handler validation
func TestWorkflowHandler_SaveDraft_Validation(t *testing.T) {
	tenantID := uuid.New()
	workflowID := uuid.New()
	handler := NewWorkflowHandler(nil)

	tests := []struct {
		name           string
		workflowID     string
		body           string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "invalid workflow UUID",
			workflowID:     "invalid-uuid",
			body:           `{"name": "Test"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "invalid JSON body",
			workflowID:     workflowID.String(),
			body:           `{invalid`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Post("/api/v1/workflows/{id}/draft", handler.SaveDraft)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/"+tt.workflowID+"/draft", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			ctx := setupTestContext(req.Context(), tenantID)
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)

			var resp ErrorResponse
			err := json.Unmarshal(rec.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedError, resp.Error.Code)
		})
	}
}

// TestWorkflowHandler_DiscardDraft_InvalidUUID tests DiscardDraft handler with invalid UUID
func TestWorkflowHandler_DiscardDraft_InvalidUUID(t *testing.T) {
	tenantID := uuid.New()
	handler := NewWorkflowHandler(nil)

	r := chi.NewRouter()
	r.Delete("/api/v1/workflows/{id}/draft", handler.DiscardDraft)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/workflows/invalid-uuid/draft", nil)
	ctx := setupTestContext(req.Context(), tenantID)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp ErrorResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

// TestWorkflowHandler_RestoreVersion_Validation tests RestoreVersion handler validation
func TestWorkflowHandler_RestoreVersion_Validation(t *testing.T) {
	tenantID := uuid.New()
	workflowID := uuid.New()
	handler := NewWorkflowHandler(nil)

	tests := []struct {
		name           string
		workflowID     string
		body           string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "invalid workflow UUID",
			workflowID:     "invalid-uuid",
			body:           `{"version": 1}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "invalid JSON body",
			workflowID:     workflowID.String(),
			body:           `{invalid`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "version less than 1",
			workflowID:     workflowID.String(),
			body:           `{"version": 0}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "negative version",
			workflowID:     workflowID.String(),
			body:           `{"version": -1}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Post("/api/v1/workflows/{id}/restore", handler.RestoreVersion)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/"+tt.workflowID+"/restore", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			ctx := setupTestContext(req.Context(), tenantID)
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)

			var resp ErrorResponse
			err := json.Unmarshal(rec.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedError, resp.Error.Code)
		})
	}
}

// TestWorkflowHandler_Publish_InvalidUUID tests Publish handler with invalid UUID
func TestWorkflowHandler_Publish_InvalidUUID(t *testing.T) {
	tenantID := uuid.New()
	handler := NewWorkflowHandler(nil)

	r := chi.NewRouter()
	r.Post("/api/v1/workflows/{id}/publish", handler.Publish)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/invalid-uuid/publish", nil)
	ctx := setupTestContext(req.Context(), tenantID)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp ErrorResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

// TestWorkflowHandler_ListVersions_InvalidUUID tests ListVersions handler with invalid UUID
func TestWorkflowHandler_ListVersions_InvalidUUID(t *testing.T) {
	tenantID := uuid.New()
	handler := NewWorkflowHandler(nil)

	r := chi.NewRouter()
	r.Get("/api/v1/workflows/{id}/versions", handler.ListVersions)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/invalid-uuid/versions", nil)
	ctx := setupTestContext(req.Context(), tenantID)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp ErrorResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

// TestWorkflowHandler_GetVersion_Validation tests GetVersion handler validation
func TestWorkflowHandler_GetVersion_Validation(t *testing.T) {
	tenantID := uuid.New()
	workflowID := uuid.New()
	handler := NewWorkflowHandler(nil)

	tests := []struct {
		name           string
		workflowID     string
		version        string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "invalid workflow UUID",
			workflowID:     "invalid-uuid",
			version:        "1",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "invalid version number",
			workflowID:     workflowID.String(),
			version:        "invalid",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "version zero",
			workflowID:     workflowID.String(),
			version:        "0",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "negative version",
			workflowID:     workflowID.String(),
			version:        "-1",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Get("/api/v1/workflows/{id}/versions/{version}", handler.GetVersion)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/"+tt.workflowID+"/versions/"+tt.version, nil)
			ctx := setupTestContext(req.Context(), tenantID)
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)

			var resp ErrorResponse
			err := json.Unmarshal(rec.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedError, resp.Error.Code)
		})
	}
}

// TestHandleError tests the error handling function
func TestHandleError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "workflow not found",
			err:            domain.ErrWorkflowNotFound,
			expectedStatus: http.StatusNotFound,
			expectedCode:   "NOT_FOUND",
		},
		{
			name:           "step not found",
			err:            domain.ErrStepNotFound,
			expectedStatus: http.StatusNotFound,
			expectedCode:   "NOT_FOUND",
		},
		{
			name:           "run not found",
			err:            domain.ErrRunNotFound,
			expectedStatus: http.StatusNotFound,
			expectedCode:   "NOT_FOUND",
		},
		{
			name:           "workflow already published",
			err:            domain.ErrWorkflowAlreadyPublished,
			expectedStatus: http.StatusConflict,
			expectedCode:   "CONFLICT",
		},
		{
			name:           "workflow has cycle",
			err:            domain.ErrWorkflowHasCycle,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "VALIDATION_ERROR",
		},
		{
			name:           "workflow has unconnected steps",
			err:            domain.ErrWorkflowHasUnconnected,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "VALIDATION_ERROR",
		},
		{
			name:           "unauthorized",
			err:            domain.ErrUnauthorized,
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "UNAUTHORIZED",
		},
		{
			name:           "forbidden",
			err:            domain.ErrForbidden,
			expectedStatus: http.StatusForbidden,
			expectedCode:   "FORBIDDEN",
		},
		{
			name:           "validation error",
			err:            domain.ValidationError{Field: "name", Message: "name is required"},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "VALIDATION_ERROR",
		},
		{
			name:           "credential not found",
			err:            domain.ErrCredentialNotFound,
			expectedStatus: http.StatusNotFound,
			expectedCode:   "NOT_FOUND",
		},
		{
			name:           "credential expired",
			err:            domain.ErrCredentialExpired,
			expectedStatus: http.StatusForbidden,
			expectedCode:   "CREDENTIAL_UNAVAILABLE",
		},
		{
			name:           "block definition not found",
			err:            domain.ErrBlockDefinitionNotFound,
			expectedStatus: http.StatusNotFound,
			expectedCode:   "NOT_FOUND",
		},
		{
			name:           "slug exists",
			err:            domain.ErrBlockDefinitionSlugExists,
			expectedStatus: http.StatusConflict,
			expectedCode:   "SLUG_EXISTS",
		},
		{
			name:           "unknown error",
			err:            errors.New("unknown error"),
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			HandleError(rec, tt.err)

			assert.Equal(t, tt.expectedStatus, rec.Code)

			var resp ErrorResponse
			err := json.Unmarshal(rec.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedCode, resp.Error.Code)
		})
	}
}

// TestJSONResponse tests JSON response helpers
func TestJSONResponse(t *testing.T) {
	t.Run("JSONData", func(t *testing.T) {
		rec := httptest.NewRecorder()
		data := map[string]string{"key": "value"}
		JSONData(rec, http.StatusOK, data)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

		var resp Response
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.NotNil(t, resp.Data)
	})

	t.Run("JSONList", func(t *testing.T) {
		rec := httptest.NewRecorder()
		data := []string{"item1", "item2"}
		JSONList(rec, http.StatusOK, data, 1, 10, 100)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp Response
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.NotNil(t, resp.Data)
		assert.NotNil(t, resp.Meta)
		assert.Equal(t, 1, resp.Meta.Page)
		assert.Equal(t, 10, resp.Meta.Limit)
		assert.Equal(t, 100, resp.Meta.Total)
	})

	t.Run("Error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		Error(rec, http.StatusBadRequest, "TEST_ERROR", "test message", map[string]string{"field": "test"})

		assert.Equal(t, http.StatusBadRequest, rec.Code)

		var resp ErrorResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "TEST_ERROR", resp.Error.Code)
		assert.Equal(t, "test message", resp.Error.Message)
	})
}

// TestParseIntQuery tests query parameter parsing
func TestParseIntQuery(t *testing.T) {
	tests := []struct {
		name         string
		queryString  string
		key          string
		defaultValue int
		expected     int
	}{
		{
			name:         "valid integer",
			queryString:  "page=5",
			key:          "page",
			defaultValue: 1,
			expected:     5,
		},
		{
			name:         "missing parameter",
			queryString:  "",
			key:          "page",
			defaultValue: 1,
			expected:     1,
		},
		{
			name:         "invalid integer",
			queryString:  "page=abc",
			key:          "page",
			defaultValue: 1,
			expected:     1,
		},
		{
			name:         "negative integer",
			queryString:  "page=-5",
			key:          "page",
			defaultValue: 1,
			expected:     -5,
		},
		{
			name:         "zero",
			queryString:  "page=0",
			key:          "page",
			defaultValue: 1,
			expected:     0,
		},
		{
			name:         "large number",
			queryString:  "limit=1000000",
			key:          "limit",
			defaultValue: 20,
			expected:     1000000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test?"+tt.queryString, nil)
			result := parseIntQuery(req, tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseIntFromString tests string to int parsing
func TestParseIntFromString(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		defaultValue int
		expected     int
	}{
		{
			name:         "valid integer",
			input:        "42",
			defaultValue: 0,
			expected:     42,
		},
		{
			name:         "empty string",
			input:        "",
			defaultValue: 10,
			expected:     10,
		},
		{
			name:         "invalid string",
			input:        "abc",
			defaultValue: 5,
			expected:     5,
		},
		{
			name:         "float string",
			input:        "3.14",
			defaultValue: 0,
			expected:     0,
		},
		{
			name:         "negative number",
			input:        "-10",
			defaultValue: 0,
			expected:     -10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseIntFromString(tt.input, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}
