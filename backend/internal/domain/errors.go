package domain

import "errors"

// Domain errors
var (
	// Project errors
	ErrProjectNotFound          = errors.New("project not found")
	ErrProjectAlreadyPublished  = errors.New("project is already published")
	ErrProjectNotPublished      = errors.New("project is not published")
	ErrProjectNotEditable       = errors.New("published project cannot be edited")
	ErrProjectHasCycle          = errors.New("project contains a cycle")
	ErrProjectHasUnconnected    = errors.New("project has unconnected steps")
	ErrProjectHasUnreachable    = errors.New("project has unreachable steps")
	ErrProjectBranchOutsideGroup = errors.New("branching blocks (condition/switch) with multiple outputs must be inside a Block Group")
	ErrProjectVersionNotFound   = errors.New("project version not found")

	// Step errors
	ErrStepNotFound     = errors.New("step not found")
	ErrInvalidStepType  = errors.New("invalid step type")
	ErrStepConfigInvalid = errors.New("step configuration is invalid")

	// Edge errors
	ErrEdgeNotFound        = errors.New("edge not found")
	ErrEdgeDuplicate       = errors.New("edge already exists")
	ErrEdgeSelfLoop        = errors.New("edge cannot connect step to itself")
	ErrEdgeCreatesCycle    = errors.New("edge would create a cycle")
	ErrEdgeInvalidPort     = errors.New("invalid port specified")
	ErrSourcePortNotFound  = errors.New("source port not found in block definition")
	ErrTargetPortNotFound  = errors.New("target port not found in block definition")

	// Run errors
	ErrRunNotFound      = errors.New("run not found")
	ErrRunNotCancellable = errors.New("run cannot be cancelled")
	ErrRunNotResumable  = errors.New("run cannot be resumed")

	// Step Run errors
	ErrStepRunNotFound = errors.New("step run not found")

	// Block Group errors
	ErrBlockGroupNotFound    = errors.New("block group not found")
	ErrBlockGroupInvalidType = errors.New("invalid block group type")
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
	ErrCredentialNotFound       = errors.New("credential not found")
	ErrCredentialExpired        = errors.New("credential has expired")
	ErrCredentialRevoked        = errors.New("credential has been revoked")
	ErrCredentialInvalidScope   = errors.New("credential scope is inconsistent with project_id and owner_user_id")
	ErrCredentialAccessDenied   = errors.New("access to credential denied")
	ErrCredentialBindingMissing = errors.New("required credential binding not found")

	// OAuth2 errors
	ErrOAuth2ProviderNotFound   = errors.New("oauth2 provider not found")
	ErrOAuth2AppNotFound        = errors.New("oauth2 app not found")
	ErrOAuth2AppAlreadyExists   = errors.New("oauth2 app already exists for this provider")
	ErrOAuth2ConnectionNotFound = errors.New("oauth2 connection not found")
	ErrOAuth2InvalidState       = errors.New("invalid oauth2 state parameter")
	ErrOAuth2TokenExpired       = errors.New("oauth2 access token expired")
	ErrOAuth2RefreshFailed      = errors.New("oauth2 token refresh failed")

	// Credential Share errors
	ErrCredentialShareNotFound  = errors.New("credential share not found")
	ErrCredentialShareDuplicate = errors.New("credential share already exists")

	// System Credential errors
	ErrSystemCredentialNotFound = errors.New("system credential not found")
	ErrSystemCredentialExpired  = errors.New("system credential has expired")
	ErrSystemCredentialRevoked  = errors.New("system credential has been revoked")

	// Copilot Session errors
	ErrCopilotSessionNotFound = errors.New("copilot session not found")

	// Block Definition errors (additional)
	ErrBlockDefinitionNotFound   = errors.New("block definition not found")
	ErrBlockDefinitionSlugExists = errors.New("block definition slug already exists")
	ErrBlockCodeHidden           = errors.New("block code is hidden for system blocks")

	// Block Inheritance errors
	ErrCircularInheritance     = errors.New("circular inheritance detected")
	ErrBlockNotInheritable     = errors.New("block cannot be inherited (no code)")
	ErrInheritanceDepthExceeded = errors.New("inheritance depth exceeded maximum limit")
	ErrParentBlockNotFound     = errors.New("parent block not found")
	ErrInternalStepNotFound    = errors.New("internal step block not found")

	// Tenant errors
	ErrTenantNotFound = errors.New("tenant not found")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrForbidden      = errors.New("forbidden")

	// Template errors
	ErrTemplateNotFound = errors.New("template not found")

	// Git Sync errors
	ErrGitSyncNotFound = errors.New("git sync configuration not found")

	// Block Package errors
	ErrBlockPackageNotFound = errors.New("block package not found")

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
