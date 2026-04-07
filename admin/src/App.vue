<script setup>
import { onErrorCaptured, ref, onMounted } from 'vue'
import { RouterView } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useThemeStore } from '@/stores/theme'

const themeStore = useThemeStore()
const error = ref(null)

onMounted(() => {
  themeStore.init()
})

onErrorCaptured((err, instance, info) => {
  console.error('Global error captured:', err, info)
  error.value = err.message || 'An unexpected error occurred'
  
  if (!err.message?.includes('cancel')) {
    ElMessage.error('发生了一些错误，请刷新页面重试')
  }
  
  return false
})

function handleRetry() {
  error.value = null
  window.location.reload()
}
</script>

<template>
  <div v-if="error" class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
    <div class="text-center p-8">
      <div class="w-20 h-20 mx-auto mb-6 rounded-full bg-red-100 dark:bg-red-900/30 flex items-center justify-center">
        <svg class="w-10 h-10 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
        </svg>
      </div>
      <h2 class="text-2xl font-bold text-gray-900 dark:text-white mb-2">出现了一些问题</h2>
      <p class="text-gray-500 dark:text-gray-400 mb-6">{{ error }}</p>
      <button @click="handleRetry"
        class="px-6 py-3 bg-indigo-500 hover:bg-indigo-600 text-white rounded-xl font-medium transition-all">
        刷新页面
      </button>
    </div>
  </div>
  <RouterView v-else />
</template>
