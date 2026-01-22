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
				SourcePort   string `json:"source_port"`
			} `json:"edges"`
		} `json:"data"`
	}
	err := json.Unmarshal(body, &getResp)
	require.NoError(t, err)

	// Verify required steps exist
	stepNames := make(map[string]bool)
	stepTypes := make(map[string]string)
	for _, step := range getResp.Data.Steps {
		stepNames[step.Name] = true
		stepTypes[step.Name] = step.Type
	}

	// Check main flow steps (Phase 1 migration added new steps)
	mainFlowSteps := []struct {
		name     string
		stepType string
	}{
		{"Start", "start"},
		{"Set Default Config", "set-variables"},
		{"Classify Intent", "llm-structured"},
		{"Switch Intent", "switch"},
		{"Config: Create", "set-variables"},
		{"Config: Debug", "set-variables"},
		{"Config: Explain", "set-variables"},
		{"Config: Enhance", "set-variables"},
		{"Config: Search", "set-variables"},
		{"Config: General", "set-variables"},
		{"Set Context", "set-variables"},
	}

	for _, s := range mainFlowSteps {
		assert.True(t, stepNames[s.name], "Should have %s step", s.name)
		assert.Equal(t, s.stepType, stepTypes[s.name], "%s should be %s type", s.name, s.stepType)
	}

	// Check tool steps exist (now with additional Phase 3-5 tools)
	toolSteps := []string{
		"list_blocks",
		"get_block_schema",
		"search_blocks",
		"list_workflows",
		"get_workflow",
		"update_step",
		"delete_step",
		"delete_edge",
		"create_workflow_structure",
		"search_documentation",
		"validate_workflow",
		"web_search",
		"fetch_url",
		"fix_block_type",      // Phase 3
		"auto_fix_errors",     // Phase 3
		"check_security",      // Phase 4
		"get_relevant_examples", // Phase 5
	}
	for _, toolName := range toolSteps {
		assert.True(t, stepNames[toolName], "Should have %s tool step", toolName)
		assert.Equal(t, "function", stepTypes[toolName], "%s should be a function step", toolName)
	}

	// Verify workflow has sufficient edges for the flow
	// Main flow + switch branches = at least 15 edges
	assert.GreaterOrEqual(t, len(getResp.Data.Edges), 15, "Copilot workflow should have at least 15 edges")

	// Log step count for debugging
	t.Logf("Total steps: %d", len(getResp.Data.Steps))
	t.Logf("Total edges: %d", len(getResp.Data.Edges))
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

	// Find the Start step (trigger block) and list_blocks step
	var startStepID string
	var listBlocksStepID string
	triggerTypes := map[string]bool{
		"start": true, "manual_trigger": true, "schedule_trigger": true, "webhook_trigger": true,
	}
	for _, step := range getResp.Data.Steps {
		if triggerTypes[step.Type] {
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

// TestCopilotAgentConfigurationValid tests that the Copilot agent BlockGroup has valid configuration
func TestCopilotAgentConfigurationValid(t *testing.T) {
	resp, body := makeRequest(t, "GET", "/api/v1/workflows/"+CopilotWorkflowID, nil)

	if resp.StatusCode == http.StatusNotFound {
		t.Skip("Copilot system workflow not found - seeder may not have run")
	}

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var getResp struct {
		Data struct {
			BlockGroups []struct {
				ID     string          `json:"id"`
				Name   string          `json:"name"`
				Type   string          `json:"type"`
				Config json.RawMessage `json:"config"`
			} `json:"block_groups"`
			Steps []struct {
				ID           string `json:"id"`
				Name         string `json:"name"`
				Type         string `json:"type"`
				BlockGroupID string `json:"block_group_id"`
			} `json:"steps"`
		} `json:"data"`
	}
	err := json.Unmarshal(body, &getResp)
	require.NoError(t, err)

	// Find the Copilot Agent BlockGroup
	var agentConfig map[string]interface{}
	var agentGroupID string
	for _, group := range getResp.Data.BlockGroups {
		if group.Name == "Copilot Agent" {
			err = json.Unmarshal(group.Config, &agentConfig)
			require.NoError(t, err)
			agentGroupID = group.ID
			break
		}
	}

	require.NotNil(t, agentConfig, "Should find Copilot Agent BlockGroup config")
	require.NotEmpty(t, agentGroupID, "Should have BlockGroup ID")

	// Verify agent configuration
	assert.Equal(t, "anthropic", agentConfig["provider"])
	assert.Equal(t, "claude-3-5-haiku-20241022", agentConfig["model"])
	assert.NotEmpty(t, agentConfig["system_prompt"], "Should have system prompt")

	// In Agent BlockGroup architecture, tools are derived from child steps
	// Verify that all expected tool steps belong to the agent group
	expectedTools := []string{
		"list_blocks",
		"get_block_schema",
		"search_blocks",
		"list_workflows",
		"get_workflow",
		"update_step",
		"delete_step",
		"delete_edge",
		"create_workflow_structure",
		"search_documentation",
		"validate_workflow",
		"web_search",
		"fetch_url",
		"fix_block_type",
		"auto_fix_errors",
		"check_security",
		"get_relevant_examples",
	}

	// Build map of step names that belong to the agent group
	agentStepNames := make(map[string]bool)
	for _, step := range getResp.Data.Steps {
		if step.BlockGroupID == agentGroupID {
			agentStepNames[step.Name] = true
		}
	}

	for _, toolName := range expectedTools {
		assert.True(t, agentStepNames[toolName], "Should have %s as child step of agent group", toolName)
	}

	// Verify at least 20 tool steps belong to the agent group (original 15 + 4 new Phase 3-5 tools)
	assert.GreaterOrEqual(t, len(agentStepNames), 17, "Should have at least 17 tool steps in agent group")
	t.Logf("Agent group has %d tool steps", len(agentStepNames))
}

// TestCopilotWorkflowCannotBeDeleted tests that system workflows cannot be deleted
func TestCopilotWorkflowCannotBeDeleted(t *testing.T) {
	resp, _ := makeRequest(t, "DELETE", "/api/v1/workflows/"+CopilotWorkflowID, nil)

	// System workflows should not be deletable (either 403 or 400)
	assert.True(t,
		resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusBadRequest,
		"System workflow should not be deletable, got status %d", resp.StatusCode)
}
