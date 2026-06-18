<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import api, { resolveApiError } from '@/api'
import SoftwareInstallLogDialog from '@/components/SoftwareInstallLogDialog.vue'
import { ElMessage, ElMessageBox } from 'element-plus'

interface ServerInfo {
  key: string
  name: string
  version: string
  status: string
  installed: boolean
  config_path: string
  vhost_dir: string
  sites_enabled: number
  is_active: boolean
  binary: string
}

interface Overview {
  active: string
  servers: ServerInfo[]
}

interface ReadinessCheck {
  key: string
  label: string
  status: string
  detail?: string
  group: string
}

interface ReadinessReport {
  score: number
  checks: ReadinessCheck[]
}

interface StackDef {
  key: string
  name: string
  description: string
  components: string[]
}

const props = withDefaults(defineProps<{ embedded?: boolean }>(), { embedded: false })

const { t } = useI18n()
const router = useRouter()
const overview = ref<Overview | null>(null)
const readiness = ref<ReadinessReport | null>(null)
const stacks = ref<StackDef[]>([])
const loading = ref(false)
const stackLoading = ref('')
const actionKey = ref('')
const configDrawer = ref(false)
const configKey = ref('')
const configContent = ref('')
const configLoading = ref(false)
const testOutput = ref('')
const lnmpLoading = ref(false)

const installLogVisible = ref(false)
const installLogKey = ref('')
const installLogName = ref('')
const installTrigger = ref(false)

const displayServers = computed(() => {
  if (!overview.value) return []
  const order = ['nginx', 'openresty']
  return order
    .map((k) => overview.value!.servers.find((s) => s.key === k))
    .filter(Boolean) as ServerInfo[]
})

function statusTagType(status: string) {
  if (status === 'running') return 'success'
  if (status === 'stopped') return 'info'
  if (status === 'simulated') return 'warning'
  return 'warning'
}

function checkTagType(status: string) {
  if (status === 'ok') return 'success'
  if (status === 'simulated') return 'warning'
  if (status === 'warn') return 'warning'
  return 'info'
}

async function loadReadiness() {
  try {
    const [rd, st]: any[] = await Promise.all([
      api.get('/system/readiness'),
      api.get('/system/stacks'),
    ])
    readiness.value = rd.data
    stacks.value = st.data || []
  } catch {
    readiness.value = null
  }
}

async function load() {
  loading.value = true
  try {
    const res: any = await api.get('/nginx/status')
    overview.value = res.data
    await loadReadiness()
  } finally {
    loading.value = false
  }
}

async function withAction(key: string, fn: () => Promise<void>) {
  actionKey.value = key
  try {
    await fn()
    await load()
  } finally {
    actionKey.value = ''
  }
}

function openInstallLog(s: ServerInfo) {
  installLogKey.value = s.key
  installLogName.value = s.name
  installTrigger.value = false
  installLogVisible.value = true
}

async function oneClickInstall(s: ServerInfo) {
  try {
    await api.post(`/nginx/${s.key}/install`)
    openInstallLog(s)
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('software.installFailed')))
  }
}

async function onInstallDone(payload: { success: boolean }) {
  installTrigger.value = false
  await load()
  if (payload.success) {
    ElMessage.success(t('nginxPage.setupDone'))
  }
}

async function installStack(stack: StackDef) {
  await ElMessageBox.confirm(stack.description, stack.name, { type: 'info' })
  stackLoading.value = stack.key
  try {
    await api.post(`/system/stacks/${stack.key}/install`)
    ElMessage.success(t('nginxPage.stackStarted', { name: stack.name }))
    openInstallLog({ key: stack.components[0] || 'nginx', name: stack.name } as ServerInfo)
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
  } finally {
    stackLoading.value = ''
  }
}

async function installLNMP() {
  const stack = stacks.value.find((s) => s.key === 'lnmp')
  if (stack) {
    await installStack(stack)
    return
  }
  await ElMessageBox.confirm(t('nginxPage.lnmpStackHint'), t('nginxPage.lnmpStack'), { type: 'info' })
  lnmpLoading.value = true
  try {
    await api.post('/nginx/stack/lnmp')
    ElMessage.success(t('nginxPage.lnmpStarted'))
    openInstallLog({ key: 'nginx', name: 'Nginx' } as ServerInfo)
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
  } finally {
    lnmpLoading.value = false
  }
}

async function startServer(s: ServerInfo) {
  await withAction(s.key, async () => {
    await api.post(`/nginx/${s.key}/start`)
    ElMessage.success(t('nginxPage.started', { name: s.name }))
  })
}

async function stopServer(s: ServerInfo) {
  await ElMessageBox.confirm(t('nginxPage.stopConfirm', { name: s.name }), t('common.warning'), { type: 'warning' })
  await withAction(s.key, async () => {
    await api.post(`/nginx/${s.key}/stop`)
    ElMessage.success(t('nginxPage.stopped', { name: s.name }))
  })
}

