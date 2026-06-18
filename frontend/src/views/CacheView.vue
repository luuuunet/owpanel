<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { EChartsOption } from 'echarts'
import api from '@/api'
import EChart from '@/components/EChart.vue'
import CacheRuleAssistant, { type RuleDraft } from '@/components/CacheRuleAssistant.vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { MagicStick } from '@element-plus/icons-vue'

const props = withDefaults(defineProps<{ embedded?: boolean }>(), { embedded: false })

const { t } = useI18n()

const activeTab = ref('performance')
const rangeHours = ref(24)
const analyticsDomain = ref('')
const loading = ref(false)
const applying = ref(false)
const preview = ref('')
const status = ref<any>({})
const analytics = ref<any>({})
const sites = ref<any[]>([])
const rules = ref<any[]>([])
const ruleDialog = ref(false)
const editingRule = ref<any>(null)
const ruleGuideOpen = ref(['guide'])
const ruleAssistantRef = ref<InstanceType<typeof CacheRuleAssistant> | null>(null)
const assistantDrawerOpen = ref(false)

const config = reactive({
  enabled: false,
  dev_mode: false,
  auto_site_enable: true,
  proxy_max_size: '5g',
  proxy_inactive: '60m',
  fastcgi_max_size: '2g',
  fastcgi_inactive: '30m',
  zone_memory: '100m',
  html_ttl_minutes: 5,
  static_ttl_hours: 168,
  bypass_cookies: 'PHPSESSID|wordpress_logged_in|session|auth_token',
  bypass_paths: '/admin|/wp-admin|/api/|/login',
  stale_enabled: true,
  honor_origin: false,
  cache_query_string: true,
})

const ruleForm = reactive({
  name: '',
  pattern: '',
  action: 'bypass',
  ttl_minutes: 0,
  website_id: 0,
  priority: 100,
  enabled: true,
})

function formatBytes(bytes: number) {
  if (!bytes || bytes <= 0) return '0 B'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`
  return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`
}

function formatNum(n: number) {
  if (!n) return '0'
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1000) return `${(n / 1000).toFixed(1)}k`
  return String(n)
}

const purgePathsDialog = ref(false)
const purgePathsSite = ref<any>(null)
const purgePathsInput = ref('')

const siteStatsMap = computed(() => {
  const map = new Map<number, any>()
  for (const row of status.value?.site_stats || []) {
    map.set(row.website_id, row)
  }
  return map
})

function siteCacheBytes(row: any) {
  const stat = siteStatsMap.value.get(row.id)
  return stat?.total_bytes || 0
}

const cacheSizeText = computed(() => {
  const bytes = status.value?.total_cache_bytes
    ?? ((status.value?.proxy_cache_bytes || 0) + (status.value?.fastcgi_cache_bytes || 0))
  return formatBytes(bytes)
})

const summary = computed(() => analytics.value?.summary || {})

const cachePresets = [
  { key: 'wordpress', labelKey: 'cache.presetWordpress' },
  { key: 'laravel', labelKey: 'cache.presetLaravel' },
  { key: 'static', labelKey: 'cache.presetStatic' },
  { key: 'ecommerce', labelKey: 'cache.presetEcommerce' },
]

function sparkOption(data: number[], color: string): EChartsOption {
  return {
    grid: { left: 0, right: 0, top: 4, bottom: 0 },
    xAxis: { type: 'category', show: false, data: data.map((_, i) => i) },
    yAxis: { type: 'value', show: false },
    series: [{
      type: 'line', data, smooth: true, symbol: 'none',
      lineStyle: { width: 2, color },
      areaStyle: { color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [
        { offset: 0, color: color + '55' }, { offset: 1, color: color + '08' },
      ]}},
    }],
  }
}

const requestsSpark = computed(() => sparkOption(analytics.value?.spark_requests || [], '#f6821f'))
const bandwidthSpark = computed(() => sparkOption(analytics.value?.spark_bandwidth || [], '#0051c3'))

const hasTraffic = computed(() => (summary.value?.total_requests || 0) > 0)

function formatChartTimeLabel(timeStr: string) {
  if (!timeStr) return ''
  const hhmm = timeStr.length >= 16 ? timeStr.slice(11, 16) : timeStr.slice(-5)
  if (rangeHours.value <= 24) return hhmm
  const md = timeStr.slice(5, 10)
  return `${md} ${hhmm}`
}

function chartAxisLabelInterval(len: number) {
  if (len <= 12) return 0
  if (len <= 24) return 1
  if (len <= 72) return 5
  return 11
}

