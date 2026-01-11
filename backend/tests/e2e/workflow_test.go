package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	baseURL  = getEnv("API_BASE_URL", "http://localhost:8080")
	tenantID = getEnv("TENANT_ID", "00000000-0000-0000-0000-000000000001")
)

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

type Workflow struct {
	ID          string `json:"id"`
	TenantID    string `json:"tenant_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Version     int    `json:"version"`
}

type Step struct {
	ID         string          `json:"id"`
	WorkflowID string          `json:"workflow_id"`
	Name       string          `json:"name"`
	Type       string          `json:"type"`
	Config     json.RawMessage `json:"config"`
}

type Run struct {
	ID         string `json:"id"`
	WorkflowID string `json:"workflow_id"`
	Status     string `json:"status"`
	Mode       string `json:"mode"`
}

func makeRequest(t *testing.T, method, path string, body interface{}) (*http.Response, []byte) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		require.NoError(t, err)
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, baseURL+path, reqBody)
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantID)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	return resp, respBody
}

func TestHealthCheck(t *testing.T) {
	resp, body := makeRequest(t, "GET", "/health", nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(t, err)
	assert.Equal(t, "ok", result["status"])
}

func TestWorkflowCRUD(t *testing.T) {
	// Create workflow
	createReq := map[string]string{
		"name":        "E2E Test Workflow " + time.Now().Format("20060102150405"),
		"description": "Created by E2E test",
	}
	resp, body := makeRequest(t, "POST", "/api/v1/workflows", createReq)
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "Create response: %s", string(body))

	var createResp struct {
		Data Workflow `json:"data"`
	}
	err := json.Unmarshal(body, &createResp)
	require.NoError(t, err)
	workflowID := createResp.Data.ID
	assert.NotEmpty(t, workflowID)
	assert.Equal(t, "draft", createResp.Data.Status)

	// Get workflow
	resp, body = makeRequest(t, "GET", "/api/v1/workflows/"+workflowID, nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Update workflow
	updateReq := map[string]string{
		"name":        createReq["name"] + " (Updated)",
		"description": "Updated by E2E test",
	}
	resp, body = makeRequest(t, "PUT", "/api/v1/workflows/"+workflowID, updateReq)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// List workflows
	resp, body = makeRequest(t, "GET", "/api/v1/workflows", nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Delete workflow
	resp, _ = makeRequest(t, "DELETE", "/api/v1/workflows/"+workflowID, nil)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify deletion
	resp, _ = makeRequest(t, "GET", "/api/v1/workflows/"+workflowID, nil)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func getWorkflowWithSteps(t *testing.T, workflowID string) (string, []Step) {
	resp, body := makeRequest(t, "GET", "/api/v1/workflows/"+workflowID, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var getResp struct {
		Data struct {
			ID    string `json:"id"`
			Steps []Step `json:"steps"`
		} `json:"data"`
	}
	err := json.Unmarshal(body, &getResp)
	require.NoError(t, err)
	return getResp.Data.ID, getResp.Data.Steps
}

func findStartStep(steps []Step) *Step {
	for _, step := range steps {
		if step.Type == "start" {
			return &step
		}
	}
	return nil
}

func TestWorkflowExecutionFlow(t *testing.T) {
	// 1. Create workflow (auto-creates Start step)
	createReq := map[string]string{
		"name":        "Execution Flow Test " + time.Now().Format("20060102150405"),
		"description": "Testing full execution flow",
	}
	resp, body := makeRequest(t, "POST", "/api/v1/workflows", createReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Create response: %s", string(body))

	var createResp struct {
		Data Workflow `json:"data"`
	}
	err := json.Unmarshal(body, &createResp)
	require.NoError(t, err)
	workflowID := createResp.Data.ID

	// Get the auto-created Start step
	_, steps := getWorkflowWithSteps(t, workflowID)
	startStep := findStartStep(steps)
	require.NotNil(t, startStep, "Workflow should have auto-created Start step")

	// 2. Add step with mock adapter
	stepReq := map[string]interface{}{
		"name": "Mock Step",
		"type": "tool",
		"config": map[string]string{
			"adapter_id": "mock",
		},
	}
	resp, body = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/steps", workflowID), stepReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Step create response: %s", string(body))

	var stepResp struct {
		Data Step `json:"data"`
	}
	err = json.Unmarshal(body, &stepResp)
	require.NoError(t, err)
	mockStepID := stepResp.Data.ID

	// 3. Connect Start step to Mock step
	edgeReq := map[string]string{
		"source_step_id": startStep.ID,
		"target_step_id": mockStepID,
	}
	resp, body = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/edges", workflowID), edgeReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Edge create response: %s", string(body))

	// 4. Publish workflow
	resp, body = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/publish", workflowID), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Publish response: %s", string(body))

	var publishResp struct {
		Data Workflow `json:"data"`
	}
	err = json.Unmarshal(body, &publishResp)
	require.NoError(t, err)
	assert.Equal(t, "published", publishResp.Data.Status)

	// 4. Execute workflow
	runReq := map[string]interface{}{
		"input": map[string]string{
			"message": "Hello from E2E test",
		},
		"mode": "test",
	}
	resp, body = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/runs", workflowID), runReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Run create response: %s", string(body))

	var runResp struct {
		Data Run `json:"data"`
	}
	err = json.Unmarshal(body, &runResp)
	require.NoError(t, err)
	runID := runResp.Data.ID
	assert.NotEmpty(t, runID)

	// 5. Wait for completion and check status
	maxWaitTime := 30 * time.Second
	pollInterval := 1 * time.Second
	deadline := time.Now().Add(maxWaitTime)

	var finalStatus string
	for time.Now().Before(deadline) {
		resp, body = makeRequest(t, "GET", fmt.Sprintf("/api/v1/runs/%s", runID), nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var getRunResp struct {
			Data Run `json:"data"`
		}
		err = json.Unmarshal(body, &getRunResp)
		require.NoError(t, err)

		finalStatus = getRunResp.Data.Status
		if finalStatus == "completed" || finalStatus == "failed" {
			break
		}
		time.Sleep(pollInterval)
	}

	assert.Equal(t, "completed", finalStatus, "Workflow should complete successfully")

	// Cleanup
	makeRequest(t, "DELETE", "/api/v1/workflows/"+workflowID, nil)
}

func TestConditionBranching(t *testing.T) {
	// Create workflow with condition step (auto-creates Start step)
	createReq := map[string]string{
		"name":        "Condition Test " + time.Now().Format("20060102150405"),
		"description": "Testing condition branching",
	}
	resp, body := makeRequest(t, "POST", "/api/v1/workflows", createReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp struct {
		Data Workflow `json:"data"`
	}
	json.Unmarshal(body, &createResp)
	workflowID := createResp.Data.ID

	// Get the auto-created Start step
	_, steps := getWorkflowWithSteps(t, workflowID)
	startStep := findStartStep(steps)
	require.NotNil(t, startStep, "Workflow should have auto-created Start step")

	// Add condition step
	condStepReq := map[string]interface{}{
		"name": "Check Value",
		"type": "condition",
		"config": map[string]string{
			"expression": "$.value > 10",
		},
	}
	resp, body = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/steps", workflowID), condStepReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var condStepResp struct {
		Data Step `json:"data"`
	}
	json.Unmarshal(body, &condStepResp)

	// Connect Start step to Condition step
	edgeReq := map[string]string{
		"source_step_id": startStep.ID,
		"target_step_id": condStepResp.Data.ID,
	}
	resp, _ = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/edges", workflowID), edgeReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Publish
	resp, body = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/publish", workflowID), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Publish response: %s", string(body))

	// Execute with value > 10
	runReq := map[string]interface{}{
		"input": map[string]int{"value": 20},
		"mode":  "test",
	}
	resp, body = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/runs", workflowID), runReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var runResp struct {
		Data Run `json:"data"`
	}
	json.Unmarshal(body, &runResp)

	// Wait for completion
	time.Sleep(3 * time.Second)
	resp, body = makeRequest(t, "GET", fmt.Sprintf("/api/v1/runs/%s", runResp.Data.ID), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var getRunResp struct {
		Data Run `json:"data"`
	}
	json.Unmarshal(body, &getRunResp)
	assert.Equal(t, "completed", getRunResp.Data.Status)

	// Cleanup
	makeRequest(t, "DELETE", "/api/v1/workflows/"+workflowID, nil)
}

func TestScheduleManagement(t *testing.T) {
	// Skip this test in environments where users table FK constraint is enforced
	// TODO: Setup proper test user in the database
	t.Skip("Skipping: requires valid user in users table for created_by FK constraint")

	// First create a workflow to schedule (auto-creates Start step)
	createReq := map[string]string{
		"name":        "Schedule Test Workflow " + time.Now().Format("20060102150405"),
		"description": "For schedule testing",
	}
	resp, body := makeRequest(t, "POST", "/api/v1/workflows", createReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp struct {
		Data Workflow `json:"data"`
	}
	json.Unmarshal(body, &createResp)
	workflowID := createResp.Data.ID

	// Get the auto-created Start step
	_, steps := getWorkflowWithSteps(t, workflowID)
	startStep := findStartStep(steps)
	require.NotNil(t, startStep)

	// Add a step
	stepReq := map[string]interface{}{
		"name":   "Mock Step",
		"type":   "tool",
		"config": map[string]string{"adapter_id": "mock"},
	}
	resp, body = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/steps", workflowID), stepReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var stepResp struct {
		Data Step `json:"data"`
	}
	json.Unmarshal(body, &stepResp)

	// Connect Start to step
	edgeReq := map[string]string{
		"source_step_id": startStep.ID,
		"target_step_id": stepResp.Data.ID,
	}
	makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/edges", workflowID), edgeReq)

	// Publish
	makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/publish", workflowID), nil)

	// Create schedule
	scheduleReq := map[string]interface{}{
		"workflow_id":     workflowID,
		"name":            "Test Schedule",
		"cron_expression": "0 0 * * *", // Daily at midnight
		"input":           map[string]string{"source": "schedule"},
	}
	resp, body = makeRequest(t, "POST", "/api/v1/schedules", scheduleReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Schedule create response: %s", string(body))

	var scheduleResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(body, &scheduleResp)
	scheduleID := scheduleResp.Data.ID
	assert.NotEmpty(t, scheduleID)

	// List schedules
	resp, _ = makeRequest(t, "GET", "/api/v1/schedules", nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Get schedule
	resp, _ = makeRequest(t, "GET", fmt.Sprintf("/api/v1/schedules/%s", scheduleID), nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Pause schedule
	resp, _ = makeRequest(t, "POST", fmt.Sprintf("/api/v1/schedules/%s/pause", scheduleID), nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Resume schedule
	resp, _ = makeRequest(t, "POST", fmt.Sprintf("/api/v1/schedules/%s/resume", scheduleID), nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Delete schedule
	resp, _ = makeRequest(t, "DELETE", fmt.Sprintf("/api/v1/schedules/%s", scheduleID), nil)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Cleanup workflow
	makeRequest(t, "DELETE", "/api/v1/workflows/"+workflowID, nil)
}

func TestWebhookManagement(t *testing.T) {
	// First create a workflow (auto-creates Start step)
	createReq := map[string]string{
		"name":        "Webhook Test Workflow " + time.Now().Format("20060102150405"),
		"description": "For webhook testing",
	}
	resp, body := makeRequest(t, "POST", "/api/v1/workflows", createReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp struct {
		Data Workflow `json:"data"`
	}
	json.Unmarshal(body, &createResp)
	workflowID := createResp.Data.ID

	// Get the auto-created Start step
	_, steps := getWorkflowWithSteps(t, workflowID)
	startStep := findStartStep(steps)
	require.NotNil(t, startStep, "Workflow should have auto-created Start step")

	// Add step
	stepReq := map[string]interface{}{
		"name":   "Mock Step",
		"type":   "tool",
		"config": map[string]string{"adapter_id": "mock"},
	}
	resp, body = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/steps", workflowID), stepReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var stepResp struct {
		Data Step `json:"data"`
	}
	json.Unmarshal(body, &stepResp)

	// Connect Start to step
	edgeReq := map[string]string{
		"source_step_id": startStep.ID,
		"target_step_id": stepResp.Data.ID,
	}
	resp, body = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/edges", workflowID), edgeReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Edge create response: %s", string(body))

	// Publish
	resp, body = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/publish", workflowID), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Publish response: %s", string(body))

	// Create webhook
	webhookReq := map[string]interface{}{
		"workflow_id": workflowID,
		"name":        "Test Webhook",
	}
	resp, body = makeRequest(t, "POST", "/api/v1/webhooks", webhookReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Webhook create response: %s", string(body))

	// Response is directly the webhook object (not wrapped in data)
	var webhook struct {
		ID     string `json:"id"`
		Secret string `json:"secret"`
	}
	json.Unmarshal(body, &webhook)
	webhookID := webhook.ID
	assert.NotEmpty(t, webhookID)
	assert.NotEmpty(t, webhook.Secret)
	originalSecret := webhook.Secret

	// List webhooks
	resp, _ = makeRequest(t, "GET", "/api/v1/webhooks", nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Get webhook
	resp, _ = makeRequest(t, "GET", fmt.Sprintf("/api/v1/webhooks/%s", webhookID), nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Disable webhook
	resp, _ = makeRequest(t, "POST", fmt.Sprintf("/api/v1/webhooks/%s/disable", webhookID), nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Enable webhook
	resp, _ = makeRequest(t, "POST", fmt.Sprintf("/api/v1/webhooks/%s/enable", webhookID), nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Regenerate secret
	resp, body = makeRequest(t, "POST", fmt.Sprintf("/api/v1/webhooks/%s/regenerate-secret", webhookID), nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var newWebhook struct {
		Secret string `json:"secret"`
	}
	json.Unmarshal(body, &newWebhook)
	assert.NotEmpty(t, newWebhook.Secret)
	assert.NotEqual(t, originalSecret, newWebhook.Secret)

	// Delete webhook
	resp, _ = makeRequest(t, "DELETE", fmt.Sprintf("/api/v1/webhooks/%s", webhookID), nil)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Cleanup workflow
	makeRequest(t, "DELETE", "/api/v1/workflows/"+workflowID, nil)
}
