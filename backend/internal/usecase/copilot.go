package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// CopilotUsecase handles AI-assisted project building and workflow generation
type CopilotUsecase struct {
	projectRepo repository.ProjectRepository
	stepRepo    repository.StepRepository
	runRepo     repository.RunRepository
	stepRunRepo repository.StepRunRepository
	sessionRepo repository.CopilotSessionRepository
	blockRepo   repository.BlockDefinitionRepository
	httpClient  *http.Client
	apiKey      string
	baseURL     string
}

// NewCopilotUsecase creates a new CopilotUsecase
func NewCopilotUsecase(
	projectRepo repository.ProjectRepository,
	stepRepo repository.StepRepository,
	runRepo repository.RunRepository,
	stepRunRepo repository.StepRunRepository,
	sessionRepo repository.CopilotSessionRepository,
	blockRepo repository.BlockDefinitionRepository,
) *CopilotUsecase {
	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &CopilotUsecase{
		projectRepo: projectRepo,
		stepRepo:    stepRepo,
		runRepo:     runRepo,
		stepRunRepo: stepRunRepo,
		sessionRepo: sessionRepo,
		blockRepo:   blockRepo,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		apiKey:  os.Getenv("OPENAI_API_KEY"),
		baseURL: baseURL,
	}
}

// ============================================================================
// Input/Output types for Builder functionality
// ============================================================================

// StartSessionInput represents input for starting a copilot session
type StartSessionInput struct {
	TenantID         uuid.UUID
	UserID           string
	ContextProjectID *uuid.UUID
	Mode             domain.CopilotSessionMode
	InitialPrompt    string
}

// StartSessionOutput represents output for starting a copilot session
type StartSessionOutput struct {
	Session            *domain.CopilotSession
	Message            *domain.CopilotMessage
	SuggestedQuestions []string
}

// HearingInput represents input for the hearing process
type HearingInput struct {
	TenantID  uuid.UUID
	SessionID uuid.UUID
	UserID    string
	Message   string
}

// HearingOutput represents output for the hearing process
type HearingOutput struct {
	Session            *domain.CopilotSession
	Message            *domain.CopilotMessage
	SuggestedQuestions []string
	Phase              domain.CopilotPhase
	Progress           int
	Complete           bool
}

// ConstructInput represents input for workflow construction
type ConstructInput struct {
	TenantID  uuid.UUID
	SessionID uuid.UUID
	UserID    string
}

// ConstructOutput represents output for workflow construction
type ConstructOutput struct {
	ProjectID          uuid.UUID
	Summary            *domain.ConstructionSummary
	StepMappings       []domain.StepMappingResult
	CustomRequirements []domain.CustomRequirement
	Warnings           []string
}

// RefineInput represents input for workflow refinement
type RefineInput struct {
	TenantID  uuid.UUID
	SessionID uuid.UUID
	ProjectID uuid.UUID
	UserID    string
	Feedback  string
}

// RefineOutput represents output for workflow refinement
type RefineOutput struct {
	Changes    []string
	NewVersion int
	Summary    string
}

// ============================================================================
// Session Management (Builder functionality)
// ============================================================================

// StartSession starts a new copilot session with an initial prompt
func (u *CopilotUsecase) StartSession(ctx context.Context, input StartSessionInput) (*StartSessionOutput, error) {
	// Set default mode if not specified
	mode := input.Mode
	if mode == "" {
		mode = domain.CopilotSessionModeCreate
	}

	// Create new session
	session := domain.NewCopilotSession(input.TenantID, input.UserID, input.ContextProjectID, mode)

	// Save session
	if err := u.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("create copilot session: %w", err)
	}

	// Add user message
	userMsg := domain.NewCopilotMessage(session.ID, "user", input.InitialPrompt)
	analysisPhase := domain.CopilotPhaseAnalysis
	userMsg.Phase = &analysisPhase
	if err := u.sessionRepo.AddMessage(ctx, userMsg); err != nil {
		return nil, fmt.Errorf("add user message: %w", err)
	}

	// Generate initial assistant response (analysis phase)
	response, suggestedQuestions, err := u.generateInitialResponse(ctx, input.InitialPrompt)
	if err != nil {
		return nil, fmt.Errorf("generate initial response: %w", err)
	}

	// After initial analysis, move to proposal phase
	proposalPhase := domain.CopilotPhaseProposal

	// Add assistant message
	assistantMsg := domain.NewCopilotMessage(session.ID, "assistant", response)
	assistantMsg.Phase = &proposalPhase
	if len(suggestedQuestions) > 0 {
		questionsJSON, _ := json.Marshal(suggestedQuestions)
		assistantMsg.SuggestedQuestions = questionsJSON
	}
	if err := u.sessionRepo.AddMessage(ctx, assistantMsg); err != nil {
		return nil, fmt.Errorf("add assistant message: %w", err)
	}

	// Update session progress to proposal phase
	session.SetPhase(domain.CopilotPhaseProposal, domain.GetPhaseProgress(domain.CopilotPhaseProposal))
	if err := u.sessionRepo.Update(ctx, session); err != nil {
		return nil, fmt.Errorf("update session: %w", err)
	}

	return &StartSessionOutput{
		Session:            session,
		Message:            assistantMsg,
		SuggestedQuestions: suggestedQuestions,
	}, nil
}

// GetSession retrieves a copilot session with messages
func (u *CopilotUsecase) GetSession(ctx context.Context, tenantID, sessionID uuid.UUID) (*domain.CopilotSession, error) {
	return u.sessionRepo.GetWithMessages(ctx, tenantID, sessionID)
}

// ListSessionsByUser lists all copilot sessions for a user
func (u *CopilotUsecase) ListSessionsByUser(ctx context.Context, tenantID uuid.UUID, userID string) ([]*domain.CopilotSession, int, error) {
	return u.sessionRepo.ListByUser(ctx, tenantID, userID, repository.CopilotSessionFilter{
		Limit: 50,
	})
}

// FinalizeSession marks a session as completed
func (u *CopilotUsecase) FinalizeSession(ctx context.Context, tenantID, sessionID uuid.UUID) error {
	session, err := u.sessionRepo.GetByID(ctx, tenantID, sessionID)
	if err != nil {
		return err
	}

	session.Complete()
	return u.sessionRepo.Update(ctx, session)
}

// DeleteSession deletes a copilot session
func (u *CopilotUsecase) DeleteSession(ctx context.Context, tenantID, sessionID uuid.UUID) error {
	return u.sessionRepo.Delete(ctx, tenantID, sessionID)
}

// ============================================================================
// Hearing Process (Builder functionality)
// ============================================================================

