<script setup lang="ts">
import * as echarts from 'echarts'
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { isChineseLocale } from '@/locales'
import { ElMessage } from 'element-plus'
import api from '@/api'
import { panelStaticPath } from '@/utils/panelBase'
import { usePerformanceProfile } from '@/composables/usePerformanceProfile'
import TrafficGeoDrawer, { type CountryInfo } from '@/components/TrafficGeoDrawer.vue'

const props = withDefaults(defineProps<{
  compact?: boolean
  dashboard?: boolean
  side?: boolean
  hours?: number
}>(), {
  compact: false,
  dashboard: false,
  side: false,
  hours: 24,
})

const { t, locale } = useI18n()
const { load: loadPerf, trafficMapSec } = usePerformanceProfile()
const effectiveMapSec = computed(() => (props.dashboard ? 30 : trafficMapSec.value))

const chartEl = ref<HTMLElement>()
const mapCanvasWrap = ref<HTMLElement>()
const mapCanvasHeight = ref(360)
const mapReady = ref(false)
const loading = ref(true)
const installingGeo = ref(false)
const autoGeoAttempted = ref(false)
const data = ref<any>(null)
const range = ref(props.hours)
const geoDrawerVisible = ref(false)
const selectedCountry = ref<CountryInfo | null>(null)
let chart: echarts.ECharts | null = null
let timer: ReturnType<typeof setInterval>
let resizeObserver: ResizeObserver | null = null

function countryLabel(c: any) {
  return isChineseLocale(locale.value) ? (c.zh || c.name) : c.name
}

function formatNum(n: number) {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M'
  if (n >= 1_000) return (n / 1_000).toFixed(1) + 'K'
  return String(n)
}

function openCountryDrawer(c: CountryInfo) {
  selectedCountry.value = c
  geoDrawerVisible.value = true
}

function findCountryByMapName(name: string): CountryInfo | null {
  const countries = data.value?.countries || []
  const hit = countries.find((c: any) => c.map_name === name || c.name === name || c.zh === name)
  if (!hit) return null
  return {
    code: hit.code,
    name: hit.name,
    zh: hit.zh,
    count: hit.count,
    bytes: hit.bytes,
    percent: hit.percent,
  }
}

function onMapClick(params: any) {
  if (params?.seriesType !== 'map' || !params?.name) return
  const c = findCountryByMapName(params.name)
  if (c) openCountryDrawer(c)
}

function isSelectedCountry(c: any) {
  return selectedCountry.value?.code === c.code && geoDrawerVisible.value
}

