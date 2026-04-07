<script setup>
import { ref, onMounted } from 'vue'
import { channelApi } from '@/api/channel'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Network, Plus, Edit2, Trash2, RefreshCw, Check, X,
  AlertTriangle, Upload, Clock, Folder, Send, Cloud, HardDrive
} from 'lucide-vue-next'

const isDark = ref(true)
const channels = ref([])
const loading = ref(false)
const showDialog = ref(false)
const dialogType = ref('create')
const editingChannel = ref(null)
const testingId = ref(null)

const channelTypes = [
  { value: 'local', label: '本地存储' },
  { value: 'telegram', label: 'Telegram' },
  { value: 'cfr2', label: 'Cloudflare R2' },
  { value: 's3', label: 'S3 兼容存储' },
  { value: 'discord', label: 'Discord' },
  { value: 'huggingface', label: 'HuggingFace' }
]

const form = ref({
  name: '',
  type: 'local',
  config: {},
  quota: { enabled: false, limitGB: 0, threshold: 90 },
  rateLimit: { dailyUploadLimit: 0, hourlyUploadLimit: 0, minIntervalMs: 0 }
})

onMounted(() => {
  isDark.value = !document.documentElement.classList.contains('light')
  loadChannels()
})

async function loadChannels() {
  loading.value = true
  try {
    const res = await channelApi.list()
    if (res.code === 0) {
      channels.value = res.data || []
    }
  } catch {
    ElMessage.error('加载渠道失败')
  } finally {
    loading.value = false
  }
}

function openCreateDialog() {
  dialogType.value = 'create'
  editingChannel.value = null
  form.value = {
    name: '',
    type: 'local',
    config: {},
    quota: { enabled: false, limitGB: 0, threshold: 90 },
    rateLimit: { dailyUploadLimit: 0, hourlyUploadLimit: 0, minIntervalMs: 0 }
  }
  showDialog.value = true
}

function openEditDialog(channel) {
  dialogType.value = 'edit'
  editingChannel.value = channel.id
  form.value = {
    name: channel.name,
    type: channel.type,
    config: channel.config || {},
    quota: {
      enabled: channel.quotaEnabled || false,
      limitGB: Math.floor((channel.quotaLimit || 0) / (1024 * 1024 * 1024)),
      threshold: channel.quotaThreshold || 90
    },
    rateLimit: {
      dailyUploadLimit: channel.dailyUploadLimit || 0,
      hourlyUploadLimit: channel.hourlyUploadLimit || 0,
      minIntervalMs: channel.minIntervalMs || 0
    }
  }
  showDialog.value = true
}

async function saveChannel() {
  if (!form.value.name) {
    ElMessage.warning('请输入渠道名称')
    return
  }

  try {
    if (dialogType.value === 'create') {
      await channelApi.create(form.value)
      ElMessage.success('创建成功')
    } else {
      await channelApi.update(editingChannel.value, form.value)
      ElMessage.success('更新成功')
    }
    showDialog.value = false
    loadChannels()
  } catch (e) {
    ElMessage.error(e.message || '操作失败')
  }
}

async function deleteChannel(channel) {
  try {
    await ElMessageBox.confirm(`确定要删除渠道「${channel.name}」吗？`, '删除确认', { type: 'warning' })
    await channelApi.delete(channel.id)
    ElMessage.success('删除成功')
    loadChannels()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error('删除失败')
  }
}

async function toggleChannel(channel) {
  try {
    if (channel.enabled) {
      await channelApi.disable(channel.id)
    } else {
      await channelApi.enable(channel.id)
    }
    ElMessage.success(channel.enabled ? '已禁用' : '已启用')
    loadChannels()
  } catch {
    ElMessage.error('操作失败')
  }
}

