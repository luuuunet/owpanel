<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import api from '@/api'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Folder, Plus, Delete } from '@element-plus/icons-vue'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const tabs = ['php', 'java', 'nodejs', 'go', 'python', 'dotnet'] as const
type RuntimeKind = (typeof tabs)[number]

const activeTab = ref<RuntimeKind>('dotnet')
const loading = ref(false)
const projects = ref<any[]>([])
const phpVersions = ref<any[]>([])
const versions = ref<string[]>([])

const dialogVisible = ref(false)
const configTab = ref('port')
const submitting = ref(false)

const pathPickerVisible = ref(false)
const pathEntries = ref<any[]>([])
const browsePath = ref('/')

interface PortRow { host_port: number; container_port: number; protocol: string }
interface EnvRow { key: string; value: string }
interface MountRow { host: string; container: string; read_only: boolean }
interface HostRow { host: string; ip: string }

const emptyForm = () => ({
  name: '',
  kind: activeTab.value as RuntimeKind,
  path: '',
  version: '',
  run_script: '',
  container_name: '',
  remark: '',
  ports: [{ host_port: 8080, container_port: 8080, protocol: 'tcp' }] as PortRow[],
  env_vars: [] as EnvRow[],
  mounts: [] as MountRow[],
  host_mappings: [] as HostRow[],
})

const form = ref(emptyForm())

const runScriptPlaceholder = computed(() => {
  const map: Record<string, string> = {
    dotnet: 'dotnet MyWebApp.dll',
    nodejs: 'node index.js',
    python: 'python app.py',
    go: './app',
    java: 'java -jar app.jar',
    php: 'php-fpm',
  }
  return map[form.value.kind] || t('runtime.runScriptHint')
})

const isPhpTab = computed(() => activeTab.value === 'php')
const tableData = computed(() => (isPhpTab.value ? phpVersions.value : projects.value))

async function loadVersions(kind: string) {
  try {
    const res: any = await api.get('/runtimes/versions', { params: { kind } })
    versions.value = res.data || []
    if (versions.value.length && !form.value.version) {
      form.value.version = versions.value[0]
    }
  } catch {
    versions.value = []
  }
}

async function loadProjects() {
  loading.value = true
  try {
    if (isPhpTab.value) {
      const res: any = await api.get('/php/versions')
      phpVersions.value = (res.data || []).map((v: any) => ({
        id: v.key,
        name: `PHP ${v.version}`,
        path: v.install_path || '-',
        version: v.version,
        external_port: v.port || '-',
        status: v.status,
        key: v.key,
      }))
      return
    }
    const res: any = await api.get('/runtimes', { params: { kind: activeTab.value } })
    projects.value = res.data || []
  } finally {
    loading.value = false
  }
}

function syncTabFromRoute() {
  const tab = String(route.query.tab || 'dotnet').toLowerCase()
  if (tabs.includes(tab as RuntimeKind)) {
    activeTab.value = tab as RuntimeKind
  }
}

watch(activeTab, (tab) => {
  router.replace({ query: { ...route.query, tab } })
  loadProjects()
})

watch(() => route.query.tab, syncTabFromRoute)

function openCreate() {
  if (isPhpTab.value) {
    router.push('/php')
    return
  }
  form.value = emptyForm()
  form.value.kind = activeTab.value
  configTab.value = 'port'
  loadVersions(activeTab.value)
  dialogVisible.value = true
}

async function loadPathDir(path: string) {
  const res: any = await api.get('/files', { params: { path } })
  pathEntries.value = res.data?.items || res.data || []
  browsePath.value = path
}

function openPathPicker() {
  pathPickerVisible.value = true
  browsePath.value = form.value.path || '/'
  loadPathDir(browsePath.value)
}

function selectPath(path: string) {
  form.value.path = path
  pathPickerVisible.value = false
}

function addPort() {
  form.value.ports.push({ host_port: 8080, container_port: 8080, protocol: 'tcp' })
}
function removePort(i: number) {
  form.value.ports.splice(i, 1)
}
function addEnv() {
  form.value.env_vars.push({ key: '', value: '' })
}
function removeEnv(i: number) {
  form.value.env_vars.splice(i, 1)
}
function addMount() {
  form.value.mounts.push({ host: '', container: '', read_only: false })
}
function removeMount(i: number) {
  form.value.mounts.splice(i, 1)
}
function addHost() {
  form.value.host_mappings.push({ host: '', ip: '' })
}
function removeHost(i: number) {
  form.value.host_mappings.splice(i, 1)
}

