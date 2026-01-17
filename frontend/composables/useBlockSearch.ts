/**
 * Block search composable
 *
 * 共通のブロック検索ロジックを提供する。
 * StepPaletteとBlockSearchModalで使用。
 */

import { ref, computed, type Ref, type ComputedRef } from 'vue'
import type { BlockDefinition, BlockCategory, BlockSubcategory } from '~/types/api'
import {
  searchBlocks,
  groupBlocksBySubcategory,
  groupBlocksByCategory,
  getSubcategoriesForCategory,
} from './useBlocks'

export interface UseBlockSearchOptions {
  /** 除外するブロックのslug */
  excludeSlugs?: string[]
}

export interface UseBlockSearchReturn {
  /** 検索クエリ */
  searchQuery: Ref<string>
  /** フィルタリングされたブロック */
  filteredBlocks: ComputedRef<BlockDefinition[]>
  /** 検索がアクティブかどうか */
  isSearchActive: ComputedRef<boolean>
  /** 検索をクリア */
  clearSearch: () => void
}

/**
 * ブロック検索のためのcomposable
 *
 * @param blocks - ブロック定義のRef
 * @param options - オプション（除外するslugなど）
 * @returns 検索状態と関数
 *
 * @example
 * const { searchQuery, filteredBlocks, isSearchActive, clearSearch } = useBlockSearch(blocks)
 */
export function useBlockSearch(
  blocks: Ref<BlockDefinition[]>,
  options: UseBlockSearchOptions = {}
): UseBlockSearchReturn {
  const { excludeSlugs = ['start'] } = options

  const searchQuery = ref('')

  const filteredBlocks = computed(() => {
    // 除外リストのブロックを除く
    let result = blocks.value.filter(b => !excludeSlugs.includes(b.slug))

    // 検索クエリがある場合はフィルタリング
    if (searchQuery.value.trim()) {
      result = searchBlocks(result, searchQuery.value)
    }

    return result
  })

  const isSearchActive = computed(() => searchQuery.value.trim().length > 0)

  function clearSearch() {
    searchQuery.value = ''
  }

  return {
    searchQuery,
    filteredBlocks,
    isSearchActive,
    clearSearch,
  }
}

export interface UseBlockSearchWithCategoryReturn extends UseBlockSearchReturn {
  /** アクティブなカテゴリ */
  activeCategory: Ref<BlockCategory>
  /** カテゴリ内のサブカテゴリでグループ化されたブロック */
  blocksBySubcategory: ComputedRef<Record<string, BlockDefinition[]>>
  /** アクティブなカテゴリ（または検索時は全て）のサブカテゴリリスト */
  activeSubcategories: ComputedRef<BlockSubcategory[]>
}

/**
 * カテゴリ付きブロック検索のためのcomposable
 *
 * StepPaletteで使用するカテゴリタブ対応版。
 *
 * @param blocks - ブロック定義のRef
 * @param options - オプション（除外するslugなど）
 * @returns 検索状態と関数（カテゴリ機能付き）
 */
export function useBlockSearchWithCategory(
  blocks: Ref<BlockDefinition[]>,
  options: UseBlockSearchOptions = {}
): UseBlockSearchWithCategoryReturn {
  const base = useBlockSearch(blocks, options)
  const activeCategory = ref<BlockCategory>('ai')

  const blocksBySubcategory = computed(() => {
    if (base.isSearchActive.value) {
      // 検索中は全カテゴリのブロックをサブカテゴリでグループ化
      return groupBlocksBySubcategory(base.filteredBlocks.value, null)
    }
    // 通常時はアクティブカテゴリのブロックをサブカテゴリでグループ化
    return groupBlocksBySubcategory(base.filteredBlocks.value, activeCategory.value)
  })

  const activeSubcategories = computed(() => {
    if (base.isSearchActive.value) {
      // 検索中はマッチするブロックのサブカテゴリを返す
      const subcats = new Set<BlockSubcategory>()
      for (const block of base.filteredBlocks.value) {
        if (block.subcategory) {
          subcats.add(block.subcategory)
        }
      }
      return Array.from(subcats)
    }
    // 通常時は現在のカテゴリに属するサブカテゴリを返す
    return getSubcategoriesForCategory(activeCategory.value)
  })

  return {
    ...base,
    activeCategory,
    blocksBySubcategory,
    activeSubcategories,
  }
}

export interface UseBlockSearchWithGroupingReturn extends UseBlockSearchReturn {
  /** カテゴリでグループ化されたブロック */
  groupedBlocks: ComputedRef<Record<BlockCategory, BlockDefinition[]>>
}

/**
 * カテゴリグループ化付きブロック検索のためのcomposable
 *
 * BlockSearchModalで使用するカテゴリグループ化版。
 *
 * @param blocks - ブロック定義のRef
 * @param options - オプション（除外するslugなど）
 * @returns 検索状態と関数（カテゴリグループ化機能付き）
 */
export function useBlockSearchWithGrouping(
  blocks: Ref<BlockDefinition[]>,
  options: UseBlockSearchOptions = {}
): UseBlockSearchWithGroupingReturn {
  const base = useBlockSearch(blocks, options)

  const groupedBlocks = computed(() => {
    return groupBlocksByCategory(base.filteredBlocks.value)
  })

  return {
    ...base,
    groupedBlocks,
  }
}
