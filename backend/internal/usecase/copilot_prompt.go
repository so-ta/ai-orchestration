package usecase

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
	"github.com/souta/ai-orchestration/internal/repository"
)

// ============================================================================
// Phase 4: Dynamic Prompt Optimization
// ============================================================================

// CopilotIntent represents the classified intent of a user message
type CopilotIntent string

const (
	// IntentCreate - User wants to create new steps/workflows
	IntentCreate CopilotIntent = "create"
	// IntentExplain - User wants to understand something
	IntentExplain CopilotIntent = "explain"
	// IntentEnhance - User wants to modify/improve existing workflow
	IntentEnhance CopilotIntent = "enhance"
	// IntentSearch - User is searching for blocks or documentation
	IntentSearch CopilotIntent = "search"
	// IntentDebug - User is debugging or troubleshooting
	IntentDebug CopilotIntent = "debug"
	// IntentGeneral - General question or chat
	IntentGeneral CopilotIntent = "general"
)

// IntentClassifier classifies user messages into intents
type IntentClassifier struct{}

// NewIntentClassifier creates a new IntentClassifier
func NewIntentClassifier() *IntentClassifier {
	return &IntentClassifier{}
}

// Classify classifies a user message into an intent
// Uses rule-based classification for efficiency (no LLM call needed)
func (ic *IntentClassifier) Classify(message string) CopilotIntent {
	msg := strings.ToLower(message)

	// Create intent patterns
	createPatterns := []string{
		"追加", "作成", "作って", "つくって", "add", "create", "build", "make",
		"ブロックを", "ステップを", "ワークフローを",
		"新しい", "新規",
	}

	// Enhance intent patterns
	enhancePatterns := []string{
		"変更", "修正", "更新", "改善", "最適化", "リファクタ",
		"modify", "change", "update", "improve", "optimize", "refactor",
		"直して", "なおして", "削除", "消して",
	}

	// Explain intent patterns
	explainPatterns := []string{
		"説明", "教えて", "何", "どう", "なぜ", "どのように",
		"explain", "what", "how", "why", "tell me", "describe",
		"仕組み", "動作", "使い方",
	}

	// Search intent patterns
	searchPatterns := []string{
		"検索", "探して", "見つけて", "どこ", "ある",
		"search", "find", "locate", "where", "look for",
		"ブロック一覧", "リスト",
	}

	// Debug intent patterns
	debugPatterns := []string{
		"エラー", "失敗", "動かない", "問題", "バグ",
		"error", "fail", "bug", "issue", "problem", "fix", "debug",
		"うまくいかない", "できない",
	}

	// Check patterns in order of specificity
	for _, pattern := range createPatterns {
		if strings.Contains(msg, pattern) {
			return IntentCreate
		}
	}

	for _, pattern := range enhancePatterns {
		if strings.Contains(msg, pattern) {
			return IntentEnhance
		}
	}

	for _, pattern := range debugPatterns {
		if strings.Contains(msg, pattern) {
			return IntentDebug
		}
	}

	for _, pattern := range searchPatterns {
		if strings.Contains(msg, pattern) {
			return IntentSearch
		}
	}

	for _, pattern := range explainPatterns {
		if strings.Contains(msg, pattern) {
			return IntentExplain
		}
	}

	return IntentGeneral
}

// ExtractBlockTypes extracts potential block types mentioned in the message
func (ic *IntentClassifier) ExtractBlockTypes(message string) []string {
	msg := strings.ToLower(message)

	// Known block type keywords
	blockKeywords := map[string]string{
		"slack":     "slack",
		"discord":   "discord",
		"llm":       "llm",
		"ai":        "llm",
		"gpt":       "llm",
		"claude":    "llm",
		"http":      "http",
		"api":       "http",
		"condition": "condition",
		"条件":        "condition",
		"分岐":        "condition",
		"switch":    "switch",
		"loop":      "loop",
		"ループ":       "loop",
		"繰り返し":      "loop",
		"email":     "email",
		"メール":       "email",
		"notion":    "notion",
		"github":    "github",
		"webhook":   "webhook-trigger",
		"schedule":  "schedule-trigger",
		"定期":        "schedule-trigger",
		"スケジュール":    "schedule-trigger",
	}

	var extracted []string
	seen := make(map[string]bool)

	for keyword, blockType := range blockKeywords {
		if strings.Contains(msg, keyword) && !seen[blockType] {
			extracted = append(extracted, blockType)
			seen[blockType] = true
		}
	}

	return extracted
}

// ============================================================================
// ContextBuilder - Builds context based on intent
// ============================================================================

// ContextBuilder builds context for the Copilot system prompt
type ContextBuilder struct {
	blockRepo   repository.BlockDefinitionRepository
	projectRepo repository.ProjectRepository
}

// NewContextBuilder creates a new ContextBuilder
func NewContextBuilder(
	blockRepo repository.BlockDefinitionRepository,
	projectRepo repository.ProjectRepository,
) *ContextBuilder {
	return &ContextBuilder{
		blockRepo:   blockRepo,
		projectRepo: projectRepo,
	}
}

