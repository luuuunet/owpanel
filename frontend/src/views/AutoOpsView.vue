<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import api from '@/api'
import { categoryLabel } from '@/locales'
import SoftwareIcon from '@/components/SoftwareIcon.vue'
import { ElMessage } from 'element-plus'
import {
  ArrowRight, Bell, Refresh, Timer, Promotion, Share, FolderOpened, Lock, Document, Box,
  Histogram, Cpu, Coin, Platform, DataAnalysis, CircleCheck, Upload, Tools, Monitor,
  MagicStick, WarningFilled,
} from '@element-plus/icons-vue'
import { cfTheme } from '@/config/theme'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const auth = useAuthStore()
const isAdmin = computed(() => !auth.user?.role || auth.user.role === 'admin')

const loading = ref(false)
const tab = ref('overview')
const hubFilter = ref('all')
const showGuideBanner = ref(
  typeof localStorage !== 'undefined' && !localStorage.getItem('autoOpsGuideSeen'),
)
const status = ref<any>(null)
const overview = ref<any>(null)
const events = ref<any[]>([])
const selectedKeys = ref<string[]>([])
const eventFilterType = ref('')
const eventFilterApp = ref('')
let timer: ReturnType<typeof setInterval>

const configForm = ref({
  enabled: true,
  interval_sec: 30,
  cooldown_sec: 300,
  max_restarts: 5,
  notify_webhook: '',
  notify_on_down: true,
  notify_on_fail: true,
  resource_enabled: false,
  cpu_threshold: 90,
  mem_threshold: 90,
  disk_threshold: 90,
  ssl_auto_renew: true,
  alert_days_ssl: 14,
  alert_days_site: 14,
  website_scan_enabled: true,
  mem_auto_relief: true,
})

const applyingPreset = ref('')

const cloudHub = ref<any>(null)
const cloudLoading = ref(false)
const applyingCloud = ref('')
const cloudStoragePick = ref<Record<string, number | null>>({})
const cloudTodosVisible = ref(false)
const cloudTodosList = ref<string[]>([])

const quickImporting = ref('')
const websiteAudits = ref<any>(null)
const auditDetail = ref<any>(null)
const auditDrawer = ref(false)
const scanningSite = ref<number | null>(null)
const scanningAll = ref(false)

const watches = computed(() => status.value?.watches || [])
const websiteAuditItems = computed(() => websiteAudits.value?.items || [])

const hubCategories = computed(() => [
  { key: 'all', label: t('autoOps.catAll'), icon: MagicStick },
  { key: 'site', label: t('autoOps.catSite'), icon: Monitor },
  { key: 'dev', label: t('autoOps.catDev'), icon: Promotion },
  { key: 'test', label: t('autoOps.catTest'), icon: DataAnalysis },
  { key: 'storage', label: t('autoOps.catStorage'), icon: FolderOpened },
  { key: 'security', label: t('autoOps.catSecurity'), icon: Lock },
  { key: 'ops', label: t('autoOps.catOps'), icon: Tools },
  { key: 'cloud', label: t('autoOps.catCloud'), icon: Share },
])

const scenarios = computed(() => [
  {
    key: 'site',
    tone: 'site',
    icon: Monitor,
    title: t('autoOps.scenarioSiteTitle'),
    desc: t('autoOps.scenarioSiteDesc'),
    cta: t('autoOps.scenarioSiteCta'),
    action: () => { hubFilter.value = 'site'; applyBeginnerPreset() },
  },
  {
    key: 'dev',
    tone: 'dev',
    icon: Promotion,
    title: t('autoOps.scenarioDevTitle'),
    desc: t('autoOps.scenarioDevDesc'),
    cta: t('autoOps.scenarioDevCta'),
    action: () => { hubFilter.value = 'dev'; goPath('/devops') },
  },
  {
    key: 'test',
    tone: 'test',
    icon: DataAnalysis,
    title: t('autoOps.scenarioTestTitle'),
    desc: t('autoOps.scenarioTestDesc'),
    cta: t('autoOps.scenarioTestCta'),
    action: () => { hubFilter.value = 'test'; tab.value = 'websites'; loadWebsiteAudits() },
  },
  {
    key: 'storage',
    tone: 'storage',
    icon: FolderOpened,
    title: t('autoOps.scenarioStorageTitle'),
    desc: t('autoOps.scenarioStorageDesc'),
    cta: t('autoOps.scenarioStorageCta'),
    action: () => { hubFilter.value = 'storage'; quickImportBackup() },
  },
  {
    key: 'security',
    tone: 'security',
    icon: Lock,
    title: t('autoOps.scenarioSecurityTitle'),
    desc: t('autoOps.scenarioSecurityDesc'),
    cta: t('autoOps.scenarioSecurityCta'),
    action: () => { hubFilter.value = 'security'; goPath('/protection', 'firewall') },
  },
])

const quickLinks = computed(() => [
  { path: '/websites', cat: 'site', icon: Monitor, title: t('menu.website'), desc: t('autoOps.linkSites'), audience: t('autoOps.audienceSite'), stat: overview.value?.sites_expiring_soon ? t('autoOps.expiringCount', { n: overview.value.sites_expiring_soon }) : '—' },
  { path: '/ssl', cat: 'security', icon: Lock, title: t('menu.ssl'), desc: t('autoOps.linkSSL'), audience: t('autoOps.audienceSite'), stat: overview.value?.ssl_expiring_soon ? t('autoOps.expiringCount', { n: overview.value.ssl_expiring_soon }) : '—' },
  { path: '/uptime', cat: 'test', icon: Bell, title: t('menu.uptime'), desc: t('autoOps.linkUptime'), audience: t('autoOps.audienceSite'), stat: overview.value ? `${(overview.value.uptime_total || 0) - (overview.value.uptime_down || 0)}/${overview.value.uptime_total || 0}` : '—' },
  { path: '/product-analytics', cat: 'test', icon: DataAnalysis, title: t('menu.abTesting'), desc: t('autoOps.linkAbTesting'), audience: t('autoOps.audienceProduct'), stat: 'A/B' },
  { path: '/auto-ops', tab: 'watch', cat: 'ops', icon: Refresh, title: t('autoOps.watchList'), desc: t('autoOps.linkWatch'), audience: t('autoOps.audienceOps'), stat: String(status.value?.watch_count ?? 0) },
  { path: '/auto-ops', tab: 'websites', cat: 'test', icon: CircleCheck, title: t('autoOps.websiteAuditTab'), desc: t('autoOps.linkWebsiteAudit'), audience: t('autoOps.audienceSite'), stat: overview.value?.website_avg_score != null ? String(overview.value.website_avg_score) : '—' },
  { path: '/cron', cat: 'ops', icon: Timer, title: t('menu.cron'), desc: t('autoOps.linkCron'), audience: t('autoOps.audienceAll'), stat: overview.value ? `${overview.value.cron_enabled || 0}/${overview.value.cron_total || 0}` : '—' },
  { path: '/backup', cat: 'storage', icon: FolderOpened, title: t('menu.backup'), desc: t('autoOps.linkBackup'), audience: t('autoOps.audienceSite'), stat: overview.value ? `${overview.value.backup_enabled || 0}/${overview.value.backup_total || 0}` : '—' },
  { path: '/oss', cat: 'storage', icon: Upload, title: t('menu.oss'), desc: t('autoOps.linkOSS'), audience: t('autoOps.audienceOps'), stat: cloudHub.value?.summary?.oss_storages ? String(cloudHub.value.summary.oss_storages) : '—' },
  { path: '/devops', cat: 'dev', icon: Promotion, title: t('menu.devops'), desc: t('autoOps.linkDevops'), audience: t('autoOps.audienceDev'), stat: 'CI/CD', adminOnly: true },
  { path: '/docker', cat: 'dev', icon: Box, title: t('menu.docker'), desc: t('autoOps.linkDocker'), audience: t('autoOps.audienceContainer'), stat: 'Docker' },
  { path: '/compose', cat: 'dev', icon: Box, title: t('menu.compose'), desc: t('autoOps.linkCompose'), audience: t('autoOps.audienceContainer'), stat: 'Stack' },
  { path: '/k8s', cat: 'dev', icon: Platform, title: t('menu.k8s'), desc: t('autoOps.linkK8s'), audience: t('autoOps.audienceContainer'), stat: overview.value?.k8s_ready ? t('k8s.ready') : (overview.value?.k8s_installed ? t('k8s.notReady') : '—'), adminOnly: true },
  { path: '/cluster', cat: 'cloud', icon: Share, title: t('menu.cluster'), desc: t('autoOps.linkCluster'), audience: t('autoOps.audienceOps'), stat: t('autoOps.multiNode') },
  { path: '/auto-ops', tab: 'cloud', cat: 'cloud', icon: Upload, title: t('autoOps.cloudTab'), desc: t('autoOps.linkCloud'), audience: t('autoOps.audienceOps'), stat: cloudHub.value?.summary?.oss_storages ? String(cloudHub.value.summary.oss_storages) : '—' },
  { path: '/protection', cat: 'security', icon: Histogram, title: t('menu.protection'), desc: t('autoOps.linkProtection'), audience: t('autoOps.audienceSecurity'), stat: 'WAF' },
  { path: '/logs', cat: 'ops', icon: Document, title: t('menu.logs'), desc: t('autoOps.linkLogs'), audience: t('autoOps.audienceOps'), stat: overview.value?.log_auto_cleanup ? t('autoOps.logCleanupOn') : t('autoOps.logCleanupOff'), adminOnly: true },
  { path: '/extensions', cat: 'dev', icon: Box, title: t('menu.extensions'), desc: t('autoOps.linkExtensions'), audience: t('autoOps.audienceDev'), stat: t('autoOps.hooks'), adminOnly: true },
])

const capabilityGroups = computed(() => {
  const order = hubCategories.value.filter((c) => c.key !== 'all')
  return order
    .map((cat) => ({
      ...cat,
      items: quickLinks.value.filter((l) => l.cat === cat.key && (!l.adminOnly || isAdmin.value)),
    }))
    .filter((g) => g.items.length > 0)
    .filter((g) => hubFilter.value === 'all' || g.key === hubFilter.value)
})

