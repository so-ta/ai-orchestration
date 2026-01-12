<script setup lang="ts">
definePageMeta({
  layout: 'default',
})

// Sample step types for preview
const stepTypes = [
  { type: 'start', label: 'Start', color: '#22c55e', icon: '▶' },
  { type: 'llm', label: 'LLM', color: '#8b5cf6', icon: '🤖' },
  { type: 'tool', label: 'HTTP Request', color: '#3b82f6', icon: '🔧' },
  { type: 'condition', label: 'Condition', color: '#f59e0b', icon: '◇' },
  { type: 'loop', label: 'Loop', color: '#06b6d4', icon: '🔄' },
  { type: 'subflow', label: 'Subflow', color: '#ec4899', icon: '📦' },
]

// Sample group types for preview
const groupTypes = [
  { type: 'parallel', label: 'Parallel', color: '#8b5cf6', icon: '⫘' },
  { type: 'try_catch', label: 'Try-Catch', color: '#ef4444', icon: '⚡' },
  { type: 'if_else', label: 'If-Else', color: '#f59e0b', icon: '◇' },
]

// Status examples
const statusExamples = [
  { status: 'running', label: 'Running' },
  { status: 'completed', label: 'Completed' },
  { status: 'failed', label: 'Failed' },
]
</script>

