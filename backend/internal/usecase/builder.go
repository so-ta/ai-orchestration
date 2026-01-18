package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// BuilderUsecase handles AI Workflow Builder operations
type BuilderUsecase struct {
	sessionRepo repository.BuilderSessionRepository
	projectRepo repository.ProjectRepository
	blockRepo   repository.BlockDefinitionRepository
	httpClient  *http.Client
	apiKey      string
	baseURL     string
}

// NewBuilderUsecase creates a new BuilderUsecase
func NewBuilderUsecase(
	sessionRepo repository.BuilderSessionRepository,
	projectRepo repository.ProjectRepository,
	blockRepo repository.BlockDefinitionRepository,
) *BuilderUsecase {
	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &BuilderUsecase{
		sessionRepo: sessionRepo,
		projectRepo: projectRepo,
		blockRepo:   blockRepo,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		apiKey:  os.Getenv("OPENAI_API_KEY"),
		baseURL: baseURL,
	}
}

// ============================================================================
// Input/Output types
// ============================================================================

// StartBuilderSessionInput represents input for starting a builder session
type StartBuilderSessionInput struct {
	TenantID      uuid.UUID
	UserID        string
	InitialPrompt string
}

// StartBuilderSessionOutput represents output for starting a builder session
type StartBuilderSessionOutput struct {
	Session            *domain.BuilderSession
	Message            *domain.BuilderMessage
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
	Session            *domain.BuilderSession
	Message            *domain.BuilderMessage
	SuggestedQuestions []string
	Phase              domain.HearingPhase
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
// Session Management
// ============================================================================

// StartSession starts a new builder session with an initial prompt
func (u *BuilderUsecase) StartSession(ctx context.Context, input StartBuilderSessionInput) (*StartBuilderSessionOutput, error) {
	// Create new session
	session := domain.NewBuilderSession(input.TenantID, input.UserID)

	// Save session
	if err := u.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("create builder session: %w", err)
	}

	// Add user message
	userMsg := domain.NewBuilderMessage(session.ID, "user", input.InitialPrompt)
	phase := domain.HearingPhasePurpose
	userMsg.Phase = &phase
	if err := u.sessionRepo.AddMessage(ctx, userMsg); err != nil {
		return nil, fmt.Errorf("add user message: %w", err)
	}

	// Generate initial assistant response
	response, suggestedQuestions, err := u.generateInitialResponse(ctx, input.InitialPrompt)
	if err != nil {
		return nil, fmt.Errorf("generate initial response: %w", err)
	}

	// Add assistant message
	assistantMsg := domain.NewBuilderMessage(session.ID, "assistant", response)
	assistantMsg.Phase = &phase
	if len(suggestedQuestions) > 0 {
		questionsJSON, _ := json.Marshal(suggestedQuestions)
		assistantMsg.SuggestedQuestions = questionsJSON
	}
	if err := u.sessionRepo.AddMessage(ctx, assistantMsg); err != nil {
		return nil, fmt.Errorf("add assistant message: %w", err)
	}

	// Update session progress
	session.SetPhase(domain.HearingPhasePurpose, domain.GetPhaseProgress(domain.HearingPhasePurpose))
	if err := u.sessionRepo.Update(ctx, session); err != nil {
		return nil, fmt.Errorf("update session: %w", err)
	}

	return &StartBuilderSessionOutput{
		Session:            session,
		Message:            assistantMsg,
		SuggestedQuestions: suggestedQuestions,
	}, nil
}

// GetSession retrieves a builder session with messages
func (u *BuilderUsecase) GetSession(ctx context.Context, tenantID, sessionID uuid.UUID) (*domain.BuilderSession, error) {
	return u.sessionRepo.GetWithMessages(ctx, tenantID, sessionID)
}

// ListSessions lists all builder sessions for a user
func (u *BuilderUsecase) ListSessions(ctx context.Context, tenantID uuid.UUID, userID string) ([]*domain.BuilderSession, int, error) {
	return u.sessionRepo.ListByUser(ctx, tenantID, userID, repository.BuilderSessionFilter{
		Limit: 50,
	})
}

