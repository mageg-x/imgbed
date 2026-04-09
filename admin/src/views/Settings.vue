<script setup>
import { ref, onMounted, reactive, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import request from '@/api/request'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Upload, Image, Lock, RefreshCw, Settings, Shield, Globe, Database } from 'lucide-vue-next'
import { createBackup, listBackups, deleteBackup, restoreBackup } from '@/api/backup'

const { t } = useI18n()
const isDark = ref(true)
const loading = ref(false)
const activeTab = ref('upload')

const uploadConfig = reactive({
  maxSize: 20,
  chunkSize: 5,
  defaultChannel: '',
  allowedTypes: '',
  autoRetry: true,
  retryCount: 3,
  compressionEnabled: true,
  compressionQuality: 80,
  compressionFormat: 'webp',
  compressionMaxWidth: 1920,
  compressionMaxHeight: 1080
})

const siteConfig = reactive({
  name: 'ImgBed',
  logo: ''
})

const authConfig = reactive({
  userPassword: '',
  adminUsername: 'admin',
  adminPassword: '',
  sessionTimeout: 86400
})

const scheduleConfig = reactive({
  strategy: 'priority'
})

const rateLimitConfig = reactive({
  enabled: true,
  rateLimit: 10,
  dailyLimit: 100,
  maxFileSize: 5
})

const cdnConfig = reactive({
  enabled: false,
  proxyUrl: '',
  cdnUrl: ''
})

// 备份相关
const backupLoading = ref(false)
const backups = ref([])
const creating = ref(false)
const deleting = ref(null)
const restoring = ref(null)

// 备份配置
const backupConfig = reactive({
  enabled: true,
  interval: 24
})

onMounted(() => {
  isDark.value = !document.documentElement.classList.contains('light')
  loadConfigs()
  loadBackups()
})