async function handleCreate() {
  if (!form.value.name.trim()) {
    ElMessage.warning(t('runtime.nameRequired'))
    return
  }
  if (!form.value.path.trim()) {
    ElMessage.warning(t('runtime.pathRequired'))
    return
  }
  if (!form.value.run_script.trim() && activeTab.value !== 'nodejs') {
    ElMessage.warning(t('runtime.scriptRequired'))
    return
  }
  submitting.value = true
  try {
    const payload = {
      kind: form.value.kind,
      name: form.value.name.trim(),
      path: form.value.path.trim(),
      version: form.value.version,
      run_script: form.value.run_script.trim(),
      container_name: form.value.container_name.trim(),
      remark: form.value.remark.trim(),
      external_port: form.value.ports[0]?.host_port || 0,
      ports: JSON.stringify(form.value.ports),
      env_vars: JSON.stringify(form.value.env_vars.filter(e => e.key)),
      mounts: JSON.stringify(form.value.mounts.filter(m => m.host && m.container)),
      host_mappings: JSON.stringify(form.value.host_mappings.filter(h => h.host && h.ip)),
    }
    await api.post('/runtimes', payload)
    ElMessage.success(t('runtime.created'))
    dialogVisible.value = false
    loadProjects()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || t('common.failed'))
  } finally {
    submitting.value = false
  }
}

async function toggle(row: any) {
  if (isPhpTab.value) {
    const action = row.status === 'running' ? 'stop' : 'start'
    await api.post(`/php/${row.key}/${action}`)
    ElMessage.success(t('common.updated'))
    loadProjects()
    return
  }
  const status = row.status === 'running' ? 'stopped' : 'running'
  await api.patch(`/runtimes/${row.id}/toggle`, {
    status,
    legacy_source: row.legacy_source || '',
    legacy_id: row.legacy_id || 0,
  })
  ElMessage.success(t('common.updated'))
  loadProjects()
}

async function handleDelete(row: any) {
  await ElMessageBox.confirm(t('runtime.deleteConfirm'), t('common.warning'), { type: 'warning' })
  const params: Record<string, string> = {}
  if (row.legacy_source) {
    params.legacy_source = row.legacy_source
    params.legacy_id = String(row.legacy_id || 0)
  }
  await api.delete(`/runtimes/${row.id}`, { params })
  ElMessage.success(t('common.deleted'))
  loadProjects()
}

onMounted(() => {
  syncTabFromRoute()
  loadProjects()
})
</script>

