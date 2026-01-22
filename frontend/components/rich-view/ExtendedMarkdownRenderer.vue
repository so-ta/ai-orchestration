<script setup lang="ts">
import { marked } from 'marked'
import type { ChartConfig, ProgressConfig } from '~/types/rich-view'

const props = defineProps<{
  content: string
}>()

interface ParsedBlock {
  type: 'html' | 'chart' | 'progress'
  content: string
  config?: ChartConfig | ProgressConfig
}

// Parse markdown and extract extended code blocks
const parsedBlocks = computed<ParsedBlock[]>(() => {
  if (!props.content) return []

  const blocks: ParsedBlock[] = []
  const codeBlockRegex = /```(chart|progress)\n([\s\S]*?)```/g

  let lastIndex = 0
  let match

  while ((match = codeBlockRegex.exec(props.content)) !== null) {
    // Add HTML content before this code block
    if (match.index > lastIndex) {
      const htmlContent = props.content.slice(lastIndex, match.index)
      if (htmlContent.trim()) {
        blocks.push({
          type: 'html',
          content: marked.parse(htmlContent, { async: false }) as string,
        })
      }
    }

    // Parse the extended code block
    const blockType = match[1] as 'chart' | 'progress'
    const configStr = match[2].trim()

    try {
      const config = JSON.parse(configStr)
      blocks.push({
        type: blockType,
        content: configStr,
        config,
      })
    } catch {
      // If JSON parsing fails, render as regular code block
      blocks.push({
        type: 'html',
        content: `<pre><code class="language-${blockType}">${escapeHtml(configStr)}</code></pre>`,
      })
    }

    lastIndex = match.index + match[0].length
  }

  // Add remaining content after last code block
  if (lastIndex < props.content.length) {
    const htmlContent = props.content.slice(lastIndex)
    if (htmlContent.trim()) {
      blocks.push({
        type: 'html',
        content: marked.parse(htmlContent, { async: false }) as string,
      })
    }
  }

  return blocks
})

function escapeHtml(str: string): string {
  return str
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;')
}
</script>

<template>
  <div class="extended-markdown">
    <template v-for="(block, index) in parsedBlocks" :key="index">
      <!-- Regular HTML content -->
      <!-- eslint-disable vue/no-v-html -- sanitize済みMarkdownのレンダリング -->
      <div
        v-if="block.type === 'html'"
        class="markdown-content"
        v-html="block.content"
      />
      <!-- eslint-enable vue/no-v-html -->

      <!-- Chart block -->
      <ChartBlock
        v-else-if="block.type === 'chart' && block.config"
        :config="block.config as ChartConfig"
      />

      <!-- Progress block -->
      <ProgressBlock
        v-else-if="block.type === 'progress' && block.config"
        :config="block.config as ProgressConfig"
      />
    </template>
  </div>
</template>

<style scoped>
.extended-markdown {
  font-size: 0.875rem;
  line-height: 1.6;
  color: var(--color-text);
}

.markdown-content :deep(h1),
.markdown-content :deep(h2),
.markdown-content :deep(h3),
.markdown-content :deep(h4) {
  margin-top: 1.5em;
  margin-bottom: 0.5em;
  font-weight: 600;
  line-height: 1.25;
}

.markdown-content :deep(h1) {
  font-size: 1.5em;
  border-bottom: 1px solid var(--color-border);
  padding-bottom: 0.3em;
}

.markdown-content :deep(h2) {
  font-size: 1.25em;
  border-bottom: 1px solid var(--color-border);
  padding-bottom: 0.3em;
}

.markdown-content :deep(h3) {
  font-size: 1.1em;
}

.markdown-content :deep(p) {
  margin-top: 0;
  margin-bottom: 1em;
}

.markdown-content :deep(ul),
.markdown-content :deep(ol) {
  padding-left: 2em;
  margin-bottom: 1em;
}

.markdown-content :deep(li) {
  margin-bottom: 0.25em;
}

.markdown-content :deep(code) {
  padding: 0.2em 0.4em;
  font-size: 0.85em;
  background: rgba(0, 0, 0, 0.05);
  border-radius: 3px;
  font-family: 'SF Mono', Monaco, 'Cascadia Code', monospace;
}

.markdown-content :deep(pre) {
  padding: 1em;
  overflow-x: auto;
  background: #1e1e1e;
  border-radius: 6px;
  margin-bottom: 1em;
}

.markdown-content :deep(pre code) {
  padding: 0;
  background: transparent;
  color: #d4d4d4;
  font-size: 0.8rem;
  line-height: 1.5;
}

.markdown-content :deep(table) {
  border-collapse: collapse;
  width: 100%;
  margin-bottom: 1em;
}

.markdown-content :deep(th),
.markdown-content :deep(td) {
  border: 1px solid var(--color-border);
  padding: 0.5em 0.75em;
  text-align: left;
}

.markdown-content :deep(th) {
  background: var(--color-surface);
  font-weight: 600;
}

.markdown-content :deep(tr:nth-child(even)) {
  background: rgba(0, 0, 0, 0.02);
}

.markdown-content :deep(blockquote) {
  margin: 0 0 1em;
  padding: 0.5em 1em;
  border-left: 4px solid var(--color-primary);
  background: rgba(0, 0, 0, 0.02);
  color: var(--color-text-secondary);
}

.markdown-content :deep(img) {
  max-width: 100%;
  height: auto;
  border-radius: 6px;
}

.markdown-content :deep(a) {
  color: var(--color-primary);
  text-decoration: none;
}

.markdown-content :deep(a:hover) {
  text-decoration: underline;
}

.markdown-content :deep(strong) {
  font-weight: 600;
}

.markdown-content :deep(hr) {
  border: none;
  border-top: 1px solid var(--color-border);
  margin: 1.5em 0;
}
</style>