const requestsChart = computed((): EChartsOption => {
  const series = analytics.value?.time_series || []
  const labels = series.map((p: any) => formatChartTimeLabel(p.time || ''))
  const cdnData = series.map((p: any) => p.cached_requests || 0)
  const originData = series.map((p: any) => p.origin_requests || 0)
  const peak = Math.max(0, ...cdnData, ...originData, ...(cdnData.map((v: number, i: number) => v + originData[i])))
  return {
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      formatter: (params: any) => {
        const items = Array.isArray(params) ? params : [params]
        const idx = items[0]?.dataIndex ?? 0
        const row = series[idx]
        const lines = [`<strong>${labels[idx] || ''}</strong>`]
        for (const item of items) {
          lines.push(`${item.marker}${item.seriesName}: ${formatNum(item.value || 0)}`)
        }
        if (row) {
          lines.push(`${t('cache.metricRequests')}: ${formatNum(row.requests || 0)}`)
        }
        return lines.join('<br/>')
      },
    },
    legend: { data: [t('cache.servedByCDN'), t('cache.servedByOrigin')] },
    grid: { left: 48, right: 16, top: 36, bottom: 28 },
    xAxis: {
      type: 'category',
      data: labels,
      boundaryGap: true,
      axisLabel: { interval: chartAxisLabelInterval(labels.length), rotate: labels.length > 48 ? 45 : 0, fontSize: 11 },
    },
    yAxis: {
      type: 'value',
      min: 0,
      minInterval: 1,
      max: peak > 0 ? undefined : 5,
      splitLine: { lineStyle: { type: 'dashed', opacity: 0.35 } },
    },
    series: [
      {
        name: t('cache.servedByCDN'),
        type: 'bar',
        stack: 'requests',
        data: cdnData,
        color: '#f6821f',
        barMaxWidth: 18,
        emphasis: { focus: 'series' },
      },
      {
        name: t('cache.servedByOrigin'),
        type: 'bar',
        stack: 'requests',
        data: originData,
        color: '#6b7280',
        barMaxWidth: 18,
        emphasis: { focus: 'series' },
      },
    ],
  }
})

const statusChart = computed((): EChartsOption => {
  const items = analytics.value?.status_breakdown || []
  const colors: Record<string, string> = {
    HIT: '#22c55e', MISS: '#ef4444', BYPASS: '#f59e0b', EXPIRED: '#8b5cf6',
    REVALIDATED: '#06b6d4', STALE: '#84cc16', STATIC: '#3b82f6', BROWSER: '#64748b',
    NONE: '#d1d5db', DYNAMIC: '#a855f7',
  }
  return {
    tooltip: { trigger: 'item' },
    legend: { orient: 'vertical', right: 8, top: 'center', textStyle: { fontSize: 11 } },
    series: [{
      type: 'pie', radius: ['42%', '68%'], center: ['38%', '50%'],
      data: items.map((x: any) => ({ name: statusLabel(x.status), value: x.count, itemStyle: { color: colors[x.status] || '#94a3b8' } })),
      label: { show: false },
    }],
  }
})

const contentTypeChart = computed((): EChartsOption => {
  const items = (analytics.value?.content_types || []).slice(0, 8)
  return {
    tooltip: { trigger: 'axis' },
    grid: { left: 72, right: 16, top: 8, bottom: 24 },
    xAxis: { type: 'value' },
    yAxis: { type: 'category', data: items.map((x: any) => x.name).reverse(), axisLabel: { fontSize: 11 } },
    series: [{ type: 'bar', data: items.map((x: any) => x.count).reverse(), color: '#0051c3', barMaxWidth: 18 }],
  }
})

const storageChart = computed((): EChartsOption => {
  const series = analytics.value?.storage_history || []
  return {
    tooltip: { trigger: 'axis', valueFormatter: (v) => formatBytes(Number(v)) },
    grid: { left: 56, right: 16, top: 16, bottom: 28 },
    xAxis: { type: 'category', data: series.map((p: any) => p.time?.slice(11) || ''), boundaryGap: false },
    yAxis: { type: 'value', axisLabel: { formatter: (v: number) => formatBytes(v) } },
    series: [{ type: 'line', smooth: true, areaStyle: { opacity: 0.3, color: '#0051c3' }, data: series.map((p: any) => p.bytes || 0), color: '#0051c3' }],
  }
})

const egressChart = computed((): EChartsOption => {
  const series = analytics.value?.time_series || []
  const labels = series.map((p: any) => formatChartTimeLabel(p.time || ''))
  return {
    tooltip: { trigger: 'axis', valueFormatter: (v) => formatBytes(Number(v)) },
    grid: { left: 56, right: 16, top: 16, bottom: 28 },
    xAxis: {
      type: 'category',
      data: labels,
      boundaryGap: true,
      axisLabel: { interval: chartAxisLabelInterval(labels.length), fontSize: 11 },
    },
    yAxis: { type: 'value', min: 0, axisLabel: { formatter: (v: number) => formatBytes(v) } },
    series: [{ name: t('cache.egressSaved'), type: 'bar', data: series.map((p: any) => p.egress_saved || 0), color: '#22c55e', barMaxWidth: 14 }],
  }
})

function statusLabel(s: string) {
  const key = `cache.status${s.charAt(0)}${s.slice(1).toLowerCase()}`
  const translated = t(key)
  return translated === key ? s : translated
}

async function loadAnalytics() {
  const res: any = await api.get('/cache/analytics', {
    params: { hours: rangeHours.value, domain: analyticsDomain.value || undefined },
  })
  analytics.value = res.data || {}
}

async function loadRules() {
  const res: any = await api.get('/cache/rules')
  rules.value = res.data || []
}

async function loadAll() {
  loading.value = true
  try {
    const [cfgRes, stRes, sitesRes]: any[] = await Promise.all([
      api.get('/cache/config'),
      api.get('/cache/status'),
      api.get('/cache/sites'),
    ])
    Object.assign(config, cfgRes.data || {})
    status.value = stRes.data || {}
    sites.value = sitesRes.data || []
    await Promise.all([loadRules(), loadAnalytics()])
  } finally {
    loading.value = false
  }
}