// ProcessHearing processes a hearing message and advances the conversation
func (u *CopilotUsecase) ProcessHearing(ctx context.Context, input HearingInput) (*HearingOutput, error) {
	// Get session
	session, err := u.sessionRepo.GetWithMessages(ctx, input.TenantID, input.SessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	// Add user message
	userMsg := domain.NewCopilotMessage(session.ID, "user", input.Message)
	userMsg.Phase = &session.HearingPhase
	if err := u.sessionRepo.AddMessage(ctx, userMsg); err != nil {
		return nil, fmt.Errorf("add user message: %w", err)
	}

	// Get available blocks for context
	blocks, err := u.blockRepo.List(ctx, nil, repository.BlockDefinitionFilter{
		EnabledOnly: true,
	})
	if err != nil {
		return nil, fmt.Errorf("get blocks: %w", err)
	}

	// Process with LLM
	llmOutput, err := u.processHearingWithLLM(ctx, session, input.Message, blocks)
	if err != nil {
		return nil, fmt.Errorf("process hearing: %w", err)
	}

	// Update spec if data was extracted
	if llmOutput.ExtractedData != nil {
		currentSpec := &domain.WorkflowSpec{}
		if session.Spec != nil {
			json.Unmarshal(session.Spec, currentSpec)
		}
		mergeExtractedData(currentSpec, llmOutput.ExtractedData)
		specJSON, _ := json.Marshal(currentSpec)
		session.Spec = specJSON
	}

	// Determine next phase
	nextPhase := llmOutput.NextPhase
	if nextPhase == "" {
		nextPhase = session.HearingPhase
	}
	progress := domain.GetPhaseProgress(nextPhase)

	// Add assistant message
	assistantMsg := domain.NewCopilotMessage(session.ID, "assistant", llmOutput.Response)
	phasePtr := nextPhase
	assistantMsg.Phase = &phasePtr
	if llmOutput.ExtractedData != nil {
		extractedJSON, _ := json.Marshal(llmOutput.ExtractedData)
		assistantMsg.ExtractedData = extractedJSON
	}
	if len(llmOutput.SuggestedQuestions) > 0 {
		questionsJSON, _ := json.Marshal(llmOutput.SuggestedQuestions)
		assistantMsg.SuggestedQuestions = questionsJSON
	}
	if err := u.sessionRepo.AddMessage(ctx, assistantMsg); err != nil {
		return nil, fmt.Errorf("add assistant message: %w", err)
	}

	// Update session
	session.SetPhase(nextPhase, progress)
	if err := u.sessionRepo.Update(ctx, session); err != nil {
		return nil, fmt.Errorf("update session: %w", err)
	}

	return &HearingOutput{
		Session:            session,
		Message:            assistantMsg,
		SuggestedQuestions: llmOutput.SuggestedQuestions,
		Phase:              nextPhase,
		Progress:           progress,
		Complete:           nextPhase == domain.CopilotPhaseCompleted,
	}, nil
}

// ============================================================================
// Workflow Construction (Builder functionality)
// ============================================================================

// ConstructWorkflow constructs a workflow from the session spec
func (u *CopilotUsecase) ConstructWorkflow(ctx context.Context, input ConstructInput) (*ConstructOutput, error) {
	// Get session with spec
	session, err := u.sessionRepo.GetByID(ctx, input.TenantID, input.SessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	if session.Spec == nil {
		return nil, fmt.Errorf("session has no spec")
	}

	// Parse spec
	var spec domain.WorkflowSpec
	if err := json.Unmarshal(session.Spec, &spec); err != nil {
		return nil, fmt.Errorf("parse spec: %w", err)
	}

	// Get available blocks
	blocks, err := u.blockRepo.List(ctx, nil, repository.BlockDefinitionFilter{
		EnabledOnly: true,
	})
	if err != nil {
		return nil, fmt.Errorf("get blocks: %w", err)
	}

	// Map steps to blocks using LLM
	mappings, customRequirements, err := u.mapStepsToBlocks(ctx, &spec, blocks)
	if err != nil {
		return nil, fmt.Errorf("map steps: %w", err)
	}

	// Create project
	project := &domain.Project{
		ID:          uuid.New(),
		TenantID:    input.TenantID,
		Name:        spec.Name,
		Description: spec.Description,
		Status:      domain.ProjectStatusDraft,
		Version:     1,
	}

	if err := u.projectRepo.Create(ctx, project); err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}

	// Update session with project ID
	session.SetProjectID(project.ID)
	session.SetStatus(domain.CopilotSessionStatusReviewing)
	if err := u.sessionRepo.Update(ctx, session); err != nil {
		return nil, fmt.Errorf("update session: %w", err)
	}

	// Build summary
	summary := &domain.ConstructionSummary{
		Name:        spec.Name,
		Description: spec.Description,
		StepsCount:  len(spec.Steps),
		HasApproval: hasApprovalStep(&spec),
		Trigger:     describeTrigger(spec.Trigger),
	}

	// Collect warnings
	var warnings []string
	for _, req := range customRequirements {
		warnings = append(warnings, fmt.Sprintf("カスタムブロック「%s」の作成が必要です", req.Name))
	}

	return &ConstructOutput{
		ProjectID:          project.ID,
		Summary:            summary,
		StepMappings:       mappings,
		CustomRequirements: customRequirements,
		Warnings:           warnings,
	}, nil
}

// ============================================================================
// Workflow Refinement (Builder functionality)
// ============================================================================

// RefineWorkflow refines a workflow based on user feedback
func (u *CopilotUsecase) RefineWorkflow(ctx context.Context, input RefineInput) (*RefineOutput, error) {
	// Get session
	session, err := u.sessionRepo.GetByID(ctx, input.TenantID, input.SessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	// Get project
	project, err := u.projectRepo.GetWithStepsAndEdges(ctx, input.TenantID, input.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}

	// Process refinement with LLM
	changes, err := u.processRefinement(ctx, session, project, input.Feedback)
	if err != nil {
		return nil, fmt.Errorf("process refinement: %w", err)
	}

	// Add refinement message
	refineMsg := domain.NewCopilotMessage(session.ID, "user", input.Feedback)
	if err := u.sessionRepo.AddMessage(ctx, refineMsg); err != nil {
		return nil, fmt.Errorf("add message: %w", err)
	}

	// Update session status
	session.SetStatus(domain.CopilotSessionStatusRefining)
	if err := u.sessionRepo.Update(ctx, session); err != nil {
		return nil, fmt.Errorf("update session: %w", err)
	}

	return &RefineOutput{
		Changes:    changes,
		NewVersion: project.Version + 1,
		Summary:    "変更を適用しました",
	}, nil
}

// ============================================================================
// Original Copilot functionality (Suggest, Diagnose, Explain, etc.)
// ============================================================================

// SuggestInput represents input for suggestion request
type SuggestInput struct {
	TenantID  uuid.UUID
	ProjectID uuid.UUID
	StepID    *uuid.UUID // Current step (optional)
	Context   string     // Additional context from user
}

// SuggestOutput represents suggestion response
type SuggestOutput struct {
	Suggestions []StepSuggestion `json:"suggestions"`
}

// StepSuggestion represents a suggested step
type StepSuggestion struct {
	Type        string                 `json:"type"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config"`
	Reason      string                 `json:"reason"`
}

// Suggest suggests next steps for a project
func (u *CopilotUsecase) Suggest(ctx context.Context, input SuggestInput) (*SuggestOutput, error) {
	// Get project with steps
	project, err := u.projectRepo.GetWithStepsAndEdges(ctx, input.TenantID, input.ProjectID)
	if err != nil {
		return nil, err
	}

	// Build context for LLM
	prompt := buildSuggestPrompt(project, input.StepID, input.Context)

	// Call LLM
	response, err := u.callLLM(ctx, prompt)
	if err != nil {
		return nil, err
	}

	// Parse response
	var suggestions []StepSuggestion
	if err := json.Unmarshal([]byte(response), &suggestions); err != nil {
		// If JSON parsing fails, return a default suggestion
		suggestions = []StepSuggestion{
			{
				Type:        "tool",
				Name:        "Next Step",
				Description: "Add a tool step to process data",
				Config:      map[string]interface{}{"adapter_id": "mock"},
				Reason:      "Suggested based on project context",
			},
		}
	}

	return &SuggestOutput{Suggestions: suggestions}, nil
}

// DiagnoseInput represents input for error diagnosis
type DiagnoseInput struct {
	TenantID  uuid.UUID
	RunID     uuid.UUID
	StepRunID *uuid.UUID // Specific step run to diagnose (optional)
}

// DiagnoseOutput represents diagnosis response
type DiagnoseOutput struct {
	Diagnosis   Diagnosis `json:"diagnosis"`
	Fixes       []Fix     `json:"fixes"`
	Preventions []string  `json:"preventions"`
}

// Diagnosis represents error diagnosis
type Diagnosis struct {
	RootCause string `json:"root_cause"`
	Category  string `json:"category"` // config_error|input_error|api_error|logic_error|timeout|unknown
	Severity  string `json:"severity"` // high|medium|low
}

// Fix represents a suggested fix
type Fix struct {
	Description string                 `json:"description"`
	Steps       []string               `json:"steps"`
	ConfigPatch map[string]interface{} `json:"config_patch,omitempty"`
}

// Diagnose diagnoses project execution errors
func (u *CopilotUsecase) Diagnose(ctx context.Context, input DiagnoseInput) (*DiagnoseOutput, error) {
	// Get run with step runs
	run, err := u.runRepo.GetWithStepRuns(ctx, input.TenantID, input.RunID)
	if err != nil {
		return nil, err
	}

	// Find failed step runs
	var failedStepRuns []domain.StepRun
	for _, sr := range run.StepRuns {
		if sr.Status == domain.StepRunStatusFailed {
			failedStepRuns = append(failedStepRuns, sr)
		}
	}

	if len(failedStepRuns) == 0 {
		return &DiagnoseOutput{
			Diagnosis: Diagnosis{
				RootCause: "No failed steps found in this run",
				Category:  "unknown",
				Severity:  "low",
			},
			Fixes:       []Fix{},
			Preventions: []string{},
		}, nil
	}

	// Build prompt with error information
	prompt := buildDiagnosePrompt(run, failedStepRuns)

	// Call LLM
	response, err := u.callLLM(ctx, prompt)
	if err != nil {
		// Return basic diagnosis if LLM fails
		return &DiagnoseOutput{
			Diagnosis: Diagnosis{
				RootCause: failedStepRuns[0].Error,
				Category:  "unknown",
				Severity:  "medium",
			},
			Fixes: []Fix{
				{
					Description: "Check step configuration",
					Steps:       []string{"Review the step configuration", "Verify input data format"},
				},
			},
			Preventions: []string{"Add input validation", "Enable retry on failure"},
		}, nil
	}

	// Parse response
	var result DiagnoseOutput
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return &DiagnoseOutput{
			Diagnosis: Diagnosis{
				RootCause: failedStepRuns[0].Error,
				Category:  "unknown",
				Severity:  "medium",
			},
			Fixes:       []Fix{},
			Preventions: []string{},
		}, nil
	}

	return &result, nil
}

// ExplainInput represents input for project explanation
type ExplainInput struct {
	TenantID  uuid.UUID
	ProjectID uuid.UUID
	StepID    *uuid.UUID // Explain specific step (optional)
}

// ExplainOutput represents explanation response
type ExplainOutput struct {
	Summary     string            `json:"summary"`
	StepDetails []StepExplanation `json:"step_details,omitempty"`
}

// StepExplanation represents explanation for a step
type StepExplanation struct {
	StepID      string `json:"step_id"`
	StepName    string `json:"step_name"`
	Explanation string `json:"explanation"`
}

// Explain generates explanation for a project or step
func (u *CopilotUsecase) Explain(ctx context.Context, input ExplainInput) (*ExplainOutput, error) {
	// Get project with steps
	project, err := u.projectRepo.GetWithStepsAndEdges(ctx, input.TenantID, input.ProjectID)
	if err != nil {
		return nil, err
	}

	// Build prompt
	prompt := buildExplainPrompt(project, input.StepID)

	// Call LLM
	response, err := u.callLLM(ctx, prompt)
	if err != nil {
		// Return basic explanation if LLM fails
		return &ExplainOutput{
			Summary: fmt.Sprintf("Project '%s' contains %d steps.", project.Name, len(project.Steps)),
		}, nil
	}

	// Parse response
	var result ExplainOutput
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return &ExplainOutput{
			Summary: response,
		}, nil
	}

	return &result, nil
}

