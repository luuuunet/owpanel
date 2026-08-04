<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import api from '@/api'
import { apiContentLang } from '@/locales'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  CopyDocument, Delete, Edit, Plus, RefreshRight, VideoPlay,
  Monitor, Cpu, Odometer, Coin, FolderOpened, Connection,
  Tools, CircleCheck, WarningFilled, InfoFilled,
} from '@element-plus/icons-vue'

const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()

const activeTab = ref('system')
const loading = ref(false)
const benchLoading = ref<string | null>(null)
const benchResults = ref<Record<string, any>>({})
const runningAll = ref(false)

const host = ref('baidu.com')
const netOutput = ref('')
const netKind = ref('')

const overview = ref<any>(null)
const processes = ref<any[]>([])
const killingPid = ref<number | null>(null)
const droppingCache = ref(false)

const ports = ref<any[]>([])
const portFilter = ref('')

const health = ref<any>(null)

const snippets = ref<any[]>([])
const snippetFilter = ref('')
const runOutput = ref('')
const runLoading = ref(false)
const snippetDialog = ref(false)
const snippetForm = ref({ id: 0, name: '', command: '', category: 'custom', remark: '' })

const tabs = computed(() => [
  { key: 'system', label: t('toolboxPage.tabSystem') },
  { key: 'bench', label: t('toolboxPage.tabBench') },
  { key: 'ports', label: t('toolboxPage.tabPorts') },
  { key: 'snippets', label: t('toolboxPage.tabSnippets') },
  { key: 'network', label: t('toolboxPage.tabNetwork') },
])

const filteredPorts = computed(() => {
  const q = portFilter.value.trim().toLowerCase()
  if (!q) return ports.value
  return ports.value.filter((p) =>
    String(p.port).includes(q) ||
    (p.process || '').toLowerCase().includes(q) ||
    (p.protocol || '').toLowerCase().includes(q),
  )
})

const filteredSnippets = computed(() => {
  const q = snippetFilter.value.trim().toLowerCase()
  if (!q) return snippets.value
  return snippets.value.filter((s) =>
    (s.name || '').toLowerCase().includes(q) ||
    (s.command || '').toLowerCase().includes(q) ||
    (s.category || '').toLowerCase().includes(q),
  )
})

const healthColor = computed(() => {
  const s = health.value?.score ?? 0
  if (s >= 80) return '#059669'
  if (s >= 60) return '#d97706'
  return '#dc2626'
})

const healthTone = computed(() => {
  const s = health.value?.score ?? 0
  if (s >= 80) return 'ok'
  if (s >= 60) return 'warn'
  return 'bad'
})

const memPct = computed(() => Math.round(overview.value?.memory?.used_percent ?? 0))

const benchCards = computed(() => [
  { kind: 'cpu', icon: Cpu, tone: 'cpu', title: t('toolboxPage.benchCPU'), desc: t('toolboxPage.benchCPUDesc') },
  { kind: 'memory', icon: Coin, tone: 'mem', title: t('toolboxPage.benchMemory'), desc: t('toolboxPage.benchMemoryDesc') },
  { kind: 'disk', icon: FolderOpened, tone: 'disk', title: t('toolboxPage.benchDisk'), desc: t('toolboxPage.benchDiskDesc') },
  { kind: 'network', icon: Connection, tone: 'net', title: t('toolboxPage.benchNetwork'), desc: t('toolboxPage.benchNetworkDesc') },
])

function formatBytes(n: number) {
  if (!n) return '0 B'
  const u = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.min(Math.floor(Math.log(n) / Math.log(1024)), u.length - 1)
  return `${(n / 1024 ** i).toFixed(1)} ${u[i]}`
}

function factorIcon(status: string) {
  if (status === 'ok') return CircleCheck
  if (status === 'warn') return WarningFilled
  return InfoFilled
}

async function runNet(type: 'ping' | 'traceroute' | 'dns') {
  loading.value = true
  netOutput.value = ''
  netKind.value = type
  try {
    const path = type === 'ping' ? '/toolbox/ping' : type === 'traceroute' ? '/toolbox/traceroute' : '/toolbox/dns'
    const body = type === 'dns' ? { domain: host.value } : { host: host.value }
    const res: any = await api.post(path, body)
    netOutput.value = res.data?.output || JSON.stringify(res.data, null, 2)
  } finally {
    loading.value = false
  }
}

