package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AgentChatSessionStatus represents the status of an agent chat session
type AgentChatSessionStatus string

const (
	AgentChatSessionStatusActive AgentChatSessionStatus = "active"
	AgentChatSessionStatusClosed AgentChatSessionStatus = "closed"
)

// AgentChatSession represents a chat session for agent-chat trigger
type AgentChatSession struct {
	ID          uuid.UUID              `json:"id"`
	TenantID    uuid.UUID              `json:"tenant_id"`
	ProjectID   uuid.UUID              `json:"project_id"`
	StartStepID uuid.UUID              `json:"start_step_id"`
	UserID      string                 `json:"user_id"`
	Status      AgentChatSessionStatus `json:"status"`
	Metadata    json.RawMessage        `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	ClosedAt    *time.Time             `json:"closed_at,omitempty"`

	// Loaded relations
	Runs []*Run `json:"runs,omitempty"`
}

// NewAgentChatSession creates a new agent chat session
func NewAgentChatSession(tenantID, projectID, startStepID uuid.UUID, userID string) *AgentChatSession {
	now := time.Now().UTC()
	return &AgentChatSession{
		ID:          uuid.New(),
		TenantID:    tenantID,
		ProjectID:   projectID,
		StartStepID: startStepID,
		UserID:      userID,
		Status:      AgentChatSessionStatusActive,
		Metadata:    json.RawMessage(`{}`),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// Close marks the session as closed
func (s *AgentChatSession) Close() {
	now := time.Now().UTC()
	s.Status = AgentChatSessionStatusClosed
	s.UpdatedAt = now
	s.ClosedAt = &now
}

// IsActive returns true if the session is active
func (s *AgentChatSession) IsActive() bool {
	return s.Status == AgentChatSessionStatusActive
}

// SetMetadata sets the session metadata
func (s *AgentChatSession) SetMetadata(metadata map[string]interface{}) error {
	if metadata == nil {
		s.Metadata = json.RawMessage(`{}`)
		return nil
	}
	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	s.Metadata = data
	s.UpdatedAt = time.Now().UTC()
	return nil
}

// GetMetadata returns the session metadata
func (s *AgentChatSession) GetMetadata() (map[string]interface{}, error) {
	if len(s.Metadata) == 0 {
		return nil, nil
	}
	var metadata map[string]interface{}
	if err := json.Unmarshal(s.Metadata, &metadata); err != nil {
		return nil, err
	}
	return metadata, nil
}

// AgentChatMessage represents a message in an agent chat session
type AgentChatMessage struct {
	ID        uuid.UUID `json:"id"`
	SessionID uuid.UUID `json:"session_id"`
	RunID     *uuid.UUID `json:"run_id,omitempty"` // Associated run if any
	Role      string    `json:"role"`              // "user", "assistant", "system"
	Content   string    `json:"content"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// NewAgentChatMessage creates a new chat message
func NewAgentChatMessage(sessionID uuid.UUID, role, content string) *AgentChatMessage {
	return &AgentChatMessage{
		ID:        uuid.New(),
		SessionID: sessionID,
		Role:      role,
		Content:   content,
		Metadata:  json.RawMessage(`{}`),
		CreatedAt: time.Now().UTC(),
	}
}

// SetRunID associates the message with a run
func (m *AgentChatMessage) SetRunID(runID uuid.UUID) {
	m.RunID = &runID
}
