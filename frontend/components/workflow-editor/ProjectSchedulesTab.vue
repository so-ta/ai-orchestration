<script setup lang="ts">
/**
 * ProjectSchedulesTab.vue
 * プロジェクト内のスケジュール管理タブ
 *
 * 機能:
 * - スケジュール一覧表示
 * - スケジュール作成/編集/削除
 * - Start Step選択（どのStartブロックを実行するか）
 */

import type { Schedule, ScheduleStatus, CreateScheduleRequest, UpdateScheduleRequest, Step } from '~/types/api'

const { t } = useI18n()
const schedulesApi = useSchedules()
const toast = useToast()
const { confirm } = useConfirm()

const props = defineProps<{
  projectId: string
  steps?: Step[]
}>()

// State
const schedules = ref<Schedule[]>([])
const loading = ref(false)
const showModal = ref(false)
const editingSchedule = ref<Schedule | null>(null)

// Form
const formData = ref({
  name: '',
  description: '',
  start_step_id: '',
  cron_expression: '',
  timezone: 'Asia/Tokyo',
  input: '{}',
})

// Get Start blocks from steps
const startBlocks = computed(() => {
  return props.steps?.filter(step => step.type === 'start') || []
})

// Fetch schedules for this project
async function fetchSchedules() {
  loading.value = true
  try {
    const result = await schedulesApi.list()
    // Filter by project ID
    schedules.value = result.filter(s => s.project_id === props.projectId)
  } catch {
    toast.error(t('errors.loadFailed'))
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchSchedules()
})

// Watch for project changes
watch(() => props.projectId, () => {
  fetchSchedules()
})

// Cron examples
const cronExamples = [
  { cron: '* * * * *', label: t('schedules.cronExamples.everyMinute') },
  { cron: '0 * * * *', label: t('schedules.cronExamples.everyHour') },
  { cron: '0 9 * * *', label: t('schedules.cronExamples.everyDay9am') },
  { cron: '0 9 * * 1', label: t('schedules.cronExamples.everyMonday') },
  { cron: '0 0 1 * *', label: t('schedules.cronExamples.firstOfMonth') },
]

// Timezone options
const timezoneOptions = [
  'Asia/Tokyo',
  'America/New_York',
  'America/Los_Angeles',
  'Europe/London',
  'Europe/Paris',
  'UTC',
]

// Status colors
function getStatusColor(status: ScheduleStatus): string {
  switch (status) {
    case 'active':
      return '#22c55e'
    case 'paused':
      return '#f59e0b'
    case 'disabled':
      return '#6b7280'
    default:
      return '#6b7280'
  }
}

// Format date
function formatDate(date: string | undefined): string {
  if (!date) return '-'
  return new Date(date).toLocaleString()
}

// Get step name by ID
function getStartStepName(stepId: string): string {
  const step = startBlocks.value.find(s => s.id === stepId)
  return step?.name || stepId.slice(0, 8)
}

// Open create modal
function openCreateModal() {
  editingSchedule.value = null
  formData.value = {
    name: '',
    description: '',
    start_step_id: startBlocks.value[0]?.id || '',
    cron_expression: '0 9 * * *',
    timezone: 'Asia/Tokyo',
    input: '{}',
  }
  showModal.value = true
}

// Open edit modal
function openEditModal(schedule: Schedule) {
  editingSchedule.value = schedule
  formData.value = {
    name: schedule.name,
    description: schedule.description || '',
    start_step_id: schedule.start_step_id,
    cron_expression: schedule.cron_expression,
    timezone: schedule.timezone,
    input: schedule.input ? JSON.stringify(schedule.input, null, 2) : '{}',
  }
  showModal.value = true
}

// Close modal
function closeModal() {
  showModal.value = false
  editingSchedule.value = null
}

// Handle form submit
async function handleSubmit() {
  try {
    let parsedInput = {}
    try {
      parsedInput = JSON.parse(formData.value.input)
    } catch {
      toast.error(t('schedules.messages.invalidJson'))
      return
    }

    if (editingSchedule.value) {
      // Update
      const updateData: UpdateScheduleRequest = {
        name: formData.value.name,
        description: formData.value.description || undefined,
        cron_expression: formData.value.cron_expression,
        timezone: formData.value.timezone,
        input: parsedInput,
      }
      await schedulesApi.update(editingSchedule.value.id, updateData)
      toast.success(t('schedules.messages.updated'))
    } else {
      // Create
      const createData: CreateScheduleRequest = {
        project_id: props.projectId,
        start_step_id: formData.value.start_step_id,
        name: formData.value.name,
        description: formData.value.description || undefined,
        cron_expression: formData.value.cron_expression,
        timezone: formData.value.timezone,
        input: parsedInput,
      }
      await schedulesApi.create(createData)
      toast.success(t('schedules.messages.created'))
    }

    closeModal()
    fetchSchedules()
  } catch (e) {
    const action = editingSchedule.value ? 'updateFailed' : 'createFailed'
    toast.error(t(`schedules.messages.${action}`), e instanceof Error ? e.message : undefined)
  }
}

