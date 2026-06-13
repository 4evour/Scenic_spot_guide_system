import { createI18n } from 'vue-i18n'
import zhCN from './locales/zh-CN.json'
import enUS from './locales/en-US.json'

type Locale = 'zh-CN' | 'en-US'

/** 语言检测优先级：1. URL query  2. localStorage  3. 浏览器  4. 默认 zh-CN */
function detectLocale(): Locale {
  // 1. URL query: ?lang=en 或 ?lang=zh-CN
  const params = new URLSearchParams(window.location.search)
  const langParam = params.get('lang')
  if (langParam === 'en' || langParam === 'en-US') return 'en-US'
  if (langParam === 'zh' || langParam === 'zh-CN') return 'zh-CN'

  // 2. localStorage
  const stored = localStorage.getItem('locale')
  if (stored === 'en-US' || stored === 'zh-CN') return stored

  // 3. 浏览器语言
  const navLang = navigator.language
  if (navLang.startsWith('en')) return 'en-US'
  if (navLang.startsWith('zh')) return 'zh-CN'

  // 4. 默认
  return 'zh-CN'
}

/** 初始 locale */
export function initLocale(): Locale {
  const locale = detectLocale()
  localStorage.setItem('locale', locale)
  return locale
}

/** 切换语言 */
export function switchLocale(locale: Locale) {
  localStorage.setItem('locale', locale)
  i18n.global.locale.value = locale
}

/** 初始语言检测结果 */
const initialLocale = initLocale()

export const i18n = createI18n({
  legacy: false,
  locale: initialLocale,
  fallbackLocale: 'zh-CN',
  messages: {
    'zh-CN': zhCN,
    'en-US': enUS,
  },
})

export default i18n