async function loadSystem() {
  loading.value = true
  try {
    const [ov, procs, hp] = await Promise.all([
      api.get('/toolbox/system/overview'),
      api.get('/toolbox/system/processes', { params: { limit: 15 } }),
      api.get('/toolbox/health', { params: { lang: apiContentLang(locale.value) } }),
    ])
    overview.value = ov.data
    processes.value = procs.data || []
    health.value = hp.data
  } catch (e: any) {
    ElMessage.error(e?.error || e?.message || t('common.failed'))
  } finally {
    loading.value = false
  }
}

async function loadPorts() {
  loading.value = true
  try {
    const res: any = await api.get('/toolbox/system/ports')
    ports.value = res.data || []
  } catch (e: any) {
    ElMessage.error(e?.error || e?.message || t('common.failed'))
  } finally {
    loading.value = false
  }
}

async function loadSnippets() {
  try {
    const res: any = await api.get('/toolbox/snippets', { params: { lang: apiContentLang(locale.value) } })
    snippets.value = res.data || []
  } catch (e: any) {
    ElMessage.error(e?.error || e?.message || t('common.failed'))
  }
}

async function killProcess(row: { pid: number; name?: string; command?: string }) {
  try {
    await ElMessageBox.confirm(
      t('toolboxPage.killProcessConfirm', { name: row.name || row.pid, pid: row.pid }),
      t('toolboxPage.killProcess'),
      { type: 'warning', confirmButtonText: t('toolboxPage.killProcess'), cancelButtonText: t('common.cancel') },
    )
  } catch {
    return
  }
  killingPid.value = row.pid
  try {
    await api.post(`/toolbox/system/processes/${row.pid}/kill`)
    ElMessage.success(t('toolboxPage.killProcessSuccess'))
    await loadSystem()
  } catch (e: any) {
    ElMessage.error(e?.error || e?.message || t('toolboxPage.killProcessFailed'))
  } finally {
    killingPid.value = null
  }
}

async function dropCache() {
  await ElMessageBox.confirm(t('toolboxPage.dropCacheConfirm'), t('common.confirm'), { type: 'warning' })
  droppingCache.value = true
  try {
    const res: any = await api.post('/toolbox/system/drop-cache')
    ElMessage.success(res.data?.message || t('toolboxPage.dropCacheDone'))
    await loadSystem()
  } catch (e: any) {
    ElMessage.error(e?.error || e?.message || t('common.failed'))
  } finally {
    droppingCache.value = false
  }
}

async function allowPort(row: any) {
  try {
    await api.post('/firewall', {
      port: row.port,
      protocol: row.protocol,
      action: 'allow',
      remark: `toolbox: ${row.process || row.port}`,
    })
    ElMessage.success(t('toolboxPage.firewallAdded'))
    await loadPorts()
  } catch (e: any) {
    ElMessage.error(e?.error || e?.message || t('common.failed'))
  }
}

async function runSnippet(item: any) {
  runLoading.value = true
  runOutput.value = ''
  try {
    const res: any = await api.post('/toolbox/snippets/run', { id: item.id })
    runOutput.value = res.data?.output || ''
    if (res.data?.exit_code > 0) {
      ElMessage.warning(t('toolboxPage.runExitCode', { code: res.data.exit_code }))
    }
  } catch (e: any) {
    ElMessage.error(e?.error || e?.message || t('common.failed'))
  } finally {
    runLoading.value = false
  }
}

function copyText(text: string) {
  navigator.clipboard.writeText(text)
  ElMessage.success(t('common.success'))
}

function openSnippetDialog(item?: any) {
  if (item?.builtin) return
  if (item && !item.builtin) {
    const id = parseInt(String(item.id).replace('user:', ''), 10)
    snippetForm.value = { id, name: item.name, command: item.command, category: item.category || 'custom', remark: item.remark || '' }
  } else {
    snippetForm.value = { id: 0, name: '', command: '', category: 'custom', remark: '' }
  }
  snippetDialog.value = true
}

async function saveSnippet() {
  try {
    if (snippetForm.value.id) {
      await api.put(`/toolbox/snippets/${snippetForm.value.id}`, snippetForm.value)
    } else {
      await api.post('/toolbox/snippets', snippetForm.value)
    }
    snippetDialog.value = false
    ElMessage.success(t('common.saved'))
    await loadSnippets()
  } catch (e: any) {
    ElMessage.error(e?.error || e?.message || t('common.failed'))
  }
}

async function deleteSnippet(item: any) {
  const id = parseInt(String(item.id).replace('user:', ''), 10)
  await ElMessageBox.confirm(t('toolboxPage.deleteSnippetConfirm'), t('common.confirm'), { type: 'warning' })
  await api.delete(`/toolbox/snippets/${id}`)
  ElMessage.success(t('common.deleted'))
  await loadSnippets()
}

