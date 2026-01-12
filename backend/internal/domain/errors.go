package domain

import "errors"

// Domain errors
var (
	// Workflow errors
	ErrWorkflowNotFound         = errors.New("workflow not found")
	ErrWorkflowAlreadyPublished = errors.New("workflow is already published")
	ErrWorkflowNotPublished     = errors.New("workflow is not published")
	ErrWorkflowNotEditable      = errors.New("published workflow cannot be edited")
	ErrWorkflowHasCycle         = errors.New("workflow contains a cycle")
	ErrWorkflowHasUnconnected   = errors.New("workflow has unconnected steps")
	ErrWorkflowHasUnreachable   = errors.New("workflow has unreachable steps")
	ErrWorkflowVersionNotFound  = errors.New("workflow version not found")

	// Step errors
	ErrStepNotFound     = errors.New("step not found")
	ErrInvalidStepType  = errors.New("invalid step type")
	ErrStepConfigInvalid = errors.New("step configuration is invalid")

	// Edge errors
	ErrEdgeNotFound    = errors.New("edge not found")
	ErrEdgeDuplicate   = errors.New("edge already exists")
	ErrEdgeSelfLoop    = errors.New("edge cannot connect step to itself")
	ErrEdgeCreatesCycle = errors.New("edge would create a cycle")

	// Run errors
	ErrRunNotFound      = errors.New("run not found")
	ErrRunNotCancellable = errors.New("run cannot be cancelled")
	ErrRunNotResumable  = errors.New("run cannot be resumed")

	// Step Run errors
	ErrStepRunNotFound = errors.New("step run not found")

	// Block Group errors
	ErrBlockGroupNotFound      = errors.New("block group not found")
	ErrBlockGroupRunNotFound   = errors.New("block group run not found")
	ErrBlockGroupInvalidType   = errors.New("invalid block group type")
	ErrStepCannotBeInGroup     = errors.New("this step type cannot be added to a block group")
	ErrBlockGroupInvalidRole   = errors.New("invalid group role for this block group type")

	// Schedule errors
	ErrScheduleNotFound    = errors.New("schedule not found")
	ErrScheduleInvalidCron = errors.New("invalid cron expression")
	ErrScheduleDisabled    = errors.New("schedule is disabled")

	// Webhook errors
	ErrWebhookNotFound       = errors.New("webhook not found")
	ErrWebhookDisabled       = errors.New("webhook is disabled")
	ErrWebhookInvalidSecret  = errors.New("invalid webhook secret")

	// Credential errors
	ErrCredentialNotFound = errors.New("credential not found")
	ErrCredentialExpired  = errors.New("credential has expired")
	ErrCredentialRevoked  = errors.New("credential has been revoked")

	// System Credential errors
	ErrSystemCredentialNotFound = errors.New("system credential not found")
	ErrSystemCredentialExpired  = errors.New("system credential has expired")
	ErrSystemCredentialRevoked  = errors.New("system credential has been revoked")

	// Block Template errors
	ErrBlockTemplateNotFound     = errors.New("block template not found")
	ErrBlockTemplateIsBuiltin    = errors.New("cannot modify built-in template")
	ErrBlockTemplateSlugExists   = errors.New("block template slug already exists")

	// Copilot Session errors
	ErrCopilotSessionNotFound = errors.New("copilot session not found")

	// Block Definition errors (additional)
	ErrBlockDefinitionNotFound   = errors.New("block definition not found")
	ErrBlockDefinitionSlugExists = errors.New("block definition slug already exists")
	ErrBlockCodeHidden           = errors.New("block code is hidden for system blocks")

	// Tenant errors
	ErrTenantNotFound = errors.New("tenant not found")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrForbidden      = errors.New("forbidden")

	// Validation errors
	ErrValidation = errors.New("validation error")
)

// ValidationError wraps a validation error with field information
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return e.Message
}

// NewValidationError creates a new validation error
func NewValidationError(field, message string) ValidationError {
	return ValidationError{Field: field, Message: message}
}
