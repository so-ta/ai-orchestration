type ProposalStatus = 'pending' | 'applied' | 'discarded'

/**
 * Composable for persisting proposal statuses to localStorage
 */
export function useProposalStatuses(workflowId: string) {
  const proposalStatuses = ref<Map<string, ProposalStatus>>(new Map())
  const storageKey = `copilot-proposal-statuses-${workflowId}`

  /**
   * Load proposal statuses from localStorage
   */
  function load() {
    try {
      const stored = localStorage.getItem(storageKey)
      if (stored) {
        const parsed = JSON.parse(stored) as Record<string, ProposalStatus>
        proposalStatuses.value = new Map(Object.entries(parsed))
      }
    } catch (e) {
      console.warn('Failed to load proposal statuses:', e)
    }
  }

  /**
   * Save proposal statuses to localStorage
   */
  function save() {
    try {
      const obj = Object.fromEntries(proposalStatuses.value)
      localStorage.setItem(storageKey, JSON.stringify(obj))
    } catch (e) {
      console.warn('Failed to save proposal statuses:', e)
    }
  }

  /**
   * Get status for a proposal
   */
  function getStatus(proposalId: string): ProposalStatus | undefined {
    return proposalStatuses.value.get(proposalId)
  }

  /**
   * Set status for a proposal
   */
  function setStatus(proposalId: string, status: ProposalStatus) {
    proposalStatuses.value.set(proposalId, status)
    save()
  }

  /**
   * Clear all statuses
   */
  function clear() {
    proposalStatuses.value.clear()
    localStorage.removeItem(storageKey)
  }

  return {
    proposalStatuses: readonly(proposalStatuses),
    load,
    save,
    getStatus,
    setStatus,
    clear,
  }
}
