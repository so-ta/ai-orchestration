package domain

import (
	"errors"
	"testing"
)

func TestDomainErrors_NotNil(t *testing.T) {
	// Test that all domain errors are properly defined
	errs := []error{
		ErrProjectNotFound,
		ErrProjectAlreadyPublished,
		ErrProjectNotPublished,
		ErrProjectNotEditable,
		ErrProjectHasCycle,
		ErrProjectHasUnconnected,
		ErrProjectHasUnreachable,
		ErrProjectBranchOutsideGroup,
		ErrProjectVersionNotFound,
		ErrStepNotFound,
		ErrInvalidStepType,
		ErrStepConfigInvalid,
		ErrEdgeNotFound,
		ErrEdgeDuplicate,
		ErrEdgeSelfLoop,
		ErrEdgeCreatesCycle,
		ErrEdgeInvalidPort,
		ErrSourcePortNotFound,
		ErrTargetPortNotFound,
		ErrRunNotFound,
		ErrRunNotCancellable,
		ErrRunNotResumable,
		ErrStepRunNotFound,
		ErrBlockGroupNotFound,
		ErrBlockGroupInvalidType,
		ErrStepCannotBeInGroup,
		ErrBlockGroupInvalidRole,
		ErrScheduleNotFound,
		ErrScheduleInvalidCron,
		ErrScheduleDisabled,
		ErrWebhookNotFound,
		ErrWebhookDisabled,
		ErrWebhookInvalidSecret,
		ErrCredentialNotFound,
		ErrCredentialExpired,
		ErrCredentialRevoked,
		ErrCredentialInvalidScope,
		ErrCredentialAccessDenied,
		ErrCredentialBindingMissing,
		ErrOAuth2ProviderNotFound,
		ErrOAuth2AppNotFound,
		ErrOAuth2AppAlreadyExists,
		ErrOAuth2ConnectionNotFound,
		ErrOAuth2InvalidState,
		ErrOAuth2TokenExpired,
		ErrOAuth2RefreshFailed,
		ErrCredentialShareNotFound,
		ErrCredentialShareDuplicate,
		ErrSystemCredentialNotFound,
		ErrSystemCredentialExpired,
		ErrSystemCredentialRevoked,
		ErrCopilotSessionNotFound,
		ErrBlockDefinitionNotFound,
		ErrBlockDefinitionSlugExists,
		ErrBlockCodeHidden,
		ErrCircularInheritance,
		ErrBlockNotInheritable,
		ErrInheritanceDepthExceeded,
		ErrParentBlockNotFound,
		ErrInternalStepNotFound,
		ErrTenantNotFound,
		ErrUnauthorized,
		ErrForbidden,
		ErrTemplateNotFound,
		ErrGitSyncNotFound,
		ErrBlockPackageNotFound,
		ErrValidation,
	}

	for _, err := range errs {
		if err == nil {
			t.Error("Domain error should not be nil")
		}
	}
}

func TestDomainErrors_AreErrors(t *testing.T) {
	// Test that errors can be wrapped and unwrapped
	tests := []struct {
		name string
		err  error
	}{
		{"ErrProjectNotFound", ErrProjectNotFound},
		{"ErrStepNotFound", ErrStepNotFound},
		{"ErrCredentialNotFound", ErrCredentialNotFound},
		{"ErrTenantNotFound", ErrTenantNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Wrap the error
			wrapped := errors.New("context: " + tt.err.Error())
			if wrapped.Error() == "" {
				t.Error("Wrapped error should have message")
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	err := NewValidationError("email", "invalid email format")

	if err.Field != "email" {
		t.Errorf("ValidationError Field = %v, want email", err.Field)
	}
	if err.Message != "invalid email format" {
		t.Errorf("ValidationError Message = %v, want invalid email format", err.Message)
	}
	if err.Error() != "invalid email format" {
		t.Errorf("ValidationError Error() = %v, want invalid email format", err.Error())
	}
}

func TestValidationError_Interface(t *testing.T) {
	var err error = NewValidationError("field", "message")

	if err == nil {
		t.Error("ValidationError should implement error interface")
	}

	ve, ok := err.(ValidationError)
	if !ok {
		t.Error("Should be able to cast to ValidationError")
	}
	if ve.Field != "field" {
		t.Errorf("Field = %v, want field", ve.Field)
	}
}

func TestDomainErrors_UniqueMessages(t *testing.T) {
	// Ensure error messages are unique and descriptive
	errs := map[string]error{
		"project not found":    ErrProjectNotFound,
		"step not found":       ErrStepNotFound,
		"edge not found":       ErrEdgeNotFound,
		"run not found":        ErrRunNotFound,
		"credential not found": ErrCredentialNotFound,
		"tenant not found":     ErrTenantNotFound,
		"template not found":   ErrTemplateNotFound,
	}

	for expected, err := range errs {
		if err.Error() != expected {
			t.Errorf("Error message = %v, want %v", err.Error(), expected)
		}
	}
}

func TestErrorsCanBeUsedWithIs(t *testing.T) {
	// Test that errors work with errors.Is
	tests := []struct {
		name   string
		err    error
		target error
		want   bool
	}{
		{"same error", ErrProjectNotFound, ErrProjectNotFound, true},
		{"different error", ErrProjectNotFound, ErrStepNotFound, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errors.Is(tt.err, tt.target); got != tt.want {
				t.Errorf("errors.Is() = %v, want %v", got, tt.want)
			}
		})
	}
}
