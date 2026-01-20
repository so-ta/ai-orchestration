package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/copilot/tools"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// AgentUsecase handles the agent-based copilot functionality
type AgentUsecase struct {
	sessionRepo repository.CopilotSessionRepository
	blockRepo   repository.BlockDefinitionRepository
	projectRepo repository.ProjectRepository
	stepRepo    repository.StepRepository
	edgeRepo    repository.EdgeRepository
	runRepo     repository.RunRepository
	stepRunRepo repository.StepRunRepository

	agentLoop    *AgentLoop
	toolRegistry *tools.Registry
}

// NewAgentUsecase creates a new agent usecase
func NewAgentUsecase(
	sessionRepo repository.CopilotSessionRepository,
	blockRepo repository.BlockDefinitionRepository,
	projectRepo repository.ProjectRepository,
	stepRepo repository.StepRepository,
	edgeRepo repository.EdgeRepository,
	runRepo repository.RunRepository,
	stepRunRepo repository.StepRunRepository,
) *AgentUsecase {
	// Create tool dependencies
	deps := &tools.Dependencies{
		BlockRepo:   blockRepo,
		ProjectRepo: projectRepo,
		StepRepo:    stepRepo,
		EdgeRepo:    edgeRepo,
		RunRepo:     runRepo,
		StepRunRepo: stepRunRepo,
	}

	// Create tool registry
	registry := tools.NewRegistry()
	if err := tools.RegisterCoreTools(registry, deps); err != nil {
		slog.Error("failed to register core tools", "error", err)
	}

	// Create LLM client
	llmClient := NewLLMClient()

	// Create agent loop
	agentLoop := NewAgentLoop(llmClient, registry, DefaultConfig())

	return &AgentUsecase{
		sessionRepo:  sessionRepo,
		blockRepo:    blockRepo,
		projectRepo:  projectRepo,
		stepRepo:     stepRepo,
		edgeRepo:     edgeRepo,
		runRepo:      runRepo,
		stepRunRepo:  stepRunRepo,
		agentLoop:    agentLoop,
		toolRegistry: registry,
	}
}

// RunAgentInput represents input for running the agent
type RunAgentInput struct {
	TenantID  uuid.UUID
	UserID    string
	SessionID uuid.UUID
	Message   string
}

// RunAgentOutput represents output from running the agent
type RunAgentOutput struct {
	SessionID      uuid.UUID              `json:"session_id"`
	Response       string                 `json:"response"`
	ToolsUsed      []string               `json:"tools_used"`
	Iterations     int                    `json:"iterations"`
	TotalTokens    int                    `json:"total_tokens"`
	UpdatedSession *domain.CopilotSession `json:"updated_session,omitempty"`
}

