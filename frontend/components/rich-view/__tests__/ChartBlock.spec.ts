import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import type { ChartConfig } from '~/types/rich-view'

// Use vi.hoisted to create mock that can be referenced in vi.mock factory
const { mockDestroyFn, MockChart } = vi.hoisted(() => {
  const mockDestroyFn = vi.fn()

  const mockChartInstance = {
    destroy: mockDestroyFn,
  }

  class MockChart {
    static register = vi.fn()
    constructor() {
      return mockChartInstance
    }
  }

  return { mockDestroyFn, MockChart }
})

vi.mock('chart.js', () => {
  return {
    Chart: MockChart,
    BarController: {},
    LineController: {},
    PieController: {},
    DoughnutController: {},
    CategoryScale: {},
    LinearScale: {},
    BarElement: {},
    LineElement: {},
    PointElement: {},
    ArcElement: {},
    Title: {},
    Tooltip: {},
    Legend: {},
  }
})

// Import component after mock setup
import ChartBlock from '../ChartBlock.vue'

describe('ChartBlock', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    // Mock canvas getContext
    HTMLCanvasElement.prototype.getContext = vi.fn(() => ({
      canvas: { width: 400, height: 300 },
    })) as unknown as typeof HTMLCanvasElement.prototype.getContext
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  const baseConfig: ChartConfig = {
    type: 'bar',
    labels: ['A', 'B', 'C'],
    datasets: [
      {
        label: 'Dataset 1',
        data: [10, 20, 30],
      },
    ],
  }

  it('renders with basic config', () => {
    const wrapper = mount(ChartBlock, {
      props: {
        config: baseConfig,
      },
    })

    expect(wrapper.find('.chart-block').exists()).toBe(true)
    expect(wrapper.find('.chart-container').exists()).toBe(true)
    expect(wrapper.find('canvas').exists()).toBe(true)
  })

  it('applies default height of 300px', () => {
    const wrapper = mount(ChartBlock, {
      props: {
        config: baseConfig,
      },
    })

    expect(wrapper.find('.chart-container').attributes('style')).toContain('height: 300px')
  })

  it('applies custom height', () => {
    const wrapper = mount(ChartBlock, {
      props: {
        config: {
          ...baseConfig,
          height: 500,
        },
      },
    })

    expect(wrapper.find('.chart-container').attributes('style')).toContain('height: 500px')
  })

  it('renders with line chart type', () => {
    const wrapper = mount(ChartBlock, {
      props: {
        config: {
          ...baseConfig,
          type: 'line',
        },
      },
    })

    expect(wrapper.find('.chart-block').exists()).toBe(true)
  })

  it('renders with pie chart type', () => {
    const wrapper = mount(ChartBlock, {
      props: {
        config: {
          ...baseConfig,
          type: 'pie',
        },
      },
    })

    expect(wrapper.find('.chart-block').exists()).toBe(true)
  })

  it('renders with doughnut chart type', () => {
    const wrapper = mount(ChartBlock, {
      props: {
        config: {
          ...baseConfig,
          type: 'doughnut',
        },
      },
    })

    expect(wrapper.find('.chart-block').exists()).toBe(true)
  })

  it('renders with multiple datasets', () => {
    const wrapper = mount(ChartBlock, {
      props: {
        config: {
          ...baseConfig,
          datasets: [
            { label: 'Dataset 1', data: [10, 20, 30] },
            { label: 'Dataset 2', data: [15, 25, 35] },
            { label: 'Dataset 3', data: [5, 10, 15] },
          ],
        },
      },
    })

    expect(wrapper.find('.chart-block').exists()).toBe(true)
  })

  it('renders with custom colors', () => {
    const wrapper = mount(ChartBlock, {
      props: {
        config: {
          ...baseConfig,
          datasets: [
            { label: 'Dataset 1', data: [10, 20, 30], color: '#ff0000' },
          ],
        },
      },
    })

    expect(wrapper.find('.chart-block').exists()).toBe(true)
  })

  it('renders with stacked option', () => {
    const wrapper = mount(ChartBlock, {
      props: {
        config: {
          ...baseConfig,
          stacked: true,
        },
      },
    })

    expect(wrapper.find('.chart-block').exists()).toBe(true)
  })

  it('renders with showLegend option', () => {
    const wrapper = mount(ChartBlock, {
      props: {
        config: {
          ...baseConfig,
          showLegend: false,
        },
      },
    })

    expect(wrapper.find('.chart-block').exists()).toBe(true)
  })

  it('destroys chart on unmount', async () => {
    const wrapper = mount(ChartBlock, {
      props: {
        config: baseConfig,
      },
    })

    wrapper.unmount()

    expect(mockDestroyFn).toHaveBeenCalled()
  })
})
