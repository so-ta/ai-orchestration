package usecase

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// StepUsecase handles step business logic
type StepUsecase struct {
	projectRepo     repository.ProjectRepository
	stepRepo        repository.StepRepository
	blockDefRepo    repository.BlockDefinitionRepository
	credentialRepo  repository.CredentialRepository
	projectChecker  *ProjectChecker
}

// NewStepUsecase creates a new StepUsecase
func NewStepUsecase(
	projectRepo repository.ProjectRepository,
	stepRepo repository.StepRepository,
	blockDefRepo repository.BlockDefinitionRepository,
	credentialRepo repository.CredentialRepository,
) *StepUsecase {
	return &StepUsecase{
		projectRepo:    projectRepo,
		stepRepo:       stepRepo,
		blockDefRepo:   blockDefRepo,
		credentialRepo: credentialRepo,
		projectChecker: NewProjectChecker(projectRepo),
	}
}

// validateCredentialBindingsTenant validates that all credential IDs in bindings belong to the tenant
func (u *StepUsecase) validateCredentialBindingsTenant(ctx context.Context, tenantID uuid.UUID, bindings json.RawMessage) error {
	if len(bindings) == 0 || string(bindings) == "null" {
		return nil
	}

	var parsed map[string]string
	if err := json.Unmarshal(bindings, &parsed); err != nil {
		return domain.ErrValidation
	}

	for _, credIDStr := range parsed {
		if credIDStr == "" {
			continue
		}
		credID, err := uuid.Parse(credIDStr)
		if err != nil {
			return domain.ErrValidation
		}
		// Verify credential belongs to the tenant
		if _, err := u.credentialRepo.GetByID(ctx, tenantID, credID); err != nil {
			return domain.ErrCredentialNotFound
		}
	}
	return nil
}

// CreateStepInput represents input for creating a step
type CreateStepInput struct {
	TenantID           uuid.UUID
	ProjectID          uuid.UUID
	Name               string
	Type               domain.StepType
	Config             json.RawMessage
	TriggerType        string          // For start blocks: manual, webhook, schedule, etc.
	TriggerConfig      json.RawMessage // Configuration for the trigger
	CredentialBindings json.RawMessage // Mapping of credential names to credential IDs
	PositionX          int
	PositionY          int
}

// Create creates a new step
func (u *StepUsecase) Create(ctx context.Context, input CreateStepInput) (*domain.Step, error) {
	// Verify project exists and is editable
	if _, err := u.projectChecker.CheckEditable(ctx, input.TenantID, input.ProjectID); err != nil {
		return nil, err
	}

	// Validate input
	if input.Name == "" {
		return nil, domain.NewValidationError("name", "name is required")
	}

	// Check if type is a built-in step type or a custom block definition
	var blockDef *domain.BlockDefinition
	if !input.Type.IsValid() {
		// Try to find as a custom block definition
		var err error
		blockDef, err = u.blockDefRepo.GetBySlug(ctx, &input.TenantID, string(input.Type))
		if err != nil || blockDef == nil {
			// Also try system blocks (tenant_id = NULL)
			blockDef, err = u.blockDefRepo.GetBySlug(ctx, nil, string(input.Type))
			if err != nil || blockDef == nil {
				return nil, domain.ErrInvalidStepType
			}
		}
	}

	// Apply ConfigDefaults from BlockDefinition if available
	mergedConfig := input.Config
	if blockDef != nil {
		mergedConfig = mergeConfigWithDefaults(input.Config, blockDef.GetEffectiveConfigDefaults())
	}

	// Normalize trigger block types to "start"
	// Trigger blocks (manual_trigger, schedule_trigger, webhook_trigger) should have type "start"
	// with the specific trigger type stored in TriggerType field
	stepType := input.Type
	if domain.IsTriggerBlockSlug(string(input.Type)) {
		stepType = domain.StepTypeStart
		// Auto-set trigger type if not provided
		if input.TriggerType == "" {
			input.TriggerType = string(domain.GetTriggerTypeFromSlug(string(input.Type)))
		}
	}

	step := domain.NewStep(input.TenantID, input.ProjectID, input.Name, stepType, mergedConfig)
	if blockDef != nil {
		step.BlockDefinitionID = &blockDef.ID
	}
	step.SetPosition(input.PositionX, input.PositionY)

	// Set trigger type/config for start blocks
	if input.TriggerType != "" {
		tt := domain.StepTriggerType(input.TriggerType)
		step.TriggerType = &tt
	}
	if len(input.TriggerConfig) > 0 {
		step.TriggerConfig = input.TriggerConfig
	}

	// Set credential bindings (skip if null or empty)
	if len(input.CredentialBindings) > 0 && string(input.CredentialBindings) != "null" {
		// Validate that all credential IDs belong to the tenant
		if err := u.validateCredentialBindingsTenant(ctx, input.TenantID, input.CredentialBindings); err != nil {
			return nil, err
		}
		step.CredentialBindings = input.CredentialBindings
	}

	if err := u.stepRepo.Create(ctx, step); err != nil {
		return nil, err
	}

	return step, nil
}

