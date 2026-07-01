<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import api, { resolveApiError } from '@/api'
import SoftwareInstallLogDialog from '@/components/SoftwareInstallLogDialog.vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  CircleCheck,
  CopyDocument,
  DataAnalysis,
  Link,
  RefreshRight,
  VideoPause,
  VideoPlay,
} from '@element-plus/icons-vue'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const analyticsProvider = ref<'openpanel' | 'posthog'>('openpanel')
const posthogFeatures = ref<any[]>([])

const appKey = computed(() => (analyticsProvider.value === 'posthog' ? 'posthog' : 'openpanel-analytics'))

const status = ref<any>({
  installed: false,
  running: false,
  dashboard_url: 'http://localhost:3300',
  api_url: 'http://localhost:3333/api',
  api_host: 'http://localhost:8020',
  features: [] as any[],
  min_ram_gb: 0,
  hint: '',
})
const websites = ref<any[]>([])
const selectedSiteId = ref<number | null>(null)
const loading = ref(false)
const saving = ref(false)
const serviceLoading = ref(false)

const installLogVisible = ref(false)
const installTrigger = ref(false)

const trackingForm = ref({
  product_analytics_enabled: false,
  product_analytics_client_id: '',
  product_analytics_api_url: 'http://localhost:3333/api',
  analytics_provider: 'openpanel',
})

const isPosthog = computed(() => analyticsProvider.value === 'posthog')
const apiFieldLabel = computed(() => (isPosthog.value ? t('productAnalytics.projectApiKey') : t('productAnalytics.clientId')))
const apiFieldHint = computed(() => (isPosthog.value ? t('productAnalytics.projectApiKeyHint') : t('productAnalytics.clientIdHint')))

const snippet = ref('')

const selectedSite = computed(() => websites.value.find((w) => w.id === selectedSiteId.value) || null)
const trackedSitesCount = computed(() => websites.value.filter((w) => w.product_analytics_enabled).length)
const deployReady = computed(() => status.value.installed && status.value.running)
const configReady = computed(
  () => !!trackingForm.value.product_analytics_client_id && trackingForm.value.product_analytics_enabled,
)
const trackReady = computed(() => configReady.value && !!snippet.value)

const kanbanColumns = computed(() => [
  {
    key: 'deploy',
    title: t('productAnalytics.kanbanDeploy'),
    done: deployReady.value,
    active: !deployReady.value,
  },
  {
    key: 'config',
    title: t('productAnalytics.kanbanConfig'),
    done: configReady.value,
    active: deployReady.value && !configReady.value,
  },
  {
    key: 'test',
    title: t('productAnalytics.kanbanTest'),
    done: trackReady.value,
    active: configReady.value && !trackReady.value,
  },
  {
    key: 'track',
    title: t('productAnalytics.kanbanTrack'),
    done: trackReady.value,
    active: configReady.value,
  },
])

async function loadStatus() {
  const path = isPosthog.value ? '/posthog/status' : '/product-analytics/status'
  const res: any = await api.get(path)
  status.value = res.data || status.value
  if (isPosthog.value && Array.isArray(status.value.features)) {
    posthogFeatures.value = status.value.features
  }
}

async function loadSnippet() {
  if (isPosthog.value) {
    const res: any = await api.get('/posthog/tracking-snippet', {
      params: {
        project_api_key: trackingForm.value.product_analytics_client_id || undefined,
        api_host: trackingForm.value.product_analytics_api_url || status.value.api_host || undefined,
      },
    })
    snippet.value = res.data?.snippet || ''
    return
  }
  const res: any = await api.get('/product-analytics/tracking-snippet', {
    params: {
      client_id: trackingForm.value.product_analytics_client_id || undefined,
      api_url: trackingForm.value.product_analytics_api_url || undefined,
    },
  })
  snippet.value = res.data?.snippet || ''
}

function applySiteToForm(site: any) {
  const provider = site.analytics_provider === 'posthog' ? 'posthog' : 'openpanel'
  analyticsProvider.value = provider
  trackingForm.value = {
    product_analytics_enabled: !!site.product_analytics_enabled,
    product_analytics_client_id: site.product_analytics_client_id || '',
    product_analytics_api_url:
      site.product_analytics_api_url ||
      (provider === 'posthog' ? status.value.api_host : status.value.api_url),
    analytics_provider: provider,
  }
}

function setProvider(p: 'openpanel' | 'posthog') {
  if (analyticsProvider.value === p && route.query.provider === p) return
  analyticsProvider.value = p
  trackingForm.value.analytics_provider = p
  if (p === 'posthog') {
    trackingForm.value.product_analytics_api_url = status.value.api_host || 'http://localhost:8020'
  } else {
    trackingForm.value.product_analytics_api_url = status.value.api_url || 'http://localhost:3333/api'
  }
  if (route.query.provider !== p) {
    router.replace({ query: { ...route.query, provider: p } })
  }
  loadStatus().then(() => loadSnippet())
}

