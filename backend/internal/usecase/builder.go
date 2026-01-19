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
	analysisPhase := domain.HearingPhaseAnalysis
	userMsg.Phase = &analysisPhase
	if err := u.sessionRepo.AddMessage(ctx, userMsg); err != nil {
		return nil, fmt.Errorf("add user message: %w", err)
	}

	// Generate initial assistant response (analysis phase)
	// This generates both the analysis AND the proposal with assumptions
	response, suggestedQuestions, err := u.generateInitialResponse(ctx, input.InitialPrompt)
	if err != nil {
		return nil, fmt.Errorf("generate initial response: %w", err)
	}

	// After initial analysis, move to proposal phase
	// The response already contains assumptions, so we're ready for user confirmation
	proposalPhase := domain.HearingPhaseProposal

	// Add assistant message
	assistantMsg := domain.NewBuilderMessage(session.ID, "assistant", response)
	assistantMsg.Phase = &proposalPhase
	if len(suggestedQuestions) > 0 {
		questionsJSON, _ := json.Marshal(suggestedQuestions)
		assistantMsg.SuggestedQuestions = questionsJSON
	}
	if err := u.sessionRepo.AddMessage(ctx, assistantMsg); err != nil {
		return nil, fmt.Errorf("add assistant message: %w", err)
	}

	// Update session progress to proposal phase (analysis is done in generateInitialResponse)
	session.SetPhase(domain.HearingPhaseProposal, domain.GetPhaseProgress(domain.HearingPhaseProposal))
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
// New 3-phase approach: AI analyzes first, then proposes assumptions
func (u *BuilderUsecase) generateInitialResponse(ctx context.Context, initialPrompt string) (string, []string, error) {
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
		Response           string   `json:"response"`
		Assumptions        struct {
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
func (u *BuilderUsecase) fallbackInitialResponse(initialPrompt string) (string, []string) {
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

// callLLM calls the LLM API
func (u *BuilderUsecase) callLLM(ctx context.Context, prompt string) (string, error) {
	if u.apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY is not set")
	}

	reqBody := map[string]interface{}{
		"model": "gpt-4o-mini",
		"messages": []map[string]string{
			{"role": "system", "content": "You are an AI workflow automation expert. Always respond with valid JSON."},
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

	slog.Info("calling LLM API", "url", u.baseURL+"/chat/completions", "model", "gpt-4o-mini")

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

	slog.Info("LLM response received", "length", len(result.Choices[0].Message.Content))
	return result.Choices[0].Message.Content, nil
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

// processHearingWithLLM processes a hearing message with LLM
// New 3-phase approach: analysis -> proposal -> completed
func (u *BuilderUsecase) processHearingWithLLM(ctx context.Context, session *domain.BuilderSession, message string, blocks []*domain.BlockDefinition) (*llmHearingOutput, error) {
	// For now, return a templated response based on phase
	// In production, this would call the system workflow (dogfooding)

	currentPhase := session.HearingPhase
	nextPhase := domain.NextPhase(currentPhase)

	var response string
	var suggestedQuestions []string

	switch nextPhase {
	case domain.HearingPhaseProposal:
		// After analysis, move to proposal phase
		response = "ご要望を分析しました。\n\n以下の前提でワークフローを構築することを提案します。\n\n【想定した条件】\n- トリガー: 手動実行\n- 主要なステップ: データ取得 → 処理 → 通知\n\n問題なければ「確認して構築」ボタンを押してください。修正があればお知らせください。"
		suggestedQuestions = []string{
			"この内容で問題ありません",
			"修正したい点があります",
		}
	case domain.HearingPhaseCompleted:
		// After proposal confirmation, move to completed
		response = "ヒアリングが完了しました。ワークフローの構築を開始できます。"
		suggestedQuestions = []string{}
	default:
		// Stay in analysis phase, gather more information
		response = "ご回答ありがとうございます。引き続き分析中です。"
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
