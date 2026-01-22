import { describe, it, expect } from 'vitest'
import type { BlockDefinition } from '~/types/api'
import {
  groupBlocksByCategory,
  groupBlocksBySubcategory,
  getSubcategoriesForCategory,
  searchBlocks,
  getBlockColor,
  categoryConfig,
  subcategoryConfig,
  subcategoryToCategory,
  blockColors,
  defaultBlockColor,
} from './useBlocks'

// Mock block definitions for testing
function createMockBlock(overrides: Partial<BlockDefinition>): BlockDefinition {
  return {
    id: overrides.id || 'test-id',
    tenant_id: overrides.tenant_id ?? undefined,
    slug: overrides.slug || 'test-block',
    name: overrides.name || 'Test Block',
    description: overrides.description || 'A test block',
    category: overrides.category || 'flow',
    subcategory: overrides.subcategory || undefined,
    icon: overrides.icon || 'test-icon',
    config_schema: overrides.config_schema || {},
    output_schema: overrides.output_schema || {},
    output_ports: overrides.output_ports || [],
    error_codes: overrides.error_codes || [],
    is_system: overrides.is_system ?? true,
    enabled: overrides.enabled ?? true,
    version: overrides.version || 1,
    ui_config: overrides.ui_config || {},
    created_at: overrides.created_at || '2024-01-01T00:00:00Z',
    updated_at: overrides.updated_at || '2024-01-01T00:00:00Z',
  }
}

describe('useBlocks - categoryConfig', () => {
  it('should have all required categories', () => {
    expect(categoryConfig).toHaveProperty('flow')
    expect(categoryConfig).toHaveProperty('ai')
    expect(categoryConfig).toHaveProperty('apps')
    expect(categoryConfig).toHaveProperty('custom')
  })

  it('should have correct order for categories', () => {
    expect(categoryConfig.flow.order).toBe(1)
    expect(categoryConfig.ai.order).toBe(2)
    expect(categoryConfig.apps.order).toBe(3)
    expect(categoryConfig.custom.order).toBe(4)
  })

  it('should have nameKey for i18n', () => {
    Object.values(categoryConfig).forEach(config => {
      expect(config.nameKey).toMatch(/^editor\.categories\.\w+$/)
    })
  })
})

describe('useBlocks - subcategoryConfig', () => {
  it('should have all AI subcategories', () => {
    expect(subcategoryConfig).toHaveProperty('chat')
    expect(subcategoryConfig).toHaveProperty('rag')
    expect(subcategoryConfig).toHaveProperty('routing')
  })

  it('should have all flow subcategories', () => {
    expect(subcategoryConfig).toHaveProperty('trigger')
    expect(subcategoryConfig).toHaveProperty('branching')
    expect(subcategoryConfig).toHaveProperty('data')
    expect(subcategoryConfig).toHaveProperty('control')
    expect(subcategoryConfig).toHaveProperty('utility')
  })

  it('should have all apps subcategories', () => {
    expect(subcategoryConfig).toHaveProperty('slack')
    expect(subcategoryConfig).toHaveProperty('discord')
    expect(subcategoryConfig).toHaveProperty('notion')
    expect(subcategoryConfig).toHaveProperty('github')
    expect(subcategoryConfig).toHaveProperty('google')
    expect(subcategoryConfig).toHaveProperty('linear')
    expect(subcategoryConfig).toHaveProperty('email')
    expect(subcategoryConfig).toHaveProperty('web')
  })
})

describe('useBlocks - subcategoryToCategory', () => {
  it('should map AI subcategories correctly', () => {
    expect(subcategoryToCategory.chat).toBe('ai')
    expect(subcategoryToCategory.rag).toBe('ai')
    expect(subcategoryToCategory.routing).toBe('ai')
  })

  it('should map flow subcategories correctly', () => {
    expect(subcategoryToCategory.trigger).toBe('flow')
    expect(subcategoryToCategory.branching).toBe('flow')
    expect(subcategoryToCategory.data).toBe('flow')
    expect(subcategoryToCategory.control).toBe('flow')
    expect(subcategoryToCategory.utility).toBe('flow')
  })

  it('should map apps subcategories correctly', () => {
    expect(subcategoryToCategory.slack).toBe('apps')
    expect(subcategoryToCategory.discord).toBe('apps')
    expect(subcategoryToCategory.notion).toBe('apps')
    expect(subcategoryToCategory.github).toBe('apps')
    expect(subcategoryToCategory.google).toBe('apps')
    expect(subcategoryToCategory.linear).toBe('apps')
    expect(subcategoryToCategory.email).toBe('apps')
    expect(subcategoryToCategory.web).toBe('apps')
  })
})

describe('useBlocks - getBlockColor', () => {
  it('should return correct color for known blocks', () => {
    expect(getBlockColor('llm')).toBe('#3b82f6')
    expect(getBlockColor('condition')).toBe('#f59e0b')
    expect(getBlockColor('start')).toBe('#10b981')
    expect(getBlockColor('tool')).toBe('#22c55e')
  })

  it('should return default color for unknown blocks', () => {
    expect(getBlockColor('unknown-block')).toBe(defaultBlockColor)
    expect(getBlockColor('')).toBe(defaultBlockColor)
  })
})

