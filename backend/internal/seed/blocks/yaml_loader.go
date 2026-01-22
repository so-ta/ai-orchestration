package blocks

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/souta/ai-orchestration/internal/domain"
	"gopkg.in/yaml.v3"
)

// YAMLBlockDefinition represents a block definition in YAML format
// This is the YAML-friendly version that gets converted to SystemBlockDefinition
type YAMLBlockDefinition struct {
	// Identifiers
	Slug    string `yaml:"slug"`
	Version int    `yaml:"version"`

	// Basic info (supports i18n with en/ja keys)
	Name        interface{} `yaml:"name"`        // string or {en: "", ja: ""}
	Description interface{} `yaml:"description"` // string or {en: "", ja: ""}
	Category    string      `yaml:"category"`
	Subcategory string `yaml:"subcategory,omitempty"`
	Icon        string `yaml:"icon"`

	// Schema definitions (as YAML objects, converted to JSON)
	ConfigSchema interface{} `yaml:"config_schema,omitempty"`
	OutputSchema interface{} `yaml:"output_schema,omitempty"`

	// Ports
	OutputPorts []YAMLOutputPort `yaml:"output_ports,omitempty"`

	// Execution
	Code        string `yaml:"code,omitempty"`
	PreProcess  string `yaml:"pre_process,omitempty"`
	PostProcess string `yaml:"post_process,omitempty"`

	// Inheritance
	ParentBlockSlug string      `yaml:"parent_block_slug,omitempty"`
	ConfigDefaults  interface{} `yaml:"config_defaults,omitempty"`

	// Declarative request/response
	Request  *YAMLRequestConfig  `yaml:"request,omitempty"`
	Response *YAMLResponseConfig `yaml:"response,omitempty"`

	// UI
	UIConfig interface{} `yaml:"ui_config,omitempty"`

	// Error handling and credentials
	ErrorCodes          []YAMLErrorCodeDef `yaml:"error_codes,omitempty"`
	RequiredCredentials interface{}        `yaml:"required_credentials,omitempty"`

	// Flags
	Enabled bool `yaml:"enabled"`

	// Group block fields
	GroupKind   string `yaml:"group_kind,omitempty"`
	IsContainer bool   `yaml:"is_container,omitempty"`

	// Internal steps for composite blocks
	InternalSteps []YAMLInternalStep `yaml:"internal_steps,omitempty"`
}

// YAMLOutputPort represents an output port in YAML
type YAMLOutputPort struct {
	Name        string      `yaml:"name"`
	Label       interface{} `yaml:"label"`       // string or {en: "", ja: ""}
	Description interface{} `yaml:"description"` // string or {en: "", ja: ""}
	IsDefault   bool        `yaml:"is_default"`
	Schema      interface{} `yaml:"schema,omitempty"`
}

// YAMLErrorCodeDef represents an error code in YAML
type YAMLErrorCodeDef struct {
	Code        string      `yaml:"code"`
	Name        interface{} `yaml:"name"`        // string or {en: "", ja: ""}
	Description interface{} `yaml:"description"` // string or {en: "", ja: ""}
	Retryable   bool        `yaml:"retryable"`
}

// YAMLRequestConfig represents declarative request config in YAML
type YAMLRequestConfig struct {
	URL         string            `yaml:"url,omitempty"`
	Method      string            `yaml:"method,omitempty"`
	Body        interface{}       `yaml:"body,omitempty"`
	Headers     map[string]string `yaml:"headers,omitempty"`
	QueryParams map[string]string `yaml:"query_params,omitempty"`
}

// YAMLResponseConfig represents declarative response config in YAML
type YAMLResponseConfig struct {
	OutputMapping map[string]string `yaml:"output_mapping,omitempty"`
	SuccessStatus []int             `yaml:"success_status,omitempty"`
}

// YAMLInternalStep represents an internal step in YAML
type YAMLInternalStep struct {
	Type      string      `yaml:"type"`
	Config    interface{} `yaml:"config,omitempty"`
	OutputKey string      `yaml:"output_key"`
}

// YAMLLoader loads block definitions from YAML files
type YAMLLoader struct {
	directories []string
}

// NewYAMLLoader creates a new YAML loader with the given directories
func NewYAMLLoader(directories ...string) *YAMLLoader {
	return &YAMLLoader{
		directories: directories,
	}
}

