package handler

import (
	"encoding/json"
	"testing"

	"github.com/souta/ai-orchestration/internal/domain"
)

func TestValidateCredentialBindings(t *testing.T) {
	tests := []struct {
		name    string
		input   json.RawMessage
		wantErr error
	}{
		{
			name:    "nil input",
			input:   nil,
			wantErr: nil,
		},
		{
			name:    "empty input",
			input:   json.RawMessage{},
			wantErr: nil,
		},
		{
			name:    "null string",
			input:   json.RawMessage(`null`),
			wantErr: nil,
		},
		{
			name:    "empty object",
			input:   json.RawMessage(`{}`),
			wantErr: nil,
		},
		{
			name:    "valid single binding",
			input:   json.RawMessage(`{"api_key": "550e8400-e29b-41d4-a716-446655440000"}`),
			wantErr: nil,
		},
		{
			name:    "valid multiple bindings",
			input:   json.RawMessage(`{"api_key": "550e8400-e29b-41d4-a716-446655440000", "oauth_token": "550e8400-e29b-41d4-a716-446655440001"}`),
			wantErr: nil,
		},
		{
			name:    "valid with empty value",
			input:   json.RawMessage(`{"api_key": ""}`),
			wantErr: nil,
		},
		{
			name:    "invalid JSON",
			input:   json.RawMessage(`{invalid`),
			wantErr: domain.ErrValidation,
		},
		{
			name:    "invalid - not an object",
			input:   json.RawMessage(`["not", "an", "object"]`),
			wantErr: domain.ErrValidation,
		},
		{
			name:    "invalid - number value",
			input:   json.RawMessage(`{"api_key": 123}`),
			wantErr: domain.ErrValidation,
		},
		{
			name:    "invalid UUID format",
			input:   json.RawMessage(`{"api_key": "not-a-uuid"}`),
			wantErr: domain.ErrValidation,
		},
		{
			name:    "invalid UUID - too short",
			input:   json.RawMessage(`{"api_key": "550e8400-e29b-41d4"}`),
			wantErr: domain.ErrValidation,
		},
		{
			name:    "invalid UUID - wrong characters",
			input:   json.RawMessage(`{"api_key": "550e8400-e29b-41d4-a716-44665544zzzz"}`),
			wantErr: domain.ErrValidation,
		},
		{
			name:    "mixed valid and invalid UUID",
			input:   json.RawMessage(`{"valid": "550e8400-e29b-41d4-a716-446655440000", "invalid": "not-valid"}`),
			wantErr: domain.ErrValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCredentialBindings(tt.input)
			if err != tt.wantErr {
				t.Errorf("validateCredentialBindings() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