watch(rangeHours, loadAnalytics)
watch(analyticsDomain, loadAnalytics)

async function saveConfig() {
  loading.value = true
  try {
    await api.put('/cache/config', { ...config })
    ElMessage.success(t('cache.saved'))
    await loadAll()
  } catch (e: any) {
    ElMessage.error(e?.error || t('common.failed'))
  } finally {
    loading.value = false
  }
}

async function toggleDevMode(val: boolean) {
  config.dev_mode = val
  await saveConfig()
}

async function loadPreview() {
  const res: any = await api.get('/cache/preview')
  preview.value = res.data?.preview || ''
}

async function applyConfig() {
  applying.value = true
  try {
    await api.put('/cache/config', { ...config })
    const res: any = await api.post('/cache/apply')
    ElMessage.success(res.data?.message || t('cache.applied'))
    await loadAll()
    await loadPreview()
  } catch (e: any) {
    ElMessage.error(e?.error || t('common.failed'))
  } finally {
    applying.value = false
  }
}

async function purgeAll() {
  await ElMessageBox.confirm(t('cache.purgeConfirm'), t('common.confirm'), { type: 'warning' })
  const res: any = await api.post('/cache/purge')
  ElMessage.success(res.data?.message || t('cache.purged'))
  await loadAll()
}

async function purgeSite(row: any) {
  await ElMessageBox.confirm(t('cache.purgeSiteConfirm', { domain: row.domain }), t('common.confirm'), { type: 'warning' })
  const res: any = await api.post(`/cache/purge/${encodeURIComponent(row.domain)}`)
  ElMessage.success(res.data?.message || t('cache.purged'))
  await loadAll()
}

function openPurgePaths(row: any) {
  purgePathsSite.value = row
  purgePathsInput.value = ''
  purgePathsDialog.value = true
}

async function submitPurgePaths() {
  const paths = purgePathsInput.value.split('\n').map((s) => s.trim()).filter(Boolean)
  if (!purgePathsSite.value?.domain) return
  const res: any = await api.post(`/cache/purge/${encodeURIComponent(purgePathsSite.value.domain)}/paths`, { paths })
  ElMessage.success(res.data?.message || t('cache.purged'))
  purgePathsDialog.value = false
  await loadAll()
}

async function applyPreset(key: string, label: string) {
  await ElMessageBox.confirm(t('cache.presetConfirm', { name: label }), t('common.confirm'), { type: 'info' })
  applying.value = true
  try {
    const res: any = await api.post(`/cache/presets/${key}`)
    ElMessage.success(res.data?.message || t('cache.presetApplied', { name: label }))
    await loadAll()
    await loadPreview()
  } catch (e: any) {
    ElMessage.error(e?.error || t('common.failed'))
  } finally {
    applying.value = false
  }
}

async function enableAllSites() {
  const res: any = await api.post('/cache/sites/enable-all')
  ElMessage.success(t('cache.enabledSites', { n: res.data?.updated ?? 0 }))
  await loadAll()
}

async function patchSite(row: any, patch: Record<string, unknown>) {
  await api.patch(`/cache/sites/${row.id}`, patch)
  Object.assign(row, patch)
  ElMessage.success(t('cache.siteUpdated'))
}

async function toggleSite(row: any, enabled: boolean) {
  await patchSite(row, { enabled })
}

async function toggleSiteDev(row: any, dev: boolean) {
  await patchSite(row, { cache_dev_mode: dev })
}

function openRuleDialog(rule?: any) {
  editingRule.value = rule || null
  if (rule) {
    Object.assign(ruleForm, {
      name: rule.name, pattern: rule.pattern, action: rule.action || 'bypass',
      ttl_minutes: rule.ttl_minutes || 0, website_id: rule.website_id || 0,
      priority: rule.priority ?? 100, enabled: rule.enabled !== false,
    })
  } else {
    Object.assign(ruleForm, {
      name: '', pattern: '', action: 'bypass', ttl_minutes: 0, website_id: 0, priority: 100, enabled: true,
    })
  }
  ruleDialog.value = true
}

function applyRuleToForm(rule: RuleDraft) {
  Object.assign(ruleForm, {
    name: rule.name,
    pattern: rule.pattern,
    action: rule.action || 'bypass',
    ttl_minutes: rule.ttl_minutes || 0,
    website_id: rule.website_id || 0,
    priority: rule.priority ?? 100,
    enabled: rule.enabled !== false,
  })
  if (!ruleDialog.value) {
    ruleDialog.value = true
  }
}

function openAssistantDrawer() {
  assistantDrawerOpen.value = true
}

function goPageRule() {
  activeTab.value = 'rules'
  openRuleDialog()
}

async function saveRule() {
  const payload = { ...ruleForm }
  if (editingRule.value?.id) {
    await api.put(`/cache/rules/${editingRule.value.id}`, payload)
  } else {
    await api.post('/cache/rules', payload)
  }
  ElMessage.success(t('cache.ruleAutoApplied'))
  ruleDialog.value = false
  await loadRules()
  await loadPreview()
}

