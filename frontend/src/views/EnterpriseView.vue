<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import api from '@/api'
import { useAuthStore } from '@/stores/auth'
import { ElMessage, ElMessageBox } from 'element-plus'
import { cfTheme } from '@/config/theme'

const { t } = useI18n()
const router = useRouter()
const auth = useAuthStore()

const tab = ref('overview')
const loading = ref(false)
const overview = ref<any>(null)
const ha = ref<any>(null)
const monitoring = ref<any>(null)
const compliance = ref<any>(null)
const auditItems = ref<any[]>([])
const auditTotal = ref(0)
const auditLoading = ref(false)
const auditSettings = ref({ retention_days: 90, syslog_forward: false, syslog_enabled: false })
const auditFilters = ref({ category: '', action: '', limit: 50, offset: 0 })

function statusTag(s: string) {
  if (s === 'pass' || s === 'online' || s === 'up' || s === 'active') return 'success'
  if (s === 'warn' || s === 'warning') return 'warning'
  if (s === 'fail' || s === 'offline' || s === 'down' || s === 'danger') return 'danger'
  return 'info'
}

function resourceColor(p: number) {
  if (p >= 90) return cfTheme.danger
  if (p >= 70) return cfTheme.warning
  return cfTheme.success
}

function levelTag(l: string) {
  if (l === 'critical') return 'danger'
  if (l === 'warn') return 'warning'
  return 'info'
}

async function loadOverview() {
  loading.value = true
  try {
    const res: any = await api.get('/enterprise/overview')
    overview.value = res.data
    ha.value = res.data?.ha
    monitoring.value = res.data?.monitoring
    compliance.value = res.data?.compliance
  } finally {
    loading.value = false
  }
}

async function loadHA() {
  const res: any = await api.get('/enterprise/ha')
  ha.value = res.data
}

async function loadMonitoring() {
  const res: any = await api.get('/enterprise/monitoring')
  monitoring.value = res.data
}

async function loadCompliance() {
  const res: any = await api.get('/enterprise/compliance')
  compliance.value = res.data
}

async function loadAudit() {
  auditLoading.value = true
  try {
    const params: Record<string, string | number> = {
      limit: auditFilters.value.limit,
      offset: auditFilters.value.offset,
    }
    if (auditFilters.value.category) params.category = auditFilters.value.category
    if (auditFilters.value.action) params.action = auditFilters.value.action
    const res: any = await api.get('/enterprise/audit-logs', { params })
    auditItems.value = res.data?.items || []
    auditTotal.value = res.data?.total || 0
  } finally {
    auditLoading.value = false
  }
}

async function loadAuditSettings() {
  const res: any = await api.get('/enterprise/audit-settings')
  auditSettings.value = res.data || auditSettings.value
}

async function saveAuditSettings() {
  await api.put('/enterprise/audit-settings', auditSettings.value)
  ElMessage.success(t('enterprisePage.settingsSaved'))
}

function exportAudit(format: 'csv' | 'json') {
  const q = new URLSearchParams({ format, token: auth.token || '' })
  if (auditFilters.value.category) q.set('category', auditFilters.value.category)
  if (auditFilters.value.action) q.set('action', auditFilters.value.action)
  window.open(`${api.defaults.baseURL}/enterprise/audit-logs/export?${q}`, '_blank')
}

async function cleanupAudit() {
  await ElMessageBox.confirm(t('enterprisePage.cleanupConfirm'), t('common.confirm'), { type: 'warning' })
  const res: any = await api.delete('/enterprise/audit-logs', { params: { days: auditSettings.value.retention_days } })
  ElMessage.success(t('enterprisePage.cleanupDone', { n: res.data?.deleted ?? 0 }))
  await loadAudit()
}

const categories = ['security', 'config', 'user', 'cluster', 'website', 'database', 'ssl', 'system', 'migration']

