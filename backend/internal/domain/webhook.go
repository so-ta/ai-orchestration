package domain

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Webhook represents a webhook endpoint for triggering workflows
type Webhook struct {
	ID              uuid.UUID       `json:"id"`
	TenantID        uuid.UUID       `json:"tenant_id"`
	WorkflowID      uuid.UUID       `json:"workflow_id"`
	WorkflowVersion int             `json:"workflow_version"`
	Name            string          `json:"name"`
	Description     string          `json:"description,omitempty"`
	Secret          string          `json:"secret"`
	InputMapping    json.RawMessage `json:"input_mapping,omitempty"`
	Enabled         bool            `json:"enabled"`
	LastTriggeredAt *time.Time      `json:"last_triggered_at,omitempty"`
	TriggerCount    int             `json:"trigger_count"`
	CreatedBy       *uuid.UUID      `json:"created_by,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// NewWebhook creates a new webhook
func NewWebhook(
	tenantID, workflowID uuid.UUID,
	workflowVersion int,
	name string,
	inputMapping json.RawMessage,
) *Webhook {
	now := time.Now().UTC()
	return &Webhook{
		ID:              uuid.New(),
		TenantID:        tenantID,
		WorkflowID:      workflowID,
		WorkflowVersion: workflowVersion,
		Name:            name,
		Secret:          generateSecret(),
		InputMapping:    inputMapping,
		Enabled:         true,
		TriggerCount:    0,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// generateSecret generates a random secret for webhook verification
func generateSecret() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// RegenerateSecret generates a new secret
func (w *Webhook) RegenerateSecret() {
	w.Secret = generateSecret()
	w.UpdatedAt = time.Now().UTC()
}

// Enable enables the webhook
func (w *Webhook) Enable() {
	w.Enabled = true
	w.UpdatedAt = time.Now().UTC()
}

// Disable disables the webhook
func (w *Webhook) Disable() {
	w.Enabled = false
	w.UpdatedAt = time.Now().UTC()
}

// RecordTrigger records that the webhook was triggered
func (w *Webhook) RecordTrigger() {
	now := time.Now().UTC()
	w.LastTriggeredAt = &now
	w.TriggerCount++
	w.UpdatedAt = now
}

// GetEndpointPath returns the webhook endpoint path
func (w *Webhook) GetEndpointPath() string {
	return "/api/v1/webhooks/" + w.ID.String() + "/trigger"
}
