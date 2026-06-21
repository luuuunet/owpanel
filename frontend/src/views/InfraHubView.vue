<script setup lang="ts">
import { onMounted, ref, watch, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import api from '@/api'
import { ElMessage, ElMessageBox } from 'element-plus'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const tab = ref('overview')
const loading = ref(false)
const overview = ref<any>(null)

const validTabs = ['overview', 'llmops', 'dataops', 'aiops', 'secops', 'orchestration']
const tabAlias: Record<string, string> = {
  vector: 'dataops', metrics: 'aiops', weights: 'llmops', security: 'secops', storage: 'dataops',
}

const linkLabels = computed<Record<string, string>>(() => ({
  k8s: t('menu.k8s'),
  cluster: t('menu.cluster'),
  docker: t('menu.docker'),
  compose: t('menu.compose'),
  ai: t('menu.aiHub'),
  devops: t('menu.devops'),
  logs: t('menu.logs'),
  protection: t('menu.protection'),
  software: t('menu.software'),
}))

function linkTitle(key: string, fallback: string) {
  return linkLabels.value[key] || fallback
}

watch(
  () => route.query.tab,
  (q) => {
    const v = String(q || '')
    tab.value = tabAlias[v] || (validTabs.includes(v) ? v : 'overview')
  },
  { immediate: true },
)

watch(tab, (v) => {
  if (route.query.tab !== v) {
    router.replace({ query: { ...route.query, tab: v } })
  }
})

async function load() {
  loading.value = true
  try {
    const res: any = await api.get('/infra-hub/overview')
    overview.value = res.data || null
  } catch (e: any) {
    ElMessage.error(e?.error || t('infraHub.loadFailed'))
  } finally {
    loading.value = false
  }
}

function go(path: string) {
  const [p, qs] = path.split('?')
  const query: Record<string, string> = {}
  if (qs) {
    for (const part of qs.split('&')) {
      const [k, v] = part.split('=')
      if (k) query[k] = v || ''
    }
  }
  router.push({ path: p, query: Object.keys(query).length ? query : undefined })
}

function goSoftware(key: string) {
  router.push({ path: '/software', query: { tab: 'store', q: key } })
}

async function snapshotWeight(id: string) {
  try {
    const res: any = await api.post('/infra-hub/weights/snapshot', { id })
    ElMessage.success(res.data?.path || t('common.success'))
    await load()
  } catch (e: any) {
    ElMessage.error(e?.error || t('common.failed'))
  }
}

async function deleteWeight(id: string) {
  await ElMessageBox.confirm(t('infraHub.confirmDeleteWeight'), t('common.warning'), { type: 'warning' })
  try {
    await api.delete('/infra-hub/weights', { params: { id } })
    ElMessage.success(t('common.deleted'))
    await load()
  } catch (e: any) {
    ElMessage.error(e?.error || t('common.failed'))
  }
}

function statusTag(running: boolean, installed?: boolean) {
  if (installed === false) return 'info'
  return running ? 'success' : 'warning'
}

onMounted(load)
</script>

<template>
  <div class="page-wrap" v-loading="loading">
    <div class="page-header">
      <div>
        <h2>{{ t('infraHub.title') }}</h2>
        <p class="muted">{{ t('infraHub.subtitle') }}</p>
      </div>
      <el-button type="primary" @click="load">{{ t('common.refresh') }}</el-button>
    </div>

    <el-tabs v-model="tab">
      <el-tab-pane :label="t('infraHub.tabOverview')" name="overview">
        <el-row :gutter="16" class="mb">
          <el-col :xs="24" :sm="8">
            <el-card shadow="never" class="stat-card">
              <el-statistic :title="t('infraHub.healthScore')" :value="overview?.health_score || 0" suffix="/ 100" />
            </el-card>
          </el-col>
          <el-col :xs="24" :sm="8">
            <el-card shadow="never" class="stat-card">
              <div class="stat-label">{{ t('infraHub.cloudNative') }}</div>
              <div class="stat-row">
                <el-tag :type="overview?.cloud_native?.k8s_ready ? 'success' : 'info'" size="small">K8s</el-tag>
                <el-tag :type="overview?.cloud_native?.cilium_ready ? 'success' : 'info'" size="small">Cilium</el-tag>
                <el-tag :type="overview?.cloud_native?.docker_running ? 'success' : 'info'" size="small">Docker</el-tag>
              </div>
              <p class="muted tiny">{{ overview?.cloud_native?.hint }}</p>
            </el-card>
          </el-col>
          <el-col :xs="24" :sm="8">
            <el-card shadow="never" class="stat-card">
              <div class="stat-label">{{ t('infraHub.aiInfra') }}</div>
              <div class="stat-row">
                <el-tag :type="overview?.ai_infra?.hf_running ? 'success' : 'info'" size="small">HF AI</el-tag>
                <el-tag :type="overview?.ai_infra?.vector_db_running ? 'success' : 'warning'" size="small">
                  {{ t('infraHub.vectorShort') }} {{ overview?.ai_infra?.vector_db_running || 0 }}/{{ overview?.ai_infra?.vector_db_total || 0 }}
                </el-tag>
                <el-tag v-if="overview?.ai_infra?.rag_ready" type="success" size="small">RAG</el-tag>
              </div>
              <p class="muted tiny">{{ overview?.ai_infra?.hint }}</p>
            </el-card>
          </el-col>
        </el-row>

        <h4 class="section-title">{{ t('infraHub.quickNav') }}</h4>
        <el-row :gutter="12" class="mb">
          <el-col v-for="link in overview?.quick_links || []" :key="link.key" :xs="12" :sm="8" :md="6" class="link-col">
            <el-card shadow="hover" class="link-card" @click="go(link.path)">
              <strong>{{ linkTitle(link.key, link.title) }}</strong>
              <span class="muted tiny">{{ link.path }}</span>
            </el-card>
          </el-col>
        </el-row>

        <el-row :gutter="12" class="mb">
          <el-col :xs="12" :sm="8" :md="4" v-for="pill in [
            { k: 'llmops', l: 'LLMOps' },
            { k: 'dataops', l: 'DataOps' },
            { k: 'aiops', l: 'AIOps' },
            { k: 'secops', l: 'SecOps' },
            { k: 'orchestration', l: 'Orchestration' },
          ]" :key="pill.k">
            <el-button class="pill-btn" @click="tab = pill.k">{{ pill.l }}</el-button>
          </el-col>
        </el-row>
      </el-tab-pane>

      <el-tab-pane :label="t('infraHub.tabLLMOps')" name="llmops">
        <el-alert type="info" :closable="false" show-icon class="mb">{{ overview?.llmops?.hint || t('infraHub.llmopsHint') }}</el-alert>
        <el-descriptions :column="3" border class="mb">
          <el-descriptions-item :label="t('infraHub.gpu')">{{ overview?.llmops?.gpu_available ? t('common.yes') : t('common.no') }}</el-descriptions-item>
          <el-descriptions-item :label="t('infraHub.snapshots')">{{ overview?.llmops?.snapshot_count || 0 }}</el-descriptions-item>
          <el-descriptions-item :label="t('infraHub.hfModel')">{{ overview?.llmops?.hf_model_id || '—' }}</el-descriptions-item>
        </el-descriptions>
        <h4>{{ t('infraHub.inferenceRuntimes') }}</h4>
        <el-table :data="overview?.llmops?.runtimes || []" stripe class="mb">
          <el-table-column prop="name" :label="t('common.name')" min-width="160" />
          <el-table-column prop="framework" width="100" />
          <el-table-column prop="scheduler" :label="t('infraHub.scheduler')" width="100" />
          <el-table-column :label="t('common.status')" width="90">
            <template #default="{ row }"><el-tag :type="row.running ? 'success' : 'info'" size="small">{{ row.running ? t('common.running') : t('common.stopped') }}</el-tag></template>
          </el-table-column>
          <el-table-column prop="endpoint" min-width="160" />
          <el-table-column :label="t('common.actions')" width="120">
            <template #default="{ row }">
              <el-button v-if="!row.running" text type="primary" size="small" @click="goSoftware(row.key === 'tgi' ? 'huggingface-ai' : row.key)">{{ t('common.install') }}</el-button>
              <el-button v-else text type="primary" size="small" @click="go('/ai')">{{ t('infraHub.openAI') }}</el-button>
            </template>
          </el-table-column>
        </el-table>
        <h4>{{ t('infraHub.modelLifecycle') }}</h4>
        <el-table :data="overview?.llmops?.models || []" stripe max-height="360">
          <el-table-column prop="name" :label="t('common.name')" min-width="140" />
          <el-table-column prop="status" width="100" />
          <el-table-column prop="size_human" :label="t('common.size')" width="100" />
          <el-table-column prop="version" :label="t('common.version')" width="120" />
          <el-table-column :label="t('common.actions')" width="160">
            <template #default="{ row }">
              <el-button text type="primary" size="small" @click="snapshotWeight(row.id)">{{ t('infraHub.snapshot') }}</el-button>
              <el-button text type="danger" size="small" @click="deleteWeight(row.id)">{{ t('common.delete') }}</el-button>
            </template>
          </el-table-column>
        </el-table>
        <p class="muted tiny mt"><code>{{ overview?.llmops?.sync_command }}</code></p>
      </el-tab-pane>

      <el-tab-pane :label="t('infraHub.tabDataOps')" name="dataops">
        <el-alert type="info" :closable="false" show-icon class="mb">{{ overview?.dataops?.hint || t('infraHub.dataopsHint') }}</el-alert>
        <h4>{{ t('infraHub.tabVector') }}</h4>
        <el-table :data="overview?.dataops?.vector_dbs || []" stripe class="mb">
          <el-table-column prop="name" width="120" />
          <el-table-column prop="use_case" min-width="200" />
          <el-table-column :label="t('common.status')" width="100">
            <template #default="{ row }"><el-tag :type="statusTag(row.running, row.installed)" size="small">{{ row.running ? t('common.running') : t('common.stopped') }}</el-tag></template>
          </el-table-column>
          <el-table-column :label="t('infraHub.collections')" min-width="140">
            <template #default="{ row }">{{ row.collections?.join(', ') || '—' }}</template>
          </el-table-column>
          <el-table-column :label="t('common.actions')" width="90">
            <template #default="{ row }"><el-button text type="primary" size="small" @click="goSoftware(row.key)">{{ t('common.install') }}</el-button></template>
          </el-table-column>
        </el-table>
        <h4>{{ t('infraHub.knowledgeApps') }}</h4>
        <el-table :data="overview?.dataops?.knowledge_apps || []" stripe class="mb">
          <el-table-column prop="name" width="140" />
          <el-table-column prop="use_case" min-width="220" />
          <el-table-column :label="t('common.status')" width="100">
            <template #default="{ row }"><el-tag :type="statusTag(row.running, row.installed)" size="small">{{ row.running ? t('common.running') : (row.installed ? t('common.stopped') : t('common.notInstalled')) }}</el-tag></template>
          </el-table-column>
          <el-table-column :label="t('common.actions')" width="90">
            <template #default="{ row }"><el-button text type="primary" size="small" @click="goSoftware(row.key)">{{ t('common.install') }}</el-button></template>
          </el-table-column>
        </el-table>
        <h4>{{ t('infraHub.tabStorage') }}</h4>
        <div v-for="eng in overview?.dataops?.storage_meta || []" :key="eng.key" class="storage-block">
          <strong>{{ eng.name }}</strong>
          <el-tag :type="statusTag(eng.running, eng.installed)" size="small" class="ml">{{ eng.running ? t('common.running') : t('common.stopped') }}</el-tag>
          <el-table v-if="eng.buckets?.length" :data="eng.buckets" size="small" stripe>
            <el-table-column prop="name" :label="t('infraHub.bucket')" />
            <el-table-column prop="object_count" :label="t('infraHub.objectCount')" width="100" />
            <el-table-column prop="total_human" :label="t('common.size')" width="100" />
          </el-table>
        </div>
      </el-tab-pane>

      <el-tab-pane :label="t('infraHub.tabAIOps')" name="aiops">
        <el-alert type="info" :closable="false" show-icon class="mb">{{ overview?.aiops?.hint || t('infraHub.aiopsHint') }}</el-alert>
        <el-row :gutter="16" class="mb">
          <el-col :span="6"><el-statistic :title="t('infraHub.healthScore')" :value="overview?.aiops?.health_score || 0" suffix="/ 100" /></el-col>
          <el-col :span="6"><el-statistic :title="t('infraHub.predictedRisk')" :value="overview?.aiops?.predicted_risk || 'low'" /></el-col>
          <el-col :span="6"><el-statistic :title="t('infraHub.logSources')" :value="overview?.aiops?.log_sources || 0" /></el-col>
          <el-col :span="6"><el-statistic :title="t('infraHub.errorLines')" :value="overview?.aiops?.error_lines || 0" /></el-col>
        </el-row>
        <p class="muted mb">{{ overview?.aiops?.alert_hint }}</p>
        <h4>{{ t('infraHub.tabMetrics') }}</h4>
        <el-table :data="overview?.aiops?.metrics_stores || []" stripe class="mb">
          <el-table-column prop="name" width="140" />
          <el-table-column :label="t('common.status')" width="100">
            <template #default="{ row }"><el-tag :type="statusTag(row.running, row.installed)" size="small">{{ row.running ? t('common.running') : t('common.stopped') }}</el-tag></template>
          </el-table-column>
          <el-table-column prop="endpoint" min-width="180" />
          <el-table-column :label="t('common.actions')" width="90">
            <template #default="{ row }"><el-button text type="primary" size="small" @click="goSoftware(row.key)">{{ t('common.install') }}</el-button></template>
          </el-table-column>
        </el-table>
        <h4>{{ t('infraHub.logInsights') }}</h4>
        <el-table :data="overview?.aiops?.log_insights || []" stripe max-height="320">
          <el-table-column prop="source" width="120" />
          <el-table-column prop="message" min-width="240" show-overflow-tooltip />
          <el-table-column prop="fix_hint" :label="t('infraHub.fixHint')" min-width="220" show-overflow-tooltip />
        </el-table>
        <el-button class="mt" type="primary" @click="go('/logs')">{{ t('infraHub.openLogs') }}</el-button>
      </el-tab-pane>

      <el-tab-pane :label="t('infraHub.tabSecOps')" name="secops">
        <el-row :gutter="16" class="mb">
          <el-col :span="6"><el-statistic :title="t('infraHub.securityScore')" :value="overview?.secops?.audit_score || 0" suffix="/ 100" /></el-col>
          <el-col :span="6"><el-statistic :title="t('infraHub.threatLevel')" :value="overview?.secops?.threat_level || 'low'" /></el-col>
          <el-col :span="6"><el-statistic :title="t('infraHub.policyCount')" :value="overview?.secops?.policy_count || 0" /></el-col>
          <el-col :span="6"><el-button type="primary" @click="go('/protection?tab=cilium')">{{ t('infraHub.openCilium') }}</el-button></el-col>
        </el-row>
        <el-alert type="warning" :closable="false" show-icon class="mb" :title="t('infraHub.aiInsight')">{{ overview?.secops?.ai_analysis }}</el-alert>
        <h4>{{ t('infraHub.autoDefense') }}</h4>
        <el-table :data="overview?.secops?.auto_defense_rules || []" stripe class="mb">
          <el-table-column prop="title" min-width="160" />
          <el-table-column prop="description" min-width="280" />
          <el-table-column prop="severity" width="90" />
          <el-table-column :label="t('common.actions')" width="100">
            <template #default="{ row }"><el-button text type="primary" size="small" @click="go(row.action)">{{ t('infraHub.apply') }}</el-button></template>
          </el-table-column>
        </el-table>
        <h4>{{ t('infraHub.recentEvents') }}</h4>
        <el-table :data="overview?.secops?.recent_events || []" stripe max-height="240">
          <el-table-column prop="source" width="120" />
          <el-table-column prop="message" min-width="320" show-overflow-tooltip />
        </el-table>
      </el-tab-pane>

      <el-tab-pane :label="t('infraHub.tabOrchestration')" name="orchestration">
        <el-alert type="info" :closable="false" show-icon class="mb">{{ overview?.orchestration?.hint || t('infraHub.orchestrationHint') }}</el-alert>
        <el-row :gutter="16" class="mb">
          <el-col :span="6"><el-statistic :title="t('infraHub.clusterNodes')" :value="overview?.orchestration?.cluster_online || 0" :suffix="'/' + (overview?.orchestration?.cluster_nodes || 0)" /></el-col>
          <el-col :span="6"><el-statistic :title="t('infraHub.k8sNodes')" :value="overview?.orchestration?.k8s_nodes || 0" /></el-col>
          <el-col :span="6"><el-statistic :title="t('infraHub.composeRunning')" :value="overview?.orchestration?.compose_running || 0" :suffix="'/' + (overview?.orchestration?.compose_total || 0)" /></el-col>
          <el-col :span="6"><el-button type="primary" @click="go('/devops')">{{ t('infraHub.openDevOps') }}</el-button></el-col>
        </el-row>
        <h4>{{ t('infraHub.pipelines') }}</h4>
        <el-table :data="overview?.orchestration?.pipelines || []" stripe>
          <el-table-column prop="name" min-width="160" />
          <el-table-column prop="type" width="100" />
          <el-table-column prop="status" width="100" />
          <el-table-column :label="t('common.actions')" width="100">
            <template #default="{ row }"><el-button text type="primary" size="small" @click="go(row.path)">{{ t('infraHub.open') }}</el-button></template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<style scoped>
.page-wrap { padding: 4px 0; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 16px; }
.page-header h2 { margin: 0 0 4px; font-size: 20px; }
.muted { color: var(--el-text-color-secondary); font-size: 13px; }
.tiny { font-size: 12px; margin-top: 6px; }
.mb { margin-bottom: 16px; }
.ml { margin-left: 8px; }
.sync-code { font-size: 12px; word-break: break-all; }
.storage-block { margin-bottom: 24px; padding-bottom: 16px; border-bottom: 1px solid var(--el-border-color-lighter); }
.storage-head { display: flex; align-items: center; gap: 8px; margin-bottom: 4px; }
.stat-card { min-height: 120px; }
.stat-label { font-size: 13px; color: var(--el-text-color-secondary); margin-bottom: 8px; }
.stat-row { display: flex; flex-wrap: wrap; gap: 6px; }
.section-title { margin: 8px 0 12px; font-size: 14px; }
.link-col { margin-bottom: 12px; }
.link-card { cursor: pointer; }
.link-card strong { display: block; margin-bottom: 4px; }
.pill-btn { width: 100%; margin-bottom: 8px; }
.mt { margin-top: 12px; }
</style>