// LoadAll loads all block definitions from all configured directories
func (l *YAMLLoader) LoadAll() ([]*SystemBlockDefinition, error) {
	var allBlocks []*SystemBlockDefinition

	for _, dir := range l.directories {
		blocks, err := l.loadFromDirectory(dir)
		if err != nil {
			return nil, fmt.Errorf("failed to load blocks from %s: %w", dir, err)
		}
		allBlocks = append(allBlocks, blocks...)
	}

	// Sort by slug for consistent ordering
	sort.Slice(allBlocks, func(i, j int) bool {
		return allBlocks[i].Slug < allBlocks[j].Slug
	})

	return allBlocks, nil
}

// loadFromDirectory loads all YAML files from a directory
func (l *YAMLLoader) loadFromDirectory(dir string) ([]*SystemBlockDefinition, error) {
	var blocks []*SystemBlockDefinition

	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// Directory doesn't exist, return empty list
		return blocks, nil
	}

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-YAML files
		if d.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		// Load file
		fileBlocks, err := l.loadFile(path)
		if err != nil {
			return fmt.Errorf("failed to load %s: %w", path, err)
		}

		blocks = append(blocks, fileBlocks...)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return blocks, nil
}

// loadFile loads block definitions from a single YAML file
// Supports multi-document YAML (multiple blocks in one file separated by ---)
func (l *YAMLLoader) loadFile(path string) ([]*SystemBlockDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var blocks []*SystemBlockDefinition

	// Split by document separator and parse each
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))

	for {
		var yamlBlock YAMLBlockDefinition
		err := decoder.Decode(&yamlBlock)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}

		// Skip empty documents
		if yamlBlock.Slug == "" {
			continue
		}

		// Convert to SystemBlockDefinition
		block, err := convertYAMLToSystemBlock(&yamlBlock)
		if err != nil {
			return nil, fmt.Errorf("failed to convert block %s: %w", yamlBlock.Slug, err)
		}

		blocks = append(blocks, block)
	}

	return blocks, nil
}

// convertYAMLToSystemBlock converts a YAML block definition to SystemBlockDefinition
func convertYAMLToSystemBlock(y *YAMLBlockDefinition) (*SystemBlockDefinition, error) {
	block := &SystemBlockDefinition{
		Slug:            y.Slug,
		Version:         y.Version,
		Name:            parseLocalizedText(y.Name),
		Description:     parseLocalizedText(y.Description),
		Category:        domain.BlockCategory(y.Category),
		Subcategory:     domain.BlockSubcategory(y.Subcategory),
		Icon:            y.Icon,
		Code:            strings.TrimSpace(y.Code),
		PreProcess:      strings.TrimSpace(y.PreProcess),
		PostProcess:     strings.TrimSpace(y.PostProcess),
		ParentBlockSlug: y.ParentBlockSlug,
		Enabled:         y.Enabled,
		GroupKind:       domain.BlockGroupKind(y.GroupKind),
		IsContainer:     y.IsContainer,
	}

	// Convert schemas to JSON (as LocalizedConfigSchema - same for both en/ja from YAML)
	if y.ConfigSchema != nil {
		jsonData, err := toJSONRawMessage(y.ConfigSchema)
		if err != nil {
			return nil, fmt.Errorf("invalid config_schema: %w", err)
		}
		block.ConfigSchema = domain.LocalizedConfigSchema{EN: jsonData, JA: jsonData}
	}

	if y.OutputSchema != nil {
		jsonData, err := toJSONRawMessage(y.OutputSchema)
		if err != nil {
			return nil, fmt.Errorf("invalid output_schema: %w", err)
		}
		block.OutputSchema = jsonData
	}

	if y.ConfigDefaults != nil {
		jsonData, err := toJSONRawMessage(y.ConfigDefaults)
		if err != nil {
			return nil, fmt.Errorf("invalid config_defaults: %w", err)
		}
		block.ConfigDefaults = jsonData
	}

	if y.UIConfig != nil {
		jsonData, err := toJSONRawMessage(y.UIConfig)
		if err != nil {
			return nil, fmt.Errorf("invalid ui_config: %w", err)
		}
		block.UIConfig = domain.LocalizedConfigSchema{EN: jsonData, JA: jsonData}
	}

	if y.RequiredCredentials != nil {
		jsonData, err := toJSONRawMessage(y.RequiredCredentials)
		if err != nil {
			return nil, fmt.Errorf("invalid required_credentials: %w", err)
		}
		block.RequiredCredentials = jsonData
	}

	// Convert output ports
	for _, p := range y.OutputPorts {
		port := domain.LocalizedOutputPort{
			Name:        p.Name,
			Label:       parseLocalizedText(p.Label),
			Description: parseLocalizedText(p.Description),
			IsDefault:   p.IsDefault,
		}
		if p.Schema != nil {
			jsonData, err := toJSONRawMessage(p.Schema)
			if err != nil {
				return nil, fmt.Errorf("invalid output port schema for %s: %w", p.Name, err)
			}
			port.Schema = jsonData
		}
		block.OutputPorts = append(block.OutputPorts, port)
	}

	// Convert error codes
	for _, e := range y.ErrorCodes {
		block.ErrorCodes = append(block.ErrorCodes, domain.LocalizedErrorCodeDef{
			Code:        e.Code,
			Name:        parseLocalizedText(e.Name),
			Description: parseLocalizedText(e.Description),
			Retryable:   e.Retryable,
		})
	}

	// Convert internal steps
	for _, s := range y.InternalSteps {
		step := domain.InternalStep{
			Type:      s.Type,
			OutputKey: s.OutputKey,
		}
		if s.Config != nil {
			jsonData, err := toJSONRawMessage(s.Config)
			if err != nil {
				return nil, fmt.Errorf("invalid internal step config: %w", err)
			}
			step.Config = jsonData
		}
		block.InternalSteps = append(block.InternalSteps, step)
	}

	// Convert declarative request config
	if y.Request != nil {
		block.Request = &domain.RequestConfig{
			URL:         y.Request.URL,
			Method:      y.Request.Method,
			Headers:     y.Request.Headers,
			QueryParams: y.Request.QueryParams,
		}
		if y.Request.Body != nil {
			bodyMap, ok := y.Request.Body.(map[string]interface{})
			if ok {
				block.Request.Body = bodyMap
			}
		}
	}

	// Convert declarative response config
	if y.Response != nil {
		block.Response = &domain.ResponseConfig{
			OutputMapping: y.Response.OutputMapping,
			SuccessStatus: y.Response.SuccessStatus,
		}
	}

	return block, nil
}

