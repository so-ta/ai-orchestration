package sandbox

import (
	"fmt"
)

// WorkflowServiceImpl implements WorkflowService for sandbox scripts
// TODO: Implement full subflow execution when needed
type WorkflowServiceImpl struct{}

// NewWorkflowService creates a new WorkflowService stub
func NewWorkflowService() *WorkflowServiceImpl {
	return &WorkflowServiceImpl{}
}

// Run executes a subflow and returns its output
// Currently returns an error as subflow execution is not yet implemented
func (s *WorkflowServiceImpl) Run(workflowID string, input map[string]interface{}) (map[string]interface{}, error) {
	return nil, fmt.Errorf("subflow execution (ctx.workflow.run) is not yet implemented. WorkflowID: %s", workflowID)
}
