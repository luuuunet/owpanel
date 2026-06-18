<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import api from '@/api'
import { apiContentLang } from '@/locales'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  CopyDocument, Delete, Edit, Plus, RefreshRight, VideoPlay,
  Monitor, Cpu, Odometer,
} from '@element-plus/icons-vue'

const { t, locale } = useI18n()

const activeTab = ref('system')
const loading = ref(false)

// Network tools
const host = ref('baidu.com')
const netOutput = ref('')

// System overview
const overview = ref<any>(null)
const processes = ref<any[]>([])
const killingPid = ref<number | null>(null)
const droppingCache = ref(false)

// Ports
const ports = ref<any[]>([])
const portFilter = ref('')

// Health
const health = ref<any>(null)

// Snippets
const snippets = ref<any[]>([])
const snippetFilter = ref('')
const runOutput = ref('')
const runLoading = ref(false)
const snippetDialog = ref(false)
const snippetForm = ref({ id: 0, name: '', command: '', category: 'custom', remark: '' })

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
  if (s >= 80) return '#67c23a'
  if (s >= 60) return '#e6a23c'
  return '#f56c6c'
})

function formatBytes(n: number) {
  if (!n) return '0 B'
  const u = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.min(Math.floor(Math.log(n) / Math.log(1024)), u.length - 1)
  return `${(n / 1024 ** i).toFixed(1)} ${u[i]}`
}

