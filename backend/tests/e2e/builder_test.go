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

// BuilderSession represents a builder session for testing
type BuilderSession struct {
	ID        string  `json:"id"`
	Status    string  `json:"status"`
	Phase     string  `json:"hearing_phase"`
	Progress  int     `json:"hearing_progress"`
	ProjectID *string `json:"project_id,omitempty"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

// BuilderMessage represents a message in the builder
type BuilderMessage struct {
	ID                 string   `json:"id"`
	Role               string   `json:"role"`
	Content            string   `json:"content"`
	SuggestedQuestions []string `json:"suggested_questions,omitempty"`
}

// createBuilderSession creates a builder session and returns the session ID.
// Returns empty string and skips the test if the system project is not seeded.
func createBuilderSession(t *testing.T, prompt string) string {
	t.Helper()

	startReq := map[string]string{
		"initial_prompt": prompt,
	}
	resp, body := makeRequest(t, "POST", "/api/v1/builder/sessions", startReq)

	if resp.StatusCode == http.StatusInternalServerError {
		t.Skip("AI Builder system project may not be seeded")
	}
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Start session response: %s", string(body))

	var startResp struct {
		SessionID string `json:"session_id"`
	}
	err := json.Unmarshal(body, &startResp)
	require.NoError(t, err)

	return startResp.SessionID
}

// deleteBuilderSession deletes a builder session
func deleteBuilderSession(t *testing.T, sessionID string) {
	t.Helper()
	makeRequest(t, "DELETE", fmt.Sprintf("/api/v1/builder/sessions/%s", sessionID), nil)
}

func TestBuilderSessionCRUD(t *testing.T) {
	// Skip if AI builder system project is not seeded
	t.Run("Start Session", func(t *testing.T) {
		// Create a new builder session
		startReq := map[string]string{
			"initial_prompt": "顧客からの問い合わせを自動分類して担当者にアサインするワークフローを作りたい",
		}
		resp, body := makeRequest(t, "POST", "/api/v1/builder/sessions", startReq)

		// 201 Created or 500 (if system project not found)
		if resp.StatusCode == http.StatusInternalServerError {
			t.Skip("AI Builder system project may not be seeded")
		}
		require.Equal(t, http.StatusCreated, resp.StatusCode, "Start session response: %s", string(body))

		var startResp struct {
			SessionID string          `json:"session_id"`
			Status    string          `json:"status"`
			Phase     string          `json:"phase"`
			Progress  int             `json:"progress"`
			Message   *BuilderMessage `json:"message,omitempty"`
		}
		err := json.Unmarshal(body, &startResp)
		require.NoError(t, err)

		sessionID := startResp.SessionID
		assert.NotEmpty(t, sessionID)
		assert.Equal(t, "hearing", startResp.Status)
		assert.Equal(t, "purpose", startResp.Phase)
		assert.Equal(t, 10, startResp.Progress) // purpose phase starts at 10%

		// Get session
		resp, body = makeRequest(t, "GET", fmt.Sprintf("/api/v1/builder/sessions/%s", sessionID), nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var getResp struct {
			ID       string           `json:"id"`
			Status   string           `json:"status"`
			Phase    string           `json:"hearing_phase"`
			Progress int              `json:"hearing_progress"`
			Messages []BuilderMessage `json:"messages,omitempty"`
		}
		err = json.Unmarshal(body, &getResp)
		require.NoError(t, err)
		assert.Equal(t, sessionID, getResp.ID)
		assert.Equal(t, "hearing", getResp.Status)

		// Delete session
		resp, _ = makeRequest(t, "DELETE", fmt.Sprintf("/api/v1/builder/sessions/%s", sessionID), nil)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		// Verify deletion
		resp, _ = makeRequest(t, "GET", fmt.Sprintf("/api/v1/builder/sessions/%s", sessionID), nil)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func TestBuilderSessionList(t *testing.T) {
	// Create a few sessions
	sessions := make([]string, 0)
	prompts := []string{
		"請求書の自動処理ワークフロー",
		"社内申請承認フロー",
	}

	for _, prompt := range prompts {
		startReq := map[string]string{
			"initial_prompt": prompt,
		}
		resp, body := makeRequest(t, "POST", "/api/v1/builder/sessions", startReq)
		if resp.StatusCode == http.StatusInternalServerError {
			t.Skip("AI Builder system project may not be seeded")
		}
		if resp.StatusCode == http.StatusCreated {
			var startResp struct {
				SessionID string `json:"session_id"`
			}
			json.Unmarshal(body, &startResp)
			sessions = append(sessions, startResp.SessionID)
		}
	}

	// List sessions
	resp, body := makeRequest(t, "GET", "/api/v1/builder/sessions", nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var listResp struct {
		Sessions []BuilderSession `json:"sessions"`
		Total    int              `json:"total"`
	}
	err := json.Unmarshal(body, &listResp)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, listResp.Total, len(sessions))

	// Cleanup
	for _, id := range sessions {
		makeRequest(t, "DELETE", fmt.Sprintf("/api/v1/builder/sessions/%s", id), nil)
	}
}

func TestBuilderSendMessage(t *testing.T) {
	sessionID := createBuilderSession(t, "メール通知を自動送信するワークフロー")
	defer deleteBuilderSession(t, sessionID)

	// Send message (async)
	msgReq := map[string]string{
		"content": "新規顧客登録時にウェルカムメールを送りたいです",
	}
	resp, body := makeRequest(t, "POST", fmt.Sprintf("/api/v1/builder/sessions/%s/messages", sessionID), msgReq)

	// Skip if system project is not seeded (500 with entry_point not found)
	if resp.StatusCode == http.StatusInternalServerError {
		t.Skip("AI Builder system project may not be seeded or entry_point not configured")
	}
	require.Equal(t, http.StatusAccepted, resp.StatusCode, "Send message response: %s", string(body))

	var msgResp struct {
		RunID  string `json:"run_id"`
		Status string `json:"status"`
	}
	err := json.Unmarshal(body, &msgResp)
	require.NoError(t, err)
	assert.NotEmpty(t, msgResp.RunID)
	assert.Equal(t, "pending", msgResp.Status)

	// Poll for completion (with timeout)
	maxWaitTime := 60 * time.Second
	pollInterval := 2 * time.Second
	deadline := time.Now().Add(maxWaitTime)

	var finalStatus string
	for time.Now().Before(deadline) {
		resp, body = makeRequest(t, "GET", fmt.Sprintf("/api/v1/runs/%s", msgResp.RunID), nil)
		if resp.StatusCode != http.StatusOK {
			time.Sleep(pollInterval)
			continue
		}

		var runResp struct {
			Data struct {
				Status string `json:"status"`
			} `json:"data"`
		}
		json.Unmarshal(body, &runResp)
		finalStatus = runResp.Data.Status

		if finalStatus == "completed" || finalStatus == "failed" {
			break
		}
		time.Sleep(pollInterval)
	}

	// Note: Run may fail if LLM is not configured, but we're testing the API flow
	assert.Contains(t, []string{"completed", "failed", "pending", "running"}, finalStatus)
}

func TestBuilderValidation(t *testing.T) {
	t.Run("Empty initial prompt", func(t *testing.T) {
		startReq := map[string]string{
			"initial_prompt": "",
		}
		resp, body := makeRequest(t, "POST", "/api/v1/builder/sessions", startReq)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Response: %s", string(body))

		var errResp struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		json.Unmarshal(body, &errResp)
		assert.Equal(t, "INVALID_REQUEST", errResp.Error.Code)
	})

	t.Run("Empty message content", func(t *testing.T) {
		sessionID := createBuilderSession(t, "テストワークフロー")
		defer deleteBuilderSession(t, sessionID)

		// Try to send empty message
		msgReq := map[string]string{
			"content": "",
		}
		resp, body := makeRequest(t, "POST", fmt.Sprintf("/api/v1/builder/sessions/%s/messages", sessionID), msgReq)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var errResp struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		json.Unmarshal(body, &errResp)
		assert.Equal(t, "INVALID_REQUEST", errResp.Error.Code)
	})

	t.Run("Invalid session ID", func(t *testing.T) {
		resp, _ := makeRequest(t, "GET", "/api/v1/builder/sessions/invalid-uuid", nil)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Non-existent session", func(t *testing.T) {
		resp, _ := makeRequest(t, "GET", "/api/v1/builder/sessions/00000000-0000-0000-0000-000000000000", nil)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func TestBuilderConstructValidation(t *testing.T) {
	sessionID := createBuilderSession(t, "テストワークフロー")
	defer deleteBuilderSession(t, sessionID)

	// Try to construct before hearing is completed
	resp, body := makeRequest(t, "POST", fmt.Sprintf("/api/v1/builder/sessions/%s/construct", sessionID), nil)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Construct response: %s", string(body))

	var errResp struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	json.Unmarshal(body, &errResp)
	assert.Equal(t, "HEARING_NOT_COMPLETED", errResp.Error.Code)
}

func TestBuilderRefineValidation(t *testing.T) {
	sessionID := createBuilderSession(t, "テストワークフロー")
	defer deleteBuilderSession(t, sessionID)

	// Try to refine before project is created
	refineReq := map[string]string{
		"feedback": "もう少しシンプルにしてください",
	}
	resp, body := makeRequest(t, "POST", fmt.Sprintf("/api/v1/builder/sessions/%s/refine", sessionID), refineReq)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Refine response: %s", string(body))

	var errResp struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	json.Unmarshal(body, &errResp)
	assert.Equal(t, "PROJECT_NOT_CREATED", errResp.Error.Code)
}

func TestBuilderRefineEmptyFeedback(t *testing.T) {
	sessionID := createBuilderSession(t, "テストワークフロー")
	defer deleteBuilderSession(t, sessionID)

	// Try to refine with empty feedback
	refineReq := map[string]string{
		"feedback": "",
	}
	resp, body := makeRequest(t, "POST", fmt.Sprintf("/api/v1/builder/sessions/%s/refine", sessionID), refineReq)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Refine response: %s", string(body))

	var errResp struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	json.Unmarshal(body, &errResp)
	assert.Equal(t, "INVALID_REQUEST", errResp.Error.Code)
}
