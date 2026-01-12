package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ============================================================================
// BlocksServiceImpl - Implementation of BlocksService
// ============================================================================

// BlocksServiceImpl provides block definition access from database
type BlocksServiceImpl struct {
	pool     *pgxpool.Pool
	tenantID uuid.UUID
	ctx      context.Context
}

// NewBlocksService creates a new BlocksServiceImpl
func NewBlocksService(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID) *BlocksServiceImpl {
	return &BlocksServiceImpl{
		pool:     pool,
		tenantID: tenantID,
		ctx:      ctx,
	}
}

// List returns all available block definitions for the tenant
func (s *BlocksServiceImpl) List() ([]map[string]interface{}, error) {
	query := `
		SELECT id, slug, name, description, category, ui_config, is_system, created_at, updated_at
		FROM block_definitions
		WHERE (tenant_id = $1 OR tenant_id IS NULL)
		ORDER BY category, name
	`

	rows, err := s.pool.Query(s.ctx, query, s.tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query block definitions: %w", err)
	}
	defer rows.Close()

	var blocks []map[string]interface{}
	for rows.Next() {
		var (
			id          uuid.UUID
			slug        string
			name        string
			description *string
			category    string
			uiConfig    []byte
			isSystem    bool
			createdAt   time.Time
			updatedAt   time.Time
		)

		if err := rows.Scan(&id, &slug, &name, &description, &category, &uiConfig, &isSystem, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan block definition: %w", err)
		}

		block := map[string]interface{}{
			"id":         id.String(),
			"slug":       slug,
			"name":       name,
			"category":   category,
			"is_system":  isSystem,
			"created_at": createdAt.Format(time.RFC3339),
			"updated_at": updatedAt.Format(time.RFC3339),
		}

		if description != nil {
			block["description"] = *description
		}

		if len(uiConfig) > 0 {
			var ui map[string]interface{}
			if err := json.Unmarshal(uiConfig, &ui); err == nil {
				block["ui_config"] = ui
			}
		}

		blocks = append(blocks, block)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating block definitions: %w", err)
	}

	return blocks, nil
}

// Get retrieves a block definition by slug
func (s *BlocksServiceImpl) Get(slug string) (map[string]interface{}, error) {
	query := `
		SELECT id, slug, name, description, category, ui_config, is_system, created_at, updated_at
		FROM block_definitions
		WHERE slug = $1 AND (tenant_id = $2 OR tenant_id IS NULL)
		LIMIT 1
	`

	var (
		id          uuid.UUID
		blockSlug   string
		name        string
		description *string
		category    string
		uiConfig    []byte
		isSystem    bool
		createdAt   time.Time
		updatedAt   time.Time
	)

	err := s.pool.QueryRow(s.ctx, query, slug, s.tenantID).Scan(
		&id, &blockSlug, &name, &description, &category, &uiConfig, &isSystem, &createdAt, &updatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("block not found: %s", slug)
		}
		return nil, fmt.Errorf("failed to get block definition: %w", err)
	}

	block := map[string]interface{}{
		"id":         id.String(),
		"slug":       blockSlug,
		"name":       name,
		"category":   category,
		"is_system":  isSystem,
		"created_at": createdAt.Format(time.RFC3339),
		"updated_at": updatedAt.Format(time.RFC3339),
	}

	if description != nil {
		block["description"] = *description
	}

	if len(uiConfig) > 0 {
		var ui map[string]interface{}
		if err := json.Unmarshal(uiConfig, &ui); err == nil {
			block["ui_config"] = ui
		}
	}

	return block, nil
}

// ============================================================================
// WorkflowsServiceImpl - Implementation of WorkflowsService
// ============================================================================

// WorkflowsServiceImpl provides workflow read access from database
type WorkflowsServiceImpl struct {
	pool     *pgxpool.Pool
	tenantID uuid.UUID
	ctx      context.Context
}

// NewWorkflowsService creates a new WorkflowsServiceImpl
func NewWorkflowsService(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID) *WorkflowsServiceImpl {
	return &WorkflowsServiceImpl{
		pool:     pool,
		tenantID: tenantID,
		ctx:      ctx,
	}
}

