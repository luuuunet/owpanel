<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '@/api'
import SoftwareInstallLogDialog from '@/components/SoftwareInstallLogDialog.vue'

const props = withDefaults(defineProps<{ embedded?: boolean }>(), { embedded: false })

interface RiskItem {
  key: string
  name: string
  level: string
  status: string
  solution: string
  fix_type?: string
}

interface FixResult {
  success: boolean
  needs_guide?: boolean
  guide_message?: string
  redirect_path?: string
  install_job_key?: string
  message?: string
}

interface ScoreFactor {
  key: string
  name: string
  score: number
  max: number
  status: string
  detail: string
}

interface ScoreReport {
  score: number
  grade: string
  summary: string
  factors: ScoreFactor[]
}

interface LoginLogItem {
  id: number
  created_at: string
  username: string
  ip: string
  user_agent: string
  success: boolean
  reason: string
}

const router = useRouter()
const { t } = useI18n()
const activeTab = ref('scan')
const risks = ref<RiskItem[]>([])
const score = ref<ScoreReport | null>(null)
const loginLogs = ref<LoginLogItem[]>([])
const loginTotal = ref(0)
const loading = ref(false)
const logsLoading = ref(false)
const accessLoading = ref(false)
const accessSaving = ref(false)
const fixingKey = ref('')
const fixingAll = ref(false)
const installDialog = ref(false)
const installAppKey = ref('')
const installAppName = ref('')

const panelAccess = ref({
  panel_ip_whitelist_enabled: 'false',
  panel_ip_whitelist: '',
  panel_ip_blacklist: '',
  password_require_strong: 'true',
  panel_security_headers: 'true',
})

const fixableWarns = computed(() =>
  risks.value.filter(r => r.status !== 'pass' && r.fix_type && r.fix_type !== 'none'),
)

const hasFixableWarns = computed(() => fixableWarns.value.length > 0)

const scoreColor = computed(() => {
  const s = score.value?.score ?? 0
  if (s >= 85) return '#67c23a'
  if (s >= 65) return '#e6a23c'
  return '#f56c6c'
})

function isFixable(row: RiskItem) {
  return row.status !== 'pass' && !!row.fix_type && row.fix_type !== 'none'
}

function reasonLabel(reason: string) {
  const map: Record<string, string> = {
    ok: t('panelSecurity.loginReasonOk'),
    invalid_credentials: t('panelSecurity.loginReasonFail'),
    locked: t('panelSecurity.loginReasonLocked'),
  }
  return map[reason] || reason
}

async function loadScore() {
  try {
    const res: any = await api.get('/security/score')
    score.value = res.data || null
  } catch {
    score.value = null
  }
}

async function scan() {
  loading.value = true
  try {
    const res: any = await api.get('/security/scan')
    risks.value = res.data || []
    await loadScore()
  } finally {
    loading.value = false
  }
}

async function loadLoginLogs() {
  logsLoading.value = true
  try {
    const res: any = await api.get('/security/login-logs', { params: { limit: 100 } })
    loginLogs.value = res.data?.items || []
    loginTotal.value = res.data?.total || 0
  } finally {
    logsLoading.value = false
  }
}

async function cleanupLoginLogs() {
  await ElMessageBox.confirm(t('panelSecurity.cleanupConfirm'), t('common.confirm'), { type: 'warning' })
  await api.delete('/security/login-logs', { params: { days: 90 } })
  ElMessage.success(t('panelSecurity.cleanupDone'))
  loadLoginLogs()
}

async function loadPanelAccess() {
  accessLoading.value = true
  try {
    const res: any = await api.get('/security/panel-access')
    const d = res.data || {}
    panelAccess.value = {
      panel_ip_whitelist_enabled: d.panel_ip_whitelist_enabled || 'false',
      panel_ip_whitelist: d.panel_ip_whitelist || '',
      panel_ip_blacklist: d.panel_ip_blacklist || '',
      password_require_strong: d.password_require_strong ?? 'true',
      panel_security_headers: d.panel_security_headers ?? 'true',
    }
  } finally {
    accessLoading.value = false
  }
}

async function savePanelAccess() {
  accessSaving.value = true
  try {
    await api.put('/security/panel-access', panelAccess.value)
    ElMessage.success(t('panelSecurity.accessSaved'))
    await loadScore()
    await scan()
  } catch (e: any) {
    ElMessage.error(e?.error || e?.message || t('common.failed'))
  } finally {
    accessSaving.value = false
  }
}