async function deleteRule(rule: any) {
  await ElMessageBox.confirm(t('cache.ruleDeleteConfirm'), t('common.confirm'), { type: 'warning' })
  await api.delete(`/cache/rules/${rule.id}`)
  ElMessage.success(t('cache.ruleAutoApplied'))
  await loadRules()
  await loadPreview()
}

function siteLabel(id: number) {
  if (!id) return t('cache.ruleGlobal')
  const s = sites.value.find((x) => x.id === id)
  return s?.domain || `#${id}`
}

const ruleExamples = computed(() => [
  { name: t('cache.exAdmin'), pattern: '/admin|/wp-admin', desc: t('cache.exAdminDesc'), rule: { name: t('cache.exAdmin'), pattern: '/admin|/wp-admin', action: 'bypass', ttl_minutes: 0, website_id: 0, priority: 10, enabled: true } },
  { name: t('cache.exApi'), pattern: '^/api/', desc: t('cache.exApiDesc'), rule: { name: t('cache.exApi'), pattern: '^/api/', action: 'bypass', ttl_minutes: 0, website_id: 0, priority: 20, enabled: true } },
  { name: t('cache.exLogin'), pattern: '/login|/register', desc: t('cache.exLoginDesc'), rule: { name: t('cache.exLogin'), pattern: '/login|/register', action: 'bypass', ttl_minutes: 0, website_id: 0, priority: 30, enabled: true } },
  { name: t('cache.exJson'), pattern: '\\.json$', desc: t('cache.exJsonDesc'), rule: { name: t('cache.exJson'), pattern: '\\.json$', action: 'bypass', ttl_minutes: 0, website_id: 0, priority: 50, enabled: true } },
  { name: t('cache.actionCache'), pattern: '\\.(css|js|woff2?)$', desc: t('cache.ttlMinutes'), rule: { name: t('cache.actionCache'), pattern: '\\.(css|js|woff2?)$', action: 'cache', ttl_minutes: 10080, website_id: 0, priority: 100, enabled: true } },
])

onMounted(async () => {
  await loadAll()
  await loadPreview()
})
</script>

