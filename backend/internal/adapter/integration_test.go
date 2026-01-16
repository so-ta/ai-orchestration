// Package adapter provides integration tests for external service adapters.
// These tests require actual API keys and make real API calls.
//
// To run integration tests:
//
//	INTEGRATION_TEST=1 go test ./internal/adapter/... -v -run Integration
//
// Required environment variables (in .env.test.local):
//   - OPENAI_API_KEY: OpenAI API key
//   - ANTHROPIC_API_KEY: Anthropic API key
package adapter

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/souta/ai-orchestration/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// OpenAI Integration Tests
// =============================================================================

func TestOpenAIAdapter_Integration_BasicChat(t *testing.T) {
	testutil.SkipIfNotIntegration(t)
	testutil.LoadTestEnv(t)
	apiKey := testutil.RequireEnvVar(t, "OPENAI_API_KEY")

	adapter := NewOpenAIAdapterWithKey(apiKey)

	config, _ := json.Marshal(OpenAIConfig{
		Model:     "gpt-4o-mini",
		Prompt:    "Say 'Hello, Integration Test!' and nothing else.",
		MaxTokens: 50,
	})

	req := &Request{
		Config: config,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := adapter.Execute(ctx, req)

	require.NoError(t, err, "OpenAI API call should succeed")
	require.NotNil(t, resp)

	var output map[string]interface{}
	err = json.Unmarshal(resp.Output, &output)
	require.NoError(t, err)

	content := output["content"].(string)
	assert.NotEmpty(t, content, "Response should have content")
	assert.Contains(t, strings.ToLower(content), "hello", "Response should contain greeting")

	t.Logf("OpenAI Response: %s", content)
	t.Logf("Duration: %dms", resp.DurationMs)
	t.Logf("Model: %s", resp.Metadata["model"])
	t.Logf("Tokens: %s", resp.Metadata["total_tokens"])
}

func TestOpenAIAdapter_Integration_WithVariables(t *testing.T) {
	testutil.SkipIfNotIntegration(t)
	testutil.LoadTestEnv(t)
	apiKey := testutil.RequireEnvVar(t, "OPENAI_API_KEY")

	adapter := NewOpenAIAdapterWithKey(apiKey)

	config, _ := json.Marshal(OpenAIConfig{
		Model:     "gpt-4o-mini",
		Prompt:    "What is {{num1}} + {{num2}}? Answer with just the number.",
		MaxTokens: 10,
	})

	input, _ := json.Marshal(map[string]interface{}{
		"num1": 5,
		"num2": 3,
	})

	req := &Request{
		Config: config,
		Input:  input,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := adapter.Execute(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)

	var output map[string]interface{}
	err = json.Unmarshal(resp.Output, &output)
	require.NoError(t, err)
	content := output["content"].(string)

	assert.Contains(t, content, "8", "Response should contain the correct answer")
	t.Logf("OpenAI Response: %s", content)
}

func TestOpenAIAdapter_Integration_SystemPrompt(t *testing.T) {
	testutil.SkipIfNotIntegration(t)
	testutil.LoadTestEnv(t)
	apiKey := testutil.RequireEnvVar(t, "OPENAI_API_KEY")

	adapter := NewOpenAIAdapterWithKey(apiKey)

	config, _ := json.Marshal(OpenAIConfig{
		Model:     "gpt-4o-mini",
		System:    "You are a pirate. Always respond in pirate speak.",
		Prompt:    "Say hello.",
		MaxTokens: 50,
	})

	req := &Request{
		Config: config,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := adapter.Execute(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)

	var output map[string]interface{}
	err = json.Unmarshal(resp.Output, &output)
	require.NoError(t, err)
	content := strings.ToLower(output["content"].(string))

	// Pirate speak typically contains these words
	hasPirateWord := strings.Contains(content, "ahoy") ||
		strings.Contains(content, "matey") ||
		strings.Contains(content, "arr") ||
		strings.Contains(content, "ye")

	assert.True(t, hasPirateWord, "Response should be in pirate speak: %s", content)
	t.Logf("OpenAI Response: %s", output["content"])
}

// =============================================================================
// Anthropic Integration Tests
// =============================================================================

func TestAnthropicAdapter_Integration_BasicChat(t *testing.T) {
	testutil.SkipIfNotIntegration(t)
	testutil.LoadTestEnv(t)
	apiKey := testutil.RequireEnvVar(t, "ANTHROPIC_API_KEY")

	adapter := NewAnthropicAdapterWithKey(apiKey)

	config, _ := json.Marshal(AnthropicConfig{
		Model:     "claude-3-haiku-20240307",
		Prompt:    "Say 'Hello, Integration Test!' and nothing else.",
		MaxTokens: 50,
	})

	req := &Request{
		Config: config,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := adapter.Execute(ctx, req)

	require.NoError(t, err, "Anthropic API call should succeed")
	require.NotNil(t, resp)

	var output map[string]interface{}
	err = json.Unmarshal(resp.Output, &output)
	require.NoError(t, err)

	content := output["content"].(string)
	assert.NotEmpty(t, content, "Response should have content")
	assert.Contains(t, strings.ToLower(content), "hello", "Response should contain greeting")

	t.Logf("Anthropic Response: %s", content)
	t.Logf("Duration: %dms", resp.DurationMs)
	t.Logf("Model: %s", resp.Metadata["model"])
	t.Logf("Input tokens: %s, Output tokens: %s",
		resp.Metadata["input_tokens"], resp.Metadata["output_tokens"])
}

func TestAnthropicAdapter_Integration_WithVariables(t *testing.T) {
	testutil.SkipIfNotIntegration(t)
	testutil.LoadTestEnv(t)
	apiKey := testutil.RequireEnvVar(t, "ANTHROPIC_API_KEY")

	adapter := NewAnthropicAdapterWithKey(apiKey)

	config, _ := json.Marshal(AnthropicConfig{
		Model:     "claude-3-haiku-20240307",
		Prompt:    "What is {{num1}} + {{num2}}? Answer with just the number.",
		MaxTokens: 10,
	})

	input, _ := json.Marshal(map[string]interface{}{
		"num1": 7,
		"num2": 4,
	})

	req := &Request{
		Config: config,
		Input:  input,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := adapter.Execute(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)

	var output map[string]interface{}
	err = json.Unmarshal(resp.Output, &output)
	require.NoError(t, err)
	content := output["content"].(string)

	assert.Contains(t, content, "11", "Response should contain the correct answer")
	t.Logf("Anthropic Response: %s", content)
}

func TestAnthropicAdapter_Integration_SystemPrompt(t *testing.T) {
	testutil.SkipIfNotIntegration(t)
	testutil.LoadTestEnv(t)
	apiKey := testutil.RequireEnvVar(t, "ANTHROPIC_API_KEY")

	adapter := NewAnthropicAdapterWithKey(apiKey)

	config, _ := json.Marshal(AnthropicConfig{
		Model:     "claude-3-haiku-20240307",
		System:    "You always respond in exactly 3 words, no more, no less.",
		Prompt:    "Describe the color blue.",
		MaxTokens: 20,
	})

	req := &Request{
		Config: config,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := adapter.Execute(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)

	var output map[string]interface{}
	err = json.Unmarshal(resp.Output, &output)
	require.NoError(t, err)
	content := output["content"].(string)

	// Count words (approximately)
	words := strings.Fields(content)
	assert.LessOrEqual(t, len(words), 5, "Response should be concise: %s", content)
	t.Logf("Anthropic Response: %s (%d words)", content, len(words))
}

// =============================================================================
// HTTP Adapter Integration Tests
// =============================================================================

func TestHTTPAdapter_Integration_PublicAPI(t *testing.T) {
	testutil.SkipIfNotIntegration(t)

	adapter := NewHTTPAdapter()

	// Using httpbin.org as a public test API
	config := HTTPConfig{
		URL:        "https://httpbin.org/get",
		Method:     "GET",
		TimeoutSec: 10,
		QueryParams: map[string]string{
			"test": "integration",
		},
	}
	configJSON, _ := json.Marshal(config)

	req := &Request{
		Input:  json.RawMessage(`{}`),
		Config: configJSON,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp, err := adapter.Execute(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "200", resp.Metadata["status_code"])

	var output HTTPOutput
	err = json.Unmarshal(resp.Output, &output)
	require.NoError(t, err)
	assert.Equal(t, 200, output.StatusCode)

	// Verify the response body contains our query parameter
	if body, ok := output.Body.(map[string]interface{}); ok {
		if args, ok := body["args"].(map[string]interface{}); ok {
			assert.Equal(t, "integration", args["test"])
		}
	}

	t.Logf("HTTP Response status: %d", output.StatusCode)
	t.Logf("Duration: %dms", resp.DurationMs)
}

func TestHTTPAdapter_Integration_POST(t *testing.T) {
	testutil.SkipIfNotIntegration(t)

	adapter := NewHTTPAdapter()

	config := HTTPConfig{
		URL:        "https://httpbin.org/post",
		Method:     "POST",
		TimeoutSec: 10,
		Headers: map[string]string{
			"X-Custom-Header": "test-value",
		},
		Body:     `{"message": "{{msg}}"}`,
		BodyType: "json",
	}
	configJSON, _ := json.Marshal(config)

	req := &Request{
		Input:  json.RawMessage(`{"msg": "Hello from integration test"}`),
		Config: configJSON,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp, err := adapter.Execute(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "200", resp.Metadata["status_code"])

	var output HTTPOutput
	err = json.Unmarshal(resp.Output, &output)
	require.NoError(t, err)

	// Verify the response echoed our data
	if body, ok := output.Body.(map[string]interface{}); ok {
		// httpbin.org echoes the JSON body in "json" field
		if jsonData, ok := body["json"].(map[string]interface{}); ok {
			assert.Equal(t, "Hello from integration test", jsonData["message"])
		}
		// Check custom header was received
		if headers, ok := body["headers"].(map[string]interface{}); ok {
			assert.Equal(t, "test-value", headers["X-Custom-Header"])
		}
	}

	t.Logf("HTTP POST successful, duration: %dms", resp.DurationMs)
}

func TestHTTPAdapter_Integration_VariableSubstitution(t *testing.T) {
	testutil.SkipIfNotIntegration(t)

	adapter := NewHTTPAdapter()

	config := HTTPConfig{
		URL:        "https://httpbin.org/anything/{{resource}}/{{id}}",
		Method:     "GET",
		TimeoutSec: 10,
	}
	configJSON, _ := json.Marshal(config)

	req := &Request{
		Input:  json.RawMessage(`{"resource": "users", "id": "12345"}`),
		Config: configJSON,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp, err := adapter.Execute(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)

	var output HTTPOutput
	err = json.Unmarshal(resp.Output, &output)
	require.NoError(t, err)

	// httpbin.org echoes the URL in the response
	if body, ok := output.Body.(map[string]interface{}); ok {
		url := body["url"].(string)
		assert.Contains(t, url, "/users/12345")
	}

	t.Logf("Variable substitution successful")
}

// =============================================================================
// Cross-Adapter Integration Tests
// =============================================================================

func TestAdapters_Integration_AllAvailable(t *testing.T) {
	testutil.SkipIfNotIntegration(t)
	testutil.LoadTestEnv(t)

	tests := []struct {
		name      string
		envKey    string
		available bool
	}{
		{"OpenAI", "OPENAI_API_KEY", false},
		{"Anthropic", "ANTHROPIC_API_KEY", false},
		{"HTTP", "", true}, // HTTP doesn't need an API key
	}

	for i := range tests {
		if tests[i].envKey == "" || os.Getenv(tests[i].envKey) != "" {
			tests[i].available = true
		}
	}

	t.Log("=== Integration Test Environment ===")
	for _, tt := range tests {
		status := "✗ NOT CONFIGURED"
		if tt.available {
			status = "✓ Available"
		}
		t.Logf("  %s: %s", tt.name, status)
	}
}
