<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import api, { resolveApiError } from '@/api'
import { ElMessage, ElMessageBox } from 'element-plus'
import DockerExecPane from '@/components/DockerExecPane.vue'

interface PortMapping {
  host_ip: string
  host_port: string
  container_port: string
  protocol: string
}

interface MountMapping {
  type: string
  source: string
  destination: string
  read_only: boolean
}

interface ContainerDetail {
  id: string
  name: string
  image: string
  image_id?: string
  status: string
  created?: string
  ports: PortMapping[]
  env: string[]
  mounts: MountMapping[]
  networks: string[]
  restart_policy: string
  command: string[]
  working_dir: string
}

interface ContainerStats {
  cpu_perc: string
  mem_usage: string
  mem_perc: string
  net_io: string
  block_io: string
  pids: string
  cpu_percent: number
  mem_percent: number
}

const props = defineProps<{
  modelValue: boolean
  containerId: string
  containerName?: string
  containerStatus?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [v: boolean]
  changed: []
}>()

const { t } = useI18n()
const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

const activeTab = ref('overview')
const loading = ref(false)
const saving = ref(false)
const actionLoading = ref(false)
const detail = ref<ContainerDetail | null>(null)
const editPorts = ref<PortMapping[]>([])
const editEnvText = ref('')
const editRestart = ref('no')
const editName = ref('')

const logsContent = ref('')
const logsLoading = ref(false)
const logsTail = ref(500)
const logsTimestamps = ref(true)
let logsTimer: ReturnType<typeof setInterval> | null = null
const logsAuto = ref(false)

const stats = ref<ContainerStats | null>(null)
const statsLoading = ref(false)
let statsTimer: ReturnType<typeof setInterval> | null = null

const inspectJson = ref('')
const inspectLoading = ref(false)

const execCmd = ref('ls -la')
const execOutput = ref('')
const execLoading = ref(false)

const isRunning = computed(() => {
  const s = (detail.value?.status || props.containerStatus || '').toLowerCase()
  return s.includes('up') && !s.includes('paused')
})
const isPaused = computed(() => (detail.value?.status || props.containerStatus || '').toLowerCase().includes('paused'))

