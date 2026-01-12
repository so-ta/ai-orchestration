package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to create a workflow for run tests
func createTestWorkflowForRuns(t *testing.T) string {
	createReq := map[string]string{
		"name":        "Run Test Workflow " + time.Now().Format("20060102150405"),
		"description": "Created for run tests",
	}
	resp, body := makeRequest(t, "POST", "/api/v1/workflows", createReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

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

	// Add a mock step
	stepReq := map[string]interface{}{
		"name": "Mock Step",
		"type": "tool",
		"config": map[string]string{
			"adapter_id": "mock",
		},
	}
	resp, body = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/steps", workflowID), stepReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var stepResp struct {
		Data Step `json:"data"`
	}
	err = json.Unmarshal(body, &stepResp)
	require.NoError(t, err)
	mockStepID := stepResp.Data.ID

	// Connect Start to Mock step
	edgeReq := map[string]string{
		"source_step_id": startStep.ID,
		"target_step_id": mockStepID,
	}
	resp, _ = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/edges", workflowID), edgeReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Publish workflow
	resp, _ = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/publish", workflowID), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	return workflowID
}

// Helper to wait for run completion
func waitForRunStatus(t *testing.T, runID string, expectedStatuses []string, timeout time.Duration) string {
	deadline := time.Now().Add(timeout)
	pollInterval := 500 * time.Millisecond

	var finalStatus string
	for time.Now().Before(deadline) {
		resp, body := makeRequest(t, "GET", fmt.Sprintf("/api/v1/runs/%s", runID), nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var getRunResp struct {
			Data Run `json:"data"`
		}
		err := json.Unmarshal(body, &getRunResp)
		require.NoError(t, err)

		finalStatus = getRunResp.Data.Status
		for _, expected := range expectedStatuses {
			if finalStatus == expected {
				return finalStatus
			}
		}
		time.Sleep(pollInterval)
	}

	return finalStatus
}

func TestRunListPagination(t *testing.T) {
	workflowID := createTestWorkflowForRuns(t)
	defer makeRequest(t, "DELETE", "/api/v1/workflows/"+workflowID, nil)

	// Create multiple runs
	for i := 0; i < 5; i++ {
		runReq := map[string]interface{}{
			"input": map[string]int{"index": i},
			"mode":  "test",
		}
		resp, body := makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/runs", workflowID), runReq)
		require.Equal(t, http.StatusCreated, resp.StatusCode, "Run %d create response: %s", i, string(body))
	}

	// List all runs
	resp, body := makeRequest(t, "GET", fmt.Sprintf("/api/v1/workflows/%s/runs", workflowID), nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var listResp struct {
		Data  []Run `json:"data"`
		Total int   `json:"total"`
		Page  int   `json:"page"`
		Limit int   `json:"limit"`
	}
	err := json.Unmarshal(body, &listResp)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(listResp.Data), 5)

	// Test pagination
	resp, body = makeRequest(t, "GET", fmt.Sprintf("/api/v1/workflows/%s/runs?page=1&limit=2", workflowID), nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	err = json.Unmarshal(body, &listResp)
	require.NoError(t, err)
	assert.Equal(t, 2, len(listResp.Data))
	assert.Equal(t, 1, listResp.Page)
	assert.Equal(t, 2, listResp.Limit)
}

func TestRunGetByID(t *testing.T) {
	workflowID := createTestWorkflowForRuns(t)
	defer makeRequest(t, "DELETE", "/api/v1/workflows/"+workflowID, nil)

	// Create a run
	runReq := map[string]interface{}{
		"input": map[string]string{"test": "value"},
		"mode":  "test",
	}
	resp, body := makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/runs", workflowID), runReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp struct {
		Data Run `json:"data"`
	}
	err := json.Unmarshal(body, &createResp)
	require.NoError(t, err)
	runID := createResp.Data.ID

	// Get run by ID
	resp, body = makeRequest(t, "GET", fmt.Sprintf("/api/v1/runs/%s", runID), nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var getResp struct {
		Data Run `json:"data"`
	}
	err = json.Unmarshal(body, &getResp)
	require.NoError(t, err)
	assert.Equal(t, runID, getResp.Data.ID)
	assert.Equal(t, workflowID, getResp.Data.WorkflowID)
}

func TestRunGetNotFound(t *testing.T) {
	// Try to get non-existent run
	resp, _ := makeRequest(t, "GET", "/api/v1/runs/00000000-0000-0000-0000-000000000000", nil)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestRunCancel(t *testing.T) {
	workflowID := createTestWorkflowForRuns(t)
	defer makeRequest(t, "DELETE", "/api/v1/workflows/"+workflowID, nil)

	// Create a run
	runReq := map[string]interface{}{
		"input": map[string]string{"test": "cancel"},
		"mode":  "test",
	}
	resp, body := makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/runs", workflowID), runReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp struct {
		Data Run `json:"data"`
	}
	err := json.Unmarshal(body, &createResp)
	require.NoError(t, err)
	runID := createResp.Data.ID

	// Try to cancel (might already be completed due to fast mock adapter)
	resp, body = makeRequest(t, "POST", fmt.Sprintf("/api/v1/runs/%s/cancel", runID), nil)
	// Accept both success and "not cancellable" errors
	assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadRequest,
		"Expected OK or BadRequest, got %d: %s", resp.StatusCode, string(body))
}

func TestRunWithModes(t *testing.T) {
	workflowID := createTestWorkflowForRuns(t)
	defer makeRequest(t, "DELETE", "/api/v1/workflows/"+workflowID, nil)

	modes := []string{"test", "production"}

	for _, mode := range modes {
		t.Run(fmt.Sprintf("mode_%s", mode), func(t *testing.T) {
			runReq := map[string]interface{}{
				"input": map[string]string{"mode_test": mode},
				"mode":  mode,
			}
			resp, body := makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/runs", workflowID), runReq)
			require.Equal(t, http.StatusCreated, resp.StatusCode, "Create response: %s", string(body))

			var createResp struct {
				Data Run `json:"data"`
			}
			err := json.Unmarshal(body, &createResp)
			require.NoError(t, err)
			assert.Equal(t, mode, createResp.Data.Mode)
		})
	}
}

func TestRunWithVersion(t *testing.T) {
	workflowID := createTestWorkflowForRuns(t)
	defer makeRequest(t, "DELETE", "/api/v1/workflows/"+workflowID, nil)

	// Execute with specific version (version 1 after publish)
	runReq := map[string]interface{}{
		"input":   map[string]string{"version_test": "v1"},
		"mode":    "test",
		"version": 1,
	}
	resp, body := makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/runs", workflowID), runReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Create response: %s", string(body))

	var createResp struct {
		Data struct {
			ID              string `json:"id"`
			WorkflowVersion int    `json:"workflow_version"`
		} `json:"data"`
	}
	err := json.Unmarshal(body, &createResp)
	require.NoError(t, err)
	assert.Equal(t, 1, createResp.Data.WorkflowVersion)
}

