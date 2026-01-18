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

func TestOpenAIAdapter_Execute(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/chat/completions", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Contains(t, r.Header.Get("Authorization"), "Bearer ")

		// Return mock response
		resp := openAIResponse{
			ID:      "chatcmpl-123",
			Object:  "chat.completion",
			Created: 1677652288,
			Model:   "gpt-4",
			Choices: []struct {
				Index        int           `json:"index"`
				Message      openAIMessage `json:"message"`
				FinishReason string        `json:"finish_reason"`
			}{
				{
					Index: 0,
					Message: openAIMessage{
						Role:    "assistant",
						Content: "Hello! How can I help you today?",
					},
					FinishReason: "stop",
				},
			},
			Usage: struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			}{
				PromptTokens:     10,
				CompletionTokens: 20,
				TotalTokens:      30,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create adapter with test server
	adapter := &OpenAIAdapter{
		id:         "openai",
		name:       "OpenAI",
		httpClient: server.Client(),
		apiKey:     "test-api-key",
		baseURL:    server.URL,
	}

	// Create request
	config, _ := json.Marshal(OpenAIConfig{
		Model:  "gpt-4",
		Prompt: "Hello {{name}}!",
		System: "You are a helpful assistant.",
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
	assert.Equal(t, "openai", resp.Metadata["adapter"])
	assert.Equal(t, "gpt-4", resp.Metadata["model"])

	// Verify output
	var output map[string]interface{}
	err = json.Unmarshal(resp.Output, &output)
	require.NoError(t, err)
	assert.Equal(t, "Hello! How can I help you today?", output["content"])
	assert.Equal(t, "stop", output["finish_reason"])
}

func TestOpenAIAdapter_Execute_NoAPIKey(t *testing.T) {
	adapter := &OpenAIAdapter{
		id:     "openai",
		name:   "OpenAI",
		apiKey: "",
	}

	req := &Request{}
	ctx := context.Background()
	_, err := adapter.Execute(ctx, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API key not configured")
}

func TestOpenAIAdapter_Execute_APIError(t *testing.T) {
	// Create mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openAIResponse{
			Error: &openAIError{
				Message: "Invalid API key",
				Type:    "invalid_request_error",
				Code:    "invalid_api_key",
			},
		}
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	adapter := &OpenAIAdapter{
		id:         "openai",
		name:       "OpenAI",
		httpClient: server.Client(),
		apiKey:     "invalid-key",
		baseURL:    server.URL,
	}

	config, _ := json.Marshal(OpenAIConfig{
		Model:  "gpt-4",
		Prompt: "Hello",
	})

	req := &Request{
		Config: config,
	}

	ctx := context.Background()
	_, err := adapter.Execute(ctx, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid API key")
}

// Note: TestSubstituteVariables was removed as the substituteVariables function
// was moved to the engine package (executor.go) where template expansion is now handled.

func TestOpenAIAdapter_Metadata(t *testing.T) {
	adapter := NewOpenAIAdapter()

	assert.Equal(t, "openai", adapter.ID())
	assert.Equal(t, "OpenAI", adapter.Name())
	assert.NotEmpty(t, adapter.InputSchema())
	assert.NotEmpty(t, adapter.OutputSchema())
}

func TestOpenAIAdapter_Execute_TemperatureZero(t *testing.T) {
	// Create mock server that verifies temperature is 0
	var receivedTemperature float64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request body to verify temperature
		var reqBody openAIRequest
		json.NewDecoder(r.Body).Decode(&reqBody)
		receivedTemperature = reqBody.Temperature

		// Return mock response
		resp := openAIResponse{
			ID:      "chatcmpl-123",
			Object:  "chat.completion",
			Created: 1677652288,
			Model:   "gpt-4",
			Choices: []struct {
				Index        int           `json:"index"`
				Message      openAIMessage `json:"message"`
				FinishReason string        `json:"finish_reason"`
			}{
				{
					Index: 0,
					Message: openAIMessage{
						Role:    "assistant",
						Content: "Response with temperature 0",
					},
					FinishReason: "stop",
				},
			},
			Usage: struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			}{
				PromptTokens:     10,
				CompletionTokens: 20,
				TotalTokens:      30,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	adapter := &OpenAIAdapter{
		id:         "openai",
		name:       "OpenAI",
		httpClient: server.Client(),
		apiKey:     "test-api-key",
		baseURL:    server.URL,
	}

	// Create request with temperature = 0 (explicitly set)
	tempZero := 0.0
	config, _ := json.Marshal(OpenAIConfig{
		Model:       "gpt-4",
		Prompt:      "Test prompt",
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

func TestOpenAIAdapter_Execute_TemperatureDefault(t *testing.T) {
	// Create mock server that verifies default temperature
	var receivedTemperature float64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody openAIRequest
		json.NewDecoder(r.Body).Decode(&reqBody)
		receivedTemperature = reqBody.Temperature

		resp := openAIResponse{
			ID:      "chatcmpl-123",
			Object:  "chat.completion",
			Created: 1677652288,
			Model:   "gpt-4",
			Choices: []struct {
				Index        int           `json:"index"`
				Message      openAIMessage `json:"message"`
				FinishReason string        `json:"finish_reason"`
			}{
				{
					Index: 0,
					Message: openAIMessage{
						Role:    "assistant",
						Content: "Response with default temperature",
					},
					FinishReason: "stop",
				},
			},
			Usage: struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			}{
				PromptTokens:     10,
				CompletionTokens: 20,
				TotalTokens:      30,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	adapter := &OpenAIAdapter{
		id:         "openai",
		name:       "OpenAI",
		httpClient: server.Client(),
		apiKey:     "test-api-key",
		baseURL:    server.URL,
	}

	// Create request without temperature (should use default)
	config, _ := json.Marshal(OpenAIConfig{
		Model:  "gpt-4",
		Prompt: "Test prompt",
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
