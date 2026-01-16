<script setup lang="ts">
/**
 * BlockTestRunner - ブロックテスト実行コンポーネント
 *
 * ブロックのコードをテスト入力で実行し、結果を確認するためのUI。
 */
import type { BlockDefinition } from '~/types/api'

interface BlockFormData {
  slug: string
  name: string
  description: string
  category: string
  icon: string
  code: string
  config_schema: string
  ui_config: string
  change_summary: string
  parent_block_id?: string
  config_defaults?: string
  pre_process?: string
  post_process?: string
}

const props = defineProps<{
  blockData: BlockFormData
  parentBlock?: BlockDefinition | null
  useInheritance?: boolean
}>()

const { t } = useI18n()

// State
const testInput = ref('{\n  "message": "Hello, World!"\n}')
const testConfig = ref('{}')
const testResult = ref<{
  success: boolean
  output?: unknown
  error?: string
  logs?: Array<{ level: string; message: string }>
  executionTime?: number
} | null>(null)
const isRunning = ref(false)

// Initialize config from schema
onMounted(() => {
  initializeConfig()
})

watch(() => props.blockData.config_schema, () => {
  initializeConfig()
})

function initializeConfig() {
  try {
    const schema = JSON.parse(props.blockData.config_schema || '{}')
    const defaultConfig: Record<string, unknown> = {}

    if (schema.properties) {
      for (const [key, prop] of Object.entries(schema.properties)) {
        const propDef = prop as { default?: unknown }
        if (propDef.default !== undefined) {
          defaultConfig[key] = propDef.default
        }
      }
    }

    // Merge with config_defaults if using inheritance
    if (props.useInheritance && props.blockData.config_defaults) {
      try {
        const defaults = JSON.parse(props.blockData.config_defaults)
        Object.assign(defaultConfig, defaults)
      } catch {
        // Ignore parse error
      }
    }

    testConfig.value = JSON.stringify(defaultConfig, null, 2)
  } catch {
    testConfig.value = '{}'
  }
}

// Run test (mock implementation - would need backend API)
async function runTest() {
  isRunning.value = true
  testResult.value = null

  try {
    // Validate JSON inputs
    let inputData: unknown
    let configData: unknown

    try {
      inputData = JSON.parse(testInput.value)
    } catch {
      throw new Error(t('blockEditor.test.invalidInput'))
    }

    try {
      configData = JSON.parse(testConfig.value)
    } catch {
      throw new Error(t('blockEditor.test.invalidConfig'))
    }

    // Simulate test execution
    // In a real implementation, this would call the backend API
    await new Promise(resolve => setTimeout(resolve, 500))

    // Mock result
    testResult.value = {
      success: true,
      output: {
        message: 'Test execution successful',
        input: inputData,
        config: configData,
        timestamp: new Date().toISOString(),
      },
      logs: [
        { level: 'info', message: 'Block execution started' },
        { level: 'info', message: 'Processing input data' },
        { level: 'info', message: 'Block execution completed' },
      ],
      executionTime: 123,
    }
  } catch (error) {
    testResult.value = {
      success: false,
      error: error instanceof Error ? error.message : String(error),
    }
  } finally {
    isRunning.value = false
  }
}

// Clear result
function clearResult() {
  testResult.value = null
}
</script>