<template>
  <div class="design-preview">
    <div class="preview-header">
      <h1>DAG Editor Design Preview</h1>
      <p class="subtitle">Minimal Linear Style - 実装前のデザイン確認</p>
    </div>

    <div class="preview-content">
      <!-- Side by side comparison -->
      <div class="comparison-grid">
        <!-- Current Design -->
        <section class="design-section">
          <h2 class="section-title">Current Design</h2>

          <!-- Blocks -->
          <div class="preview-group">
            <h3>Blocks</h3>
            <div class="blocks-grid">
              <div
                v-for="step in stepTypes"
                :key="step.type"
                class="current-block"
                :style="{ borderColor: step.color }"
              >
                <div class="current-block-header" :style="{ backgroundColor: step.color }">
                  <span class="current-block-icon">{{ step.icon }}</span>
                  <span class="current-block-type">{{ step.type.toUpperCase() }}</span>
                </div>
                <div class="current-block-label">{{ step.label }}</div>
              </div>
            </div>
          </div>

          <!-- Groups -->
          <div class="preview-group">
            <h3>Block Groups</h3>
            <div class="groups-grid">
              <div
                v-for="group in groupTypes"
                :key="group.type"
                class="current-group"
                :style="{ borderColor: group.color, '--group-color': group.color }"
              >
                <div class="current-group-header" :style="{ backgroundColor: group.color }">
                  <span class="current-group-icon">{{ group.icon }}</span>
                  <span class="current-group-type">{{ group.type.toUpperCase().replace('_', '-') }}</span>
                  <span class="current-group-name">{{ group.label }}</span>
                </div>
                <div class="current-group-content">
                  <div class="placeholder-block" />
                </div>
              </div>
            </div>
          </div>

          <!-- Status -->
          <div class="preview-group">
            <h3>Status Indicators</h3>
            <div class="status-grid">
              <div
                v-for="status in statusExamples"
                :key="status.status"
                class="current-block current-status"
                :class="`current-status-${status.status}`"
              >
                <div class="current-block-header" style="background-color: #3b82f6;">
                  <span class="current-block-type">TOOL</span>
                </div>
                <div class="current-block-label">{{ status.label }}</div>
                <div class="current-status-badge" :class="`badge-${status.status}`">
                  <span v-if="status.status === 'completed'">✓</span>
                  <span v-else-if="status.status === 'failed'">✕</span>
                  <span v-else>●</span>
                </div>
              </div>
            </div>
          </div>
        </section>

        <!-- New Design -->
        <section class="design-section new-design">
          <h2 class="section-title">Minimal Linear (New)</h2>

          <!-- Blocks -->
          <div class="preview-group">
            <h3>Blocks</h3>
            <div class="blocks-grid">
              <div
                v-for="step in stepTypes"
                :key="step.type"
                class="new-block"
              >
                <div class="new-block-header">
                  <span class="new-block-indicator" :style="{ backgroundColor: step.color }" />
                  <span class="new-block-type">{{ step.type.toUpperCase() }}</span>
                </div>
                <div class="new-block-label">{{ step.label }}</div>
                <div class="new-handle new-handle-left" />
                <div class="new-handle new-handle-right" />
              </div>
            </div>
          </div>

          <!-- Groups -->
          <div class="preview-group">
            <h3>Block Groups</h3>
            <div class="groups-grid">
              <div
                v-for="group in groupTypes"
                :key="group.type"
                class="new-group"
                :style="{ '--group-color': group.color }"
              >
                <div class="new-group-header">
                  <span class="new-group-indicator" :style="{ backgroundColor: group.color }" />
                  <span class="new-group-type">{{ group.type.toUpperCase().replace('_', '-') }}</span>
                  <span class="new-group-name">{{ group.label }}</span>
                </div>
                <div class="new-group-content">
                  <div class="placeholder-block-new" />
                </div>
              </div>
            </div>
          </div>

          <!-- Status -->
          <div class="preview-group">
            <h3>Status Indicators</h3>
            <div class="status-grid">
              <div
                v-for="status in statusExamples"
                :key="status.status"
                class="new-block new-status"
                :class="`new-status-${status.status}`"
              >
                <div class="new-block-header">
                  <span class="new-block-indicator" style="background-color: #3b82f6;" />
                  <span class="new-block-type">TOOL</span>
                </div>
                <div class="new-block-label">{{ status.label }}</div>
              </div>
            </div>
          </div>
        </section>
      </div>

      <!-- Interactive Preview -->
      <section class="interactive-section">
        <h2 class="section-title">Interactive Preview</h2>
        <p class="section-desc">ホバーして操作感を確認してください</p>

        <div class="interactive-canvas">
          <!-- Sample workflow with new design -->
          <div class="canvas-grid">
            <div class="new-block interactive">
              <div class="new-block-header">
                <span class="new-block-indicator" style="background-color: #22c55e;" />
                <span class="new-block-type">START</span>
              </div>
              <div class="new-block-label">Trigger</div>
              <div class="new-handle new-handle-right" />
            </div>

            <svg class="connection-line" width="60" height="2">
              <line x1="0" y1="1" x2="60" y2="1" stroke="#d4d4d4" stroke-width="1.5" />
            </svg>

            <div class="new-block interactive">
              <div class="new-block-header">
                <span class="new-block-indicator" style="background-color: #8b5cf6;" />
                <span class="new-block-type">LLM</span>
              </div>
              <div class="new-block-label">Generate Response</div>
              <div class="new-handle new-handle-left" />
              <div class="new-handle new-handle-right" />
            </div>

            <svg class="connection-line" width="60" height="2">
              <line x1="0" y1="1" x2="60" y2="1" stroke="#d4d4d4" stroke-width="1.5" />
            </svg>

            <div class="new-group interactive" style="--group-color: #f59e0b;">
              <div class="new-group-header">
                <span class="new-group-indicator" style="background-color: #f59e0b;" />
                <span class="new-group-type">IF-ELSE</span>
                <span class="new-group-name">Check Result</span>
              </div>
              <div class="new-group-content">
                <div class="new-block" style="transform: scale(0.85);">
                  <div class="new-block-header">
                    <span class="new-block-indicator" style="background-color: #3b82f6;" />
                    <span class="new-block-type">TOOL</span>
                  </div>
                  <div class="new-block-label">Success</div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- Design Tokens -->
      <section class="tokens-section">
        <h2 class="section-title">Design Tokens</h2>
        <div class="tokens-grid">
          <div class="token-group">
            <h4>Borders</h4>
            <div class="token-item">
              <span class="token-sample border-sample" />
              <span class="token-label">1px solid #e5e5e5</span>
            </div>
            <div class="token-item">
              <span class="token-sample border-hover-sample" />
              <span class="token-label">1px solid #d4d4d4 (hover)</span>
            </div>
          </div>
          <div class="token-group">
            <h4>Backgrounds</h4>
            <div class="token-item">
              <span class="token-sample bg-sample" />
              <span class="token-label">#ffffff (base)</span>
            </div>
            <div class="token-item">
              <span class="token-sample bg-subtle-sample" />
              <span class="token-label">#fafafa (subtle)</span>
            </div>
          </div>
          <div class="token-group">
            <h4>Typography</h4>
            <div class="token-item">
              <span class="token-text-primary">Primary</span>
              <span class="token-label">#171717</span>
            </div>
            <div class="token-item">
              <span class="token-text-secondary">Secondary</span>
              <span class="token-label">#737373</span>
            </div>
          </div>
          <div class="token-group">
            <h4>Radius</h4>
            <div class="token-item">
              <span class="token-sample radius-8" />
              <span class="token-label">8px (blocks)</span>
            </div>
            <div class="token-item">
              <span class="token-sample radius-12" />
              <span class="token-label">12px (groups)</span>
            </div>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.design-preview {
  min-height: 100vh;
  background: #f5f5f5;
  padding: 24px;
}

