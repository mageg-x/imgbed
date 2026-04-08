<script setup>
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { channelApi } from '@/api/channel'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Network, Plus, Edit2, Trash2, RefreshCw, Check, X,
  AlertTriangle, Upload, Clock, Folder, Send, Cloud, HardDrive
} from 'lucide-vue-next'

const { t } = useI18n()

const isDark = ref(true)
const channels = ref([])
const loading = ref(false)
const showDialog = ref(false)
const dialogType = ref('create')
const editingChannel = ref(null)
const testingId = ref(null)

const channelTypes = computed(() => [
  { value: 'local', label: t('channels.localStorage') },
  { value: 'telegram', label: 'Telegram' },
  { value: 'cfr2', label: 'Cloudflare R2' },
  { value: 's3', label: t('channels.s3Compatible') },
  { value: 'discord', label: 'Discord' },
  { value: 'huggingface', label: 'HuggingFace' }
])

const form = ref({
  name: '',
  type: 'local',
  config: {},
  weight: 100,
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
    ElMessage.error(t('channels.loadFailed'))
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
    weight: 100,
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
    weight: channel.weight || 100,
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
    ElMessage.warning(t('channels.pleaseInputChannelName'))
    return
  }

  try {
    if (dialogType.value === 'create') {
      await channelApi.create(form.value)
      ElMessage.success(t('common.createSuccess'))
    } else {
      await channelApi.update(editingChannel.value, form.value)
      ElMessage.success(t('common.updateSuccess'))
    }
    showDialog.value = false
    loadChannels()
  } catch (e) {
    ElMessage.error(e.message || t('common.operateFailed'))
  }
}

async function deleteChannel(channel) {
  try {
    await ElMessageBox.confirm(t('channels.deleteConfirm', { name: channel.name }), t('common.confirmDelete'), { type: 'warning' })
    await channelApi.delete(channel.id)
    ElMessage.success(t('common.deleteSuccess'))
    loadChannels()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error(t('common.deleteFailed'))
  }
}

async function toggleChannel(channel) {
  try {
    if (channel.enabled) {
      await channelApi.disable(channel.id)
    } else {
      await channelApi.enable(channel.id)
    }
    ElMessage.success(channel.enabled ? t('common.disabled2') : t('common.enabled2'))
    loadChannels()
  } catch {
    ElMessage.error(t('common.operateFailed'))
  }
}

