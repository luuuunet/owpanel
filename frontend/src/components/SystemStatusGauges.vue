<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { EChartsOption } from 'echarts'
import { Cpu, Odometer, Coin, FolderOpened, MagicStick, Connection, Sort } from '@element-plus/icons-vue'
import EChart from '@/components/EChart.vue'
import HealthScoreCard from '@/components/HealthScoreCard.vue'
import { useChartTheme } from '@/composables/useChartTheme'

const props = withDefaults(defineProps<{
  stats: {
    cpu?: { usage_percent?: number; cores?: number }
    memory?: { used?: number; total?: number; used_percent?: number }
    load?: { load1?: number; load5?: number; load15?: number }
    swap?: { used?: number; total?: number; used_percent?: number }
    network?: { upload_rate?: number; download_rate?: number }
    disk?: { mount: string; used: number; total: number; used_percent?: number }[]
  } | null
  freeingMemory?: boolean
  optimizing?: boolean
  embedded?: boolean
  compactRow?: boolean
  sidebar?: boolean
  dashboard?: boolean
  showHead?: boolean
  hideHealthScore?: boolean
}>(), { embedded: false, compactRow: false, sidebar: false, dashboard: false, showHead: true, hideHealthScore: false })

const emit = defineEmits<{ freeMemory: []; optimize: [] }>()

const { t } = useI18n()
const { colors, themeKey } = useChartTheme()

function gaugeColor(pct: number) {
  if (pct < 50) return colors.value.success
  if (pct < 80) return colors.value.warning
  return colors.value.danger
}

function buildGauge(value: number, color: string, max = 100): EChartsOption {
  const v = Math.min(max, Math.max(0, value))
  const c = colors.value
  const compact = props.compactRow
  return {
    backgroundColor: 'transparent',
    series: [{
      type: 'gauge' as const,
      startAngle: 210,
      endAngle: -30,
      min: 0,
      max,
      radius: compact ? '88%' : '92%',
      center: ['50%', compact ? '62%' : '58%'],
      progress: {
        show: true,
        width: compact ? 8 : 10,
        roundCap: true,
        itemStyle: { color },
      },
      axisLine: {
        lineStyle: { width: compact ? 8 : 10, color: [[1, c.gaugeTrack]] },
        roundCap: true,
      },
      axisTick: { show: false },
      splitLine: { show: false },
      axisLabel: { show: false },
      pointer: { show: false },
      title: { show: false },
      detail: {
        valueAnimation: true,
        fontSize: compact ? 16 : 22,
        fontWeight: 700,
        color: c.text,
        offsetCenter: [0, compact ? '10%' : '8%'],
        formatter: max === 100 ? '{value}%' : '{value}',
      },
      data: [{ value: Math.round(v * 10) / 10 }],
    }],
  }
}

const loadPercent = computed(() => {
  const load1 = props.stats?.load?.load1 ?? 0
  const cores = props.stats?.cpu?.cores || 1
  return Math.min(100, (load1 / cores) * 100)
})

const cpuPercent = computed(() => props.stats?.cpu?.usage_percent ?? 0)
const memPercent = computed(() => props.stats?.memory?.used_percent ?? 0)

const primaryDisk = computed(() => {
  const disks = props.stats?.disk || []
  if (!disks.length) return null
  return disks.find(d => d.mount === '/' || d.mount === 'C:' || d.mount === 'C:\\') || disks[0]
})

const diskPercent = computed(() => primaryDisk.value?.used_percent ?? 0)
const swapPercent = computed(() => props.stats?.swap?.used_percent ?? 0)

const networkUp = computed(() => props.stats?.network?.upload_rate ?? 0)
const networkDown = computed(() => props.stats?.network?.download_rate ?? 0)

const networkActivity = computed(() => {
  const total = networkUp.value + networkDown.value
  const cap = 100 * 1024 * 1024
  return Math.min(100, (total / cap) * 100)
})

function formatRate(bps: number) {
  if (!bps || bps < 0) return '0 B/s'
  const units = ['B/s', 'KB/s', 'MB/s', 'GB/s']
  const i = Math.min(Math.floor(Math.log(bps) / Math.log(1024)), units.length - 1)
  return `${(bps / Math.pow(1024, i)).toFixed(i > 0 ? 1 : 0)} ${units[i]}`
}

function formatMountLabel(mount: string) {
  const m = (mount || '').trim()
  if (m === '/' || m === '\\') return t('dashboard.rootDisk')
  if (/^C:\\?$/i.test(m)) return t('dashboard.systemDisk')
  return m || t('dashboard.rootDisk')
}

