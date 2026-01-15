package blocks

import "sort"

// Registry holds all system block definitions
type Registry struct {
	blocks map[string]*SystemBlockDefinition
}

// NewRegistry creates a new block registry with all system blocks
func NewRegistry() *Registry {
	r := &Registry{
		blocks: make(map[string]*SystemBlockDefinition),
	}

	// Register all blocks by category
	r.registerAIBlocks()
	r.registerLogicBlocks()
	r.registerControlBlocks()
	r.registerDataBlocks()
	r.registerIntegrationBlocks()
	r.registerUtilityBlocks()
	r.registerRAGBlocks()
	r.registerGroupBlocks()

	return r
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