// Delete schedule
async function handleDelete(schedule: Schedule) {
  const confirmed = await confirm({
    title: t('schedules.deleteTitle'),
    message: t('schedules.confirmDelete'),
    confirmText: t('common.delete'),
    cancelText: t('common.cancel'),
    variant: 'danger',
  })

  if (confirmed) {
    try {
      await schedulesApi.remove(schedule.id)
      toast.success(t('schedules.messages.deleted'))
      fetchSchedules()
    } catch (e) {
      toast.error(t('schedules.messages.deleteFailed'), e instanceof Error ? e.message : undefined)
    }
  }
}

// Pause/Resume schedule
async function toggleStatus(schedule: Schedule) {
  try {
    if (schedule.status === 'active') {
      await schedulesApi.pause(schedule.id)
      toast.success(t('schedules.messages.paused'))
    } else {
      await schedulesApi.resume(schedule.id)
      toast.success(t('schedules.messages.resumed'))
    }
    fetchSchedules()
  } catch (e) {
    const action = schedule.status === 'active' ? 'pauseFailed' : 'resumeFailed'
    toast.error(t(`schedules.messages.${action}`), e instanceof Error ? e.message : undefined)
  }
}

// Apply cron example
function applyCron(cron: string) {
  formData.value.cron_expression = cron
}
</script>

<template>
  <div class="schedules-tab">
    <!-- Header -->
    <div class="tab-header">
      <h3 class="tab-title">{{ t('schedules.title') }}</h3>
      <button class="btn btn-primary btn-sm" @click="openCreateModal">
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="12" y1="5" x2="12" y2="19" />
          <line x1="5" y1="12" x2="19" y2="12" />
        </svg>
        {{ t('schedules.newSchedule') }}
      </button>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="loading-state">
      <span>{{ t('common.loading') }}</span>
    </div>

    <!-- Empty state -->
    <div v-else-if="schedules.length === 0" class="empty-state">
      <svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <circle cx="12" cy="12" r="10" />
        <polyline points="12 6 12 12 16 14" />
      </svg>
      <p class="empty-title">{{ t('schedules.noSchedules') }}</p>
      <p class="empty-desc">{{ t('schedules.noSchedulesDesc') }}</p>
    </div>

    <!-- Schedules list -->
    <div v-else class="schedules-list">
      <div v-for="schedule in schedules" :key="schedule.id" class="schedule-card">
        <div class="schedule-header">
          <div class="schedule-name">{{ schedule.name }}</div>
          <div
            class="schedule-status"
            :style="{ backgroundColor: getStatusColor(schedule.status) + '20', color: getStatusColor(schedule.status) }"
          >
            {{ t(`schedules.status.${schedule.status}`) }}
          </div>
        </div>

        <div class="schedule-info">
          <div class="info-row">
            <span class="info-label">{{ t('schedules.cronExpression') }}:</span>
            <code class="info-value">{{ schedule.cron_expression }}</code>
          </div>
          <div class="info-row">
            <span class="info-label">{{ t('workflows.tabs.start') }}:</span>
            <span class="info-value">{{ getStartStepName(schedule.start_step_id) }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">{{ t('schedules.nextRun') }}:</span>
            <span class="info-value">{{ formatDate(schedule.next_run_at) }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">{{ t('schedules.lastRun') }}:</span>
            <span class="info-value">{{ formatDate(schedule.last_run_at) }}</span>
          </div>
        </div>

        <div class="schedule-actions">
          <button
            class="btn btn-ghost btn-sm"
            :title="schedule.status === 'active' ? t('schedules.actions.pause') : t('schedules.actions.resume')"
            @click="toggleStatus(schedule)"
          >
            <svg v-if="schedule.status === 'active'" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="6" y="4" width="4" height="16" />
              <rect x="14" y="4" width="4" height="16" />
            </svg>
            <svg v-else xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polygon points="5 3 19 12 5 21 5 3" />
            </svg>
          </button>
          <button class="btn btn-ghost btn-sm" :title="t('common.edit')" @click="openEditModal(schedule)">
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
              <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
            </svg>
          </button>
          <button class="btn btn-ghost btn-sm btn-danger" :title="t('common.delete')" @click="handleDelete(schedule)">
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="3 6 5 6 21 6" />
              <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
            </svg>
          </button>
        </div>
      </div>
    </div>

    <!-- Modal -->
    <div v-if="showModal" class="modal-overlay" @click.self="closeModal">
      <div class="modal-content">
        <div class="modal-header">
          <h4>{{ editingSchedule ? t('schedules.editSchedule') : t('schedules.newSchedule') }}</h4>
          <button class="btn btn-ghost btn-sm" @click="closeModal">
            <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </div>

        <form class="modal-body" @submit.prevent="handleSubmit">
          <div class="form-group">
            <label class="form-label">{{ t('schedules.form.name') }}</label>
            <input
              v-model="formData.name"
              type="text"
              class="form-input"
              :placeholder="t('schedules.form.namePlaceholder')"
              required
            >
          </div>

          <div class="form-group">
            <label class="form-label">{{ t('schedules.form.description') }}</label>
            <textarea
              v-model="formData.description"
              class="form-input"
              rows="2"
              :placeholder="t('schedules.form.descriptionPlaceholder')"
            />
          </div>

          <div v-if="!editingSchedule && startBlocks.length > 1" class="form-group">
            <label class="form-label">{{ t('workflows.tabs.start') }} (Start Block)</label>
            <select v-model="formData.start_step_id" class="form-input" required>
              <option v-for="step in startBlocks" :key="step.id" :value="step.id">
                {{ step.name }}
              </option>
            </select>
          </div>

          <div class="form-group">
            <label class="form-label">{{ t('schedules.form.cronExpression') }}</label>
            <input
              v-model="formData.cron_expression"
              type="text"
              class="form-input font-mono"
              :placeholder="t('schedules.form.cronExpressionPlaceholder')"
              required
            >
            <p class="form-hint">{{ t('schedules.form.cronHint') }}</p>
            <div class="cron-examples">
              <button
                v-for="example in cronExamples"
                :key="example.cron"
                type="button"
                class="cron-example-btn"
                @click="applyCron(example.cron)"
              >
                {{ example.label }}
              </button>
            </div>
          </div>

          <div class="form-group">
            <label class="form-label">{{ t('schedules.form.timezone') }}</label>
            <select v-model="formData.timezone" class="form-input">
              <option v-for="tz in timezoneOptions" :key="tz" :value="tz">{{ tz }}</option>
            </select>
          </div>

          <div class="form-group">
            <label class="form-label">{{ t('schedules.form.input') }}</label>
            <textarea
              v-model="formData.input"
              class="form-input font-mono"
              rows="4"
              :placeholder="t('schedules.form.inputPlaceholder')"
            />
          </div>

          <div class="modal-actions">
            <button type="button" class="btn btn-ghost" @click="closeModal">
              {{ t('common.cancel') }}
            </button>
            <button type="submit" class="btn btn-primary">
              {{ editingSchedule ? t('common.save') : t('common.create') }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<style scoped>
.schedules-tab {
  padding: 1rem;
}

.tab-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.tab-title {
  font-size: 1rem;
  font-weight: 600;
  color: var(--color-text);
  margin: 0;
}

.loading-state,
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 3rem 1rem;
  text-align: center;
  color: var(--color-text-secondary);
}

.empty-state svg {
  margin-bottom: 1rem;
  opacity: 0.5;
}

.empty-title {
  font-weight: 500;
  margin: 0 0 0.5rem;
}

.empty-desc {
  font-size: 0.875rem;
  margin: 0;
}

.schedules-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.schedule-card {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 8px;
  padding: 1rem;
}

.schedule-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.75rem;
}

.schedule-name {
  font-weight: 500;
  color: var(--color-text);
}

.schedule-status {
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 500;
}

.schedule-info {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  margin-bottom: 0.75rem;
}

.info-row {
  display: flex;
  gap: 0.5rem;
  font-size: 0.875rem;
}

.info-label {
  color: var(--color-text-secondary);
}

.info-value {
  color: var(--color-text);
}

.schedule-actions {
  display: flex;
  gap: 0.5rem;
  border-top: 1px solid var(--color-border);
  padding-top: 0.75rem;
}

/* Modal styles */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-content {
  background: var(--color-surface);
  border-radius: 12px;
  width: 100%;
  max-width: 480px;
  max-height: 90vh;
  overflow-y: auto;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 1.5rem;
  border-bottom: 1px solid var(--color-border);
}

.modal-header h4 {
  margin: 0;
  font-size: 1rem;
  font-weight: 600;
}

.modal-body {
  padding: 1.5rem;
}

.form-group {
  margin-bottom: 1rem;
}

.form-label {
  display: block;
  margin-bottom: 0.5rem;
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--color-text);
}

