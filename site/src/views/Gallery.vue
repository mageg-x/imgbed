<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { fileApi } from '@/api/file'
import { useAuthStore } from '@/stores/auth'
import { useThemeStore } from '@/stores/theme'
import { ElMessage } from 'element-plus'
import { Image, Folder, Link, RefreshCw, Sun, Moon, ArrowLeft, X, Copy, Grid3x3, List, Search } from 'lucide-vue-next'

const router = useRouter()
const authStore = useAuthStore()
const themeStore = useThemeStore()
const files = ref([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(30)
const total = ref(0)
const previewUrl = ref('')
const showPreview = ref(false)
const viewMode = ref('grid')

const search = ref('')
const olderThan = ref(0)

function getOrigin() {
  return window.location.origin
}

const filterPresets = [
  { label: '全部', value: 0 },
  { label: '今天', value: -1 },
  { label: '7天内', value: 7 },
  { label: '30天内', value: 30 },
  { label: '90天内', value: 90 },
]

onMounted(async () => {
  themeStore.init()
  const savedViewMode = localStorage.getItem('galleryViewMode')
  if (savedViewMode) {
    viewMode.value = savedViewMode
  }
  const authenticated = await authStore.checkSession()
  if (!authenticated) {
    ElMessage.warning('请先登录')
    router.push('/login')
    return
  }
  loadGallery()
})

async function loadGallery() {
  loading.value = true
  try {
    const params = { page: page.value, pageSize: pageSize.value }
    if (search.value) params.search = search.value
    if (olderThan.value < 0) {
      const today = new Date()
      today.setHours(0, 0, 0, 0)
      params.startTime = Math.floor(today.getTime() / 1000)
    } else if (olderThan.value > 0) {
      params.startTime = Math.floor(Date.now() / 1000) - olderThan.value * 24 * 60 * 60
    }
    const res = await fileApi.list(params)
    if (res.code === 0) {
      files.value = res.data?.list || res.data?.items || []
      total.value = res.data?.total || 0
    }
  } catch {
    ElMessage.error('加载图库失败')
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  page.value = 1
  loadGallery()
}

function handlePresetClick(value) {
  olderThan.value = value
  page.value = 1
  loadGallery()
}

function clearFilters() {
  search.value = ''
  olderThan.value = 0
  page.value = 1
  loadGallery()
}

function handlePageChange(p) {
  page.value = p
  loadGallery()
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

async function copyUrl(url) {
  try {
    const fullUrl = url.startsWith('http://') || url.startsWith('https://')
      ? url
      : getOrigin() + url
    await navigator.clipboard.writeText(fullUrl)
    ElMessage.success('链接已复制')
  } catch {
    ElMessage.error('复制失败')
  }
}

function openPreview(url) {
  previewUrl.value = url
  showPreview.value = true
}

function setViewMode(mode) {
  viewMode.value = mode
  localStorage.setItem('galleryViewMode', mode)
}

function formatDate(timestamp) {
  if (!timestamp) return '-'
  const date = new Date(timestamp * 1000)
  return date.toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' })
}

function formatSize(bytes) {
  if (!bytes) return '-'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}
</script>

<template>
  <div class="min-h-screen " :class="themeStore.isDark ? 'bg-[var(--bg-primary)]' : 'bg-gray-50'">
    <!-- 顶部导航 -->
    <header class="sticky top-0 z-50 border-b backdrop-blur-xl"
      :class="themeStore.isDark ? 'bg-[var(--bg-primary)]/80 border-[var(--border)]' : 'bg-white/80 border-gray-200'">
      <div class=" px-4 sm:px-6 py-3 sm:py-4 flex items-center justify-between">
        <div class="flex items-center gap-3 sm:gap-4">
          <el-tooltip content="返回首页" placement="bottom">
            <button @click="router.push('/')" class="p-2 rounded-lg transition-all"
              :class="themeStore.isDark ? 'hover:bg-white/5 text-gray-400' : 'hover:bg-gray-100 text-gray-600'">
              <ArrowLeft class="w-5 h-5" />
            </button>
          </el-tooltip>
          <div class="flex items-center gap-2 sm:gap-3">
            <img src="/imgbed.webp" alt="ImgBed"
              class="w-8 h-8 sm:w-10 sm:h-10 rounded-xl object-cover shadow-lg shadow-indigo-500/30" />
            <span class="text-lg sm:text-xl font-bold">
              <span class="text-gradient">我的图库</span>
            </span>
          </div>
        </div>

        <div class="flex items-center gap-1 sm:gap-2">
          <div class="flex items-center rounded-lg border overflow-hidden"
            :class="themeStore.isDark ? 'border-[var(--border)]' : 'border-gray-200'">
            <el-tooltip content="网格视图" placement="bottom">
              <button @click="setViewMode('grid')" class="p-1.5 sm:p-2 transition-all" :class="viewMode === 'grid'
                ? 'bg-gradient-to-r from-indigo-500 to-purple-500 text-white'
                : (themeStore.isDark ? 'text-gray-400 hover:text-white' : 'text-gray-600 hover:text-gray-900')">
                <Grid3x3 class="w-4 h-4" />
              </button>
            </el-tooltip>
            <el-tooltip content="列表视图" placement="bottom">
              <button @click="setViewMode('list')" class="p-1.5 sm:p-2 transition-all" :class="viewMode === 'list'
                ? 'bg-gradient-to-r from-indigo-500 to-purple-500 text-white'
                : (themeStore.isDark ? 'text-gray-400 hover:text-white' : 'text-gray-600 hover:text-gray-900')">
                <List class="w-4 h-4" />
              </button>
            </el-tooltip>
          </div>
          <el-tooltip :content="themeStore.isDark ? '切换亮色模式' : '切换暗色模式'" placement="bottom">
            <button @click="themeStore.toggle" class="p-2 rounded-lg transition-all"
              :class="themeStore.isDark ? 'text-gray-400 hover:text-white hover:bg-white/5' : 'text-gray-600 hover:text-gray-900 hover:bg-gray-100'">
              <Sun v-if="themeStore.isDark" class="w-5 h-5" />
              <Moon v-else class="w-5 h-5" />
            </button>
          </el-tooltip>
          <el-tooltip content="刷新" placement="bottom">
            <button @click="loadGallery" class="p-2 rounded-lg transition-all"
              :class="themeStore.isDark ? 'text-gray-400 hover:text-white hover:bg-white/5' : 'text-gray-600 hover:text-gray-900 hover:bg-gray-100'">
              <RefreshCw class="w-5 h-5" />
            </button>
          </el-tooltip>
        </div>
      </div>
    </header>
    <div class="container max-w-6xl mx-auto ">
      <!-- 搜索和筛选 -->
      <div class="max-w-7xl mx-auto px-4 sm:px-6 py-4 sm:py-6">
        <div class="flex flex-col sm:flex-row gap-3 sm:gap-4">
          <!-- 搜索框 -->
          <div class="relative flex-1">
            <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4"
              :class="themeStore.isDark ? 'text-gray-500' : 'text-gray-400'" />
            <input v-model="search" type="text" placeholder="搜索文件名..." @keyup.enter="handleSearch"
              class="w-full pl-10 pr-4 py-2.5 rounded-xl border transition-all" :class="themeStore.isDark
                ? 'bg-[var(--bg-secondary)] border-[var(--border)] text-white placeholder-gray-500 focus:border-indigo-500'
                : 'bg-white border-gray-200 text-gray-800 placeholder-gray-400 focus:border-indigo-500'" />
          </div>

          <!-- 时间筛选 -->
          <div class="flex items-center gap-1 rounded-lg border overflow-hidden"
            :class="themeStore.isDark ? 'border-[var(--border)]' : 'border-gray-200'">
            <button v-for="preset in filterPresets" :key="preset.value" @click="handlePresetClick(preset.value)"
              class="px-3 py-2 text-sm transition-all"
              :class="olderThan === preset.value
                ? 'bg-gradient-to-r from-indigo-500 to-purple-500 text-white'
                : (themeStore.isDark ? 'text-gray-400 hover:text-white hover:bg-white/5' : 'text-gray-600 hover:text-gray-900 hover:bg-gray-50')">
              {{ preset.label }}
            </button>
          </div>
        </div>

        <!-- 当前筛选状态 -->
        <div v-if="search || olderThan !== 0" class="flex items-center gap-2 mt-3 text-sm"
          :class="themeStore.isDark ? 'text-gray-400' : 'text-gray-500'">
          <span>当前筛选：</span>
          <span v-if="search" class="px-2 py-0.5 rounded bg-indigo-500/10 text-indigo-500">
            搜索: {{ search }}
          </span>
          <span v-if="olderThan === -1" class="px-2 py-0.5 rounded bg-indigo-500/10 text-indigo-500">
            今天
          </span>
          <span v-else-if="olderThan > 0" class="px-2 py-0.5 rounded bg-indigo-500/10 text-indigo-500">
            {{ olderThan }}天内
          </span>
          <button @click="clearFilters" class="ml-2 text-indigo-500 hover:underline">清除筛选</button>
        </div>
      </div>

      <!-- 内容 -->
      <main class="max-w-7xl mx-auto px-4 sm:px-6 py-6 sm:py-8">
        <!-- 加载状态 -->
        <div v-if="loading" class="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-3 sm:gap-4">
          <div v-for="i in 12" :key="i" class="aspect-square rounded-xl loading-shimmer"></div>
        </div>

        <!-- 空状态 -->
        <div v-else-if="files.length === 0" class="text-center py-20 sm:py-32">
          <div class="w-20 h-20 sm:w-24 sm:h-24 mx-auto rounded-2xl flex items-center justify-center mb-4"
            :class="themeStore.isDark ? 'bg-[var(--bg-secondary)]' : 'bg-gray-100'">
            <Folder class="w-10 h-10 sm:w-12 sm:h-12" :class="themeStore.isDark ? 'text-gray-600' : 'text-gray-400'" />
          </div>
          <p class="text-base sm:text-lg mb-3 sm:mb-4" :class="themeStore.isDark ? 'text-gray-400' : 'text-gray-500'">
            暂无图片
          </p>
          <button @click="router.push('/')"
            class="px-5 sm:px-6 py-2 sm:py-2.5 rounded-xl text-white bg-gradient-to-r from-indigo-500 to-purple-500 hover:from-indigo-600 hover:to-purple-600 transition-all shadow-lg shadow-indigo-500/25">
            去上传
          </button>
        </div>

        <!-- 瀑布流网格 -->
        <div v-else-if="viewMode === 'grid'"
          class=" max-w-6xl mx-auto grid grid-cols-3 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-6 xl:grid-cols-7 gap-6 sm:gap-4">
          <div v-for="(file, idx) in files" :key="file.id"
            class="group relative aspect-square rounded-xl overflow-hidden cursor-pointer transition-all duration-300 hover:shadow-xl hover:shadow-black/20 hover:-translate-y-1"
            :style="{ animationDelay: (idx % 12) * 50 + 'ms' }" @click="openPreview(file.url)">

            <!-- 图片 -->
            <img :src="file.url" :alt="file.name"
              class="w-full h-full object-cover transition-transform duration-500 group-hover:scale-110"
              loading="lazy" />

            <!-- 悬浮层 -->
            <div
              class="absolute inset-0 bg-gradient-to-t from-black/80 via-black/20 to-transparent opacity-0 group-hover:opacity-100 transition-all duration-300">
              <div class="absolute bottom-0 left-0 right-0 p-3">
                <p class="text-white text-sm font-medium truncate">{{ file.name }}</p>
              </div>
              <div class="absolute top-2 sm:top-3 right-2 sm:right-3 flex gap-1.5 sm:gap-2">
                <el-tooltip content="复制链接" placement="top">
                  <button @click.stop="copyUrl(file.url)"
                    class="p-1.5 sm:p-2 rounded-lg bg-white/20 backdrop-blur-sm hover:bg-white/30 transition-all">
                    <Copy class="w-3.5 h-3.5 sm:w-4 sm:h-4 text-white" />
                  </button>
                </el-tooltip>
              </div>
            </div>
          </div>
        </div>

        <!-- 列表视图 -->
        <div v-else-if="viewMode === 'list'" class="rounded-xl border overflow-hidden"
          :class="themeStore.isDark ? 'bg-[var(--bg-secondary)] border-[var(--border)]' : 'bg-white border-gray-200'">
          <div class="overflow-x-auto">
            <table class="w-full min-w-[500px]">
              <thead :class="themeStore.isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-50'">
                <tr>
                  <th class="p-3 sm:p-4 text-left text-sm font-medium"
                    :class="themeStore.isDark ? 'text-gray-400' : 'text-gray-500'">预览</th>
                  <th class="p-3 sm:p-4 text-left text-sm font-medium"
                    :class="themeStore.isDark ? 'text-gray-400' : 'text-gray-500'">文件名</th>
                  <th class="p-3 sm:p-4 text-left text-sm font-medium hidden sm:table-cell"
                    :class="themeStore.isDark ? 'text-gray-400' : 'text-gray-500'">大小</th>
                  <th class="p-3 sm:p-4 text-left text-sm font-medium hidden md:table-cell"
                    :class="themeStore.isDark ? 'text-gray-400' : 'text-gray-500'">上传时间</th>
                  <th class="p-3 sm:p-4 text-right text-sm font-medium"
                    :class="themeStore.isDark ? 'text-gray-400' : 'text-gray-500'">操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="file in files" :key="file.id"
                  class="border-t transition-all hover:bg-[var(--bg-hover)] cursor-pointer"
                  :class="themeStore.isDark ? 'border-[var(--border)]' : 'border-gray-100'"
                  @click="openPreview(file.url)">
                  <td class="p-3 sm:p-4">
                    <div class="w-12 h-12 rounded-lg overflow-hidden"
                      :class="themeStore.isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-100'">
                      <img :src="file.url" :alt="file.name" class="w-full h-full object-cover" />
                    </div>
                  </td>
                  <td class="p-3 sm:p-4">
                    <span class="font-medium truncate max-w-[150px] sm:max-w-[200px] block">{{ file.name }}</span>
                  </td>
                  <td class="p-3 sm:p-4 text-sm hidden sm:table-cell"
                    :class="themeStore.isDark ? 'text-gray-400' : 'text-gray-500'">
                    {{ formatSize(file.size) }}
                  </td>
                  <td class="p-3 sm:p-4 text-sm hidden md:table-cell"
                    :class="themeStore.isDark ? 'text-gray-400' : 'text-gray-500'">
                    {{ formatDate(file.createdAt) }}
                  </td>
                  <td class="p-3 sm:p-4 text-right">
                    <el-tooltip content="复制链接" placement="top">
                      <button @click.stop="copyUrl(file.url)"
                        class="p-1.5 sm:p-2 rounded-lg transition-all hover:bg-indigo-500/10">
                        <Copy class="w-4 h-4 text-indigo-500" />
                      </button>
                    </el-tooltip>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <!-- 分页 -->
        <div v-if="total > pageSize" class="flex justify-center mt-6 sm:mt-10">
          <div class="flex items-center gap-1">
            <button v-for="p in Math.min(7, Math.ceil(total / pageSize))" :key="p" @click="handlePageChange(p)"
              class="w-9 h-9 sm:w-10 sm:h-10 rounded-lg font-medium transition-all"
              :class="page === p
                ? 'bg-gradient-to-r from-indigo-500 to-purple-500 text-white shadow-lg'
                : (themeStore.isDark ? 'bg-[var(--bg-secondary)] hover:bg-[var(--bg-hover)] text-gray-300' : 'bg-white hover:bg-gray-100 text-gray-700')">
              {{ p }}
            </button>
          </div>
        </div>
      </main>
    </div>


    <!-- 预览弹窗 -->
    <el-dialog v-model="showPreview" width="90% sm:80%" top="5vh" :show-close="false" :close-on-click-modal="true">
      <div class="relative">
        <el-tooltip content="关闭预览" placement="top">
          <button @click="showPreview = false"
            class="absolute -top-10 sm:-top-12 right-0 p-2 rounded-lg text-white/80 hover:text-white transition-all">
            <X class="w-5 h-5 sm:w-6 sm:h-6" />
          </button>
        </el-tooltip>
        <img :src="previewUrl" class="w-full h-auto rounded-xl" />
      </div>
    </el-dialog>
  </div>
</template>
