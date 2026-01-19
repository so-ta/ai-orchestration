package blocks

import (
	"path/filepath"
	"runtime"
	"sort"
)

// Registry holds all system block definitions
type Registry struct {
	blocks map[string]*SystemBlockDefinition
}

// NewRegistry creates a new block registry with all system blocks
// This loads blocks from both Go code and YAML files (YAML takes precedence for overrides)
func NewRegistry() *Registry {
	r := &Registry{
		blocks: make(map[string]*SystemBlockDefinition),
	}

	// Register all blocks from Go code by category
	r.registerAIBlocks()
	r.registerLogicBlocks()
	r.registerControlBlocks()
	r.registerDataBlocks()
	r.registerIntegrationBlocks()
	r.registerUtilityBlocks()
	r.registerRAGBlocks()
	r.registerGroupBlocks()

	// Load additional/override blocks from YAML files
	r.loadYAMLBlocks()

	return r
}

// loadYAMLBlocks loads block definitions from YAML files
// YAML blocks override Go blocks with the same slug
func (r *Registry) loadYAMLBlocks() {
	// Get the directory of this source file to find yaml/ relative to it
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return
	}
	yamlDir := filepath.Join(filepath.Dir(filename), "yaml")

	loader := NewYAMLLoader(yamlDir)
	blocks, err := loader.LoadAll()
	if err != nil {
		// Log error but don't fail - Go blocks will be used
		return
	}

	// Register YAML blocks (overrides existing blocks with same slug)
	for _, block := range blocks {
		r.register(block)
	}
}

// NewRegistryFromYAML creates a registry loading only from YAML directories
// This is useful for testing or when you want to use only YAML-defined blocks
func NewRegistryFromYAML(directories ...string) (*Registry, error) {
	r := &Registry{
		blocks: make(map[string]*SystemBlockDefinition),
	}

	loader := NewYAMLLoader(directories...)
	blocks, err := loader.LoadAll()
	if err != nil {
		return nil, err
	}

	for _, block := range blocks {
		r.register(block)
	}

	return r, nil
}

// GetAll returns all registered blocks sorted by slug
func (r *Registry) GetAll() []*SystemBlockDefinition {
	result := make([]*SystemBlockDefinition, 0, len(r.blocks))
	for _, b := range r.blocks {
		result = append(result, b)
	}
	// Sort by slug for consistent ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].Slug < result[j].Slug
	})
	return result
}

// GetBySlug returns a block by its slug
func (r *Registry) GetBySlug(slug string) (*SystemBlockDefinition, bool) {
	b, ok := r.blocks[slug]
	return b, ok
}

// Count returns the number of registered blocks
func (r *Registry) Count() int {
	return len(r.blocks)
}

// register adds a block to the registry
func (r *Registry) register(block *SystemBlockDefinition) {
	r.blocks[block.Slug] = block
}
