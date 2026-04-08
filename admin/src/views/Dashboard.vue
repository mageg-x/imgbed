<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import request from '@/api/request'
import { ElMessage } from 'element-plus'
import * as echarts from 'echarts'
import {
  Folder, Monitor, Upload, Cloud, RefreshCw, TrendingUp, CheckCircle, BarChart3, PieChart, LineChart, Calendar
} from 'lucide-vue-next'

const { t } = useI18n()
const router = useRouter()
const loading = ref(false)
const isDark = ref(true)

const dashboard = ref({
  totalFiles: 0,
  totalSize: 0,
  todayUploads: 0,
  todaySize: 0,
  enabledChannels: 0,
  channelStatuses: []
})

const stats = ref({
  overview: null,
  channels: [],
  trend: [],
  weekly: []
})

// 图表 DOM 引用
const pieChartRef = ref(null)
const lineChartRef = ref(null)
const barChartRef = ref(null)
const weeklyChartRef = ref(null)

let pieChart = null
let lineChart = null
let barChart = null
let weeklyChart = null

onMounted(() => {
  isDark.value = !document.documentElement.classList.contains('light')
  loadDashboard()
  loadStats()
})

onUnmounted(() => {
  if (pieChart) {
    pieChart.dispose()
    pieChart = null
  }
  if (lineChart) {
    lineChart.dispose()
    lineChart = null
  }
  if (barChart) {
    barChart.dispose()
    barChart = null
  }
  if (weeklyChart) {
    weeklyChart.dispose()
    weeklyChart = null
  }
  window.removeEventListener('resize', handleResize)
})

function handleResize() {
  pieChart?.resize()
  lineChart?.resize()
  barChart?.resize()
  weeklyChart?.resize()
}

