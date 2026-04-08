import { createI18n } from 'vue-i18n'
import zhCN from './zh-CN'
import enUS from './en-US'

function getDefaultLocale() {
  const stored = localStorage.getItem('locale')
  if (stored) return stored
  const lang = navigator.language.toLowerCase()
  if (lang.includes('zh')) return 'zh-CN'
  return 'en-US'
}

const i18n = createI18n({
  legacy: false,
  locale: getDefaultLocale(),
  fallbackLocale: 'en-US',
  messages: {
    'zh-CN': zhCN,
    'en-US': enUS
  }
})

export const availableLocales = [
  { code: 'zh-CN', name: '简体中文' },
  { code: 'en-US', name: 'English' }
]

export function setLocale(locale) {
  i18n.global.locale.value = locale
  localStorage.setItem('locale', locale)
  document.documentElement.lang = locale
}

export default i18n