async function loadWebsites() {
  const res: any = await api.get('/websites')
  websites.value = res.data || []
  if (!selectedSiteId.value && websites.value.length) {
    selectedSiteId.value = websites.value[0].id
  }
}

async function refreshAll() {
  loading.value = true
  try {
    await Promise.all([loadStatus(), loadWebsites()])
    if (selectedSite.value) applySiteToForm(selectedSite.value)
    await loadSnippet()
  } finally {
    loading.value = false
  }
}

function installAbTool() {
  installLogVisible.value = true
  installTrigger.value = true
}

async function onInstallDone() {
  installTrigger.value = false
  await refreshAll()
}

async function startService() {
  serviceLoading.value = true
  try {
    await api.post(`/software/${appKey.value}/start`)
    ElMessage.success(t('productAnalytics.started'))
    await loadStatus()
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('productAnalytics.startFailed')))
  } finally {
    serviceLoading.value = false
  }
}

const uninstalling = ref(false)

async function uninstallAbTool() {
  try {
    await ElMessageBox.confirm(
      t('productAnalytics.uninstallConfirm'),
      t('common.warning'),
      { type: 'warning', confirmButtonText: t('common.uninstall'), cancelButtonText: t('common.cancel') }
    )
  } catch {
    return
  }
  uninstalling.value = true
  try {
    await api.post(`/software/${appKey.value}/uninstall`)
    ElMessage.success(t('productAnalytics.uninstallDone'))
    await refreshAll()
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('productAnalytics.uninstallFailed')))
  } finally {
    uninstalling.value = false
  }
}

async function stopService() {
  serviceLoading.value = true
  try {
    await api.post(`/software/${appKey.value}/stop`)
    ElMessage.success(t('productAnalytics.stopped'))
    await loadStatus()
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
  } finally {
    serviceLoading.value = false
  }
}

async function saveTracking() {
  if (!selectedSiteId.value) return
  saving.value = true
  try {
    const res: any = await api.put(`/websites/${selectedSiteId.value}/product-analytics`, trackingForm.value)
    const idx = websites.value.findIndex((w) => w.id === selectedSiteId.value)
    if (idx >= 0) websites.value[idx] = res.data
    ElMessage.success(t('common.saved'))
    await loadSnippet()
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
  } finally {
    saving.value = false
  }
}

async function copySnippet() {
  if (!snippet.value) return
  try {
    await navigator.clipboard.writeText(snippet.value)
    ElMessage.success(t('productAnalytics.snippetCopied'))
  } catch {
    ElMessage.error(t('productAnalytics.copyFailed'))
  }
}

function openDashboard() {
  window.open(status.value.dashboard_url, '_blank', 'noopener')
}

watch(selectedSiteId, (id) => {
  const site = websites.value.find((w) => w.id === id)
  if (site) {
    applySiteToForm(site)
    loadSnippet()
  }
})

watch(
  () => route.query.provider,
  (q) => {
    const p = String(q || '')
    if (p === 'posthog' || p === 'openpanel') setProvider(p as 'openpanel' | 'posthog')
  },
  { immediate: true },
)

watch(
  () => [trackingForm.value.product_analytics_client_id, trackingForm.value.product_analytics_api_url, analyticsProvider.value],
  () => loadSnippet(),
)

onMounted(refreshAll)
</script>