.preview-header {
  text-align: center;
  margin-bottom: 32px;
}

.preview-header h1 {
  font-size: 28px;
  font-weight: 600;
  color: #171717;
  margin-bottom: 8px;
}

.subtitle {
  font-size: 14px;
  color: #737373;
}

.preview-content {
  max-width: 1400px;
  margin: 0 auto;
}

.comparison-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 24px;
  margin-bottom: 32px;
}

.design-section {
  background: white;
  border-radius: 16px;
  padding: 24px;
  border: 1px solid #e5e5e5;
}

.section-title {
  font-size: 18px;
  font-weight: 600;
  color: #171717;
  margin-bottom: 20px;
  padding-bottom: 12px;
  border-bottom: 1px solid #e5e5e5;
}

.preview-group {
  margin-bottom: 24px;
}

.preview-group h3 {
  font-size: 13px;
  font-weight: 600;
  color: #737373;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-bottom: 12px;
}

.blocks-grid,
.groups-grid,
.status-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

/* ========== CURRENT DESIGN STYLES ========== */

.current-block {
  background: white;
  border: 2px solid;
  border-radius: 8px;
  min-width: 140px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  position: relative;
}

.current-block-header {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border-radius: 6px 6px 0 0;
  color: white;
}

.current-block-icon {
  font-size: 12px;
}

.current-block-type {
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.current-block-label {
  padding: 10px 12px;
  font-size: 13px;
  font-weight: 500;
  color: #1e293b;
}

.current-group {
  background: rgba(0, 0, 0, 0.02);
  border: 2px dashed;
  border-radius: 8px;
  min-width: 200px;
  min-height: 120px;
}

.current-group-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 6px 6px 0 0;
  color: white;
  font-size: 12px;
}

.current-group-icon {
  font-size: 14px;
}

.current-group-type {
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.05em;
}

.current-group-name {
  margin-left: auto;
  font-weight: 600;
}

.current-group-content {
  padding: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.placeholder-block {
  width: 80px;
  height: 50px;
  background: white;
  border: 2px solid #94a3b8;
  border-radius: 6px;
  opacity: 0.5;
}

.current-status {
  border-color: #3b82f6;
}

.current-status-badge {
  position: absolute;
  top: -8px;
  right: -8px;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 11px;
  font-weight: 700;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
}

.badge-running {
  background: #3b82f6;
  animation: pulse 2s ease-in-out infinite;
}

.badge-completed {
  background: #22c55e;
}

.badge-failed {
  background: #ef4444;
}

@keyframes pulse {
  0%, 100% { transform: scale(1); }
  50% { transform: scale(1.1); }
}

/* ========== NEW MINIMAL LINEAR STYLES ========== */

.new-block {
  background: #ffffff;
  border: 1px solid #e5e5e5;
  border-radius: 8px;
  min-width: 140px;
  position: relative;
  transition: border-color 0.15s, background-color 0.15s;
}

.new-block:hover,
.new-block.interactive:hover {
  border-color: #d4d4d4;
  background-color: #fafafa;
}

.new-block.interactive:hover .new-handle {
  opacity: 1;
}

.new-block-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px 4px;
}

