<script setup lang="ts">
import type { ExecutionLog } from '~/types/execution'

const { t } = useI18n()

const props = defineProps<{
  logs: ExecutionLog[]
  isOpen: boolean
  panelHeight: number
}>()

const emit = defineEmits<{
  (e: 'update:isOpen', value: boolean): void
  (e: 'update:panelHeight', value: number): void
  (e: 'clear'): void
}>()

// Refs
const logContainer = ref<HTMLElement | null>(null)
const isResizing = ref(false)
const startY = ref(0)
const startHeight = ref(0)

// Auto-scroll to bottom when new logs are added
const autoScroll = ref(true)

// Watch for new logs and auto-scroll
watch(() => props.logs.length, () => {
  if (autoScroll.value) {
    nextTick(() => {
      scrollToBottom()
    })
  }
})

function scrollToBottom() {
  if (logContainer.value) {
    logContainer.value.scrollTop = logContainer.value.scrollHeight
  }
}

// Toggle panel
function togglePanel() {
  emit('update:isOpen', !props.isOpen)
}

// Clear logs
function clearLogs() {
  emit('clear')
}

// Copy logs to clipboard
async function copyLogs() {
  const text = props.logs.map(log => {
    const time = formatTime(log.timestamp)
    return `[${time}] [${log.level.toUpperCase()}] ${log.message}`
  }).join('\n')

  try {
    await navigator.clipboard.writeText(text)
    // Could show a toast here
  } catch (e) {
    console.error('Failed to copy logs:', e)
  }
}

// Resize handling
function startResize(e: MouseEvent) {
  isResizing.value = true
  startY.value = e.clientY
  startHeight.value = props.panelHeight
  document.addEventListener('mousemove', onResize)
  document.addEventListener('mouseup', stopResize)
  document.body.style.cursor = 'ns-resize'
  document.body.style.userSelect = 'none'
}

function onResize(e: MouseEvent) {
  if (!isResizing.value) return
  const delta = startY.value - e.clientY
  const newHeight = Math.max(100, Math.min(600, startHeight.value + delta))
  emit('update:panelHeight', newHeight)
}

function stopResize() {
  isResizing.value = false
  document.removeEventListener('mousemove', onResize)
  document.removeEventListener('mouseup', stopResize)
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
}

// Format timestamp
function formatTime(date: Date): string {
  return date.toLocaleTimeString('ja-JP', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    fractionalSecondDigits: 3,
  })
}

// Get log level icon (reserved for future use)
function _getLevelIcon(level: string): string {
  switch (level) {
    case 'info': return 'info'
    case 'warn': return 'warning'
    case 'error': return 'error'
    case 'success': return 'success'
    default: return 'info'
  }
}

// Handle scroll to detect if user scrolled up
function onScroll() {
  if (logContainer.value) {
    const { scrollTop, scrollHeight, clientHeight } = logContainer.value
    // If user is near bottom (within 50px), enable auto-scroll
    autoScroll.value = scrollHeight - scrollTop - clientHeight < 50
  }
}
</script>

