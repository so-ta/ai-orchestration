package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// CopilotSessionMode represents the mode of a copilot session
type CopilotSessionMode string

const (
	CopilotSessionModeCreate  CopilotSessionMode = "create"  // Create new workflow
	CopilotSessionModeEnhance CopilotSessionMode = "enhance" // Enhance existing workflow
	CopilotSessionModeExplain CopilotSessionMode = "explain" // Explain workflow
)

// CopilotSessionStatus represents the status of a copilot session
type CopilotSessionStatus string

const (
	CopilotSessionStatusHearing   CopilotSessionStatus = "hearing"
	CopilotSessionStatusBuilding  CopilotSessionStatus = "building"
	CopilotSessionStatusReviewing CopilotSessionStatus = "reviewing"
	CopilotSessionStatusRefining  CopilotSessionStatus = "refining"
	CopilotSessionStatusCompleted CopilotSessionStatus = "completed"
	CopilotSessionStatusAbandoned CopilotSessionStatus = "abandoned"
)

// CopilotPhase represents the current phase of the hearing process
// 3-phase approach: AI thinks first → proposes → confirms
type CopilotPhase string

const (
	CopilotPhaseAnalysis  CopilotPhase = "analysis"  // AI is analyzing and thinking
	CopilotPhaseProposal  CopilotPhase = "proposal"  // AI proposes spec and clarifying questions
	CopilotPhaseCompleted CopilotPhase = "completed" // Hearing completed
)

// ClarifyingPoint represents a question that needs user clarification
type ClarifyingPoint struct {
	ID       string   `json:"id"`
	Question string   `json:"question"`
	Options  []string `json:"options,omitempty"`
	Required bool     `json:"required"`
	Answer   string   `json:"answer,omitempty"`
}

// CopilotSession represents a copilot session for AI-assisted workflow creation/enhancement
type CopilotSession struct {
	ID       uuid.UUID `json:"id"`
	TenantID uuid.UUID `json:"tenant_id"`
	UserID   string    `json:"user_id"`

	// Context: which workflow this session is scoped to (NULL for global create)
	ContextProjectID *uuid.UUID `json:"context_project_id,omitempty"`

	// Mode: create (new workflow), enhance (improve existing), explain (understand)
	Mode CopilotSessionMode `json:"mode"`

	// Title: derived from first user message
	Title string `json:"title,omitempty"`

	// Status
	Status          CopilotSessionStatus `json:"status"`
	HearingPhase    CopilotPhase         `json:"hearing_phase"`
	HearingProgress int                  `json:"hearing_progress"` // 0-100

	// Generated artifacts
	Spec      json.RawMessage `json:"spec,omitempty"`       // WorkflowSpec as JSON
	ProjectID *uuid.UUID      `json:"project_id,omitempty"` // Generated/modified project

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Loaded relations
	Messages []CopilotMessage `json:"messages,omitempty"`
}

