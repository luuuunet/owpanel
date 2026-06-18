import { computed } from 'vue'
import { cfTheme } from '@/config/theme'
import { useThemeStore } from '@/stores/theme'

function readCssVar(name: string, fallback: string) {
  if (typeof document === 'undefined') return fallback
  const value = getComputedStyle(document.documentElement).getPropertyValue(name).trim()
  return value || fallback
}

export function useChartTheme() {
  const themeStore = useThemeStore()

  const isDark = computed(() => themeStore.resolvedTheme === 'dark')
  const themeKey = computed(() => `${themeStore.resolvedTheme}-${themeStore.darkVariant}`)

  const colors = computed(() => {
    const dark = isDark.value
    const chart = readCssVar('--cf-chart', dark ? '#0070f3' : cfTheme.orange)
    return {
      text: readCssVar('--cf-text', dark ? '#ffffff' : '#1f2937'),
      textSecondary: readCssVar('--cf-text-muted', dark ? '#999999' : '#64748b'),
      surface: readCssVar('--cf-surface', dark ? '#161616' : '#ffffff'),
      border: readCssVar('--cf-border', dark ? '#222222' : '#e2e8f0'),
      gaugeTrack: dark ? readCssVar('--cf-navy-light', '#222222') : '#eef1f5',
      pieFree: dark ? readCssVar('--cf-navy-light', '#222222') : '#eef1f5',
      splitLine: dark ? 'rgba(255, 255, 255, 0.06)' : '#f1f5f9',
      axisLabel: readCssVar('--cf-text-muted', dark ? '#666666' : '#909399'),
      tooltipBg: dark ? 'rgba(22, 22, 22, 0.98)' : 'rgba(255, 255, 255, 0.96)',
      tooltipBorder: readCssVar('--cf-border', dark ? '#222222' : '#e2e8f0'),
      chart,
      chartSecondary: readCssVar('--cf-chart-secondary', dark ? '#4a9eed' : cfTheme.link),
      orange: cfTheme.orange,
      success: cfTheme.success,
      warning: cfTheme.warning,
      danger: cfTheme.danger,
      link: dark ? readCssVar('--cf-link', '#0070f3') : cfTheme.link,
    }
  })

  function axisStyle() {
    const c = colors.value
    return {
      axisLine: { lineStyle: { color: c.border } },
      axisLabel: { color: c.axisLabel, fontSize: 10 },
      splitLine: { lineStyle: { color: c.splitLine } },
    }
  }

  function titleStyle(text: string, fontSize = 13) {
    return {
      text,
      left: 12,
      top: 6,
      textStyle: { fontSize, fontWeight: 600 as const, color: colors.value.text },
    }
  }

  function tooltipStyle() {
    const c = colors.value
    return {
      trigger: 'axis' as const,
      backgroundColor: c.tooltipBg,
      borderColor: c.tooltipBorder,
      textStyle: { color: c.text, fontSize: 12 },
    }
  }

  return { isDark, themeKey, colors, axisStyle, titleStyle, tooltipStyle }
}
