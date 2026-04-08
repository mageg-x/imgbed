<script setup>
import { ref, onMounted, computed, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useThemeStore } from '@/stores/theme'
import { ElMessage } from 'element-plus'
import { Lock, Sun, Moon, Image, LogOut } from 'lucide-vue-next'

const { t } = useI18n()
const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const themeStore = useThemeStore()

const mode = ref('user')
const username = ref('')
const password = ref('')
const loading = ref(false)
const checkLoading = ref(true)

// 已登录但非 user（admin 登录了主站）
const isLoggedInAsAdmin = computed(() => {
  return checkLoading.value === false && authStore.isAuthenticated && authStore.user?.role === 'admin'
})

onMounted(async () => {
  themeStore.init()
  // 强制检查 session，确保 user 数据已加载
  if (authStore.isAuthenticated) {
    await authStore.checkSession()
  }
  checkLoading.value = false
})

async function handleLogout() {
  await authStore.logout()
  window.location.reload()
}

async function handleLogin() {
  if (mode.value === 'user') {
    if (!password.value) {
      ElMessage.warning(t('login.pleaseInputAccessPassword'))
      return
    }
    loading.value = true
    const res = await authStore.login(password.value)
    loading.value = false
    if (res.success) {
      ElMessage.success(t('login.loginSuccess'))
      router.push(route.query.redirect || '/')
    } else {
      ElMessage.error(res.message || t('login.loginFailed'))
    }
  } else {
    if (!username.value || !password.value) {
      ElMessage.warning(t('login.pleaseInputUsernameAndPassword'))
      return
    }
    loading.value = true
    const res = await authStore.adminLogin({ username: username.value, password: password.value })
    loading.value = false
    if (res.success) {
      ElMessage.success(t('login.adminLoginSuccess'))
      window.location.href = '/admin/'
    } else {
      ElMessage.error(res.message || t('login.loginFailed'))
    }
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

    <!-- 主题切换 -->
    <el-tooltip :content="themeStore.isDark ? t('common.switchToLightMode') : t('common.switchToDarkMode')" placement="bottom">
      <button @click="themeStore.toggle"
        class="absolute top-4 right-4 sm:top-6 sm:right-6 p-2.5 sm:p-3 rounded-xl border transition-all z-10"
        :class="themeStore.isDark ? 'bg-[var(--bg-secondary)] border-[var(--border)] text-gray-400 hover:text-white' : 'bg-white border-gray-200 text-gray-600 hover:text-gray-900'">
        <Sun v-if="themeStore.isDark" class="w-5 h-5" />
        <Moon v-else class="w-5 h-5" />
      </button>
    </el-tooltip>

    <div class="w-full max-w-sm sm:max-w-md relative z-10 px-4">
      <!-- Logo -->
      <div class="text-center mb-6 sm:mb-8">
        <img src="/imgbed.webp" alt="ImgBed"
          class="w-14 h-14 sm:w-16 sm:h-16 rounded-2xl object-cover mx-auto mb-3 sm:mb-4 shadow-xl shadow-indigo-500/30" />
        <h1 class="text-2xl sm:text-3xl font-bold">
          <span class="text-gradient">Img</span><span
            :class="themeStore.isDark ? 'text-white' : 'text-gray-800'">Bed</span>
        </h1>
        <p class="text-xs sm:text-sm mt-1.5 sm:mt-2" :class="themeStore.isDark ? 'text-gray-400' : 'text-gray-500'">
          {{ t('login.subtitle') }}
        </p>
      </div>

      <!-- 登录卡片 -->
      <div class="rounded-xl sm:rounded-2xl border p-6 sm:p-8 shadow-xl"
        :class="themeStore.isDark ? 'bg-[var(--bg-secondary)]/80 backdrop-blur-xl border-[var(--border)]' : 'bg-white border-gray-200'">

        <!-- admin 已登录，显示无权限提示 -->
        <div v-if="isLoggedInAsAdmin" class="text-center py-8">
          <div
            class="w-16 h-16 rounded-full bg-red-100 dark:bg-red-900/30 mx-auto mb-4 flex items-center justify-center">
            <Lock class="w-8 h-8 text-red-500" />
          </div>
          <h3 class="text-lg font-medium mb-2" :class="themeStore.isDark ? 'text-white' : 'text-gray-800'">
            {{ t('login.accessDenied') }}
          </h3>
          <p class="text-sm mb-6" :class="themeStore.isDark ? 'text-gray-400' : 'text-gray-500'">
            {{ t('login.adminLoggedInTip') }}
          </p>
          <button @click="handleLogout"
            class="w-full py-3 rounded-xl font-medium text-white bg-gradient-to-r from-indigo-500 to-purple-500 hover:from-indigo-600 hover:to-purple-600 transition-all shadow-lg shadow-indigo-500/25 flex items-center justify-center gap-2">
            <LogOut class="w-4 h-4" />
            {{ t('login.logoutAndReLogin') }}
          </button>
        </div>

        <!-- 正常登录表单 -->
        <template v-else>
          <!-- 模式切换 -->
          <div class="flex rounded-xl p-1 mb-4 sm:mb-6"
            :class="themeStore.isDark ? 'bg-[var(--bg-hover)]' : 'bg-gray-100'">
            <button @click="mode = 'user'" class="flex-1 py-2 rounded-lg text-sm font-medium transition-all" :class="mode === 'user'
              ? 'bg-gradient-to-r from-indigo-500 to-purple-500 text-white shadow-lg'
              : (themeStore.isDark ? 'text-gray-400' : 'text-gray-600')">
              <span class="hidden sm:inline">{{ t('login.userLogin') }}</span>
              <span class="sm:hidden">{{ t('login.user') }}</span>
            </button>
            <button @click="mode = 'admin'" class="flex-1 py-2 rounded-lg text-sm font-medium transition-all" :class="mode === 'admin'
              ? 'bg-gradient-to-r from-indigo-500 to-purple-500 text-white shadow-lg'
              : (themeStore.isDark ? 'text-gray-400' : 'text-gray-600')">
              <span class="hidden sm:inline">{{ t('login.adminLogin') }}</span>
              <span class="sm:hidden">{{ t('login.admin') }}</span>
            </button>
          </div>

          <form @submit.prevent="handleLogin" class="space-y-4">
            <div v-if="mode === 'admin'">
              <label class="block text-sm font-medium mb-2"
                :class="themeStore.isDark ? 'text-gray-300' : 'text-gray-700'">{{ t('login.username') }}</label>
              <div class="relative">
                <Lock class="absolute left-3 sm:left-4 top-1/2 -translate-y-1/2 w-4 h-4 sm:w-5 sm:h-5"
                  :class="themeStore.isDark ? 'text-gray-500' : 'text-gray-400'" />
                <input v-model="username" type="text" :placeholder="t('login.pleaseInputUsername')"
                  class="w-full pl-10 sm:pl-12 pr-4 py-2.5 sm:py-3 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50"
                  :class="themeStore.isDark ? 'bg-[var(--bg-hover)] border-[var(--border)] text-white placeholder-gray-500' : 'bg-gray-50 border-gray-200 text-gray-800'" />
              </div>
            </div>

            <div>
              <label class="block text-sm font-medium mb-2"
                :class="themeStore.isDark ? 'text-gray-300' : 'text-gray-700'">
                {{ mode === 'admin' ? t('login.password') : t('login.accessPassword') }}
              </label>
              <div class="relative">
                <Lock class="absolute left-3 sm:left-4 top-1/2 -translate-y-1/2 w-4 h-4 sm:w-5 sm:h-5"
                  :class="themeStore.isDark ? 'text-gray-500' : 'text-gray-400'" />
                <input v-model="password" type="password" :placeholder="mode === 'admin' ? t('login.pleaseInputPassword') : t('login.pleaseInputAccessPassword')"
                  class="w-full pl-10 sm:pl-12 pr-4 py-2.5 sm:py-3 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50"
                  :class="themeStore.isDark ? 'bg-[var(--bg-hover)] border-[var(--border)] text-white placeholder-gray-500' : 'bg-gray-50 border-gray-200 text-gray-800'" />
              </div>
            </div>

            <button type="submit" :disabled="loading"
              class="w-full py-2.5 sm:py-3 rounded-xl font-medium text-white bg-gradient-to-r from-indigo-500 to-purple-500 hover:from-indigo-600 hover:to-purple-600 transition-all shadow-lg shadow-indigo-500/25 disabled:opacity-50 disabled:cursor-not-allowed">
              <span v-if="loading">{{ t('login.loggingIn') }}...</span>
              <span v-else>{{ t('login.login') }}</span>
            </button>
          </form>
        </template>

        <div class="mt-4 sm:mt-6 text-center">
          <router-link to="/" class="text-xs sm:text-sm transition-all"
            :class="themeStore.isDark ? 'text-gray-400 hover:text-indigo-400' : 'text-gray-500 hover:text-indigo-600'">
            {{ t('common.backToHome') }}
          </router-link>
        </div>
      </div>
    </div>
  </div>
</template>
