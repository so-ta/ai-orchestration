<script setup lang="ts">
import GeneralSettingsTab from './settings/tabs/GeneralSettingsTab.vue'
import NotificationsTab from './settings/tabs/NotificationsTab.vue'
import CredentialsTab from './settings/tabs/CredentialsTab.vue'
import OAuth2Tab from './settings/tabs/OAuth2Tab.vue'

const props = defineProps<{
  show: boolean
  initialTab?: string
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const { t } = useI18n()

const activeTab = ref('general')
const credentialsTabRef = ref<InstanceType<typeof CredentialsTab> | null>(null)
const oauth2TabRef = ref<InstanceType<typeof OAuth2Tab> | null>(null)

const tabs = computed(() => [
  { id: 'general', label: t('settings.general') },
  { id: 'notifications', label: t('settings.notifications') },
  { id: 'credentials', label: t('credentials.title') },
  { id: 'oauth2', label: t('oauth.title') },
])

// Watch for tab change to trigger data fetching
watch(activeTab, (newTab) => {
  if (newTab === 'credentials') {
    credentialsTabRef.value?.fetchCredentials()
  } else if (newTab === 'oauth2') {
    oauth2TabRef.value?.fetchProviders()
  }
})

// Watch for modal open
watch(() => props.show, (isOpen) => {
  if (isOpen) {
    // Set initial tab if specified
    if (props.initialTab) {
      activeTab.value = props.initialTab
    }
    // Fetch data for the active tab
    nextTick(() => {
      if (activeTab.value === 'credentials') {
        credentialsTabRef.value?.fetchCredentials()
      } else if (activeTab.value === 'oauth2') {
        oauth2TabRef.value?.fetchProviders()
      }
    })
  }
})
</script>

<template>
  <UiModal
    :show="show"
    :title="t('settings.title')"
    size="lg"
    @close="emit('close')"
  >
    <div class="settings-modal">
      <!-- Tab navigation -->
      <div class="tabs-nav">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          :class="['tab-button', { active: activeTab === tab.id }]"
          @click="activeTab = tab.id"
        >
          {{ tab.label }}
        </button>
      </div>

      <div class="tab-content">
        <GeneralSettingsTab v-if="activeTab === 'general'" />
        <NotificationsTab v-if="activeTab === 'notifications'" />
        <CredentialsTab v-if="activeTab === 'credentials'" ref="credentialsTabRef" />
        <OAuth2Tab
          v-if="activeTab === 'oauth2'"
          ref="oauth2TabRef"
          @close-parent="emit('close')"
        />
      </div>
    </div>

    <template #footer>
      <button class="btn btn-secondary" @click="emit('close')">
        {{ t('common.close') }}
      </button>
    </template>
  </UiModal>
</template>

<style scoped>
.settings-modal {
  min-height: 400px;
}

.tabs-nav {
  display: flex;
  gap: 0.5rem;
  border-bottom: 1px solid var(--color-border);
  margin-bottom: 1.5rem;
}

.tab-button {
  padding: 0.75rem 1rem;
  background: none;
  border: none;
  color: var(--color-text-secondary);
  cursor: pointer;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
  transition: all 0.2s;
  font-size: 0.875rem;
}

.tab-button:hover {
  color: var(--color-text);
}

.tab-button.active {
  color: var(--color-primary);
  border-bottom-color: var(--color-primary);
}
</style>
