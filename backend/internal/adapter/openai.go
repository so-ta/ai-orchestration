package adapter

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

// OpenAIAdapter implements the Adapter interface for OpenAI API
type OpenAIAdapter struct {
	id         string
	name       string
	httpClient *http.Client
	apiKey     string
	baseURL    string
}

// OpenAIConfig holds the configuration for OpenAI adapter
type OpenAIConfig struct {
	Model       string   `json:"model"`        // gpt-4, gpt-4-turbo, gpt-3.5-turbo
	Prompt      string   `json:"prompt"`       // User prompt template with {{variable}} placeholders
	System      string   `json:"system"`       // System message
	Temperature *float64 `json:"temperature"`  // 0.0 - 2.0 (nil = use default 0.7)
	MaxTokens   int      `json:"max_tokens"`   // Maximum tokens to generate
	TopP        float64  `json:"top_p"`        // Nucleus sampling
	Stop        []string `json:"stop"`         // Stop sequences
}

// OpenAI API request/response types
type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature float64         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	TopP        float64         `json:"top_p,omitempty"`
	Stop        []string        `json:"stop,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int           `json:"index"`
		Message      openAIMessage `json:"message"`
		FinishReason string        `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *openAIError `json:"error,omitempty"`
}

type openAIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// NewOpenAIAdapter creates a new OpenAI adapter
func NewOpenAIAdapter() *OpenAIAdapter {
	return &OpenAIAdapter{
		id:   "openai",
		name: "OpenAI",
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		apiKey:  os.Getenv("OPENAI_API_KEY"),
		baseURL: getEnvOrDefault("OPENAI_BASE_URL", "https://api.openai.com/v1"),
	}
}

// NewOpenAIAdapterWithKey creates an OpenAI adapter with a specific API key
func NewOpenAIAdapterWithKey(apiKey string) *OpenAIAdapter {
	adapter := NewOpenAIAdapter()
	adapter.apiKey = apiKey
	return adapter
}

func (a *OpenAIAdapter) ID() string   { return a.id }
func (a *OpenAIAdapter) Name() string { return a.name }

// Execute runs the OpenAI adapter
func (a *OpenAIAdapter) Execute(ctx context.Context, req *Request) (*Response, error) {
	start := time.Now()

	// Check API key
	apiKey := a.apiKey
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key not configured")
	}

	// Parse config
	var config OpenAIConfig
	if req.Config != nil {
		if err := json.Unmarshal(req.Config, &config); err != nil {
			return nil, fmt.Errorf("invalid OpenAI config: %w", err)
		}
	}

	// Set defaults
	if config.Model == "" {
		config.Model = "gpt-4"
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = 2048
	}

	// Handle temperature: use default 0.7 only if not explicitly set (nil)
	var temperature float64 = 0.7
	if config.Temperature != nil {
		temperature = *config.Temperature
	}

	// Config templates are now expanded by Executor before reaching the adapter
	// Prompt can be used directly from config
	prompt := config.Prompt

	// Build messages
	messages := []openAIMessage{}
	if config.System != "" {
		messages = append(messages, openAIMessage{
			Role:    "system",
			Content: config.System,
		})
	}
	messages = append(messages, openAIMessage{
		Role:    "user",
		Content: prompt,
	})

	// Build request
	apiReq := openAIRequest{
		Model:       config.Model,
		Messages:    messages,
		Temperature: temperature,
		MaxTokens:   config.MaxTokens,
	}
	if config.TopP > 0 {
		apiReq.TopP = config.TopP
	}
	if len(config.Stop) > 0 {
		apiReq.Stop = config.Stop
	}

	// Make HTTP request
	reqBody, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call OpenAI API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp openAIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API errors
	if apiResp.Error != nil {
		return nil, fmt.Errorf("OpenAI API error: %s (type: %s, code: %s)",
			apiResp.Error.Message, apiResp.Error.Type, apiResp.Error.Code)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API returned status %d: %s", resp.StatusCode, string(body))
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("OpenAI API returned no choices")
	}

	// Build output
	output := map[string]interface{}{
		"content":       apiResp.Choices[0].Message.Content,
		"model":         apiResp.Model,
		"finish_reason": apiResp.Choices[0].FinishReason,
		"usage": map[string]int{
			"prompt_tokens":     apiResp.Usage.PromptTokens,
			"completion_tokens": apiResp.Usage.CompletionTokens,
			"total_tokens":      apiResp.Usage.TotalTokens,
		},
	}

	outputJSON, err := json.Marshal(output)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal output: %w", err)
	}

	return &Response{
		Output:     outputJSON,
		DurationMs: int(time.Since(start).Milliseconds()),
		Metadata: map[string]string{
			"adapter":           a.id,
			"model":             apiResp.Model,
			"prompt_tokens":     fmt.Sprintf("%d", apiResp.Usage.PromptTokens),
			"completion_tokens": fmt.Sprintf("%d", apiResp.Usage.CompletionTokens),
			"total_tokens":      fmt.Sprintf("%d", apiResp.Usage.TotalTokens),
		},
	}, nil
}

func (a *OpenAIAdapter) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"description": "Input data for variable substitution in the prompt template",
		"additionalProperties": true
	}`)
}

func (a *OpenAIAdapter) OutputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"content": {"type": "string", "description": "Generated text content"},
			"model": {"type": "string", "description": "Model used"},
			"finish_reason": {"type": "string", "description": "Reason for completion"},
			"usage": {
				"type": "object",
				"properties": {
					"prompt_tokens": {"type": "integer"},
					"completion_tokens": {"type": "integer"},
					"total_tokens": {"type": "integer"}
				}
			}
		},
		"required": ["content"]
	}`)
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
