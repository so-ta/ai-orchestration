package usecase

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock implementations for WebhookUsecase tests

type mockWebhookRepo struct {
	createFunc           func(ctx context.Context, webhook *domain.Webhook) error
	getByIDFunc          func(ctx context.Context, tenantID, id uuid.UUID) (*domain.Webhook, error)
	getByIDForTriggerFn  func(ctx context.Context, id uuid.UUID) (*domain.Webhook, error)
	listByTenantFn       func(ctx context.Context, tenantID uuid.UUID, filter repository.WebhookFilter) ([]*domain.Webhook, int, error)
	listByWorkflowFn     func(ctx context.Context, tenantID, workflowID uuid.UUID) ([]*domain.Webhook, error)
	updateFunc           func(ctx context.Context, webhook *domain.Webhook) error
	deleteFunc           func(ctx context.Context, tenantID, id uuid.UUID) error
}

func (m *mockWebhookRepo) Create(ctx context.Context, webhook *domain.Webhook) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, webhook)
	}
	return nil
}

func (m *mockWebhookRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Webhook, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, tenantID, id)
	}
	return nil, domain.ErrWebhookNotFound
}

func (m *mockWebhookRepo) GetByIDForTrigger(ctx context.Context, id uuid.UUID) (*domain.Webhook, error) {
	if m.getByIDForTriggerFn != nil {
		return m.getByIDForTriggerFn(ctx, id)
	}
	return nil, domain.ErrWebhookNotFound
}

func (m *mockWebhookRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID, filter repository.WebhookFilter) ([]*domain.Webhook, int, error) {
	if m.listByTenantFn != nil {
		return m.listByTenantFn(ctx, tenantID, filter)
	}
	return nil, 0, nil
}

func (m *mockWebhookRepo) ListByWorkflow(ctx context.Context, tenantID, workflowID uuid.UUID) ([]*domain.Webhook, error) {
	if m.listByWorkflowFn != nil {
		return m.listByWorkflowFn(ctx, tenantID, workflowID)
	}
	return nil, nil
}

func (m *mockWebhookRepo) Update(ctx context.Context, webhook *domain.Webhook) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, webhook)
	}
	return nil
}

func (m *mockWebhookRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, tenantID, id)
	}
	return nil
}

// Test helper to create WebhookUsecase with mocks
func newTestWebhookUsecase(
	webhookRepo *mockWebhookRepo,
	workflowRepo *mockWorkflowRepo,
	runRepo *mockRunRepo,
) *WebhookUsecase {
	return &WebhookUsecase{
		webhookRepo:  webhookRepo,
		workflowRepo: workflowRepo,
		runRepo:      runRepo,
	}
}

// Tests for Create

func TestWebhookUsecase_Create(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	workflowID := uuid.New()

	t.Run("success", func(t *testing.T) {
		workflowRepo := &mockWorkflowRepo{
			getByIDFunc: func(ctx context.Context, tid, id uuid.UUID) (*domain.Workflow, error) {
				return &domain.Workflow{
					ID:       workflowID,
					TenantID: tenantID,
					Status:   domain.WorkflowStatusPublished,
					Version:  1,
				}, nil
			},
		}
		webhookRepo := &mockWebhookRepo{
			createFunc: func(ctx context.Context, webhook *domain.Webhook) error {
				return nil
			},
		}

		uc := newTestWebhookUsecase(webhookRepo, workflowRepo, nil)

		input := CreateWebhookInput{
			TenantID:    tenantID,
			WorkflowID:  workflowID,
			Name:        "Test Webhook",
			Description: "Test Description",
		}

		webhook, err := uc.Create(ctx, input)
		require.NoError(t, err)
		assert.NotNil(t, webhook)
		assert.Equal(t, "Test Webhook", webhook.Name)
		assert.Equal(t, "Test Description", webhook.Description)
		assert.Equal(t, workflowID, webhook.WorkflowID)
		assert.Equal(t, 1, webhook.WorkflowVersion)
		assert.True(t, webhook.Enabled)
		assert.NotEmpty(t, webhook.Secret)
	})

	t.Run("empty name error", func(t *testing.T) {
		uc := newTestWebhookUsecase(nil, nil, nil)

		input := CreateWebhookInput{
			TenantID:   tenantID,
			WorkflowID: workflowID,
			Name:       "",
		}

		_, err := uc.Create(ctx, input)
		require.Error(t, err)
		var validationErr domain.ValidationError
		assert.True(t, errors.As(err, &validationErr))
		assert.Equal(t, "name", validationErr.Field)
	})

	t.Run("workflow not found", func(t *testing.T) {
		workflowRepo := &mockWorkflowRepo{
			getByIDFunc: func(ctx context.Context, tid, id uuid.UUID) (*domain.Workflow, error) {
				return nil, domain.ErrWorkflowNotFound
			},
		}

		uc := newTestWebhookUsecase(nil, workflowRepo, nil)

		input := CreateWebhookInput{
			TenantID:   tenantID,
			WorkflowID: workflowID,
			Name:       "Test Webhook",
		}

		_, err := uc.Create(ctx, input)
		assert.ErrorIs(t, err, domain.ErrWorkflowNotFound)
	})

	t.Run("workflow not published", func(t *testing.T) {
		workflowRepo := &mockWorkflowRepo{
			getByIDFunc: func(ctx context.Context, tid, id uuid.UUID) (*domain.Workflow, error) {
				return &domain.Workflow{
					ID:       workflowID,
					TenantID: tenantID,
					Status:   domain.WorkflowStatusDraft,
				}, nil
			},
		}

		uc := newTestWebhookUsecase(nil, workflowRepo, nil)

		input := CreateWebhookInput{
			TenantID:   tenantID,
			WorkflowID: workflowID,
			Name:       "Test Webhook",
		}

		_, err := uc.Create(ctx, input)
		require.Error(t, err)
		var validationErr domain.ValidationError
		assert.True(t, errors.As(err, &validationErr))
		assert.Equal(t, "workflow_id", validationErr.Field)
	})

	t.Run("repository create error", func(t *testing.T) {
		repoErr := errors.New("database error")
		workflowRepo := &mockWorkflowRepo{
			getByIDFunc: func(ctx context.Context, tid, id uuid.UUID) (*domain.Workflow, error) {
				return &domain.Workflow{
					ID:       workflowID,
					TenantID: tenantID,
					Status:   domain.WorkflowStatusPublished,
					Version:  1,
				}, nil
			},
		}
		webhookRepo := &mockWebhookRepo{
			createFunc: func(ctx context.Context, webhook *domain.Webhook) error {
				return repoErr
			},
		}

		uc := newTestWebhookUsecase(webhookRepo, workflowRepo, nil)

		input := CreateWebhookInput{
			TenantID:   tenantID,
			WorkflowID: workflowID,
			Name:       "Test Webhook",
		}

		_, err := uc.Create(ctx, input)
		assert.ErrorIs(t, err, repoErr)
	})
}