<template>
  <div class="ab-board-page" v-loading="loading">
    <div class="page-header">
      <div>
        <h2>{{ t('productAnalytics.title') }}</h2>
        <p class="subtitle">{{ t('productAnalytics.subtitle') }}</p>
      </div>
      <el-button :icon="RefreshRight" :loading="loading" @click="refreshAll">{{ t('common.refresh') }}</el-button>
    </div>

    <el-segmented v-model="analyticsProvider" :options="[
      { label: t('productAnalytics.providerOpenPanel'), value: 'openpanel' },
      { label: t('productAnalytics.providerPosthog'), value: 'posthog' },
    ]" class="provider-switch mb" @change="setProvider(analyticsProvider)" />

    <el-alert v-if="isPosthog && status.hint" type="warning" :closable="false" show-icon class="mb">
      {{ status.hint }}
      <span v-if="status.min_ram_gb"> ({{ t('productAnalytics.minRam', { gb: status.min_ram_gb }) }})</span>
    </el-alert>

    <div class="metric-strip">
      <div class="metric-card">
        <span class="metric-label">{{ t('productAnalytics.installed') }}</span>
        <el-tag :type="status.installed ? 'success' : 'info'" size="large">
          {{ status.installed ? t('common.yes') : t('common.no') }}
        </el-tag>
      </div>
      <div class="metric-card">
        <span class="metric-label">{{ t('productAnalytics.running') }}</span>
        <el-tag :type="status.running ? 'success' : 'warning'" size="large">
          {{ status.running ? t('productAnalytics.runningYes') : t('productAnalytics.runningNo') }}
        </el-tag>
      </div>
      <div class="metric-card">
        <span class="metric-label">{{ t('productAnalytics.trackedSites') }}</span>
        <strong class="metric-value">{{ trackedSitesCount }}</strong>
      </div>
      <div class="metric-card metric-card--wide">
        <span class="metric-label">{{ t('productAnalytics.dashboardUrl') }}</span>
        <el-link :href="status.dashboard_url" target="_blank" type="primary">{{ status.dashboard_url }}</el-link>
      </div>
    </div>

    <div class="kanban-board">
      <div
        v-for="col in kanbanColumns"
        :key="col.key"
        class="kanban-col"
        :class="{ 'kanban-col--done': col.done, 'kanban-col--active': col.active }"
      >
        <div class="kanban-col-head">
          <span class="kanban-step">{{ col.title }}</span>
          <el-tag v-if="col.done" type="success" size="small" :icon="CircleCheck">{{ t('productAnalytics.stepDone') }}</el-tag>
          <el-tag v-else-if="col.active" type="warning" size="small">{{ t('productAnalytics.stepCurrent') }}</el-tag>
        </div>

        <!-- Deploy -->
        <template v-if="col.key === 'deploy'">
          <p class="kanban-desc">{{ isPosthog ? t('productAnalytics.kanbanDeployDescPosthog') : t('productAnalytics.kanbanDeployDesc') }}</p>
          <div class="kanban-meta">
            <div><span class="muted">{{ t('productAnalytics.apiUrl') }}</span> <code>{{ isPosthog ? status.api_host : status.api_url }}</code></div>
          </div>
          <el-table v-if="isPosthog && posthogFeatures.length" :data="posthogFeatures" size="small" stripe class="mb">
            <el-table-column prop="name" :label="t('productAnalytics.featureCol')" />
            <el-table-column :label="t('common.status')" width="90">
              <template #default="{ row }"><el-tag size="small" type="success">{{ row.enabled ? t('common.yes') : t('common.no') }}</el-tag></template>
            </el-table-column>
          </el-table>
          <div class="kanban-actions">
            <el-button
              v-if="!status.installed"
              type="primary"
              size="large"
              :icon="DataAnalysis"
              @click="installAbTool"
            >
              {{ t('productAnalytics.installOneClick') }}
            </el-button>
            <template v-else>
              <el-button
                v-if="!status.running"
                type="primary"
                :icon="VideoPlay"
                :loading="serviceLoading"
                @click="startService"
              >
                {{ t('productAnalytics.startService') }}
              </el-button>
              <el-button
                v-else
                :icon="VideoPause"
                :loading="serviceLoading"
                @click="stopService"
              >
                {{ t('productAnalytics.stopService') }}
              </el-button>
              <el-button type="primary" plain :icon="Link" :disabled="!status.running" @click="openDashboard">
                {{ t('productAnalytics.openDashboard') }}
              </el-button>
              <el-button
                type="danger"
                plain
                :loading="uninstalling"
                @click="uninstallAbTool"
              >
                {{ t('productAnalytics.uninstall') }}
              </el-button>
            </template>
          </div>
        </template>

        <!-- Config -->
        <template v-else-if="col.key === 'config'">
          <p class="kanban-desc">{{ t('productAnalytics.kanbanConfigDesc') }}</p>
          <el-form label-position="top" class="compact-form">
            <el-form-item :label="t('productAnalytics.selectWebsite')">
              <el-select v-model="selectedSiteId" filterable style="width: 100%" :disabled="!deployReady">
                <el-option v-for="site in websites" :key="site.id" :label="site.domain" :value="site.id" />
              </el-select>
            </el-form-item>
            <el-form-item :label="t('productAnalytics.enabled')">
              <el-switch v-model="trackingForm.product_analytics_enabled" :disabled="!deployReady" />
            </el-form-item>
            <el-form-item :label="apiFieldLabel">
              <el-input
                v-model="trackingForm.product_analytics_client_id"
                :placeholder="apiFieldHint"
                :disabled="!deployReady"
              />
            </el-form-item>
            <el-form-item :label="t('productAnalytics.apiUrlField')">
              <el-input v-model="trackingForm.product_analytics_api_url" :disabled="!deployReady" />
            </el-form-item>
            <el-button type="primary" :loading="saving" :disabled="!deployReady || !selectedSiteId" @click="saveTracking">
              {{ t('common.save') }}
            </el-button>
          </el-form>
        </template>

        <!-- Test -->
        <template v-else-if="col.key === 'test'">
          <p class="kanban-desc">{{ t('productAnalytics.kanbanTestDesc') }}</p>
          <ul v-if="isPosthog" class="feature-list">
            <li v-for="f in posthogFeatures" :key="f.key">{{ f.name }}</li>
          </ul>
          <ul v-else class="feature-list">
            <li>{{ t('productAnalytics.featureAB') }}</li>
            <li>{{ t('productAnalytics.featureFunnels') }}</li>
            <li>{{ t('productAnalytics.featureReplay') }}</li>
            <li>{{ t('productAnalytics.featureCohorts') }}</li>
          </ul>
          <el-button type="primary" plain :icon="Link" :disabled="!deployReady" @click="openDashboard">
            {{ t('productAnalytics.createExperiment') }}
          </el-button>
        </template>

        <!-- Track -->
        <template v-else>
          <p class="kanban-desc">{{ t('productAnalytics.kanbanTrackDesc') }}</p>
          <div class="snippet-toolbar">
            <el-button text type="primary" :icon="CopyDocument" :disabled="!snippet" @click="copySnippet">
              {{ t('productAnalytics.copySnippet') }}
            </el-button>
          </div>
          <pre class="snippet-box">{{ snippet || t('productAnalytics.snippetEmpty') }}</pre>
        </template>
      </div>
    </div>

    <SoftwareInstallLogDialog
      v-model="installLogVisible"
      :app-key="appKey"
      :app-name="t('productAnalytics.toolName')"
      :trigger-install="installTrigger"
      @done="onInstallDone"
    />
  </div>
