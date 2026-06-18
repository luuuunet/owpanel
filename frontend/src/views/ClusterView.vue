<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import api from '@/api'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import ClusterFlowCanvas, { type FlowGraph } from '@/components/ClusterFlowCanvas.vue'
import { cfTheme } from '@/config/theme'

const { t } = useI18n()
const auth = useAuthStore()

const tab = ref('quick')
const loading = ref(false)
const overview = ref<any>({})
const nodes = ref<any[]>([])
const balancers = ref<any[]>([])
const agentToken = ref('')
const quickLog = ref('')

const quickLb = ref({
  name: '', domain: 'app.example.com', listen_port: 80, algorithm: 'round_robin',
  node_ids: [] as number[], auto_setup: true,
})
const quickRepl = ref({
  master_node_id: 0, slave_node_id: 0, repl_user: 'repl', db_name: 'app_db', auto_setup: true,
})
const quickLbLoading = ref(false)
const quickReplLoading = ref(false)

const flowGraph = ref<FlowGraph>({ nodes: [], edges: [] })
const workflowMeta = ref<any>(null)
const runLog = ref('')
const workflowStatus = ref('')
const flowLoading = ref(false)
const showAssistant = ref(false)
const flowLoaded = ref(false)
const workflowRunning = ref(false)

const nodeDialog = ref(false)
const editingNode = ref<any>(null)
const monitorDrawer = ref(false)
const monitorNode = ref<any>(null)
const monitorData = ref<any>(null)
const monitorLoading = ref(false)
const provisionLoading = ref<number | null>(null)
const sshTestLoading = ref<number | null>(null)
const nodeForm = ref({
  name: '', host: '', port: 8888, safe_path: '', agent_token: '',
  role: 'worker', tags: '', remark: '', website_host: '', website_port: 80,
  ssh_host: '', ssh_port: 22, ssh_user: 'root', ssh_password: '',
  provision_role: 'lb_backend', auto_provision: false,
})

const lbDialog = ref(false)
const editingLB = ref<any>(null)
const lbForm = ref({
  name: '', domain: '', listen_port: 80, ssl: false,
  algorithm: 'round_robin', health_check: true, health_path: '/',
  health_interval: 10, sticky_session: false, websocket: true, enabled: true, remark: '',
})

const backendDialog = ref(false)
const currentLB = ref<any>(null)
const backendForm = ref({ node_id: 0, host: '', port: 80, weight: 1, enabled: true })

const algorithms = computed(() => [
  { value: 'round_robin', label: t('clusterPage.algoRoundRobin') },
  { value: 'least_conn', label: t('clusterPage.algoLeastConn') },
  { value: 'ip_hash', label: t('clusterPage.algoIpHash') },
  { value: 'random', label: t('clusterPage.algoRandom') },
])

function statusTag(s: string) {
  if (s === 'online' || s === 'active' || s === 'up') return 'success'
  if (s === 'offline' || s === 'down') return 'danger'
  return 'info'
}

function provisionTag(s: string) {
  if (s === 'ready') return 'success'
  if (s === 'failed') return 'danger'
  if (s === 'provisioning') return 'warning'
  return 'info'
}

function resourceColor(p: number) {
  if (p >= 90) return cfTheme.danger
  if (p >= 70) return cfTheme.warning
  return cfTheme.success
}

const joinInfo = ref<any>(null)
const joinRoles = [
  { key: 'worker', labelKey: 'joinRoleWorker' },
  { key: 'lb_backend', labelKey: 'joinRoleLbBackend' },
  { key: 'db_slave', labelKey: 'joinRoleDbSlave' },
  { key: 'db_master', labelKey: 'joinRoleDbMaster' },
]

async function loadJoinInfo() {
  if (auth.user?.role !== 'admin') return
  try {
    const res: any = await api.get('/cluster/join-info')
    joinInfo.value = res.data || null
  } catch { /* ignore */ }
}

async function copyJoinCmd(role: string) {
  const cmd = joinInfo.value?.commands?.[role]
  if (!cmd) return
  await navigator.clipboard.writeText(cmd)
  ElMessage.success(t('clusterPage.joinCmdCopied'))
}

const backendNodes = computed(() => nodes.value.filter((n) => !n.is_local))
const dbNodes = computed(() => nodes.value.filter((n) => !n.is_local))

async function loadAll() {
  loading.value = true
  try {
    const [ov, n, lb, tok]: any[] = await Promise.all([
      api.get('/cluster/overview'),
      api.get('/cluster/nodes'),
      api.get('/load-balancers'),
      auth.user?.role === 'admin' ? api.get('/cluster/agent/token') : Promise.resolve({ data: {} }),
    ])
    overview.value = ov.data?.overview || {}
    nodes.value = n.data || []
    balancers.value = lb.data || []
    agentToken.value = tok.data?.token || ''
    if (!quickLb.value.node_ids.length) {
      quickLb.value.node_ids = backendNodes.value.map((n: any) => n.id)
    }
    await loadJoinInfo()
  } finally {
    loading.value = false
  }
}

