package usecase

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// ============================================================================
// Mock Credential Repository for Step Tests
// ============================================================================

type mockCredentialRepoForStep struct {
	credentials map[uuid.UUID]*domain.Credential
	getErr      error
}

func newMockCredentialRepoForStep() *mockCredentialRepoForStep {
	return &mockCredentialRepoForStep{
		credentials: make(map[uuid.UUID]*domain.Credential),
	}
}

func (m *mockCredentialRepoForStep) addCredential(tenantID uuid.UUID, cred *domain.Credential) {
	cred.TenantID = tenantID
	m.credentials[cred.ID] = cred
}

func (m *mockCredentialRepoForStep) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Credential, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	cred, ok := m.credentials[id]
	if !ok {
		return nil, domain.ErrCredentialNotFound
	}
	if cred.TenantID != tenantID {
		return nil, domain.ErrCredentialNotFound
	}
	return cred, nil
}

func (m *mockCredentialRepoForStep) GetByName(ctx context.Context, tenantID uuid.UUID, name string) (*domain.Credential, error) {
	for _, cred := range m.credentials {
		if cred.TenantID == tenantID && cred.Name == name {
			return cred, nil
		}
	}
	return nil, domain.ErrCredentialNotFound
}

func (m *mockCredentialRepoForStep) List(ctx context.Context, tenantID uuid.UUID, filter repository.CredentialFilter) ([]*domain.Credential, int, error) {
	var result []*domain.Credential
	for _, cred := range m.credentials {
		if cred.TenantID == tenantID {
			result = append(result, cred)
		}
	}
	return result, len(result), nil
}

func (m *mockCredentialRepoForStep) Create(ctx context.Context, cred *domain.Credential) error {
	m.credentials[cred.ID] = cred
	return nil
}

func (m *mockCredentialRepoForStep) Update(ctx context.Context, cred *domain.Credential) error {
	m.credentials[cred.ID] = cred
	return nil
}

func (m *mockCredentialRepoForStep) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	delete(m.credentials, id)
	return nil
}

func (m *mockCredentialRepoForStep) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.CredentialStatus) error {
	if cred, ok := m.credentials[id]; ok {
		cred.Status = status
	}
	return nil
}

// ============================================================================
// Test Helpers
// ============================================================================

func createTestStepUsecase(credRepo *mockCredentialRepoForStep) *StepUsecase {
	return &StepUsecase{
		credentialRepo: credRepo,
	}
}

// ============================================================================
// Tests for validateCredentialBindingsTenant
// ============================================================================

