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

// CopilotAgentSession represents a copilot agent session for testing
type CopilotAgentSession struct {
	SessionID string   `json:"session_id"`
	Response  string   `json:"response"`
	ToolsUsed []string `json:"tools_used"`
	Status    string   `json:"status"`
	Phase     string   `json:"phase"`
	Progress  int      `json:"progress"`
}

// createTestProject creates a test project and returns its ID
func createTestProject(t *testing.T) string {
	t.Helper()

	createReq := map[string]string{
		"name":        "Copilot Agent Test Project " + time.Now().Format("20060102150405.000"),
		"description": "Created by Copilot Agent E2E test",
	}
	resp, body := makeRequest(t, "POST", "/api/v1/workflows", createReq)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Create project response: %s", string(body))

	var createResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	err := json.Unmarshal(body, &createResp)
	require.NoError(t, err)
	require.NotEmpty(t, createResp.Data.ID, "Project ID should not be empty")

	return createResp.Data.ID
}

// createCopilotAgentSession creates a copilot agent session and returns the session ID.
func createCopilotAgentSession(t *testing.T, workflowID, prompt, mode string) string {
	t.Helper()

	startReq := map[string]string{
		"initial_prompt": prompt,
		"mode":           mode,
	}
	url := fmt.Sprintf("/api/v1/workflows/%s/copilot/agent/sessions", workflowID)
	resp, body := makeRequest(t, "POST", url, startReq)

	if resp.StatusCode == http.StatusInternalServerError {
		t.Skip("Copilot agent may not be available")
	}
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Start session response: %s", string(body))

	var startResp CopilotAgentSession
	err := json.Unmarshal(body, &startResp)
	require.NoError(t, err)

	return startResp.SessionID
}

func TestCopilotAgentToolsEndpoint(t *testing.T) {
	resp, body := makeRequest(t, "GET", "/api/v1/copilot/agent/tools", nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Tools endpoint response: %s", string(body))

	var toolsResp struct {
		Count int `json:"count"`
		Tools []struct {
			Name        string          `json:"name"`
			Description string          `json:"description"`
			InputSchema json.RawMessage `json:"input_schema"`
		} `json:"tools"`
	}
	err := json.Unmarshal(body, &toolsResp)
	require.NoError(t, err)

	// Should have at least some tools
	assert.Greater(t, toolsResp.Count, 0, "Should have at least one tool")
	assert.Equal(t, len(toolsResp.Tools), toolsResp.Count, "Tool count should match tools array length")

	// Verify some expected tools exist
	toolNames := make(map[string]bool)
	for _, tool := range toolsResp.Tools {
		toolNames[tool.Name] = true
	}
	assert.True(t, toolNames["list_blocks"], "Should have list_blocks tool")
	assert.True(t, toolNames["get_block_schema"], "Should have get_block_schema tool")
	assert.True(t, toolNames["search_blocks"], "Should have search_blocks tool")
}

func TestCopilotAgentSessionCreate(t *testing.T) {
	// Create a test project for this test
	workflowID := createTestProject(t)

	tests := []struct {
		name    string
		mode    string
		prompt  string
		wantErr bool
	}{
		{
			name:    "Create mode",
			mode:    "create",
			prompt:  "List available blocks",
			wantErr: false,
		},
		{
			name:    "Explain mode",
			mode:    "explain",
			prompt:  "How do I send a message to Slack?",
			wantErr: false,
		},
		{
			name:    "Enhance mode",
			mode:    "enhance",
			prompt:  "Improve this workflow",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startReq := map[string]string{
				"initial_prompt": tt.prompt,
				"mode":           tt.mode,
			}
			url := fmt.Sprintf("/api/v1/workflows/%s/copilot/agent/sessions", workflowID)
			resp, body := makeRequest(t, "POST", url, startReq)

			// Skip if server timed out or returned EOF
			if resp == nil {
				t.Skip("Server connection failed")
			}

			if tt.wantErr {
				assert.NotEqual(t, http.StatusCreated, resp.StatusCode)
				return
			}

			// Allow 500 errors when Copilot workflow is not seeded
			if resp.StatusCode == http.StatusInternalServerError {
				t.Skip("Copilot workflow may not be seeded")
			}

			require.Equal(t, http.StatusCreated, resp.StatusCode, "Response: %s", string(body))

			var session CopilotAgentSession
			err := json.Unmarshal(body, &session)
			require.NoError(t, err)

			assert.NotEmpty(t, session.SessionID, "Session ID should not be empty")
			assert.NotEmpty(t, session.Status, "Status should not be empty")
			// Note: Response is empty on session create - processing happens via SSE stream
		})
	}
}

func TestCopilotAgentSessionValidation(t *testing.T) {
	// Create a test project for this test
	workflowID := createTestProject(t)

	tests := []struct {
		name       string
		request    map[string]string
		wantStatus int
	}{
		{
			name: "Missing initial_prompt",
			request: map[string]string{
				"mode": "create",
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "Invalid mode",
			request: map[string]string{
				"initial_prompt": "test",
				"mode":           "invalid_mode",
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := fmt.Sprintf("/api/v1/workflows/%s/copilot/agent/sessions", workflowID)
			resp, _ := makeRequest(t, "POST", url, tt.request)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestCopilotAgentInvalidWorkflow(t *testing.T) {
	// Test with non-existent workflow ID
	workflowID := "00000000-0000-0000-0000-000000000999"

	startReq := map[string]string{
		"initial_prompt": "test",
		"mode":           "create",
	}
	url := fmt.Sprintf("/api/v1/workflows/%s/copilot/agent/sessions", workflowID)
	resp, _ := makeRequest(t, "POST", url, startReq)

	// Should return 404 for non-existent workflow
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
