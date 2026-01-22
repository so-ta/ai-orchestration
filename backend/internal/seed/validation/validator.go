package validation

import (
	"encoding/json"
	"fmt"

	"github.com/souta/ai-orchestration/internal/seed/blocks"
)

// ValidationError represents a validation error with context
type ValidationError struct {
	BlockSlug string
	Field     string
	Message   string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("[%s.%s] %s", e.BlockSlug, e.Field, e.Message)
}

// BlockValidator validates block definitions
type BlockValidator struct {
	jsValidator     *JSValidator
	schemaValidator *SchemaValidator
}

// NewBlockValidator creates a new block validator
func NewBlockValidator() *BlockValidator {
	return &BlockValidator{
		jsValidator:     NewJSValidator(),
		schemaValidator: NewSchemaValidator(),
	}
}

// ValidateBlock validates a single block definition
func (v *BlockValidator) ValidateBlock(block *blocks.SystemBlockDefinition) []ValidationError {
	var errors []ValidationError

	// Required fields
	if block.Slug == "" {
		errors = append(errors, ValidationError{block.Slug, "slug", "slug is required"})
	}
	if block.Name.EN == "" && block.Name.JA == "" {
		errors = append(errors, ValidationError{block.Slug, "name", "name is required"})
	}
	if !block.Category.IsValid() {
		errors = append(errors, ValidationError{block.Slug, "category", fmt.Sprintf("invalid category: %s", block.Category)})
	}
	if block.Version < 1 {
		errors = append(errors, ValidationError{block.Slug, "version", "version must be >= 1"})
	}

	// JavaScript validation
	if err := v.jsValidator.ValidateSyntax(block.Code); err != nil {
		errors = append(errors, ValidationError{block.Slug, "code", err.Error()})
	}

	// Schema validation (validate EN version, both should be structurally similar)
	if err := v.schemaValidator.ValidateSchema(block.ConfigSchema.EN); err != nil {
		errors = append(errors, ValidationError{block.Slug, "config_schema", err.Error()})
	}
	if err := v.schemaValidator.ValidateSchema(block.OutputSchema); err != nil {
		errors = append(errors, ValidationError{block.Slug, "output_schema", err.Error()})
	}
	if err := v.schemaValidator.ValidateSchema(block.UIConfig.EN); err != nil {
		errors = append(errors, ValidationError{block.Slug, "ui_config", err.Error()})
	}

	// Validate output ports
	for i, port := range block.OutputPorts {
		if port.Name == "" {
			errors = append(errors, ValidationError{block.Slug, fmt.Sprintf("output_ports[%d].name", i), "port name is required"})
		}
		if port.Schema != nil {
			if err := v.schemaValidator.ValidateSchema(port.Schema); err != nil {
				errors = append(errors, ValidationError{block.Slug, fmt.Sprintf("output_ports[%d].schema", i), err.Error()})
			}
		}
	}

	// Validate error codes JSON
	if len(block.ErrorCodes) > 0 {
		errorCodesJSON, err := json.Marshal(block.ErrorCodes)
		if err != nil {
			errors = append(errors, ValidationError{block.Slug, "error_codes", fmt.Sprintf("failed to marshal: %v", err)})
		} else if err := v.schemaValidator.ValidateJSONArray(errorCodesJSON); err != nil {
			errors = append(errors, ValidationError{block.Slug, "error_codes", err.Error()})
		}
	}

	// Validate required credentials JSON
	if len(block.RequiredCredentials) > 0 {
		if err := v.schemaValidator.ValidateJSONArray(block.RequiredCredentials); err != nil {
			errors = append(errors, ValidationError{block.Slug, "required_credentials", err.Error()})
		}
	}

	return errors
}

// ValidateAll validates all blocks in a registry
func (v *BlockValidator) ValidateAll(registry *blocks.Registry) []ValidationError {
	var allErrors []ValidationError

	for _, block := range registry.GetAll() {
		blockErrors := v.ValidateBlock(block)
		allErrors = append(allErrors, blockErrors...)
	}

	return allErrors
}

// ValidateAllWithResult validates all blocks and returns a summary
type ValidationResult struct {
	TotalBlocks   int
	ValidBlocks   int
	InvalidBlocks int
	Errors        []ValidationError
}

func (v *BlockValidator) ValidateAllWithResult(registry *blocks.Registry) *ValidationResult {
	result := &ValidationResult{
		TotalBlocks: registry.Count(),
		Errors:      []ValidationError{},
	}

	invalidSlugs := make(map[string]bool)

	for _, block := range registry.GetAll() {
		blockErrors := v.ValidateBlock(block)
		if len(blockErrors) > 0 {
			invalidSlugs[block.Slug] = true
			result.Errors = append(result.Errors, blockErrors...)
		}
	}

	result.InvalidBlocks = len(invalidSlugs)
	result.ValidBlocks = result.TotalBlocks - result.InvalidBlocks

	return result
}
