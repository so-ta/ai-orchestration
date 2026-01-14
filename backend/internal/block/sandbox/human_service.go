package sandbox

import (
	"fmt"
)

// HumanServiceImpl implements HumanService for sandbox scripts
// TODO: Implement full human-in-the-loop functionality when needed
type HumanServiceImpl struct{}

// NewHumanService creates a new HumanService stub
func NewHumanService() *HumanServiceImpl {
	return &HumanServiceImpl{}
}

// RequestApproval requests human approval and waits for response
// Currently returns an error as human approval is not yet implemented
func (s *HumanServiceImpl) RequestApproval(request map[string]interface{}) (map[string]interface{}, error) {
	instructions, _ := request["instructions"].(string)
	return nil, fmt.Errorf("human approval (ctx.human.requestApproval) is not yet implemented. Instructions: %s", instructions)
}
