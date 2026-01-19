package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AgentMemoryRole represents the role of a message in agent memory
type AgentMemoryRole string

const (
	AgentMemoryRoleUser      AgentMemoryRole = "user"
	AgentMemoryRoleAssistant AgentMemoryRole = "assistant"
	AgentMemoryRoleSystem    AgentMemoryRole = "system"
	AgentMemoryRoleTool      AgentMemoryRole = "tool"
)

// ToolCall represents a tool call made by the agent
type ToolCall struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"` // "function"
	Function ToolCallFunc    `json:"function"`
}

// ToolCallFunc represents the function details of a tool call
type ToolCallFunc struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// AgentMemory represents a single message in the agent's conversation history
type AgentMemory struct {
	ID             uuid.UUID       `json:"id"`
	TenantID       uuid.UUID       `json:"tenant_id"`
	RunID          uuid.UUID       `json:"run_id"`
	StepID         uuid.UUID       `json:"step_id"`
	Role           AgentMemoryRole `json:"role"`
	Content        string          `json:"content"`
	ToolCalls      []ToolCall      `json:"tool_calls,omitempty"`
	ToolCallID     *string         `json:"tool_call_id,omitempty"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
	SequenceNumber int             `json:"sequence_number"`
	CreatedAt      time.Time       `json:"created_at"`
}

// NewAgentMemory creates a new agent memory entry
func NewAgentMemory(tenantID, runID, stepID uuid.UUID, role AgentMemoryRole, content string, seqNum int) *AgentMemory {
	return &AgentMemory{
		ID:             uuid.New(),
		TenantID:       tenantID,
		RunID:          runID,
		StepID:         stepID,
		Role:           role,
		Content:        content,
		SequenceNumber: seqNum,
		CreatedAt:      time.Now().UTC(),
	}
}

// NewUserMessage creates a new user message
func NewUserMessage(tenantID, runID, stepID uuid.UUID, content string, seqNum int) *AgentMemory {
	return NewAgentMemory(tenantID, runID, stepID, AgentMemoryRoleUser, content, seqNum)
}

// NewAssistantMessage creates a new assistant message
func NewAssistantMessage(tenantID, runID, stepID uuid.UUID, content string, seqNum int) *AgentMemory {
	return NewAgentMemory(tenantID, runID, stepID, AgentMemoryRoleAssistant, content, seqNum)
}

// NewSystemMessage creates a new system message
func NewSystemMessage(tenantID, runID, stepID uuid.UUID, content string, seqNum int) *AgentMemory {
	return NewAgentMemory(tenantID, runID, stepID, AgentMemoryRoleSystem, content, seqNum)
}

// NewToolMessage creates a new tool result message
func NewToolMessage(tenantID, runID, stepID uuid.UUID, content string, toolCallID string, seqNum int) *AgentMemory {
	m := NewAgentMemory(tenantID, runID, stepID, AgentMemoryRoleTool, content, seqNum)
	m.ToolCallID = &toolCallID
	return m
}

// WithToolCalls adds tool calls to an assistant message
func (m *AgentMemory) WithToolCalls(toolCalls []ToolCall) *AgentMemory {
	m.ToolCalls = toolCalls
	return m
}

// WithMetadata adds metadata to a message
func (m *AgentMemory) WithMetadata(metadata map[string]interface{}) *AgentMemory {
	if metadata != nil {
		data, _ := json.Marshal(metadata)
		m.Metadata = data
	}
	return m
}

// ToLLMMessage converts agent memory to LLM message format
func (m *AgentMemory) ToLLMMessage() map[string]interface{} {
	msg := map[string]interface{}{
		"role":    string(m.Role),
		"content": m.Content,
	}

	if len(m.ToolCalls) > 0 {
		msg["tool_calls"] = m.ToolCalls
	}

	if m.ToolCallID != nil {
		msg["tool_call_id"] = *m.ToolCallID
	}

	return msg
}

// AgentMemoryKey represents the unique key for agent memory within a run
type AgentMemoryKey struct {
	RunID  uuid.UUID
	StepID uuid.UUID
}

// NewAgentMemoryKey creates a new memory key
func NewAgentMemoryKey(runID, stepID uuid.UUID) AgentMemoryKey {
	return AgentMemoryKey{
		RunID:  runID,
		StepID: stepID,
	}
}

// AgentMemoryBuffer represents a buffer of agent memory messages
type AgentMemoryBuffer struct {
	Messages   []*AgentMemory `json:"messages"`
	WindowSize int            `json:"window_size"`
}

// NewAgentMemoryBuffer creates a new memory buffer with a window size
func NewAgentMemoryBuffer(windowSize int) *AgentMemoryBuffer {
	if windowSize <= 0 {
		windowSize = 10 // Default window size
	}
	return &AgentMemoryBuffer{
		Messages:   make([]*AgentMemory, 0),
		WindowSize: windowSize,
	}
}

// Add adds a message to the buffer, trimming if necessary
func (b *AgentMemoryBuffer) Add(msg *AgentMemory) {
	b.Messages = append(b.Messages, msg)
	if len(b.Messages) > b.WindowSize {
		// Keep only the last WindowSize messages
		b.Messages = b.Messages[len(b.Messages)-b.WindowSize:]
	}
}

// ToLLMMessages converts the buffer to LLM message format
func (b *AgentMemoryBuffer) ToLLMMessages() []map[string]interface{} {
	messages := make([]map[string]interface{}, len(b.Messages))
	for i, m := range b.Messages {
		messages[i] = m.ToLLMMessage()
	}
	return messages
}

// GetLastN returns the last N messages from the buffer
func (b *AgentMemoryBuffer) GetLastN(n int) []*AgentMemory {
	if n <= 0 || len(b.Messages) == 0 {
		return nil
	}
	if n > len(b.Messages) {
		n = len(b.Messages)
	}
	return b.Messages[len(b.Messages)-n:]
}

// Clear clears all messages from the buffer
func (b *AgentMemoryBuffer) Clear() {
	b.Messages = make([]*AgentMemory, 0)
}
