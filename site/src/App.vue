<script setup>
import { ref, onErrorCaptured, onMounted } from 'vue'
import { RouterView, useRouter } from 'vue-router'
import { useThemeStore } from '@/stores/theme'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const themeStore = useThemeStore()
const authStore = useAuthStore()
const error = ref(null)

onErrorCaptured((err, instance, info) => {
  console.error('Global error captured:', err, info)
  error.value = err.message || 'An unexpected error occurred'
  return false
})

onMounted(async () => {
  themeStore.init()
  await authStore.checkSession()
})

function handleRetry() {
  error.value = null
  window.location.reload()
}
</script>

<template>
  <div v-if="error" class="min-h-screen flex items-center justify-center"
    :class="themeStore.isDark ? 'bg-[var(--bg-primary)]' : 'bg-gray-50'">
    <div class="text-center p-8">
      <div class="w-20 h-20 mx-auto mb-6 rounded-full flex items-center justify-center"
        :class="themeStore.isDark ? 'bg-red-900/30' : 'bg-red-100'">
        <svg class="w-10 h-10 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
            d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
        </svg>
      </div>
      <h2 class="text-2xl font-bold mb-2" :class="themeStore.isDark ? 'text-white' : 'text-gray-900'">出现了一些问题</h2>
      <p class="mb-6" :class="themeStore.isDark ? 'text-gray-400' : 'text-gray-500'">{{ error }}</p>
      <button @click="handleRetry"
        class="px-6 py-3 bg-gradient-to-r from-indigo-500 to-purple-500 hover:from-indigo-600 hover:to-purple-600 text-white rounded-xl font-medium transition-all shadow-lg shadow-indigo-500/25">
        刷新页面
      </button>
    </div>
  </div>
  <RouterView v-else v-slot="{ Component, route }">
    <transition name="page" mode="out-in">
      <component :is="Component" :key="route.path" />
    </transition>
  </RouterView>
</template>

<style>
.page-enter-active,
.page-leave-active {
  transition: opacity 0.25s ease, transform 0.25s ease;
}

.page-enter-from {
  opacity: 0;
  transform: translateY(10px);
}

.page-leave-to {
  opacity: 0;
  transform: translateY(-10px);
}
</style>
