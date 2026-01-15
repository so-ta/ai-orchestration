/**
 * Composable for managing block preferences (recently used, favorites)
 * Data is stored in localStorage per tenant
 */

const STORAGE_KEY_PREFIX = 'block-preferences'
const MAX_RECENT_BLOCKS = 10

interface BlockPreferences {
  recentBlockSlugs: string[]
  favoriteBlockSlugs: string[]
}

function getStorageKey(tenantId: string): string {
  return `${STORAGE_KEY_PREFIX}-${tenantId}`
}

function loadPreferences(tenantId: string): BlockPreferences {
  if (typeof window === 'undefined') {
    return { recentBlockSlugs: [], favoriteBlockSlugs: [] }
  }

  try {
    const stored = localStorage.getItem(getStorageKey(tenantId))
    if (stored) {
      const parsed = JSON.parse(stored)
      return {
        recentBlockSlugs: Array.isArray(parsed.recentBlockSlugs) ? parsed.recentBlockSlugs : [],
        favoriteBlockSlugs: Array.isArray(parsed.favoriteBlockSlugs) ? parsed.favoriteBlockSlugs : [],
      }
    }
  } catch {
    // Invalid JSON, return default
  }

  return { recentBlockSlugs: [], favoriteBlockSlugs: [] }
}

function savePreferences(tenantId: string, prefs: BlockPreferences): void {
  if (typeof window === 'undefined') return

  try {
    localStorage.setItem(getStorageKey(tenantId), JSON.stringify(prefs))
  } catch {
    // localStorage full or disabled
  }
}

export function useBlockPreferences() {
  // Get current tenant ID from auth
  const auth = useAuth()
  const tenantId = computed(() => auth.user.value?.tenantId || 'default')

  // Reactive preferences
  const preferences = ref<BlockPreferences>({ recentBlockSlugs: [], favoriteBlockSlugs: [] })

  // Load preferences when tenant changes
  watch(tenantId, (newTenantId) => {
    preferences.value = loadPreferences(newTenantId)
  }, { immediate: true })

  // Track block usage (add to recent)
  function trackUsage(blockSlug: string): void {
    const prefs = preferences.value

    // Remove if already exists
    const index = prefs.recentBlockSlugs.indexOf(blockSlug)
    if (index !== -1) {
      prefs.recentBlockSlugs.splice(index, 1)
    }

    // Add to front
    prefs.recentBlockSlugs.unshift(blockSlug)

    // Limit size
    if (prefs.recentBlockSlugs.length > MAX_RECENT_BLOCKS) {
      prefs.recentBlockSlugs = prefs.recentBlockSlugs.slice(0, MAX_RECENT_BLOCKS)
    }

    savePreferences(tenantId.value, prefs)
  }

  // Toggle favorite status
  function toggleFavorite(blockSlug: string): void {
    const prefs = preferences.value
    const index = prefs.favoriteBlockSlugs.indexOf(blockSlug)

    if (index !== -1) {
      prefs.favoriteBlockSlugs.splice(index, 1)
    } else {
      prefs.favoriteBlockSlugs.push(blockSlug)
    }

    savePreferences(tenantId.value, prefs)
  }

  // Check if a block is favorited
  function isFavorite(blockSlug: string): boolean {
    return preferences.value.favoriteBlockSlugs.includes(blockSlug)
  }

  // Get recent block slugs
  const recentBlockSlugs = computed(() => preferences.value.recentBlockSlugs)

  // Get favorite block slugs
  const favoriteBlockSlugs = computed(() => preferences.value.favoriteBlockSlugs)

  return {
    recentBlockSlugs,
    favoriteBlockSlugs,
    trackUsage,
    toggleFavorite,
    isFavorite,
  }
}
