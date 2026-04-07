<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useThemeStore } from '@/stores/theme'
import { uploadApi } from '@/api/upload'
import { fileApi } from '@/api/file'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Upload, Image, Folder, Link, Check, X, Trash2,
  Sun, Moon, FileText, Film, Video, RefreshCw
} from 'lucide-vue-next'

const router = useRouter()
const authStore = useAuthStore()
const themeStore = useThemeStore()

const uploadQueue = ref([])
const isUploading = ref(false)
const pasteEnabled = ref(true)
const isDragover = ref(false)
const fileInput = ref(null)
const maxFileSize = ref(20 * 1024 * 1024)

// 站点配置
const siteConfig = ref({
  name: 'ImgBed',
  logo: ''
})
const allowedTypes = ref([
  'image/jpeg', 'image/png', 'image/gif', 'image/webp', 'image/svg+xml', 'image/bmp',
  'video/mp4', 'video/webm', 'video/quicktime', 'video/x-msvideo',
  'audio/mpeg', 'audio/wav', 'audio/ogg', 'audio/flac',
  'application/pdf', 'application/zip'
])
const allowedExtensions = ref([
  '.jpg', '.jpeg', '.png', '.gif', '.webp', '.svg', '.bmp',
  '.mp4', '.webm', '.mov', '.avi',
  '.mp3', '.wav', '.ogg', '.flac',
  '.pdf', '.zip'
])

const stats = computed(() => {
  const t = uploadQueue.value.length
  const c = uploadQueue.value.filter(f => f.status === 'done').length
  const f = uploadQueue.value.filter(f => f.status === 'error').length
  return { total: t, done: c, failed: f }
})

function getOrigin() {
  return window.location.origin
}

onMounted(() => {
  themeStore.init()
  document.addEventListener('paste', handlePaste)
  loadConfig()
})

onUnmounted(() => {
  document.removeEventListener('paste', handlePaste)
})

async function loadConfig() {
  try {
    const [uploadRes, siteRes] = await Promise.all([
      fetch('/api/v1/config/upload').then(r => r.json()),
      fetch('/api/v1/config/site').then(r => r.json())
    ])
    if (uploadRes.code === 0 && uploadRes.data?.maxSize) {
      maxFileSize.value = uploadRes.data.maxSize
    }
    if (siteRes.code === 0 && siteRes.data) {
      siteConfig.value = {
        name: siteRes.data.name || 'ImgBed',
        logo: siteRes.data.logo || ''
      }
    }
  } catch { }
}

function handlePaste(e) {
  if (!pasteEnabled.value) return
  if (!authStore.isAuthenticated) {
    ElMessage.warning('请先登录')
    return
  }
  const items = e.clipboardData?.items
  if (!items) return
  for (const item of items) {
    if (item.kind === 'file') {
      const file = item.getAsFile()
      if (file) addToQueue(file)
    }
  }
}

function handleDrop(e) {
  e.preventDefault()
  isDragover.value = false
  if (!authStore.isAuthenticated) {
    ElMessage.warning('请先登录')
    return
  }
  const files = e.dataTransfer?.files
  if (files) {
    for (const file of files) addToQueue(file)
  }
}

function handleDragOver(e) {
  e.preventDefault()
  isDragover.value = true
}

function handleDragLeave() {
  isDragover.value = false
}

function handleFileChange(e) {
  const files = e.target.files
  if (files) {
    for (const file of files) addToQueue(file)
  }
  e.target.value = ''
}

function triggerFileInput() {
  if (!authStore.isAuthenticated) {
    ElMessage.warning('请先登录')
    router.push('/login')
    return
  }
  fileInput.value?.click()
}

function validateFileType(file) {
  if (file.type && allowedTypes.value.some(t => {
    if (t.endsWith('/*')) {
      return file.type.startsWith(t.slice(0, -1))
    }
    return file.type === t
  })) {
    return true
  }

  const ext = '.' + file.name.split('.').pop().toLowerCase()
  if (allowedExtensions.value.includes(ext)) {
    return true
  }

  return false
}

function addToQueue(file) {
  if (file.size > maxFileSize.value) {
    const limitMB = Math.round(maxFileSize.value / (1024 * 1024))
    ElMessage.warning(`${file.name} 超过 ${limitMB}MB 限制`)
    return
  }

  if (!validateFileType(file)) {
    ElMessage.warning(`${file.name} 文件类型不支持`)
    return
  }

  const item = {
    id: Date.now() + Math.random(),
    file,
    name: file.name,
    size: file.size,
    type: file.type,
    status: 'pending',
    progress: 0,
    url: '',
    error: ''
  }

  uploadQueue.value.unshift(item)
  processQueue()
}

