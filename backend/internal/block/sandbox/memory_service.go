package sandbox

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/google/uuid"

	"github.com/souta/ai-orchestration/internal/domain"
)

// MemoryRepository interface for agent memory persistence
type MemoryRepository interface {
	Create(ctx context.Context, memory *domain.AgentMemory) error
	CreateBatch(ctx context.Context, memories []*domain.AgentMemory) error
	GetByRunAndStep(ctx context.Context, runID, stepID uuid.UUID) ([]*domain.AgentMemory, error)
	GetLastNByRunAndStep(ctx context.Context, runID, stepID uuid.UUID, n int) ([]*domain.AgentMemory, error)
	GetNextSequenceNumber(ctx context.Context, runID, stepID uuid.UUID) (int, error)
	DeleteByRunAndStep(ctx context.Context, runID, stepID uuid.UUID) error
}

// MemoryService provides memory management for agent blocks
type MemoryService struct {
	ctx      context.Context
	repo     MemoryRepository
	tenantID uuid.UUID
	runID    uuid.UUID
	stepID   uuid.UUID

	// In-memory buffer for current execution
	buffer     []*domain.AgentMemory
	bufferLock sync.Mutex
	nextSeq    int
}

// NewMemoryService creates a new MemoryService
func NewMemoryService(ctx context.Context, repo MemoryRepository, tenantID, runID, stepID uuid.UUID) *MemoryService {
	return &MemoryService{
		ctx:      ctx,
		repo:     repo,
		tenantID: tenantID,
		runID:    runID,
		stepID:   stepID,
		buffer:   make([]*domain.AgentMemory, 0),
		nextSeq:  1,
	}
}

// Initialize loads existing memory from the database
func (s *MemoryService) Initialize() error {
	if s.repo == nil {
		return nil
	}

	// Get next sequence number
	nextSeq, err := s.repo.GetNextSequenceNumber(s.ctx, s.runID, s.stepID)
	if err != nil {
		return err
	}
	s.nextSeq = nextSeq

	// Load existing memory
	memories, err := s.repo.GetByRunAndStep(s.ctx, s.runID, s.stepID)
	if err != nil {
		return err
	}
	s.buffer = memories

	return nil
}

// Get returns all messages in the memory buffer
func (s *MemoryService) Get(key string) []map[string]interface{} {
	s.bufferLock.Lock()
	defer s.bufferLock.Unlock()

	messages := make([]map[string]interface{}, len(s.buffer))
	for i, m := range s.buffer {
		messages[i] = m.ToLLMMessage()
	}
	return messages
}

// GetLastN returns the last N messages from memory
func (s *MemoryService) GetLastN(n int) []map[string]interface{} {
	s.bufferLock.Lock()
	defer s.bufferLock.Unlock()

	if n <= 0 || len(s.buffer) == 0 {
		return nil
	}

	start := len(s.buffer) - n
	if start < 0 {
		start = 0
	}

	messages := make([]map[string]interface{}, len(s.buffer)-start)
	for i, m := range s.buffer[start:] {
		messages[i] = m.ToLLMMessage()
	}
	return messages
}

// Add adds a message to memory
func (s *MemoryService) Add(role string, content string) error {
	return s.AddWithToolCalls(role, content, nil, nil)
}

// AddWithToolCalls adds a message with tool calls to memory
func (s *MemoryService) AddWithToolCalls(role string, content string, toolCalls []domain.ToolCall, toolCallID *string) error {
	s.bufferLock.Lock()
	defer s.bufferLock.Unlock()

	memory := &domain.AgentMemory{
		ID:             uuid.New(),
		TenantID:       s.tenantID,
		RunID:          s.runID,
		StepID:         s.stepID,
		Role:           domain.AgentMemoryRole(role),
		Content:        content,
		ToolCalls:      toolCalls,
		ToolCallID:     toolCallID,
		SequenceNumber: s.nextSeq,
	}

	s.buffer = append(s.buffer, memory)
	s.nextSeq++

	// Persist if repository is available
	if s.repo != nil {
		return s.repo.Create(s.ctx, memory)
	}

	return nil
}

