package sandbox

import (
	"fmt"
)

// AdapterServiceImpl implements AdapterService for sandbox scripts
// TODO: Implement full adapter execution when needed
type AdapterServiceImpl struct{}

// NewAdapterService creates a new AdapterService stub
func NewAdapterService() *AdapterServiceImpl {
	return &AdapterServiceImpl{}
}

// Call executes an adapter and returns its output
// Currently returns an error as adapter execution is not yet implemented
func (s *AdapterServiceImpl) Call(adapterID string, input map[string]interface{}) (map[string]interface{}, error) {
	return nil, fmt.Errorf("adapter execution (ctx.adapter.call) is not yet implemented. AdapterID: %s", adapterID)
}
