package adapter

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewMockAdapter(t *testing.T) {
	adapter := NewMockAdapter()

	assert.NotNil(t, adapter)
	assert.Equal(t, "mock", adapter.ID())
	assert.Equal(t, "Mock Adapter", adapter.Name())
}

func TestMockAdapter_Execute_Success(t *testing.T) {
	adapter := NewMockAdapter()

	req := &Request{
		Input: json.RawMessage(`{"message": "hello"}`),
		Config: json.RawMessage(`{
			"delay_ms": 10,
			"success_rate": 1.0
		}`),
	}

	resp, err := adapter.Execute(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Greater(t, resp.DurationMs, 0)
	assert.Equal(t, "mock", resp.Metadata["adapter"])

	// Check output contains success
	var output map[string]interface{}
	err = json.Unmarshal(resp.Output, &output)
	assert.NoError(t, err)
	assert.Equal(t, true, output["success"])
}

func TestMockAdapter_Execute_CustomResponse(t *testing.T) {
	adapter := NewMockAdapter()

	customResponse := `{"custom": "response", "value": 42}`
	config := MockConfig{
		DelayMs:  10,
		Response: customResponse,
	}
	configJSON, _ := json.Marshal(config)

	req := &Request{
		Input:  json.RawMessage(`{}`),
		Config: configJSON,
	}

	resp, err := adapter.Execute(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.JSONEq(t, customResponse, string(resp.Output))
}

func TestMockAdapter_Execute_Failure(t *testing.T) {
	adapter := NewMockAdapter()

	// Set success_rate to -1 to guarantee failure (rand.Float64() is always > -1)
	config := MockConfig{
		DelayMs:     10,
		SuccessRate: -1.0, // Always fail
		ErrorMsg:    "simulated error",
	}
	configJSON, _ := json.Marshal(config)

	req := &Request{
		Input:  json.RawMessage(`{}`),
		Config: configJSON,
	}

	resp, err := adapter.Execute(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "simulated error")
}

func TestMockAdapter_Execute_ContextCancellation(t *testing.T) {
	adapter := NewMockAdapter()

	ctx, cancel := context.WithCancel(context.Background())

	req := &Request{
		Input: json.RawMessage(`{}`),
		Config: json.RawMessage(`{
			"delay_ms": 1000
		}`),
	}

	// Cancel immediately
	cancel()

	resp, err := adapter.Execute(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestMockAdapter_Execute_ContextTimeout(t *testing.T) {
	adapter := NewMockAdapter()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	req := &Request{
		Input: json.RawMessage(`{}`),
		Config: json.RawMessage(`{
			"delay_ms": 1000
		}`),
	}

	resp, err := adapter.Execute(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestMockAdapter_Execute_InvalidConfig(t *testing.T) {
	adapter := NewMockAdapter()

	req := &Request{
		Input:  json.RawMessage(`{}`),
		Config: json.RawMessage(`invalid json`),
	}

	resp, err := adapter.Execute(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestMockAdapter_Execute_NoConfig(t *testing.T) {
	adapter := NewMockAdapter()

	req := &Request{
		Input: json.RawMessage(`{"test": "data"}`),
	}

	resp, err := adapter.Execute(context.Background(), req)

	// Should use default values and succeed
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestMockAdapter_InputSchema(t *testing.T) {
	adapter := NewMockAdapter()
	schema := adapter.InputSchema()

	assert.NotNil(t, schema)

	var parsed map[string]interface{}
	err := json.Unmarshal(schema, &parsed)
	assert.NoError(t, err)
	assert.Equal(t, "object", parsed["type"])
}

func TestMockAdapter_OutputSchema(t *testing.T) {
	adapter := NewMockAdapter()
	schema := adapter.OutputSchema()

	assert.NotNil(t, schema)

	var parsed map[string]interface{}
	err := json.Unmarshal(schema, &parsed)
	assert.NoError(t, err)
	assert.Equal(t, "object", parsed["type"])
}
