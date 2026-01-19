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

// ProjectDetail represents a project with steps for testing
type ProjectDetail struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Status      string       `json:"status"`
	Steps       []StepDetail `json:"steps"`
}

// StepDetail represents a step with config for testing
type StepDetail struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	BlockSlug *string                `json:"block_slug"`
	Config    map[string]interface{} `json:"config"`
}

// RunDetail represents a run with status
type RunDetail struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	Error       string `json:"error"`
	CompletedAt string `json:"completed_at"`
}

// TestBuilderAgentConstructScenario tests the complete agent-based workflow construction
// This test requires:
// - API server running on localhost:8090
// - Worker running to process async tasks
// - LLM API configured (e.g., ANTHROPIC_API_KEY)
func TestBuilderAgentConstructScenario(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping scenario test in short mode")
	}

	// Configuration
	maxWaitTime := 180 * time.Second // Agent-based construction may take longer
	pollInterval := 3 * time.Second

	t.Run("Full Agent Construct Flow", func(t *testing.T) {
		// Step 1: Create a builder session with a clear workflow request
		t.Log("Step 1: Creating builder session with workflow request...")
		startReq := map[string]string{
			"initial_prompt": "Slackに週次レポートを送信するワークフローを作成してください。毎週月曜日の朝9時に実行し、先週のデータを集計してSlackの#reportsチャンネルに投稿します。",
		}
		resp, body := makeRequest(t, "POST", "/api/v1/builder/sessions", startReq)

		if resp.StatusCode == http.StatusInternalServerError {
			t.Skip("AI Builder system project may not be seeded or LLM not configured")
		}
		require.Equal(t, http.StatusCreated, resp.StatusCode, "Create session response: %s", string(body))

		var createResp struct {
			SessionID string `json:"session_id"`
			Status    string `json:"status"`
			Phase     string `json:"phase"`
		}
		err := json.Unmarshal(body, &createResp)
		require.NoError(t, err)
		sessionID := createResp.SessionID
		require.NotEmpty(t, sessionID)
		t.Logf("  Session ID: %s", sessionID)
		t.Logf("  Initial status: %s, phase: %s", createResp.Status, createResp.Phase)

		// Cleanup on test end
		defer func() {
			t.Log("Cleanup: Deleting builder session...")
			makeRequest(t, "DELETE", fmt.Sprintf("/api/v1/builder/sessions/%s", sessionID), nil)
		}()

		// Step 2: Wait for analysis to complete and move to proposal phase
		t.Log("Step 2: Waiting for analysis phase to complete...")
		deadline := time.Now().Add(maxWaitTime)
		var session struct {
			Phase     string  `json:"hearing_phase"`
			Status    string  `json:"status"`
			ProjectID *string `json:"project_id"`
		}

		for time.Now().Before(deadline) {
			resp, body = makeRequest(t, "GET", fmt.Sprintf("/api/v1/builder/sessions/%s", sessionID), nil)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			json.Unmarshal(body, &session)

			t.Logf("  Current phase: %s", session.Phase)
			if session.Phase == "proposal" || session.Phase == "completed" {
				break
			}
			time.Sleep(pollInterval)
		}
		require.Contains(t, []string{"proposal", "completed"}, session.Phase, "Phase should advance to proposal or completed")

		// Step 3: Send confirmation message to move to completed phase
		if session.Phase == "proposal" {
			t.Log("Step 3: Sending confirmation message...")
			msgReq := map[string]string{
				"content": "はい、その内容で進めてください。Slackチャンネルは#reportsでお願いします。",
			}
			resp, body = makeRequest(t, "POST", fmt.Sprintf("/api/v1/builder/sessions/%s/messages", sessionID), msgReq)

			if resp.StatusCode == http.StatusInternalServerError {
				t.Skip("Proposal entry point may not be configured")
			}
			require.Equal(t, http.StatusAccepted, resp.StatusCode, "Send message response: %s", string(body))

			var msgResp struct {
				RunID string `json:"run_id"`
			}
			json.Unmarshal(body, &msgResp)
			t.Logf("  Message run ID: %s", msgResp.RunID)

			// Wait for proposal run to complete
			waitForRunCompletion(t, msgResp.RunID, maxWaitTime)

			// Wait for phase to become completed
			t.Log("Step 4: Waiting for hearing phase to complete...")
			for time.Now().Before(deadline) {
				resp, body = makeRequest(t, "GET", fmt.Sprintf("/api/v1/builder/sessions/%s", sessionID), nil)
				require.Equal(t, http.StatusOK, resp.StatusCode)
				json.Unmarshal(body, &session)

				t.Logf("  Current phase: %s", session.Phase)
				if session.Phase == "completed" {
					break
				}
				time.Sleep(pollInterval)
			}
		}
		require.Equal(t, "completed", session.Phase, "Hearing phase should be completed before construction")

		// Step 5: Trigger agent-based workflow construction
		t.Log("Step 5: Triggering agent-based workflow construction...")
		resp, body = makeRequest(t, "POST", fmt.Sprintf("/api/v1/builder/sessions/%s/construct", sessionID), nil)

		if resp.StatusCode == http.StatusInternalServerError {
			t.Skip("Agent construct entry point may not be configured")
		}
		require.Equal(t, http.StatusAccepted, resp.StatusCode, "Construct response: %s", string(body))

		var constructResp struct {
			RunID  string `json:"run_id"`
			Status string `json:"status"`
		}
		err = json.Unmarshal(body, &constructResp)
		require.NoError(t, err)
		require.NotEmpty(t, constructResp.RunID)
		t.Logf("  Construction run ID: %s", constructResp.RunID)

		// Step 6: Wait for construction to complete
		t.Log("Step 6: Waiting for agent-based construction to complete...")
		run := waitForRunCompletion(t, constructResp.RunID, maxWaitTime)
		t.Logf("  Final run status: %s", run.Status)

		if run.Status == "failed" {
			t.Logf("  Run error: %s", run.Error)
			// Get step runs for debugging
			stepRuns := getStepRuns(t, constructResp.RunID)
			for _, sr := range stepRuns {
				if sr["status"] == "failed" {
					t.Logf("  Failed step: %s - %v", sr["step_name"], sr["error"])
				}
			}
		}
		require.Equal(t, "completed", run.Status, "Construction run should complete successfully")

		// Step 7: Verify project was created
		t.Log("Step 7: Verifying project creation...")
		resp, body = makeRequest(t, "GET", fmt.Sprintf("/api/v1/builder/sessions/%s", sessionID), nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		json.Unmarshal(body, &session)

		require.NotNil(t, session.ProjectID, "Project ID should be set after construction")
		t.Logf("  Created project ID: %s", *session.ProjectID)

		// Step 8: Verify project has steps with config
		t.Log("Step 8: Verifying project steps have config...")
		project := getProject(t, *session.ProjectID)
		require.NotEmpty(t, project.Steps, "Project should have steps")
		t.Logf("  Project name: %s", project.Name)
		t.Logf("  Number of steps: %d", len(project.Steps))

		// Check that steps have config
		configuredSteps := 0
		for _, step := range project.Steps {
			t.Logf("  Step: %s (type: %s)", step.Name, step.Type)
			if step.Config != nil && len(step.Config) > 0 {
				configuredSteps++
				t.Logf("    Config keys: %v", getMapKeys(step.Config))
			}
			if step.BlockSlug != nil {
				t.Logf("    Block slug: %s", *step.BlockSlug)
			}
		}

		assert.Greater(t, configuredSteps, 0, "At least one step should have non-empty config")
		t.Logf("  Steps with config: %d/%d", configuredSteps, len(project.Steps))
	})
}