async function handleFixResult(result: FixResult) {
  if (result.install_job_key) {
    installAppKey.value = result.install_job_key
    installAppName.value = result.install_job_key === 'fail2ban' ? 'Fail2ban' : result.install_job_key
    installDialog.value = true
    if (result.message) ElMessage.success(result.message)
    return
  }
  if (result.needs_guide) {
    await ElMessageBox.confirm(
      result.guide_message || t('securityCheck.guideDefault'),
      t('securityCheck.guideTitle'),
      {
        confirmButtonText: result.redirect_path ? t('securityCheck.goConfigure') : t('common.ok'),
        cancelButtonText: t('common.cancel'),
        type: 'info',
      },
    ).then(() => {
      if (result.redirect_path) router.push(result.redirect_path)
    }).catch(() => {})
    return
  }
  if (result.success) {
    ElMessage.success(result.message || t('securityCheck.fixSuccess'))
    await scan()
  }
}

async function fixItem(row: RiskItem) {
  if (!isFixable(row)) return
  fixingKey.value = row.key
  try {
    const res: any = await api.post(`/security/check/${row.key}/fix`)
    await handleFixResult(res.data || {})
  } catch (e: any) {
    ElMessage.error(e?.error || e?.message || t('securityCheck.fixFailed'))
  } finally {
    fixingKey.value = ''
  }
}

async function fixAll() {
  if (!hasFixableWarns.value) return
  fixingAll.value = true
  try {
    const res: any = await api.post('/security/check/fix-all')
    const results: Array<{ key: string; result?: FixResult; error?: string }> = res.data?.results || []
    let successCount = 0
    let guideCount = 0
    let failCount = 0
    for (const item of results) {
      if (item.error) {
        failCount++
        continue
      }
      const r = item.result
      if (!r) continue
      if (r.install_job_key) {
        installAppKey.value = r.install_job_key
        installAppName.value = r.install_job_key === 'fail2ban' ? 'Fail2ban' : r.install_job_key
        installDialog.value = true
        successCount++
      } else if (r.needs_guide) {
        guideCount++
      } else if (r.success) {
        successCount++
      } else {
        failCount++
      }
    }
    await scan()
    if (guideCount > 0) {
      ElMessage.warning(t('securityCheck.fixAllGuideHint', { n: guideCount }))
    }
    if (successCount > 0) {
      ElMessage.success(t('securityCheck.fixAllSuccess', { n: successCount }))
    }
    if (failCount > 0 && successCount === 0 && guideCount === 0) {
      ElMessage.error(t('securityCheck.fixAllFailed'))
    }
  } catch (e: any) {
    ElMessage.error(e?.error || e?.message || t('securityCheck.fixFailed'))
  } finally {
    fixingAll.value = false
  }
}

function onInstallDone(payload: { success: boolean }) {
  if (payload.success) scan()
}

function onTabChange(tab: string) {
  if (tab === 'logs' && !loginLogs.value.length) loadLoginLogs()
  if (tab === 'access' && !accessLoading.value) loadPanelAccess()
}

scan()
loadScore()
</script>

