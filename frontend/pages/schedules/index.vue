<script setup lang="ts">
import type { Schedule, ScheduleStatus, CreateScheduleRequest, UpdateScheduleRequest, Workflow } from '~/types/api'

const { t } = useI18n()
const schedulesApi = useSchedules()
const toast = useToast()

// State
const schedules = ref<Schedule[]>([])
const workflows = ref<Workflow[]>([])
const loading = ref(false)
const showModal = ref(false)
const editingSchedule = ref<Schedule | null>(null)

// Filters
const filterStatus = ref<ScheduleStatus | ''>('')
const filterWorkflow = ref('')

// Form
const formData = ref({
  name: '',
  description: '',
  workflow_id: '',
  cron_expression: '',
  timezone: 'Asia/Tokyo',
  input: '{}',
})

// Fetch data
const fetchSchedules = async () => {
  loading.value = true
  try {
    const result = await schedulesApi.list()
    schedules.value = result
  } catch (e) {
    toast.error(t('errors.loadFailed'))
  } finally {
    loading.value = false
  }
}

const fetchWorkflows = async () => {
  try {
    const config = useRuntimeConfig()
    const baseUrl = config.public.apiBase || 'http://localhost:8080'
    const tenantId = import.meta.client
      ? (localStorage.getItem('tenant_id') || '00000000-0000-0000-0000-000000000001')
      : '00000000-0000-0000-0000-000000000001'
    const response = await fetch(`${baseUrl}/workflows`, {
      headers: {
        'Content-Type': 'application/json',
        'X-Tenant-ID': tenantId,
      },
    })
    if (response.ok) {
      const result = await response.json()
      workflows.value = result.data
    }
  } catch (e) {
    console.error('Failed to fetch workflows:', e)
  }
}

onMounted(() => {
  fetchSchedules()
  fetchWorkflows()
})

// Computed
const filteredSchedules = computed(() => {
  return schedules.value.filter((s) => {
    if (filterStatus.value && s.status !== filterStatus.value) return false
    if (filterWorkflow.value && s.workflow_id !== filterWorkflow.value) return false
    return true
  })
})

const statusOptions: ScheduleStatus[] = ['active', 'paused', 'disabled']

const cronExamples = [
  { cron: '* * * * *', label: 'everyMinute' },
  { cron: '0 * * * *', label: 'everyHour' },
  { cron: '0 9 * * *', label: 'everyDay9am' },
  { cron: '0 9 * * 1', label: 'everyMonday' },
  { cron: '0 0 1 * *', label: 'firstOfMonth' },
]

// Methods
const getWorkflowName = (workflowId: string): string => {
  const workflow = workflows.value.find((w) => w.id === workflowId)
  return workflow?.name || workflowId
}