// FinalizeSession marks a session as completed
func (u *BuilderUsecase) FinalizeSession(ctx context.Context, tenantID, sessionID uuid.UUID) error {
	session, err := u.sessionRepo.GetByID(ctx, tenantID, sessionID)
	if err != nil {
		return err
	}

	session.Complete()
	return u.sessionRepo.Update(ctx, session)
}

// DeleteSession deletes a builder session
func (u *BuilderUsecase) DeleteSession(ctx context.Context, tenantID, sessionID uuid.UUID) error {
	return u.sessionRepo.Delete(ctx, tenantID, sessionID)
}

// ============================================================================
// Hearing Process
// ============================================================================

// ProcessHearing processes a hearing message and advances the conversation
func (u *BuilderUsecase) ProcessHearing(ctx context.Context, input HearingInput) (*HearingOutput, error) {
	// Get session
	session, err := u.sessionRepo.GetWithMessages(ctx, input.TenantID, input.SessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	// Add user message
	userMsg := domain.NewBuilderMessage(session.ID, "user", input.Message)
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
	assistantMsg := domain.NewBuilderMessage(session.ID, "assistant", llmOutput.Response)
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
		Complete:           nextPhase == domain.HearingPhaseCompleted,
	}, nil
}

// ============================================================================
// Workflow Construction
// ============================================================================

