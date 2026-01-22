/**
 * useVirtualization composable for large workflow optimization
 *
 * Provides viewport-based rendering optimization for DAG editors
 * with many nodes. Only nodes within the visible viewport (plus a buffer)
 * are rendered, significantly improving performance for large workflows.
 */

import type { Node, Edge } from '@vue-flow/core'

// Buffer size in pixels around the viewport
const VIEWPORT_BUFFER = 200

export interface Viewport {
  x: number
  y: number
  zoom: number
}

export interface ViewportBounds {
  minX: number
  maxX: number
  minY: number
  maxY: number
}

export interface UseVirtualizationOptions {
  viewport: Ref<Viewport>
  containerWidth: Ref<number>
  containerHeight: Ref<number>
  enabled?: Ref<boolean>
  buffer?: number
}

export interface UseVirtualizationReturn {
  visibleBounds: ComputedRef<ViewportBounds>
  isNodeVisible: (node: Node) => boolean
  isEdgeVisible: (edge: Edge, nodes: Node[]) => boolean
  filterVisibleNodes: <T extends Node>(nodes: T[]) => T[]
  filterVisibleEdges: <T extends Edge>(edges: T[], nodes: Node[]) => T[]
}

/**
 * Composable for viewport-based virtualization
 */
export function useVirtualization(options: UseVirtualizationOptions): UseVirtualizationReturn {
  const {
    viewport,
    containerWidth,
    containerHeight,
    enabled = ref(true),
    buffer = VIEWPORT_BUFFER,
  } = options

  /**
   * Calculate the visible bounds in world coordinates
   */
  const visibleBounds = computed<ViewportBounds>(() => {
    const { x, y, zoom } = viewport.value
    const width = containerWidth.value
    const height = containerHeight.value

    // Convert viewport coordinates to world coordinates
    // Viewport x,y represents the top-left corner of the visible area in world space
    // (negative because pan is inverted)
    const worldX = -x / zoom
    const worldY = -y / zoom
    const worldWidth = width / zoom
    const worldHeight = height / zoom

    return {
      minX: worldX - buffer,
      maxX: worldX + worldWidth + buffer,
      minY: worldY - buffer,
      maxY: worldY + worldHeight + buffer,
    }
  })

  /**
   * Check if a node is within the visible bounds
   */
  function isNodeVisible(node: Node): boolean {
    if (!enabled.value) return true

    const { minX, maxX, minY, maxY } = visibleBounds.value

    // Node dimensions (default if not specified)
    // width/height can be number, string, or function - we handle number only
    const nodeWidth = typeof node.width === 'number' ? node.width : 200
    const nodeHeight = typeof node.height === 'number' ? node.height : 80

    const nodeMinX = node.position.x
    const nodeMaxX = node.position.x + nodeWidth
    const nodeMinY = node.position.y
    const nodeMaxY = node.position.y + nodeHeight

    // Check for intersection with visible bounds
    return !(
      nodeMaxX < minX ||
      nodeMinX > maxX ||
      nodeMaxY < minY ||
      nodeMinY > maxY
    )
  }

  /**
   * Check if an edge should be rendered
   * An edge is visible if either its source or target node is visible
   */
  function isEdgeVisible(edge: Edge, nodes: Node[]): boolean {
    if (!enabled.value) return true

    const sourceNode = nodes.find(n => n.id === edge.source)
    const targetNode = nodes.find(n => n.id === edge.target)

    // If we can't find the nodes, show the edge anyway
    if (!sourceNode || !targetNode) return true

    // Edge is visible if either endpoint is visible
    return isNodeVisible(sourceNode) || isNodeVisible(targetNode)
  }

  /**
   * Filter an array of nodes to only those that are visible
   */
  function filterVisibleNodes<T extends Node>(nodes: T[]): T[] {
    if (!enabled.value) return nodes
    return nodes.filter(isNodeVisible)
  }

  /**
   * Filter an array of edges to only those that are visible
   */
  function filterVisibleEdges<T extends Edge>(edges: T[], nodes: Node[]): T[] {
    if (!enabled.value) return edges
    return edges.filter(edge => isEdgeVisible(edge, nodes))
  }

  return {
    visibleBounds,
    isNodeVisible,
    isEdgeVisible,
    filterVisibleNodes,
    filterVisibleEdges,
  }
}

// ============================================================================
// Node Cache for Performance
// ============================================================================

/**
 * Simple cache for computed node data
 * Invalidates when step data changes (based on updated_at)
 */
export class NodeCache<T> {
  private cache = new Map<string, { data: T; timestamp: string }>()

  /**
   * Get a cached value or compute and store it
   */
  getOrCompute(key: string, timestamp: string, compute: () => T): T {
    const cached = this.cache.get(key)

    if (cached && cached.timestamp === timestamp) {
      return cached.data
    }

    const data = compute()
    this.cache.set(key, { data, timestamp })
    return data
  }

  /**
   * Clear stale entries not in the current keys set
   */
  cleanup(currentKeys: Set<string>): void {
    for (const key of this.cache.keys()) {
      if (!currentKeys.has(key)) {
        this.cache.delete(key)
      }
    }
  }

  /**
   * Clear all cached data
   */
  clear(): void {
    this.cache.clear()
  }

  /**
   * Get cache size for debugging
   */
  get size(): number {
    return this.cache.size
  }
}

// ============================================================================
// Batch Update Utilities
// ============================================================================

/**
 * Debounce function for batch updates
 */
export function useDebouncedUpdate<T>(
  callback: (items: T[]) => void,
  delay: number = 16 // ~60fps
) {
  let pendingItems: T[] = []
  let timeoutId: ReturnType<typeof setTimeout> | null = null

  function add(item: T) {
    pendingItems.push(item)

    if (timeoutId === null) {
      timeoutId = setTimeout(() => {
        callback([...pendingItems])
        pendingItems = []
        timeoutId = null
      }, delay)
    }
  }

  function flush() {
    if (timeoutId !== null) {
      clearTimeout(timeoutId)
      timeoutId = null
    }
    if (pendingItems.length > 0) {
      callback([...pendingItems])
      pendingItems = []
    }
  }

  function clear() {
    if (timeoutId !== null) {
      clearTimeout(timeoutId)
      timeoutId = null
    }
    pendingItems = []
  }

  return { add, flush, clear }
}

// ============================================================================
// Performance Monitoring
// ============================================================================

/**
 * Simple performance monitor for development
 */
export function usePerformanceMonitor(_name: string) {
  const frameCount = ref(0)
  const fps = ref(0)
  let lastTime = performance.now()
  let animationId: number | null = null

  function startMonitoring() {
    const measure = () => {
      frameCount.value++
      const currentTime = performance.now()

      if (currentTime - lastTime >= 1000) {
        fps.value = frameCount.value
        frameCount.value = 0
        lastTime = currentTime
      }

      animationId = requestAnimationFrame(measure)
    }

    animationId = requestAnimationFrame(measure)
  }

  function stopMonitoring() {
    if (animationId !== null) {
      cancelAnimationFrame(animationId)
      animationId = null
    }
  }

  if (import.meta.dev) {
    onMounted(startMonitoring)
    onUnmounted(stopMonitoring)
  }

  return {
    fps: readonly(fps),
    isMonitoring: computed(() => animationId !== null),
  }
}