const scoreCards = computed(() => {
  const ov = overview.value
  if (!ov) return []
  return [
    { label: t('enterprisePage.haHealth'), value: ov.ha?.grade || '-', sub: `${ov.ha?.node_online ?? 0}/${ov.ha?.node_total ?? 0} ${t('enterprisePage.nodes')}` },
    { label: t('enterprisePage.complianceGrade'), value: ov.compliance?.grade || '-', sub: `${ov.compliance?.score ?? 0}/100` },
    { label: t('enterprisePage.audit24h'), value: ov.audit_stats?.total_24h ?? 0, sub: `${t('enterprisePage.failed')}: ${ov.audit_stats?.failed_24h ?? 0}` },
    { label: t('enterprisePage.uptimeAlerts'), value: ov.uptime_alerts ?? 0, sub: t('enterprisePage.securityScore', { score: ov.security_score ?? 0, grade: ov.security_grade ?? '-' }) },
  ]
})

async function onTabChange(name: string | number) {
  if (name === 'ha') await loadHA()
  if (name === 'monitoring') await loadMonitoring()
  if (name === 'compliance') await loadCompliance()
  if (name === 'audit') {
    await loadAudit()
    await loadAuditSettings()
  }
}

onMounted(loadOverview)
</script>

<template>
  <div class="enterprise-page" v-loading="loading">
    <div class="page-header">
      <div>
        <h2>{{ t('enterprisePage.title') }}</h2>
        <p class="subtitle">{{ t('enterprisePage.subtitle') }}</p>
      </div>
      <el-button @click="loadOverview">{{ t('common.refresh') }}</el-button>
    </div>

    <el-tabs v-model="tab" @tab-change="onTabChange">
      <el-tab-pane :label="t('enterprisePage.tabOverview')" name="overview">
        <div class="score-grid">
          <el-card v-for="(c, i) in scoreCards" :key="i" shadow="hover" class="score-card">
            <div class="score-label">{{ c.label }}</div>
            <div class="score-value">{{ c.value }}</div>
            <div class="score-sub">{{ c.sub }}</div>
          </el-card>
        </div>
        <el-card v-if="ha?.recommendations?.length" class="mt-16">
          <template #header>{{ t('enterprisePage.recommendations') }}</template>
          <ul class="rec-list">
            <li v-for="(r, i) in ha.recommendations" :key="i">{{ r }}</li>
          </ul>
        </el-card>
      </el-tab-pane>

      <el-tab-pane :label="t('enterprisePage.tabHA')" name="ha">
        <div class="tab-toolbar">
          <el-tag :type="ha?.healthy ? 'success' : 'danger'" size="large">
            {{ t('enterprisePage.haGrade') }}: {{ ha?.grade || '-' }}
          </el-tag>
          <el-button link type="primary" @click="router.push('/cluster')">{{ t('enterprisePage.goCluster') }}</el-button>
        </div>
        <el-table :data="ha?.nodes || []" stripe>
          <el-table-column prop="name" :label="t('clusterPage.nodes')" />
          <el-table-column prop="host" :label="t('clusterPage.address')" />
          <el-table-column prop="role" :label="t('clusterPage.role')" width="100" />
          <el-table-column :label="t('common.status')" width="100">
            <template #default="{ row }">
              <el-tag :type="statusTag(row.status)" size="small">{{ row.status }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column :label="t('clusterPage.resources')" width="180">
            <template #default="{ row }">
              CPU {{ row.cpu_percent?.toFixed?.(1) ?? 0 }}% / MEM {{ row.mem_percent?.toFixed?.(1) ?? 0 }}%
            </template>
          </el-table-column>
        </el-table>
        <h4 class="section-title">{{ t('clusterPage.loadBalancers') }}</h4>
        <el-table :data="ha?.load_balancers || []" stripe>
          <el-table-column prop="name" :label="t('common.name')" />
          <el-table-column prop="domain" label="Domain" />
          <el-table-column prop="status" :label="t('common.status')" width="100">
            <template #default="{ row }">
              <el-tag :type="statusTag(row.status)" size="small">{{ row.status }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="backends" :label="t('clusterPage.backends')" width="90" />
        </el-table>
        <div v-if="ha?.replication_hints?.length" class="mt-16">
          <h4 class="section-title">{{ t('enterprisePage.replication') }}</h4>
          <el-table :data="ha.replication_hints" size="small">
            <el-table-column prop="role" label="Role" width="120" />
            <el-table-column prop="master_node" label="Master" />
            <el-table-column prop="slave_node" label="Slave" />
            <el-table-column prop="status" :label="t('common.status')" width="100" />
          </el-table>
        </div>
      </el-tab-pane>

      <el-tab-pane :label="t('enterprisePage.tabMonitoring')" name="monitoring">
        <el-row :gutter="16" class="mb-16">
          <el-col :span="6">
            <el-statistic :title="t('enterprisePage.uptimeTotal')" :value="monitoring?.uptime?.total ?? 0" />
          </el-col>
          <el-col :span="6">
            <el-statistic :title="t('enterprisePage.uptimeUp')" :value="monitoring?.uptime?.up ?? 0" />
          </el-col>
          <el-col :span="6">
            <el-statistic :title="t('enterprisePage.uptimeDown')" :value="monitoring?.uptime?.down ?? 0" />
          </el-col>
          <el-col :span="6">
            <el-statistic :title="t('enterprisePage.clusterNodes')" :value="`${monitoring?.cluster_online ?? 0}/${monitoring?.cluster_total ?? 0}`" />
          </el-col>
        </el-row>
        <el-table :data="monitoring?.nodes || []" stripe>
          <el-table-column prop="name" :label="t('common.name')" />
          <el-table-column prop="host" :label="t('clusterPage.address')" />
          <el-table-column prop="role" label="Role" width="90" />
          <el-table-column :label="t('common.status')" width="90">
            <template #default="{ row }">
              <el-tag :type="statusTag(row.status)" size="small">{{ row.status }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="CPU" width="100">
            <template #default="{ row }">
              <span :style="{ color: resourceColor(row.cpu_percent) }">{{ row.cpu_percent?.toFixed?.(1) }}%</span>
            </template>
          </el-table-column>
          <el-table-column label="MEM" width="100">
            <template #default="{ row }">
              <span :style="{ color: resourceColor(row.mem_percent) }">{{ row.mem_percent?.toFixed?.(1) }}%</span>
            </template>
          </el-table-column>
          <el-table-column label="Disk" width="100">
            <template #default="{ row }">
              <span :style="{ color: resourceColor(row.disk_percent) }">{{ row.disk_percent?.toFixed?.(1) }}%</span>
            </template>
          </el-table-column>
          <el-table-column prop="load1" label="Load" width="80" />
        </el-table>
      </el-tab-pane>

      <el-tab-pane :label="t('enterprisePage.tabCompliance')" name="compliance">
        <div class="tab-toolbar">
          <el-tag type="success" size="large">
            {{ t('enterprisePage.complianceGrade') }}: {{ compliance?.score ?? 0 }} ({{ compliance?.grade }})
          </el-tag>
          <span class="muted">{{ compliance?.summary }}</span>
        </div>
        <el-table :data="compliance?.checks || []" stripe>
          <el-table-column prop="name" :label="t('enterprisePage.checkItem')" />
          <el-table-column :label="t('common.status')" width="100">
            <template #default="{ row }">
              <el-tag :type="statusTag(row.status)" size="small">{{ row.status }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="detail" :label="t('enterprisePage.detail')" />
        </el-table>
      </el-tab-pane>

      <el-tab-pane :label="t('enterprisePage.tabAudit')" name="audit">
        <div class="audit-toolbar">
          <el-select v-model="auditFilters.category" clearable :placeholder="t('enterprisePage.category')" style="width: 140px" @change="loadAudit">
            <el-option v-for="c in categories" :key="c" :label="c" :value="c" />
          </el-select>
          <el-input v-model="auditFilters.action" clearable :placeholder="t('enterprisePage.action')" style="width: 160px" @keyup.enter="loadAudit" />
          <el-button @click="loadAudit">{{ t('enterprisePage.query') }}</el-button>
          <el-button @click="exportAudit('csv')">{{ t('enterprisePage.exportCsv') }}</el-button>
          <el-button @click="exportAudit('json')">{{ t('enterprisePage.exportJson') }}</el-button>
          <el-button type="danger" plain @click="cleanupAudit">{{ t('enterprisePage.cleanup') }}</el-button>
        </div>
        <el-table v-loading="auditLoading" :data="auditItems" stripe>
          <el-table-column prop="created_at" :label="t('enterprisePage.time')" width="170">
            <template #default="{ row }">{{ new Date(row.created_at).toLocaleString() }}</template>
          </el-table-column>
          <el-table-column prop="username" :label="t('common.username')" width="100" />
          <el-table-column prop="ip" label="IP" width="120" />
          <el-table-column prop="category" :label="t('enterprisePage.category')" width="100" />
          <el-table-column prop="action" :label="t('enterprisePage.action')" width="120" />
          <el-table-column prop="resource" :label="t('enterprisePage.resource')" width="140" show-overflow-tooltip />
          <el-table-column :label="t('enterprisePage.level')" width="80">
            <template #default="{ row }">
              <el-tag :type="levelTag(row.level)" size="small">{{ row.level }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="success" :label="t('enterprisePage.success')" width="70">
            <template #default="{ row }">
              <el-tag :type="row.success ? 'success' : 'danger'" size="small">{{ row.success ? 'OK' : 'FAIL' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="detail" :label="t('enterprisePage.detail')" show-overflow-tooltip />
        </el-table>
        <el-pagination
          v-if="auditTotal > auditFilters.limit"
          class="mt-16"
          layout="total, prev, pager, next"
          :total="auditTotal"
          :page-size="auditFilters.limit"
          @current-change="(p: number) => { auditFilters.offset = (p - 1) * auditFilters.limit; loadAudit() }"
        />
        <el-card class="mt-16 settings-card">
          <template #header>{{ t('enterprisePage.retentionSettings') }}</template>
          <el-form label-width="160px">
            <el-form-item :label="t('enterprisePage.retentionDays')">
              <el-input-number v-model="auditSettings.retention_days" :min="7" :max="3650" />
            </el-form-item>
            <el-form-item :label="t('enterprisePage.syslogForward')">
              <el-switch v-model="auditSettings.syslog_forward" />
              <span v-if="auditSettings.syslog_enabled" class="muted ml-8">{{ t('enterprisePage.syslogActive') }}</span>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="saveAuditSettings">{{ t('common.save') }}</el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<style scoped>
.enterprise-page { padding: 0 4px; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 16px; }
.page-header h2 { margin: 0 0 4px; font-size: 20px; }
.subtitle { margin: 0; color: var(--el-text-color-secondary); font-size: 13px; }
.score-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 16px; }
.score-card { text-align: center; }
.score-label { font-size: 13px; color: var(--el-text-color-secondary); }
.score-value { font-size: 28px; font-weight: 600; margin: 8px 0; }
.score-sub { font-size: 12px; color: var(--el-text-color-secondary); }
.mt-16 { margin-top: 16px; }
.mb-16 { margin-bottom: 16px; }
.ml-8 { margin-left: 8px; }
.tab-toolbar { display: flex; align-items: center; gap: 12px; margin-bottom: 16px; }
.section-title { margin: 20px 0 8px; font-size: 14px; }
.rec-list { margin: 0; padding-left: 20px; }
.audit-toolbar { display: flex; flex-wrap: wrap; gap: 8px; margin-bottom: 12px; }
.muted { color: var(--el-text-color-secondary); font-size: 13px; }
@media (max-width: 900px) { .score-grid { grid-template-columns: repeat(2, 1fr); } }
</style>
