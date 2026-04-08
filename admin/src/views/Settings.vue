<script setup>
import { ref, onMounted, reactive } from 'vue'
import request from '@/api/request'
import { ElMessage } from 'element-plus'
import { Upload, Image, Lock, RefreshCw, Settings, Shield, Globe } from 'lucide-vue-next'

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
  proxyUrl: ''
})

onMounted(() => {
  isDark.value = !document.documentElement.classList.contains('light')
  loadConfigs()
})

async function loadConfigs() {
  loading.value = true
  try {
    const [uploadRes, appRes, jwtRes, siteRes, authRes, scheduleRes, rateLimitRes, cdnRes] = await Promise.all([
      request.get('/config/upload').catch(() => ({ data: null })),
      request.get('/config/app').catch(() => ({ data: null })),
      request.get('/config/jwt').catch(() => ({ data: null })),
      request.get('/config/site').catch(() => ({ data: null })),
      request.get('/config/auth').catch(() => ({ data: null })),
      request.get('/config/schedule').catch(() => ({ data: null })),
      request.get('/config/rate-limit').catch(() => ({ data: null })),
      request.get('/config/cdn').catch(() => ({ data: null }))
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
    }
  } catch {
    ElMessage.error('加载配置失败')
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
    ElMessage.success('应用设置已保存')
  } catch {
    ElMessage.error('保存失败')
  }
}

async function saveJwtConfig() {
  try {
    await request.put('/config/jwt', {
      expire: jwtConfig.expire
    })
    ElMessage.success('JWT设置已保存')
  } catch {
    ElMessage.error('保存失败')
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
    ElMessage.success('上传配置已保存')
  } catch {
    ElMessage.error('保存失败')
  }
}

async function saveSiteConfig() {
  try {
    await request.put('/config/site', {
      name: siteConfig.name,
      logo: siteConfig.logo
    })
    ElMessage.success('站点配置已保存')
  } catch {
    ElMessage.error('保存失败')
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
    ElMessage.success('认证配置已保存')
  } catch {
    ElMessage.error('保存失败')
  }
}

async function saveScheduleConfig() {
  try {
    await request.put('/config/schedule', {
      strategy: scheduleConfig.strategy
    })
    ElMessage.success('调度策略已保存')
  } catch {
    ElMessage.error('保存失败')
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
    ElMessage.success('速率限制配置已保存')
  } catch {
    ElMessage.error('保存失败')
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
      proxyUrl: cdnConfig.proxyUrl
    })
    ElMessage.success('CDN配置已保存')
  } catch {
    ElMessage.error('保存失败')
  }
}

