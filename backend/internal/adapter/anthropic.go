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

// AnthropicAdapter implements the Adapter interface for Anthropic Claude API
type AnthropicAdapter struct {
	id         string
	name       string
	httpClient *http.Client
	apiKey     string
	baseURL    string
}

// AnthropicConfig holds the configuration for Anthropic adapter
type AnthropicConfig struct {
	Model       string   `json:"model"`        // claude-3-opus-20240229, claude-3-sonnet-20240229, claude-3-haiku-20240307
	Prompt      string   `json:"prompt"`       // User prompt template with {{variable}} placeholders
	System      string   `json:"system"`       // System message
	MaxTokens   int      `json:"max_tokens"`   // Maximum tokens to generate (required by Anthropic)
	Temperature *float64 `json:"temperature"`  // 0.0 - 1.0 (nil = use default 0.7)
	TopP        float64  `json:"top_p"`        // Nucleus sampling
	TopK        int      `json:"top_k"`        // Top-k sampling
	Stop        []string `json:"stop"`         // Stop sequences
}

// Anthropic API request/response types
type anthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	Messages    []anthropicMessage `json:"messages"`
	System      string             `json:"system,omitempty"`
	Temperature float64            `json:"temperature,omitempty"`
	TopP        float64            `json:"top_p,omitempty"`
	TopK        int                `json:"top_k,omitempty"`
	StopSeq     []string           `json:"stop_sequences,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Role         string `json:"role"`
	Model        string `json:"model"`
	StopReason   string `json:"stop_reason"`
	StopSequence string `json:"stop_sequence,omitempty"`
	Content      []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error *anthropicError `json:"error,omitempty"`
}

type anthropicError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// NewAnthropicAdapter creates a new Anthropic adapter
func NewAnthropicAdapter() *AnthropicAdapter {
	return &AnthropicAdapter{
		id:   "anthropic",
		name: "Anthropic Claude",
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		apiKey:  os.Getenv("ANTHROPIC_API_KEY"),
		baseURL: getEnvOrDefault("ANTHROPIC_BASE_URL", "https://api.anthropic.com"),
	}
}

// NewAnthropicAdapterWithKey creates an Anthropic adapter with a specific API key
func NewAnthropicAdapterWithKey(apiKey string) *AnthropicAdapter {
	adapter := NewAnthropicAdapter()
	adapter.apiKey = apiKey
	return adapter
}

func (a *AnthropicAdapter) ID() string   { return a.id }
func (a *AnthropicAdapter) Name() string { return a.name }

// Execute runs the Anthropic adapter
func (a *AnthropicAdapter) Execute(ctx context.Context, req *Request) (*Response, error) {
	start := time.Now()

	// Check API key
	apiKey := a.apiKey
	if apiKey == "" {
		return nil, fmt.Errorf("Anthropic API key not configured")
	}

	// Parse config
	var config AnthropicConfig
	if req.Config != nil {
		if err := json.Unmarshal(req.Config, &config); err != nil {
			return nil, fmt.Errorf("invalid Anthropic config: %w", err)
		}
	}

	// Set defaults
	if config.Model == "" {
		config.Model = "claude-3-sonnet-20240229"
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = 4096
	}

	// Handle temperature: use default 0.7 only if not explicitly set (nil)
	var temperature float64 = 0.7
	if config.Temperature != nil {
		temperature = *config.Temperature
	}

	// Config templates are now expanded by Executor before reaching the adapter
	// Prompt can be used directly from config
	prompt := config.Prompt

	// Build request
	apiReq := anthropicRequest{
		Model:     config.Model,
		MaxTokens: config.MaxTokens,
		Messages: []anthropicMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	if config.System != "" {
		apiReq.System = config.System
	}
	// Always set temperature (use the computed value which is either user-specified or default)
	apiReq.Temperature = temperature
	if config.TopP > 0 {
		apiReq.TopP = config.TopP
	}
	if config.TopK > 0 {
		apiReq.TopK = config.TopK
	}
	if len(config.Stop) > 0 {
		apiReq.StopSeq = config.Stop
	}

	// Make HTTP request
	reqBody, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/v1/messages", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call Anthropic API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API errors
	if apiResp.Error != nil {
		return nil, fmt.Errorf("Anthropic API error: %s (type: %s)",
			apiResp.Error.Message, apiResp.Error.Type)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Anthropic API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Extract content
	var content string
	for _, block := range apiResp.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	// Build output
	output := map[string]interface{}{
		"content":     content,
		"model":       apiResp.Model,
		"stop_reason": apiResp.StopReason,
		"usage": map[string]int{
			"input_tokens":  apiResp.Usage.InputTokens,
			"output_tokens": apiResp.Usage.OutputTokens,
			"total_tokens":  apiResp.Usage.InputTokens + apiResp.Usage.OutputTokens,
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
			"adapter":       a.id,
			"model":         apiResp.Model,
			"input_tokens":  fmt.Sprintf("%d", apiResp.Usage.InputTokens),
			"output_tokens": fmt.Sprintf("%d", apiResp.Usage.OutputTokens),
			"stop_reason":   apiResp.StopReason,
		},
	}, nil
}

func (a *AnthropicAdapter) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"description": "Input data for variable substitution in the prompt template",
		"additionalProperties": true
	}`)
}

func (a *AnthropicAdapter) OutputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"content": {"type": "string", "description": "Generated text content"},
			"model": {"type": "string", "description": "Model used"},
			"stop_reason": {"type": "string", "description": "Reason for stopping"},
			"usage": {
				"type": "object",
				"properties": {
					"input_tokens": {"type": "integer"},
					"output_tokens": {"type": "integer"},
					"total_tokens": {"type": "integer"}
				}
			}
		},
		"required": ["content"]
	}`)
}
