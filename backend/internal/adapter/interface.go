package adapter

import (
	"context"
	"encoding/json"
)

// Adapter defines the interface for external integrations
type Adapter interface {
	// ID returns the unique identifier of the adapter
	ID() string

	// Name returns the display name of the adapter
	Name() string

	// Execute runs the adapter with the given request
	Execute(ctx context.Context, req *Request) (*Response, error)

	// InputSchema returns the JSON Schema for the input
	InputSchema() json.RawMessage

	// OutputSchema returns the JSON Schema for the output
	OutputSchema() json.RawMessage
}

// Request represents an adapter execution request
type Request struct {
	Input         json.RawMessage   `json:"input"`
	Config        json.RawMessage   `json:"config"`
	CorrelationID string            `json:"correlation_id"`
	Timeout       int               `json:"timeout_ms"`
	Metadata      map[string]string `json:"metadata"`
}

// Response represents an adapter execution response
type Response struct {
	Output    json.RawMessage `json:"output"`
	DurationMs int            `json:"duration_ms"`
	Metadata  map[string]string `json:"metadata"`
}

// Registry holds all registered adapters
type Registry struct {
	adapters map[string]Adapter
}

// NewRegistry creates a new adapter registry
func NewRegistry() *Registry {
	return &Registry{
		adapters: make(map[string]Adapter),
	}
}

// Register adds an adapter to the registry
func (r *Registry) Register(adapter Adapter) {
	r.adapters[adapter.ID()] = adapter
}

// Get retrieves an adapter by ID
func (r *Registry) Get(id string) (Adapter, bool) {
	adapter, ok := r.adapters[id]
	return adapter, ok
}

// List returns all registered adapters
func (r *Registry) List() []Adapter {
	adapters := make([]Adapter, 0, len(r.adapters))
	for _, adapter := range r.adapters {
		adapters = append(adapters, adapter)
	}
	return adapters
}
