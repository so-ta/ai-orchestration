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

func TestStepHandler_Create_Validation(t *testing.T) {
	handler := NewStepHandler(nil)

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
			body:           `{"name": "Test", "type": "start", "config": {}}`,
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "empty workflow ID",
			workflowID:     "",
			body:           `{"name": "Test", "type": "start", "config": {}}`,
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
			req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/"+tt.workflowID+"/steps", bytes.NewBufferString(tt.body))
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

func TestStepHandler_List_Validation(t *testing.T) {
	handler := NewStepHandler(nil)

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
			req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/"+tt.workflowID+"/steps", nil)
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

func TestStepHandler_Update_Validation(t *testing.T) {
	handler := NewStepHandler(nil)

	tests := []struct {
		name           string
		workflowID     string
		stepID         string
		body           string
		tenantID       string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "invalid workflow ID",
			workflowID:     "not-a-uuid",
			stepID:         uuid.New().String(),
			body:           `{"name": "Updated", "type": "start", "config": {}}`,
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "invalid step ID",
			workflowID:     uuid.New().String(),
			stepID:         "not-a-uuid",
			body:           `{"name": "Updated", "type": "start", "config": {}}`,
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "invalid JSON body",
			workflowID:     uuid.New().String(),
			stepID:         uuid.New().String(),
			body:           `{invalid json`,
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "empty body",
			workflowID:     uuid.New().String(),
			stepID:         uuid.New().String(),
			body:           ``,
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/api/v1/workflows/"+tt.workflowID+"/steps/"+tt.stepID, bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Tenant-ID", tt.tenantID)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.workflowID)
			rctx.URLParams.Add("step_id", tt.stepID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rec := httptest.NewRecorder()
			handler.Update(rec, req)

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

func TestStepHandler_Delete_Validation(t *testing.T) {
	handler := NewStepHandler(nil)

	tests := []struct {
		name           string
		workflowID     string
		stepID         string
		tenantID       string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "invalid workflow ID",
			workflowID:     "not-a-uuid",
			stepID:         uuid.New().String(),
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "invalid step ID",
			workflowID:     uuid.New().String(),
			stepID:         "not-a-uuid",
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/workflows/"+tt.workflowID+"/steps/"+tt.stepID, nil)
			req.Header.Set("X-Tenant-ID", tt.tenantID)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.workflowID)
			rctx.URLParams.Add("step_id", tt.stepID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rec := httptest.NewRecorder()
			handler.Delete(rec, req)

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

func TestNewStepHandler(t *testing.T) {
	handler := NewStepHandler(nil)
	assert.NotNil(t, handler)
}
