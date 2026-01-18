package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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

// ============================================================================
// BuilderSessionsServiceImpl - Implementation of BuilderSessionsService
// ============================================================================

// BuilderSessionsServiceImpl provides builder session access from database
type BuilderSessionsServiceImpl struct {
	pool     *pgxpool.Pool
	tenantID uuid.UUID
	ctx      context.Context
}

// NewBuilderSessionsService creates a new BuilderSessionsServiceImpl
func NewBuilderSessionsService(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID) *BuilderSessionsServiceImpl {
	return &BuilderSessionsServiceImpl{
		pool:     pool,
		tenantID: tenantID,
		ctx:      ctx,
	}
}

// Get retrieves a builder session by ID
func (s *BuilderSessionsServiceImpl) Get(sessionID string) (map[string]interface{}, error) {
	sID, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID: %w", err)
	}

	query := `
		SELECT id, tenant_id, user_id, copilot_session_id, status, hearing_phase,
		       hearing_progress, spec, project_id, created_at, updated_at
		FROM builder_sessions
		WHERE id = $1 AND tenant_id = $2
	`

	var (
		id               uuid.UUID
		tenantID         uuid.UUID
		userID           string
		copilotSessionID *uuid.UUID
		status           string
		hearingPhase     string
		hearingProgress  int
		spec             []byte
		projectID        *uuid.UUID
		createdAt        time.Time
		updatedAt        time.Time
	)

	err = s.pool.QueryRow(s.ctx, query, sID, s.tenantID).Scan(
		&id, &tenantID, &userID, &copilotSessionID, &status, &hearingPhase,
		&hearingProgress, &spec, &projectID, &createdAt, &updatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("builder session not found: %s", sessionID)
		}
		return nil, fmt.Errorf("failed to get builder session: %w", err)
	}

	session := map[string]interface{}{
		"id":               id.String(),
		"tenant_id":        tenantID.String(),
		"user_id":          userID,
		"status":           status,
		"hearing_phase":    hearingPhase,
		"hearing_progress": hearingProgress,
		"created_at":       createdAt.Format(time.RFC3339),
		"updated_at":       updatedAt.Format(time.RFC3339),
	}

	if copilotSessionID != nil {
		session["copilot_session_id"] = copilotSessionID.String()
	}

	if projectID != nil {
		session["project_id"] = projectID.String()
	}

	if len(spec) > 0 {
		var s interface{}
		if err := json.Unmarshal(spec, &s); err == nil {
			session["spec"] = s
		}
	}

	// Get messages
	messagesQuery := `
		SELECT id, role, content, phase, extracted_data, suggested_questions, created_at
		FROM builder_messages
		WHERE session_id = $1
		ORDER BY created_at ASC
	`

	rows, err := s.pool.Query(s.ctx, messagesQuery, sID)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []map[string]interface{}
	for rows.Next() {
		var (
			msgID              uuid.UUID
			role               string
			content            string
			phase              *string
			extractedData      []byte
			suggestedQuestions []byte
			msgCreatedAt       time.Time
		)

		if err := rows.Scan(&msgID, &role, &content, &phase, &extractedData, &suggestedQuestions, &msgCreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		msg := map[string]interface{}{
			"id":         msgID.String(),
			"role":       role,
			"content":    content,
			"created_at": msgCreatedAt.Format(time.RFC3339),
		}

		if phase != nil {
			msg["phase"] = *phase
		}

		if len(extractedData) > 0 {
			var ed interface{}
			if err := json.Unmarshal(extractedData, &ed); err == nil {
				msg["extracted_data"] = ed
			}
		}

		if len(suggestedQuestions) > 0 {
			var sq interface{}
			if err := json.Unmarshal(suggestedQuestions, &sq); err == nil {
				msg["suggested_questions"] = sq
			}
		}

		messages = append(messages, msg)
	}

	session["messages"] = messages

	return session, nil
}