async function calcFileSHA256(file) {
  const buffer = await file.arrayBuffer()
  const hashBuffer = await crypto.subtle.digest('SHA-256', buffer)
  const hashArray = Array.from(new Uint8Array(hashBuffer))
  return hashArray.map(b => b.toString(16).padStart(2, '0')).join('')
}

async function processQueue() {
  if (isUploading.value) return
  const pending = uploadQueue.value.filter(f => f.status === 'pending')
  if (!pending.length) return

  isUploading.value = true
  for (const item of pending) {
    item.status = 'uploading'
    try {
      item.progress = 5
      const checksum = await calcFileSHA256(item.file)
      item.progress = 10
      const existRes = await fileApi.checkExists(checksum)
      item.progress = 30
      if (existRes.code === 0 && existRes.data?.exists) {
        item.status = 'done'
        item.fileId = existRes.data.file?.id
        item.url = existRes.data.file?.url || `/api/v1/file/${existRes.data.file?.id}`
        item.progress = 100
        continue
      }

      item.progress = 30
      const res = await uploadApi.upload(item.file, {
        onProgress: (p) => { item.progress = 30 + p * 0.7 }
      })
      if (res.code === 0) {
        item.status = 'done'
        item.fileId = res.data?.id
        item.url = res.data?.url || `/api/v1/file/${res.data?.id}`
        item.progress = 100
      } else {
        throw new Error(res.message)
      }
    } catch (err) {
      item.status = 'error'
      item.error = err.message || '上传失败'
    }
  }
  isUploading.value = false
}

async function removeItem(id) {
  const item = uploadQueue.value.find(f => f.id === id)
  if (!item) return

  if (item.status === 'done' && item.fileId) {
    try {
      await ElMessageBox.confirm('确定要从服务器删除此文件吗？', '确认删除', {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning'
      })
      await fileApi.deleteFile(item.fileId)
      ElMessage.success('已删除')
    } catch (err) {
      if (err?.response?.status === 404) {
        ElMessage.info('文件已在服务器上被删除')
      } else if (err?.response?.status === 401) {
        ElMessage.error('登录已过期，请重新登录')
        authStore.logout()
      } else if (err !== 'cancel') {
        ElMessage.error(err?.message || '删除失败')
      } else {
        return
      }
    }
  }

  const idx = uploadQueue.value.findIndex(f => f.id === id)
  if (idx > -1) uploadQueue.value.splice(idx, 1)
}

async function retryItem(item) {
  item.status = 'pending'
  item.progress = 0
  item.error = ''
  item.url = ''
  processQueue()
}