<template>
  <div class="execution-log-panel" :class="{ open: isOpen }">
    <!-- Resize Handle -->
    <div
      v-if="isOpen"
      class="resize-handle"
      @mousedown="startResize"
    >
      <div class="resize-grip"/>
    </div>

    <!-- Header Bar (always visible) -->
    <div class="panel-header" @click="togglePanel">
      <div class="header-left">
        <svg
          class="toggle-icon"
          :class="{ rotated: isOpen }"
          xmlns="http://www.w3.org/2000/svg"
          width="14"
          height="14"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <polyline points="18 15 12 9 6 15"/>
        </svg>
        <span class="header-title">{{ t('execution.logPanel.title') }}</span>
        <span v-if="logs.length > 0" class="log-count">({{ logs.length }})</span>
      </div>
      <div class="header-actions" @click.stop>
        <button
          class="action-btn"
          :title="t('execution.logPanel.copy')"
          :disabled="logs.length === 0"
          @click="copyLogs"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>
            <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
          </svg>
        </button>
        <button
          class="action-btn"
          :title="t('execution.logPanel.clear')"
          :disabled="logs.length === 0"
          @click="clearLogs"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="3 6 5 6 21 6"/>
            <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
          </svg>
        </button>
        <button
          class="action-btn"
          :title="isOpen ? t('execution.logPanel.close') : t('execution.logPanel.open')"
          @click="togglePanel"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18"/>
            <line x1="6" y1="6" x2="18" y2="18"/>
          </svg>
        </button>
      </div>
    </div>

    <!-- Log Content -->
    <div
      v-if="isOpen"
      ref="logContainer"
      class="log-content"
      :style="{ height: `${panelHeight - 36}px` }"
      @scroll="onScroll"
    >
      <!-- Empty State -->
      <div v-if="logs.length === 0" class="empty-logs">
        <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <polyline points="4 17 10 11 4 5"/>
          <line x1="12" y1="19" x2="20" y2="19"/>
        </svg>
        <p>{{ t('execution.logPanel.empty') }}</p>
      </div>

      <!-- Log Entries -->
      <div v-else class="log-entries">
        <div
          v-for="log in logs"
          :key="log.id"
          :class="['log-entry', `log-${log.level}`]"
        >
          <span class="log-time">{{ formatTime(log.timestamp) }}</span>
          <span :class="['log-level', log.level]">
            <!-- Info Icon -->
            <svg v-if="log.level === 'info'" xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"/>
              <line x1="12" y1="16" x2="12" y2="12"/>
              <line x1="12" y1="8" x2="12.01" y2="8"/>
            </svg>
            <!-- Warning Icon -->
            <svg v-else-if="log.level === 'warn'" xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/>
              <line x1="12" y1="9" x2="12" y2="13"/>
              <line x1="12" y1="17" x2="12.01" y2="17"/>
            </svg>
            <!-- Error Icon -->
            <svg v-else-if="log.level === 'error'" xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"/>
              <line x1="15" y1="9" x2="9" y2="15"/>
              <line x1="9" y1="9" x2="15" y2="15"/>
            </svg>
            <!-- Success Icon -->
            <svg v-else-if="log.level === 'success'" xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
              <polyline points="22 4 12 14.01 9 11.01"/>
            </svg>
          </span>
          <span v-if="log.stepName" class="log-step">[{{ log.stepName }}]</span>
          <span class="log-message">{{ log.message }}</span>
          <details v-if="log.data" class="log-data">
            <summary>{{ t('execution.logPanel.showData') }}</summary>
            <pre>{{ JSON.stringify(log.data, null, 2) }}</pre>
          </details>
        </div>
      </div>

      <!-- Auto-scroll indicator -->
      <div v-if="!autoScroll && logs.length > 0" class="scroll-indicator" @click="scrollToBottom">
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="6 9 12 15 18 9"/>
        </svg>
        {{ t('execution.logPanel.scrollToBottom') }}
      </div>
    </div>
  </div>
</template>

<style scoped>
.execution-log-panel {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  background: #1e1e1e;
  border-top: 1px solid #3c3c3c;
  z-index: 100;
  display: flex;
  flex-direction: column;
  transition: height 0.2s ease;
}

.execution-log-panel:not(.open) {
  height: 36px;
}

/* Resize Handle */
.resize-handle {
  position: absolute;
  top: -4px;
  left: 0;
  right: 0;
  height: 8px;
  cursor: ns-resize;
  z-index: 10;
}

.resize-grip {
  width: 40px;
  height: 4px;
  background: #555;
  border-radius: 2px;
  margin: 2px auto;
  opacity: 0;
  transition: opacity 0.15s;
}

.resize-handle:hover .resize-grip,
.resize-handle:active .resize-grip {
  opacity: 1;
}

