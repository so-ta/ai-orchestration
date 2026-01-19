// +build ignore

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	baseURL  = "http://localhost:8090"
	tenantID = "00000000-0000-0000-0000-000000000001"
)

type TestCase struct {
	Name        string
	WorkflowID  string
	Mode        string
	Message     string
	ExpectTools []string
}

func main() {
	fmt.Println("=== Copilot Agent Flow Test ===")
	fmt.Println()

	// Test Cases
	testCases := []TestCase{
		{
			Name:        "Explain Mode: Slackへの通知方法",
			WorkflowID:  "a0000000-0000-0000-0000-000000000200", // Demo Workflows
			Mode:        "explain",
			Message:     "Slackにメッセージを送信する方法を教えてください",
			ExpectTools: []string{"search_documentation", "search_blocks"},
		},
		{
			Name:        "Explain Mode: LLMブロックの使い方",
			WorkflowID:  "a0000000-0000-0000-0000-000000000200", // Demo Workflows
			Mode:        "explain",
			Message:     "LLMブロックでAIを呼び出すにはどうすればいいですか？設定項目を教えてください。",
			ExpectTools: []string{"search_blocks", "get_block_schema"},
		},
		{
			Name:        "Create Mode: 簡単なワークフロー作成",
			WorkflowID:  "a0000000-0000-0000-0000-000000000200", // Demo Workflows
			Mode:        "create",
			Message:     "利用可能なブロックの一覧を教えてください",
			ExpectTools: []string{"list_blocks"},
		},
	}

	for i, tc := range testCases {
		fmt.Printf("--- Test %d: %s ---\n", i+1, tc.Name)
		runTestCase(tc)
		fmt.Println()
		time.Sleep(2 * time.Second) // Rate limiting
	}

	fmt.Println("=== All Tests Completed ===")
}

func runTestCase(tc TestCase) {
	// 1. Start agent session
	sessionID, response, err := startSession(tc.WorkflowID, tc.Message, tc.Mode)
	if err != nil {
		fmt.Printf("❌ Failed to start session: %v\n", err)
		return
	}
	fmt.Printf("✅ Session started: %s\n", sessionID)
	fmt.Printf("   Response: %s\n", truncate(response, 200))

	// 2. Test SSE streaming with follow-up
	fmt.Println("\n   Testing SSE streaming...")
	toolsUsed, streamResponse, err := testStreaming(tc.WorkflowID, sessionID, "その他に何かありますか？")
	if err != nil {
		fmt.Printf("⚠️  SSE streaming test: %v\n", err)
	} else {
		fmt.Printf("✅ SSE streaming works\n")
		fmt.Printf("   Tools used: %v\n", toolsUsed)
		fmt.Printf("   Response: %s\n", truncate(streamResponse, 200))
	}

	// 3. Verify expected tools were available
	fmt.Println("\n   Checking available tools...")
	tools, err := getAvailableTools()
	if err != nil {
		fmt.Printf("⚠️  Failed to get tools: %v\n", err)
	} else {
		for _, expected := range tc.ExpectTools {
			found := false
			for _, tool := range tools {
				if tool == expected {
					found = true
					break
				}
			}
			if found {
				fmt.Printf("   ✅ Tool '%s' is available\n", expected)
			} else {
				fmt.Printf("   ⚠️  Tool '%s' not found\n", expected)
			}
		}
	}
}

func startSession(workflowID, message, mode string) (string, string, error) {
	url := baseURL + "/api/v1/copilot/agent/sessions"
	if workflowID != "" {
		url = baseURL + "/api/v1/workflows/" + workflowID + "/copilot/agent/sessions"
	}

	body := map[string]interface{}{
		"initial_prompt": message,
		"mode":           mode,
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantID)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		SessionID string `json:"session_id"`
		Response  string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", err
	}

	return result.SessionID, result.Response, nil
}

func testStreaming(workflowID, sessionID, message string) ([]string, string, error) {
	url := fmt.Sprintf("%s/api/v1/workflows/%s/copilot/agent/sessions/%s/stream?message=%s",
		baseURL, workflowID, sessionID, message)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("Accept", "text/event-stream")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var toolsUsed []string
	var finalResponse string

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			var event map[string]interface{}
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}

			if eventType, ok := event["type"].(string); ok {
				switch eventType {
				case "tool_call":
					if eventData, ok := event["data"].(map[string]interface{}); ok {
						if tool, ok := eventData["tool"].(string); ok {
							toolsUsed = append(toolsUsed, tool)
						}
					}
				case "complete":
					if eventData, ok := event["data"].(map[string]interface{}); ok {
						if response, ok := eventData["response"].(string); ok {
							finalResponse = response
						}
					}
				}
			}
		}
	}

	return toolsUsed, finalResponse, scanner.Err()
}

func getAvailableTools() ([]string, error) {
	url := baseURL + "/api/v1/copilot/agent/tools"

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("X-Tenant-ID", tenantID)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var tools []string
	for _, t := range result.Tools {
		tools = append(tools, t.Name)
	}
	return tools, nil
}

func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
