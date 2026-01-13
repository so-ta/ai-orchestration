package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestRun_Start(t *testing.T) {
	run := &Run{
		ID:     uuid.New(),
		Status: RunStatusPending,
	}

	run.Start()

	assert.Equal(t, RunStatusRunning, run.Status)
	assert.NotNil(t, run.StartedAt)
	assert.True(t, run.StartedAt.Before(time.Now().Add(time.Second)))
}

func TestRun_Complete(t *testing.T) {
	run := &Run{
		ID:     uuid.New(),
		Status: RunStatusRunning,
	}

	output := json.RawMessage(`{"result": "success"}`)
	run.Complete(output)

	assert.Equal(t, RunStatusCompleted, run.Status)
	assert.NotNil(t, run.CompletedAt)
	assert.Equal(t, output, run.Output)
}

func TestRun_Fail(t *testing.T) {
	run := &Run{
		ID:     uuid.New(),
		Status: RunStatusRunning,
	}

	run.Fail("something went wrong")

	assert.Equal(t, RunStatusFailed, run.Status)
	assert.NotNil(t, run.CompletedAt)
	assert.NotNil(t, run.Error)
	assert.Equal(t, "something went wrong", *run.Error)
}

func TestRun_Cancel(t *testing.T) {
	run := &Run{
		ID:     uuid.New(),
		Status: RunStatusRunning,
	}

	run.Cancel()

	assert.Equal(t, RunStatusCancelled, run.Status)
	assert.NotNil(t, run.CompletedAt)
}

func TestNewRun(t *testing.T) {
	tenantID := uuid.New()
	workflowID := uuid.New()
	input := json.RawMessage(`{"key": "value"}`)

	run := NewRun(tenantID, workflowID, 1, input, TriggerTypeTest)

	assert.NotEqual(t, uuid.Nil, run.ID)
	assert.Equal(t, workflowID, run.WorkflowID)
	assert.Equal(t, tenantID, run.TenantID)
	assert.Equal(t, 1, run.WorkflowVersion)
	assert.Equal(t, RunStatusPending, run.Status)
	assert.Equal(t, TriggerTypeTest, run.TriggeredBy)
	assert.Equal(t, input, run.Input)
	assert.False(t, run.CreatedAt.IsZero())
}

func TestRun_DurationMs(t *testing.T) {
	t.Run("returns nil when not started", func(t *testing.T) {
		run := &Run{Status: RunStatusPending}
		assert.Nil(t, run.DurationMs())
	})

	t.Run("returns nil when not completed", func(t *testing.T) {
		now := time.Now()
		run := &Run{
			Status:    RunStatusRunning,
			StartedAt: &now,
		}
		assert.Nil(t, run.DurationMs())
	})

	t.Run("returns duration when completed", func(t *testing.T) {
		start := time.Now()
		end := start.Add(500 * time.Millisecond)
		run := &Run{
			Status:      RunStatusCompleted,
			StartedAt:   &start,
			CompletedAt: &end,
		}
		duration := run.DurationMs()
		assert.NotNil(t, duration)
		assert.Equal(t, int64(500), *duration)
	})
}

func TestRunStatus_Values(t *testing.T) {
	assert.Equal(t, RunStatus("pending"), RunStatusPending)
	assert.Equal(t, RunStatus("running"), RunStatusRunning)
	assert.Equal(t, RunStatus("completed"), RunStatusCompleted)
	assert.Equal(t, RunStatus("failed"), RunStatusFailed)
	assert.Equal(t, RunStatus("cancelled"), RunStatusCancelled)
}

func TestTriggerType_Values(t *testing.T) {
	assert.Equal(t, TriggerType("manual"), TriggerTypeManual)
	assert.Equal(t, TriggerType("schedule"), TriggerTypeSchedule)
	assert.Equal(t, TriggerType("webhook"), TriggerTypeWebhook)
	assert.Equal(t, TriggerType("test"), TriggerTypeTest)
	assert.Equal(t, TriggerType("internal"), TriggerTypeInternal)
}
