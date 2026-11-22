import { createI18n } from 'vue-i18n'
import zhCN from './locales/zh-CN'
import enUS from './locales/en-US'

export type SupportedLocale = 'zh-CN' | 'en-US'

const savedLocale = localStorage.getItem('act.locale')
const browserLocale: SupportedLocale = navigator.language.toLowerCase().startsWith('zh') ? 'zh-CN' : 'en-US'
const initialLocale: SupportedLocale =
  savedLocale === 'zh-CN' || savedLocale === 'en-US' ? savedLocale : browserLocale

export const i18n = createI18n({
  legacy: false,
  locale: initialLocale,
  fallbackLocale: 'en-US',
  messages: {
    'zh-CN': zhCN,
    'en-US': enUS,
  },
})

export function setLocale(locale: string) {
  const next: SupportedLocale = locale === 'en-US' ? 'en-US' : 'zh-CN'
  i18n.global.locale.value = next
  localStorage.setItem('act.locale', next)
  document.documentElement.lang = next
}

export default i18n
