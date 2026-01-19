// Package tools provides the tool registry and definitions for the Copilot agent.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// Tool represents a callable tool that the agent can use
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
	Handler     ToolHandler     `json:"-"`
}

// ToolHandler is the function signature for tool execution
type ToolHandler func(ctx context.Context, input json.RawMessage) (json.RawMessage, error)

// ToolCall represents a tool call from the LLM
type ToolCall struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	ToolUseID string          `json:"tool_use_id"`
	Content   json.RawMessage `json:"content"`
	IsError   bool            `json:"is_error,omitempty"`
}

// Registry manages available tools for the agent
type Registry struct {
	mu    sync.RWMutex
	tools map[string]*Tool
}

// NewRegistry creates a new tool registry
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]*Tool),
	}
}

// Register adds a tool to the registry
func (r *Registry) Register(tool *Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if tool.Name == "" {
		return fmt.Errorf("tool name is required")
	}
	if tool.Handler == nil {
		return fmt.Errorf("tool handler is required for tool %s", tool.Name)
	}
	if _, exists := r.tools[tool.Name]; exists {
		return fmt.Errorf("tool %s already registered", tool.Name)
	}

	r.tools[tool.Name] = tool
	return nil
}

// Get retrieves a tool by name
func (r *Registry) Get(name string) (*Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	return tool, exists
}

// GetAll returns all registered tools
func (r *Registry) GetAll() []*Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]*Tool, 0, len(r.tools))
	for _, t := range r.tools {
		tools = append(tools, t)
	}
	return tools
}

// GetToolDefinitions returns tool definitions for LLM API calls
func (r *Registry) GetToolDefinitions() []ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	definitions := make([]ToolDefinition, 0, len(r.tools))
	for _, t := range r.tools {
		definitions = append(definitions, ToolDefinition{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		})
	}
	return definitions
}

// Execute runs a tool by name with the given input
func (r *Registry) Execute(ctx context.Context, name string, input json.RawMessage) (json.RawMessage, error) {
	tool, exists := r.Get(name)
	if !exists {
		return nil, fmt.Errorf("tool %s not found", name)
	}

	return tool.Handler(ctx, input)
}

// ToolDefinition is the structure sent to the LLM API
type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// MarshalJSON implements json.Marshaler for Anthropic API format
func (td ToolDefinition) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"name":         td.Name,
		"description":  td.Description,
		"input_schema": td.InputSchema,
	})
}
