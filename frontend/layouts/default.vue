<script setup lang="ts">
const { t } = useI18n()
const { isAdmin } = useAuth()

const showSettingsModal = ref(false)

function openSettings() {
  showSettingsModal.value = true
}
</script>

<template>
  <div class="layout">
    <!-- Minimal Header -->
    <header class="header">
      <div class="header-left">
        <NuxtLink to="/projects" class="logo">
          <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2" />
          </svg>
          <span class="logo-text">AI Orchestration</span>
        </NuxtLink>
      </div>

      <div class="header-right">
        <!-- Admin Link (conditional) -->
        <NuxtLink v-if="isAdmin()" to="/admin" class="admin-link">
          <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
          </svg>
          {{ t('nav.admin') }}
        </NuxtLink>

        <!-- User Menu -->
        <UserMenu @open-settings="openSettings" />
      </div>
    </header>

    <!-- Main Content -->
    <main class="main-content">
      <slot />
    </main>

    <!-- Settings Modal -->
    <SettingsModal :show="showSettingsModal" @close="showSettingsModal = false" />

    <!-- Toast notifications -->
    <ToastContainer />
    <!-- Global confirmation dialog -->
    <UiConfirmDialog />
  </div>
</template>

<style scoped>
.layout {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}

.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 1.5rem;
  height: 56px;
  background: white;
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}

.header-left {
  display: flex;
  align-items: center;
}

.logo {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  text-decoration: none;
  color: var(--color-primary);
}

.logo-text {
  font-size: 1.125rem;
  font-weight: 700;
  color: var(--color-text);
}

.header-right {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.admin-link {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
  text-decoration: none;
  color: var(--color-text-secondary);
  font-size: 0.875rem;
  font-weight: 500;
  border-radius: var(--radius);
  transition: all 0.15s;
}

.admin-link:hover {
  background: var(--color-surface);
  color: var(--color-text);
}

.main-content {
  flex: 1;
  padding: 1.5rem 2rem;
  overflow-y: auto;
  background: var(--color-background);
}

@media (max-width: 768px) {
  .header {
    padding: 0 1rem;
  }

  .logo-text {
    display: none;
  }

  .main-content {
    padding: 1rem;
  }
}
</style>