<template>
  <div class="cache-page" v-loading="loading">
    <div class="page-header" :class="{ 'page-header--embedded': props.embedded }">
      <div v-if="!props.embedded">
        <h2>{{ t('cache.title') }}</h2>
        <p class="subtitle">{{ t('cache.subtitle') }}</p>
      </div>
      <div class="header-actions">
        <el-select v-model="analyticsDomain" clearable :placeholder="t('cache.filterDomain')" style="width: 180px; margin-right: 8px">
          <el-option value="" :label="t('cache.allDomains')" />
          <el-option v-for="s in sites" :key="s.id" :value="s.domain" :label="s.domain" />
        </el-select>
        <el-select v-model="rangeHours" style="width: 120px">
          <el-option :value="6" :label="t('cache.range6h')" />
          <el-option :value="24" :label="t('cache.range24h')" />
          <el-option :value="72" :label="t('cache.range72h')" />
          <el-option :value="168" :label="t('cache.range7d')" />
        </el-select>
        <el-button :loading="applying" type="primary" @click="applyConfig">{{ t('cache.apply') }}</el-button>
        <el-button type="danger" plain @click="purgeAll">{{ t('cache.purgeAll') }}</el-button>
      </div>
    </div>

    <el-alert v-if="status.dev_mode" type="warning" :closable="false" show-icon class="hint-alert">
      {{ t('cache.devModeActive') }}
    </el-alert>
    <el-alert v-if="config.enabled && status.nginx_include_ok === false && status.nginx_include_hint" type="info" :closable="false" show-icon class="hint-alert">
      {{ status.nginx_include_hint }}
    </el-alert>
    <el-alert v-else-if="config.enabled && status.nginx_include_ok" type="success" :closable="true" show-icon class="hint-alert">
      {{ t('cache.nginxIncludeOk') }}
    </el-alert>

    <div class="cache-layout">
      <aside class="cache-sidebar">
        <div class="metric-card">
          <div class="metric-head">
            <span class="metric-title">{{ t('cache.metricRequests') }}</span>
            <span class="metric-val">{{ formatNum(summary.total_requests || 0) }}</span>
          </div>
          <EChart :option="requestsSpark" height="56px" />
        </div>
        <div class="metric-card">
          <div class="metric-head">
            <span class="metric-title">{{ t('cache.metricBandwidth') }}</span>
            <span class="metric-val">{{ formatBytes(summary.total_bandwidth || 0) }}</span>
          </div>
          <EChart :option="bandwidthSpark" height="56px" />
        </div>
        <div class="metric-card compact">
          <div class="metric-title">{{ t('cache.hitRate') }}</div>
          <div class="metric-val lg">{{ (summary.cache_hit_rate || 0).toFixed(1) }}%</div>
        </div>
        <div class="metric-card compact">
          <div class="metric-title">{{ t('cache.egressSaved') }}</div>
          <div class="metric-val lg">{{ formatBytes(summary.egress_saved || 0) }}</div>
        </div>
        <div class="metric-card compact">
          <div class="metric-title">{{ t('cache.diskUsage') }}</div>
          <div class="metric-val lg">{{ cacheSizeText }}</div>
        </div>

        <div class="quick-actions">
          <div class="qa-title">{{ t('cache.quickActions') }}</div>
          <el-button class="qa-btn" @click="purgeAll">{{ t('cache.purgeAll') }}</el-button>
          <el-button class="qa-btn" @click="goPageRule">{{ t('cache.createPageRule') }}</el-button>
          <div class="dev-toggle">
            <span>{{ t('cache.devMode') }}</span>
            <el-switch :model-value="config.dev_mode" @change="toggleDevMode" />
          </div>
        </div>
      </aside>

      <main class="cache-main">
        <el-tabs v-model="activeTab">
          <el-tab-pane :label="t('cache.tabPerformance')" name="performance">
            <el-row :gutter="16">
              <el-col :span="24">
                <el-card shadow="never" class="chart-card">
                  <template #header>{{ t('cache.requestsSummary') }}</template>
                  <div v-if="!hasTraffic" class="chart-empty">
                    <p>{{ t('cache.noTraffic') }}</p>
                    <p class="chart-empty-hint">{{ t('cache.noTrafficChartHint') }}</p>
                  </div>
                  <EChart v-else :option="requestsChart" height="260px" />
                </el-card>
              </el-col>
              <el-col :xs="24" :md="10">
                <el-card shadow="never" class="chart-card">
                  <template #header>{{ t('cache.statusBreakdown') }}</template>
                  <EChart :option="statusChart" height="260px" />
                  <div class="status-list">
                    <div v-for="item in analytics.status_breakdown || []" :key="item.status" class="status-row">
                      <span>{{ statusLabel(item.status) }}</span>
                      <span>{{ formatNum(item.count) }}</span>
                    </div>
                  </div>
                </el-card>
              </el-col>
              <el-col :xs="24" :md="14">
                <el-card shadow="never" class="chart-card">
                  <template #header>{{ t('cache.byContentType') }}</template>
                  <EChart :option="contentTypeChart" height="260px" />
                </el-card>
              </el-col>
              <el-col :span="24">
                <el-card shadow="never" class="chart-card">
                  <template #header>{{ t('cache.topPaths') }}</template>
                  <el-table :data="analytics.top_paths || []" size="small" stripe max-height="280">
                    <el-table-column prop="name" :label="t('cache.path')" min-width="280" show-overflow-tooltip />
                    <el-table-column :label="t('cache.metricRequests')" width="100" align="right">
                      <template #default="{ row }">{{ formatNum(row.count) }}</template>
                    </el-table-column>
                    <el-table-column :label="t('cache.metricBandwidth')" width="120" align="right">
                      <template #default="{ row }">{{ formatBytes(row.bytes) }}</template>
                    </el-table-column>
                  </el-table>
                </el-card>
              </el-col>
            </el-row>
          </el-tab-pane>

          <el-tab-pane :label="t('cache.tabReserve')" name="reserve">
            <el-row :gutter="16">
              <el-col :xs="24" :sm="8">
                <el-card shadow="hover" class="reserve-stat">
                  <div class="reserve-label">{{ t('cache.currentStorage') }}</div>
                  <div class="reserve-val">{{ formatBytes(summary.current_storage || 0) }}</div>
                </el-card>
              </el-col>
              <el-col :xs="24" :sm="8">
                <el-card shadow="hover" class="reserve-stat">
                  <div class="reserve-label">{{ t('cache.requestsServed') }}</div>
                  <div class="reserve-val">{{ formatNum(summary.cached_requests || 0) }}</div>
                </el-card>
              </el-col>
              <el-col :xs="24" :sm="8">
                <el-card shadow="hover" class="reserve-stat">
                  <div class="reserve-label">{{ t('cache.egressSaved') }}</div>
                  <div class="reserve-val">{{ formatBytes(summary.egress_saved || 0) }}</div>
                </el-card>
              </el-col>
              <el-col :span="24">
                <el-card shadow="never" class="chart-card">
                  <template #header>{{ t('cache.storageOverTime') }}</template>
                  <EChart :option="storageChart" height="240px" />
                </el-card>
              </el-col>
              <el-col :span="24">
                <el-card shadow="never" class="chart-card">
                  <template #header>{{ t('cache.egressOverTime') }}</template>
                  <EChart :option="egressChart" height="240px" />
                </el-card>
              </el-col>
            </el-row>
          </el-tab-pane>

          <el-tab-pane :label="t('cache.tabSettings')" name="settings">
            <div class="preset-row">
              <span class="preset-label">{{ t('cache.applyPreset') }}</span>
              <el-button v-for="p in cachePresets" :key="p.key" size="small" @click="applyPreset(p.key, t(p.labelKey))">
                {{ t(p.labelKey) }}
              </el-button>
            </div>
            <el-form label-width="180px" class="cache-form">
              <el-form-item :label="t('cache.enabled')"><el-switch v-model="config.enabled" /></el-form-item>
              <el-form-item :label="t('cache.devMode')">
                <el-switch :model-value="config.dev_mode" @change="toggleDevMode" />
                <span class="field-hint">{{ t('cache.devModeHint') }}</span>
              </el-form-item>
              <el-form-item :label="t('cache.autoSiteEnable')">
                <el-switch v-model="config.auto_site_enable" />
                <span class="field-hint">{{ t('cache.autoSiteEnableHint') }}</span>
              </el-form-item>
              <el-form-item :label="t('cache.htmlTTL')">
                <el-input-number v-model="config.html_ttl_minutes" :min="1" :max="1440" />
                <span class="field-hint">{{ t('cache.minutes') }}</span>
              </el-form-item>
              <el-form-item :label="t('cache.staticTTL')">
                <el-input-number v-model="config.static_ttl_hours" :min="1" :max="8760" />
                <span class="field-hint">{{ t('cache.hours') }}</span>
              </el-form-item>
              <el-form-item :label="t('cache.zoneMemory')"><el-input v-model="config.zone_memory" style="width:160px" /></el-form-item>
              <el-form-item :label="t('cache.maxSize')"><el-input v-model="config.proxy_max_size" style="width:160px" /></el-form-item>
              <el-form-item :label="t('cache.inactive')"><el-input v-model="config.proxy_inactive" style="width:160px" /></el-form-item>
              <el-form-item :label="t('cache.fastcgiMaxSize')"><el-input v-model="config.fastcgi_max_size" style="width:160px" /></el-form-item>
              <el-form-item :label="t('cache.fastcgiInactive')"><el-input v-model="config.fastcgi_inactive" style="width:160px" /></el-form-item>
              <el-form-item :label="t('cache.cacheQueryString')">
                <el-switch v-model="config.cache_query_string" />
                <span class="field-hint">{{ t('cache.cacheQueryStringHint') }}</span>
              </el-form-item>
              <el-form-item :label="t('cache.honorOrigin')">
                <el-switch v-model="config.honor_origin" />
                <span class="field-hint">{{ t('cache.honorOriginHint') }}</span>
              </el-form-item>
              <el-form-item :label="t('cache.bypassCookies')"><el-input v-model="config.bypass_cookies" type="textarea" :rows="2" /></el-form-item>
              <el-form-item :label="t('cache.bypassPaths')"><el-input v-model="config.bypass_paths" type="textarea" :rows="2" /></el-form-item>
              <el-form-item :label="t('cache.staleEnabled')"><el-switch v-model="config.stale_enabled" /></el-form-item>
              <el-form-item><el-button type="primary" @click="saveConfig">{{ t('common.save') }}</el-button></el-form-item>
            </el-form>
          </el-tab-pane>

          <el-tab-pane :label="t('cache.tabRules')" name="rules">
            <el-collapse v-model="ruleGuideOpen" class="rule-guide">
              <el-collapse-item :title="t('cache.ruleGuideTitle')" name="guide">
                <p class="guide-intro">{{ t('cache.ruleGuideIntro') }}</p>
                <ol class="guide-steps">
                  <li v-for="i in 5" :key="i">{{ t(`cache.ruleGuideStep${i}`) }}</li>
                </ol>
                <el-table :data="ruleExamples" size="small" stripe class="example-table">
                  <el-table-column prop="name" :label="t('common.name')" width="140" />
                  <el-table-column prop="pattern" :label="t('cache.rulePattern')" min-width="200">
                    <template #default="{ row }"><code>{{ row.pattern }}</code></template>
                  </el-table-column>
                  <el-table-column prop="desc" :label="t('common.description')" min-width="180" />
                  <el-table-column :label="t('common.actions')" width="100">
                    <template #default="{ row }">
                      <el-button text type="primary" size="small" @click="applyRuleToForm(row.rule)">{{ t('cache.useExample') }}</el-button>
                    </template>
                  </el-table-column>
                </el-table>
              </el-collapse-item>
            </el-collapse>

            <div class="rules-main">
                <div class="sites-toolbar">
                  <el-button type="primary" @click="openRuleDialog()">{{ t('cache.addRule') }}</el-button>
                  <el-button type="primary" plain :icon="MagicStick" @click="openAssistantDrawer">{{ t('cache.openAssistant') }}</el-button>
                  <span class="toolbar-hint">{{ t('cache.ruleToolbarHint') }}</span>
                </div>
                <el-table :data="rules" stripe>
                  <el-table-column prop="name" :label="t('common.name')" min-width="140" />
                  <el-table-column prop="pattern" :label="t('cache.rulePattern')" min-width="180">
                    <template #default="{ row }"><code class="pattern-code">{{ row.pattern }}</code></template>
                  </el-table-column>
                  <el-table-column :label="t('cache.ruleScope')" width="140">
                    <template #default="{ row }">{{ siteLabel(row.website_id) }}</template>
                  </el-table-column>
                  <el-table-column prop="action" :label="t('cache.ruleAction')" width="110">
                    <template #default="{ row }">
                      {{ row.action === 'cache' ? t('cache.actionCache') : t('cache.actionBypass') }}
                      <span v-if="row.action === 'cache' && row.ttl_minutes" class="ttl-tag">{{ row.ttl_minutes }}m</span>
                    </template>
                  </el-table-column>
                  <el-table-column prop="priority" :label="t('cache.rulePriority')" width="90" />
                  <el-table-column :label="t('common.status')" width="90">
                    <template #default="{ row }">
                      <el-tag :type="row.enabled ? 'success' : 'info'" size="small">{{ row.enabled ? t('common.enabled') : t('common.disabled') }}</el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column :label="t('common.actions')" width="160">
                    <template #default="{ row }">
                      <el-button text type="primary" @click="openRuleDialog(row)">{{ t('common.edit') }}</el-button>
                      <el-button text type="danger" @click="deleteRule(row)">{{ t('common.delete') }}</el-button>
                    </template>
                  </el-table-column>
                </el-table>
              </div>
          </el-tab-pane>

          <el-tab-pane :label="t('cache.tabSites')" name="sites">
            <div class="sites-toolbar">
              <el-button type="primary" plain @click="enableAllSites">{{ t('cache.enableAllRunning') }}</el-button>
            </div>
            <el-table :data="sites" stripe>
              <el-table-column prop="domain" :label="t('cache.domain')" min-width="160" />
              <el-table-column prop="status" :label="t('common.status')" width="90" />
              <el-table-column :label="t('cache.proxyMode')" width="110">
                <template #default="{ row }">{{ row.proxy_pass ? t('cache.reverseProxy') : (row.php ? 'PHP' : t('cache.static')) }}</template>
              </el-table-column>
              <el-table-column :label="t('cache.cdnCache')" width="90">
                <template #default="{ row }"><el-switch :model-value="row.cache_enabled" @change="(v: boolean) => toggleSite(row, v)" /></template>
              </el-table-column>
              <el-table-column :label="t('cache.devMode')" width="90">
                <template #default="{ row }">
                  <el-switch :model-value="row.cache_dev_mode" :disabled="!row.cache_enabled" @change="(v: boolean) => toggleSiteDev(row, v)" />
                </template>
              </el-table-column>
              <el-table-column :label="t('cache.siteCacheSize')" width="110" align="right">
                <template #default="{ row }">{{ formatBytes(siteCacheBytes(row)) }}</template>
              </el-table-column>
              <el-table-column :label="t('common.actions')" width="200">
                <template #default="{ row }">
                  <el-button text type="warning" :disabled="!row.cache_enabled" @click="purgeSite(row)">{{ t('cache.purgeSite') }}</el-button>
                  <el-button text type="warning" :disabled="!row.cache_enabled" @click="openPurgePaths(row)">{{ t('cache.purgePaths') }}</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>

          <el-tab-pane :label="t('cache.tabPreview')" name="preview">
            <el-button class="preview-btn" @click="loadPreview">{{ t('common.refresh') }}</el-button>
            <el-input v-model="preview" type="textarea" :rows="22" readonly class="preview-box" />
          </el-tab-pane>
        </el-tabs>
      </main>
    </div>

    <el-dialog v-model="ruleDialog" :title="editingRule ? t('cache.editRule') : t('cache.addRule')" width="560px" class="rule-dialog" destroy-on-close>
      <el-form label-width="110px" class="rule-form">
        <el-form-item :label="t('common.name')"><el-input v-model="ruleForm.name" /></el-form-item>
        <el-form-item :label="t('cache.rulePattern')">
          <el-input v-model="ruleForm.pattern" placeholder="/api/|\.json$" />
          <div class="field-hint block">{{ t('cache.patternHint') }}</div>
        </el-form-item>
        <el-form-item :label="t('cache.ruleScope')">
          <el-select v-model="ruleForm.website_id" style="width:100%">
            <el-option :value="0" :label="t('cache.ruleGlobal')" />
            <el-option v-for="s in sites" :key="s.id" :value="s.id" :label="s.domain" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('cache.ruleAction')">
          <el-select v-model="ruleForm.action" style="width:100%">
            <el-option value="bypass" :label="t('cache.actionBypass')" />
            <el-option value="cache" :label="t('cache.actionCache')" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="ruleForm.action === 'cache'" :label="t('cache.ttlMinutes')">
          <el-input-number v-model="ruleForm.ttl_minutes" :min="1" :max="43200" />
          <span class="field-hint">{{ t('cache.minutes') }}</span>
        </el-form-item>
        <el-form-item :label="t('cache.rulePriority')">
          <el-input-number v-model="ruleForm.priority" :min="1" :max="9999" />
          <span class="field-hint">{{ t('cache.priorityHint') }}</span>
        </el-form-item>
        <el-form-item :label="t('common.status')"><el-switch v-model="ruleForm.enabled" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button :icon="MagicStick" @click="openAssistantDrawer">{{ t('cache.openAssistant') }}</el-button>
        <el-button @click="ruleDialog = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="saveRule">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>

    <el-drawer
      v-model="assistantDrawerOpen"
      :title="t('cache.ruleAssistant')"
      direction="rtl"
      size="400px"
      append-to-body
      class="assistant-drawer"
      :z-index="3000"
    >
      <CacheRuleAssistant
        ref="ruleAssistantRef"
        :sites="sites"
        :rules="rules"
        :config="config"
        @apply="applyRuleToForm"
      />
    </el-drawer>

    <el-dialog v-model="purgePathsDialog" :title="t('cache.purgePathsTitle')" width="480px" destroy-on-close>
      <p class="field-hint block">{{ t('cache.purgePathsHint') }}</p>
      <p v-if="purgePathsSite" class="purge-domain">{{ purgePathsSite.domain }}</p>
      <el-input v-model="purgePathsInput" type="textarea" :rows="6" :placeholder="t('cache.purgePathsPlaceholder')" />
      <template #footer>
        <el-button @click="purgePathsDialog = false">{{ t('common.cancel') }}</el-button>
        <el-button type="warning" @click="submitPurgePaths">{{ t('cache.purgePaths') }}</el-button>
      </template>
    </el-dialog>

    <el-button
      v-show="activeTab === 'rules' && !assistantDrawerOpen"
      class="assistant-fab"
      type="primary"
      circle
      :icon="MagicStick"
      :title="t('cache.openAssistant')"
      @click="openAssistantDrawer"
    />
  </div>
