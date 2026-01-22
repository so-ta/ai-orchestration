package domain

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestNewStep(t *testing.T) {
	tenantID := uuid.New()
	projectID := uuid.New()
	name := "Test Step"
	stepType := StepTypeLLM
	config := json.RawMessage(`{"model": "gpt-4"}`)

	step := NewStep(tenantID, projectID, name, stepType, config)

	if step.ID == uuid.Nil {
		t.Error("NewStep() should generate a non-nil UUID")
	}
	if step.TenantID != tenantID {
		t.Errorf("NewStep() TenantID = %v, want %v", step.TenantID, tenantID)
	}
	if step.ProjectID != projectID {
		t.Errorf("NewStep() ProjectID = %v, want %v", step.ProjectID, projectID)
	}
	if step.Name != name {
		t.Errorf("NewStep() Name = %v, want %v", step.Name, name)
	}
	if step.Type != stepType {
		t.Errorf("NewStep() Type = %v, want %v", step.Type, stepType)
	}
	if string(step.Config) != string(config) {
		t.Error("NewStep() Config mismatch")
	}
	if step.CreatedAt.IsZero() {
		t.Error("NewStep() CreatedAt should not be zero")
	}
	if step.UpdatedAt.IsZero() {
		t.Error("NewStep() UpdatedAt should not be zero")
	}
}

func TestNewStartStep(t *testing.T) {
	tenantID := uuid.New()
	projectID := uuid.New()
	name := "Manual Start"
	triggerType := StepTriggerTypeManual
	triggerConfig := json.RawMessage(`{}`)

	step := NewStartStep(tenantID, projectID, name, triggerType, triggerConfig)

	if step.Type != StepTypeStart {
		t.Errorf("NewStartStep() Type = %v, want %v", step.Type, StepTypeStart)
	}
	if step.TriggerType == nil || *step.TriggerType != triggerType {
		t.Error("NewStartStep() TriggerType mismatch")
	}
}

func TestStepType_IsValid(t *testing.T) {
	validTypes := []StepType{
		StepTypeStart, StepTypeLLM, StepTypeTool, StepTypeCondition,
		StepTypeSwitch, StepTypeMap, StepTypeSubflow, StepTypeWait,
		StepTypeFunction, StepTypeRouter, StepTypeHumanInLoop,
		StepTypeFilter, StepTypeSplit, StepTypeAggregate, StepTypeError,
		StepTypeNote, StepTypeLog,
	}

	for _, st := range validTypes {
		t.Run(string(st), func(t *testing.T) {
			if !st.IsValid() {
				t.Errorf("IsValid() = false for valid type %v", st)
			}
		})
	}

	invalidTypes := []StepType{
		StepType("invalid"),
		StepType(""),
		StepType("unknown"),
	}

	for _, st := range invalidTypes {
		t.Run(string(st), func(t *testing.T) {
			if st.IsValid() {
				t.Errorf("IsValid() = true for invalid type %v", st)
			}
		})
	}
}

