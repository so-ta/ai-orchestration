<script setup lang="ts">
import type { DevRole } from '~/composables/useAuth'

const { t } = useI18n()
const { isAuthenticated, isLoading, user, login, logout, isDevMode, devRole, setDevRole, isAdmin } = useAuth()

const route = useRoute()

// Base menu items for all users
const baseMenuItems = computed(() => [
  { name: t('nav.dashboard'), path: '/', icon: 'home' },
  { name: t('nav.workflows'), path: '/workflows', icon: 'workflow' },
  { name: t('nav.runs'), path: '/runs', icon: 'play' },
  { name: t('nav.schedules'), path: '/schedules', icon: 'clock' },
  { name: t('nav.webhooks'), path: '/webhooks', icon: 'webhook' },
  { name: t('nav.auditLogs'), path: '/audit-logs', icon: 'audit' },
  { name: t('nav.settings'), path: '/settings', icon: 'settings' }
])

// Admin-only menu items
const adminMenuItems = computed(() => [
  { name: t('nav.admin'), path: '/admin', icon: 'admin' }
])

// Combined menu items based on role
const menuItems = computed(() => {
  if (isAdmin()) {
    return [...baseMenuItems.value, ...adminMenuItems.value]
  }
  return baseMenuItems.value
})

// Handle dev role change
function handleDevRoleChange(event: Event) {
  const target = event.target as HTMLSelectElement
  setDevRole(target.value as DevRole)
}

// Check if the current route matches the menu item (including child routes)
function isActiveRoute(itemPath: string): boolean {
  // Dashboard (/) should only match exact path
  if (itemPath === '/') {
    return route.path === '/'
  }
  // Other routes match if current path starts with the menu item path
  return route.path.startsWith(itemPath)
}

async function handleLogin() {
  await login()
}

async function handleLogout() {
  await logout()
}
</script>

<template>
  <div class="layout">
    <aside class="sidebar">
      <div class="sidebar-header">
        <div class="logo">
          <svg xmlns="http://www.w3.org/2000/svg" width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"></polygon>
          </svg>
          <span class="logo-text">AI Orchestration</span>
        </div>
      </div>

      <nav class="sidebar-nav">
        <ul class="nav-list">
          <li v-for="item in menuItems" :key="item.path">
            <NuxtLink
              :to="item.path"
              :class="['nav-link', { active: isActiveRoute(item.path) }]"
            >
              <span class="nav-icon">
                <svg v-if="item.icon === 'home'" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"></path>
                  <polyline points="9 22 9 12 15 12 15 22"></polyline>
                </svg>
                <svg v-else-if="item.icon === 'workflow'" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
                  <polyline points="14 2 14 8 20 8"></polyline>
                  <line x1="16" y1="13" x2="8" y2="13"></line>
                  <line x1="16" y1="17" x2="8" y2="17"></line>
                </svg>
                <svg v-else-if="item.icon === 'play'" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polygon points="5 3 19 12 5 21 5 3"></polygon>
                </svg>
                <svg v-else-if="item.icon === 'clock'" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="10"></circle>
                  <polyline points="12 6 12 12 16 14"></polyline>
                </svg>
                <!-- Webhook icon (link) -->
                <svg v-else-if="item.icon === 'webhook'" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"></path>
                  <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"></path>
                </svg>
                <!-- Audit icon (clipboard list) -->
                <svg v-else-if="item.icon === 'audit'" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"></path>
                  <rect x="8" y="2" width="8" height="4" rx="1" ry="1"></rect>
                  <line x1="12" y1="11" x2="12" y2="17"></line>
                  <line x1="9" y1="14" x2="15" y2="14"></line>
                </svg>
                <svg v-else-if="item.icon === 'settings'" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="3"></circle>
                  <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path>
                </svg>
                <!-- Admin icon (shield) -->
                <svg v-else-if="item.icon === 'admin'" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"></path>
                </svg>
              </span>
              {{ item.name }}
            </NuxtLink>
          </li>
        </ul>
      </nav>

      <!-- User section at bottom of sidebar -->
      <div class="user-section">
        <div v-if="isLoading" class="loading-auth">
          <div class="loading-dot"></div>
          {{ $t('common.loading') }}
        </div>
        <div v-else-if="isAuthenticated && user" class="user-info">
          <div class="user-avatar">
            {{ (user.name || user.email || 'U')[0].toUpperCase() }}
          </div>
          <div class="user-details">
            <span class="user-name">{{ user.name || user.email }}</span>
            <span class="user-role">{{ user.roles[0] || 'user' }}</span>
          </div>
          <button class="btn-logout" @click="handleLogout" :title="$t('nav.logout')">
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"></path>
              <polyline points="16 17 21 12 16 7"></polyline>
              <line x1="21" y1="12" x2="9" y2="12"></line>
            </svg>
          </button>
        </div>
        <!-- Dev mode with role switcher -->
        <div v-else-if="isDevMode" class="dev-mode-section">
          <div class="dev-user-info">
            <div class="user-avatar dev-avatar">
              {{ devRole === 'admin' ? 'A' : 'U' }}
            </div>
            <div class="user-details">
              <span class="user-name">{{ devRole === 'admin' ? 'Admin User' : 'SaaS User' }}</span>
              <span class="user-role">{{ devRole }}</span>
            </div>
          </div>
          <div class="role-switcher">
            <label class="role-label">{{ $t('nav.testRole') }}</label>
            <select class="role-select" :value="devRole" @change="handleDevRoleChange">
              <option value="admin">{{ $t('nav.roleAdmin') }}</option>
              <option value="user">{{ $t('nav.roleUser') }}</option>
            </select>
          </div>
          <span class="dev-mode-badge">{{ $t('nav.developmentMode') }}</span>
        </div>
        <!-- Login button when not in dev mode and not authenticated -->
        <div v-else class="login-section">
          <button class="btn-login" @click="handleLogin">
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4"></path>
              <polyline points="10 17 15 12 10 7"></polyline>
              <line x1="15" y1="12" x2="3" y2="12"></line>
            </svg>
            {{ $t('nav.login') }}
          </button>
        </div>
      </div>
    </aside>
    <main class="main-content">
      <slot />
    </main>

    <!-- Toast notifications -->
    <ToastContainer />
  </div>