</template>

<style scoped>
.ab-board-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.page-header h2 {
  margin: 0;
}

.subtitle {
  margin: 6px 0 0;
  color: var(--cf-text-muted);
  font-size: 14px;
}

.metric-strip {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.metric-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 14px 16px;
  border-radius: 12px;
  background: var(--el-fill-color-lighter);
  border: 1px solid var(--el-border-color-lighter);
}

.metric-card--wide {
  grid-column: span 1;
}

.metric-label {
  color: var(--cf-text-muted);
  font-size: 12px;
}

.metric-value {
  font-size: 28px;
  line-height: 1;
  color: var(--cf-orange, #f6821f);
}

.kanban-board {
  display: grid;
  grid-template-columns: repeat(4, minmax(240px, 1fr));
  gap: 14px;
  align-items: stretch;
  overflow-x: auto;
  padding-bottom: 4px;
}

.kanban-col {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-height: 360px;
  padding: 16px;
  border-radius: 14px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  box-shadow: var(--apple-shadow-sm, 0 1px 2px rgba(0, 0, 0, 0.04));
}

.kanban-col--active {
  border-color: rgba(246, 130, 31, 0.45);
  box-shadow: 0 0 0 1px rgba(246, 130, 31, 0.12);
}

.kanban-col--done {
  border-color: rgba(103, 194, 58, 0.35);
}

.kanban-col-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.kanban-step {
  font-weight: 600;
  font-size: 15px;
}

.kanban-desc {
  margin: 0;
  color: var(--cf-text-muted);
  font-size: 13px;
  line-height: 1.55;
}

.kanban-meta {
  font-size: 12px;
}

.kanban-meta code {
  word-break: break-all;
}

.kanban-actions {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: auto;
}

.compact-form :deep(.el-form-item) {
  margin-bottom: 12px;
}

.feature-list {
  margin: 0;
  padding-left: 18px;
  color: var(--el-text-color-regular);
  font-size: 13px;
  line-height: 1.7;
}

.snippet-toolbar {
  display: flex;
  justify-content: flex-end;
}

.snippet-box {
  flex: 1;
  margin: 0;
  padding: 12px;
  min-height: 160px;
  background: var(--cf-bg-muted, #f5f7fa);
  border-radius: 8px;
  overflow: auto;
  font-size: 11px;
  line-height: 1.45;
  white-space: pre-wrap;
  word-break: break-word;
}

.muted {
  color: var(--cf-text-muted);
}

@media (max-width: 1200px) {
  .metric-strip {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .kanban-board {
    grid-template-columns: repeat(2, minmax(260px, 1fr));
  }
}

@media (max-width: 768px) {
  .metric-strip,
  .kanban-board {
    grid-template-columns: 1fr;
  }
}
</style>
