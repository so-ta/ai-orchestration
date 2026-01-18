package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// BuilderSessionStatus represents the status of a builder session
type BuilderSessionStatus string

const (
	BuilderSessionStatusHearing   BuilderSessionStatus = "hearing"
	BuilderSessionStatusBuilding  BuilderSessionStatus = "building"
	BuilderSessionStatusReviewing BuilderSessionStatus = "reviewing"
	BuilderSessionStatusRefining  BuilderSessionStatus = "refining"
	BuilderSessionStatusCompleted BuilderSessionStatus = "completed"
	BuilderSessionStatusAbandoned BuilderSessionStatus = "abandoned"
)

// HearingPhase represents the current phase of the hearing process
type HearingPhase string

const (
	HearingPhasePurpose       HearingPhase = "purpose"
	HearingPhaseConditions    HearingPhase = "conditions"
	HearingPhaseActors        HearingPhase = "actors"
	HearingPhaseFrequency     HearingPhase = "frequency"
	HearingPhaseIntegrations  HearingPhase = "integrations"
	HearingPhasePainPoints    HearingPhase = "pain_points"
	HearingPhaseConfirmation  HearingPhase = "confirmation"
	HearingPhaseCompleted     HearingPhase = "completed"
)

// BuilderSession represents a workflow builder session
type BuilderSession struct {
	ID               uuid.UUID            `json:"id"`
	TenantID         uuid.UUID            `json:"tenant_id"`
	UserID           string               `json:"user_id"`
	CopilotSessionID *uuid.UUID           `json:"copilot_session_id,omitempty"`

	// Status
	Status          BuilderSessionStatus `json:"status"`
	HearingPhase    HearingPhase         `json:"hearing_phase"`
	HearingProgress int                  `json:"hearing_progress"` // 0-100

	// Generated artifacts
	Spec      json.RawMessage `json:"spec,omitempty"`       // WorkflowSpec as JSON
	ProjectID *uuid.UUID      `json:"project_id,omitempty"` // Generated project

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Loaded relations
	Messages []BuilderMessage `json:"messages,omitempty"`
}

