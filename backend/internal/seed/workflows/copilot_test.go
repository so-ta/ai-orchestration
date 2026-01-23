package workflows

import (
	"encoding/json"
	"testing"

	"github.com/souta/ai-orchestration/internal/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopilotWorkflow_Validation(t *testing.T) {
	wf := CopilotWorkflow()

	err := wf.Validate()
	require.NoError(t, err, "Copilot workflow should be valid")
}

func TestCopilotWorkflow_HasRequiredSteps(t *testing.T) {
	wf := CopilotWorkflow()

	// Check required steps exist
	requiredSteps := []string{
		"start",
		"set_default_config",
		"classify_intent",
		"switch_intent",
		"set_config_create",
		"set_config_debug",
		"set_config_explain",
		"set_config_enhance",
		"set_config_search",
		"set_config_general",
		"set_context",
	}

	stepsByTempID := make(map[string]bool)
	for _, step := range wf.Steps {
		stepsByTempID[step.TempID] = true
	}

	for _, required := range requiredSteps {
		assert.True(t, stepsByTempID[required], "Step %s should exist", required)
	}
}

func TestCopilotWorkflow_SwitchIntentConfig(t *testing.T) {
	wf := CopilotWorkflow()

	// Find switch_intent step
	var switchStep *SystemStepDefinition
	for i, step := range wf.Steps {
		if step.TempID == "switch_intent" {
			switchStep = &wf.Steps[i]
			break
		}
	}

	require.NotNil(t, switchStep, "switch_intent step should exist")

	// Parse config
	var config struct {
		Mode  string `json:"mode"`
		Cases []struct {
			Name       string `json:"name"`
			Expression string `json:"expression"`
			IsDefault  bool   `json:"is_default"`
		} `json:"cases"`
	}
	err := json.Unmarshal(switchStep.Config, &config)
	require.NoError(t, err, "Config should be valid JSON")

	// Verify config structure
	assert.Equal(t, "rules", config.Mode, "Mode should be 'rules'")
	assert.Len(t, config.Cases, 6, "Should have 6 cases (5 intents + default)")

	// Check each case has valid expression format (JSONPath style)
	for _, c := range config.Cases {
		if !c.IsDefault {
			assert.Contains(t, c.Expression, "$.intent", "Expression should use JSONPath format: %s", c.Expression)
			assert.Contains(t, c.Expression, "==", "Expression should use == operator: %s", c.Expression)
		}
	}
}

func TestCopilotWorkflow_SwitchExpressionEvaluation(t *testing.T) {
	evaluator := engine.NewConditionEvaluator()

	tests := []struct {
		name       string
		expression string
		data       map[string]interface{}
		expected   bool
	}{
		{
			name:       "intent is create",
			expression: "$.intent == 'create'",
			data:       map[string]interface{}{"intent": "create"},
			expected:   true,
		},
		{
			name:       "intent is not create",
			expression: "$.intent == 'create'",
			data:       map[string]interface{}{"intent": "debug"},
			expected:   false,
		},
		{
			name:       "intent is debug",
			expression: "$.intent == 'debug'",
			data:       map[string]interface{}{"intent": "debug"},
			expected:   true,
		},
		{
			name:       "intent is explain",
			expression: "$.intent == 'explain'",
			data:       map[string]interface{}{"intent": "explain"},
			expected:   true,
		},
		{
			name:       "intent is enhance",
			expression: "$.intent == 'enhance'",
			data:       map[string]interface{}{"intent": "enhance"},
			expected:   true,
		},
		{
			name:       "intent is search",
			expression: "$.intent == 'search'",
			data:       map[string]interface{}{"intent": "search"},
			expected:   true,
		},
		{
			name:       "intent is general (should not match any specific case)",
			expression: "$.intent == 'create'",
			data:       map[string]interface{}{"intent": "general"},
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dataJSON, _ := json.Marshal(tt.data)
			result, err := evaluator.Evaluate(tt.expression, dataJSON)
			require.NoError(t, err, "Evaluation should not error")
			assert.Equal(t, tt.expected, result, "Expression: %s, Data: %v", tt.expression, tt.data)
		})
	}
}

