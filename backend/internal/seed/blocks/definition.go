package blocks

import (
	"encoding/json"

	"github.com/souta/ai-orchestration/internal/domain"
)

// BlockTestCase defines a test case for block code execution
type BlockTestCase struct {
	Name           string                 // Test name
	Input          map[string]interface{} // Input data
	Config         map[string]interface{} // Block configuration
	ExpectedOutput map[string]interface{} // Expected output (partial match)
	ExpectError    bool                   // Whether an error is expected
	ErrorContains  string                 // Partial match for error message
}

// SystemBlockDefinition represents a programmatically-defined system block
type SystemBlockDefinition struct {
	// Identifiers
	Slug    string `json:"slug"`
	Version int    `json:"version"` // Explicit version (increment when IN/OUT schema changes)

	// Basic info
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Category    domain.BlockCategory    `json:"category"`
	Subcategory domain.BlockSubcategory `json:"subcategory,omitempty"`
	Icon        string                  `json:"icon"`

	// Schema definitions
	ConfigSchema json.RawMessage     `json:"config_schema"`
	OutputSchema json.RawMessage     `json:"output_schema,omitempty"`
	InputPorts   []domain.InputPort  `json:"input_ports"`
	OutputPorts  []domain.OutputPort `json:"output_ports"`

	// Execution code
	Code string `json:"code"`

	// UI settings
	UIConfig json.RawMessage `json:"ui_config"`

	// Error handling and credentials
	ErrorCodes          []domain.ErrorCodeDef `json:"error_codes"`
	RequiredCredentials json.RawMessage       `json:"required_credentials,omitempty"`

	// Flags
	Enabled bool `json:"enabled"`

	// Group block fields (Phase B: unified block model for groups)
	GroupKind   domain.BlockGroupKind `json:"group_kind,omitempty"`
	IsContainer bool                  `json:"is_container,omitempty"`

	// === Block Inheritance/Extension fields ===
	// ParentBlockSlug: Slug of the parent block to inherit from (resolved to ID at migration time)
	ParentBlockSlug string `json:"parent_block_slug,omitempty"`
	// ConfigDefaults: Default values for parent's config_schema (merged at execution time)
	ConfigDefaults json.RawMessage `json:"config_defaults,omitempty"`
	// PreProcess: JavaScript code executed before main code (input transformation)
	PreProcess string `json:"pre_process,omitempty"`
	// PostProcess: JavaScript code executed after main code (output transformation)
	PostProcess string `json:"post_process,omitempty"`
	// InternalSteps: Array of steps to execute sequentially inside the block
	InternalSteps []domain.InternalStep `json:"internal_steps,omitempty"`

	// === Declarative Request/Response Configuration ===
	// Request: Declarative HTTP request configuration (alternative to PreProcess)
	Request *domain.RequestConfig `json:"request,omitempty"`
	// Response: Declarative response processing configuration (alternative to PostProcess)
	Response *domain.ResponseConfig `json:"response,omitempty"`

	// Test cases (for testing only, not stored in DB)
	TestCases []BlockTestCase `json:"-"`
}

// DefaultInputPorts returns the default single input port
func DefaultInputPorts() []domain.InputPort {
	return []domain.InputPort{
		{Name: "input", Label: "Input", Required: true},
	}
}

// DefaultOutputPorts returns the default single output port
func DefaultOutputPorts() []domain.OutputPort {
	return []domain.OutputPort{
		{Name: "output", Label: "Output", IsDefault: true},
	}
}
