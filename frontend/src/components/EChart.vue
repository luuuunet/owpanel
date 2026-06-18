<script setup lang="ts">
import * as echarts from 'echarts'
import { nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useThemeStore } from '@/stores/theme'

const props = defineProps<{
  option: echarts.EChartsOption
  height?: string
}>()

const chartEl = ref<HTMLElement>()
let chart: echarts.ECharts | null = null
let resizeObserver: ResizeObserver | null = null
const themeStore = useThemeStore()

function render() {
  if (!chart) return
  chart.setOption(props.option, { notMerge: true, lazyUpdate: true })
}

onMounted(() => {
  if (!chartEl.value) return
  chart = echarts.init(chartEl.value)
  render()

  resizeObserver = new ResizeObserver(() => chart?.resize())
  resizeObserver.observe(chartEl.value)
})

watch(() => props.option, render, { deep: true })

watch(
  () => [themeStore.resolvedTheme, themeStore.darkVariant] as const,
  () => {
    nextTick(() => {
      chart?.resize()
      render()
    })
  },
)

onUnmounted(() => {
  resizeObserver?.disconnect()
  chart?.dispose()
  chart = null
})
</script>

<template>
  <div ref="chartEl" class="echart" :style="height ? { height } : undefined" />
</template>

<style scoped>
.echart {
  width: 100%;
  height: 280px;
}
</style>
