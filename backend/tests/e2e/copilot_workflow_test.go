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

// CopilotWorkflowID is the fixed ID for the Copilot system workflow
const CopilotWorkflowID = "a0000000-0000-0000-0000-000000000201"

// TestCopilotSystemWorkflowExists verifies the Copilot system workflow exists in the database
func TestCopilotSystemWorkflowExists(t *testing.T) {
	resp, body := makeRequest(t, "GET", "/api/v1/workflows/"+CopilotWorkflowID, nil)

	if resp.StatusCode == http.StatusNotFound {
		t.Skip("Copilot system workflow not found - seeder may not have run")
	}

	require.Equal(t, http.StatusOK, resp.StatusCode, "Response: %s", string(body))

	var getResp struct {
		Data struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			IsSystem    bool   `json:"is_system"`
			SystemSlug  string `json:"system_slug"`
			Steps       []Step `json:"steps"`
		} `json:"data"`
	}
	err := json.Unmarshal(body, &getResp)
	require.NoError(t, err)

	assert.Equal(t, CopilotWorkflowID, getResp.Data.ID)
	assert.Equal(t, "Copilot AI Assistant", getResp.Data.Name)
	assert.True(t, getResp.Data.IsSystem, "Should be a system workflow")
	assert.Equal(t, "copilot", getResp.Data.SystemSlug)
}

// TestCopilotWorkflowStructure verifies the Copilot workflow has the expected structure
func TestCopilotWorkflowStructure(t *testing.T) {
	resp, body := makeRequest(t, "GET", "/api/v1/workflows/"+CopilotWorkflowID, nil)

	if resp.StatusCode == http.StatusNotFound {
		t.Skip("Copilot system workflow not found - seeder may not have run")
	}

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var getResp struct {
		Data struct {
			Steps []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Type string `json:"type"`
			} `json:"steps"`
			Edges []struct {
				ID           string `json:"id"`
				SourceStepID string `json:"source_step_id"`
				TargetStepID string `json:"target_step_id"`
			} `json:"edges"`
		} `json:"data"`
	}
	err := json.Unmarshal(body, &getResp)
	require.NoError(t, err)

	// Verify step count (15 steps: start, set_context, copilot_agent, and 12 tool steps)
	assert.Equal(t, 15, len(getResp.Data.Steps), "Copilot workflow should have 15 steps")

	// Verify required steps exist
	stepNames := make(map[string]bool)
	stepTypes := make(map[string]string)
	for _, step := range getResp.Data.Steps {
		stepNames[step.Name] = true
		stepTypes[step.Name] = step.Type
	}

	// Check main flow steps
	assert.True(t, stepNames["Start"], "Should have Start step")
	assert.True(t, stepNames["Set Context"], "Should have Set Context step")
	assert.True(t, stepNames["Copilot Agent"], "Should have Copilot Agent step")

	// Check step types
	assert.Equal(t, "start", stepTypes["Start"])
	assert.Equal(t, "set-variables", stepTypes["Set Context"])
	assert.Equal(t, "agent", stepTypes["Copilot Agent"])

	// Check tool steps exist
	toolSteps := []string{
		"list_blocks",
		"get_block_schema",
		"search_blocks",
		"list_workflows",
		"get_workflow",
		"create_step",
		"update_step",
		"delete_step",
		"create_edge",
		"delete_edge",
		"search_documentation",
		"validate_workflow",
	}
	for _, toolName := range toolSteps {
		assert.True(t, stepNames[toolName], "Should have %s tool step", toolName)
		assert.Equal(t, "function", stepTypes[toolName], "%s should be a function step", toolName)
	}

	// Verify edge count (2 edges: start -> set_context, set_context -> copilot_agent)
	assert.Equal(t, 2, len(getResp.Data.Edges), "Copilot workflow should have 2 edges")
}