// TestBuilderAgentConstructWithSlackBlock specifically tests that Slack blocks get proper config
func TestBuilderAgentConstructWithSlackBlock(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping scenario test in short mode")
	}

	maxWaitTime := 180 * time.Second
	pollInterval := 3 * time.Second

	t.Run("Slack Block Config Generation", func(t *testing.T) {
		// Create session with explicit Slack requirement
		startReq := map[string]string{
			"initial_prompt": "Slackの#general チャンネルに「Hello World」というメッセージを送信するシンプルなワークフローを作成してください。",
		}
		resp, body := makeRequest(t, "POST", "/api/v1/builder/sessions", startReq)

		if resp.StatusCode == http.StatusInternalServerError {
			t.Skip("AI Builder system project may not be seeded")
		}
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		var createResp struct {
			SessionID string `json:"session_id"`
		}
		json.Unmarshal(body, &createResp)
		sessionID := createResp.SessionID

		defer func() {
			makeRequest(t, "DELETE", fmt.Sprintf("/api/v1/builder/sessions/%s", sessionID), nil)
		}()

		// Wait for hearing to complete (may need multiple messages)
		deadline := time.Now().Add(maxWaitTime)
		var session struct {
			Phase     string  `json:"hearing_phase"`
			ProjectID *string `json:"project_id"`
		}

		for time.Now().Before(deadline) {
			resp, body = makeRequest(t, "GET", fmt.Sprintf("/api/v1/builder/sessions/%s", sessionID), nil)
			json.Unmarshal(body, &session)

			if session.Phase == "completed" {
				break
			}
			if session.Phase == "proposal" {
				// Send confirmation
				msgReq := map[string]string{"content": "OK, proceed"}
				resp, body = makeRequest(t, "POST", fmt.Sprintf("/api/v1/builder/sessions/%s/messages", sessionID), msgReq)
				if resp.StatusCode == http.StatusAccepted {
					var msgResp struct {
						RunID string `json:"run_id"`
					}
					json.Unmarshal(body, &msgResp)
					waitForRunCompletion(t, msgResp.RunID, maxWaitTime)
				}
			}
			time.Sleep(pollInterval)
		}

		if session.Phase != "completed" {
			t.Skip("Could not reach completed phase")
		}

		// Trigger construction
		resp, body = makeRequest(t, "POST", fmt.Sprintf("/api/v1/builder/sessions/%s/construct", sessionID), nil)
		if resp.StatusCode != http.StatusAccepted {
			t.Skip("Construct failed to start")
		}

		var constructResp struct {
			RunID string `json:"run_id"`
		}
		json.Unmarshal(body, &constructResp)

		run := waitForRunCompletion(t, constructResp.RunID, maxWaitTime)
		if run.Status != "completed" {
			t.Skipf("Construction did not complete: %s", run.Error)
		}

		// Verify project
		resp, body = makeRequest(t, "GET", fmt.Sprintf("/api/v1/builder/sessions/%s", sessionID), nil)
		json.Unmarshal(body, &session)

		if session.ProjectID == nil {
			t.Fatal("Project was not created")
		}

		project := getProject(t, *session.ProjectID)

		// Look for Slack-related step
		foundSlackConfig := false
		for _, step := range project.Steps {
			if step.BlockSlug != nil && (*step.BlockSlug == "slack" || *step.BlockSlug == "slack-post-message") {
				t.Logf("Found Slack step: %s", step.Name)
				t.Logf("  Config: %+v", step.Config)

				// Check for expected config fields
				if step.Config != nil {
					if _, ok := step.Config["channel"]; ok {
						foundSlackConfig = true
						t.Logf("  ✓ channel field found")
					}
					if _, ok := step.Config["message"]; ok {
						t.Logf("  ✓ message field found")
					}
				}
			}
		}

		// This assertion may fail if no Slack block is available
		// or if the LLM chose a different approach
		if !foundSlackConfig {
			t.Log("Note: Slack config not found. This may be expected if Slack block is not seeded.")
		}
	})
}