<template>
  <div class="runtimes-page">
    <div class="page-header">
      <h2>{{ t('runtime.title') }}</h2>
      <el-button type="primary" @click="openCreate">
        {{ isPhpTab ? t('runtime.managePhp') : t('runtime.create') }}
      </el-button>
    </div>

    <el-tabs v-model="activeTab" class="runtime-tabs">
      <el-tab-pane v-for="tab in tabs" :key="tab" :label="t(`runtime.tab.${tab}`)" :name="tab" />
    </el-tabs>

    <el-alert v-if="!isPhpTab" type="info" :closable="false" show-icon class="hint">
      {{ t('runtime.hint') }}
    </el-alert>

    <el-table v-loading="loading" :data="tableData" stripe>
      <el-table-column prop="name" :label="t('runtime.colName')" min-width="140" />
      <el-table-column prop="path" :label="t('runtime.colPath')" min-width="200" show-overflow-tooltip />
      <el-table-column prop="version" :label="t('runtime.colVersion')" width="100" />
      <el-table-column prop="external_port" :label="t('runtime.colPort')" width="110" />
      <el-table-column :label="t('common.status')" width="100">
        <template #default="{ row }">
          <el-tag :type="row.status === 'running' ? 'success' : 'info'">{{ row.status }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column v-if="!isPhpTab" prop="container_name" :label="t('runtime.containerName')" min-width="140" show-overflow-tooltip />
      <el-table-column v-if="!isPhpTab" prop="remark" :label="t('runtime.remark')" min-width="120" show-overflow-tooltip />
      <el-table-column :label="t('common.actions')" width="180" fixed="right">
        <template #default="{ row }">
          <el-button text type="primary" @click="toggle(row)">
            {{ row.status === 'running' ? t('common.stop') : t('common.start') }}
          </el-button>
          <el-button v-if="!isPhpTab && row.legacy_source !== 'java'" text type="danger" @click="handleDelete(row)">
            {{ t('common.delete') }}
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-empty v-if="!loading && !tableData.length" :description="t('runtime.empty')" />

    <el-drawer v-model="dialogVisible" :title="t('runtime.createTitle')" size="520px" direction="rtl">
      <el-form :model="form" label-position="top">
        <el-form-item :label="t('runtime.name')" required>
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item :label="t('runtime.application')" required>
          <div class="app-row">
            <el-select v-model="form.kind" disabled style="flex: 1">
              <el-option v-for="tab in tabs.filter(k => k !== 'php')" :key="tab" :label="t(`runtime.tab.${tab}`)" :value="tab" />
            </el-select>
            <el-select v-model="form.version" style="flex: 1">
              <el-option v-for="v in versions" :key="v" :label="v" :value="v" />
            </el-select>
          </div>
        </el-form-item>
        <el-form-item :label="t('runtime.codeDir')" required>
          <el-input v-model="form.path">
            <template #append>
              <el-button :icon="Folder" @click="openPathPicker" />
            </template>
          </el-input>
        </el-form-item>
        <el-form-item :label="t('runtime.runScript')" required>
          <el-input v-model="form.run_script" :placeholder="runScriptPlaceholder" />
        </el-form-item>
        <el-form-item :label="t('runtime.containerName')" required>
          <el-input v-model="form.container_name" :placeholder="t('runtime.containerHint')" />
        </el-form-item>
        <el-form-item :label="t('runtime.remark')">
          <el-input v-model="form.remark" type="textarea" :rows="2" />
        </el-form-item>

        <el-tabs v-model="configTab" class="config-tabs">
          <el-tab-pane :label="t('runtime.config.port')" name="port">
            <div v-for="(p, i) in form.ports" :key="i" class="config-row">
              <el-input-number v-model="p.host_port" :min="1" :max="65535" :placeholder="t('runtime.hostPort')" />
              <el-input-number v-model="p.container_port" :min="1" :max="65535" :placeholder="t('runtime.containerPort')" />
              <el-select v-model="p.protocol" style="width: 90px">
                <el-option label="TCP" value="tcp" />
                <el-option label="UDP" value="udp" />
              </el-select>
              <el-button :icon="Delete" text type="danger" @click="removePort(i)" />
            </div>
            <el-button :icon="Plus" text type="primary" @click="addPort">{{ t('runtime.add') }}</el-button>
          </el-tab-pane>
          <el-tab-pane :label="t('runtime.config.env')" name="env">
            <div v-for="(e, i) in form.env_vars" :key="i" class="config-row">
              <el-input v-model="e.key" :placeholder="t('runtime.envKey')" />
              <el-input v-model="e.value" :placeholder="t('runtime.envValue')" />
              <el-button :icon="Delete" text type="danger" @click="removeEnv(i)" />
            </div>
            <el-button :icon="Plus" text type="primary" @click="addEnv">{{ t('runtime.add') }}</el-button>
          </el-tab-pane>
          <el-tab-pane :label="t('runtime.config.mount')" name="mount">
            <div v-for="(m, i) in form.mounts" :key="i" class="config-row mount-row">
              <el-input v-model="m.host" :placeholder="t('runtime.mountHost')" />
              <el-input v-model="m.container" :placeholder="t('runtime.mountContainer')" />
              <el-checkbox v-model="m.read_only">{{ t('runtime.readOnly') }}</el-checkbox>
              <el-button :icon="Delete" text type="danger" @click="removeMount(i)" />
            </div>
            <el-button :icon="Plus" text type="primary" @click="addMount">{{ t('runtime.add') }}</el-button>
          </el-tab-pane>
          <el-tab-pane :label="t('runtime.config.host')" name="host">
            <div v-for="(h, i) in form.host_mappings" :key="i" class="config-row">
              <el-input v-model="h.host" :placeholder="t('runtime.hostName')" />
              <el-input v-model="h.ip" placeholder="127.0.0.1" />
              <el-button :icon="Delete" text type="danger" @click="removeHost(i)" />
            </div>
            <el-button :icon="Plus" text type="primary" @click="addHost">{{ t('runtime.add') }}</el-button>
          </el-tab-pane>
        </el-tabs>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="submitting" @click="handleCreate">{{ t('common.create') }}</el-button>
      </template>
    </el-drawer>

    <el-dialog v-model="pathPickerVisible" :title="t('runtime.selectPath')" width="520px">
      <div class="path-bar">{{ browsePath }}</div>
      <el-table :data="pathEntries" stripe max-height="320" @row-dblclick="(row: any) => row.is_dir && loadPathDir(row.path)">
        <el-table-column prop="name" label="Name" />
        <el-table-column width="80">
          <template #default="{ row }">
            <el-button v-if="row.is_dir" text type="primary" @click="loadPathDir(row.path)">Open</el-button>
          </template>
        </el-table-column>
      </el-table>
      <template #footer>
        <el-button @click="pathPickerVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="selectPath(browsePath)">{{ t('common.confirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.runtimes-page .hint { margin-bottom: 12px; }
.runtime-tabs { margin-bottom: 12px; }
.app-row { display: flex; gap: 8px; width: 100%; }
.config-tabs { margin-top: 8px; }
.config-row { display: flex; gap: 8px; align-items: center; margin-bottom: 8px; flex-wrap: wrap; }
.mount-row { align-items: flex-start; }
.path-bar { font-family: monospace; margin-bottom: 8px; word-break: break-all; }
</style>
