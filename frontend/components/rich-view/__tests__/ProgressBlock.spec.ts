import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ProgressBlock from '../ProgressBlock.vue'

describe('ProgressBlock', () => {
  it('renders with basic config', () => {
    const wrapper = mount(ProgressBlock, {
      props: {
        config: {
          value: 50,
        },
      },
    })

    expect(wrapper.html()).toContain('progress-block')
    expect(wrapper.text()).toContain('50%')
  })

  it('renders with label', () => {
    const wrapper = mount(ProgressBlock, {
      props: {
        config: {
          value: 75,
          label: 'Loading...',
        },
      },
    })

    expect(wrapper.text()).toContain('Loading...')
    expect(wrapper.text()).toContain('75%')
  })

  it('clamps value to 0-100 range', () => {
    const wrapperOver = mount(ProgressBlock, {
      props: {
        config: {
          value: 150,
        },
      },
    })
    expect(wrapperOver.text()).toContain('100%')

    const wrapperUnder = mount(ProgressBlock, {
      props: {
        config: {
          value: -10,
        },
      },
    })
    expect(wrapperUnder.text()).toContain('0%')
  })

  it('applies auto color based on value', () => {
    const wrapperHigh = mount(ProgressBlock, {
      props: {
        config: {
          value: 80,
          color: 'auto',
        },
      },
    })
    // High value should be green
    expect(wrapperHigh.find('.progress-bar').attributes('style')).toContain('#10b981')

    const wrapperMid = mount(ProgressBlock, {
      props: {
        config: {
          value: 50,
          color: 'auto',
        },
      },
    })
    // Mid value should be amber
    expect(wrapperMid.find('.progress-bar').attributes('style')).toContain('#f59e0b')

    const wrapperLow = mount(ProgressBlock, {
      props: {
        config: {
          value: 20,
          color: 'auto',
        },
      },
    })
    // Low value should be red
    expect(wrapperLow.find('.progress-bar').attributes('style')).toContain('#ef4444')
  })

  it('applies custom color', () => {
    const wrapper = mount(ProgressBlock, {
      props: {
        config: {
          value: 50,
          color: '#8b5cf6',
        },
      },
    })

    expect(wrapper.find('.progress-bar').attributes('style')).toContain('#8b5cf6')
  })

  it('applies size classes', () => {
    const wrapperSm = mount(ProgressBlock, {
      props: {
        config: {
          value: 50,
          size: 'sm',
        },
      },
    })
    expect(wrapperSm.find('.progress-block').classes()).toContain('progress-sm')

    const wrapperLg = mount(ProgressBlock, {
      props: {
        config: {
          value: 50,
          size: 'lg',
        },
      },
    })
    expect(wrapperLg.find('.progress-block').classes()).toContain('progress-lg')
  })

  it('sets correct progress bar width', () => {
    const wrapper = mount(ProgressBlock, {
      props: {
        config: {
          value: 67,
        },
      },
    })

    expect(wrapper.find('.progress-bar').attributes('style')).toContain('width: 67%')
  })
})