// CopilotMessage represents a message in a copilot session
type CopilotMessage struct {
	ID        uuid.UUID `json:"id"`
	SessionID uuid.UUID `json:"session_id"`
	Role      string    `json:"role"` // "user", "assistant", "system"
	Content   string    `json:"content"`

	// Metadata
	Phase              *CopilotPhase   `json:"phase,omitempty"`
	ExtractedData      json.RawMessage `json:"extracted_data,omitempty"`
	SuggestedQuestions json.RawMessage `json:"suggested_questions,omitempty"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
}

// NewCopilotSession creates a new copilot session
func NewCopilotSession(tenantID uuid.UUID, userID string, contextProjectID *uuid.UUID, mode CopilotSessionMode) *CopilotSession {
	now := time.Now().UTC()
	return &CopilotSession{
		ID:               uuid.New(),
		TenantID:         tenantID,
		UserID:           userID,
		ContextProjectID: contextProjectID,
		Mode:             mode,
		Status:           CopilotSessionStatusHearing,
		HearingPhase:     CopilotPhaseAnalysis,
		HearingProgress:  0,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// NewCopilotMessage creates a new copilot message
func NewCopilotMessage(sessionID uuid.UUID, role, content string) *CopilotMessage {
	return &CopilotMessage{
		ID:        uuid.New(),
		SessionID: sessionID,
		Role:      role,
		Content:   content,
		CreatedAt: time.Now().UTC(),
	}
}

// AddMessage adds a message to the session
func (s *CopilotSession) AddMessage(role, content string) *CopilotMessage {
	msg := NewCopilotMessage(s.ID, role, content)
	s.Messages = append(s.Messages, *msg)
	s.UpdatedAt = time.Now().UTC()
	return msg
}

// SetPhase updates the hearing phase and progress
func (s *CopilotSession) SetPhase(phase CopilotPhase, progress int) {
	s.HearingPhase = phase
	s.HearingProgress = progress
	s.UpdatedAt = time.Now().UTC()
}

// SetSpec sets the workflow spec
func (s *CopilotSession) SetSpec(spec json.RawMessage) {
	s.Spec = spec
	s.UpdatedAt = time.Now().UTC()
}

// SetProjectID sets the generated project ID
func (s *CopilotSession) SetProjectID(projectID uuid.UUID) {
	s.ProjectID = &projectID
	s.UpdatedAt = time.Now().UTC()
}

// SetStatus sets the session status
func (s *CopilotSession) SetStatus(status CopilotSessionStatus) {
	s.Status = status
	s.UpdatedAt = time.Now().UTC()
}

// SetTitle sets the session title (typically from first user message)
func (s *CopilotSession) SetTitle(title string) {
	// Truncate if too long (max 200 chars)
	if len(title) > 200 {
		title = title[:197] + "..."
	}
	s.Title = title
	s.UpdatedAt = time.Now().UTC()
}

// Complete marks the session as completed
func (s *CopilotSession) Complete() {
	s.Status = CopilotSessionStatusCompleted
	s.HearingPhase = CopilotPhaseCompleted
	s.UpdatedAt = time.Now().UTC()
}

// Abandon marks the session as abandoned
func (s *CopilotSession) Abandon() {
	s.Status = CopilotSessionStatusAbandoned
	s.UpdatedAt = time.Now().UTC()
}

// IsActive returns true if the session is still active
func (s *CopilotSession) IsActive() bool {
	return s.Status != CopilotSessionStatusCompleted && s.Status != CopilotSessionStatusAbandoned
}

// GetPhaseProgress returns the default progress for a phase
// 3-phase: analysis (0-50%) → proposal (50-90%) → completed (100%)
func GetPhaseProgress(phase CopilotPhase) int {
	switch phase {
	case CopilotPhaseAnalysis:
		return 30
	case CopilotPhaseProposal:
		return 70
	case CopilotPhaseCompleted:
		return 100
	default:
		return 0
	}
}

// NextPhase returns the next copilot phase
func NextPhase(current CopilotPhase) CopilotPhase {
	switch current {
	case CopilotPhaseAnalysis:
		return CopilotPhaseProposal
	case CopilotPhaseProposal:
		return CopilotPhaseCompleted
	default:
		return CopilotPhaseCompleted
	}
}

// ============================================================================
// WorkflowSpec types for JSON serialization
// ============================================================================

// WorkflowSpec represents the internal DSL for workflow specification
type WorkflowSpec struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	Purpose         string            `json:"purpose"`
	SuccessCriteria []string          `json:"success_criteria,omitempty"`
	BusinessDomain  string            `json:"business_domain,omitempty"`
	Trigger         *TriggerSpec      `json:"trigger,omitempty"`
	Completion      *CompletionSpec   `json:"completion,omitempty"`
	Actors          []ActorSpec       `json:"actors,omitempty"`
	Steps           []StepSpec        `json:"steps,omitempty"`
	Integrations    []IntegrationSpec `json:"integrations,omitempty"`
	Constraints     *ConstraintSpec   `json:"constraints,omitempty"`
	Assumptions     []AssumptionSpec  `json:"assumptions,omitempty"`
	HeardAt         *time.Time        `json:"heard_at,omitempty"`
	Version         int               `json:"version"`
}

// TriggerSpec represents the workflow trigger specification
type TriggerSpec struct {
	Type        string `json:"type"` // manual, schedule, webhook, event
	Schedule    string `json:"schedule,omitempty"`
	EventSource string `json:"event_source,omitempty"`
	Description string `json:"description"`
}

// CompletionSpec represents the workflow completion specification
type CompletionSpec struct {
	Description string       `json:"description"`
	Outputs     []OutputSpec `json:"outputs,omitempty"`
}

// OutputSpec represents an output specification
type OutputSpec struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // document, notification, data, approval, other
	Format      string `json:"format,omitempty"`
	Destination string `json:"destination,omitempty"`
}

// ActorSpec represents an actor specification
type ActorSpec struct {
	Role        string `json:"role"` // executor, approver, reviewer, viewer
	Description string `json:"description"`
	Count       string `json:"count"` // single, multiple, optional
}

// StepSpec represents a step specification
type StepSpec struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"` // input, transform, decision, action, etc.
	Inputs      []string               `json:"inputs,omitempty"`
	Outputs     []string               `json:"outputs,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`

	// Mapping result
	MappedBlock *BlockMappingResult `json:"mapped_block,omitempty"`

	// Flow control
	Branches      []BranchSpec       `json:"branches,omitempty"`
	ErrorHandling *ErrorHandlingSpec `json:"error_handling,omitempty"`
}

