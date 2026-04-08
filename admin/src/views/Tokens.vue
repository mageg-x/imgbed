<script setup>
import { ref, onMounted } from 'vue'
import { tokenApi } from '@/api/token'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Key, Plus, RefreshCw, Trash2, Check, X, Copy, AlertTriangle, Eye, EyeOff, Shield
} from 'lucide-vue-next'

const isDark = ref(true)
const tokens = ref([])
const loading = ref(false)
const showDialog = ref(false)
const showTokenDialog = ref(false)
const newTokenInfo = ref(null)
const showSecrets = ref({})

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
  { label: '全部', value: '*' }
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
    // 后端返回的 permissions 是逗号分隔字符串，需要转为数组
    tokens.value = (res.data || []).map(t => ({
      ...t,
      permissions: typeof t.permissions === 'string'
        ? t.permissions.split(',').map(p => p.trim())
        : t.permissions
    }))
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

function getPermissionColor(perm) {
  const colors = {
    upload: 'from-blue-500 to-cyan-500',
    download: 'from-emerald-500 to-teal-500',
    read: 'from-purple-500 to-pink-500',
    delete: 'from-red-500 to-orange-500',
    '*': 'from-indigo-500 to-violet-500'
  }
  return colors[perm] || 'from-gray-500 to-gray-600'
}