// Update updates a builder session
func (s *BuilderSessionsServiceImpl) Update(sessionID string, updates map[string]interface{}) error {
	sID, err := uuid.Parse(sessionID)
	if err != nil {
		return fmt.Errorf("invalid session ID: %w", err)
	}

	// Build dynamic update query
	setClauses := []string{"updated_at = NOW()"}
	args := []interface{}{}
	argIndex := 1

	if status, ok := updates["status"].(string); ok {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, status)
		argIndex++
	}

	if phase, ok := updates["hearing_phase"].(string); ok {
		setClauses = append(setClauses, fmt.Sprintf("hearing_phase = $%d", argIndex))
		args = append(args, phase)
		argIndex++
	}

	if progress, ok := updates["hearing_progress"]; ok {
		var progressInt int
		switch v := progress.(type) {
		case int:
			progressInt = v
		case float64:
			progressInt = int(v)
		}
		setClauses = append(setClauses, fmt.Sprintf("hearing_progress = $%d", argIndex))
		args = append(args, progressInt)
		argIndex++
	}

	if spec, ok := updates["spec"]; ok {
		specJSON, err := json.Marshal(spec)
		if err != nil {
			return fmt.Errorf("failed to marshal spec: %w", err)
		}
		setClauses = append(setClauses, fmt.Sprintf("spec = $%d", argIndex))
		args = append(args, specJSON)
		argIndex++
	}

	if projectID, ok := updates["project_id"].(string); ok {
		pID, err := uuid.Parse(projectID)
		if err != nil {
			return fmt.Errorf("invalid project ID: %w", err)
		}
		setClauses = append(setClauses, fmt.Sprintf("project_id = $%d", argIndex))
		args = append(args, pID)
		argIndex++
	}

	args = append(args, sID, s.tenantID)

	query := fmt.Sprintf(
		"UPDATE builder_sessions SET %s WHERE id = $%d AND tenant_id = $%d",
		strings.Join(setClauses, ", "),
		argIndex,
		argIndex+1,
	)

	result, err := s.pool.Exec(s.ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update builder session: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("builder session not found: %s", sessionID)
	}

	return nil
}

