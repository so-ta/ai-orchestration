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

// BlockGroup represents a block group in the workflow
type BlockGroup struct {
	ID          string          `json:"id"`
	WorkflowID  string          `json:"workflow_id"`
	Name        string          `json:"name"`
	Type        string          `json:"type"`
	Config      json.RawMessage `json:"config,omitempty"`
	PreProcess  json.RawMessage `json:"pre_process,omitempty"`
	PostProcess json.RawMessage `json:"post_process,omitempty"`
}

// createBlockGroupWorkflow creates a workflow with block groups for testing
func createBlockGroupWorkflow(t *testing.T, name string) (workflowID string, startStepID string) {
	createReq := map[string]string{
		"name":        name + " " + time.Now().Format("20060102150405"),
		"description": "Block group test workflow",
	}
	resp, body := makeRequest(t, "POST", "/api/v1/workflows", createReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Create response: %s", string(body))

	var createResp struct {
		Data Workflow `json:"data"`
	}
	err := json.Unmarshal(body, &createResp)
	require.NoError(t, err)
	workflowID = createResp.Data.ID

	// Get the auto-created Start step
	_, steps := getWorkflowWithSteps(t, workflowID)
	startStep := findStartStep(steps)
	require.NotNil(t, startStep, "Workflow should have auto-created Start step")

	return workflowID, startStep.ID
}

// TestE2E_BlockGroupWorkflow_SaveAndLoad tests saving and loading workflows with block groups
func TestE2E_BlockGroupWorkflow_SaveAndLoad(t *testing.T) {
	workflowID, startStepID := createBlockGroupWorkflow(t, "BlockGroup Save Load Test")
	defer makeRequest(t, "DELETE", "/api/v1/workflows/"+workflowID, nil)

	// Create a parallel block group
	groupReq := map[string]interface{}{
		"name":      "Parallel Group",
		"type":      "parallel",
		"position_x": 200,
		"position_y": 100,
		"width":     400,
		"height":    300,
	}
	resp, body := makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/block-groups", workflowID), groupReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Block group create response: %s", string(body))

	var groupResp struct {
		Data BlockGroup `json:"data"`
	}
	err := json.Unmarshal(body, &groupResp)
	require.NoError(t, err)
	groupID := groupResp.Data.ID
	assert.NotEmpty(t, groupID)
	assert.Equal(t, "Parallel Group", groupResp.Data.Name)
	assert.Equal(t, "parallel", groupResp.Data.Type)

	// Create steps inside the block group
	branchAReq := map[string]interface{}{
		"name":           "Branch A",
		"type":           "function",
		"block_group_id": groupID,
		"group_role":     "body",
		"config": map[string]interface{}{
			"code":     "return { processed: 'A', input: input };",
			"language": "javascript",
		},
	}
	resp, body = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/steps", workflowID), branchAReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Branch A create response: %s", string(body))

	var branchAResp struct {
		Data Step `json:"data"`
	}
	json.Unmarshal(body, &branchAResp)
	branchAID := branchAResp.Data.ID

	branchBReq := map[string]interface{}{
		"name":           "Branch B",
		"type":           "function",
		"block_group_id": groupID,
		"group_role":     "body",
		"config": map[string]interface{}{
			"code":     "return { processed: 'B', input: input };",
			"language": "javascript",
		},
	}
	resp, body = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/steps", workflowID), branchBReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Branch B create response: %s", string(body))

	var branchBResp struct {
		Data Step `json:"data"`
	}
	json.Unmarshal(body, &branchBResp)
	branchBID := branchBResp.Data.ID

	// Create a merge step outside the group
	mergeReq := map[string]interface{}{
		"name": "Merge",
		"type": "function",
		"config": map[string]interface{}{
			"code":     "return { merged: true, input: input };",
			"language": "javascript",
		},
	}
	resp, body = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/steps", workflowID), mergeReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Merge step create response: %s", string(body))

	var mergeResp struct {
		Data Step `json:"data"`
	}
	json.Unmarshal(body, &mergeResp)
	mergeID := mergeResp.Data.ID

	// Connect: start -> branchA, start -> branchB
	edge1Req := map[string]string{
		"source_step_id": startStepID,
		"target_step_id": branchAID,
	}
	resp, _ = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/edges", workflowID), edge1Req)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	edge2Req := map[string]string{
		"source_step_id": startStepID,
		"target_step_id": branchBID,
	}
	resp, _ = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/edges", workflowID), edge2Req)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Connect: branchA -> merge, branchB -> merge
	edge3Req := map[string]string{
		"source_step_id": branchAID,
		"target_step_id": mergeID,
	}
	resp, _ = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/edges", workflowID), edge3Req)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	edge4Req := map[string]string{
		"source_step_id": branchBID,
		"target_step_id": mergeID,
	}
	resp, _ = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/edges", workflowID), edge4Req)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Publish workflow
	resp, body = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/publish", workflowID), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Publish response: %s", string(body))

	// Verify version 1 contains block groups
	resp, body = makeRequest(t, "GET", fmt.Sprintf("/api/v1/workflows/%s/versions/1", workflowID), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var versionResp struct {
		Data struct {
			Version    int             `json:"version"`
			Definition json.RawMessage `json:"definition"`
		} `json:"data"`
	}
	err = json.Unmarshal(body, &versionResp)
	require.NoError(t, err)
	assert.Equal(t, 1, versionResp.Data.Version)

	// Parse definition and verify block_groups
	var definition struct {
		Name        string       `json:"name"`
		Steps       []Step       `json:"steps"`
		BlockGroups []BlockGroup `json:"block_groups"`
	}
	err = json.Unmarshal(versionResp.Data.Definition, &definition)
	require.NoError(t, err, "Definition: %s", string(versionResp.Data.Definition))
	require.Len(t, definition.BlockGroups, 1, "Definition should contain block_groups. Got: %s", string(versionResp.Data.Definition))
	assert.Equal(t, "Parallel Group", definition.BlockGroups[0].Name)
	assert.Equal(t, "parallel", definition.BlockGroups[0].Type)
}

