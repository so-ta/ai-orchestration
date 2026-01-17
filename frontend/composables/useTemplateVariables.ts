/**
 * Template Variables Composable
 *
 * テンプレート変数の検出、フォーマット、解決を提供する。
 * ExecutionTabとPropertiesPanelで使用。
 */

import { computed, type Ref, type ComputedRef } from 'vue'

// テンプレート変数のパターン: {{variable}} または {{path.to.value}}
const TEMPLATE_REGEX = /\{\{([^}]+)\}\}/g

// i18nパーサーとの競合を避けるため、文字コードから生成
const OPEN_BRACE = String.fromCharCode(123, 123) // {{
const CLOSE_BRACE = String.fromCharCode(125, 125) // }}

export interface TemplatePreviewItem {
  /** 変数名（パス） */
  variable: string
  /** 解決された値（または未解決の場合はプレースホルダー） */
  resolved: string
  /** 解決されたかどうか */
  isResolved: boolean
}

export interface UseTemplateVariablesReturn {
  /** 設定内のテンプレート変数リスト */
  variables: ComputedRef<string[]>
  /** 変数を表示用にフォーマット */
  formatVariable: (variable: string) => string
  /** 変数の値を解決 */
  resolveVariable: (varPath: string, context: Record<string, unknown>) => string
  /** プレビュー用の変数リスト（解決状態付き） */
  createPreview: (context: Record<string, unknown>) => TemplatePreviewItem[]
}

/**
 * オブジェクト内のすべてのテンプレート変数を抽出する
 *
 * @param obj - 検索対象のオブジェクト
 * @returns テンプレート変数名のセット
 */
export function extractTemplateVariables(obj: unknown): Set<string> {
  const variables = new Set<string>()

  function findVariables(value: unknown): void {
    if (typeof value === 'string') {
      // 正規表現をリセット（グローバルフラグ使用時）
      const regex = new RegExp(TEMPLATE_REGEX.source, 'g')
      let match
      while ((match = regex.exec(value)) !== null) {
        const varName = match[1].trim()
        variables.add(varName)
      }
    } else if (Array.isArray(value)) {
      for (const item of value) {
        findVariables(item)
      }
    } else if (value && typeof value === 'object') {
      for (const v of Object.values(value)) {
        findVariables(v)
      }
    }
  }

  findVariables(obj)
  return variables
}

/**
 * テンプレート変数を表示用にフォーマットする
 *
 * @param variable - 変数名
 * @returns フォーマットされた変数（{{variable}}形式）
 */
export function formatTemplateVariable(variable: string): string {
  return OPEN_BRACE + variable + CLOSE_BRACE
}

/**
 * テンプレート変数の値を解決する
 *
 * @param varPath - 変数のパス（例: "message" または "data.content"）
 * @param context - 変数を解決するためのコンテキスト
 * @returns 解決された値（文字列）、未解決の場合はプレースホルダー
 */
export function resolveTemplateVariable(
  varPath: string,
  context: Record<string, unknown>
): string {
  const parts = varPath.split('.')
  let value: unknown = context

  for (const part of parts) {
    if (value && typeof value === 'object') {
      value = (value as Record<string, unknown>)[part]
    } else {
      return formatTemplateVariable(varPath)
    }
  }

  if (value === undefined || value === null) {
    return formatTemplateVariable(varPath)
  }

  if (typeof value === 'object') {
    return JSON.stringify(value)
  }

  return String(value)
}

/**
 * テンプレート変数のためのcomposable
 *
 * @param config - 設定オブジェクトのRef
 * @returns テンプレート変数の操作関数
 *
 * @example
 * const { variables, formatVariable, resolveVariable, createPreview } = useTemplateVariables(
 *   computed(() => props.step?.config)
 * )
 *
 * // 変数リストを取得
 * console.log(variables.value) // ['message', 'data.content']
 *
 * // フォーマット
 * formatVariable('message') // '{{message}}'
 *
 * // 解決
 * resolveVariable('message', { message: 'Hello' }) // 'Hello'
 *
 * // プレビュー作成
 * createPreview({ message: 'Hello' })
 * // [{ variable: 'message', resolved: 'Hello', isResolved: true }]
 */
export function useTemplateVariables(
  config: Ref<Record<string, unknown> | null | undefined>
): UseTemplateVariablesReturn {
  const variables = computed<string[]>(() => {
    if (!config.value) return []
    return Array.from(extractTemplateVariables(config.value))
  })

  function createPreview(context: Record<string, unknown>): TemplatePreviewItem[] {
    return variables.value.map(variable => {
      const resolved = resolveTemplateVariable(variable, context)
      return {
        variable,
        resolved,
        isResolved: !resolved.startsWith(OPEN_BRACE),
      }
    })
  }

  return {
    variables,
    formatVariable: formatTemplateVariable,
    resolveVariable: resolveTemplateVariable,
    createPreview,
  }
}
