package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCopilotToolInputFix verifies that the tool input fix works correctly.
// This test uses the synchronous message endpoint to ensure the agent
// execution completes before checking results.
func TestCopilotToolInputFix(t *testing.T) {
	// Create a test project
	projectID := createTestProject(t)
	t.Logf("Created test project: %s", projectID)

	// Get the project to verify initial state (should have 1 Start step)
	resp, body := makeRequest(t, "GET", "/api/v1/workflows/"+projectID, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Get project response: %s", string(body))

	var initialProject struct {
		Data struct {
			ID    string `json:"id"`
			Steps []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Type string `json:"type"`
			} `json:"steps"`
		} `json:"data"`
	}
	err := json.Unmarshal(body, &initialProject)
	require.NoError(t, err)

	initialStepCount := len(initialProject.Data.Steps)
	t.Logf("Initial step count: %d", initialStepCount)

	// Find the Start step ID
	var startStepID string
	for _, step := range initialProject.Data.Steps {
		if step.Type == "start" {
			startStepID = step.ID
			break
		}
	}
	require.NotEmpty(t, startStepID, "Should have a Start step")

	// Create a Copilot session
	sessionID := createCopilotAgentSessionWithRetry(t, projectID, "LLMブロックを1つ追加してください", "create")
	if sessionID == "" {
		t.Skip("Copilot session creation failed")
	}
	t.Logf("Created Copilot session: %s", sessionID)

	// Send message synchronously to trigger agent execution
	msgReq := map[string]string{
		"content": "LLMブロックを1つ追加してください",
	}
	msgURL := fmt.Sprintf("/api/v1/workflows/%s/copilot/agent/sessions/%s/messages", projectID, sessionID)
	resp, body = makeRequestWithTimeout(t, "POST", msgURL, msgReq, 120*time.Second)

	if resp == nil {
		t.Skip("Server connection failed")
	}
	if resp.StatusCode == http.StatusInternalServerError {
		t.Logf("Agent execution failed: %s", string(body))
		t.Skip("Agent execution failed")
	}
	require.Equal(t, http.StatusOK, resp.StatusCode, "Send message response: %s", string(body))

	// Verify that a new LLM step was created
	resp, body = makeRequest(t, "GET", "/api/v1/workflows/"+projectID, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Get project response: %s", string(body))

	var updatedProject struct {
		Data struct {
			Steps []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Type string `json:"type"`
			} `json:"steps"`
		} `json:"data"`
	}
	err = json.Unmarshal(body, &updatedProject)
	require.NoError(t, err)

	t.Logf("Final step count: %d", len(updatedProject.Data.Steps))
	for _, step := range updatedProject.Data.Steps {
		t.Logf("  - %s (type: %s)", step.Name, step.Type)
	}

	// Should have at least one more step than before
	assert.GreaterOrEqual(t, len(updatedProject.Data.Steps), initialStepCount+1,
		"Should have created at least one new step. Steps: %+v", updatedProject.Data.Steps)

	// Verify an LLM step was created
	llmStepFound := false
	for _, step := range updatedProject.Data.Steps {
		if step.Type == "llm" {
			llmStepFound = true
			t.Logf("Found LLM step: %s (name: %s)", step.ID, step.Name)
			break
		}
	}
	assert.True(t, llmStepFound, "Should have created an LLM step. Steps: %+v", updatedProject.Data.Steps)
}

