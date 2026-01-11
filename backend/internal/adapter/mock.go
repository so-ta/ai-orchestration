package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"time"
)

// MockAdapter is a mock adapter for testing and development
type MockAdapter struct {
	id   string
	name string
}

// NewMockAdapter creates a new MockAdapter
func NewMockAdapter() *MockAdapter {
	return &MockAdapter{
		id:   "mock",
		name: "Mock Adapter",
	}
}

func (a *MockAdapter) ID() string   { return a.id }
func (a *MockAdapter) Name() string { return a.name }

// MockConfig represents the configuration for mock adapter
type MockConfig struct {
	DelayMs     int     `json:"delay_ms"`
	SuccessRate float64 `json:"success_rate"`
	Response    string  `json:"response"`
	ErrorMsg    string  `json:"error_message"`
}

// Execute runs the mock adapter
func (a *MockAdapter) Execute(ctx context.Context, req *Request) (*Response, error) {
	start := time.Now()

	// Parse config
	var config MockConfig
	if req.Config != nil {
		if err := json.Unmarshal(req.Config, &config); err != nil {
			return nil, err
		}
	}

	// Default values
	if config.DelayMs == 0 {
		config.DelayMs = 100
	}
	if config.SuccessRate == 0 {
		config.SuccessRate = 1.0
	}

	// Simulate delay
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(time.Duration(config.DelayMs) * time.Millisecond):
	}

	// Simulate failure based on success rate
	if rand.Float64() > config.SuccessRate {
		errMsg := config.ErrorMsg
		if errMsg == "" {
			errMsg = "mock adapter simulated failure"
		}
		return nil, errors.New(errMsg)
	}

	// Build response
	var output json.RawMessage
	if config.Response != "" {
		output = json.RawMessage(config.Response)
	} else {
		// Echo input with success status
		output, _ = json.Marshal(map[string]interface{}{
			"success": true,
			"input":   json.RawMessage(req.Input),
			"message": "Mock adapter executed successfully",
		})
	}

	return &Response{
		Output:     output,
		DurationMs: int(time.Since(start).Milliseconds()),
		Metadata: map[string]string{
			"adapter": a.id,
		},
	}, nil
}

func (a *MockAdapter) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"message": {"type": "string"}
		}
	}`)
}

func (a *MockAdapter) OutputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"success": {"type": "boolean"},
			"message": {"type": "string"}
		}
	}`)
}