async function quickCreateLB() {
  if (!quickLb.value.domain.trim()) {
    ElMessage.warning(t('clusterPage.quickLbNeedDomain'))
    return
  }
  if (!quickLb.value.node_ids.length) {
    ElMessage.warning(t('clusterPage.quickLbNeedNodes'))
    return
  }
  quickLbLoading.value = true
  quickLog.value = ''
  try {
    const res: any = await api.post('/cluster/quick/lb', quickLb.value)
    quickLog.value = res.data?.log || res.data?.message || ''
    if (res.data?.status === 'ok') {
      ElMessage.success(res.data.message)
      tab.value = 'lb'
    } else {
      ElMessage.warning(res.data?.message || t('common.failed'))
    }
    await loadAll()
  } catch (e: any) {
    ElMessage.error(e?.error || t('common.failed'))
  } finally {
    quickLbLoading.value = false
  }
}

async function quickCreateRepl() {
  if (!quickRepl.value.master_node_id || !quickRepl.value.slave_node_id) {
    ElMessage.warning(t('clusterPage.quickReplNeedNodes'))
    return
  }
  quickReplLoading.value = true
  quickLog.value = ''
  try {
    const res: any = await api.post('/cluster/quick/replication', quickRepl.value)
    quickLog.value = res.data?.log || res.data?.message || ''
    ElMessage.success(res.data?.message || t('clusterPage.quickReplDone'))
  } catch (e: any) {
    ElMessage.error(e?.error || t('common.failed'))
  } finally {
    quickReplLoading.value = false
  }
}

function openAddNode() {
  editingNode.value = null
  nodeForm.value = {
    name: '', host: '', port: 8888, safe_path: '', agent_token: agentToken.value,
    role: 'worker', tags: '', remark: '', website_host: '', website_port: 80,
    ssh_host: '', ssh_port: 22, ssh_user: 'root', ssh_password: '',
    provision_role: 'lb_backend', auto_provision: false,
  }
  nodeDialog.value = true
}

function openEditNode(row: any) {
  if (row.is_local) {
    ElMessage.info(t('clusterPage.localNodeHint'))
    return
  }
  editingNode.value = row
  nodeForm.value = {
    name: row.name, host: row.host, port: row.port, safe_path: row.safe_path || '',
    agent_token: '', role: row.role, tags: row.tags || '', remark: row.remark || '',
    website_host: row.website_host || '', website_port: row.website_port || 80,
    ssh_host: row.ssh_host || '', ssh_port: row.ssh_port || 22, ssh_user: row.ssh_user || 'root',
    ssh_password: '', provision_role: row.provision_role || 'lb_backend', auto_provision: false,
  }
  nodeDialog.value = true
}

async function saveNode() {
  if (editingNode.value) {
    await api.put(`/cluster/nodes/${editingNode.value.id}`, nodeForm.value)
  } else {
    await api.post('/cluster/nodes', nodeForm.value)
  }
  ElMessage.success(t('common.success'))
  nodeDialog.value = false
  loadAll()
}

async function testNode(row: any) {
  const res: any = await api.post(`/cluster/nodes/${row.id}/test`)
  ElMessage.success(t('clusterPage.testOk', { status: res.data?.status }))
  loadAll()
}

async function testSSH(row: any) {
  sshTestLoading.value = row.id
  try {
    const res: any = await api.post(`/cluster/nodes/${row.id}/ssh/test`)
    if (res.data?.ok) {
      ElMessage.success(res.data.message || t('clusterPage.sshTestOk'))
    } else {
      ElMessage.error(res.data?.message || t('clusterPage.sshTestFail'))
    }
  } catch (e: any) {
    ElMessage.error(e?.error || t('clusterPage.sshTestFail'))
  } finally {
    sshTestLoading.value = null
  }
}

async function provisionNode(row: any) {
  await ElMessageBox.confirm(t('clusterPage.provisionConfirm', { name: row.name }), t('common.warning'), { type: 'warning' })
  provisionLoading.value = row.id
  try {
    const res: any = await api.post(`/cluster/nodes/${row.id}/provision`)
    if (res.data?.status === 'ready') {
      ElMessage.success(res.data?.message || t('clusterPage.provisionDone'))
    } else {
      ElMessage.warning(res.data?.message || t('clusterPage.provisionFail'))
    }
    loadAll()
  } catch (e: any) {
    ElMessage.error(e?.error || t('clusterPage.provisionFail'))
  } finally {
    provisionLoading.value = null
  }
}

async function openMonitor(row: any) {
  monitorNode.value = row
  monitorDrawer.value = true
  await refreshMonitor()
}