function clearDone() {
  uploadQueue.value = uploadQueue.value.filter(f => f.status !== 'done')
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

async function copyAllUrls() {
  const urls = uploadQueue.value
    .filter(f => f.status === 'done' && f.url)
    .map(f => {
      if (f.url.startsWith('http://') || f.url.startsWith('https://')) {
        return f.url
      }
      return window.location.origin + f.url
    })
    .join('\n')
  if (urls) {
    await navigator.clipboard.writeText(urls)
    ElMessage.success('全部链接已复制')
  }
}

function formatSize(bytes) {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

function getFileTypeIcon(type) {
  if (!type) return FileText
  if (type.startsWith('image/')) return Image
  if (type.startsWith('video/')) return Film
  if (type.startsWith('audio/')) return Video
  return FileText
}

function isImageType(type) {
  return type?.startsWith('image/')
}
</script>

<template>
  <div class="h-screen bg-[var(--bg-primary)] transition-colors duration-300 flex flex-col overflow-hidden">
    <header class="flex-shrink-0 border-b backdrop-blur-xl"
      :class="themeStore.isDark ? 'bg-[var(--bg-primary)]/80 border-[var(--border)]' : 'bg-white/80 border-gray-200'">
      <div class="max-w-6xl mx-auto px-4 sm:px-6 py-3 sm:py-4 flex items-center justify-between gap-4">
        <div class="flex items-center gap-2 sm:gap-3">
          <img v-if="siteConfig.logo" :src="siteConfig.logo" :alt="siteConfig.name"
            class="w-8 h-8 sm:w-10 sm:h-10 rounded-xl object-cover shadow-lg shadow-indigo-500/30" />
          <img v-else src="/imgbed.webp" :alt="siteConfig.name"
            class="w-8 h-8 sm:w-10 sm:h-10 rounded-xl object-cover shadow-lg shadow-indigo-500/30" />
          <span class="text-lg sm:text-xl font-bold">
            <span class="text-gradient">{{ siteConfig.name.slice(0, 3) }}</span><span
              :class="themeStore.isDark ? 'text-white' : 'text-gray-800'">{{ siteConfig.name.slice(3) }}</span>
          </span>
        </div>

        <nav class="flex items-center gap-1">
          <button @click="router.push('/gallery')"
            class="p-2 sm:px-4 rounded-lg text-sm font-medium transition-all flex items-center gap-1"
            :class="themeStore.isDark ? 'text-gray-400 hover:text-white hover:bg-white/5' : 'text-gray-600 hover:text-gray-900 hover:bg-gray-100'">
            <Folder class="w-4 h-4" />
            <span class="hidden sm:inline">我的图库</span>
          </button>

          <div class="w-px h-5 sm:h-6 mx-1 sm:mx-2" :class="themeStore.isDark ? 'bg-gray-700' : 'bg-gray-300'"></div>

          <button @click="themeStore.toggle" class="p-2 rounded-lg transition-all"
            :class="themeStore.isDark ? 'text-gray-400 hover:text-white hover:bg-white/5' : 'text-gray-600 hover:text-gray-900 hover:bg-gray-100'">
            <Sun v-if="themeStore.isDark" class="w-5 h-5" />
            <Moon v-else class="w-5 h-5" />
          </button>

          <button v-if="!authStore.isAuthenticated" @click="router.push('/login')"
            class="ml-1 sm:ml-2 px-3 sm:px-4 py-2 rounded-lg text-sm font-medium text-white bg-gradient-to-r from-indigo-500 to-purple-500 hover:from-indigo-600 hover:to-purple-600 transition-all shadow-lg shadow-indigo-500/25 flex items-center gap-1">
            <span class="hidden sm:inline">登录</span>
            <span class="sm:hidden">登</span>
          </button>
          <button v-else @click="authStore.logout(); router.push('/')"
            class="p-2 sm:px-4 rounded-lg text-sm font-medium transition-all flex items-center gap-1"
            :class="themeStore.isDark ? 'text-gray-400 hover:text-white hover:bg-white/5' : 'text-gray-600 hover:text-gray-900 hover:bg-gray-100'">
            <span class="hidden sm:inline">退出</span>
          </button>
        </nav>
      </div>
    </header>

    <main class="flex-1 max-w-6xl mx-auto px-4 sm:px-6 py-4 sm:py-8 overflow-y-auto w-full">
      <div class="text-center mb-4 sm:mb-6">
        <h1 class="text-xl sm:text-2xl md:text-3xl font-bold mb-1 sm:mb-2">
          <span class="text-gradient">文件上传</span>
        </h1>
        <p class="text-sm sm:text-base" :class="themeStore.isDark ? 'text-gray-400' : 'text-gray-500'">
          {{ authStore.isAuthenticated ? '拖拽、粘贴或选择文件上传，最大 20MB' : '请先登录后上传文件' }}
        </p>
      </div>

      <div
        class="relative rounded-xl sm:rounded-2xl border-2 border-dashed transition-all duration-300 cursor-pointer overflow-hidden"
        :class="isDragover
          ? 'border-indigo-500 bg-indigo-500/5'
          : themeStore.isDark ? 'border-gray-700 bg-[var(--bg-secondary)]' : 'border-gray-300 bg-gray-50'"
        @drop.prevent="handleDrop" @dragover.prevent="handleDragOver" @dragleave.prevent="handleDragLeave"
        @click="triggerFileInput">
        <input ref="fileInput" type="file" multiple class="hidden" @change="handleFileChange" />

        <div class="py-8 sm:py-12 text-center">
          <div
            class="w-16 h-16 sm:w-20 sm:h-20 mx-auto mb-4 sm:mb-5 rounded-2xl bg-gradient-to-br from-indigo-500/20 to-purple-500/20 flex items-center justify-center">
            <Upload class="w-8 h-8 sm:w-10 sm:h-10 text-indigo-500" />
          </div>
          <p class="text-base sm:text-lg font-medium mb-1 sm:mb-2">
            <template v-if="authStore.isAuthenticated">
              将文件<span class="text-indigo-500">拖拽到此处</span>，或点击选择
            </template>
            <template v-else>
              <span class="text-indigo-500" @click.stop="router.push('/login')">点击登录</span> 后上传文件
            </template>
          </p>
          <p class="text-xs sm:text-sm" :class="themeStore.isDark ? 'text-gray-500' : 'text-gray-400'">
            支持 JPG、PNG、GIF、WebP、SVG、BMP 等图片格式
          </p>
        </div>
      </div>

      <div class="flex flex-wrap items-center gap-2 sm:gap-3 mt-3 sm:mt-4">
        <el-tooltip v-if="authStore.isAuthenticated" content="开启后可通过 Ctrl+V 粘贴图片上传" placement="top">
          <label class="flex items-center gap-2 text-sm cursor-pointer">
            <input type="checkbox" v-model="pasteEnabled" class="w-4 h-4 rounded accent-indigo-500" />
            <span :class="themeStore.isDark ? 'text-gray-400' : 'text-gray-500'">粘贴</span>
          </label>
        </el-tooltip>

        <div class="flex-1"></div>

        <el-tooltip v-if="stats.done > 0" content="复制所有上传成功的文件链接" placement="top">
          <button @click="copyAllUrls"
            class="px-3 sm:px-4 py-2 rounded-xl text-sm font-medium text-white bg-gradient-to-r from-indigo-500 to-purple-500 hover:from-indigo-600 hover:to-purple-600 transition-all shadow-lg shadow-indigo-500/25">
            <span class="hidden sm:inline">复制全部链接</span>
            <span class="sm:hidden">复制</span>
          </button>
        </el-tooltip>

        <el-tooltip v-if="stats.done > 0" content="清空已完成的记录" placement="top">
          <button @click="clearDone" class="px-3 sm:px-4 py-2 rounded-xl text-sm font-medium transition-all"
            :class="themeStore.isDark ? 'text-gray-400 hover:text-white hover:bg-white/5' : 'text-gray-500 hover:text-gray-700 hover:bg-gray-100'">
            清空
          </button>
        </el-tooltip>
      </div>

      <div v-if="uploadQueue.length > 0" class="mt-3 sm:mt-4 space-y-2 sm:space-y-3 overflow-y-auto"
        style="max-height: calc(100vh - 420px);">
        <TransitionGroup name="list">
          <div v-for="item in uploadQueue" :key="item.id"
            class="group flex items-center gap-2 sm:gap-4 p-3 sm:p-4 rounded-xl border transition-all duration-300 hover:shadow-lg"
            :class="themeStore.isDark ? 'bg-[var(--bg-secondary)] border-[var(--border)]' : 'bg-white border-gray-100'">

            <div class="w-10 h-10 sm:w-12 sm:h-12 rounded-lg overflow-hidden flex-shrink-0"
              :class="themeStore.isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-100'">
              <img v-if="isImageType(item.type) && item.url" :src="item.url" class="w-full h-full object-cover" />
              <component v-else :is="getFileTypeIcon(item.type)" class="w-5 h-5 sm:w-6 sm:h-6 m-2 sm:m-3"
                :class="themeStore.isDark ? 'text-gray-500' : 'text-gray-400'" />
            </div>

            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2">
                <p class="font-medium text-sm sm:text-base truncate">{{ item.name }}</p>
                <Check v-if="item.status === 'done'" class="w-4 h-4 text-green-500 flex-shrink-0" />
                <X v-if="item.status === 'error'" class="w-4 h-4 text-red-500 flex-shrink-0" />
              </div>
              <div class="flex items-center gap-2 sm:gap-3 text-xs mt-0.5">
                <span :class="themeStore.isDark ? 'text-gray-500' : 'text-gray-400'">{{ formatSize(item.size) }}</span>
                <span v-if="item.status === 'error'" class="text-red-500 text-xs">{{ item.error }}</span>
              </div>
              <div v-if="item.status === 'uploading'" class="mt-1.5 sm:mt-2 h-1 rounded-full overflow-hidden"
                :class="themeStore.isDark ? 'bg-gray-700' : 'bg-gray-200'">
                <div class="h-full bg-gradient-to-r from-indigo-500 to-purple-500 transition-all duration-300"
                  :style="{ width: item.progress + '%' }"></div>
              </div>
            </div>

            <div class="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-all">
              <el-tooltip v-if="item.status === 'error'" content="重试" placement="top">
                <button @click="retryItem(item)" class="p-1.5 sm:p-2 rounded-lg transition-all hover:bg-orange-500/10">
                  <RefreshCw class="w-4 h-4 text-orange-500" />
                </button>
              </el-tooltip>
              <el-tooltip v-if="item.status === 'done'" content="复制链接" placement="top">
                <button @click="copyUrl(item.url)"
                  class="p-1.5 sm:p-2 rounded-lg transition-all hover:bg-indigo-500/10">
                  <Link class="w-4 h-4 text-indigo-500" />
                </button>
              </el-tooltip>
              <el-tooltip content="移除" placement="top">
                <button @click="removeItem(item.id)" class="p-1.5 sm:p-2 rounded-lg transition-all hover:bg-red-500/10">
                  <Trash2 class="w-4 h-4 text-red-500" />
                </button>
              </el-tooltip>
            </div>
          </div>
        </TransitionGroup>
      </div>

    </main>
  </div>
</template>