func TestCopilotWorkflow_EdgeConnections(t *testing.T) {
	wf := CopilotWorkflow()

	// Build a map of edges by source
	edgesBySource := make(map[string][]SystemEdgeDefinition)
	for _, edge := range wf.Edges {
		edgesBySource[edge.SourceTempID] = append(edgesBySource[edge.SourceTempID], edge)
	}

	// Check switch_intent has 6 outgoing edges
	switchEdges := edgesBySource["switch_intent"]
	assert.Len(t, switchEdges, 6, "switch_intent should have 6 outgoing edges")

	// Check each config step connects to set_context
	configSteps := []string{
		"set_config_create",
		"set_config_debug",
		"set_config_explain",
		"set_config_enhance",
		"set_config_search",
		"set_config_general",
	}

	for _, stepName := range configSteps {
		edges := edgesBySource[stepName]
		require.Len(t, edges, 1, "%s should have exactly 1 outgoing edge", stepName)
		assert.Equal(t, "set_context", edges[0].TargetTempID, "%s should connect to set_context", stepName)
	}
}

func TestCopilotWorkflow_SetDefaultConfigStep(t *testing.T) {
	wf := CopilotWorkflow()

	// Find set_default_config step
	var configStep *SystemStepDefinition
	for i, step := range wf.Steps {
		if step.TempID == "set_default_config" {
			configStep = &wf.Steps[i]
			break
		}
	}

	require.NotNil(t, configStep, "set_default_config step should exist")
	assert.Equal(t, "set-variables", configStep.Type, "Should be set-variables type")

	// Parse config
	var config struct {
		Variables []struct {
			Name  string      `json:"name"`
			Value interface{} `json:"value"`
			Type  string      `json:"type"`
		} `json:"variables"`
		MergeInput bool `json:"merge_input"`
	}
	err := json.Unmarshal(configStep.Config, &config)
	require.NoError(t, err, "Config should be valid JSON")

	// Check required variables are defined
	expectedVars := map[string]bool{
		"llm_model":              false,
		"llm_temperature":        false,
		"llm_max_tokens":         false,
		"confidence_threshold":   false,
		"max_refinement_retries": false,
	}

	for _, v := range config.Variables {
		if _, exists := expectedVars[v.Name]; exists {
			expectedVars[v.Name] = true
		}
	}

	for varName, found := range expectedVars {
		assert.True(t, found, "Variable %s should be defined in set_default_config", varName)
	}

	assert.True(t, config.MergeInput, "merge_input should be true")
}

func TestCopilotWorkflow_AgentGroupHasTools(t *testing.T) {
	wf := CopilotWorkflow()

	// Check agent group exists
	require.Len(t, wf.BlockGroups, 1, "Should have exactly 1 block group")
	agentGroup := wf.BlockGroups[0]
	assert.Equal(t, "agent", agentGroup.Type, "Block group should be of type 'agent'")

	// Count steps inside the agent group
	toolSteps := 0
	for _, step := range wf.Steps {
		if step.BlockGroupTempID == agentGroup.TempID {
			toolSteps++
		}
	}

	// Should have multiple tool steps
	assert.GreaterOrEqual(t, toolSteps, 10, "Agent group should have at least 10 tool steps")

	// Check specific tools exist
	expectedTools := []string{
		"list_blocks",
		"get_block_schema",
		"search_blocks",
		"add_step",
		"add_edge",
		"check_workflow_readiness",
		"fix_block_type",
		"check_security",
		"get_relevant_examples",
	}

	toolsByName := make(map[string]bool)
	for _, step := range wf.Steps {
		if step.BlockGroupTempID == agentGroup.TempID {
			toolsByName[step.TempID] = true
		}
	}

	for _, tool := range expectedTools {
		assert.True(t, toolsByName[tool], "Tool %s should exist in agent group", tool)
	}
}