async function refreshMonitor() {
  if (!monitorNode.value) return
  monitorLoading.value = true
  try {
    const res: any = await api.get(`/cluster/nodes/${monitorNode.value.id}/monitor`)
    monitorData.value = res.data || {}
    await loadAll()
    const updated = nodes.value.find((n: any) => n.id === monitorNode.value.id)
    if (updated) monitorNode.value = updated
  } catch (e: any) {
    ElMessage.error(e?.error || t('clusterPage.monitorFail'))
  } finally {
    monitorLoading.value = false
  }
}

async function deleteNode(row: any) {
  await ElMessageBox.confirm(t('clusterPage.deleteNodeConfirm', { name: row.name }), t('common.warning'), { type: 'warning' })
  await api.delete(`/cluster/nodes/${row.id}`)
  ElMessage.success(t('common.deleted'))
  loadAll()
}

async function syncAll() {
  await api.post('/cluster/nodes/sync-all')
  ElMessage.success(t('clusterPage.syncDone'))
  loadAll()
}

async function copyToken() {
  if (!agentToken.value) return
  await navigator.clipboard.writeText(agentToken.value)
  ElMessage.success(t('clusterPage.tokenCopied'))
}

async function regenerateToken() {
  await ElMessageBox.confirm(t('clusterPage.regenerateConfirm'), t('common.warning'), { type: 'warning' })
  const res: any = await api.post('/cluster/agent/regenerate-token')
  agentToken.value = res.data?.token || ''
  ElMessage.success(t('clusterPage.tokenRegenerated'))
}

function openAddLB() {
  editingLB.value = null
  lbForm.value = {
    name: '', domain: '', listen_port: 80, ssl: false, algorithm: 'round_robin',
    health_check: true, health_path: '/', health_interval: 10,
    sticky_session: false, websocket: true, enabled: true, remark: '',
  }
  lbDialog.value = true
}

function openEditLB(row: any) {
  editingLB.value = row
  lbForm.value = { ...row }
  lbDialog.value = true
}

async function saveLB() {
  if (editingLB.value) {
    await api.put(`/load-balancers/${editingLB.value.id}`, lbForm.value)
  } else {
    await api.post('/load-balancers', lbForm.value)
  }
  ElMessage.success(t('common.success'))
  lbDialog.value = false
  loadAll()
}

async function applyLB(row: any) {
  try {
    await api.post(`/load-balancers/${row.id}/apply`)
    ElMessage.success(t('clusterPage.lbApplied'))
    loadAll()
  } catch (e: any) {
    ElMessage.error(e?.error || e?.message || t('common.failed'))
  }
}

async function deleteLB(row: any) {
  await ElMessageBox.confirm(t('clusterPage.deleteLbConfirm', { name: row.name }), t('common.warning'), { type: 'warning' })
  await api.delete(`/load-balancers/${row.id}`)
  loadAll()
}

function openAddBackend(row: any) {
  currentLB.value = row
  backendForm.value = { node_id: 0, host: '', port: 80, weight: 1, enabled: true }
  backendDialog.value = true
}

async function saveBackend() {
  await api.post(`/load-balancers/${currentLB.value.id}/backends`, backendForm.value)
  ElMessage.success(t('common.success'))
  backendDialog.value = false
  loadAll()
}

async function deleteBackend(lb: any, b: any) {
  await api.delete(`/load-balancers/${lb.id}/backends/${b.id}`)
  loadAll()
}

async function loadWorkflow() {
  flowLoading.value = true
  try {
    const res: any = await api.get('/cluster/workflow')
    flowGraph.value = res.data?.graph || { nodes: [], edges: [] }
    workflowMeta.value = res.data?.workflow || null
    if (res.data?.workflow?.last_run_log) {
      runLog.value = res.data.workflow.last_run_log
      workflowStatus.value = res.data.workflow.status || ''
    }
    flowLoaded.value = true
  } catch (e: any) {
    ElMessage.error(e?.error || t('common.failed'))
  } finally {
    flowLoading.value = false
  }
}

async function saveWorkflow() {
  try {
    const res: any = await api.put('/cluster/workflow', { graph: flowGraph.value })
    workflowMeta.value = res.data?.workflow || workflowMeta.value
    ElMessage.success(t('common.success'))
  } catch (e: any) {
    ElMessage.error(e?.error || t('common.failed'))
  }
}

async function runWorkflow() {
  workflowRunning.value = true
  try {
    const res: any = await api.post('/cluster/workflow/run', { graph: flowGraph.value })
    runLog.value = res.data?.log || ''
    workflowStatus.value = res.data?.status || ''
    ElMessage.success(t('clusterPage.flowRunDone'))
    await loadAll()
    await loadWorkflow()
  } catch (e: any) {
    ElMessage.error(e?.error || t('common.failed'))
  } finally {
    workflowRunning.value = false
  }
}

async function syncWorkflowNodes() {
  try {
    const res: any = await api.post('/cluster/workflow/sync-nodes', { graph: flowGraph.value })
    flowGraph.value = res.data?.graph || flowGraph.value
    workflowMeta.value = res.data?.workflow || workflowMeta.value
    ElMessage.success(t('clusterPage.flowSyncDone'))
  } catch (e: any) {
    ElMessage.error(e?.error || t('common.failed'))
  }
}

