package domain

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWebhook(t *testing.T) {
	tenantID := uuid.New()
	workflowID := uuid.New()
	version := 1
	name := "Test Webhook"
	inputMapping := json.RawMessage(`{"key": "value"}`)

	webhook, err := NewWebhook(tenantID, workflowID, version, name, inputMapping)
	require.NoError(t, err)
	require.NotNil(t, webhook)

	assert.NotEmpty(t, webhook.ID)
	assert.Equal(t, tenantID, webhook.TenantID)
	assert.Equal(t, workflowID, webhook.WorkflowID)
	assert.Equal(t, version, webhook.WorkflowVersion)
	assert.Equal(t, name, webhook.Name)
	assert.NotEmpty(t, webhook.Secret)
	assert.Len(t, webhook.Secret, 64) // 32 bytes = 64 hex characters
	assert.Equal(t, inputMapping, webhook.InputMapping)
	assert.True(t, webhook.Enabled)
	assert.Equal(t, 0, webhook.TriggerCount)
	assert.False(t, webhook.CreatedAt.IsZero())
	assert.False(t, webhook.UpdatedAt.IsZero())
}

func TestWebhook_RegenerateSecret(t *testing.T) {
	tenantID := uuid.New()
	workflowID := uuid.New()

	webhook, err := NewWebhook(tenantID, workflowID, 1, "Test", nil)
	require.NoError(t, err)

	originalSecret := webhook.Secret
	originalUpdatedAt := webhook.UpdatedAt

	err = webhook.RegenerateSecret()
	require.NoError(t, err)

	assert.NotEqual(t, originalSecret, webhook.Secret)
	assert.Len(t, webhook.Secret, 64)
	assert.True(t, webhook.UpdatedAt.After(originalUpdatedAt) || webhook.UpdatedAt.Equal(originalUpdatedAt))
}

func TestWebhook_EnableDisable(t *testing.T) {
	webhook, err := NewWebhook(uuid.New(), uuid.New(), 1, "Test", nil)
	require.NoError(t, err)

	assert.True(t, webhook.Enabled)

	webhook.Disable()
	assert.False(t, webhook.Enabled)

	webhook.Enable()
	assert.True(t, webhook.Enabled)
}

func TestWebhook_RecordTrigger(t *testing.T) {
	webhook, err := NewWebhook(uuid.New(), uuid.New(), 1, "Test", nil)
	require.NoError(t, err)

	assert.Nil(t, webhook.LastTriggeredAt)
	assert.Equal(t, 0, webhook.TriggerCount)

	webhook.RecordTrigger()
	assert.NotNil(t, webhook.LastTriggeredAt)
	assert.Equal(t, 1, webhook.TriggerCount)

	webhook.RecordTrigger()
	assert.Equal(t, 2, webhook.TriggerCount)
}

func TestWebhook_GetEndpointPath(t *testing.T) {
	webhook, err := NewWebhook(uuid.New(), uuid.New(), 1, "Test", nil)
	require.NoError(t, err)

	path := webhook.GetEndpointPath()
	assert.Contains(t, path, "/api/v1/webhooks/")
	assert.Contains(t, path, webhook.ID.String())
	assert.Contains(t, path, "/trigger")
}

func TestGenerateSecret_UniqueSecrets(t *testing.T) {
	// Generate multiple secrets and verify they are unique
	secrets := make(map[string]bool)
	for i := 0; i < 10; i++ {
		secret, err := generateSecret()
		require.NoError(t, err)
		assert.Len(t, secret, 64)

		// Verify uniqueness
		assert.False(t, secrets[secret], "duplicate secret generated")
		secrets[secret] = true
	}
}
