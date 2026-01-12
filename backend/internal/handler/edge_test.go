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

func TestEdgeHandler_Create_Validation(t *testing.T) {
	handler := NewEdgeHandler(nil)

	sourceID := uuid.New().String()
	targetID := uuid.New().String()

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
			body:           `{"source_step_id": "` + sourceID + `", "target_step_id": "` + targetID + `"}`,
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "empty workflow ID",
			workflowID:     "",
			body:           `{"source_step_id": "` + sourceID + `", "target_step_id": "` + targetID + `"}`,
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
			name:           "invalid source_step_id",
			workflowID:     uuid.New().String(),
			body:           `{"source_step_id": "not-a-uuid", "target_step_id": "` + targetID + `"}`,
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "invalid target_step_id",
			workflowID:     uuid.New().String(),
			body:           `{"source_step_id": "` + sourceID + `", "target_step_id": "not-a-uuid"}`,
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "missing source_step_id",
			workflowID:     uuid.New().String(),
			body:           `{"target_step_id": "` + targetID + `"}`,
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "missing target_step_id",
			workflowID:     uuid.New().String(),
			body:           `{"source_step_id": "` + sourceID + `"}`,
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
			req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/"+tt.workflowID+"/edges", bytes.NewBufferString(tt.body))
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

func TestEdgeHandler_List_Validation(t *testing.T) {
	handler := NewEdgeHandler(nil)

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
			req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/"+tt.workflowID+"/edges", nil)
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

func TestEdgeHandler_Delete_Validation(t *testing.T) {
	handler := NewEdgeHandler(nil)

	tests := []struct {
		name           string
		workflowID     string
		edgeID         string
		tenantID       string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "invalid workflow ID",
			workflowID:     "not-a-uuid",
			edgeID:         uuid.New().String(),
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "invalid edge ID",
			workflowID:     uuid.New().String(),
			edgeID:         "not-a-uuid",
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/workflows/"+tt.workflowID+"/edges/"+tt.edgeID, nil)
			req.Header.Set("X-Tenant-ID", tt.tenantID)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.workflowID)
			rctx.URLParams.Add("edge_id", tt.edgeID)
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

func TestNewEdgeHandler(t *testing.T) {
	handler := NewEdgeHandler(nil)
	assert.NotNil(t, handler)
}