// TestE2E_BlockGroupWorkflow_Execution tests executing a workflow with block groups
func TestE2E_BlockGroupWorkflow_Execution(t *testing.T) {
	workflowID, startStepID := createBlockGroupWorkflow(t, "BlockGroup Execution Test")
	defer makeRequest(t, "DELETE", "/api/v1/workflows/"+workflowID, nil)

	// Create a parallel block group
	groupReq := map[string]interface{}{
		"name":      "Test Parallel",
		"type":      "parallel",
		"position_x": 200,
		"position_y": 100,
		"width":     400,
		"height":    200,
	}
	resp, body := makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/block-groups", workflowID), groupReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Block group create response: %s", string(body))

	var groupResp struct {
		Data BlockGroup `json:"data"`
	}
	json.Unmarshal(body, &groupResp)
	groupID := groupResp.Data.ID

	// Create step inside the group (will be executed in parallel)
	funcReq := map[string]interface{}{
		"name":           "Process",
		"type":           "function",
		"block_group_id": groupID,
		"group_role":     "body",
		"config": map[string]interface{}{
			"code":     "var v = input.value || 10; return { result: v * 2 };",
			"language": "javascript",
		},
	}
	resp, body = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/steps", workflowID), funcReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Function step create response: %s", string(body))

	var funcResp struct {
		Data Step `json:"data"`
	}
	json.Unmarshal(body, &funcResp)
	funcID := funcResp.Data.ID

	// Create output step outside the group
	outputReq := map[string]interface{}{
		"name": "Output",
		"type": "function",
		"config": map[string]interface{}{
			"code":     "return { final: 'done', input: input };",
			"language": "javascript",
		},
	}
	resp, body = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/steps", workflowID), outputReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Output step create response: %s", string(body))

	var outputResp struct {
		Data Step `json:"data"`
	}
	json.Unmarshal(body, &outputResp)
	outputID := outputResp.Data.ID

	// Connect: start -> process -> output
	edge1Req := map[string]string{
		"source_step_id": startStepID,
		"target_step_id": funcID,
	}
	resp, _ = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/edges", workflowID), edge1Req)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	edge2Req := map[string]string{
		"source_step_id": funcID,
		"target_step_id": outputID,
	}
	resp, _ = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/edges", workflowID), edge2Req)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Publish workflow
	resp, body = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/publish", workflowID), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Publish response: %s", string(body))

	// Execute workflow
	runReq := map[string]interface{}{
		"input":        map[string]int{"value": 5},
		"triggered_by": "test",
	}
	resp, body = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/runs", workflowID), runReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Run create response: %s", string(body))

	var runResp struct {
		Data Run `json:"data"`
	}
	err := json.Unmarshal(body, &runResp)
	require.NoError(t, err)
	runID := runResp.Data.ID

	// Wait for completion
	finalStatus := waitForRunStatus(t, runID, []string{"completed", "failed"}, 30*time.Second)
	assert.Equal(t, "completed", finalStatus, "Workflow should complete successfully")

	// Verify step runs
	resp, body = makeRequest(t, "GET", fmt.Sprintf("/api/v1/runs/%s", runID), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

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
	assert.GreaterOrEqual(t, len(runWithSteps.Data.StepRuns), 2, "Should have step runs for Start, Process, and Output")
}

