import { computed, type Ref } from 'vue'

export interface SwitchCase {
  name: string
  expression: string
  is_default?: boolean
}

export interface StepConfig {
  cases?: SwitchCase[]
  [key: string]: unknown
}

export function useSwitchCases(formConfig: Ref<StepConfig>) {
  const switchCases = computed({
    get: () => (formConfig.value.cases as SwitchCase[]) || [],
    set: (val) => {
      formConfig.value.cases = val
    }
  })

  function addSwitchCase() {
    const cases = [...(formConfig.value.cases || [])]
    const newIndex = cases.length + 1
    cases.push({
      name: `case_${newIndex}`,
      expression: '',
      is_default: false
    })
    formConfig.value.cases = cases
  }

  function removeSwitchCase(index: number) {
    const cases = [...(formConfig.value.cases || [])]
    cases.splice(index, 1)
    formConfig.value.cases = cases
  }

  function updateSwitchCase(index: number, field: 'name' | 'expression' | 'is_default', value: string | boolean) {
    const cases = [...(formConfig.value.cases || [])]
    if (cases[index]) {
      if (field === 'is_default') {
        // Only one case can be default
        cases.forEach((c, i) => {
          c.is_default = i === index ? Boolean(value) : false
        })
      } else if (field === 'name') {
        cases[index].name = value as string
      } else if (field === 'expression') {
        cases[index].expression = value as string
      }
      formConfig.value.cases = cases
    }
  }

  function getCaseDisplayName(caseName: string, index: number): string {
    return caseName || 'case_' + (index + 1)
  }

  return {
    switchCases,
    addSwitchCase,
    removeSwitchCase,
    updateSwitchCase,
    getCaseDisplayName
  }
}