// ContextData holds the built context
type ContextData struct {
	Intent           CopilotIntent
	RelevantBlocks   []*domain.BlockDefinition
	WorkflowState    *WorkflowState
	DocumentSnippets []string
}

// WorkflowState represents the current workflow state for context
type WorkflowState struct {
	ProjectID   uuid.UUID
	Name        string
	StepCount   int
	HasTrigger  bool
	TriggerType string
	StepTypes   []string
}

// Build builds context based on the intent and message
func (cb *ContextBuilder) Build(
	ctx context.Context,
	tenantID uuid.UUID,
	projectID *uuid.UUID,
	intent CopilotIntent,
	mentionedBlocks []string,
) (*ContextData, error) {
	data := &ContextData{
		Intent: intent,
	}

	// Get relevant blocks based on intent and mentioned blocks
	blocks, err := cb.getRelevantBlocks(ctx, tenantID, intent, mentionedBlocks)
	if err != nil {
		return nil, fmt.Errorf("get relevant blocks: %w", err)
	}
	data.RelevantBlocks = blocks

	// Get workflow state if project context exists
	if projectID != nil {
		state, err := cb.getWorkflowState(ctx, tenantID, *projectID)
		if err == nil {
			data.WorkflowState = state
		}
	}

	return data, nil
}

// getRelevantBlocks retrieves blocks relevant to the intent
func (cb *ContextBuilder) getRelevantBlocks(
	ctx context.Context,
	tenantID uuid.UUID,
	intent CopilotIntent,
	mentionedBlocks []string,
) ([]*domain.BlockDefinition, error) {
	// Get all enabled blocks
	allBlocks, err := cb.blockRepo.List(ctx, &tenantID, repository.BlockDefinitionFilter{
		EnabledOnly: true,
	})
	if err != nil {
		return nil, err
	}

	// If specific blocks are mentioned, prioritize those
	if len(mentionedBlocks) > 0 {
		var relevant []*domain.BlockDefinition
		mentionedSet := make(map[string]bool)
		for _, slug := range mentionedBlocks {
			mentionedSet[slug] = true
		}

		// Add mentioned blocks first
		for _, b := range allBlocks {
			if mentionedSet[b.Slug] {
				relevant = append(relevant, b)
			}
		}

		// For create intent, also include trigger blocks if not mentioned
		if intent == IntentCreate {
			for _, b := range allBlocks {
				if b.Category == "trigger" && !mentionedSet[b.Slug] {
					relevant = append(relevant, b)
				}
			}
		}

		// Limit to 10 blocks max
		if len(relevant) > 10 {
			return relevant[:10], nil
		}
		return relevant, nil
	}

	// Filter based on intent
	var relevant []*domain.BlockDefinition

	switch intent {
	case IntentCreate:
		// For create, include trigger blocks and common blocks
		priorityCategories := map[string]bool{
			"trigger":     true,
			"integration": true,
			"ai":          true,
		}
		for _, b := range allBlocks {
			if priorityCategories[string(b.Category)] {
				relevant = append(relevant, b)
			}
		}

	case IntentEnhance, IntentDebug:
		// For enhance/debug, include control flow and utility blocks
		priorityCategories := map[string]bool{
			"control": true,
			"utility": true,
			"data":    true,
		}
		for _, b := range allBlocks {
			if priorityCategories[string(b.Category)] {
				relevant = append(relevant, b)
			}
		}

	case IntentSearch:
		// For search, include all blocks but limit count
		relevant = allBlocks

	default:
		// For general/explain, include a representative sample
		categoryCount := make(map[string]int)
		for _, b := range allBlocks {
			cat := string(b.Category)
			if categoryCount[cat] < 2 {
				relevant = append(relevant, b)
				categoryCount[cat]++
			}
		}
	}

	// Limit to 15 blocks max
	if len(relevant) > 15 {
		return relevant[:15], nil
	}
	return relevant, nil
}

// getWorkflowState retrieves the current workflow state
func (cb *ContextBuilder) getWorkflowState(
	ctx context.Context,
	tenantID uuid.UUID,
	projectID uuid.UUID,
) (*WorkflowState, error) {
	project, err := cb.projectRepo.GetWithStepsAndEdges(ctx, tenantID, projectID)
	if err != nil {
		return nil, err
	}

	state := &WorkflowState{
		ProjectID: project.ID,
		Name:      project.Name,
		StepCount: len(project.Steps),
	}

	stepTypes := make(map[string]bool)
	for _, step := range project.Steps {
		stepType := string(step.Type)
		stepTypes[stepType] = true

		// Check for trigger
		if strings.Contains(stepType, "trigger") || stepType == "start" {
			state.HasTrigger = true
			state.TriggerType = stepType
		}
	}

	for t := range stepTypes {
		state.StepTypes = append(state.StepTypes, t)
	}

	return state, nil
}

// ============================================================================
// DynamicPromptGenerator - Generates optimized prompts
// ============================================================================