function loadDetail() {
  const l = props.stats?.load
  if (!l) return ''
  return t('dashboard.loadDetail', {
    l1: (l.load1 ?? 0).toFixed(2),
    l5: (l.load5 ?? 0).toFixed(2),
    l15: (l.load15 ?? 0).toFixed(2),
  })
}

function loadHint(pct: number) {
  if (pct < 30) return t('dashboard.loadSmooth')
  if (pct < 70) return t('dashboard.loadModerate')
  return t('dashboard.loadHeavy')
}

const cards = computed(() => {
  if (!props.stats) return []
  themeKey.value
  const items = [
    {
      key: 'load',
      icon: Odometer,
      label: t('dashboard.loadStatus'),
      labelTip: loadDetail(),
      sub: loadHint(loadPercent.value),
      option: buildGauge(loadPercent.value, gaugeColor(loadPercent.value)),
    },
    {
      key: 'cpu',
      icon: Cpu,
      label: t('dashboard.cpuUsage'),
      labelTip: '',
      sub: t('dashboard.cores', { n: props.stats.cpu?.cores ?? 0 }),
      option: buildGauge(cpuPercent.value, gaugeColor(cpuPercent.value)),
    },
    {
      key: 'memory',
      icon: Coin,
      label: t('dashboard.memoryUsage'),
      labelTip: '',
      sub: `${formatGB(props.stats.memory?.used ?? 0)} / ${formatGB(props.stats.memory?.total ?? 0)}`,
      option: buildGauge(memPercent.value, gaugeColor(memPercent.value)),
    },
    {
      key: 'network',
      icon: Connection,
      label: t('dashboard.networkRealtime'),
      labelTip: '',
      sub: `↑ ${formatRate(networkUp.value)} · ↓ ${formatRate(networkDown.value)}`,
      option: buildGauge(networkActivity.value, gaugeColor(Math.max(networkActivity.value, 5))),
    },
  ]
  if ((props.stats.swap?.total ?? 0) > 0) {
    items.push({
      key: 'swap',
      icon: Sort,
      label: t('dashboard.swapUsage'),
      labelTip: '',
      sub: `${formatGB(props.stats.swap?.used ?? 0)} / ${formatGB(props.stats.swap?.total ?? 0)}`,
      option: buildGauge(swapPercent.value, gaugeColor(swapPercent.value)),
    })
  }
  if (primaryDisk.value) {
    items.push({
      key: 'disk',
      icon: FolderOpened,
      label: formatMountLabel(primaryDisk.value.mount),
      labelTip: primaryDisk.value.mount,
      sub: `${formatGB(primaryDisk.value.used)} / ${formatGB(primaryDisk.value.total)}`,
      option: buildGauge(diskPercent.value, gaugeColor(diskPercent.value)),
    })
  }
  return items
})

function formatGB(bytes: number) {
  if (!bytes) return '0 GB'
  return `${(bytes / 1024 / 1024 / 1024).toFixed(1)} GB`
}
</script>