/* Header */
.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 0.75rem;
  height: 36px;
  background: #252526;
  cursor: pointer;
  user-select: none;
  flex-shrink: 0;
}

.panel-header:hover {
  background: #2a2d2e;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.toggle-icon {
  color: #858585;
  transition: transform 0.2s;
}

.toggle-icon.rotated {
  transform: rotate(180deg);
}

.header-title {
  font-size: 0.75rem;
  font-weight: 500;
  color: #cccccc;
}

.log-count {
  font-size: 0.6875rem;
  color: #858585;
}

.header-actions {
  display: flex;
  gap: 0.25rem;
}

.action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  padding: 0;
  background: transparent;
  border: none;
  border-radius: 4px;
  color: #858585;
  cursor: pointer;
  transition: all 0.15s;
}

.action-btn:hover:not(:disabled) {
  background: #3c3c3c;
  color: #cccccc;
}

.action-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

/* Log Content */
.log-content {
  flex: 1;
  overflow-y: auto;
  padding: 0.5rem;
  font-family: 'SF Mono', Monaco, 'Cascadia Code', 'Consolas', monospace;
  font-size: 0.75rem;
  line-height: 1.5;
}

/* Empty State */
.empty-logs {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #858585;
  text-align: center;
}

.empty-logs svg {
  margin-bottom: 0.5rem;
  opacity: 0.5;
}

.empty-logs p {
  margin: 0;
  font-size: 0.8125rem;
}

/* Log Entries */
.log-entries {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.log-entry {
  display: flex;
  align-items: flex-start;
  gap: 0.5rem;
  padding: 0.25rem 0.375rem;
  border-radius: 3px;
  word-break: break-word;
}

.log-entry:hover {
  background: rgba(255, 255, 255, 0.03);
}

.log-time {
  color: #858585;
  flex-shrink: 0;
  font-size: 0.6875rem;
}

.log-level {
  display: flex;
  align-items: center;
  flex-shrink: 0;
}

.log-level.info {
  color: #3794ff;
}

.log-level.warn {
  color: #cca700;
}

.log-level.error {
  color: #f14c4c;
}

.log-level.success {
  color: #89d185;
}

.log-step {
  color: #ce9178;
  flex-shrink: 0;
  font-size: 0.6875rem;
}

.log-message {
  color: #cccccc;
  flex: 1;
}

.log-info .log-message {
  color: #cccccc;
}

.log-warn .log-message {
  color: #cca700;
}

.log-error .log-message {
  color: #f14c4c;
}

.log-success .log-message {
  color: #89d185;
}

/* Log Data */
.log-data {
  margin-top: 0.25rem;
  margin-left: 5rem;
}

.log-data summary {
  cursor: pointer;
  color: #858585;
  font-size: 0.6875rem;
}

.log-data summary:hover {
  color: #cccccc;
}

.log-data pre {
  margin: 0.25rem 0 0 0;
  padding: 0.5rem;
  background: #2d2d2d;
  border-radius: 4px;
  overflow-x: auto;
  color: #cccccc;
  font-size: 0.6875rem;
}

/* Scroll Indicator */
.scroll-indicator {
  position: sticky;
  bottom: 0.5rem;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.25rem;
  padding: 0.375rem 0.75rem;
  background: #3c3c3c;
  border-radius: 20px;
  margin: 0.5rem auto;
  width: fit-content;
  cursor: pointer;
  color: #cccccc;
  font-size: 0.6875rem;
  transition: background 0.15s;
}

.scroll-indicator:hover {
  background: #4c4c4c;
}

/* Scrollbar */
.log-content::-webkit-scrollbar {
  width: 8px;
}

.log-content::-webkit-scrollbar-track {
  background: transparent;
}

.log-content::-webkit-scrollbar-thumb {
  background: #424242;
  border-radius: 4px;
}

.log-content::-webkit-scrollbar-thumb:hover {
  background: #555555;
}
</style>