// GetByID retrieves a step by ID
func (u *StepUsecase) GetByID(ctx context.Context, tenantID, projectID, stepID uuid.UUID) (*domain.Step, error) {
	// Verify project exists
	if _, err := u.projectChecker.CheckExists(ctx, tenantID, projectID); err != nil {
		return nil, err
	}
	return u.stepRepo.GetByID(ctx, tenantID, projectID, stepID)
}

// GetByIDOnly retrieves a step by ID only (without tenant/project verification)
// Used for webhook triggers where only step ID is known from the URL
func (u *StepUsecase) GetByIDOnly(ctx context.Context, stepID uuid.UUID) (*domain.Step, error) {
	return u.stepRepo.GetByIDOnly(ctx, stepID)
}

// List lists steps for a project
func (u *StepUsecase) List(ctx context.Context, tenantID, projectID uuid.UUID) ([]*domain.Step, error) {
	// Verify project exists
	if _, err := u.projectChecker.CheckExists(ctx, tenantID, projectID); err != nil {
		return nil, err
	}
	return u.stepRepo.ListByProject(ctx, tenantID, projectID)
}

// UpdateStepInput represents input for updating a step
type UpdateStepInput struct {
	TenantID           uuid.UUID
	ProjectID          uuid.UUID
	StepID             uuid.UUID
	Name               string
	Type               domain.StepType
	Config             json.RawMessage
	TriggerType        string          // For start blocks: manual, webhook, schedule, etc.
	TriggerConfig      json.RawMessage // Configuration for the trigger
	CredentialBindings json.RawMessage // Mapping of credential names to credential IDs
	PositionX          *int
	PositionY          *int
}

// Update updates a step
func (u *StepUsecase) Update(ctx context.Context, input UpdateStepInput) (*domain.Step, error) {
	// Verify project is editable
	if _, err := u.projectChecker.CheckEditable(ctx, input.TenantID, input.ProjectID); err != nil {
		return nil, err
	}

	step, err := u.stepRepo.GetByID(ctx, input.TenantID, input.ProjectID, input.StepID)
	if err != nil {
		return nil, err
	}

	if input.Name != "" {
		step.Name = input.Name
	}
	if input.Type != "" {
		// Normalize trigger block types to "start"
		stepType := input.Type
		if domain.IsTriggerBlockSlug(string(input.Type)) {
			stepType = domain.StepTypeStart
			// Auto-set trigger type if not provided
			if input.TriggerType == "" {
				input.TriggerType = string(domain.GetTriggerTypeFromSlug(string(input.Type)))
			}
		}
		if stepType.IsValid() {
			step.Type = stepType
		}
	}
	if input.Config != nil {
		step.Config = input.Config
	}
	if input.PositionX != nil {
		step.PositionX = *input.PositionX
	}
	if input.PositionY != nil {
		step.PositionY = *input.PositionY
	}
	// Update trigger type/config for start blocks
	if input.TriggerType != "" {
		tt := domain.StepTriggerType(input.TriggerType)
		step.TriggerType = &tt
	}
	if len(input.TriggerConfig) > 0 {
		step.TriggerConfig = input.TriggerConfig
	}

	// Update credential bindings (skip if null or empty)
	if len(input.CredentialBindings) > 0 && string(input.CredentialBindings) != "null" {
		// Validate that all credential IDs belong to the tenant
		if err := u.validateCredentialBindingsTenant(ctx, input.TenantID, input.CredentialBindings); err != nil {
			return nil, err
		}
		step.CredentialBindings = input.CredentialBindings
	}

	if err := u.stepRepo.Update(ctx, step); err != nil {
		return nil, err
	}

	return step, nil
}