describe('useBlocks - groupBlocksByCategory', () => {
  it('should group blocks by category', () => {
    const blocks = [
      createMockBlock({ slug: 'llm', name: 'LLM', category: 'ai' }),
      createMockBlock({ slug: 'condition', name: 'Condition', category: 'flow' }),
      createMockBlock({ slug: 'slack', name: 'Slack', category: 'apps' }),
      createMockBlock({ slug: 'custom1', name: 'Custom', category: 'custom' }),
    ]

    const grouped = groupBlocksByCategory(blocks)

    expect(grouped.ai).toHaveLength(1)
    expect(grouped.ai[0].slug).toBe('llm')
    expect(grouped.flow).toHaveLength(1)
    expect(grouped.flow[0].slug).toBe('condition')
    expect(grouped.apps).toHaveLength(1)
    expect(grouped.apps[0].slug).toBe('slack')
    expect(grouped.custom).toHaveLength(1)
    expect(grouped.custom[0].slug).toBe('custom1')
  })

  it('should return empty arrays for categories with no blocks', () => {
    const blocks = [
      createMockBlock({ slug: 'llm', name: 'LLM', category: 'ai' }),
    ]

    const grouped = groupBlocksByCategory(blocks)

    expect(grouped.ai).toHaveLength(1)
    expect(grouped.flow).toHaveLength(0)
    expect(grouped.apps).toHaveLength(0)
    expect(grouped.custom).toHaveLength(0)
  })

  it('should handle empty input', () => {
    const grouped = groupBlocksByCategory([])

    expect(grouped.ai).toHaveLength(0)
    expect(grouped.flow).toHaveLength(0)
    expect(grouped.apps).toHaveLength(0)
    expect(grouped.custom).toHaveLength(0)
  })

  it('should sort blocks by subcategory order then name', () => {
    const blocks = [
      createMockBlock({ slug: 'b', name: 'B Block', category: 'ai', subcategory: 'chat' }),
      createMockBlock({ slug: 'a', name: 'A Block', category: 'ai', subcategory: 'chat' }),
      createMockBlock({ slug: 'c', name: 'C Block', category: 'ai', subcategory: 'rag' }),
    ]

    const grouped = groupBlocksByCategory(blocks)

    // chat (order 1) should come before rag (order 2)
    // Within chat, A should come before B
    expect(grouped.ai[0].name).toBe('A Block')
    expect(grouped.ai[1].name).toBe('B Block')
    expect(grouped.ai[2].name).toBe('C Block')
  })
})

describe('useBlocks - groupBlocksBySubcategory', () => {
  it('should group blocks by subcategory within a category', () => {
    const blocks = [
      createMockBlock({ slug: 'chat1', name: 'Chat 1', category: 'ai', subcategory: 'chat' }),
      createMockBlock({ slug: 'chat2', name: 'Chat 2', category: 'ai', subcategory: 'chat' }),
      createMockBlock({ slug: 'rag1', name: 'RAG 1', category: 'ai', subcategory: 'rag' }),
      createMockBlock({ slug: 'flow1', name: 'Flow 1', category: 'flow', subcategory: 'branching' }),
    ]

    const grouped = groupBlocksBySubcategory(blocks, 'ai')

    expect(grouped.chat).toHaveLength(2)
    expect(grouped.rag).toHaveLength(1)
    expect(grouped.branching).toBeUndefined() // Different category
  })

  it('should group all blocks when category is null', () => {
    const blocks = [
      createMockBlock({ slug: 'chat1', name: 'Chat 1', category: 'ai', subcategory: 'chat' }),
      createMockBlock({ slug: 'flow1', name: 'Flow 1', category: 'flow', subcategory: 'branching' }),
    ]

    const grouped = groupBlocksBySubcategory(blocks, null)

    expect(grouped.chat).toHaveLength(1)
    expect(grouped.branching).toHaveLength(1)
  })

  it('should handle trigger blocks specially', () => {
    const blocks = [
      createMockBlock({ slug: 'manual_trigger', name: 'Manual Trigger', category: 'flow', subcategory: 'control' }),
      createMockBlock({ slug: 'schedule_trigger', name: 'Schedule Trigger', category: 'flow', subcategory: 'control' }),
      createMockBlock({ slug: 'webhook_trigger', name: 'Webhook Trigger', category: 'flow', subcategory: 'control' }),
    ]

    const grouped = groupBlocksBySubcategory(blocks, 'flow')

    // All trigger blocks should be in 'trigger' subcategory
    expect(grouped.trigger).toHaveLength(3)
    expect(grouped.control).toBeUndefined()
  })

  it('should sort blocks within subcategory by name', () => {
    const blocks = [
      createMockBlock({ slug: 'c', name: 'C Block', category: 'ai', subcategory: 'chat' }),
      createMockBlock({ slug: 'a', name: 'A Block', category: 'ai', subcategory: 'chat' }),
      createMockBlock({ slug: 'b', name: 'B Block', category: 'ai', subcategory: 'chat' }),
    ]

    const grouped = groupBlocksBySubcategory(blocks, 'ai')

    expect(grouped.chat[0].name).toBe('A Block')
    expect(grouped.chat[1].name).toBe('B Block')
    expect(grouped.chat[2].name).toBe('C Block')
  })

  it('should use other for blocks without subcategory', () => {
    const blocks = [
      createMockBlock({ slug: 'no-sub', name: 'No Sub', category: 'ai', subcategory: undefined }),
    ]

    const grouped = groupBlocksBySubcategory(blocks, 'ai')

    expect(grouped.other).toHaveLength(1)
  })
})