async function runBench(kind: string) {
  if (benchLoading.value) {
    ElMessage.warning(t('toolboxPage.benchBusy'))
    return
  }
  benchLoading.value = kind
  try {
    const res: any = await api.post(`/toolbox/bench/${kind}`, {}, { timeout: 120000 })
    benchResults.value = { ...benchResults.value, [kind]: res.data }
  } catch (e: any) {
    ElMessage.error(e?.error || e?.message || t('common.failed'))
  } finally {
    benchLoading.value = null
  }
}

async function runAllBench() {
  if (benchLoading.value || runningAll.value) {
    ElMessage.warning(t('toolboxPage.benchBusy'))
    return
  }
  runningAll.value = true
  try {
    for (const card of benchCards.value) {
      benchLoading.value = card.kind
      try {
        const res: any = await api.post(`/toolbox/bench/${card.kind}`, {}, { timeout: 120000 })
        benchResults.value = { ...benchResults.value, [card.kind]: res.data }
      } catch (e: any) {
        ElMessage.error(`${card.title}: ${e?.error || e?.message || t('common.failed')}`)
        break
      }
    }
  } finally {
    benchLoading.value = null
    runningAll.value = false
  }
}

function selectTab(name: string) {
  activeTab.value = name
  if (route.query.tab !== name) {
    router.replace({ query: { ...route.query, tab: name } })
  }
  if (name === 'system') loadSystem()
  else if (name === 'ports') loadPorts()
  else if (name === 'snippets') {
    const q = String(route.query.q || '').trim()
    if (q) snippetFilter.value = q
    loadSnippets()
  }
}

onMounted(() => {
  const tab = String(route.query.tab || '')
  if (['system', 'bench', 'ports', 'snippets', 'network'].includes(tab)) {
    activeTab.value = tab
  }
  const q = String(route.query.q || '').trim()
  if (q) snippetFilter.value = q
  selectTab(activeTab.value)
})
</script>

