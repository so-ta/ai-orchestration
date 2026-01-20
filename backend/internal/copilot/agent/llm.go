// Package agent provides the agent loop and LLM integration for the Copilot.
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/souta/ai-orchestration/internal/copilot/tools"
)

// LLMClient handles communication with the Anthropic API
type LLMClient struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewLLMClient creates a new LLM client
func NewLLMClient() *LLMClient {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	baseURL := os.Getenv("ANTHROPIC_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	model := os.Getenv("ANTHROPIC_MODEL")
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	return &LLMClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// Message represents a conversation message
type Message struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// TextContent represents text content in a message
type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ToolUseContent represents a tool use request from the assistant
type ToolUseContent struct {
	Type  string          `json:"type"`
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// ToolResultContent represents a tool result from the user
type ToolResultContent struct {
	Type      string `json:"type"`
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error,omitempty"`
}

// ChatRequest represents a request to the chat API
type ChatRequest struct {
	Model       string                 `json:"model"`
	System      string                 `json:"system,omitempty"`
	Messages    []Message              `json:"messages"`
	Tools       []tools.ToolDefinition `json:"tools,omitempty"`
	MaxTokens   int                    `json:"max_tokens"`
	Temperature float64                `json:"temperature,omitempty"`
}

// ChatResponse represents a response from the chat API
type ChatResponse struct {
	ID           string        `json:"id"`
	Type         string        `json:"type"`
	Role         string        `json:"role"`
	Content      []ContentItem `json:"content"`
	Model        string        `json:"model"`
	StopReason   string        `json:"stop_reason"`
	StopSequence *string       `json:"stop_sequence,omitempty"`
	Usage        Usage         `json:"usage"`
}

// ContentItem represents an item in the content array
type ContentItem struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

// Usage represents token usage
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// ChatWithTools sends a chat request with tool definitions
func (c *LLMClient) ChatWithTools(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if c.apiKey == "" {
		slog.Info("LLM client: no API key, using mock")
		return c.mockChatWithTools(req)
	}

	if req.Model == "" {
		req.Model = c.model
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = 4096
	}

	slog.Info("LLM client: starting API call", "model", req.Model, "tool_count", len(req.Tools))

	reqBody, err := json.Marshal(req)
	if err != nil {
		slog.Error("LLM client: failed to marshal request", "error", err)
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/messages", bytes.NewReader(reqBody))
	if err != nil {
		slog.Error("LLM client: failed to create request", "error", err)
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	slog.Info("LLM client: sending request to Anthropic API")
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		slog.Error("LLM client: request failed", "error", err)
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	slog.Info("LLM client: received response", "status", resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("LLM client: failed to read response body", "error", err)
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		slog.Error("LLM client: API error", "status", resp.StatusCode, "body", string(body))
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		slog.Error("LLM client: failed to parse response", "error", err)
		return nil, fmt.Errorf("parse response: %w", err)
	}

	slog.Info("LLM client: API call successful", "stop_reason", chatResp.StopReason)
	return &chatResp, nil
}

// GetToolCalls extracts tool calls from a response
func (r *ChatResponse) GetToolCalls() []tools.ToolCall {
	var toolCalls []tools.ToolCall
	for _, item := range r.Content {
		if item.Type == "tool_use" {
			toolCalls = append(toolCalls, tools.ToolCall{
				ID:    item.ID,
				Name:  item.Name,
				Input: item.Input,
			})
		}
	}
	return toolCalls
}

// GetTextContent extracts text content from a response
func (r *ChatResponse) GetTextContent() string {
	var text string
	for _, item := range r.Content {
		if item.Type == "text" {
			text += item.Text
		}
	}
	return text
}

// HasToolUse checks if the response contains tool use requests
func (r *ChatResponse) HasToolUse() bool {
	return r.StopReason == "tool_use"
}

// IsEndTurn checks if this is the final response
func (r *ChatResponse) IsEndTurn() bool {
	return r.StopReason == "end_turn"
}

// CreateTextMessage creates a message with text content
func CreateTextMessage(role string, text string) Message {
	content := []TextContent{{Type: "text", Text: text}}
	contentJSON, _ := json.Marshal(content)
	return Message{
		Role:    role,
		Content: contentJSON,
	}
}

// CreateToolResultMessage creates a message with tool results
func CreateToolResultMessage(results []tools.ToolResult) Message {
	content := make([]ToolResultContent, 0, len(results))
	for _, r := range results {
		content = append(content, ToolResultContent{
			Type:      "tool_result",
			ToolUseID: r.ToolUseID,
			Content:   string(r.Content),
			IsError:   r.IsError,
		})
	}
	contentJSON, _ := json.Marshal(content)
	return Message{
		Role:    "user",
		Content: contentJSON,
	}
}

// mockChatWithTools provides mock responses when API key is not configured
func (c *LLMClient) mockChatWithTools(req ChatRequest) (*ChatResponse, error) {
	// Extract the last user message
	var lastUserMessage string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		msg := req.Messages[i]
		if msg.Role == "user" {
			var content []TextContent
			if err := json.Unmarshal(msg.Content, &content); err == nil && len(content) > 0 {
				lastUserMessage = content[0].Text
				break
			}
		}
	}

	// Generate mock response with setup instructions
	var responseText string
	if len(req.Tools) > 0 {
		responseText = fmt.Sprintf(
			"⚠️ **APIキー未設定**\n\n"+
				"ANTHROPIC_API_KEYが設定されていないため、モックレスポンスを返しています。\n\n"+
				"**設定方法:**\n"+
				"1. プロジェクトルートに `.env` ファイルを作成\n"+
				"2. 以下の環境変数を設定:\n"+
				"   ```\n"+
				"   ANTHROPIC_API_KEY=your-api-key-here\n"+
				"   ```\n"+
				"3. APIサーバーを再起動\n\n"+
				"---\n"+
				"**受信したメッセージ:** %s\n"+
				"**利用可能なツール:** %d個\n\n"+
				"本番環境では、エージェントがツールを使用してワークフローを分析・構築します。",
			lastUserMessage, len(req.Tools))
	} else {
		responseText = fmt.Sprintf(
			"⚠️ **APIキー未設定**\n\n"+
				"ANTHROPIC_API_KEYが設定されていないため、モックレスポンスを返しています。\n\n"+
				"**設定方法:**\n"+
				"1. プロジェクトルートに `.env` ファイルを作成\n"+
				"2. 以下の環境変数を設定:\n"+
				"   ```\n"+
				"   ANTHROPIC_API_KEY=your-api-key-here\n"+
				"   ```\n"+
				"3. APIサーバーを再起動\n\n"+
				"---\n"+
				"**受信したメッセージ:** %s",
			lastUserMessage)
	}

	return &ChatResponse{
		ID:   "mock-response",
		Type: "message",
		Role: "assistant",
		Content: []ContentItem{
			{Type: "text", Text: responseText},
		},
		Model:      "mock",
		StopReason: "end_turn",
		Usage: Usage{
			InputTokens:  0,
			OutputTokens: 0,
		},
	}, nil
}