async function testChannel(channel) {
  testingId.value = channel.id
  try {
    const res = await channelApi.test(channel.id)
    if (res.data?.success) {
      ElMessage.success(t('channels.connectionSuccess'))
    } else {
      const errorMsg = res.data?.error || res.data?.message || t('channels.connectionFailed')
      const errorDetail = res.data?.detail || ''
      const fullError = errorDetail ? `${errorMsg}\n${errorDetail}` : errorMsg
      ElMessageBox.alert(fullError, t('channels.testFailed'), {
        confirmButtonText: t('common.confirm'),
        type: 'error',
        customClass: isDark.value ? 'dark-message-box' : ''
      })
    }
  } catch (err) {
    const errorMsg = err.response?.data?.message || err.message || t('channels.testFailed')
    ElMessageBox.alert(errorMsg, t('channels.testFailed'), {
      confirmButtonText: t('common.confirm'),
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
  return channelTypes.value.find(t => t.value === type)?.label || type
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
          <option value="">{{ t('channels.allTypes') }}</option>
          <option v-for="t in channelTypes" :key="t.value" :value="t.value">{{ t.label }}</option>
        </select>
      </div>

      <div class="flex items-center gap-2 sm:gap-3">
        <el-tooltip :content="t('common.refresh')" placement="top">
          <button @click="loadChannels" class="p-2 sm:p-2.5 rounded-xl border transition-all hover:border-indigo-500"
            :class="isDark ? 'border-[var(--border)] bg-[var(--bg-secondary)]' : 'border-gray-200 bg-white'">
            <RefreshCw class="w-4 h-4 sm:w-5 sm:h-5" />
          </button>
        </el-tooltip>
        <button @click="openCreateDialog"
          class="btn-gradient px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl flex items-center gap-1 sm:gap-2 text-sm">
          <Plus class="w-4 h-4 sm:w-5 sm:h-5" />
          <span class="hidden sm:inline">{{ t('channels.addChannel') }}</span>
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
      <p class="text-sm sm:text-base" :class="isDark ? 'text-gray-400' : 'text-gray-500'">{{ t('channels.noChannels') }}</p>
      <button @click="openCreateDialog" class="mt-3 btn-gradient px-5 sm:px-6 py-2 rounded-xl text-sm">
        {{ t('channels.addFirstChannel') }}
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
              {{ channel.enabled ? (channel.status || t('common.normal')) : t('common.disabled') }}
            </span>
          </div>
        </div>

        <!-- 状态标签 -->
        <div class="flex items-center gap-2 mb-3 sm:mb-4">
          <span class="px-2 py-0.5 sm:py-1 rounded-lg text-xs font-medium" :class="channel.enabled
            ? 'bg-green-500/10 text-green-500'
            : 'bg-gray-500/10 text-gray-500'">
            {{ channel.enabled ? t('common.enabled') : t('common.disabled') }}
          </span>
          <span class="px-2 py-0.5 sm:py-1 rounded-lg text-xs font-medium bg-indigo-500/10 text-indigo-500">
            {{ t('channels.weight') }} {{ channel.weight || 100 }}
          </span>
        </div>

        <!-- 存储 -->
        <div class="mb-3 sm:mb-4">
          <div class="flex items-center justify-between text-xs sm:text-sm mb-1.5 sm:mb-2">
            <span :class="isDark ? 'text-gray-400' : 'text-gray-500'">{{ t('channels.storageUsage') }}</span>
            <span class="text-xs sm:text-sm">
              <template v-if="channel.quotaEnabled && (channel.quotaLimit || channel.totalSpace)">
                {{ formatSize(channel.usedSpace) }} / {{ formatSize(channel.quotaLimit || channel.totalSpace) }}
              </template>
              <template v-else>
                {{ formatSize(channel.usedSpace) }} / {{ t('channels.noLimit') }}
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
            <p class="text-xs" :class="isDark ? 'text-gray-500' : 'text-gray-400'">{{ t('channels.today') }}</p>
          </div>
          <div class="p-1.5 sm:p-2 rounded-lg" :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-50'">
            <Clock class="w-3 h-3 sm:w-4 sm:h-4 mx-auto mb-0.5 sm:mb-1"
              :class="isDark ? 'text-gray-500' : 'text-gray-400'" />
            <p class="font-medium">{{ channel.hourlyUploads || 0 }}</p>
            <p class="text-xs" :class="isDark ? 'text-gray-500' : 'text-gray-400'">{{ t('channels.hour') }}</p>
          </div>
          <div class="p-1.5 sm:p-2 rounded-lg" :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-50'">
            <AlertTriangle class="w-3 h-3 sm:w-4 sm:h-4 mx-auto mb-0.5 sm:mb-1"
              :class="isDark ? 'text-gray-500' : 'text-gray-400'" />
            <p class="font-medium">{{ channel.quotaThreshold || 90 }}%</p>
            <p class="text-xs" :class="isDark ? 'text-gray-500' : 'text-gray-400'">{{ t('channels.threshold') }}</p>
          </div>
        </div>

        <!-- 操作 -->
        <div class="flex items-center gap-0.5 sm:gap-2 pt-3 sm:pt-4 border-t overflow-hidden"
          :style="{ borderColor: 'var(--border)' }">
          <el-tooltip :content="channel.enabled ? t('common.disabled2') : t('common.enabled2')" placement="top">
            <button @click="toggleChannel(channel)"
              class="flex-1 min-w-0 px-1.5 sm:px-3 py-1.5 sm:py-2 rounded-lg text-xs sm:text-sm font-medium transition-all truncate"
              :class="channel.enabled
                ? 'text-red-500 hover:bg-red-500/10'
                : 'text-green-500 hover:bg-green-500/10'">
              {{ channel.enabled ? t('common.disabled2') : t('common.enabled2') }}
            </button>
          </el-tooltip>
          <el-tooltip :content="t('channels.testConnection')" placement="top">
            <button @click="testChannel(channel)" :disabled="testingId === channel.id"
              class="flex-1 min-w-0 px-1.5 sm:px-3 py-1.5 sm:py-2 rounded-lg text-xs sm:text-sm font-medium transition-all truncate"
              :class="isDark ? 'hover:bg-[var(--bg-hover)]' : 'hover:bg-gray-100'">
              {{ testingId === channel.id ? t('common.testing') : t('common.test') }}
            </button>
          </el-tooltip>
          <el-tooltip :content="t('channels.editChannel')" placement="top">
            <button @click="openEditDialog(channel)"
              class="flex-1 min-w-0 px-1.5 sm:px-3 py-1.5 sm:py-2 rounded-lg text-xs sm:text-sm font-medium transition-all truncate"
              :class="isDark ? 'hover:bg-[var(--bg-hover)]' : 'hover:bg-gray-100'">
              {{ t('common.edit') }}
            </button>
          </el-tooltip>
          <el-tooltip :content="channel.type === 'local' ? t('channels.localChannelCannotDelete') : t('common.delete')" placement="top">
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
    <el-dialog v-model="showDialog" :title="dialogType === 'create' ? t('channels.addChannel') : t('channels.editChannel')" width="520px"
      class="!max-w-[95vw] channel-dialog" :close-on-click-modal="false">
      <div class="max-h-[70vh] overflow-y-auto pr-1 -mr-1">

        <!-- 基础信息卡片 -->
        <div class="dialog-card mb-3">
          <div class="flex items-center gap-2 mb-3">
            <div class="w-8 h-8 rounded-lg bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center shadow-lg shadow-indigo-500/30">
              <Network class="w-4 h-4 text-white" />
            </div>
            <div>
              <h3 class="text-sm font-semibold">{{ t('channels.basicInfo') }}</h3>
              <p class="text-xs" :class="isDark ? 'text-gray-500' : 'text-gray-400'">{{ t('channels.basicInfoTip') }}</p>
            </div>
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-xs font-medium mb-1.5" :class="isDark ? 'text-gray-400' : 'text-gray-500'">{{ t('channels.channelName') }}</label>
              <input v-model="form.name" type="text" :placeholder="t('channels.channelNamePlaceholder')"
                class="dialog-input w-full" />
            </div>
            <div>
              <label class="block text-xs font-medium mb-1.5" :class="isDark ? 'text-gray-400' : 'text-gray-500'">{{ t('channels.storageType') }}</label>
              <select v-model="form.type" :disabled="dialogType === 'edit'" class="dialog-input w-full">
                <option v-for="t in channelTypes" :key="t.value" :value="t.value">{{ t.label }}</option>
              </select>
            </div>
          </div>
        </div>

        <!-- 权重卡片 -->
        <div class="dialog-card mb-3">
          <div class="flex items-center gap-2 mb-3">
            <div class="w-8 h-8 rounded-lg bg-gradient-to-br from-amber-500 to-orange-600 flex items-center justify-center shadow-lg shadow-amber-500/30">
              <Folder class="w-4 h-4 text-white" />
            </div>
            <div>
              <h3 class="text-sm font-semibold">{{ t('channels.weightConfig') }}</h3>
              <p class="text-xs" :class="isDark ? 'text-gray-500' : 'text-gray-400'">{{ t('channels.weightTip') }}</p>
            </div>
          </div>
          <div class="flex items-center gap-4">
            <input v-model.number="form.weight" type="range" min="1" max="100"
              class="weight-slider flex-1"
              :style="{ background: `linear-gradient(to right, #f59e0b 0%, #f59e0b ${form.weight}%, ${isDark ? '#374151' : '#e5e7eb'} ${form.weight}%, ${isDark ? '#374151' : '#e5e7eb'} 100%)` }" />
            <div class="w-14 h-9 rounded-lg bg-gradient-to-br from-amber-500 to-orange-500 flex items-center justify-center shadow-lg shadow-amber-500/20">
              <span class="text-white font-bold text-sm">{{ form.weight }}</span>
            </div>
          </div>
        </div>

        <!-- 配置卡片 -->
        <div class="dialog-card mb-3">
          <div class="flex items-center gap-2 mb-3">
            <div class="w-8 h-8 rounded-lg bg-gradient-to-br from-green-500 to-emerald-600 flex items-center justify-center shadow-lg shadow-green-500/30">
              <component :is="getTypeIcon(form.type)" class="w-4 h-4 text-white" />
            </div>
            <div>
              <h3 class="text-sm font-semibold">{{ t('channels.connectionConfig') }}</h3>
              <p class="text-xs" :class="isDark ? 'text-gray-500' : 'text-gray-400'">{{ getTypeLabel(form.type) }} {{ t('channels.connectionSettings') }}</p>
            </div>
          </div>

          <div class="space-y-2.5">
            <template v-if="form.type === 'local'">
              <div>
                <label class="dialog-label">{{ t('channels.storagePath') }}</label>
                <input v-model="form.config.path" placeholder="./data" class="dialog-input w-full" />
              </div>
            </template>

            <template v-else-if="form.type === 'telegram'">
              <div>
                <label class="dialog-label">Bot Token</label>
                <input v-model="form.config.botToken" placeholder="123456:ABC-DEF..." class="dialog-input w-full" />
              </div>
              <div>
                <label class="dialog-label">Channel ID</label>
                <input v-model="form.config.channelId" placeholder="-1001234567890" class="dialog-input w-full" />
              </div>
              <div>
                <label class="dialog-label">备用 Channel（可选）</label>
                <input v-model="form.config.channelId2" placeholder="-1009876543210" class="dialog-input w-full" />
              </div>
            </template>

            <template v-else-if="form.type === 'cfr2'">
              <div>
                <label class="dialog-label">Access Key ID</label>
                <input v-model="form.config.accessKey" placeholder="xxxxxxxxxxxxxxxxxxxx" class="dialog-input w-full" />
              </div>
              <div>
                <label class="dialog-label">Secret Access Key</label>
                <input v-model="form.config.secretKey" placeholder="xxxxxxxxxxxxxxxxxxxx" type="password" class="dialog-input w-full" />
              </div>
              <div class="grid grid-cols-2 gap-2.5">
                <div>
                  <label class="dialog-label">Bucket</label>
                  <input v-model="form.config.bucket" placeholder="my-bucket" class="dialog-input w-full" />
                </div>
                <div>
                  <label class="dialog-label">Account ID</label>
                  <input v-model="form.config.accountId" placeholder="xxxxxxxxxxxx" class="dialog-input w-full" />
                </div>
              </div>
              <div>
                <label class="dialog-label">公共 URL</label>
                <input v-model="form.config.publicUrl" placeholder="https://pub-xxx.r2.dev" class="dialog-input w-full" />
              </div>
            </template>

            <template v-else-if="form.type === 's3'">
              <div class="grid grid-cols-2 gap-2.5">
                <div>
                  <label class="dialog-label">Access Key</label>
                  <input v-model="form.config.accessKey" placeholder="xxxxxxxx" class="dialog-input w-full" />
                </div>
                <div>
                  <label class="dialog-label">Secret Key</label>
                  <input v-model="form.config.secretKey" placeholder="xxxxxxxx" type="password" class="dialog-input w-full" />
                </div>
              </div>
              <div class="grid grid-cols-2 gap-2.5">
                <div>
                  <label class="dialog-label">Bucket</label>
                  <input v-model="form.config.bucket" placeholder="my-bucket" class="dialog-input w-full" />
                </div>
                <div>
                  <label class="dialog-label">Endpoint</label>
                  <input v-model="form.config.endpoint" placeholder="cos.ap-guangzhou..." class="dialog-input w-full" />
                </div>
              </div>
              <div>
                <label class="dialog-label">Region</label>
                <input v-model="form.config.region" placeholder="ap-guangzhou" class="dialog-input w-full" />
              </div>
            </template>

            <template v-else-if="form.type === 'discord'">
              <div>
                <label class="dialog-label">Bot Token</label>
                <input v-model="form.config.botToken" placeholder="xxxxxxxx.xxxxxx.xxxxxx" class="dialog-input w-full" />
              </div>
              <div>
                <label class="dialog-label">Channel ID</label>
                <input v-model="form.config.channelId" placeholder="123456789012345678" class="dialog-input w-full" />
              </div>
              <label class="flex items-center gap-2 cursor-pointer py-1">
                <input type="checkbox" v-model="form.config.isNitro" class="w-4 h-4 rounded accent-amber-500" />
                <span class="text-sm" :class="isDark ? 'text-gray-400' : 'text-gray-500'">{{ t('channels.nitroTip') }}</span>
              </label>
            </template>

            <template v-else-if="form.type === 'huggingface'">
              <div>
                <label class="dialog-label">HF Token</label>
                <input v-model="form.config.token" placeholder="hf_xxxxxxxxxxxxxxxxxxxx" class="dialog-input w-full" />
              </div>
              <div class="grid grid-cols-2 gap-2.5">
                <div>
                  <label class="dialog-label">仓库 ID</label>
                  <input v-model="form.config.repoId" placeholder="username/imgbed" class="dialog-input w-full" />
                </div>
                <div>
                  <label class="dialog-label">类型</label>
                  <select v-model="form.config.repoType" class="dialog-input w-full">
                    <option value="dataset">Dataset</option>
                    <option value="model">Model</option>
                    <option value="space">Space</option>
                  </select>
                </div>
              </div>
            </template>
          </div>
        </div>

        <!-- 配额卡片 -->
        <div class="dialog-card">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
              <div class="w-8 h-8 rounded-lg bg-gradient-to-br from-rose-500 to-pink-600 flex items-center justify-center shadow-lg shadow-rose-500/30">
                <HardDrive class="w-4 h-4 text-white" />
              </div>
              <div>
                <h3 class="text-sm font-semibold">{{ t('channels.quotaConfig') }}</h3>
                <p class="text-xs" :class="isDark ? 'text-gray-500' : 'text-gray-400'">{{ t('channels.quotaTip') }}</p>
              </div>
            </div>
            <label class="relative inline-flex items-center cursor-pointer">
              <input type="checkbox" v-model="form.quota.enabled" class="sr-only peer" />
              <div class="w-11 h-6 rounded-full peer transition-colors
                peer-checked:after:translate-x-full
                after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all
                bg-gray-300 peer-checked:bg-gradient-to-r peer-checked:from-rose-500 peer-checked:to-pink-500">
              </div>
            </label>
          </div>
          <div v-if="form.quota.enabled" class="mt-3 pt-3 border-t" :class="isDark ? 'border-gray-700/50' : 'border-gray-200'">
            <div class="grid grid-cols-3 gap-2.5">
              <div>
                <label class="dialog-label">{{ t('channels.storageLimit') }}</label>
                <input v-model.number="form.quota.limitGB" type="number" min="0" placeholder="0" class="dialog-input w-full" />
              </div>
              <div>
                <label class="dialog-label">{{ t('channels.alarmThreshold') }}</label>
                <input v-model.number="form.quota.threshold" type="number" min="1" max="100" placeholder="90" class="dialog-input w-full" />
              </div>
              <div>
                <label class="dialog-label">{{ t('channels.dailyUpload') }}</label>
                <input v-model.number="form.rateLimit.dailyUploadLimit" type="number" min="0" placeholder="0" class="dialog-input w-full" />
              </div>
            </div>
          </div>
        </div>
      </div>

      <template #footer>
        <div class="flex justify-end gap-3 pt-3 border-t" :class="isDark ? 'border-gray-700/50' : 'border-gray-200'">
          <button @click="showDialog = false" class="px-5 py-2.5 rounded-xl transition-all text-sm font-medium"
            :class="isDark ? 'bg-gray-700/50 hover:bg-gray-600/50 text-gray-300' : 'bg-gray-100 hover:bg-gray-200 text-gray-700'">
            {{ t('common.cancel') }}
          </button>
          <button @click="saveChannel" class="btn-gradient px-6 py-2.5 rounded-xl text-sm font-medium shadow-lg">
            {{ dialogType === 'create' ? t('channels.addChannel') : t('common.save') }}
          </button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
/* 滑块样式 */
.weight-slider {
  -webkit-appearance: none;
  appearance: none;
  height: 6px;
  border-radius: 3px;
}

.weight-slider::-webkit-slider-thumb {
  -webkit-appearance: none;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: #f59e0b;
  cursor: pointer;
  box-shadow: 0 2px 6px rgba(245, 158, 11, 0.4);
  transition: transform 0.15s;
}

.weight-slider::-webkit-slider-thumb:hover {
  transform: scale(1.15);
}

.weight-slider::-moz-range-thumb {
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: #f59e0b;
  border: none;
  cursor: pointer;
  box-shadow: 0 2px 6px rgba(245, 158, 11, 0.4);
}

/* 对话框样式 */
:deep(.channel-dialog) {
  border-radius: 16px;
  overflow: hidden;
}

:deep(.channel-dialog .el-dialog__header) {
  padding: 16px 20px 12px;
  margin: 0;
  border-bottom: 1px solid transparent;
}

:deep(.channel-dialog .el-dialog__title) {
  font-size: 16px;
  font-weight: 600;
}

:deep(.channel-dialog .el-dialog__body) {
  padding: 16px 20px;
}

/* 卡片样式 */
.dialog-card {
  padding: 14px;
  border-radius: 12px;
  border: 1px solid;
  transition: all 0.2s;
}

:deep(.dark) .dialog-card {
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.08) 0%, rgba(139, 92, 246, 0.05) 100%);
  border-color: rgba(99, 102, 241, 0.2);
}

:deep(.light) .dialog-card {
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.04) 0%, rgba(139, 92, 246, 0.02) 100%);
  border-color: rgba(99, 102, 241, 0.15);
}