<template>
  <div class="tb-page">
    <header class="tb-hero">
      <div class="tb-hero-main">
        <div class="tb-hero-badge">
          <el-icon :size="22"><Tools /></el-icon>
        </div>
        <div>
          <h1 class="tb-title">{{ t('toolboxPage.title') }}</h1>
          <p class="tb-sub">{{ t('toolboxPage.subtitle') }}</p>
        </div>
      </div>
      <nav class="tb-tabs" role="tablist">
        <button
          v-for="tab in tabs"
          :key="tab.key"
          type="button"
          class="tb-tab"
          :class="{ active: activeTab === tab.key }"
          role="tab"
          :aria-selected="activeTab === tab.key"
          @click="selectTab(tab.key)"
        >
          {{ tab.label }}
        </button>
      </nav>
    </header>

    <!-- System -->
    <section v-show="activeTab === 'system'" v-loading="loading" class="tb-section">
      <div class="metric-row">
        <article class="metric-tile tone-uptime">
          <div class="metric-icon"><el-icon :size="18"><Odometer /></el-icon></div>
          <div class="metric-body">
            <div class="metric-label">{{ t('toolboxPage.uptime') }}</div>
            <div class="metric-value">{{ overview?.uptime_human || '—' }}</div>
            <div class="metric-foot">{{ overview?.hostname || '—' }}</div>
          </div>
        </article>
        <article class="metric-tile tone-mem">
          <div class="metric-icon"><el-icon :size="18"><Monitor /></el-icon></div>
          <div class="metric-body">
            <div class="metric-label">{{ t('toolboxPage.memory') }}</div>
            <div class="metric-value">{{ memPct || '—' }}<span v-if="overview" class="metric-unit">%</span></div>
            <div class="metric-foot">
              {{ formatBytes(overview?.memory?.used) }} / {{ formatBytes(overview?.memory?.total) }}
            </div>
            <div class="metric-bar"><i :style="{ width: `${Math.min(100, memPct)}%` }" /></div>
          </div>
        </article>
        <article class="metric-tile tone-cpu">
          <div class="metric-icon"><el-icon :size="18"><Cpu /></el-icon></div>
          <div class="metric-body">
            <div class="metric-label">{{ t('toolboxPage.load') }}</div>
            <div class="metric-value">{{ overview?.load1?.toFixed?.(2) ?? '—' }}</div>
            <div class="metric-foot">{{ overview?.cpu_count ?? '—' }} {{ t('toolboxPage.cores') }}</div>
          </div>
        </article>
      </div>

      <div class="sys-grid">
        <aside class="panel health-panel" :class="`health-${healthTone}`">
          <div class="panel-head">
            <span>{{ t('toolboxPage.healthScore') }}</span>
            <el-button link :icon="RefreshRight" @click="loadSystem" />
          </div>
          <div v-if="health" class="health-body">
            <el-progress type="dashboard" :percentage="health.score" :color="healthColor" :width="128" :stroke-width="10">
              <template #default>
                <span class="health-num">{{ health.score }}</span>
                <span class="health-grade">{{ health.grade }}</span>
              </template>
            </el-progress>
            <p class="health-summary">{{ health.summary }}</p>
            <div class="factor-list">
              <div v-for="f in health.factors" :key="f.key" class="factor-row" :class="f.status">
                <el-icon :size="14"><component :is="factorIcon(f.status)" /></el-icon>
                <span class="factor-label">{{ f.label }}</span>
                <span class="factor-detail">{{ f.detail }}</span>
              </div>
            </div>
          </div>
          <div class="panel-actions">
            <el-button type="warning" plain :loading="droppingCache" @click="dropCache">
              {{ t('toolboxPage.dropCache') }}
            </el-button>
          </div>
        </aside>

        <div class="sys-main">
          <div class="panel">
            <div class="panel-head">{{ t('toolboxPage.diskUsage') }}</div>
            <el-table :data="overview?.disks || []" size="small" class="tb-table" empty-text="—">
              <el-table-column prop="mount" :label="t('toolboxPage.mount')" min-width="110" />
              <el-table-column :label="t('toolboxPage.used')" width="140">
                <template #default="{ row }">
                  <div class="disk-cell">
                    <span>{{ row.used_percent?.toFixed(1) }}%</span>
                    <div class="disk-bar"><i :style="{ width: `${Math.min(100, row.used_percent || 0)}%` }" /></div>
                  </div>
                </template>
              </el-table-column>
              <el-table-column :label="t('toolboxPage.size')" min-width="160">
                <template #default="{ row }">{{ formatBytes(row.used) }} / {{ formatBytes(row.total) }}</template>
              </el-table-column>
            </el-table>
          </div>

          <div class="panel">
            <div class="panel-head">{{ t('toolboxPage.topProcesses') }}</div>
            <el-table :data="processes" size="small" class="tb-table" max-height="320" empty-text="—">
              <el-table-column prop="pid" label="PID" width="70" />
              <el-table-column prop="name" :label="t('toolboxPage.process')" width="120" show-overflow-tooltip />
              <el-table-column prop="user" :label="t('toolboxPage.user')" width="90" show-overflow-tooltip />
              <el-table-column :label="t('toolboxPage.memCol')" width="80">
                <template #default="{ row }">{{ row.memory?.toFixed(1) }}%</template>
              </el-table-column>
              <el-table-column prop="command" :label="t('toolboxPage.command')" show-overflow-tooltip />
              <el-table-column :label="t('common.actions')" width="100" fixed="right">
                <template #default="{ row }">
                  <el-button
                    link
                    type="danger"
                    size="small"
                    :loading="killingPid === row.pid"
                    :disabled="row.pid <= 1"
                    @click="killProcess(row)"
                  >
                    {{ t('toolboxPage.killProcess') }}
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </div>
      </div>
    </section>

    <!-- Bench -->
    <section v-show="activeTab === 'bench'" class="tb-section">
      <div class="bench-banner">
        <div>
          <h3>{{ t('toolboxPage.tabBench') }}</h3>
          <p>{{ t('toolboxPage.benchHint') }}</p>
        </div>
        <el-button type="primary" :loading="runningAll" :disabled="!!benchLoading && !runningAll" @click="runAllBench">
          {{ t('toolboxPage.benchRunAll') }}
        </el-button>
      </div>

      <div class="bench-grid">
        <article
          v-for="card in benchCards"
          :key="card.kind"
          class="bench-card"
          :class="[`tone-${card.tone}`, { running: benchLoading === card.kind, done: !!benchResults[card.kind] }]"
        >
          <div class="bench-top">
            <div class="bench-icon"><el-icon :size="20"><component :is="card.icon" /></el-icon></div>
            <div>
              <div class="bench-title">{{ card.title }}</div>
              <div class="bench-desc">{{ card.desc }}</div>
            </div>
          </div>

          <div class="bench-stage">
            <template v-if="benchResults[card.kind]">
              <div class="bench-score-line">
                <span class="bench-score">{{ benchResults[card.kind].score }}</span>
                <span class="bench-unit">{{ benchResults[card.kind].unit }}</span>
              </div>
              <div class="bench-detail">{{ benchResults[card.kind].detail }}</div>
              <div class="bench-meta">{{ t('toolboxPage.benchDuration', { ms: benchResults[card.kind].duration_ms }) }}</div>
            </template>
            <template v-else-if="benchLoading === card.kind">
              <div class="bench-pulse" />
              <div class="bench-idle-title">{{ t('toolboxPage.benchRunning') }}</div>
            </template>
            <template v-else>
              <div class="bench-idle-mark">—</div>
              <div class="bench-idle-title">{{ t('toolboxPage.benchIdle') }}</div>
              <div class="bench-idle-hint">{{ t('toolboxPage.benchIdleHint') }}</div>
            </template>
          </div>

          <button
            type="button"
            class="bench-btn"
            :disabled="!!benchLoading && benchLoading !== card.kind"
            @click="runBench(card.kind)"
          >
            <span v-if="benchLoading === card.kind" class="bench-spinner" />
            {{ benchLoading === card.kind ? t('toolboxPage.benchRunning') : t('toolboxPage.benchRun') }}
          </button>
        </article>
      </div>
    </section>

    <!-- Ports -->
    <section v-show="activeTab === 'ports'" v-loading="loading" class="tb-section">
      <div class="panel">
        <div class="panel-head with-tools">
          <span>{{ t('toolboxPage.tabPorts') }}</span>
          <div class="panel-tools">
            <el-input v-model="portFilter" :placeholder="t('toolboxPage.portFilter')" clearable class="tool-input" />
            <el-button :icon="RefreshRight" @click="loadPorts">{{ t('common.refresh') }}</el-button>
          </div>
        </div>
        <el-table :data="filteredPorts" stripe class="tb-table">
          <el-table-column prop="port" :label="t('toolboxPage.port')" width="80" sortable />
          <el-table-column prop="protocol" label="Proto" width="70" />
          <el-table-column prop="address" :label="t('toolboxPage.bindAddr')" width="130" />
          <el-table-column prop="process" :label="t('toolboxPage.process')" width="140" show-overflow-tooltip />
          <el-table-column prop="pid" label="PID" width="70" />
          <el-table-column prop="user" :label="t('toolboxPage.user')" width="100" show-overflow-tooltip />
          <el-table-column prop="command" :label="t('toolboxPage.command')" show-overflow-tooltip />
          <el-table-column :label="t('toolboxPage.firewall')" width="120">
            <template #default="{ row }">
              <el-tag v-if="row.firewalled" type="success" size="small" effect="light" round>{{ t('toolboxPage.fwAllowed') }}</el-tag>
              <el-button v-else link type="primary" size="small" @click="allowPort(row)">{{ t('toolboxPage.fwAllow') }}</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </section>

    <!-- Snippets -->
    <section v-show="activeTab === 'snippets'" class="tb-section">
      <div class="panel">
        <div class="panel-head with-tools">
          <span>{{ t('toolboxPage.tabSnippets') }}</span>
          <div class="panel-tools">
            <el-input v-model="snippetFilter" :placeholder="t('toolboxPage.snippetFilter')" clearable class="tool-input" />
            <el-button type="primary" :icon="Plus" @click="openSnippetDialog()">{{ t('toolboxPage.addSnippet') }}</el-button>
            <el-button :icon="RefreshRight" @click="loadSnippets">{{ t('common.refresh') }}</el-button>
          </div>
        </div>
        <el-table :data="filteredSnippets" stripe class="tb-table" v-loading="runLoading">
          <el-table-column prop="name" :label="t('toolboxPage.snippetName')" min-width="140" />
          <el-table-column prop="category" :label="t('toolboxPage.category')" width="100" />
          <el-table-column prop="command" :label="t('toolboxPage.command')" show-overflow-tooltip />
          <el-table-column prop="remark" :label="t('toolboxPage.remark')" width="160" show-overflow-tooltip />
          <el-table-column :label="t('common.actions')" width="200" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" :icon="VideoPlay" @click="runSnippet(row)">{{ t('toolboxPage.run') }}</el-button>
              <el-button link :icon="CopyDocument" @click="copyText(row.command)" />
              <el-button v-if="!row.builtin" link :icon="Edit" @click="openSnippetDialog(row)" />
              <el-button v-if="!row.builtin" link type="danger" :icon="Delete" @click="deleteSnippet(row)" />
            </template>
          </el-table-column>
        </el-table>
      </div>
      <div v-if="runOutput" class="panel term-panel">
        <div class="panel-head">{{ t('toolboxPage.output') }}</div>
        <pre class="term">{{ runOutput }}</pre>
      </div>
    </section>

    <!-- Network -->
    <section v-show="activeTab === 'network'" class="tb-section">
      <div class="panel net-panel">
        <div class="panel-head">{{ t('toolboxPage.tabNetwork') }}</div>
        <div class="net-bar">
          <el-input v-model="host" :placeholder="t('toolboxPage.hostPlaceholder')" class="net-input" clearable />
          <el-button type="primary" :loading="loading && netKind === 'ping'" @click="runNet('ping')">Ping</el-button>
          <el-button :loading="loading && netKind === 'traceroute'" @click="runNet('traceroute')">Traceroute</el-button>
          <el-button :loading="loading && netKind === 'dns'" @click="runNet('dns')">{{ t('toolboxPage.dnsQuery') }}</el-button>
        </div>
      </div>
      <div class="panel term-panel">
        <div class="panel-head">{{ t('toolboxPage.output') }}</div>
        <pre class="term">{{ netOutput || t('toolboxPage.emptyOutput') }}</pre>
      </div>
    </section>

    <el-dialog v-model="snippetDialog" :title="snippetForm.id ? t('toolboxPage.editSnippet') : t('toolboxPage.addSnippet')" width="560px">
      <el-form label-width="80px">
        <el-form-item :label="t('toolboxPage.snippetName')"><el-input v-model="snippetForm.name" /></el-form-item>
        <el-form-item :label="t('toolboxPage.category')"><el-input v-model="snippetForm.category" /></el-form-item>
        <el-form-item :label="t('toolboxPage.command')"><el-input v-model="snippetForm.command" type="textarea" :rows="4" /></el-form-item>
        <el-form-item :label="t('toolboxPage.remark')"><el-input v-model="snippetForm.remark" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="snippetDialog = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="saveSnippet">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.tb-page {
  --tb-radius: 14px;
  --tb-border: var(--el-border-color-lighter);
  --tb-surface: var(--el-bg-color);
  --tb-muted: var(--el-text-color-secondary);
  padding-bottom: 32px;
}