async function reloadServer(s: ServerInfo) {
  await withAction(s.key, async () => {
    await api.post(`/nginx/${s.key}/reload`)
    ElMessage.success(t('nginxPage.reloaded'))
  })
}

async function testConfig(s: ServerInfo) {
  testOutput.value = ''
  await withAction(s.key, async () => {
    const res: any = await api.post(`/nginx/${s.key}/test`)
    testOutput.value = res.data?.output || t('nginxPage.testOk')
    ElMessage.success(t('nginxPage.testOk'))
  })
}

async function openConfig(s: ServerInfo) {
  configKey.value = s.key
  configLoading.value = true
  configDrawer.value = true
  try {
    const res: any = await api.get(`/nginx/${s.key}/config`)
    configContent.value = res.data?.content || ''
  } catch {
    configContent.value = ''
  } finally {
    configLoading.value = false
  }
}

async function saveConfig() {
  configLoading.value = true
  try {
    await api.put(`/nginx/${configKey.value}/config`, { content: configContent.value })
    ElMessage.success(t('common.saved'))
    configDrawer.value = false
    await load()
  } finally {
    configLoading.value = false
  }
}

function goSoftwareStore(key: string) {
  router.push({ path: '/software', query: { q: key } })
}

onMounted(load)
</script>

<template>
  <div v-loading="loading">
    <div class="page-header" :class="{ 'page-header--embedded': props.embedded }">
      <h2 v-if="!props.embedded">{{ t('page.nginx') }}</h2>
      <div class="header-actions">
        <el-button type="primary" :loading="lnmpLoading" @click="installLNMP">{{ t('nginxPage.lnmpStack') }}</el-button>
        <el-button :loading="loading" @click="load">{{ t('common.refresh') }}</el-button>
      </div>
    </div>

    <el-alert type="info" :closable="false" show-icon class="hint-alert">
      <template #title>{{ t('nginxPage.hintTitle') }}</template>
      <p class="hint-text">{{ t('nginxPage.hintBody') }}</p>
    </el-alert>

    <el-card v-if="readiness" shadow="hover" class="readiness-card">
      <template #header>
        <div class="card-head">
          <span>{{ t('nginxPage.readinessTitle') }}</span>
          <el-tag :type="readiness.score >= 70 ? 'success' : readiness.score >= 40 ? 'warning' : 'danger'">
            {{ t('nginxPage.readinessScore', { n: readiness.score }) }}
          </el-tag>
        </div>
      </template>
      <el-table :data="readiness.checks" size="small" stripe>
        <el-table-column prop="label" :label="t('common.name')" min-width="160" />
        <el-table-column prop="group" :label="t('common.type')" width="100" />
        <el-table-column :label="t('common.status')" width="110">
          <template #default="{ row }">
            <el-tag :type="checkTagType(row.status)" size="small">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="detail" :label="t('common.description')" min-width="200" show-overflow-tooltip />
      </el-table>
    </el-card>

    <el-card shadow="hover" class="stack-card">
      <template #header>{{ t('nginxPage.stacksTitle') }}</template>
      <el-row :gutter="12">
        <el-col v-for="stack in stacks" :key="stack.key" :xs="24" :sm="12" :lg="6">
          <div class="stack-item">
            <div class="stack-name">{{ stack.name }}</div>
            <div class="stack-desc">{{ stack.description }}</div>
            <el-button
              type="primary"
              size="small"
              :loading="stackLoading === stack.key"
              @click="installStack(stack)"
            >
              {{ t('nginxPage.oneClickInstall', { name: stack.name }) }}
            </el-button>
          </div>
        </el-col>
      </el-row>
    </el-card>

    <el-row :gutter="20" class="server-row">
      <el-col v-for="s in displayServers" :key="s.key" :xs="24" :lg="12">
        <el-card shadow="hover" class="server-card" :class="{ active: s.is_active }">
          <template #header>
            <div class="card-head">
              <div class="card-title">
                <span class="name">{{ s.name }}</span>
                <el-tag v-if="s.is_active" type="warning" size="small" effect="dark">{{ t('nginxPage.active') }}</el-tag>
              </div>
              <el-tag :type="statusTagType(s.status)" size="small">{{ s.status || 'unknown' }}</el-tag>
            </div>
          </template>

          <el-descriptions :column="1" size="small" border>
            <el-descriptions-item :label="t('nginxPage.installed')">
              <el-tag :type="s.installed ? 'success' : 'info'" size="small">
                {{ s.installed ? t('common.yes') : t('common.no') }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item :label="t('nginxPage.version')">
              <span class="mono">{{ s.version || '—' }}</span>
            </el-descriptions-item>
            <el-descriptions-item :label="t('nginxPage.sites')">{{ s.sites_enabled }}</el-descriptions-item>
            <el-descriptions-item :label="t('nginxPage.configPath')">
              <span class="mono path">{{ s.config_path || '—' }}</span>
            </el-descriptions-item>
            <el-descriptions-item :label="t('nginxPage.vhostDir')">
              <span class="mono path">{{ s.vhost_dir || '—' }}</span>
            </el-descriptions-item>
          </el-descriptions>

          <div class="actions">
            <template v-if="!s.installed">
              <el-button type="primary" :loading="actionKey === s.key" @click="oneClickInstall(s)">
                {{ t('nginxPage.oneClickInstall', { name: s.name }) }}
              </el-button>
              <el-button link @click="goSoftwareStore(s.key)">{{ t('nginxPage.goSoftwareStore') }}</el-button>
            </template>
            <template v-else>
              <el-button
                type="primary"
                :loading="actionKey === s.key"
                :disabled="s.status === 'running' && s.is_active"
                @click="startServer(s)"
              >
                {{ s.is_active && s.status === 'running' ? t('nginxPage.running') : t('nginxPage.setActiveStart') }}
              </el-button>
              <el-button :loading="actionKey === s.key" :disabled="s.status !== 'running'" @click="reloadServer(s)">
                {{ t('nginxPage.reload') }}
              </el-button>
              <el-button :loading="actionKey === s.key" @click="testConfig(s)">{{ t('nginxPage.test') }}</el-button>
              <el-button :loading="actionKey === s.key" @click="openConfig(s)">{{ t('nginxPage.editConfig') }}</el-button>
              <el-button
                type="warning"
                plain
                :loading="actionKey === s.key"
                :disabled="s.status !== 'running'"
                @click="stopServer(s)"
              >
                {{ t('common.stop') }}
              </el-button>
            </template>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-card v-if="testOutput" shadow="never" class="test-card">
      <template #header>{{ t('nginxPage.testResult') }}</template>
      <pre class="test-output">{{ testOutput }}</pre>
    </el-card>

    <el-card shadow="never" class="guide-card">
      <template #header>{{ t('nginxPage.guideTitle') }}</template>
      <ol class="guide-list">
        <li>{{ t('nginxPage.guide1') }}</li>
        <li>{{ t('nginxPage.guide2') }}</li>
        <li>{{ t('nginxPage.guide3') }}</li>
        <li>{{ t('nginxPage.guide4') }}</li>
      </ol>
    </el-card>

    <SoftwareInstallLogDialog
      v-model="installLogVisible"
      :app-key="installLogKey"
      :app-name="installLogName"
      :trigger-install="installTrigger"
      @done="onInstallDone"
    />

    <el-drawer v-model="configDrawer" :title="t('nginxPage.editMainConfig')" size="55%" destroy-on-close>
      <el-input
        v-model="configContent"
        v-loading="configLoading"
        type="textarea"
        :rows="24"
        class="config-editor"
        spellcheck="false"
      />
      <template #footer>
        <el-button @click="configDrawer = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="configLoading" @click="saveConfig">{{ t('common.save') }}</el-button>
      </template>
    </el-drawer>
  </div>
</template>

<style scoped>
.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
}
.header-actions {
  display: flex;
  gap: 8px;
}
.readiness-card,
.stack-card {
  margin-bottom: 20px;
}
.stack-item {
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  padding: 12px;
  margin-bottom: 12px;
  min-height: 120px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.stack-name {
  font-weight: 600;
}
.stack-desc {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  flex: 1;
}
.hint-text {
  margin: 4px 0 0;
  line-height: 1.6;
  font-size: 13px;
}
.server-row {
  margin-bottom: 20px;
}
.server-card.active {
  border-color: var(--cf-orange, #f6821f);
}
.card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.card-title {
  display: flex;
  align-items: center;
  gap: 8px;
}
.card-title .name {
  font-weight: 600;
  font-size: 16px;
}
.mono {
  font-family: Consolas, Monaco, monospace;
  font-size: 12px;
}
.path {
  word-break: break-all;
}
.actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 16px;
}
.test-card {
  margin-bottom: 20px;
}
.test-output {
  margin: 0;
  padding: 12px;
  background: var(--el-fill-color-light);
  border-radius: 6px;
  font-size: 12px;
  white-space: pre-wrap;
  word-break: break-all;
}
.guide-card {
  margin-bottom: 20px;
}
.guide-list {
  margin: 0;
  padding-left: 20px;
  line-height: 1.8;
  font-size: 13px;
  color: var(--el-text-color-regular);
}
.config-editor :deep(textarea) {
  font-family: Consolas, Monaco, monospace;
  font-size: 13px;
}
</style>
