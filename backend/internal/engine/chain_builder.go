package engine

import (
	"github.com/google/uuid"
	"github.com/souta/ai-orchestration/internal/domain"
)

// ChainBuilder builds step chains from entry points within a block group.
// Entry points are steps with no incoming edges from within the same group.
type ChainBuilder struct {
	steps []*domain.Step
	edges []*domain.Edge
}

// NewChainBuilder creates a new ChainBuilder
func NewChainBuilder(steps []*domain.Step, edges []*domain.Edge) *ChainBuilder {
	return &ChainBuilder{
		steps: steps,
		edges: edges,
	}
}

// ToolChain represents an entry point and its chain of steps
type ToolChain struct {
	EntryPoint  *domain.Step   // The entry point step (no incoming edges)
	Chain       []*domain.Step // Steps in execution order
	ToolName    string         // Tool name (from step.ToolName or step.Name)
	Description string         // Tool description
	InputSchema interface{}    // Tool input schema
}

// FindEntryPoints detects steps with no incoming edges from within the same group.
// These are the entry points (tool starting points for agent groups, branch starts for parallel, etc.)
func (cb *ChainBuilder) FindEntryPoints(groupID uuid.UUID) []*domain.Step {
	// Get steps belonging to this group
	groupSteps := cb.filterStepsByGroup(groupID)
	if len(groupSteps) == 0 {
		return nil
	}

	// Build a set of step IDs in this group for quick lookup
	groupStepIDs := make(map[uuid.UUID]bool)
	for _, step := range groupSteps {
		groupStepIDs[step.ID] = true
	}

	// Find steps that have incoming edges from within the same group
	hasIncomingEdge := make(map[uuid.UUID]bool)
	for _, edge := range cb.edges {
		// Only consider edges that target a step in this group
		if edge.TargetStepID == nil {
			continue
		}
		if !groupStepIDs[*edge.TargetStepID] {
			continue
		}

		// Check if source is also in this group
		if edge.SourceStepID != nil && groupStepIDs[*edge.SourceStepID] {
			hasIncomingEdge[*edge.TargetStepID] = true
		}
	}

	// Steps without incoming edges from the same group are entry points
	var entryPoints []*domain.Step
	for _, step := range groupSteps {
		if !hasIncomingEdge[step.ID] {
			entryPoints = append(entryPoints, step)
		}
	}

	return entryPoints
}

// BuildChain builds a chain of steps starting from an entry point.
// It follows edges to find the sequence of steps to execute.
// Returns steps in execution order.
func (cb *ChainBuilder) BuildChain(entryPoint *domain.Step) []*domain.Step {
	if entryPoint == nil || entryPoint.BlockGroupID == nil {
		return nil
	}

	groupID := *entryPoint.BlockGroupID

	// Get steps in this group
	groupSteps := cb.filterStepsByGroup(groupID)
	stepMap := make(map[uuid.UUID]*domain.Step)
	for _, step := range groupSteps {
		stepMap[step.ID] = step
	}

	// Build outgoing edge map (source step -> target steps)
	outgoingEdges := make(map[uuid.UUID][]*domain.Step)
	for _, edge := range cb.edges {
		if edge.SourceStepID == nil || edge.TargetStepID == nil {
			continue
		}
		targetStep, ok := stepMap[*edge.TargetStepID]
		if !ok {
			continue // Target is not in this group
		}
		sourceStep, ok := stepMap[*edge.SourceStepID]
		if !ok {
			continue // Source is not in this group
		}
		outgoingEdges[sourceStep.ID] = append(outgoingEdges[sourceStep.ID], targetStep)
	}

	// Build chain by following edges (simple linear traversal)
	// Note: This assumes linear chains. For branching, we'd need more complex logic.
	var chain []*domain.Step
	visited := make(map[uuid.UUID]bool)
	current := entryPoint

	for current != nil && !visited[current.ID] {
		chain = append(chain, current)
		visited[current.ID] = true

		// Find next step (follow first outgoing edge within the group)
		nextSteps := outgoingEdges[current.ID]
		current = nil
		for _, next := range nextSteps {
			if !visited[next.ID] {
				current = next
				break
			}
		}
	}

	return chain
}

// BuildToolChains builds tool chains for all entry points in a group.
// Each entry point becomes a tool that executes its chain.
func (cb *ChainBuilder) BuildToolChains(groupID uuid.UUID) []*ToolChain {
	entryPoints := cb.FindEntryPoints(groupID)
	if len(entryPoints) == 0 {
		return nil
	}

	var toolChains []*ToolChain
	for _, ep := range entryPoints {
		chain := cb.BuildChain(ep)

		// Get tool name (prefer ToolName, fallback to step Name)
		toolName := ep.Name
		if ep.ToolName != nil && *ep.ToolName != "" {
			toolName = *ep.ToolName
		}

		// Get description
		description := ep.Name
		if ep.ToolDescription != nil && *ep.ToolDescription != "" {
			description = *ep.ToolDescription
		}

		// Get input schema
		var inputSchema interface{}
		if len(ep.ToolInputSchema) > 0 {
			inputSchema = ep.ToolInputSchema
		}

		toolChains = append(toolChains, &ToolChain{
			EntryPoint:  ep,
			Chain:       chain,
			ToolName:    toolName,
			Description: description,
			InputSchema: inputSchema,
		})
	}

	return toolChains
}

// filterStepsByGroup returns steps belonging to the specified group
func (cb *ChainBuilder) filterStepsByGroup(groupID uuid.UUID) []*domain.Step {
	var result []*domain.Step
	for _, step := range cb.steps {
		if step.BlockGroupID != nil && *step.BlockGroupID == groupID {
			result = append(result, step)
		}
	}
	return result
}

// ValidateEntryPointCount validates that the group has the expected number of entry points.
// Returns an error message if validation fails, empty string if valid.
func (cb *ChainBuilder) ValidateEntryPointCount(groupID uuid.UUID, groupType domain.BlockGroupType) string {
	entryPoints := cb.FindEntryPoints(groupID)
	count := len(entryPoints)

	switch groupType {
	case domain.BlockGroupTypeAgent:
		if count == 0 {
			return "agent group requires at least one entry point step"
		}
	case domain.BlockGroupTypeParallel:
		if count == 0 {
			return "parallel group requires at least one entry point"
		}
	case domain.BlockGroupTypeTryCatch:
		if count != 1 {
			return "try_catch group requires exactly one entry point"
		}
	case domain.BlockGroupTypeForeach:
		if count != 1 {
			return "foreach group requires exactly one entry point"
		}
	case domain.BlockGroupTypeWhile:
		if count != 1 {
			return "while group requires exactly one entry point"
		}
	}

	return ""
}