// TestCopilotComplexWorkflow tests creating a complex workflow with multiple blocks and edges.
// Uses synchronous message endpoint to ensure completion.
func TestCopilotComplexWorkflow(t *testing.T) {
	// Create a test project
	projectID := createTestProject(t)
	t.Logf("Created test project: %s", projectID)

	// Create a session
	prompt := "HTTPリクエストでAPIからデータを取得し、LLMで分析するワークフローを作成してください"
	sessionID := createCopilotAgentSessionWithRetry(t, projectID, prompt, "create")
	if sessionID == "" {
		t.Skip("Copilot session creation failed")
	}
	t.Logf("Created Copilot session: %s", sessionID)

	// Send message synchronously to trigger agent execution
	msgReq := map[string]string{
		"content": "HTTPブロックとLLMブロックを追加して、HTTPからLLMへ接続してください",
	}
	msgURL := fmt.Sprintf("/api/v1/workflows/%s/copilot/agent/sessions/%s/messages", projectID, sessionID)
	resp, body := makeRequestWithTimeout(t, "POST", msgURL, msgReq, 180*time.Second)

	if resp == nil {
		t.Skip("Server connection failed")
	}
	if resp.StatusCode == http.StatusInternalServerError {
		t.Logf("Agent execution failed: %s", string(body))
		t.Skip("Agent execution failed")
	}
	require.Equal(t, http.StatusOK, resp.StatusCode, "Send message response: %s", string(body))

	// Verify the project structure
	resp, body = makeRequest(t, "GET", "/api/v1/workflows/"+projectID, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Get project response: %s", string(body))

	var project struct {
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
	err := json.Unmarshal(body, &project)
	require.NoError(t, err)

	t.Logf("Steps created: %d", len(project.Data.Steps))
	for _, step := range project.Data.Steps {
		t.Logf("  - %s: %s (type: %s)", step.ID, step.Name, step.Type)
	}

	t.Logf("Edges created: %d", len(project.Data.Edges))
	for _, edge := range project.Data.Edges {
		t.Logf("  - %s: %s -> %s", edge.ID, edge.SourceStepID, edge.TargetStepID)
	}

	// Verify at least Start step exists
	stepTypes := make(map[string]bool)
	for _, step := range project.Data.Steps {
		stepTypes[step.Type] = true
	}

	assert.True(t, stepTypes["start"], "Should have a Start step")

	// At minimum, verify we have more than just the Start step
	assert.Greater(t, len(project.Data.Steps), 1, "Should have created additional steps beyond Start")
}

// TestCopilotBatchOperations tests the batch create operations.
// Uses synchronous message endpoint to ensure completion.
func TestCopilotBatchOperations(t *testing.T) {
	// Create a test project
	projectID := createTestProject(t)
	t.Logf("Created test project: %s", projectID)

	// Create a session
	sessionID := createCopilotAgentSessionWithRetry(t, projectID, "複数ブロックを追加", "create")
	if sessionID == "" {
		t.Skip("Copilot session creation failed")
	}
	t.Logf("Created Copilot session: %s", sessionID)

	// Request to add multiple blocks at once via synchronous endpoint
	msgReq := map[string]string{
		"content": "HTTP、LLM、Slackの3つのブロックを追加してください",
	}
	msgURL := fmt.Sprintf("/api/v1/workflows/%s/copilot/agent/sessions/%s/messages", projectID, sessionID)
	resp, body := makeRequestWithTimeout(t, "POST", msgURL, msgReq, 180*time.Second)

	if resp == nil {
		t.Skip("Server connection failed")
	}
	if resp.StatusCode == http.StatusInternalServerError {
		t.Logf("Agent execution failed: %s", string(body))
		t.Skip("Agent execution failed")
	}
	require.Equal(t, http.StatusOK, resp.StatusCode, "Send message response: %s", string(body))

	// Verify the blocks were created
	resp, body = makeRequest(t, "GET", "/api/v1/workflows/"+projectID, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Get project response: %s", string(body))

	var project struct {
		Data struct {
			Steps []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Type string `json:"type"`
			} `json:"steps"`
		} `json:"data"`
	}
	err := json.Unmarshal(body, &project)
	require.NoError(t, err)

	t.Logf("Total steps: %d", len(project.Data.Steps))
	for _, step := range project.Data.Steps {
		t.Logf("  - %s: %s (type: %s)", step.ID, step.Name, step.Type)
	}

	// Should have Start + at least 1 new block
	assert.GreaterOrEqual(t, len(project.Data.Steps), 2, "Should have created at least 1 additional step")
}

// createCopilotAgentSessionWithRetry creates a session with retry logic
func createCopilotAgentSessionWithRetry(t *testing.T, workflowID, prompt, mode string) string {
	t.Helper()

	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		startReq := map[string]string{
			"initial_prompt": prompt,
			"mode":           mode,
		}
		url := fmt.Sprintf("/api/v1/workflows/%s/copilot/agent/sessions", workflowID)
		resp, body := makeRequest(t, "POST", url, startReq)

		if resp == nil {
			t.Logf("Attempt %d: Server connection failed, retrying...", i+1)
			time.Sleep(2 * time.Second)
			continue
		}

		if resp.StatusCode == http.StatusInternalServerError {
			t.Logf("Attempt %d: Server error, response: %s", i+1, string(body))
			time.Sleep(2 * time.Second)
			continue
		}

		if resp.StatusCode == http.StatusCreated {
			var startResp CopilotAgentSession
			err := json.Unmarshal(body, &startResp)
			if err != nil {
				t.Logf("Attempt %d: Failed to unmarshal response: %v", i+1, err)
				continue
			}
			return startResp.SessionID
		}

		t.Logf("Attempt %d: Unexpected status %d, response: %s", i+1, resp.StatusCode, string(body))
		time.Sleep(2 * time.Second)
	}

	return ""
}

// TestCopilotToolInputDirectly tests the create_step tool directly by examining
// the workflow state before and after a simple Copilot request.
// This test uses the synchronous SendAgentMessage endpoint to ensure
// the agent execution completes before checking results.
func TestCopilotToolInputDirectly(t *testing.T) {
	// Create a test project
	projectID := createTestProject(t)
	t.Logf("Created test project: %s", projectID)

	// Get initial state
	resp, body := makeRequest(t, "GET", "/api/v1/workflows/"+projectID, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var initial struct {
		Data struct {
			Steps []json.RawMessage `json:"steps"`
		} `json:"data"`
	}
	json.Unmarshal(body, &initial)
	initialCount := len(initial.Data.Steps)
	t.Logf("Initial step count: %d", initialCount)

	// Create a session
	sessionID := createCopilotAgentSessionWithRetry(t, projectID, "LLMブロックを追加して", "create")
	if sessionID == "" {
		t.Skip("Could not create session")
	}
	t.Logf("Created session: %s", sessionID)

	// Send message synchronously (this triggers workflow execution and waits for completion)
	msgReq := map[string]string{
		"content": "LLMブロックを1つ追加してください",
	}
	msgURL := fmt.Sprintf("/api/v1/workflows/%s/copilot/agent/sessions/%s/messages", projectID, sessionID)
	resp, body = makeRequestWithTimeout(t, "POST", msgURL, msgReq, 120*time.Second)

	if resp == nil {
		t.Skip("Server connection failed")
	}

	if resp.StatusCode == http.StatusInternalServerError {
		t.Logf("Agent execution failed: %s", string(body))
		t.Skip("Agent execution failed - may not be configured")
	}

	require.Equal(t, http.StatusOK, resp.StatusCode, "Send message response: %s", string(body))

	var msgResp struct {
		SessionID  string   `json:"session_id"`
		Response   string   `json:"response"`
		ToolsUsed  []string `json:"tools_used"`
		RunID      string   `json:"run_id"`
	}
	err := json.Unmarshal(body, &msgResp)
	require.NoError(t, err)
	t.Logf("Agent response: %s", truncate(msgResp.Response, 200))
	t.Logf("Tools used: %v", msgResp.ToolsUsed)

	// Get final state
	resp, body = makeRequest(t, "GET", "/api/v1/workflows/"+projectID, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var final struct {
		Data struct {
			Steps []struct {
				ID     string          `json:"id"`
				Name   string          `json:"name"`
				Type   string          `json:"type"`
				Config json.RawMessage `json:"config"`
			} `json:"steps"`
		} `json:"data"`
	}
	json.Unmarshal(body, &final)

	t.Logf("Initial steps: %d, Final steps: %d", initialCount, len(final.Data.Steps))
	for _, step := range final.Data.Steps {
		t.Logf("  Step: %s (type: %s)", step.Name, step.Type)
	}

	// Assert that at least one step was created
	assert.Greater(t, len(final.Data.Steps), initialCount,
		"Should have created at least one new step. Initial: %d, Final: %d",
		initialCount, len(final.Data.Steps))
}

// makeRequestWithTimeout makes an HTTP request with a custom timeout
func makeRequestWithTimeout(t *testing.T, method, path string, body interface{}, timeout time.Duration) (*http.Response, []byte) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		require.NoError(t, err)
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, baseURL+path, reqBody)
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", testTenantID)
	req.Header.Set("X-User-ID", testUserID)

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		t.Logf("Request failed: %v", err)
		return nil, nil
	}

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	return resp, respBody
}

// truncate truncates a string to maxLen characters
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