// Delete deletes a step
func (u *StepUsecase) Delete(ctx context.Context, tenantID, projectID, stepID uuid.UUID) error {
	// Verify project is editable
	if _, err := u.projectChecker.CheckEditable(ctx, tenantID, projectID); err != nil {
		return err
	}

	return u.stepRepo.Delete(ctx, tenantID, projectID, stepID)
}

// UpdateRetryConfigInput represents input for updating retry config
type UpdateRetryConfigInput struct {
	TenantID   uuid.UUID
	ProjectID  uuid.UUID
	StepID     uuid.UUID
	RetryConfig *domain.RetryConfig
}

// UpdateRetryConfig updates the retry configuration for a step
func (u *StepUsecase) UpdateRetryConfig(ctx context.Context, input UpdateRetryConfigInput) (*domain.Step, error) {
	// Verify project is editable
	if _, err := u.projectChecker.CheckEditable(ctx, input.TenantID, input.ProjectID); err != nil {
		return nil, err
	}

	step, err := u.stepRepo.GetByID(ctx, input.TenantID, input.ProjectID, input.StepID)
	if err != nil {
		return nil, err
	}

	if input.RetryConfig != nil {
		// Validate retry config
		if input.RetryConfig.MaxRetries < 0 {
			return nil, domain.NewValidationError("max_retries", "max_retries must be non-negative")
		}
		if input.RetryConfig.DelayMs < 0 {
			return nil, domain.NewValidationError("delay_ms", "delay_ms must be non-negative")
		}

		retryJSON, err := json.Marshal(input.RetryConfig)
		if err != nil {
			return nil, err
		}
		step.RetryConfig = retryJSON
	} else {
		step.RetryConfig = nil
	}

	if err := u.stepRepo.Update(ctx, step); err != nil {
		return nil, err
	}

	return step, nil
}