function formatBytes(bytes: number) {
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`
}

const isEmpty = computed(() => !data.value?.total_requests)

async function loadMap() {
  const res = await fetch(panelStaticPath('/geo/world.json'))
  const json = await res.json()
  echarts.registerMap('world', json)
  mapReady.value = true
}

async function loadData() {
  loading.value = true
  try {
    const params: Record<string, number | string> = { hours: range.value }
    if (props.dashboard) params.live = 1
    const res: any = await api.get('/analytics/traffic-map', { params })
    data.value = res.data
    if (!data.value?.total_requests && !data.value?.geo_db_ready && !installingGeo.value && !autoGeoAttempted.value) {
      autoGeoAttempted.value = true
      await installGeoIP()
    }
  } finally {
    loading.value = false
  }
}

async function installGeoIP() {
  installingGeo.value = true
  try {
    await api.post('/analytics/geoip/install')
    ElMessage.success(t('traffic.installGeoSuccess'))
    await loadData()
  } catch (e: any) {
    ElMessage.error(e?.error || e?.message || t('traffic.installGeoFailed'))
  } finally {
    installingGeo.value = false
  }
}

const mapOption = computed(() => {
  if (!data.value || !mapReady.value) return null

  const countries = data.value.countries || []
  const maxVal = Math.max(...countries.map((c: any) => c.count), 1)

  const mapData = countries.map((c: any) => ({
    name: c.map_name,
    value: c.count,
    code: c.code,
    bytes: c.bytes,
    percent: c.percent,
  }))

  return {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'item',
      formatter: (p: any) => {
        if (p.seriesType !== 'map') return p.name
        const v = p.value || 0
        const extra = p.data?.percent != null ? `<br/>${t('traffic.share')}: ${p.data.percent}%` : ''
        const bw = p.data?.bytes ? `<br/>${t('traffic.bandwidth')}: ${formatBytes(p.data.bytes)}` : ''
        return `<b>${p.name}</b><br/>${t('traffic.pageViews')}: ${formatNum(v)}${extra}${bw}`
      },
    },
    visualMap: {
      type: 'continuous',
      min: 0,
      max: maxVal,
      orient: props.dashboard ? 'horizontal' : 'vertical',
      left: props.compact ? 8 : 16,
      bottom: props.dashboard ? 8 : (props.compact ? 12 : 24),
      calculable: false,
      inRange: {
        color: ['#eef6ff', '#bfdbfe', '#60a5fa', '#2563eb', '#1d4ed8', '#1e3a8a'],
      },
      text: [t('traffic.highPv'), t('traffic.lowPv')],
      textStyle: { color: '#64748b', fontSize: 11 },
      itemWidth: props.dashboard ? 88 : 12,
      itemHeight: props.dashboard ? 10 : 80,
    },
    geo: {
      map: 'world',
      roam: true,
      scaleLimit: { min: 1, max: 6 },
      zoom: props.dashboard ? 1.42 : (props.compact ? 1.18 : 1.12),
      center: props.dashboard ? [10, 18] : [20, 24],
      layoutCenter: ['50%', '50%'],
      layoutSize: props.dashboard ? '118%' : '108%',
      aspectScale: 0.82,
      itemStyle: {
        areaColor: '#f1f5f9',
        borderColor: '#cbd5e1',
        borderWidth: 0.5,
      },
      emphasis: {
        itemStyle: { areaColor: '#dbeafe' },
        label: { show: false },
      },
      label: { show: false },
    },
    series: [
      {
        name: t('traffic.pageViews'),
        type: 'map',
        map: 'world',
        geoIndex: 0,
        data: mapData,
        emphasis: {
          label: { show: true, color: '#1e293b', fontSize: 11 },
        },
      },
    ],
  } as echarts.EChartsOption
})

function render() {
  if (!chart || !mapOption.value) return
  chart.setOption(mapOption.value, { notMerge: false, lazyUpdate: true })
}

function syncMapCanvasHeight() {
  if (!props.dashboard || !mapCanvasWrap.value) return
  const wrap = mapCanvasWrap.value
  const hint = 22
  const minH = 340
  const maxH = 520
  const h = Math.min(maxH, Math.max(minH, wrap.clientHeight - hint))
  if (h > 0 && h !== mapCanvasHeight.value) {
    mapCanvasHeight.value = h
    chart?.resize()
  }
}

const mapCanvasStyle = computed(() => {
  if (props.side) return { height: '220px' }
  if (props.dashboard) return { height: `${mapCanvasHeight.value}px` }
  if (props.compact) return { height: '320px' }
  return { height: '480px' }
})

function startMapTimer() {
  clearInterval(timer)
  timer = setInterval(loadData, effectiveMapSec.value * 1000)
}

onMounted(async () => {
  await loadPerf(true)
  await loadMap()
  await loadData()
  if (!chartEl.value) return
  chart = echarts.init(chartEl.value)
  chart.on('click', onMapClick)
  render()
  resizeObserver = new ResizeObserver(() => {
    syncMapCanvasHeight()
    chart?.resize()
  })
  resizeObserver.observe(chartEl.value)
  if (mapCanvasWrap.value) resizeObserver.observe(mapCanvasWrap.value)
  syncMapCanvasHeight()
  startMapTimer()
})

watch(effectiveMapSec, () => startMapTimer())

watch(mapOption, () => {
  render()
  syncMapCanvasHeight()
}, { deep: true })
watch(range, loadData)

onUnmounted(() => {
  clearInterval(timer)
  resizeObserver?.disconnect()
  chart?.dispose()
})
</script>

<template>
  <div class="traffic-map" :class="{ compact, dashboard, side }">
    <div class="map-toolbar">
      <div class="stats-row">
        <div class="stat-item">
          <span class="stat-label">{{ t('traffic.pageViews') }}</span>
          <span class="stat-value">{{ formatNum(data?.total_requests || 0) }}</span>
        </div>
        <div class="stat-item">
          <span class="stat-label">{{ t('traffic.uniqueVisitors') }}</span>
          <span class="stat-value">{{ formatNum(data?.unique_ips || 0) }}</span>
        </div>
        <div class="stat-item">
          <span class="stat-label">{{ t('traffic.bandwidth') }}</span>
          <span class="stat-value">{{ formatBytes(data?.total_bytes || 0) }}</span>
        </div>
        <el-tag v-if="data?.geo_db_ready" type="success" size="small">GeoIP2</el-tag>
        <template v-else>
          <el-tag type="warning" size="small">{{ t('traffic.noGeoDb') }}</el-tag>
          <el-button type="primary" size="small" :loading="installingGeo" @click="installGeoIP">
            {{ t('traffic.installGeo') }}
          </el-button>
        </template>
      </div>
      <el-radio-group v-if="!compact" v-model="range" size="small">
        <el-radio-button :value="24">{{ t('traffic.h24') }}</el-radio-button>
        <el-radio-button :value="168">{{ t('traffic.d7') }}</el-radio-button>
        <el-radio-button :value="720">{{ t('traffic.d30') }}</el-radio-button>
      </el-radio-group>
    </div>

    <div class="map-body">
      <div ref="mapCanvasWrap" class="map-canvas-wrap">
        <div v-if="loading && !data" class="map-loading"><el-skeleton :rows="6" animated /></div>
        <div v-if="!loading && isEmpty" class="map-empty">
          <el-empty :description="t('traffic.emptyHint')">
            <template #default>
              <p class="empty-detail">{{ t('traffic.emptyDetail') }}</p>
              <ul v-if="data?.log_paths?.length" class="log-list">
                <li v-for="p in data.log_paths.slice(0, 4)" :key="p">{{ p }}</li>
              </ul>
              <div class="empty-actions">
                <el-button v-if="!data?.geo_db_ready" type="primary" :loading="installingGeo" @click="installGeoIP">
                  {{ t('traffic.installGeo') }}
                </el-button>
                <el-button @click="loadData">{{ t('common.refresh') }}</el-button>
              </div>
            </template>
          </el-empty>
        </div>
        <div ref="chartEl" class="map-canvas" :style="mapCanvasStyle" />
        <p v-if="!isEmpty && mapReady" class="map-interact-hint">{{ t('traffic.mapInteractHint') }}</p>
      </div>

      <div v-if="!compact && !isEmpty" class="country-panel">
        <div class="panel-title">{{ t('traffic.topCountries') }}</div>
        <div v-for="(c, i) in (data?.countries || []).slice(0, 15)" :key="c.code" class="country-row clickable" :class="{ selected: isSelectedCountry(c) }" @click="openCountryDrawer(c)">
          <div class="country-head">
            <span class="rank">{{ i + 1 }}</span>
            <span class="name">{{ countryLabel(c) }}</span>
            <span class="count">{{ formatNum(c.count) }}</span>
          </div>
          <el-progress
            :percentage="c.percent"
            :show-text="false"
            :stroke-width="6"
            color="#2563eb"
          />
          <span class="percent">{{ c.percent }}%</span>
        </div>
      </div>

      <div v-if="compact && !isEmpty && dashboard" class="dashboard-countries">
        <div class="panel-title">{{ t('traffic.topCountries') }}</div>
        <div v-for="(c, i) in (data?.countries || []).slice(0, 8)" :key="c.code" class="country-row clickable" :class="{ selected: isSelectedCountry(c) }" @click="openCountryDrawer(c)">
          <div class="country-head">
            <span class="rank">{{ i + 1 }}</span>
            <span class="name">{{ countryLabel(c) }}</span>
            <span class="count">{{ formatNum(c.count) }}</span>
          </div>
          <el-progress
            :percentage="c.percent"
            :show-text="false"
            :stroke-width="5"
            color="#2563eb"
          />
        </div>
      </div>

      <div v-else-if="compact && !isEmpty" class="compact-countries">
        <div v-for="(c, i) in (data?.countries || []).slice(0, 5)" :key="c.code" class="compact-row clickable" :class="{ selected: isSelectedCountry(c) }" @click="openCountryDrawer(c)">
          <span class="rank">{{ i + 1 }}</span>
          <span class="name">{{ countryLabel(c) }}</span>
          <span class="count">{{ formatNum(c.count) }}</span>
        </div>
      </div>
    </div>

    <TrafficGeoDrawer
      v-model:visible="geoDrawerVisible"
      :country="selectedCountry"
      :hours="range"
    />
  </div>
</template>

<style scoped>
.traffic-map {
  background: var(--el-bg-color);
  border-radius: 8px;
  padding: 12px 16px 16px;
}
.map-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  margin-bottom: 12px;
}
.stats-row {
  display: flex;
  align-items: center;
  gap: 20px;
  flex-wrap: wrap;
}
.stat-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.stat-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.stat-value {
  font-size: 20px;
  font-weight: 700;
  color: var(--el-text-color-primary);
  line-height: 1.2;
}
.map-body {
  display: flex;
  gap: 16px;
  align-items: stretch;
}
.map-canvas-wrap {
  flex: 1;
  min-width: 0;
  position: relative;
}
.map-canvas { width: 100%; }
.map-interact-hint {
  margin: 6px 0 0;
  font-size: 11px;
  color: var(--el-text-color-secondary);
  text-align: center;
}
.map-loading {
  position: absolute;
  inset: 0;
  z-index: 2;
  padding: 24px;
  background: rgba(255, 255, 255, 0.85);
}
.map-empty {
  position: absolute;
  inset: 0;
  z-index: 2;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.92);
}
.empty-detail {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  margin: 8px 0;
  max-width: 360px;
  text-align: center;
}
.empty-actions {
  display: flex;
  gap: 8px;
  justify-content: center;
  margin-top: 12px;
}
.log-list {
  text-align: left;
  font-size: 12px;
  color: var(--el-text-color-placeholder);
  margin: 0;
  padding-left: 18px;
}
.country-panel {
  width: 260px;
  flex-shrink: 0;
  max-height: 480px;
  overflow-y: auto;
  padding: 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-fill-color-blank);
}
.panel-title {
  font-weight: 600;
  margin-bottom: 12px;
  font-size: 14px;
}
.country-row { margin-bottom: 14px; }
.country-row.clickable,
.compact-row.clickable {
  cursor: pointer;
  border-radius: 6px;
  padding: 4px 6px;
  margin-left: -6px;
  margin-right: -6px;
  transition: background 0.15s;
}
.country-row.clickable:hover,
.compact-row.clickable:hover {
  background: var(--el-fill-color-light);
}
.country-row.selected,
.compact-row.selected {
  background: #eff6ff;
  outline: 1px solid #bfdbfe;
}
.country-head {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 4px;
}
.country-row .rank {
  color: var(--el-text-color-placeholder);
  font-size: 12px;
  width: 18px;
}
.country-row .name {
  font-size: 13px;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.country-row .count {
  font-size: 13px;
  color: #2563eb;
  font-weight: 600;
}
.country-row .percent {
  font-size: 11px;
  color: var(--el-text-color-secondary);
  float: right;
  margin-top: 2px;
}
.traffic-map.compact { padding: 8px 12px; }
.traffic-map.compact .map-body { flex-direction: column; }
.traffic-map.compact.dashboard {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  padding: 10px 12px 12px;
}
.traffic-map.compact.dashboard .map-toolbar {
  flex-shrink: 0;
  margin-bottom: 10px;
}
.traffic-map.compact.dashboard .map-body {
  flex: 1;
  min-height: 0;
  flex-direction: row;
  align-items: stretch;
  gap: 12px;
}
.traffic-map.compact.dashboard .map-canvas-wrap {
  flex: 1;
  min-width: 0;
  min-height: 340px;
  display: flex;
  flex-direction: column;
}
.traffic-map.compact.dashboard .map-canvas {
  flex: 1;
  min-height: 340px;
}
.traffic-map.compact.dashboard .dashboard-countries {
  width: 188px;
  flex-shrink: 0;
  max-height: none;
  align-self: stretch;
  overflow-y: auto;
  padding: 8px 10px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-fill-color-blank);
}
.traffic-map.compact.dashboard .dashboard-countries .panel-title {
  font-weight: 600;
  margin-bottom: 8px;
  font-size: 13px;
}
.traffic-map.compact.dashboard .dashboard-countries .country-row {
  margin-bottom: 10px;
}
.traffic-map.compact.dashboard .dashboard-countries .country-head {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-bottom: 3px;
}
.traffic-map.compact.dashboard .dashboard-countries .rank {
  color: var(--el-text-color-placeholder);
  font-size: 11px;
  width: 16px;
}
.traffic-map.compact.dashboard .dashboard-countries .name {
  font-size: 12px;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.traffic-map.compact.dashboard .dashboard-countries .count {
  font-size: 12px;
  color: #2563eb;
  font-weight: 600;
}
.traffic-map.compact.dashboard.side .dashboard-countries {
  width: 132px;
  max-height: 220px;
  padding: 6px 8px;
}
.traffic-map.compact.dashboard.side .dashboard-countries .country-row {
  margin-bottom: 8px;
}
.traffic-map.compact.dashboard.side .map-toolbar .stats-row {
  gap: 10px;
}
.traffic-map.compact.dashboard.side .stat-item .stat-value {
  font-size: 15px;
}
@media (max-width: 768px) {
  .traffic-map.compact.dashboard .map-body {
    flex-direction: column;
  }
  .traffic-map.compact.dashboard .dashboard-countries {
    width: 100%;
    max-height: none;
  }
}
.compact-countries {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 8px;
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.compact-row {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
}
.compact-row .rank { color: var(--el-text-color-placeholder); width: 16px; }
.compact-row .name { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.compact-row .count { color: #2563eb; font-weight: 600; }
</style>
