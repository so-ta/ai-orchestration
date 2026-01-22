import { nextTick } from 'vue'

/**
 * Composable for chat auto-scroll functionality
 */
export function useChatScroll() {
  const chatMessagesRef = ref<HTMLElement | null>(null)
  const isUserScrolledUp = ref(false)

  /**
   * Check if user is near bottom (within threshold)
   */
  function isNearBottom(threshold = 100): boolean {
    const el = chatMessagesRef.value
    if (!el) return true
    return el.scrollHeight - el.scrollTop - el.clientHeight < threshold
  }

  /**
   * Scroll to bottom
   */
  function scrollToBottom(smooth = true) {
    const el = chatMessagesRef.value
    if (!el) return
    el.scrollTo({
      top: el.scrollHeight,
      behavior: smooth ? 'smooth' : 'instant',
    })
  }

  /**
   * Handle scroll event to track user position
   */
  function handleChatScroll() {
    isUserScrolledUp.value = !isNearBottom()
  }

  /**
   * Auto-scroll when content changes (if user hasn't scrolled up)
   */
  function autoScrollIfNeeded(instant = false) {
    if (!isUserScrolledUp.value) {
      nextTick(() => scrollToBottom(!instant))
    }
  }

  /**
   * Reset scroll state (resume auto-scroll)
   */
  function resetScrollState() {
    isUserScrolledUp.value = false
  }

  return {
    chatMessagesRef,
    isUserScrolledUp: readonly(isUserScrolledUp),
    isNearBottom,
    scrollToBottom,
    handleChatScroll,
    autoScrollIfNeeded,
    resetScrollState,
  }
}