// BlockMappingResult represents the result of mapping a step to a block
type BlockMappingResult struct {
	Slug           string `json:"slug,omitempty"`
	Confidence     string `json:"confidence"` // high, medium, low
	CustomRequired bool   `json:"custom_required"`
	CustomReason   string `json:"custom_reason,omitempty"`
}

// BranchSpec represents a branch specification
type BranchSpec struct {
	Condition    string `json:"condition"`
	TargetStepID string `json:"target_step_id"`
}

// ErrorHandlingSpec represents error handling specification
type ErrorHandlingSpec struct {
	OnError        string `json:"on_error"` // retry, skip, abort, notify, fallback
	RetryCount     int    `json:"retry_count,omitempty"`
	RetryDelay     int    `json:"retry_delay,omitempty"` // ms
	FallbackStepID string `json:"fallback_step_id,omitempty"`
	NotifyTo       string `json:"notify_to,omitempty"`
}

// IntegrationSpec represents an integration specification
type IntegrationSpec struct {
	Service         string   `json:"service"`
	Operation       string   `json:"operation"`
	HasCredentials  bool     `json:"has_credentials"`
	RequiredSecrets []string `json:"required_secrets,omitempty"`
}

// ConstraintSpec represents constraint specification
type ConstraintSpec struct {
	Frequency string   `json:"frequency,omitempty"` // once, daily, weekly, monthly, on-demand
	Deadline  string   `json:"deadline,omitempty"`
	SLA       string   `json:"sla,omitempty"`
	Security  []string `json:"security,omitempty"`
}

// AssumptionSpec represents an assumption specification
type AssumptionSpec struct {
	ID          string `json:"id"`
	Category    string `json:"category"` // trigger, actor, step, integration, constraint
	Description string `json:"description"`
	Default     string `json:"default"`
	Confirmed   bool   `json:"confirmed"`
}

// ============================================================================
// Construction Result types
// ============================================================================

// ConstructionResult represents the result of workflow construction
type ConstructionResult struct {
	ProjectID          uuid.UUID           `json:"project_id"`
	Summary            ConstructionSummary `json:"summary"`
	StepMappings       []StepMappingResult `json:"step_mappings"`
	CustomRequirements []CustomRequirement `json:"custom_requirements,omitempty"`
	Warnings           []string            `json:"warnings,omitempty"`
}

// ConstructionSummary represents a summary of the constructed workflow
type ConstructionSummary struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	StepsCount  int    `json:"steps_count"`
	HasApproval bool   `json:"has_approval"`
	Trigger     string `json:"trigger"`
}

// StepMappingResult represents the mapping result for a step
type StepMappingResult struct {
	Name           string `json:"name"`
	Block          string `json:"block,omitempty"`
	Confidence     string `json:"confidence,omitempty"`
	CustomRequired bool   `json:"custom_required"`
	Reason         string `json:"reason,omitempty"`
}

// CustomRequirement represents a custom block requirement
type CustomRequirement struct {
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	Inputs          map[string]string `json:"inputs,omitempty"`
	Outputs         map[string]string `json:"outputs,omitempty"`
	EstimatedEffort string            `json:"estimated_effort,omitempty"` // low, medium, high
}
