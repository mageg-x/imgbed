<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useThemeStore } from '@/stores/theme'
import { ElMessage } from 'element-plus'
import { Lock, Sun, Moon, ArrowLeft, LogOut, Globe } from 'lucide-vue-next'
import { availableLocales, setLocale } from '@/i18n'

const { t, locale } = useI18n()
const router = useRouter()
const authStore = useAuthStore()
const themeStore = useThemeStore()
const isLangDropdownOpen = ref(false)

const username = ref('')
const password = ref('')
const loading = ref(false)
const checkLoading = ref(true)

// 已登录但非 admin
const isLoggedInAsUser = computed(() => {
  return checkLoading.value === false && authStore.isAuthenticated && authStore.user?.role !== 'admin'
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
  if (!username.value || !password.value) {
    ElMessage.warning(t('login.pleaseInputUsername'))
    return
  }

  loading.value = true
  const res = await authStore.login(username.value, password.value)
  loading.value = false

  if (res.success) {
    ElMessage.success(t('login.loginSuccess'))
    router.push({ name: 'Dashboard' })
  } else {
    ElMessage.error(res.message || t('login.loginFailed'))
  }
}

function goHome() {
  window.location.href = '/'
}

function handleLocaleChange(lang) {
  setLocale(lang)
  isLangDropdownOpen.value = false
}

function closeLangDropdown() {
  isLangDropdownOpen.value = false
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
    <button @click="goHome()" class="absolute top-6 left-6 p-3 rounded-xl border transition-all z-10"
      :class="themeStore.isDark ? 'bg-[var(--bg-secondary)] border-[var(--border)] text-gray-400 hover:text-white' : 'bg-white border-gray-200 text-gray-600 hover:text-gray-900'">
      <ArrowLeft class="w-5 h-5" />
    </button>

    <!-- 主题切换 -->
    <button @click="themeStore.toggle" class="absolute top-6 right-6 p-3 rounded-xl border transition-all z-10"
      :class="themeStore.isDark ? 'bg-[var(--bg-secondary)] border-[var(--border)] text-gray-400 hover:text-white' : 'bg-white border-gray-200 text-gray-600 hover:text-gray-900'">
      <Sun v-if="themeStore.isDark" class="w-5 h-5" />
      <Moon v-else class="w-5 h-5" />
    </button>

    <!-- 语言切换下拉菜单 -->
    <div class="absolute top-6 right-20 z-10">
      <div class="relative">
        <button @click="isLangDropdownOpen = !isLangDropdownOpen"
          class="p-3 rounded-xl border transition-all"
          :class="themeStore.isDark ? 'bg-[var(--bg-secondary)] border-[var(--border)] text-gray-400 hover:text-white' : 'bg-white border-gray-200 text-gray-600 hover:text-gray-900'">
          <Globe class="w-5 h-5" />
        </button>

        <transition name="fade">
          <div v-if="isLangDropdownOpen"
            class="absolute right-0 mt-2 w-36 rounded-xl border shadow-xl overflow-hidden z-50"
            :class="themeStore.isDark ? 'bg-[var(--bg-secondary)] border-[var(--border)]' : 'bg-white border-gray-200'">
            <div @click="closeLangDropdown" class="fixed inset-0"></div>
            <div class="relative">
              <button v-for="lang in availableLocales" :key="lang.code"
                @click="handleLocaleChange(lang.code)"
                class="w-full px-4 py-2.5 text-left text-sm flex items-center justify-between transition-all"
                :class="locale === lang.code
                  ? (themeStore.isDark ? 'bg-indigo-500/20 text-indigo-400' : 'bg-indigo-50 text-indigo-600')
                  : (themeStore.isDark ? 'text-gray-300 hover:bg-white/5' : 'text-gray-700 hover:bg-gray-50')">
                <span>{{ lang.name }}</span>
                <span v-if="locale === lang.code" class="w-2 h-2 rounded-full bg-indigo-500"></span>
              </button>
            </div>
          </div>
        </transition>
      </div>
    </div>

    <div class="w-full max-w-md relative z-10">
      <!-- Logo -->
      <div class="text-center mb-8">
        <img src="/imgbed.webp" alt="Logo"
          class="w-16 h-16 rounded-2xl mx-auto mb-4 shadow-xl shadow-indigo-500/30 object-contain" />
        <h1 class="text-3xl font-bold">
          <span class="text-gradient">Img</span><span
            :class="themeStore.isDark ? 'text-white' : 'text-gray-800'">Bed</span>
        </h1>
        <p class="text-sm mt-2" :class="themeStore.isDark ? 'text-gray-400' : 'text-gray-500'">
          {{ t('login.title') }}
        </p>
      </div>

      <!-- 登录卡片 -->
      <div class="rounded-2xl border p-8 shadow-xl"
        :class="themeStore.isDark ? 'bg-[var(--bg-secondary)]/80 backdrop-blur-xl border-[var(--border)]' : 'bg-white border-gray-200'">

        <!-- 非 admin 用户已登录，显示无权限提示 -->
        <div v-if="isLoggedInAsUser" class="text-center py-8">
          <div
            class="w-16 h-16 rounded-full bg-red-100 dark:bg-red-900/30 mx-auto mb-4 flex items-center justify-center">
            <Lock class="w-8 h-8 text-red-500" />
          </div>
          <h3 class="text-lg font-medium mb-2" :class="themeStore.isDark ? 'text-white' : 'text-gray-800'">
            {{ t('login.noPermission') || 'No Permission' }}
          </h3>
          <p class="text-sm mb-6" :class="themeStore.isDark ? 'text-gray-400' : 'text-gray-500'">
            {{ t('login.currentUser', { username: authStore.user?.username || 'user' }) }}
          </p>
          <button @click="handleLogout"
            class="w-full py-3 rounded-xl font-medium text-white bg-gradient-to-r from-indigo-500 to-purple-500 hover:from-indigo-600 hover:to-purple-600 transition-all shadow-lg shadow-indigo-500/25 flex items-center justify-center gap-2">
            <LogOut class="w-4 h-4" />
            {{ t('login.logoutAndReLogin') }}
          </button>
        </div>

        <!-- 正常登录表单 -->
        <form v-else @submit.prevent="handleLogin" class="space-y-4">
          <div>
            <label class="block text-sm font-medium mb-2"
              :class="themeStore.isDark ? 'text-gray-300' : 'text-gray-700'">{{ t('login.username') }}</label>
            <div class="relative">
              <Lock class="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5"
                :class="themeStore.isDark ? 'text-gray-500' : 'text-gray-400'" />
              <input v-model="username" type="text" :placeholder="t('login.username')"
                class="w-full pl-12 pr-4 py-3 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50"
                :class="themeStore.isDark ? 'bg-[var(--bg-hover)] border-[var(--border)] text-white placeholder-gray-500' : 'bg-gray-50 border-gray-200 text-gray-800'" />
            </div>
          </div>

          <div>
            <label class="block text-sm font-medium mb-2"
              :class="themeStore.isDark ? 'text-gray-300' : 'text-gray-700'">{{ t('login.password') }}</label>
            <div class="relative">
              <Lock class="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5"
                :class="themeStore.isDark ? 'text-gray-500' : 'text-gray-400'" />
              <input v-model="password" type="password" :placeholder="t('login.password')"
                class="w-full pl-12 pr-4 py-3 rounded-xl border transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500/50"
                :class="themeStore.isDark ? 'bg-[var(--bg-hover)] border-[var(--border)] text-white placeholder-gray-500' : 'bg-gray-50 border-gray-200 text-gray-800'"
                @keyup.enter="handleLogin" />
            </div>
          </div>

          <button type="submit" :disabled="loading"
            class="w-full py-3 rounded-xl font-medium text-white bg-gradient-to-r from-indigo-500 to-purple-500 hover:from-indigo-600 hover:to-purple-600 transition-all shadow-lg shadow-indigo-500/25 disabled:opacity-50 disabled:cursor-not-allowed">
            <span v-if="loading">{{ t('common.loading') }}</span>
            <span v-else>{{ t('login.loginButton') }}</span>
          </button>
        </form>

        <div class="mt-6 text-center text-sm" :class="themeStore.isDark ? 'text-gray-500' : 'text-gray-400'">
          {{ t('login.defaultAccount') }}: admin / admin
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.15s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