// Get retrieves a workflow by ID
func (s *WorkflowsServiceImpl) Get(workflowID string) (map[string]interface{}, error) {
	wfID, err := uuid.Parse(workflowID)
	if err != nil {
		return nil, fmt.Errorf("invalid workflow ID: %w", err)
	}

	// Get workflow
	query := `
		SELECT id, tenant_id, name, description, status, version, input_schema, output_schema,
		       created_at, updated_at, is_system, system_slug
		FROM workflows
		WHERE id = $1 AND (tenant_id = $2 OR is_system = TRUE) AND deleted_at IS NULL
	`

	var (
		id           uuid.UUID
		tenantID     uuid.UUID
		name         string
		description  *string
		status       string
		version      int
		inputSchema  []byte
		outputSchema []byte
		createdAt    time.Time
		updatedAt    time.Time
		isSystem     bool
		systemSlug   *string
	)

	err = s.pool.QueryRow(s.ctx, query, wfID, s.tenantID).Scan(
		&id, &tenantID, &name, &description, &status, &version,
		&inputSchema, &outputSchema, &createdAt, &updatedAt, &isSystem, &systemSlug,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("workflow not found: %s", workflowID)
		}
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	workflow := map[string]interface{}{
		"id":         id.String(),
		"tenant_id":  tenantID.String(),
		"name":       name,
		"status":     status,
		"version":    version,
		"is_system":  isSystem,
		"created_at": createdAt.Format(time.RFC3339),
		"updated_at": updatedAt.Format(time.RFC3339),
	}

	if description != nil {
		workflow["description"] = *description
	}

	if systemSlug != nil {
		workflow["system_slug"] = *systemSlug
	}

	if len(inputSchema) > 0 {
		var schema interface{}
		if err := json.Unmarshal(inputSchema, &schema); err == nil {
			workflow["input_schema"] = schema
		}
	}

	if len(outputSchema) > 0 {
		var schema interface{}
		if err := json.Unmarshal(outputSchema, &schema); err == nil {
			workflow["output_schema"] = schema
		}
	}

	// Get steps
	stepsQuery := `
		SELECT id, name, type, config, position_x, position_y
		FROM steps
		WHERE workflow_id = $1 AND deleted_at IS NULL
		ORDER BY created_at
	`

	rows, err := s.pool.Query(s.ctx, stepsQuery, wfID)
	if err != nil {
		return nil, fmt.Errorf("failed to query steps: %w", err)
	}
	defer rows.Close()

	var steps []map[string]interface{}
	for rows.Next() {
		var (
			stepID    uuid.UUID
			stepName  string
			stepType  string
			config    []byte
			positionX float64
			positionY float64
		)

		if err := rows.Scan(&stepID, &stepName, &stepType, &config, &positionX, &positionY); err != nil {
			return nil, fmt.Errorf("failed to scan step: %w", err)
		}

		step := map[string]interface{}{
			"id":         stepID.String(),
			"name":       stepName,
			"type":       stepType,
			"position_x": positionX,
			"position_y": positionY,
		}

		if len(config) > 0 {
			var cfg interface{}
			if err := json.Unmarshal(config, &cfg); err == nil {
				step["config"] = cfg
			}
		}

		steps = append(steps, step)
	}

	workflow["steps"] = steps

	return workflow, nil
}

// List retrieves all workflows for the tenant
func (s *WorkflowsServiceImpl) List() ([]map[string]interface{}, error) {
	query := `
		SELECT id, tenant_id, name, description, status, version, created_at, updated_at, is_system, system_slug
		FROM workflows
		WHERE (tenant_id = $1 OR is_system = TRUE) AND deleted_at IS NULL
		ORDER BY created_at DESC
	`

	rows, err := s.pool.Query(s.ctx, query, s.tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query workflows: %w", err)
	}
	defer rows.Close()

	var workflows []map[string]interface{}
	for rows.Next() {
		var (
			id          uuid.UUID
			tenantID    uuid.UUID
			name        string
			description *string
			status      string
			version     int
			createdAt   time.Time
			updatedAt   time.Time
			isSystem    bool
			systemSlug  *string
		)

		if err := rows.Scan(&id, &tenantID, &name, &description, &status, &version,
			&createdAt, &updatedAt, &isSystem, &systemSlug); err != nil {
			return nil, fmt.Errorf("failed to scan workflow: %w", err)
		}

		workflow := map[string]interface{}{
			"id":         id.String(),
			"tenant_id":  tenantID.String(),
			"name":       name,
			"status":     status,
			"version":    version,
			"is_system":  isSystem,
			"created_at": createdAt.Format(time.RFC3339),
			"updated_at": updatedAt.Format(time.RFC3339),
		}

		if description != nil {
			workflow["description"] = *description
		}

		if systemSlug != nil {
			workflow["system_slug"] = *systemSlug
		}

		workflows = append(workflows, workflow)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating workflows: %w", err)
	}

	return workflows, nil
}

// ============================================================================
// RunsServiceImpl - Implementation of RunsService
// ============================================================================

// RunsServiceImpl provides run read access from database
type RunsServiceImpl struct {
	pool     *pgxpool.Pool
	tenantID uuid.UUID
	ctx      context.Context
}

// NewRunsService creates a new RunsServiceImpl
func NewRunsService(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID) *RunsServiceImpl {
	return &RunsServiceImpl{
		pool:     pool,
		tenantID: tenantID,
		ctx:      ctx,
	}
}