// AddUser adds a user message
func (s *MemoryService) AddUser(content string) error {
	return s.Add("user", content)
}

// AddAssistant adds an assistant message
func (s *MemoryService) AddAssistant(content string) error {
	return s.Add("assistant", content)
}

// AddAssistantWithToolCalls adds an assistant message with tool calls
func (s *MemoryService) AddAssistantWithToolCalls(content string, toolCalls []domain.ToolCall) error {
	return s.AddWithToolCalls("assistant", content, toolCalls, nil)
}

// AddSystem adds a system message
func (s *MemoryService) AddSystem(content string) error {
	return s.Add("system", content)
}

// AddTool adds a tool result message
func (s *MemoryService) AddTool(content string, toolCallID string) error {
	return s.AddWithToolCalls("tool", content, nil, &toolCallID)
}

// Clear clears all messages from memory
func (s *MemoryService) Clear() error {
	s.bufferLock.Lock()
	defer s.bufferLock.Unlock()

	s.buffer = make([]*domain.AgentMemory, 0)
	s.nextSeq = 1

	if s.repo != nil {
		return s.repo.DeleteByRunAndStep(s.ctx, s.runID, s.stepID)
	}

	return nil
}

// Count returns the number of messages in memory
func (s *MemoryService) Count() int {
	s.bufferLock.Lock()
	defer s.bufferLock.Unlock()
	return len(s.buffer)
}

// ToLLMMessages converts all memory to LLM message format
func (s *MemoryService) ToLLMMessages() []map[string]interface{} {
	return s.Get("")
}

// MemoryServiceWrapper wraps MemoryService for JavaScript access
type MemoryServiceWrapper struct {
	service *MemoryService
}

// NewMemoryServiceWrapper creates a wrapper for JavaScript access
func NewMemoryServiceWrapper(service *MemoryService) *MemoryServiceWrapper {
	return &MemoryServiceWrapper{service: service}
}

// Get returns all messages (for JavaScript)
func (w *MemoryServiceWrapper) Get(key string) interface{} {
	return w.service.Get(key)
}

// GetLastN returns the last N messages (for JavaScript)
func (w *MemoryServiceWrapper) GetLastN(n int) interface{} {
	return w.service.GetLastN(n)
}

// Add adds a message (for JavaScript)
func (w *MemoryServiceWrapper) Add(role, content string) error {
	return w.service.Add(role, content)
}

// AddUser adds a user message (for JavaScript)
func (w *MemoryServiceWrapper) AddUser(content string) error {
	return w.service.AddUser(content)
}

// AddAssistant adds an assistant message (for JavaScript)
func (w *MemoryServiceWrapper) AddAssistant(content string) error {
	return w.service.AddAssistant(content)
}

// AddSystem adds a system message (for JavaScript)
func (w *MemoryServiceWrapper) AddSystem(content string) error {
	return w.service.AddSystem(content)
}

// AddTool adds a tool result message (for JavaScript)
func (w *MemoryServiceWrapper) AddTool(content, toolCallID string) error {
	return w.service.AddTool(content, toolCallID)
}

// AddWithToolCalls adds a message with tool calls (for JavaScript)
func (w *MemoryServiceWrapper) AddWithToolCalls(role, content string, toolCallsRaw interface{}) error {
	var toolCalls []domain.ToolCall
	if toolCallsRaw != nil {
		// Convert from JavaScript object to Go struct
		data, err := json.Marshal(toolCallsRaw)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(data, &toolCalls); err != nil {
			return err
		}
	}
	return w.service.AddWithToolCalls(role, content, toolCalls, nil)
}

// Clear clears all messages (for JavaScript)
func (w *MemoryServiceWrapper) Clear() error {
	return w.service.Clear()
}

// Count returns the message count (for JavaScript)
func (w *MemoryServiceWrapper) Count() int {
	return w.service.Count()
}