// Tests for GetByID

func TestWebhookUsecase_GetByID(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	webhookID := uuid.New()

	t.Run("success", func(t *testing.T) {
		expectedWebhook := &domain.Webhook{
			ID:       webhookID,
			TenantID: tenantID,
			Name:     "Test Webhook",
		}
		webhookRepo := &mockWebhookRepo{
			getByIDFunc: func(ctx context.Context, tid, id uuid.UUID) (*domain.Webhook, error) {
				if tid == tenantID && id == webhookID {
					return expectedWebhook, nil
				}
				return nil, domain.ErrWebhookNotFound
			},
		}

		uc := newTestWebhookUsecase(webhookRepo, nil, nil)

		webhook, err := uc.GetByID(ctx, tenantID, webhookID)
		require.NoError(t, err)
		assert.Equal(t, expectedWebhook, webhook)
	})

	t.Run("not found", func(t *testing.T) {
		webhookRepo := &mockWebhookRepo{
			getByIDFunc: func(ctx context.Context, tid, id uuid.UUID) (*domain.Webhook, error) {
				return nil, domain.ErrWebhookNotFound
			},
		}

		uc := newTestWebhookUsecase(webhookRepo, nil, nil)

		_, err := uc.GetByID(ctx, tenantID, uuid.New())
		assert.ErrorIs(t, err, domain.ErrWebhookNotFound)
	})
}

// Tests for List

func TestWebhookUsecase_List(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	workflowID := uuid.New()

	webhooks := []*domain.Webhook{
		{ID: uuid.New(), TenantID: tenantID, Name: "Webhook 1"},
		{ID: uuid.New(), TenantID: tenantID, Name: "Webhook 2"},
	}

	t.Run("default pagination", func(t *testing.T) {
		webhookRepo := &mockWebhookRepo{
			listByTenantFn: func(ctx context.Context, tid uuid.UUID, filter repository.WebhookFilter) ([]*domain.Webhook, int, error) {
				return webhooks, len(webhooks), nil
			},
		}

		uc := newTestWebhookUsecase(webhookRepo, nil, nil)

		input := ListWebhooksInput{
			TenantID: tenantID,
		}
		output, err := uc.List(ctx, input)
		require.NoError(t, err)
		assert.Len(t, output.Webhooks, 2)
		assert.Equal(t, 1, output.Page)
		assert.Equal(t, 20, output.Limit)
	})

	t.Run("custom pagination", func(t *testing.T) {
		webhookRepo := &mockWebhookRepo{
			listByTenantFn: func(ctx context.Context, tid uuid.UUID, filter repository.WebhookFilter) ([]*domain.Webhook, int, error) {
				assert.Equal(t, 2, filter.Page)
				assert.Equal(t, 10, filter.Limit)
				return webhooks, len(webhooks), nil
			},
		}

		uc := newTestWebhookUsecase(webhookRepo, nil, nil)

		input := ListWebhooksInput{
			TenantID: tenantID,
			Page:     2,
			Limit:    10,
		}
		output, err := uc.List(ctx, input)
		require.NoError(t, err)
		assert.Equal(t, 2, output.Page)
		assert.Equal(t, 10, output.Limit)
	})

	t.Run("with workflow filter", func(t *testing.T) {
		webhookRepo := &mockWebhookRepo{
			listByTenantFn: func(ctx context.Context, tid uuid.UUID, filter repository.WebhookFilter) ([]*domain.Webhook, int, error) {
				assert.NotNil(t, filter.WorkflowID)
				assert.Equal(t, workflowID, *filter.WorkflowID)
				return webhooks, len(webhooks), nil
			},
		}

		uc := newTestWebhookUsecase(webhookRepo, nil, nil)

		input := ListWebhooksInput{
			TenantID:   tenantID,
			WorkflowID: &workflowID,
		}
		output, err := uc.List(ctx, input)
		require.NoError(t, err)
		assert.Len(t, output.Webhooks, 2)
	})

	t.Run("limit capped at 100", func(t *testing.T) {
		webhookRepo := &mockWebhookRepo{
			listByTenantFn: func(ctx context.Context, tid uuid.UUID, filter repository.WebhookFilter) ([]*domain.Webhook, int, error) {
				return webhooks, len(webhooks), nil
			},
		}

		uc := newTestWebhookUsecase(webhookRepo, nil, nil)

		input := ListWebhooksInput{
			TenantID: tenantID,
			Limit:    500,
		}
		output, err := uc.List(ctx, input)
		require.NoError(t, err)
		assert.Equal(t, 20, output.Limit) // Should be capped to default 20
	})
}

