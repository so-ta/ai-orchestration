package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	jobQueueKey     = "aio:jobs:pending"
	jobDataKeyPrefix = "aio:jobs:data:"
)

// Job represents a workflow execution job
type Job struct {
	ID              string    `json:"id"`
	TenantID        uuid.UUID `json:"tenant_id"`
	WorkflowID      uuid.UUID `json:"workflow_id"`
	WorkflowVersion int       `json:"workflow_version"`
	RunID           uuid.UUID `json:"run_id"`
	Input           json.RawMessage `json:"input"`
	CreatedAt       time.Time `json:"created_at"`
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

	// Store job data
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	dataKey := jobDataKeyPrefix + job.ID
	if err := q.client.Set(ctx, dataKey, data, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to store job data: %w", err)
	}

	// Add to queue
	if err := q.client.LPush(ctx, jobQueueKey, job.ID).Err(); err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}

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
		return nil, fmt.Errorf("failed to get job data: %w", err)
	}

	var job Job
	if err := json.Unmarshal(data, &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	// Delete job data after dequeue
	q.client.Del(ctx, dataKey)

	return &job, nil
}

// Length returns the number of pending jobs
func (q *Queue) Length(ctx context.Context) (int64, error) {
	return q.client.LLen(ctx, jobQueueKey).Result()
}