func TestStep_IsStartBlock(t *testing.T) {
	tests := []struct {
		stepType StepType
		want     bool
	}{
		{StepTypeStart, true},
		{StepTypeLLM, false},
		{StepTypeTool, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.stepType), func(t *testing.T) {
			step := &Step{Type: tt.stepType}
			if got := step.IsStartBlock(); got != tt.want {
				t.Errorf("IsStartBlock() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStep_GetTriggerType(t *testing.T) {
	tests := []struct {
		name        string
		triggerType *StepTriggerType
		want        StepTriggerType
	}{
		{
			name:        "nil trigger type",
			triggerType: nil,
			want:        StepTriggerTypeManual,
		},
		{
			name:        "manual trigger",
			triggerType: func() *StepTriggerType { t := StepTriggerTypeManual; return &t }(),
			want:        StepTriggerTypeManual,
		},
		{
			name:        "webhook trigger",
			triggerType: func() *StepTriggerType { t := StepTriggerTypeWebhook; return &t }(),
			want:        StepTriggerTypeWebhook,
		},
		{
			name:        "schedule trigger",
			triggerType: func() *StepTriggerType { t := StepTriggerTypeSchedule; return &t }(),
			want:        StepTriggerTypeSchedule,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &Step{TriggerType: tt.triggerType}
			if got := step.GetTriggerType(); got != tt.want {
				t.Errorf("GetTriggerType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStepTriggerType_IsValid(t *testing.T) {
	validTypes := []StepTriggerType{
		StepTriggerTypeManual, StepTriggerTypeWebhook, StepTriggerTypeSchedule,
		StepTriggerTypeSlack, StepTriggerTypeEmail,
	}

	for _, tt := range validTypes {
		t.Run(string(tt), func(t *testing.T) {
			if !tt.IsValid() {
				t.Errorf("IsValid() = false for valid type %v", tt)
			}
		})
	}

	invalidTypes := []StepTriggerType{
		StepTriggerType("invalid"),
		StepTriggerType(""),
	}

	for _, it := range invalidTypes {
		t.Run(string(it), func(t *testing.T) {
			if it.IsValid() {
				t.Errorf("IsValid() = true for invalid type %v", it)
			}
		})
	}
}

func TestStep_SetPosition(t *testing.T) {
	step := NewStep(uuid.New(), uuid.New(), "Test", StepTypeLLM, nil)

	step.SetPosition(100, 200)

	if step.PositionX != 100 {
		t.Errorf("SetPosition() PositionX = %v, want 100", step.PositionX)
	}
	if step.PositionY != 200 {
		t.Errorf("SetPosition() PositionY = %v, want 200", step.PositionY)
	}
}

func TestIsTriggerBlockSlug(t *testing.T) {
	tests := []struct {
		slug string
		want bool
	}{
		{"manual_trigger", true},
		{"schedule_trigger", true},
		{"webhook_trigger", true},
		{"llm", false},
		{"tool", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.slug, func(t *testing.T) {
			if got := IsTriggerBlockSlug(tt.slug); got != tt.want {
				t.Errorf("IsTriggerBlockSlug(%v) = %v, want %v", tt.slug, got, tt.want)
			}
		})
	}
}

func TestGetTriggerTypeFromSlug(t *testing.T) {
	tests := []struct {
		slug string
		want StepTriggerType
	}{
		{"manual_trigger", StepTriggerTypeManual},
		{"schedule_trigger", StepTriggerTypeSchedule},
		{"webhook_trigger", StepTriggerTypeWebhook},
		{"unknown", StepTriggerTypeManual}, // default
	}

	for _, tt := range tests {
		t.Run(tt.slug, func(t *testing.T) {
			if got := GetTriggerTypeFromSlug(tt.slug); got != tt.want {
				t.Errorf("GetTriggerTypeFromSlug(%v) = %v, want %v", tt.slug, got, tt.want)
			}
		})
	}
}

func TestStep_RetryConfig(t *testing.T) {
	step := NewStep(uuid.New(), uuid.New(), "Test", StepTypeLLM, nil)

	// Default config
	config, err := step.GetRetryConfig()
	if err != nil {
		t.Fatalf("GetRetryConfig() error = %v", err)
	}
	if config.MaxRetries != 0 {
		t.Errorf("Default MaxRetries = %v, want 0", config.MaxRetries)
	}

	// Set config
	newConfig := &RetryConfig{
		MaxRetries:         3,
		DelayMs:            2000,
		ExponentialBackoff: true,
	}
	err = step.SetRetryConfig(newConfig)
	if err != nil {
		t.Fatalf("SetRetryConfig() error = %v", err)
	}

	got, err := step.GetRetryConfig()
	if err != nil {
		t.Fatalf("GetRetryConfig() error = %v", err)
	}
	if got.MaxRetries != 3 {
		t.Errorf("MaxRetries = %v, want 3", got.MaxRetries)
	}
	if got.DelayMs != 2000 {
		t.Errorf("DelayMs = %v, want 2000", got.DelayMs)
	}
	if !got.ExponentialBackoff {
		t.Error("ExponentialBackoff should be true")
	}
}

func TestRetryConfig_ShouldRetryError(t *testing.T) {
	tests := []struct {
		name      string
		config    RetryConfig
		errorCode string
		attempt   int
		want      bool
	}{
		{
			name:      "max retries 0",
			config:    RetryConfig{MaxRetries: 0},
			errorCode: "any",
			attempt:   0,
			want:      false,
		},
		{
			name:      "attempt exceeded",
			config:    RetryConfig{MaxRetries: 3},
			errorCode: "any",
			attempt:   3,
			want:      false,
		},
		{
			name:      "retry all errors",
			config:    RetryConfig{MaxRetries: 3},
			errorCode: "any",
			attempt:   0,
			want:      true,
		},
		{
			name:      "matching error code",
			config:    RetryConfig{MaxRetries: 3, RetryOnErrors: []string{"timeout", "network"}},
			errorCode: "timeout",
			attempt:   0,
			want:      true,
		},
		{
			name:      "non-matching error code",
			config:    RetryConfig{MaxRetries: 3, RetryOnErrors: []string{"timeout"}},
			errorCode: "validation",
			attempt:   0,
			want:      false,
		},
		{
			name:      "wildcard error code",
			config:    RetryConfig{MaxRetries: 3, RetryOnErrors: []string{"*"}},
			errorCode: "any",
			attempt:   0,
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.ShouldRetryError(tt.errorCode, tt.attempt); got != tt.want {
				t.Errorf("ShouldRetryError(%v, %v) = %v, want %v", tt.errorCode, tt.attempt, got, tt.want)
			}
		})
	}
}

func TestRetryConfig_GetDelayForAttempt(t *testing.T) {
	tests := []struct {
		name       string
		config     RetryConfig
		attempt    int
		wantMin    int
		wantMax    int
	}{
		{
			name:    "no exponential backoff",
			config:  RetryConfig{DelayMs: 1000, ExponentialBackoff: false},
			attempt: 3,
			wantMin: 1000,
			wantMax: 1000,
		},
		{
			name:    "exponential backoff attempt 0",
			config:  RetryConfig{DelayMs: 1000, ExponentialBackoff: true, MaxDelayMs: 30000},
			attempt: 0,
			wantMin: 1000,
			wantMax: 1000,
		},
		{
			name:    "exponential backoff attempt 1",
			config:  RetryConfig{DelayMs: 1000, ExponentialBackoff: true, MaxDelayMs: 30000},
			attempt: 1,
			wantMin: 2000,
			wantMax: 2000,
		},
		{
			name:    "exponential backoff capped",
			config:  RetryConfig{DelayMs: 10000, ExponentialBackoff: true, MaxDelayMs: 15000},
			attempt: 5,
			wantMin: 15000,
			wantMax: 15000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetDelayForAttempt(tt.attempt)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("GetDelayForAttempt(%v) = %v, want between %v and %v", tt.attempt, got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestStep_GetCredentialBindings(t *testing.T) {
	step := NewStep(uuid.New(), uuid.New(), "Test", StepTypeTool, nil)

	// Empty bindings
	bindings, err := step.GetCredentialBindings()
	if err != nil {
		t.Fatalf("GetCredentialBindings() error = %v", err)
	}
	if len(bindings) != 0 {
		t.Errorf("Empty bindings should return empty map, got %v", len(bindings))
	}

	// Set bindings
	credID := uuid.New()
	step.CredentialBindings = json.RawMessage(`{"api_key": "` + credID.String() + `"}`)

	bindings, err = step.GetCredentialBindings()
	if err != nil {
		t.Fatalf("GetCredentialBindings() error = %v", err)
	}
	if bindings["api_key"] != credID {
		t.Errorf("GetCredentialBindings() api_key mismatch")
	}
}
