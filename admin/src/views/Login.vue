<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useThemeStore } from '@/stores/theme'
import { ElMessage } from 'element-plus'
import { Lock, Sun, Moon, ArrowLeft } from 'lucide-vue-next'

const router = useRouter()
const authStore = useAuthStore()
const themeStore = useThemeStore()

const username = ref('')
const password = ref('')
const loading = ref(false)

onMounted(() => {
  themeStore.init()
})

async function handleLogin() {
  if (!username.value || !password.value) {
    ElMessage.warning('请输入用户名和密码')
    return
  }

  loading.value = true
  const res = await authStore.login(username.value, password.value)
  loading.value = false

  if (res.success) {
    ElMessage.success('登录成功')
    router.push({ name: 'Dashboard' })
  } else {
    ElMessage.error(res.message || '登录失败')
  }
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center p-6 relative overflow-hidden"
    :class="themeStore.isDark ? 'bg-[var(--bg-primary)]' : 'bg-gray-50'">

    <!-- 背景装饰 -->
    <div class="absolute inset-0 overflow-hidden pointer-events-none">
      <div class="absolute -top-40 -right-40 w-80 h-80 rounded-full bg-indigo-500/20 blur-3xl"></div>
      <div class="absolute -bottom-40 -left-40 w-80 h-80 rounded-full bg-purple-500/20 blur-3xl"></div>
    </div>

    <!-- 返回按钮 -->
    <button @click="router.push('/')" class="absolute top-6 left-6 p-3 rounded-xl border transition-all z-10"
      :class="themeStore.isDark ? 'bg-[var(--bg-secondary)] border-[var(--border)] text-gray-400 hover:text-white' : 'bg-white border-gray-200 text-gray-600 hover:text-gray-900'">
      <ArrowLeft class="w-5 h-5" />
    </button>

    <!-- 主题切换 -->
    <button @click="themeStore.toggle" class="absolute top-6 right-6 p-3 rounded-xl border transition-all z-10"
      :class="themeStore.isDark ? 'bg-[var(--bg-secondary)] border-[var(--border)] text-gray-400 hover:text-white' : 'bg-white border-gray-200 text-gray-600 hover:text-gray-900'">
      <Sun v-if="themeStore.isDark" class="w-5 h-5" />
      <Moon v-else class="w-5 h-5" />
    </button>

    <div class="w-full max-w-md relative z-10">
      <!-- Logo -->
      <div class="text-center mb-8">
        <img src="/imgbed.webp" alt="Logo"
          class="w-16 h-16 rounded-2xl mx-auto mb-4 shadow-xl shadow-indigo-500/30 object-contain" />
        <h1 class="text-3xl font-bold">
          <span class="text-gradient">Img</span><span :class="themeStore.isDark ? 'text-white' : 'text-gray-800'">Bed</span>
        </h1>
        <p class="text-sm mt-2" :class="themeStore.isDark ? 'text-gray-400' : 'text-gray-500'">
          管理后台
        </p>
      </div>

      <!-- 登录卡片 -->
      <div class="rounded-2xl border p-8 shadow-xl"
        :class="themeStore.isDark ? 'bg-[var(--bg-secondary)]/80 backdrop-blur-xl border-[var(--border)]' : 'bg-white border-gray-200'">

        <form @submit.prevent="handleLogin" class="space-y-4">
          <div>
            <label class="block text-sm font-medium mb-2"
              :class="themeStore.isDark ? 'text-gray-300' : 'text-gray-700'">用户名</label>
            <div class="relative">
              <Lock class="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5"
                :class="themeStore.isDark ? 'text-gray-500' : 'text-gray-400'" />
              <input v-model="username" type="text" placeholder="请输入用户名"
                class="w-full pl-12 pr-4 py-3 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50"
                :class="themeStore.isDark ? 'bg-[var(--bg-hover)] border-[var(--border)] text-white placeholder-gray-500' : 'bg-gray-50 border-gray-200 text-gray-800'" />
            </div>
          </div>

          <div>
            <label class="block text-sm font-medium mb-2" :class="themeStore.isDark ? 'text-gray-300' : 'text-gray-700'">密码</label>
            <div class="relative">
              <Lock class="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5"
                :class="themeStore.isDark ? 'text-gray-500' : 'text-gray-400'" />
              <input v-model="password" type="password" placeholder="请输入密码"
                class="w-full pl-12 pr-4 py-3 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50"
                :class="themeStore.isDark ? 'bg-[var(--bg-hover)] border-[var(--border)] text-white placeholder-gray-500' : 'bg-gray-50 border-gray-200 text-gray-800'"
                @keyup.enter="handleLogin" />
            </div>
          </div>

          <button type="submit" :disabled="loading"
            class="w-full py-3 rounded-xl font-medium text-white bg-gradient-to-r from-indigo-500 to-purple-500 hover:from-indigo-600 hover:to-purple-600 transition-all shadow-lg shadow-indigo-500/25 disabled:opacity-50 disabled:cursor-not-allowed">
            <span v-if="loading">登录中...</span>
            <span v-else>登 录</span>
          </button>
        </form>

        <div class="mt-6 text-center text-sm" :class="themeStore.isDark ? 'text-gray-500' : 'text-gray-400'">
          默认账号: admin / admin
        </div>
      </div>
    </div>
  </div>
</template>
