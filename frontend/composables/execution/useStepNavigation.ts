import type { Ref, ComputedRef } from 'vue'
import type { Step, Run, BlockDefinition } from '~/types/api'

interface SuggestedField {
  name: string
  value: unknown
  type: string
}

interface StepNavigationProps {
  step: Ref<Step | null> | ComputedRef<Step | null>
  steps: Ref<Step[]> | ComputedRef<Step[]>
  edges: Ref<Array<{ id: string; source_step_id?: string | null; target_step_id?: string | null }>> | ComputedRef<Array<{ id: string; source_step_id?: string | null; target_step_id?: string | null }>>
  blocks: Ref<BlockDefinition[]> | ComputedRef<BlockDefinition[]>
  testRuns: Ref<Run[]>
}

/**
 * Composable for step navigation and DAG relationship logic
 */
export function useStepNavigation(props: StepNavigationProps) {
  const { step, steps, edges, blocks, testRuns } = props

  // Find the start step
  // If a Start step is selected, use that one; otherwise fall back to the first Start step
  const startStep = computed(() => {
    if (step.value?.type === 'start') {
      return step.value
    }
    return steps.value.find(s => s.type === 'start')
  })

  // Find the first executable step (after start)
  const firstExecutableStep = computed(() => {
    if (!startStep.value) return null
    const edge = edges.value.find(e => e.source_step_id === startStep.value!.id)
    if (!edge) return null
    return steps.value.find(s => s.id === edge.target_step_id) || null
  })

  // Get the block definition for the first executable step (for workflow execution)
  const firstStepBlock = computed(() => {
    if (!firstExecutableStep.value) return null
    return blocks.value.find(b => b.slug === firstExecutableStep.value!.type) || null
  })

  // Check if the selected step is a start step
  const isStartStep = computed(() => step.value?.type === 'start')

  // Get the block definition for the selected step (for step execution)
  // If start step is selected, use the first executable step's block
  const selectedStepBlock = computed(() => {
    if (!step.value) return null
    // For start step, use the first executable step's block definition
    if (isStartStep.value) {
      return firstStepBlock.value
    }
    return blocks.value.find(b => b.slug === step.value!.type) || null
  })

  // Get the effective step for display (used in descriptions)
  const effectiveStep = computed(() => {
    if (isStartStep.value) {
      return firstExecutableStep.value
    }
    return step.value
  })

  // Get previous step in the workflow (for autocomplete)
  const previousStep = computed(() => {
    if (!step.value) return null
    // Find edge that targets current step
    const incomingEdge = edges.value.find(e => e.target_step_id === step.value!.id)
    if (!incomingEdge) return null
    return steps.value.find(s => s.id === incomingEdge.source_step_id) || null
  })

  // Get previous step's output from latest run (for autocomplete)
  const previousStepOutput = computed(() => {
    if (!previousStep.value || !testRuns.value.length) return null

    for (const run of testRuns.value) {
      if (run.step_runs) {
        const stepRun = run.step_runs.find(sr =>
          sr.step_id === previousStep.value!.id &&
          sr.status === 'completed' &&
          sr.output
        )
        if (stepRun?.output) {
          return stepRun.output as Record<string, unknown>
        }
      }
    }
    return null
  })

  // Get suggested fields from previous step output
  const suggestedFields = computed<SuggestedField[]>(() => {
    const output = previousStepOutput.value
    if (!output || typeof output !== 'object') return []

    return Object.entries(output).map(([name, value]) => ({
      name,
      value,
      type: Array.isArray(value) ? 'array' : typeof value
    }))
  })

  // Get the latest step run output for current step
  const latestStepRunOutput = computed(() => {
    if (!step.value || !testRuns.value.length) return null

    // Find the latest completed step run for this step
    for (const run of testRuns.value) {
      if (run.step_runs) {
        const stepRun = run.step_runs.find(sr =>
          sr.step_id === step.value!.id &&
          sr.status === 'completed' &&
          sr.output
        )
        if (stepRun?.output) {
          return stepRun.output
        }
      }
    }
    return null
  })

  return {
    startStep,
    firstExecutableStep,
    firstStepBlock,
    isStartStep,
    selectedStepBlock,
    effectiveStep,
    previousStep,
    previousStepOutput,
    suggestedFields,
    latestStepRunOutput,
  }
}
