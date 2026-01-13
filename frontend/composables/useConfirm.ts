// Confirmation dialog composable
export interface ConfirmOptions {
  title: string
  message: string
  confirmText?: string
  cancelText?: string
  variant?: 'default' | 'danger'
}

interface ConfirmState {
  show: boolean
  options: ConfirmOptions | null
  resolve: ((value: boolean) => void) | null
}

const state = ref<ConfirmState>({
  show: false,
  options: null,
  resolve: null,
})

export function useConfirm() {
  /**
   * Show a confirmation dialog and wait for user response
   * @param options - Dialog configuration
   * @returns Promise that resolves to true if confirmed, false if cancelled
   */
  function confirm(options: ConfirmOptions | string): Promise<boolean> {
    // Allow simple string message as shorthand
    const opts: ConfirmOptions = typeof options === 'string'
      ? { title: 'Confirm', message: options }
      : options

    return new Promise((resolve) => {
      state.value = {
        show: true,
        options: {
          confirmText: 'OK',
          cancelText: 'Cancel',
          variant: 'default',
          ...opts,
        },
        resolve,
      }
    })
  }

  function handleConfirm() {
    if (state.value.resolve) {
      state.value.resolve(true)
    }
    close()
  }

  function handleCancel() {
    if (state.value.resolve) {
      state.value.resolve(false)
    }
    close()
  }

  function close() {
    state.value = {
      show: false,
      options: null,
      resolve: null,
    }
  }

  return {
    state: readonly(state),
    confirm,
    handleConfirm,
    handleCancel,
    close,
  }
}