<template>
  <div>
    <div class="page-header" :class="{ 'page-header--embedded': props.embedded }">
      <h2 v-if="!props.embedded">{{ t('page.security') }}</h2>
      <div class="header-actions">
        <el-button
          v-if="activeTab === 'scan' && hasFixableWarns"
          type="warning"
          :loading="fixingAll"
          @click="fixAll"
        >
          {{ t('securityCheck.fixAll') }}
        </el-button>
        <el-button v-if="activeTab === 'scan'" type="primary" :loading="loading" @click="scan">{{ t('waf.scan') }}</el-button>
      </div>
    </div>

    <div v-if="score" class="score-card">
      <div class="score-ring" :style="{ borderColor: scoreColor }">
        <span class="score-value" :style="{ color: scoreColor }">{{ score.score }}</span>
        <span class="score-grade">{{ score.grade }}</span>
      </div>
      <div class="score-meta">
        <h3>{{ t('panelSecurity.scoreTitle') }}</h3>
        <p>{{ score.summary }}</p>
        <div class="score-factors">
          <el-tag
            v-for="f in score.factors"
            :key="f.key"
            size="small"
            :type="f.status === 'ok' ? 'success' : f.status === 'warn' ? 'warning' : 'danger'"
          >
            {{ f.name }} {{ f.score }}/{{ f.max }}
          </el-tag>
        </div>
      </div>
    </div>

    <el-tabs v-model="activeTab" @tab-change="onTabChange">
      <el-tab-pane :label="t('panelSecurity.tabScan')" name="scan">
        <el-alert :title="t('waf.scanHint')" type="warning" show-icon :closable="false" style="margin-bottom: 16px" />
        <el-table :data="risks" stripe v-loading="loading">
          <el-table-column prop="name" :label="t('waf.checkItem')" />
          <el-table-column prop="level" :label="t('waf.riskLevel')" width="100">
            <template #default="{ row }">
              <el-tag :type="row.level === 'high' ? 'danger' : row.level === 'medium' ? 'warning' : 'info'">{{ row.level }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="status" :label="t('common.status')" width="100">
            <template #default="{ row }">
              <el-tag :type="row.status === 'pass' ? 'success' : 'warning'">{{ row.status }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="solution" :label="t('waf.solution')" show-overflow-tooltip />
          <el-table-column :label="t('common.actions')" width="100" fixed="right">
            <template #default="{ row }">
              <el-button
                v-if="isFixable(row)"
                type="primary"
                link
                size="small"
                :loading="fixingKey === row.key"
                @click="fixItem(row)"
              >
                {{ t('securityCheck.fix') }}
              </el-button>
              <span v-else class="muted">—</span>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane :label="t('panelSecurity.tabLoginLogs')" name="logs">
        <div class="tab-toolbar">
          <span class="muted">{{ t('panelSecurity.loginLogsHint', { n: loginTotal }) }}</span>
          <el-button size="small" :loading="logsLoading" @click="loadLoginLogs">{{ t('common.refresh') }}</el-button>
          <el-button size="small" type="warning" @click="cleanupLoginLogs">{{ t('panelSecurity.cleanupLogs') }}</el-button>
        </div>
        <el-table :data="loginLogs" stripe v-loading="logsLoading">
          <el-table-column prop="created_at" :label="t('panelSecurity.time')" width="170">
            <template #default="{ row }">{{ new Date(row.created_at).toLocaleString() }}</template>
          </el-table-column>
          <el-table-column prop="username" :label="t('common.username')" width="120" />
          <el-table-column prop="ip" label="IP" width="140" />
          <el-table-column prop="success" :label="t('common.status')" width="90">
            <template #default="{ row }">
              <el-tag :type="row.success ? 'success' : 'danger'" size="small">
                {{ row.success ? t('panelSecurity.loginSuccess') : t('panelSecurity.loginFailed') }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="reason" :label="t('panelSecurity.reason')" width="120">
            <template #default="{ row }">{{ reasonLabel(row.reason) }}</template>
          </el-table-column>
          <el-table-column prop="user_agent" :label="t('panelSecurity.userAgent')" show-overflow-tooltip />
        </el-table>
      </el-tab-pane>

      <el-tab-pane :label="t('panelSecurity.tabAccess')" name="access">
        <el-alert :title="t('panelSecurity.accessHint')" type="info" show-icon :closable="false" style="margin-bottom: 16px" />
        <el-form v-loading="accessLoading" label-width="180px" style="max-width: 720px">
          <el-form-item :label="t('panelSecurity.ipWhitelistEnabled')">
            <el-switch v-model="panelAccess.panel_ip_whitelist_enabled" active-value="true" inactive-value="false" />
          </el-form-item>
          <el-form-item :label="t('panelSecurity.ipWhitelist')">
            <el-input
              v-model="panelAccess.panel_ip_whitelist"
              type="textarea"
              :rows="4"
              :placeholder="t('panelSecurity.ipListPlaceholder')"
            />
          </el-form-item>
          <el-form-item :label="t('panelSecurity.ipBlacklist')">
            <el-input
              v-model="panelAccess.panel_ip_blacklist"
              type="textarea"
              :rows="3"
              :placeholder="t('panelSecurity.ipListPlaceholder')"
            />
          </el-form-item>
          <el-form-item :label="t('panelSecurity.strongPassword')">
            <el-switch v-model="panelAccess.password_require_strong" active-value="true" inactive-value="false" />
          </el-form-item>
          <el-form-item :label="t('panelSecurity.securityHeaders')">
            <el-switch v-model="panelAccess.panel_security_headers" active-value="true" inactive-value="false" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" :loading="accessSaving" @click="savePanelAccess">{{ t('common.save') }}</el-button>
          </el-form-item>
        </el-form>
      </el-tab-pane>
    </el-tabs>

    <SoftwareInstallLogDialog
      v-model="installDialog"
      :app-key="installAppKey"
      :app-name="installAppName"
      :trigger-install="false"
      @done="onInstallDone"
    />
  </div>
</template>

<style scoped>
.header-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.muted {
  color: var(--el-text-color-placeholder);
}

.score-card {
  display: flex;
  gap: 20px;
  align-items: center;
  margin-bottom: 20px;
  padding: 16px 20px;
  border-radius: 8px;
  background: var(--el-fill-color-lighter);
}

.score-ring {
  width: 88px;
  height: 88px;
  border-radius: 50%;
  border: 4px solid;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.score-value {
  font-size: 28px;
  font-weight: 700;
  line-height: 1;
}

.score-grade {
  font-size: 14px;
  color: var(--el-text-color-secondary);
}

.score-meta h3 {
  margin: 0 0 6px;
  font-size: 16px;
}

.score-meta p {
  margin: 0 0 10px;
  color: var(--el-text-color-secondary);
}

.score-factors {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.tab-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}
</style>