const healthChips = computed(() => [
  {
    label: t('autoOps.serviceWatch'),
    value: String(status.value?.watch_count ?? 0),
    warn: (status.value?.down_count ?? 0) > 0,
    click: () => { tab.value = 'watch' },
  },
  {
    label: t('autoOps.uptimeMonitors'),
    value: String(overview.value?.uptime_total ?? 0),
    warn: (overview.value?.uptime_down ?? 0) > 0,
    click: () => goPath('/uptime'),
  },
  {
    label: t('autoOps.backupTasks'),
    value: `${overview.value?.backup_enabled ?? 0}/${overview.value?.backup_total ?? 0}`,
    warn: false,
    click: () => goPath('/backup'),
  },
  {
    label: t('autoOps.sslExpiring'),
    value: String(overview.value?.ssl_expiring_soon ?? 0),
    warn: (overview.value?.ssl_expiring_soon ?? 0) > 0,
    click: () => goPath('/ssl'),
  },
  {
    label: t('autoOps.websiteAudit'),
    value: overview.value?.website_avg_score != null ? String(overview.value.website_avg_score) : '—',
    warn: (overview.value?.website_issues ?? 0) > 0,
    click: () => { tab.value = 'websites'; loadWebsiteAudits() },
  },
  {
    label: t('autoOps.cronJobs'),
    value: `${overview.value?.cron_enabled ?? 0}/${overview.value?.cron_total ?? 0}`,
    warn: (overview.value?.cron_failed ?? 0) > 0,
    click: () => goPath('/cron'),
  },
])

function dismissGuideBanner() {
  showGuideBanner.value = false
  localStorage.setItem('autoOpsGuideSeen', '1')
}

function openGuide() {
  tab.value = 'guide'
  dismissGuideBanner()
}

function openCapability(link: { path: string; tab?: string; adminOnly?: boolean }) {
  if (link.adminOnly && !isAdmin.value) {
    ElMessage.warning(t('autoOps.adminOnly'))
    return
  }
  goPath(link.path, link.tab)
}

const beginnerPaths = computed(() => [
  {
    key: 'site',
    icon: '🌐',
    title: t('autoOps.pathSiteTitle'),
    desc: t('autoOps.pathSiteDesc'),
    steps: [
      { text: t('autoOps.pathSite1'), path: '/websites' },
      { text: t('autoOps.pathSite2'), path: '/ssl' },
      { text: t('autoOps.pathSite3'), path: '/auto-ops', tab: 'watch' },
      { text: t('autoOps.pathSite4'), path: '/backup' },
      { text: t('autoOps.pathSite5'), path: '/uptime' },
    ],
  },
  {
    key: 'ops',
    icon: '🔧',
    title: t('autoOps.pathOpsTitle'),
    desc: t('autoOps.pathOpsDesc'),
    steps: [
      { text: t('autoOps.pathOps1'), path: '/auto-ops', tab: 'settings' },
      { text: t('autoOps.pathOps2'), path: '/auto-ops', tab: 'watch' },
      { text: t('autoOps.pathOps3'), path: '/cron' },
      { text: t('autoOps.pathOps4'), path: '/auto-ops', tab: 'events' },
      { text: t('autoOps.pathOps5'), path: '/logs' },
    ],
  },
  {
    key: 'container',
    icon: '📦',
    title: t('autoOps.pathContainerTitle'),
    desc: t('autoOps.pathContainerDesc'),
    steps: [
      { text: t('autoOps.pathContainer1'), path: '/docker' },
      { text: t('autoOps.pathContainer2'), path: '/k8s' },
      { text: t('autoOps.pathContainer3'), path: '/devops' },
      { text: t('autoOps.pathContainer4'), path: '/cluster' },
    ],
  },
  {
    key: 'cloud',
    icon: '☁️',
    title: t('autoOps.pathCloudTitle'),
    desc: t('autoOps.pathCloudDesc'),
    steps: [
      { text: t('autoOps.pathCloud1'), path: '/auto-ops', tab: 'cloud' },
      { text: t('autoOps.pathCloud2'), path: '/oss' },
      { text: t('autoOps.pathCloud3'), path: '/dns' },
      { text: t('autoOps.pathCloud4'), path: '/backup' },
      { text: t('autoOps.pathCloud5'), path: '/auto-ops', tab: 'settings' },
    ],
  },
])

const compareRows = computed(() => [
  { feature: t('autoOps.cmpResourceMonitor'), ow: true, bt: true, op: true, ali: true, aws: true, gcp: true },
  { feature: t('autoOps.cmpMetricAlarm'), ow: true, bt: true, op: true, ali: true, aws: true, gcp: true },
  { feature: t('autoOps.cmpServiceWatch'), ow: true, bt: true, op: true, ali: 'partial', aws: 'partial', gcp: 'partial' },
  { feature: t('autoOps.cmpSslRenew'), ow: true, bt: true, op: true, ali: true, aws: true, gcp: 'partial' },
  { feature: t('autoOps.cmpBackup'), ow: true, bt: true, op: true, ali: 'partial', aws: true, gcp: true },
  { feature: t('autoOps.cmpAutoSnapshot'), ow: true, bt: true, op: true, ali: 'partial', aws: true, gcp: true },
  { feature: t('autoOps.cmpCron'), ow: true, bt: true, op: true, ali: 'partial', aws: 'partial', gcp: 'partial' },
  { feature: t('autoOps.cmpUptime'), ow: true, bt: true, op: true, ali: 'partial', aws: 'partial', gcp: true },
  { feature: t('autoOps.cmpWebsiteAudit'), ow: true, bt: false, op: false, ali: false, aws: false, gcp: false },
  { feature: t('autoOps.cmpK8s'), ow: true, bt: false, op: true, ali: 'partial', aws: true, gcp: true },
  { feature: t('autoOps.cmpLoadBalancer'), ow: 'partial', bt: 'partial', op: false, ali: true, aws: true, gcp: true },
  { feature: t('autoOps.cmpAbTest'), ow: true, bt: false, op: false, ali: false, aws: false, gcp: false },
  { feature: t('autoOps.cmpDevops'), ow: true, bt: false, op: false, ali: 'partial', aws: true, gcp: true },
  { feature: t('autoOps.cmpMemRelief'), ow: true, bt: false, op: false, ali: false, aws: false, gcp: false },
  { feature: t('autoOps.cmpWebhook'), ow: true, bt: true, op: true, ali: true, aws: true, gcp: true },
  { feature: t('autoOps.cmpCommandRun'), ow: true, bt: true, op: true, ali: true, aws: 'partial', gcp: 'partial' },
  { feature: t('autoOps.cmpGlobalProbe'), ow: false, bt: false, op: false, ali: false, aws: 'partial', gcp: true },
  { feature: t('autoOps.cmpHooks'), ow: true, bt: true, op: false, ali: false, aws: 'partial', gcp: 'partial' },
])

const compareCols = [
  { key: 'ow', label: 'autoOps.cmpOw' },
  { key: 'bt', label: 'autoOps.cmpBt' },
  { key: 'op', label: 'autoOps.cmpOp' },
  { key: 'ali', label: 'autoOps.cmpAli' },
  { key: 'aws', label: 'autoOps.cmpAws' },
  { key: 'gcp', label: 'autoOps.cmpGcp' },
]

const glossaryItems = computed(() => [
  { term: t('autoOps.glossaryWatch'), def: t('autoOps.glossaryWatchDef') },
  { term: t('autoOps.glossaryCron'), def: t('autoOps.glossaryCronDef') },
  { term: t('autoOps.glossaryWebhook'), def: t('autoOps.glossaryWebhookDef') },
  { term: t('autoOps.glossarySSL'), def: t('autoOps.glossarySSLDef') },
  { term: t('autoOps.glossaryUptime'), def: t('autoOps.glossaryUptimeDef') },
  { term: t('autoOps.glossaryK8s'), def: t('autoOps.glossaryK8sDef') },
])

const cloudSummaryTags = computed(() => {
  const s = cloudHub.value?.summary
  if (!s) return []
  return [
    { label: t('autoOps.cloudStatOSS'), value: s.oss_storages },
    { label: t('autoOps.cloudStatDNS'), value: s.dns_providers },
    { label: t('autoOps.cloudStatBackup'), value: s.backup_tasks },
    { label: t('autoOps.cloudStatBackupOSS'), value: s.backup_with_oss },
    { label: t('autoOps.cloudStatUptime'), value: s.uptime_monitors },
  ]
})

function cloudFeatureLabel(key: string) {
  const map: Record<string, string> = {
    oss: 'autoOps.cloudFeatureOss',
    dns: 'autoOps.cloudFeatureDns',
    backup: 'autoOps.cloudFeatureBackup',
    uptime: 'autoOps.cloudFeatureUptime',
    autops: 'autoOps.cloudFeatureAutops',
    sync: 'autoOps.cloudFeatureSync',
    mail: 'autoOps.cloudFeatureMail',
    cluster: 'autoOps.cloudFeatureCluster',
    monitor: 'autoOps.cloudFeatureAutops',
  }
  return t(map[key] || key)
}

function goPath(path: string, tabName?: string, query?: Record<string, string>) {
  if (path === '/auto-ops' && tabName) {
    tab.value = tabName
    router.replace({ path, query: { tab: tabName, ...query } })
    return
  }
  const q = { ...(query || {}), ...(tabName ? { tab: tabName } : {}) }
  router.push({ path, query: Object.keys(q).length ? q : undefined })
}

function goOSSProvider(provider: string) {
  router.push({ path: '/oss', query: { provider, add: '1' } })
}