async function testChannel(channel) {
  testingId.value = channel.id
  try {
    const res = await channelApi.test(channel.id)
    if (res.data?.success) {
      ElMessage.success('连接成功')
    } else {
      const errorMsg = res.data?.error || res.data?.message || '连接失败'
      const errorDetail = res.data?.detail || ''
      const fullError = errorDetail ? `${errorMsg}\n${errorDetail}` : errorMsg
      ElMessageBox.alert(fullError, '测试失败', {
        confirmButtonText: '确定',
        type: 'error',
        customClass: isDark.value ? 'dark-message-box' : ''
      })
    }
  } catch (err) {
    const errorMsg = err.response?.data?.message || err.message || '测试失败'
    ElMessageBox.alert(errorMsg, '测试失败', {
      confirmButtonText: '确定',
      type: 'error',
      customClass: isDark.value ? 'dark-message-box' : ''
    })
  } finally {
    testingId.value = null
  }
}

function formatSize(bytes) {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

function getTypeLabel(type) {
  return channelTypes.find(t => t.value === type)?.label || type
}

function getTypeIcon(type) {
  const icons = {
    local: HardDrive,
    telegram: Send,
    cfr2: Cloud,
    s3: Cloud,
    discord: Send,
    huggingface: Folder
  }
  return icons[type] || Network
}

function getStatusColor(status) {
  if (status === 'healthy') return 'text-green-500'
  if (status === 'warning') return 'text-yellow-500'
  if (status === 'error') return 'text-red-500'
  return 'text-gray-500'
}
</script>

<template>
  <div class="space-y-4 sm:space-y-6">
    <!-- 工具栏 -->
    <div class="flex items-center justify-between gap-2 sm:gap-3">
      <div class="flex items-center gap-2 sm:gap-3">
        <select
          class="px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
          :class="isDark ? 'bg-[var(--bg-secondary)] border-[var(--border)] text-white' : 'bg-white border-gray-200 text-gray-800'">
          <option value="">全部类型</option>
          <option v-for="t in channelTypes" :key="t.value" :value="t.value">{{ t.label }}</option>
        </select>
      </div>

      <div class="flex items-center gap-2 sm:gap-3">
        <el-tooltip content="刷新列表" placement="top">
          <button @click="loadChannels" class="p-2 sm:p-2.5 rounded-xl border transition-all hover:border-indigo-500"
            :class="isDark ? 'border-[var(--border)] bg-[var(--bg-secondary)]' : 'border-gray-200 bg-white'">
            <RefreshCw class="w-4 h-4 sm:w-5 sm:h-5" />
          </button>
        </el-tooltip>
        <button @click="openCreateDialog"
          class="btn-gradient px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl flex items-center gap-1 sm:gap-2 text-sm">
          <Plus class="w-4 h-4 sm:w-5 sm:h-5" />
          <span class="hidden sm:inline">添加渠道</span>
        </button>
      </div>
    </div>

    <!-- 渠道列表 -->
    <div v-if="loading" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4">
      <div v-for="i in 3" :key="i" class="h-48 sm:h-64 rounded-xl loading-shimmer"></div>
    </div>

    <div v-else-if="channels.length === 0" class="text-center py-20 sm:py-32">
      <div class="w-20 h-20 sm:w-24 sm:h-24 mx-auto rounded-2xl flex items-center justify-center mb-3 sm:mb-4"
        :class="isDark ? 'bg-[var(--bg-secondary)]' : 'bg-gray-100'">
        <Network class="w-10 h-10 sm:w-12 sm:h-12" :class="isDark ? 'text-gray-600' : 'text-gray-400'" />
      </div>
      <p class="text-sm sm:text-base" :class="isDark ? 'text-gray-400' : 'text-gray-500'">暂无渠道</p>
      <button @click="openCreateDialog" class="mt-3 btn-gradient px-5 sm:px-6 py-2 rounded-xl text-sm">
        添加第一个渠道
      </button>
    </div>

    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4">
      <div v-for="channel in channels" :key="channel.id"
        class="card p-4 sm:p-6 hover-lift animate-fade-in overflow-hidden">

        <!-- 头部 -->
        <div class="flex items-start justify-between mb-3 sm:mb-4">
          <div class="flex items-center gap-2 sm:gap-3">
            <div
              class="w-10 h-10 sm:w-12 sm:h-12 rounded-xl flex items-center justify-center shadow-lg shadow-indigo-500/25"
              :class="isDark ? 'bg-indigo-600' : 'bg-indigo-500'">
              <span class="text-white font-bold text-base sm:text-lg">{{ channel.type?.charAt(0).toUpperCase() || '?'
                }}</span>
            </div>
            <div>
              <p class="font-semibold text-sm sm:text-base">{{ channel.name }}</p>
              <p class="text-xs" :class="isDark ? 'text-gray-500' : 'text-gray-400'">{{ getTypeLabel(channel.type) }}
              </p>
            </div>
          </div>
          <div class="flex items-center gap-1 sm:gap-1.5">
            <span class="w-2 h-2 sm:w-2.5 sm:h-2.5 rounded-full" :class="{
              'bg-green-500': channel.enabled && channel.status === 'healthy',
              'bg-yellow-500': channel.enabled && channel.status === 'warning',
              'bg-red-500': !channel.enabled || channel.status === 'error',
              'bg-gray-500': !channel.enabled
            }"></span>
            <span class="text-xs font-medium" :class="getStatusColor(channel.status)">
              {{ channel.enabled ? (channel.status || '正常') : '已禁用' }}
            </span>
          </div>
        </div>

        <!-- 状态标签 -->
        <div class="flex items-center gap-2 mb-3 sm:mb-4">
          <span class="px-2 py-0.5 sm:py-1 rounded-lg text-xs font-medium" :class="channel.enabled
            ? 'bg-green-500/10 text-green-500'
            : 'bg-gray-500/10 text-gray-500'">
            {{ channel.enabled ? '已启用' : '已禁用' }}
          </span>
        </div>

        <!-- 存储 -->
        <div class="mb-3 sm:mb-4">
          <div class="flex items-center justify-between text-xs sm:text-sm mb-1.5 sm:mb-2">
            <span :class="isDark ? 'text-gray-400' : 'text-gray-500'">存储使用</span>
            <span class="text-xs sm:text-sm">
              <template v-if="channel.quotaEnabled && (channel.quotaLimit || channel.totalSpace)">
                {{ formatSize(channel.usedSpace) }} / {{ formatSize(channel.quotaLimit || channel.totalSpace) }}
              </template>
              <template v-else>
                {{ formatSize(channel.usedSpace) }} / 无限制
              </template>
            </span>
          </div>
          <div class="progress-bar" v-if="channel.quotaEnabled && (channel.quotaLimit || channel.totalSpace)">
            <div class="progress" :style="{
              width: ((channel.usedSpace / (channel.quotaLimit || channel.totalSpace || 1)) * 100) + '%',
              background: (channel.usedSpace / (channel.quotaLimit || channel.totalSpace || 1)) > 0.9 ? 'var(--danger)' :
                (channel.usedSpace / (channel.quotaLimit || channel.totalSpace || 1)) > 0.7 ? 'var(--warning)' : ''
            }"></div>
          </div>
        </div>

        <!-- 配额信息 -->
        <div class="grid grid-cols-3 gap-1.5 sm:gap-2 mb-3 sm:mb-4 text-center text-xs sm:text-sm">
          <div class="p-1.5 sm:p-2 rounded-lg" :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-50'">
            <Upload class="w-3 h-3 sm:w-4 sm:h-4 mx-auto mb-0.5 sm:mb-1"
              :class="isDark ? 'text-gray-500' : 'text-gray-400'" />
            <p class="font-medium">{{ channel.dailyUploads || 0 }}</p>
            <p class="text-xs" :class="isDark ? 'text-gray-500' : 'text-gray-400'">今日</p>
          </div>
          <div class="p-1.5 sm:p-2 rounded-lg" :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-50'">
            <Clock class="w-3 h-3 sm:w-4 sm:h-4 mx-auto mb-0.5 sm:mb-1"
              :class="isDark ? 'text-gray-500' : 'text-gray-400'" />
            <p class="font-medium">{{ channel.hourlyUploads || 0 }}</p>
            <p class="text-xs" :class="isDark ? 'text-gray-500' : 'text-gray-400'">小时</p>
          </div>
          <div class="p-1.5 sm:p-2 rounded-lg" :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-50'">
            <AlertTriangle class="w-3 h-3 sm:w-4 sm:h-4 mx-auto mb-0.5 sm:mb-1"
              :class="isDark ? 'text-gray-500' : 'text-gray-400'" />
            <p class="font-medium">{{ channel.quotaThreshold || 90 }}%</p>
            <p class="text-xs" :class="isDark ? 'text-gray-500' : 'text-gray-400'">阈值</p>
          </div>
        </div>

        <!-- 操作 -->
        <div class="flex items-center gap-0.5 sm:gap-2 pt-3 sm:pt-4 border-t overflow-hidden"
          :style="{ borderColor: 'var(--border)' }">
          <el-tooltip :content="channel.enabled ? '禁用此渠道' : '启用此渠道'" placement="top">
            <button @click="toggleChannel(channel)"
              class="flex-1 min-w-0 px-1.5 sm:px-3 py-1.5 sm:py-2 rounded-lg text-xs sm:text-sm font-medium transition-all truncate"
              :class="channel.enabled
                ? 'text-red-500 hover:bg-red-500/10'
                : 'text-green-500 hover:bg-green-500/10'">
              {{ channel.enabled ? '禁用' : '启用' }}
            </button>
          </el-tooltip>
          <el-tooltip content="测试连接" placement="top">
            <button @click="testChannel(channel)" :disabled="testingId === channel.id"
              class="flex-1 min-w-0 px-1.5 sm:px-3 py-1.5 sm:py-2 rounded-lg text-xs sm:text-sm font-medium transition-all truncate"
              :class="isDark ? 'hover:bg-[var(--bg-hover)]' : 'hover:bg-gray-100'">
              {{ testingId === channel.id ? '测试中' : '测试' }}
            </button>
          </el-tooltip>
          <el-tooltip content="编辑渠道" placement="top">
            <button @click="openEditDialog(channel)"
              class="flex-1 min-w-0 px-1.5 sm:px-3 py-1.5 sm:py-2 rounded-lg text-xs sm:text-sm font-medium transition-all truncate"
              :class="isDark ? 'hover:bg-[var(--bg-hover)]' : 'hover:bg-gray-100'">
              编辑
            </button>
          </el-tooltip>
          <el-tooltip :content="channel.type === 'local' ? '本地存储渠道不能删除' : '删除渠道'" placement="top">
            <button @click="deleteChannel(channel)" :disabled="channel.type === 'local'"
              class="flex-shrink-0 p-1.5 sm:p-1.5 rounded-lg transition-all flex items-center justify-center" :class="channel.type === 'local'
                ? 'text-gray-400 cursor-not-allowed'
                : 'text-red-500 hover:bg-red-500/10'">
              <Trash2 class="w-3.5 h-3.5 sm:w-4 sm:h-4" />
            </button>
          </el-tooltip>
        </div>
      </div>
    </div>

    <!-- 添加/编辑弹窗 -->
    <el-dialog v-model="showDialog" :title="dialogType === 'create' ? '添加渠道' : '编辑渠道'" width="600px"
      class="!max-w-[90vw] sm:!max-w-[600px]" :close-on-click-modal="false">
      <div class="space-y-4">
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 sm:gap-4">
          <div>
            <label class="block text-sm font-medium mb-2">渠道名称</label>
            <input v-model="form.name" type="text" placeholder="请输入渠道名称"
              class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
              :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)] text-white' : 'bg-gray-50 border-gray-200 text-gray-800'" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-2">存储类型</label>
            <select v-model="form.type" :disabled="dialogType === 'edit'"
              class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
              :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)] text-white' : 'bg-gray-50 border-gray-200 text-gray-800'">
              <option v-for="t in channelTypes" :key="t.value" :value="t.value">{{ t.label }}</option>
            </select>
          </div>
        </div>

        <!-- 配置 -->
        <div>
          <label class="block text-sm font-medium mb-2">配置信息</label>
          <div class="p-4 rounded-xl space-y-3"
            :class="isDark ? 'bg-[var(--bg-hover)] border border-[var(--border)]' : 'bg-gray-50 border border-gray-200'">

            <template v-if="form.type === 'local'">
              <input v-model="form.config.path" placeholder="存储路径，如 ./data" class="w-full px-3 py-2 rounded-lg border"
                :class="isDark ? 'bg-[var(--bg-card)] border-[var(--border)]' : 'bg-white border-gray-200'" />
            </template>

            <template v-else-if="form.type === 'telegram'">
              <input v-model="form.config.botToken" placeholder="Bot Token" class="w-full px-3 py-2 rounded-lg border"
                :class="isDark ? 'bg-[var(--bg-card)] border-[var(--border)]' : 'bg-white border-gray-200'" />
              <input v-model="form.config.channelId" placeholder="Channel ID" class="w-full px-3 py-2 rounded-lg border"
                :class="isDark ? 'bg-[var(--bg-card)] border-[var(--border)]' : 'bg-white border-gray-200'" />
              <input v-model="form.config.channelId2" placeholder="备用 Channel ID（可选，用于负载均衡）"
                class="w-full px-3 py-2 rounded-lg border"
                :class="isDark ? 'bg-[var(--bg-card)] border-[var(--border)]' : 'bg-white border-gray-200'" />
            </template>

            <template v-else-if="form.type === 'cfr2'">
              <p class="text-xs mb-2" :class="isDark ? 'text-gray-500' : 'text-gray-400'">
                请在 Cloudflare R2 控制台创建 API Token，获取 Access Key ID 和 Secret Access Key
              </p>
              <input v-model="form.config.accessKey" placeholder="Access Key ID"
                class="w-full px-3 py-2 rounded-lg border text-sm"
                :class="isDark ? 'bg-[var(--bg-card)] border-[var(--border)]' : 'bg-white border-gray-200'" />
              <input v-model="form.config.secretKey" placeholder="Secret Access Key" type="password"
                class="w-full px-3 py-2 rounded-lg border text-sm"
                :class="isDark ? 'bg-[var(--bg-card)] border-[var(--border)]' : 'bg-white border-gray-200'" />
              <input v-model="form.config.bucket" placeholder="Bucket 名称"
                class="w-full px-3 py-2 rounded-lg border text-sm"
                :class="isDark ? 'bg-[var(--bg-card)] border-[var(--border)]' : 'bg-white border-gray-200'" />
              <p class="text-xs -mt-1 mb-2 text-amber-500">⚠️ Bucket 名称不能包含空格</p>
              <input v-model="form.config.accountId" placeholder="Account ID"
                class="w-full px-3 py-2 rounded-lg border text-sm"
                :class="isDark ? 'bg-[var(--bg-card)] border-[var(--border)]' : 'bg-white border-gray-200'" />
              <p class="text-xs -mt-1 mb-2 text-amber-500">⚠️ 必须是 R2 账户页面显示的完整 Account ID</p>
              <div class="border-t pt-3 mt-3" :class="isDark ? 'border-[var(--border)]' : 'border-gray-200'">
                <p class="text-xs font-medium mb-2" :class="isDark ? 'text-gray-400' : 'text-gray-500'">
                  公共访问 URL（必填）
                </p>
                <p class="text-xs mb-2" :class="isDark ? 'text-gray-500' : 'text-gray-400'">
                  在 R2 存储桶设置中启用"公共访问"，填写自定义域名或公共开发 URL
                </p>
                <input v-model="form.config.publicUrl" placeholder="https://pub-xxx.r2.dev"
                  class="w-full px-3 py-2 rounded-lg border text-sm"
                  :class="isDark ? 'bg-[var(--bg-card)] border-[var(--border)]' : 'bg-white border-gray-200'" />
                <p class="text-xs mt-1 text-amber-500">
                  ⚠️ 不填写公共 URL 将无法直接访问文件
                </p>
              </div>
            </template>

            <template v-else-if="form.type === 's3'">
              <p class="text-xs mb-2" :class="isDark ? 'text-gray-500' : 'text-gray-400'">
                Endpoint 只需填写地域端点（如 cos.ap-guangzhou.myqcloud.com），SDK 会自动拼接 Bucket
              </p>
              <input v-model="form.config.accessKey" placeholder="Access Key ID"
                class="w-full px-3 py-2 rounded-lg border text-sm"
                :class="isDark ? 'bg-[var(--bg-card)] border-[var(--border)]' : 'bg-white border-gray-200'" />
              <input v-model="form.config.secretKey" placeholder="Secret Access Key" type="password"
                class="w-full px-3 py-2 rounded-lg border text-sm"
                :class="isDark ? 'bg-[var(--bg-card)] border-[var(--border)]' : 'bg-white border-gray-200'" />
              <input v-model="form.config.bucket" placeholder="Bucket 名称"
                class="w-full px-3 py-2 rounded-lg border text-sm"
                :class="isDark ? 'bg-[var(--bg-card)] border-[var(--border)]' : 'bg-white border-gray-200'" />
              <p class="text-xs -mt-1 mb-2 text-amber-500">⚠️ Bucket 名称不能包含空格</p>
              <input v-model="form.config.endpoint" placeholder="cos.ap-guangzhou.myqcloud.com"
                class="w-full px-3 py-2 rounded-lg border text-sm"
                :class="isDark ? 'bg-[var(--bg-card)] border-[var(--border)]' : 'bg-white border-gray-200'" />
              <p class="text-xs -mt-1 mb-2 text-amber-500">⚠️ 只需填地域端点，不要包含 Bucket 名称</p>
              <input v-model="form.config.region" placeholder="Region (如: ap-guangzhou)"
                class="w-full px-3 py-2 rounded-lg border text-sm"
                :class="isDark ? 'bg-[var(--bg-card)] border-[var(--border)]' : 'bg-white border-gray-200'" />
            </template>

            <template v-else-if="form.type === 'discord'">
              <p class="text-xs mb-2" :class="isDark ? 'text-gray-500' : 'text-gray-400'">
                请在 Discord 开发者平台创建 Bot，获取 Token 和邀请到频道
              </p>
              <input v-model="form.config.botToken" placeholder="Bot Token"
                class="w-full px-3 py-2 rounded-lg border text-sm"
                :class="isDark ? 'bg-[var(--bg-card)] border-[var(--border)]' : 'bg-white border-gray-200'" />
              <input v-model="form.config.channelId" placeholder="Channel ID"
                class="w-full px-3 py-2 rounded-lg border text-sm"
                :class="isDark ? 'bg-[var(--bg-card)] border-[var(--border)]' : 'bg-white border-gray-200'" />
              <label class="flex items-center gap-2 cursor-pointer mt-2">
                <input type="checkbox" v-model="form.config.isNitro" class="w-4 h-4 rounded accent-indigo-500" />
                <span class="text-sm" :class="isDark ? 'text-gray-400' : 'text-gray-500'">
                  Nitro 会员（支持 25MB，否则 8MB）
                </span>
              </label>
            </template>

            <template v-else-if="form.type === 'huggingface'">
              <p class="text-xs mb-3" :class="isDark ? 'text-gray-500' : 'text-gray-400'">
                请在 HuggingFace 设置中创建 Access Token，选择 write 权限
              </p>
              <input v-model="form.config.token" placeholder="HF Token (hf_xxxxxxxx)"
                class="w-full px-3 py-2 rounded-lg border text-sm"
                :class="isDark ? 'bg-[var(--bg-card)] border-[var(--border)]' : 'bg-white border-gray-200'" />
              <input v-model="form.config.repoId" placeholder="仓库 ID (如: username/imgbed-storage)"
                class="w-full px-3 py-2 rounded-lg border text-sm"
                :class="isDark ? 'bg-[var(--bg-card)] border-[var(--border)]' : 'bg-white border-gray-200'" />
              <select v-model="form.config.repoType" class="w-full px-3 py-2 rounded-lg border text-sm"
                :class="isDark ? 'bg-[var(--bg-card)] border-[var(--border)]' : 'bg-white border-gray-200'">
                <option value="dataset">Dataset (数据集)</option>
                <option value="model">Model (模型)</option>
                <option value="space">Space (空间)</option>
              </select>
              <p class="text-xs mt-2" :class="isDark ? 'text-gray-600' : 'text-gray-400'">
                文件将上传到 HuggingFace 数据集/模型仓库，可通过公开 URL 访问
              </p>
            </template>

            <template v-else>
              <p class="text-sm" :class="isDark ? 'text-gray-500' : 'text-gray-400'">请根据选择的类型填写相应配置</p>
            </template>
          </div>
        </div>

        <!-- 配额 -->
        <div class="border-t pt-4" :style="{ borderColor: 'var(--border)' }">
          <div class="flex items-center justify-between mb-3">
            <h3 class="font-medium">配额设置</h3>
            <label class="flex items-center gap-2 cursor-pointer">
              <input type="checkbox" v-model="form.quota.enabled" class="w-4 h-4 rounded accent-indigo-500" />
              <span class="text-sm" :class="isDark ? 'text-gray-400' : 'text-gray-500'">启用配额</span>
            </label>
          </div>
          <div class="grid grid-cols-1 sm:grid-cols-3 gap-3 sm:gap-4">
            <div>
              <label class="block text-xs mb-1" :class="isDark ? 'text-gray-500' : 'text-gray-400'">存储上限 (GB)</label>
              <input v-model.number="form.quota.limitGB" type="number" min="0"
                class="w-full px-3 py-2 rounded-lg border text-sm"
                :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
            </div>
            <div>
              <label class="block text-xs mb-1" :class="isDark ? 'text-gray-500' : 'text-gray-400'">告警阈值 (%)</label>
              <input v-model.number="form.quota.threshold" type="number" min="1" max="100"
                class="w-full px-3 py-2 rounded-lg border text-sm"
                :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
            </div>
            <div>
              <label class="block text-xs mb-1" :class="isDark ? 'text-gray-500' : 'text-gray-400'">每日上传次数</label>
              <input v-model.number="form.rateLimit.dailyUploadLimit" type="number" min="0"
                class="w-full px-3 py-2 rounded-lg border text-sm"
                :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
              <p class="text-xs mt-1" :class="isDark ? 'text-gray-600' : 'text-gray-400'">0 表示不限制</p>
            </div>
          </div>
        </div>
      </div>

      <template #footer>
        <div class="flex flex-col sm:flex-row justify-end gap-2">
          <button @click="showDialog = false" class="px-4 py-2 rounded-lg transition-all order-2 sm:order-1"
            :class="isDark ? 'hover:bg-[var(--bg-hover)]' : 'hover:bg-gray-100'">取消</button>
          <button @click="saveChannel" class="btn-gradient px-4 py-2 rounded-lg order-1 sm:order-2">保存</button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>