async function loadConfigs() {
  loading.value = true
  try {
    const [uploadRes, appRes, jwtRes, siteRes, authRes, scheduleRes, rateLimitRes, cdnRes, backupRes] = await Promise.all([
      request.get('/config/upload').catch(() => ({ data: null })),
      request.get('/config/app').catch(() => ({ data: null })),
      request.get('/config/jwt').catch(() => ({ data: null })),
      request.get('/config/site').catch(() => ({ data: null })),
      request.get('/config/auth').catch(() => ({ data: null })),
      request.get('/config/schedule').catch(() => ({ data: null })),
      request.get('/config/rate-limit').catch(() => ({ data: null })),
      request.get('/config/cdn').catch(() => ({ data: null })),
      request.get('/config/backup').catch(() => ({ data: null }))
    ])

    if (uploadRes.data) {
      uploadConfig.maxSize = Math.round(uploadRes.data.maxSize / (1024 * 1024)) || 20
      uploadConfig.chunkSize = Math.round(uploadRes.data.chunkSize / (1024 * 1024)) || 5
      uploadConfig.defaultChannel = uploadRes.data.defaultChannel || ''
      uploadConfig.allowedTypes = (uploadRes.data.allowedTypes || []).join(',')
      uploadConfig.autoRetry = uploadRes.data.autoRetry !== false
      uploadConfig.retryCount = uploadRes.data.retryCount || 3
      // 压缩配置
      if (uploadRes.data.compression) {
        uploadConfig.compressionEnabled = uploadRes.data.compression.enabled !== false
        uploadConfig.compressionQuality = uploadRes.data.compression.quality || 80
        uploadConfig.compressionFormat = uploadRes.data.compression.format || 'webp'
        uploadConfig.compressionMaxWidth = uploadRes.data.compression.maxWidth || 1920
        uploadConfig.compressionMaxHeight = uploadRes.data.compression.maxHeight || 1080
      }
    }

    if (appRes.data) {
      appConfig.host = appRes.data.host || '0.0.0.0'
      appConfig.port = appRes.data.port || 8080
      appConfig.mode = appRes.data.mode || 'debug'
    }

    if (jwtRes.data) {
      jwtConfig.expire = jwtRes.data.expire || 86400
    }

    if (siteRes.data) {
      siteConfig.name = siteRes.data.name || 'ImgBed'
      siteConfig.logo = siteRes.data.logo || ''
    }

    if (authRes.data) {
      authConfig.userPassword = authRes.data.userPassword || ''
      authConfig.adminUsername = authRes.data.adminUsername || 'admin'
      authConfig.sessionTimeout = authRes.data.sessionTimeout || 86400
    }

    if (scheduleRes.data) {
      scheduleConfig.strategy = scheduleRes.data.strategy || 'priority'
    }

    if (rateLimitRes.data) {
      rateLimitConfig.enabled = rateLimitRes.data.enabled !== false
      rateLimitConfig.rateLimit = rateLimitRes.data.rateLimit || 10
      rateLimitConfig.dailyLimit = rateLimitRes.data.dailyLimit || 100
      rateLimitConfig.maxFileSize = Math.round((rateLimitRes.data.maxFileSize || 5242880) / (1024 * 1024))
    }

    if (cdnRes && cdnRes.data) {
      cdnConfig.enabled = cdnRes.data.enabled || false
      cdnConfig.proxyUrl = cdnRes.data.proxyUrl || ''
      cdnConfig.cdnUrl = cdnRes.data.cdnUrl || ''
    }

    if (backupRes && backupRes.data) {
      backupConfig.enabled = backupRes.data.enabled !== false
      backupConfig.interval = backupRes.data.interval || 24
    }
  } catch {
    ElMessage.error(t('common.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function saveAppConfig() {
  try {
    await request.put('/config/app', {
      host: appConfig.host,
      port: appConfig.port
    })
    ElMessage.success(t('settings.saveSuccess'))
  } catch {
    ElMessage.error(t('settings.saveFailed'))
  }
}

async function saveJwtConfig() {
  try {
    await request.put('/config/jwt', {
      expire: jwtConfig.expire
    })
    ElMessage.success(t('settings.saveSuccess'))
  } catch {
    ElMessage.error(t('settings.saveFailed'))
  }
}

async function saveUploadConfig() {
  try {
    await request.put('/config/upload', {
      maxSize: uploadConfig.maxSize * 1024 * 1024,
      chunkSize: uploadConfig.chunkSize * 1024 * 1024,
      defaultChannel: uploadConfig.defaultChannel,
      allowedTypes: uploadConfig.allowedTypes.split(',').filter(t => t.trim()),
      autoRetry: uploadConfig.autoRetry,
      retryCount: uploadConfig.retryCount,
      compression: {
        enabled: uploadConfig.compressionEnabled,
        quality: uploadConfig.compressionQuality,
        format: uploadConfig.compressionFormat,
        maxWidth: uploadConfig.compressionMaxWidth,
        maxHeight: uploadConfig.compressionMaxHeight
      }
    })
    ElMessage.success(t('settings.saveSuccess'))
  } catch {
    ElMessage.error(t('settings.saveFailed'))
  }
}

async function saveSiteConfig() {
  try {
    await request.put('/config/site', {
      name: siteConfig.name,
      logo: siteConfig.logo
    })
    ElMessage.success(t('settings.saveSuccess'))
  } catch {
    ElMessage.error(t('settings.saveFailed'))
  }
}

async function saveAuthConfig() {
  try {
    const data = {
      userPassword: authConfig.userPassword,
      adminUsername: authConfig.adminUsername,
      sessionTimeout: authConfig.sessionTimeout
    }
    if (authConfig.adminPassword) {
      data.adminPassword = authConfig.adminPassword
    }
    await request.put('/config/auth', data)
    ElMessage.success(t('settings.saveSuccess'))
  } catch {
    ElMessage.error(t('settings.saveFailed'))
  }
}

async function saveScheduleConfig() {
  try {
    await request.put('/config/schedule', {
      strategy: scheduleConfig.strategy
    })
    ElMessage.success(t('settings.saveSuccess'))
  } catch {
    ElMessage.error(t('settings.saveFailed'))
  }
}

async function saveRateLimitConfig() {
  try {
    await request.put('/config/rate-limit', {
      enabled: rateLimitConfig.enabled,
      rateLimit: rateLimitConfig.rateLimit,
      dailyLimit: rateLimitConfig.dailyLimit,
      maxFileSize: rateLimitConfig.maxFileSize * 1024 * 1024
    })
    ElMessage.success(t('settings.saveSuccess'))
  } catch {
    ElMessage.error(t('settings.saveFailed'))
  }
}

async function loadCdnConfig() {
  try {
    const res = await request.get('/config/cdn')
    if (res.data) {
      cdnConfig.enabled = res.data.enabled || false
      cdnConfig.proxyUrl = res.data.proxyUrl || ''
    }
  } catch {
    // ignore
  }
}

async function saveCdnConfig() {
  try {
    await request.put('/config/cdn', {
      enabled: cdnConfig.enabled,
      proxyUrl: cdnConfig.proxyUrl,
      cdnUrl: cdnConfig.cdnUrl
    })
    ElMessage.success(t('settings.saveSuccess'))
  } catch {
    ElMessage.error(t('settings.saveFailed'))
  }
}

async function saveBackupConfig() {
  try {
    await request.put('/config/backup', {
      enabled: backupConfig.enabled,
      interval: backupConfig.interval
    })
    ElMessage.success(t('settings.saveSuccess'))
  } catch {
    ElMessage.error(t('settings.saveFailed'))
  }
}

// 备份相关方法
async function loadBackups() {
  backupLoading.value = true
  try {
    const res = await listBackups()
    if (res.code === 0) {
      backups.value = res.data || []
    }
  } catch {
    ElMessage.error(t('backup.loadError'))
  } finally {
    backupLoading.value = false
  }
}

async function handleCreateBackup() {
  creating.value = true
  try {
    const res = await createBackup()
    if (res.code === 0) {
      ElMessage.success(t('backup.createSuccess'))
      await loadBackups()
    }
  } catch {
    ElMessage.error(t('backup.createError'))
  } finally {
    creating.value = false
  }
}

async function handleDeleteBackup(backupPath) {
  try {
    await ElMessageBox.confirm(
      t('backup.deleteConfirm'),
      t('backup.deleteConfirmTitle'),
      {
        confirmButtonText: t('backup.confirm'),
        cancelButtonText: t('backup.cancel'),
        type: 'warning'
      }
    )
    
    deleting.value = backupPath
    const res = await deleteBackup(backupPath)
    if (res.code === 0) {
      ElMessage.success(t('backup.deleteSuccess'))
      await loadBackups()
    }
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(t('backup.deleteError'))
    }
  } finally {
    deleting.value = null
  }
}

async function handleRestoreBackup(backupPath) {
  try {
    await ElMessageBox.confirm(
      t('backup.restoreConfirm'),
      t('backup.restoreConfirmTitle'),
      {
        confirmButtonText: t('backup.confirm'),
        cancelButtonText: t('backup.cancel'),
        type: 'warning'
      }
    )
    
    restoring.value = backupPath
    const res = await restoreBackup(backupPath)
    if (res.code === 0) {
      ElMessage.success(t('backup.restoreSuccess'))
      await ElMessageBox.alert(
        t('backup.restoreNotice'),
        t('backup.restoreNoticeTitle'),
        {
          confirmButtonText: t('backup.confirm')
        }
      )
    }
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(t('backup.restoreError'))
    }
  } finally {
    restoring.value = null
  }
}

const formatSize = (size) => {
  if (size < 1024) {
    return size + ' B'
  } else if (size < 1024 * 1024) {
    return (size / 1024).toFixed(2) + ' KB'
  } else {
    return (size / (1024 * 1024)).toFixed(2) + ' MB'
  }
}

const tabs = computed(() => [
  { name: 'upload', label: t('settings.uploadSettings'), icon: Upload },
  { name: 'app', label: t('settings.appSettings'), icon: Settings },
  { name: 'jwt', label: t('settings.jwtSettings'), icon: Shield },
  { name: 'site', label: t('settings.siteSettings'), icon: Image },
  { name: 'auth', label: t('settings.authSettings'), icon: Lock },
  { name: 'schedule', label: t('settings.scheduleSettings'), icon: Settings },
  { name: 'rateLimit', label: t('settings.rateLimitSettings'), icon: Shield },
  { name: 'cdn', label: t('settings.cdnSettings'), icon: Globe },
  { name: 'backup', label: t('settings.backupSettings'), icon: Database }
])

const appConfig = reactive({
  host: '0.0.0.0',
  port: 8080,
  mode: 'debug'
})

const jwtConfig = reactive({
  expire: 86400
})
</script>

<template>
  <div class="flex flex-col h-full">
    <!-- 设置卡片 -->
    <div class="card overflow-hidden flex-1 flex flex-col">
      <!-- 标签页导航 -->
      <div class="tabs-nav flex border-b overflow-x-auto" :style="{ borderColor: 'var(--border)' }">
        <button v-for="tab in tabs" :key="tab.name" @click="activeTab = tab.name"
          class="flex items-center gap-1.5 sm:gap-2 px-3 sm:px-4 py-3 font-medium whitespace-nowrap transition-all border-b-2 text-sm flex-shrink-0"
          :class="activeTab === tab.name
            ? 'text-indigo-500 border-indigo-500'
            : isDark ? 'text-gray-400 border-transparent hover:text-white' : 'text-gray-500 border-transparent hover:text-gray-800'">
          <component :is="tab.icon" class="w-4 h-4" />
          <span class="hidden md:inline">{{ tab.label }}</span>
        </button>
      </div>

      <!-- 内容区 -->
      <div class="p-3 sm:p-6 settings-content flex-1 overflow-auto">
        <div v-if="loading" class="space-y-4">
          <div v-for="i in 4" :key="i" class="h-12 rounded-xl loading-shimmer"></div>
        </div>

        <!-- 上传设置 -->
        <div v-else-if="activeTab === 'upload'" class="max-w-2xl space-y-4 sm:space-y-6">
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 sm:gap-6">
            <div>
              <label class="block text-sm font-medium mb-2">
                <el-tooltip :content="t('settings.upload.maxSizeTip')" placement="top">
                  <span class="cursor-help">{{ t('settings.upload.maxSize') }}</span>
                </el-tooltip>
              </label>
              <input v-model.number="uploadConfig.maxSize" type="number" min="1" max="100"
                class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
                :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
            </div>
            <div>
              <label class="block text-sm font-medium mb-2">
                <el-tooltip :content="t('settings.upload.chunkSizeTip')" placement="top">
                  <span class="cursor-help">{{ t('settings.upload.chunkSize') }}</span>
                </el-tooltip>
              </label>
              <input v-model.number="uploadConfig.chunkSize" type="number" min="1" max="50"
                class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
                :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
            </div>
          </div>

          <div>
            <label class="block text-sm font-medium mb-2">
              <el-tooltip :content="t('settings.upload.defaultChannelTip')" placement="top">
                <span class="cursor-help">{{ t('settings.upload.defaultChannel') }}</span>
              </el-tooltip>
            </label>
            <input v-model="uploadConfig.defaultChannel" type="text" :placeholder="t('settings.upload.defaultChannelPlaceholder')"
              class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
              :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
          </div>

          <div>
            <label class="block text-sm font-medium mb-2">
              <el-tooltip :content="t('settings.upload.allowedTypesTip')" placement="top">
                <span class="cursor-help">{{ t('settings.upload.allowedTypes') }}</span>
              </el-tooltip>
            </label>
            <input v-model="uploadConfig.allowedTypes" type="text" :placeholder="t('settings.upload.allowedTypesPlaceholder')"
              class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
              :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
            <p class="text-xs mt-1.5" :class="isDark ? 'text-gray-500' : 'text-gray-400'">{{ t('settings.upload.allowedTypesHint') }}</p>
          </div>

          <div class="flex items-center justify-between p-3 sm:p-4 rounded-xl"
            :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-50'">
            <div>
              <p class="font-medium text-sm">
                <el-tooltip :content="t('settings.upload.autoRetryTip')" placement="top">
                  <span class="cursor-help">{{ t('settings.upload.autoRetry') }}</span>
                </el-tooltip>
              </p>
              <p class="text-xs" :class="isDark ? 'text-gray-500' : 'text-gray-400'">{{ t('settings.upload.autoRetryDesc') }}</p>
            </div>
            <input type="checkbox" v-model="uploadConfig.autoRetry"
              class="w-4 h-4 sm:w-5 sm:h-5 rounded cursor-pointer accent-indigo-500" />
          </div>

          <div v-if="uploadConfig.autoRetry">
            <label class="block text-sm font-medium mb-2">
              <el-tooltip :content="t('settings.upload.retryCountTip')" placement="top">
                <span class="cursor-help">{{ t('settings.upload.retryCount') }}</span>
              </el-tooltip>
            </label>
            <input v-model.number="uploadConfig.retryCount" type="number" min="1" max="10"
              class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 max-w-[200px] text-sm"
              :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
          </div>

          <!-- 图片压缩配置 -->
          <div  :style="{ borderColor: 'var(--border)' }">
            <h3 class=" font-normal text-sm mb-3">{{ t('settings.upload.compressionConfig') }}</h3>

            <div class="flex items-center justify-between p-3 sm:p-4 rounded-xl mb-4"
              :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-50'">
              <div>
                <el-tooltip :content="t('settings.upload.compressionEnabledTip')" placement="top">
                  <span class="font-medium text-sm cursor-help">{{ t('settings.upload.compressionEnabled') }}</span>
                </el-tooltip>
                <p class="text-xs" :class="isDark ? 'text-gray-500' : 'text-gray-400'">{{ t('settings.upload.compressionEnabledDesc') }}</p>
              </div>
              <input type="checkbox" v-model="uploadConfig.compressionEnabled"
                class="w-4 h-4 sm:w-5 sm:h-5 rounded cursor-pointer accent-indigo-500" />
            </div>

            <div v-if="uploadConfig.compressionEnabled" class="space-y-4">
              <div class="grid grid-cols-2 gap-4">
                <div>
                  <label class="block text-sm font-medium mb-2">
                    <el-tooltip :content="t('settings.upload.compressionQualityTip')" placement="top">
                      <span class="cursor-help">{{ t('settings.upload.compressionQuality') }}</span>
                    </el-tooltip>
                  </label>
                  <input v-model.number="uploadConfig.compressionQuality" type="number" min="1" max="100"
                    class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
                    :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
                  <p class="text-xs mt-1" :class="isDark ? 'text-gray-500' : 'text-gray-400'">{{ t('settings.upload.compressionQualityHint') }}</p>
                </div>
                <div>
                  <label class="block text-sm font-medium mb-2">
                    <el-tooltip :content="t('settings.upload.compressionFormatTip')" placement="top">
                      <span class="cursor-help">{{ t('settings.upload.compressionFormat') }}</span>
                    </el-tooltip>
                  </label>
                  <select v-model="uploadConfig.compressionFormat"
                    class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
                    :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'">
                    <option value="webp">{{ t('settings.upload.formatWebp') }}</option>
                    <option value="jpeg">{{ t('settings.upload.formatJpeg') }}</option>
                    <option value="png">{{ t('settings.upload.formatPng') }}</option>
                    <option value="original">{{ t('settings.upload.formatOriginal') }}</option>
                  </select>
                </div>
              </div>

              <div class="grid grid-cols-2 gap-4">
                <div>
                  <label class="block text-sm font-medium mb-2">
                    <el-tooltip :content="t('settings.upload.compressionMaxWidthTip')" placement="top">
                      <span class="cursor-help">{{ t('settings.upload.compressionMaxWidth') }}</span>
                    </el-tooltip>
                  </label>
                  <input v-model.number="uploadConfig.compressionMaxWidth" type="number" min="100" max="10000"
                    class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
                    :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
                </div>
                <div>
                  <label class="block text-sm font-medium mb-2">
                    <el-tooltip :content="t('settings.upload.compressionMaxHeightTip')" placement="top">
                      <span class="cursor-help">{{ t('settings.upload.compressionMaxHeight') }}</span>
                    </el-tooltip>
                  </label>
                  <input v-model.number="uploadConfig.compressionMaxHeight" type="number" min="100" max="10000"
                    class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
                    :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
                </div>
              </div>
            </div>
          </div>

          <button @click="saveUploadConfig"
            class="btn-gradient px-5 sm:px-6 py-2 sm:py-2.5 rounded-xl text-sm w-full sm:w-auto">
            {{ t('settings.upload.saveUploadConfig') }}
          </button>
        </div>

        <!-- 应用设置 -->
        <div v-else-if="activeTab === 'app'" class="max-w-2xl space-y-4 sm:space-y-6">
          <div>
            <label class="block text-sm font-medium mb-2">
              <el-tooltip :content="t('settings.app.listenAddressTip')" placement="top">
                <span class="cursor-help">{{ t('settings.app.listenAddress') }}</span>
              </el-tooltip>
            </label>
            <input v-model="appConfig.host" type="text" :placeholder="t('settings.app.listenAddressPlaceholder')"
              class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
              :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
            <p class="text-xs mt-1.5" :class="isDark ? 'text-gray-500' : 'text-gray-400'">{{ t('settings.app.listenAddressDesc') }}</p>
          </div>

          <div>
            <label class="block text-sm font-medium mb-2">
              <el-tooltip :content="t('settings.app.listenPortTip')" placement="top">
                <span class="cursor-help">{{ t('settings.app.listenPort') }}</span>
              </el-tooltip>
            </label>
            <input v-model.number="appConfig.port" type="number" min="1" max="65535" :placeholder="t('settings.app.listenPortPlaceholder')"
              class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 max-w-[200px] text-sm"
              :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
          </div>

          <div  :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-50'">
            <h3 class=" font-normal text-sm mb-2">
              <el-tooltip :content="t('settings.app.runModeTip')" placement="top">
                <span class="cursor-help">{{ t('settings.app.runMode') }}</span>
              </el-tooltip>
            </h3>
            <div class="flex gap-4">
              <label class="flex items-center gap-2 cursor-pointer">
                <input type="radio" v-model="appConfig.mode" value="debug" class="accent-indigo-500" />
                <span class="text-sm">{{ t('settings.app.debug') }}</span>
              </label>
              <label class="flex items-center gap-2 cursor-pointer">
                <input type="radio" v-model="appConfig.mode" value="release" class="accent-indigo-500" />
                <span class="text-sm">{{ t('settings.app.release') }}</span>
              </label>
            </div>
            <p class="text-xs mt-2" :class="isDark ? 'text-gray-500' : 'text-gray-400'">
              {{ t('settings.app.runModeDesc') }}
            </p>
          </div>

          <button @click="saveAppConfig"
            class="btn-gradient px-5 sm:px-6 py-2 sm:py-2.5 rounded-xl text-sm w-full sm:w-auto">
            {{ t('settings.app.saveAppSettings') }}
          </button>
        </div>

        <!-- JWT设置 -->
        <div v-else-if="activeTab === 'jwt'" class="max-w-2xl space-y-4 sm:space-y-6">
          <div>
            <label class="block text-sm font-medium mb-2">
              <el-tooltip :content="t('settings.jwt.secretTip')" placement="top">
                <span class="cursor-help">{{ t('settings.jwt.secret') }}</span>
              </el-tooltip>
            </label>
            <input v-model="jwtConfig.secret" type="password" :placeholder="t('settings.jwt.secretPlaceholder')"
              class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
              :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
            <p class="text-xs mt-1.5" :class="isDark ? 'text-gray-500' : 'text-gray-400'">{{ t('settings.jwt.secretDesc') }}</p>
          </div>

          <div>
            <label class="block text-sm font-medium mb-2">
              <el-tooltip :content="t('settings.jwt.expireTip')" placement="top">
                <span class="cursor-help">{{ t('settings.jwt.expire') }}</span>
              </el-tooltip>
            </label>
            <input v-model.number="jwtConfig.expire" type="number" min="3600" max="604800"
              class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 max-w-[200px] text-sm"
              :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
            <p class="text-xs mt-1.5" :class="isDark ? 'text-gray-500' : 'text-gray-400'">{{ t('settings.jwt.expireDesc') }}</p>
          </div>

          <button @click="saveJwtConfig"
            class="btn-gradient px-5 sm:px-6 py-2 sm:py-2.5 rounded-xl text-sm w-full sm:w-auto">
            {{ t('settings.jwt.saveJwtSettings') }}
          </button>
        </div>

        <!-- 站点设置 -->
        <div v-else-if="activeTab === 'site'" class="max-w-2xl space-y-4 sm:space-y-6">
          <div>
            <label class="block text-sm font-medium mb-2">
              <el-tooltip :content="t('settings.site.siteNameTip')" placement="top">
                <span class="cursor-help">{{ t('settings.site.siteName') }}</span>
              </el-tooltip>
            </label>
            <input v-model="siteConfig.name" type="text" :placeholder="t('settings.site.siteNamePlaceholder')"
              class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
              :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
          </div>

          <div>
            <label class="block text-sm font-medium mb-2">
              <el-tooltip :content="t('settings.site.logoUrlTip')" placement="top">
                <span class="cursor-help">{{ t('settings.site.logoUrl') }}</span>
              </el-tooltip>
            </label>
            <input v-model="siteConfig.logo" type="text" :placeholder="t('settings.site.logoUrlPlaceholder')"
              class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
              :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
          </div>

          <button @click="saveSiteConfig"
            class="btn-gradient px-5 sm:px-6 py-2 sm:py-2.5 rounded-xl text-sm w-full sm:w-auto">
            {{ t('settings.site.saveSiteSettings') }}
          </button>
        </div>

        <!-- 认证设置 -->
        <div v-else-if="activeTab === 'auth'" class="max-w-2xl space-y-4 sm:space-y-6">
          <div>
            <label class="block text-sm font-medium mb-2">
              <el-tooltip :content="t('settings.auth.userPasswordTip')" placement="top">
                <span class="cursor-help">{{ t('settings.auth.userPassword') }}</span>
              </el-tooltip>
            </label>
            <input v-model="authConfig.userPassword" type="password" :placeholder="t('settings.auth.userPasswordPlaceholder')"
              class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
              :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
            <p class="text-xs mt-1.5" :class="isDark ? 'text-gray-500' : 'text-gray-400'">{{ t('settings.auth.userPasswordDesc') }}</p>
          </div>

          <div  :style="{ borderColor: 'var(--border)' }">
            <h3 class="font-normal text-sm mb-3 sm:mb-4">{{ t('settings.auth.adminSettings') }}</h3>

            <div class="space-y-4">
              <div>
                <label class="block text-sm font-medium mb-2">
                  <el-tooltip :content="t('settings.auth.adminUsernameTip')" placement="top">
                    <span class="cursor-help">{{ t('settings.auth.adminUsername') }}</span>
                  </el-tooltip>
                </label>
                <input v-model="authConfig.adminUsername" type="text" :placeholder="t('settings.auth.adminUsernamePlaceholder')"
                  class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
                  :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
              </div>

              <div>
                <label class="block text-sm font-medium mb-2">
                  <el-tooltip :content="t('settings.auth.adminPasswordTip')" placement="top">
                    <span class="cursor-help">{{ t('settings.auth.adminPassword') }}</span>
                  </el-tooltip>
                </label>
                <input v-model="authConfig.adminPassword" type="password" :placeholder="t('settings.auth.adminPasswordPlaceholder')"
                  class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
                  :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
                <p class="text-xs mt-1.5" :class="isDark ? 'text-gray-500' : 'text-gray-400'">{{ t('settings.auth.adminPasswordDesc') }}</p>
              </div>

              <div>
                <label class="block text-sm font-medium mb-2">
                  <el-tooltip :content="t('settings.auth.sessionTimeoutTip')" placement="top">
                    <span class="cursor-help">{{ t('settings.auth.sessionTimeout') }}</span>
                  </el-tooltip>
                </label>
                <input v-model.number="authConfig.sessionTimeout" type="number" min="3600" max="604800"
                  class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 max-w-[200px] text-sm"
                  :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
                <p class="text-xs mt-1.5" :class="isDark ? 'text-gray-500' : 'text-gray-400'">{{ t('settings.auth.sessionTimeoutDesc') }}</p>
              </div>
            </div>
          </div>

          <button @click="saveAuthConfig"
            class="btn-gradient px-5 sm:px-6 py-2 sm:py-2.5 rounded-xl text-sm w-full sm:w-auto">
            {{ t('settings.auth.saveAuthSettings') }}
          </button>
        </div>

        <!-- 调度策略设置 -->
        <div v-else-if="activeTab === 'schedule'" class="max-w-2xl space-y-4 sm:space-y-6">
          <div class="p-4 rounded-xl" :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-50'">
            <h3 class="font-medium mb-3">
              <el-tooltip :content="t('settings.schedule.strategyTip')" placement="top">
                <span class="cursor-help">{{ t('settings.schedule.strategyTitle') }}</span>
              </el-tooltip>
            </h3>
            <p class="text-sm mb-4" :class="isDark ? 'text-gray-400' : 'text-gray-500'">
              {{ t('settings.schedule.strategyDesc') }}
            </p>
            <div class="space-y-3">
              <label class="flex items-start gap-3 cursor-pointer">
                <input type="radio" v-model="scheduleConfig.strategy" value="priority" class="mt-1 accent-indigo-500" />
                <div>
                  <span class="font-medium">{{ t('settings.schedule.priorityMode') }}</span>
                  <p class="text-sm mt-0.5" :class="isDark ? 'text-gray-400' : 'text-gray-500'">
                    {{ t('settings.schedule.priorityModeDesc') }}
                  </p>
                </div>
              </label>
              <label class="flex items-start gap-3 cursor-pointer">
                <input type="radio" v-model="scheduleConfig.strategy" value="weight" class="mt-1 accent-indigo-500" />
                <div>
                  <span class="font-medium">{{ t('settings.schedule.weightMode') }}</span>
                  <p class="text-sm mt-0.5" :class="isDark ? 'text-gray-400' : 'text-gray-500'">
                    {{ t('settings.schedule.weightModeDesc') }}
                  </p>
                </div>
              </label>
              <label class="flex items-start gap-3 cursor-pointer">
                <input type="radio" v-model="scheduleConfig.strategy" value="round-robin"
                  class="mt-1 accent-indigo-500" />
                <div>
                  <span class="font-medium">{{ t('settings.schedule.roundRobinMode') }}</span>
                  <p class="text-sm mt-0.5" :class="isDark ? 'text-gray-400' : 'text-gray-500'">
                    {{ t('settings.schedule.roundRobinModeDesc') }}
                  </p>
                </div>
              </label>
              <label class="flex items-start gap-3 cursor-pointer">
                <input type="radio" v-model="scheduleConfig.strategy" value="random" class="mt-1 accent-indigo-500" />
                <div>
                  <span class="font-medium">{{ t('settings.schedule.randomMode') }}</span>
                  <p class="text-sm mt-0.5" :class="isDark ? 'text-gray-400' : 'text-gray-500'">
                    {{ t('settings.schedule.randomModeDesc') }}
                  </p>
                </div>
              </label>
            </div>
          </div>

          <button @click="saveScheduleConfig"
            class="btn-gradient px-5 sm:px-6 py-2 sm:py-2.5 rounded-xl text-sm w-full sm:w-auto">
            {{ t('settings.schedule.saveScheduleSettings') }}
          </button>
        </div>

        <!-- 速率限制设置 -->
        <div v-else-if="activeTab === 'rateLimit'" class="max-w-2xl space-y-4 sm:space-y-6">
          <div class="p-4 rounded-xl" :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-50'">
            <div class="flex flex-wrap items-center justify-between gap-2 mb-4">
              <div>
                <h3 class="font-medium">
                  <el-tooltip :content="t('settings.rateLimit.rateLimitTip')" placement="top">
                    <span class="cursor-help">{{ t('settings.rateLimit.rateLimitTitle') }}</span>
                  </el-tooltip>
                </h3>
                <p class="text-sm mt-0.5" :class="isDark ? 'text-gray-400' : 'text-gray-500'">
                  {{ t('settings.rateLimit.rateLimitDesc') }}
                </p>
              </div>
              <label class="flex items-center gap-2 cursor-pointer">
                <input type="checkbox" v-model="rateLimitConfig.enabled" class="w-4 h-4 accent-indigo-500" />
                <span class="text-sm">{{ t('settings.rateLimit.enable') }}</span>
              </label>
            </div>

            <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <label class="block text-sm font-medium mb-2">
                  <el-tooltip :content="t('settings.rateLimit.perMinuteLimitTip')" placement="top">
                    <span class="cursor-help">{{ t('settings.rateLimit.perMinuteLimit') }}</span>
                  </el-tooltip>
                </label>
                <input v-model.number="rateLimitConfig.rateLimit" type="number" min="1" max="100"
                  class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
                  :class="isDark ? 'bg-[var(--bg-secondary)] border-[var(--border)]' : 'bg-white border-gray-200'" />
                <p class="text-xs mt-1.5" :class="isDark ? 'text-gray-500' : 'text-gray-400'">
                  {{ t('settings.rateLimit.perMinuteLimitDesc') }}
                </p>
              </div>
              <div>
                <label class="block text-sm font-medium mb-2">
                  <el-tooltip :content="t('settings.rateLimit.dailyLimitTip')" placement="top">
                    <span class="cursor-help">{{ t('settings.rateLimit.dailyLimit') }}</span>
                  </el-tooltip>
                </label>
                <input v-model.number="rateLimitConfig.dailyLimit" type="number" min="1" max="1000"
                  class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
                  :class="isDark ? 'bg-[var(--bg-secondary)] border-[var(--border)]' : 'bg-white border-gray-200'" />
                <p class="text-xs mt-1.5" :class="isDark ? 'text-gray-500' : 'text-gray-400'">
                  {{ t('settings.rateLimit.dailyLimitDesc') }}
                </p>
              </div>
              <div class="sm:col-span-2">
                <label class="block text-sm font-medium mb-2">
                  <el-tooltip :content="t('settings.rateLimit.maxFileSizeTip')" placement="top">
                    <span class="cursor-help">{{ t('settings.rateLimit.maxFileSize') }}</span>
                  </el-tooltip>
                </label>
                <input v-model.number="rateLimitConfig.maxFileSize" type="number" min="1" max="50"
                  class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
                  :class="isDark ? 'bg-[var(--bg-secondary)] border-[var(--border)]' : 'bg-white border-gray-200'" />
                <p class="text-xs mt-1.5" :class="isDark ? 'text-gray-500' : 'text-gray-400'">
                  {{ t('settings.rateLimit.maxFileSizeDesc') }}
                </p>
              </div>
            </div>
          </div>

          <button @click="saveRateLimitConfig"
            class="btn-gradient px-5 sm:px-6 py-2 sm:py-2.5 rounded-xl text-sm w-full sm:w-auto">
            {{ t('settings.rateLimit.saveRateLimitSettings') }}
          </button>
        </div>

        <!-- CDN加速设置 -->
        <div v-else-if="activeTab === 'cdn'" class="max-w-2xl space-y-4 sm:space-y-6">
          <div class="p-4 rounded-xl" :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-50'">
            <div class="flex flex-wrap items-center justify-between gap-2 mb-4">
              <div>
                <h3 class="font-medium">{{ t('settings.cdn.cdnProxyTitle') }}</h3>
                <p class="text-sm mt-0.5" :class="isDark ? 'text-gray-400' : 'text-gray-500'">
                  {{ t('settings.cdn.cdnProxyDesc') }}
                </p>
              </div>
              <label class="flex items-center gap-2 cursor-pointer">
                <input type="checkbox" v-model="cdnConfig.enabled" class="w-4 h-4 accent-indigo-500" />
                <span class="text-sm">{{ t('settings.cdn.enable') }}</span>
              </label>
            </div>

            <div v-if="cdnConfig.enabled" class="space-y-4">
              <div>
                <input v-model="cdnConfig.proxyUrl" type="text" :placeholder="t('settings.cdn.proxyAddressPlaceholder')"
                  class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
                  :class="isDark ? 'bg-[var(--bg-secondary)] border-[var(--border)]' : 'bg-white border-gray-200'" />
                <p class="text-xs mt-1.5" :class="isDark ? 'text-gray-500' : 'text-gray-400'">
                  {{ t('settings.cdn.proxyAddressDesc') }}
                </p>
              </div>
              <div>
                <input v-model="cdnConfig.cdnUrl" type="text" :placeholder="t('settings.cdn.cdnUrlPlaceholder')"
                  class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
                  :class="isDark ? 'bg-[var(--bg-secondary)] border-[var(--border)]' : 'bg-white border-gray-200'" />
                <p class="text-xs mt-1.5" :class="isDark ? 'text-gray-500' : 'text-gray-400'">
                  {{ t('settings.cdn.cdnUrlDesc') }}
                </p>
              </div>
            </div>
          </div>

          <div class="p-4 rounded-xl" :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-50'">
            <h3 class="font-medium mb-2">{{ t('settings.cdn.featureTitle') }}</h3>
            <div class="text-sm space-y-3" :class="isDark ? 'text-gray-400' : 'text-gray-500'">
              <div>
                <span class="font-medium" :class="isDark ? 'text-gray-300' : 'text-gray-700'">{{ t('settings.cdn.downloadAcceleration') }}：</span>
                <ul class="ml-4 mt-1 space-y-1 list-disc">
                  <li>{{ t('settings.cdn.downloadAccelerationDesc1') }}</li>
                  <li v-html="t('settings.cdn.downloadAccelerationDesc2')"></li>
                </ul>
              </div>
              <div>
                <span class="font-medium" :class="isDark ? 'text-gray-300' : 'text-gray-700'">{{ t('settings.cdn.uploadProxy') }}：</span>
                <ul class="ml-4 mt-1 space-y-1 list-disc">
                  <li>{{ t('settings.cdn.uploadProxyDesc1') }}</li>
                  <li v-html="t('settings.cdn.uploadProxyDesc2')"></li>
                </ul>
              </div>
            </div>
          </div>

          <div class="p-4 rounded-xl" :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-50'">
            <h3 class="font-medium mb-2">{{ t('settings.cdn.deployTitle') }}</h3>
            <ol class="text-sm space-y-1 list-decimal list-inside" :class="isDark ? 'text-gray-400' : 'text-gray-500'">
              <li>{{ t('settings.cdn.deployStep1') }}</li>
              <li>{{ t('settings.cdn.deployStep2') }}</li>
              <li>{{ t('settings.cdn.deployStep3') }}</li>
            </ol>
          </div>

          <button @click="saveCdnConfig"
            class="btn-gradient px-5 sm:px-6 py-2 sm:py-2.5 rounded-xl text-sm w-full sm:w-auto">
            {{ t('settings.cdn.saveCdnSettings') }}
          </button>
        </div>

        <!-- 备份设置 -->
        <div v-else-if="activeTab === 'backup'" class="max-w-4xl space-y-4 sm:space-y-6">
          <div class="p-4 rounded-xl shadow-lg border border-solid" :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-50'">
            <div class="flex flex-wrap items-center justify-between gap-2 mb-4">
              <div>
                <h3 class="font-medium">{{ t('settings.backup.autoBackupTitle') }}</h3>
                <p class="text-sm mt-0.5" :class="isDark ? 'text-gray-400' : 'text-gray-500'">
                  {{ t('settings.backup.autoBackupDesc') }}
                </p>
              </div>
              <label class="flex items-center gap-2 cursor-pointer">
                <input type="checkbox" v-model="backupConfig.enabled" class="w-4 h-4 accent-indigo-500" />
                <span class="text-sm">{{ t('settings.backup.enable') }}</span>
              </label>
            </div>

            <div v-if="backupConfig.enabled" class="space-y-4">
              <div>
                <label class="block text-sm font-medium mb-2">
                  {{ t('settings.backup.interval') }}
                </label>
                <input v-model.number="backupConfig.interval" type="number" min="1" max="168"
                  class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm max-w-[200px]"
                  :class="isDark ? 'bg-[var(--bg-secondary)] border-[var(--border)]' : 'bg-white border-gray-200'" />
                <p class="text-xs mt-1.5" :class="isDark ? 'text-gray-500' : 'text-gray-400'">
                  {{ t('settings.backup.intervalDesc') }}
                </p>
              </div>
            </div>
          </div>

          <button @click="saveBackupConfig"
            class="btn-gradient px-5 sm:px-6 py-2 sm:py-2.5 rounded-xl text-sm w-full sm:w-auto">
            {{ t('settings.backup.saveBackupSettings') }}
          </button>

          <div class="flex justify-between items-center mb-4">
            <h3 class="font-medium text-lg">{{ t('backup.title') }}</h3>
            <el-button type="primary" @click="handleCreateBackup" :loading="creating">
              {{ t('backup.create') }}
            </el-button>
          </div>

          <div class="p-4 rounded-xl shadow-lg  border border-solid" :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-50'">
            <el-table v-loading="backupLoading" :data="backups" style="width: 100%">
              <el-table-column prop="name" :label="t('backup.fileName')" width="300">
                <template #default="{ row }">
                  {{ row.name }}
                </template>
              </el-table-column>
              <el-table-column prop="size" :label="t('backup.size')" width="120">
                <template #default="{ row }">
                  {{ formatSize(row.size) }}
                </template>
              </el-table-column>
              <el-table-column prop="created_at" :label="t('backup.createdAt')" width="200">
                <template #default="{ row }">
                  {{ row.created_at }}
                </template>
              </el-table-column>
              <el-table-column :label="t('backup.action')" width="200" fixed="right">
                <template #default="{ row }">
                  <el-button size="small" type="primary" @click="handleRestoreBackup(row.path)" :loading="restoring === row.path">
                    {{ t('backup.restore') }}
                  </el-button>
                  <el-button size="small" type="danger" @click="handleDeleteBackup(row.path)" :loading="deleting === row.path" style="margin-left: 8px">
                    {{ t('backup.delete') }}
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
            <div v-if="backups.length === 0" class="text-center py-8 text-gray-500">
              {{ t('backup.noBackups') }}
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.tabs-nav {
  flex-shrink: 0;
  overflow-x: auto;
  overflow-y: hidden;
}

.settings-content {
  overflow-y: auto;
  overflow-x: hidden;
}
</style>
