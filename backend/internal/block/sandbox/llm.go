package sandbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// LLMServiceImpl implements LLMService for sandbox scripts
type LLMServiceImpl struct {
	httpClient    *http.Client
	ctx           context.Context
	openaiBaseURL string
	anthropicBaseURL string
}

// NewLLMService creates a new LLMService
func NewLLMService(ctx context.Context) *LLMServiceImpl {
	return &LLMServiceImpl{
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		ctx:              ctx,
		openaiBaseURL:    getEnvOrDefault("OPENAI_BASE_URL", "https://api.openai.com"),
		anthropicBaseURL: getEnvOrDefault("ANTHROPIC_BASE_URL", "https://api.anthropic.com"),
	}
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// Chat performs a chat completion request
// Supported providers: openai, anthropic
func (s *LLMServiceImpl) Chat(provider, model string, request map[string]interface{}) (map[string]interface{}, error) {
	switch provider {
	case "openai":
		return s.chatOpenAI(model, request)
	case "anthropic":
		return s.chatAnthropic(model, request)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s (supported: openai, anthropic)", provider)
	}
}

// chatOpenAI calls OpenAI's chat completion API
func (s *LLMServiceImpl) chatOpenAI(model string, request map[string]interface{}) (map[string]interface{}, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("LLM provider (openai) is not configured")
	}

	// Default model
	if model == "" {
		model = "gpt-4"
	}

	// Build OpenAI request
	openaiReq := map[string]interface{}{
		"model": model,
	}

	// Copy messages
	if messages, ok := request["messages"]; ok {
		openaiReq["messages"] = messages
	}

	// Copy optional parameters
	if temp, ok := request["temperature"]; ok {
		openaiReq["temperature"] = temp
	}
	if maxTokens, ok := request["max_tokens"]; ok {
		openaiReq["max_tokens"] = maxTokens
	}
	if topP, ok := request["top_p"]; ok {
		openaiReq["top_p"] = topP
	}
	if stop, ok := request["stop"]; ok {
		openaiReq["stop"] = stop
	}

	jsonBody, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(s.ctx, "POST", s.openaiBaseURL+"/v1/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("chat request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var respData struct {
		ID      string `json:"id"`
		Choices []struct {
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &respData); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract content
	content := ""
	if len(respData.Choices) > 0 {
		content = respData.Choices[0].Message.Content
	}

	return map[string]interface{}{
		"content": content,
		"usage": map[string]interface{}{
			"input_tokens":  respData.Usage.PromptTokens,
			"output_tokens": respData.Usage.CompletionTokens,
		},
	}, nil
}

// chatAnthropic calls Anthropic's messages API
func (s *LLMServiceImpl) chatAnthropic(model string, request map[string]interface{}) (map[string]interface{}, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("LLM provider (anthropic) is not configured")
	}

	// Default model
	if model == "" {
		model = "claude-3-5-sonnet-20241022"
	}

	// Build Anthropic request
	anthropicReq := map[string]interface{}{
		"model": model,
	}

	// Convert messages format (Anthropic requires system message separate)
	if messages, ok := request["messages"].([]interface{}); ok {
		var systemMsg string
		var anthropicMsgs []map[string]interface{}

		for _, m := range messages {
			msg, ok := m.(map[string]interface{})
			if !ok {
				continue
			}
			role, _ := msg["role"].(string)
			content, _ := msg["content"].(string)

			if role == "system" {
				systemMsg = content
			} else {
				anthropicMsgs = append(anthropicMsgs, map[string]interface{}{
					"role":    role,
					"content": content,
				})
			}
		}

		if systemMsg != "" {
			anthropicReq["system"] = systemMsg
		}
		anthropicReq["messages"] = anthropicMsgs
	}

	// Max tokens is required for Anthropic
	if maxTokens, ok := request["max_tokens"]; ok {
		anthropicReq["max_tokens"] = maxTokens
	} else {
		anthropicReq["max_tokens"] = 4096 // Default
	}

	// Copy optional parameters
	if temp, ok := request["temperature"]; ok {
		anthropicReq["temperature"] = temp
	}
	if topP, ok := request["top_p"]; ok {
		anthropicReq["top_p"] = topP
	}
	if stop, ok := request["stop"]; ok {
		anthropicReq["stop_sequences"] = stop
	}

	jsonBody, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(s.ctx, "POST", s.anthropicBaseURL+"/v1/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("chat request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Anthropic API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var respData struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &respData); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract content
	content := ""
	for _, c := range respData.Content {
		if c.Type == "text" {
			content += c.Text
		}
	}

	return map[string]interface{}{
		"content": content,
		"usage": map[string]interface{}{
			"input_tokens":  respData.Usage.InputTokens,
			"output_tokens": respData.Usage.OutputTokens,
		},
	}, nil
}