function toggleSecretVisibility(token) {
  showSecrets.value[token.token] = !showSecrets.value[token.token]
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
    <div class="p-3 sm:p-4 rounded-2xl border"
      :class="isDark ? 'bg-indigo-500/10 border-indigo-500/30' : 'bg-indigo-50 border-indigo-200'">
      <div class="flex items-start gap-2">
        <Shield class="w-4 h-4 sm:w-5 sm:h-5 text-indigo-500 flex-shrink-0 mt-0.5" />
        <p class="text-xs sm:text-sm flex-1">
          <span class="font-medium text-indigo-500">使用说明</span>
          <span :class="isDark ? 'text-gray-400' : 'text-gray-600'"> API Token 用于程序化访问。请在请求头中添加 </span>
          <code class="px-1 py-0.5 sm:px-1.5 sm:py-0.5 rounded text-xs sm:text-sm font-mono"
            :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-100'">
            X-API-Token
          </code>
          <span :class="isDark ? 'text-gray-400' : 'text-gray-600'"> 和 </span>
          <code class="px-1 py-0.5 sm:px-1.5 sm:py-0.5 rounded text-xs sm:text-sm font-mono"
            :class="isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-100'">
            X-API-Secret
          </code>
          <span :class="isDark ? 'text-gray-400' : 'text-gray-600'"> 进行认证。</span>
        </p>
      </div>
    </div>

    <!-- Token 列表 -->
    <div v-if="loading" class="grid grid-cols-1 md:grid-cols-2 gap-3 sm:gap-4">
      <div v-for="i in 4" :key="i" class="h-48 sm:h-52 rounded-2xl loading-shimmer"></div>
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
        class="group relative rounded-2xl border overflow-hidden transition-all duration-300 hover:shadow-2xl hover:-translate-y-1"
        :class="isDark
          ? 'bg-[var(--bg-secondary)] border-[var(--border)] hover:border-indigo-500/50'
          : 'bg-white border-gray-200 hover:border-indigo-400 hover:shadow-indigo-200'">

        <!-- 顶部渐变状态条 -->
        <div class="h-1.5"
          :class="isExpired(token) ? 'bg-gradient-to-r from-red-500 to-orange-500' : token.enabled ? 'bg-gradient-to-r from-emerald-400 via-cyan-500 to-indigo-500' : 'bg-gradient-to-r from-gray-400 to-gray-500'">
        </div>

        <div class="p-4 sm:p-5">
          <!-- 头部区域 -->
          <div class="flex items-start justify-between mb-4">
            <div class="flex items-center gap-3">
              <div class="w-12 h-12 rounded-xl flex items-center justify-center transition-transform group-hover:scale-110 shadow-lg"
                :class="isExpired(token) ? 'bg-gradient-to-br from-red-500 to-orange-600 shadow-red-500/30' : token.enabled ? 'bg-gradient-to-br from-indigo-500 to-purple-600 shadow-indigo-500/30' : 'bg-gradient-to-br from-gray-400 to-gray-500 shadow-gray-500/20'">
                <Key class="w-5 h-5 text-white" />
              </div>
              <div>
                <p class="font-bold text-sm sm:text-base" :class="isDark ? 'text-white' : 'text-gray-800'">{{ token.name }}</p>
                <div class="flex items-center gap-1.5 mt-1">
                  <span class="w-2 h-2 rounded-full animate-pulse"
                    :class="isExpired(token) ? 'bg-red-500' : token.enabled ? 'bg-emerald-500' : 'bg-gray-400'">
                  </span>
                  <span class="text-xs font-medium"
                    :class="isExpired(token) ? 'text-red-500' : token.enabled ? 'text-emerald-500' : 'text-gray-400'">
                    {{ isExpired(token) ? '已过期' : token.enabled ? '正常' : '已禁用' }}
                  </span>
                </div>
              </div>
            </div>
          </div>

          <!-- Token ID -->
          <div class="mb-3">
            <div class="flex items-center justify-between gap-2 mb-1">
              <span class="text-xs font-medium" :class="isDark ? 'text-indigo-400' : 'text-indigo-600'">Token ID</span>
              <button @click="copyToClipboard(token.token)"
                class="p-1 rounded transition-all hover:bg-indigo-500/10"
                :class="isDark ? 'hover:text-indigo-400 text-gray-500' : 'hover:text-indigo-500 text-gray-400'">
                <Copy class="w-3.5 h-3.5" />
              </button>
            </div>
            <p class="text-xs font-mono truncate" :class="isDark ? 'text-gray-300' : 'text-gray-600'">
              {{ token.token.substring(0, 32) }}...
            </p>
          </div>

          <!-- 权限 -->
          <div class="mb-3">
            <p class="text-xs font-medium mb-2" :class="isDark ? 'text-indigo-400' : 'text-indigo-600'">权限</p>
            <div class="flex flex-wrap gap-1.5">
              <span v-for="perm in token.permissions" :key="perm"
                class="px-2.5 py-1 rounded-lg text-xs font-medium bg-gradient-to-r text-white shadow-sm"
                :class="getPermissionColor(perm)">
                {{ getPermissionLabel(perm) }}
              </span>
            </div>
          </div>

          <!-- 信息网格 -->
          <div class="grid grid-cols-2 gap-2 mb-4">
            <div>
              <p class="text-xs font-medium" :class="isDark ? 'text-indigo-400' : 'text-indigo-600'">过期时间</p>
              <p class="text-xs font-medium mt-0.5" :class="isExpired(token) ? 'text-red-500' : isDark ? 'text-gray-300' : 'text-gray-600'">
                {{ formatDate(token.expiresAt) }}
              </p>
            </div>
            <div>
              <p class="text-xs font-medium" :class="isDark ? 'text-indigo-400' : 'text-indigo-600'">最后使用</p>
              <p class="text-xs font-medium mt-0.5" :class="isDark ? 'text-gray-300' : 'text-gray-600'">
                {{ token.lastUsedAt ? formatDate(token.lastUsedAt) : '从未' }}
              </p>
            </div>
          </div>

          <!-- 操作按钮 -->
          <div class="flex justify-center gap-2 sm:gap-3 md:gap-4">
            <button @click="toggleToken(token)"
              class="w-14 h-10 rounded-xl transition-all flex items-center justify-center gap-1.5"
              :class="token.enabled
                ? 'bg-red-500/10 text-red-500 hover:bg-red-500/20 border border-red-500/20'
                : 'bg-emerald-500/10 text-emerald-500 hover:bg-emerald-500/20 border border-emerald-500/20'">
              <X v-if="token.enabled" class="w-3.5 h-3.5" />
              <Check v-else class="w-3.5 h-3.5" />
              <span class="text-xs">{{ token.enabled ? '禁用' : '启用' }}</span>
            </button>
            <button @click="deleteToken(token)"
              class="w-14 h-10 rounded-xl bg-red-500/10 text-red-500 hover:bg-red-500/20 transition-all border border-red-500/20 flex items-center justify-center gap-1.5">
              <Trash2 class="w-3.5 h-3.5" />
              <span class="text-xs">删除</span>
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 创建弹窗 -->
    <el-dialog v-model="showDialog" title="创建 API Token" width="90vw" class="!max-w-[480px] token-dialog" :close-on-click-modal="false">
      <div class="space-y-4">
        <!-- 名称 -->
        <div>
          <label class="block text-xs sm:text-sm font-medium mb-2">名称</label>
          <input v-model="form.name" type="text" placeholder="请输入 Token 名称"
            class="w-full px-4 py-2.5 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50 text-sm"
            :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)] text-white' : 'bg-gray-50 border-gray-200 text-gray-800'" />
        </div>

        <!-- 权限 -->
        <div>
          <label class="block text-xs sm:text-sm font-medium mb-2">权限</label>
          <div class="flex flex-wrap gap-2">
            <button v-for="opt in permissionOptions" :key="opt.value" @click="
              form.permissions.includes(opt.value)
                ? form.permissions = form.permissions.filter(p => p !== opt.value)
                : form.permissions.push(opt.value)
            " class="flex items-center gap-2 px-3 py-2 rounded-xl text-xs sm:text-sm font-medium transition-all border"
              :class="form.permissions.includes(opt.value)
                ? 'bg-gradient-to-r from-indigo-500 to-purple-500 text-white border-transparent shadow-lg'
                : isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'">
              <Check v-if="form.permissions.includes(opt.value)" class="w-3.5 h-3.5" />
              {{ opt.label }}
            </button>
          </div>
        </div>

        <!-- 有效期 -->
        <div>
          <label class="block text-xs sm:text-sm font-medium mb-2">有效期</label>
          <div class="flex flex-wrap gap-2">
            <button v-for="opt in expiryOptions" :key="opt.value" @click="form.expiresIn = opt.value"
              class="px-4 py-2 rounded-xl text-xs sm:text-sm font-medium transition-all border"
              :class="form.expiresIn === opt.value
                ? 'bg-gradient-to-r from-indigo-500 to-purple-500 text-white border-transparent shadow-lg'
                : isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'">
              {{ opt.label }}
            </button>
          </div>
        </div>
      </div>

      <template #footer>
        <div class="flex flex-col sm:flex-row justify-end gap-2">
          <button @click="showDialog = false" class="px-4 py-2 rounded-xl transition-all"
            :class="isDark ? 'hover:bg-[var(--bg-hover)]' : 'hover:bg-gray-100'">取消</button>
          <button @click="createToken" class="btn-gradient px-4 py-2 rounded-xl">创建</button>
        </div>
      </template>
    </el-dialog>

    <!-- Token 创建成功弹窗 -->
    <el-dialog v-model="showTokenDialog" title="Token 创建成功" width="90vw" class="!max-w-[480px] token-dialog" :close-on-click-modal="false">
      <div class="p-4 rounded-2xl border-2 border-red-500/30 bg-red-500/10 mb-4">
        <div class="flex items-start gap-3">
          <AlertTriangle class="w-5 h-5 text-red-500 flex-shrink-0 mt-0.5" />
          <div>
            <p class="text-sm text-red-500 font-bold mb-1">重要提示</p>
            <p class="text-xs sm:text-sm text-red-400">Secret 仅显示一次，关闭后将无法再次查看！请务必立即复制并安全保存。</p>
          </div>
        </div>
      </div>

      <div class="space-y-4">
        <div>
          <label class="block text-xs sm:text-sm font-medium mb-2">Token</label>
          <div class="flex items-center gap-2">
            <input :value="newTokenInfo?.token" readonly
              class="flex-1 px-4 py-2.5 rounded-xl border text-xs sm:text-sm font-mono min-w-0"
              :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
            <button @click="copyToClipboard(newTokenInfo?.token)"
              class="p-2.5 rounded-xl border transition-all hover:border-indigo-500 flex-shrink-0"
              :class="isDark ? 'border-[var(--border)]' : 'border-gray-200'">
              <Copy class="w-4 h-4 sm:w-5 sm:h-5" />
            </button>
          </div>
        </div>

        <div>
          <label class="block text-xs sm:text-sm font-medium mb-2">Secret</label>
          <div class="flex items-center gap-2">
            <input :value="newTokenInfo?.secret" readonly :type="showSecrets[newTokenInfo?.token] ? 'text' : 'password'"
              class="flex-1 px-4 py-2.5 rounded-xl border text-xs sm:text-sm font-mono min-w-0"
              :class="isDark ? 'bg-[var(--bg-hover)] border-[var(--border)]' : 'bg-gray-50 border-gray-200'" />
            <button @click="toggleSecretVisibility(newTokenInfo)"
              class="p-2.5 rounded-xl border transition-all hover:border-indigo-500 flex-shrink-0"
              :class="isDark ? 'border-[var(--border)]' : 'border-gray-200'">
              <Eye v-if="showSecrets[newTokenInfo?.token]" class="w-4 h-4 sm:w-5 sm:h-5" />
              <EyeOff v-else class="w-4 h-4 sm:w-5 sm:h-5" />
            </button>
            <button @click="copyToClipboard(newTokenInfo?.secret)"
              class="p-2.5 rounded-xl border transition-all hover:border-indigo-500 flex-shrink-0"
              :class="isDark ? 'border-[var(--border)]' : 'border-gray-200'">
              <Copy class="w-4 h-4 sm:w-5 sm:h-5" />
            </button>
          </div>
        </div>
      </div>

      <template #footer>
        <button @click="showTokenDialog = false"
          class="btn-gradient px-4 py-2.5 rounded-xl w-full sm:w-auto font-medium">我已安全保存</button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.token-dialog :deep(.el-dialog) {
  border-radius: 1rem;
}

.token-dialog :deep(.el-dialog__header) {
  padding: 1rem 1.5rem;
  border-bottom: 1px solid var(--el-border-color);
  margin-right: 0;
}

.token-dialog :deep(.el-dialog__body) {
  padding: 1.5rem;
}

.token-dialog :deep(.el-dialog__footer) {
  padding: 1rem 1.5rem;
  border-top: 1px solid var(--el-border-color);
}
</style>
