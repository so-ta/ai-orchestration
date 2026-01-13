<script setup lang="ts">
import {
  Chart,
  BarController,
  LineController,
  PieController,
  DoughnutController,
  CategoryScale,
  LinearScale,
  BarElement,
  LineElement,
  PointElement,
  ArcElement,
  Title,
  Tooltip,
  Legend,
} from 'chart.js'
import type { ChartConfig } from '~/types/rich-view'

// Register Chart.js components
Chart.register(
  BarController,
  LineController,
  PieController,
  DoughnutController,
  CategoryScale,
  LinearScale,
  BarElement,
  LineElement,
  PointElement,
  ArcElement,
  Title,
  Tooltip,
  Legend,
)

const props = defineProps<{
  config: ChartConfig
}>()

const canvasRef = ref<HTMLCanvasElement | null>(null)
let chartInstance: Chart | null = null

// Default color palette
const defaultColors = [
  '#3b82f6', // blue
  '#10b981', // green
  '#f59e0b', // amber
  '#ef4444', // red
  '#8b5cf6', // purple
  '#ec4899', // pink
  '#06b6d4', // cyan
  '#84cc16', // lime
]

function getChartData() {
  return {
    labels: props.config.labels,
    datasets: props.config.datasets.map((dataset, index) => ({
      label: dataset.label || `Dataset ${index + 1}`,
      data: dataset.data,
      backgroundColor: dataset.color || defaultColors[index % defaultColors.length],
      borderColor: dataset.color || defaultColors[index % defaultColors.length],
      borderWidth: props.config.type === 'line' ? 2 : 0,
      fill: false,
      tension: 0.1,
    })),
  }
}

function getChartOptions() {
  const isPie = props.config.type === 'pie' || props.config.type === 'doughnut'

  const baseOptions = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: {
        display: props.config.showLegend !== false,
        position: (isPie ? 'right' : 'top') as 'right' | 'top',
      },
    },
  }

  if (isPie) {
    return baseOptions
  }

  return {
    ...baseOptions,
    scales: {
      x: {
        stacked: props.config.stacked || false,
        grid: {
          display: false,
        },
      },
      y: {
        stacked: props.config.stacked || false,
        beginAtZero: true,
        grid: {
          color: 'rgba(0, 0, 0, 0.05)',
        },
      },
    },
  }
}

function createChart() {
  if (!canvasRef.value) return

  // Destroy existing chart if any
  if (chartInstance) {
    chartInstance.destroy()
    chartInstance = null
  }

  const ctx = canvasRef.value.getContext('2d')
  if (!ctx) return

  chartInstance = new Chart(ctx, {
    type: props.config.type,
    data: getChartData(),
    options: getChartOptions(),
  })
}

// Watch for config changes
watch(
  () => props.config,
  () => {
    createChart()
  },
  { deep: true },
)

onMounted(() => {
  createChart()
})

onBeforeUnmount(() => {
  if (chartInstance) {
    chartInstance.destroy()
    chartInstance = null
  }
})
</script>

<template>
  <div class="chart-block">
    <div
      class="chart-container"
      :style="{ height: `${config.height || 300}px` }"
    >
      <canvas ref="canvasRef" />
    </div>
  </div>
</template>

<style scoped>
.chart-block {
  margin: 1rem 0;
  padding: 1rem;
  background: white;
  border: 1px solid var(--color-border);
  border-radius: 8px;
}

.chart-container {
  position: relative;
  width: 100%;
}
</style>
