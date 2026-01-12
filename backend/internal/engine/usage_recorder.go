package engine

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// UsageRecorder records LLM API usage for cost tracking
type UsageRecorder struct {
	repo   repository.UsageRepository
	logger *slog.Logger
}

// NewUsageRecorder creates a new UsageRecorder
func NewUsageRecorder(repo repository.UsageRepository, logger *slog.Logger) *UsageRecorder {
	if logger == nil {
		logger = slog.Default()
	}
	return &UsageRecorder{
		repo:   repo,
		logger: logger,
	}
}

// RecordParams contains parameters for recording usage
type RecordParams struct {
	TenantID     uuid.UUID
	WorkflowID   *uuid.UUID
	RunID        *uuid.UUID
	StepRunID    *uuid.UUID
	Provider     string
	Model        string
	Operation    string
	InputTokens  int
	OutputTokens int
	LatencyMs    int
	Success      bool
	ErrorMessage string
}

// Record records a usage event
func (r *UsageRecorder) Record(ctx context.Context, params RecordParams) error {
	if r.repo == nil {
		return nil // No repository configured, skip recording
	}

	var latencyPtr *int
	if params.LatencyMs > 0 {
		latencyPtr = &params.LatencyMs
	}

	record := domain.NewUsageRecord(
		params.TenantID,
		params.WorkflowID,
		params.RunID,
		params.StepRunID,
		params.Provider,
		params.Model,
		params.Operation,
		params.InputTokens,
		params.OutputTokens,
		latencyPtr,
		params.Success,
		params.ErrorMessage,
	)

	if err := r.repo.Create(ctx, record); err != nil {
		r.logger.Error("Failed to record usage",
			"error", err,
			"tenant_id", params.TenantID,
			"provider", params.Provider,
			"model", params.Model,
		)
		// Don't fail the request if usage recording fails
		return nil
	}

	r.logger.Debug("Usage recorded",
		"tenant_id", params.TenantID,
		"provider", params.Provider,
		"model", params.Model,
		"input_tokens", params.InputTokens,
		"output_tokens", params.OutputTokens,
		"cost_usd", record.TotalCostUSD,
	)

	return nil
}

// RecordFromMetadata records usage from adapter response metadata
func (r *UsageRecorder) RecordFromMetadata(
	ctx context.Context,
	tenantID uuid.UUID,
	workflowID, runID, stepRunID *uuid.UUID,
	metadata map[string]string,
	latencyMs int,
	success bool,
	errorMessage string,
) error {
	if r.repo == nil {
		return nil
	}

	provider := metadata["adapter"]
	if provider == "" {
		provider = metadata["provider"]
	}
	model := metadata["model"]

	// Skip if no provider/model info
	if provider == "" || model == "" {
		return nil
	}

	// Parse token counts from metadata
	inputTokens := parseIntFromMetadata(metadata, "prompt_tokens", "input_tokens")
	outputTokens := parseIntFromMetadata(metadata, "completion_tokens", "output_tokens")

	// Skip if no token data
	if inputTokens == 0 && outputTokens == 0 {
		return nil
	}

	return r.Record(ctx, RecordParams{
		TenantID:     tenantID,
		WorkflowID:   workflowID,
		RunID:        runID,
		StepRunID:    stepRunID,
		Provider:     provider,
		Model:        model,
		Operation:    "chat",
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		LatencyMs:    latencyMs,
		Success:      success,
		ErrorMessage: errorMessage,
	})
}

// parseIntFromMetadata tries multiple keys and returns the first valid integer
func parseIntFromMetadata(metadata map[string]string, keys ...string) int {
	for _, key := range keys {
		if val, ok := metadata[key]; ok {
			if n, err := strconv.Atoi(val); err == nil {
				return n
			}
		}
	}
	return 0
}
