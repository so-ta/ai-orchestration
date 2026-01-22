<script setup lang="ts">
/**
 * CopilotOAuthButton.vue
 *
 * OAuth connection button with existing credentials dropdown.
 * Handles service-specific branding and connection flows.
 */

const { t } = useI18n()

const props = defineProps<{
  service: string
  serviceName: string
  serviceIcon?: string
  existingCredentials?: Array<{ id: string; name: string }>
  disabled?: boolean
}>()

const emit = defineEmits<{
  connect: [credentialId: string, credentialName: string]
}>()

// State
const showDropdown = ref(false)
const isConnecting = ref(false)

// Service-specific icons (emoji fallbacks)
const serviceIcons: Record<string, string> = {
  slack: '💬',
  discord: '🎮',
  github: '🐙',
  google: '🔵',
  notion: '📝',
  openai: '🤖',
  anthropic: '🧠',
  email: '📧',
}

const displayIcon = computed(() => {
  return props.serviceIcon || serviceIcons[props.service] || '🔑'
})

// Service-specific button colors
const serviceColors: Record<string, { bg: string; hover: string }> = {
  slack: { bg: '#4A154B', hover: '#611f64' },
  discord: { bg: '#5865F2', hover: '#4752c4' },
  github: { bg: '#24292e', hover: '#373e47' },
  google: { bg: '#4285f4', hover: '#3367d6' },
  notion: { bg: '#000000', hover: '#333333' },
}

const buttonStyle = computed(() => {
  const colors = serviceColors[props.service]
  if (colors) {
    return {
      '--btn-bg': colors.bg,
      '--btn-hover': colors.hover,
    }
  }
  return {}
})

// Handle new OAuth connection
async function handleConnect() {
  if (props.disabled || isConnecting.value) return

  isConnecting.value = true

  try {
    // Open OAuth popup window
    const width = 600
    const height = 700
    const left = window.screenX + (window.outerWidth - width) / 2
    const top = window.screenY + (window.outerHeight - height) / 2

    const popup = window.open(
      `/api/v1/oauth/${props.service}/connect`,
      'oauth-connect',
      `width=${width},height=${height},left=${left},top=${top}`
    )

    if (!popup) {
      console.error('Failed to open OAuth popup')
      isConnecting.value = false
      return
    }

    // Listen for OAuth callback
    const handleMessage = (event: MessageEvent) => {
      if (event.origin !== window.location.origin) return

      if (event.data.type === 'oauth-success') {
        emit('connect', event.data.credentialId, event.data.credentialName || props.serviceName)
        window.removeEventListener('message', handleMessage)
        isConnecting.value = false
      } else if (event.data.type === 'oauth-error') {
        console.error('OAuth error:', event.data.error)
        window.removeEventListener('message', handleMessage)
        isConnecting.value = false
      }
    }

    window.addEventListener('message', handleMessage)

    // Fallback: check if popup is closed
    const checkClosed = setInterval(() => {
      if (popup.closed) {
        clearInterval(checkClosed)
        window.removeEventListener('message', handleMessage)
        isConnecting.value = false
      }
    }, 1000)
  } catch (error) {
    console.error('OAuth connection failed:', error)
    isConnecting.value = false
  }
}

// Handle selecting existing credential
function handleSelectCredential(credential: { id: string; name: string }) {
  emit('connect', credential.id, credential.name)
  showDropdown.value = false
}

// Close dropdown when clicking outside
function handleClickOutside(event: MouseEvent) {
  const target = event.target as HTMLElement
  if (!target.closest('.oauth-button-container')) {
    showDropdown.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>

<template>
  <div class="oauth-button-container">
    <!-- Main connect button -->
    <button
      class="oauth-button"
      :class="{ connecting: isConnecting, 'has-service-color': !!serviceColors[service] }"
      :style="buttonStyle"
      :disabled="disabled || isConnecting"
      @click="handleConnect"
    >
      <span class="service-icon">{{ displayIcon }}</span>
      <span v-if="isConnecting" class="button-text">
        {{ t('copilot.oauth.connecting') }}
      </span>
      <span v-else class="button-text">
        {{ t('copilot.oauth.connectWith', { service: serviceName }) }}
      </span>
      <span v-if="isConnecting" class="loading-spinner" />
    </button>

    <!-- Existing credentials section -->
    <div v-if="existingCredentials && existingCredentials.length > 0" class="existing-section">
      <span class="or-divider">{{ t('copilot.oauth.or') }}</span>
      <div class="dropdown-container">
        <button
          class="dropdown-trigger"
          :disabled="disabled"
          @click.stop="showDropdown = !showDropdown"
        >
          {{ t('copilot.oauth.selectExisting') }}
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="14"
            height="14"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            class="chevron"
            :class="{ rotated: showDropdown }"
          >
            <polyline points="6 9 12 15 18 9" />
          </svg>
        </button>

        <div v-if="showDropdown" class="dropdown-menu">
          <button
            v-for="cred in existingCredentials"
            :key="cred.id"
            class="dropdown-item"
            @click="handleSelectCredential(cred)"
          >
            <span class="cred-icon">🔑</span>
            <span class="cred-name">{{ cred.name }}</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.oauth-button-container {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.oauth-button {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 0.625rem 1rem;
  font-size: 0.875rem;
  font-weight: 500;
  color: white;
  background: var(--btn-bg, var(--color-primary));
  border: none;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s;
  position: relative;
}

.oauth-button:hover:not(:disabled) {
  background: var(--btn-hover, var(--color-primary-hover));
  transform: translateY(-1px);
}

.oauth-button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  transform: none;
}

.oauth-button.connecting {
  cursor: wait;
}

.service-icon {
  font-size: 1.125rem;
}

.button-text {
  flex: 1;
  text-align: center;
}

.loading-spinner {
  width: 14px;
  height: 14px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

/* Existing credentials section */
.existing-section {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.or-divider {
  font-size: 0.75rem;
  color: var(--color-text-tertiary);
  text-align: center;
}

.dropdown-container {
  position: relative;
}

.dropdown-trigger {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  width: 100%;
  padding: 0.5rem 0.75rem;
  font-size: 0.8125rem;
  color: var(--color-text-secondary);
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
}

.dropdown-trigger:hover:not(:disabled) {
  color: var(--color-text);
  border-color: var(--color-text-secondary);
}

.dropdown-trigger:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.chevron {
  transition: transform 0.2s;
}

.chevron.rotated {
  transform: rotate(180deg);
}

.dropdown-menu {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  right: 0;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 6px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  z-index: 10;
  overflow: hidden;
}

.dropdown-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  width: 100%;
  padding: 0.5rem 0.75rem;
  font-size: 0.8125rem;
  color: var(--color-text);
  background: transparent;
  border: none;
  cursor: pointer;
  transition: background 0.15s;
}

.dropdown-item:hover {
  background: var(--color-background);
}

.cred-icon {
  font-size: 0.875rem;
}

.cred-name {
  flex: 1;
  text-align: left;
}
</style>