describe('useBlocks - getSubcategoriesForCategory', () => {
  it('should return AI subcategories in order', () => {
    const subcategories = getSubcategoriesForCategory('ai')

    expect(subcategories).toContain('chat')
    expect(subcategories).toContain('rag')
    expect(subcategories).toContain('routing')
    // Verify order
    expect(subcategories.indexOf('chat')).toBeLessThan(subcategories.indexOf('rag'))
  })

  it('should return flow subcategories', () => {
    const subcategories = getSubcategoriesForCategory('flow')

    expect(subcategories).toContain('trigger')
    expect(subcategories).toContain('branching')
    expect(subcategories).toContain('data')
    expect(subcategories).toContain('control')
    expect(subcategories).toContain('utility')
    // Should have 5 flow subcategories
    expect(subcategories).toHaveLength(5)
  })

  it('should return apps subcategories', () => {
    const subcategories = getSubcategoriesForCategory('apps')

    expect(subcategories).toContain('slack')
    expect(subcategories).toContain('discord')
    expect(subcategories).toContain('notion')
    expect(subcategories).toContain('github')
  })

  it('should return empty array for custom category', () => {
    const subcategories = getSubcategoriesForCategory('custom')

    expect(subcategories).toHaveLength(0)
  })
})

describe('useBlocks - searchBlocks', () => {
  const testBlocks = [
    createMockBlock({ slug: 'llm', name: 'LLM Chat', description: 'AI language model' }),
    createMockBlock({ slug: 'condition', name: 'Condition', description: 'Branch logic' }),
    createMockBlock({ slug: 'slack-message', name: 'Slack Message', description: 'Send messages to Slack' }),
  ]

  it('should return all blocks for empty query', () => {
    const results = searchBlocks(testBlocks, '')
    expect(results).toHaveLength(3)
  })

  it('should return all blocks for whitespace-only query', () => {
    const results = searchBlocks(testBlocks, '   ')
    expect(results).toHaveLength(3)
  })

  it('should search by name', () => {
    const results = searchBlocks(testBlocks, 'Chat')
    // Smart search with aliases: 'chat' also matches 'message' (slack-message)
    expect(results.length).toBeGreaterThanOrEqual(1)
    // LLM should be first result due to direct name match
    expect(results[0].slug).toBe('llm')
  })

  it('should search by slug', () => {
    const results = searchBlocks(testBlocks, 'slack-message')
    expect(results).toHaveLength(1)
    expect(results[0].slug).toBe('slack-message')
  })

  it('should search by description', () => {
    const results = searchBlocks(testBlocks, 'language')
    expect(results).toHaveLength(1)
    expect(results[0].slug).toBe('llm')
  })

  it('should be case-insensitive', () => {
    const results = searchBlocks(testBlocks, 'SLACK')
    expect(results).toHaveLength(1)
    expect(results[0].slug).toBe('slack-message')
  })

  it('should match partial strings', () => {
    const results = searchBlocks(testBlocks, 'tion')
    expect(results).toHaveLength(1)
    expect(results[0].slug).toBe('condition')
  })

  it('should return empty array when no matches', () => {
    const results = searchBlocks(testBlocks, 'xyz123')
    expect(results).toHaveLength(0)
  })

  it('should handle blocks without description', () => {
    const blocksNoDesc = [
      createMockBlock({ slug: 'test', name: 'Test', description: undefined }),
    ]
    const results = searchBlocks(blocksNoDesc, 'test')
    expect(results).toHaveLength(1)
  })
})

describe('useBlocks - blockColors', () => {
  it('should have colors for AI blocks', () => {
    expect(blockColors.llm).toBeDefined()
    expect(blockColors.router).toBeDefined()
  })

  it('should have colors for logic blocks', () => {
    expect(blockColors.condition).toBeDefined()
    expect(blockColors.switch).toBeDefined()
    expect(blockColors.loop).toBeDefined()
    expect(blockColors.map).toBeDefined()
    expect(blockColors.join).toBeDefined()
  })

  it('should have colors for data blocks', () => {
    expect(blockColors.filter).toBeDefined()
    expect(blockColors.split).toBeDefined()
    expect(blockColors.aggregate).toBeDefined()
    expect(blockColors.transform).toBeDefined()
  })

  it('should have colors for trigger blocks', () => {
    expect(blockColors.start).toBeDefined()
    expect(blockColors.manual_trigger).toBeDefined()
    expect(blockColors.schedule_trigger).toBeDefined()
    expect(blockColors.webhook_trigger).toBeDefined()
  })
})