// AddMessage adds a message to a builder session
func (s *BuilderSessionsServiceImpl) AddMessage(sessionID string, message map[string]interface{}) error {
	sID, err := uuid.Parse(sessionID)
	if err != nil {
		return fmt.Errorf("invalid session ID: %w", err)
	}

	role, _ := message["role"].(string)
	content, _ := message["content"].(string)

	if role == "" || content == "" {
		return fmt.Errorf("role and content are required")
	}

	var phase *string
	if p, ok := message["phase"].(string); ok {
		phase = &p
	}

	var extractedData []byte
	if ed, ok := message["extracted_data"]; ok {
		extractedData, _ = json.Marshal(ed)
	}

	var suggestedQuestions []byte
	if sq, ok := message["suggested_questions"]; ok {
		suggestedQuestions, _ = json.Marshal(sq)
	}

	query := `
		INSERT INTO builder_messages (session_id, role, content, phase, extracted_data, suggested_questions)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err = s.pool.Exec(s.ctx, query, sID, role, content, phase, extractedData, suggestedQuestions)
	if err != nil {
		return fmt.Errorf("failed to add message: %w", err)
	}

	return nil
}


// ============================================================================
// ProjectsServiceImpl - Implementation of ProjectsService for builder
// ============================================================================

// ProjectsServiceImpl provides project management for builder workflows
type ProjectsServiceImpl struct {
	pool     *pgxpool.Pool
	tenantID uuid.UUID
	ctx      context.Context
}

// NewProjectsService creates a new ProjectsServiceImpl
func NewProjectsService(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID) *ProjectsServiceImpl {
	return &ProjectsServiceImpl{
		pool:     pool,
		tenantID: tenantID,
		ctx:      ctx,
	}
}

// Get retrieves a project by ID
func (s *ProjectsServiceImpl) Get(projectID string) (map[string]interface{}, error) {
	pID, err := uuid.Parse(projectID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	query := `
		SELECT id, tenant_id, name, description, status, version, created_at, updated_at
		FROM projects
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`

	var (
		id          uuid.UUID
		tenantID    uuid.UUID
		name        string
		description *string
		status      string
		version     int
		createdAt   time.Time
		updatedAt   time.Time
	)

	err = s.pool.QueryRow(s.ctx, query, pID, s.tenantID).Scan(
		&id, &tenantID, &name, &description, &status, &version, &createdAt, &updatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("project not found: %s", projectID)
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	project := map[string]interface{}{
		"id":         id.String(),
		"tenant_id":  tenantID.String(),
		"name":       name,
		"status":     status,
		"version":    version,
		"created_at": createdAt.Format(time.RFC3339),
		"updated_at": updatedAt.Format(time.RFC3339),
	}

	if description != nil {
		project["description"] = *description
	}

	return project, nil
}

// Create creates a new project
func (s *ProjectsServiceImpl) Create(data map[string]interface{}) (map[string]interface{}, error) {
	name, _ := data["name"].(string)
	description, _ := data["description"].(string)
	status, _ := data["status"].(string)
	createdBy, _ := data["created_by"].(string)

	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if status == "" {
		status = "draft"
	}

	var desc *string
	if description != "" {
		desc = &description
	}

	query := `
		INSERT INTO projects (tenant_id, name, description, status, version, created_by)
		VALUES ($1, $2, $3, $4, 1, $5)
		RETURNING id, created_at, updated_at
	`

	var (
		id        uuid.UUID
		createdAt time.Time
		updatedAt time.Time
	)

	err := s.pool.QueryRow(s.ctx, query, s.tenantID, name, desc, status, createdBy).Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	return map[string]interface{}{
		"id":         id.String(),
		"tenant_id":  s.tenantID.String(),
		"name":       name,
		"status":     status,
		"version":    1,
		"created_at": createdAt.Format(time.RFC3339),
		"updated_at": updatedAt.Format(time.RFC3339),
	}, nil
}

// Update updates a project
func (s *ProjectsServiceImpl) Update(projectID string, updates map[string]interface{}) error {
	pID, err := uuid.Parse(projectID)
	if err != nil {
		return fmt.Errorf("invalid project ID: %w", err)
	}

	setClauses := []string{"updated_at = NOW()"}
	args := []interface{}{}
	argIndex := 1

	if startStepID, ok := updates["start_step_id"].(string); ok {
		ssID, err := uuid.Parse(startStepID)
		if err != nil {
			return fmt.Errorf("invalid start_step_id: %w", err)
		}
		setClauses = append(setClauses, fmt.Sprintf("start_step_id = $%d", argIndex))
		args = append(args, ssID)
		argIndex++
	}

	if name, ok := updates["name"].(string); ok {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, name)
		argIndex++
	}

	if status, ok := updates["status"].(string); ok {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, status)
		argIndex++
	}

	args = append(args, pID, s.tenantID)

	query := fmt.Sprintf(
		"UPDATE projects SET %s WHERE id = $%d AND tenant_id = $%d AND deleted_at IS NULL",
		strings.Join(setClauses, ", "),
		argIndex,
		argIndex+1,
	)

	_, err = s.pool.Exec(s.ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update project: %w", err)
	}

	return nil
}

// IncrementVersion increments the project version
func (s *ProjectsServiceImpl) IncrementVersion(projectID string) error {
	pID, err := uuid.Parse(projectID)
	if err != nil {
		return fmt.Errorf("invalid project ID: %w", err)
	}

	query := `
		UPDATE projects SET version = version + 1, updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`

	_, err = s.pool.Exec(s.ctx, query, pID, s.tenantID)
	if err != nil {
		return fmt.Errorf("failed to increment version: %w", err)
	}

	return nil
}

// ============================================================================
// StepsServiceImpl - Implementation of StepsService for builder
// ============================================================================

// StepsServiceImpl provides step management for builder workflows
type StepsServiceImpl struct {
	pool     *pgxpool.Pool
	tenantID uuid.UUID
	ctx      context.Context
}

// NewStepsService creates a new StepsServiceImpl
func NewStepsService(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID) *StepsServiceImpl {
	return &StepsServiceImpl{
		pool:     pool,
		tenantID: tenantID,
		ctx:      ctx,
	}
}

// ListByProject retrieves all steps for a project
func (s *StepsServiceImpl) ListByProject(projectID string) ([]map[string]interface{}, error) {
	pID, err := uuid.Parse(projectID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	query := `
		SELECT s.id, s.name, s.type, s.config, s.position_x, s.position_y, s.created_at
		FROM steps s
		JOIN projects p ON s.project_id = p.id
		WHERE s.project_id = $1 AND p.tenant_id = $2 AND s.deleted_at IS NULL
		ORDER BY s.created_at
	`

	rows, err := s.pool.Query(s.ctx, query, pID, s.tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query steps: %w", err)
	}
	defer rows.Close()

	var steps []map[string]interface{}
	for rows.Next() {
		var (
			id        uuid.UUID
			name      string
			stepType  string
			config    []byte
			positionX float64
			positionY float64
			createdAt time.Time
		)

		if err := rows.Scan(&id, &name, &stepType, &config, &positionX, &positionY, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan step: %w", err)
		}

		step := map[string]interface{}{
			"id":         id.String(),
			"name":       name,
			"type":       stepType,
			"position_x": positionX,
			"position_y": positionY,
			"created_at": createdAt.Format(time.RFC3339),
		}

		if len(config) > 0 {
			var cfg interface{}
			if err := json.Unmarshal(config, &cfg); err == nil {
				step["config"] = cfg
			}
		}

		steps = append(steps, step)
	}

	return steps, nil
}

// Create creates a new step
func (s *StepsServiceImpl) Create(data map[string]interface{}) (map[string]interface{}, error) {
	projectID, _ := data["project_id"].(string)
	name, _ := data["name"].(string)
	stepType, _ := data["type"].(string)
	config := data["config"]
	positionX, _ := data["position_x"].(float64)
	positionY, _ := data["position_y"].(float64)
	blockSlug, _ := data["block_slug"].(string)

	if projectID == "" || name == "" || stepType == "" {
		return nil, fmt.Errorf("project_id, name, and type are required")
	}

	pID, err := uuid.Parse(projectID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	var configJSON []byte
	if config != nil {
		configJSON, _ = json.Marshal(config)
	}

	// Look up block_definition_id from slug if provided
	var blockDefID *uuid.UUID
	if blockSlug != "" {
		var bdID uuid.UUID
		err := s.pool.QueryRow(s.ctx,
			"SELECT id FROM block_definitions WHERE slug = $1 AND (tenant_id = $2 OR tenant_id IS NULL) LIMIT 1",
			blockSlug, s.tenantID,
		).Scan(&bdID)
		if err == nil {
			blockDefID = &bdID
		}
	}

	query := `
		INSERT INTO steps (project_id, name, type, config, position_x, position_y, block_definition_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`

	var (
		id        uuid.UUID
		createdAt time.Time
	)

	err = s.pool.QueryRow(s.ctx, query, pID, name, stepType, configJSON, positionX, positionY, blockDefID).Scan(&id, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create step: %w", err)
	}

	return map[string]interface{}{
		"id":         id.String(),
		"project_id": projectID,
		"name":       name,
		"type":       stepType,
		"position_x": positionX,
		"position_y": positionY,
		"created_at": createdAt.Format(time.RFC3339),
	}, nil
}

// Update updates a step
func (s *StepsServiceImpl) Update(stepID string, updates map[string]interface{}) error {
	sID, err := uuid.Parse(stepID)
	if err != nil {
		return fmt.Errorf("invalid step ID: %w", err)
	}

	setClauses := []string{"updated_at = NOW()"}
	args := []interface{}{}
	argIndex := 1

	if name, ok := updates["name"].(string); ok {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, name)
		argIndex++
	}

	if config, ok := updates["config"]; ok {
		configJSON, _ := json.Marshal(config)
		setClauses = append(setClauses, fmt.Sprintf("config = $%d", argIndex))
		args = append(args, configJSON)
		argIndex++
	}

	args = append(args, sID)

	query := fmt.Sprintf(
		"UPDATE steps SET %s WHERE id = $%d AND deleted_at IS NULL",
		strings.Join(setClauses, ", "),
		argIndex,
	)

	_, err = s.pool.Exec(s.ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update step: %w", err)
	}

	return nil
}

// Delete soft-deletes a step
func (s *StepsServiceImpl) Delete(stepID string) error {
	sID, err := uuid.Parse(stepID)
	if err != nil {
		return fmt.Errorf("invalid step ID: %w", err)
	}

	query := `UPDATE steps SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`

	_, err = s.pool.Exec(s.ctx, query, sID)
	if err != nil {
		return fmt.Errorf("failed to delete step: %w", err)
	}

	return nil
}

// ============================================================================
// EdgesServiceImpl - Implementation of EdgesService for builder
// ============================================================================

// EdgesServiceImpl provides edge management for builder workflows
type EdgesServiceImpl struct {
	pool     *pgxpool.Pool
	tenantID uuid.UUID
	ctx      context.Context
}

// NewEdgesService creates a new EdgesServiceImpl
func NewEdgesService(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID) *EdgesServiceImpl {
	return &EdgesServiceImpl{
		pool:     pool,
		tenantID: tenantID,
		ctx:      ctx,
	}
}

// ListByProject retrieves all edges for a project
func (s *EdgesServiceImpl) ListByProject(projectID string) ([]map[string]interface{}, error) {
	pID, err := uuid.Parse(projectID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	query := `
		SELECT e.id, e.source_step_id, e.target_step_id, e.source_port, e.target_port, e.condition
		FROM edges e
		JOIN projects p ON e.project_id = p.id
		WHERE e.project_id = $1 AND p.tenant_id = $2 AND e.deleted_at IS NULL
	`

	rows, err := s.pool.Query(s.ctx, query, pID, s.tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query edges: %w", err)
	}
	defer rows.Close()

	var edges []map[string]interface{}
	for rows.Next() {
		var (
			id           uuid.UUID
			sourceStepID uuid.UUID
			targetStepID uuid.UUID
			sourcePort   string
			targetPort   string
			condition    *string
		)

		if err := rows.Scan(&id, &sourceStepID, &targetStepID, &sourcePort, &targetPort, &condition); err != nil {
			return nil, fmt.Errorf("failed to scan edge: %w", err)
		}

		edge := map[string]interface{}{
			"id":             id.String(),
			"source_step_id": sourceStepID.String(),
			"target_step_id": targetStepID.String(),
			"source_port":    sourcePort,
			"target_port":    targetPort,
		}

		if condition != nil {
			edge["condition"] = *condition
		}

		edges = append(edges, edge)
	}

	return edges, nil
}

// Create creates a new edge
func (s *EdgesServiceImpl) Create(data map[string]interface{}) (map[string]interface{}, error) {
	projectID, _ := data["project_id"].(string)
	sourceStepID, _ := data["source_step_id"].(string)
	targetStepID, _ := data["target_step_id"].(string)
	sourcePort, _ := data["source_port"].(string)
	targetPort, _ := data["target_port"].(string)

	if projectID == "" || sourceStepID == "" || targetStepID == "" {
		return nil, fmt.Errorf("project_id, source_step_id, and target_step_id are required")
	}

	pID, err := uuid.Parse(projectID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	ssID, err := uuid.Parse(sourceStepID)
	if err != nil {
		return nil, fmt.Errorf("invalid source_step_id: %w", err)
	}

	tsID, err := uuid.Parse(targetStepID)
	if err != nil {
		return nil, fmt.Errorf("invalid target_step_id: %w", err)
	}

	if sourcePort == "" {
		sourcePort = "output"
	}
	if targetPort == "" {
		targetPort = "input"
	}

	query := `
		INSERT INTO edges (project_id, source_step_id, target_step_id, source_port, target_port)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	var id uuid.UUID
	err = s.pool.QueryRow(s.ctx, query, pID, ssID, tsID, sourcePort, targetPort).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("failed to create edge: %w", err)
	}

	return map[string]interface{}{
		"id":             id.String(),
		"project_id":     projectID,
		"source_step_id": sourceStepID,
		"target_step_id": targetStepID,
		"source_port":    sourcePort,
		"target_port":    targetPort,
	}, nil
}

// Delete soft-deletes an edge
func (s *EdgesServiceImpl) Delete(edgeID string) error {
	eID, err := uuid.Parse(edgeID)
	if err != nil {
		return fmt.Errorf("invalid edge ID: %w", err)
	}

	query := `UPDATE edges SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`

	_, err = s.pool.Exec(s.ctx, query, eID)
	if err != nil {
		return fmt.Errorf("failed to delete edge: %w", err)
	}

	return nil
}
