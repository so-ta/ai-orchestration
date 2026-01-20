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

// ToolCall represents a tool call from LLM
type ToolCall struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"`
	Function ToolCallFunction  `json:"function"`
}

// ToolCallFunction represents the function part of a tool call.
// Note: Arguments is a JSON string as per OpenAI API specification.
// The API returns function arguments as a stringified JSON object, not a parsed object.
// See: https://platform.openai.com/docs/api-reference/chat/object#chat/object-choices
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string, e.g. "{\"location\": \"Boston\"}"
}

// chatOpenAI calls OpenAI's chat completion API
func (s *LLMServiceImpl) chatOpenAI(model string, request map[string]interface{}) (map[string]interface{}, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return s.mockChat("openai", model, request)
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

	// Copy tool parameters for OpenAI Function Calling API (tools API, not legacy functions API)
	// See: https://platform.openai.com/docs/guides/function-calling
	// Note: OpenAI has two APIs:
	// - Legacy: "functions" and "function_call" (deprecated since Nov 2023)
	// - Current: "tools" and "tool_choice" (recommended, used here)
	// The "tools" parameter is an array of tool definitions with type: "function"
	// The "tool_choice" parameter controls how the model selects tools
	if tools, ok := request["tools"]; ok {
		openaiReq["tools"] = tools
	}
	if toolChoice, ok := request["tool_choice"]; ok {
		openaiReq["tool_choice"] = toolChoice
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

	// Parse response with tool calls support
	// Response struct includes: ID, Choices (with Message, FinishReason), and Usage (with token counts)
	var respData struct {
		ID      string `json:"id"`
		Choices []struct {
			Message struct {
				Role       string     `json:"role"`
				Content    string     `json:"content"`
				ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
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

	// Extract content, finish_reason and tool calls
	content := ""
	finishReason := ""
	var toolCalls []map[string]interface{}
	if len(respData.Choices) > 0 {
		content = respData.Choices[0].Message.Content
		finishReason = respData.Choices[0].FinishReason
		// Convert tool calls to generic map format
		for _, tc := range respData.Choices[0].Message.ToolCalls {
			toolCalls = append(toolCalls, map[string]interface{}{
				"id":   tc.ID,
				"type": tc.Type,
				"function": map[string]interface{}{
					"name":      tc.Function.Name,
					"arguments": tc.Function.Arguments,
				},
			})
		}
	}

	result := map[string]interface{}{
		"content":       content,
		"finish_reason": finishReason,
		"usage": map[string]interface{}{
			"input_tokens":  respData.Usage.PromptTokens,
			"output_tokens": respData.Usage.CompletionTokens,
			"total_tokens":  respData.Usage.TotalTokens,
		},
	}

	// Add tool_calls only if present
	if len(toolCalls) > 0 {
		result["tool_calls"] = toolCalls
	}

	return result, nil
}

// chatAnthropic calls Anthropic's messages API
func (s *LLMServiceImpl) chatAnthropic(model string, request map[string]interface{}) (map[string]interface{}, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return s.mockChat("anthropic", model, request)
	}

	// Default model: Claude Sonnet 4 (current production model as of 2025-05)
	// See: https://docs.anthropic.com/en/docs/about-claude/models
	if model == "" {
		model = "claude-sonnet-4-20250514"
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

			if role == "system" {
				if content, ok := msg["content"].(string); ok {
					systemMsg = content
				}
			} else if role == "tool" {
				// Convert tool result to Anthropic format
				toolCallID, _ := msg["tool_call_id"].(string)
				content, _ := msg["content"].(string)
				anthropicMsgs = append(anthropicMsgs, map[string]interface{}{
					"role": "user",
					"content": []map[string]interface{}{
						{
							"type":        "tool_result",
							"tool_use_id": toolCallID,
							"content":     content,
						},
					},
				})
			} else {
				// Handle both string content and content array (for tool_calls)
				if content, ok := msg["content"].(string); ok {
					anthropicMsgs = append(anthropicMsgs, map[string]interface{}{
						"role":    role,
						"content": content,
					})
				} else if contentArr, ok := msg["content"].([]interface{}); ok {
					anthropicMsgs = append(anthropicMsgs, map[string]interface{}{
						"role":    role,
						"content": contentArr,
					})
				}
				// Handle assistant messages with tool_calls
				if role == "assistant" {
					if toolCalls, ok := msg["tool_calls"].([]interface{}); ok && len(toolCalls) > 0 {
						// Build content array with text and tool_use blocks
						var contentBlocks []map[string]interface{}
						if textContent, ok := msg["content"].(string); ok && textContent != "" {
							contentBlocks = append(contentBlocks, map[string]interface{}{
								"type": "text",
								"text": textContent,
							})
						}
						for _, tc := range toolCalls {
							if tcMap, ok := tc.(map[string]interface{}); ok {
								fn, _ := tcMap["function"].(map[string]interface{})
								name, _ := fn["name"].(string)
								argsStr, _ := fn["arguments"].(string)
								var argsMap map[string]interface{}
								if err := json.Unmarshal([]byte(argsStr), &argsMap); err != nil {
									// If JSON parsing fails, use empty map to avoid nil input
									argsMap = make(map[string]interface{})
								}
								contentBlocks = append(contentBlocks, map[string]interface{}{
									"type":  "tool_use",
									"id":    tcMap["id"],
									"name":  name,
									"input": argsMap,
								})
							}
						}
						// Replace the last message with proper content blocks
						if len(anthropicMsgs) > 0 {
							anthropicMsgs[len(anthropicMsgs)-1] = map[string]interface{}{
								"role":    "assistant",
								"content": contentBlocks,
							}
						}
					}
				}
			}
		}

		// In Anthropic Messages API, system prompt is a top-level parameter, NOT a message role
		// See: https://docs.anthropic.com/en/api/messages
		// The "system" parameter is separate from "messages" array
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

	// Convert OpenAI-style tools to Anthropic format
	if tools, ok := request["tools"].([]interface{}); ok && len(tools) > 0 {
		var anthropicTools []map[string]interface{}
		for _, t := range tools {
			if tool, ok := t.(map[string]interface{}); ok {
				if fn, ok := tool["function"].(map[string]interface{}); ok {
					anthropicTools = append(anthropicTools, map[string]interface{}{
						"name":         fn["name"],
						"description":  fn["description"],
						"input_schema": fn["parameters"],
					})
				}
			}
		}
		if len(anthropicTools) > 0 {
			anthropicReq["tools"] = anthropicTools
		}
	}

	// Convert tool_choice
	if toolChoice, ok := request["tool_choice"].(string); ok {
		switch toolChoice {
		case "none":
			// Don't include tool_choice
		case "required":
			anthropicReq["tool_choice"] = map[string]interface{}{"type": "any"}
		case "auto":
			anthropicReq["tool_choice"] = map[string]interface{}{"type": "auto"}
		}
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

	// Parse response with tool_use support
	var respData struct {
		Content []struct {
			Type  string                 `json:"type"`
			Text  string                 `json:"text,omitempty"`
			ID    string                 `json:"id,omitempty"`
			Name  string                 `json:"name,omitempty"`
			Input map[string]interface{} `json:"input,omitempty"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
		Usage      struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &respData); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract content and tool calls
	content := ""
	var toolCalls []map[string]interface{}
	for _, c := range respData.Content {
		if c.Type == "text" {
			content += c.Text
		} else if c.Type == "tool_use" {
			// Convert Anthropic tool_use to OpenAI-style tool_calls format
			argsBytes, _ := json.Marshal(c.Input)
			toolCalls = append(toolCalls, map[string]interface{}{
				"id":   c.ID,
				"type": "function",
				"function": map[string]interface{}{
					"name":      c.Name,
					"arguments": string(argsBytes),
				},
			})
		}
	}

	result := map[string]interface{}{
		"content":       content,
		"finish_reason": respData.StopReason, // Use unified key name across providers
		"usage": map[string]interface{}{
			"input_tokens":  respData.Usage.InputTokens,
			"output_tokens": respData.Usage.OutputTokens,
			"total_tokens":  respData.Usage.InputTokens + respData.Usage.OutputTokens,
		},
	}

	// Add tool_calls only if present
	if len(toolCalls) > 0 {
		result["tool_calls"] = toolCalls
	}

	return result, nil
}

// mockChat provides mock responses when LLM API keys are not configured
// This allows testing and development without actual API credentials
func (s *LLMServiceImpl) mockChat(provider, model string, request map[string]interface{}) (map[string]interface{}, error) {
	// Extract the last user message for context
	var lastUserMessage string
	if messages, ok := request["messages"].([]interface{}); ok {
		for i := len(messages) - 1; i >= 0; i-- {
			if msg, ok := messages[i].(map[string]interface{}); ok {
				if role, _ := msg["role"].(string); role == "user" {
					if content, ok := msg["content"].(string); ok {
						lastUserMessage = content
						break
					}
				}
			}
		}
	}

	// Check if tools are available
	var toolCount int
	if tools, ok := request["tools"].([]interface{}); ok {
		toolCount = len(tools)
	}

	// Determine the environment variable name
	envVarName := "OPENAI_API_KEY"
	if provider == "anthropic" {
		envVarName = "ANTHROPIC_API_KEY"
	}

	// Generate mock response with clear setup instructions
	responseText := fmt.Sprintf(
		"⚠️ **APIキー未設定**\n\n"+
			"LLMプロバイダー「%s」のAPIキーが設定されていないため、モックレスポンスを返しています。\n\n"+
			"**設定方法:**\n"+
			"1. プロジェクトルートに `.env` ファイルを作成\n"+
			"2. 以下の環境変数を設定:\n"+
			"   ```\n"+
			"   %s=your-api-key-here\n"+
			"   ```\n"+
			"3. APIサーバーを再起動\n\n"+
			"---\n"+
			"**受信したメッセージ:** %s\n"+
			"**利用可能なツール:** %d個\n"+
			"**使用モデル:** %s",
		provider, envVarName, lastUserMessage, toolCount, model)

	return map[string]interface{}{
		"content":       responseText,
		"finish_reason": "end_turn",
		"usage": map[string]interface{}{
			"input_tokens":  0,
			"output_tokens": 0,
			"total_tokens":  0,
		},
	}, nil
}