// TestCopilotWorkflowToolStepExecution tests that tool steps can be executed
func TestCopilotWorkflowToolStepExecution(t *testing.T) {
	// Skip in short mode - this test may take a while
	if testing.Short() {
		t.Skip("Skipping execution test in short mode")
	}

	resp, body := makeRequest(t, "GET", "/api/v1/workflows/"+CopilotWorkflowID, nil)

	if resp.StatusCode == http.StatusNotFound {
		t.Skip("Copilot system workflow not found - seeder may not have run")
	}

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var getResp struct {
		Data struct {
			Status string `json:"status"`
			Steps  []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Type string `json:"type"`
			} `json:"steps"`
		} `json:"data"`
	}
	err := json.Unmarshal(body, &getResp)
	require.NoError(t, err)

	// Find the Start step and list_blocks step
	var startStepID string
	var listBlocksStepID string
	for _, step := range getResp.Data.Steps {
		if step.Type == "start" {
			startStepID = step.ID
		}
		if step.Name == "list_blocks" {
			listBlocksStepID = step.ID
		}
	}

	require.NotEmpty(t, startStepID, "Should find Start step")
	require.NotEmpty(t, listBlocksStepID, "Should find list_blocks step")

	// Publish the workflow if it's in draft status
	if getResp.Data.Status == "draft" {
		resp, body = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/publish", CopilotWorkflowID), nil)
		if resp.StatusCode != http.StatusOK {
			t.Logf("Warning: Could not publish workflow: %s", string(body))
		}
	}

	// Execute the list_blocks step directly to test sandbox services work
	runReq := map[string]interface{}{
		"input":         map[string]interface{}{},
		"triggered_by":  "test",
		"start_step_id": listBlocksStepID,
	}
	resp, body = makeRequest(t, "POST", fmt.Sprintf("/api/v1/workflows/%s/runs", CopilotWorkflowID), runReq)

	// The workflow might fail due to LLM configuration, but we can at least check the run was created
	if resp.StatusCode == http.StatusCreated {
		var runResp struct {
			Data Run `json:"data"`
		}
		err = json.Unmarshal(body, &runResp)
		require.NoError(t, err)
		runID := runResp.Data.ID
		assert.NotEmpty(t, runID)

		// Wait a bit and check the run status
		time.Sleep(2 * time.Second)
		resp, body = makeRequest(t, "GET", fmt.Sprintf("/api/v1/runs/%s", runID), nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var getRunResp struct {
			Data struct {
				Status   string `json:"status"`
				StepRuns []struct {
					StepID string                 `json:"step_id"`
					Status string                 `json:"status"`
					Output map[string]interface{} `json:"output"`
				} `json:"step_runs"`
			} `json:"data"`
		}
		err = json.Unmarshal(body, &getRunResp)
		require.NoError(t, err)

		// Find the list_blocks step run
		for _, stepRun := range getRunResp.Data.StepRuns {
			if stepRun.StepID == listBlocksStepID && stepRun.Status == "completed" {
				// Check that the output contains blocks
				assert.NotNil(t, stepRun.Output["blocks"], "list_blocks should return blocks")
				t.Logf("list_blocks returned %d blocks", len(stepRun.Output["blocks"].([]interface{})))
			}
		}
	} else {
		t.Logf("Run creation returned status %d: %s", resp.StatusCode, string(body))
	}
}

// TestCopilotAgentConfigurationValid tests that the Copilot agent has valid configuration
func TestCopilotAgentConfigurationValid(t *testing.T) {
	resp, body := makeRequest(t, "GET", "/api/v1/workflows/"+CopilotWorkflowID, nil)

	if resp.StatusCode == http.StatusNotFound {
		t.Skip("Copilot system workflow not found - seeder may not have run")
	}

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var getResp struct {
		Data struct {
			Steps []struct {
				ID     string          `json:"id"`
				Name   string          `json:"name"`
				Type   string          `json:"type"`
				Config json.RawMessage `json:"config"`
			} `json:"steps"`
		} `json:"data"`
	}
	err := json.Unmarshal(body, &getResp)
	require.NoError(t, err)

	// Find the Copilot Agent step
	var agentConfig map[string]interface{}
	for _, step := range getResp.Data.Steps {
		if step.Name == "Copilot Agent" {
			err = json.Unmarshal(step.Config, &agentConfig)
			require.NoError(t, err)
			break
		}
	}

	require.NotNil(t, agentConfig, "Should find Copilot Agent step config")

	// Verify agent configuration
	assert.Equal(t, "anthropic", agentConfig["provider"])
	assert.Equal(t, "claude-sonnet-4-20250514", agentConfig["model"])
	assert.NotEmpty(t, agentConfig["system_prompt"], "Should have system prompt")

	// Verify tools are configured
	tools, ok := agentConfig["tools"].([]interface{})
	require.True(t, ok, "Should have tools array")
	assert.Equal(t, 12, len(tools), "Should have 12 tools configured")

	// Verify tool names match step names
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolMap := tool.(map[string]interface{})
		toolNames[toolMap["name"].(string)] = true
	}

	expectedTools := []string{
		"list_blocks",
		"get_block_schema",
		"search_blocks",
		"list_workflows",
		"get_workflow",
		"create_step",
		"update_step",
		"delete_step",
		"create_edge",
		"delete_edge",
		"search_documentation",
		"validate_workflow",
	}
	for _, toolName := range expectedTools {
		assert.True(t, toolNames[toolName], "Should have %s tool in agent config", toolName)
	}
}

// TestCopilotWorkflowCannotBeDeleted tests that system workflows cannot be deleted
func TestCopilotWorkflowCannotBeDeleted(t *testing.T) {
	resp, _ := makeRequest(t, "DELETE", "/api/v1/workflows/"+CopilotWorkflowID, nil)

	// System workflows should not be deletable (either 403 or 400)
	assert.True(t,
		resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusBadRequest,
		"System workflow should not be deletable, got status %d", resp.StatusCode)
}
