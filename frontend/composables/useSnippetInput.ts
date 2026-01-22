/**
 * useSnippetInput - Slot-based template input
 *
 * Parses templates like "[頻度] に [情報] を [通知先] に送りたい"
 * into visual slots that users can fill in by clicking.
 */

export interface TemplateSlot {
  type: 'text' | 'slot'
  content: string // For text: the actual text, for slot: the label
  value?: string // For slot: the filled value (undefined if not filled)
}

interface UseSnippetInputOptions {
  initialTemplate?: string
}

export function useSnippetInput(options: UseSnippetInputOptions = {}) {
  // Raw template string (e.g., "[頻度] に [情報] を [通知先] に送りたい")
  const templateString = ref(options.initialTemplate ?? '')

  // Parsed template parts
  const templateParts = ref<TemplateSlot[]>([])

  // Current selected slot index (for slots only, not text parts)
  const selectedSlotIndex = ref(0)

  // Slot values (keyed by slot label)
  const slotValues = ref<Record<string, string>>({})

  /**
   * Parse template string into parts
   */
  function parseTemplate(template: string): TemplateSlot[] {
    if (!template) return []

    const parts: TemplateSlot[] = []
    const regex = /\[([^\]]+)\]/g
    let lastIndex = 0
    let match: RegExpExecArray | null

    while ((match = regex.exec(template)) !== null) {
      // Add text before this slot
      if (match.index > lastIndex) {
        parts.push({
          type: 'text',
          content: template.slice(lastIndex, match.index),
        })
      }

      // Add the slot
      parts.push({
        type: 'slot',
        content: match[1], // The label inside brackets
      })

      lastIndex = match.index + match[0].length
    }

    // Add remaining text after last slot
    if (lastIndex < template.length) {
      parts.push({
        type: 'text',
        content: template.slice(lastIndex),
      })
    }

    return parts
  }

  /**
   * Get only the slot parts (for iteration)
   */
  const slots = computed(() => {
    return templateParts.value.filter((p): p is TemplateSlot & { type: 'slot' } => p.type === 'slot')
  })

  /**
   * Current selected slot
   */
  const currentSlot = computed(() => {
    return slots.value[selectedSlotIndex.value] ?? null
  })

  /**
   * Current slot label (for showing examples)
   */
  const currentSlotLabel = computed(() => {
    return currentSlot.value?.content ?? null
  })

  /**
   * Check if all slots are filled
   */
  const allSlotsFilled = computed(() => {
    return slots.value.every(slot => slotValues.value[slot.content]?.trim())
  })

  /**
   * Check if there are any slots
   */
  const hasSlots = computed(() => slots.value.length > 0)

  /**
   * Get filled value for a slot label
   */
  function getSlotValue(label: string): string | undefined {
    return slotValues.value[label]
  }

  /**
   * Set template and reset state
   */
  function setTemplate(template: string) {
    templateString.value = template
    templateParts.value = parseTemplate(template)
    slotValues.value = {}
    selectedSlotIndex.value = 0
  }

  /**
   * Select a slot by its index in the slots array
   */
  function selectSlot(index: number) {
    if (index >= 0 && index < slots.value.length) {
      selectedSlotIndex.value = index
    }
  }

  /**
   * Select next slot
   */
  function selectNextSlot() {
    if (slots.value.length === 0) return
    const nextIndex = (selectedSlotIndex.value + 1) % slots.value.length
    selectSlot(nextIndex)
  }

  /**
   * Select previous slot
   */
  function selectPrevSlot() {
    if (slots.value.length === 0) return
    const prevIndex = selectedSlotIndex.value <= 0
      ? slots.value.length - 1
      : selectedSlotIndex.value - 1
    selectSlot(prevIndex)
  }

  /**
   * Fill current slot with a value and move to next unfilled slot
   */
  function fillCurrentSlot(value: string) {
    const slot = currentSlot.value
    if (!slot) return

    slotValues.value[slot.content] = value

    // Find next unfilled slot
    const currentIdx = selectedSlotIndex.value
    for (let i = 1; i <= slots.value.length; i++) {
      const nextIdx = (currentIdx + i) % slots.value.length
      const nextSlot = slots.value[nextIdx]
      if (!slotValues.value[nextSlot.content]?.trim()) {
        selectSlot(nextIdx)
        return
      }
    }
  }

  /**
   * Fill a specific slot by label
   */
  function fillSlot(label: string, value: string) {
    slotValues.value[label] = value
  }

  /**
   * Clear a slot value
   */
  function clearSlot(label: string) {
    const { [label]: _, ...rest } = slotValues.value
    slotValues.value = rest
  }

  /**
   * Build the final prompt string
   */
  const finalPrompt = computed(() => {
    return templateParts.value
      .map((part) => {
        if (part.type === 'text') {
          return part.content
        } else {
          const value = slotValues.value[part.content]
          return value?.trim() || `[${part.content}]`
        }
      })
      .join('')
  })

  /**
   * Check if a slot is filled
   */
  function isSlotFilled(label: string): boolean {
    return !!slotValues.value[label]?.trim()
  }

  /**
   * Get slot index by label
   */
  function getSlotIndex(label: string): number {
    return slots.value.findIndex(s => s.content === label)
  }

  return {
    templateString,
    templateParts,
    slots,
    selectedSlotIndex,
    currentSlot,
    currentSlotLabel,
    slotValues,
    allSlotsFilled,
    hasSlots,
    finalPrompt,
    setTemplate,
    selectSlot,
    selectNextSlot,
    selectPrevSlot,
    fillCurrentSlot,
    fillSlot,
    clearSlot,
    getSlotValue,
    isSlotFilled,
    getSlotIndex,
  }
}
