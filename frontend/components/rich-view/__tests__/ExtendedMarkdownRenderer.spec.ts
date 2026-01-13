import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ExtendedMarkdownRenderer from '../ExtendedMarkdownRenderer.vue'

describe('ExtendedMarkdownRenderer', () => {
  it('renders basic markdown content', () => {
    const wrapper = mount(ExtendedMarkdownRenderer, {
      props: {
        content: '# Hello World\n\nThis is a paragraph.',
      },
    })

    const html = wrapper.html()
    expect(html).toContain('<h1>')
    expect(html).toContain('Hello World')
    expect(html).toContain('<p>')
  })

  it('renders markdown tables', () => {
    const wrapper = mount(ExtendedMarkdownRenderer, {
      props: {
        content: '| Name | Value |\n|------|-------|\n| A | 100 |',
      },
    })

    const html = wrapper.html()
    expect(html).toContain('<table>')
    expect(html).toContain('<th>')
    expect(html).toContain('Name')
  })

  it('renders code blocks with language', () => {
    const wrapper = mount(ExtendedMarkdownRenderer, {
      props: {
        content: '```javascript\nconst x = 1;\n```',
      },
    })

    const html = wrapper.html()
    expect(html).toContain('<pre>')
    expect(html).toContain('<code')
    expect(html).toContain('const x = 1;')
  })

  it('renders empty content without error', () => {
    const wrapper = mount(ExtendedMarkdownRenderer, {
      props: {
        content: '',
      },
    })

    expect(wrapper.html()).toContain('extended-markdown')
  })

  it('renders mixed content with chart blocks', () => {
    const wrapper = mount(ExtendedMarkdownRenderer, {
      props: {
        content: `# Report

Some text before chart.

\`\`\`chart
{"type": "bar", "labels": ["A", "B"], "datasets": [{"data": [1, 2]}]}
\`\`\`

Some text after chart.`,
      },
      global: {
        stubs: {
          ChartBlock: true,
          ProgressBlock: true,
        },
      },
    })

    const html = wrapper.html()
    expect(html).toContain('Report')
    expect(html).toContain('Some text before chart')
    expect(html).toContain('chart-block-stub')
    expect(html).toContain('Some text after chart')
  })

  it('renders progress blocks', () => {
    const wrapper = mount(ExtendedMarkdownRenderer, {
      props: {
        content: `\`\`\`progress
{"value": 75, "label": "Loading"}
\`\`\``,
      },
      global: {
        stubs: {
          ChartBlock: true,
          ProgressBlock: true,
        },
      },
    })

    expect(wrapper.html()).toContain('progress-block-stub')
  })

  it('handles invalid JSON in extended blocks gracefully', () => {
    const wrapper = mount(ExtendedMarkdownRenderer, {
      props: {
        content: '```chart\ninvalid json\n```',
      },
      global: {
        stubs: {
          ChartBlock: true,
          ProgressBlock: true,
        },
      },
    })

    const html = wrapper.html()
    // Should render as regular code block when JSON is invalid
    expect(html).toContain('<pre>')
    expect(html).toContain('invalid json')
  })
})