.tb-hero {
  margin-bottom: 20px;
  padding: 20px 22px 16px;
  border: 1px solid var(--tb-border);
  border-radius: 18px;
  background:
    radial-gradient(1200px 180px at 0% 0%, color-mix(in srgb, var(--el-color-primary) 16%, transparent), transparent 60%),
    linear-gradient(180deg, var(--el-fill-color-blank, #fff) 0%, var(--el-fill-color-lighter) 100%);
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
}

.tb-hero-main {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-bottom: 16px;
}

.tb-hero-badge {
  width: 48px;
  height: 48px;
  border-radius: 14px;
  display: grid;
  place-items: center;
  color: #fff;
  background: linear-gradient(145deg, var(--el-color-primary-light-3), var(--el-color-primary));
  box-shadow: 0 8px 18px color-mix(in srgb, var(--el-color-primary) 35%, transparent);
}

.tb-title {
  margin: 0;
  font-size: 24px;
  font-weight: 700;
  letter-spacing: -0.02em;
  color: var(--el-text-color-primary);
}

.tb-sub {
  margin: 4px 0 0;
  font-size: 13px;
  color: var(--tb-muted);
}

.tb-tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  padding: 4px;
  border-radius: 12px;
  background: color-mix(in srgb, var(--el-fill-color) 80%, transparent);
  border: 1px solid var(--tb-border);
}

.tb-tab {
  border: 0;
  background: transparent;
  color: var(--tb-muted);
  font-size: 13px;
  font-weight: 600;
  padding: 8px 14px;
  border-radius: 9px;
  cursor: pointer;
  transition: all .18s ease;
}

.tb-tab:hover {
  color: var(--el-text-color-primary);
  background: color-mix(in srgb, var(--el-bg-color) 70%, transparent);
}

.tb-tab.active {
  color: var(--el-color-primary);
  background: var(--el-bg-color);
  box-shadow: 0 1px 3px rgba(15, 23, 42, 0.08);
}

.tb-section { animation: fadeIn .22s ease; }
@keyframes fadeIn {
  from { opacity: 0; transform: translateY(4px); }
  to { opacity: 1; transform: none; }
}

.metric-row {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 14px;
  margin-bottom: 14px;
}

.metric-tile {
  display: flex;
  gap: 12px;
  padding: 16px;
  border-radius: var(--tb-radius);
  border: 1px solid var(--tb-border);
  background: var(--tb-surface);
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.03);
}