async function applyBeginnerPreset() {
  applyingPreset.value = 'site'
  try {
    const res: any = await api.post('/auto-ops/presets/site')
    const d = res.data || {}
    ElMessage.success(t('autoOps.presetSiteAppliedDetail', {
      watch: d.autops?.watch_count ?? 0,
      uptime: d.uptime?.created ?? 0,
      backup: (d.backup_websites?.created ?? 0) + (d.backup_databases?.created ?? 0),
    }))
    await load()
    await loadOverview()
    tab.value = 'watch'
  } catch (e: any) {
    ElMessage.error(e?.error || e?.message || t('autoOps.updateFailed'))
  } finally {
    applyingPreset.value = ''
  }
}

async function applyOpsPreset() {
  applyingPreset.value = 'ops'
  try {
    await api.post('/auto-ops/presets/ops')
    ElMessage.success(t('autoOps.presetOpsApplied'))
    await load()
    await loadOverview()
    tab.value = 'settings'
  } catch (e: any) {
    ElMessage.error(e?.error || e?.message || t('autoOps.updateFailed'))
  } finally {
    applyingPreset.value = ''
  }
}

async function quickImportUptime() {
  quickImporting.value = 'uptime'
  try {
    const res: any = await api.post('/uptime/import-websites', { interval_sec: 300 })
    const d = res.data || {}
    ElMessage.success(t('autoOps.quickImportUptimeDone', { created: d.created || 0, skipped: d.skipped || 0 }))
    await loadOverview()
  } catch (e: any) {
    ElMessage.error(e?.error || e?.message || t('autoOps.updateFailed'))
  } finally {
    quickImporting.value = ''
  }
}

async function quickImportBackup() {
  quickImporting.value = 'backup'
  try {
    const res: any = await api.post('/backup/presets', { preset: 'websites', schedule: '0 2 * * *' })
    const d = res.data || {}
    ElMessage.success(t('autoOps.quickImportBackupDone', { created: d.created || 0, skipped: d.skipped || 0 }))
    await loadOverview()
  } catch (e: any) {
    ElMessage.error(e?.error || e?.message || t('autoOps.updateFailed'))
  } finally {
    quickImporting.value = ''
  }
}

async function loadCloudHub() {
  cloudLoading.value = true
  try {
    const res: any = await api.get('/cloud/hub')
    cloudHub.value = res.data
    for (const v of res.data?.vendors || []) {
      if (cloudStoragePick.value[v.key] === undefined) {
        cloudStoragePick.value[v.key] = v.storages?.[0]?.id ?? null
      }
    }
  } catch (e: any) {
    ElMessage.error(e?.error || e?.message || t('autoOps.updateFailed'))
  } finally {
    cloudLoading.value = false
  }
}

async function applyCloudPreset(vendorKey: string) {
  applyingCloud.value = vendorKey
  try {
    const storageId = cloudStoragePick.value[vendorKey]
    const res: any = await api.post(`/cloud/presets/${vendorKey}`, {
      oss_storage_id: storageId || undefined,
      include_ops: true,
      link_backup: true,
      create_sync: !!storageId,
    })
    const d = res.data || {}
    const backupN = (d.backup_websites?.created ?? 0) + (d.backup_databases?.created ?? 0)
    ElMessage.success(t('autoOps.cloudPresetApplied', {
      watch: d.autops?.watch_count ?? 0,
      uptime: d.uptime?.created ?? 0,
      backup: backupN,
      linked: d.backup_linked ?? 0,
    }))
    if (d.todos?.length) {
      cloudTodosList.value = d.todos
      cloudTodosVisible.value = true
    }
    await load()
    await loadOverview()
    await loadCloudHub()
  } catch (e: any) {
    ElMessage.error(e?.error || e?.message || t('autoOps.updateFailed'))
  } finally {
    applyingCloud.value = ''
  }
}

function liveTagType(s: string) {
  if (s === 'running') return 'success'
  if (s === 'stopped') return 'info'
  return 'warning'
}

function liveStatusLabel(s: string) {
  if (s === 'running') return t('common.running')
  if (s === 'stopped') return t('common.stopped')
  return s || '—'
}

function eventTagType(type: string) {
  if (type === 'restart_ok') return 'success'
  if (type === 'down_detected') return 'warning'
  if (type?.startsWith('resource_')) return 'danger'
  if (type === 'cron_failed') return 'danger'
  if (type === 'ssl_expiring' || type === 'site_expiring') return 'warning'
  if (type === 'ssl_renew_fail') return 'danger'
  if (type === 'ssl_renew_ok') return 'success'
  if (type === 'restart_fail' || type === 'restart_skipped') return 'danger'
  return 'info'
}

function eventLabel(type: string) {
  const map: Record<string, string> = {
    down_detected: t('autoOps.eventDown'),
    restart_ok: t('autoOps.eventRestartOk'),
    restart_fail: t('autoOps.eventRestartFail'),
    restart_skipped: t('autoOps.eventSkipped'),
    resource_cpu: t('autoOps.eventResourceCPU'),
    resource_memory: t('autoOps.eventResourceMem'),
    resource_disk: t('autoOps.eventResourceDisk'),
    cron_failed: t('autoOps.eventCronFailed'),
    ssl_expiring: t('autoOps.eventSSLExpiring'),
    site_expiring: t('autoOps.eventSiteExpiring'),
    ssl_renew_fail: t('autoOps.eventSSLRenewFail'),
    ssl_renew_ok: t('autoOps.eventSSLRenewOk'),
    website_issue: t('autoOps.eventWebsiteIssue'),
  }
  return map[type] || type
}

function gradeTagType(grade: string) {
  if (grade === 'A' || grade === 'B') return 'success'
  if (grade === 'C') return 'warning'
  return 'danger'
}

function severityTagType(sev: string) {
  if (sev === 'critical') return 'danger'
  if (sev === 'warning') return 'warning'
  return 'info'
}

function formatTime(v: string) {
  if (!v) return '—'
  return new Date(v).toLocaleString()
}

function resourceColor(pct: number) {
  if (pct >= 90) return cfTheme.danger
  if (pct >= 75) return cfTheme.warning
  return cfTheme.success
}

function resourcePercent(v: number | undefined | null) {
  const n = Math.min(100, Math.max(0, Number(v) || 0))
  return Math.round(n * 10) / 10
}

function resourceFormat(pct: number) {
  return `${resourcePercent(pct)}%`
}

async function loadWebsiteAudits() {
  try {
    const res: any = await api.get('/auto-ops/website-audits')
    websiteAudits.value = res.data
  } catch {
    /* optional */
  }
}

async function scanAllWebsites() {
  scanningAll.value = true
  try {
    const res: any = await api.post('/auto-ops/website-scan')
    websiteAudits.value = res.data
    ElMessage.success(t('autoOps.websiteScanDone'))
    await loadOverview()
    await loadEvents()
  } catch (e: any) {
    ElMessage.error(e?.error || e?.message || t('autoOps.updateFailed'))
  } finally {
    scanningAll.value = false
  }
}

async function scanOneWebsite(row: { site_id: number }) {
  scanningSite.value = row.site_id
  try {
    const res: any = await api.post(`/auto-ops/website-audits/${row.site_id}/scan`)
    await loadWebsiteAudits()
    auditDetail.value = res.data
    auditDrawer.value = true
    ElMessage.success(t('autoOps.websiteScanDone'))
  } catch (e: any) {
    ElMessage.error(e?.error || e?.message || t('autoOps.updateFailed'))
  } finally {
    scanningSite.value = null
  }
}

async function openAuditDetail(row: { site_id: number }) {
  try {
    const res: any = await api.get(`/auto-ops/website-audits/${row.site_id}`)
    auditDetail.value = res.data
    auditDrawer.value = true
  } catch (e: any) {
    ElMessage.error(e?.error || e?.message || t('autoOps.updateFailed'))
  }
}

async function load() {
  loading.value = true
  try {
    const res: any = await api.get('/auto-ops/status')
    status.value = res.data
    if (res.data?.config) {
      configForm.value = { ...configForm.value, ...res.data.config }
    }
  } catch (e: any) {
    ElMessage.error(e?.error || e?.message || t('autoOps.updateFailed'))
  } finally {
    loading.value = false
  }
}

async function loadOverview() {
  try {
    const res: any = await api.get('/auto-ops/overview')
    overview.value = res.data
    if (res.data?.status) status.value = res.data.status
  } catch {
    /* optional */
  }
}

async function loadEvents() {
  const params: Record<string, string> = { limit: '100' }
  if (eventFilterType.value) params.event_type = eventFilterType.value
  if (eventFilterApp.value) params.app_key = eventFilterApp.value
  const res: any = await api.get('/auto-ops/events', { params })
  events.value = res.data || []
}

async function saveConfig() {
  await api.put('/auto-ops/config', configForm.value)
  ElMessage.success(t('autoOps.configSaved'))
  load()
}

async function scanNow() {
  loading.value = true
  try {
    const res: any = await api.post('/auto-ops/scan')
    status.value = res.data
    ElMessage.success(t('autoOps.scanDone'))
    await loadOverview()
    await loadEvents()
    await loadWebsiteAudits()
  } finally {
    loading.value = false
  }
}

async function patchWatch(row: any, field: 'watch_enabled' | 'auto_restart', val: boolean) {
  const payload: Record<string, boolean> = { [field]: val }
  if (field === 'auto_restart' && val) payload.watch_enabled = true
  if (field === 'watch_enabled' && !val) payload.auto_restart = false
  try {
    await api.patch(`/auto-ops/watch/${row.key}`, payload)
    ElMessage.success(t('autoOps.bulkUpdated'))
    await load()
  } catch (e: any) {
    ElMessage.error(e?.error || e?.message || t('autoOps.updateFailed'))
    await load()
  }
}

async function bulkEnable(autoRestart: boolean) {
  if (!selectedKeys.value.length) return
  try {
    await api.post('/auto-ops/watch/bulk', {
      keys: selectedKeys.value,
      watch_enabled: true,
      auto_restart: autoRestart,
    })
    ElMessage.success(t('autoOps.bulkUpdated'))
    await load()
  } catch (e: any) {
    ElMessage.error(e?.error || e?.message || t('autoOps.updateFailed'))
  }
}

