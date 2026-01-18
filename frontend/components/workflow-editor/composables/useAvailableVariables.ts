import { computed, type ComputedRef } from 'vue'
import type { Step } from '~/types/api'

export interface AvailableVariable {
  path: string
  type: string
  title?: string
  description?: string
  source: string
}

interface Edge {
  id: string
  source_step_id?: string | null
  target_step_id?: string | null
}

interface OutputSchemaProperty {
  type: string
  title?: string
  description?: string
}

interface ParsedOutputSchema {
  type: string
  properties: Record<string, OutputSchemaProperty>
  required?: string[]
}

export function useAvailableVariables(
  step: ComputedRef<Step | null>,
  steps: ComputedRef<Step[] | undefined>,
  edges: ComputedRef<Edge[] | undefined>
) {
  const previousSteps = computed(() => {
    if (!step.value || !edges.value || !steps.value) return []

    const incomingEdges = edges.value.filter(e => e.target_step_id === step.value?.id)
    const prevStepIds = incomingEdges.map(e => e.source_step_id)

    return steps.value.filter(s => prevStepIds.includes(s.id))
  })

  const availableInputVariables = computed<AvailableVariable[]>(() => {
    const variables: AvailableVariable[] = []

    for (const prevStep of previousSteps.value) {
      const config = prevStep.config as Record<string, unknown> | undefined
      if (!config) continue

      const outputSchema = config.output_schema as ParsedOutputSchema | undefined
      if (!outputSchema || outputSchema.type !== 'object' || !outputSchema.properties) {
        variables.push({
          path: `$.steps.${prevStep.name}.output`,
          type: 'object',
          title: prevStep.name,
          source: prevStep.name
        })
        continue
      }

      for (const [fieldName, fieldDef] of Object.entries(outputSchema.properties)) {
        variables.push({
          path: `$.steps.${prevStep.name}.output.${fieldName}`,
          type: fieldDef.type || 'any',
          title: fieldDef.title || fieldName,
          description: fieldDef.description,
          source: prevStep.name
        })
      }
    }

    variables.unshift({
      path: '$.input',
      type: 'object',
      title: 'ワークフロー入力',
      source: 'input'
    })

    return variables
  })

  const hasAvailableVariables = computed(() => availableInputVariables.value.length > 1)

  return {
    previousSteps,
    availableInputVariables,
    hasAvailableVariables
  }
}