// RunAgent executes the agent for a given session and message
func (u *AgentUsecase) RunAgent(ctx context.Context, input RunAgentInput) (*RunAgentOutput, error) {
	// Get session with messages
	session, err := u.sessionRepo.GetWithMessages(ctx, input.TenantID, input.SessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	// Add user message to session
	userMsg := domain.NewCopilotMessage(session.ID, "user", input.Message)
	if err := u.sessionRepo.AddMessage(ctx, userMsg); err != nil {
		return nil, fmt.Errorf("add user message: %w", err)
	}

	// Run agent (no streaming)
	result, err := u.agentLoop.Run(ctx, RunInput{
		TenantID:  input.TenantID,
		UserID:    input.UserID,
		ProjectID: session.ContextProjectID,
		SessionID: session.ID,
		Message:   input.Message,
		Mode:      session.Mode,
		History:   session.Messages,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("run agent: %w", err)
	}

	// Add assistant message to session
	assistantMsg := domain.NewCopilotMessage(session.ID, "assistant", result.Response)
	if err := u.sessionRepo.AddMessage(ctx, assistantMsg); err != nil {
		return nil, fmt.Errorf("add assistant message: %w", err)
	}

	// Update session if tools were used
	if len(result.ToolsUsed) > 0 {
		session.SetPhase(domain.CopilotPhaseProposal, 70)
		if err := u.sessionRepo.Update(ctx, session); err != nil {
			slog.Warn("failed to update session phase", "error", err)
		}
	}

	// Reload session with new messages
	updatedSession, _ := u.sessionRepo.GetWithMessages(ctx, input.TenantID, input.SessionID)

	return &RunAgentOutput{
		SessionID:      session.ID,
		Response:       result.Response,
		ToolsUsed:      result.ToolsUsed,
		Iterations:     result.Iterations,
		TotalTokens:    result.TotalTokens,
		UpdatedSession: updatedSession,
	}, nil
}

// RunAgentWithStreaming runs the agent with event streaming
func (u *AgentUsecase) RunAgentWithStreaming(ctx context.Context, input RunAgentInput, events chan<- Event) (*RunAgentOutput, error) {
	// Get session with messages
	session, err := u.sessionRepo.GetWithMessages(ctx, input.TenantID, input.SessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	// Add user message to session
	userMsg := domain.NewCopilotMessage(session.ID, "user", input.Message)
	if err := u.sessionRepo.AddMessage(ctx, userMsg); err != nil {
		return nil, fmt.Errorf("add user message: %w", err)
	}

	// Run agent with streaming
	result, err := u.agentLoop.Run(ctx, RunInput{
		TenantID:  input.TenantID,
		UserID:    input.UserID,
		ProjectID: session.ContextProjectID,
		SessionID: session.ID,
		Message:   input.Message,
		Mode:      session.Mode,
		History:   session.Messages,
	}, events)
	if err != nil {
		return nil, fmt.Errorf("run agent: %w", err)
	}

	// Add assistant message to session
	assistantMsg := domain.NewCopilotMessage(session.ID, "assistant", result.Response)
	if err := u.sessionRepo.AddMessage(ctx, assistantMsg); err != nil {
		return nil, fmt.Errorf("add assistant message: %w", err)
	}

	// Update session if tools were used
	if len(result.ToolsUsed) > 0 {
		session.SetPhase(domain.CopilotPhaseProposal, 70)
		if err := u.sessionRepo.Update(ctx, session); err != nil {
			slog.Warn("failed to update session phase", "error", err)
		}
	}

	// Reload session with new messages
	updatedSession, _ := u.sessionRepo.GetWithMessages(ctx, input.TenantID, input.SessionID)

	return &RunAgentOutput{
		SessionID:      session.ID,
		Response:       result.Response,
		ToolsUsed:      result.ToolsUsed,
		Iterations:     result.Iterations,
		TotalTokens:    result.TotalTokens,
		UpdatedSession: updatedSession,
	}, nil
}

// StartAgentSessionInput represents input for starting an agent session
type StartAgentSessionInput struct {
	TenantID         uuid.UUID
	UserID           string
	ContextProjectID *uuid.UUID
	Mode             domain.CopilotSessionMode
	InitialPrompt    string
}

// StartAgentSessionOutput represents output for starting an agent session
type StartAgentSessionOutput struct {
	Session   *domain.CopilotSession `json:"session"`
	Response  string                 `json:"response"`
	ToolsUsed []string               `json:"tools_used"`
}

// CreateAgentSessionOnly creates a new agent session without processing the initial prompt.
// The initial prompt is stored as a user message, but the agent loop is not run.
// Use RunAgentWithStreaming to process the message via SSE.
func (u *AgentUsecase) CreateAgentSessionOnly(ctx context.Context, input StartAgentSessionInput) (*StartAgentSessionOutput, error) {
	// Verify project exists if specified
	if input.ContextProjectID != nil {
		project, err := u.projectRepo.GetByID(ctx, input.TenantID, *input.ContextProjectID)
		if err != nil {
			return nil, fmt.Errorf("get project: %w", err)
		}
		if project == nil {
			return nil, domain.ErrProjectNotFound
		}
	}

	// Set default mode if not specified
	mode := input.Mode
	if mode == "" {
		mode = domain.CopilotSessionModeCreate
	}

	// Create new session
	session := domain.NewCopilotSession(input.TenantID, input.UserID, input.ContextProjectID, mode)
	if err := u.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	// Store initial prompt in session title for reference
	session.Title = truncateString(input.InitialPrompt, 100)
	if err := u.sessionRepo.Update(ctx, session); err != nil {
		slog.Warn("failed to update session title", "error", err)
	}

	return &StartAgentSessionOutput{
		Session:   session,
		Response:  "", // Will be populated via SSE stream
		ToolsUsed: []string{},
	}, nil
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
}

// StartAgentSession starts a new agent session and processes the initial prompt (synchronous)
// Deprecated: Use CreateAgentSessionOnly + RunAgentWithStreaming for better timeout handling
func (u *AgentUsecase) StartAgentSession(ctx context.Context, input StartAgentSessionInput) (*StartAgentSessionOutput, error) {
	// Verify project exists if specified
	if input.ContextProjectID != nil {
		project, err := u.projectRepo.GetByID(ctx, input.TenantID, *input.ContextProjectID)
		if err != nil {
			return nil, fmt.Errorf("get project: %w", err)
		}
		if project == nil {
			return nil, domain.ErrProjectNotFound
		}
	}

	// Set default mode if not specified
	mode := input.Mode
	if mode == "" {
		mode = domain.CopilotSessionModeCreate
	}

	// Create new session
	session := domain.NewCopilotSession(input.TenantID, input.UserID, input.ContextProjectID, mode)
	if err := u.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	// Add user message
	userMsg := domain.NewCopilotMessage(session.ID, "user", input.InitialPrompt)
	if err := u.sessionRepo.AddMessage(ctx, userMsg); err != nil {
		return nil, fmt.Errorf("add user message: %w", err)
	}

	// Run agent for initial response
	result, err := u.agentLoop.Run(ctx, RunInput{
		TenantID:  input.TenantID,
		UserID:    input.UserID,
		ProjectID: input.ContextProjectID,
		SessionID: session.ID,
		Message:   input.InitialPrompt,
		Mode:      mode,
		History:   []domain.CopilotMessage{},
	}, nil)
	if err != nil {
		slog.Error("agent run failed", "error", err)
		// Return a fallback response
		result = &Result{
			Response:  "申し訳ありませんが、リクエストの処理中にエラーが発生しました。もう一度お試しください。",
			ToolsUsed: []string{},
		}
	}

	// Add assistant message
	assistantMsg := domain.NewCopilotMessage(session.ID, "assistant", result.Response)
	if err := u.sessionRepo.AddMessage(ctx, assistantMsg); err != nil {
		return nil, fmt.Errorf("add assistant message: %w", err)
	}

	// Update session phase
	session.SetPhase(domain.CopilotPhaseProposal, 30)
	if err := u.sessionRepo.Update(ctx, session); err != nil {
		return nil, fmt.Errorf("update session: %w", err)
	}

	// Reload session with messages
	session, _ = u.sessionRepo.GetWithMessages(ctx, input.TenantID, session.ID)

	return &StartAgentSessionOutput{
		Session:   session,
		Response:  result.Response,
		ToolsUsed: result.ToolsUsed,
	}, nil
}

// GetSession retrieves a session with its messages
func (u *AgentUsecase) GetSession(ctx context.Context, tenantID uuid.UUID, sessionID uuid.UUID) (*domain.CopilotSession, error) {
	return u.sessionRepo.GetWithMessages(ctx, tenantID, sessionID)
}

// GetActiveSessionByProject retrieves the most recent active session for a user in a project
func (u *AgentUsecase) GetActiveSessionByProject(ctx context.Context, tenantID uuid.UUID, userID string, projectID uuid.UUID) (*domain.CopilotSession, error) {
	return u.sessionRepo.GetActiveByUserAndProject(ctx, tenantID, userID, projectID)
}

// GetAvailableTools returns the list of available tools
func (u *AgentUsecase) GetAvailableTools() []map[string]interface{} {
	toolDefs := u.toolRegistry.GetToolDefinitions()
	result := make([]map[string]interface{}, 0, len(toolDefs))

	for _, t := range toolDefs {
		var schema map[string]interface{}
		json.Unmarshal(t.InputSchema, &schema)

		result = append(result, map[string]interface{}{
			"name":         t.Name,
			"description":  t.Description,
			"input_schema": schema,
		})
	}

	return result
}
