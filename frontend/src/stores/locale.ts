import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { defineStore } from 'pinia'
import type { LocaleCode } from '@/locales'
import { htmlLang, saveLocale } from '@/locales'
import i18n from '@/locales'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import zhTw from 'element-plus/es/locale/lang/zh-tw'
import en from 'element-plus/es/locale/lang/en'

export const useLocaleStore = defineStore('locale', () => {
  const locale = ref<LocaleCode>(i18n.global.locale.value as LocaleCode)

  function setLocale(code: LocaleCode) {
    locale.value = code
    i18n.global.locale.value = code
    saveLocale(code)
    document.documentElement.lang = htmlLang(code)
  }

  function elementLocale() {
    if (locale.value === 'zh-CN') return zhCn
    if (locale.value === 'zh-TW') return zhTw
    return en
  }

  watch(
    locale,
    () => {
      document.documentElement.lang = htmlLang(locale.value)
    },
    { immediate: true },
  )

  return { locale, setLocale, elementLocale }
})

export function useLocale() {
  const store = useLocaleStore()
  const { t } = useI18n()
  return { t, locale: store.locale, setLocale: store.setLocale }
}