// GetRetryConfig retrieves the retry configuration for a step
func (u *StepUsecase) GetRetryConfig(ctx context.Context, tenantID, projectID, stepID uuid.UUID) (*domain.RetryConfig, error) {
	step, err := u.stepRepo.GetByID(ctx, tenantID, projectID, stepID)
	if err != nil {
		return nil, err
	}

	if len(step.RetryConfig) == 0 || string(step.RetryConfig) == "null" {
		config := domain.DefaultRetryConfig()
		return &config, nil
	}

	var config domain.RetryConfig
	if err := json.Unmarshal(step.RetryConfig, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// EnableTrigger enables the trigger for a Start block
func (u *StepUsecase) EnableTrigger(ctx context.Context, tenantID, projectID, stepID uuid.UUID) (*domain.Step, error) {
	return u.setTriggerEnabled(ctx, tenantID, projectID, stepID, true)
}

// DisableTrigger disables the trigger for a Start block
func (u *StepUsecase) DisableTrigger(ctx context.Context, tenantID, projectID, stepID uuid.UUID) (*domain.Step, error) {
	return u.setTriggerEnabled(ctx, tenantID, projectID, stepID, false)
}

// setTriggerEnabled sets the enabled state for a trigger
func (u *StepUsecase) setTriggerEnabled(ctx context.Context, tenantID, projectID, stepID uuid.UUID, enabled bool) (*domain.Step, error) {
	// Verify project is editable
	if _, err := u.projectChecker.CheckEditable(ctx, tenantID, projectID); err != nil {
		return nil, err
	}

	step, err := u.stepRepo.GetByID(ctx, tenantID, projectID, stepID)
	if err != nil {
		return nil, err
	}

	// Verify this is a Start block with a trigger type
	if step.Type != domain.StepTypeStart {
		return nil, domain.NewValidationError("type", "step is not a start block")
	}
	if step.TriggerType == nil {
		return nil, domain.NewValidationError("trigger_type", "step has no trigger type")
	}

	// Manual triggers don't need enabled/disabled state
	if *step.TriggerType == domain.StepTriggerTypeManual {
		return nil, domain.NewValidationError("trigger_type", "manual triggers cannot be enabled/disabled")
	}

	// Parse current trigger config
	var triggerConfig map[string]interface{}
	if len(step.TriggerConfig) > 0 && string(step.TriggerConfig) != "null" {
		if err := json.Unmarshal(step.TriggerConfig, &triggerConfig); err != nil {
			triggerConfig = make(map[string]interface{})
		}
	} else {
		triggerConfig = make(map[string]interface{})
	}

	// Update enabled field
	triggerConfig["enabled"] = enabled

	// Marshal back to JSON
	updatedConfig, err := json.Marshal(triggerConfig)
	if err != nil {
		return nil, err
	}
	step.TriggerConfig = updatedConfig

	if err := u.stepRepo.Update(ctx, step); err != nil {
		return nil, err
	}

	return step, nil
}

// GetTriggerStatus returns the trigger status for a Start block
func (u *StepUsecase) GetTriggerStatus(ctx context.Context, tenantID, projectID, stepID uuid.UUID) (bool, error) {
	step, err := u.stepRepo.GetByID(ctx, tenantID, projectID, stepID)
	if err != nil {
		return false, err
	}

	if step.Type != domain.StepTypeStart || step.TriggerType == nil {
		return false, domain.NewValidationError("type", "step is not a start block with trigger")
	}

	// Manual triggers are always "enabled"
	if *step.TriggerType == domain.StepTriggerTypeManual {
		return true, nil
	}

	// Parse trigger config to get enabled state
	if len(step.TriggerConfig) == 0 || string(step.TriggerConfig) == "null" {
		return false, nil // Default to disabled if no config
	}

	var triggerConfig map[string]interface{}
	if err := json.Unmarshal(step.TriggerConfig, &triggerConfig); err != nil {
		return false, nil
	}

	enabled, ok := triggerConfig["enabled"].(bool)
	if !ok {
		return false, nil
	}

	return enabled, nil
}

// mergeConfigWithDefaults merges user config with default values from BlockDefinition
// User config takes precedence over defaults
func mergeConfigWithDefaults(userConfig json.RawMessage, defaults json.RawMessage) json.RawMessage {
	// If no defaults, return user config as-is
	if len(defaults) == 0 || string(defaults) == "null" {
		return userConfig
	}

	// Parse defaults
	var defaultMap map[string]interface{}
	if err := json.Unmarshal(defaults, &defaultMap); err != nil {
		return userConfig
	}

	// If no user config, return defaults
	if len(userConfig) == 0 || string(userConfig) == "null" {
		return defaults
	}

	// Parse user config
	var userMap map[string]interface{}
	if err := json.Unmarshal(userConfig, &userMap); err != nil {
		return userConfig
	}

	// Merge: start with defaults, overlay user config
	merged := make(map[string]interface{})
	for k, v := range defaultMap {
		merged[k] = v
	}
	for k, v := range userMap {
		merged[k] = v
	}

	// Marshal back to JSON
	result, err := json.Marshal(merged)
	if err != nil {
		return userConfig
	}

	return result
}
