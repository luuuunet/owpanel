import { computed, ref, watch } from 'vue'
import { defineStore } from 'pinia'
import {
  applyThemeToDOM,
  DEFAULT_THEME_PREFS,
  loadThemePrefs,
  resolveTheme,
  saveThemePrefs,
  type DarkThemeVariant,
  type ThemeMode,
  type ThemePrefs,
} from '@/config/themes'

function systemPrefersDark() {
  return window.matchMedia('(prefers-color-scheme: dark)').matches
}

export function bootstrapTheme() {
  const prefs = loadThemePrefs()
  applyThemeToDOM(resolveTheme(prefs.mode, systemPrefersDark()), prefs.darkVariant)
}

export const useThemeStore = defineStore('theme', () => {
  const prefs = loadThemePrefs()
  const mode = ref<ThemeMode>(prefs.mode)
  const darkVariant = ref<DarkThemeVariant>(prefs.darkVariant)
  const systemDark = ref(systemPrefersDark())

  const resolvedTheme = computed(() => resolveTheme(mode.value, systemDark.value))

  function apply() {
    applyThemeToDOM(resolvedTheme.value, darkVariant.value)
    saveThemePrefs({ mode: mode.value, darkVariant: darkVariant.value })
  }

  function setMode(next: ThemeMode) {
    mode.value = next
  }

  function setDarkVariant(next: DarkThemeVariant) {
    darkVariant.value = next
  }

  function updatePrefs(next: Partial<ThemePrefs>) {
    if (next.mode) mode.value = next.mode
    if (next.darkVariant) darkVariant.value = next.darkVariant
  }

  function cycleMode() {
    const order: ThemeMode[] = ['light', 'dark', 'system']
    const idx = order.indexOf(mode.value)
    mode.value = order[(idx + 1) % order.length]
  }

  const media = window.matchMedia('(prefers-color-scheme: dark)')
  media.addEventListener('change', (e) => {
    systemDark.value = e.matches
  })

  watch([mode, darkVariant, systemDark], apply, { immediate: true })

  return {
    mode,
    darkVariant,
    resolvedTheme,
    setMode,
    setDarkVariant,
    updatePrefs,
    cycleMode,
    resetTheme: () => {
      mode.value = DEFAULT_THEME_PREFS.mode
      darkVariant.value = DEFAULT_THEME_PREFS.darkVariant
    },
  }
})
