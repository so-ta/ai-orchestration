<script setup lang="ts">
const { t } = useI18n()

defineProps<{
  disabled?: boolean
}>()
</script>

<template>
  <div class="form-section">
    <h4 class="section-title">{{ t('stepConfig.start.title') }}</h4>
    <div class="start-info-box">
      <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="currentColor" class="start-icon">
        <polygon points="5 3 19 12 5 21 5 3" />
      </svg>
      <div class="start-info-content">
        <p class="start-info-title">{{ t('stepConfig.start.entryPoint') }}</p>
        <p class="start-info-desc">{{ t('stepConfig.start.description') }}</p>
      </div>
    </div>
    <div class="start-warning">
      <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/>
        <line x1="12" y1="9" x2="12" y2="13"/>
        <line x1="12" y1="17" x2="12.01" y2="17"/>
      </svg>
      <span>{{ t('stepConfig.start.cannotDelete') }}</span>
    </div>
  </div>
</template>

<style scoped>
.form-section {
  margin-bottom: 1.5rem;
  padding-bottom: 1.5rem;
  border-bottom: 1px solid var(--color-border);
}

.section-title {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--color-text);
  margin: 0 0 0.75rem 0;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.start-info-box {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 1rem;
  background: linear-gradient(135deg, #ecfdf5 0%, #d1fae5 100%);
  border: 1px solid #a7f3d0;
  border-radius: 8px;
}

.start-icon {
  color: #10b981;
  flex-shrink: 0;
}

.start-info-content {
  flex: 1;
}

.start-info-title {
  font-size: 0.875rem;
  font-weight: 600;
  color: #047857;
  margin: 0 0 0.25rem 0;
}

.start-info-desc {
  font-size: 0.75rem;
  color: #065f46;
  margin: 0;
  line-height: 1.4;
}

.start-warning {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.625rem 0.75rem;
  background: #fef3c7;
  border: 1px solid #fcd34d;
  border-radius: 6px;
  margin-top: 0.75rem;
  font-size: 0.6875rem;
  color: #92400e;
}

.start-warning svg {
  flex-shrink: 0;
  color: #f59e0b;
}
</style>
