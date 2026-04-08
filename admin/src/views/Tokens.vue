<script setup>
import { ref, onMounted } from 'vue'
import { tokenApi } from '@/api/token'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Key, Plus, RefreshCw, Trash2, Check, X, Copy, AlertTriangle
} from 'lucide-vue-next'

const isDark = ref(true)
const tokens = ref([])
const loading = ref(false)
const showDialog = ref(false)
const showTokenDialog = ref(false)
const newTokenInfo = ref(null)

const form = ref({
  name: '',
  permissions: ['upload', 'download'],
  expiresIn: 0
})

const permissionOptions = [
  { label: '上传', value: 'upload' },
  { label: '下载', value: 'download' },
  { label: '读取', value: 'read' },
  { label: '删除', value: 'delete' },
  { label: '全部权限', value: '*' }
]

const expiryOptions = [
  { label: '永不过期', value: 0 },
  { label: '7 天', value: 7 },
  { label: '30 天', value: 30 },
  { label: '90 天', value: 90 },
  { label: '365 天', value: 365 }
]

onMounted(() => {
  isDark.value = !document.documentElement.classList.contains('light')
  loadTokens()
})

async function loadTokens() {
  loading.value = true
  try {
    const res = await tokenApi.list()
    tokens.value = res.data || []
  } catch {
    ElMessage.error('加载 Token 列表失败')
  } finally {
    loading.value = false
  }
}

function openCreateDialog() {
  form.value = {
    name: '',
    permissions: ['upload', 'download'],
    expiresIn: 0
  }
  showDialog.value = true
}

async function createToken() {
  if (!form.value.name) {
    ElMessage.warning('请输入 Token 名称')
    return
  }

  try {
    const res = await tokenApi.create(form.value)
    newTokenInfo.value = res.data
    showDialog.value = false
    showTokenDialog.value = true
    loadTokens()
  } catch {
    ElMessage.error('创建失败')
  }
}

async function deleteToken(token) {
  try {
    await ElMessageBox.confirm(`确定要删除 Token「${token.name}」吗？删除后无法恢复。`, '删除确认', { type: 'warning' })
    await tokenApi.delete(token.token)
    ElMessage.success('删除成功')
    loadTokens()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error('删除失败')
  }
}

async function toggleToken(token) {
  try {
    await tokenApi.toggle(token.token, !token.enabled)
    ElMessage.success(token.enabled ? '已禁用' : '已启用')
    loadTokens()
  } catch {
    ElMessage.error('操作失败')
  }
}

async function copyToClipboard(text) {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制到剪贴板')
  } catch {
    ElMessage.error('复制失败')
  }
}

function formatDate(timestamp) {
  if (!timestamp || timestamp === 0) return '永不过期'
  return new Date(timestamp * 1000).toLocaleString('zh-CN')
}

function isExpired(token) {
  if (!token.expiresAt || token.expiresAt === 0) return false
  return Date.now() > token.expiresAt * 1000
}

function getPermissionLabel(perm) {
  const map = { upload: '上传', download: '下载', read: '读取', delete: '删除', '*': '全部' }
  return map[perm] || perm
}
</script>

