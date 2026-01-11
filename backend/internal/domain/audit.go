package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AuditAction represents the type of action being audited
type AuditAction string

const (
	// Workflow actions
	AuditActionWorkflowCreate  AuditAction = "workflow.create"
	AuditActionWorkflowUpdate  AuditAction = "workflow.update"
	AuditActionWorkflowDelete  AuditAction = "workflow.delete"
	AuditActionWorkflowPublish AuditAction = "workflow.publish"

	// Step actions
	AuditActionStepCreate AuditAction = "step.create"
	AuditActionStepUpdate AuditAction = "step.update"
	AuditActionStepDelete AuditAction = "step.delete"

	// Edge actions
	AuditActionEdgeCreate AuditAction = "edge.create"
	AuditActionEdgeDelete AuditAction = "edge.delete"

	// Run actions
	AuditActionRunCreate AuditAction = "run.create"
	AuditActionRunCancel AuditAction = "run.cancel"

	// Schedule actions
	AuditActionScheduleCreate  AuditAction = "schedule.create"
	AuditActionScheduleUpdate  AuditAction = "schedule.update"
	AuditActionScheduleDelete  AuditAction = "schedule.delete"
	AuditActionSchedulePause   AuditAction = "schedule.pause"
	AuditActionScheduleResume  AuditAction = "schedule.resume"
	AuditActionScheduleTrigger AuditAction = "schedule.trigger"

	// Webhook actions
	AuditActionWebhookCreate           AuditAction = "webhook.create"
	AuditActionWebhookUpdate           AuditAction = "webhook.update"
	AuditActionWebhookDelete           AuditAction = "webhook.delete"
	AuditActionWebhookEnable           AuditAction = "webhook.enable"
	AuditActionWebhookDisable          AuditAction = "webhook.disable"
	AuditActionWebhookTrigger          AuditAction = "webhook.trigger"
	AuditActionWebhookRegenerateSecret AuditAction = "webhook.regenerate_secret"

	// Auth actions
	AuditActionLogin  AuditAction = "auth.login"
	AuditActionLogout AuditAction = "auth.logout"

	// Secret actions
	AuditActionSecretCreate AuditAction = "secret.create"
	AuditActionSecretUpdate AuditAction = "secret.update"
	AuditActionSecretDelete AuditAction = "secret.delete"
)

// AuditResourceType represents the type of resource being audited
type AuditResourceType string

const (
	AuditResourceWorkflow AuditResourceType = "workflow"
	AuditResourceStep     AuditResourceType = "step"
	AuditResourceEdge     AuditResourceType = "edge"
	AuditResourceRun      AuditResourceType = "run"
	AuditResourceSchedule AuditResourceType = "schedule"
	AuditResourceWebhook  AuditResourceType = "webhook"
	AuditResourceUser     AuditResourceType = "user"
	AuditResourceSecret   AuditResourceType = "secret"
)

// AuditLog represents an audit log entry
type AuditLog struct {
	ID           uuid.UUID         `json:"id"`
	TenantID     uuid.UUID         `json:"tenant_id"`
	ActorID      *uuid.UUID        `json:"actor_id,omitempty"`
	ActorEmail   string            `json:"actor_email,omitempty"`
	Action       AuditAction       `json:"action"`
	ResourceType AuditResourceType `json:"resource_type"`
	ResourceID   *uuid.UUID        `json:"resource_id,omitempty"`
	Metadata     json.RawMessage   `json:"metadata,omitempty"`
	IPAddress    string            `json:"ip_address,omitempty"`
	UserAgent    string            `json:"user_agent,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
}

// NewAuditLog creates a new audit log entry
func NewAuditLog(
	tenantID uuid.UUID,
	actorID *uuid.UUID,
	actorEmail string,
	action AuditAction,
	resourceType AuditResourceType,
	resourceID *uuid.UUID,
	metadata json.RawMessage,
) *AuditLog {
	return &AuditLog{
		ID:           uuid.New(),
		TenantID:     tenantID,
		ActorID:      actorID,
		ActorEmail:   actorEmail,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Metadata:     metadata,
		CreatedAt:    time.Now().UTC(),
	}
}

// SetRequestInfo sets the request information on the audit log
func (a *AuditLog) SetRequestInfo(ipAddress, userAgent string) {
	a.IPAddress = ipAddress
	a.UserAgent = userAgent
}