// ConstructWorkflow constructs a workflow from the session spec
func (u *BuilderUsecase) ConstructWorkflow(ctx context.Context, input ConstructInput) (*ConstructOutput, error) {
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
	session.SetStatus(domain.BuilderSessionStatusReviewing)
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
// Workflow Refinement
// ============================================================================

// RefineWorkflow refines a workflow based on user feedback
func (u *BuilderUsecase) RefineWorkflow(ctx context.Context, input RefineInput) (*RefineOutput, error) {
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
	refineMsg := domain.NewBuilderMessage(session.ID, "user", input.Feedback)
	if err := u.sessionRepo.AddMessage(ctx, refineMsg); err != nil {
		return nil, fmt.Errorf("add message: %w", err)
	}

	// Update session status
	session.SetStatus(domain.BuilderSessionStatusRefining)
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
// LLM Integration (private methods)
// ============================================================================

// llmHearingOutput represents the output from hearing LLM call
type llmHearingOutput struct {
	Response           string
	ExtractedData      map[string]interface{}
	SuggestedQuestions []string
	NextPhase          domain.HearingPhase
	Confidence         string
}

// generateInitialResponse generates the initial assistant response
func (u *BuilderUsecase) generateInitialResponse(ctx context.Context, initialPrompt string) (string, []string, error) {
	// For now, return a templated response
	// In production, this would call an LLM
	response := fmt.Sprintf(`ワークフローの作成をお手伝いします。

「%s」についてお聞かせください。

まず、このワークフローの目的を確認させてください：
- 最終的に何が達成されれば成功と言えますか？
- 成果物はありますか？（レポート、通知、データ更新など）`, initialPrompt)

	suggestedQuestions := []string{
		"このワークフローで達成したいゴールは何ですか？",
		"成果物の形式は何ですか？（PDF、Excel、Slack通知など）",
	}

	return response, suggestedQuestions, nil
}

// processHearingWithLLM processes a hearing message with LLM
func (u *BuilderUsecase) processHearingWithLLM(ctx context.Context, session *domain.BuilderSession, message string, blocks []*domain.BlockDefinition) (*llmHearingOutput, error) {
	// For now, return a templated response based on phase
	// In production, this would call an LLM with proper prompts

	currentPhase := session.HearingPhase
	nextPhase := domain.NextPhase(currentPhase)

	var response string
	var suggestedQuestions []string

	switch nextPhase {
	case domain.HearingPhaseConditions:
		response = "承知しました。次に、ワークフローの開始条件と終了条件を確認させてください。\n\n- いつ・どのようなタイミングでこのフローが始まりますか？\n- どの状態になったら完了とみなしますか？"
		suggestedQuestions = []string{
			"手動で開始しますか、それとも定期実行ですか？",
			"完了時に何が出力されますか？",
		}
	case domain.HearingPhaseActors:
		response = "開始・終了条件を理解しました。次に、このワークフローに関わる人物や役割を確認させてください。\n\n- 作業を実行する担当者は誰ですか？\n- 承認やレビューが必要ですか？"
		suggestedQuestions = []string{
			"承認者は必要ですか？",
			"複数人が関わりますか？",
		}
	case domain.HearingPhaseFrequency:
		response = "関係者を把握しました。次に、実行頻度と期限について確認させてください。\n\n- どのくらいの頻度で実行されますか？\n- 期限や締切はありますか？"
		suggestedQuestions = []string{
			"毎日/毎週/毎月のいずれですか？",
			"期限はありますか？",
		}
	case domain.HearingPhaseIntegrations:
		response = "実行頻度を確認しました。次に、使用するツールやシステムを確認させてください。\n\n- すでに使っているツールはありますか？（Slack、メール、Google Drive、Notionなど）"
		suggestedQuestions = []string{
			"通知先はSlackですか、メールですか？",
			"データの保存先はありますか？",
		}
	case domain.HearingPhasePainPoints:
		response = "ツールを把握しました。最後に、現在の課題や改善したいポイントを教えてください。\n\n- なぜワークフローを作りたいのですか？\n- 現在うまくいっていない点はありますか？"
		suggestedQuestions = []string{
			"手動作業で困っていることはありますか？",
			"自動化したい部分はどこですか？",
		}
	case domain.HearingPhaseConfirmation:
		response = "ありがとうございます。いただいた情報をまとめます。\n\n以下の前提でワークフローを構築してよろしいでしょうか？\n\n（詳細なサマリーがここに入ります）\n\n問題なければ「構築を開始」ボタンを押してください。修正があればお知らせください。"
		suggestedQuestions = []string{
			"この内容で問題ありません",
			"修正したい点があります",
		}
	case domain.HearingPhaseCompleted:
		response = "ヒアリングが完了しました。ワークフローの構築を開始できます。"
		suggestedQuestions = []string{}
	default:
		response = "ご回答ありがとうございます。続けてお聞かせください。"
		suggestedQuestions = []string{}
	}

	return &llmHearingOutput{
		Response:           response,
		ExtractedData:      nil, // Would be extracted by LLM in production
		SuggestedQuestions: suggestedQuestions,
		NextPhase:          nextPhase,
		Confidence:         "high",
	}, nil
}

// mapStepsToBlocks maps workflow spec steps to available blocks
func (u *BuilderUsecase) mapStepsToBlocks(ctx context.Context, spec *domain.WorkflowSpec, blocks []*domain.BlockDefinition) ([]domain.StepMappingResult, []domain.CustomRequirement, error) {
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
			// Check if we have slack or discord blocks
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
			// Would need more sophisticated matching in production
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
func (u *BuilderUsecase) processRefinement(ctx context.Context, session *domain.BuilderSession, project *domain.Project, feedback string) ([]string, error) {
	// In production, this would use LLM to interpret feedback and apply changes
	// For now, return a placeholder
	return []string{"フィードバックを受け付けました"}, nil
}

// ============================================================================
// Helper functions
// ============================================================================

// mergeExtractedData merges extracted data into the workflow spec
func mergeExtractedData(spec *domain.WorkflowSpec, data map[string]interface{}) {
	// Simple merge - in production this would be more sophisticated
	if name, ok := data["name"].(string); ok {
		spec.Name = name
	}
	if desc, ok := data["description"].(string); ok {
		spec.Description = desc
	}
	if purpose, ok := data["purpose"].(string); ok {
		spec.Purpose = purpose
	}
	// ... additional fields would be merged here
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