<template>
  <div v-if="stats" class="status-gauges" :class="{ embedded, 'compact-row': compactRow, sidebar, dashboard }">
    <div v-if="showHead" class="status-head">
      <div class="status-head-left">
        <span class="status-title">{{ t('dashboard.statusTitle') }}</span>
        <span class="status-tip">{{ t('dashboard.loadStatusTip') }}</span>
      </div>
      <el-button type="primary" size="small" :loading="optimizing" @click="emit('optimize')">
        <el-icon class="btn-icon"><MagicStick /></el-icon>
        {{ t('dashboard.optimizeOneClick') }}
      </el-button>
    </div>
    <div class="gauge-grid">
      <HealthScoreCard v-if="!hideHealthScore" embedded :compact="compactRow || sidebar || dashboard" />
      <div v-for="card in cards" :key="card.key" class="gauge-card">
        <div v-if="!compactRow" class="gauge-icon" :class="card.key">
          <el-icon :size="20"><component :is="card.icon" /></el-icon>
        </div>
        <el-tooltip v-if="card.labelTip" :content="card.labelTip" placement="top">
          <div class="gauge-label">{{ card.label }}</div>
        </el-tooltip>
        <div v-else class="gauge-label">{{ card.label }}</div>
        <EChart :option="card.option" :height="compactRow ? (dashboard ? '80px' : (sidebar ? '72px' : '88px')) : '132px'" />
        <div class="gauge-sub">{{ card.sub }}</div>
        <el-button
          v-if="card.key === 'memory'"
          class="free-mem-btn"
          type="primary"
          link
          size="small"
          :loading="freeingMemory"
          @click="emit('freeMemory')"
        >
          {{ t('dashboard.freeMemory') }}
        </el-button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.status-gauges {
  border: 1px solid var(--cf-border);
  border-radius: 12px;
  padding: 14px 16px 8px;
  margin-bottom: 14px;
  background: var(--cf-surface);
  box-shadow: var(--apple-shadow-sm, var(--cf-shadow));
}
.status-gauges.embedded {
  border: none;
  border-radius: 0;
  padding: 0;
  margin: 0 0 10px;
  background: transparent;
  box-shadow: none;
}
.status-gauges.compact-row .gauge-grid {
  display: flex;
  flex-wrap: nowrap;
  gap: 8px;
  max-width: none;
  margin: 0;
  overflow-x: auto;
  padding-bottom: 4px;
  scrollbar-width: thin;
}
.status-gauges.compact-row .gauge-card,
.status-gauges.compact-row :deep(.health-gauge-card) {
  flex: 1 1 0;
  min-width: 108px;
  padding: 6px 4px 4px;
}
.status-gauges.compact-row .gauge-label {
  font-size: 11px;
}
.status-gauges.compact-row .gauge-sub {
  font-size: 10px;
  min-height: 14px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.status-gauges.compact-row .free-mem-btn,
.status-gauges.compact-row :deep(.health-link) {
  font-size: 11px;
}
.status-gauges.compact-row :deep(.health-progress) {
  width: 72px !important;
  height: 72px !important;
}
.status-gauges.compact-row :deep(.health-progress .el-progress-circle) {
  width: 72px !important;
  height: 72px !important;
}
.status-gauges.compact-row :deep(.health-gauge-card .gauge-sub) {
  min-height: 14px;
  -webkit-line-clamp: 1;
}
.status-gauges.compact-row.sidebar .gauge-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  overflow-x: visible;
  gap: 6px;
}
.status-gauges.compact-row.sidebar .gauge-card,
.status-gauges.compact-row.sidebar :deep(.health-gauge-card) {
  flex: unset;
  min-width: 0;
  padding: 4px 2px 2px;
}
.status-gauges.compact-row.sidebar .gauge-sub {
  font-size: 9px;
}
.status-gauges.compact-row.sidebar :deep(.health-gauge-card .gauge-sub),
.status-gauges.compact-row.sidebar :deep(.health-link),
.status-gauges.compact-row.sidebar .free-mem-btn {
  display: none;
}
.status-gauges.compact-row.sidebar :deep(.health-progress),
.status-gauges.compact-row.sidebar :deep(.health-progress .el-progress-circle) {
  width: 56px !important;
  height: 56px !important;
}
.status-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 8px;
  flex-wrap: wrap;
}
.status-head-left {
  display: flex;
  align-items: baseline;
  gap: 10px;
  flex-wrap: wrap;
  min-width: 0;
}
.btn-icon { margin-right: 4px; }
.status-title { font-size: 15px; font-weight: 600; color: var(--el-text-color-primary); }
.status-tip { font-size: 12px; color: var(--el-text-color-secondary); }
.gauge-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 12px;
  max-width: 1280px;
  margin: 0 auto;
}
.gauge-card {
  position: relative;
  border: 1px solid var(--cf-border);
  border-radius: 10px;
  padding: 10px 8px 6px;
  background: var(--el-fill-color-lighter);
  text-align: center;
  transition: box-shadow 0.15s, border-color 0.15s;
}
.gauge-card:hover { box-shadow: var(--apple-shadow-sm, 0 4px 12px rgba(0, 0, 0, 0.12)); }
.gauge-icon {
  width: 36px; height: 36px; border-radius: 10px;
  display: inline-flex; align-items: center; justify-content: center;
  margin-bottom: 4px;
}
.gauge-icon.load { background: rgba(0, 81, 195, 0.14); color: var(--cf-link, #0051c3); }
.gauge-icon.cpu { background: rgba(246, 130, 31, 0.14); color: var(--cf-orange, #f6821f); }
.gauge-icon.memory { background: rgba(5, 150, 105, 0.14); color: var(--el-color-success, #059669); }
.gauge-icon.network { background: rgba(14, 165, 233, 0.14); color: #0ea5e9; }
.gauge-icon.swap { background: rgba(168, 85, 247, 0.14); color: #a855f7; }
.gauge-icon.disk { background: rgba(98, 106, 239, 0.14); color: #626aef; }
.gauge-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--el-text-color-regular);
  margin-bottom: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  padding: 0 4px;
}
.gauge-sub {
  margin-top: -4px;
  font-size: 11px;
  color: var(--el-text-color-secondary);
  min-height: 16px;
}
.free-mem-btn {
  margin-top: 2px;
  font-size: 12px;
}
@media (max-width: 900px) {
  .gauge-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    max-width: 480px;
    margin: 0 auto;
  }
}
@media (max-width: 560px) {
  .gauge-grid {
    grid-template-columns: 1fr;
    max-width: 240px;
  }
}
</style>
