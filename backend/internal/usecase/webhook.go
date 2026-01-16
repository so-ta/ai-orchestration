package usecase

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
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
	stepRepo     repository.StepRepository
}

// NewWebhookUsecase creates a new WebhookUsecase
func NewWebhookUsecase(
	webhookRepo repository.WebhookRepository,
	workflowRepo repository.WorkflowRepository,
	runRepo repository.RunRepository,
	stepRepo repository.StepRepository,
) *WebhookUsecase {
	return &WebhookUsecase{
		webhookRepo:  webhookRepo,
		workflowRepo: workflowRepo,
		runRepo:      runRepo,
		stepRepo:     stepRepo,
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

	webhook, err := domain.NewWebhook(
		input.TenantID,
		input.WorkflowID,
		workflow.Version,
		input.Name,
		input.InputMapping,
	)
	if err != nil {
		return nil, err
	}
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
	input.Page, input.Limit = NormalizePagination(input.Page, input.Limit)

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

	if err := webhook.RegenerateSecret(); err != nil {
		return nil, err
	}

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
	if u.hasValidInputMapping(webhook.InputMapping) {
		mappedInput, err := u.applyInputMapping(input.Payload, webhook.InputMapping)
		if err != nil {
			return nil, fmt.Errorf("failed to apply input mapping: %w", err)
		}
		runInput = mappedInput
	} else {
		runInput = input.Payload
	}

	// Validate input against Start step's input_schema
	if err := u.validateWorkflowInput(ctx, webhook.TenantID, webhook.WorkflowID, runInput); err != nil {
		return nil, err
	}

	// Create a new run
	run := domain.NewRun(
		webhook.TenantID,
		webhook.WorkflowID,
		webhook.WorkflowVersion,
		runInput,
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

// validateWorkflowInput validates workflow input against Start step's input_schema.
// Note: This function intentionally skips validation (returns nil) when:
// - stepRepo is not available
// - Steps cannot be retrieved (e.g., database errors)
// - No start step is found
// - No input_schema is defined
// This design choice prioritizes workflow execution over strict validation,
// as the validation is an optional enhancement rather than a hard requirement.
func (u *WebhookUsecase) validateWorkflowInput(ctx context.Context, tenantID, workflowID uuid.UUID, input json.RawMessage) error {
	if u.stepRepo == nil {
		return nil // Skip validation if stepRepo is not available
	}

	// Get all steps for the workflow
	// Note: Errors are intentionally ignored to allow workflow execution to proceed
	// even when step metadata is temporarily unavailable
	steps, err := u.stepRepo.ListByWorkflow(ctx, tenantID, workflowID)
	if err != nil {
		return nil // Skip validation if we can't get steps
	}

	// Find the start step
	var startStep *domain.Step
	for _, step := range steps {
		if step.Type == "start" {
			startStep = step
			break
		}
	}

	if startStep == nil {
		return nil // No start step, skip validation
	}

	// Extract input_schema from start step's config
	inputSchema := ExtractInputSchemaFromConfig(startStep.Config)
	if inputSchema == nil {
		return nil // No input_schema defined, skip validation
	}

	// Validate input against schema
	return domain.ValidateInputSchema(input, inputSchema)
}

// Note: extractInputSchemaFromStepConfig has been moved to helpers.go as ExtractInputSchemaFromConfig

// hasValidInputMapping checks if the input mapping is configured and non-empty
func (u *WebhookUsecase) hasValidInputMapping(mapping json.RawMessage) bool {
	if mapping == nil || len(mapping) == 0 {
		return false
	}
	// Parse to check if it's a valid non-empty mapping
	var mappingConfig map[string]string
	if err := json.Unmarshal(mapping, &mappingConfig); err != nil {
		return false
	}
	return len(mappingConfig) > 0
}

// applyInputMapping transforms the payload according to the input mapping configuration.
// The mapping format is: {"output_field": "$.input.path", ...}
// Example: {"event_type": "$.action", "repo_name": "$.repository.name"}
func (u *WebhookUsecase) applyInputMapping(payload json.RawMessage, mapping json.RawMessage) (json.RawMessage, error) {
	// Parse the payload
	var payloadData map[string]interface{}
	if err := json.Unmarshal(payload, &payloadData); err != nil {
		return nil, fmt.Errorf("failed to parse payload: %w", err)
	}

	// Parse the mapping configuration
	var mappingConfig map[string]string
	if err := json.Unmarshal(mapping, &mappingConfig); err != nil {
		return nil, fmt.Errorf("failed to parse input mapping: %w", err)
	}

	// Apply the mapping
	result := make(map[string]interface{})
	for outputField, inputPath := range mappingConfig {
		value, err := u.resolvePath(inputPath, payloadData)
		if err != nil {
			// Skip fields that don't exist in the payload
			continue
		}
		result[outputField] = value
	}

	// Marshal the result
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal mapped result: %w", err)
	}

	return resultJSON, nil
}

// resolvePath resolves a JSONPath expression (e.g., "$.field.nested") against data
func (u *WebhookUsecase) resolvePath(path string, data map[string]interface{}) (interface{}, error) {
	// Remove JSONPath prefix
	path = strings.TrimPrefix(path, "$.")
	path = strings.TrimPrefix(path, "$")

	if path == "" {
		return data, nil
	}

	parts := strings.Split(path, ".")
	var current interface{} = data

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			val, ok := v[part]
			if !ok {
				return nil, fmt.Errorf("field not found: %s", part)
			}
			current = val
		default:
			return nil, fmt.Errorf("cannot access %s on non-object", part)
		}
	}

	return current, nil
}
