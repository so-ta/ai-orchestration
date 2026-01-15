package domain

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestStepType_IsValid(t *testing.T) {
	tests := []struct {
		stepType StepType
		expected bool
	}{
		{StepTypeLLM, true},
		{StepTypeTool, true},
		{StepTypeCondition, true},
		{StepTypeMap, true},
		{StepTypeJoin, true},
		{StepTypeSubflow, true},
		{StepType("invalid"), false},
		{StepType(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.stepType), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.stepType.IsValid())
		})
	}
}

func TestValidStepTypes(t *testing.T) {
	types := ValidStepTypes()
	assert.Len(t, types, 18)
	assert.Contains(t, types, StepTypeStart)
	assert.Contains(t, types, StepTypeLLM)
	assert.Contains(t, types, StepTypeTool)
	assert.Contains(t, types, StepTypeCondition)
	assert.Contains(t, types, StepTypeSwitch)
	assert.Contains(t, types, StepTypeMap)
	assert.Contains(t, types, StepTypeJoin)
	assert.Contains(t, types, StepTypeSubflow)
	assert.Contains(t, types, StepTypeWait)
	assert.Contains(t, types, StepTypeFunction)
	assert.Contains(t, types, StepTypeRouter)
	assert.Contains(t, types, StepTypeHumanInLoop)
	assert.Contains(t, types, StepTypeFilter)
	assert.Contains(t, types, StepTypeSplit)
	assert.Contains(t, types, StepTypeAggregate)
	assert.Contains(t, types, StepTypeError)
	assert.Contains(t, types, StepTypeNote)
	assert.Contains(t, types, StepTypeLog)
}

func TestNewStep(t *testing.T) {
	tenantID := uuid.New()
	workflowID := uuid.New()
	config := json.RawMessage(`{"adapter_id": "mock", "settings": {"timeout": 30}}`)

	step := NewStep(tenantID, workflowID, "Test Step", StepTypeTool, config)

	assert.NotEqual(t, uuid.Nil, step.ID)
	assert.Equal(t, tenantID, step.TenantID)
	assert.Equal(t, workflowID, step.WorkflowID)
	assert.Equal(t, "Test Step", step.Name)
	assert.Equal(t, StepTypeTool, step.Type)
	assert.Equal(t, config, step.Config)
	assert.False(t, step.CreatedAt.IsZero())
	assert.False(t, step.UpdatedAt.IsZero())
}

func TestStep_SetPosition(t *testing.T) {
	step := &Step{
		ID:        uuid.New(),
		PositionX: 0,
		PositionY: 0,
	}

	step.SetPosition(100, 200)

	assert.Equal(t, 100, step.PositionX)
	assert.Equal(t, 200, step.PositionY)
	assert.False(t, step.UpdatedAt.IsZero())
}

func TestLLMStepConfig(t *testing.T) {
	config := LLMStepConfig{
		Model:          "gpt-4",
		PromptTemplate: "Hello {{.input}}",
		MaxTokens:      1000,
		Temperature:    0.7,
	}

	data, err := json.Marshal(config)
	assert.NoError(t, err)

	var decoded LLMStepConfig
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, config.Model, decoded.Model)
	assert.Equal(t, config.PromptTemplate, decoded.PromptTemplate)
	assert.Equal(t, config.MaxTokens, decoded.MaxTokens)
	assert.Equal(t, config.Temperature, decoded.Temperature)
}

func TestToolStepConfig(t *testing.T) {
	config := ToolStepConfig{
		AdapterID:    "http-api",
		InputMapping: json.RawMessage(`{"url": "{{.api_url}}"}`),
	}

	data, err := json.Marshal(config)
	assert.NoError(t, err)

	var decoded ToolStepConfig
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, config.AdapterID, decoded.AdapterID)
}

func TestConditionStepConfig(t *testing.T) {
	config := ConditionStepConfig{
		Expression: "result.status == 'success'",
	}

	data, err := json.Marshal(config)
	assert.NoError(t, err)

	var decoded ConditionStepConfig
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, config.Expression, decoded.Expression)
}

func TestMapStepConfig(t *testing.T) {
	config := MapStepConfig{
		InputPath:  "items",
		Parallel:   true,
		MaxWorkers: 5,
	}

	data, err := json.Marshal(config)
	assert.NoError(t, err)

	var decoded MapStepConfig
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, config.InputPath, decoded.InputPath)
	assert.Equal(t, config.Parallel, decoded.Parallel)
	assert.Equal(t, config.MaxWorkers, decoded.MaxWorkers)
}

func TestSubflowStepConfig(t *testing.T) {
	workflowID := uuid.New()
	config := SubflowStepConfig{
		WorkflowID:      workflowID,
		WorkflowVersion: 2,
		InputMapping:    json.RawMessage(`{"input": "{{.data}}"}`),
	}

	data, err := json.Marshal(config)
	assert.NoError(t, err)

	var decoded SubflowStepConfig
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, config.WorkflowID, decoded.WorkflowID)
	assert.Equal(t, config.WorkflowVersion, decoded.WorkflowVersion)
}
