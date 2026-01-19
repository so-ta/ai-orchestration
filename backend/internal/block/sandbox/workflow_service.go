package sandbox

import (
	"fmt"
)

// StepExecutor is a function that executes a step by name and returns its output
// This is injected by the executor to avoid circular dependencies
type StepExecutor func(stepName string, input map[string]interface{}) (map[string]interface{}, error)

// WorkflowServiceImpl implements WorkflowService for sandbox scripts
type WorkflowServiceImpl struct {
	stepExecutor StepExecutor
}

// NewWorkflowService creates a new WorkflowService stub without step execution capability
// Use NewWorkflowServiceWithExecutor for full functionality
func NewWorkflowService() *WorkflowServiceImpl {
	return &WorkflowServiceImpl{}
}

// NewWorkflowServiceWithExecutor creates a WorkflowService with step execution capability
// The stepExecutor function is injected by the engine executor
func NewWorkflowServiceWithExecutor(stepExecutor StepExecutor) *WorkflowServiceImpl {
	return &WorkflowServiceImpl{
		stepExecutor: stepExecutor,
	}
}

// Run executes a subflow and returns its output
// Currently returns an error as subflow execution is not yet implemented
func (s *WorkflowServiceImpl) Run(workflowID string, input map[string]interface{}) (map[string]interface{}, error) {
	return nil, fmt.Errorf("subflow execution (ctx.workflow.run) is not yet implemented. WorkflowID: %s", workflowID)
}

// ExecuteStep executes a step within the current workflow by name
// This enables agent blocks to call other steps as tools
func (s *WorkflowServiceImpl) ExecuteStep(stepName string, input map[string]interface{}) (map[string]interface{}, error) {
	if s.stepExecutor == nil {
		return nil, fmt.Errorf("step execution (ctx.workflow.executeStep) is not available in this context")
	}
	return s.stepExecutor(stepName, input)
}