func TestStepUsecase_ValidateCredentialBindingsTenant(t *testing.T) {
	tenantID := uuid.New()
	otherTenantID := uuid.New()
	credID := uuid.New()

	tests := []struct {
		name     string
		tenantID uuid.UUID
		bindings json.RawMessage
		setup    func(*mockCredentialRepoForStep)
		wantErr  error
	}{
		{
			name:     "nil bindings",
			tenantID: tenantID,
			bindings: nil,
			setup:    func(m *mockCredentialRepoForStep) {},
			wantErr:  nil,
		},
		{
			name:     "empty bindings",
			tenantID: tenantID,
			bindings: json.RawMessage{},
			setup:    func(m *mockCredentialRepoForStep) {},
			wantErr:  nil,
		},
		{
			name:     "null string",
			tenantID: tenantID,
			bindings: json.RawMessage(`null`),
			setup:    func(m *mockCredentialRepoForStep) {},
			wantErr:  nil,
		},
		{
			name:     "empty object",
			tenantID: tenantID,
			bindings: json.RawMessage(`{}`),
			setup:    func(m *mockCredentialRepoForStep) {},
			wantErr:  nil,
		},
		{
			name:     "valid binding - credential belongs to tenant",
			tenantID: tenantID,
			bindings: json.RawMessage(`{"api_key": "` + credID.String() + `"}`),
			setup: func(m *mockCredentialRepoForStep) {
				cred := &domain.Credential{ID: credID, Name: "API Key"}
				m.addCredential(tenantID, cred)
			},
			wantErr: nil,
		},
		{
			name:     "valid binding with empty value",
			tenantID: tenantID,
			bindings: json.RawMessage(`{"api_key": ""}`),
			setup:    func(m *mockCredentialRepoForStep) {},
			wantErr:  nil,
		},
		{
			name:     "invalid JSON",
			tenantID: tenantID,
			bindings: json.RawMessage(`{invalid`),
			setup:    func(m *mockCredentialRepoForStep) {},
			wantErr:  domain.ErrValidation,
		},
		{
			name:     "invalid UUID format",
			tenantID: tenantID,
			bindings: json.RawMessage(`{"api_key": "not-a-uuid"}`),
			setup:    func(m *mockCredentialRepoForStep) {},
			wantErr:  domain.ErrValidation,
		},
		{
			name:     "credential not found",
			tenantID: tenantID,
			bindings: json.RawMessage(`{"api_key": "` + credID.String() + `"}`),
			setup:    func(m *mockCredentialRepoForStep) {}, // No credential added
			wantErr:  domain.ErrCredentialNotFound,
		},
		{
			name:     "credential belongs to different tenant",
			tenantID: tenantID,
			bindings: json.RawMessage(`{"api_key": "` + credID.String() + `"}`),
			setup: func(m *mockCredentialRepoForStep) {
				cred := &domain.Credential{ID: credID, Name: "API Key"}
				m.addCredential(otherTenantID, cred) // Add to different tenant
			},
			wantErr: domain.ErrCredentialNotFound,
		},
		{
			name:     "multiple bindings - all valid",
			tenantID: tenantID,
			bindings: func() json.RawMessage {
				credID2 := uuid.New()
				return json.RawMessage(`{"api_key": "` + credID.String() + `", "oauth": "` + credID2.String() + `"}`)
			}(),
			setup: func(m *mockCredentialRepoForStep) {
				cred1 := &domain.Credential{ID: credID, Name: "API Key"}
				m.addCredential(tenantID, cred1)
				// Note: credID2 is generated inline, so we parse it from the bindings
				// For simplicity, we'll use a known UUID
			},
			wantErr: domain.ErrCredentialNotFound, // Second credential not found
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			credRepo := newMockCredentialRepoForStep()
			tt.setup(credRepo)

			usecase := createTestStepUsecase(credRepo)
			err := usecase.validateCredentialBindingsTenant(context.Background(), tt.tenantID, tt.bindings)

			if err != tt.wantErr {
				t.Errorf("validateCredentialBindingsTenant() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStepUsecase_ValidateCredentialBindingsTenant_MultipleBindings(t *testing.T) {
	tenantID := uuid.New()
	credID1 := uuid.New()
	credID2 := uuid.New()

	t.Run("multiple bindings - all valid", func(t *testing.T) {
		credRepo := newMockCredentialRepoForStep()
		cred1 := &domain.Credential{ID: credID1, Name: "API Key"}
		cred2 := &domain.Credential{ID: credID2, Name: "OAuth Token"}
		credRepo.addCredential(tenantID, cred1)
		credRepo.addCredential(tenantID, cred2)

		usecase := createTestStepUsecase(credRepo)
		bindings := json.RawMessage(`{"api_key": "` + credID1.String() + `", "oauth": "` + credID2.String() + `"}`)

		err := usecase.validateCredentialBindingsTenant(context.Background(), tenantID, bindings)
		if err != nil {
			t.Errorf("validateCredentialBindingsTenant() unexpected error = %v", err)
		}
	})

	t.Run("multiple bindings - one invalid", func(t *testing.T) {
		credRepo := newMockCredentialRepoForStep()
		cred1 := &domain.Credential{ID: credID1, Name: "API Key"}
		credRepo.addCredential(tenantID, cred1)
		// credID2 is not added

		usecase := createTestStepUsecase(credRepo)
		bindings := json.RawMessage(`{"api_key": "` + credID1.String() + `", "oauth": "` + credID2.String() + `"}`)

		err := usecase.validateCredentialBindingsTenant(context.Background(), tenantID, bindings)
		if err != domain.ErrCredentialNotFound {
			t.Errorf("validateCredentialBindingsTenant() error = %v, want ErrCredentialNotFound", err)
		}
	})
}