<template>
  <div class="block-test-runner">
    <div class="test-inputs">
      <!-- Test Input -->
      <div class="input-section">
        <label class="input-label">{{ t('blockEditor.test.input') }}</label>
        <textarea
          v-model="testInput"
          class="code-input"
          rows="6"
          spellcheck="false"
          :placeholder="t('blockEditor.test.inputPlaceholder')"
        />
      </div>

      <!-- Test Config -->
      <div class="input-section">
        <label class="input-label">{{ t('blockEditor.test.config') }}</label>
        <textarea
          v-model="testConfig"
          class="code-input"
          rows="6"
          spellcheck="false"
          :placeholder="t('blockEditor.test.configPlaceholder')"
        />
      </div>
    </div>

    <!-- Run Button -->
    <div class="test-actions">
      <button
        class="btn btn-primary"
        :disabled="isRunning"
        @click="runTest"
      >
        <span v-if="isRunning">{{ t('blockEditor.test.running') }}</span>
        <span v-else>{{ t('blockEditor.test.run') }}</span>
      </button>
      <button
        v-if="testResult"
        class="btn btn-secondary"
        @click="clearResult"
      >
        {{ t('blockEditor.test.clear') }}
      </button>
    </div>

    <!-- Test Result -->
    <div v-if="testResult" class="test-result" :class="{ success: testResult.success, error: !testResult.success }">
      <div class="result-header">
        <span class="result-icon">{{ testResult.success ? '&#10003;' : '&#10007;' }}</span>
        <span class="result-title">
          {{ testResult.success ? t('blockEditor.test.success') : t('blockEditor.test.failed') }}
        </span>
        <span v-if="testResult.executionTime" class="result-time">
          {{ testResult.executionTime }}ms
        </span>
      </div>

      <!-- Error -->
      <div v-if="testResult.error" class="result-error">
        {{ testResult.error }}
      </div>

      <!-- Output -->
      <div v-if="testResult.output" class="result-output">
        <label class="output-label">{{ t('blockEditor.test.output') }}</label>
        <pre class="output-content">{{ JSON.stringify(testResult.output, null, 2) }}</pre>
      </div>

      <!-- Logs -->
      <div v-if="testResult.logs && testResult.logs.length > 0" class="result-logs">
        <label class="output-label">{{ t('blockEditor.test.logs') }}</label>
        <div class="logs-list">
          <div
            v-for="(log, index) in testResult.logs"
            :key="index"
            class="log-entry"
            :class="'log-' + log.level"
          >
            <span class="log-level">{{ log.level }}</span>
            <span class="log-message">{{ log.message }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Info -->
    <div class="test-info">
      <p>{{ t('blockEditor.test.info') }}</p>
    </div>
  </div>
</template>

<style scoped>
.block-test-runner {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.test-inputs {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
}

.input-section {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.input-label {
  font-size: 0.875rem;
  font-weight: 500;
}

.code-input {
  width: 100%;
  padding: 0.75rem;
  border: 1px solid var(--color-border);
  border-radius: 0.375rem;
  background: var(--color-background);
  color: var(--color-text);
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 0.8125rem;
  line-height: 1.5;
  resize: vertical;
}

.code-input:focus {
  outline: none;
  border-color: var(--color-primary);
}

.test-actions {
  display: flex;
  gap: 0.75rem;
}

.btn {
  padding: 0.5rem 1rem;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
}

.btn-primary {
  background: var(--color-primary);
  color: white;
  border: none;
}

.btn-primary:hover:not(:disabled) {
  opacity: 0.9;
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-secondary {
  background: var(--color-surface);
  color: var(--color-text);
  border: 1px solid var(--color-border);
}

.btn-secondary:hover {
  background: var(--color-background);
}

.test-result {
  border: 1px solid var(--color-border);
  border-radius: 0.5rem;
  overflow: hidden;
}

.test-result.success {
  border-color: rgba(34, 197, 94, 0.3);
}

.test-result.error {
  border-color: rgba(239, 68, 68, 0.3);
}

.result-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 1rem;
  background: var(--color-background);
  border-bottom: 1px solid var(--color-border);
}

.test-result.success .result-header {
  background: rgba(34, 197, 94, 0.05);
}

.test-result.error .result-header {
  background: rgba(239, 68, 68, 0.05);
}

.result-icon {
  font-size: 1rem;
}

.test-result.success .result-icon {
  color: #16a34a;
}

.test-result.error .result-icon {
  color: #ef4444;
}

.result-title {
  font-weight: 600;
  font-size: 0.875rem;
}

.result-time {
  margin-left: auto;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

.result-error {
  padding: 0.75rem 1rem;
  background: rgba(239, 68, 68, 0.05);
  color: #ef4444;
  font-size: 0.875rem;
}

.result-output,
.result-logs {
  padding: 1rem;
}

.output-label {
  display: block;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--color-text-secondary);
  text-transform: uppercase;
  margin-bottom: 0.5rem;
}

.output-content {
  margin: 0;
  padding: 0.75rem;
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: 0.375rem;
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 0.75rem;
  line-height: 1.5;
  overflow-x: auto;
}

.logs-list {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.log-entry {
  display: flex;
  gap: 0.5rem;
  padding: 0.375rem 0.5rem;
  background: var(--color-background);
  border-radius: 0.25rem;
  font-size: 0.8125rem;
}

.log-level {
  font-weight: 600;
  text-transform: uppercase;
  font-size: 0.6875rem;
  padding: 0.125rem 0.25rem;
  border-radius: 0.125rem;
}

.log-info .log-level {
  background: rgba(59, 130, 246, 0.1);
  color: #3b82f6;
}

.log-warn .log-level {
  background: rgba(245, 158, 11, 0.1);
  color: #f59e0b;
}

.log-error .log-level {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
}

.log-debug .log-level {
  background: rgba(107, 114, 128, 0.1);
  color: #6b7280;
}

.log-message {
  color: var(--color-text);
}

.test-info {
  padding: 0.75rem;
  background: rgba(99, 102, 241, 0.05);
  border: 1px solid rgba(99, 102, 241, 0.1);
  border-radius: 0.375rem;
}

.test-info p {
  margin: 0;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}
</style>
