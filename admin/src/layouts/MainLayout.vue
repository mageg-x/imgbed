<script setup>
import { ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useThemeStore } from '@/stores/theme'
import {
  Home, Folder, Network, Key, Settings, Sun, Moon,
  Menu, X, PanelLeftClose, Maximize2, LogOut, Code
} from 'lucide-vue-next'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const themeStore = useThemeStore()
const isCollapsed = ref(false)
const isMobileMenuOpen = ref(false)

const menuItems = [
  { path: '/', label: '仪表盘', icon: Home },
  { path: '/files', label: '文件管理', icon: Folder },
  { path: '/channels', label: '渠道管理', icon: Network },
  { path: '/tokens', label: 'API Token', icon: Key },
  { path: '/integration', label: '集成示例', icon: Code },
  { path: '/settings', label: '系统设置', icon: Settings }
]

const activeNav = computed(() => route.path)

function handleLogout() {
  authStore.logout()
  router.push('/login')
}

function closeMobileMenu() {
  isMobileMenuOpen.value = false
}
</script>

<template>
  <div class="min-h-screen flex" :class="themeStore.isDark ? 'bg-[var(--bg-primary)]' : 'bg-gray-50'">
    <!-- 移动端 hamburger -->
    <button @click="isMobileMenuOpen = !isMobileMenuOpen" class="fixed top-4 left-4 z-50 p-2 rounded-lg lg:hidden"
      :class="themeStore.isDark ? 'bg-[var(--bg-secondary)] text-white' : 'bg-white text-gray-900'">
      <Menu v-if="!isMobileMenuOpen" class="w-6 h-6" />
      <X v-else class="w-6 h-6" />
    </button>

    <!-- 移动端遮罩 -->
    <div v-if="isMobileMenuOpen" @click="closeMobileMenu" class="fixed inset-0 bg-black/50 z-40 lg:hidden"></div>

    <!-- 侧边栏 -->
    <aside class="fixed left-0 top-0 h-full flex flex-col transition-all duration-300 z-50" :class="[
      isCollapsed ? 'w-[72px]' : 'w-64',
      isMobileMenuOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0'
    ]" :style="{ background: themeStore.isDark ? 'var(--bg-secondary)' : 'white' }">

      <!-- Logo -->
      <div class="h-16 flex items-center gap-3 px-4 border-b pt-14 lg:pt-0" :style="{ borderColor: 'var(--border)' }">
        <img src="/imgbed.webp" alt="Logo" class="w-10 h-10 rounded-xl flex-shrink-0 object-contain" />
        <transition name="fade">
          <span v-if="!isCollapsed" class="font-bold text-lg text-gradient whitespace-nowrap">ImgBed</span>
        </transition>
      </div>

      <!-- 导航 -->
      <nav class="flex-1 p-3 space-y-1 overflow-y-auto">
        <router-link v-for="item in menuItems" :key="item.path" :to="item.path" @click="closeMobileMenu"
          class="flex items-center gap-3 px-4 py-3 rounded-xl transition-all group whitespace-nowrap"
          :class="activeNav === item.path
            ? 'bg-gradient-to-r from-indigo-500 to-purple-500 text-white shadow-lg shadow-indigo-500/25'
            : (themeStore.isDark ? 'text-gray-400 hover:text-white hover:bg-[var(--bg-hover)]' : 'text-gray-600 hover:text-gray-900 hover:bg-gray-100')">
          <component :is="item.icon" class="w-5 h-5 flex-shrink-0" />
          <transition name="fade">
            <span v-if="!isCollapsed" class="font-medium">{{ item.label }}</span>
          </transition>
        </router-link>
      </nav>

      <!-- 底部 -->
      <div class="p-3 space-y-1 border-t" :style="{ borderColor: 'var(--border)' }">
        <button @click="isCollapsed = !isCollapsed"
          class="w-full flex items-center gap-3 px-4 py-3 rounded-xl transition-all whitespace-nowrap"
          :class="themeStore.isDark ? 'text-gray-400 hover:text-white hover:bg-[var(--bg-hover)]' : 'text-gray-600 hover:text-gray-900 hover:bg-gray-100'">
          <PanelLeftClose v-if="!isCollapsed" class="w-5 h-5 flex-shrink-0" />
          <Maximize2 v-else class="w-5 h-5 flex-shrink-0" />
          <transition name="fade">
            <span v-if="!isCollapsed">收起</span>
          </transition>
        </button>
      </div>
    </aside>

    <!-- 主内容 -->
    <main class="flex-1 transition-all duration-300 w-full" :class="isCollapsed ? 'lg:ml-[72px]' : 'lg:ml-64'">

      <!-- 顶部栏 -->
      <header class="sticky top-0 z-40 border-b backdrop-blur-xl px-4 sm:px-6 py-4"
        :class="themeStore.isDark ? 'bg-[var(--bg-primary)]/80 border-[var(--border)]' : 'bg-white/80 border-gray-200'">
        <div class="flex items-center justify-between gap-4">
          <h1 class="text-xl font-semibold pl-10 lg:pl-0">{{ route.meta?.title || '管理后台' }}</h1>
          <div class="flex items-center gap-2 sm:gap-4 flex-wrap">
            <button @click="themeStore.toggle" class="p-2 rounded-lg transition-all border-0"
              :class="themeStore.isDark ? 'text-gray-400 hover:text-white hover:bg-[var(--bg-hover)]' : 'text-gray-600 hover:text-gray-900 hover:bg-gray-100'">
              <Sun v-if="themeStore.isDark" class="w-5 h-5" />
              <Moon v-else class="w-5 h-5" />
            </button>
            <span class="text-sm px-3 py-1 rounded-lg hidden sm:inline"
              :class="themeStore.isDark ? 'bg-[var(--bg-hover)] text-gray-400' : 'bg-gray-100 text-gray-600'">
              管理员
            </span>
            <button @click="handleLogout"
              class="px-2 sm:px-4 py-2 rounded-lg text-sm font-medium transition-all border-0 whitespace-nowrap"
              :class="themeStore.isDark ? 'bg-red-500/10 text-red-500 hover:bg-red-500/20' : 'bg-red-50 text-red-600 hover:bg-red-100'">
              <LogOut class="w-4 h-4 inline" />
              <span class="hidden sm:inline ml-1">退出</span>
            </button>
          </div>
        </div>
      </header>

      <!-- 内容区 -->
      <div class="p-4 sm:p-6">
        <RouterView />
      </div>
    </main>
  </div>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

.page-enter-active,
.page-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.page-enter-from {
  opacity: 0;
  transform: translateY(10px);
}

.page-leave-to {
  opacity: 0;
}
</style>