async function runNet(type: 'ping' | 'traceroute' | 'dns') {
  loading.value = true
  netOutput.value = ''
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

function onTabChange(name: string | number) {
  if (name === 'system') loadSystem()
  else if (name === 'ports') loadPorts()
  else if (name === 'snippets') loadSnippets()
}

onMounted(() => {
  loadSystem()
})
</script>

<template>
  <div class="toolbox-page">
    <div class="page-header">
      <h2>{{ t('toolboxPage.title') }}</h2>
      <p class="subtitle">{{ t('toolboxPage.subtitle') }}</p>
    </div>

    <el-tabs v-model="activeTab" @tab-change="onTabChange">
      <!-- System -->
      <el-tab-pane :label="t('toolboxPage.tabSystem')" name="system">
        <div v-loading="loading" class="tab-body">
          <el-row :gutter="16" class="stat-row">
            <el-col :xs="24" :sm="8">
              <el-card shadow="hover" class="stat-card">
                <div class="stat-label"><el-icon><Odometer /></el-icon> {{ t('toolboxPage.uptime') }}</div>
                <div class="stat-value">{{ overview?.uptime_human || '—' }}</div>
                <div class="stat-sub">{{ overview?.hostname }}</div>
              </el-card>
            </el-col>
            <el-col :xs="24" :sm="8">
              <el-card shadow="hover" class="stat-card">
                <div class="stat-label"><el-icon><Monitor /></el-icon> {{ t('toolboxPage.memory') }}</div>
                <div class="stat-value">{{ overview?.memory?.used_percent?.toFixed?.(1) ?? '—' }}%</div>
                <div class="stat-sub">{{ formatBytes(overview?.memory?.used) }} / {{ formatBytes(overview?.memory?.total) }}</div>
              </el-card>
            </el-col>
            <el-col :xs="24" :sm="8">
              <el-card shadow="hover" class="stat-card">
                <div class="stat-label"><el-icon><Cpu /></el-icon> {{ t('toolboxPage.load') }}</div>
                <div class="stat-value">{{ overview?.load1?.toFixed?.(2) ?? '—' }}</div>
                <div class="stat-sub">{{ overview?.cpu_count }} {{ t('toolboxPage.cores') }}</div>
              </el-card>
            </el-col>
          </el-row>

          <el-row :gutter="16">
            <el-col :xs="24" :md="8">
              <el-card shadow="hover">
                <template #header>
                  <span>{{ t('toolboxPage.healthScore') }}</span>
                  <el-button link :icon="RefreshRight" @click="loadSystem" />
                </template>
                <div v-if="health" class="health-block">
                  <el-progress type="dashboard" :percentage="health.score" :color="healthColor" :width="110">
                    <template #default>
                      <span class="health-num">{{ health.score }}</span>
                      <span class="health-grade">{{ health.grade }}</span>
                    </template>
                  </el-progress>
                  <p class="health-summary">{{ health.summary }}</p>
                  <div v-for="f in health.factors" :key="f.key" class="factor-row">
                    <span>{{ f.label }}</span>
                    <el-tag :type="f.status === 'ok' ? 'success' : f.status === 'warn' ? 'warning' : 'danger'" size="small">{{ f.detail }}</el-tag>
                  </div>
                </div>
              </el-card>
              <el-card shadow="hover" style="margin-top: 12px">
                <template #header>{{ t('toolboxPage.quickActions') }}</template>
                <el-button type="warning" :loading="droppingCache" @click="dropCache">{{ t('toolboxPage.dropCache') }}</el-button>
              </el-card>
            </el-col>
            <el-col :xs="24" :md="16">
              <el-card shadow="hover" style="margin-bottom: 12px">
                <template #header>{{ t('toolboxPage.diskUsage') }}</template>
                <el-table :data="overview?.disks || []" size="small" stripe>
                  <el-table-column prop="mount" :label="t('toolboxPage.mount')" min-width="100" />
                  <el-table-column :label="t('toolboxPage.used')" width="100">
                    <template #default="{ row }">{{ row.used_percent?.toFixed(1) }}%</template>
                  </el-table-column>
                  <el-table-column :label="t('toolboxPage.size')" min-width="160">
                    <template #default="{ row }">{{ formatBytes(row.used) }} / {{ formatBytes(row.total) }}</template>
                  </el-table-column>
                </el-table>
              </el-card>
              <el-card shadow="hover">
                <template #header>{{ t('toolboxPage.topProcesses') }}</template>
                <el-table :data="processes" size="small" stripe max-height="320">
                  <el-table-column prop="pid" label="PID" width="70" />
                  <el-table-column prop="name" :label="t('toolboxPage.process')" width="120" show-overflow-tooltip />
                  <el-table-column prop="user" :label="t('toolboxPage.user')" width="90" show-overflow-tooltip />
                  <el-table-column :label="t('toolboxPage.memCol')" width="70">
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
              </el-card>
            </el-col>
          </el-row>
        </div>
      </el-tab-pane>

      <!-- Ports -->
      <el-tab-pane :label="t('toolboxPage.tabPorts')" name="ports">
        <div v-loading="loading" class="tab-body">
          <div class="toolbar">
            <el-input v-model="portFilter" :placeholder="t('toolboxPage.portFilter')" clearable style="width: 240px" />
            <el-button :icon="RefreshRight" @click="loadPorts">{{ t('common.refresh') }}</el-button>
          </div>
          <el-table :data="filteredPorts" stripe>
            <el-table-column prop="port" :label="t('toolboxPage.port')" width="80" sortable />
            <el-table-column prop="protocol" label="Proto" width="70" />
            <el-table-column prop="address" :label="t('toolboxPage.bindAddr')" width="120" />
            <el-table-column prop="process" :label="t('toolboxPage.process')" width="140" show-overflow-tooltip />
            <el-table-column prop="pid" label="PID" width="70" />
            <el-table-column prop="user" :label="t('toolboxPage.user')" width="100" show-overflow-tooltip />
            <el-table-column prop="command" :label="t('toolboxPage.command')" show-overflow-tooltip />
            <el-table-column :label="t('toolboxPage.firewall')" width="120">
              <template #default="{ row }">
                <el-tag v-if="row.firewalled" type="success" size="small">{{ t('toolboxPage.fwAllowed') }}</el-tag>
                <el-button v-else link type="primary" size="small" @click="allowPort(row)">{{ t('toolboxPage.fwAllow') }}</el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </el-tab-pane>

      <!-- Snippets -->
      <el-tab-pane :label="t('toolboxPage.tabSnippets')" name="snippets">
        <div class="tab-body">
          <div class="toolbar">
            <el-input v-model="snippetFilter" :placeholder="t('toolboxPage.snippetFilter')" clearable style="width: 240px" />
            <el-button type="primary" :icon="Plus" @click="openSnippetDialog()">{{ t('toolboxPage.addSnippet') }}</el-button>
            <el-button :icon="RefreshRight" @click="loadSnippets">{{ t('common.refresh') }}</el-button>
          </div>
          <el-table :data="filteredSnippets" stripe v-loading="runLoading">
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
          <el-card v-if="runOutput" shadow="hover" style="margin-top: 12px">
            <template #header>{{ t('toolboxPage.output') }}</template>
            <pre class="output">{{ runOutput }}</pre>
          </el-card>
        </div>
      </el-tab-pane>

      <!-- Network -->
      <el-tab-pane :label="t('toolboxPage.tabNetwork')" name="network">
        <el-card shadow="hover" style="margin-bottom: 16px">
          <el-input v-model="host" :placeholder="t('toolboxPage.hostPlaceholder')" style="width: 300px; margin-right: 12px" />
          <el-button type="primary" :loading="loading" @click="runNet('ping')">Ping</el-button>
          <el-button :loading="loading" @click="runNet('traceroute')">Traceroute</el-button>
          <el-button :loading="loading" @click="runNet('dns')">{{ t('toolboxPage.dnsQuery') }}</el-button>
        </el-card>
        <el-card shadow="hover">
          <template #header>{{ t('toolboxPage.output') }}</template>
          <pre class="output">{{ netOutput || t('toolboxPage.emptyOutput') }}</pre>
        </el-card>
      </el-tab-pane>
    </el-tabs>

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
.toolbox-page { padding-bottom: 24px; }
.page-header { margin-bottom: 16px; }
.subtitle { color: var(--el-text-color-secondary); margin: 4px 0 0; font-size: 13px; }
.tab-body { padding-top: 8px; }
.stat-row { margin-bottom: 16px; }
.stat-card { text-align: center; }
.stat-label { display: flex; align-items: center; justify-content: center; gap: 4px; color: var(--el-text-color-secondary); font-size: 13px; }
.stat-value { font-size: 28px; font-weight: 600; margin: 8px 0 4px; }
.stat-sub { font-size: 12px; color: var(--el-text-color-secondary); }
.health-block { text-align: center; }
.health-num { display: block; font-size: 22px; font-weight: 700; }
.health-grade { font-size: 12px; color: var(--el-text-color-secondary); }
.health-summary { font-size: 13px; color: var(--el-text-color-secondary); margin: 8px 0 12px; }
.factor-row { display: flex; justify-content: space-between; align-items: center; font-size: 12px; padding: 4px 0; border-top: 1px solid var(--el-border-color-lighter); }
.toolbar { display: flex; gap: 8px; margin-bottom: 12px; flex-wrap: wrap; }
.output {
  background: #1e1e1e; color: #d4d4d4; padding: 16px; border-radius: 6px;
  min-height: 160px; max-height: 400px; overflow: auto; font-size: 13px; white-space: pre-wrap;
}
</style>
