package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestRunHandler_Create_Validation(t *testing.T) {
	handler := NewRunHandler(nil)

	tests := []struct {
		name           string
		workflowID     string
		body           string
		tenantID       string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "invalid workflow ID",
			workflowID:     "not-a-uuid",
			body:           `{"input": {}, "mode": "test"}`,
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "empty workflow ID",
			workflowID:     "",
			body:           `{"input": {}, "mode": "test"}`,
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "invalid JSON body",
			workflowID:     uuid.New().String(),
			body:           `{invalid json`,
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "empty body",
			workflowID:     uuid.New().String(),
			body:           ``,
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/"+tt.workflowID+"/runs", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Tenant-ID", tt.tenantID)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.workflowID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rec := httptest.NewRecorder()
			handler.Create(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)

			var resp map[string]interface{}
			json.Unmarshal(rec.Body.Bytes(), &resp)
			if tt.expectedError != "" {
				assert.NotNil(t, resp["error"])
				errMap := resp["error"].(map[string]interface{})
				assert.Equal(t, tt.expectedError, errMap["code"])
			}
		})
	}
}

func TestRunHandler_List_Validation(t *testing.T) {
	handler := NewRunHandler(nil)

	tests := []struct {
		name           string
		workflowID     string
		tenantID       string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "invalid workflow ID",
			workflowID:     "not-a-uuid",
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "empty workflow ID",
			workflowID:     "",
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/"+tt.workflowID+"/runs", nil)
			req.Header.Set("X-Tenant-ID", tt.tenantID)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.workflowID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rec := httptest.NewRecorder()
			handler.List(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)

			var resp map[string]interface{}
			json.Unmarshal(rec.Body.Bytes(), &resp)
			if tt.expectedError != "" {
				assert.NotNil(t, resp["error"])
				errMap := resp["error"].(map[string]interface{})
				assert.Equal(t, tt.expectedError, errMap["code"])
			}
		})
	}
}

func TestRunHandler_Get_Validation(t *testing.T) {
	handler := NewRunHandler(nil)

	tests := []struct {
		name           string
		runID          string
		tenantID       string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "invalid run ID",
			runID:          "not-a-uuid",
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "empty run ID",
			runID:          "",
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/"+tt.runID, nil)
			req.Header.Set("X-Tenant-ID", tt.tenantID)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("run_id", tt.runID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rec := httptest.NewRecorder()
			handler.Get(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)

			var resp map[string]interface{}
			json.Unmarshal(rec.Body.Bytes(), &resp)
			if tt.expectedError != "" {
				assert.NotNil(t, resp["error"])
				errMap := resp["error"].(map[string]interface{})
				assert.Equal(t, tt.expectedError, errMap["code"])
			}
		})
	}
}

func TestRunHandler_Cancel_Validation(t *testing.T) {
	handler := NewRunHandler(nil)

	tests := []struct {
		name           string
		runID          string
		tenantID       string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "invalid run ID",
			runID:          "not-a-uuid",
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "empty run ID",
			runID:          "",
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/"+tt.runID+"/cancel", nil)
			req.Header.Set("X-Tenant-ID", tt.tenantID)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("run_id", tt.runID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rec := httptest.NewRecorder()
			handler.Cancel(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)

			var resp map[string]interface{}
			json.Unmarshal(rec.Body.Bytes(), &resp)
			if tt.expectedError != "" {
				assert.NotNil(t, resp["error"])
				errMap := resp["error"].(map[string]interface{})
				assert.Equal(t, tt.expectedError, errMap["code"])
			}
		})
	}
}

func TestRunHandler_ExecuteSingleStep_Validation(t *testing.T) {
	handler := NewRunHandler(nil)

	tests := []struct {
		name           string
		runID          string
		stepID         string
		body           string
		tenantID       string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "invalid run ID",
			runID:          "not-a-uuid",
			stepID:         uuid.New().String(),
			body:           `{}`,
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "invalid step ID",
			runID:          uuid.New().String(),
			stepID:         "not-a-uuid",
			body:           `{}`,
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "invalid JSON body",
			runID:          uuid.New().String(),
			stepID:         uuid.New().String(),
			body:           `{invalid`,
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/"+tt.runID+"/steps/"+tt.stepID+"/execute", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Tenant-ID", tt.tenantID)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("run_id", tt.runID)
			rctx.URLParams.Add("step_id", tt.stepID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rec := httptest.NewRecorder()
			handler.ExecuteSingleStep(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)

			var resp map[string]interface{}
			json.Unmarshal(rec.Body.Bytes(), &resp)
			if tt.expectedError != "" {
				assert.NotNil(t, resp["error"])
				errMap := resp["error"].(map[string]interface{})
				assert.Equal(t, tt.expectedError, errMap["code"])
			}
		})
	}
}

func TestRunHandler_ResumeFromStep_Validation(t *testing.T) {
	handler := NewRunHandler(nil)

	tests := []struct {
		name           string
		runID          string
		body           string
		tenantID       string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "invalid run ID",
			runID:          "not-a-uuid",
			body:           `{"from_step_id": "` + uuid.New().String() + `"}`,
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "invalid JSON body",
			runID:          uuid.New().String(),
			body:           `{invalid`,
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "invalid from_step_id",
			runID:          uuid.New().String(),
			body:           `{"from_step_id": "not-a-uuid"}`,
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "missing from_step_id",
			runID:          uuid.New().String(),
			body:           `{}`,
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/"+tt.runID+"/resume", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Tenant-ID", tt.tenantID)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("run_id", tt.runID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rec := httptest.NewRecorder()
			handler.ResumeFromStep(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)

			var resp map[string]interface{}
			json.Unmarshal(rec.Body.Bytes(), &resp)
			if tt.expectedError != "" {
				assert.NotNil(t, resp["error"])
				errMap := resp["error"].(map[string]interface{})
				assert.Equal(t, tt.expectedError, errMap["code"])
			}
		})
	}
}

func TestRunHandler_GetStepHistory_Validation(t *testing.T) {
	handler := NewRunHandler(nil)

	tests := []struct {
		name           string
		runID          string
		stepID         string
		tenantID       string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "invalid run ID",
			runID:          "not-a-uuid",
			stepID:         uuid.New().String(),
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "invalid step ID",
			runID:          uuid.New().String(),
			stepID:         "not-a-uuid",
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/"+tt.runID+"/steps/"+tt.stepID+"/history", nil)
			req.Header.Set("X-Tenant-ID", tt.tenantID)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("run_id", tt.runID)
			rctx.URLParams.Add("step_id", tt.stepID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rec := httptest.NewRecorder()
			handler.GetStepHistory(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)

			var resp map[string]interface{}
			json.Unmarshal(rec.Body.Bytes(), &resp)
			if tt.expectedError != "" {
				assert.NotNil(t, resp["error"])
				errMap := resp["error"].(map[string]interface{})
				assert.Equal(t, tt.expectedError, errMap["code"])
			}
		})
	}
}

func TestNewRunHandler(t *testing.T) {
	handler := NewRunHandler(nil)
	assert.NotNil(t, handler)
}