// Get retrieves a run by ID
func (s *RunsServiceImpl) Get(runID string) (map[string]interface{}, error) {
	rID, err := uuid.Parse(runID)
	if err != nil {
		return nil, fmt.Errorf("invalid run ID: %w", err)
	}

	query := `
		SELECT r.id, r.workflow_id, r.tenant_id, r.status, r.mode, r.trigger_type,
		       r.input, r.output, r.error, r.started_at, r.completed_at, r.created_at,
		       r.trigger_source, r.trigger_metadata,
		       w.name as workflow_name
		FROM runs r
		JOIN workflows w ON r.workflow_id = w.id
		WHERE r.id = $1 AND r.tenant_id = $2
	`

	var (
		id              uuid.UUID
		workflowID      uuid.UUID
		tenantID        uuid.UUID
		status          string
		mode            string
		triggerType     string
		input           []byte
		output          []byte
		runError        *string
		startedAt       *time.Time
		completedAt     *time.Time
		createdAt       time.Time
		triggerSource   *string
		triggerMetadata []byte
		workflowName    string
	)

	err = s.pool.QueryRow(s.ctx, query, rID, s.tenantID).Scan(
		&id, &workflowID, &tenantID, &status, &mode, &triggerType,
		&input, &output, &runError, &startedAt, &completedAt, &createdAt,
		&triggerSource, &triggerMetadata, &workflowName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("run not found: %s", runID)
		}
		return nil, fmt.Errorf("failed to get run: %w", err)
	}

	run := map[string]interface{}{
		"id":            id.String(),
		"workflow_id":   workflowID.String(),
		"workflow_name": workflowName,
		"tenant_id":     tenantID.String(),
		"status":        status,
		"mode":          mode,
		"trigger_type":  triggerType,
		"created_at":    createdAt.Format(time.RFC3339),
	}

	if len(input) > 0 {
		var inp interface{}
		if err := json.Unmarshal(input, &inp); err == nil {
			run["input"] = inp
		}
	}

	if len(output) > 0 {
		var out interface{}
		if err := json.Unmarshal(output, &out); err == nil {
			run["output"] = out
		}
	}

	if runError != nil {
		run["error"] = *runError
	}

	if startedAt != nil {
		run["started_at"] = startedAt.Format(time.RFC3339)
	}

	if completedAt != nil {
		run["completed_at"] = completedAt.Format(time.RFC3339)
	}

	if triggerSource != nil {
		run["trigger_source"] = *triggerSource
	}

	if len(triggerMetadata) > 0 {
		var meta interface{}
		if err := json.Unmarshal(triggerMetadata, &meta); err == nil {
			run["trigger_metadata"] = meta
		}
	}

	return run, nil
}

// GetStepRuns retrieves all step runs for a run
func (s *RunsServiceImpl) GetStepRuns(runID string) ([]map[string]interface{}, error) {
	rID, err := uuid.Parse(runID)
	if err != nil {
		return nil, fmt.Errorf("invalid run ID: %w", err)
	}

	// First verify the run belongs to the tenant
	var tenantID uuid.UUID
	err = s.pool.QueryRow(s.ctx, "SELECT tenant_id FROM runs WHERE id = $1", rID).Scan(&tenantID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("run not found: %s", runID)
		}
		return nil, fmt.Errorf("failed to verify run: %w", err)
	}

	if tenantID != s.tenantID {
		return nil, fmt.Errorf("run not found: %s", runID)
	}

	query := `
		SELECT sr.id, sr.step_id, sr.status, sr.input, sr.output, sr.error,
		       sr.started_at, sr.completed_at, sr.created_at,
		       s.name as step_name, s.type as step_type
		FROM step_runs sr
		JOIN steps s ON sr.step_id = s.id
		WHERE sr.run_id = $1
		ORDER BY sr.created_at
	`

	rows, err := s.pool.Query(s.ctx, query, rID)
	if err != nil {
		return nil, fmt.Errorf("failed to query step runs: %w", err)
	}
	defer rows.Close()

	var stepRuns []map[string]interface{}
	for rows.Next() {
		var (
			id          uuid.UUID
			stepID      uuid.UUID
			status      string
			input       []byte
			output      []byte
			stepError   *string
			startedAt   *time.Time
			completedAt *time.Time
			createdAt   time.Time
			stepName    string
			stepType    string
		)

		if err := rows.Scan(&id, &stepID, &status, &input, &output, &stepError,
			&startedAt, &completedAt, &createdAt, &stepName, &stepType); err != nil {
			return nil, fmt.Errorf("failed to scan step run: %w", err)
		}

		stepRun := map[string]interface{}{
			"id":         id.String(),
			"step_id":    stepID.String(),
			"step_name":  stepName,
			"step_type":  stepType,
			"status":     status,
			"created_at": createdAt.Format(time.RFC3339),
		}

		if len(input) > 0 {
			var inp interface{}
			if err := json.Unmarshal(input, &inp); err == nil {
				stepRun["input"] = inp
			}
		}

		if len(output) > 0 {
			var out interface{}
			if err := json.Unmarshal(output, &out); err == nil {
				stepRun["output"] = out
			}
		}

		if stepError != nil {
			stepRun["error"] = *stepError
		}

		if startedAt != nil {
			stepRun["started_at"] = startedAt.Format(time.RFC3339)
		}

		if completedAt != nil {
			stepRun["completed_at"] = completedAt.Format(time.RFC3339)
		}

		stepRuns = append(stepRuns, stepRun)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating step runs: %w", err)
	}

	return stepRuns, nil
}
