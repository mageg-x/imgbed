<script setup>
import { onErrorCaptured, ref, onMounted, computed } from 'vue'
import { RouterView } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { useThemeStore } from '@/stores/theme'
import { availableLocales, setLocale } from '@/i18n'

const { t, locale } = useI18n()
const themeStore = useThemeStore()
const error = ref(null)

const currentLocaleName = computed(() => {
  return availableLocales.find(l => l.code === locale.value)?.name || 'English'
})

onMounted(() => {
  themeStore.init()
})

onErrorCaptured((err, instance, info) => {
  console.error('Global error captured:', err, info)
  error.value = err.message || 'An unexpected error occurred'

  if (!err.message?.includes('cancel')) {
    ElMessage.error(t('app.pleaseRefresh'))
  }

  return false
})

function handleRetry() {
  error.value = null
  window.location.reload()
}

function handleLocaleChange(lang) {
  setLocale(lang)
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
      <h2 class="text-2xl font-bold text-gray-900 dark:text-white mb-2">{{ t('app.errorOccurred') }}</h2>
      <p class="text-gray-500 dark:text-gray-400 mb-6">{{ error }}</p>
      <button @click="handleRetry"
        class="px-6 py-3 bg-indigo-500 hover:bg-indigo-600 text-white rounded-xl font-medium transition-all">
        {{ t('app.refreshPage') }}
      </button>
    </div>
  </div>
  <RouterView v-else />
</template>
