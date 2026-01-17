<script setup lang="ts">
import type { DevRole } from '~/composables/useAuth'

const emit = defineEmits<{
  (e: 'openSettings'): void
}>()

const { t } = useI18n()
const { isAuthenticated, isLoading, user, login, logout, isDevMode, devRole, setDevRole } = useAuth()

const isOpen = ref(false)
const menuRef = ref<HTMLElement | null>(null)

// Close dropdown when clicking outside
function handleClickOutside(event: MouseEvent) {
  if (menuRef.value && !menuRef.value.contains(event.target as Node)) {
    isOpen.value = false
  }
}

function toggleMenu() {
  isOpen.value = !isOpen.value
}

function handleOpenSettings() {
  isOpen.value = false
  emit('openSettings')
}

async function handleLogout() {
  isOpen.value = false
  await logout()
}

async function handleLogin() {
  await login()
}

function handleDevRoleChange(event: Event) {
  const target = event.target as HTMLSelectElement
  setDevRole(target.value as DevRole)
}

// Get display name
const displayName = computed(() => {
  if (isDevMode.value) {
    return devRole.value === 'admin' ? 'Admin User' : 'SaaS User'
  }
  return user.value?.name || user.value?.email || 'User'
})

// Get avatar letter
const avatarLetter = computed(() => {
  if (isDevMode.value) {
    return devRole.value === 'admin' ? 'A' : 'U'
  }
  return (user.value?.name || user.value?.email || 'U')[0].toUpperCase()
})

// Get role display
const roleDisplay = computed(() => {
  if (isDevMode.value) {
    return devRole.value
  }
  return user.value?.roles?.[0] || 'user'
})

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>

<template>
  <div ref="menuRef" class="user-menu">
    <!-- Loading state -->
    <div v-if="isLoading" class="loading-state">
      <div class="loading-dot" />
    </div>

    <!-- Login button (not authenticated and not dev mode) -->
    <button
      v-else-if="!isAuthenticated && !isDevMode"
      class="btn-login"
      @click="handleLogin"
    >
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4" />
        <polyline points="10 17 15 12 10 7" />
        <line x1="15" y1="12" x2="3" y2="12" />
      </svg>
      {{ t('nav.login') }}
    </button>

    <!-- User dropdown trigger -->
    <button
      v-else
      class="user-trigger"
      :class="{ open: isOpen }"
      @click="toggleMenu"
    >
      <div class="user-avatar" :class="{ 'dev-avatar': isDevMode }">
        {{ avatarLetter }}
      </div>
      <svg class="chevron" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <polyline points="6 9 12 15 18 9" />
      </svg>
    </button>

    <!-- Dropdown menu -->
    <Transition name="dropdown">
      <div v-if="isOpen && (isAuthenticated || isDevMode)" class="dropdown">
        <!-- User info -->
        <div class="dropdown-header">
          <div class="user-avatar large" :class="{ 'dev-avatar': isDevMode }">
            {{ avatarLetter }}
          </div>
          <div class="user-info">
            <span class="user-name">{{ displayName }}</span>
            <span class="user-role">{{ roleDisplay }}</span>
          </div>
        </div>

        <div class="dropdown-divider" />

        <!-- Dev mode role switcher -->
        <div v-if="isDevMode" class="dev-mode-section">
          <div class="dev-badge">
            {{ t('nav.developmentMode') }}
          </div>
          <div class="role-switcher">
            <label class="role-label">{{ t('nav.testRole') }}</label>
            <select class="role-select" :value="devRole" @change="handleDevRoleChange">
              <option value="admin">{{ t('nav.roleAdmin') }}</option>
              <option value="user">{{ t('nav.roleUser') }}</option>
            </select>
          </div>
          <div class="dropdown-divider" />
        </div>

        <!-- Menu items -->
        <div class="dropdown-items">
          <button class="dropdown-item" @click="handleOpenSettings">
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="3" />
              <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" />
            </svg>
            {{ t('settings.title') }}
          </button>

          <button class="dropdown-item logout" @click="handleLogout">
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
              <polyline points="16 17 21 12 16 7" />
              <line x1="21" y1="12" x2="9" y2="12" />
            </svg>
            {{ t('nav.logout') }}
          </button>
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.user-menu {
  position: relative;
}

.loading-state {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
}

.loading-dot {
  width: 8px;
  height: 8px;
  background: var(--color-primary);
  border-radius: 50%;
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

.btn-login {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  background: var(--color-primary);
  border: none;
  border-radius: var(--radius);
  color: white;
  cursor: pointer;
  font-size: 0.875rem;
  font-weight: 500;
  transition: opacity 0.15s;
}

.btn-login:hover {
  opacity: 0.9;
}

.user-trigger {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.25rem;
  padding-right: 0.5rem;
  background: transparent;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  cursor: pointer;
  transition: all 0.15s;
}

.user-trigger:hover,
.user-trigger.open {
  background: var(--color-surface);
  border-color: var(--color-border-dark, #d1d5db);
}

.user-avatar {
  width: 32px;
  height: 32px;
  background: var(--color-primary);
  color: white;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 0.875rem;
  flex-shrink: 0;
}

.user-avatar.large {
  width: 40px;
  height: 40px;
  font-size: 1rem;
}

.user-avatar.dev-avatar {
  background: var(--color-warning, #f59e0b);
}

.chevron {
  color: var(--color-text-secondary);
  transition: transform 0.15s;
}

.user-trigger.open .chevron {
  transform: rotate(180deg);
}

.dropdown {
  position: absolute;
  top: calc(100% + 0.5rem);
  right: 0;
  min-width: 240px;
  background: white;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg, 0.5rem);
  box-shadow: 0 10px 15px -3px rgb(0 0 0 / 0.1), 0 4px 6px -4px rgb(0 0 0 / 0.1);
  z-index: 1000;
  overflow: hidden;
}

.dropdown-header {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 1rem;
}

.user-info {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.user-name {
  font-weight: 500;
  color: var(--color-text);
  font-size: 0.875rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-role {
  color: var(--color-text-secondary);
  font-size: 0.75rem;
  text-transform: capitalize;
}

.dropdown-divider {
  height: 1px;
  background: var(--color-border);
  margin: 0;
}

.dev-mode-section {
  padding: 0.75rem 1rem;
}

.dev-badge {
  display: inline-block;
  padding: 0.25rem 0.5rem;
  background: #fef3c7;
  color: #92400e;
  font-size: 0.625rem;
  font-weight: 600;
  text-transform: uppercase;
  border-radius: var(--radius);
  margin-bottom: 0.75rem;
}

.role-switcher {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.role-label {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

.role-select {
  padding: 0.5rem;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  background: white;
  font-size: 0.875rem;
  cursor: pointer;
}

.role-select:focus {
  outline: none;
  border-color: var(--color-primary);
}

.dropdown-items {
  padding: 0.5rem;
}

.dropdown-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  width: 100%;
  padding: 0.625rem 0.75rem;
  background: transparent;
  border: none;
  border-radius: var(--radius);
  color: var(--color-text);
  font-size: 0.875rem;
  text-align: left;
  cursor: pointer;
  transition: background 0.15s;
}

.dropdown-item:hover {
  background: var(--color-surface);
}

.dropdown-item svg {
  color: var(--color-text-secondary);
}

.dropdown-item.logout:hover {
  background: #fef2f2;
  color: var(--color-error);
}

.dropdown-item.logout:hover svg {
  color: var(--color-error);
}

/* Dropdown animation */
.dropdown-enter-active,
.dropdown-leave-active {
  transition: opacity 0.15s, transform 0.15s;
}

.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
