<script setup lang="ts">
import type { DevRole } from '~/composables/useAuth'

const { t } = useI18n()
const { isAuthenticated, isLoading, user, login, logout, isDevMode, devRole, setDevRole } = useAuth()

const emit = defineEmits<{
  openSettings: []
}>()

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
  <div ref="menuRef" class="floating-account">
    <!-- Loading state -->
    <div v-if="isLoading" class="loading-state">
      <div class="loading-dot" />
    </div>

    <!-- Login button -->
    <button
      v-else-if="!isAuthenticated && !isDevMode"
      class="login-btn"
      @click="handleLogin"
    >
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4" />
        <polyline points="10 17 15 12 10 7" />
        <line x1="15" y1="12" x2="3" y2="12" />
      </svg>
    </button>

    <!-- Avatar Button (Miro style) -->
    <button
      v-else
      class="account-trigger"
      :class="{ open: isOpen }"
      @click="toggleMenu"
    >
      <div class="avatar" :class="{ 'dev-avatar': isDevMode }">
        {{ avatarLetter }}
      </div>
      <!-- Dropdown indicator -->
      <svg class="dropdown-chevron" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
        <path d="m6 9 6 6 6-6" />
      </svg>
    </button>

    <!-- Dropdown menu -->
    <Transition name="dropdown">
      <div v-if="isOpen && (isAuthenticated || isDevMode)" class="dropdown">
        <!-- User info -->
        <div class="dropdown-header">
          <div class="avatar large" :class="{ 'dev-avatar': isDevMode }">
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
          <div class="dev-badge">{{ t('nav.developmentMode') }}</div>
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
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="3" />
              <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" />
            </svg>
            {{ t('settings.title') }}
          </button>

          <button class="dropdown-item logout" @click="handleLogout">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
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
.floating-account {
  position: fixed;
  top: 12px;
  right: 12px;
  z-index: 100;
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
  background: #3b82f6;
  border-radius: 50%;
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

.login-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(12px);
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
  cursor: pointer;
  color: #6b7280;
  transition: all 0.15s;
}

.login-btn:hover {
  background: white;
  color: #3b82f6;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
}

/* Account trigger button (Miro style - minimal) */
.account-trigger {
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 0;
  background: transparent;
  border: none;
  cursor: pointer;
  transition: opacity 0.15s;
}

.account-trigger:hover {
  opacity: 0.85;
}

.avatar {
  width: 32px;
  height: 32px;
  background: linear-gradient(135deg, #8b5cf6, #6d28d9);
  color: white;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 13px;
  flex-shrink: 0;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.dropdown-chevron {
  color: #6b7280;
  transition: transform 0.15s;
}

.account-trigger.open .dropdown-chevron {
  transform: rotate(180deg);
}

.avatar.large {
  width: 36px;
  height: 36px;
  font-size: 14px;
}

.avatar.dev-avatar {
  background: linear-gradient(135deg, #8b5cf6, #6d28d9);
}

/* Dropdown */
.dropdown {
  position: absolute;
  top: calc(100% + 8px);
  right: 0;
  min-width: 240px;
  background: white;
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 12px;
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.12);
  overflow: hidden;
}

.dropdown-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px;
}

.user-info {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.user-name {
  font-weight: 500;
  color: #111827;
  font-size: 14px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-role {
  color: #6b7280;
  font-size: 12px;
  text-transform: capitalize;
}

.dropdown-divider {
  height: 1px;
  background: #e5e7eb;
  margin: 0;
}

.dev-mode-section {
  padding: 12px 16px;
}

.dev-badge {
  display: inline-block;
  padding: 4px 8px;
  background: #fef3c7;
  color: #92400e;
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  border-radius: 4px;
  margin-bottom: 12px;
}

.role-switcher {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.role-label {
  font-size: 12px;
  color: #6b7280;
}

.role-select {
  padding: 8px;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  background: white;
  font-size: 14px;
  cursor: pointer;
}

.role-select:focus {
  outline: none;
  border-color: #3b82f6;
}

.dropdown-items {
  padding: 8px;
}

.dropdown-item {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  padding: 10px 12px;
  background: transparent;
  border: none;
  border-radius: 8px;
  color: #374151;
  font-size: 14px;
  text-align: left;
  cursor: pointer;
  transition: background 0.15s;
}

.dropdown-item:hover {
  background: #f3f4f6;
}

.dropdown-item svg {
  color: #6b7280;
}

.dropdown-item.logout:hover {
  background: #fef2f2;
  color: #dc2626;
}

.dropdown-item.logout:hover svg {
  color: #dc2626;
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
