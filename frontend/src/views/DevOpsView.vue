<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import api, { resolveApiError } from '@/api'
import { ElMessage } from 'element-plus'

const { t } = useI18n()
const tab = ref('cicd')
const loading = ref(false)

const websites = ref<any[]>([])
const selectedSiteId = ref<number | null>(null)
const deployForm = ref<any>({})
const deployJobs = ref<any[]>([])
const dockerfile = ref('')

const slowLogs = ref<any>({ entries: [], by_source: {} })
const trafficAnomalies = ref<any[]>([])
const auditReport = ref<any>({ items: [] })
const cveResult = ref<any>({ items: [] })
const composeApps = ref<any[]>([])

async function loadWebsites() {
  const res: any = await api.get('/websites/projects')
  websites.value = res.data || []
  if (!selectedSiteId.value && websites.value.length) {
    selectedSiteId.value = websites.value[0].id
    await loadDeployConfig()
  }
}

async function loadDeployConfig() {
  if (!selectedSiteId.value) return
  const res: any = await api.get(`/devops/deploy/config/${selectedSiteId.value}`)
  deployForm.value = { ...(res.data || {}), webhook_secret: '' }
}

async function saveDeployConfig() {
  if (!selectedSiteId.value) return
  loading.value = true
  try {
    const payload = { ...deployForm.value }
    delete payload.domain
    delete payload.root_path
    delete payload.hook_url
    delete payload.ci_url
    const res: any = await api.put(`/devops/deploy/config/${selectedSiteId.value}`, payload)
    deployForm.value = res.data
    ElMessage.success(t('devops.saved'))
  } finally {
    loading.value = false
  }
}

async function triggerDeploy() {
  if (!selectedSiteId.value) return
  loading.value = true
  try {
    await api.post(`/devops/deploy/trigger/${selectedSiteId.value}`)
    ElMessage.success(t('devops.deployStarted'))
    await loadDeployJobs()
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
  } finally {
    loading.value = false
  }
}

async function loadDeployJobs() {
  const q = selectedSiteId.value ? `?website_id=${selectedSiteId.value}` : ''
  const res: any = await api.get(`/devops/deploy/jobs${q}`)
  deployJobs.value = res.data || []
}

async function exportDockerfile() {
  if (!selectedSiteId.value) return
  const res: any = await api.get(`/devops/deploy/dockerfile/${selectedSiteId.value}`)
  dockerfile.value = res.data?.content || ''
}

async function saveDockerfile() {
  if (!selectedSiteId.value) return
  const res: any = await api.post(`/devops/deploy/dockerfile/${selectedSiteId.value}/save`, { content: dockerfile.value })
  ElMessage.success(t('devops.dockerfileSaved', { path: res.data?.path }))
}

async function loadSlowLogs() {
  const res: any = await api.get('/devops/diagnostics/slow-logs')
  slowLogs.value = res.data || { entries: [] }
}

async function loadTrafficAnomalies() {
  const res: any = await api.get('/devops/diagnostics/traffic-anomalies')
  trafficAnomalies.value = res.data || []
}

async function loadAudit() {
  loading.value = true
  try {
    const res: any = await api.get('/devops/audit/config')
    auditReport.value = res.data || { items: [] }
  } finally {
    loading.value = false
  }
}

async function loadCVE() {
  loading.value = true
  try {
    const res: any = await api.get('/devops/security/cve')
    cveResult.value = res.data || { items: [] }
  } finally {
    loading.value = false
  }
}

async function loadCompose() {
  const res: any = await api.get('/compose')
  composeApps.value = res.data || []
}

async function rollingCompose(row: any) {
  await api.post(`/compose/${row.id}/rolling`)
  ElMessage.success(t('devops.rollingDone'))
  await loadCompose()
}

async function blueGreenCompose(row: any) {
  await api.post(`/compose/${row.id}/blue-green`)
  ElMessage.success(t('devops.blueGreenDone'))
  await loadCompose()
}

function copyText(text: string) {
  navigator.clipboard.writeText(text)
  ElMessage.success(t('common.success'))
}

async function onTabChange(name: string) {
  if (name === 'cicd') {
    await loadDeployJobs()
  } else if (name === 'diagnostics') {
    await loadSlowLogs()
    await loadTrafficAnomalies()
  } else if (name === 'audit') {
    await loadAudit()
  } else if (name === 'docker') {
    await loadCompose()
  } else if (name === 'security') {
    await loadCVE()
  }
}

