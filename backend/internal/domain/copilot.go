package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// CopilotSession represents a chat session for a user and project
type CopilotSession struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	UserID    string    `json:"user_id"`
	ProjectID uuid.UUID `json:"project_id"`
	Title     string    `json:"title"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Loaded relations
	Messages []CopilotMessage `json:"messages,omitempty"`
}

// CopilotMessage represents a single message in a copilot session
type CopilotMessage struct {
	ID        uuid.UUID       `json:"id"`
	SessionID uuid.UUID       `json:"session_id"`
	Role      string          `json:"role"` // "user" or "assistant"
	Content   string          `json:"content"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

// NewCopilotSession creates a new copilot session
func NewCopilotSession(tenantID uuid.UUID, userID string, projectID uuid.UUID) *CopilotSession {
	now := time.Now().UTC()
	return &CopilotSession{
		ID:        uuid.New(),
		TenantID:  tenantID,
		UserID:    userID,
		ProjectID: projectID,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// NewCopilotMessage creates a new copilot message
func NewCopilotMessage(sessionID uuid.UUID, role, content string) *CopilotMessage {
	return &CopilotMessage{
		ID:        uuid.New(),
		SessionID: sessionID,
		Role:      role,
		Content:   content,
		CreatedAt: time.Now().UTC(),
	}
}

// AddMessage adds a message to the session
func (s *CopilotSession) AddMessage(role, content string) *CopilotMessage {
	msg := NewCopilotMessage(s.ID, role, content)
	s.Messages = append(s.Messages, *msg)
	return msg
}

// SetTitle sets the session title (usually from first message)
func (s *CopilotSession) SetTitle(title string) {
	if len(title) > 100 {
		title = title[:100] + "..."
	}
	s.Title = title
	s.UpdatedAt = time.Now().UTC()
}

// Close marks the session as inactive
func (s *CopilotSession) Close() {
	s.IsActive = false
	s.UpdatedAt = time.Now().UTC()
}