.form-input {
  width: 100%;
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  background: var(--color-bg);
  color: var(--color-text);
  font-size: 0.875rem;
}

.form-input:focus {
  outline: none;
  border-color: var(--color-primary);
}

.form-hint {
  margin-top: 0.25rem;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

.font-mono {
  font-family: 'SF Mono', Monaco, monospace;
}

.cron-examples {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-top: 0.5rem;
}

.cron-example-btn {
  padding: 0.25rem 0.5rem;
  font-size: 0.75rem;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  background: var(--color-bg);
  color: var(--color-text-secondary);
  cursor: pointer;
}

.cron-example-btn:hover {
  border-color: var(--color-primary);
  color: var(--color-primary);
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  margin-top: 1.5rem;
}

/* Buttons */
.btn {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  border: none;
  border-radius: 6px;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: background-color 0.15s, color 0.15s;
}

.btn-sm {
  padding: 0.375rem 0.75rem;
  font-size: 0.8125rem;
}

.btn-primary {
  background: var(--color-primary);
  color: white;
}

.btn-primary:hover {
  background: var(--color-primary-dark);
}

.btn-ghost {
  background: transparent;
  color: var(--color-text-secondary);
}

.btn-ghost:hover {
  background: var(--color-bg-hover);
  color: var(--color-text);
}

.btn-danger {
  color: #ef4444;
}

.btn-danger:hover {
  background: rgba(239, 68, 68, 0.1);
}
</style>