onMounted(async () => {
  await loadWebsites()
  await loadDeployJobs()
})
</script>

<template>
  <div>
    <div class="page-header">
      <h2>{{ t('devops.title') }}</h2>
    </div>
    <el-alert :title="t('devops.hint')" type="info" show-icon :closable="false" style="margin-bottom: 16px" />

    <el-tabs v-model="tab" @tab-change="onTabChange">
      <el-tab-pane :label="t('devops.tabCicd')" name="cicd">
        <el-row :gutter="16">
          <el-col :span="14">
            <el-card shadow="hover">
              <template #header>{{ t('devops.deployConfig') }}</template>
              <el-form label-width="120px">
                <el-form-item :label="t('devops.site')">
                  <el-select v-model="selectedSiteId" style="width: 100%" @change="loadDeployConfig">
                    <el-option v-for="w in websites" :key="w.id" :label="w.domain" :value="w.id" />
                  </el-select>
                </el-form-item>
                <el-form-item :label="t('devops.enabled')">
                  <el-switch v-model="deployForm.enabled" />
                </el-form-item>
                <el-form-item :label="t('devops.repoUrl')">
                  <el-input v-model="deployForm.repo_url" placeholder="https://github.com/user/repo.git" />
                </el-form-item>
                <el-form-item :label="t('devops.branch')">
                  <el-input v-model="deployForm.branch" />
                </el-form-item>
                <el-form-item :label="t('devops.ciProvider')">
                  <el-select v-model="deployForm.ci_provider" style="width: 100%">
                    <el-option label="GitHub / GitLab WebHook" value="webhook" />
                    <el-option label="GitHub Actions" value="github_actions" />
                    <el-option label="GitLab CI" value="gitlab_ci" />
                    <el-option label="Manual" value="manual" />
                  </el-select>
                </el-form-item>
                <el-form-item :label="t('devops.deployScript')">
                  <el-input v-model="deployForm.deploy_script" type="textarea" :rows="4" :placeholder="t('devops.deployScriptHint')" />
                </el-form-item>
                <el-form-item :label="t('devops.webhookUrl')">
                  <el-input v-model="deployForm.hook_url" readonly>
                    <template #append>
                      <el-button @click="copyText(deployForm.hook_url)">{{ t('common.copy') }}</el-button>
                    </template>
                  </el-input>
                </el-form-item>
                <el-form-item :label="t('devops.ciUrl')">
                  <el-input v-model="deployForm.ci_url" readonly>
                    <template #append>
                      <el-button @click="copyText(deployForm.ci_url)">{{ t('common.copy') }}</el-button>
                    </template>
                  </el-input>
                </el-form-item>
                <el-form-item>
                  <el-button type="primary" :loading="loading" @click="saveDeployConfig">{{ t('common.save') }}</el-button>
                  <el-button type="success" :loading="loading" @click="triggerDeploy">{{ t('devops.deployNow') }}</el-button>
                  <el-button @click="exportDockerfile">{{ t('devops.exportDockerfile') }}</el-button>
                </el-form-item>
              </el-form>
            </el-card>
            <el-card v-if="dockerfile" shadow="hover" style="margin-top: 16px">
              <template #header>{{ t('devops.dockerfilePreview') }}</template>
              <el-input v-model="dockerfile" type="textarea" :rows="12" />
              <el-button style="margin-top: 8px" @click="saveDockerfile">{{ t('devops.saveDockerfile') }}</el-button>
            </el-card>
          </el-col>
          <el-col :span="10">
            <el-card shadow="hover">
              <template #header>{{ t('devops.deployHistory') }}</template>
              <el-table :data="deployJobs" size="small" max-height="480">
                <el-table-column prop="trigger" :label="t('devops.trigger')" width="90" />
                <el-table-column prop="status" :label="t('common.status')" width="80">
                  <template #default="{ row }">
                    <el-tag :type="row.status === 'success' ? 'success' : row.status === 'failed' ? 'danger' : 'info'" size="small">{{ row.status }}</el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="started_at" :label="t('devops.startedAt')" min-width="140" />
              </el-table>
            </el-card>
          </el-col>
        </el-row>
      </el-tab-pane>

      <el-tab-pane :label="t('devops.tabDiagnostics')" name="diagnostics">
        <el-row :gutter="16">
          <el-col :span="14">
            <el-card shadow="hover">
              <template #header>{{ t('devops.slowLogs') }}</template>
              <el-table :data="slowLogs.entries" size="small" max-height="420">
                <el-table-column prop="source" width="90" />
                <el-table-column prop="domain" width="120" />
                <el-table-column prop="duration_ms" :label="t('devops.durationMs')" width="90" />
                <el-table-column prop="message" show-overflow-tooltip />
              </el-table>
            </el-card>
          </el-col>
          <el-col :span="10">
            <el-card shadow="hover">
              <template #header>{{ t('devops.trafficAnomaly') }}</template>
              <el-table :data="trafficAnomalies" size="small" max-height="420">
                <el-table-column prop="domain" min-width="100" />
                <el-table-column prop="change_pct" :label="t('devops.changePct')" width="80">
                  <template #default="{ row }">{{ row.change_pct }}%</template>
                </el-table-column>
                <el-table-column prop="severity" width="80">
                  <template #default="{ row }">
                    <el-tag :type="row.severity === 'high' ? 'danger' : 'warning'" size="small">{{ row.severity }}</el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="hint" show-overflow-tooltip />
              </el-table>
            </el-card>
          </el-col>
        </el-row>
      </el-tab-pane>

      <el-tab-pane :label="t('devops.tabAudit')" name="audit">
        <el-card shadow="hover">
          <template #header>
            <span>{{ t('devops.configAudit') }}</span>
            <el-button style="float: right" size="small" :loading="loading" @click="loadAudit">{{ t('waf.scan') }}</el-button>
          </template>
          <el-alert
            v-if="auditReport.items?.length"
            :title="t('devops.auditSummary', { pass: auditReport.pass_count, warn: auditReport.warn_count, fail: auditReport.fail_count })"
            type="warning"
            show-icon
            :closable="false"
            style="margin-bottom: 12px"
          />
          <el-table :data="auditReport.items" stripe>
            <el-table-column prop="category" width="100" />
            <el-table-column prop="target" min-width="140" />
            <el-table-column prop="status" width="80">
              <template #default="{ row }">
                <el-tag :type="row.status === 'pass' ? 'success' : row.status === 'fail' ? 'danger' : 'warning'" size="small">{{ row.status }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="detail" show-overflow-tooltip />
            <el-table-column prop="solution" show-overflow-tooltip />
          </el-table>
        </el-card>
      </el-tab-pane>

      <el-tab-pane :label="t('devops.tabDocker')" name="docker">
        <el-card shadow="hover">
          <template #header>{{ t('devops.composeLifecycle') }}</template>
          <el-table :data="composeApps" stripe>
            <el-table-column prop="name" :label="t('common.name')" />
            <el-table-column prop="path" show-overflow-tooltip />
            <el-table-column prop="live_status" :label="t('common.status')" width="100" />
            <el-table-column :label="t('common.actions')" width="240">
              <template #default="{ row }">
                <el-button size="small" @click="rollingCompose(row)">{{ t('devops.rollingUpdate') }}</el-button>
                <el-button size="small" type="primary" @click="blueGreenCompose(row)">{{ t('devops.blueGreen') }}</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-tab-pane>

      <el-tab-pane :label="t('devops.tabSecurity')" name="security">
        <el-card shadow="hover">
          <template #header>
            <span>{{ t('devops.cveScan') }}</span>
            <el-button style="float: right" size="small" :loading="loading" @click="loadCVE">{{ t('waf.scan') }}</el-button>
          </template>
          <el-alert
            v-if="cveResult.high_count"
            :title="t('devops.cveHighAlert', { n: cveResult.high_count })"
            type="error"
            show-icon
            :closable="false"
            style="margin-bottom: 12px"
          />
          <el-table :data="cveResult.items" stripe>
            <el-table-column prop="software" width="120" />
            <el-table-column prop="version" width="80" />
            <el-table-column prop="cve" width="140" />
            <el-table-column prop="severity" width="80">
              <template #default="{ row }">
                <el-tag :type="row.severity === 'high' ? 'danger' : 'warning'" size="small">{{ row.severity }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="description" show-overflow-tooltip />
            <el-table-column prop="fix" show-overflow-tooltip />
          </el-table>
          <p v-if="cveResult.package_updates" class="muted">{{ t('devops.packageUpdates', { n: cveResult.package_updates }) }}</p>
        </el-card>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<style scoped>
.muted {
  margin-top: 12px;
  color: #909399;
  font-size: 13px;
}
</style>