async function loadDetail() {
  if (!props.containerId) return
  loading.value = true
  try {
    const res: any = await api.get(`/docker/containers/${props.containerId}`)
    detail.value = res.data
    editPorts.value = (res.data.ports || []).map((p: PortMapping) => ({ ...p, protocol: p.protocol || 'tcp' }))
    if (!editPorts.value.length) {
      editPorts.value = [{ host_ip: '', host_port: '', container_port: '', protocol: 'tcp' }]
    }
    editEnvText.value = (res.data.env || []).join('\n')
    editRestart.value = res.data.restart_policy || 'no'
    editName.value = (res.data.name || '').replace(/^\//, '')
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
    visible.value = false
  } finally {
    loading.value = false
  }
}

async function loadLogs() {
  if (!props.containerId) return
  logsLoading.value = true
  try {
    const res: any = await api.get(`/docker/containers/${props.containerId}/logs`, {
      params: { tail: logsTail.value, timestamps: logsTimestamps.value ? '1' : '0' },
    })
    logsContent.value = res.data?.content || t('docker.noLogs')
  } catch (e: any) {
    logsContent.value = resolveApiError(e, t('common.failed'))
  } finally {
    logsLoading.value = false
  }
}

async function loadStats() {
  if (!props.containerId || !isRunning.value) {
    stats.value = null
    return
  }
  statsLoading.value = true
  try {
    const res: any = await api.get(`/docker/containers/${props.containerId}/stats`)
    stats.value = res.data
  } catch {
    stats.value = null
  } finally {
    statsLoading.value = false
  }
}

async function loadInspect() {
  if (!props.containerId) return
  inspectLoading.value = true
  try {
    const res: any = await api.get(`/docker/containers/${props.containerId}/inspect`)
    inspectJson.value = JSON.stringify(res.data?.inspect ?? res.data, null, 2)
  } catch (e: any) {
    inspectJson.value = resolveApiError(e, t('common.failed'))
  } finally {
    inspectLoading.value = false
  }
}

function stopLogsAuto() {
  if (logsTimer) {
    clearInterval(logsTimer)
    logsTimer = null
  }
  logsAuto.value = false
}

function startLogsAuto() {
  stopLogsAuto()
  logsAuto.value = true
  logsTimer = setInterval(() => { loadLogs() }, 3000)
}

function stopStatsAuto() {
  if (statsTimer) {
    clearInterval(statsTimer)
    statsTimer = null
  }
}

function startStatsAuto() {
  stopStatsAuto()
  loadStats()
  statsTimer = setInterval(() => { loadStats() }, 2500)
}

async function doAction(action: 'start' | 'stop' | 'restart' | 'pause' | 'unpause' | 'kill') {
  if (!props.containerId) return
  if (action === 'kill') {
    try {
      await ElMessageBox.confirm(t('docker.killConfirm', { name: detail.value?.name || props.containerName }), t('common.warning'), { type: 'warning' })
    } catch { return }
  }
  actionLoading.value = true
  try {
    if (action === 'kill') {
      await api.post(`/docker/containers/${props.containerId}/kill`, { signal: 'SIGKILL' })
    } else {
      await api.post(`/docker/containers/${props.containerId}/${action}`)
    }
    ElMessage.success(t('common.success'))
    await loadDetail()
    emit('changed')
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
  } finally {
    actionLoading.value = false
  }
}

async function renameContainer() {
  const name = editName.value.trim()
  if (!name || !props.containerId) return
  actionLoading.value = true
  try {
    await api.post(`/docker/containers/${props.containerId}/rename`, { name })
    ElMessage.success(t('docker.renamed'))
    await loadDetail()
    emit('changed')
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
  } finally {
    actionLoading.value = false
  }
}

function addPortRow() {
  editPorts.value.push({ host_ip: '', host_port: '', container_port: '', protocol: 'tcp' })
}
function removePortRow(i: number) {
  editPorts.value.splice(i, 1)
}

async function saveAndRecreate() {
  if (!detail.value || !props.containerId) return
  try {
    await ElMessageBox.confirm(t('docker.recreateConfirm'), t('common.warning'), { type: 'warning' })
  } catch { return }
  saving.value = true
  try {
    const ports = editPorts.value.filter((p) => p.host_port && p.container_port)
    const env = editEnvText.value.split('\n').map((l) => l.trim()).filter(Boolean)
    await api.post(`/docker/containers/${props.containerId}/recreate`, {
      ports,
      env,
      restart_policy: editRestart.value,
    })
    ElMessage.success(t('docker.recreateSuccess'))
    visible.value = false
    emit('changed')
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
  } finally {
    saving.value = false
  }
}

async function runExecOnce() {
  const cmd = execCmd.value.trim()
  if (!cmd || !props.containerId) return
  execLoading.value = true
  execOutput.value = ''
  try {
    const res: any = await api.post(`/docker/containers/${props.containerId}/exec`, { cmd })
    execOutput.value = res.data?.output || ''
  } catch (e: any) {
    execOutput.value = resolveApiError(e, t('common.failed'))
  } finally {
    execLoading.value = false
  }
}

async function copyInspect() {
  try {
    await navigator.clipboard.writeText(inspectJson.value)
    ElMessage.success(t('docker.pathCopied'))
  } catch {
    ElMessage.error(t('common.failed'))
  }
}

watch(visible, (v) => {
  if (v) {
    activeTab.value = 'overview'
    loadDetail()
  } else {
    stopLogsAuto()
    stopStatsAuto()
  }
})

watch(activeTab, (tab) => {
  stopLogsAuto()
  stopStatsAuto()
  if (!visible.value) return
  if (tab === 'logs') loadLogs()
  if (tab === 'stats') startStatsAuto()
  if (tab === 'inspect') loadInspect()
})

onBeforeUnmount(() => {
  stopLogsAuto()
  stopStatsAuto()
})
</script>

<template>
  <el-drawer
    v-model="visible"
    :title="detail?.name || containerName || t('docker.containerDetail')"
    size="720px"
    destroy-on-close
    class="docker-container-drawer"
  >
    <div v-loading="loading" class="drawer-body">
      <div class="lifecycle">
        <el-button size="small" type="success" :loading="actionLoading" :disabled="isRunning" @click="doAction('start')">{{ t('common.start') }}</el-button>
        <el-button size="small" type="warning" :loading="actionLoading" :disabled="!isRunning" @click="doAction('stop')">{{ t('common.stop') }}</el-button>
        <el-button size="small" :loading="actionLoading" :disabled="!isRunning" @click="doAction('restart')">{{ t('docker.restart') }}</el-button>
        <el-button size="small" :loading="actionLoading" :disabled="!isRunning || isPaused" @click="doAction('pause')">{{ t('docker.pause') }}</el-button>
        <el-button size="small" :loading="actionLoading" :disabled="!isPaused" @click="doAction('unpause')">{{ t('docker.unpause') }}</el-button>
        <el-button size="small" type="danger" plain :loading="actionLoading" @click="doAction('kill')">{{ t('docker.kill') }}</el-button>
      </div>

      <el-tabs v-model="activeTab" class="detail-tabs">
        <el-tab-pane :label="t('docker.tabOverview')" name="overview">
          <template v-if="detail">
            <el-descriptions :column="1" border size="small">
              <el-descriptions-item :label="t('common.name')">{{ detail.name }}</el-descriptions-item>
              <el-descriptions-item label="ID">{{ detail.id }}</el-descriptions-item>
              <el-descriptions-item :label="t('docker.image')">{{ detail.image }}</el-descriptions-item>
              <el-descriptions-item :label="t('common.status')">{{ detail.status }}</el-descriptions-item>
              <el-descriptions-item :label="t('docker.restartPolicy')">{{ detail.restart_policy || 'no' }}</el-descriptions-item>
              <el-descriptions-item :label="t('docker.command')">{{ (detail.command || []).join(' ') || '—' }}</el-descriptions-item>
              <el-descriptions-item :label="t('docker.workingDir')">{{ detail.working_dir || '—' }}</el-descriptions-item>
              <el-descriptions-item :label="t('docker.networks')">{{ (detail.networks || []).join(', ') || '—' }}</el-descriptions-item>
            </el-descriptions>

            <h4>{{ t('docker.portMappings') }}</h4>
            <el-table v-if="detail.ports?.length" :data="detail.ports" size="small" stripe>
              <el-table-column prop="host_port" :label="t('docker.hostPort')" width="100" />
              <el-table-column prop="container_port" :label="t('docker.containerPort')" width="110" />
              <el-table-column prop="protocol" label="Proto" width="70" />
              <el-table-column prop="host_ip" label="IP" min-width="100" />
            </el-table>
            <el-empty v-else :description="t('docker.noPortMapping')" :image-size="48" />

            <h4>{{ t('docker.volumeMounts') }}</h4>
            <el-table v-if="detail.mounts?.length" :data="detail.mounts" size="small" stripe>
              <el-table-column prop="source" :label="t('common.path')" min-width="160" show-overflow-tooltip />
              <el-table-column prop="destination" :label="t('docker.mountTarget')" min-width="140" show-overflow-tooltip />
              <el-table-column prop="type" :label="t('common.type')" width="80" />
            </el-table>
            <p v-else class="muted">—</p>

            <h4>{{ t('docker.rename') }}</h4>
            <div class="inline-row">
              <el-input v-model="editName" style="max-width: 280px" />
              <el-button :loading="actionLoading" @click="renameContainer">{{ t('common.save') }}</el-button>
            </div>
          </template>
        </el-tab-pane>

        <el-tab-pane :label="t('docker.logs')" name="logs">
          <div class="toolbar">
            <el-select v-model="logsTail" style="width: 120px" @change="loadLogs">
              <el-option :label="'100'" :value="100" />
              <el-option :label="'500'" :value="500" />
              <el-option :label="'1000'" :value="1000" />
              <el-option :label="'2000'" :value="2000" />
            </el-select>
            <el-checkbox v-model="logsTimestamps" @change="loadLogs">{{ t('docker.logsTimestamps') }}</el-checkbox>
            <el-button size="small" :loading="logsLoading" @click="loadLogs">{{ t('common.refresh') }}</el-button>
            <el-button size="small" :type="logsAuto ? 'warning' : 'primary'" plain @click="logsAuto ? stopLogsAuto() : startLogsAuto()">
              {{ logsAuto ? t('docker.logsAutoStop') : t('docker.logsAuto') }}
            </el-button>
          </div>
          <pre v-loading="logsLoading" class="log-pre">{{ logsContent }}</pre>
        </el-tab-pane>

        <el-tab-pane :label="t('docker.tabStats')" name="stats">
          <div v-loading="statsLoading">
            <template v-if="stats">
              <div class="stats-grid">
                <div class="stat-card">
                  <div class="stat-label">CPU</div>
                  <div class="stat-value">{{ stats.cpu_perc }}</div>
                  <el-progress :percentage="Math.min(100, stats.cpu_percent || 0)" :stroke-width="8" />
                </div>
                <div class="stat-card">
                  <div class="stat-label">Memory</div>
                  <div class="stat-value">{{ stats.mem_usage }}</div>
                  <el-progress :percentage="Math.min(100, stats.mem_percent || 0)" :stroke-width="8" status="success" />
                </div>
                <div class="stat-card">
                  <div class="stat-label">Network I/O</div>
                  <div class="stat-value sm">{{ stats.net_io }}</div>
                </div>
                <div class="stat-card">
                  <div class="stat-label">Block I/O</div>
                  <div class="stat-value sm">{{ stats.block_io }}</div>
                </div>
                <div class="stat-card">
                  <div class="stat-label">PIDs</div>
                  <div class="stat-value">{{ stats.pids }}</div>
                </div>
              </div>
            </template>
            <el-empty v-else :description="t('docker.statsUnavailable')" :image-size="64" />
          </div>
        </el-tab-pane>

        <el-tab-pane :label="t('docker.tabConsole')" name="console">
          <DockerExecPane v-if="visible && activeTab === 'console'" :container-id="containerId" :active="activeTab === 'console'" />
          <div class="exec-once">
            <h4>{{ t('docker.execOnce') }}</h4>
            <div class="inline-row">
              <el-input v-model="execCmd" :placeholder="t('docker.execPlaceholder')" @keyup.enter="runExecOnce" />
              <el-button type="primary" :loading="execLoading" @click="runExecOnce">{{ t('docker.execRun') }}</el-button>
            </div>
            <pre v-if="execOutput" class="log-pre exec-out">{{ execOutput }}</pre>
          </div>
        </el-tab-pane>

        <el-tab-pane :label="t('docker.tabEnv')" name="env">
          <el-alert type="info" :closable="false" show-icon class="hint">{{ t('docker.portChangeHint') }}</el-alert>
          <h4>{{ t('docker.portMappings') }}</h4>
          <div v-for="(p, i) in editPorts" :key="i" class="port-row">
            <el-input v-model="p.host_port" :placeholder="t('docker.hostPort')" />
            <span>→</span>
            <el-input v-model="p.container_port" :placeholder="t('docker.containerPort')" />
            <el-select v-model="p.protocol" style="width: 90px">
              <el-option label="TCP" value="tcp" />
              <el-option label="UDP" value="udp" />
            </el-select>
            <el-button text type="danger" @click="removePortRow(i)">{{ t('common.delete') }}</el-button>
          </div>
          <el-button text type="primary" @click="addPortRow">+ {{ t('docker.addPort') }}</el-button>

          <h4>{{ t('docker.envVars') }}</h4>
          <el-input v-model="editEnvText" type="textarea" :rows="8" :placeholder="t('docker.envPlaceholder')" />

          <h4>{{ t('docker.restartPolicy') }}</h4>
          <el-select v-model="editRestart" style="width: 220px">
            <el-option :label="t('docker.restartNo')" value="no" />
            <el-option :label="t('docker.restartAlways')" value="always" />
            <el-option :label="t('docker.restartUnlessStopped')" value="unless-stopped" />
          </el-select>

          <div class="footer-actions">
            <el-button type="primary" :loading="saving" @click="saveAndRecreate">{{ t('docker.saveAndRecreate') }}</el-button>
          </div>
        </el-tab-pane>

        <el-tab-pane :label="t('docker.tabInspect')" name="inspect">
          <div class="toolbar">
            <el-button size="small" :loading="inspectLoading" @click="loadInspect">{{ t('common.refresh') }}</el-button>
            <el-button size="small" @click="copyInspect">{{ t('docker.copyJson') }}</el-button>
          </div>
          <pre v-loading="inspectLoading" class="log-pre inspect">{{ inspectJson }}</pre>
        </el-tab-pane>
      </el-tabs>
    </div>
  </el-drawer>
</template>

<style scoped>
.drawer-body { min-height: 200px; }
.lifecycle { display: flex; flex-wrap: wrap; gap: 6px; margin-bottom: 12px; }
.detail-tabs { margin-top: 4px; }
.toolbar { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; margin-bottom: 10px; }
.log-pre {
  max-height: 55vh; overflow: auto; background: #1e1e1e; color: #d4d4d4;
  padding: 12px; border-radius: 6px; font-size: 12px; line-height: 1.5;
  white-space: pre-wrap; word-break: break-all; margin: 0;
}
.log-pre.inspect { max-height: 62vh; }
.stats-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 10px; }
.stat-card {
  padding: 14px; border-radius: 10px; border: 1px solid var(--el-border-color-lighter);
  background: var(--el-fill-color-blank);
}
.stat-label { font-size: 12px; color: var(--el-text-color-secondary); margin-bottom: 4px; }
.stat-value { font-size: 22px; font-weight: 600; margin-bottom: 8px; }
.stat-value.sm { font-size: 14px; font-weight: 500; word-break: break-all; }
.port-row { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; }
.port-row .el-input { flex: 1; }
.inline-row { display: flex; gap: 8px; align-items: center; }
.hint { margin-bottom: 12px; }
.footer-actions { margin-top: 16px; }
.exec-once { margin-top: 16px; border-top: 1px solid var(--el-border-color-lighter); padding-top: 12px; }
.exec-out { margin-top: 8px; max-height: 200px; }
h4 { margin: 16px 0 8px; font-size: 14px; }
.muted { color: var(--el-text-color-placeholder); }
@media (max-width: 640px) {
  .stats-grid { grid-template-columns: 1fr; }
}
</style>