.new-block-indicator {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.new-block-type {
  font-size: 11px;
  font-weight: 500;
  color: #737373;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.new-block-label {
  padding: 4px 12px 12px;
  font-size: 14px;
  font-weight: 500;
  color: #171717;
}

.new-handle {
  position: absolute;
  width: 8px;
  height: 8px;
  background: #ffffff;
  border: 1.5px solid #d4d4d4;
  border-radius: 50%;
  top: 50%;
  transform: translateY(-50%);
  opacity: 0;
  transition: opacity 0.15s, border-color 0.15s, background-color 0.15s;
}

.new-handle-left {
  left: -4px;
}

.new-handle-right {
  right: -4px;
}

.new-block:hover .new-handle {
  opacity: 1;
}

.new-handle:hover {
  border-color: #3b82f6;
  background: #3b82f6;
}

.new-group {
  background: rgba(0, 0, 0, 0.01);
  border: 1px solid #e5e5e5;
  border-radius: 12px;
  min-width: 200px;
  min-height: 120px;
  transition: border-color 0.15s;
}

.new-group:hover,
.new-group.interactive:hover {
  border-color: #d4d4d4;
}

.new-group-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border-bottom: 1px solid #e5e5e5;
  background: rgba(0, 0, 0, 0.02);
  border-radius: 12px 12px 0 0;
}

.new-group-indicator {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.new-group-type {
  font-size: 11px;
  font-weight: 600;
  color: #737373;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.new-group-name {
  font-size: 13px;
  font-weight: 500;
  color: #171717;
  margin-left: auto;
}

.new-group-content {
  padding: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.placeholder-block-new {
  width: 80px;
  height: 50px;
  background: #fafafa;
  border: 1px dashed #d4d4d4;
  border-radius: 6px;
}

/* Status - Left border accent */
.new-status-running {
  border-color: #3b82f6;
  animation: pulse-border 2s ease-in-out infinite;
}

.new-status-completed {
  border-left: 3px solid #22c55e;
}

.new-status-failed {
  border-left: 3px solid #ef4444;
}

@keyframes pulse-border {
  0%, 100% { box-shadow: 0 0 0 0 rgba(59, 130, 246, 0.3); }
  50% { box-shadow: 0 0 0 3px rgba(59, 130, 246, 0); }
}

/* ========== INTERACTIVE SECTION ========== */

.interactive-section {
  background: white;
  border-radius: 16px;
  padding: 24px;
  border: 1px solid #e5e5e5;
  margin-bottom: 32px;
}

.section-desc {
  font-size: 13px;
  color: #737373;
  margin-bottom: 20px;
}

.interactive-canvas {
  background: #fafafa;
  background-image: radial-gradient(circle, #e5e5e5 1px, transparent 1px);
  background-size: 24px 24px;
  border-radius: 12px;
  padding: 40px;
  min-height: 200px;
}

.canvas-grid {
  display: flex;
  align-items: center;
  gap: 0;
}

.connection-line {
  margin: 0 -4px;
}

/* ========== TOKENS SECTION ========== */

.tokens-section {
  background: white;
  border-radius: 16px;
  padding: 24px;
  border: 1px solid #e5e5e5;
}

.tokens-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 24px;
}

.token-group h4 {
  font-size: 12px;
  font-weight: 600;
  color: #737373;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-bottom: 12px;
}

.token-item {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 8px;
}

.token-sample {
  width: 32px;
  height: 32px;
  border-radius: 6px;
}

.border-sample {
  background: white;
  border: 1px solid #e5e5e5;
}

.border-hover-sample {
  background: white;
  border: 1px solid #d4d4d4;
}

.bg-sample {
  background: #ffffff;
  border: 1px solid #e5e5e5;
}

.bg-subtle-sample {
  background: #fafafa;
  border: 1px solid #e5e5e5;
}

.radius-8 {
  background: #e5e5e5;
  border-radius: 8px;
}

.radius-12 {
  background: #e5e5e5;
  border-radius: 12px;
}

.token-label {
  font-size: 12px;
  color: #737373;
  font-family: 'SF Mono', Monaco, monospace;
}

.token-text-primary {
  font-size: 14px;
  font-weight: 500;
  color: #171717;
}

.token-text-secondary {
  font-size: 14px;
  color: #737373;
}

/* Responsive */
@media (max-width: 1200px) {
  .comparison-grid {
    grid-template-columns: 1fr;
  }

  .tokens-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}
</style>
