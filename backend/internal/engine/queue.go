package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	jobQueueKey     = "aio:jobs:pending"
	jobDataKeyPrefix = "aio:jobs:data:"
)

// ExecutionMode represents the type of execution
type ExecutionMode string

const (
	// ExecutionModeFull is the default full project execution
	ExecutionModeFull ExecutionMode = "full"
	// ExecutionModeSingleStep executes only a single step
	ExecutionModeSingleStep ExecutionMode = "single_step"
	// ExecutionModeResume resumes execution from a specific step
	ExecutionModeResume ExecutionMode = "resume"
)

// Job represents a project execution job
type Job struct {
	ID             string          `json:"id"`
	TenantID       uuid.UUID       `json:"tenant_id"`
	ProjectID      uuid.UUID       `json:"project_id"`
	ProjectVersion int             `json:"project_version"`
	RunID          uuid.UUID       `json:"run_id"`
	Input          json.RawMessage `json:"input"`
	CreatedAt      time.Time       `json:"created_at"`

	// For system projects, the project's tenant_id may differ from the run's tenant_id
	// This allows the worker to fetch the project using the correct tenant
	ProjectTenantID *uuid.UUID `json:"project_tenant_id,omitempty"`

	// Partial execution fields
	ExecutionMode   ExecutionMode              `json:"execution_mode,omitempty"`   // "full", "single_step", "resume"
	TargetStepID    *uuid.UUID                 `json:"target_step_id,omitempty"`   // Target step for single_step/resume
	StepInput       json.RawMessage            `json:"step_input,omitempty"`       // Custom input for the target step
	InjectedOutputs map[string]json.RawMessage `json:"injected_outputs,omitempty"` // Previous step outputs to inject
}

// Queue manages the job queue
type Queue struct {
	client *redis.Client
}

// NewQueue creates a new job queue
func NewQueue(client *redis.Client) *Queue {
	return &Queue{client: client}
}

// Enqueue adds a job to the queue
func (q *Queue) Enqueue(ctx context.Context, job *Job) error {
	job.ID = uuid.New().String()
	job.CreatedAt = time.Now().UTC()

	slog.Info("Enqueuing job", "job_id", job.ID, "run_id", job.RunID, "project_id", job.ProjectID)

	// Store job data
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	dataKey := jobDataKeyPrefix + job.ID
	if err := q.client.Set(ctx, dataKey, data, 24*time.Hour).Err(); err != nil {
		slog.Error("Failed to store job data", "error", err)
		return fmt.Errorf("failed to store job data: %w", err)
	}

	// Add to queue
	if err := q.client.LPush(ctx, jobQueueKey, job.ID).Err(); err != nil {
		slog.Error("Failed to enqueue job", "error", err)
		return fmt.Errorf("failed to enqueue job: %w", err)
	}

	slog.Info("Job enqueued successfully", "job_id", job.ID, "queue_key", jobQueueKey)
	return nil
}

// Dequeue retrieves a job from the queue (blocking)
func (q *Queue) Dequeue(ctx context.Context, timeout time.Duration) (*Job, error) {
	result, err := q.client.BRPop(ctx, timeout, jobQueueKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // timeout, no job
		}
		return nil, fmt.Errorf("failed to dequeue job: %w", err)
	}

	if len(result) < 2 {
		return nil, nil
	}

	jobID := result[1]

	// Get job data
	dataKey := jobDataKeyPrefix + jobID
	data, err := q.client.Get(ctx, dataKey).Bytes()
	if err != nil {
		return nil, fmt.Errorf("failed to get job data for job %s: %w", jobID, err)
	}

	var job Job
	if err := json.Unmarshal(data, &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job %s: %w", jobID, err)
	}

	// Delete job data after dequeue (best effort, log errors)
	if err := q.client.Del(ctx, dataKey).Err(); err != nil {
		slog.Warn("Failed to delete job data from Redis",
			"job_id", jobID,
			"key", dataKey,
			"error", err,
		)
	}

	return &job, nil
}

// Length returns the number of pending jobs
func (q *Queue) Length(ctx context.Context) (int64, error) {
	return q.client.LLen(ctx, jobQueueKey).Result()
}
