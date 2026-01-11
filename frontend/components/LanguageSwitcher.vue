<script setup lang="ts">
const { locale, locales, setLocale } = useI18n()

const availableLocales = computed(() => {
  return locales.value.filter(l => typeof l !== 'string') as Array<{ code: string; name: string }>
})

async function handleLocaleChange(event: Event) {
  const target = event.target as HTMLSelectElement
  const newLocale = target.value as 'ja' | 'en'
  await setLocale(newLocale)
}
</script>

<template>
  <div class="language-switcher">
    <label class="switcher-label">
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <circle cx="12" cy="12" r="10"></circle>
        <line x1="2" y1="12" x2="22" y2="12"></line>
        <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"></path>
      </svg>
      <span>{{ $t('settings.language') }}</span>
    </label>
    <select
      :value="locale"
      class="locale-select"
      @change="handleLocaleChange"
    >
      <option
        v-for="loc in availableLocales"
        :key="loc.code"
        :value="loc.code"
      >
        {{ loc.name }}
      </option>
    </select>
  </div>
</template>

<style scoped>
.language-switcher {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.75rem 1rem;
  background: white;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
}

.switcher-label {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--color-text);
}

.switcher-label svg {
  color: var(--color-text-secondary);
}

.locale-select {
  padding: 0.5rem 0.75rem;
  font-size: 0.875rem;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  background: var(--color-surface);
  color: var(--color-text);
  cursor: pointer;
  min-width: 120px;
}

.locale-select:focus {
  outline: none;
  border-color: var(--color-primary);
}
</style>
