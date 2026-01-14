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

// contextKey type for test context values (matches middleware package)
type testContextKey string

const (
	testTenantIDKey testContextKey = "tenantID"
	testUserIDKey   testContextKey = "userID"
)

// setTestContext sets tenant and user IDs in context for testing
func setTestContext(ctx context.Context) context.Context {
	ctx = context.WithValue(ctx, testTenantIDKey, uuid.New())
	ctx = context.WithValue(ctx, testUserIDKey, uuid.New())
	return ctx
}

// TestCopilotHandler_SuggestForStep_InvalidJSON tests SuggestForStep with invalid JSON body
func TestCopilotHandler_SuggestForStep_InvalidJSON(t *testing.T) {
	// Create handler with nil usecases (we won't reach them due to JSON error)
	handler := NewCopilotHandler(nil, nil)

	// Create request with invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/123/steps/456/copilot/suggest", bytes.NewBufferString("{invalid json"))
	req.Header.Set("Content-Type", "application/json")

	// Set up chi context with URL parameters
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", uuid.New().String())
	rctx.URLParams.Add("step_id", uuid.New().String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	handler.SuggestForStep(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&errResp)
	assert.NoError(t, err)
	assert.Equal(t, "INVALID_REQUEST", errResp["error"].(map[string]interface{})["code"])
}

// TestCopilotHandler_Suggest_InvalidJSON tests Suggest with invalid JSON body
func TestCopilotHandler_Suggest_InvalidJSON(t *testing.T) {
	handler := NewCopilotHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/copilot/suggest", bytes.NewBufferString("{invalid json"))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.Suggest(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&errResp)
	assert.NoError(t, err)
	assert.Equal(t, "INVALID_REQUEST", errResp["error"].(map[string]interface{})["code"])
}

// TestCopilotHandler_AsyncGenerateWorkflow_InvalidJSON tests AsyncGenerateWorkflow with invalid JSON body
func TestCopilotHandler_AsyncGenerateWorkflow_InvalidJSON(t *testing.T) {
	handler := NewCopilotHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/copilot/async/generate", bytes.NewBufferString("{invalid json"))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.AsyncGenerateWorkflow(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&errResp)
	assert.NoError(t, err)
	assert.Equal(t, "INVALID_REQUEST", errResp["error"].(map[string]interface{})["code"])
}

// TestCopilotHandler_AsyncSuggest_InvalidJSON tests AsyncSuggest with invalid JSON body
func TestCopilotHandler_AsyncSuggest_InvalidJSON(t *testing.T) {
	handler := NewCopilotHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/copilot/async/suggest", bytes.NewBufferString("{invalid json"))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.AsyncSuggest(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&errResp)
	assert.NoError(t, err)
	assert.Equal(t, "INVALID_REQUEST", errResp["error"].(map[string]interface{})["code"])
}

// TestCopilotHandler_AsyncDiagnose_InvalidJSON tests AsyncDiagnose with invalid JSON body
func TestCopilotHandler_AsyncDiagnose_InvalidJSON(t *testing.T) {
	handler := NewCopilotHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/copilot/async/diagnose", bytes.NewBufferString("{invalid json"))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.AsyncDiagnose(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&errResp)
	assert.NoError(t, err)
	assert.Equal(t, "INVALID_REQUEST", errResp["error"].(map[string]interface{})["code"])
}

// TestCopilotHandler_AsyncOptimize_InvalidJSON tests AsyncOptimize with invalid JSON body
func TestCopilotHandler_AsyncOptimize_InvalidJSON(t *testing.T) {
	handler := NewCopilotHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/copilot/async/optimize", bytes.NewBufferString("{invalid json"))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.AsyncOptimize(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&errResp)
	assert.NoError(t, err)
	assert.Equal(t, "INVALID_REQUEST", errResp["error"].(map[string]interface{})["code"])
}

// TestCopilotHandler_GenerateWorkflow_InvalidJSON tests GenerateWorkflow with invalid JSON body
func TestCopilotHandler_GenerateWorkflow_InvalidJSON(t *testing.T) {
	handler := NewCopilotHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/"+uuid.New().String()+"/generate", bytes.NewBufferString("{invalid json"))
	req.Header.Set("Content-Type", "application/json")

	// Set up chi context with URL parameters
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", uuid.New().String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	handler.GenerateWorkflow(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&errResp)
	assert.NoError(t, err)
	assert.Equal(t, "INVALID_REQUEST", errResp["error"].(map[string]interface{})["code"])
}

// TestCopilotHandler_Diagnose_InvalidJSON tests Diagnose with invalid JSON body
func TestCopilotHandler_Diagnose_InvalidJSON(t *testing.T) {
	handler := NewCopilotHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/copilot/diagnose", bytes.NewBufferString("{invalid json"))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.Diagnose(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&errResp)
	assert.NoError(t, err)
	assert.Equal(t, "INVALID_REQUEST", errResp["error"].(map[string]interface{})["code"])
}

// TestCopilotHandler_Optimize_InvalidJSON tests Optimize with invalid JSON body
func TestCopilotHandler_Optimize_InvalidJSON(t *testing.T) {
	handler := NewCopilotHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/copilot/optimize", bytes.NewBufferString("{invalid json"))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.Optimize(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&errResp)
	assert.NoError(t, err)
	assert.Equal(t, "INVALID_REQUEST", errResp["error"].(map[string]interface{})["code"])
}
