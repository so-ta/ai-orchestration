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
// All text fields support multiple languages through LocalizedText
type SystemBlockDefinition struct {
	// Identifiers
	Slug    string `json:"slug"`
	Version int    `json:"version"` // Explicit version (increment when IN/OUT schema changes)

	// Basic info (localized)
	Name        domain.LocalizedText    `json:"name"`
	Description domain.LocalizedText    `json:"description"`
	Category    domain.BlockCategory    `json:"category"`
	Subcategory domain.BlockSubcategory `json:"subcategory,omitempty"`
	Icon        string                  `json:"icon"`

	// Schema definitions (localized for config labels/descriptions)
	ConfigSchema domain.LocalizedConfigSchema `json:"config_schema"`
	OutputSchema json.RawMessage              `json:"output_schema,omitempty"`
	OutputPorts  []domain.LocalizedOutputPort `json:"output_ports"`

	// Execution code
	Code string `json:"code"`

	// UI settings (localized for group titles, etc.)
	UIConfig domain.LocalizedConfigSchema `json:"ui_config"`

	// Error handling and credentials (localized)
	ErrorCodes          []domain.LocalizedErrorCodeDef `json:"error_codes"`
	RequiredCredentials json.RawMessage                `json:"required_credentials,omitempty"`

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

// DefaultOutputPorts returns the default single output port (localized)
func DefaultOutputPorts() []domain.LocalizedOutputPort {
	return []domain.LocalizedOutputPort{
		{
			Name:      "output",
			Label:     domain.L("Output", "出力"),
			IsDefault: true,
		},
	}
}

// Helper functions for creating localized content

// LText creates a LocalizedText with English and Japanese
func LText(en, ja string) domain.LocalizedText {
	return domain.L(en, ja)
}

// LSchema creates a LocalizedConfigSchema with English and Japanese JSON schemas
func LSchema(en, ja string) domain.LocalizedConfigSchema {
	return domain.LocalizedConfigSchema{
		EN: json.RawMessage(en),
		JA: json.RawMessage(ja),
	}
}

// LPort creates a LocalizedOutputPort
func LPort(name string, labelEN, labelJA string, isDefault bool) domain.LocalizedOutputPort {
	return domain.LocalizedOutputPort{
		Name:      name,
		Label:     domain.L(labelEN, labelJA),
		IsDefault: isDefault,
	}
}

// LPortWithDesc creates a LocalizedOutputPort with description
func LPortWithDesc(name string, labelEN, labelJA, descEN, descJA string, isDefault bool) domain.LocalizedOutputPort {
	return domain.LocalizedOutputPort{
		Name:        name,
		Label:       domain.L(labelEN, labelJA),
		Description: domain.L(descEN, descJA),
		IsDefault:   isDefault,
	}
}

// LPortWithSchema creates a LocalizedOutputPort with schema
func LPortWithSchema(name string, labelEN, labelJA, descEN, descJA string, isDefault bool, schema json.RawMessage) domain.LocalizedOutputPort {
	return domain.LocalizedOutputPort{
		Name:        name,
		Label:       domain.L(labelEN, labelJA),
		Description: domain.L(descEN, descJA),
		IsDefault:   isDefault,
		Schema:      schema,
	}
}

// LError creates a LocalizedErrorCodeDef
func LError(code, nameEN, nameJA, descEN, descJA string, retryable bool) domain.LocalizedErrorCodeDef {
	return domain.LocalizedErrorCodeDef{
		Code:        code,
		Name:        domain.L(nameEN, nameJA),
		Description: domain.L(descEN, descJA),
		Retryable:   retryable,
	}
}