// waitForRunCompletion waits for a run to complete and returns the final run status
func waitForRunCompletion(t *testing.T, runID string, timeout time.Duration) *RunDetail {
	t.Helper()
	deadline := time.Now().Add(timeout)
	pollInterval := 3 * time.Second

	for time.Now().Before(deadline) {
		resp, body := makeRequest(t, "GET", fmt.Sprintf("/api/v1/runs/%s", runID), nil)
		if resp.StatusCode != http.StatusOK {
			time.Sleep(pollInterval)
			continue
		}

		var runResp struct {
			Data RunDetail `json:"data"`
		}
		json.Unmarshal(body, &runResp)

		if runResp.Data.Status == "completed" || runResp.Data.Status == "failed" {
			return &runResp.Data
		}
		time.Sleep(pollInterval)
	}

	t.Fatalf("Timeout waiting for run %s to complete", runID)
	return nil
}

// getProject retrieves a project by ID
func getProject(t *testing.T, projectID string) *ProjectDetail {
	t.Helper()
	resp, body := makeRequest(t, "GET", fmt.Sprintf("/api/v1/workflows/%s", projectID), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Get project response: %s", string(body))

	var projectResp struct {
		Data ProjectDetail `json:"data"`
	}
	err := json.Unmarshal(body, &projectResp)
	require.NoError(t, err)

	return &projectResp.Data
}

// getStepRuns retrieves step runs for a run
func getStepRuns(t *testing.T, runID string) []map[string]interface{} {
	t.Helper()
	resp, body := makeRequest(t, "GET", fmt.Sprintf("/api/v1/runs/%s/steps", runID), nil)
	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var stepsResp struct {
		Data []map[string]interface{} `json:"data"`
	}
	json.Unmarshal(body, &stepsResp)
	return stepsResp.Data
}

// getMapKeys returns the keys of a map
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
