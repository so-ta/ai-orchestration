<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'

const usage = useUsage()

// Period selection
const selectedPeriod = ref('month')
const periodOptions = [
  { value: 'day', label: 'Today' },
  { value: 'month', label: 'This Month' },
]

// Loading state
const loading = ref(true)
const error = ref<string | null>(null)

// Load dashboard data
async function loadData() {
  try {
    loading.value = true
    error.value = null
    await usage.fetchDashboardData(selectedPeriod.value)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load usage data'
  } finally {
    loading.value = false
  }
}

// Watch for period changes
watch(selectedPeriod, () => {
  loadData()
})

onMounted(() => {
  loadData()
})

// Format currency
function formatCurrency(value: number): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 2,
    maximumFractionDigits: 4,
  }).format(value)
}

// Format number with K/M suffix
function formatNumber(value: number): string {
  if (value >= 1000000) {
    return (value / 1000000).toFixed(1) + 'M'
  }
  if (value >= 1000) {
    return (value / 1000).toFixed(1) + 'K'
  }
  return value.toString()
}

// Format date
function formatDate(date: string): string {
  return new Date(date).toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
  })
}

// Budget status color
const budgetStatusColor = computed(() => {
  if (!usage.budgetStatus.value) return 'bg-gray-500'
  const consumed = usage.budgetStatus.value.consumed
  if (consumed >= 100) return 'bg-red-500'
  if (consumed >= 80) return 'bg-yellow-500'
  return 'bg-green-500'
})

// Budget status text
const budgetStatusText = computed(() => {
  if (!usage.budgetStatus.value) return 'No budget set'
  const { consumed, limit } = usage.budgetStatus.value
  if (consumed >= 100) return 'Budget exceeded'
  if (consumed >= 80) return 'Approaching limit'
  return 'On track'
})

// Model data for chart
const modelChartData = computed(() => {
  return Object.entries(usage.modelData.value).map(([model, data]) => ({
    name: model,
    value: data.cost_usd,
    provider: data.provider,
  }))
})

// Daily data for chart
const dailyChartData = computed(() => {
  return usage.dailyData.value.map(d => ({
    date: formatDate(d.date),
    cost: d.total_cost_usd,
    requests: d.total_requests,
  }))
})
</script>