const tabs = [
  { name: 'upload', label: '上传设置', icon: Upload },
  { name: 'app', label: '应用设置', icon: Settings },
  { name: 'jwt', label: 'JWT设置', icon: Shield },
  { name: 'site', label: '站点设置', icon: Image },
  { name: 'auth', label: '认证设置', icon: Lock },
  { name: 'schedule', label: '调度策略', icon: Settings },
  { name: 'rateLimit', label: '速率限制', icon: Shield },
  { name: 'cdn', label: 'CDN加速', icon: Globe }
]

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
  <div class="space-y-4 sm:space-y-6">
    <!-- 设置卡片 -->
    <div class="card overflow-hidden">
      <!-- 标签页导航 -->
      <div class="flex border-b overflow-x-auto scrollbar-thin" :style="{ borderColor: 'var(--border)' }">
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
      <div class="p-3 sm:p-6">
        <div v-if="loading" class="space-y-4">
          <div v-for="i in 4" :key="i" class="h-12 rounded-xl loading-shimmer"></div>
        </div>

        <!-- 上传设置 -->
        <div v-else-if="activeTab === 'upload'" class="max-w-2xl space-y-4 sm:space-y-6">
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 sm:gap-6">
            <div>
              <el-tooltip content="单个文件最大允许上传的大小（MB）" placement="top">
                <label class="block text-sm font-medium mb-2 cursor-help">最大文件大小 (MB)</label>
              </el-tooltip>
              <input v-model.number="uploadConfig.maxSize" type="number" min="1" max="100"
                class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
                :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
            </div>
            <div>
              <el-tooltip content="分片上传时每个分片的大小（MB）" placement="top">
                <label class="block text-sm font-medium mb-2 cursor-help">分片大小 (MB)</label>
              </el-tooltip>
              <input v-model.number="uploadConfig.chunkSize" type="number" min="1" max="50"
                class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
                :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
            </div>
          </div>

          <div>
            <el-tooltip content="文件上传时的默认存储渠道，留空则自动选择" placement="top">
              <label class="block text-sm font-medium mb-2 cursor-help">默认渠道</label>
            </el-tooltip>
            <input v-model="uploadConfig.defaultChannel" type="text" placeholder="留空则自动选择"
              class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
              :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
          </div>

          <div>
            <el-tooltip content="允许上传的文件 MIME 类型，多个用逗号分隔，留空允许所有类型" placement="top">
              <label class="block text-sm font-medium mb-2 cursor-help">允许的文件类型</label>
            </el-tooltip>
            <input v-model="uploadConfig.allowedTypes" type="text" placeholder="jpg,png,gif,mp4 (留空允许所有)"
              class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
              :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
            <p class="text-xs mt-1.5" :class="isDark ? 'text-gray-500' : 'text-gray-400'">多个类型用逗号分隔</p>
          </div>

          <div class="flex items-center justify-between p-3 sm:p-4 rounded-xl"
            :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-50'">
            <div>
              <el-tooltip content="上传失败时自动重试上传" placement="top">
                <p class="font-medium text-sm cursor-help">自动重试</p>
              </el-tooltip>
              <p class="text-xs" :class="isDark ? 'text-gray-500' : 'text-gray-400'">上传失败时自动重试</p>
            </div>
            <input type="checkbox" v-model="uploadConfig.autoRetry"
              class="w-4 h-4 sm:w-5 sm:h-5 rounded cursor-pointer accent-indigo-500" />
          </div>

          <div v-if="uploadConfig.autoRetry">
            <el-tooltip content="自动重试的最大次数" placement="top">
              <label class="block text-sm font-medium mb-2 cursor-help">重试次数</label>
            </el-tooltip>
            <input v-model.number="uploadConfig.retryCount" type="number" min="1" max="10"
              class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 max-w-[200px] text-sm"
              :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
          </div>

          <!-- 图片压缩配置 -->
          <div class="border-t pt-4 mt-4" :style="{ borderColor: 'var(--border)' }">
            <h3 class="font-medium mb-3">图片压缩配置</h3>

            <div class="flex items-center justify-between p-3 sm:p-4 rounded-xl mb-4"
              :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-50'">
              <el-tooltip content="上传时自动压缩图片以节省存储空间" placement="top">
                <div>
                  <p class="font-medium text-sm cursor-help">启用压缩</p>
                  <p class="text-xs" :class="isDark ? 'text-gray-500' : 'text-gray-400'">上传时自动压缩图片</p>
                </div>
              </el-tooltip>
              <input type="checkbox" v-model="uploadConfig.compressionEnabled"
                class="w-4 h-4 sm:w-5 sm:h-5 rounded cursor-pointer accent-indigo-500" />
            </div>

            <div v-if="uploadConfig.compressionEnabled" class="space-y-4">
              <div class="grid grid-cols-2 gap-4">
                <div>
                  <el-tooltip content="压缩质量 1-100，越高质量越好但文件越大" placement="top">
                    <label class="block text-sm font-medium mb-2 cursor-help">压缩质量</label>
                  </el-tooltip>
                  <input v-model.number="uploadConfig.compressionQuality" type="number" min="1" max="100"
                    class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
                    :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
                  <p class="text-xs mt-1" :class="isDark ? 'text-gray-500' : 'text-gray-400'">1-100，越高质量越好</p>
                </div>
                <div>
                  <el-tooltip content="压缩后的图片格式" placement="top">
                    <label class="block text-sm font-medium mb-2 cursor-help">输出格式</label>
                  </el-tooltip>
                  <select v-model="uploadConfig.compressionFormat"
                    class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
                    :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'">
                    <option value="webp">WebP (推荐)</option>
                    <option value="jpeg">JPEG</option>
                    <option value="png">PNG</option>
                    <option value="original">保持原格式</option>
                  </select>
                </div>
              </div>

              <div class="grid grid-cols-2 gap-4">
                <div>
                  <el-tooltip content="图片最大宽度(像素)，超过则按比例缩小" placement="top">
                    <label class="block text-sm font-medium mb-2 cursor-help">最大宽度</label>
                  </el-tooltip>
                  <input v-model.number="uploadConfig.compressionMaxWidth" type="number" min="100" max="10000"
                    class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
                    :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
                </div>
                <div>
                  <el-tooltip content="图片最大高度(像素)，超过则按比例缩小" placement="top">
                    <label class="block text-sm font-medium mb-2 cursor-help">最大高度</label>
                  </el-tooltip>
                  <input v-model.number="uploadConfig.compressionMaxHeight" type="number" min="100" max="10000"
                    class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
                    :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
                </div>
              </div>
            </div>
          </div>

          <button @click="saveUploadConfig"
            class="btn-gradient px-5 sm:px-6 py-2 sm:py-2.5 rounded-xl text-sm w-full sm:w-auto">
            保存上传配置
          </button>
        </div>

        <!-- 应用设置 -->
        <div v-else-if="activeTab === 'app'" class="max-w-2xl space-y-4 sm:space-y-6">
          <div>
            <el-tooltip content="服务器监听地址" placement="top">
              <label class="block text-sm font-medium mb-2 cursor-help">监听地址</label>
            </el-tooltip>
            <input v-model="appConfig.host" type="text" placeholder="0.0.0.0"
              class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
              :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
            <p class="text-xs mt-1.5" :class="isDark ? 'text-gray-500' : 'text-gray-400'">0.0.0.0 监听所有网络接口</p>
          </div>

          <div>
            <el-tooltip content="服务器监听端口" placement="top">
              <label class="block text-sm font-medium mb-2 cursor-help">监听端口</label>
            </el-tooltip>
            <input v-model.number="appConfig.port" type="number" min="1" max="65535" placeholder="8080"
              class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 max-w-[200px] text-sm"
              :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
          </div>

          <div class="p-4 rounded-xl" :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-50'">
            <el-tooltip content="运行模式影响日志输出和错误处理" placement="top">
              <h3 class="font-medium mb-2 cursor-help">运行模式</h3>
            </el-tooltip>
            <div class="flex gap-4">
              <label class="flex items-center gap-2 cursor-pointer">
                <input type="radio" v-model="appConfig.mode" value="debug" class="accent-indigo-500" />
                <span class="text-sm">Debug</span>
              </label>
              <label class="flex items-center gap-2 cursor-pointer">
                <input type="radio" v-model="appConfig.mode" value="release" class="accent-indigo-500" />
                <span class="text-sm">Release</span>
              </label>
            </div>
            <p class="text-xs mt-2" :class="isDark ? 'text-gray-500' : 'text-gray-400'">
              Debug模式输出详细日志，Release模式仅输出错误
            </p>
          </div>

          <button @click="saveAppConfig"
            class="btn-gradient px-5 sm:px-6 py-2 sm:py-2.5 rounded-xl text-sm w-full sm:w-auto">
            保存应用设置
          </button>
        </div>

        <!-- JWT设置 -->
        <div v-else-if="activeTab === 'jwt'" class="max-w-2xl space-y-4 sm:space-y-6">
          <div>
            <el-tooltip content="JWT签名的密钥，生产环境务必修改" placement="top">
              <label class="block text-sm font-medium mb-2 cursor-help">JWT密钥</label>
            </el-tooltip>
            <input v-model="jwtConfig.secret" type="password" placeholder="输入新密钥以修改"
              class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
              :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
            <p class="text-xs mt-1.5" :class="isDark ? 'text-gray-500' : 'text-gray-400'">留空则保持当前密钥，修改后所有现有token将失效</p>
          </div>

          <div>
            <el-tooltip content="JWT token过期时间（秒）" placement="top">
              <label class="block text-sm font-medium mb-2 cursor-help">Token过期时间 (秒)</label>
            </el-tooltip>
            <input v-model.number="jwtConfig.expire" type="number" min="3600" max="604800"
              class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 max-w-[200px] text-sm"
              :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
            <p class="text-xs mt-1.5" :class="isDark ? 'text-gray-500' : 'text-gray-400'">默认24小时 (86400秒)，最小1小时，最大7天</p>
          </div>

          <button @click="saveJwtConfig"
            class="btn-gradient px-5 sm:px-6 py-2 sm:py-2.5 rounded-xl text-sm w-full sm:w-auto">
            保存JWT设置
          </button>
        </div>

        <!-- 站点设置 -->
        <div v-else-if="activeTab === 'site'" class="max-w-2xl space-y-4 sm:space-y-6">
          <div>
            <el-tooltip content="网站标题，将显示在浏览器标签页" placement="top">
              <label class="block text-sm font-medium mb-2 cursor-help">站点名称</label>
            </el-tooltip>
            <input v-model="siteConfig.name" type="text" placeholder="ImgBed"
              class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
              :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
          </div>

          <div>
            <el-tooltip content="网站 Logo 图片 URL，留空使用默认图标" placement="top">
              <label class="block text-sm font-medium mb-2 cursor-help">Logo URL</label>
            </el-tooltip>
            <input v-model="siteConfig.logo" type="text" placeholder="Logo 图片地址"
              class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
              :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
          </div>

          <button @click="saveSiteConfig"
            class="btn-gradient px-5 sm:px-6 py-2 sm:py-2.5 rounded-xl text-sm w-full sm:w-auto">
            保存站点配置
          </button>
        </div>

        <!-- 认证设置 -->
        <div v-else-if="activeTab === 'auth'" class="max-w-2xl space-y-4 sm:space-y-6">
          <div>
            <el-tooltip content="设置访问密码后，用户需要密码才能上传文件" placement="top">
              <label class="block text-sm font-medium mb-2 cursor-help">访问密码</label>
            </el-tooltip>
            <input v-model="authConfig.userPassword" type="password" placeholder="用户访问密码 (留空则公开访问)"
              class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
              :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
            <p class="text-xs mt-1.5" :class="isDark ? 'text-gray-500' : 'text-gray-400'">设置后用户需要密码才能上传</p>
          </div>

          <div class="border-t pt-4 sm:pt-6" :style="{ borderColor: 'var(--border)' }">
            <h3 class="font-medium mb-3 sm:mb-4">管理员设置</h3>

            <div class="space-y-4">
              <div>
                <el-tooltip content="管理员登录用户名" placement="top">
                  <label class="block text-sm font-medium mb-2 cursor-help">管理员用户名</label>
                </el-tooltip>
                <input v-model="authConfig.adminUsername" type="text" placeholder="admin"
                  class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
                  :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
              </div>

              <div>
                <el-tooltip content="修改管理员密码，留空则保持不变" placement="top">
                  <label class="block text-sm font-medium mb-2 cursor-help">管理员密码</label>
                </el-tooltip>
                <input v-model="authConfig.adminPassword" type="password" placeholder="输入新密码以修改"
                  class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
                  :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
                <p class="text-xs mt-1.5" :class="isDark ? 'text-gray-500' : 'text-gray-400'">留空则不修改密码</p>
              </div>

              <div>
                <el-tooltip content="管理员登录会话有效期，超过后需要重新登录" placement="top">
                  <label class="block text-sm font-medium mb-2 cursor-help">会话超时 (秒)</label>
                </el-tooltip>
                <input v-model.number="authConfig.sessionTimeout" type="number" min="3600" max="604800"
                  class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 max-w-[200px] text-sm"
                  :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
                <p class="text-xs mt-1.5" :class="isDark ? 'text-gray-500' : 'text-gray-400'">默认 24 小时 (86400秒)</p>
              </div>
            </div>
          </div>

          <button @click="saveAuthConfig"
            class="btn-gradient px-5 sm:px-6 py-2 sm:py-2.5 rounded-xl text-sm w-full sm:w-auto">
            保存认证配置
          </button>
        </div>

        <!-- 调度策略设置 -->
        <div v-else-if="activeTab === 'schedule'" class="max-w-2xl space-y-4 sm:space-y-6">
          <div class="p-4 rounded-xl" :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-50'">
            <el-tooltip content="选择上传文件时如何选择存储渠道" placement="top">
              <h3 class="font-medium mb-3 cursor-help">渠道调度策略</h3>
            </el-tooltip>
            <p class="text-sm mb-4" :class="isDark ? 'text-gray-400' : 'text-gray-500'">
              选择上传文件时如何选择存储渠道
            </p>
            <div class="space-y-3">
              <label class="flex items-start gap-3 cursor-pointer">
                <input type="radio" v-model="scheduleConfig.strategy" value="priority" class="mt-1 accent-indigo-500" />
                <div>
                  <span class="font-medium">优先级模式</span>
                  <p class="text-sm mt-0.5" :class="isDark ? 'text-gray-400' : 'text-gray-500'">
                    按渠道优先级顺序选择，优先使用高优先级渠道
                  </p>
                </div>
              </label>
              <label class="flex items-start gap-3 cursor-pointer">
                <input type="radio" v-model="scheduleConfig.strategy" value="weight" class="mt-1 accent-indigo-500" />
                <div>
                  <span class="font-medium">权重模式</span>
                  <p class="text-sm mt-0.5" :class="isDark ? 'text-gray-400' : 'text-gray-500'">
                    按渠道权重随机分配，权重越高被选中概率越大
                  </p>
                </div>
              </label>
              <label class="flex items-start gap-3 cursor-pointer">
                <input type="radio" v-model="scheduleConfig.strategy" value="round-robin"
                  class="mt-1 accent-indigo-500" />
                <div>
                  <span class="font-medium">轮询模式</span>
                  <p class="text-sm mt-0.5" :class="isDark ? 'text-gray-400' : 'text-gray-500'">
                    按顺序依次使用各个渠道，实现负载均衡
                  </p>
                </div>
              </label>
              <label class="flex items-start gap-3 cursor-pointer">
                <input type="radio" v-model="scheduleConfig.strategy" value="random" class="mt-1 accent-indigo-500" />
                <div>
                  <span class="font-medium">随机模式</span>
                  <p class="text-sm mt-0.5" :class="isDark ? 'text-gray-400' : 'text-gray-500'">
                    随机选择一个可用渠道
                  </p>
                </div>
              </label>
            </div>
          </div>

          <button @click="saveScheduleConfig"
            class="btn-gradient px-5 sm:px-6 py-2 sm:py-2.5 rounded-xl text-sm w-full sm:w-auto">
            保存调度策略
          </button>
        </div>

        <!-- 速率限制设置 -->
        <div v-else-if="activeTab === 'rateLimit'" class="max-w-2xl space-y-4 sm:space-y-6">
          <div class="p-4 rounded-xl" :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-50'">
            <div class="flex flex-wrap items-center justify-between gap-2 mb-4">
              <div>
                <el-tooltip content="限制所有用户的上传频率，防止滥用" placement="top">
                  <h3 class="font-medium cursor-help">上传速率限制</h3>
                </el-tooltip>
                <p class="text-sm mt-0.5" :class="isDark ? 'text-gray-400' : 'text-gray-500'">
                  限制所有用户的上传频率
                </p>
              </div>
              <label class="flex items-center gap-2 cursor-pointer">
                <input type="checkbox" v-model="rateLimitConfig.enabled" class="w-4 h-4 accent-indigo-500" />
                <span class="text-sm">启用</span>
              </label>
            </div>

            <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <el-tooltip content="每分钟最多上传的文件数量" placement="top">
                  <label class="block text-sm font-medium mb-2 cursor-help">每分钟限制</label>
                </el-tooltip>
                <input v-model.number="rateLimitConfig.rateLimit" type="number" min="1" max="100"
                  class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
                  :class="isDark ? 'bg-[var(--bg-secondary)] border-[var(--border)]' : 'bg-white border-gray-200'" />
                <p class="text-xs mt-1.5" :class="isDark ? 'text-gray-500' : 'text-gray-400'">
                  每分钟最大上传文件数量
                </p>
              </div>
              <div>
                <el-tooltip content="每天最多上传的文件数量" placement="top">
                  <label class="block text-sm font-medium mb-2 cursor-help">每日限制</label>
                </el-tooltip>
                <input v-model.number="rateLimitConfig.dailyLimit" type="number" min="1" max="1000"
                  class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
                  :class="isDark ? 'bg-[var(--bg-secondary)] border-[var(--border)]' : 'bg-white border-gray-200'" />
                <p class="text-xs mt-1.5" :class="isDark ? 'text-gray-500' : 'text-gray-400'">
                  每天最大上传文件数量
                </p>
              </div>
              <div class="sm:col-span-2">
                <el-tooltip content="单次上传文件的大小限制" placement="top">
                  <label class="block text-sm font-medium mb-2 cursor-help">最大文件大小 (MB)</label>
                </el-tooltip>
                <input v-model.number="rateLimitConfig.maxFileSize" type="number" min="1" max="50"
                  class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
                  :class="isDark ? 'bg-[var(--bg-secondary)] border-[var(--border)]' : 'bg-white border-gray-200'" />
                <p class="text-xs mt-1.5" :class="isDark ? 'text-gray-500' : 'text-gray-400'">
                  单文件最大大小限制
                </p>
              </div>
            </div>
          </div>

          <button @click="saveRateLimitConfig"
            class="btn-gradient px-5 sm:px-6 py-2 sm:py-2.5 rounded-xl text-sm w-full sm:w-auto">
            保存速率限制配置
          </button>
        </div>

        <!-- CDN加速设置 -->
        <div v-else-if="activeTab === 'cdn'" class="max-w-2xl space-y-4 sm:space-y-6">
          <div class="p-4 rounded-xl" :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-50'">
            <div class="flex flex-wrap items-center justify-between gap-2 mb-4">
              <div>
                <h3 class="font-medium">CDN 代理加速</h3>
                <p class="text-sm mt-0.5" :class="isDark ? 'text-gray-400' : 'text-gray-500'">
                  启用后同时开启下载加速和上传代理，解决被墙问题
                </p>
              </div>
              <label class="flex items-center gap-2 cursor-pointer">
                <input type="checkbox" v-model="cdnConfig.enabled" class="w-4 h-4 accent-indigo-500" />
                <span class="text-sm">启用</span>
              </label>
            </div>

            <div v-if="cdnConfig.enabled" class="space-y-4">
              <div>
                <label class="block text-sm font-medium mb-2">代理地址</label>
                <input v-model="cdnConfig.proxyUrl" type="text" placeholder="https://img-proxy.xxx.workers.dev"
                  class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
                  :class="isDark ? 'bg-[var(--bg-secondary)] border-[var(--border)]' : 'bg-white border-gray-200'" />
                <p class="text-xs mt-1.5" :class="isDark ? 'text-gray-500' : 'text-gray-400'">
                  填写 Cloudflare Worker 地址，下载加速和上传代理共用同一地址
                </p>
              </div>
            </div>
          </div>

          <div class="p-4 rounded-xl" :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-50'">
            <h3 class="font-medium mb-2">功能说明</h3>
            <div class="text-sm space-y-3" :class="isDark ? 'text-gray-400' : 'text-gray-500'">
              <div>
                <span class="font-medium" :class="isDark ? 'text-gray-300' : 'text-gray-700'">下载加速：</span>
                <ul class="ml-4 mt-1 space-y-1 list-disc">
                  <li>将图片直链转换为 CDN 代理地址，提升访问速度</li>
                  <li>格式：<code class="px-1 py-0.5 rounded text-xs" :class="isDark ? 'bg-gray-700' : 'bg-gray-200'">{proxyUrl}/{base58(原始URL)}/{文件名}</code></li>
                </ul>
              </div>
              <div>
                <span class="font-medium" :class="isDark ? 'text-gray-300' : 'text-gray-700'">上传代理：</span>
                <ul class="ml-4 mt-1 space-y-1 list-disc">
                  <li>代理上传请求到 Telegram/Discord/HuggingFace/S3/R2 等服务</li>
                  <li>格式：<code class="px-1 py-0.5 rounded text-xs" :class="isDark ? 'bg-gray-700' : 'bg-gray-200'">{proxyUrl}/proxy/{base58(目标host)}/{路径}</code></li>
                </ul>
              </div>
            </div>
          </div>

          <div class="p-4 rounded-xl" :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-50'">
            <h3 class="font-medium mb-2">部署说明</h3>
            <ol class="text-sm space-y-1 list-decimal list-inside" :class="isDark ? 'text-gray-400' : 'text-gray-500'">
              <li>将 <code class="px-1 py-0.5 rounded text-xs" :class="isDark ? 'bg-gray-700' : 'bg-gray-200'">server/proxy/worker.js</code> 部署到 Cloudflare Workers</li>
              <li>Worker 同时支持下载代理（根路径）和上传代理（/proxy/ 路径）</li>
              <li>在此填写 Worker 地址并启用即可</li>
            </ol>
          </div>

          <button @click="saveCdnConfig"
            class="btn-gradient px-5 sm:px-6 py-2 sm:py-2.5 rounded-xl text-sm w-full sm:w-auto">
            保存 CDN 配置
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