function applyAssistantGraph(g: FlowGraph) {
  flowGraph.value = g
}

watch(tab, (v) => {
  if (v === 'workflow' && !flowLoaded.value) loadWorkflow()
})

onMounted(() => { loadAll() })
</script>

<template>
  <div class="cluster-page" v-loading="loading">
    <div class="page-header">
      <div>
        <h2>{{ t('clusterPage.title') }}</h2>
        <p class="hint">{{ t('clusterPage.subtitle') }}</p>
      </div>
      <el-button @click="syncAll">{{ t('clusterPage.syncAll') }}</el-button>
    </div>

    <el-row :gutter="16" class="overview-row">
      <el-col :xs="12" :sm="6">
        <el-card shadow="hover"><div class="stat-val">{{ overview.node_online ?? 0 }}/{{ overview.node_total ?? 0 }}</div><div class="stat-label">{{ t('clusterPage.nodesOnline') }}</div></el-card>
      </el-col>
      <el-col :xs="12" :sm="6">
        <el-card shadow="hover"><div class="stat-val">{{ overview.lb_active ?? 0 }}/{{ overview.lb_total ?? 0 }}</div><div class="stat-label">{{ t('clusterPage.lbActive') }}</div></el-card>
      </el-col>
      <el-col :xs="12" :sm="6">
        <el-card shadow="hover"><div class="stat-val">{{ overview.backend_total ?? 0 }}</div><div class="stat-label">{{ t('clusterPage.backends') }}</div></el-card>
      </el-col>
      <el-col :xs="12" :sm="6">
        <el-card shadow="hover" class="token-card">
          <div class="stat-label">{{ t('clusterPage.agentToken') }}</div>
          <div class="token-row">
            <code class="token-preview">{{ agentToken ? agentToken.slice(0, 12) + '…' : '—' }}</code>
            <el-button v-if="agentToken" text type="primary" size="small" @click="copyToken">{{ t('clusterPage.copy') }}</el-button>
            <el-button v-if="auth.user?.role === 'admin'" text size="small" @click="regenerateToken">{{ t('clusterPage.regenerate') }}</el-button>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-tabs v-model="tab">
      <el-tab-pane :label="t('clusterPage.quickTab')" name="quick">
        <el-alert type="success" :closable="false" show-icon :title="t('clusterPage.simpleSteps')" class="flow-hint" />
        <el-card v-if="joinInfo && auth.user?.role === 'admin'" shadow="never" class="join-card">
          <template #header><strong>{{ t('clusterPage.joinTitle') }}</strong></template>
          <p class="join-intro">{{ t('clusterPage.joinAutoWire') }}</p>
          <div class="join-row join-row-main">
            <div class="join-label">{{ t('clusterPage.joinRoleWorker') }}</div>
            <code class="join-cmd">{{ joinInfo.commands?.worker || '—' }}</code>
            <el-button type="primary" size="small" @click="copyJoinCmd('worker')">{{ t('clusterPage.joinCopyCmd') }}</el-button>
          </div>
          <el-collapse class="join-more">
            <el-collapse-item :title="t('clusterPage.joinMoreRoles')" name="more">
              <div v-for="r in joinRoles.filter(x => x.key !== 'worker')" :key="r.key" class="join-row">
                <div class="join-label">{{ t(`clusterPage.${r.labelKey}`) }}</div>
                <code class="join-cmd">{{ joinInfo.commands?.[r.key] || '—' }}</code>
                <el-button size="small" @click="copyJoinCmd(r.key)">{{ t('clusterPage.joinCopyCmd') }}</el-button>
              </div>
            </el-collapse-item>
          </el-collapse>
        </el-card>
        <p class="quick-intro">{{ t('clusterPage.quickIntro') }}</p>
        <el-row :gutter="16">
          <el-col :xs="24" :md="12">
            <el-card shadow="never" class="quick-card">
              <template #header><strong>{{ t('clusterPage.quickLbTitle') }}</strong></template>
              <el-form label-width="100px">
                <el-form-item :label="t('common.name')"><el-input v-model="quickLb.name" :placeholder="t('clusterPage.quickLbNameHint')" /></el-form-item>
                <el-form-item label="Domain"><el-input v-model="quickLb.domain" placeholder="app.example.com" /></el-form-item>
                <el-form-item :label="t('common.port')"><el-input-number v-model="quickLb.listen_port" :min="1" :max="65535" /></el-form-item>
                <el-form-item :label="t('clusterPage.algorithm')">
                  <el-select v-model="quickLb.algorithm" style="width:100%">
                    <el-option v-for="a in algorithms" :key="a.value" :label="a.label" :value="a.value" />
                  </el-select>
                </el-form-item>
                <el-form-item :label="t('clusterPage.quickPickWorkers')">
                  <el-checkbox-group v-model="quickLb.node_ids" class="node-checks">
                    <el-checkbox v-for="n in backendNodes" :key="n.id" :label="n.id">
                      {{ n.name }} ({{ n.website_host || n.host }}:{{ n.website_port || 80 }})
                    </el-checkbox>
                  </el-checkbox-group>
                  <p v-if="!backendNodes.length" class="empty-hint">{{ t('clusterPage.quickNoWorkers') }}</p>
                </el-form-item>
                <el-form-item :label="t('clusterPage.quickAutoSetup')"><el-switch v-model="quickLb.auto_setup" /></el-form-item>
                <el-button type="primary" :loading="quickLbLoading" :disabled="!backendNodes.length" @click="quickCreateLB">
                  {{ t('clusterPage.quickLbBtn') }}
                </el-button>
              </el-form>
            </el-card>
          </el-col>
          <el-col :xs="24" :md="12">
            <el-card shadow="never" class="quick-card">
              <template #header><strong>{{ t('clusterPage.quickReplTitle') }}</strong></template>
              <el-form label-width="100px">
                <el-form-item :label="t('clusterPage.quickMaster')">
                  <el-select v-model="quickRepl.master_node_id" style="width:100%" :placeholder="t('clusterPage.quickPickMaster')">
                    <el-option v-for="n in dbNodes" :key="n.id" :value="n.id" :label="`${n.name} (${n.host})`" />
                  </el-select>
                </el-form-item>
                <el-form-item :label="t('clusterPage.quickSlave')">
                  <el-select v-model="quickRepl.slave_node_id" style="width:100%" :placeholder="t('clusterPage.quickPickSlave')">
                    <el-option v-for="n in dbNodes" :key="n.id" :value="n.id" :label="`${n.name} (${n.host})`" />
                  </el-select>
                </el-form-item>
                <el-form-item :label="t('clusterPage.flowReplUser')"><el-input v-model="quickRepl.repl_user" /></el-form-item>
                <el-form-item :label="t('clusterPage.flowDbName')"><el-input v-model="quickRepl.db_name" /></el-form-item>
                <el-form-item :label="t('clusterPage.quickAutoSetup')"><el-switch v-model="quickRepl.auto_setup" /></el-form-item>
                <el-button type="primary" :loading="quickReplLoading" :disabled="dbNodes.length < 2" @click="quickCreateRepl">
                  {{ t('clusterPage.quickReplBtn') }}
                </el-button>
              </el-form>
            </el-card>
          </el-col>
        </el-row>
        <el-alert v-if="quickLog" type="info" :closable="false" show-icon :title="t('clusterPage.quickLog')" class="quick-log">
          <pre>{{ quickLog }}</pre>
        </el-alert>
      </el-tab-pane>

      <el-tab-pane :label="t('clusterPage.nodes')" name="nodes">
        <div class="toolbar">
          <el-button type="primary" @click="openAddNode">{{ t('clusterPage.addNode') }}</el-button>
        </div>
        <el-table :data="nodes" stripe>
          <el-table-column prop="name" :label="t('common.name')" min-width="120" />
          <el-table-column :label="t('clusterPage.address')" min-width="180">
            <template #default="{ row }">{{ row.host }}:{{ row.port }}<span v-if="row.safe_path">/{{ row.safe_path }}</span></template>
          </el-table-column>
          <el-table-column prop="role" :label="t('clusterPage.role')" width="90" />
          <el-table-column :label="t('common.status')" width="90">
            <template #default="{ row }"><el-tag :type="statusTag(row.status)" size="small">{{ row.status }}</el-tag></template>
          </el-table-column>
          <el-table-column :label="t('clusterPage.resources')" min-width="200">
            <template #default="{ row }">
              <div class="res-bars">
                <div class="res-row"><span>CPU</span><el-progress :percentage="Math.min(100, row.cpu_percent || 0)" :stroke-width="8" :color="resourceColor(row.cpu_percent || 0)" :show-text="false" /><span class="res-val">{{ (row.cpu_percent || 0).toFixed(0) }}%</span></div>
                <div class="res-row"><span>MEM</span><el-progress :percentage="Math.min(100, row.mem_percent || 0)" :stroke-width="8" :color="resourceColor(row.mem_percent || 0)" :show-text="false" /><span class="res-val">{{ (row.mem_percent || 0).toFixed(0) }}%</span></div>
                <div v-if="row.disk_percent" class="res-row"><span>DISK</span><el-progress :percentage="Math.min(100, row.disk_percent || 0)" :stroke-width="8" :color="resourceColor(row.disk_percent || 0)" :show-text="false" /><span class="res-val">{{ (row.disk_percent || 0).toFixed(0) }}%</span></div>
              </div>
            </template>
          </el-table-column>
          <el-table-column :label="t('clusterPage.provisionStatus')" width="100">
            <template #default="{ row }">
              <el-tag v-if="!row.is_local" :type="provisionTag(row.provision_status || 'none')" size="small">{{ row.provision_status || 'none' }}</el-tag>
              <span v-else>—</span>
            </template>
          </el-table-column>
          <el-table-column prop="hostname" :label="t('dashboard.hostname')" width="120" show-overflow-tooltip />
          <el-table-column :label="t('common.actions')" width="320" fixed="right">
            <template #default="{ row }">
              <el-button text type="primary" @click="openMonitor(row)">{{ t('clusterPage.monitor') }}</el-button>
              <el-button text type="primary" @click="testNode(row)">{{ t('clusterPage.test') }}</el-button>
              <el-button v-if="!row.is_local && (row.has_ssh_password || row.ssh_host)" text :loading="sshTestLoading === row.id" @click="testSSH(row)">{{ t('clusterPage.testSSH') }}</el-button>
              <el-button v-if="!row.is_local && row.has_ssh_password" text type="success" :loading="provisionLoading === row.id" @click="provisionNode(row)">{{ t('clusterPage.autoProvision') }}</el-button>
              <el-button v-if="!row.is_local" text @click="openEditNode(row)">{{ t('common.edit') }}</el-button>
              <el-button v-if="!row.is_local" text type="danger" @click="deleteNode(row)">{{ t('common.delete') }}</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane :label="t('clusterPage.loadBalancers')" name="lb">
        <div class="toolbar">
          <el-button type="primary" @click="openAddLB">{{ t('clusterPage.addLB') }}</el-button>
        </div>
        <el-table :data="balancers" stripe row-key="id">
          <el-table-column prop="name" :label="t('common.name')" width="140" />
          <el-table-column prop="domain" label="Domain" min-width="160" />
          <el-table-column :label="t('clusterPage.algorithm')" width="120">
            <template #default="{ row }">{{ algorithms.find(a => a.value === row.algorithm)?.label || row.algorithm }}</template>
          </el-table-column>
          <el-table-column :label="t('common.status')" width="90">
            <template #default="{ row }"><el-tag :type="statusTag(row.status)" size="small">{{ row.status }}</el-tag></template>
          </el-table-column>
          <el-table-column :label="t('clusterPage.backends')" min-width="200">
            <template #default="{ row }">
              <el-tag v-for="b in row.backends || []" :key="b.id" size="small" :type="statusTag(b.status)" class="backend-tag" closable @close="deleteBackend(row, b)">
                {{ b.host }}:{{ b.port }}
              </el-tag>
              <el-button text type="primary" size="small" @click="openAddBackend(row)">+</el-button>
            </template>
          </el-table-column>
          <el-table-column :label="t('common.actions')" width="220" fixed="right">
            <template #default="{ row }">
              <el-button text type="success" @click="applyLB(row)">{{ t('clusterPage.apply') }}</el-button>
              <el-button text @click="openEditLB(row)">{{ t('common.edit') }}</el-button>
              <el-button text type="danger" @click="deleteLB(row)">{{ t('common.delete') }}</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane :label="t('clusterPage.guide')" name="guide">
        <el-alert type="info" :closable="false" show-icon :title="t('clusterPage.guideTitle')">
          <ol class="guide-list">
            <li v-for="i in 4" :key="i">{{ t(`clusterPage.guideStep${i}`) }}</li>
          </ol>
        </el-alert>
      </el-tab-pane>

      <el-tab-pane :label="t('clusterPage.flowTab')" name="workflow">
        <el-alert type="info" :closable="false" show-icon :title="t('clusterPage.flowHint')" class="flow-hint" />
        <div class="workflow-toolbar">
          <el-button type="primary" plain @click="showAssistant = true">{{ t('clusterPage.openAssistant') }}</el-button>
        </div>
        <div v-loading="flowLoading || workflowRunning" class="workflow-layout">
          <ClusterFlowCanvas
            v-model="flowGraph"
            :cluster-nodes="nodes"
            :run-log="runLog"
            :workflow-status="workflowStatus"
            @save="saveWorkflow"
            @run="runWorkflow"
            @sync-nodes="syncWorkflowNodes"
          />
        </div>
        <el-drawer v-model="showAssistant" :title="t('clusterPage.flowAssistant')" direction="rtl" size="420px" append-to-body destroy-on-close>
          <ClusterAssistant :nodes="nodes" :graph="flowGraph" :balancers="balancers" @apply-graph="applyAssistantGraph" />
        </el-drawer>
      </el-tab-pane>
    </el-tabs>

    <el-dialog v-model="nodeDialog" :title="editingNode ? t('clusterPage.editNode') : t('clusterPage.addNode')" width="620px">
      <el-form label-width="120px">
        <el-divider content-position="left">{{ t('clusterPage.panelConn') }}</el-divider>
        <el-form-item :label="t('common.name')"><el-input v-model="nodeForm.name" /></el-form-item>
        <el-form-item :label="t('clusterPage.host')"><el-input v-model="nodeForm.host" placeholder="192.168.1.10" /></el-form-item>
        <el-form-item :label="t('common.port')"><el-input-number v-model="nodeForm.port" :min="1" :max="65535" /></el-form-item>
        <el-form-item :label="t('clusterPage.safePath')"><el-input v-model="nodeForm.safe_path" placeholder="bb276bbd" /></el-form-item>
        <el-form-item :label="t('clusterPage.agentToken')"><el-input v-model="nodeForm.agent_token" type="password" show-password /></el-form-item>
        <el-form-item :label="t('clusterPage.backendHost')"><el-input v-model="nodeForm.website_host" :placeholder="t('clusterPage.backendHostHint')" /></el-form-item>
        <el-form-item :label="t('clusterPage.backendPort')"><el-input-number v-model="nodeForm.website_port" :min="1" :max="65535" /></el-form-item>
        <el-form-item :label="t('clusterPage.role')">
          <el-select v-model="nodeForm.role"><el-option value="worker" label="Worker" /><el-option value="master" label="Master" /></el-select>
        </el-form-item>
        <el-collapse>
          <el-collapse-item :title="t('clusterPage.sshSection')" name="ssh">
            <el-form-item :label="t('clusterPage.sshHost')"><el-input v-model="nodeForm.ssh_host" :placeholder="t('clusterPage.sshHostHint')" /></el-form-item>
            <el-form-item :label="t('clusterPage.sshPort')"><el-input-number v-model="nodeForm.ssh_port" :min="1" :max="65535" /></el-form-item>
            <el-form-item :label="t('clusterPage.sshUser')"><el-input v-model="nodeForm.ssh_user" placeholder="root" /></el-form-item>
            <el-form-item :label="t('clusterPage.sshPassword')">
              <el-input v-model="nodeForm.ssh_password" type="password" show-password :placeholder="editingNode?.has_ssh_password ? t('clusterPage.sshPasswordKeep') : ''" />
            </el-form-item>
            <el-form-item :label="t('clusterPage.autoProvisionOnSave')"><el-switch v-model="nodeForm.auto_provision" /></el-form-item>
          </el-collapse-item>
        </el-collapse>
        <el-form-item :label="t('common.description')"><el-input v-model="nodeForm.remark" type="textarea" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="nodeDialog = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="saveNode">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="lbDialog" :title="editingLB ? t('clusterPage.editLB') : t('clusterPage.addLB')" width="560px">
      <el-form label-width="120px">
        <el-form-item :label="t('common.name')"><el-input v-model="lbForm.name" /></el-form-item>
        <el-form-item label="Domain"><el-input v-model="lbForm.domain" placeholder="lb.example.com" /></el-form-item>
        <el-form-item :label="t('common.port')"><el-input-number v-model="lbForm.listen_port" :min="1" :max="65535" /></el-form-item>
        <el-form-item :label="t('clusterPage.algorithm')">
          <el-select v-model="lbForm.algorithm" style="width:100%"><el-option v-for="a in algorithms" :key="a.value" :label="a.label" :value="a.value" /></el-select>
        </el-form-item>
        <el-form-item :label="t('clusterPage.sticky')"><el-switch v-model="lbForm.sticky_session" /></el-form-item>
        <el-form-item :label="t('clusterPage.websocket')"><el-switch v-model="lbForm.websocket" /></el-form-item>
        <el-form-item :label="t('clusterPage.healthCheck')"><el-switch v-model="lbForm.health_check" /></el-form-item>
        <el-form-item :label="t('common.enabled')"><el-switch v-model="lbForm.enabled" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="lbDialog = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="saveLB">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="backendDialog" :title="t('clusterPage.addBackend')" width="480px">
      <el-form label-width="100px">
        <el-form-item :label="t('clusterPage.pickNode')">
          <el-select v-model="backendForm.node_id" clearable style="width:100%" @change="(id: number) => { const n = nodes.find(x => x.id === id); if (n) { backendForm.host = n.website_host || n.host; backendForm.port = n.website_port || 80 } }">
            <el-option v-for="n in nodes" :key="n.id" :label="n.name" :value="n.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('clusterPage.host')"><el-input v-model="backendForm.host" /></el-form-item>
        <el-form-item :label="t('common.port')"><el-input-number v-model="backendForm.port" :min="1" :max="65535" /></el-form-item>
        <el-form-item :label="t('clusterPage.weight')"><el-input-number v-model="backendForm.weight" :min="1" :max="100" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="backendDialog = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="saveBackend">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="monitorDrawer" :title="t('clusterPage.monitorTitle', { name: monitorNode?.name || '' })" direction="rtl" size="400px" append-to-body>
      <div v-loading="monitorLoading">
        <div v-if="monitorData" class="monitor-grid">
          <el-card shadow="never"><div class="mon-val">{{ (monitorData.cpu || 0).toFixed(1) }}%</div><div class="mon-label">CPU</div><el-progress :percentage="Math.min(100, monitorData.cpu || 0)" :color="resourceColor(monitorData.cpu || 0)" /></el-card>
          <el-card shadow="never"><div class="mon-val">{{ (monitorData.memory || 0).toFixed(1) }}%</div><div class="mon-label">{{ t('clusterPage.memory') }}</div><el-progress :percentage="Math.min(100, monitorData.memory || 0)" :color="resourceColor(monitorData.memory || 0)" /></el-card>
          <el-card shadow="never"><div class="mon-val">{{ (monitorData.disk || 0).toFixed(1) }}%</div><div class="mon-label">{{ t('clusterPage.disk') }}</div><el-progress :percentage="Math.min(100, monitorData.disk || 0)" :color="resourceColor(monitorData.disk || 0)" /></el-card>
          <el-card shadow="never"><div class="mon-val">{{ (monitorData.load1 || 0).toFixed(2) }}</div><div class="mon-label">{{ t('clusterPage.load') }}</div></el-card>
        </div>
        <el-descriptions v-if="monitorData" :column="1" border size="small" class="mon-meta">
          <el-descriptions-item :label="t('dashboard.hostname')">{{ monitorData.hostname || '—' }}</el-descriptions-item>
          <el-descriptions-item :label="t('clusterPage.collectedAt')">{{ monitorData.collected_at || '—' }}</el-descriptions-item>
          <el-descriptions-item :label="t('clusterPage.monitorVia')">{{ monitorData.via_ssh ? 'SSH' : t('clusterPage.monitorLocal') }}</el-descriptions-item>
        </el-descriptions>
        <el-button type="primary" plain style="margin-top:16px;width:100%" @click="refreshMonitor">{{ t('clusterPage.refreshMonitor') }}</el-button>
        <el-alert v-if="monitorNode?.provision_log" type="info" :closable="false" show-icon :title="t('clusterPage.provisionLog')" style="margin-top:16px">
          <pre class="provision-log">{{ monitorNode.provision_log }}</pre>
        </el-alert>
      </div>
    </el-drawer>
  </div>
