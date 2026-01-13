// Rich View Output types for Extended Markdown

/**
 * Block output with optional markdown representation
 */
export interface BlockOutput {
  data: unknown
  markdown?: string
}

/**
 * Chart configuration for ```chart code blocks
 */
export interface ChartConfig {
  type: 'bar' | 'line' | 'pie' | 'doughnut'
  labels: string[]
  datasets: ChartDataset[]
  height?: number
  stacked?: boolean
  showLegend?: boolean
}

export interface ChartDataset {
  label?: string
  data: number[]
  color?: string
}

/**
 * Progress configuration for ```progress code blocks
 */
export interface ProgressConfig {
  value: number
  label?: string
  color?: string | 'auto'
  size?: 'sm' | 'md' | 'lg'
}

/**
 * Extended code block types
 */
export type ExtendedCodeBlockType = 'chart' | 'progress'

/**
 * Parsed extended code block
 */
export interface ExtendedCodeBlock {
  type: ExtendedCodeBlockType
  config: ChartConfig | ProgressConfig
  raw: string
}
