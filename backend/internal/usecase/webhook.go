package usecase

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// WebhookUsecase handles webhook business logic
type WebhookUsecase struct {
	webhookRepo  repository.WebhookRepository
	workflowRepo repository.WorkflowRepository
	runRepo      repository.RunRepository
}

// NewWebhookUsecase creates a new WebhookUsecase
func NewWebhookUsecase(
	webhookRepo repository.WebhookRepository,
	workflowRepo repository.WorkflowRepository,
	runRepo repository.RunRepository,
) *WebhookUsecase {
	return &WebhookUsecase{
		webhookRepo:  webhookRepo,
		workflowRepo: workflowRepo,
		runRepo:      runRepo,
	}
}

// CreateWebhookInput represents input for creating a webhook
type CreateWebhookInput struct {
	TenantID     uuid.UUID
	WorkflowID   uuid.UUID
	Name         string
	Description  string
	InputMapping json.RawMessage
	CreatedBy    *uuid.UUID
}

// Create creates a new webhook
func (u *WebhookUsecase) Create(ctx context.Context, input CreateWebhookInput) (*domain.Webhook, error) {
	// Validate input
	if input.Name == "" {
		return nil, domain.NewValidationError("name", "name is required")
	}

	// Verify workflow exists and is published
	workflow, err := u.workflowRepo.GetByID(ctx, input.TenantID, input.WorkflowID)
	if err != nil {
		return nil, err
	}
	if workflow.Status != domain.WorkflowStatusPublished {
		return nil, domain.NewValidationError("workflow_id", "workflow must be published")
	}

	webhook := domain.NewWebhook(
		input.TenantID,
		input.WorkflowID,
		workflow.Version,
		input.Name,
		input.InputMapping,
	)
	webhook.Description = input.Description
	webhook.CreatedBy = input.CreatedBy

	if err := u.webhookRepo.Create(ctx, webhook); err != nil {
		return nil, err
	}

	return webhook, nil
}

// GetByID retrieves a webhook by ID
func (u *WebhookUsecase) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Webhook, error) {
	return u.webhookRepo.GetByID(ctx, tenantID, id)
}

// ListWebhooksInput represents input for listing webhooks
type ListWebhooksInput struct {
	TenantID   uuid.UUID
	WorkflowID *uuid.UUID
	Enabled    *bool
	Page       int
	Limit      int
}

// ListWebhooksOutput represents output for listing webhooks
type ListWebhooksOutput struct {
	Webhooks []*domain.Webhook
	Total    int
	Page     int
	Limit    int
}

// List lists webhooks with pagination
func (u *WebhookUsecase) List(ctx context.Context, input ListWebhooksInput) (*ListWebhooksOutput, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.Limit < 1 || input.Limit > 100 {
		input.Limit = 20
	}

	filter := repository.WebhookFilter{
		WorkflowID: input.WorkflowID,
		Enabled:    input.Enabled,
		Page:       input.Page,
		Limit:      input.Limit,
	}

	webhooks, total, err := u.webhookRepo.ListByTenant(ctx, input.TenantID, filter)
	if err != nil {
		return nil, err
	}

	return &ListWebhooksOutput{
		Webhooks: webhooks,
		Total:    total,
		Page:     input.Page,
		Limit:    input.Limit,
	}, nil
}

// UpdateWebhookInput represents input for updating a webhook
type UpdateWebhookInput struct {
	TenantID     uuid.UUID
	ID           uuid.UUID
	Name         string
	Description  string
	InputMapping json.RawMessage
}

// Update updates a webhook
func (u *WebhookUsecase) Update(ctx context.Context, input UpdateWebhookInput) (*domain.Webhook, error) {
	webhook, err := u.webhookRepo.GetByID(ctx, input.TenantID, input.ID)
	if err != nil {
		return nil, err
	}

	if input.Name != "" {
		webhook.Name = input.Name
	}
	webhook.Description = input.Description

	if input.InputMapping != nil {
		webhook.InputMapping = input.InputMapping
	}

	webhook.UpdatedAt = time.Now().UTC()

	if err := u.webhookRepo.Update(ctx, webhook); err != nil {
		return nil, err
	}

	return webhook, nil
}

// Delete deletes a webhook
func (u *WebhookUsecase) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return u.webhookRepo.Delete(ctx, tenantID, id)
}

// Enable enables a webhook
func (u *WebhookUsecase) Enable(ctx context.Context, tenantID, id uuid.UUID) (*domain.Webhook, error) {
	webhook, err := u.webhookRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	webhook.Enable()

	if err := u.webhookRepo.Update(ctx, webhook); err != nil {
		return nil, err
	}

	return webhook, nil
}

// Disable disables a webhook
func (u *WebhookUsecase) Disable(ctx context.Context, tenantID, id uuid.UUID) (*domain.Webhook, error) {
	webhook, err := u.webhookRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	webhook.Disable()

	if err := u.webhookRepo.Update(ctx, webhook); err != nil {
		return nil, err
	}

	return webhook, nil
}

// RegenerateSecret regenerates the webhook secret
func (u *WebhookUsecase) RegenerateSecret(ctx context.Context, tenantID, id uuid.UUID) (*domain.Webhook, error) {
	webhook, err := u.webhookRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	webhook.RegenerateSecret()

	if err := u.webhookRepo.Update(ctx, webhook); err != nil {
		return nil, err
	}

	return webhook, nil
}

// TriggerWebhookInput represents input for triggering a webhook
type TriggerWebhookInput struct {
	WebhookID uuid.UUID
	Signature string
	Payload   json.RawMessage
}

// Trigger triggers a webhook and creates a run
func (u *WebhookUsecase) Trigger(ctx context.Context, input TriggerWebhookInput) (*domain.Run, error) {
	webhook, err := u.webhookRepo.GetByIDForTrigger(ctx, input.WebhookID)
	if err != nil {
		return nil, err
	}

	if !webhook.Enabled {
		return nil, domain.ErrWebhookDisabled
	}

	// Verify signature
	if !u.verifySignature(webhook.Secret, input.Payload, input.Signature) {
		return nil, domain.ErrWebhookInvalidSecret
	}

	// Apply input mapping if configured
	var runInput json.RawMessage
	if webhook.InputMapping != nil {
		// In production, apply the mapping transformation
		// For now, just use the raw payload
		runInput = input.Payload
	} else {
		runInput = input.Payload
	}

	// Create a new run
	run := domain.NewRun(
		webhook.TenantID,
		webhook.WorkflowID,
		webhook.WorkflowVersion,
		runInput,
		domain.RunModeProduction,
		domain.TriggerTypeWebhook,
	)

	if err := u.runRepo.Create(ctx, run); err != nil {
		return nil, err
	}

	// Update webhook stats
	webhook.RecordTrigger()
	if err := u.webhookRepo.Update(ctx, webhook); err != nil {
		// Log error but don't fail - run was created successfully
	}

	return run, nil
}

// verifySignature verifies the webhook signature
func (u *WebhookUsecase) verifySignature(secret string, payload json.RawMessage, signature string) bool {
	if signature == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSig))
}
