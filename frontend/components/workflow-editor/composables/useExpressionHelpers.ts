import type { Ref } from 'vue'

export interface ExpressionConfig {
  expression?: string
  [key: string]: unknown
}

export const expressionTemplates = {
  equals: '$.field == "value"',
  notEquals: '$.field != "value"',
  greaterThan: '$.field > 0',
  lessThan: '$.field < 0',
  exists: '$.field'
} as const

export function useExpressionHelpers(formConfig: Ref<ExpressionConfig>) {
  function insertExpression(expr: string) {
    formConfig.value.expression = (formConfig.value.expression || '') + expr
  }

  return {
    expressionTemplates,
    insertExpression
  }
}