const getStatusColor = (status: ScheduleStatus): string => {
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

const formatDate = (date: string | undefined): string => {
  if (!date) return '-'
  return new Date(date).toLocaleString()
}

const openCreateModal = () => {
  editingSchedule.value = null
  formData.value = {
    name: '',
    description: '',
    workflow_id: '',
    cron_expression: '',
    timezone: 'Asia/Tokyo',
    input: '{}',
  }
  showModal.value = true
}

const openEditModal = (schedule: Schedule) => {
  editingSchedule.value = schedule
  formData.value = {
    name: schedule.name,
    description: schedule.description || '',
    workflow_id: schedule.workflow_id,
    cron_expression: schedule.cron_expression,
    timezone: schedule.timezone,
    input: schedule.input ? JSON.stringify(schedule.input, null, 2) : '{}',
  }
  showModal.value = true
}

const handleSubmit = async () => {
  try {
    let parsedInput = {}
    try {
      parsedInput = JSON.parse(formData.value.input)
    } catch {
      toast.error(t('schedules.messages.invalidJson'))
      return
    }

    if (editingSchedule.value) {
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
      const createData: CreateScheduleRequest = {
        workflow_id: formData.value.workflow_id,
        name: formData.value.name,
        description: formData.value.description || undefined,
        cron_expression: formData.value.cron_expression,
        timezone: formData.value.timezone,
        input: parsedInput,
      }
      await schedulesApi.create(createData)
      toast.success(t('schedules.messages.created'))
    }

    showModal.value = false
    fetchSchedules()
  } catch (e) {
    toast.error(
      editingSchedule.value
        ? t('schedules.messages.updateFailed')
        : t('schedules.messages.createFailed'),
    )
  }
}

const handleDelete = async (schedule: Schedule) => {
  if (!confirm(t('schedules.confirmDelete'))) return

  try {
    await schedulesApi.remove(schedule.id)
    toast.success(t('schedules.messages.deleted'))
    fetchSchedules()
  } catch (e) {
    toast.error(t('schedules.messages.deleteFailed'))
  }
}

const handlePause = async (schedule: Schedule) => {
  try {
    await schedulesApi.pause(schedule.id)
    toast.success(t('schedules.messages.paused'))
    fetchSchedules()
  } catch (e) {
    toast.error(t('schedules.messages.pauseFailed'))
  }
}

const handleResume = async (schedule: Schedule) => {
  try {
    await schedulesApi.resume(schedule.id)
    toast.success(t('schedules.messages.resumed'))
    fetchSchedules()
  } catch (e) {
    toast.error(t('schedules.messages.resumeFailed'))
  }
}

const applyCronExample = (cron: string) => {
  formData.value.cron_expression = cron
}
</script>

<template>
  <div>
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ $t('schedules.title') }}</h1>
        <p class="text-secondary">{{ $t('schedules.subtitle') }}</p>
      </div>
      <button class="btn btn-primary" @click="openCreateModal">
        + {{ $t('schedules.newSchedule') }}
      </button>
    </div>

    <!-- Filters -->
    <div class="filters-bar">
      <select v-model="filterStatus" class="filter-select">
        <option value="">{{ $t('schedules.allStatuses') }}</option>
        <option v-for="status in statusOptions" :key="status" :value="status">
          {{ $t(`schedules.status.${status}`) }}
        </option>
      </select>
      <select v-model="filterWorkflow" class="filter-select">
        <option value="">{{ $t('schedules.allWorkflows') }}</option>
        <option v-for="wf in workflows" :key="wf.id" :value="wf.id">
          {{ wf.name }}
        </option>
      </select>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="loading-state">
      {{ $t('common.loading') }}
    </div>

    <!-- Empty State -->
    <div v-else-if="filteredSchedules.length === 0" class="empty-state">
      <p>{{ $t('schedules.noSchedules') }}</p>
      <p class="text-secondary">{{ $t('schedules.noSchedulesDesc') }}</p>
    </div>

    <!-- Schedules Table -->
    <div v-else class="table-container">
      <table class="data-table">
        <thead>
          <tr>
            <th>{{ $t('schedules.table.name') }}</th>
            <th>{{ $t('schedules.table.workflow') }}</th>
            <th>{{ $t('schedules.table.cron') }}</th>
            <th>{{ $t('schedules.table.status') }}</th>
            <th>{{ $t('schedules.table.nextRun') }}</th>
            <th>{{ $t('schedules.table.lastRun') }}</th>
            <th>{{ $t('schedules.table.runCount') }}</th>
            <th>{{ $t('schedules.table.actions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="schedule in filteredSchedules" :key="schedule.id">
            <td>
              <div class="schedule-name">{{ schedule.name }}</div>
              <div v-if="schedule.description" class="schedule-desc text-secondary">
                {{ schedule.description }}
              </div>
            </td>
            <td>
              <NuxtLink :to="`/workflows/${schedule.workflow_id}`" class="workflow-link">
                {{ getWorkflowName(schedule.workflow_id) }}
              </NuxtLink>
            </td>
            <td>
              <code class="cron-code">{{ schedule.cron_expression }}</code>
            </td>
            <td>
              <span class="status-badge" :style="{ backgroundColor: getStatusColor(schedule.status) }">
                {{ $t(`schedules.status.${schedule.status}`) }}
              </span>
            </td>
            <td>{{ formatDate(schedule.next_run_at) }}</td>
            <td>{{ formatDate(schedule.last_run_at) }}</td>
            <td>{{ schedule.run_count }}</td>
            <td>
              <div class="action-buttons">
                <button v-if="schedule.status === 'active'" class="btn btn-sm btn-secondary" @click="handlePause(schedule)">
                  {{ $t('schedules.actions.pause') }}
                </button>
                <button v-else class="btn btn-sm btn-secondary" @click="handleResume(schedule)">
                  {{ $t('schedules.actions.resume') }}
                </button>
                <button class="btn btn-sm btn-secondary" @click="openEditModal(schedule)">
                  {{ $t('common.edit') }}
                </button>
                <button class="btn btn-sm btn-danger" @click="handleDelete(schedule)">
                  {{ $t('common.delete') }}
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Modal -->
    <div v-if="showModal" class="modal-overlay" @click.self="showModal = false">
      <div class="modal-content">
        <div class="modal-header">
          <h2>{{ editingSchedule ? $t('schedules.editSchedule') : $t('schedules.newSchedule') }}</h2>
          <button class="modal-close" @click="showModal = false">&times;</button>
        </div>
        <form @submit.prevent="handleSubmit">
          <div class="form-group">
            <label>{{ $t('schedules.form.name') }}</label>
            <input
              v-model="formData.name"
              type="text"
              class="form-input"
              :placeholder="$t('schedules.form.namePlaceholder')"
              required
            />
          </div>

          <div class="form-group">
            <label>{{ $t('schedules.form.description') }}</label>
            <textarea
              v-model="formData.description"
              class="form-input"
              :placeholder="$t('schedules.form.descriptionPlaceholder')"
              rows="2"
            />
          </div>

          <div v-if="!editingSchedule" class="form-group">
            <label>{{ $t('schedules.form.workflow') }}</label>
            <select v-model="formData.workflow_id" class="form-input" required>
              <option value="">{{ $t('schedules.form.workflowPlaceholder') }}</option>
              <option v-for="wf in workflows" :key="wf.id" :value="wf.id">
                {{ wf.name }}
              </option>
            </select>
          </div>

          <div class="form-group">
            <label>{{ $t('schedules.form.cronExpression') }}</label>
            <input
              v-model="formData.cron_expression"
              type="text"
              class="form-input"
              :placeholder="$t('schedules.form.cronExpressionPlaceholder')"
              required
            />
            <p class="form-hint">{{ $t('schedules.form.cronHint') }}</p>
            <div class="cron-examples">
              <span class="cron-examples-title">{{ $t('schedules.cronExamples.title') }}:</span>
              <button
                v-for="example in cronExamples"
                :key="example.cron"
                type="button"
                class="cron-example-btn"
                @click="applyCronExample(example.cron)"
              >
                {{ example.cron }} ({{ $t(`schedules.cronExamples.${example.label}`) }})
              </button>
            </div>
          </div>

          <div class="form-group">
            <label>{{ $t('schedules.form.timezone') }}</label>
            <select v-model="formData.timezone" class="form-input">
              <option value="Asia/Tokyo">Asia/Tokyo</option>
              <option value="UTC">UTC</option>
              <option value="America/New_York">America/New_York</option>
              <option value="Europe/London">Europe/London</option>
            </select>
          </div>

          <div class="form-group">
            <label>{{ $t('schedules.form.input') }}</label>
            <textarea
              v-model="formData.input"
              class="form-input code-input"
              :placeholder="$t('schedules.form.inputPlaceholder')"
              rows="4"
            />
          </div>

          <div class="modal-actions">
            <button type="button" class="btn btn-secondary" @click="showModal = false">
              {{ $t('common.cancel') }}
            </button>
            <button type="submit" class="btn btn-primary">
              {{ editingSchedule ? $t('common.save') : $t('common.create') }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 1.5rem;
}

.page-title {
  font-size: 1.5rem;
  font-weight: 600;
  margin: 0;
}

.filters-bar {
  display: flex;
  gap: 1rem;
  margin-bottom: 1rem;
}

.filter-select {
  padding: 0.5rem 1rem;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  background: var(--color-surface);
  min-width: 160px;
}

.loading-state,
.empty-state {
  text-align: center;
  padding: 3rem;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
}

.table-container {
  overflow-x: auto;
}

.data-table {
  width: 100%;
  border-collapse: collapse;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
}

.data-table th,
.data-table td {
  padding: 0.75rem 1rem;
  text-align: left;
  border-bottom: 1px solid var(--color-border);
}

.data-table th {
  background: var(--color-background);
  font-weight: 600;
  font-size: 0.875rem;
}

.schedule-name {
  font-weight: 500;
}

.schedule-desc {
  font-size: 0.75rem;
  margin-top: 0.25rem;
}

.workflow-link {
  color: var(--color-primary);
  text-decoration: none;
}

.workflow-link:hover {
  text-decoration: underline;
}

.cron-code {
  background: var(--color-background);
  padding: 0.25rem 0.5rem;
  border-radius: var(--radius-sm);
  font-size: 0.875rem;
}

.status-badge {
  display: inline-block;
  padding: 0.25rem 0.5rem;
  border-radius: var(--radius-sm);
  color: white;
  font-size: 0.75rem;
  font-weight: 500;
}

.action-buttons {
  display: flex;
  gap: 0.5rem;
}

.btn {
  padding: 0.5rem 1rem;
  border: none;
  border-radius: var(--radius);
  cursor: pointer;
  font-size: 0.875rem;
}

.btn-primary {
  background: var(--color-primary);
  color: white;
}

.btn-secondary {
  background: var(--color-background);
  border: 1px solid var(--color-border);
}

.btn-danger {
  background: #ef4444;
  color: white;
}

.btn-sm {
  padding: 0.25rem 0.5rem;
  font-size: 0.75rem;
}

/* Modal */
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
  border-radius: var(--radius);
  width: 90%;
  max-width: 600px;
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

.modal-header h2 {
  margin: 0;
  font-size: 1.25rem;
}

.modal-close {
  background: none;
  border: none;
  font-size: 1.5rem;
  cursor: pointer;
  color: var(--color-text-secondary);
}

.modal-content form {
  padding: 1.5rem;
}

.form-group {
  margin-bottom: 1rem;
}

.form-group label {
  display: block;
  margin-bottom: 0.5rem;
  font-weight: 500;
}

.form-input {
  width: 100%;
  padding: 0.5rem;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  background: var(--color-background);
}

.form-hint {
  margin-top: 0.25rem;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

.code-input {
  font-family: monospace;
  font-size: 0.875rem;
}

.cron-examples {
  margin-top: 0.5rem;
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  align-items: center;
}

.cron-examples-title {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

.cron-example-btn {
  padding: 0.25rem 0.5rem;
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: 0.75rem;
  cursor: pointer;
}

.cron-example-btn:hover {
  background: var(--color-surface);
}

.modal-actions {
  display: flex;
  gap: 1rem;
  justify-content: flex-end;
  margin-top: 1.5rem;
}
</style>