:deep(.dark) .dialog-card:hover {
  border-color: rgba(99, 102, 241, 0.35);
  box-shadow: 0 0 20px rgba(99, 102, 241, 0.1);
}

:deep(.light) .dialog-card:hover {
  border-color: rgba(99, 102, 241, 0.3);
  box-shadow: 0 4px 12px rgba(99, 102, 241, 0.08);
}

/* 输入框样式 */
.dialog-input {
  padding: 8px 12px;
  border-radius: 8px;
  border: 1px solid;
  font-size: 13px;
  transition: all 0.2s;
}

:deep(.dark) .dialog-input {
  background: rgba(0, 0, 0, 0.2);
  border-color: rgba(255, 255, 255, 0.1);
  color: #f1f5f9;
}

:deep(.dark) .dialog-input:focus {
  border-color: rgba(99, 102, 241, 0.5);
  background: rgba(0, 0, 0, 0.3);
  outline: none;
  box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.15);
}

:deep(.dark) .dialog-input::placeholder {
  color: rgba(255, 255, 255, 0.3);
}

:deep(.light) .dialog-input {
  background: rgba(255, 255, 255, 0.8);
  border-color: rgba(0, 0, 0, 0.1);
  color: #1e293b;
}

:deep(.light) .dialog-input:focus {
  border-color: rgba(99, 102, 241, 0.5);
  background: #fff;
  outline: none;
  box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.1);
}

:deep(.light) .dialog-input::placeholder {
  color: #94a3b8;
}

.dialog-label {
  display: block;
  font-size: 11px;
  margin-bottom: 6px;
  font-weight: 500;
}

:deep(.dark) .dialog-label {
  color: rgba(255, 255, 255, 0.5);
}

:deep(.light) .dialog-label {
  color: #64748b;
}

/* select 样式 */
:deep(.dark) .dialog-input option {
  background: #1e1e2e;
  color: #f1f5f9;
}

:deep(.light) .dialog-input option {
  background: #fff;
  color: #1e293b;
}
</style>