<template>
  <div class="space-y-4 sm:space-y-6">
    <!-- 操作按钮 -->
    <div class="flex items-center justify-between gap-2">
      <div class="flex items-center gap-2 sm:gap-3">
        <el-tooltip content="刷新列表" placement="top">
          <button @click="loadTokens" class="p-2 sm:p-2.5 rounded-xl border transition-all hover:border-indigo-500"
            :class="isDark ? 'border-[var(--border)] bg-[var(--bg-secondary)]' : 'border-gray-200 bg-white'">
            <RefreshCw class="w-4 h-4 sm:w-5 sm:h-5" />
          </button>
        </el-tooltip>
        <button @click="openCreateDialog"
          class="btn-gradient px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl flex items-center gap-1 sm:gap-2 text-sm">
          <Plus class="w-4 h-4 sm:w-5 sm:h-5" />
          <span class="hidden sm:inline">创建 Token</span>
        </button>
      </div>
    </div>

    <!-- 提示信息 -->
    <div class="p-3 sm:p-4 rounded-xl border"
      :class="isDark ? 'bg-indigo-500/10 border-indigo-500/30' : 'bg-indigo-50 border-indigo-200'">
      <p class="text-xs sm:text-sm">
        <span class="font-medium text-indigo-500">使用说明</span>
        <span :class="isDark ? 'text-gray-400' : 'text-gray-600'"> API Token 用于程序化访问 ImgBed API。请在请求头中添加 </span>
        <code class="px-1 py-0.5 sm:px-1.5 sm:py-0.5 rounded text-xs sm:text-sm"
          :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-100'">
          X-API-Token 和 X-API-Secret
        </code>
        <span :class="isDark ? 'text-gray-400' : 'text-gray-600'"> 进行认证。</span>
      </p>
    </div>

    <!-- Token 列表 -->
    <div v-if="loading" class="grid grid-cols-1 md:grid-cols-2 gap-3 sm:gap-4">
      <div v-for="i in 4" :key="i" class="h-36 sm:h-40 rounded-xl loading-shimmer"></div>
    </div>

    <div v-else-if="tokens.length === 0" class="text-center py-16 sm:py-24">
      <div class="w-20 h-20 sm:w-24 sm:h-24 mx-auto rounded-2xl flex items-center justify-center mb-3 sm:mb-4"
        :class="isDark ? 'bg-[var(--bg-secondary)]' : 'bg-gray-100'">
        <Key class="w-10 h-10 sm:w-12 sm:h-12" :class="isDark ? 'text-gray-600' : 'text-gray-400'" />
      </div>
      <p class="text-sm sm:text-base" :class="isDark ? 'text-gray-400' : 'text-gray-500'">暂无 Token</p>
      <button @click="openCreateDialog" class="mt-3 btn-gradient px-5 sm:px-6 py-2 rounded-xl text-sm">
        创建第一个 Token
      </button>
    </div>

    <div v-else class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4 sm:gap-5">
      <div v-for="token in tokens" :key="token.token"
        class="group relative rounded-2xl border p-4 sm:p-5 transition-all duration-300 hover:shadow-xl hover:-translate-y-1"
        :class="isDark
          ? 'bg-[var(--bg-secondary)] border-[var(--border)] hover:border-indigo-500/50 hover:shadow-indigo-500/10'
          : 'bg-white border-gray-200 hover:border-indigo-400 hover:shadow-indigo-200'">

        <!-- 顶部状态条 -->
        <div class="absolute top-0 left-4 right-4 h-1 rounded-b-full opacity-0 group-hover:opacity-100 transition-opacity"
          :class="token.enabled ? 'bg-gradient-to-r from-green-400 to-emerald-500' : 'bg-gradient-to-r from-gray-400 to-gray-500'">
        </div>

        <!-- 头部 -->
        <div class="flex items-start justify-between mb-4">
          <div class="flex items-center gap-3">
            <div class="w-11 h-11 rounded-xl flex items-center justify-center transition-transform group-hover:scale-110"
              :class="token.enabled
                ? 'bg-gradient-to-br from-indigo-500 to-purple-600 shadow-lg shadow-indigo-500/30'
                : 'bg-gradient-to-br from-gray-400 to-gray-500 shadow-lg shadow-gray-500/20'">
              <Key class="w-5 h-5 text-white" />
            </div>
            <div>
              <p class="font-semibold text-sm sm:text-base" :class="isDark ? 'text-white' : 'text-gray-800'">{{ token.name }}</p>
              <span class="inline-flex items-center gap-1 text-xs font-medium mt-0.5"
                :class="isExpired(token) ? 'text-red-500' : token.enabled ? 'text-green-500' : 'text-gray-400'">
                <span class="w-1.5 h-1.5 rounded-full"
                  :class="isExpired(token) ? 'bg-red-500' : token.enabled ? 'bg-green-500 animate-pulse' : 'bg-gray-400'">
                </span>
                {{ isExpired(token) ? '已过期' : token.enabled ? '正常' : '已禁用' }}
              </span>
            </div>
          </div>
        </div>

        <!-- Token 信息卡 -->
        <div class="mb-4 p-3 rounded-xl" :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-50'">
          <div class="flex items-center justify-between gap-2">
            <div class="flex-1 min-w-0">
              <p class="text-xs" :class="isDark ? 'text-gray-500' : 'text-gray-400'">Token ID</p>
              <code class="text-xs sm:text-sm truncate block font-mono mt-0.5" :class="isDark ? 'text-gray-300' : 'text-gray-600'">
                {{ token.token.substring(0, 24) }}...
              </code>
            </div>
            <button @click="copyToClipboard(token.token)"
              class="p-2 rounded-lg transition-all hover:bg-indigo-500/10 flex-shrink-0"
              :class="isDark ? 'hover:text-indigo-400' : 'hover:text-indigo-500'">
              <Copy class="w-4 h-4" />
            </button>
          </div>
        </div>

        <!-- 权限标签 -->
        <div class="flex flex-wrap gap-1.5 mb-4">
          <span v-for="perm in token.permissions" :key="perm"
            class="px-2 py-0.5 rounded-md text-xs font-medium"
            :class="isDark
              ? 'bg-indigo-500/15 text-indigo-400'
              : 'bg-indigo-50 text-indigo-600'">
            {{ getPermissionLabel(perm) }}
          </span>
        </div>

        <!-- 信息行 -->
        <div class="flex items-center justify-between text-xs mb-4 pb-4 border-b" :class="isDark ? 'border-[var(--border)] text-gray-400' : 'border-gray-100 text-gray-500'">
          <div>
            <span class="opacity-60">过期</span>
            <span class="ml-1 font-medium" :class="isExpired(token) ? 'text-red-500' : ''">{{ formatDate(token.expiresAt) }}</span>
          </div>
          <div>
            <span class="opacity-60">使用</span>
            <span class="ml-1 font-medium">{{ token.lastUsedAt ? formatDate(token.lastUsedAt) : '从未' }}</span>
          </div>
        </div>

        <!-- 操作按钮 -->
        <div class="flex items-center gap-2">
          <button @click="toggleToken(token)"
            class="flex-1 py-2 px-3 rounded-xl text-xs font-medium transition-all flex items-center justify-center gap-1.5"
            :class="token.enabled
              ? 'bg-red-500/10 text-red-500 hover:bg-red-500/20'
              : 'bg-green-500/10 text-green-500 hover:bg-green-500/20'">
            <X v-if="token.enabled" class="w-3.5 h-3.5" />
            <Check v-else class="w-3.5 h-3.5" />
            {{ token.enabled ? '禁用' : '启用' }}
          </button>
          <button @click="deleteToken(token)"
            class="py-2 px-3 rounded-xl text-xs font-medium bg-red-500/10 text-red-500 hover:bg-red-500/20 transition-all">
            <Trash2 class="w-3.5 h-3.5" />
          </button>
        </div>
      </div>
    </div>

    <!-- 创建弹窗 -->
    <el-dialog v-model="showDialog" title="创建 API Token" width="90vw" class="!max-w-[500px]"
      :close-on-click-modal="false">
      <div class="space-y-3 sm:space-y-4">
        <div>
          <label class="block text-xs sm:text-sm font-medium mb-2">名称</label>
          <input v-model="form.name" type="text" placeholder="请输入 Token 名称"
            class="w-full px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
            :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)] text-white' : 'bg-gray-50 border-gray-200 text-gray-800'" />
        </div>

        <div>
          <label class="block text-xs sm:text-sm font-medium mb-2">权限</label>
          <div class="flex flex-wrap gap-2">
            <label v-for="opt in permissionOptions" :key="opt.value"
              class="flex items-center gap-1.5 sm:gap-2 px-2.5 sm:px-3 py-1.5 sm:py-2 rounded-lg cursor-pointer transition-all text-xs sm:text-sm"
              :class="form.permissions.includes(opt.value)
                ? 'bg-indigo-500/10 text-indigo-500'
                : isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-50'">
              <input type="checkbox" :value="opt.value" v-model="form.permissions"
                class="w-3.5 h-3.5 sm:w-4 sm:h-4 rounded accent-indigo-500" />
              {{ opt.label }}
            </label>
          </div>
        </div>

        <div>
          <label class="block text-xs sm:text-sm font-medium mb-2">有效期</label>
          <div class="flex flex-wrap gap-2">
            <button v-for="opt in expiryOptions" :key="opt.value" @click="form.expiresIn = opt.value"
              class="px-3 sm:px-4 py-1.5 sm:py-2 rounded-lg text-xs sm:text-sm font-medium transition-all" :class="form.expiresIn === opt.value
                ? 'bg-gradient-to-r from-indigo-500 to-purple-500 text-white shadow-lg'
                : isDark ? 'bg-[var(--bg-hover)] hover:bg-[var(--bg-secondary)]' : 'bg-gray-50 hover:bg-gray-100'">
              {{ opt.label }}
            </button>
          </div>
        </div>
      </div>

      <template #footer>
        <div class="flex flex-col sm:flex-row justify-end gap-2">
          <button @click="showDialog = false" class="px-4 py-2 rounded-lg transition-all order-2 sm:order-1"
            :class="isDark ? 'hover:bg-[var(--bg-hover)]' : 'hover:bg-gray-100'">取消</button>
          <button @click="createToken" class="btn-gradient px-4 py-2 rounded-lg order-1 sm:order-2">创建</button>
        </div>
      </template>
    </el-dialog>

    <!-- Token 创建成功弹窗 -->
    <el-dialog v-model="showTokenDialog" title="Token 创建成功" width="90vw" class="!max-w-[500px]"
      :close-on-click-modal="false">
      <div class="p-3 sm:p-4 rounded-xl border-2 border-red-500/50 bg-red-500/10 mb-3 sm:mb-4">
        <div class="flex items-start gap-2 sm:gap-3">
          <AlertTriangle class="w-4 h-4 sm:w-5 sm:h-5 text-red-500 flex-shrink-0 mt-0.5" />
          <div>
            <p class="text-xs sm:text-sm text-red-500 font-bold mb-1">重要提示</p>
            <p class="text-xs sm:text-sm text-red-400">Secret 仅显示一次，关闭后将无法再次查看！请务必立即复制并安全保存。</p>
          </div>
        </div>
      </div>

      <div class="space-y-3 sm:space-y-4">
        <div>
          <label class="block text-xs sm:text-sm font-medium mb-2">Token</label>
          <div class="flex items-center gap-2">
            <input :value="newTokenInfo?.token" readonly
              class="flex-1 px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border text-xs sm:text-sm min-w-0"
              :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
            <el-tooltip content="复制" placement="top">
              <button @click="copyToClipboard(newTokenInfo?.token)"
                class="p-2 sm:p-2.5 rounded-xl border transition-all hover:border-indigo-500 flex-shrink-0"
                :class="isDark ? 'border-[var(--border)]' : 'border-gray-200'">
                <Copy class="w-4 h-4 sm:w-5 sm:h-5" />
              </button>
            </el-tooltip>
          </div>
        </div>

        <div>
          <label class="block text-xs sm:text-sm font-medium mb-2">Secret</label>
          <div class="flex items-center gap-2">
            <input :value="newTokenInfo?.secret" readonly type="password"
              class="flex-1 px-3 sm:px-4 py-2 sm:py-2.5 rounded-xl border text-xs sm:text-sm min-w-0"
              :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
            <el-tooltip content="复制" placement="top">
              <button @click="copyToClipboard(newTokenInfo?.secret)"
                class="p-2 sm:p-2.5 rounded-xl border transition-all hover:border-indigo-500 flex-shrink-0"
                :class="isDark ? 'border-[var(--border)]' : 'border-gray-200'">
                <Copy class="w-4 h-4 sm:w-5 sm:h-5" />
              </button>
            </el-tooltip>
          </div>
        </div>
      </div>

      <template #footer>
        <button @click="showTokenDialog = false"
          class="btn-gradient px-4 py-2 rounded-lg w-full sm:w-auto">我已安全保存</button>
      </template>
    </el-dialog>
  </div>
</template>