</template>

<style scoped>
.cache-page { width: 100%; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 16px; flex-wrap: wrap; gap: 12px; }
.page-header h2 { margin: 0 0 4px; }
.subtitle { margin: 0; font-size: 13px; color: var(--el-text-color-secondary); }
.header-actions { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
.hint-alert { margin-bottom: 16px; }
.cache-layout { display: grid; grid-template-columns: 240px 1fr; gap: 16px; align-items: start; }
@media (max-width: 960px) { .cache-layout { grid-template-columns: 1fr; } }
.cache-sidebar { display: flex; flex-direction: column; gap: 12px; }
.metric-card { background: var(--el-bg-color); border: 1px solid var(--el-border-color-lighter); border-radius: 8px; padding: 12px; }
.metric-card.compact { padding: 14px 12px; }
.metric-head { display: flex; justify-content: space-between; align-items: baseline; margin-bottom: 4px; }
.metric-title { font-size: 12px; color: var(--el-text-color-secondary); text-transform: uppercase; letter-spacing: 0.03em; }
.metric-val { font-size: 18px; font-weight: 700; color: var(--el-text-color-primary); }
.metric-val.lg { font-size: 22px; margin-top: 4px; }
.quick-actions { background: var(--el-bg-color); border: 1px solid var(--el-border-color-lighter); border-radius: 8px; padding: 12px; }
.qa-title { font-size: 12px; font-weight: 600; margin-bottom: 10px; color: var(--el-text-color-secondary); }
.qa-btn { width: 100%; margin: 0 0 8px; }
.dev-toggle { display: flex; justify-content: space-between; align-items: center; margin-top: 8px; font-size: 13px; }
.cache-main { min-width: 0; }
.chart-card { margin-bottom: 16px; }
.chart-empty {
  min-height: 260px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  color: var(--el-text-color-secondary);
  font-size: 14px;
  padding: 24px;
}
.chart-empty-hint { margin: 8px 0 0; font-size: 13px; opacity: 0.85; max-width: 520px; }
.chart-card :deep(.el-card__header) { font-weight: 600; font-size: 14px; padding: 12px 16px; }
.status-list { margin-top: 8px; border-top: 1px solid var(--el-border-color-lighter); padding-top: 8px; }
.status-row { display: flex; justify-content: space-between; font-size: 12px; padding: 4px 0; color: var(--el-text-color-regular); }
.reserve-stat { text-align: center; margin-bottom: 16px; }
.reserve-label { font-size: 13px; color: var(--el-text-color-secondary); margin-bottom: 8px; }
.reserve-val { font-size: 26px; font-weight: 700; }
.cache-form { width: 100%; max-width: none; margin-top: 8px; }
.preset-row { display: flex; align-items: center; flex-wrap: wrap; gap: 8px; margin-bottom: 16px; }
.preset-label { font-size: 13px; color: var(--el-text-color-secondary); margin-right: 4px; }
.ttl-tag { margin-left: 4px; font-size: 11px; color: var(--el-text-color-secondary); }
.purge-domain { font-weight: 600; margin: 8px 0; }
.field-hint { margin-left: 10px; color: #909399; font-size: 13px; }
.sites-toolbar { margin-bottom: 12px; display: flex; align-items: center; flex-wrap: wrap; gap: 8px; }
.preview-btn { margin-bottom: 8px; }
.preview-box { font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; font-size: 12px; }
.rule-guide { margin-bottom: 16px; }
.guide-intro { margin: 0 0 10px; font-size: 13px; color: var(--el-text-color-secondary); }
.guide-steps { margin: 0 0 12px; padding-left: 20px; line-height: 1.8; font-size: 13px; }
.example-table { margin-top: 8px; }
.example-table code { font-size: 12px; }
.rules-main { min-width: 0; }
.toolbar-hint { font-size: 13px; color: var(--el-text-color-secondary); }
.pattern-code { font-size: 12px; }
.rule-form { padding-right: 8px; }
.field-hint.block { display: block; margin-left: 0; margin-top: 6px; }
.assistant-fab {
  position: fixed;
  right: 28px;
  bottom: 32px;
  z-index: 2000;
  width: 52px;
  height: 52px;
  box-shadow: 0 4px 16px rgba(64, 158, 255, 0.45);
}
.assistant-drawer :deep(.el-drawer__body) {
  padding: 12px 16px 20px;
  display: flex;
  flex-direction: column;
  height: calc(100% - 56px);
}
.assistant-drawer :deep(.rule-assistant) {
  flex: 1;
  min-height: 0;
}
.assistant-drawer :deep(.chat-box) {
  max-height: none;
  flex: 1;
}
</style>
