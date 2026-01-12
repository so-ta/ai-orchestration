<script setup lang="ts">
const { t } = useI18n()
const { isAuthenticated, isLoading, user, login, logout } = useAuth()

const route = useRoute()

const menuItems = computed(() => [
  { name: t('nav.dashboard'), path: '/', icon: 'home' },
  { name: t('nav.workflows'), path: '/workflows', icon: 'workflow' },
  { name: t('nav.runs'), path: '/runs', icon: 'play' },
  { name: t('nav.schedules'), path: '/schedules', icon: 'clock' },
  { name: t('nav.settings'), path: '/settings', icon: 'settings' }
])

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
                <svg v-else-if="item.icon === 'settings'" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="3"></circle>
                  <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path>
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
        <div v-else class="login-section">
          <button class="btn-login" @click="handleLogin">
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4"></path>
              <polyline points="10 17 15 12 10 7"></polyline>
              <line x1="15" y1="12" x2="3" y2="12"></line>
            </svg>
            {{ $t('nav.login') }}
          </button>
          <span class="dev-mode">{{ $t('nav.developmentMode') }}</span>
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
