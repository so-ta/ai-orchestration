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

func TestBlockGroupHandler_Create_Validation(t *testing.T) {
	handler := NewBlockGroupHandler(nil)

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
			body:           `{"name": "Test Group", "type": "parallel", "position": {"x": 0, "y": 0}, "size": {"width": 400, "height": 300}}`,
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "empty workflow ID",
			workflowID:     "",
			body:           `{"name": "Test Group", "type": "parallel", "position": {"x": 0, "y": 0}, "size": {"width": 400, "height": 300}}`,
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
			req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/"+tt.workflowID+"/block-groups", bytes.NewBufferString(tt.body))
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

func TestBlockGroupHandler_List_Validation(t *testing.T) {
	handler := NewBlockGroupHandler(nil)

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
			req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/"+tt.workflowID+"/block-groups", nil)
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

func TestBlockGroupHandler_Get_Validation(t *testing.T) {
	handler := NewBlockGroupHandler(nil)

	tests := []struct {
		name           string
		workflowID     string
		groupID        string
		tenantID       string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "invalid workflow ID",
			workflowID:     "not-a-uuid",
			groupID:        uuid.New().String(),
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "invalid group ID",
			workflowID:     uuid.New().String(),
			groupID:        "not-a-uuid",
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/"+tt.workflowID+"/block-groups/"+tt.groupID, nil)
			req.Header.Set("X-Tenant-ID", tt.tenantID)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.workflowID)
			rctx.URLParams.Add("group_id", tt.groupID)
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

func TestBlockGroupHandler_Update_Validation(t *testing.T) {
	handler := NewBlockGroupHandler(nil)

	tests := []struct {
		name           string
		workflowID     string
		groupID        string
		body           string
		tenantID       string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "invalid workflow ID",
			workflowID:     "not-a-uuid",
			groupID:        uuid.New().String(),
			body:           `{"name": "Updated Group"}`,
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "invalid group ID",
			workflowID:     uuid.New().String(),
			groupID:        "not-a-uuid",
			body:           `{"name": "Updated Group"}`,
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "invalid JSON body",
			workflowID:     uuid.New().String(),
			groupID:        uuid.New().String(),
			body:           `{invalid json`,
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "empty body",
			workflowID:     uuid.New().String(),
			groupID:        uuid.New().String(),
			body:           ``,
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/api/v1/workflows/"+tt.workflowID+"/block-groups/"+tt.groupID, bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Tenant-ID", tt.tenantID)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.workflowID)
			rctx.URLParams.Add("group_id", tt.groupID)
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

func TestBlockGroupHandler_Delete_Validation(t *testing.T) {
	handler := NewBlockGroupHandler(nil)

	tests := []struct {
		name           string
		workflowID     string
		groupID        string
		tenantID       string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "invalid workflow ID",
			workflowID:     "not-a-uuid",
			groupID:        uuid.New().String(),
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "invalid group ID",
			workflowID:     uuid.New().String(),
			groupID:        "not-a-uuid",
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/workflows/"+tt.workflowID+"/block-groups/"+tt.groupID, nil)
			req.Header.Set("X-Tenant-ID", tt.tenantID)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.workflowID)
			rctx.URLParams.Add("group_id", tt.groupID)
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

func TestBlockGroupHandler_AddStepToGroup_Validation(t *testing.T) {
	handler := NewBlockGroupHandler(nil)

	tests := []struct {
		name           string
		workflowID     string
		groupID        string
		body           string
		tenantID       string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "invalid workflow ID",
			workflowID:     "not-a-uuid",
			groupID:        uuid.New().String(),
			body:           `{"step_id": "` + uuid.New().String() + `", "group_role": "body"}`,
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "invalid group ID",
			workflowID:     uuid.New().String(),
			groupID:        "not-a-uuid",
			body:           `{"step_id": "` + uuid.New().String() + `", "group_role": "body"}`,
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "invalid step ID",
			workflowID:     uuid.New().String(),
			groupID:        uuid.New().String(),
			body:           `{"step_id": "not-a-uuid", "group_role": "body"}`,
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "invalid JSON body",
			workflowID:     uuid.New().String(),
			groupID:        uuid.New().String(),
			body:           `{invalid json`,
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/"+tt.workflowID+"/block-groups/"+tt.groupID+"/steps", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Tenant-ID", tt.tenantID)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.workflowID)
			rctx.URLParams.Add("group_id", tt.groupID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rec := httptest.NewRecorder()
			handler.AddStepToGroup(rec, req)

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

func TestBlockGroupHandler_RemoveStepFromGroup_Validation(t *testing.T) {
	handler := NewBlockGroupHandler(nil)

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
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/workflows/"+tt.workflowID+"/block-groups/"+uuid.New().String()+"/steps/"+tt.stepID, nil)
			req.Header.Set("X-Tenant-ID", tt.tenantID)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.workflowID)
			rctx.URLParams.Add("step_id", tt.stepID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rec := httptest.NewRecorder()
			handler.RemoveStepFromGroup(rec, req)

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

func TestBlockGroupHandler_GetStepsByGroup_Validation(t *testing.T) {
	handler := NewBlockGroupHandler(nil)

	tests := []struct {
		name           string
		workflowID     string
		groupID        string
		tenantID       string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "invalid workflow ID",
			workflowID:     "not-a-uuid",
			groupID:        uuid.New().String(),
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "invalid group ID",
			workflowID:     uuid.New().String(),
			groupID:        "not-a-uuid",
			tenantID:       uuid.New().String(),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/"+tt.workflowID+"/block-groups/"+tt.groupID+"/steps", nil)
			req.Header.Set("X-Tenant-ID", tt.tenantID)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.workflowID)
			rctx.URLParams.Add("group_id", tt.groupID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rec := httptest.NewRecorder()
			handler.GetStepsByGroup(rec, req)

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

func TestNewBlockGroupHandler(t *testing.T) {
	handler := NewBlockGroupHandler(nil)
	assert.NotNil(t, handler)
}