// TestE2E_BlockGroupWorkflow_MultipleGroups tests workflows with multiple block groups
func TestE2E_BlockGroupWorkflow_MultipleGroups(t *testing.T) {
	workflowID, startStepID := createBlockGroupWorkflow(t, "Multiple BlockGroups Test")
	defer makeRequest(t, "DELETE", "/api/v1/workflows/"+workflowID, nil)

	// Create parallel group
	parallelReq := map[string]interface{}{
		"name":      "Parallel Section",
		"type":      "parallel",
		"position_x": 200,
		"position_y": 100,
	}
	resp, body := makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/block-groups", workflowID), parallelReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var parallelResp struct {
		Data BlockGroup `json:"data"`
	}
	json.Unmarshal(body, &parallelResp)
	parallelID := parallelResp.Data.ID

	// Create foreach group
	foreachReq := map[string]interface{}{
		"name":      "ForEach Section",
		"type":      "foreach",
		"position_x": 700,
		"position_y": 100,
	}
	resp, body = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/block-groups", workflowID), foreachReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var foreachResp struct {
		Data BlockGroup `json:"data"`
	}
	json.Unmarshal(body, &foreachResp)
	foreachID := foreachResp.Data.ID

	// List block groups to verify
	resp, body = makeRequest(t, "GET", fmt.Sprintf("/api/v1/workflows/%s/block-groups", workflowID), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var listResp struct {
		Data []BlockGroup `json:"data"`
	}
	err := json.Unmarshal(body, &listResp)
	require.NoError(t, err)
	assert.Len(t, listResp.Data, 2)

	// Create step in parallel group
	step1Req := map[string]interface{}{
		"name":           "Parallel Step",
		"type":           "function",
		"block_group_id": parallelID,
		"group_role":     "body",
		"config": map[string]interface{}{
			"code":     "return { parallel: true };",
			"language": "javascript",
		},
	}
	resp, body = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/steps", workflowID), step1Req)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var step1Resp struct {
		Data Step `json:"data"`
	}
	json.Unmarshal(body, &step1Resp)
	step1ID := step1Resp.Data.ID

	// Create step in foreach group
	step2Req := map[string]interface{}{
		"name":           "ForEach Step",
		"type":           "function",
		"block_group_id": foreachID,
		"group_role":     "body",
		"config": map[string]interface{}{
			"code":     "return { foreach: true };",
			"language": "javascript",
		},
	}
	resp, body = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/steps", workflowID), step2Req)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var step2Resp struct {
		Data Step `json:"data"`
	}
	json.Unmarshal(body, &step2Resp)
	step2ID := step2Resp.Data.ID

	// Connect: start -> parallel step -> foreach step
	edge1Req := map[string]string{
		"source_step_id": startStepID,
		"target_step_id": step1ID,
	}
	resp, _ = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/edges", workflowID), edge1Req)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	edge2Req := map[string]string{
		"source_step_id": step1ID,
		"target_step_id": step2ID,
	}
	resp, _ = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/edges", workflowID), edge2Req)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Publish
	resp, body = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/publish", workflowID), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Publish response: %s", string(body))

	// Execute
	runReq := map[string]interface{}{
		"input":        map[string]string{"test": "multiple_groups"},
		"triggered_by": "test",
	}
	resp, body = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/runs", workflowID), runReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var runResp struct {
		Data Run `json:"data"`
	}
	json.Unmarshal(body, &runResp)
	runID := runResp.Data.ID

	// Wait for completion
	finalStatus := waitForRunStatus(t, runID, []string{"completed", "failed"}, 30*time.Second)
	assert.Equal(t, "completed", finalStatus, "Workflow with multiple groups should complete")

	// Verify version contains both groups
	resp, body = makeRequest(t, "GET", fmt.Sprintf("/api/v1/workflows/%s/versions/1", workflowID), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var versionResp struct {
		Data struct {
			Definition json.RawMessage `json:"definition"`
		} `json:"data"`
	}
	json.Unmarshal(body, &versionResp)

	var definition struct {
		BlockGroups []BlockGroup `json:"block_groups"`
	}
	json.Unmarshal(versionResp.Data.Definition, &definition)
	assert.Len(t, definition.BlockGroups, 2, "Version should have both block groups")
}