.metric-icon {
  width: 40px;
  height: 40px;
  border-radius: 12px;
  display: grid;
  place-items: center;
  flex-shrink: 0;
}

.tone-uptime .metric-icon { background: #eff6ff; color: #2563eb; }
.tone-mem .metric-icon { background: #ecfdf5; color: #059669; }
.tone-cpu .metric-icon { background: #fff7ed; color: #ea580c; }

.metric-label { font-size: 12px; color: var(--tb-muted); margin-bottom: 4px; }
.metric-value { font-size: 26px; font-weight: 720; letter-spacing: -0.03em; line-height: 1.1; }
.metric-unit { font-size: 14px; margin-left: 2px; color: var(--tb-muted); font-weight: 600; }
.metric-foot { margin-top: 6px; font-size: 12px; color: var(--tb-muted); }
.metric-bar {
  margin-top: 10px;
  height: 5px;
  border-radius: 999px;
  background: var(--el-fill-color);
  overflow: hidden;
}
.metric-bar i {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: linear-gradient(90deg, #34d399, #059669);
}

.sys-grid {
  display: grid;
  grid-template-columns: minmax(260px, 320px) 1fr;
  gap: 14px;
}

.sys-main { display: flex; flex-direction: column; gap: 14px; min-width: 0; }

.panel {
  border: 1px solid var(--tb-border);
  border-radius: var(--tb-radius);
  background: var(--tb-surface);
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.03);
  overflow: hidden;
}

.panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 14px 16px;
  font-weight: 650;
  border-bottom: 1px solid var(--tb-border);
  background: linear-gradient(180deg, var(--el-fill-color-blank, #fff), var(--el-fill-color-lighter));
}

.panel-head.with-tools { flex-wrap: wrap; }
.panel-tools { display: flex; gap: 8px; flex-wrap: wrap; align-items: center; }
.tool-input { width: 240px; }

.panel-actions { padding: 12px 16px 16px; }

.tb-table { --el-table-header-bg-color: transparent; }
.tb-table :deep(th.el-table__cell) {
  font-weight: 600;
  color: var(--tb-muted);
  background: transparent !important;
}

.health-panel { display: flex; flex-direction: column; }
.health-body {
  padding: 20px 16px 8px;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
}
.health-num { display: block; font-size: 28px; font-weight: 760; letter-spacing: -0.03em; }
.health-grade { font-size: 12px; color: var(--tb-muted); }
.health-summary { margin: 10px 0 14px; font-size: 13px; color: var(--tb-muted); line-height: 1.5; }
.factor-list { width: 100%; display: flex; flex-direction: column; gap: 6px; }
.factor-row {
  display: grid;
  grid-template-columns: 16px 1fr auto;
  gap: 8px;
  align-items: center;
  text-align: left;
  padding: 8px 10px;
  border-radius: 10px;
  background: var(--el-fill-color-lighter);
  font-size: 12px;
}
.factor-row.ok { color: #059669; }
.factor-row.warn { color: #d97706; }
.factor-row.bad, .factor-row.error, .factor-row.critical { color: #dc2626; }
.factor-label { color: var(--el-text-color-regular); }
.factor-detail { color: inherit; font-weight: 600; }

.disk-cell { display: flex; flex-direction: column; gap: 4px; }
.disk-bar {
  height: 4px;
  border-radius: 999px;
  background: var(--el-fill-color);
  overflow: hidden;
}
.disk-bar i {
  display: block;
  height: 100%;
  background: linear-gradient(90deg, var(--el-color-primary-light-3), var(--el-color-primary));
}

.bench-banner {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  padding: 18px 20px;
  margin-bottom: 14px;
  border-radius: var(--tb-radius);
  border: 1px solid color-mix(in srgb, var(--el-color-primary) 22%, var(--tb-border));
  background:
    radial-gradient(500px 120px at 100% 0%, color-mix(in srgb, var(--el-color-primary) 18%, transparent), transparent 70%),
    var(--tb-surface);
}
.bench-banner h3 { margin: 0 0 4px; font-size: 16px; }
.bench-banner p { margin: 0; font-size: 13px; color: var(--tb-muted); max-width: 640px; line-height: 1.5; }

.bench-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 14px;
}

.bench-card {
  display: flex;
  flex-direction: column;
  min-height: 280px;
  padding: 16px;
  border-radius: 16px;
  border: 1px solid var(--tb-border);
  background: var(--tb-surface);
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.03);
  transition: transform .18s ease, box-shadow .18s ease, border-color .18s ease;
}
.bench-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 10px 24px rgba(15, 23, 42, 0.07);
}
.bench-card.running {
  border-color: color-mix(in srgb, var(--el-color-primary) 45%, var(--tb-border));
}
.bench-card.done {
  border-color: color-mix(in srgb, #059669 35%, var(--tb-border));
}

.bench-top { display: flex; gap: 12px; align-items: flex-start; }
.bench-icon {
  width: 42px;
  height: 42px;
  border-radius: 12px;
  display: grid;
  place-items: center;
  flex-shrink: 0;
}
.tone-cpu .bench-icon { background: #fff7ed; color: #ea580c; }
.tone-mem .bench-icon { background: #ecfdf5; color: #059669; }
.tone-disk .bench-icon { background: #eff6ff; color: #2563eb; }
.tone-net .bench-icon { background: #f5f3ff; color: #7c3aed; }

.bench-title { font-size: 15px; font-weight: 700; }
.bench-desc { margin-top: 3px; font-size: 12px; line-height: 1.45; color: var(--tb-muted); }

.bench-stage {
  flex: 1;
  margin: 18px 0 14px;
  padding: 16px 14px;
  border-radius: 12px;
  background: var(--el-fill-color-lighter);
  border: 1px dashed var(--tb-border);
  display: flex;
  flex-direction: column;
  justify-content: center;
  min-height: 118px;
}
.bench-card.done .bench-stage {
  border-style: solid;
  background: color-mix(in srgb, #ecfdf5 70%, var(--el-fill-color-lighter));
}

.bench-score-line { display: flex; align-items: baseline; gap: 6px; }
.bench-score {
  font-size: 34px;
  font-weight: 780;
  letter-spacing: -0.04em;
  color: var(--el-text-color-primary);
  line-height: 1;
}
.bench-unit { font-size: 13px; font-weight: 650; color: var(--tb-muted); }
.bench-detail { margin-top: 8px; font-size: 12px; line-height: 1.45; color: var(--el-text-color-regular); }
.bench-meta { margin-top: 6px; font-size: 11px; color: var(--tb-muted); }

.bench-idle-mark {
  font-size: 28px;
  font-weight: 700;
  color: color-mix(in srgb, var(--tb-muted) 55%, transparent);
  line-height: 1;
}
.bench-idle-title { margin-top: 6px; font-size: 13px; font-weight: 650; color: var(--el-text-color-regular); }
.bench-idle-hint { margin-top: 2px; font-size: 12px; color: var(--tb-muted); }

.bench-pulse {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  border: 3px solid color-mix(in srgb, var(--el-color-primary) 25%, transparent);
  border-top-color: var(--el-color-primary);
  animation: spin .7s linear infinite;
  margin-bottom: 8px;
}

.bench-btn {
  width: 100%;
  border: 0;
  border-radius: 11px;
  padding: 11px 14px;
  font-size: 13px;
  font-weight: 700;
  cursor: pointer;
  color: #fff;
  background: linear-gradient(145deg, var(--el-color-primary-light-3), var(--el-color-primary));
  box-shadow: 0 6px 14px color-mix(in srgb, var(--el-color-primary) 28%, transparent);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  transition: filter .15s ease, transform .15s ease;
}
.bench-btn:hover:not(:disabled) { filter: brightness(1.03); transform: translateY(-1px); }
.bench-btn:disabled { opacity: .55; cursor: not-allowed; box-shadow: none; }

.bench-spinner {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  border: 2px solid rgba(255,255,255,.35);
  border-top-color: #fff;
  animation: spin .7s linear infinite;
}

@keyframes spin { to { transform: rotate(360deg); } }

.net-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  padding: 16px;
  align-items: center;
}
.net-input { width: min(360px, 100%); }

.term-panel { margin-top: 14px; }
.term {
  margin: 0;
  padding: 16px 18px;
  min-height: 180px;
  max-height: 420px;
  overflow: auto;
  font-size: 12.5px;
  line-height: 1.55;
  white-space: pre-wrap;
  color: #e2e8f0;
  background:
    linear-gradient(180deg, rgba(15, 23, 42, 0.02), transparent 40%),
    #0f172a;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
}

@media (max-width: 1100px) {
  .bench-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .sys-grid { grid-template-columns: 1fr; }
}
@media (max-width: 720px) {
  .metric-row, .bench-grid { grid-template-columns: 1fr; }
  .tb-hero { padding: 16px; }
  .tool-input, .net-input { width: 100%; }
  .bench-banner { flex-direction: column; align-items: stretch; }
}
</style>
