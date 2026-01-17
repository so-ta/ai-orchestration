<script setup lang="ts">
useI18n()

definePageMeta({
  layout: 'default',
  middleware: ['admin'],
})

const adminLinks = [
  {
    title: 'admin.tenants.title',
    description: 'admin.tenants.subtitle',
    href: '/admin/tenants',
    icon: 'tenant',
  },
  {
    title: 'admin.blocks.title',
    description: 'admin.blocks.subtitle',
    href: '/admin/blocks',
    icon: 'block',
  },
]
</script>

<template>
  <div>
    <div class="mb-6">
      <h1 style="font-size: 1.5rem; font-weight: 600;">
        {{ $t('admin.title') }}
      </h1>
      <p class="text-secondary" style="margin-top: 0.25rem;">
        {{ $t('admin.subtitle') }}
      </p>
    </div>

    <!-- Admin role notice -->
    <div class="warning-card">
      <p>{{ $t('admin.operatorNote') }}</p>
    </div>

    <!-- Admin navigation cards -->
    <div class="admin-grid">
      <NuxtLink
        v-for="link in adminLinks"
        :key="link.href"
        :to="link.href"
        class="admin-card"
      >
        <div class="admin-card-icon">
          <span v-if="link.icon === 'tenant'" class="icon-tenant">🏢</span>
          <span v-if="link.icon === 'block'" class="icon-block">🧩</span>
        </div>
        <div class="admin-card-content">
          <h3>{{ $t(link.title) }}</h3>
          <p class="text-secondary">{{ $t(link.description) }}</p>
        </div>
        <div class="admin-card-arrow">→</div>
      </NuxtLink>
    </div>
  </div>
</template>

<style scoped>
.warning-card {
  background: #fef3c7;
  border: 1px solid #fcd34d;
  border-radius: var(--radius);
  padding: 1rem;
  margin-bottom: 1.5rem;
}

.warning-card p {
  color: #92400e;
  font-size: 0.875rem;
  margin: 0;
}

.admin-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 1rem;
}

.admin-card {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1.5rem;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  text-decoration: none;
  color: inherit;
  transition: all 0.2s;
}

.admin-card:hover {
  border-color: var(--color-primary);
  transform: translateY(-2px);
  box-shadow: var(--shadow-lg);
}

.admin-card-icon {
  font-size: 2rem;
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-background);
  border-radius: var(--radius);
}

.admin-card-content {
  flex: 1;
}

.admin-card-content h3 {
  font-size: 1rem;
  font-weight: 600;
  margin: 0 0 0.25rem;
}

.admin-card-content p {
  font-size: 0.875rem;
  margin: 0;
}

.admin-card-arrow {
  font-size: 1.25rem;
  color: var(--color-text-secondary);
}
</style>