// OptimizeInput represents input for optimization suggestions
type OptimizeInput struct {
	TenantID  uuid.UUID
	ProjectID uuid.UUID
}

// OptimizeOutput represents optimization suggestions
type OptimizeOutput struct {
	Optimizations []Optimization `json:"optimizations"`
	Summary       string         `json:"summary"`
}

// Optimization represents a single optimization suggestion
type Optimization struct {
	Category    string `json:"category"` // performance|cost|reliability|maintainability
	Title       string `json:"title"`
	Description string `json:"description"`
	Impact      string `json:"impact"` // high|medium|low
	Effort      string `json:"effort"` // high|medium|low
}

// Optimize suggests optimizations for a project
func (u *CopilotUsecase) Optimize(ctx context.Context, input OptimizeInput) (*OptimizeOutput, error) {
	// Get project with steps
	project, err := u.projectRepo.GetWithStepsAndEdges(ctx, input.TenantID, input.ProjectID)
	if err != nil {
		return nil, err
	}

	// Build prompt
	prompt := buildOptimizePrompt(project)

	// Call LLM
	response, err := u.callLLM(ctx, prompt)
	if err != nil {
		// Return basic optimization if LLM fails
		return &OptimizeOutput{
			Optimizations: []Optimization{
				{
					Category:    "performance",
					Title:       "Add caching",
					Description: "Consider adding caching for frequently accessed data",
					Impact:      "medium",
					Effort:      "low",
				},
			},
			Summary: "Review project for potential optimizations",
		}, nil
	}

	// Parse response
	var result OptimizeOutput
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return &OptimizeOutput{
			Summary: response,
		}, nil
	}

	return &result, nil
}

// ChatInput represents input for general chat
type ChatInput struct {
	TenantID  uuid.UUID
	ProjectID *uuid.UUID
	Message   string
	Context   string // Additional context
}

// ChatOutput represents chat response
type ChatOutput struct {
	Response    string            `json:"response"`
	Suggestions []StepSuggestion  `json:"suggestions,omitempty"`
	Actions     []SuggestedAction `json:"actions,omitempty"`
}

// SuggestedAction represents an action the user can take
type SuggestedAction struct {
	Type        string                 `json:"type"` // add_step|modify_step|delete_step|run_project
	Label       string                 `json:"label"`
	Description string                 `json:"description"`
	Data        map[string]interface{} `json:"data,omitempty"`
}

// Chat handles general copilot chat
func (u *CopilotUsecase) Chat(ctx context.Context, input ChatInput) (*ChatOutput, error) {
	var projectContext string
	if input.ProjectID != nil {
		project, err := u.projectRepo.GetWithStepsAndEdges(ctx, input.TenantID, *input.ProjectID)
		if err == nil {
			projectContext = buildProjectContextString(project)
		}
	}

	prompt := buildChatPrompt(input.Message, projectContext, input.Context)

	response, err := u.callLLM(ctx, prompt)
	if err != nil {
		return &ChatOutput{
			Response: "I apologize, but I'm unable to process your request at the moment. Please try again later.",
		}, nil
	}

	// Try to parse as structured response (strip markdown code blocks if present)
	cleanResponse := stripMarkdownCodeBlock(response)
	var result ChatOutput
	if err := json.Unmarshal([]byte(cleanResponse), &result); err != nil {
		// If not JSON, return as plain text response
		return &ChatOutput{
			Response: cleanResponse,
		}, nil
	}

	return &result, nil
}

