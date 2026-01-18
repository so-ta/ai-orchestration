<script setup lang="ts">
const { t } = useI18n()
</script>

<template>
  <div class="properties-empty">
    <div class="empty-icon">
      <svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <path d="M12 3l8 4.5v9L12 21l-8-4.5v-9L12 3z"/>
        <path d="M12 12l8-4.5"/>
        <path d="M12 12v9"/>
        <path d="M12 12L4 7.5"/>
      </svg>
    </div>
    <p class="empty-title">{{ t('editor.noStepSelected') }}</p>
    <p class="empty-desc">{{ t('editor.selectStepHint') }}</p>

    <!-- Quick Tips -->
    <div class="empty-tips">
      <div class="empty-tips-title">{{ t('editor.quickTips') }}</div>
      <ul class="empty-tips-list">
        <li>
          <span class="tip-icon">
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M5 12h14"/>
              <path d="M12 5v14"/>
            </svg>
          </span>
          {{ t('editor.tipDragBlock') }}
        </li>
        <li>
          <span class="tip-icon">
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/>
              <polyline points="15 3 21 3 21 9"/>
              <line x1="10" y1="14" x2="21" y2="3"/>
            </svg>
          </span>
          {{ t('editor.tipConnectNodes') }}
        </li>
        <li>
          <span class="tip-icon">
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
            </svg>
          </span>
          {{ t('editor.tipClickToSelect') }}
        </li>
      </ul>
    </div>

    <!-- Keyboard Shortcuts -->
    <div class="empty-shortcuts">
      <div class="empty-shortcuts-title">{{ t('editor.keyboardShortcuts') }}</div>
      <div class="shortcut-item">
        <kbd>Delete</kbd>
        <span>{{ t('editor.shortcutDelete') }}</span>
      </div>
      <div class="shortcut-item">
        <kbd>Esc</kbd>
        <span>{{ t('editor.shortcutDeselect') }}</span>
      </div>
      <div class="shortcut-item">
        <kbd>Ctrl</kbd> + <kbd>C</kbd>
        <span>{{ t('editor.shortcutCopy') }}</span>
      </div>
      <div class="shortcut-item">
        <kbd>Ctrl</kbd> + <kbd>V</kbd>
        <span>{{ t('editor.shortcutPaste') }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.properties-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 1.5rem;
  text-align: center;
  overflow-y: auto;
  height: 100%;
}

.empty-icon {
  color: var(--color-primary);
  opacity: 0.6;
  margin-bottom: 1rem;
  margin-top: 1rem;
}

.empty-title {
  font-weight: 600;
  font-size: 1rem;
  color: var(--color-text);
  margin: 0;
}

.empty-desc {
  font-size: 0.8125rem;
  color: var(--color-text-secondary);
  margin-top: 0.5rem;
  line-height: 1.5;
}

/* Quick Tips */
.empty-tips {
  width: 100%;
  margin-top: 1.5rem;
  padding: 1rem;
  background: linear-gradient(135deg, #f0f9ff 0%, #e0f2fe 100%);
  border-radius: 8px;
  text-align: left;
}

.empty-tips-title {
  font-size: 0.6875rem;
  font-weight: 600;
  color: #0369a1;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-bottom: 0.75rem;
}

.empty-tips-list {
  list-style: none;
  padding: 0;
  margin: 0;
}

.empty-tips-list li {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.75rem;
  color: #0c4a6e;
  padding: 0.375rem 0;
}

.tip-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  background: white;
  border-radius: 4px;
  color: #0284c7;
  flex-shrink: 0;
}

/* Keyboard Shortcuts */
.empty-shortcuts {
  width: 100%;
  margin-top: 1rem;
  padding: 1rem;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 8px;
  text-align: left;
}

.empty-shortcuts-title {
  font-size: 0.6875rem;
  font-weight: 600;
  color: var(--color-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-bottom: 0.75rem;
}

.shortcut-item {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  padding: 0.25rem 0;
}

.shortcut-item kbd {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 1.5rem;
  padding: 0.125rem 0.375rem;
  font-family: inherit;
  font-size: 0.625rem;
  font-weight: 500;
  background: white;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
}

.shortcut-item span {
  margin-left: auto;
  color: var(--color-text);
}
</style>