// BuilderMessage represents a message in a builder session
type BuilderMessage struct {
	ID        uuid.UUID `json:"id"`
	SessionID uuid.UUID `json:"session_id"`
	Role      string    `json:"role"` // "user", "assistant", "system"
	Content   string    `json:"content"`

	// Metadata
	Phase              *HearingPhase   `json:"phase,omitempty"`
	ExtractedData      json.RawMessage `json:"extracted_data,omitempty"`
	SuggestedQuestions json.RawMessage `json:"suggested_questions,omitempty"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
}

// NewBuilderSession creates a new builder session
func NewBuilderSession(tenantID uuid.UUID, userID string) *BuilderSession {
	now := time.Now().UTC()
	return &BuilderSession{
		ID:              uuid.New(),
		TenantID:        tenantID,
		UserID:          userID,
		Status:          BuilderSessionStatusHearing,
		HearingPhase:    HearingPhasePurpose,
		HearingProgress: 0,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// NewBuilderMessage creates a new builder message
func NewBuilderMessage(sessionID uuid.UUID, role, content string) *BuilderMessage {
	return &BuilderMessage{
		ID:        uuid.New(),
		SessionID: sessionID,
		Role:      role,
		Content:   content,
		CreatedAt: time.Now().UTC(),
	}
}

// AddMessage adds a message to the session
func (s *BuilderSession) AddMessage(role, content string) *BuilderMessage {
	msg := NewBuilderMessage(s.ID, role, content)
	s.Messages = append(s.Messages, *msg)
	s.UpdatedAt = time.Now().UTC()
	return msg
}

// SetPhase updates the hearing phase and progress
func (s *BuilderSession) SetPhase(phase HearingPhase, progress int) {
	s.HearingPhase = phase
	s.HearingProgress = progress
	s.UpdatedAt = time.Now().UTC()
}

// SetSpec sets the workflow spec
func (s *BuilderSession) SetSpec(spec json.RawMessage) {
	s.Spec = spec
	s.UpdatedAt = time.Now().UTC()
}

// SetProjectID sets the generated project ID
func (s *BuilderSession) SetProjectID(projectID uuid.UUID) {
	s.ProjectID = &projectID
	s.UpdatedAt = time.Now().UTC()
}

// SetStatus sets the session status
func (s *BuilderSession) SetStatus(status BuilderSessionStatus) {
	s.Status = status
	s.UpdatedAt = time.Now().UTC()
}

// Complete marks the session as completed
func (s *BuilderSession) Complete() {
	s.Status = BuilderSessionStatusCompleted
	s.HearingPhase = HearingPhaseCompleted
	s.UpdatedAt = time.Now().UTC()
}

// Abandon marks the session as abandoned
func (s *BuilderSession) Abandon() {
	s.Status = BuilderSessionStatusAbandoned
	s.UpdatedAt = time.Now().UTC()
}

// IsActive returns true if the session is still active
func (s *BuilderSession) IsActive() bool {
	return s.Status != BuilderSessionStatusCompleted && s.Status != BuilderSessionStatusAbandoned
}

// GetPhaseProgress returns the default progress for a phase
func GetPhaseProgress(phase HearingPhase) int {
	switch phase {
	case HearingPhasePurpose:
		return 10
	case HearingPhaseConditions:
		return 25
	case HearingPhaseActors:
		return 40
	case HearingPhaseFrequency:
		return 55
	case HearingPhaseIntegrations:
		return 70
	case HearingPhasePainPoints:
		return 85
	case HearingPhaseConfirmation:
		return 95
	case HearingPhaseCompleted:
		return 100
	default:
		return 0
	}
}

// NextPhase returns the next hearing phase
func NextPhase(current HearingPhase) HearingPhase {
	switch current {
	case HearingPhasePurpose:
		return HearingPhaseConditions
	case HearingPhaseConditions:
		return HearingPhaseActors
	case HearingPhaseActors:
		return HearingPhaseFrequency
	case HearingPhaseFrequency:
		return HearingPhaseIntegrations
	case HearingPhaseIntegrations:
		return HearingPhasePainPoints
	case HearingPhasePainPoints:
		return HearingPhaseConfirmation
	case HearingPhaseConfirmation:
		return HearingPhaseCompleted
	default:
		return HearingPhaseCompleted
	}
}

// ============================================================================
// WorkflowSpec types for JSON serialization
// ============================================================================

// WorkflowSpec represents the internal DSL for workflow specification
type WorkflowSpec struct {
	ID               string             `json:"id"`
	Name             string             `json:"name"`
	Description      string             `json:"description"`
	Purpose          string             `json:"purpose"`
	SuccessCriteria  []string           `json:"success_criteria,omitempty"`
	BusinessDomain   string             `json:"business_domain,omitempty"`
	Trigger          *TriggerSpec       `json:"trigger,omitempty"`
	Completion       *CompletionSpec    `json:"completion,omitempty"`
	Actors           []ActorSpec        `json:"actors,omitempty"`
	Steps            []StepSpec         `json:"steps,omitempty"`
	Integrations     []IntegrationSpec  `json:"integrations,omitempty"`
	Constraints      *ConstraintSpec    `json:"constraints,omitempty"`
	Assumptions      []AssumptionSpec   `json:"assumptions,omitempty"`
	HeardAt          *time.Time         `json:"heard_at,omitempty"`
	Version          int                `json:"version"`
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
	ProjectID           uuid.UUID             `json:"project_id"`
	Summary             ConstructionSummary   `json:"summary"`
	StepMappings        []StepMappingResult   `json:"step_mappings"`
	CustomRequirements  []CustomRequirement   `json:"custom_requirements,omitempty"`
	Warnings            []string              `json:"warnings,omitempty"`
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
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Inputs          map[string]string      `json:"inputs,omitempty"`
	Outputs         map[string]string      `json:"outputs,omitempty"`
	EstimatedEffort string                 `json:"estimated_effort,omitempty"` // low, medium, high
}
