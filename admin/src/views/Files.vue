<script setup>
import { ref, onMounted, computed, watchEffect } from 'vue'
import { useI18n } from 'vue-i18n'
import request from '@/api/request'
import { ElMessage, ElMessageBox, ElLoading } from 'element-plus'
import {
  FileText, Film, Video, Folder, Trash2, Link, Copy,
  RefreshCw, Search, Grid3x3, List, Download, Eye, Calendar, Zap, Info, CheckSquare,
  MoreHorizontal
} from 'lucide-vue-next'
import { VueDatePicker } from '@vuepic/vue-datepicker'
import '@vuepic/vue-datepicker/dist/main.css'

const { t } = useI18n()
const isDark = ref(true)
const origin = window.location.origin
const files = ref([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(30)
const total = ref(0)
const search = ref('')
const viewMode = ref('grid')
const selected = ref([])

// 时间筛选
const olderThan = ref(0)  // 0表示不限，7/30/90/365分别表示天数
const startDate = ref('')
const endDate = ref('')
const showDatePicker = ref(false)

// 清理相关
const cleanupVisible = ref(false)
const cleanupLoading = ref(false)
const cleanupPreview = ref(null)

// 预览相关
const previewVisible = ref(false)
const previewFile = ref(null)

// 复制相关
const copyVisible = ref(false)
const copyFile = ref(null)

// 文件详情弹窗
const detailVisible = ref(false)
const detailFile = ref(null)
const detailLoading = ref(false)
const selectAllRef = ref(null)
const imageErrors = ref(new Set())

function openCopyMenu(file) {
  copyFile.value = file
  copyVisible.value = true
}

function closeCopyMenu() {
  copyVisible.value = false
  copyFile.value = null
}

async function copyText(text, label) {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success(t('common.copyToClipboard'))
    closeCopyMenu()
  } catch {
    ElMessage.error(t('common.copyFailed'))
  }
}

async function copyFileUrl(file) {
  let url = file.url || file.links?.url
  // 只有本地存储（channelType=local）且没有公网URL时，才用本地代理
  if (!url || file.channelType === 'local') {
    url = `${window.location.origin}/api/v1/file/${file.id}`
  }
  await copyText(url, t('common.link'))
}

function getLinks(file) {
  let url = file.url || file.links?.url
  // 只有本地存储（channelType=local）且没有公网URL时，才用本地代理
  if (!url || file.channelType === 'local') {
    url = `${window.location.origin}/api/v1/file/${file.id}`
  }
  const name = file.name || 'image'
  return {
    url,
    markdown: `![${name}](${url})`,
    html: `<img src="${url}" alt="${name}">`,
    bbcode: `[img]${url}[/img]`
  }
}

const hasSelection = computed(() => selected.value.length > 0)

// 全选当前页相关
const isAllCurrentPageSelected = computed(() => {
  return files.value.length > 0 && files.value.every(f => selected.value.some(s => s.id === f.id))
})

function toggleSelectAllCurrentPage() {
  // 直接基于当前状态翻转，不再依赖 event 参数
  if (isAllCurrentPageSelected.value) {
    // 取消当前页全选
    const currentPageIds = new Set(files.value.map(f => f.id))
    selected.value = selected.value.filter(f => !currentPageIds.has(f.id))
  } else {
    selectAllCurrentPage()
  }
}

// 设置全选checkbox的indeterminate状态
watchEffect(() => {
  if (selectAllRef.value) {
    const selectedCount = files.value.filter(f => selected.value.some(s => s.id === f.id)).length
    selectAllRef.value.indeterminate = selectedCount > 0 && selectedCount < files.value.length
  }
})

// 快速筛选预设
const filterPresets = [
  { label: t('common.all'), value: 0 },
  { label: t('files.daysAgo', [7]), value: 7 },
  { label: t('files.daysAgo', [30]), value: 30 },
  { label: t('files.daysAgo', [90]), value: 90 },
  { label: t('files.oneYearAgo'), value: 365 },
]

onMounted(() => {
  isDark.value = !document.documentElement.classList.contains('light')
  loadFiles()
})

async function loadFiles() {
  loading.value = true
  try {
    const params = { page: page.value, pageSize: pageSize.value }
    if (search.value) params.search = search.value
    if (olderThan.value > 0) params.olderThan = olderThan.value
    if (startDate.value && endDate.value) {
      params.startTime = Math.floor(new Date(startDate.value).getTime() / 1000)
      params.endTime = Math.floor(new Date(endDate.value).getTime() / 1000)
    }

    const res = await request.get('/admin/files', { params })
    if (res.code === 0) {
      files.value = res.data?.list || res.data?.items || []
      total.value = res.data?.total || 0
    }
  } catch {
    ElMessage.error(t('common.loadFailed'))
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  page.value = 1
  loadFiles()
}

function handlePageChange(p) {
  page.value = p
  loadFiles()
}

function handlePresetClick(value) {
  olderThan.value = value
  startDate.value = ''
  endDate.value = ''
  showDatePicker.value = false
  page.value = 1
  loadFiles()
}

function handleDateConfirm() {
  olderThan.value = 0
  page.value = 1
  loadFiles()
}

function clearDateFilter() {
  startDate.value = ''
  endDate.value = ''
  olderThan.value = 0
  showDatePicker.value = false
  page.value = 1
  loadFiles()
}

function selectAllCurrentPage() {
  for (const file of files.value) {
    if (!selected.value.some(s => s.id === file.id)) {
      selected.value.push(file)
    }
  }
}

async function selectAllFiltered() {
  const params = {}
  if (search.value) params.search = search.value
  if (olderThan.value > 0) params.olderThan = olderThan.value
  if (startDate.value && endDate.value) {
    params.startTime = Math.floor(new Date(startDate.value).getTime() / 1000)
    params.endTime = Math.floor(new Date(endDate.value).getTime() / 1000)
  }

  try {
    ElMessage.info(t('files.fetchingFilterResults'))
    const res = await request.get('/files/ids', { params })
    if (res.code === 0 && res.data?.ids) {
      const ids = res.data.ids
      const existingIds = new Set(selected.value.map(f => f.id))
      const newFiles = files.value.filter(f => ids.includes(f.id) && !existingIds.has(f.id))
      selected.value = [...selected.value, ...newFiles]
      ElMessage.success(t('files.selectedCount', [ids.length]))
    }
  } catch {
    ElMessage.error(t('files.fetchFilterFailed'))
  }
}

function clearSelection() {
  selected.value = []
}

async function openFileDetail(file) {
  detailLoading.value = false
  detailVisible.value = true
  console.log('openFileDetail:', JSON.stringify({ id: file.id, url: file.url, links: file.links }))
  // 确保 url 是绝对路径（相对路径需要加上 origin）
  if (file.url && file.url.startsWith('/')) {
    file.url = origin + file.url
  }
  if (file.links?.url && file.links.url.startsWith('/')) {
    file.links.url = origin + file.links.url
  }
  // 原始链接也可能是相对路径
  if (file.originalUrl && file.originalUrl.startsWith('/')) {
    file.originalUrl = origin + file.originalUrl
  }
  detailFile.value = file
}

function formatSize(bytes) {
  if (!bytes) return '-'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

function getOrigin() {
  return origin
}

function formatDate(ts) {
  if (!ts) return '-'
  let date = new Date(ts)
  // 如果是 Unix 时间戳（秒），转换为毫秒
  if (ts > 0 && ts < 10000000000) {
    date = new Date(ts * 1000)
  }
  if (isNaN(date.getTime())) return '-'
  return date.toLocaleString('zh-CN', {
    year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit'
  })
}

function getIcon(type_) {
  if (!type_) return FileText
  if (type_.startsWith('image/')) return FileText
  if (type_.startsWith('video/')) return Film
  if (type_.startsWith('audio/')) return Video
  return FileText
}

function isImageType(type_) {
  return type_?.startsWith('image/')
}

function isImageUrl(url, type_) {
  if (type_?.startsWith('image/')) return true
  if (!url) return false
  const ext = url.split('?')[0].split('#')[0].toLowerCase()
  const imageExts = ['.svg', '.png', '.jpg', '.jpeg', '.gif', '.webp', '.bmp', '.ico']
  return imageExts.some(e => ext.endsWith(e))
}

function hasImageError(fileId) {
  return imageErrors.value.has(fileId)
}

function handleImageError(fileId) {
  imageErrors.value.add(fileId)
}

function isPreviewable(type_) {
  if (!type_) return false
  return type_.startsWith('video/') || type_.startsWith('audio/') || type_.startsWith('image/')
}

function openPreview(file) {
  previewFile.value = file
  previewVisible.value = true
}

function closePreview() {
  previewVisible.value = false
  previewFile.value = null
}

function downloadFile(file) {
  const link = document.createElement('a')
  link.href = file.url || `/api/v1/file/${file.id}/download`
  link.download = file.name
  link.target = '_blank'
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
}

async function deleteFile(file) {
  try {
    await ElMessageBox.confirm(t('files.deleteConfirm', { name: file.name }), t('files.deleteConfirmTitle'), { type: 'warning' })
    await request.delete(`/admin/files/${file.id}`)
    ElMessage.success(t('common.deleteSuccess'))
    loadFiles()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error(t('common.deleteFailed'))
  }
}

async function deleteSelected() {
  if (!hasSelection.value) return
  try {
    const previewFiles = selected.value.slice(0, 5)
    const hasMore = selected.value.length > 5
    let message = `<div class="max-h-48 overflow-y-auto">`
    message += `<p class="mb-2">${t('files.batchDeleteTip', [selected.value.length])}</p>`
    message += `<ul class="space-y-1 text-sm">`
    for (const file of previewFiles) {
      message += `<li class="flex items-center gap-2">
        <span class="truncate max-w-[200px]">${file.name}</span>
        <span class="text-gray-400 text-xs">${formatSize(file.size)}</span>
      </li>`
    }
    if (hasMore) {
      message += `<li class="text-gray-400">${t('files.moreFiles', [selected.value.length - 5])}</li>`
    }
    message += `</ul></div>`

    await ElMessageBox.confirm(message, t('files.batchDeleteConfirm'), {
      type: 'warning',
      dangerouslyUseHTMLString: true,
      confirmButtonText: t('common.confirmDelete'),
      cancelButtonText: t('common.cancel')
    })

    const loading = ElLoading.service({
      lock: true,
      text: t('files.deleting', [selected.value.length]),
      background: 'rgba(0, 0, 0, 0.7)'
    })

    try {
      await request.delete('/admin/files', { data: { ids: selected.value.map(f => f.id) } })
      ElMessage.success(t('files.deleteSuccess', [selected.value.length]))
      selected.value = []
      loadFiles()
    } finally {
      loading.close()
    }
  } catch (e) {
    if (e !== 'cancel') ElMessage.error(t('common.deleteFailed'))
  }
}

function toggleSelect(file, checked) {
  if (checked) {
    selected.value.push(file)
  } else {
    selected.value = selected.value.filter(s => s.id !== file.id)
  }
}

// 打开清理弹窗
function openCleanup() {
  cleanupVisible.value = true
  cleanupPreview.value = null
}

// 预览清理
async function previewCleanup() {
  cleanupLoading.value = true
  try {
    const params = {}
    if (olderThan.value > 0) params.olderThan = olderThan.value
    if (startDate.value && endDate.value) {
      params.startTime = Math.floor(new Date(startDate.value).getTime() / 1000)
      params.endTime = Math.floor(new Date(endDate.value).getTime() / 1000)
    }
    const res = await request.post('/files/cleanup/preview', params)
    if (res.code === 0) {
      cleanupPreview.value = res.data
    } else {
      ElMessage.error(res.message || t('files.previewFailed'))
    }
  } catch {
    ElMessage.error(t('files.previewFailed'))
  } finally {
    cleanupLoading.value = false
  }
}

// 执行清理
async function executeCleanup() {
  try {
    await ElMessageBox.confirm(
      t('files.cleanupConfirm', [cleanupPreview.value?.count || 0, formatSize(cleanupPreview.value?.totalSize || 0)]),
      t('files.cleanupConfirmTitle'),
      { type: 'warning' }
    )

    cleanupLoading.value = true
    const params = {}
    if (olderThan.value > 0) params.olderThan = olderThan.value
    if (startDate.value && endDate.value) {
      params.startTime = Math.floor(new Date(startDate.value).getTime() / 1000)
      params.endTime = Math.floor(new Date(endDate.value).getTime() / 1000)
    }

    const res = await request.post('/files/cleanup', params)
    if (res.code === 0) {
      ElMessage.success(t('files.cleanupComplete', [res.data.deletedCount]))
      cleanupVisible.value = false
      loadFiles()
    } else {
      ElMessage.error(res.message || t('files.cleanupFailed'))
    }
  } catch (e) {
    if (e !== 'cancel') ElMessage.error(t('files.cleanupFailed'))
  } finally {
    cleanupLoading.value = false
  }
}
</script>

<template>
  <div class="space-y-6">
    <!-- 工具栏 -->
    <div class="rounded-xl sm:rounded-2xl border py-3 sm:py-4 flex flex-wrap items-center gap-2 sm:gap-3"
      :class="isDark ? 'bg-[var(--bg-secondary)]/50 border-[var(--border)]' : 'bg-white border-gray-200'">

      <!-- 搜索 -->
      <div class="relative flex-1 min-w-[150px] sm:min-w-[200px]">
        <Search class="absolute left-3 sm:left-4 top-1/2 -translate-y-1/2 w-4 h-4 sm:w-5 sm:h-5"
          :class="isDark ? 'text-gray-500' : 'text-gray-400'" />
        <input v-model="search" type="text" :placeholder="t('files.searchPlaceholder')"
          class="w-full pl-10 sm:pl-12 pr-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
          :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)] text-white placeholder-gray-500' : 'bg-gray-50 border-gray-200 text-gray-800'"
          @keyup.enter="handleSearch" />
      </div>

      <!-- 快速筛选 -->
      <div class="flex items-center gap-1 flex-wrap">
        <button v-for="preset in filterPresets" :key="preset.value" @click="handlePresetClick(preset.value)"
          class="px-2.5 sm:px-3 py-1.5 rounded-lg text-xs sm:text-sm font-medium transition-all whitespace-nowrap"
          :class="olderThan === preset.value && !startDate && !endDate
            ? 'bg-gradient-to-r from-indigo-500 to-purple-500 text-white shadow-lg'
            : (isDark ? 'bg-[var(--bg-hover)] text-gray-400 hover:text-white' : 'bg-gray-100 text-gray-600 hover:text-gray-800')">
          {{ preset.label }}
        </button>
      </div>

      <!-- 日期范围 -->
      <div class="flex items-center gap-2">
        <VueDatePicker v-model="startDate" :placeholder="t('files.startDate')" format="YYYY-MM-DD" class="!w-auto !text-xs sm:!text-sm rounded-xl border"
          :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)] text-white' : 'bg-gray-50 border-gray-200 text-gray-800'"
          @update:modelValue="handleDateConfirm" />
        <span class="text-xs sm:text-sm" :class="isDark ? 'text-gray-400' : 'text-gray-500'">{{ t('common.to') }}</span>
        <VueDatePicker v-model="endDate" :placeholder="t('files.endDate')" format="YYYY-MM-DD" class="!w-auto !text-xs sm:!text-sm rounded-xl border"
          :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)] text-white' : 'bg-gray-50 border-gray-200 text-gray-800'"
          @update:modelValue="handleDateConfirm" />
      </div>

      <!-- 清理按钮 -->
      <el-tooltip :content="t('files.oneClickCleanup')" placement="top">
        <button @click="openCleanup"
          class="flex sm:ml-auto items-center gap-1.5 px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl bg-red-500/10 text-red-500 hover:bg-red-500/20 transition-all text-sm border border-red-500/30">
          <Zap class="w-4 h-4" />
          <span class="hidden sm:inline">{{ t('files.oneClickCleanup') }}</span>
        </button>
      </el-tooltip>

      <!-- 视图切换 -->
      <div class="flex items-center rounded-xl p-1" :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-100'">
        <el-tooltip :content="t('files.gridView')" placement="top">
          <button @click="viewMode = 'grid'" class="p-1.5 sm:p-2 rounded-lg transition-all" :class="viewMode === 'grid'
            ? 'bg-gradient-to-r from-indigo-500 to-purple-500 text-white shadow-lg'
            : (isDark ? 'text-gray-400' : 'text-gray-600')">
            <Grid3x3 class="w-3.5 h-3.5 sm:w-4 sm:h-4" />
          </button>
        </el-tooltip>
        <el-tooltip :content="t('files.listView')" placement="top">
          <button @click="viewMode = 'list'" class="p-1.5 sm:p-2 rounded-lg transition-all" :class="viewMode === 'list'
            ? 'bg-gradient-to-r from-indigo-500 to-purple-500 text-white shadow-lg'
            : (isDark ? 'text-gray-400' : 'text-gray-600')">
            <List class="w-3.5 h-3.5 sm:w-4 sm:h-4" />
          </button>
        </el-tooltip>
      </div>

      <el-tooltip :content="t('files.refreshList')" placement="top">
        <button @click="loadFiles" class="p-2 sm:p-2.5 rounded-xl border transition-all hover:border-indigo-500"
          :class="isDark ? 'border-[var(--border)] bg-[var(--bg-secondary)]' : 'border-gray-200 bg-white'">
          <RefreshCw class="w-4 h-4 sm:w-5 sm:h-5" />
        </button>
      </el-tooltip>
    </div>

    <!-- 当前筛选状态 -->
    <div v-if="olderThan > 0 || (startDate && endDate)" class="flex items-center gap-2 text-sm"
      :class="isDark ? 'text-gray-400' : 'text-gray-500'">
      <span>{{ t('files.currentFilter') }}:</span>
      <span v-if="olderThan > 0" class="px-2 py-0.5 rounded bg-indigo-500/10 text-indigo-500">
        {{ t('files.daysAgo', [olderThan]) }}
      </span>
      <span v-if="startDate && endDate" class="px-2 py-0.5 rounded bg-indigo-500/10 text-indigo-500">
        {{ startDate }} {{ t('common.to') }} {{ endDate }}
      </span>
      <button @click="clearDateFilter" class="ml-2 text-indigo-500 hover:underline">{{ t('files.clearFilter') }}</button>
    </div>

    <!-- 全选工具栏 -->
    <div v-if="files.length > 0" class="flex items-center gap-2 text-sm"
      :class="isDark ? 'text-gray-400' : 'text-gray-500'">
      <input ref="selectAllRef" type="checkbox" :checked="isAllCurrentPageSelected" @change="toggleSelectAllCurrentPage"
        class="w-5 h-5 rounded cursor-pointer accent-indigo-500" />
      <el-tooltip :content="t('files.selectAllCurrentPage')" placement="top">
        <button @click="selectAllCurrentPage"
          class="flex items-center gap-1 px-2 py-1 rounded-lg transition-all hover:bg-indigo-500/10 hover:text-indigo-500">
          <CheckSquare class="w-4 h-4" />
          <span>{{ t('files.selectCurrentPage') }}</span>
        </button>
      </el-tooltip>
      <el-tooltip :content="t('files.selectAllFiltered')" placement="top">
        <button @click="selectAllFiltered"
          class="flex items-center gap-1 px-2 py-1 rounded-lg transition-all hover:bg-indigo-500/10 hover:text-indigo-500">
          <CheckSquare class="w-4 h-4" />
          <span>{{ t('files.selectFiltered') }}</span>
        </button>
      </el-tooltip>
      <span class="text-xs" :class="isDark ? 'text-gray-600' : 'text-gray-400'">|</span>
      <span class="text-xs">{{ t('files.totalFiles', [total]) }}</span>
    </div>

    <!-- 批量操作 -->
    <transition name="slide">
      <div v-if="hasSelection"
        class="rounded-xl border border-indigo-500/50 bg-indigo-500/10 p-3 sm:p-4 flex items-center gap-2 sm:gap-4">
        <span class="text-indigo-500 font-medium text-sm">{{ t('files.selected', [selected.length]) }}</span>
        <div class="flex-1"></div>
        <el-tooltip :content="t('files.deleteSelectedFiles')" placement="top">
          <button @click="deleteSelected"
            class="px-3 sm:px-4 py-1.5 sm:py-2 rounded-lg bg-red-500 text-white hover:bg-red-600 transition-all text-sm">
            {{ t('files.batchDelete') }}
          </button>
        </el-tooltip>
        <button @click="selected = []" class="px-3 sm:px-4 py-1.5 sm:py-2 rounded-lg transition-all text-sm"
          :class="isDark ? 'bg-[var(--bg-hover)] hover:bg-[var(--bg-secondary)]' : 'bg-gray-200 hover:bg-gray-300'">
          {{ t('common.cancel') }}
        </button>
      </div>
    </transition>

    <!-- 内容 -->
    <div v-if="loading"
      :class="viewMode === 'grid' ? 'grid grid-cols-3 sm:grid-cols-4 md:grid-cols-4 lg:grid-cols-6 gap-3 sm:gap-4' : 'space-y-2 sm:space-y-3'">
      <div v-for="i in 12" :key="i"
        :class="viewMode === 'grid' ? 'aspect-square rounded-xl loading-shimmer' : 'h-12 sm:h-16 rounded-xl loading-shimmer'">
      </div>
    </div>

    <div v-else-if="files.length === 0" class="text-center py-20 sm:py-32">
      <div class="w-20 h-20 sm:w-24 sm:h-24 mx-auto rounded-2xl flex items-center justify-center mb-3 sm:mb-4"
        :class="isDark ? 'bg-[var(--bg-secondary)]' : 'bg-gray-100'">
        <Folder class="w-10 h-10 sm:w-12 sm:h-12" :class="isDark ? 'text-gray-600' : 'text-gray-400'" />
      </div>
      <p class="text-sm sm:text-base" :class="isDark ? 'text-gray-400' : 'text-gray-500'">{{ t('files.noFiles') }}</p>
    </div>

    <!-- 网格视图 -->
    <div v-else-if="viewMode === 'grid'"
      class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-3 sm:gap-4">
      <div v-for="file in files" :key="file.id"
        class="group relative flex flex-col min-w-36 rounded-xl border overflow-hidden transition-all duration-300 hover:shadow-xl hover:-translate-y-1"
        :class="isDark ? 'bg-[var(--bg-secondary)] border-[var(--border)]' : 'bg-white border-gray-100'">

        <div class="absolute top-2 left-2 z-10">
          <input type="checkbox" :checked="selected.some(s => s.id === file.id)"
            @change="(e) => toggleSelect(file, e.target.checked)"
            class="w-5 h-5 rounded cursor-pointer accent-indigo-500" />
        </div>

        <div class="aspect-[4/3] flex items-center justify-center cursor-pointer flex-shrink-0 max-h-24"
          :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-50'"
          @click="isPreviewable(file.type) && openPreview(file)">
          <img v-if="isImageUrl(file.url, file.type) && !hasImageError(file.id)" :src="file.url || `/api/v1/file/${file.id}`" :alt="file.name"
            class="w-full h-full object-cover" @error="handleImageError(file.id)" />
          <div v-else-if="isImageUrl(file.url, file.type) && hasImageError(file.id)" class="w-full h-full flex flex-col items-center justify-center gap-1"
            :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-100'">
            <FileText class="w-8 h-8" :class="isDark ? 'text-gray-500' : 'text-gray-400'" />
            <span class="text-xs" :class="isDark ? 'text-gray-500' : 'text-gray-400'">{{ t('common.loadFailed') }}</span>
          </div>
          <component v-else :is="getIcon(file.type)" class="w-10 h-10 sm:w-12 sm:h-12"
            :class="isDark ? 'text-gray-600' : 'text-gray-400'" />
        </div>

        <div class="p-2 sm:p-3 flex flex-col flex-1 min-h-0">
          <div class="flex-1 min-h-0">
            <p class="text-xs sm:text-sm font-medium truncate" :title="file.name">{{ file.name }}</p>
            <p class="text-xs mt-0.5" :class="isDark ? 'text-gray-500' : 'text-gray-400'">
              {{ formatSize(file.size) }}
            </p>
          </div>
          <!-- 按钮：始终显示在底部 -->
          <div class="flex justify-center gap-2 mt-2">
            <el-tooltip :content="t('common.details')" placement="top">
              <button @click.stop="openFileDetail(file)" class="p-1.5 mx-2 rounded-lg hover:bg-black/10 transition-all">
                <Info class="w-4 h-4 text-blue-500" />
              </button>
            </el-tooltip>
            <el-tooltip :content="t('common.copy')" placement="top">
              <button @click.stop="copyFileUrl(file)" class="p-1.5 rounded-lg hover:bg-black/10 transition-all">
                <Link class="w-4 h-4 text-indigo-500" />
              </button>
            </el-tooltip>
            <el-tooltip :content="t('common.download')" placement="top">
              <button @click.stop="downloadFile(file)" class="p-1.5 rounded-lg hover:bg-black/10 transition-all">
                <Download class="w-4 h-4 text-green-500" />
              </button>
            </el-tooltip>
            <el-tooltip :content="t('common.delete')" placement="top">
              <button @click.stop="deleteFile(file)" class="p-1.5 rounded-lg hover:bg-black/10 transition-all">
                <Trash2 class="w-4 h-4 text-red-500" />
              </button>
            </el-tooltip>
          </div>
        </div>
      </div>
    </div>

    <!-- 列表视图 -->
    <div v-else class="rounded-xl sm:rounded-2xl border overflow-hidden"
      :class="isDark ? 'bg-[var(--bg-secondary)]/50 border-[var(--border)]' : 'bg-white border-gray-200'">
      <div class="overflow-x-auto -mx-4 sm:mx-0">
        <table class="w-full">
          <thead :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-50'">
            <tr>
              <th class="w-10 p-3 sm:p-4"></th>
              <th class="text-left p-3 sm:p-4 text-sm font-medium w-16 sm:w-32"
                :class="isDark ? 'text-gray-400' : 'text-gray-500'">
                {{ t('files.fileName') }}</th>
              <th class="text-center p-3 sm:p-4 text-sm font-medium w-20 sm:w-24 hidden sm:table-cell"
                :class="isDark ? 'text-gray-400' : 'text-gray-500'">{{ t('files.fileSize') }}</th>
              <th class="text-center p-3 sm:p-4 text-sm font-medium w-32 sm:w-40 hidden sm:table-cell"
                :class="isDark ? 'text-gray-400' : 'text-gray-500'">{{ t('files.uploadTime') }}</th>
              <th class="text-center p-1 sm:p-4 text-sm font-medium w-28 sm:w-40"
                :class="isDark ? 'text-gray-400' : 'text-gray-500'">{{ t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="file in files" :key="file.id" class="border-t transition-all hover:shadow-md"
              :class="isDark ? 'border-[var(--border)]' : 'border-gray-100'">
              <td class="p-3 sm:p-4 justify-center">
                <input type="checkbox" :checked="selected.some(s => s.id === file.id)"
                  @change="(e) => toggleSelect(file, e.target.checked)"
                  class="w-4 h-4 rounded cursor-pointer accent-indigo-500" />
              </td>
              <td class="p-2 justify-center sm:p-4 w-16 sm:w-32">
                <div class="flex items-center gap-1">
                  <div
                    class="w-6 h-6 sm:w-8 sm:h-8 rounded-lg overflow-hidden flex-shrink-0 flex items-center justify-center"
                    :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-100'">
                    <img v-if="isImageUrl(file.url, file.type) && !hasImageError(file.id)" :src="file.url || `/api/v1/file/${file.id}`" :alt="file.name"
                      class="w-full h-full object-cover" @error="handleImageError(file.id)" />
                    <FileText v-else-if="isImageUrl(file.url, file.type) && hasImageError(file.id)" class="w-3 h-3 sm:w-4 sm:h-4"
                      :class="isDark ? 'text-gray-500' : 'text-gray-400'" />
                    <component v-else :is="getIcon(file.type)" class="w-3 h-3 sm:w-4 sm:h-4"
                      :class="isDark ? 'text-gray-500' : 'text-gray-400'" />
                  </div>
                  <span class="font-medium truncate text-xs sm:text-sm max-w-[20px] sm:max-w-[80px]">{{ file.name
                  }}</span>
                </div>
              </td>
              <td class="p-2 sm:p-4 text-xs text-center sm:text-sm hidden sm:table-cell"
                :class="isDark ? 'text-gray-400' : 'text-gray-500'">
                {{ formatSize(file.size) }}
              </td>
              <td class="p-3 sm:p-4 text-sm  text-center hidden sm:table-cell"
                :class="isDark ? 'text-gray-400' : 'text-gray-500'">
                {{ formatDate(file.createdAt) }}
              </td>
              <td class="p-1 sm:p-4 w-28 sm:w-40 whitespace-nowrap">
                <div class="flex items-center  justify-around gap-0 sm:gap-1 flex-shrink-0">
                  <el-tooltip :content="t('common.details')" placement="top">
                    <button @click="openFileDetail(file)"
                      class="p-0.5 rounded-lg transition-all hover:bg-blue-500/10 border"
                      :class="isDark ? 'border-[var(--border)]' : 'border-gray-200'">
                      <Info class="w-3 h-3 text-blue-500" />
                    </button>
                  </el-tooltip>
                  <el-tooltip :content="t('files.copyLink')" placement="top">
                    <button @click="copyFileUrl(file)"
                      class="p-0.5 rounded-lg transition-all hover:bg-indigo-500/10 border"
                      :class="isDark ? 'border-[var(--border)]' : 'border-gray-200'">
                      <Copy class="w-3 h-3 text-indigo-500" />
                    </button>
                  </el-tooltip>
                  <el-tooltip :content="t('common.download')" placement="top">
                    <button @click="downloadFile(file)"
                      class="p-0.5 rounded-lg transition-all hover:bg-green-500/10 border"
                      :class="isDark ? 'border-[var(--border)]' : 'border-gray-200'">
                      <Download class="w-3 h-3 text-green-500" />
                    </button>
                  </el-tooltip>
                  <el-popover trigger="click" placement="bottom-end" :width="140">
                    <template #reference>
                      <button class="p-0.5 rounded-lg transition-all hover:bg-gray-100 border"
                        :class="isDark ? 'border-[var(--border)] hover:bg-[var(--bg-hover)]' : 'border-gray-200'">
                        <MoreHorizontal class="w-3 h-3 text-gray-500" />
                      </button>
                    </template>
                    <div class="space-y-1">
                      <button v-if="isPreviewable(file.type)" @click="openPreview(file)"
                        class="w-full flex items-center gap-2 px-3 py-2 rounded-lg text-sm transition-all hover:bg-indigo-500/10 text-left"
                        :class="isDark ? 'text-gray-300' : 'text-gray-700'">
                        <Eye class="w-4 h-4 text-indigo-500" /> {{ t('common.preview') }}
                      </button>
                      <button @click="deleteFile(file)"
                        class="w-full flex items-center gap-2 px-3 py-2 rounded-lg text-sm transition-all hover:bg-red-500/10 text-left text-red-500">
                        <Trash2 class="w-4 h-4" /> {{ t('common.delete') }}
                      </button>
                    </div>
                  </el-popover>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- 分页 -->
    <div v-if="total > pageSize" class="flex justify-center">
      <div class="flex items-center gap-1">
        <button v-for="p in Math.min(7, Math.ceil(total / pageSize))" :key="p" @click="handlePageChange(p)"
          class="w-9 h-9 sm:w-10 sm:h-10 rounded-lg font-medium transition-all text-sm"
          :class="page === p
            ? 'bg-gradient-to-r from-indigo-500 to-purple-500 text-white shadow-lg'
            : (isDark ? 'bg-[var(--bg-secondary)] hover:bg-[var(--bg-hover)] text-gray-300' : 'bg-white hover:bg-gray-100 text-gray-700')">
          {{ p }}
        </button>
      </div>
    </div>
  </div>

  <!-- 预览弹窗 -->
  <el-dialog v-model="previewVisible" :title="t('files.filePreview')" width="90vw" class="!max-w-[800px]" :class="isDark ? 'dark' : ''">
    <div v-if="previewFile" class="flex flex-col items-center">
      <img v-if="previewFile.type?.startsWith('image/')" :src="previewFile.url || `/api/v1/file/${previewFile.id}`"
        :alt="previewFile.name" class="max-w-full max-h-[50vh] sm:max-h-[60vh] object-contain rounded-lg" />
      <video v-else-if="previewFile.type?.startsWith('video/')"
        :src="previewFile.url || `/api/v1/file/${previewFile.id}`" controls
        class="max-w-full max-h-[50vh] sm:max-h-[60vh] rounded-lg"></video>
      <audio v-else-if="previewFile.type?.startsWith('audio/')"
        :src="previewFile.url || `/api/v1/file/${previewFile.id}`" controls class="w-full mt-4"></audio>
      <div class="mt-3 sm:mt-4 text-center">
        <p class="font-medium text-sm sm:text-base">{{ previewFile.name }}</p>
        <p class="text-xs sm:text-sm mt-1" :class="isDark ? 'text-gray-400' : 'text-gray-500'">
          {{ formatSize(previewFile.size) }}
        </p>
      </div>
    </div>
  </el-dialog>

  <!-- 清理弹窗 -->
  <el-dialog v-model="cleanupVisible" :title="t('files.oneClickCleanup')" width="90vw" class="!max-w-[600px]" :class="isDark ? 'dark' : ''">
    <div class="space-y-3 sm:space-y-4">
      <div class="p-3 sm:p-4 rounded-xl" :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-50'">
        <p class="text-xs sm:text-sm" :class="isDark ? 'text-gray-400' : 'text-gray-500'">
          {{ t('files.cleanupCondition') }}
        </p>
        <div class="mt-2 flex flex-wrap gap-2">
          <span v-if="olderThan > 0" class="px-2 py-1 rounded bg-indigo-500/10 text-indigo-500 text-xs sm:text-sm">
            {{ t('files.daysAgoFiles', [olderThan]) }}
          </span>
          <span v-else-if="startDate && endDate"
            class="px-2 py-1 rounded bg-indigo-500/10 text-indigo-500 text-xs sm:text-sm">
            {{ startDate }} {{ t('common.to') }} {{ endDate }}
          </span>
          <span v-else class="px-2 py-1 rounded bg-gray-500/10 text-gray-500 text-xs sm:text-sm">
            {{ t('files.allFiles') }}
          </span>
        </div>
      </div>

      <!-- 预览结果 -->
      <div v-if="cleanupPreview" class="space-y-3">
        <div class="grid grid-cols-2 gap-2 sm:gap-3">
          <div class="p-3 sm:p-4 rounded-xl text-center" :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-50'">
            <p class="text-xl sm:text-2xl font-bold text-red-500">{{ cleanupPreview.count }}</p>
            <p class="text-xs mt-1" :class="isDark ? 'text-gray-400' : 'text-gray-500'">{{ t('files.fileCount') }}</p>
          </div>
          <div class="p-3 sm:p-4 rounded-xl text-center" :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-50'">
            <p class="text-xl sm:text-2xl font-bold text-red-500">{{ formatSize(cleanupPreview.totalSize) }}</p>
            <p class="text-xs mt-1" :class="isDark ? 'text-gray-400' : 'text-gray-500'">{{ t('files.willReleaseSpace') }}</p>
          </div>
        </div>

        <!-- 预览列表 -->
        <div v-if="cleanupPreview.preview?.length > 0">
          <p class="text-xs sm:text-sm font-medium mb-2" :class="isDark ? 'text-gray-400' : 'text-gray-500'">{{ t('files.previewTip', [cleanupPreview.preview.length]) }}</p>
          <div class="max-h-32 sm:max-h-40 overflow-y-auto rounded-lg border"
            :class="isDark ? 'border-[var(--border)]' : 'border-gray-200'">
            <div v-for="f in cleanupPreview.preview" :key="f.id"
              class="flex items-center justify-between p-2 text-xs sm:text-sm border-b last:border-b-0"
              :class="isDark ? 'border-[var(--border)]' : 'border-gray-100'">
              <span class="truncate flex-1 mr-2">{{ f.name }}</span>
              <span class="text-xs whitespace-nowrap" :class="isDark ? 'text-gray-500' : 'text-gray-400'">{{
                formatDate(f.uploadedAt)
                }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- 操作按钮 -->
      <div class="flex flex-col sm:flex-row justify-end gap-2">
        <button @click="previewCleanup" :disabled="cleanupLoading"
          class="px-4 py-2 rounded-xl border transition-all text-sm"
          :class="isDark ? 'border-[var(--border)] hover:bg-[var(--bg-hover)]' : 'border-gray-200 hover:bg-gray-50'">
          {{ cleanupPreview ? t('files.refreshPreview') : t('common.preview') }}
        </button>
        <button v-if="cleanupPreview" @click="executeCleanup" :disabled="cleanupLoading || cleanupPreview.count === 0"
          class="px-4 py-2 rounded-xl bg-red-500 text-white hover:bg-red-600 transition-all text-sm disabled:opacity-50">
          {{ t('files.confirmCleanup') }}
        </button>
      </div>
    </div>
  </el-dialog>

  <!-- 文件详情弹窗 -->
  <el-dialog v-model="detailVisible" :title="t('files.fileDetails')" width="90vw" :class="isDark ? 'dark' : ''"
    class="!max-w-[500px] sm:!max-w-[500px]">
    <div v-if="detailLoading" class="flex justify-center py-8">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-500"></div>
    </div>
    <div v-else-if="detailFile" class="space-y-3 sm:space-y-4">
      <!-- 预览图 -->
      <div v-if="isImageUrl(detailFile.links?.url || detailFile.url, detailFile.type)" class="flex justify-center">
        <img :src="detailFile.links?.url || detailFile.url || `${getOrigin()}/api/v1/file/${detailFile.id}`"
          :alt="detailFile.name" class="max-h-48 sm:max-h-60 object-contain rounded-lg border w-full"
          :class="isDark ? 'border-[var(--border)]' : 'border-gray-200'" />
      </div>

      <!-- 基本信息 -->
      <div class="space-y-2 sm:space-y-3">
        <div class="flex flex-col sm:flex-row sm:justify-between sm:items-center py-2 border-b gap-1"
          :class="isDark ? 'border-[var(--border)]' : 'border-gray-100'">
          <span class="text-xs sm:text-sm" :class="isDark ? 'text-gray-400' : 'text-gray-500'">{{ t('files.fileName') }}</span>
          <span class="text-xs sm:text-sm font-medium truncate" :title="detailFile.name">{{ detailFile.name }}</span>
        </div>
        <div class="flex flex-col sm:flex-row sm:justify-between sm:items-center py-2 border-b gap-1"
          :class="isDark ? 'border-[var(--border)]' : 'border-gray-100'">
          <span class="text-xs sm:text-sm" :class="isDark ? 'text-gray-400' : 'text-gray-500'">{{ t('files.fileSize') }}</span>
          <span class="text-xs sm:text-sm font-medium">{{ formatSize(detailFile.size) }}</span>
        </div>
        <div class="flex flex-col sm:flex-row sm:justify-between sm:items-center py-2 border-b gap-1"
          :class="isDark ? 'border-[var(--border)]' : 'border-gray-100'">
          <span class="text-xs sm:text-sm" :class="isDark ? 'text-gray-400' : 'text-gray-500'">{{ t('files.fileType') }}</span>
          <span class="text-xs sm:text-sm" :class="isDark ? 'text-gray-300' : 'text-gray-600'">{{ detailFile.type ||
            t('common.unknown')
            }}</span>
        </div>
        <div class="flex flex-col sm:flex-row sm:justify-between sm:items-center py-2 border-b gap-1"
          :class="isDark ? 'border-[var(--border)]' : 'border-gray-100'">
          <span class="text-xs sm:text-sm" :class="isDark ? 'text-gray-400' : 'text-gray-500'">{{ t('files.storageChannel') }}</span>
          <span class="text-xs sm:text-sm" :class="isDark ? 'text-gray-300' : 'text-gray-600'">{{ detailFile.channelType
            || '-' }}</span>
        </div>
        <div class="flex flex-col sm:flex-row sm:justify-between sm:items-center py-2 border-b gap-1"
          :class="isDark ? 'border-[var(--border)]' : 'border-gray-100'">
          <span class="text-xs sm:text-sm" :class="isDark ? 'text-gray-400' : 'text-gray-500'">{{ t('files.uploadTime') }}</span>
          <span class="text-xs sm:text-sm">{{ formatDate(detailFile.createdAt || detailFile.uploadedAt) }}</span>
        </div>
        <div class="flex flex-col sm:flex-row sm:justify-between sm:items-center py-2 border-b gap-1"
          :class="isDark ? 'border-[var(--border)]' : 'border-gray-100'">
          <span class="text-xs sm:text-sm" :class="isDark ? 'text-gray-400' : 'text-gray-500'">{{ t('common.source') }}</span>
          <span class="text-xs sm:text-sm">{{ detailFile.source || '-' }}</span>
        </div>
        <div v-if="detailFile.accessCount !== undefined"
          class="flex flex-col sm:flex-row sm:justify-between sm:items-center py-2 border-b gap-1"
          :class="isDark ? 'border-[var(--border)]' : 'border-gray-100'">
          <span class="text-xs sm:text-sm" :class="isDark ? 'text-gray-400' : 'text-gray-500'">{{ t('common.accessCount') }}</span>
          <span class="text-xs sm:text-sm font-medium">{{ detailFile.accessCount }}</span>
        </div>
        <div v-if="detailFile.checksum"
          class="flex flex-col sm:flex-row sm:justify-between sm:items-center py-2 border-b gap-1"
          :class="isDark ? 'border-[var(--border)]' : 'border-gray-100'">
          <span class="text-xs sm:text-sm" :class="isDark ? 'text-gray-400' : 'text-gray-500'">{{ t('files.checksum') }}</span>
          <span class="text-xs sm:text-sm font-mono truncate max-w-[200px] sm:max-w-none"
            :class="isDark ? 'text-gray-300' : 'text-gray-600'" :title="detailFile.checksum">{{
              detailFile.checksum.substring(0, 16) }}...</span>
        </div>
      </div>

      <!-- 链接操作 -->
      <div class="pt-2 space-y-3">
        <div>
          <p class="text-xs sm:text-sm mb-1" :class="isDark ? 'text-gray-400' : 'text-gray-500'">{{ t('files.originalLink') }}</p>
          <div class="flex flex-col sm:flex-row gap-2">
            <input :value="detailFile.originalUrl || ''" readonly
              class="flex-1 px-3 py-2 rounded-lg border text-xs sm:text-sm min-w-0"
              :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)] text-gray-300' : 'bg-gray-50 border-gray-200'" />
            <button @click="copyText(detailFile.originalUrl, t('files.originalLink'))"
              class="px-4 py-2 rounded-lg bg-gray-500 text-white hover:bg-gray-600 transition-all text-xs sm:text-sm whitespace-nowrap">
              {{ t('common.copy') }}
            </button>
          </div>
        </div>
        <div>
          <p class="text-xs sm:text-sm mb-1" :class="isDark ? 'text-gray-400' : 'text-gray-500'">{{ t('files.cdnLink') }}</p>
          <div class="flex flex-col sm:flex-row gap-2">
            <input :value="detailFile.url || detailFile.links?.url || ''" readonly
              class="flex-1 px-3 py-2 rounded-lg border text-xs sm:text-sm min-w-0"
              :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)] text-gray-300' : 'bg-gray-50 border-gray-200'" />
            <button @click="copyText(detailFile.url || detailFile.links?.url, t('files.cdnLink'))"
              class="px-4 py-2 rounded-lg bg-indigo-500 text-white hover:bg-indigo-600 transition-all text-xs sm:text-sm whitespace-nowrap">
              {{ t('common.copy') }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </el-dialog>
</template>

<style scoped>
.slide-enter-active,
.slide-leave-active {
  transition: all 0.3s ease;
}

.slide-enter-from,
.slide-leave-to {
  opacity: 0;
  transform: translateY(-10px);
}
</style>