// toJSONRawMessage converts an interface{} to json.RawMessage
func toJSONRawMessage(v interface{}) (json.RawMessage, error) {
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(jsonBytes), nil
}

// parseLocalizedText converts a YAML value to LocalizedText
// Supports both plain string and {en: "", ja: ""} format
func parseLocalizedText(v interface{}) domain.LocalizedText {
	if v == nil {
		return domain.LocalizedText{}
	}

	// Plain string - use for both languages
	if s, ok := v.(string); ok {
		return domain.LocalizedText{EN: s, JA: s}
	}

	// Map with en/ja keys
	if m, ok := v.(map[string]interface{}); ok {
		lt := domain.LocalizedText{}
		if en, ok := m["en"].(string); ok {
			lt.EN = en
		}
		if ja, ok := m["ja"].(string); ok {
			lt.JA = ja
		}
		// If only one language is provided, use it for both
		if lt.EN == "" && lt.JA != "" {
			lt.EN = lt.JA
		}
		if lt.JA == "" && lt.EN != "" {
			lt.JA = lt.EN
		}
		return lt
	}

	return domain.LocalizedText{}
}

// LoadFromEmbed loads blocks from embedded filesystem (for future use)
// This can be used to embed YAML files directly into the binary
func (l *YAMLLoader) LoadFromEmbed(fsys fs.FS, root string) ([]*SystemBlockDefinition, error) {
	var blocks []*SystemBlockDefinition

	err := fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		decoder := yaml.NewDecoder(strings.NewReader(string(data)))
		for {
			var yamlBlock YAMLBlockDefinition
			err := decoder.Decode(&yamlBlock)
			if err != nil {
				if err.Error() == "EOF" {
					break
				}
				return fmt.Errorf("failed to parse %s: %w", path, err)
			}

			if yamlBlock.Slug == "" {
				continue
			}

			block, err := convertYAMLToSystemBlock(&yamlBlock)
			if err != nil {
				return fmt.Errorf("failed to convert block %s in %s: %w", yamlBlock.Slug, path, err)
			}

			blocks = append(blocks, block)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i].Slug < blocks[j].Slug
	})

	return blocks, nil
}