async function loadDashboard() {
  loading.value = true
  try {
    const res = await request.get('/admin/dashboard')
    if (res.code === 0) {
      dashboard.value = res.data
    }
  } catch {
    ElMessage.error(t('common.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function loadStats() {
  try {
    // 并行加载四个统计接口
    const [overviewRes, channelsRes, trendRes, weeklyRes] = await Promise.all([
      request.get('/stats/overview').catch(() => ({ code: 0, data: null })),
      request.get('/stats/channels').catch(() => ({ code: 0, data: { items: [] } })),
      request.get('/stats/trend').catch(() => ({ code: 0, data: { items: [] } })),
      request.get('/stats/weekly').catch(() => ({ code: 0, data: { items: [] } }))
    ])

    if (overviewRes.code === 0) {
      stats.value.overview = overviewRes.data
    }
    if (channelsRes.code === 0) {
      stats.value.channels = channelsRes.data?.items || []
    }
    if (trendRes.code === 0) {
      stats.value.trend = trendRes.data?.items || []
    }
    if (weeklyRes.code === 0) {
      stats.value.weekly = weeklyRes.data?.items || []
    }

    // 初始化图表
    initCharts()
  } catch (err) {
    console.error('Failed to load stats', err)
  }
}

function initCharts() {
  const textColor = isDark.value ? '#9ca3af' : '#6b7280'
  const bgColor = isDark.value ? '#1f2937' : '#ffffff'

  // 渠道使用占比饼图
  if (pieChartRef.value && stats.value.channels.length > 0) {
    pieChart = echarts.init(pieChartRef.value)
    const pieData = stats.value.channels.map((ch, i) => ({
      name: ch.channelName || ch.channelId,
      value: ch.totalUploads || 0
    }))

    pieChart.setOption({
      backgroundColor: bgColor,
      tooltip: { trigger: 'item' },
      legend: {
        orient: 'vertical',
        left: 'left',
        textStyle: { color: textColor }
      },
      series: [{
        name: t('dashboard.channelStats'),
        type: 'pie',
        radius: ['40%', '70%'],
        avoidLabelOverlap: false,
        itemStyle: { borderRadius: 10, borderColor: bgColor, borderWidth: 2 },
        label: { show: false },
        emphasis: {
          label: { show: true, fontSize: 14, fontWeight: 'bold' }
        },
        labelLine: { show: false },
        data: pieData,
        color: ['#6366f1', '#8b5cf6', '#d946ef', '#f43f5e', '#f97316', '#eab308', '#22c55e', '#14b8a6']
      }]
    })
  }

  // 每日上传趋势折线图
  if (lineChartRef.value && stats.value.trend.length > 0) {
    lineChart = echarts.init(lineChartRef.value)
    const dates = stats.value.trend.map(t => t.date)
    const uploads = stats.value.trend.map(t => t.uploads)
    const success = stats.value.trend.map(t => t.success)

    lineChart.setOption({
      backgroundColor: bgColor,
      tooltip: { trigger: 'axis' },
      legend: {
        data: [t('dashboard.uploads') || 'Uploads', t('dashboard.success') || 'Success'],
        textStyle: { color: textColor }
      },
      grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
      xAxis: {
        type: 'category',
        boundaryGap: false,
        data: dates,
        axisLine: { lineStyle: { color: textColor } },
        axisLabel: { color: textColor }
      },
      yAxis: {
        type: 'value',
        axisLine: { lineStyle: { color: textColor } },
        axisLabel: { color: textColor },
        splitLine: { lineStyle: { color: isDark.value ? '#374151' : '#e5e7eb' } }
      },
      series: [
        {
          name: t('dashboard.uploads') || 'Uploads',
          type: 'line',
          smooth: true,
          lineStyle: { width: 2 },
          areaStyle: { opacity: 0.2 },
          data: uploads,
          itemStyle: { color: '#6366f1' }
        },
        {
          name: t('dashboard.success') || 'Success',
          type: 'line',
          smooth: true,
          lineStyle: { width: 2 },
          areaStyle: { opacity: 0.2 },
          data: success,
          itemStyle: { color: '#22c55e' }
        }
      ]
    })
  }

  // 各渠道成功率柱状图
  if (barChartRef.value && stats.value.channels.length > 0) {
    barChart = echarts.init(barChartRef.value)
    const channels = stats.value.channels.map(ch => ch.channelName || ch.channelId)
    const successRates = stats.value.channels.map(ch => {
      if (ch.totalUploads > 0) {
        return ((ch.successCount || ch.totalUploads) / ch.totalUploads * 100).toFixed(1)
      }
      return 0
    })

    barChart.setOption({
      backgroundColor: bgColor,
      tooltip: { trigger: 'axis', formatter: '{b}: {c}%' },
      grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
      xAxis: {
        type: 'category',
        data: channels,
        axisLine: { lineStyle: { color: textColor } },
        axisLabel: { color: textColor, rotate: 30 }
      },
      yAxis: {
        type: 'value',
        max: 100,
        axisLine: { lineStyle: { color: textColor } },
        axisLabel: { color: textColor, formatter: '{value}%' },
        splitLine: { lineStyle: { color: isDark.value ? '#374151' : '#e5e7eb' } }
      },
      series: [{
        name: t('dashboard.successRate') || 'Success Rate',
        type: 'bar',
        barWidth: '60%',
        data: successRates,
        itemStyle: {
          color: (params) => {
            const rate = params.value
            if (rate >= 90) return '#22c55e'
            if (rate >= 70) return '#eab308'
            return '#ef4444'
          },
          borderRadius: [4, 4, 0, 0]
        },
        label: { show: true, position: 'top', formatter: '{c}%', color: textColor }
      }]
    })
  }

  // 每周上传统计柱状图
  if (weeklyChartRef.value && stats.value.weekly.length > 0) {
    weeklyChart = echarts.init(weeklyChartRef.value)
    const weeks = stats.value.weekly.map(w => `${w.weekStart.slice(5)} ~ ${w.weekEnd.slice(5)}`).reverse()
    const uploads = stats.value.weekly.map(w => w.uploads).reverse()
    const success = stats.value.weekly.map(w => w.success).reverse()

    weeklyChart.setOption({
      backgroundColor: bgColor,
      tooltip: { trigger: 'axis' },
      legend: {
        data: [t('dashboard.uploads') || 'Uploads', t('dashboard.success') || 'Success'],
        textStyle: { color: textColor }
      },
      grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
      xAxis: {
        type: 'category',
        data: weeks,
        axisLine: { lineStyle: { color: textColor } },
        axisLabel: { color: textColor, rotate: 45, fontSize: 10 }
      },
      yAxis: {
        type: 'value',
        axisLine: { lineStyle: { color: textColor } },
        axisLabel: { color: textColor },
        splitLine: { lineStyle: { color: isDark.value ? '#374151' : '#e5e7eb' } }
      },
      series: [
        {
          name: t('dashboard.uploads') || 'Uploads',
          type: 'bar',
          barWidth: '40%',
          data: uploads,
          itemStyle: { color: '#6366f1', borderRadius: [4, 4, 0, 0] }
        },
        {
          name: t('dashboard.success') || 'Success',
          type: 'bar',
          barWidth: '40%',
          data: success,
          itemStyle: { color: '#22c55e', borderRadius: [4, 4, 0, 0] }
        }
      ]
    })
  }
}

// 监听窗口变化，重绘图表
window.addEventListener('resize', handleResize)

function formatSize(bytes) {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

function getChannelStatusColor(status) {
  if (status === 'healthy') return 'bg-green-500'
  if (status === 'warning') return 'bg-yellow-500'
  if (status === 'error') return 'bg-red-500'
  return 'bg-gray-500'
}

function getSuccessRateClass(rate) {
  if (rate >= 90) return 'text-green-500'
  if (rate >= 70) return 'text-yellow-500'
  return 'text-red-500'
}
</script>

<template>
  <div class="space-y-6">
    <!-- 统计卡片 -->
    <div class="grid grid-cols-2 lg:grid-cols-4 gap-3 sm:gap-4">
      <div class="card p-4 sm:p-6 hover-lift animate-fade-in">
        <div class="flex items-start justify-between">
          <div>
            <p class="text-xs sm:text-sm font-medium" :class="isDark ? 'text-gray-400' : 'text-gray-500'">{{ t('dashboard.totalFiles') }}</p>
            <p class="text-2xl sm:text-3xl font-bold mt-1">{{ dashboard.totalFiles?.toLocaleString() || 0 }}</p>
          </div>
          <div class="w-10 h-10 sm:w-12 sm:h-12 rounded-xl bg-indigo-500/10 flex items-center justify-center">
            <Folder class="w-5 h-5 sm:w-6 sm:h-6 text-indigo-500" />
          </div>
        </div>
        <div class="mt-2 sm:mt-3 flex items-center gap-1 text-xs sm:text-sm"
          :class="isDark ? 'text-gray-400' : 'text-gray-500'">
          <TrendingUp class="w-3 h-3 sm:w-4 sm:h-4 text-green-500" />
          <span>{{ t('dashboard.allFiles') || 'All Files' }}</span>
        </div>
      </div>

      <div class="card p-4 sm:p-6 hover-lift animate-fade-in delay-100">
        <div class="flex items-start justify-between">
          <div>
            <p class="text-xs sm:text-sm font-medium" :class="isDark ? 'text-gray-400' : 'text-gray-500'">{{ t('dashboard.totalStorage') }}</p>
            <p class="text-2xl sm:text-3xl font-bold mt-1">{{ formatSize(dashboard.totalSize) }}</p>
          </div>
          <div class="w-10 h-10 sm:w-12 sm:h-12 rounded-xl bg-purple-500/10 flex items-center justify-center">
            <Monitor class="w-5 h-5 sm:w-6 sm:h-6 text-purple-500" />
          </div>
        </div>
        <div class="mt-2 sm:mt-3 flex items-center gap-1 text-xs sm:text-sm"
          :class="isDark ? 'text-gray-400' : 'text-gray-500'">
          <TrendingUp class="w-3 h-3 sm:w-4 sm:h-4 text-green-500" />
          <span>{{ t('dashboard.storage') || 'Storage' }}</span>
        </div>
      </div>

      <div class="card p-4 sm:p-6 hover-lift animate-fade-in delay-200">
        <div class="flex items-start justify-between">
          <div>
            <p class="text-xs sm:text-sm font-medium" :class="isDark ? 'text-gray-400' : 'text-gray-500'">{{ t('dashboard.todayUploads') }}</p>
            <p class="text-2xl sm:text-3xl font-bold mt-1">{{ dashboard.todayUploads || 0 }}</p>
          </div>
          <div class="w-10 h-10 sm:w-12 sm:h-12 rounded-xl bg-green-500/10 flex items-center justify-center">
            <Upload class="w-5 h-5 sm:w-6 sm:h-6 text-green-500" />
          </div>
        </div>
        <p class="text-xs sm:text-sm mt-2 sm:mt-3" :class="isDark ? 'text-gray-400' : 'text-gray-500'">
          {{ t('dashboard.bandwidth') || 'Bandwidth' }} {{ formatSize(dashboard.todaySize) }}
        </p>
      </div>

      <div class="card p-4 sm:p-6 hover-lift animate-fade-in delay-300">
        <div class="flex items-start justify-between">
          <div>
            <p class="text-xs sm:text-sm font-medium" :class="isDark ? 'text-gray-400' : 'text-gray-500'">{{ t('dashboard.activeChannels') || 'Active Channels' }}</p>
            <p class="text-2xl sm:text-3xl font-bold mt-1">{{ dashboard.enabledChannels || 0 }}</p>
          </div>
          <div class="w-10 h-10 sm:w-12 sm:h-12 rounded-xl bg-cyan-500/10 flex items-center justify-center">
            <Cloud class="w-5 h-5 sm:w-6 sm:h-6 text-cyan-500" />
          </div>
        </div>
        <p class="text-xs sm:text-sm mt-2 sm:mt-3" :class="isDark ? 'text-gray-400' : 'text-gray-500'">
          {{ t('dashboard.totalChannels') || 'Total' }} {{ dashboard.channelStatuses?.length || 0 }} {{ t('dashboard.channels') || 'channels' }}
        </p>
      </div>
    </div>

    <!-- 成功率统计卡片 -->
    <div class="grid grid-cols-2 lg:grid-cols-4 gap-3 sm:gap-4" v-if="stats.overview">
      <div class="card p-4 sm:p-6 hover-lift animate-fade-in">
        <div class="flex items-start justify-between">
          <div>
            <p class="text-xs sm:text-sm font-medium" :class="isDark ? 'text-gray-400' : 'text-gray-500'">{{ t('dashboard.overallSuccessRate') || 'Success Rate' }}</p>
            <p class="text-2xl sm:text-3xl font-bold mt-1" :class="getSuccessRateClass(stats.overview.successRate)">
              {{ (stats.overview.successRate || 0).toFixed(1) }}%
            </p>
          </div>
          <div class="w-10 h-10 sm:w-12 sm:h-12 rounded-xl bg-green-500/10 flex items-center justify-center">
            <CheckCircle class="w-5 h-5 sm:w-6 sm:h-6 text-green-500" />
          </div>
        </div>
        <div class="mt-2 sm:mt-3">
          <div class="progress-bar">
            <div class="progress" :style="{ width: (stats.overview.successRate || 0) + '%' }"></div>
          </div>
        </div>
      </div>

      <div class="card p-4 sm:p-6 hover-lift animate-fade-in delay-100">
        <div class="flex items-start justify-between">
          <div>
            <p class="text-xs sm:text-sm font-medium" :class="isDark ? 'text-gray-400' : 'text-gray-500'">{{ t('dashboard.totalSuccess') || 'Total Success' }}</p>
            <p class="text-2xl sm:text-3xl font-bold mt-1 text-green-500">{{ (stats.overview.totalSuccess || 0).toLocaleString() }}</p>
          </div>
          <div class="w-10 h-10 sm:w-12 sm:h-12 rounded-xl bg-green-500/10 flex items-center justify-center">
            <CheckCircle class="w-5 h-5 sm:w-6 sm:h-6 text-green-500" />
          </div>
        </div>
        <p class="text-xs sm:text-sm mt-2 sm:mt-3" :class="isDark ? 'text-gray-400' : 'text-gray-500'">
          {{ t('dashboard.successfulUploads') || 'Successful uploads' }}
        </p>
      </div>

      <div class="card p-4 sm:p-6 hover-lift animate-fade-in delay-200">
        <div class="flex items-start justify-between">
          <div>
            <p class="text-xs sm:text-sm font-medium" :class="isDark ? 'text-gray-400' : 'text-gray-500'">{{ t('dashboard.totalFailed') || 'Total Failed' }}</p>
            <p class="text-2xl sm:text-3xl font-bold mt-1 text-red-500">{{ (stats.overview.totalFailed || 0).toLocaleString() }}</p>
          </div>
          <div class="w-10 h-10 sm:w-12 sm:h-12 rounded-xl bg-red-500/10 flex items-center justify-center">
            <BarChart3 class="w-5 h-5 sm:w-6 sm:h-6 text-red-500" />
          </div>
        </div>
        <p class="text-xs sm:text-sm mt-2 sm:mt-3" :class="isDark ? 'text-gray-400' : 'text-gray-500'">
          {{ t('dashboard.failedUploads') || 'Failed uploads' }}
        </p>
      </div>

      <div class="card p-4 sm:p-6 hover-lift animate-fade-in delay-300">
        <div class="flex items-start justify-between">
          <div>
            <p class="text-xs sm:text-sm font-medium" :class="isDark ? 'text-gray-400' : 'text-gray-500'">{{ t('dashboard.totalUploads') || 'Total Uploads' }}</p>
            <p class="text-2xl sm:text-3xl font-bold mt-1">{{ (stats.overview.totalUploads || 0).toLocaleString() }}</p>
          </div>
          <div class="w-10 h-10 sm:w-12 sm:h-12 rounded-xl bg-indigo-500/10 flex items-center justify-center">
            <Upload class="w-5 h-5 sm:w-6 sm:h-6 text-indigo-500" />
          </div>
        </div>
        <p class="text-xs sm:text-sm mt-2 sm:mt-3" :class="isDark ? 'text-gray-400' : 'text-gray-500'">
          {{ t('dashboard.cumulativeUploads') || 'Cumulative uploads' }}
        </p>
      </div>
    </div>

    <!-- 图表区域 -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-4 sm:gap-6">
      <!-- 每日上传趋势折线图 -->
      <div class="card p-4 sm:p-6 animate-fade-in">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-base sm:text-lg font-semibold flex items-center gap-2">
            <LineChart class="w-5 h-5 text-indigo-500" />
            {{ t('dashboard.uploadTrend') }}
          </h2>
          <el-tooltip :content="t('common.refresh')" placement="top">
            <button @click="loadStats" class="p-2 rounded-lg transition-all hover:bg-[var(--bg-hover)]"
              :class="isDark ? 'text-gray-400' : 'text-gray-600'">
              <RefreshCw class="w-4 h-4" />
            </button>
          </el-tooltip>
        </div>
        <div ref="lineChartRef" class="w-full h-64"></div>
      </div>

      <!-- 各渠道成功率柱状图 -->
      <div class="card p-4 sm:p-6 animate-fade-in delay-100">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-base sm:text-lg font-semibold flex items-center gap-2">
            <BarChart3 class="w-5 h-5 text-green-500" />
            {{ t('dashboard.channelSuccessRate') || 'Channel Success Rate' }}
          </h2>
          <el-tooltip :content="t('common.refresh')" placement="top">
            <button @click="loadStats" class="p-2 rounded-lg transition-all hover:bg-[var(--bg-hover)]"
              :class="isDark ? 'text-gray-400' : 'text-gray-600'">
              <RefreshCw class="w-4 h-4" />
            </button>
          </el-tooltip>
        </div>
        <div ref="barChartRef" class="w-full h-64"></div>
      </div>
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-3 gap-4 sm:gap-6">
      <!-- 渠道使用占比饼图 -->
      <div class="card p-4 sm:p-6 animate-fade-in delay-200">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-base sm:text-lg font-semibold flex items-center gap-2">
            <PieChart class="w-5 h-5 text-purple-500" />
            {{ t('dashboard.channelDistribution') || 'Channel Distribution' }}
          </h2>
          <el-tooltip :content="t('common.refresh')" placement="top">
            <button @click="loadStats" class="p-2 rounded-lg transition-all hover:bg-[var(--bg-hover)]"
              :class="isDark ? 'text-gray-400' : 'text-gray-600'">
              <RefreshCw class="w-4 h-4" />
            </button>
          </el-tooltip>
        </div>
        <div ref="pieChartRef" class="w-full h-64"></div>
      </div>

      <!-- 每周上传统计 -->
      <div class="lg:col-span-2 card p-4 sm:p-6 animate-fade-in delay-300">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-base sm:text-lg font-semibold flex items-center gap-2">
            <Calendar class="w-5 h-5 text-cyan-500" />
            {{ t('dashboard.weeklyUploadStats') || 'Weekly Upload Stats' }}
          </h2>
          <el-tooltip :content="t('common.refresh')" placement="top">
            <button @click="loadStats" class="p-2 rounded-lg transition-all hover:bg-[var(--bg-hover)]"
              :class="isDark ? 'text-gray-400' : 'text-gray-600'">
              <RefreshCw class="w-4 h-4" />
            </button>
          </el-tooltip>
        </div>
        <div ref="weeklyChartRef" class="w-full h-64"></div>
      </div>
    </div>

    <!-- 渠道状态 -->
    <div class="card p-4 sm:p-6 animate-fade-in">
      <div class="flex items-center justify-between mb-4 sm:mb-6">
        <h2 class="text-base sm:text-lg font-semibold">{{ t('dashboard.channelStatus') || 'Channel Status' }}</h2>
        <el-tooltip :content="t('common.refresh')" placement="top">
          <button @click="loadDashboard" class="p-2 rounded-lg transition-all hover:bg-[var(--bg-hover)]"
            :class="isDark ? 'text-gray-400' : 'text-gray-600'">
            <RefreshCw class="w-4 h-4" />
          </button>
        </el-tooltip>
      </div>

      <div v-if="loading" class="space-y-4">
        <div v-for="i in 3" :key="i" class="h-20 rounded-xl loading-shimmer"></div>
      </div>

      <div v-else-if="!dashboard.channelStatuses?.length" class="text-center py-8 sm:py-12">
        <Cloud class="w-10 h-10 sm:w-12 sm:h-12 mx-auto mb-2 sm:mb-3"
          :class="isDark ? 'text-gray-600' : 'text-gray-400'" />
        <p class="text-sm" :class="isDark ? 'text-gray-400' : 'text-gray-500'">{{ t('channels.noChannels') }}</p>
        <button @click="router.push('/channels')" class="mt-2 sm:mt-3 text-sm text-indigo-500 hover:text-indigo-600">
          {{ t('channels.addFirstChannel') }} →
        </button>
      </div>

      <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4">
        <div v-for="channel in dashboard.channelStatuses" :key="channel.id"
          class="relative p-3 sm:p-4 rounded-xl border-2 transition-all hover:shadow-lg overflow-hidden"
          :class="isDark ? 'bg-[var(--bg-card)] border-gray-600' : 'bg-white border-gray-300'">
          <!-- 左侧状态色条 -->
          <div class="absolute left-0 top-0 bottom-0 w-1 rounded-l-xl" :class="{
            'bg-green-500': channel.status === 'healthy',
            'bg-yellow-500': channel.status === 'warning',
            'bg-red-500': channel.status === 'error',
            'bg-gray-400': !channel.status
          }"></div>

          <div class="flex items-center justify-between mb-2 sm:mb-3 pl-2">
            <div class="flex items-center gap-2 sm:gap-3">
              <div
                class="w-9 h-9 sm:w-11 sm:h-11 rounded-xl flex items-center justify-center text-base sm:text-lg font-bold shadow-sm"
                :class="isDark ? 'bg-[var(--bg-primary)] text-white' : 'bg-indigo-500 text-white'">
                {{ channel.name?.[0]?.toUpperCase() || '?' }}
              </div>
              <div>
                <p class="font-semibold text-sm sm:text-base">{{ channel.name }}</p>
                <p class="text-xs" :class="isDark ? 'text-gray-500' : 'text-gray-400'">
                  {{ channel.type }}
                </p>
              </div>
            </div>
            <div class="flex items-center gap-2">
              <span class="w-3 h-3 rounded-full ring-2 ring-offset-1" :class="[
                getChannelStatusColor(channel.status),
                isDark ? 'ring-[var(--bg-card)]' : 'ring-white'
              ]"></span>
              <span class="text-xs sm:text-sm font-semibold px-2 py-0.5 rounded-full" :class="{
                'bg-green-100 text-green-700': channel.status === 'healthy',
                'bg-yellow-100 text-yellow-700': channel.status === 'warning',
                'bg-red-100 text-red-700': channel.status === 'error',
                'bg-gray-100 text-gray-500': !channel.status
              }">
                {{ channel.status === 'healthy' ? t('common.normal') : channel.status === 'warning' ? t('common.warning') : channel.status === 'error' ? t('common.error') : 'Unknown' }}
              </span>
            </div>
          </div>

          <div class="space-y-1.5 sm:space-y-2 pl-2">
            <div class="flex items-center justify-between text-xs sm:text-sm">
              <span :class="isDark ? 'text-gray-400' : 'text-gray-500'">{{ t('channels.storageUsage') || 'Storage' }}</span>
              <span class="text-xs sm:text-sm font-medium">{{ formatSize(channel.usedSpace) }} / {{ formatSize(channel.totalSpace) }}</span>
            </div>
            <div class="progress-bar">
              <div class="progress" :style="{
                width: (channel.usagePercent || 0) + '%',
                background: channel.usagePercent > 90 ? 'var(--danger)' : channel.usagePercent > 70 ? 'var(--warning)' : 'var(--primary)'
              }"></div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 快捷入口 -->
    <div class="card p-4 sm:p-6 animate-fade-in delay-400">
      <h2 class="text-base sm:text-lg font-semibold mb-3 sm:mb-4">{{ t('dashboard.quickAccess') || 'Quick Access' }}</h2>
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-2 sm:gap-3">
        <el-tooltip :content="t('files.title')" placement="top">
          <button @click="router.push('/files')"
            class="flex items-center gap-2 sm:gap-3 p-3 sm:p-4 rounded-xl border transition-all hover:border-indigo-500 hover:shadow-lg"
            :class="isDark ? 'border-[var(--border)] hover:bg-[var(--bg-hover)]' : 'border-gray-200 hover:bg-gray-50'">
            <Folder class="w-4 h-4 sm:w-5 sm:h-5 text-indigo-500" />
            <span class="font-medium text-xs sm:text-sm">{{ t('nav.files') }}</span>
          </button>
        </el-tooltip>
        <el-tooltip :content="t('channels.title')" placement="top">
          <button @click="router.push('/channels')"
            class="flex items-center gap-2 sm:gap-3 p-3 sm:p-4 rounded-xl border transition-all hover:border-indigo-500 hover:shadow-lg"
            :class="isDark ? 'border-[var(--border)] hover:bg-[var(--bg-hover)]' : 'border-gray-200 hover:bg-gray-50'">
            <Cloud class="w-4 h-4 sm:w-5 sm:h-5 text-purple-500" />
            <span class="font-medium text-xs sm:text-sm">{{ t('nav.channels') }}</span>
          </button>
        </el-tooltip>
        <el-tooltip :content="t('tokens.title')" placement="top">
          <button @click="router.push('/tokens')"
            class="flex items-center gap-2 sm:gap-3 p-3 sm:p-4 rounded-xl border transition-all hover:border-indigo-500 hover:shadow-lg"
            :class="isDark ? 'border-[var(--border)] hover:bg-[var(--bg-hover)]' : 'border-gray-200 hover:bg-gray-50'">
            <Monitor class="w-4 h-4 sm:w-5 sm:h-5 text-cyan-500" />
            <span class="font-medium text-xs sm:text-sm">{{ t('nav.tokens') }}</span>
          </button>
        </el-tooltip>
        <el-tooltip :content="t('settings.title')" placement="top">
          <button @click="router.push('/settings')"
            class="flex items-center gap-2 sm:gap-3 p-3 sm:p-4 rounded-xl border transition-all hover:border-indigo-500 hover:shadow-lg"
            :class="isDark ? 'border-[var(--border)] hover:bg-[var(--bg-hover)]' : 'border-gray-200 hover:bg-gray-50'">
            <Monitor class="w-4 h-4 sm:w-5 sm:h-5 text-orange-500" />
            <span class="font-medium text-xs sm:text-sm">{{ t('nav.settings') }}</span>
          </button>
        </el-tooltip>
      </div>
    </div>
  </div>
</template>
