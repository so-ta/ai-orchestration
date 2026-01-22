<script setup lang="ts">
import { ref, computed } from 'vue'
import type { Step } from '~/types/api'
import SchemaTree, { type SchemaNode } from './SchemaTree.vue'

const props = defineProps<{
  previousSteps: Step[]
  testResults?: Record<string, unknown>
}>()

const emit = defineEmits<{
  (e: 'insert', path: string): void
}>()

// Collapsed state per step
const collapsedSteps = ref<Set<string>>(new Set())

// Parse output schema from step config
function parseOutputSchema(step: Step): SchemaNode | null {
  const config = step.config as Record<string, unknown> | undefined
  if (!config) return null

  const outputSchema = config.output_schema as { type?: string; properties?: Record<string, unknown>; required?: string[] } | undefined
  if (!outputSchema || outputSchema.type !== 'object' || !outputSchema.properties) {
    // Return generic output node
    return {
      name: 'output',
      type: 'object',
      path: `$.steps.${step.name}.output`,
      children: []
    }
  }

  function parseProperties(
    properties: Record<string, unknown>,
    parentPath: string,
    required: string[] = []
  ): SchemaNode[] {
    const nodes: SchemaNode[] = []

    for (const [name, propDef] of Object.entries(properties)) {
      const prop = propDef as { type?: string; description?: string; properties?: Record<string, unknown>; items?: { properties?: Record<string, unknown> }; required?: string[] }
      const path = `${parentPath}.${name}`
      const node: SchemaNode = {
        name,
        type: prop.type || 'any',
        description: prop.description,
        path,
        required: required.includes(name)
      }

      // Handle nested objects
      if (prop.type === 'object' && prop.properties) {
        node.children = parseProperties(prop.properties, path, prop.required)
      }

      // Handle arrays with object items
      if (prop.type === 'array' && prop.items?.properties) {
        node.children = parseProperties(prop.items.properties, `${path}[0]`, [])
      }

      nodes.push(node)
    }

    return nodes
  }

  return {
    name: 'output',
    type: 'object',
    path: `$.steps.${step.name}.output`,
    children: parseProperties(outputSchema.properties, `$.steps.${step.name}.output`, outputSchema.required)
  }
}

// Build schema tree for all previous steps
const schemaNodes = computed<Array<{ step: Step; schema: SchemaNode }>>(() => {
  const nodes: Array<{ step: Step; schema: SchemaNode }> = []

  for (const step of props.previousSteps) {
    const schema = parseOutputSchema(step)
    if (schema) {
      nodes.push({ step, schema })
    }
  }

  return nodes
})

function toggleStep(stepId: string) {
  if (collapsedSteps.value.has(stepId)) {
    collapsedSteps.value.delete(stepId)
  } else {
    collapsedSteps.value.add(stepId)
  }
  collapsedSteps.value = new Set(collapsedSteps.value)
}

function formatTemplateVariable(path: string): string {
  const openBrace = String.fromCharCode(123, 123)
  const closeBrace = String.fromCharCode(125, 125)
  return openBrace + path + closeBrace
}

function handleSelect(path: string) {
  emit('insert', path)
}

function handleDragStart(event: DragEvent, path: string) {
  const template = formatTemplateVariable(path)
  event.dataTransfer?.setData('text/plain', template)
  event.dataTransfer!.effectAllowed = 'copy'
}
</script>

<template>
  <div v-if="schemaNodes.length > 0" class="output-schema-preview">
    <h4 class="section-title">
      <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"/>
        <polyline points="22,6 12,13 2,6"/>
      </svg>
      前ステップの出力スキーマ
    </h4>

    <div class="schema-list">
      <div
        v-for="{ step, schema } in schemaNodes"
        :key="step.id"
        class="step-schema"
      >
        <button
          class="step-header"
          @click="toggleStep(step.id)"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="10"
            height="10"
            viewBox="0 0 24 24"
            fill="currentColor"
            :class="{ expanded: !collapsedSteps.has(step.id) }"
          >
            <path d="M8 5v14l11-7z"/>
          </svg>
          <span class="step-name">{{ step.name }}</span>
          <span class="step-type">{{ step.type }}</span>
        </button>

        <div v-if="!collapsedSteps.has(step.id)" class="step-schema-content">
          <SchemaTree
            :schema="schema"
            :draggable="true"
            @select="handleSelect"
            @drag-start="handleDragStart"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.output-schema-preview {
  background: linear-gradient(135deg, #eff6ff 0%, #dbeafe 100%);
  border: 1px solid #93c5fd;
  border-radius: 8px;
  padding: 0.875rem;
  margin-top: 0.5rem;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  font-size: 0.75rem;
  font-weight: 600;
  color: #1e40af;
  margin: 0 0 0.75rem 0;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.section-title svg {
  color: #3b82f6;
}

.schema-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  background: rgba(255, 255, 255, 0.7);
  border-radius: 6px;
  padding: 0.5rem;
  max-height: 300px;
  overflow-y: auto;
}

.step-schema {
  display: flex;
  flex-direction: column;
}

.step-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 8px;
  background: rgba(255, 255, 255, 0.8);
  border: 1px solid #bfdbfe;
  border-radius: 4px;
  cursor: pointer;
  font-size: 11px;
  font-weight: 500;
  color: #1e40af;
  transition: background-color 0.1s;
}

.step-header:hover {
  background: white;
}

.step-header svg {
  color: #3b82f6;
  transition: transform 0.15s;
}

.step-header svg.expanded {
  transform: rotate(90deg);
}

.step-name {
  flex: 1;
}

.step-type {
  font-size: 10px;
  color: #3b82f6;
  background: #dbeafe;
  padding: 2px 6px;
  border-radius: 3px;
}

.step-schema-content {
  margin-top: 4px;
  padding: 8px;
  background: white;
  border-radius: 4px;
  border: 1px solid #e0e7ff;
}

.schema-list::-webkit-scrollbar {
  width: 6px;
}

.schema-list::-webkit-scrollbar-track {
  background: transparent;
}

.schema-list::-webkit-scrollbar-thumb {
  background: #93c5fd;
  border-radius: 3px;
}
</style>
