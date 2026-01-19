// +build ignore

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	baseURL  = "http://localhost:8090/api/v1"
	tenantID = "00000000-0000-0000-0000-000000000001"
)

type Session struct {
	ID           string `json:"id"`
	SessionID    string `json:"session_id"`
	Status       string `json:"status"`
	Phase        string `json:"phase"`
	HearingPhase string `json:"hearing_phase"`
	ProjectID    string `json:"project_id"`
}

func (s *Session) GetID() string {
	if s.SessionID != "" {
		return s.SessionID
	}
	return s.ID
}

func (s *Session) GetPhase() string {
	if s.Phase != "" {
		return s.Phase
	}
	return s.HearingPhase
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Run struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	Error       string `json:"error"`
	CompletedAt string `json:"completed_at"`
}

func main() {
	fmt.Println("=== Builder Flow E2E Test ===\n")

	// Step 1: Create session with initial request
	fmt.Println("Step 1: Creating builder session...")
	session := createSession("weekly report automation")
	sessionID := session.GetID()
	fmt.Printf("  Session ID: %s\n", sessionID)
	fmt.Printf("  Status: %s\n", session.Status)
	fmt.Printf("  Phase: %s\n", session.GetPhase())

	// The initial response already includes analysis, so check phase
	if session.GetPhase() == "analysis" {
		fmt.Println("\nStep 2: Analysis already completed in initial response")
	} else {
		fmt.Println("\nStep 2: Waiting for analysis phase to complete...")
		session = waitForPhase(sessionID, "proposal", 60)
	}
	fmt.Printf("  Phase: %s\n", session.GetPhase())

	// Step 3: Send confirmation message
	fmt.Println("\nStep 3: Sending confirmation message...")
	sendMessage(sessionID, "OK, proceed with these assumptions")

	// Wait for proposal to complete
	fmt.Println("\nStep 4: Waiting for proposal phase to complete...")
	session = waitForPhase(sessionID, "completed", 60)
	fmt.Printf("  Phase: %s\n", session.GetPhase())

	// Step 5: Trigger workflow construction
	fmt.Println("\nStep 5: Triggering workflow construction...")
	runID := triggerConstruct(sessionID)
	fmt.Printf("  Run ID: %s\n", runID)

	// Step 6: Wait for construction to complete
	fmt.Println("\nStep 6: Waiting for construction to complete...")
	run := waitForRunComplete(runID, 120)
	fmt.Printf("  Run Status: %s\n", run.Status)
	if run.Error != "" {
		fmt.Printf("  Error: %s\n", run.Error)
	}

	// Step 7: Check if project was created
	fmt.Println("\nStep 7: Checking if project was created...")
	session = getSession(sessionID)
	fmt.Printf("  Session Status: %s\n", session.Status)
	fmt.Printf("  Project ID: %s\n", session.ProjectID)

	if session.ProjectID != "" && run.Status == "completed" {
		fmt.Println("\n✅ SUCCESS: Workflow was created successfully!")
		fmt.Printf("   Project ID: %s\n", session.ProjectID)
	} else {
		fmt.Println("\n❌ FAILED: Workflow creation failed")
		if run.Error != "" {
			fmt.Printf("   Error: %s\n", run.Error)
		}
	}
}

func createSession(initialMessage string) *Session {
	body := map[string]interface{}{
		"initial_prompt": initialMessage,
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", baseURL+"/builder/sessions", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantID)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		panic(fmt.Sprintf("Failed to create session: %v", err))
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 201 {
		panic(fmt.Sprintf("Failed to create session: %s - %s", resp.Status, string(respBody)))
	}

	var session Session
	json.Unmarshal(respBody, &session)
	return &session
}

func getSession(sessionID string) *Session {
	req, _ := http.NewRequest("GET", baseURL+"/builder/sessions/"+sessionID, nil)
	req.Header.Set("X-Tenant-ID", tenantID)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		panic(fmt.Sprintf("Failed to get session: %v", err))
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var session Session
	json.Unmarshal(respBody, &session)
	return &session
}

func waitForPhase(sessionID string, targetPhase string, timeoutSec int) *Session {
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	for time.Now().Before(deadline) {
		session := getSession(sessionID)
		phase := session.GetPhase()
		fmt.Printf("  Current phase: %s\n", phase)
		if phase == targetPhase {
			return session
		}
		time.Sleep(2 * time.Second)
	}
	panic(fmt.Sprintf("Timeout waiting for phase %s", targetPhase))
}

func sendMessage(sessionID string, content string) {
	body := map[string]interface{}{
		"content": content,
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", baseURL+"/builder/sessions/"+sessionID+"/messages", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantID)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		panic(fmt.Sprintf("Failed to send message: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != 202 && resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		panic(fmt.Sprintf("Failed to send message: %s - %s", resp.Status, string(respBody)))
	}
	fmt.Println("  Message sent successfully")
}

func triggerConstruct(sessionID string) string {
	req, _ := http.NewRequest("POST", baseURL+"/builder/sessions/"+sessionID+"/construct", nil)
	req.Header.Set("X-Tenant-ID", tenantID)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		panic(fmt.Sprintf("Failed to trigger construct: %v", err))
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 202 && resp.StatusCode != 200 {
		panic(fmt.Sprintf("Failed to trigger construct: %s - %s", resp.Status, string(respBody)))
	}

	var result struct {
		RunID string `json:"run_id"`
	}
	json.Unmarshal(respBody, &result)
	return result.RunID
}

func waitForRunComplete(runID string, timeoutSec int) *Run {
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	for time.Now().Before(deadline) {
		run := getRun(runID)
		fmt.Printf("  Run status: %s\n", run.Status)
		if run.Status == "completed" || run.Status == "failed" {
			return run
		}
		time.Sleep(2 * time.Second)
	}
	panic("Timeout waiting for run to complete")
}

func getRun(runID string) *Run {
	req, _ := http.NewRequest("GET", baseURL+"/runs/"+runID, nil)
	req.Header.Set("X-Tenant-ID", tenantID)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		panic(fmt.Sprintf("Failed to get run: %v", err))
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result struct {
		Data Run `json:"data"`
	}
	json.Unmarshal(respBody, &result)
	return &result.Data
}
