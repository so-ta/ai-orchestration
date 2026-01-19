package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/copilot/tools"
	"github.com/souta/ai-orchestration/internal/domain"
)

// Config holds the agent loop configuration
type Config struct {
	MaxIterations      int
	MaxRetries         int
	RetryOnParseError  bool
	RetryOnValidation  bool
	StreamEvents       bool
	Temperature        float64
}

// DefaultConfig returns the default agent configuration
func DefaultConfig() Config {
	return Config{
		MaxIterations:      20,
		MaxRetries:         3,
		RetryOnParseError:  true,
		RetryOnValidation:  true,
		StreamEvents:       true,
		Temperature:        0.3,
	}
}

// AgentLoop manages the multi-step reasoning loop
type AgentLoop struct {
	llmClient    *LLMClient
	toolRegistry *tools.Registry
	config       Config
	logger       *slog.Logger
}

// NewAgentLoop creates a new agent loop
func NewAgentLoop(llmClient *LLMClient, toolRegistry *tools.Registry, config Config) *AgentLoop {
	return &AgentLoop{
		llmClient:    llmClient,
		toolRegistry: toolRegistry,
		config:       config,
		logger:       slog.Default(),
	}
}

// Event represents an event during agent execution
type Event struct {
	Type      EventType       `json:"type"`
	Timestamp time.Time       `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

// EventType represents the type of agent event
type EventType string

const (
	EventTypeThinking     EventType = "thinking"
	EventTypeToolCall     EventType = "tool_call"
	EventTypeToolResult   EventType = "tool_result"
	EventTypePartialText  EventType = "partial_text"
	EventTypeComplete     EventType = "complete"
	EventTypeError        EventType = "error"
)

// Result represents the final result of agent execution
type Result struct {
	Response      string            `json:"response"`
	ToolsUsed     []string          `json:"tools_used"`
	Iterations    int               `json:"iterations"`
	TotalTokens   int               `json:"total_tokens"`
	WorkflowSpec  *domain.WorkflowSpec `json:"workflow_spec,omitempty"`
}

// RunInput represents input for running the agent
type RunInput struct {
	TenantID    uuid.UUID
	UserID      string
	ProjectID   *uuid.UUID
	SessionID   uuid.UUID
	Message     string
	Mode        domain.CopilotSessionMode
	History     []domain.CopilotMessage
}

// Run executes the agent loop
func (a *AgentLoop) Run(ctx context.Context, input RunInput, events chan<- Event) (*Result, error) {
	// Set context values for tools
	ctx = context.WithValue(ctx, tools.TenantIDKey, input.TenantID)
	ctx = context.WithValue(ctx, tools.UserIDKey, input.UserID)
	if input.ProjectID != nil {
		ctx = context.WithValue(ctx, tools.ProjectIDKey, *input.ProjectID)
	}

	// Build system prompt based on mode
	systemPrompt := a.buildSystemPrompt(input.Mode, input.ProjectID)

	// Build initial messages from history
	messages := a.buildMessagesFromHistory(input.History)

	// Add current user message
	messages = append(messages, CreateTextMessage("user", input.Message))

	// Get tool definitions
	toolDefs := a.toolRegistry.GetToolDefinitions()

	var toolsUsed []string
	totalTokens := 0

	for iteration := 0; iteration < a.config.MaxIterations; iteration++ {
		a.logger.Info("agent loop iteration", "iteration", iteration, "message_count", len(messages))

		// Send thinking event
		a.sendEvent(events, EventTypeThinking, map[string]interface{}{
			"iteration": iteration,
			"message":   "推論中...",
		})

		// Call LLM with tools
		resp, err := a.llmClient.ChatWithTools(ctx, ChatRequest{
			System:      systemPrompt,
			Messages:    messages,
			Tools:       toolDefs,
			MaxTokens:   4096,
			Temperature: a.config.Temperature,
		})
		if err != nil {
			a.sendEvent(events, EventTypeError, map[string]interface{}{
				"error": err.Error(),
			})
			return nil, fmt.Errorf("LLM call failed: %w", err)
		}

		totalTokens += resp.Usage.InputTokens + resp.Usage.OutputTokens

		// Check if LLM wants to use tools
		if resp.HasToolUse() {
			toolCalls := resp.GetToolCalls()

			// Send tool call events
			for _, tc := range toolCalls {
				a.sendEvent(events, EventTypeToolCall, map[string]interface{}{
					"tool":  tc.Name,
					"input": tc.Input,
				})
				toolsUsed = append(toolsUsed, tc.Name)
			}

			// Execute tools
			toolResults, err := a.executeToolCalls(ctx, toolCalls, events)
			if err != nil {
				// Self-correction: if tool execution fails, inform the LLM
				toolResults = []tools.ToolResult{{
					ToolUseID: toolCalls[0].ID,
					Content:   json.RawMessage(fmt.Sprintf(`{"error": "%s"}`, err.Error())),
					IsError:   true,
				}}
			}

			// Add assistant message with tool use
			assistantContent, _ := json.Marshal(resp.Content)
			messages = append(messages, Message{
				Role:    "assistant",
				Content: assistantContent,
			})

			// Add tool results as user message
			messages = append(messages, CreateToolResultMessage(toolResults))

			continue
		}

		// No tool use - this is the final response
		responseText := resp.GetTextContent()

		// Send partial text for any text content
		if responseText != "" {
			a.sendEvent(events, EventTypePartialText, map[string]interface{}{
				"text": responseText,
			})
		}

		// Check if we're done
		if resp.IsEndTurn() {
			a.sendEvent(events, EventTypeComplete, map[string]interface{}{
				"response": responseText,
			})

			return &Result{
				Response:    responseText,
				ToolsUsed:   toolsUsed,
				Iterations:  iteration + 1,
				TotalTokens: totalTokens,
			}, nil
		}
	}

	// Max iterations reached
	return &Result{
		Response:    "最大反復回数に達しました。処理を中断します。",
		ToolsUsed:   toolsUsed,
		Iterations:  a.config.MaxIterations,
		TotalTokens: totalTokens,
	}, nil
}

// executeToolCalls executes tool calls and returns results
func (a *AgentLoop) executeToolCalls(ctx context.Context, toolCalls []tools.ToolCall, events chan<- Event) ([]tools.ToolResult, error) {
	results := make([]tools.ToolResult, 0, len(toolCalls))

	for _, tc := range toolCalls {
		a.logger.Info("executing tool", "tool", tc.Name, "input", string(tc.Input))

		result, err := a.toolRegistry.Execute(ctx, tc.Name, tc.Input)
		if err != nil {
			a.logger.Error("tool execution failed", "tool", tc.Name, "error", err)
			errorResult, _ := json.Marshal(map[string]string{
				"error": err.Error(),
			})
			results = append(results, tools.ToolResult{
				ToolUseID: tc.ID,
				Content:   errorResult,
				IsError:   true,
			})

			a.sendEvent(events, EventTypeToolResult, map[string]interface{}{
				"tool":     tc.Name,
				"is_error": true,
				"error":    err.Error(),
			})
			continue
		}

		results = append(results, tools.ToolResult{
			ToolUseID: tc.ID,
			Content:   result,
			IsError:   false,
		})

		a.sendEvent(events, EventTypeToolResult, map[string]interface{}{
			"tool":   tc.Name,
			"result": result,
		})
	}

	return results, nil
}

// buildSystemPrompt builds the system prompt based on mode
func (a *AgentLoop) buildSystemPrompt(mode domain.CopilotSessionMode, projectID *uuid.UUID) string {
	basePrompt := `あなたはワークフロー自動化プラットフォームのAIアシスタント「Copilot」です。

## あなたの役割
ユーザーがワークフローを設計・構築・改善できるようにサポートします。

## 利用可能なツール
以下のツールを使用してコンテキストを収集し、ワークフローを操作できます：

### コンテキスト収集
- list_blocks: 利用可能なブロック一覧を取得
- get_block_schema: ブロックの設定スキーマを取得
- search_blocks: ブロックをセマンティック検索
- list_workflows: ワークフロー一覧を取得
- get_workflow: ワークフローの詳細を取得
- get_workflow_runs: 実行履歴を取得
- search_documentation: ドキュメントを検索（topic: workflow/blocks/integrations/best-practices）

### 分析・診断
- diagnose_workflow: ワークフローを診断し改善提案を行う（focus: all/errors/performance/structure）

### ワークフロー操作
- create_step: ステップを作成
- update_step: ステップを更新
- delete_step: ステップを削除
- create_edge: エッジを作成
- delete_edge: エッジを削除
- validate_workflow: ワークフローを検証

## 動作原則
1. **自律的に情報収集**: 必要な情報はツールを使って自分で取得してください
2. **段階的に推論**: 複雑な問題は小さなステップに分解してください
3. **検証を忘れずに**: ワークフローを変更したら validate_workflow で検証してください
4. **明確に説明**: 行った操作と理由をユーザーに説明してください
5. **エラーに対応**: ツールがエラーを返した場合は、代替アプローチを試してください

## 回答の言語
ユーザーと同じ言語で回答してください。`

	switch mode {
	case domain.CopilotSessionModeCreate:
		return basePrompt + `

## 現在のモード: 新規作成
ユーザーの要件を聞き取り、新しいワークフローを設計・構築してください。

### プロセス
1. ユーザーの要件を理解する
2. 適切なブロックを検索・選択する
3. ワークフロー構造を提案する
4. ユーザーの確認を得る
5. ステップとエッジを作成する
6. ワークフローを検証する`

	case domain.CopilotSessionModeEnhance:
		if projectID != nil {
			return basePrompt + fmt.Sprintf(`

## 現在のモード: 改善
既存のワークフローを分析し、改善提案を行ってください。

### 対象ワークフロー
ID: %s

### プロセス
1. diagnose_workflow で包括的な診断を実行する（構造、エラー、パフォーマンスを分析）
2. 診断結果に基づいて具体的な改善提案を行う
3. 必要に応じて get_workflow や get_workflow_runs で詳細を確認する
4. ユーザーの確認を得てから変更を実行する

### 改善の観点
- 構造: エントリーポイント、孤立ステップ、並列化の機会
- 信頼性: エラーハンドリング、リトライロジック
- パフォーマンス: 並列処理、不要なステップの削除
- コスト: LLMモデルの最適化、呼び出し回数の削減`, projectID.String())
		}
		return basePrompt + `

## 現在のモード: 改善
既存のワークフローを分析し、改善提案を行ってください。

### プロセス
1. ユーザーに対象のワークフローを確認する
2. diagnose_workflow で診断を実行する
3. 具体的な改善提案を行う`

	case domain.CopilotSessionModeExplain:
		return basePrompt + `

## 現在のモード: 説明
プラットフォームの使い方やワークフローについて説明してください。

### プロセス
1. ユーザーの質問を理解する
2. 必要に応じてドキュメントを検索する
3. 必要に応じてブロック情報を取得する
4. わかりやすく説明する`

	default:
		return basePrompt
	}
}

// buildMessagesFromHistory converts domain messages to LLM messages
func (a *AgentLoop) buildMessagesFromHistory(history []domain.CopilotMessage) []Message {
	messages := make([]Message, 0, len(history))

	// Only include recent history (last 20 messages)
	start := 0
	if len(history) > 20 {
		start = len(history) - 20
	}

	for _, msg := range history[start:] {
		if msg.Role == "system" {
			continue // Skip system messages
		}
		messages = append(messages, CreateTextMessage(msg.Role, msg.Content))
	}

	return messages
}

// sendEvent sends an event to the event channel
func (a *AgentLoop) sendEvent(events chan<- Event, eventType EventType, data map[string]interface{}) {
	if events == nil {
		return
	}

	dataJSON, _ := json.Marshal(data)
	select {
	case events <- Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      dataJSON,
	}:
	default:
		// Channel full, skip event
		a.logger.Warn("event channel full, skipping event", "type", eventType)
	}
}