<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-900">
    <!-- Header -->
    <div class="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
        <div class="flex items-center justify-between">
          <div>
            <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Usage & Cost Tracking</h1>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              Monitor your LLM API usage and manage budgets
            </p>
          </div>
          <div class="flex items-center gap-4">
            <!-- Period selector -->
            <select
              v-model="selectedPeriod"
              class="block rounded-md border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white shadow-sm focus:border-blue-500 focus:ring-blue-500"
            >
              <option v-for="opt in periodOptions" :key="opt.value" :value="opt.value">
                {{ opt.label }}
              </option>
            </select>
            <!-- Refresh button -->
            <button
              @click="loadData"
              :disabled="loading"
              class="inline-flex items-center px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm text-sm font-medium text-gray-700 dark:text-gray-200 bg-white dark:bg-gray-700 hover:bg-gray-50 dark:hover:bg-gray-600 disabled:opacity-50"
            >
              <svg v-if="loading" class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
              Refresh
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Error message -->
    <div v-if="error" class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
      <div class="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md p-4">
        <p class="text-sm text-red-700 dark:text-red-400">{{ error }}</p>
      </div>
    </div>

    <!-- Main content -->
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <!-- Summary Cards -->
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        <!-- Total Cost -->
        <div class="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
          <div class="flex items-center">
            <div class="flex-shrink-0">
              <div class="w-10 h-10 rounded-full bg-blue-100 dark:bg-blue-900 flex items-center justify-center">
                <svg class="w-6 h-6 text-blue-600 dark:text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              </div>
            </div>
            <div class="ml-4">
              <p class="text-sm font-medium text-gray-500 dark:text-gray-400">Total Cost</p>
              <p class="text-2xl font-semibold text-gray-900 dark:text-white">
                {{ usage.summary.value ? formatCurrency(usage.summary.value.total_cost_usd) : '$0.00' }}
              </p>
            </div>
          </div>
        </div>

        <!-- Total Requests -->
        <div class="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
          <div class="flex items-center">
            <div class="flex-shrink-0">
              <div class="w-10 h-10 rounded-full bg-green-100 dark:bg-green-900 flex items-center justify-center">
                <svg class="w-6 h-6 text-green-600 dark:text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 12l3-3 3 3 4-4M8 21l4-4 4 4M3 4h18M4 4h16v12a1 1 0 01-1 1H5a1 1 0 01-1-1V4z" />
                </svg>
              </div>
            </div>
            <div class="ml-4">
              <p class="text-sm font-medium text-gray-500 dark:text-gray-400">Total Requests</p>
              <p class="text-2xl font-semibold text-gray-900 dark:text-white">
                {{ usage.summary.value ? formatNumber(usage.summary.value.total_requests) : '0' }}
              </p>
            </div>
          </div>
        </div>

        <!-- Total Tokens -->
        <div class="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
          <div class="flex items-center">
            <div class="flex-shrink-0">
              <div class="w-10 h-10 rounded-full bg-purple-100 dark:bg-purple-900 flex items-center justify-center">
                <svg class="w-6 h-6 text-purple-600 dark:text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 7h6m0 10v-3m-3 3h.01M9 17h.01M9 14h.01M12 14h.01M15 11h.01M12 11h.01M9 11h.01M7 21h10a2 2 0 002-2V5a2 2 0 00-2-2H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
                </svg>
              </div>
            </div>
            <div class="ml-4">
              <p class="text-sm font-medium text-gray-500 dark:text-gray-400">Total Tokens</p>
              <p class="text-2xl font-semibold text-gray-900 dark:text-white">
                {{ usage.summary.value ? formatNumber(usage.summary.value.total_input_tokens + usage.summary.value.total_output_tokens) : '0' }}
              </p>
            </div>
          </div>
        </div>

        <!-- Budget Status -->
        <div class="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
          <div class="flex items-center">
            <div class="flex-shrink-0">
              <div class="w-10 h-10 rounded-full" :class="budgetStatusColor.replace('bg-', 'bg-opacity-20 ')">
                <div class="w-full h-full rounded-full flex items-center justify-center" :class="budgetStatusColor.replace('bg-', 'text-').replace('-500', '-600')">
                  <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
                  </svg>
                </div>
              </div>
            </div>
            <div class="ml-4">
              <p class="text-sm font-medium text-gray-500 dark:text-gray-400">Budget Status</p>
              <p class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ budgetStatusText }}
              </p>
              <p v-if="usage.budgetStatus.value" class="text-sm text-gray-500 dark:text-gray-400">
                {{ usage.budgetStatus.value.consumed.toFixed(1) }}% used
              </p>
            </div>
          </div>
        </div>
      </div>

      <!-- Charts Row -->
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
        <!-- Daily Cost Chart -->
        <div class="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
          <h3 class="text-lg font-medium text-gray-900 dark:text-white mb-4">Daily Cost Trend</h3>
          <div v-if="dailyChartData.length > 0" class="h-64">
            <div class="flex items-end justify-between h-48 gap-1">
              <div
                v-for="(day, index) in dailyChartData"
                :key="index"
                class="flex-1 flex flex-col items-center"
              >
                <div
                  class="w-full bg-blue-500 rounded-t transition-all duration-300 hover:bg-blue-600"
                  :style="{ height: `${Math.max((day.cost / Math.max(...dailyChartData.map(d => d.cost))) * 100, 5)}%` }"
                  :title="`${day.date}: ${formatCurrency(day.cost)}`"
                ></div>
                <span class="text-xs text-gray-500 dark:text-gray-400 mt-2 truncate w-full text-center">
                  {{ day.date }}
                </span>
              </div>
            </div>
          </div>
          <div v-else class="h-64 flex items-center justify-center text-gray-500 dark:text-gray-400">
            No data available
          </div>
        </div>

        <!-- Model Breakdown -->
        <div class="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
          <h3 class="text-lg font-medium text-gray-900 dark:text-white mb-4">Cost by Model</h3>
          <div v-if="usage.topModels.value.length > 0" class="space-y-3">
            <div
              v-for="model in usage.topModels.value"
              :key="model.model"
              class="flex items-center"
            >
              <div class="flex-1">
                <div class="flex items-center justify-between mb-1">
                  <span class="text-sm font-medium text-gray-900 dark:text-white">{{ model.model }}</span>
                  <span class="text-sm text-gray-500 dark:text-gray-400">{{ formatCurrency(model.cost_usd) }}</span>
                </div>
                <div class="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                  <div
                    class="bg-blue-500 h-2 rounded-full transition-all duration-300"
                    :style="{ width: `${(model.cost_usd / usage.topModels.value[0].cost_usd) * 100}%` }"
                  ></div>
                </div>
              </div>
            </div>
          </div>
          <div v-else class="h-48 flex items-center justify-center text-gray-500 dark:text-gray-400">
            No data available
          </div>
        </div>
      </div>

      <!-- Tables Row -->
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <!-- Top Workflows -->
        <div class="bg-white dark:bg-gray-800 rounded-lg shadow">
          <div class="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
            <h3 class="text-lg font-medium text-gray-900 dark:text-white">Top Workflows by Cost</h3>
          </div>
          <div class="overflow-x-auto">
            <table v-if="usage.topProjects.value.length > 0" class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
              <thead class="bg-gray-50 dark:bg-gray-900">
                <tr>
                  <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Workflow</th>
                  <th class="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Cost</th>
                  <th class="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Requests</th>
                </tr>
              </thead>
              <tbody class="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
                <tr v-for="wf in usage.topProjects.value" :key="wf.project_id">
                  <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">
                    {{ wf.project_name }}
                  </td>
                  <td class="px-6 py-4 whitespace-nowrap text-sm text-right text-gray-500 dark:text-gray-400">
                    {{ formatCurrency(wf.total_cost_usd) }}
                  </td>
                  <td class="px-6 py-4 whitespace-nowrap text-sm text-right text-gray-500 dark:text-gray-400">
                    {{ formatNumber(wf.total_requests) }}
                  </td>
                </tr>
              </tbody>
            </table>
            <div v-else class="px-6 py-12 text-center text-gray-500 dark:text-gray-400">
              No workflow data available
            </div>
          </div>
        </div>

        <!-- Budgets -->
        <div class="bg-white dark:bg-gray-800 rounded-lg shadow">
          <div class="px-6 py-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
            <h3 class="text-lg font-medium text-gray-900 dark:text-white">Budget Settings</h3>
            <NuxtLink
              to="/usage/budgets"
              class="text-sm link-primary"
            >
              Manage
            </NuxtLink>
          </div>
          <div class="overflow-x-auto">
            <table v-if="usage.budgets.value.length > 0" class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
              <thead class="bg-gray-50 dark:bg-gray-900">
                <tr>
                  <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Type</th>
                  <th class="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Limit</th>
                  <th class="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Alert At</th>
                  <th class="px-6 py-3 text-center text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Status</th>
                </tr>
              </thead>
              <tbody class="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
                <tr v-for="budget in usage.budgets.value" :key="budget.id">
                  <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white capitalize">
                    {{ budget.budget_type }}
                  </td>
                  <td class="px-6 py-4 whitespace-nowrap text-sm text-right text-gray-500 dark:text-gray-400">
                    {{ formatCurrency(budget.budget_amount_usd) }}
                  </td>
                  <td class="px-6 py-4 whitespace-nowrap text-sm text-right text-gray-500 dark:text-gray-400">
                    {{ (budget.alert_threshold * 100).toFixed(0) }}%
                  </td>
                  <td class="px-6 py-4 whitespace-nowrap text-center">
                    <span
                      class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium"
                      :class="budget.enabled ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200' : 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300'"
                    >
                      {{ budget.enabled ? 'Active' : 'Disabled' }}
                    </span>
                  </td>
                </tr>
              </tbody>
            </table>
            <div v-else class="px-6 py-12 text-center text-gray-500 dark:text-gray-400">
              <p>No budgets configured</p>
              <NuxtLink
                to="/usage/budgets"
                class="mt-2 inline-block link-primary"
              >
                Set up a budget
              </NuxtLink>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