// DynamicPromptGenerator generates dynamic system prompts based on context
type DynamicPromptGenerator struct{}

// NewDynamicPromptGenerator creates a new DynamicPromptGenerator
func NewDynamicPromptGenerator() *DynamicPromptGenerator {
	return &DynamicPromptGenerator{}
}

// GenerateSystemPrompt generates an optimized system prompt based on context
func (dpg *DynamicPromptGenerator) GenerateSystemPrompt(data *ContextData) string {
	var sb strings.Builder

	// Base identity
	sb.WriteString("You are Copilot, an AI assistant for the AI Orchestration platform.\n\n")

	// Intent-specific instructions
	sb.WriteString(dpg.getIntentInstructions(data.Intent))

	// Relevant blocks section (only if blocks are available)
	if len(data.RelevantBlocks) > 0 {
		sb.WriteString("\n## Available Blocks\n")
		for _, b := range data.RelevantBlocks {
			desc := b.Description
			if len(desc) > 100 {
				desc = desc[:100] + "..."
			}
			sb.WriteString(fmt.Sprintf("- **%s** (%s): %s\n", b.Slug, b.Category, desc))
		}
	}

	// Workflow state section (only if available)
	if data.WorkflowState != nil {
		sb.WriteString("\n## Current Workflow State\n")
		sb.WriteString(fmt.Sprintf("- Name: %s\n", data.WorkflowState.Name))
		sb.WriteString(fmt.Sprintf("- Steps: %d\n", data.WorkflowState.StepCount))
		sb.WriteString(fmt.Sprintf("- Has Trigger: %v\n", data.WorkflowState.HasTrigger))
		if data.WorkflowState.TriggerType != "" {
			sb.WriteString(fmt.Sprintf("- Trigger Type: %s\n", data.WorkflowState.TriggerType))
		}
	}

	// Core rules (always include, but condensed)
	sb.WriteString(dpg.getCoreRules())

	return sb.String()
}

// getIntentInstructions returns intent-specific instructions
func (dpg *DynamicPromptGenerator) getIntentInstructions(intent CopilotIntent) string {
	switch intent {
	case IntentCreate:
		return `## Your Task: Create
Help the user create new workflow elements. Follow this pattern:
1. Call get_block_schema to understand block configuration
2. Call create_workflow_structure to create steps with proper config
3. ALWAYS include config with required fields

**CRITICAL**: Always call get_block_schema BEFORE creating any step.
`

	case IntentEnhance:
		return `## Your Task: Enhance/Modify
Help the user modify or improve their workflow.
1. First get_workflow to understand current state
2. Use update_step for modifications
3. Use delete_step/delete_edge carefully
4. Explain changes after making them
`

	case IntentExplain:
		return `## Your Task: Explain
Help the user understand workflow concepts.
1. Use search_documentation for platform features
2. Use get_block_schema for block details
3. Provide clear, concise explanations
`

	case IntentSearch:
		return `## Your Task: Search
Help the user find blocks or information.
1. Use search_blocks for semantic block discovery
2. Use list_blocks for full block listing
3. Use search_documentation for platform docs
`

	case IntentDebug:
		return `## Your Task: Debug
Help the user troubleshoot issues.
1. Use validate_workflow to check structure
2. Review step configurations
3. Suggest fixes with clear explanations
`

	default:
		return `## Your Task: Assist
Help the user with their request.
1. Use appropriate tools based on the request
2. Be proactive with suggestions
3. Explain your actions clearly
`
	}
}

// getCoreRules returns condensed core rules
func (dpg *DynamicPromptGenerator) getCoreRules() string {
	return `
## Core Rules
1. Use tools immediately - don't just describe what you'll do
2. ALWAYS call get_block_schema before creating steps
3. Include config with all required fields in steps
4. Respond in the same language as the user
5. Be concise - show results, not lengthy explanations

## Config Generation (CRITICAL)
Before creating any step:
1. get_block_schema(slug) -> get required_fields and defaults
2. create_workflow_structure with complete config

Block defaults are auto-merged, but you MUST provide user-specific values.
`
}

// ============================================================================
// Helper functions for extracting service names from messages
// ============================================================================

// extractServiceNames extracts potential service names from a message
func extractServiceNames(message string) []string {
	// Common service name patterns
	servicePatterns := []string{
		`(?i)(slack|discord|notion|github|google\s*sheets?|gmail|email)`,
		`(?i)(stripe|twilio|aws|gcp|azure|openai|anthropic)`,
		`(?i)(freee|salesforce|hubspot|shopify|zendesk)`,
	}

	var services []string
	seen := make(map[string]bool)

	for _, pattern := range servicePatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllString(message, -1)
		for _, match := range matches {
			normalized := strings.ToLower(strings.TrimSpace(match))
			normalized = strings.ReplaceAll(normalized, " ", "-")
			if !seen[normalized] {
				services = append(services, normalized)
				seen[normalized] = true
			}
		}
	}

	return services
}
