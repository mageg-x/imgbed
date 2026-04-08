<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useThemeStore } from '@/stores/theme'
import { fileApi } from '@/api/file'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Image, FileText, Film, Video, Folder, Trash2, Link,
  RefreshCw, Search, Grid3x3, List, ArrowLeft, Sun, Moon, Copy, Download
} from 'lucide-vue-next'

const { t } = useI18n()
const router = useRouter()
const authStore = useAuthStore()
const themeStore = useThemeStore()
const files = ref([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(30)
const total = ref(0)
const search = ref('')
const viewMode = ref('grid')
const selected = ref([])
const imageErrors = ref(new Set())

// 预览相关
const previewVisible = ref(false)
const previewFile = ref(null)

const hasSelection = computed(() => selected.value.length > 0)

function getOrigin() {
  return window.location.origin
}

onMounted(() => {
  themeStore.init()
  loadFiles()
})

async function loadFiles() {
  loading.value = true
  try {
    const params = {
      page: page.value,
      pageSize: pageSize.value
    }
    if (search.value) params.search = search.value

    const res = await fileApi.list(params)
    if (res.code === 0) {
      files.value = res.data?.items || res.data?.list || []
      total.value = res.data?.total || 0
    }
  } catch {
    ElMessage.error(t('browse.loadFailed'))
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
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

function formatSize(bytes) {
  if (!bytes) return '-'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

function formatDate(ts) {
  if (!ts) return '-'
  return new Date(ts * 1000).toLocaleString('zh-CN', {
    year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit'
  })
}

function getIcon(type_) {
  if (!type_) return FileText
  if (type_.startsWith('image/')) return Image
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

async function copyUrl(url) {
  try {
    const fullUrl = url.startsWith('http://') || url.startsWith('https://')
      ? url
      : getOrigin() + url
    await navigator.clipboard.writeText(fullUrl)
    ElMessage.success(t('common.linkCopied'))
  } catch {
    ElMessage.error(t('common.copyFailed'))
  }
}

async function deleteFile(file) {
  try {
    await ElMessageBox.confirm(t('browse.deleteConfirm', { 0: file.name }), t('browse.confirmDelete'), { type: 'warning' })
    await fileApi.deleteFile(file.id)
    ElMessage.success(t('common.deleteSuccess'))
    loadFiles()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error(t('common.deleteFailed'))
  }
}

async function deleteSelected() {
  if (!hasSelection.value) return
  try {
    await ElMessageBox.confirm(t('browse.batchDeleteConfirm', { 0: selected.value.length }), t('browse.batchDeleteTitle'), { type: 'warning' })
    await fileApi.deleteMultiple(selected.value.map(f => f.id))
    ElMessage.success(t('common.deleteSuccess'))
    selected.value = []
    loadFiles()
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

async function downloadFile(file) {
  try {
    const url = window.location.origin + '/api/v1/file/' + file.id + '/download'
    const link = document.createElement('a')
    link.href = url
    link.download = file.name
    link.click()
    ElMessage.success(t('common.downloadStart'))
  } catch {
    ElMessage.error(t('common.downloadFailed'))
  }
}
</script>

<template>
  <div class="min-h-screen" :class="themeStore.isDark ? 'bg-[var(--bg-primary)]' : 'bg-gray-50'">
    <!-- 顶部导航 -->
    <header class="sticky top-0 z-50 border-b backdrop-blur-xl"
      :class="themeStore.isDark ? 'bg-[var(--bg-primary)]/80 border-[var(--border)]' : 'bg-white/80 border-gray-200'">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 py-3 sm:py-4 flex items-center justify-between">
        <div class="flex items-center gap-3 sm:gap-4">
          <el-tooltip :content="t('common.backToHome')" placement="bottom">
            <button @click="router.push('/')" class="p-2 rounded-lg transition-all"
              :class="themeStore.isDark ? 'hover:bg-white/5 text-gray-400' : 'hover:bg-gray-100 text-gray-600'">
              <ArrowLeft class="w-5 h-5" />
            </button>
          </el-tooltip>
          <div class="flex items-center gap-2 sm:gap-3">
            <img src="/imgbed.webp" alt="ImgBed"
              class="w-8 h-8 sm:w-10 sm:h-10 rounded-xl object-cover shadow-lg shadow-indigo-500/30" />
            <span class="text-lg sm:text-xl font-bold">
              <span class="text-gradient">{{ t('browse.title') }}</span>
            </span>
          </div>
        </div>

        <div class="flex items-center gap-1 sm:gap-2">
          <el-tooltip :content="themeStore.isDark ? t('common.switchToLightMode') : t('common.switchToDarkMode')" placement="bottom">
            <button @click="themeStore.toggle" class="p-2 rounded-lg transition-all"
              :class="themeStore.isDark ? 'text-gray-400 hover:text-white hover:bg-white/5' : 'text-gray-600 hover:text-gray-900 hover:bg-gray-100'">
              <Sun v-if="themeStore.isDark" class="w-5 h-5" />
              <Moon v-else class="w-5 h-5" />
            </button>
          </el-tooltip>
          <el-tooltip :content="t('common.refresh')" placement="bottom">
            <button @click="loadFiles" class="p-2 rounded-lg transition-all"
              :class="themeStore.isDark ? 'text-gray-400 hover:text-white hover:bg-white/5' : 'text-gray-600 hover:text-gray-900 hover:bg-gray-100'">
              <RefreshCw class="w-5 h-5" />
            </button>
          </el-tooltip>
        </div>
      </div>
    </header>

    <!-- 工具栏 -->
    <div class="max-w-7xl mx-auto px-4 sm:px-6 py-3 sm:py-4">
      <div class="rounded-xl sm:rounded-2xl border p-3 sm:p-4 flex flex-wrap items-center gap-2 sm:gap-3"
        :class="themeStore.isDark ? 'bg-[var(--bg-secondary)]/50 border-[var(--border)]' : 'bg-white border-gray-200'">

        <!-- 搜索 -->
        <div class="relative flex-1 min-w-[200px]">
          <Search class="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5"
            :class="themeStore.isDark ? 'text-gray-500' : 'text-gray-400'" />
          <input v-model="search" type="text" :placeholder="t('browse.searchPlaceholder')"
            class="w-full pl-12 pr-4 py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50"
            :class="themeStore.isDark ? 'bg-[var(--bg-hover)] border-[var(--border)] text-white placeholder-gray-500' : 'bg-gray-50 border-gray-200 text-gray-800'"
            @keyup.enter="handleSearch" />
        </div>

        <!-- 视图切换 -->
        <div class="flex items-center rounded-xl p-1"
          :class="themeStore.isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-100'">
          <button @click="viewMode = 'grid'" class="p-2 rounded-lg transition-all" :class="viewMode === 'grid'
            ? 'bg-gradient-to-r from-indigo-500 to-purple-500 text-white shadow-lg'
            : (themeStore.isDark ? 'text-gray-400' : 'text-gray-600')">
            <Grid3x3 class="w-4 h-4" />
          </button>
          <button @click="viewMode = 'list'" class="p-2 rounded-lg transition-all" :class="viewMode === 'list'
            ? 'bg-gradient-to-r from-indigo-500 to-purple-500 text-white shadow-lg'
            : (themeStore.isDark ? 'text-gray-400' : 'text-gray-600')">
            <List class="w-4 h-4" />
          </button>
        </div>
      </div>
    </div>

    <!-- 批量操作 -->
    <transition name="slide">
      <div v-if="hasSelection" class="max-w-7xl mx-auto px-4 sm:px-6 py-2">
        <div
          class="rounded-xl border border-indigo-500/50 bg-indigo-500/10 p-3 sm:p-4 flex items-center gap-3 sm:gap-4">
          <span class="text-indigo-500 font-medium text-sm sm:text-base">{{ t('browse.selectedCount', { 0: selected.length }) }}</span>
          <div class="flex-1"></div>
          <el-tooltip :content="t('browse.deleteSelectedTip')" placement="top">
            <button @click="deleteSelected"
              class="px-3 sm:px-4 py-1.5 sm:py-2 rounded-lg bg-red-500 text-white hover:bg-red-600 transition-all text-sm">
              {{ t('browse.batchDelete') }}
            </button>
          </el-tooltip>
          <button @click="selected = []" class="px-3 sm:px-4 py-1.5 sm:py-2 rounded-lg transition-all text-sm"
            :class="themeStore.isDark ? 'bg-[var(--bg-hover)] hover:bg-[var(--bg-secondary)]' : 'bg-gray-200 hover:bg-gray-300'">
            {{ t('browse.cancel') }}
          </button>
        </div>
      </div>
    </transition>

    <!-- 内容 -->
    <main class="max-w-7xl mx-auto px-4 sm:px-6 pb-8 sm:pb-10">
      <!-- 加载状态 -->
      <div v-if="loading"
        :class="viewMode === 'grid' ? 'grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-4' : 'space-y-3'">
        <div v-for="i in 12" :key="i"
          :class="viewMode === 'grid' ? 'aspect-square rounded-xl loading-shimmer' : 'h-16 rounded-xl loading-shimmer'">
        </div>
      </div>

      <!-- 空状态 -->
      <div v-else-if="files.length === 0" class="text-center py-20 sm:py-32">
        <div class="w-20 h-20 sm:w-24 sm:h-24 mx-auto rounded-2xl flex items-center justify-center mb-4"
          :class="themeStore.isDark ? 'bg-[var(--bg-secondary)]' : 'bg-gray-100'">
          <Folder class="w-10 h-10 sm:w-12 sm:h-12" :class="themeStore.isDark ? 'text-gray-600' : 'text-gray-400'" />
        </div>
        <p class="text-base sm:text-lg mb-3 sm:mb-4" :class="themeStore.isDark ? 'text-gray-400' : 'text-gray-500'">{{ t('browse.noFiles') }}
        </p>
        <button @click="router.push('/')"
          class="px-5 sm:px-6 py-2 sm:py-2.5 rounded-xl text-white bg-gradient-to-r from-indigo-500 to-purple-500 hover:from-indigo-600 hover:to-purple-600 transition-all shadow-lg shadow-indigo-500/25">
          {{ t('browse.goUpload') }}
        </button>
      </div>

      <!-- 网格视图 -->
      <div v-else-if="viewMode === 'grid'"
        class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-3 sm:gap-4">
        <div v-for="(file, idx) in files" :key="file.id"
          class="group relative rounded-xl border overflow-hidden transition-all duration-300 hover:shadow-xl hover:-translate-y-1"
          :class="themeStore.isDark ? 'bg-[var(--bg-secondary)] border-[var(--border)]' : 'bg-white border-gray-100'">

          <!-- 选择框 -->
          <div class="absolute top-2 left-2 z-10 opacity-0 group-hover:opacity-100 transition-all">
            <input type="checkbox" :checked="selected.some(s => s.id === file.id)"
              @change="(e) => toggleSelect(file, e.target.checked)"
              class="w-5 h-5 rounded cursor-pointer accent-indigo-500" />
          </div>

          <!-- 预览 -->
          <div class="aspect-square flex items-center justify-center cursor-pointer"
            :class="themeStore.isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-50'"
            @click="isPreviewable(file.type) && openPreview(file)">
            <img v-if="file.url && !hasImageError(file.id)" :src="file.url" :alt="file.name" class="w-full h-full object-cover" @error="handleImageError(file.id)" />
            <component v-else-if="file.url && hasImageError(file.id)" :is="getIcon(file.type)" class="w-10 h-10 sm:w-12 sm:h-12"
              :class="themeStore.isDark ? 'text-gray-600' : 'text-gray-400'" />
            <component v-else :is="getIcon(file.type)" class="w-10 h-10 sm:w-12 sm:h-12"
              :class="themeStore.isDark ? 'text-gray-600' : 'text-gray-400'" />
          </div>

          <!-- 信息 -->
          <div class="p-2 sm:p-3">
            <p class="text-sm font-medium truncate" :title="file.name">{{ file.name }}</p>
            <p class="text-xs mt-0.5" :class="themeStore.isDark ? 'text-gray-500' : 'text-gray-400'">
              {{ formatSize(file.size) }}
            </p>
          </div>

          <!-- 操作 -->
          <div class="absolute top-2 right-2 z-10 opacity-0 group-hover:opacity-100 transition-all flex gap-1">
            <el-tooltip :content="t('common.copyLink')" placement="top">
              <button @click.stop="copyUrl(file.url)"
                class="p-1.5 rounded-lg bg-white/90 backdrop-blur-sm border shadow-sm hover:bg-white transition-all">
                <Link class="w-4 h-4 text-indigo-500" />
              </button>
            </el-tooltip>
            <el-tooltip :content="t('common.download')" placement="top">
              <button @click.stop="downloadFile(file)"
                class="p-1.5 rounded-lg bg-white/90 backdrop-blur-sm border shadow-sm hover:bg-white transition-all">
                <Download class="w-4 h-4 text-green-500" />
              </button>
            </el-tooltip>
            <el-tooltip :content="t('common.delete')" placement="top">
              <button @click.stop="deleteFile(file)"
                class="p-1.5 rounded-lg bg-white/90 backdrop-blur-sm border shadow-sm hover:bg-white transition-all">
                <Trash2 class="w-4 h-4 text-red-500" />
              </button>
            </el-tooltip>
          </div>
        </div>
      </div>

      <!-- 列表视图 -->
      <div v-else class="rounded-xl sm:rounded-2xl border overflow-hidden"
        :class="themeStore.isDark ? 'bg-[var(--bg-secondary)]/50 border-[var(--border)]' : 'bg-white border-gray-200'">
        <div class="overflow-x-auto">
          <table class="w-full min-w-[600px]">
            <thead :class="themeStore.isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-50'">
              <tr>
                <th class="w-10 p-3 sm:p-4"></th>
                <th class="text-left p-3 sm:p-4 text-sm font-medium"
                  :class="themeStore.isDark ? 'text-gray-400' : 'text-gray-500'">{{ t('browse.fileName') }}</th>
                <th class="text-left p-3 sm:p-4 text-sm font-medium w-24"
                  :class="themeStore.isDark ? 'text-gray-400' : 'text-gray-500'">{{ t('browse.fileSize') }}</th>
                <th class="text-left p-3 sm:p-4 text-sm font-medium w-40 hidden sm:table-cell"
                  :class="themeStore.isDark ? 'text-gray-400' : 'text-gray-500'">{{ t('browse.uploadTime') }}</th>
                <th class="text-right p-3 sm:p-4 text-sm font-medium w-24"
                  :class="themeStore.isDark ? 'text-gray-400' : 'text-gray-500'">{{ t('browse.operation') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="file in files" :key="file.id" class="border-t transition-all hover:shadow-md"
                :class="themeStore.isDark ? 'border-[var(--border)]' : 'border-gray-100'">
                <td class="p-3 sm:p-4">
                  <input type="checkbox" :checked="selected.some(s => s.id === file.id)"
                    @change="(e) => toggleSelect(file, e.target.checked)"
                    class="w-4 h-4 rounded cursor-pointer accent-indigo-500" />
                </td>
                <td class="p-3 sm:p-4">
                  <div class="flex items-center gap-2 sm:gap-3">
                    <div
                      class="w-8 h-8 sm:w-10 sm:h-10 rounded-lg overflow-hidden flex-shrink-0 flex items-center justify-center"
                      :class="themeStore.isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-100'">
                      <img v-if="file.url && !hasImageError(file.id)" :src="file.url" :alt="file.name"
                        class="w-full h-full object-cover" @error="handleImageError(file.id)" />
                      <component v-else-if="file.url && hasImageError(file.id)" :is="getIcon(file.type)" class="w-4 h-4 sm:w-5 sm:h-5"
                        :class="themeStore.isDark ? 'text-gray-500' : 'text-gray-400'" />
                      <component v-else :is="getIcon(file.type)" class="w-4 h-4 sm:w-5 sm:h-5"
                        :class="themeStore.isDark ? 'text-gray-500' : 'text-gray-400'" />
                    </div>
                    <span class="font-medium text-sm sm:text-base truncate max-w-[120px] sm:max-w-none">{{ file.name
                    }}</span>
                  </div>
                </td>
                <td class="p-3 sm:p-4 text-sm" :class="themeStore.isDark ? 'text-gray-400' : 'text-gray-500'">
                  {{ formatSize(file.size) }}
                </td>
                <td class="p-3 sm:p-4 text-sm hidden sm:table-cell"
                  :class="themeStore.isDark ? 'text-gray-400' : 'text-gray-500'">
                  {{ formatDate(file.createdAt || file.uploadedAt) }}
                </td>
                <td class="p-3 sm:p-4">
                  <div class="flex items-center justify-end gap-1">
                    <el-tooltip :content="t('common.copyLink')" placement="top">
                      <button @click="copyUrl(file.url)"
                        class="p-1.5 sm:p-2 rounded-lg transition-all hover:bg-indigo-500/10">
                        <Copy class="w-4 h-4 text-indigo-500" />
                      </button>
                    </el-tooltip>
                    <el-tooltip :content="t('common.download')" placement="top">
                      <button @click="downloadFile(file)"
                        class="p-1.5 sm:p-2 rounded-lg transition-all hover:bg-green-500/10">
                        <Download class="w-4 h-4 text-green-500" />
                      </button>
                    </el-tooltip>
                    <el-tooltip :content="t('common.delete')" placement="top">
                      <button @click="deleteFile(file)"
                        class="p-1.5 sm:p-2 rounded-lg transition-all hover:bg-red-500/10">
                        <Trash2 class="w-4 h-4 text-red-500" />
                      </button>
                    </el-tooltip>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- 分页 -->
      <div v-if="total > pageSize" class="flex justify-center mt-10">
        <div class="flex items-center gap-1">
          <button v-for="p in Math.min(7, Math.ceil(total / pageSize))" :key="p" @click="handlePageChange(p)"
            class="w-10 h-10 rounded-lg font-medium transition-all"
            :class="page === p
              ? 'bg-gradient-to-r from-indigo-500 to-purple-500 text-white shadow-lg'
              : (themeStore.isDark ? 'bg-[var(--bg-secondary)] hover:bg-[var(--bg-hover)] text-gray-300' : 'bg-white hover:bg-gray-100 text-gray-700')">
            {{ p }}
          </button>
        </div>
      </div>
    </main>

    <!-- 预览弹窗 -->
    <el-dialog v-model="previewVisible" :title="t('browse.filePreview')" width="800px">
      <div v-if="previewFile" class="flex flex-col items-center">
        <img v-if="previewFile.type?.startsWith('image/')" :src="previewFile.url" :alt="previewFile.name"
          class="max-w-full max-h-[60vh] object-contain rounded-lg" />
        <video v-else-if="previewFile.type?.startsWith('video/')" :src="previewFile.url" controls
          class="max-w-full max-h-[60vh] rounded-lg"></video>
        <audio v-else-if="previewFile.type?.startsWith('audio/')" :src="previewFile.url" controls
          class="w-full mt-4"></audio>
        <div class="mt-4 text-center">
          <p class="font-medium">{{ previewFile.name }}</p>
          <p class="text-sm mt-1" :class="themeStore.isDark ? 'text-gray-400' : 'text-gray-500'">
            {{ formatSize(previewFile.size) }}
          </p>
        </div>
      </div>
    </el-dialog>
  </div>
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