</template>

<style scoped>
.cluster-page { width: 100%; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 16px; flex-wrap: wrap; gap: 12px; }
.page-header h2 { margin: 0 0 4px; }
.hint { margin: 0; font-size: 13px; color: var(--el-text-color-secondary); }
.overview-row { margin-bottom: 16px; }
.stat-val { font-size: 22px; font-weight: 700; }
.stat-label { font-size: 13px; color: var(--el-text-color-secondary); margin-top: 4px; }
.token-preview { font-size: 12px; }
.token-row { display: flex; align-items: center; gap: 8px; margin-top: 6px; flex-wrap: wrap; }
.toolbar { margin-bottom: 12px; }
.backend-tag { margin-right: 6px; margin-bottom: 4px; }
.guide-list { margin: 8px 0 0; padding-left: 18px; line-height: 1.8; }
.quick-intro { margin: 0 0 16px; color: var(--el-text-color-secondary); font-size: 14px; }
.quick-card { margin-bottom: 16px; }
.node-checks { display: flex; flex-direction: column; gap: 6px; }
.empty-hint { margin: 0; font-size: 12px; color: var(--el-color-warning); }
.quick-log pre { margin: 8px 0 0; font-size: 12px; white-space: pre-wrap; }
.res-bars { font-size: 12px; }
.res-row { display: flex; align-items: center; gap: 6px; margin-bottom: 4px; }
.res-row span:first-child { width: 32px; color: var(--el-text-color-secondary); }
.res-row .el-progress { flex: 1; }
.res-val { width: 36px; text-align: right; font-size: 11px; }
.monitor-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; margin-bottom: 16px; }
.mon-val { font-size: 22px; font-weight: 700; }
.mon-label { font-size: 12px; color: var(--el-text-color-secondary); margin-bottom: 8px; }
.mon-meta { margin-top: 8px; }
.provision-log { margin: 0; font-size: 11px; white-space: pre-wrap; max-height: 200px; overflow: auto; }
.join-card { margin-bottom: 16px; }
.join-intro { margin: 0 0 12px; font-size: 13px; color: var(--el-text-color-secondary); }
.join-alert { margin-bottom: 8px; }
.join-row { display: flex; align-items: flex-start; gap: 10px; margin-bottom: 10px; flex-wrap: wrap; }
.join-row-main { margin-bottom: 8px; }
.join-more { margin-top: 4px; }
.join-label { width: 160px; flex-shrink: 0; font-size: 13px; font-weight: 600; padding-top: 6px; }
.join-cmd { flex: 1; min-width: 200px; font-size: 11px; padding: 8px; background: var(--el-fill-color-light); border-radius: 6px; word-break: break-all; }
.flow-hint { margin-bottom: 12px; }
.workflow-toolbar { margin-bottom: 12px; }
.workflow-layout { min-height: 560px; }
</style>