func TestRunWithInvalidVersion(t *testing.T) {
	workflowID := createTestWorkflowForRuns(t)
	defer makeRequest(t, "DELETE", "/api/v1/workflows/"+workflowID, nil)

	// Execute with non-existent version
	runReq := map[string]interface{}{
		"input":   map[string]string{"version_test": "v999"},
		"mode":    "test",
		"version": 999,
	}
	resp, _ := makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/runs", workflowID), runReq)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestStepOperations(t *testing.T) {
	// Create workflow
	createReq := map[string]string{
		"name":        "Step Operations Test " + time.Now().Format("20060102150405"),
		"description": "Testing step operations",
	}
	resp, body := makeRequest(t, "POST", "/api/v1/workflows", createReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp struct {
		Data Workflow `json:"data"`
	}
	err := json.Unmarshal(body, &createResp)
	require.NoError(t, err)
	workflowID := createResp.Data.ID
	defer makeRequest(t, "DELETE", "/api/v1/workflows/"+workflowID, nil)

	// List steps (should have auto-created Start step)
	resp, body = makeRequest(t, "GET", fmt.Sprintf("/api/v1/workflows/%s/steps", workflowID), nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var listResp struct {
		Data []Step `json:"data"`
	}
	err = json.Unmarshal(body, &listResp)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(listResp.Data), 1, "Should have at least Start step")

	// Create a step
	stepReq := map[string]interface{}{
		"name": "Test LLM Step",
		"type": "llm",
		"config": map[string]string{
			"provider": "openai",
			"model":    "gpt-4",
		},
	}
	resp, body = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/steps", workflowID), stepReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var stepResp struct {
		Data Step `json:"data"`
	}
	err = json.Unmarshal(body, &stepResp)
	require.NoError(t, err)
	stepID := stepResp.Data.ID
	assert.Equal(t, "Test LLM Step", stepResp.Data.Name)
	assert.Equal(t, "llm", stepResp.Data.Type)

	// Update step
	updateReq := map[string]interface{}{
		"name": "Updated LLM Step",
	}
	resp, body = makeRequest(t, "PUT", fmt.Sprintf("/api/v1/workflows/%s/steps/%s", workflowID, stepID), updateReq)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	err = json.Unmarshal(body, &stepResp)
	require.NoError(t, err)
	assert.Equal(t, "Updated LLM Step", stepResp.Data.Name)

	// Delete step
	resp, _ = makeRequest(t, "DELETE", fmt.Sprintf("/api/v1/workflows/%s/steps/%s", workflowID, stepID), nil)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestEdgeOperations(t *testing.T) {
	// Create workflow
	createReq := map[string]string{
		"name":        "Edge Operations Test " + time.Now().Format("20060102150405"),
		"description": "Testing edge operations",
	}
	resp, body := makeRequest(t, "POST", "/api/v1/workflows", createReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp struct {
		Data Workflow `json:"data"`
	}
	err := json.Unmarshal(body, &createResp)
	require.NoError(t, err)
	workflowID := createResp.Data.ID
	defer makeRequest(t, "DELETE", "/api/v1/workflows/"+workflowID, nil)

	// Get Start step
	_, steps := getWorkflowWithSteps(t, workflowID)
	startStep := findStartStep(steps)
	require.NotNil(t, startStep)

	// Create target step
	stepReq := map[string]interface{}{
		"name": "Target Step",
		"type": "tool",
		"config": map[string]string{
			"adapter_id": "mock",
		},
	}
	resp, body = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/steps", workflowID), stepReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var stepResp struct {
		Data Step `json:"data"`
	}
	err = json.Unmarshal(body, &stepResp)
	require.NoError(t, err)
	targetStepID := stepResp.Data.ID

	// List edges (should be empty initially)
	resp, body = makeRequest(t, "GET", fmt.Sprintf("/api/v1/workflows/%s/edges", workflowID), nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Create edge
	edgeReq := map[string]string{
		"source_step_id": startStep.ID,
		"target_step_id": targetStepID,
	}
	resp, body = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/edges", workflowID), edgeReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var edgeResp struct {
		Data struct {
			ID           string `json:"id"`
			SourceStepID string `json:"source_step_id"`
			TargetStepID string `json:"target_step_id"`
		} `json:"data"`
	}
	err = json.Unmarshal(body, &edgeResp)
	require.NoError(t, err)
	edgeID := edgeResp.Data.ID
	assert.Equal(t, startStep.ID, edgeResp.Data.SourceStepID)
	assert.Equal(t, targetStepID, edgeResp.Data.TargetStepID)

	// List edges (should have one now)
	resp, body = makeRequest(t, "GET", fmt.Sprintf("/api/v1/workflows/%s/edges", workflowID), nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var listResp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	err = json.Unmarshal(body, &listResp)
	require.NoError(t, err)
	assert.Len(t, listResp.Data, 1)

	// Delete edge
	resp, _ = makeRequest(t, "DELETE", fmt.Sprintf("/api/v1/workflows/%s/edges/%s", workflowID, edgeID), nil)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestVersionOperations(t *testing.T) {
	workflowID := createTestWorkflowForRuns(t)
	defer makeRequest(t, "DELETE", "/api/v1/workflows/"+workflowID, nil)

	// List versions
	resp, body := makeRequest(t, "GET", fmt.Sprintf("/api/v1/workflows/%s/versions", workflowID), nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var listResp struct {
		Data []struct {
			Version   int    `json:"version"`
			CreatedAt string `json:"created_at"`
		} `json:"data"`
	}
	err := json.Unmarshal(body, &listResp)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(listResp.Data), 1, "Should have at least one version")

	// Get specific version
	resp, body = makeRequest(t, "GET", fmt.Sprintf("/api/v1/workflows/%s/versions/1", workflowID), nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var versionResp struct {
		Data struct {
			Version    int             `json:"version"`
			Definition json.RawMessage `json:"definition"`
		} `json:"data"`
	}
	err = json.Unmarshal(body, &versionResp)
	require.NoError(t, err)
	assert.Equal(t, 1, versionResp.Data.Version)
	assert.NotEmpty(t, versionResp.Data.Definition)
}

func TestRunCompletionWithStepRuns(t *testing.T) {
	workflowID := createTestWorkflowForRuns(t)
	defer makeRequest(t, "DELETE", "/api/v1/workflows/"+workflowID, nil)

	// Create and execute a run
	runReq := map[string]interface{}{
		"input": map[string]string{"test": "step_runs"},
		"mode":  "test",
	}
	resp, body := makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/runs", workflowID), runReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp struct {
		Data Run `json:"data"`
	}
	err := json.Unmarshal(body, &createResp)
	require.NoError(t, err)
	runID := createResp.Data.ID

	// Wait for completion
	finalStatus := waitForRunStatus(t, runID, []string{"completed", "failed"}, 30*time.Second)
	assert.Equal(t, "completed", finalStatus)

	// Get run with step runs
	resp, body = makeRequest(t, "GET", fmt.Sprintf("/api/v1/runs/%s", runID), nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var runWithSteps struct {
		Data struct {
			ID       string `json:"id"`
			Status   string `json:"status"`
			StepRuns []struct {
				ID       string `json:"id"`
				StepID   string `json:"step_id"`
				StepName string `json:"step_name"`
				Status   string `json:"status"`
			} `json:"step_runs"`
		} `json:"data"`
	}
	err = json.Unmarshal(body, &runWithSteps)
	require.NoError(t, err)
	assert.Equal(t, "completed", runWithSteps.Data.Status)
	// Step runs should be populated
	assert.GreaterOrEqual(t, len(runWithSteps.Data.StepRuns), 1, "Should have step runs after completion")
}