</template>

<style scoped>
.sidebar {
  display: flex;
  flex-direction: column;
  width: 260px;
  background-color: white;
  border-right: 1px solid var(--color-border);
}

.sidebar-header {
  padding: 1.25rem 1rem;
  border-bottom: 1px solid var(--color-border);
}

.logo {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  color: var(--color-primary);
}

.logo-text {
  font-size: 1.125rem;
  font-weight: 700;
}

.sidebar-nav {
  flex: 1;
  padding: 1rem 0.75rem;
  overflow-y: auto;
}

.nav-list {
  list-style: none;
  margin: 0;
  padding: 0;
}

.nav-link {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.625rem 0.875rem;
  border-radius: var(--radius);
  text-decoration: none;
  color: var(--color-text-secondary);
  font-size: 0.875rem;
  font-weight: 500;
  transition: all 0.15s;
  margin-bottom: 0.25rem;
}

.nav-link:hover {
  background: var(--color-surface);
  color: var(--color-text);
}

.nav-link.active {
  background: #eff6ff;
  color: var(--color-primary);
}

.nav-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
}

.user-section {
  padding: 1rem;
  border-top: 1px solid var(--color-border);
  background: var(--color-surface);
}

.loading-auth {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: var(--color-text-secondary);
  font-size: 0.875rem;
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

.user-info {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.user-avatar {
  width: 36px;
  height: 36px;
  background: var(--color-primary);
  color: white;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 0.875rem;
  flex-shrink: 0;
}

.user-details {
  display: flex;
  flex-direction: column;
  flex: 1;
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

.btn-logout {
  padding: 0.5rem;
  background: transparent;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.15s;
  display: flex;
  align-items: center;
  justify-content: center;
}

.btn-logout:hover {
  background: white;
  color: var(--color-error);
  border-color: var(--color-error);
}

.login-section {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.btn-login {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 0.625rem 1rem;
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

.dev-mode {
  color: var(--color-text-secondary);
  font-size: 0.75rem;
  text-align: center;
}

/* Dev mode section with role switcher */
.dev-mode-section {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.dev-user-info {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.dev-avatar {
  background: var(--color-warning, #f59e0b);
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

.dev-mode-badge {
  display: inline-block;
  padding: 0.25rem 0.5rem;
  background: #fef3c7;
  color: #92400e;
  font-size: 0.625rem;
  font-weight: 600;
  text-transform: uppercase;
  border-radius: var(--radius);
  text-align: center;
}

.main-content {
  flex: 1;
  padding: 1.5rem 2rem;
  overflow-y: auto;
  background: var(--color-background);
}

@media (max-width: 768px) {
  .sidebar {
    width: 100%;
    position: fixed;
    bottom: 0;
    left: 0;
    right: 0;
    z-index: 100;
    flex-direction: row;
    height: auto;
    border-right: none;
    border-top: 1px solid var(--color-border);
  }

  .sidebar-header,
  .user-section {
    display: none;
  }

  .sidebar-nav {
    padding: 0.5rem;
    width: 100%;
  }

  .nav-list {
    display: flex;
    justify-content: space-around;
  }

  .nav-link {
    flex-direction: column;
    padding: 0.5rem;
    font-size: 0.625rem;
    gap: 0.25rem;
  }

  .main-content {
    padding-bottom: 5rem;
  }
}
</style>