onMounted(() => {
  load()
  loadOverview()
  loadEvents()
  const qTab = String(route.query.tab || '')
  if (['watch', 'events', 'settings', 'websites', 'guide', 'cloud', 'overview'].includes(qTab)) {
    tab.value = qTab
  }
  if (tab.value === 'guide') dismissGuideBanner()
  loadWebsiteAudits()
  loadCloudHub()
  timer = setInterval(() => {
    load()
    if (tab.value === 'overview') loadOverview()
    if (tab.value === 'events') loadEvents()
    if (tab.value === 'websites') loadWebsiteAudits()
  }, 15000)
})

onUnmounted(() => clearInterval(timer))
</script>

<template>
  <div class="auto-ops-page" v-loading="loading">
    <header class="ao-hero">
      <div class="ao-hero-main">
        <div class="ao-hero-badge">
          <el-icon :size="22"><Tools /></el-icon>
        </div>
        <div>
          <h1 class="ao-title">{{ t('autoOps.title') }}</h1>
          <p class="ao-sub">{{ t('autoOps.subtitleShort') }}</p>
          <p v-if="status?.last_scan" class="ao-meta">{{ t('autoOps.lastScan') }}: {{ formatTime(status.last_scan) }}</p>
        </div>
      </div>
      <div class="ao-hero-actions">
        <div class="ao-status-pills">
          <span class="ao-pill" :class="status?.config?.enabled ? 'ok' : 'off'">
            {{ status?.config?.enabled ? t('autoOps.enabled') : t('autoOps.disabled') }}
          </span>
          <span class="ao-pill">{{ t('autoOps.watching', { n: status?.watch_count ?? 0 }) }}</span>
          <span v-if="(status?.down_count ?? 0) > 0" class="ao-pill danger">
            <el-icon><WarningFilled /></el-icon>
            {{ t('autoOps.downCount', { n: status?.down_count }) }}
          </span>
        </div>
        <el-button @click="openGuide">{{ t('autoOps.guideTab') }}</el-button>
        <el-button type="primary" @click="scanNow">{{ t('autoOps.scanNow') }}</el-button>
      </div>
    </header>

    <div v-if="showGuideBanner" class="ao-banner">
      <div>
        <strong>{{ t('autoOps.bannerTitle') }}</strong>
        <p>{{ t('autoOps.bannerBody') }}</p>
      </div>
      <div class="ao-banner-actions">
        <el-button type="primary" @click="openGuide">{{ t('autoOps.bannerGuide') }}</el-button>
        <el-button @click="dismissGuideBanner">{{ t('autoOps.bannerDismiss') }}</el-button>
      </div>
    </div>

    <el-tabs v-model="tab" class="ao-tabs" @tab-change="(name: string) => { if (name === 'websites') loadWebsiteAudits(); if (name === 'cloud') loadCloudHub(); if (name === 'guide') dismissGuideBanner() }">
      <el-tab-pane :label="t('autoOps.guideTab')" name="guide">
        <el-alert type="info" :closable="false" show-icon class="guide-intro">
          <template #title>{{ t('autoOps.guideIntroTitle') }}</template>
          <template #default>
            <p class="guide-intro-text">{{ t('autoOps.guideIntroBody') }}</p>
          </template>
        </el-alert>

        <h3 class="section-title">{{ t('autoOps.presetTitle') }}</h3>
        <p class="section-desc">{{ t('autoOps.presetDesc') }}</p>
        <div class="preset-row">
          <el-card shadow="never" class="preset-card">
            <div class="preset-head">🚀 {{ t('autoOps.presetSiteTitle') }}</div>
            <p class="preset-body">{{ t('autoOps.presetSiteBody') }}</p>
            <ul class="preset-list">
              <li>{{ t('autoOps.presetSiteItem1') }}</li>
              <li>{{ t('autoOps.presetSiteItem2') }}</li>
              <li>{{ t('autoOps.presetSiteItem3') }}</li>
            </ul>
            <el-button type="primary" :loading="applyingPreset === 'site'" @click="applyBeginnerPreset">
              {{ t('autoOps.presetSiteBtn') }}
            </el-button>
          </el-card>
          <el-card shadow="never" class="preset-card">
            <div class="preset-head">🛡️ {{ t('autoOps.presetOpsTitle') }}</div>
            <p class="preset-body">{{ t('autoOps.presetOpsBody') }}</p>
            <ul class="preset-list">
              <li>{{ t('autoOps.presetOpsItem1') }}</li>
              <li>{{ t('autoOps.presetOpsItem2') }}</li>
              <li>{{ t('autoOps.presetOpsItem3') }}</li>
            </ul>
            <el-button type="success" :loading="applyingPreset === 'ops'" @click="applyOpsPreset">
              {{ t('autoOps.presetOpsBtn') }}
            </el-button>
          </el-card>
        </div>

        <h3 class="section-title">{{ t('autoOps.pathTitle') }}</h3>
        <p class="section-desc">{{ t('autoOps.pathDesc') }}</p>
        <div class="path-grid">
          <el-card v-for="path in beginnerPaths" :key="path.key" shadow="never" class="path-card">
            <div class="path-head">
              <span class="path-icon">{{ path.icon }}</span>
              <div>
                <div class="path-title">{{ path.title }}</div>
                <div class="path-desc">{{ path.desc }}</div>
              </div>
            </div>
            <ol class="path-steps">
              <li v-for="(step, i) in path.steps" :key="i">
                <button type="button" class="path-step-link" @click="goPath(step.path, step.tab)">
                  {{ step.text }}
                  <el-icon><ArrowRight /></el-icon>
                </button>
              </li>
            </ol>
          </el-card>
        </div>

        <h3 class="section-title">{{ t('autoOps.cmpTitle') }}</h3>
        <p class="section-desc">{{ t('autoOps.cmpDesc') }}</p>
        <div class="cmp-wrap">
          <el-table :data="compareRows" stripe class="cmp-table">
            <el-table-column prop="feature" :label="t('autoOps.cmpFeature')" min-width="220" fixed />
            <el-table-column v-for="col in compareCols" :key="col.key" :label="t(col.label)" width="88" align="center">
              <template #default="{ row }">
                <el-icon v-if="row[col.key] === true" color="var(--el-color-success)"><CircleCheck /></el-icon>
                <span v-else-if="row[col.key] === 'partial'" class="cmp-partial" :title="t('autoOps.cmpPartialHint')">~</span>
                <span v-else class="cmp-no">—</span>
              </template>
            </el-table-column>
          </el-table>
        </div>
        <p class="cmp-legend">{{ t('autoOps.cmpLegend') }}</p>

        <el-alert type="success" :closable="false" show-icon class="guide-doc">
          <template #title>{{ t('autoOps.guideDocTitle') }}</template>
          <template #default>{{ t('autoOps.guideDocBody') }}</template>
        </el-alert>

        <h3 class="section-title">{{ t('autoOps.glossaryTitle') }}</h3>
        <div class="glossary-grid">
          <div v-for="item in glossaryItems" :key="item.term" class="glossary-item">
            <strong>{{ item.term }}</strong>
            <p>{{ item.def }}</p>
          </div>
        </div>

        <el-alert type="warning" :closable="false" show-icon class="guide-faq">
          <template #title>{{ t('autoOps.faqTitle') }}</template>
          <template #default>
            <dl class="faq-list">
              <dt>{{ t('autoOps.faq1Q') }}</dt>
              <dd>{{ t('autoOps.faq1A') }}</dd>
              <dt>{{ t('autoOps.faq2Q') }}</dt>
              <dd>{{ t('autoOps.faq2A') }}</dd>
              <dt>{{ t('autoOps.faq3Q') }}</dt>
              <dd>{{ t('autoOps.faq3A') }}</dd>
            </dl>
          </template>
        </el-alert>
      </el-tab-pane>

      <el-tab-pane :label="t('autoOps.cloudTab')" name="cloud">
        <div v-loading="cloudLoading">
          <el-alert type="info" :closable="false" show-icon class="guide-intro">
            <template #title>{{ t('autoOps.cloudIntroTitle') }}</template>
            <template #default>{{ t('autoOps.cloudIntroBody') }}</template>
          </el-alert>

          <div v-if="cloudHub?.summary" class="cloud-summary">
            <el-tag v-for="s in cloudSummaryTags" :key="s.label" type="info" effect="plain">{{ s.label }}: {{ s.value }}</el-tag>
          </div>

          <h3 class="section-title">{{ t('autoOps.cloudIntegrationsTitle') }}</h3>
          <p class="section-desc">{{ t('autoOps.cloudIntegrationsDesc') }}</p>
          <div class="cloud-int-grid">
            <el-card v-for="item in cloudHub?.integrations || []" :key="item.key" shadow="never" class="cloud-int-card" @click="goPath(item.route, item.key === 'autops' ? 'overview' : undefined, item.key === 'sync' ? { tab: 'tasks' } : undefined)">
              <div class="cloud-int-head">
                <span>{{ cloudFeatureLabel(item.key) }}</span>
                <el-tag size="small" :type="item.configured ? 'success' : 'info'">{{ item.configured ? t('autoOps.cloudConfigured') : t('autoOps.cloudNotConfigured') }}</el-tag>
              </div>
              <p class="cloud-int-desc">{{ t(`autoOps.cloudFeatureDesc_${item.key}`) }}</p>
              <div class="cloud-int-foot">{{ t('autoOps.cloudCount', { n: item.count }) }} · {{ t('autoOps.cloudOpen') }} →</div>
            </el-card>
          </div>

          <h3 class="section-title">{{ t('autoOps.cloudVendorsTitle') }}</h3>
          <p class="section-desc">{{ t('autoOps.cloudVendorsDesc') }}</p>
          <div class="cloud-vendor-grid">
            <el-card v-for="v in cloudHub?.vendors || []" :key="v.key" shadow="never" class="cloud-vendor-card">
              <div class="preset-head">{{ v.name }}</div>
              <p class="preset-body">{{ v.description }}</p>
              <div class="cloud-vendor-tags">
                <el-tag v-if="v.oss_count" size="small" type="success">OSS ×{{ v.oss_count }}</el-tag>
                <el-tag v-if="v.dns_ready" size="small" type="success">DNS</el-tag>
                <el-tag v-if="v.mail_ready" size="small" type="success">{{ t('autoOps.cloudMail') }}</el-tag>
              </div>
              <el-form v-if="v.storages?.length" label-width="80px" class="cloud-storage-pick">
                <el-form-item :label="t('autoOps.cloudStorage')">
                  <el-select v-model="cloudStoragePick[v.key]" style="width: 100%">
                    <el-option v-for="st in v.storages" :key="st.id" :label="`${st.name} (${st.bucket || st.provider})`" :value="st.id" />
                  </el-select>
                </el-form-item>
              </el-form>
              <ul class="preset-list">
                <li v-for="f in v.features" :key="f.key">{{ cloudFeatureLabel(f.key) }} — {{ f.configured ? '✓' : '—' }}</li>
              </ul>
              <el-alert v-if="!v.storages?.length" type="warning" :closable="false" show-icon class="cloud-setup-hint">
                <template #default>
                  {{ t('autoOps.cloudSetupOssHint', { name: v.name }) }}
                  <el-button link type="primary" @click="goOSSProvider(v.key)">{{ t('autoOps.cloudSetupOssBtn') }}</el-button>
                </template>
              </el-alert>
              <div class="cloud-vendor-actions">
                <el-button type="primary" :loading="applyingCloud === v.key" @click="applyCloudPreset(v.key)">
                  {{ t('autoOps.cloudPresetBtn', { name: v.name }) }}
                </el-button>
                <el-button @click="goOSSProvider(v.key)">{{ t('autoOps.cloudLinkOSS') }}</el-button>
                <el-button v-if="v.key === 'aliyun' || v.key === 'tencent'" @click="goPath('/dns')">{{ t('autoOps.cloudLinkDNS') }}</el-button>
              </div>
            </el-card>
          </div>

          <el-alert type="success" :closable="false" show-icon class="guide-doc">
            <template #title>{{ t('autoOps.cloudPackTitle') }}</template>
            <template #default>
              <ul class="preset-list">
                <li>{{ t('autoOps.cloudPackItem1') }}</li>
                <li>{{ t('autoOps.cloudPackItem2') }}</li>
                <li>{{ t('autoOps.cloudPackItem3') }}</li>
                <li>{{ t('autoOps.cloudPackItem4') }}</li>
                <li>{{ t('autoOps.cloudPackItem5') }}</li>
              </ul>
            </template>
          </el-alert>
        </div>
      </el-tab-pane>

      <el-tab-pane :label="t('autoOps.overview')" name="overview">
        <section class="ao-section">
          <div class="ao-section-head">
            <h3>{{ t('autoOps.scenarioTitle') }}</h3>
            <p>{{ t('autoOps.scenarioDesc') }}</p>
          </div>
          <div class="scenario-grid">
            <article
              v-for="sc in scenarios"
              :key="sc.key"
              class="scenario-card"
              :class="[`tone-${sc.tone}`, { active: hubFilter === sc.key }]"
              @click="hubFilter = sc.key"
            >
              <div class="scenario-icon"><el-icon :size="20"><component :is="sc.icon" /></el-icon></div>
              <div class="scenario-body">
                <h4>{{ sc.title }}</h4>
                <p>{{ sc.desc }}</p>
              </div>
              <el-button
                size="small"
                type="primary"
                plain
                :loading="(sc.key === 'site' && applyingPreset === 'site') || (sc.key === 'storage' && quickImporting === 'backup')"
                @click.stop="sc.action()"
              >
                {{ sc.cta }}
              </el-button>
            </article>
          </div>
        </section>

        <section class="ao-section health-section">
          <div class="ao-section-head">
            <h3>{{ t('autoOps.healthTitle') }}</h3>
            <p>{{ t('autoOps.healthDesc') }}</p>
          </div>
          <div class="health-board">
            <div class="gauge-row">
              <div class="gauge-card">
                <el-progress type="dashboard" :percentage="resourcePercent(overview?.cpu_percent)" :color="resourceColor(overview?.cpu_percent || 0)" :width="100" :stroke-width="9">
                  <template #default><span class="gauge-val">{{ resourceFormat(overview?.cpu_percent) }}</span></template>
                </el-progress>
                <div class="resource-label"><el-icon><Cpu /></el-icon><span>CPU</span></div>
              </div>
              <div class="gauge-card">
                <el-progress type="dashboard" :percentage="resourcePercent(overview?.memory_percent)" :color="resourceColor(overview?.memory_percent || 0)" :width="100" :stroke-width="9">
                  <template #default><span class="gauge-val">{{ resourceFormat(overview?.memory_percent) }}</span></template>
                </el-progress>
                <div class="resource-label"><el-icon><Coin /></el-icon><span>{{ t('autoOps.memory') }}</span></div>
              </div>
              <div class="gauge-card">
                <el-progress type="dashboard" :percentage="resourcePercent(overview?.disk_percent)" :color="resourceColor(overview?.disk_percent || 0)" :width="100" :stroke-width="9">
                  <template #default><span class="gauge-val">{{ resourceFormat(overview?.disk_percent) }}</span></template>
                </el-progress>
                <div class="resource-label"><el-icon><FolderOpened /></el-icon><span>{{ t('autoOps.disk') }}</span></div>
              </div>
            </div>
            <div class="chip-row">
              <button
                v-for="chip in healthChips"
                :key="chip.label"
                type="button"
                class="health-chip"
                :class="{ warn: chip.warn }"
                @click="chip.click()"
              >
                <span class="health-chip-label">{{ chip.label }}</span>
                <strong>{{ chip.value }}</strong>
              </button>
            </div>
          </div>
        </section>

        <section class="ao-section">
          <div class="ao-section-head">
            <h3>{{ t('autoOps.quickActionsTitle') }}</h3>
            <p>{{ t('autoOps.quickActionsDesc') }}</p>
          </div>
          <div class="action-grid">
            <button type="button" class="action-tile" @click="scanNow">
              <el-icon><Refresh /></el-icon>
              <span>{{ t('autoOps.scanNow') }}</span>
            </button>
            <button type="button" class="action-tile" :disabled="!!quickImporting" @click="quickImportUptime">
              <el-icon><Bell /></el-icon>
              <span>{{ t('autoOps.quickImportUptime') }}</span>
            </button>
            <button type="button" class="action-tile" :disabled="!!quickImporting" @click="quickImportBackup">
              <el-icon><FolderOpened /></el-icon>
              <span>{{ t('autoOps.quickImportBackup') }}</span>
            </button>
            <button type="button" class="action-tile" :disabled="!!applyingPreset" @click="applyBeginnerPreset">
              <el-icon><MagicStick /></el-icon>
              <span>{{ t('autoOps.presetSiteBtn') }}</span>
            </button>
            <button type="button" class="action-tile" @click="tab = 'watch'">
              <el-icon><Monitor /></el-icon>
              <span>{{ t('autoOps.watchList') }}</span>
            </button>
            <button type="button" class="action-tile" @click="tab = 'settings'">
              <el-icon><Tools /></el-icon>
              <span>{{ t('autoOps.settings') }}</span>
            </button>
          </div>
        </section>

        <section class="ao-section">
          <div class="ao-section-head row">
            <div>
              <h3>{{ t('autoOps.hubTitle') }}</h3>
              <p>{{ t('autoOps.hubDesc') }}</p>
            </div>
          </div>
          <div class="hub-filters">
            <button
              v-for="cat in hubCategories"
              :key="cat.key"
              type="button"
              class="hub-filter"
              :class="{ active: hubFilter === cat.key }"
              @click="hubFilter = cat.key"
            >
              <el-icon><component :is="cat.icon" /></el-icon>
              {{ cat.label }}
            </button>
          </div>

          <div v-for="group in capabilityGroups" :key="group.key" class="cap-group">
            <div class="cap-group-head">
              <el-icon><component :is="group.icon" /></el-icon>
              <span>{{ group.label }}</span>
              <small>{{ t('autoOps.hubCount', { n: group.items.length }) }}</small>
            </div>
            <div class="cap-grid">
              <button
                v-for="link in group.items"
                :key="`${link.path}-${link.tab || ''}-${link.title}`"
                type="button"
                class="cap-card"
                @click="openCapability(link)"
              >
                <div class="cap-icon"><el-icon :size="18"><component :is="link.icon" /></el-icon></div>
                <div class="cap-main">
                  <div class="cap-title">{{ link.title }}</div>
                  <div class="cap-desc">{{ link.desc }}</div>
                  <div class="cap-meta">{{ link.audience }}</div>
                </div>
                <div class="cap-side">
                  <span class="cap-stat">{{ link.stat }}</span>
                  <el-icon><ArrowRight /></el-icon>
                </div>
              </button>
            </div>
          </div>
        </section>
      </el-tab-pane>

      <el-tab-pane :label="t('autoOps.watchList')" name="watch">
        <el-alert type="info" :closable="false" show-icon class="watch-hint">
          {{ t('autoOps.watchHint') }}
        </el-alert>
        <div class="toolbar">
          <el-button :disabled="!selectedKeys.length" @click="bulkEnable(false)">
            {{ t('autoOps.enableWatch') }}
          </el-button>
          <el-button type="success" :disabled="!selectedKeys.length" @click="bulkEnable(true)">
            {{ t('autoOps.enableWatchRestart') }}
          </el-button>
        </div>
        <el-table
          :data="watches"
          stripe
          :row-class-name="({ row }: any) => row.key === route.query.key ? 'highlight-row' : ''"
          @selection-change="(rows: any[]) => selectedKeys = rows.map(r => r.key)"
        >
          <el-table-column type="selection" width="48" />
          <el-table-column :label="t('autoOps.software')" min-width="180">
            <template #default="{ row }">
              <div class="table-app-name">
                <SoftwareIcon :app-key="row.key" :size="32" />
                <span>{{ row.name }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column :label="t('autoOps.category')" width="120">
            <template #default="{ row }">{{ categoryLabel(row.category, t) }}</template>
          </el-table-column>
          <el-table-column :label="t('autoOps.liveStatus')" width="100">
            <template #default="{ row }">
              <el-tag :type="liveTagType(row.live_status)" size="small">{{ liveStatusLabel(row.live_status) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column :label="t('autoOps.watchEnabled')" width="100">
            <template #default="{ row }">
              <el-switch :model-value="row.watch_enabled" @change="(v: boolean) => patchWatch(row, 'watch_enabled', v)" />
            </template>
          </el-table-column>
          <el-table-column :label="t('autoOps.autoRestart')" width="100">
            <template #default="{ row }">
              <el-switch :model-value="row.auto_restart" @change="(v: boolean) => patchWatch(row, 'auto_restart', v)" />
            </template>
          </el-table-column>
          <el-table-column :label="t('autoOps.lastEvent')" min-width="160">
            <template #default="{ row }">
              <template v-if="row.last_event">
                <el-tag :type="eventTagType(row.last_event)" size="small">{{ eventLabel(row.last_event) }}</el-tag>
                <span class="event-time">{{ formatTime(row.last_event_at) }}</span>
              </template>
              <span v-else>—</span>
            </template>
          </el-table-column>
        </el-table>
        <el-empty v-if="!watches.length && !loading" :description="t('autoOps.noInstalled')">
          <el-button type="primary" @click="router.push('/software?tab=installed')">{{ t('autoOps.goSoftware') }}</el-button>
        </el-empty>
      </el-tab-pane>

      <el-tab-pane :label="t('autoOps.websiteAuditTab')" name="websites">
        <el-alert type="info" :closable="false" show-icon class="watch-hint">
          {{ t('autoOps.websiteAuditHint') }}
        </el-alert>
        <div class="toolbar">
          <el-button type="primary" :loading="scanningAll" @click="scanAllWebsites">{{ t('autoOps.websiteScanAll') }}</el-button>
          <el-button :icon="Refresh" @click="loadWebsiteAudits">{{ t('common.refresh') }}</el-button>
          <span v-if="websiteAudits?.last_scan" class="last-scan-inline">{{ t('autoOps.lastScan') }}: {{ formatTime(websiteAudits.last_scan) }}</span>
        </div>
        <el-table :data="websiteAuditItems" stripe>
          <el-table-column prop="domain" :label="t('autoOps.websiteDomain')" min-width="160" />
          <el-table-column :label="t('autoOps.websiteScore')" width="100" sortable :sort-method="(a: any, b: any) => a.score - b.score">
            <template #default="{ row }">
              <el-tag :type="gradeTagType(row.grade)" size="small">{{ row.score }} · {{ row.grade }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column :label="t('autoOps.websiteHTTP')" width="80">
            <template #default="{ row }">{{ row.http_status || '—' }}</template>
          </el-table-column>
          <el-table-column :label="t('autoOps.websiteLatency')" width="90">
            <template #default="{ row }">{{ row.latency_ms }}ms</template>
          </el-table-column>
          <el-table-column :label="t('autoOps.websiteIssuesCol')" width="110">
            <template #default="{ row }">
              <span v-if="row.critical" class="issue-critical">{{ row.critical }} {{ t('autoOps.severityCritical') }}</span>
              <span v-if="row.warning" class="issue-warn">{{ row.warning }} {{ t('autoOps.severityWarning') }}</span>
              <span v-if="!row.critical && !row.warning">—</span>
            </template>
          </el-table-column>
          <el-table-column prop="top_issue" :label="t('autoOps.websiteTopIssue')" min-width="180" show-overflow-tooltip />
          <el-table-column :label="t('common.actions')" width="160" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" size="small" @click="openAuditDetail(row)">{{ t('autoOps.websiteViewReport') }}</el-button>
              <el-button link size="small" :loading="scanningSite === row.site_id" @click="scanOneWebsite(row)">{{ t('autoOps.websiteRescan') }}</el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-empty v-if="!websiteAuditItems.length" :description="t('autoOps.websiteAuditEmpty')">
          <el-button type="primary" :loading="scanningAll" @click="scanAllWebsites">{{ t('autoOps.websiteScanAll') }}</el-button>
        </el-empty>
      </el-tab-pane>

      <el-tab-pane :label="t('autoOps.events')" name="events">
        <div class="toolbar">
          <el-select v-model="eventFilterType" clearable :placeholder="t('autoOps.filterEventType')" style="width: 160px" @change="loadEvents">
            <el-option value="down_detected" :label="t('autoOps.eventDown')" />
            <el-option value="restart_ok" :label="t('autoOps.eventRestartOk')" />
            <el-option value="restart_fail" :label="t('autoOps.eventRestartFail')" />
            <el-option value="resource_cpu" :label="t('autoOps.eventResourceCPU')" />
            <el-option value="resource_memory" :label="t('autoOps.eventResourceMem')" />
            <el-option value="resource_disk" :label="t('autoOps.eventResourceDisk')" />
            <el-option value="cron_failed" :label="t('autoOps.eventCronFailed')" />
            <el-option value="ssl_expiring" :label="t('autoOps.eventSSLExpiring')" />
            <el-option value="site_expiring" :label="t('autoOps.eventSiteExpiring')" />
            <el-option value="website_issue" :label="t('autoOps.eventWebsiteIssue')" />
            <el-option value="ssl_renew_fail" :label="t('autoOps.eventSSLRenewFail')" />
          </el-select>
          <el-input v-model="eventFilterApp" clearable :placeholder="t('autoOps.filterAppKey')" style="width: 160px" @keyup.enter="loadEvents" />
          <el-button :icon="Refresh" @click="loadEvents">{{ t('common.refresh') }}</el-button>
        </div>
        <el-table :data="events" stripe max-height="520">
          <el-table-column :label="t('autoOps.time')" width="170">
            <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
          </el-table-column>
          <el-table-column prop="app_name" :label="t('autoOps.software')" width="140" />
          <el-table-column :label="t('autoOps.eventType')" width="130">
            <template #default="{ row }">
              <el-tag :type="eventTagType(row.event_type)" size="small">{{ eventLabel(row.event_type) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="message" :label="t('autoOps.message')" show-overflow-tooltip />
          <el-table-column prop="status" :label="t('autoOps.status')" width="90" />
        </el-table>
      </el-tab-pane>

      <el-tab-pane :label="t('autoOps.settings')" name="settings">
        <el-form label-width="180px" style="max-width: 640px">
          <el-divider content-position="left">{{ t('autoOps.sectionPolicy') }}</el-divider>
          <el-form-item :label="t('autoOps.enabled')">
            <el-switch v-model="configForm.enabled" />
          </el-form-item>
          <el-form-item :label="t('autoOps.intervalSec')">
            <el-input-number v-model="configForm.interval_sec" :min="10" :max="600" />
            <span class="form-hint">{{ t('autoOps.intervalHint') }}</span>
          </el-form-item>
          <el-form-item :label="t('autoOps.cooldownSec')">
            <el-input-number v-model="configForm.cooldown_sec" :min="60" :max="3600" />
            <span class="form-hint">{{ t('autoOps.cooldownHint') }}</span>
          </el-form-item>
          <el-form-item :label="t('autoOps.maxRestarts')">
            <el-input-number v-model="configForm.max_restarts" :min="1" :max="20" />
            <span class="form-hint">{{ t('autoOps.maxRestartsHint') }}</span>
          </el-form-item>

          <el-divider content-position="left">{{ t('autoOps.sectionAlert') }}</el-divider>
          <el-form-item :label="t('autoOps.notifyWebhook')">
            <el-input v-model="configForm.notify_webhook" :placeholder="t('autoOps.notifyWebhookHint')" />
          </el-form-item>
          <el-form-item :label="t('autoOps.notifyOnDown')">
            <el-switch v-model="configForm.notify_on_down" />
          </el-form-item>
          <el-form-item :label="t('autoOps.notifyOnFail')">
            <el-switch v-model="configForm.notify_on_fail" />
          </el-form-item>

          <el-divider content-position="left">{{ t('autoOps.sectionResource') }}</el-divider>
          <el-form-item :label="t('autoOps.resourceEnabled')">
            <el-switch v-model="configForm.resource_enabled" />
          </el-form-item>
          <el-form-item v-if="configForm.resource_enabled" :label="t('autoOps.cpuThreshold')">
            <el-input-number v-model="configForm.cpu_threshold" :min="50" :max="100" />
          </el-form-item>
          <el-form-item v-if="configForm.resource_enabled" :label="t('autoOps.memThreshold')">
            <el-input-number v-model="configForm.mem_threshold" :min="50" :max="100" />
          </el-form-item>
          <el-form-item v-if="configForm.resource_enabled" :label="t('autoOps.diskThreshold')">
            <el-input-number v-model="configForm.disk_threshold" :min="50" :max="100" />
          </el-form-item>
          <el-form-item :label="t('autoOps.memAutoRelief')">
            <el-switch v-model="configForm.mem_auto_relief" />
            <span class="form-hint">{{ t('autoOps.memAutoReliefHint') }}</span>
          </el-form-item>

          <el-divider content-position="left">{{ t('autoOps.sectionExpiry') }}</el-divider>
          <el-form-item :label="t('autoOps.sslAutoRenew')">
            <el-switch v-model="configForm.ssl_auto_renew" />
            <span class="form-hint">{{ t('autoOps.sslAutoRenewHint') }}</span>
          </el-form-item>
          <el-form-item :label="t('autoOps.alertDaysSSL')">
            <el-input-number v-model="configForm.alert_days_ssl" :min="1" :max="90" />
          </el-form-item>
          <el-form-item :label="t('autoOps.alertDaysSite')">
            <el-input-number v-model="configForm.alert_days_site" :min="1" :max="365" />
          </el-form-item>

          <el-divider content-position="left">{{ t('autoOps.sectionWebsite') }}</el-divider>
          <el-form-item :label="t('autoOps.websiteScanEnabled')">
            <el-switch v-model="configForm.website_scan_enabled" />
            <span class="form-hint">{{ t('autoOps.websiteScanEnabledHint') }}</span>
          </el-form-item>

          <el-form-item>
            <el-button type="primary" @click="saveConfig">{{ t('common.save') }}</el-button>
          </el-form-item>
        </el-form>
        <el-alert type="info" :closable="false" show-icon :title="t('autoOps.guideTitle')">
          <template #default>
            <ol class="guide-list">
              <li>{{ t('autoOps.guide1') }}</li>
              <li>{{ t('autoOps.guide2') }}</li>
              <li>{{ t('autoOps.guide3') }}</li>
              <li>{{ t('autoOps.guide4') }}</li>
              <li>{{ t('autoOps.guide5') }}</li>
            </ol>
          </template>
        </el-alert>
      </el-tab-pane>
    </el-tabs>

    <el-drawer v-model="auditDrawer" :title="auditDetail?.domain || t('autoOps.websiteAuditTab')" size="520px">
      <div v-if="auditDetail" class="audit-drawer">
        <div class="audit-head">
          <el-tag :type="gradeTagType(auditDetail.grade)" size="large">{{ auditDetail.score }} / 100 · {{ auditDetail.grade }}</el-tag>
          <span class="audit-meta">{{ auditDetail.url }}</span>
        </div>
        <div class="audit-cats">
          <div v-for="c in auditDetail.categories" :key="c.key" class="audit-cat">
            <span>{{ c.label }}</span>
            <el-tag :type="gradeTagType(c.grade)" size="small">{{ c.score }} · {{ c.grade }}</el-tag>
          </div>
        </div>
        <h4>{{ t('autoOps.websiteFindings') }}</h4>
        <div v-for="(f, i) in auditDetail.findings" :key="i" class="finding-card">
          <div class="finding-head">
            <el-tag :type="severityTagType(f.severity)" size="small">{{ f.severity }}</el-tag>
            <strong>{{ f.title }}</strong>
            <span class="finding-cat">{{ f.category }}</span>
          </div>
          <p class="finding-detail">{{ f.detail }}</p>
          <p v-if="f.fix_hint" class="finding-fix">{{ t('autoOps.websiteFixHint') }}: {{ f.fix_hint }}</p>
        </div>
        <el-empty v-if="!auditDetail.findings?.length" :description="t('autoOps.websiteNoIssues')" />
      </div>
    </el-drawer>

    <el-dialog v-model="cloudTodosVisible" :title="t('autoOps.cloudTodosTitle')" width="520px">
      <p class="cloud-todos-intro">{{ t('autoOps.cloudTodosIntro') }}</p>
      <ol class="cloud-todos-list">
        <li v-for="(item, i) in cloudTodosList" :key="i">{{ item }}</li>
      </ol>
      <template #footer>
        <el-button @click="goPath('/auto-ops', 'settings')">{{ t('autoOps.cloudTodosWebhook') }}</el-button>
        <el-button type="primary" @click="cloudTodosVisible = false">{{ t('common.confirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.ao-hero {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
  margin-bottom: 16px;
  padding: 20px 22px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 18px;
  background:
    radial-gradient(900px 160px at 0% 0%, color-mix(in srgb, var(--el-color-primary) 16%, transparent), transparent 60%),
    linear-gradient(180deg, var(--el-fill-color-blank, #fff), var(--el-fill-color-lighter));
}
.ao-hero-main { display: flex; gap: 14px; align-items: center; }
.ao-hero-badge {
  width: 48px; height: 48px; border-radius: 14px; display: grid; place-items: center; color: #fff;
  background: linear-gradient(145deg, var(--el-color-primary-light-3), var(--el-color-primary));
  box-shadow: 0 8px 18px color-mix(in srgb, var(--el-color-primary) 30%, transparent);
}
.ao-title { margin: 0; font-size: 24px; font-weight: 740; letter-spacing: -0.02em; }
.ao-sub { margin: 4px 0 0; font-size: 13px; color: var(--el-text-color-secondary); max-width: 520px; line-height: 1.5; }
.ao-meta { margin: 4px 0 0; font-size: 12px; color: var(--el-text-color-secondary); }
.ao-hero-actions { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; }
.ao-status-pills { display: flex; flex-wrap: wrap; gap: 6px; margin-right: 4px; }
.ao-pill {
  display: inline-flex; align-items: center; gap: 4px;
  padding: 4px 10px; border-radius: 999px; font-size: 12px; font-weight: 600;
  background: var(--el-fill-color); color: var(--el-text-color-regular);
}
.ao-pill.ok { background: #ecfdf5; color: #059669; }
.ao-pill.off { background: var(--el-fill-color); color: var(--el-text-color-secondary); }
.ao-pill.danger { background: #fef2f2; color: #dc2626; }

.ao-banner {
  display: flex; justify-content: space-between; gap: 16px; flex-wrap: wrap; align-items: center;
  margin-bottom: 16px; padding: 14px 16px; border-radius: 14px;
  border: 1px solid color-mix(in srgb, var(--el-color-primary) 28%, var(--el-border-color-lighter));
  background: color-mix(in srgb, var(--el-color-primary) 8%, var(--el-bg-color));
}
.ao-banner strong { display: block; margin-bottom: 4px; }
.ao-banner p { margin: 0; font-size: 13px; color: var(--el-text-color-secondary); line-height: 1.5; }
.ao-banner-actions { display: flex; gap: 8px; flex-wrap: wrap; }

.ao-tabs :deep(.el-tabs__header) { margin-bottom: 18px; }
.ao-section { margin-bottom: 22px; }
.ao-section-head { margin-bottom: 12px; }
.ao-section-head h3 { margin: 0 0 4px; font-size: 16px; font-weight: 700; }
.ao-section-head p { margin: 0; font-size: 13px; color: var(--el-text-color-secondary); line-height: 1.5; }
.ao-section-head.row { display: flex; justify-content: space-between; gap: 12px; align-items: flex-end; }

.scenario-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 12px;
}
.scenario-card {
  display: flex; flex-direction: column; gap: 12px;
  padding: 14px; border-radius: 14px; border: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color); cursor: pointer;
  transition: transform .15s ease, box-shadow .15s ease, border-color .15s ease;
}
.scenario-card:hover { transform: translateY(-2px); box-shadow: 0 10px 22px rgba(15,23,42,.06); }
.scenario-card.active { border-color: color-mix(in srgb, var(--el-color-primary) 45%, var(--el-border-color-lighter)); }
.scenario-icon {
  width: 40px; height: 40px; border-radius: 12px; display: grid; place-items: center;
}
.tone-site .scenario-icon { background: #eff6ff; color: #2563eb; }
.tone-dev .scenario-icon { background: #fff7ed; color: #ea580c; }
.tone-test .scenario-icon { background: #f5f3ff; color: #7c3aed; }
.tone-storage .scenario-icon { background: #ecfdf5; color: #059669; }
.tone-security .scenario-icon { background: #fef2f2; color: #dc2626; }
.scenario-body h4 { margin: 0 0 4px; font-size: 14px; font-weight: 700; }
.scenario-body p { margin: 0; font-size: 12px; color: var(--el-text-color-secondary); line-height: 1.45; min-height: 34px; }

.health-board {
  display: grid; grid-template-columns: minmax(280px, 360px) 1fr; gap: 14px;
  padding: 16px; border-radius: 16px; border: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);
}
.gauge-row { display: grid; grid-template-columns: repeat(3, 1fr); gap: 8px; }
.gauge-card { text-align: center; }
.chip-row { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 10px; align-content: start; }
.health-chip {
  border: 1px solid var(--el-border-color-lighter); border-radius: 12px; background: var(--el-fill-color-lighter);
  padding: 12px 14px; text-align: left; cursor: pointer; transition: border-color .15s ease;
}
.health-chip:hover { border-color: var(--el-color-primary-light-5); }
.health-chip.warn { border-color: color-mix(in srgb, #dc2626 35%, var(--el-border-color-lighter)); background: #fef2f2; }
.health-chip-label { display: block; font-size: 12px; color: var(--el-text-color-secondary); margin-bottom: 4px; }
.health-chip strong { font-size: 20px; letter-spacing: -0.02em; }

.action-grid { display: grid; grid-template-columns: repeat(6, minmax(0, 1fr)); gap: 10px; }
.action-tile {
  display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 8px;
  min-height: 88px; padding: 12px; border-radius: 14px; border: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color); cursor: pointer; font-size: 12px; font-weight: 650; color: var(--el-text-color-primary);
  transition: transform .15s ease, border-color .15s ease, box-shadow .15s ease;
}
.action-tile .el-icon { font-size: 20px; color: var(--el-color-primary); }
.action-tile:hover:not(:disabled) {
  transform: translateY(-1px); border-color: var(--el-color-primary-light-5);
  box-shadow: 0 8px 18px rgba(15,23,42,.05);
}
.action-tile:disabled { opacity: .55; cursor: not-allowed; }

.hub-filters { display: flex; flex-wrap: wrap; gap: 8px; margin-bottom: 14px; }
.hub-filter {
  display: inline-flex; align-items: center; gap: 6px;
  border: 1px solid var(--el-border-color-lighter); background: var(--el-bg-color);
  border-radius: 999px; padding: 7px 12px; font-size: 12px; font-weight: 650; cursor: pointer;
  color: var(--el-text-color-secondary);
}
.hub-filter.active, .hub-filter:hover {
  color: var(--el-color-primary);
  border-color: color-mix(in srgb, var(--el-color-primary) 40%, var(--el-border-color-lighter));
  background: color-mix(in srgb, var(--el-color-primary) 8%, var(--el-bg-color));
}
.cap-group { margin-bottom: 16px; }
.cap-group-head {
  display: flex; align-items: center; gap: 8px; margin-bottom: 10px;
  font-weight: 700; font-size: 14px;
}
.cap-group-head small { margin-left: auto; font-weight: 500; color: var(--el-text-color-secondary); }
.cap-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 10px; }
.cap-card {
  display: flex; align-items: center; gap: 12px; text-align: left;
  padding: 14px; border-radius: 14px; border: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color); cursor: pointer; transition: border-color .15s ease, transform .15s ease;
}
.cap-card:hover { border-color: var(--el-color-primary-light-5); transform: translateY(-1px); }
.cap-icon {
  width: 38px; height: 38px; border-radius: 11px; display: grid; place-items: center; flex-shrink: 0;
  background: var(--el-fill-color-lighter); color: var(--el-color-primary);
}
.cap-main { min-width: 0; flex: 1; }
.cap-title { font-size: 14px; font-weight: 700; }
.cap-desc { margin-top: 2px; font-size: 12px; color: var(--el-text-color-secondary); line-height: 1.4; }
.cap-meta { margin-top: 4px; font-size: 11px; color: var(--el-color-primary); }
.cap-side { display: flex; flex-direction: column; align-items: flex-end; gap: 6px; color: var(--el-text-color-secondary); }
.cap-stat { font-size: 12px; font-weight: 650; }

@media (max-width: 1200px) {
  .scenario-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); }
  .action-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); }
  .health-board { grid-template-columns: 1fr; }
}
@media (max-width: 720px) {
  .scenario-grid, .action-grid, .chip-row, .gauge-row { grid-template-columns: 1fr; }
}

.guide-intro { margin-bottom: 20px; }
.guide-intro-text { margin: 8px 0 0; line-height: 1.7; font-size: 13px; }
.section-title { margin: 24px 0 8px; font-size: 16px; font-weight: 600; }
.section-desc { margin: 0 0 12px; font-size: 13px; color: var(--el-text-color-secondary); line-height: 1.6; }
.preset-row { display: grid; grid-template-columns: repeat(auto-fit, minmax(280px, 1fr)); gap: 12px; margin-bottom: 8px; }
.preset-card { border: 1px solid var(--el-border-color-lighter); }
.preset-head { font-weight: 600; margin-bottom: 8px; }
.preset-body { margin: 0 0 10px; font-size: 13px; color: var(--el-text-color-regular); line-height: 1.6; }
.preset-list { margin: 0 0 14px; padding-left: 18px; font-size: 13px; line-height: 1.7; color: var(--el-text-color-secondary); }
.path-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(260px, 1fr)); gap: 12px; }
.path-card { border: 1px solid var(--el-border-color-lighter); }
.path-head { display: flex; gap: 10px; margin-bottom: 10px; }
.path-icon { font-size: 28px; line-height: 1; }
.path-title { font-weight: 600; margin-bottom: 4px; }
.path-desc { font-size: 12px; color: var(--el-text-color-secondary); line-height: 1.5; }
.path-steps { margin: 0; padding-left: 18px; }
.path-step-link {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 0;
  border: none;
  background: none;
  color: var(--el-color-primary);
  font-size: 13px;
  cursor: pointer;
  line-height: 1.8;
}
.path-step-link:hover { text-decoration: underline; }
.cmp-wrap { overflow-x: auto; margin-bottom: 8px; }
.cmp-table { min-width: 760px; }
.cmp-partial { color: var(--el-color-warning); font-weight: 600; }
.cmp-legend { margin: 0 0 16px; font-size: 12px; color: var(--el-text-color-secondary); }
.cmp-no { color: var(--el-text-color-placeholder); }
.guide-doc { margin-bottom: 16px; }
.glossary-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); gap: 10px; margin-bottom: 16px; }
.glossary-item { padding: 12px; border-radius: 8px; background: var(--el-fill-color-light); }
.glossary-item strong { display: block; margin-bottom: 6px; font-size: 13px; }
.glossary-item p { margin: 0; font-size: 12px; color: var(--el-text-color-secondary); line-height: 1.6; }
.guide-faq { margin-top: 20px; }
.faq-list { margin: 8px 0 0; }
.faq-list dt { font-weight: 600; font-size: 13px; margin-top: 10px; }
.faq-list dd { margin: 4px 0 0; font-size: 13px; color: var(--el-text-color-secondary); line-height: 1.6; }
.link-title-row { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }
.link-audience { margin-top: 4px; font-size: 11px; color: var(--el-color-primary); }
.overview-quick { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; margin-bottom: 12px; }
.overview-quick-hint { font-size: 12px; color: var(--el-text-color-secondary); }
.auto-ops-page { width: 100%; }
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}
.page-header h2 { margin: 0 0 4px; }
.hint { margin: 0; font-size: 13px; color: var(--el-text-color-secondary); }
.last-scan { margin: 4px 0 0; font-size: 12px; color: var(--el-text-color-secondary); }
.header-stats { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; }
.toolbar { margin-bottom: 12px; display: flex; gap: 8px; flex-wrap: wrap; }
.watch-hint, .overview-hint { margin-bottom: 12px; }
.event-time { margin-left: 8px; font-size: 12px; color: var(--el-text-color-secondary); }
.form-hint { margin-left: 12px; font-size: 12px; color: var(--el-text-color-secondary); }
.guide-list { margin: 8px 0 0; padding-left: 18px; line-height: 1.7; }
.last-scan-inline { font-size: 12px; color: var(--el-text-color-secondary); align-self: center; }
.issue-critical { color: var(--el-color-danger); margin-right: 8px; font-size: 12px; }
.issue-warn { color: var(--el-color-warning); font-size: 12px; }
.audit-drawer { display: flex; flex-direction: column; gap: 12px; }
.audit-head { display: flex; flex-direction: column; gap: 6px; }
.audit-meta { font-size: 12px; color: var(--el-text-color-secondary); word-break: break-all; }
.audit-cats { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 8px; }
.audit-cat { display: flex; justify-content: space-between; align-items: center; padding: 8px 10px; border-radius: 8px; background: var(--el-fill-color-light); font-size: 12px; }
.finding-card { border: 1px solid var(--el-border-color-lighter); border-radius: 8px; padding: 10px 12px; margin-bottom: 8px; }
.finding-head { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; margin-bottom: 6px; }
.finding-cat { margin-left: auto; font-size: 11px; color: var(--el-text-color-secondary); text-transform: uppercase; }
.finding-detail { margin: 0; font-size: 13px; color: var(--el-text-color-regular); }
.finding-fix { margin: 6px 0 0; font-size: 12px; color: var(--el-color-primary); }
.table-app-name { display: flex; align-items: center; gap: 8px; }
:deep(.highlight-row) { background-color: var(--el-color-primary-light-9) !important; }
.overview-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 12px;
  margin-bottom: 20px;
}
.resource-row {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
  grid-column: 1 / -1;
}
@media (max-width: 720px) {
  .resource-row { grid-template-columns: 1fr; }
}
.stat-card { text-align: center; }
.stat-card-resource { padding: 12px 8px 14px; }
.resource-gauge {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}
.gauge-val {
  font-size: 20px;
  font-weight: 700;
  line-height: 1;
  color: var(--el-text-color-primary);
}
.resource-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.resource-label .el-icon {
  font-size: 16px;
  color: var(--el-color-primary);
}
.stat-icon {
  font-size: 22px;
  color: var(--el-color-primary);
  margin-bottom: 8px;
}
.stat-card.clickable { cursor: pointer; }
.stat-card.clickable:hover { border-color: var(--el-color-primary); }
.stat-label { font-size: 13px; color: var(--el-text-color-secondary); margin-bottom: 8px; }
.stat-val { margin-top: 6px; font-weight: 600; }
.stat-big { font-size: 28px; font-weight: 700; margin: 8px 0; }
.stat-warn { font-size: 12px; color: var(--el-color-danger); }
.section-title { margin: 0 0 12px; font-size: 15px; }
.link-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(260px, 1fr)); gap: 10px; margin-bottom: 16px; }
.link-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 10px;
  background: var(--el-bg-color);
  cursor: pointer;
  text-align: left;
  transition: border-color 0.15s, box-shadow 0.15s;
}
.link-card:hover { border-color: var(--el-color-primary); box-shadow: 0 2px 8px rgba(0,0,0,0.06); }
.link-icon { font-size: 22px; color: var(--el-color-primary); }
.link-body { flex: 1; min-width: 0; }
.link-title { font-weight: 600; font-size: 14px; }
.link-desc { font-size: 12px; color: var(--el-text-color-secondary); margin-top: 2px; }
.link-stat { font-size: 12px; color: var(--el-text-color-secondary); }
.cloud-summary { display: flex; flex-wrap: wrap; gap: 8px; margin-bottom: 16px; }
.cloud-int-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); gap: 12px; margin-bottom: 20px; }
.cloud-int-card { cursor: pointer; border: 1px solid var(--el-border-color-lighter); transition: border-color 0.15s; }
.cloud-int-card:hover { border-color: var(--el-color-primary-light-5); }
.cloud-int-head { display: flex; justify-content: space-between; align-items: center; gap: 8px; font-weight: 600; margin-bottom: 6px; }
.cloud-int-desc { margin: 0 0 8px; font-size: 12px; color: var(--el-text-color-secondary); line-height: 1.5; min-height: 36px; }
.cloud-int-foot { font-size: 12px; color: var(--el-color-primary); }
.cloud-vendor-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 12px; margin-bottom: 16px; }
.cloud-vendor-card { border: 1px solid var(--el-border-color-lighter); }
.cloud-vendor-tags { display: flex; flex-wrap: wrap; gap: 6px; margin-bottom: 8px; }
.cloud-storage-pick { margin: 8px 0; }
.cloud-vendor-actions { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 8px; }
.cloud-setup-hint { margin: 8px 0; }
.cloud-todos-intro { margin: 0 0 12px; font-size: 13px; color: var(--el-text-color-secondary); }
.cloud-todos-list { margin: 0; padding-left: 20px; line-height: 1.8; font-size: 13px; }
</style>