// Tests for Update

func TestWebhookUsecase_Update(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	webhookID := uuid.New()

	t.Run("success", func(t *testing.T) {
		originalWebhook := &domain.Webhook{
			ID:          webhookID,
			TenantID:    tenantID,
			Name:        "Original Name",
			Description: "Original Description",
		}
		webhookRepo := &mockWebhookRepo{
			getByIDFunc: func(ctx context.Context, tid, id uuid.UUID) (*domain.Webhook, error) {
				return originalWebhook, nil
			},
			updateFunc: func(ctx context.Context, webhook *domain.Webhook) error {
				return nil
			},
		}

		uc := newTestWebhookUsecase(webhookRepo, nil, nil)

		input := UpdateWebhookInput{
			TenantID:    tenantID,
			ID:          webhookID,
			Name:        "Updated Name",
			Description: "Updated Description",
		}
		webhook, err := uc.Update(ctx, input)
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", webhook.Name)
		assert.Equal(t, "Updated Description", webhook.Description)
	})

	t.Run("not found", func(t *testing.T) {
		webhookRepo := &mockWebhookRepo{
			getByIDFunc: func(ctx context.Context, tid, id uuid.UUID) (*domain.Webhook, error) {
				return nil, domain.ErrWebhookNotFound
			},
		}

		uc := newTestWebhookUsecase(webhookRepo, nil, nil)

		input := UpdateWebhookInput{
			TenantID: tenantID,
			ID:       uuid.New(),
			Name:     "Updated Name",
		}
		_, err := uc.Update(ctx, input)
		assert.ErrorIs(t, err, domain.ErrWebhookNotFound)
	})

	t.Run("update error", func(t *testing.T) {
		repoErr := errors.New("update error")
		webhookRepo := &mockWebhookRepo{
			getByIDFunc: func(ctx context.Context, tid, id uuid.UUID) (*domain.Webhook, error) {
				return &domain.Webhook{ID: webhookID, TenantID: tenantID}, nil
			},
			updateFunc: func(ctx context.Context, webhook *domain.Webhook) error {
				return repoErr
			},
		}

		uc := newTestWebhookUsecase(webhookRepo, nil, nil)

		input := UpdateWebhookInput{
			TenantID: tenantID,
			ID:       webhookID,
			Name:     "Updated Name",
		}
		_, err := uc.Update(ctx, input)
		assert.ErrorIs(t, err, repoErr)
	})
}

// Tests for Delete

