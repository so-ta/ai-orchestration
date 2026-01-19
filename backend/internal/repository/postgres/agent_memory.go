package postgres

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/souta/ai-orchestration/internal/domain"
)

// AgentMemoryRepository handles agent memory persistence
type AgentMemoryRepository struct {
	db *pgxpool.Pool
}

// NewAgentMemoryRepository creates a new AgentMemoryRepository
func NewAgentMemoryRepository(db *pgxpool.Pool) *AgentMemoryRepository {
	return &AgentMemoryRepository{db: db}
}

// Create creates a new agent memory entry
func (r *AgentMemoryRepository) Create(ctx context.Context, memory *domain.AgentMemory) error {
	var toolCallsJSON []byte
	if len(memory.ToolCalls) > 0 {
		var err error
		toolCallsJSON, err = json.Marshal(memory.ToolCalls)
		if err != nil {
			return err
		}
	}

	query := `
		INSERT INTO agent_memory (
			id, tenant_id, run_id, step_id, role, content,
			tool_calls, tool_call_id, metadata, sequence_number, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.db.Exec(ctx, query,
		memory.ID,
		memory.TenantID,
		memory.RunID,
		memory.StepID,
		memory.Role,
		memory.Content,
		toolCallsJSON,
		memory.ToolCallID,
		memory.Metadata,
		memory.SequenceNumber,
		memory.CreatedAt,
	)

	return err
}

// CreateBatch creates multiple agent memory entries in a single transaction
func (r *AgentMemoryRepository) CreateBatch(ctx context.Context, memories []*domain.AgentMemory) error {
	if len(memories) == 0 {
		return nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, memory := range memories {
		var toolCallsJSON []byte
		if len(memory.ToolCalls) > 0 {
			toolCallsJSON, err = json.Marshal(memory.ToolCalls)
			if err != nil {
				return err
			}
		}

		query := `
			INSERT INTO agent_memory (
				id, tenant_id, run_id, step_id, role, content,
				tool_calls, tool_call_id, metadata, sequence_number, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`

		_, err = tx.Exec(ctx, query,
			memory.ID,
			memory.TenantID,
			memory.RunID,
			memory.StepID,
			memory.Role,
			memory.Content,
			toolCallsJSON,
			memory.ToolCallID,
			memory.Metadata,
			memory.SequenceNumber,
			memory.CreatedAt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// GetByRunAndStep retrieves all memory entries for a run/step combination
func (r *AgentMemoryRepository) GetByRunAndStep(ctx context.Context, runID, stepID uuid.UUID) ([]*domain.AgentMemory, error) {
	query := `
		SELECT id, tenant_id, run_id, step_id, role, content,
		       tool_calls, tool_call_id, metadata, sequence_number, created_at
		FROM agent_memory
		WHERE run_id = $1 AND step_id = $2
		ORDER BY sequence_number ASC
	`

	rows, err := r.db.Query(ctx, query, runID, stepID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memories []*domain.AgentMemory
	for rows.Next() {
		memory := &domain.AgentMemory{}
		var toolCallsJSON []byte

		err := rows.Scan(
			&memory.ID,
			&memory.TenantID,
			&memory.RunID,
			&memory.StepID,
			&memory.Role,
			&memory.Content,
			&toolCallsJSON,
			&memory.ToolCallID,
			&memory.Metadata,
			&memory.SequenceNumber,
			&memory.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if len(toolCallsJSON) > 0 {
			if err := json.Unmarshal(toolCallsJSON, &memory.ToolCalls); err != nil {
				return nil, err
			}
		}

		memories = append(memories, memory)
	}

	return memories, rows.Err()
}

// GetLastNByRunAndStep retrieves the last N memory entries for a run/step combination
func (r *AgentMemoryRepository) GetLastNByRunAndStep(ctx context.Context, runID, stepID uuid.UUID, n int) ([]*domain.AgentMemory, error) {
	query := `
		SELECT id, tenant_id, run_id, step_id, role, content,
		       tool_calls, tool_call_id, metadata, sequence_number, created_at
		FROM agent_memory
		WHERE run_id = $1 AND step_id = $2
		ORDER BY sequence_number DESC
		LIMIT $3
	`

	rows, err := r.db.Query(ctx, query, runID, stepID, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memories []*domain.AgentMemory
	for rows.Next() {
		memory := &domain.AgentMemory{}
		var toolCallsJSON []byte

		err := rows.Scan(
			&memory.ID,
			&memory.TenantID,
			&memory.RunID,
			&memory.StepID,
			&memory.Role,
			&memory.Content,
			&toolCallsJSON,
			&memory.ToolCallID,
			&memory.Metadata,
			&memory.SequenceNumber,
			&memory.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if len(toolCallsJSON) > 0 {
			if err := json.Unmarshal(toolCallsJSON, &memory.ToolCalls); err != nil {
				return nil, err
			}
		}

		memories = append(memories, memory)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Reverse to get chronological order
	for i, j := 0, len(memories)-1; i < j; i, j = i+1, j-1 {
		memories[i], memories[j] = memories[j], memories[i]
	}

	return memories, nil
}

// GetNextSequenceNumber returns the next sequence number for a run/step combination
func (r *AgentMemoryRepository) GetNextSequenceNumber(ctx context.Context, runID, stepID uuid.UUID) (int, error) {
	query := `
		SELECT COALESCE(MAX(sequence_number), 0) + 1
		FROM agent_memory
		WHERE run_id = $1 AND step_id = $2
	`

	var nextSeq int
	err := r.db.QueryRow(ctx, query, runID, stepID).Scan(&nextSeq)
	if err != nil {
		return 0, err
	}

	return nextSeq, nil
}

// DeleteByRunAndStep deletes all memory entries for a run/step combination
func (r *AgentMemoryRepository) DeleteByRunAndStep(ctx context.Context, runID, stepID uuid.UUID) error {
	query := `DELETE FROM agent_memory WHERE run_id = $1 AND step_id = $2`
	_, err := r.db.Exec(ctx, query, runID, stepID)
	return err
}

// DeleteByRun deletes all memory entries for a run
func (r *AgentMemoryRepository) DeleteByRun(ctx context.Context, runID uuid.UUID) error {
	query := `DELETE FROM agent_memory WHERE run_id = $1`
	_, err := r.db.Exec(ctx, query, runID)
	return err
}

// Count returns the number of memory entries for a run/step combination
func (r *AgentMemoryRepository) Count(ctx context.Context, runID, stepID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM agent_memory WHERE run_id = $1 AND step_id = $2`

	var count int
	err := r.db.QueryRow(ctx, query, runID, stepID).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}
