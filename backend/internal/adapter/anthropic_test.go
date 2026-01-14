package adapter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnthropicAdapter_Execute(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/messages", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.NotEmpty(t, r.Header.Get("x-api-key"))
		assert.Equal(t, "2023-06-01", r.Header.Get("anthropic-version"))

		// Return mock response
		resp := anthropicResponse{
			ID:         "msg_123",
			Type:       "message",
			Role:       "assistant",
			Model:      "claude-3-sonnet-20240229",
			StopReason: "end_turn",
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{
				{
					Type: "text",
					Text: "Hello! I'm Claude, an AI assistant.",
				},
			},
			Usage: struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			}{
				InputTokens:  15,
				OutputTokens: 25,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create adapter with test server
	adapter := &AnthropicAdapter{
		id:         "anthropic",
		name:       "Anthropic Claude",
		httpClient: server.Client(),
		apiKey:     "test-api-key",
		baseURL:    server.URL,
	}

	// Create request
	config, _ := json.Marshal(AnthropicConfig{
		Model:     "claude-3-sonnet-20240229",
		Prompt:    "Hello {{name}}!",
		System:    "You are a helpful assistant.",
		MaxTokens: 1024,
	})

	input, _ := json.Marshal(map[string]string{
		"name": "World",
	})

	req := &Request{
		Input:  input,
		Config: config,
	}

	// Execute
	ctx := context.Background()
	resp, err := adapter.Execute(ctx, req)

	// Verify
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.GreaterOrEqual(t, resp.DurationMs, 0) // May be 0 in fast test environments
	assert.Equal(t, "anthropic", resp.Metadata["adapter"])
	assert.Equal(t, "claude-3-sonnet-20240229", resp.Metadata["model"])

	// Verify output
	var output map[string]interface{}
	err = json.Unmarshal(resp.Output, &output)
	require.NoError(t, err)
	assert.Equal(t, "Hello! I'm Claude, an AI assistant.", output["content"])
	assert.Equal(t, "end_turn", output["stop_reason"])
}

func TestAnthropicAdapter_Execute_NoAPIKey(t *testing.T) {
	adapter := &AnthropicAdapter{
		id:     "anthropic",
		name:   "Anthropic Claude",
		apiKey: "",
	}

	req := &Request{}
	ctx := context.Background()
	_, err := adapter.Execute(ctx, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API key not configured")
}

func TestAnthropicAdapter_Execute_APIError(t *testing.T) {
	// Create mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := anthropicResponse{
			Error: &anthropicError{
				Type:    "authentication_error",
				Message: "Invalid API key",
			},
		}
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	adapter := &AnthropicAdapter{
		id:         "anthropic",
		name:       "Anthropic Claude",
		httpClient: server.Client(),
		apiKey:     "invalid-key",
		baseURL:    server.URL,
	}

	config, _ := json.Marshal(AnthropicConfig{
		Model:     "claude-3-sonnet-20240229",
		Prompt:    "Hello",
		MaxTokens: 1024,
	})

	req := &Request{
		Config: config,
	}

	ctx := context.Background()
	_, err := adapter.Execute(ctx, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid API key")
}

func TestAnthropicAdapter_Metadata(t *testing.T) {
	adapter := NewAnthropicAdapter()

	assert.Equal(t, "anthropic", adapter.ID())
	assert.Equal(t, "Anthropic Claude", adapter.Name())
	assert.NotEmpty(t, adapter.InputSchema())
	assert.NotEmpty(t, adapter.OutputSchema())
}

func TestAnthropicAdapter_MultipleContentBlocks(t *testing.T) {
	// Create mock server with multiple content blocks
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := anthropicResponse{
			ID:         "msg_456",
			Type:       "message",
			Role:       "assistant",
			Model:      "claude-3-opus-20240229",
			StopReason: "end_turn",
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{
				{Type: "text", Text: "First part. "},
				{Type: "text", Text: "Second part."},
			},
			Usage: struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			}{
				InputTokens:  10,
				OutputTokens: 15,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	adapter := &AnthropicAdapter{
		id:         "anthropic",
		name:       "Anthropic Claude",
		httpClient: server.Client(),
		apiKey:     "test-key",
		baseURL:    server.URL,
	}

	config, _ := json.Marshal(AnthropicConfig{
		Model:     "claude-3-opus-20240229",
		Prompt:    "Test",
		MaxTokens: 100,
	})

	req := &Request{Config: config}
	ctx := context.Background()
	resp, err := adapter.Execute(ctx, req)

	require.NoError(t, err)

	var output map[string]interface{}
	json.Unmarshal(resp.Output, &output)
	assert.Equal(t, "First part. Second part.", output["content"])
}

func TestAnthropicAdapter_Execute_TemperatureZero(t *testing.T) {
	// Create mock server that verifies temperature is 0
	var receivedTemperature float64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request body to verify temperature
		var reqBody anthropicRequest
		json.NewDecoder(r.Body).Decode(&reqBody)
		receivedTemperature = reqBody.Temperature

		resp := anthropicResponse{
			ID:         "msg_123",
			Type:       "message",
			Role:       "assistant",
			Model:      "claude-3-sonnet-20240229",
			StopReason: "end_turn",
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{
				{Type: "text", Text: "Response with temperature 0"},
			},
			Usage: struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			}{
				InputTokens:  10,
				OutputTokens: 20,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	adapter := &AnthropicAdapter{
		id:         "anthropic",
		name:       "Anthropic Claude",
		httpClient: server.Client(),
		apiKey:     "test-api-key",
		baseURL:    server.URL,
	}

	// Create request with temperature = 0 (explicitly set)
	tempZero := 0.0
	config, _ := json.Marshal(AnthropicConfig{
		Model:       "claude-3-sonnet-20240229",
		Prompt:      "Test prompt",
		MaxTokens:   1024,
		Temperature: &tempZero,
	})

	req := &Request{
		Config: config,
	}

	ctx := context.Background()
	resp, err := adapter.Execute(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	// Verify that temperature 0 was sent to the API (not replaced with default 0.7)
	assert.Equal(t, 0.0, receivedTemperature, "temperature=0 should be sent to API, not replaced with default")
}

func TestAnthropicAdapter_Execute_TemperatureDefault(t *testing.T) {
	// Create mock server that verifies default temperature
	var receivedTemperature float64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody anthropicRequest
		json.NewDecoder(r.Body).Decode(&reqBody)
		receivedTemperature = reqBody.Temperature

		resp := anthropicResponse{
			ID:         "msg_123",
			Type:       "message",
			Role:       "assistant",
			Model:      "claude-3-sonnet-20240229",
			StopReason: "end_turn",
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{
				{Type: "text", Text: "Response with default temperature"},
			},
			Usage: struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			}{
				InputTokens:  10,
				OutputTokens: 20,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	adapter := &AnthropicAdapter{
		id:         "anthropic",
		name:       "Anthropic Claude",
		httpClient: server.Client(),
		apiKey:     "test-api-key",
		baseURL:    server.URL,
	}

	// Create request without temperature (should use default)
	config, _ := json.Marshal(AnthropicConfig{
		Model:     "claude-3-sonnet-20240229",
		Prompt:    "Test prompt",
		MaxTokens: 1024,
		// Temperature not set - should use default 0.7
	})

	req := &Request{
		Config: config,
	}

	ctx := context.Background()
	resp, err := adapter.Execute(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	// Verify that default temperature 0.7 was used
	assert.Equal(t, 0.7, receivedTemperature, "default temperature should be 0.7 when not specified")
}