func TestWebhookUsecase_Delete(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	webhookID := uuid.New()

	t.Run("success", func(t *testing.T) {
		webhookRepo := &mockWebhookRepo{
			deleteFunc: func(ctx context.Context, tid, id uuid.UUID) error {
				return nil
			},
		}

		uc := newTestWebhookUsecase(webhookRepo, nil, nil)

		err := uc.Delete(ctx, tenantID, webhookID)
		require.NoError(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		webhookRepo := &mockWebhookRepo{
			deleteFunc: func(ctx context.Context, tid, id uuid.UUID) error {
				return domain.ErrWebhookNotFound
			},
		}

		uc := newTestWebhookUsecase(webhookRepo, nil, nil)

		err := uc.Delete(ctx, tenantID, uuid.New())
		assert.ErrorIs(t, err, domain.ErrWebhookNotFound)
	})
}

// Tests for Enable

func TestWebhookUsecase_Enable(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	webhookID := uuid.New()

	t.Run("success", func(t *testing.T) {
		webhook := &domain.Webhook{
			ID:       webhookID,
			TenantID: tenantID,
			Enabled:  false,
		}
		webhookRepo := &mockWebhookRepo{
			getByIDFunc: func(ctx context.Context, tid, id uuid.UUID) (*domain.Webhook, error) {
				return webhook, nil
			},
			updateFunc: func(ctx context.Context, w *domain.Webhook) error {
				return nil
			},
		}

		uc := newTestWebhookUsecase(webhookRepo, nil, nil)

		result, err := uc.Enable(ctx, tenantID, webhookID)
		require.NoError(t, err)
		assert.True(t, result.Enabled)
	})

	t.Run("not found", func(t *testing.T) {
		webhookRepo := &mockWebhookRepo{
			getByIDFunc: func(ctx context.Context, tid, id uuid.UUID) (*domain.Webhook, error) {
				return nil, domain.ErrWebhookNotFound
			},
		}

		uc := newTestWebhookUsecase(webhookRepo, nil, nil)

		_, err := uc.Enable(ctx, tenantID, uuid.New())
		assert.ErrorIs(t, err, domain.ErrWebhookNotFound)
	})
}

// Tests for Disable

func TestWebhookUsecase_Disable(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	webhookID := uuid.New()

	t.Run("success", func(t *testing.T) {
		webhook := &domain.Webhook{
			ID:       webhookID,
			TenantID: tenantID,
			Enabled:  true,
		}
		webhookRepo := &mockWebhookRepo{
			getByIDFunc: func(ctx context.Context, tid, id uuid.UUID) (*domain.Webhook, error) {
				return webhook, nil
			},
			updateFunc: func(ctx context.Context, w *domain.Webhook) error {
				return nil
			},
		}

		uc := newTestWebhookUsecase(webhookRepo, nil, nil)

		result, err := uc.Disable(ctx, tenantID, webhookID)
		require.NoError(t, err)
		assert.False(t, result.Enabled)
	})

	t.Run("not found", func(t *testing.T) {
		webhookRepo := &mockWebhookRepo{
			getByIDFunc: func(ctx context.Context, tid, id uuid.UUID) (*domain.Webhook, error) {
				return nil, domain.ErrWebhookNotFound
			},
		}

		uc := newTestWebhookUsecase(webhookRepo, nil, nil)

		_, err := uc.Disable(ctx, tenantID, uuid.New())
		assert.ErrorIs(t, err, domain.ErrWebhookNotFound)
	})
}

// Tests for RegenerateSecret

func TestWebhookUsecase_RegenerateSecret(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	webhookID := uuid.New()

	t.Run("success", func(t *testing.T) {
		originalSecret := "original-secret"
		webhook := &domain.Webhook{
			ID:       webhookID,
			TenantID: tenantID,
			Secret:   originalSecret,
		}
		webhookRepo := &mockWebhookRepo{
			getByIDFunc: func(ctx context.Context, tid, id uuid.UUID) (*domain.Webhook, error) {
				return webhook, nil
			},
			updateFunc: func(ctx context.Context, w *domain.Webhook) error {
				return nil
			},
		}

		uc := newTestWebhookUsecase(webhookRepo, nil, nil)

		result, err := uc.RegenerateSecret(ctx, tenantID, webhookID)
		require.NoError(t, err)
		assert.NotEqual(t, originalSecret, result.Secret)
		assert.NotEmpty(t, result.Secret)
	})

	t.Run("not found", func(t *testing.T) {
		webhookRepo := &mockWebhookRepo{
			getByIDFunc: func(ctx context.Context, tid, id uuid.UUID) (*domain.Webhook, error) {
				return nil, domain.ErrWebhookNotFound
			},
		}

		uc := newTestWebhookUsecase(webhookRepo, nil, nil)

		_, err := uc.RegenerateSecret(ctx, tenantID, uuid.New())
		assert.ErrorIs(t, err, domain.ErrWebhookNotFound)
	})

	t.Run("update error", func(t *testing.T) {
		repoErr := errors.New("update error")
		webhookRepo := &mockWebhookRepo{
			getByIDFunc: func(ctx context.Context, tid, id uuid.UUID) (*domain.Webhook, error) {
				return &domain.Webhook{ID: webhookID, TenantID: tenantID, Secret: "secret"}, nil
			},
			updateFunc: func(ctx context.Context, w *domain.Webhook) error {
				return repoErr
			},
		}

		uc := newTestWebhookUsecase(webhookRepo, nil, nil)

		_, err := uc.RegenerateSecret(ctx, tenantID, webhookID)
		assert.ErrorIs(t, err, repoErr)
	})
}

// Tests for Trigger

func TestWebhookUsecase_Trigger(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	webhookID := uuid.New()
	workflowID := uuid.New()
	secret := "test-secret"

	// Helper to generate valid signature
	generateSignature := func(payload []byte, secret string) string {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(payload)
		return hex.EncodeToString(mac.Sum(nil))
	}

	t.Run("success", func(t *testing.T) {
		payload := json.RawMessage(`{"key": "value"}`)
		signature := generateSignature(payload, secret)

		webhook := &domain.Webhook{
			ID:              webhookID,
			TenantID:        tenantID,
			WorkflowID:      workflowID,
			WorkflowVersion: 1,
			Secret:          secret,
			Enabled:         true,
		}
		webhookRepo := &mockWebhookRepo{
			getByIDForTriggerFn: func(ctx context.Context, id uuid.UUID) (*domain.Webhook, error) {
				return webhook, nil
			},
			updateFunc: func(ctx context.Context, w *domain.Webhook) error {
				return nil
			},
		}
		runRepo := &mockRunRepo{
			createFunc: func(ctx context.Context, run *domain.Run) error {
				return nil
			},
		}

		uc := newTestWebhookUsecase(webhookRepo, nil, runRepo)

		input := TriggerWebhookInput{
			WebhookID: webhookID,
			Signature: signature,
			Payload:   payload,
		}

		run, err := uc.Trigger(ctx, input)
		require.NoError(t, err)
		assert.NotNil(t, run)
		assert.Equal(t, workflowID, run.WorkflowID)
		assert.Equal(t, domain.TriggerTypeWebhook, run.TriggeredBy)
	})

	t.Run("webhook not found", func(t *testing.T) {
		webhookRepo := &mockWebhookRepo{
			getByIDForTriggerFn: func(ctx context.Context, id uuid.UUID) (*domain.Webhook, error) {
				return nil, domain.ErrWebhookNotFound
			},
		}

		uc := newTestWebhookUsecase(webhookRepo, nil, nil)

		input := TriggerWebhookInput{
			WebhookID: uuid.New(),
			Signature: "any",
			Payload:   json.RawMessage(`{}`),
		}

		_, err := uc.Trigger(ctx, input)
		assert.ErrorIs(t, err, domain.ErrWebhookNotFound)
	})

	t.Run("webhook disabled", func(t *testing.T) {
		webhook := &domain.Webhook{
			ID:       webhookID,
			TenantID: tenantID,
			Secret:   secret,
			Enabled:  false,
		}
		webhookRepo := &mockWebhookRepo{
			getByIDForTriggerFn: func(ctx context.Context, id uuid.UUID) (*domain.Webhook, error) {
				return webhook, nil
			},
		}

		uc := newTestWebhookUsecase(webhookRepo, nil, nil)

		input := TriggerWebhookInput{
			WebhookID: webhookID,
			Signature: "any",
			Payload:   json.RawMessage(`{}`),
		}

		_, err := uc.Trigger(ctx, input)
		assert.ErrorIs(t, err, domain.ErrWebhookDisabled)
	})

	t.Run("invalid signature", func(t *testing.T) {
		webhook := &domain.Webhook{
			ID:       webhookID,
			TenantID: tenantID,
			Secret:   secret,
			Enabled:  true,
		}
		webhookRepo := &mockWebhookRepo{
			getByIDForTriggerFn: func(ctx context.Context, id uuid.UUID) (*domain.Webhook, error) {
				return webhook, nil
			},
		}

		uc := newTestWebhookUsecase(webhookRepo, nil, nil)

		input := TriggerWebhookInput{
			WebhookID: webhookID,
			Signature: "invalid-signature",
			Payload:   json.RawMessage(`{}`),
		}

		_, err := uc.Trigger(ctx, input)
		assert.ErrorIs(t, err, domain.ErrWebhookInvalidSecret)
	})

	t.Run("empty signature", func(t *testing.T) {
		webhook := &domain.Webhook{
			ID:       webhookID,
			TenantID: tenantID,
			Secret:   secret,
			Enabled:  true,
		}
		webhookRepo := &mockWebhookRepo{
			getByIDForTriggerFn: func(ctx context.Context, id uuid.UUID) (*domain.Webhook, error) {
				return webhook, nil
			},
		}

		uc := newTestWebhookUsecase(webhookRepo, nil, nil)

		input := TriggerWebhookInput{
			WebhookID: webhookID,
			Signature: "",
			Payload:   json.RawMessage(`{}`),
		}

		_, err := uc.Trigger(ctx, input)
		assert.ErrorIs(t, err, domain.ErrWebhookInvalidSecret)
	})

	t.Run("run creation error", func(t *testing.T) {
		payload := json.RawMessage(`{"key": "value"}`)
		signature := generateSignature(payload, secret)
		repoErr := errors.New("run creation error")

		webhook := &domain.Webhook{
			ID:              webhookID,
			TenantID:        tenantID,
			WorkflowID:      workflowID,
			WorkflowVersion: 1,
			Secret:          secret,
			Enabled:         true,
		}
		webhookRepo := &mockWebhookRepo{
			getByIDForTriggerFn: func(ctx context.Context, id uuid.UUID) (*domain.Webhook, error) {
				return webhook, nil
			},
		}
		runRepo := &mockRunRepo{
			createFunc: func(ctx context.Context, run *domain.Run) error {
				return repoErr
			},
		}

		uc := newTestWebhookUsecase(webhookRepo, nil, runRepo)

		input := TriggerWebhookInput{
			WebhookID: webhookID,
			Signature: signature,
			Payload:   payload,
		}

		_, err := uc.Trigger(ctx, input)
		assert.ErrorIs(t, err, repoErr)
	})
}

// Tests for verifySignature

func TestWebhookUsecase_verifySignature(t *testing.T) {
	uc := &WebhookUsecase{}
	secret := "test-secret"
	payload := json.RawMessage(`{"key": "value"}`)

	// Generate valid signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	validSignature := hex.EncodeToString(mac.Sum(nil))

	t.Run("valid signature", func(t *testing.T) {
		result := uc.verifySignature(secret, payload, validSignature)
		assert.True(t, result)
	})

	t.Run("invalid signature", func(t *testing.T) {
		result := uc.verifySignature(secret, payload, "invalid-signature")
		assert.False(t, result)
	})

	t.Run("empty signature", func(t *testing.T) {
		result := uc.verifySignature(secret, payload, "")
		assert.False(t, result)
	})

	t.Run("wrong secret", func(t *testing.T) {
		result := uc.verifySignature("wrong-secret", payload, validSignature)
		assert.False(t, result)
	})

	t.Run("different payload", func(t *testing.T) {
		differentPayload := json.RawMessage(`{"different": "payload"}`)
		result := uc.verifySignature(secret, differentPayload, validSignature)
		assert.False(t, result)
	})
}

// Tests for applyInputMapping

func TestWebhookUsecase_applyInputMapping(t *testing.T) {
	uc := &WebhookUsecase{}

	t.Run("simple mapping", func(t *testing.T) {
		payload := json.RawMessage(`{"action": "opened", "repository": {"name": "test-repo"}}`)
		mapping := json.RawMessage(`{"event_type": "$.action", "repo_name": "$.repository.name"}`)

		result, err := uc.applyInputMapping(payload, mapping)
		require.NoError(t, err)

		var resultMap map[string]interface{}
		err = json.Unmarshal(result, &resultMap)
		require.NoError(t, err)

		assert.Equal(t, "opened", resultMap["event_type"])
		assert.Equal(t, "test-repo", resultMap["repo_name"])
	})

	t.Run("root reference", func(t *testing.T) {
		payload := json.RawMessage(`{"key": "value"}`)
		mapping := json.RawMessage(`{"all": "$"}`)

		result, err := uc.applyInputMapping(payload, mapping)
		require.NoError(t, err)

		var resultMap map[string]interface{}
		err = json.Unmarshal(result, &resultMap)
		require.NoError(t, err)

		allData, ok := resultMap["all"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "value", allData["key"])
	})

	t.Run("missing field skipped", func(t *testing.T) {
		payload := json.RawMessage(`{"existing": "value"}`)
		mapping := json.RawMessage(`{"found": "$.existing", "not_found": "$.missing.field"}`)

		result, err := uc.applyInputMapping(payload, mapping)
		require.NoError(t, err)

		var resultMap map[string]interface{}
		err = json.Unmarshal(result, &resultMap)
		require.NoError(t, err)

		assert.Equal(t, "value", resultMap["found"])
		_, exists := resultMap["not_found"]
		assert.False(t, exists)
	})

	t.Run("nested path", func(t *testing.T) {
		payload := json.RawMessage(`{"level1": {"level2": {"level3": "deep_value"}}}`)
		mapping := json.RawMessage(`{"deep": "$.level1.level2.level3"}`)

		result, err := uc.applyInputMapping(payload, mapping)
		require.NoError(t, err)

		var resultMap map[string]interface{}
		err = json.Unmarshal(result, &resultMap)
		require.NoError(t, err)

		assert.Equal(t, "deep_value", resultMap["deep"])
	})

	t.Run("invalid payload json", func(t *testing.T) {
		payload := json.RawMessage(`{invalid json}`)
		mapping := json.RawMessage(`{"field": "$.value"}`)

		_, err := uc.applyInputMapping(payload, mapping)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse payload")
	})

	t.Run("invalid mapping json", func(t *testing.T) {
		payload := json.RawMessage(`{"key": "value"}`)
		mapping := json.RawMessage(`{invalid mapping}`)

		_, err := uc.applyInputMapping(payload, mapping)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse input mapping")
	})

	t.Run("empty mapping", func(t *testing.T) {
		payload := json.RawMessage(`{"key": "value"}`)
		mapping := json.RawMessage(`{}`)

		result, err := uc.applyInputMapping(payload, mapping)
		require.NoError(t, err)

		var resultMap map[string]interface{}
		err = json.Unmarshal(result, &resultMap)
		require.NoError(t, err)

		assert.Empty(t, resultMap)
	})
}

// Tests for resolvePath

func TestWebhookUsecase_resolvePath(t *testing.T) {
	uc := &WebhookUsecase{}
	data := map[string]interface{}{
		"simple": "value",
		"nested": map[string]interface{}{
			"field": "nested_value",
			"deep": map[string]interface{}{
				"value": "deep_value",
			},
		},
	}

	t.Run("simple field", func(t *testing.T) {
		result, err := uc.resolvePath("$.simple", data)
		require.NoError(t, err)
		assert.Equal(t, "value", result)
	})

	t.Run("nested field", func(t *testing.T) {
		result, err := uc.resolvePath("$.nested.field", data)
		require.NoError(t, err)
		assert.Equal(t, "nested_value", result)
	})

	t.Run("deep nested field", func(t *testing.T) {
		result, err := uc.resolvePath("$.nested.deep.value", data)
		require.NoError(t, err)
		assert.Equal(t, "deep_value", result)
	})

	t.Run("root reference", func(t *testing.T) {
		result, err := uc.resolvePath("$", data)
		require.NoError(t, err)
		assert.Equal(t, data, result)
	})

	t.Run("field not found", func(t *testing.T) {
		_, err := uc.resolvePath("$.nonexistent", data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "field not found")
	})

	t.Run("access on non-object", func(t *testing.T) {
		_, err := uc.resolvePath("$.simple.invalid", data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot access")
	})
}

// Tests for Trigger with input mapping

func TestWebhookUsecase_Trigger_WithInputMapping(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	webhookID := uuid.New()
	workflowID := uuid.New()
	secret := "test-secret"

	generateSignature := func(payload []byte, secret string) string {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(payload)
		return hex.EncodeToString(mac.Sum(nil))
	}

	t.Run("with input mapping", func(t *testing.T) {
		payload := json.RawMessage(`{"action": "opened", "repository": {"name": "test-repo"}}`)
		signature := generateSignature(payload, secret)
		inputMapping := json.RawMessage(`{"event_type": "$.action", "repo_name": "$.repository.name"}`)

		var capturedInput json.RawMessage
		webhook := &domain.Webhook{
			ID:              webhookID,
			TenantID:        tenantID,
			WorkflowID:      workflowID,
			WorkflowVersion: 1,
			Secret:          secret,
			Enabled:         true,
			InputMapping:    inputMapping,
		}
		webhookRepo := &mockWebhookRepo{
			getByIDForTriggerFn: func(ctx context.Context, id uuid.UUID) (*domain.Webhook, error) {
				return webhook, nil
			},
			updateFunc: func(ctx context.Context, w *domain.Webhook) error {
				return nil
			},
		}
		runRepo := &mockRunRepo{
			createFunc: func(ctx context.Context, run *domain.Run) error {
				capturedInput = run.Input
				return nil
			},
		}

		uc := newTestWebhookUsecase(webhookRepo, nil, runRepo)

		input := TriggerWebhookInput{
			WebhookID: webhookID,
			Signature: signature,
			Payload:   payload,
		}

		run, err := uc.Trigger(ctx, input)
		require.NoError(t, err)
		assert.NotNil(t, run)

		// Verify the input was transformed
		var resultMap map[string]interface{}
		err = json.Unmarshal(capturedInput, &resultMap)
		require.NoError(t, err)
		assert.Equal(t, "opened", resultMap["event_type"])
		assert.Equal(t, "test-repo", resultMap["repo_name"])
	})

	t.Run("without input mapping uses raw payload", func(t *testing.T) {
		payload := json.RawMessage(`{"action": "opened"}`)
		signature := generateSignature(payload, secret)

		var capturedInput json.RawMessage
		webhook := &domain.Webhook{
			ID:              webhookID,
			TenantID:        tenantID,
			WorkflowID:      workflowID,
			WorkflowVersion: 1,
			Secret:          secret,
			Enabled:         true,
			InputMapping:    nil,
		}
		webhookRepo := &mockWebhookRepo{
			getByIDForTriggerFn: func(ctx context.Context, id uuid.UUID) (*domain.Webhook, error) {
				return webhook, nil
			},
			updateFunc: func(ctx context.Context, w *domain.Webhook) error {
				return nil
			},
		}
		runRepo := &mockRunRepo{
			createFunc: func(ctx context.Context, run *domain.Run) error {
				capturedInput = run.Input
				return nil
			},
		}

		uc := newTestWebhookUsecase(webhookRepo, nil, runRepo)

		input := TriggerWebhookInput{
			WebhookID: webhookID,
			Signature: signature,
			Payload:   payload,
		}

		run, err := uc.Trigger(ctx, input)
		require.NoError(t, err)
		assert.NotNil(t, run)

		// Verify the raw payload was used
		assert.JSONEq(t, string(payload), string(capturedInput))
	})

	t.Run("empty input mapping uses raw payload", func(t *testing.T) {
		payload := json.RawMessage(`{"action": "opened", "data": "test"}`)
		signature := generateSignature(payload, secret)

		var capturedInput json.RawMessage
		webhook := &domain.Webhook{
			ID:              webhookID,
			TenantID:        tenantID,
			WorkflowID:      workflowID,
			WorkflowVersion: 1,
			Secret:          secret,
			Enabled:         true,
			InputMapping:    json.RawMessage(`{}`), // empty mapping
		}
		webhookRepo := &mockWebhookRepo{
			getByIDForTriggerFn: func(ctx context.Context, id uuid.UUID) (*domain.Webhook, error) {
				return webhook, nil
			},
			updateFunc: func(ctx context.Context, w *domain.Webhook) error {
				return nil
			},
		}
		runRepo := &mockRunRepo{
			createFunc: func(ctx context.Context, run *domain.Run) error {
				capturedInput = run.Input
				return nil
			},
		}

		uc := newTestWebhookUsecase(webhookRepo, nil, runRepo)

		input := TriggerWebhookInput{
			WebhookID: webhookID,
			Signature: signature,
			Payload:   payload,
		}

		run, err := uc.Trigger(ctx, input)
		require.NoError(t, err)
		assert.NotNil(t, run)

		// Verify the raw payload was used (not transformed to {})
		assert.JSONEq(t, string(payload), string(capturedInput))
	})
}

// Tests for hasValidInputMapping

func TestWebhookUsecase_hasValidInputMapping(t *testing.T) {
	uc := &WebhookUsecase{}

	t.Run("nil mapping", func(t *testing.T) {
		result := uc.hasValidInputMapping(nil)
		assert.False(t, result)
	})

	t.Run("empty bytes", func(t *testing.T) {
		result := uc.hasValidInputMapping(json.RawMessage{})
		assert.False(t, result)
	})

	t.Run("empty json object", func(t *testing.T) {
		result := uc.hasValidInputMapping(json.RawMessage(`{}`))
		assert.False(t, result)
	})

	t.Run("invalid json", func(t *testing.T) {
		result := uc.hasValidInputMapping(json.RawMessage(`{invalid}`))
		assert.False(t, result)
	})

	t.Run("valid mapping", func(t *testing.T) {
		result := uc.hasValidInputMapping(json.RawMessage(`{"field": "$.value"}`))
		assert.True(t, result)
	})
}

// Tests for validateWorkflowInput
// Note: Uses mockStepRepo from run_test.go

func TestWebhookUsecase_validateWorkflowInput(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	workflowID := uuid.New()

	t.Run("no stepRepo skips validation", func(t *testing.T) {
		uc := &WebhookUsecase{
			stepRepo: nil,
		}

		err := uc.validateWorkflowInput(ctx, tenantID, workflowID, json.RawMessage(`{}`))
		assert.NoError(t, err)
	})

	t.Run("stepRepo error skips validation", func(t *testing.T) {
		stepRepo := &mockStepRepo{
			listByWorkflowFn: func(ctx context.Context, tid, wid uuid.UUID) ([]*domain.Step, error) {
				return nil, errors.New("database error")
			},
		}
		uc := &WebhookUsecase{
			stepRepo: stepRepo,
		}

		err := uc.validateWorkflowInput(ctx, tenantID, workflowID, json.RawMessage(`{}`))
		assert.NoError(t, err)
	})

	t.Run("no start step skips validation", func(t *testing.T) {
		stepRepo := &mockStepRepo{
			listByWorkflowFn: func(ctx context.Context, tid, wid uuid.UUID) ([]*domain.Step, error) {
				return []*domain.Step{
					{ID: uuid.New(), Type: "llm"},
					{ID: uuid.New(), Type: "tool"},
				}, nil
			},
		}
		uc := &WebhookUsecase{
			stepRepo: stepRepo,
		}

		err := uc.validateWorkflowInput(ctx, tenantID, workflowID, json.RawMessage(`{}`))
		assert.NoError(t, err)
	})

	t.Run("start step without input_schema skips validation", func(t *testing.T) {
		stepRepo := &mockStepRepo{
			listByWorkflowFn: func(ctx context.Context, tid, wid uuid.UUID) ([]*domain.Step, error) {
				return []*domain.Step{
					{ID: uuid.New(), Type: "start", Config: json.RawMessage(`{}`)},
				}, nil
			},
		}
		uc := &WebhookUsecase{
			stepRepo: stepRepo,
		}

		err := uc.validateWorkflowInput(ctx, tenantID, workflowID, json.RawMessage(`{}`))
		assert.NoError(t, err)
	})

	t.Run("start step with empty config skips validation", func(t *testing.T) {
		stepRepo := &mockStepRepo{
			listByWorkflowFn: func(ctx context.Context, tid, wid uuid.UUID) ([]*domain.Step, error) {
				return []*domain.Step{
					{ID: uuid.New(), Type: "start", Config: nil},
				}, nil
			},
		}
		uc := &WebhookUsecase{
			stepRepo: stepRepo,
		}

		err := uc.validateWorkflowInput(ctx, tenantID, workflowID, json.RawMessage(`{}`))
		assert.NoError(t, err)
	})

	t.Run("valid input passes validation", func(t *testing.T) {
		inputSchema := `{
			"type": "object",
			"properties": {
				"name": {"type": "string"},
				"age": {"type": "number"}
			},
			"required": ["name"]
		}`
		stepRepo := &mockStepRepo{
			listByWorkflowFn: func(ctx context.Context, tid, wid uuid.UUID) ([]*domain.Step, error) {
				return []*domain.Step{
					{ID: uuid.New(), Type: "start", Config: json.RawMessage(`{"input_schema": ` + inputSchema + `}`)},
				}, nil
			},
		}
		uc := &WebhookUsecase{
			stepRepo: stepRepo,
		}

		input := json.RawMessage(`{"name": "John", "age": 30}`)
		err := uc.validateWorkflowInput(ctx, tenantID, workflowID, input)
		assert.NoError(t, err)
	})

	t.Run("invalid input fails validation - missing required field", func(t *testing.T) {
		inputSchema := `{
			"type": "object",
			"properties": {
				"name": {"type": "string"}
			},
			"required": ["name"]
		}`
		stepRepo := &mockStepRepo{
			listByWorkflowFn: func(ctx context.Context, tid, wid uuid.UUID) ([]*domain.Step, error) {
				return []*domain.Step{
					{ID: uuid.New(), Type: "start", Config: json.RawMessage(`{"input_schema": ` + inputSchema + `}`)},
				}, nil
			},
		}
		uc := &WebhookUsecase{
			stepRepo: stepRepo,
		}

		input := json.RawMessage(`{}`)
		err := uc.validateWorkflowInput(ctx, tenantID, workflowID, input)
		assert.Error(t, err)
	})
}

// Tests for extractInputSchemaFromStepConfig

func TestExtractInputSchemaFromStepConfig(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		result := extractInputSchemaFromStepConfig(nil)
		assert.Nil(t, result)
	})

	t.Run("empty config", func(t *testing.T) {
		result := extractInputSchemaFromStepConfig(json.RawMessage{})
		assert.Nil(t, result)
	})

	t.Run("invalid json", func(t *testing.T) {
		result := extractInputSchemaFromStepConfig(json.RawMessage(`{invalid}`))
		assert.Nil(t, result)
	})

	t.Run("no input_schema key", func(t *testing.T) {
		result := extractInputSchemaFromStepConfig(json.RawMessage(`{"other": "value"}`))
		assert.Nil(t, result)
	})

	t.Run("valid input_schema", func(t *testing.T) {
		config := json.RawMessage(`{"input_schema": {"type": "object"}}`)
		result := extractInputSchemaFromStepConfig(config)
		assert.NotNil(t, result)
		assert.JSONEq(t, `{"type": "object"}`, string(result))
	})
}