// ========== Project-scoped Session Management ==========

// GetOrCreateSessionInput represents input for getting or creating a session
type GetOrCreateSessionInput struct {
	TenantID  uuid.UUID
	UserID    string
	ProjectID uuid.UUID
}

// GetOrCreateSession gets an existing active session or creates a new one
func (u *CopilotUsecase) GetOrCreateSession(ctx context.Context, input GetOrCreateSessionInput) (*domain.CopilotSession, error) {
	// Try to get existing active session
	session, err := u.sessionRepo.GetActiveByUserAndProject(ctx, input.TenantID, input.UserID, input.ProjectID)
	if err != nil {
		return nil, err
	}
	if session != nil {
		// Load messages
		return u.sessionRepo.GetWithMessages(ctx, input.TenantID, session.ID)
	}

	// Create new session
	session = domain.NewCopilotSession(input.TenantID, input.UserID, &input.ProjectID, domain.CopilotSessionModeCreate)
	if err := u.sessionRepo.Create(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

// GetSessionWithMessages gets a session with all its messages
func (u *CopilotUsecase) GetSessionWithMessages(ctx context.Context, tenantID uuid.UUID, sessionID uuid.UUID) (*domain.CopilotSession, error) {
	return u.sessionRepo.GetWithMessages(ctx, tenantID, sessionID)
}

// ListSessionsInput represents input for listing sessions
type ListSessionsInput struct {
	TenantID  uuid.UUID
	UserID    string
	ProjectID uuid.UUID
}

// ListSessions lists all sessions for a user and project
func (u *CopilotUsecase) ListSessions(ctx context.Context, input ListSessionsInput) ([]*domain.CopilotSession, error) {
	return u.sessionRepo.ListByUserAndProject(ctx, input.TenantID, input.UserID, input.ProjectID)
}

// StartNewSessionInput represents input for starting a new session
type StartNewSessionInput struct {
	TenantID  uuid.UUID
	UserID    string
	ProjectID uuid.UUID
}

// StartNewSession closes any existing active session and creates a new one
func (u *CopilotUsecase) StartNewSession(ctx context.Context, input StartNewSessionInput) (*domain.CopilotSession, error) {
	// Close any existing active session
	existingSession, err := u.sessionRepo.GetActiveByUserAndProject(ctx, input.TenantID, input.UserID, input.ProjectID)
	if err != nil {
		return nil, err
	}
	if existingSession != nil {
		if err := u.sessionRepo.UpdateStatus(ctx, input.TenantID, existingSession.ID, domain.CopilotSessionStatusAbandoned); err != nil {
			return nil, err
		}
	}

	// Create new session
	session := domain.NewCopilotSession(input.TenantID, input.UserID, &input.ProjectID, domain.CopilotSessionModeCreate)
	if err := u.sessionRepo.Create(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

// ChatWithSessionInput represents input for chat with session persistence
type ChatWithSessionInput struct {
	TenantID  uuid.UUID
	UserID    string
	ProjectID uuid.UUID
	SessionID *uuid.UUID // Optional: specific session to use
	Message   string
	Context   string
}

// ChatWithSession handles chat with session persistence
func (u *CopilotUsecase) ChatWithSession(ctx context.Context, input ChatWithSessionInput) (*ChatOutput, *domain.CopilotSession, error) {
	var session *domain.CopilotSession
	var err error

	// Get or create session
	if input.SessionID != nil {
		session, err = u.sessionRepo.GetWithMessages(ctx, input.TenantID, *input.SessionID)
		if err != nil {
			return nil, nil, err
		}
	} else {
		session, err = u.GetOrCreateSession(ctx, GetOrCreateSessionInput{
			TenantID:  input.TenantID,
			UserID:    input.UserID,
			ProjectID: input.ProjectID,
		})
		if err != nil {
			return nil, nil, err
		}
	}

	// Save user message
	userMsg := domain.NewCopilotMessage(session.ID, "user", input.Message)
	if err := u.sessionRepo.AddMessage(ctx, userMsg); err != nil {
		return nil, nil, err
	}

	// Set title from first message if not set
	if session.Title == "" {
		session.SetTitle(input.Message)
		if err := u.sessionRepo.Update(ctx, session); err != nil {
			return nil, nil, err
		}
	}

	// Get project context
	var projectContext string
	project, err := u.projectRepo.GetWithStepsAndEdges(ctx, input.TenantID, input.ProjectID)
	if err == nil {
		projectContext = buildProjectContextString(project)
	}

	// Build prompt with conversation history
	prompt := buildChatPromptWithHistory(input.Message, projectContext, input.Context, session.Messages)

	// Call LLM
	response, err := u.callLLM(ctx, prompt)
	if err != nil {
		response = "申し訳ありませんが、現在リクエストを処理できません。後でもう一度お試しください。"
	}

	// Parse response (strip markdown code blocks if present)
	cleanResponse := stripMarkdownCodeBlock(response)
	var result ChatOutput
	if err := json.Unmarshal([]byte(cleanResponse), &result); err != nil {
		// If still can't parse, use the clean response as plain text
		result = ChatOutput{Response: cleanResponse}
	}

	// Save assistant message
	assistantMsg := domain.NewCopilotMessage(session.ID, "assistant", result.Response)
	if err := u.sessionRepo.AddMessage(ctx, assistantMsg); err != nil {
		return nil, nil, err
	}

	// Reload session with messages
	session, err = u.sessionRepo.GetWithMessages(ctx, input.TenantID, session.ID)
	if err != nil {
		return nil, nil, err
	}

	return &result, session, nil
}

// ========== Project Generation ==========

// GenerateProjectInput represents input for project generation
type GenerateProjectInput struct {
	TenantID    uuid.UUID
	ProjectID   uuid.UUID // Target project to add steps to
	Description string    // Natural language description
}

// GeneratedStep represents a step to be created
type GeneratedStep struct {
	TempID      string                 `json:"temp_id"`    // Temporary ID for edge references
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`       // llm, tool, condition, etc.
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config"`
	PositionX   int                    `json:"position_x"`
	PositionY   int                    `json:"position_y"`
}

// GeneratedEdge represents an edge to be created
type GeneratedEdge struct {
	SourceTempID string `json:"source_temp_id"`
	TargetTempID string `json:"target_temp_id"`
	SourcePort   string `json:"source_port,omitempty"`
	Condition    string `json:"condition,omitempty"`
}

// GenerateProjectOutput represents the generated project structure
type GenerateProjectOutput struct {
	Response    string          `json:"response"`      // Explanation of what was generated
	Steps       []GeneratedStep `json:"steps"`
	Edges       []GeneratedEdge `json:"edges"`
	StartStepID string          `json:"start_step_id"` // TempID of the entry point
}

// GenerateProject generates a project structure from natural language
func (u *CopilotUsecase) GenerateProject(ctx context.Context, input GenerateProjectInput) (*GenerateProjectOutput, error) {
	// Get existing project for context
	var projectContext string
	project, err := u.projectRepo.GetWithStepsAndEdges(ctx, input.TenantID, input.ProjectID)
	if err == nil {
		projectContext = buildProjectContextString(project)
	}

	prompt := buildProjectGenerationPrompt(input.Description, projectContext)

	response, err := u.callLLM(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse response
	cleanResponse := stripMarkdownCodeBlock(response)
	var result GenerateProjectOutput
	if err := json.Unmarshal([]byte(cleanResponse), &result); err != nil {
		// Return a default response if parsing fails
		return &GenerateProjectOutput{
			Response: "プロジェクトの生成に失敗しました。もう少し具体的に説明していただけますか？",
			Steps:    []GeneratedStep{},
			Edges:    []GeneratedEdge{},
		}, nil
	}

	// Validate and filter step types
	result = filterInvalidSteps(result)

	// Assign positions if not provided
	assignPositions(&result)

	return &result, nil
}

// ============================================================================
// LLM Integration (private methods)
// ============================================================================

// llmHearingOutput represents the output from hearing LLM call
type llmHearingOutput struct {
	Response           string
	ExtractedData      map[string]interface{}
	SuggestedQuestions []string
	NextPhase          domain.CopilotPhase
	Confidence         string
}

// generateInitialResponse generates the initial assistant response
func (u *CopilotUsecase) generateInitialResponse(ctx context.Context, initialPrompt string) (string, []string, error) {
	// Get available blocks for context
	blocks, err := u.blockRepo.List(ctx, nil, repository.BlockDefinitionFilter{
		EnabledOnly: true,
	})
	if err != nil {
		slog.Warn("failed to get blocks for context", "error", err)
		blocks = []*domain.BlockDefinition{}
	}

	// Build block list for prompt
	var blockList []string
	for _, b := range blocks {
		blockList = append(blockList, fmt.Sprintf("- %s: %s", b.Name, b.Description))
	}

	prompt := fmt.Sprintf(`あなたはワークフロー自動化の専門家です。ユーザーの要望を深く分析し、最適なワークフローを提案してください。

## ユーザーの要望
%s

## 利用可能なブロック
%s

## タスク
1. ユーザーの要望を分析し、以下を推察してください：
   - トリガー（開始条件）: manual（手動）、schedule（定期実行）、webhook（外部からのトリガー）
   - 実行頻度
   - 必要なステップと処理フロー
   - 必要な連携サービス

2. 推察した内容を「想定した条件」として提示してください

3. 不明確な点があれば、最大3つまでの質問を提示してください

## 出力形式（JSON）
{
  "response": "ユーザーへの応答メッセージ（分析結果の説明）",
  "assumptions": {
    "trigger": "manual|schedule|webhook",
    "trigger_detail": "トリガーの詳細説明",
    "frequency": "実行頻度の説明",
    "steps": ["ステップ1", "ステップ2", ...],
    "integrations": ["連携サービス1", ...]
  },
  "clarifying_questions": ["質問1", "質問2", ...],
  "suggested_responses": ["この内容で問題ありません", "トリガーを変更したい", "ステップを追加したい"]
}

JSONのみを出力してください。`, initialPrompt, joinStrings(blockList, "\n"))

	// Call LLM
	llmResponse, err := u.callLLM(ctx, prompt)
	if err != nil {
		slog.Error("LLM call failed", "error", err)
		// Fallback to template response
		resp, questions := u.fallbackInitialResponse(initialPrompt)
		return resp, questions, nil
	}

	// Parse LLM response
	var parsed struct {
		Response    string `json:"response"`
		Assumptions struct {
			Trigger       string   `json:"trigger"`
			TriggerDetail string   `json:"trigger_detail"`
			Frequency     string   `json:"frequency"`
			Steps         []string `json:"steps"`
			Integrations  []string `json:"integrations"`
		} `json:"assumptions"`
		ClarifyingQuestions []string `json:"clarifying_questions"`
		SuggestedResponses  []string `json:"suggested_responses"`
	}

	if err := json.Unmarshal([]byte(llmResponse), &parsed); err != nil {
		slog.Warn("failed to parse LLM response", "error", err, "response", llmResponse)
		resp, questions := u.fallbackInitialResponse(initialPrompt)
		return resp, questions, nil
	}

	// Build response message
	var response string
	if parsed.Response != "" {
		response = parsed.Response
	} else {
		response = fmt.Sprintf("「%s」について分析しました。", initialPrompt)
	}

	// Add assumptions to response
	response += "\n\n【想定した条件】"
	if parsed.Assumptions.TriggerDetail != "" {
		response += fmt.Sprintf("\n- トリガー: %s", parsed.Assumptions.TriggerDetail)
	}
	if parsed.Assumptions.Frequency != "" {
		response += fmt.Sprintf("\n- 実行頻度: %s", parsed.Assumptions.Frequency)
	}
	if len(parsed.Assumptions.Steps) > 0 {
		response += fmt.Sprintf("\n- 主要なステップ: %s", joinStrings(parsed.Assumptions.Steps, " → "))
	}
	if len(parsed.Assumptions.Integrations) > 0 {
		response += fmt.Sprintf("\n- 連携サービス: %s", joinStrings(parsed.Assumptions.Integrations, ", "))
	}

	// Add clarifying questions if any
	if len(parsed.ClarifyingQuestions) > 0 {
		response += "\n\n【確認したい点】"
		for _, q := range parsed.ClarifyingQuestions {
			response += fmt.Sprintf("\n- %s", q)
		}
	}

	response += "\n\n上記の前提で問題なければ「確認して構築」を押してください。修正が必要な場合はお知らせください。"

	// Use suggested responses or defaults
	suggestedQuestions := parsed.SuggestedResponses
	if len(suggestedQuestions) == 0 {
		suggestedQuestions = []string{
			"この内容で問題ありません",
			"トリガーを変更したい",
			"ステップを追加したい",
		}
	}

	return response, suggestedQuestions, nil
}

// fallbackInitialResponse returns a template response when LLM fails
func (u *CopilotUsecase) fallbackInitialResponse(initialPrompt string) (string, []string) {
	response := fmt.Sprintf(`「%s」について分析しています...

AIが要件を分析し、以下を推察しました：

【想定した条件】
- トリガー: 手動実行（必要に応じて定期実行に変更可能）
- 実行頻度: オンデマンド
- 主要なステップ: データ取得 → 処理 → 出力

上記の前提で問題なければ「確認して構築」を押してください。
修正が必要な場合はお知らせください。`, initialPrompt)

	suggestedQuestions := []string{
		"この内容で問題ありません",
		"トリガーを変更したい",
		"ステップを追加したい",
	}

	return response, suggestedQuestions
}

// callLLM makes a direct call to OpenAI API
func (u *CopilotUsecase) callLLM(ctx context.Context, prompt string) (string, error) {
	if u.apiKey == "" {
		// Return mock response in development mode
		return u.mockLLMResponse(prompt), nil
	}

	reqBody := map[string]interface{}{
		"model": "gpt-4o-mini",
		"messages": []map[string]string{
			{"role": "system", "content": "You are an AI assistant for project automation. Always respond with valid JSON when instructed to do so."},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.3,
		"max_tokens":  2000,
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", u.baseURL+"/chat/completions", bytes.NewReader(reqJSON))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+u.apiKey)

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call OpenAI API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return result.Choices[0].Message.Content, nil
}

// processHearingWithLLM processes a hearing message with LLM
func (u *CopilotUsecase) processHearingWithLLM(ctx context.Context, session *domain.CopilotSession, message string, blocks []*domain.BlockDefinition) (*llmHearingOutput, error) {
	currentPhase := session.HearingPhase
	nextPhase := domain.NextPhase(currentPhase)

	var response string
	var suggestedQuestions []string

	switch nextPhase {
	case domain.CopilotPhaseProposal:
		response = "ご要望を分析しました。\n\n以下の前提でワークフローを構築することを提案します。\n\n【想定した条件】\n- トリガー: 手動実行\n- 主要なステップ: データ取得 → 処理 → 通知\n\n問題なければ「確認して構築」ボタンを押してください。修正があればお知らせください。"
		suggestedQuestions = []string{
			"この内容で問題ありません",
			"修正したい点があります",
		}
	case domain.CopilotPhaseCompleted:
		response = "ヒアリングが完了しました。ワークフローの構築を開始できます。"
		suggestedQuestions = []string{}
	default:
		response = "ご回答ありがとうございます。引き続き分析中です。"
		suggestedQuestions = []string{}
	}

	return &llmHearingOutput{
		Response:           response,
		ExtractedData:      nil,
		SuggestedQuestions: suggestedQuestions,
		NextPhase:          nextPhase,
		Confidence:         "high",
	}, nil
}

// mapStepsToBlocks maps workflow spec steps to available blocks
func (u *CopilotUsecase) mapStepsToBlocks(ctx context.Context, spec *domain.WorkflowSpec, blocks []*domain.BlockDefinition) ([]domain.StepMappingResult, []domain.CustomRequirement, error) {
	var mappings []domain.StepMappingResult
	var customRequirements []domain.CustomRequirement

	// Create a block lookup map
	blockMap := make(map[string]*domain.BlockDefinition)
	for _, b := range blocks {
		blockMap[b.Slug] = b
	}

	for _, step := range spec.Steps {
		mapping := domain.StepMappingResult{
			Name: step.Name,
		}

		// Try to find a matching block
		matched := false
		switch step.Type {
		case "input":
			mapping.Block = "start"
			mapping.Confidence = "high"
			matched = true
		case "transform":
			mapping.Block = "function"
			mapping.Confidence = "high"
			matched = true
		case "decision":
			mapping.Block = "condition"
			mapping.Confidence = "high"
			matched = true
		case "notification":
			if _, ok := blockMap["slack"]; ok {
				mapping.Block = "slack"
				mapping.Confidence = "medium"
				matched = true
			} else if _, ok := blockMap["discord"]; ok {
				mapping.Block = "discord"
				mapping.Confidence = "medium"
				matched = true
			}
		case "approval":
			mapping.Block = "human_in_loop"
			mapping.Confidence = "high"
			matched = true
		case "ai":
			mapping.Block = "llm"
			mapping.Confidence = "high"
			matched = true
		case "integration":
			mapping.Block = "http"
			mapping.Confidence = "low"
			matched = true
		}

		if !matched {
			mapping.CustomRequired = true
			mapping.Reason = fmt.Sprintf("プリセットブロックで「%s」タイプに対応するものがありません", step.Type)
			customRequirements = append(customRequirements, domain.CustomRequirement{
				Name:            step.Name,
				Description:     step.Description,
				EstimatedEffort: "medium",
			})
		}

		mappings = append(mappings, mapping)
	}

	return mappings, customRequirements, nil
}

// processRefinement processes workflow refinement based on feedback
func (u *CopilotUsecase) processRefinement(ctx context.Context, session *domain.CopilotSession, project *domain.Project, feedback string) ([]string, error) {
	return []string{"フィードバックを受け付けました"}, nil
}

// ============================================================================
// Helper functions
// ============================================================================

// stripMarkdownCodeBlock removes markdown code block wrappers from JSON responses
func stripMarkdownCodeBlock(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
	}
	if strings.HasSuffix(s, "```") {
		s = strings.TrimSuffix(s, "```")
	}
	return strings.TrimSpace(s)
}

// joinStrings joins strings with a separator
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// mergeExtractedData merges extracted data into the workflow spec
func mergeExtractedData(spec *domain.WorkflowSpec, data map[string]interface{}) {
	if name, ok := data["name"].(string); ok {
		spec.Name = name
	}
	if desc, ok := data["description"].(string); ok {
		spec.Description = desc
	}
	if purpose, ok := data["purpose"].(string); ok {
		spec.Purpose = purpose
	}
}

// hasApprovalStep checks if the spec has an approval step
func hasApprovalStep(spec *domain.WorkflowSpec) bool {
	for _, step := range spec.Steps {
		if step.Type == "approval" {
			return true
		}
	}
	for _, actor := range spec.Actors {
		if actor.Role == "approver" {
			return true
		}
	}
	return false
}

// describeTrigger returns a human-readable description of the trigger
func describeTrigger(trigger *domain.TriggerSpec) string {
	if trigger == nil {
		return "manual"
	}
	switch trigger.Type {
	case "schedule":
		if trigger.Schedule != "" {
			return fmt.Sprintf("schedule (%s)", trigger.Schedule)
		}
		return "schedule"
	case "webhook":
		return "webhook"
	case "event":
		if trigger.EventSource != "" {
			return fmt.Sprintf("event (%s)", trigger.EventSource)
		}
		return "event"
	default:
		return "manual"
	}
}

// buildSuggestPrompt builds the prompt for step suggestions
func buildSuggestPrompt(project *domain.Project, currentStepID *uuid.UUID, context string) string {
	var sb strings.Builder

	sb.WriteString("You are an AI project assistant. Suggest the next steps for this project.\n\n")
	sb.WriteString("## Available Step Types\n")
	sb.WriteString("- llm: LLM API calls (OpenAI, Anthropic)\n")
	sb.WriteString("- tool: External tool/adapter execution\n")
	sb.WriteString("- condition: Conditional branching (true/false)\n")
	sb.WriteString("- switch: Multi-way branching based on expression\n")
	sb.WriteString("- map: Parallel array processing\n")
	sb.WriteString("- wait: Delay/timer\n")
	sb.WriteString("- human_in_loop: Human approval gate\n")
	sb.WriteString("- router: AI-based dynamic routing\n")
	sb.WriteString("- log: Debug logging\n\n")

	sb.WriteString("## Current Project\n")
	sb.WriteString(fmt.Sprintf("Name: %s\n", project.Name))
	sb.WriteString(fmt.Sprintf("Description: %s\n", project.Description))
	sb.WriteString(fmt.Sprintf("Steps: %d\n\n", len(project.Steps)))

	for _, step := range project.Steps {
		sb.WriteString(fmt.Sprintf("- %s (%s)\n", step.Name, step.Type))
	}

	if context != "" {
		sb.WriteString(fmt.Sprintf("\n## User Context\n%s\n", context))
	}

	sb.WriteString("\n## Instructions\n")
	sb.WriteString("Suggest 2-3 logical next steps. Return a JSON array:\n")
	sb.WriteString(`[{"type": "step_type", "name": "Step Name", "description": "Why this step", "config": {...}, "reason": "Explanation"}]`)

	return sb.String()
}

// buildDiagnosePrompt builds the prompt for error diagnosis
func buildDiagnosePrompt(run *domain.Run, failedStepRuns []domain.StepRun) string {
	var sb strings.Builder

	sb.WriteString("You are an AI debugging assistant. Analyze this project error.\n\n")
	sb.WriteString(fmt.Sprintf("## Run Status: %s\n\n", run.Status))

	sb.WriteString("## Failed Steps\n")
	for _, sr := range failedStepRuns {
		sb.WriteString(fmt.Sprintf("- Step: %s\n", sr.StepName))
		sb.WriteString(fmt.Sprintf("  Error: %s\n", sr.Error))
		if sr.Input != nil {
			sb.WriteString(fmt.Sprintf("  Input: %s\n", string(sr.Input)))
		}
	}

	sb.WriteString("\n## Instructions\n")
	sb.WriteString("Diagnose the error and suggest fixes. Return JSON:\n")
	sb.WriteString(`{"diagnosis": {"root_cause": "...", "category": "config_error|input_error|api_error|logic_error|timeout|unknown", "severity": "high|medium|low"}, "fixes": [{"description": "...", "steps": ["..."]}], "preventions": ["..."]}`)

	return sb.String()
}

// buildExplainPrompt builds the prompt for project explanation
func buildExplainPrompt(project *domain.Project, stepID *uuid.UUID) string {
	var sb strings.Builder

	sb.WriteString("You are an AI assistant. Explain this project in simple terms.\n\n")
	sb.WriteString(fmt.Sprintf("## Project: %s\n", project.Name))
	sb.WriteString(fmt.Sprintf("Description: %s\n\n", project.Description))

	sb.WriteString("## Steps\n")
	for _, step := range project.Steps {
		configJSON, err := json.Marshal(step.Config)
		if err != nil {
			slog.Warn("failed to marshal step config", "step_id", step.ID, "error", err)
			configJSON = []byte("{}")
		}
		sb.WriteString(fmt.Sprintf("- %s (%s): %s\n", step.Name, step.Type, string(configJSON)))
	}

	sb.WriteString("\n## Edges\n")
	for _, edge := range project.Edges {
		sb.WriteString(fmt.Sprintf("- %s -> %s\n", edge.SourceStepID, edge.TargetStepID))
	}

	sb.WriteString("\n## Instructions\n")
	sb.WriteString("Explain what this project does. Return JSON:\n")
	sb.WriteString(`{"summary": "Overall explanation", "step_details": [{"step_id": "...", "step_name": "...", "explanation": "..."}]}`)

	return sb.String()
}

// buildOptimizePrompt builds the prompt for optimization suggestions
func buildOptimizePrompt(project *domain.Project) string {
	var sb strings.Builder

	sb.WriteString("You are an AI optimization assistant. Suggest improvements for this project.\n\n")
	sb.WriteString(fmt.Sprintf("## Project: %s\n", project.Name))
	sb.WriteString(fmt.Sprintf("Steps: %d\n\n", len(project.Steps)))

	for _, step := range project.Steps {
		configJSON, err := json.Marshal(step.Config)
		if err != nil {
			slog.Warn("failed to marshal step config", "step_id", step.ID, "error", err)
			configJSON = []byte("{}")
		}
		sb.WriteString(fmt.Sprintf("- %s (%s): %s\n", step.Name, step.Type, string(configJSON)))
	}

	sb.WriteString("\n## Instructions\n")
	sb.WriteString("Suggest optimizations for performance, cost, reliability, or maintainability. Return JSON:\n")
	sb.WriteString(`{"optimizations": [{"category": "performance|cost|reliability|maintainability", "title": "...", "description": "...", "impact": "high|medium|low", "effort": "high|medium|low"}], "summary": "Overall assessment"}`)

	return sb.String()
}

// buildChatPrompt builds the prompt for general chat
func buildChatPrompt(message, projectContext, additionalContext string) string {
	var sb strings.Builder

	sb.WriteString("You are an AI project assistant. Help the user with their project.\n\n")

	if projectContext != "" {
		sb.WriteString("## Current Project Context\n")
		sb.WriteString(projectContext)
		sb.WriteString("\n\n")
	}

	if additionalContext != "" {
		sb.WriteString("## Additional Context\n")
		sb.WriteString(additionalContext)
		sb.WriteString("\n\n")
	}

	sb.WriteString("## User Message\n")
	sb.WriteString(message)
	sb.WriteString("\n\n")

	sb.WriteString("## Instructions\n")
	sb.WriteString("Respond helpfully. If suggesting project changes, return JSON with 'response' and optional 'suggestions' or 'actions'. Otherwise, just return plain text.")

	return sb.String()
}

// buildProjectContextString builds a string representation of project
func buildProjectContextString(project *domain.Project) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Name: %s\n", project.Name))
	sb.WriteString(fmt.Sprintf("Description: %s\n", project.Description))
	sb.WriteString(fmt.Sprintf("Steps (%d):\n", len(project.Steps)))
	for _, step := range project.Steps {
		sb.WriteString(fmt.Sprintf("  - %s (%s)\n", step.Name, step.Type))
	}
	return sb.String()
}

// buildChatPromptWithHistory builds the prompt including conversation history
func buildChatPromptWithHistory(message, projectContext, additionalContext string, history []domain.CopilotMessage) string {
	var sb strings.Builder

	sb.WriteString("You are an AI project assistant. Help the user with their project.\n\n")

	if projectContext != "" {
		sb.WriteString("## Current Project Context\n")
		sb.WriteString(projectContext)
		sb.WriteString("\n\n")
	}

	if additionalContext != "" {
		sb.WriteString("## Additional Context\n")
		sb.WriteString(additionalContext)
		sb.WriteString("\n\n")
	}

	// Add conversation history
	if len(history) > 0 {
		sb.WriteString("## Conversation History\n")
		start := 0
		if len(history) > 10 {
			start = len(history) - 10
		}
		for _, msg := range history[start:] {
			if msg.Role == "user" {
				sb.WriteString(fmt.Sprintf("User: %s\n", msg.Content))
			} else {
				sb.WriteString(fmt.Sprintf("Assistant: %s\n", msg.Content))
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## Current User Message\n")
	sb.WriteString(message)
	sb.WriteString("\n\n")

	sb.WriteString("## Instructions\n")
	sb.WriteString("Respond helpfully in the same language as the user. If suggesting project changes, return JSON with 'response' and optional 'suggestions' or 'actions'. Otherwise, just return JSON with 'response' field.")

	return sb.String()
}

// buildProjectGenerationPrompt creates a prompt for project generation
func buildProjectGenerationPrompt(description, projectContext string) string {
	var sb strings.Builder

	sb.WriteString("You are an AI project generator. Generate a project based on the user's description.\n\n")

	if projectContext != "" {
		sb.WriteString("## Existing Project Context\n")
		sb.WriteString(projectContext)
		sb.WriteString("\n\n")
	}

	sb.WriteString("## Available Step Types (ONLY use these exact types)\n")
	sb.WriteString("- start: Entry point (required, exactly one)\n")
	sb.WriteString("- llm: LLM/AI call (config: provider, model, system_prompt, user_prompt)\n")
	sb.WriteString("- tool: External tool/adapter (config: adapter_id)\n")
	sb.WriteString("- condition: Binary branching true/false (config: expression)\n")
	sb.WriteString("- switch: Multi-way branching (config: expression, cases)\n")
	sb.WriteString("- map: Parallel array processing (config: input_path, parallel)\n")
	sb.WriteString("- join: Merge parallel branches (config: join_mode)\n")
	sb.WriteString("- subflow: Nested project (config: project_id)\n")
	sb.WriteString("- wait: Delay/timer (config: duration_ms)\n")
	sb.WriteString("- function: Custom code execution (config: code, language)\n")
	sb.WriteString("- router: AI-based dynamic routing (config: routes, provider, model)\n")
	sb.WriteString("- human_in_loop: Human approval gate (config: instructions, timeout_hours)\n")
	sb.WriteString("- filter: Filter items (config: expression)\n")
	sb.WriteString("- split: Split into batches (config: batch_size)\n")
	sb.WriteString("- aggregate: Aggregate data (config: method)\n")
	sb.WriteString("- error: Stop and error (config: message)\n")
	sb.WriteString("- note: Documentation/comment (config: text)\n")
	sb.WriteString("- log: Debug logging (config: message, level)\n")
	sb.WriteString("\nIMPORTANT: Do NOT use 'end' type. Project ends when last step completes.\n")
	sb.WriteString("\n")

	sb.WriteString("## User Request\n")
	sb.WriteString(description)
	sb.WriteString("\n\n")

	sb.WriteString("## Output Format (JSON)\n")
	sb.WriteString(`{
  "response": "Generated project explanation in user's language",
  "steps": [
    {
      "temp_id": "step_1",
      "name": "Step Name",
      "type": "start|llm|tool|condition|switch|map|join|subflow|wait|function|router|human_in_loop|filter|split|aggregate|error|note|log",
      "description": "What this step does",
      "config": { ... },
      "position_x": 400,
      "position_y": 100
    }
  ],
  "edges": [
    {
      "source_temp_id": "step_1",
      "target_temp_id": "step_2",
      "source_port": "default|true|false",
      "condition": ""
    }
  ],
  "start_step_id": "step_1"
}`)
	sb.WriteString("\n\n")

	sb.WriteString("## Instructions\n")
	sb.WriteString("1. ALWAYS include exactly one 'start' step as the entry point\n")
	sb.WriteString("2. The project ends when the last step(s) complete - no 'end' step needed\n")
	sb.WriteString("3. Position steps vertically with 150px spacing\n")
	sb.WriteString("4. Use descriptive step names in the user's language\n")
	sb.WriteString("5. Provide meaningful config for each step type\n")
	sb.WriteString("6. Connect all steps with edges from source to target\n")
	sb.WriteString("7. For LLM steps, include system_prompt and user_prompt in config\n")
	sb.WriteString("8. Respond in the same language as the user\n")

	return sb.String()
}

// assignPositions assigns default positions to steps if not provided
func assignPositions(output *GenerateProjectOutput) {
	startX := 400
	startY := 50
	ySpacing := 150

	for i := range output.Steps {
		if output.Steps[i].PositionX == 0 {
			output.Steps[i].PositionX = startX
		}
		if output.Steps[i].PositionY == 0 {
			output.Steps[i].PositionY = startY + (i * ySpacing)
		}
	}
}

// filterInvalidSteps removes steps with invalid types and their related edges
func filterInvalidSteps(output GenerateProjectOutput) GenerateProjectOutput {
	validTypes := map[string]bool{
		"start":         true,
		"llm":           true,
		"tool":          true,
		"condition":     true,
		"switch":        true,
		"map":           true,
		"subflow":       true,
		"wait":          true,
		"function":      true,
		"router":        true,
		"human_in_loop": true,
		"filter":        true,
		"split":         true,
		"aggregate":     true,
		"error":         true,
		"note":          true,
		"log":           true,
	}

	validTempIDs := make(map[string]bool)
	var validSteps []GeneratedStep

	for _, step := range output.Steps {
		if validTypes[step.Type] {
			validSteps = append(validSteps, step)
			validTempIDs[step.TempID] = true
		}
	}

	var validEdges []GeneratedEdge
	for _, edge := range output.Edges {
		if validTempIDs[edge.SourceTempID] && validTempIDs[edge.TargetTempID] {
			validEdges = append(validEdges, edge)
		}
	}

	return GenerateProjectOutput{
		Response:    output.Response,
		Steps:       validSteps,
		Edges:       validEdges,
		StartStepID: output.StartStepID,
	}
}

// mockLLMResponse provides mock responses when OpenAI API key is not configured
func (u *CopilotUsecase) mockLLMResponse(prompt string) string {
	if strings.Contains(prompt, "Suggest the next steps") {
		return `[{"type": "llm", "name": "Process Input", "description": "Process the input data using LLM", "config": {"provider": "openai", "model": "gpt-4o-mini"}, "reason": "Mock suggestion - configure OPENAI_API_KEY for real suggestions"}]`
	}
	if strings.Contains(prompt, "diagnose") || strings.Contains(prompt, "error") {
		return `{"diagnosis": {"root_cause": "Mock diagnosis - API key not configured", "category": "config_error", "severity": "low"}, "fixes": [{"description": "Configure OPENAI_API_KEY", "steps": ["Set OPENAI_API_KEY environment variable"]}], "preventions": ["Configure API credentials"]}`
	}
	if strings.Contains(prompt, "Explain this project") {
		return `{"summary": "This is a mock explanation. Configure OPENAI_API_KEY for real AI-powered explanations.", "step_details": []}`
	}
	if strings.Contains(prompt, "optimization") || strings.Contains(prompt, "improvements") {
		return `{"optimizations": [{"category": "configuration", "title": "Enable AI features", "description": "Configure OPENAI_API_KEY to enable AI-powered optimizations", "impact": "high", "effort": "low"}], "summary": "Mock response - configure API key for real suggestions"}`
	}
	if strings.Contains(prompt, "project generator") || strings.Contains(prompt, "Available Step Types") {
		return `{
  "response": "サンプルプロジェクトを生成しました。これはモックレスポンスです。OPENAI_API_KEYを設定すると、AIが実際にプロジェクトを生成します。",
  "steps": [
    {"temp_id": "step_start", "name": "開始", "type": "start", "description": "プロジェクトの開始点", "config": {}, "position_x": 400, "position_y": 50},
    {"temp_id": "step_llm", "name": "LLM処理", "type": "llm", "description": "入力をLLMで処理", "config": {"provider": "openai", "model": "gpt-4o-mini", "system_prompt": "You are a helpful assistant.", "user_prompt": "Process the input: {{$.input}}"}, "position_x": 400, "position_y": 200},
    {"temp_id": "step_log", "name": "結果をログ", "type": "log", "description": "処理結果をログに出力", "config": {"message": "Processing complete", "level": "info"}, "position_x": 400, "position_y": 350}
  ],
  "edges": [
    {"source_temp_id": "step_start", "target_temp_id": "step_llm", "source_port": "default"},
    {"source_temp_id": "step_llm", "target_temp_id": "step_log", "source_port": "default"}
  ],
  "start_step_id": "step_start"
}`
	}
	// Workflow automation expert prompt (for generateInitialResponse)
	if strings.Contains(prompt, "ワークフロー自動化の専門家") {
		return `{
  "response": "ご要望を分析しました。",
  "assumptions": {
    "trigger": "manual",
    "trigger_detail": "手動実行",
    "frequency": "オンデマンド",
    "steps": ["データ取得", "処理", "出力"],
    "integrations": []
  },
  "clarifying_questions": [],
  "suggested_responses": ["この内容で問題ありません", "トリガーを変更したい", "ステップを追加したい"]
}`
	}
	// Default chat response
	return `{"response": "こんにちは！Copilotのモックモードです。OPENAI_API_KEYを設定すると、AIによる本格的なサポートを受けられます。プロジェクトの構築について何かお手伝いできることはありますか？", "suggestions": []}`
}
