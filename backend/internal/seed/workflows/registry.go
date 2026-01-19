package workflows

import "sort"

// Registry holds all system workflow definitions
type Registry struct {
	workflows map[string]*SystemWorkflowDefinition
}

// NewRegistry creates a new workflow registry with all system workflows
func NewRegistry() *Registry {
	r := &Registry{
		workflows: make(map[string]*SystemWorkflowDefinition),
	}

	// Register all system workflows
	r.registerRAGWorkflows()
	r.registerDemoWorkflows()
	r.registerCopilotWorkflows()

	return r
}

// GetAll returns all registered workflows sorted by system_slug
func (r *Registry) GetAll() []*SystemWorkflowDefinition {
	result := make([]*SystemWorkflowDefinition, 0, len(r.workflows))
	for _, w := range r.workflows {
		result = append(result, w)
	}
	// Sort by system_slug for consistent ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].SystemSlug < result[j].SystemSlug
	})
	return result
}

// GetBySlug returns a workflow by its system_slug
func (r *Registry) GetBySlug(slug string) (*SystemWorkflowDefinition, bool) {
	w, ok := r.workflows[slug]
	return w, ok
}

// Count returns the number of registered workflows
func (r *Registry) Count() int {
	return len(r.workflows)
}

// register adds a workflow to the registry
func (r *Registry) register(workflow *SystemWorkflowDefinition) {
	r.workflows[workflow.SystemSlug] = workflow
}
