<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import api, { resolveApiError } from '@/api'
import { ElMessage, ElMessageBox } from 'element-plus'
import SoftwareInstallLogDialog from '@/components/SoftwareInstallLogDialog.vue'
import { cfTheme } from '@/config/theme'

const props = withDefaults(defineProps<{ embedded?: boolean }>(), { embedded: false })

interface StatusInfo {
  k3s_running: boolean
  cilium_ready: boolean
  host_firewall_on: boolean
  hubble_enabled: boolean
  cilium_version?: string
  ready_pods: number
  total_pods: number
  kernel_ok: boolean
  hint?: string
  linux_only: boolean
  hubble_ui_hint?: string
}

interface SetupStep {
  key: string
  title: string
  description: string
  done: boolean
  current: boolean
  action?: string
}

interface ChecklistItem {
  key: string
  label: string
  pass: boolean
  level: string
  hint?: string
}

interface PresetItem {
  key: string
  name: string
  description: string
  ports?: string
  applied: boolean
}

interface PolicyRow {
  name: string
  namespace: string
  kind: string
}

const { t } = useI18n()
const activeTab = ref('dashboard')
const loading = ref(false)
const applying = ref(false)
const stackLoading = ref(false)
const wizardLoading = ref(false)

const dashboard = ref<{
  status: StatusInfo | null
  health_score: number
  setup_steps: SetupStep[]
  checklist: ChecklistItem[]
  policy_count: number
  presets: PresetItem[]
} | null>(null)

const policies = ref<PolicyRow[]>([])
const policyYaml = ref('')
const advancedOpen = ref<string[]>([])

const installDialog = ref(false)
const installAppKey = ref('k3s')
const installAppName = ref('K3s')
const installTrigger = ref(false)

const form = reactive({
  host_firewall_enabled: true,
  hubble_enabled: true,
  hubble_ui_enabled: true,
  audit_mode: true,
  network_device: '',
})

const status = computed(() => dashboard.value?.status || null)
const ciliumReady = computed(() => !!status.value?.cilium_ready)
const setupSteps = computed(() => dashboard.value?.setup_steps || [])
const presets = computed(() => dashboard.value?.presets || [])
const healthScore = computed(() => dashboard.value?.health_score ?? 0)
const healthColor = computed(() => {
  if (healthScore.value >= 80) return cfTheme.success
  if (healthScore.value >= 50) return cfTheme.warning
  return cfTheme.danger
})

const statusCards = computed(() => {
  const s = status.value
  if (!s) return []
  return [
    { key: 'k3s', label: t('cilium.cardK3s'), value: s.k3s_running ? t('cilium.running') : t('cilium.stopped'), type: s.k3s_running ? 'success' : 'info' },
    { key: 'cilium', label: t('cilium.cardCilium'), value: s.cilium_ready ? t('cilium.ready') : t('cilium.notReady'), type: s.cilium_ready ? 'success' : 'warning' },
    { key: 'host', label: t('cilium.cardHostFw'), value: s.host_firewall_on ? t('common.yes') : t('common.no'), type: s.host_firewall_on ? 'success' : 'info' },
    { key: 'pods', label: t('cilium.cardPods'), value: `${s.ready_pods}/${s.total_pods}`, type: s.cilium_ready ? 'success' : 'warning' },
    { key: 'kernel', label: t('cilium.cardKernel'), value: s.kernel_ok ? '5.10+' : '<5.10', type: s.kernel_ok ? 'success' : 'warning' },
    { key: 'policies', label: t('cilium.cardPolicies'), value: String(dashboard.value?.policy_count ?? 0), type: (dashboard.value?.policy_count ?? 0) > 0 ? 'success' : 'info' },
  ] as const
})

const guideFaq = computed(() => [
  t('cilium.faq1'), t('cilium.faq2'), t('cilium.faq3'), t('cilium.faq4'), t('cilium.faq5'),
])

const compareRows = computed(() => [
  { aspect: t('cilium.compareScope'), ufw: t('cilium.compareUfwScope'), cilium: t('cilium.compareCiliumScope') },
  { aspect: t('cilium.compareTech'), ufw: t('cilium.compareUfwTech'), cilium: t('cilium.compareCiliumTech') },
  { aspect: t('cilium.compareNeed'), ufw: t('cilium.compareUfwNeed'), cilium: t('cilium.compareCiliumNeed') },
  { aspect: t('cilium.compareUse'), ufw: t('cilium.compareUfwUse'), cilium: t('cilium.compareCiliumUse') },
])

async function loadDashboard() {
  const res: any = await api.get('/cilium/dashboard')
  dashboard.value = res.data || null
  if (dashboard.value?.status) {
    /* synced via config load */
  }
}

async function loadConfig() {
  const res: any = await api.get('/cilium/config')
  const cfg = res.data || {}
  form.host_firewall_enabled = cfg.host_firewall_enabled ?? true
  form.hubble_enabled = cfg.hubble_enabled ?? true
  form.hubble_ui_enabled = cfg.hubble_ui_enabled ?? true
  form.audit_mode = cfg.audit_mode ?? true
  form.network_device = cfg.network_device || ''
}

async function loadPolicies() {
  try {
    const res: any = await api.get('/cilium/policies')
    policies.value = res.data || []
  } catch {
    policies.value = []
  }
}

async function loadAll() {
  loading.value = true
  try {
    await Promise.all([loadDashboard(), loadConfig(), loadPolicies()])
  } finally {
    loading.value = false
  }
}

async function applyCilium() {
  applying.value = true
  try {
    await api.patch('/cilium/config', { ...form })
    const res: any = await api.post('/cilium/apply')
    ElMessage.success(res.data?.message || t('cilium.helmApplied'))
    await loadAll()
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('cilium.applyFailed')))
  } finally {
    applying.value = false
  }
}

async function runWizard() {
  await ElMessageBox.confirm(t('cilium.wizardConfirm'), t('common.confirm'), { type: 'info' })
  wizardLoading.value = true
  try {
    const res: any = await api.post('/cilium/wizard')
    const steps = (res.data?.steps || []).join('\n')
    ElMessage.success(res.data?.message || t('cilium.wizardDone'))
    if (steps) ElMessageBox.alert(steps, t('cilium.wizardTitle'), { type: 'success' })
    await loadAll()
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
  } finally {
    wizardLoading.value = false
  }
}

async function installStack() {
  await ElMessageBox.confirm(t('cilium.installStackConfirm'), t('common.confirm'), { type: 'info' })
  stackLoading.value = true
  try {
    const res: any = await api.post('/cilium/install-stack', { install_k3s: true, install_cilium: true })
    ElMessage.success(res.data?.message || t('cilium.installStackDone'))
    await loadAll()
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
  } finally {
    stackLoading.value = false
  }
}

async function applyPreset(key: string) {
  applying.value = true
  try {
    const res: any = await api.post(`/cilium/policies/preset/${key}`)
    ElMessage.success(res.data?.message || t('cilium.presetApplied'))
    await loadAll()
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
  } finally {
    applying.value = false
  }
}

async function applyBaseline() {
  applying.value = true
  try {
    const res: any = await api.post('/cilium/policies/baseline')
    ElMessage.success(res.data?.message || t('cilium.baselineApplied'))
    await loadAll()
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
  } finally {
    applying.value = false
  }
}

async function toggleAudit(enabled: boolean) {
  applying.value = true
  try {
    await api.post('/cilium/audit-mode', { enabled })
    form.audit_mode = enabled
    ElMessage.success(enabled ? t('cilium.auditOn') : t('cilium.auditOff'))
    await loadAll()
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
  } finally {
    applying.value = false
  }
}

async function runStepAction(step: SetupStep) {
  if (step.done) return
  switch (step.action) {
    case 'install_k3s':
      await installStack()
      break
    case 'install_cilium':
      openInstall('cilium', 'Cilium')
      break
    case 'apply_helm':
      form.host_firewall_enabled = true
      await applyCilium()
      break
    case 'apply_baseline':
      await applyBaseline()
      break
    case 'disable_audit':
      await toggleAudit(false)
      break
    default:
      break
  }
}

function openInstall(key: string, name: string) {
  installAppKey.value = key
  installAppName.value = name
  installTrigger.value = true
  installDialog.value = true
}

function onInstallDone(payload: { success: boolean }) {
  if (payload.success) loadAll()
}

async function applyPolicyYaml() {
  if (!policyYaml.value.trim()) return
  applying.value = true
  try {
    const res: any = await api.post('/cilium/policies', { yaml: policyYaml.value })
    ElMessage.success(res.data?.message || t('cilium.policyApplied'))
    policyYaml.value = ''
    await loadAll()
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
  } finally {
    applying.value = false
  }
}

async function deletePolicy(row: PolicyRow) {
  await ElMessageBox.confirm(t('cilium.policyDeleteConfirm', { name: row.name }), t('common.confirm'), { type: 'warning' })
  try {
    await api.delete(`/cilium/policies/${encodeURIComponent(row.name)}`, {
      params: { kind: row.kind, namespace: row.namespace },
    })
    ElMessage.success(t('cilium.policyDeleted'))
    await loadAll()
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
  }
}

onMounted(loadAll)
</script>

<template>
  <div class="cilium-view" v-loading="loading">
    <div class="cilium-header">
      <div>
        <h3 v-if="!props.embedded" class="view-title">{{ t('cilium.statusTitle') }}</h3>
        <p class="view-sub">{{ t('cilium.statusHint') }}</p>
      </div>
      <div class="header-actions">
        <el-button :loading="loading" @click="loadAll">{{ t('common.refresh') }}</el-button>
        <el-button type="primary" :loading="wizardLoading" :disabled="status?.linux_only" @click="runWizard">
          {{ t('cilium.wizardOneClick') }}
        </el-button>
      </div>
    </div>

    <el-tabs v-model="activeTab" type="border-card" class="cilium-tabs">
      <!-- 看板 -->
      <el-tab-pane :label="t('cilium.tabDashboard')" name="dashboard">
        <el-row :gutter="16" class="dashboard-top">
          <el-col :xs="24" :md="6">
            <el-card shadow="never" class="health-card">
              <div class="health-ring">
                <el-progress type="dashboard" :percentage="healthScore" :color="healthColor" :width="120">
                  <template #default>
                    <span class="health-num">{{ healthScore }}</span>
                    <span class="health-label">{{ t('cilium.healthScore') }}</span>
                  </template>
                </el-progress>
              </div>
              <p v-if="status?.hint" class="health-hint">{{ status.hint }}</p>
            </el-card>
          </el-col>
          <el-col :xs="24" :md="18">
            <el-card shadow="never" class="status-card">
              <template #header><span>{{ t('cilium.statusOverview') }}</span></template>
              <el-row :gutter="10">
                <el-col v-for="c in statusCards" :key="c.key" :xs="12" :sm="8" :md="4">
                  <div class="stat-pill">
                    <div class="stat-label">{{ c.label }}</div>
                    <el-tag :type="c.type" size="small" round>{{ c.value }}</el-tag>
                  </div>
                </el-col>
              </el-row>
            </el-card>
          </el-col>
        </el-row>

        <el-card shadow="never" class="section-card">
          <template #header><span>{{ t('cilium.setupWizard') }}</span></template>
          <el-steps :active="setupSteps.findIndex(s => s.current)" finish-status="success" align-center class="setup-steps">
            <el-step
              v-for="step in setupSteps"
              :key="step.key"
              :title="step.title"
              :description="step.description"
              :status="step.done ? 'success' : step.current ? 'process' : 'wait'"
            />
          </el-steps>
          <div class="step-actions">
            <el-button
              v-for="step in setupSteps.filter(s => s.current && !s.done)"
              :key="step.key"
              type="primary"
              size="small"
              @click="runStepAction(step)"
            >
              {{ t('cilium.doStep', { step: step.title }) }}
            </el-button>
          </div>
        </el-card>

        <el-card shadow="never" class="section-card">
          <template #header>
            <div class="card-header-row">
              <span>{{ t('cilium.quickPresets') }}</span>
              <el-button type="primary" size="small" :disabled="!ciliumReady" :loading="applying" @click="applyBaseline">
                {{ t('cilium.applyBaseline') }}
              </el-button>
            </div>
          </template>
          <el-row :gutter="12">
            <el-col v-for="p in presets" :key="p.key" :xs="24" :sm="12" :md="8">
              <div class="preset-card" :class="{ applied: p.applied }">
                <div class="preset-head">
                  <strong>{{ p.name }}</strong>
                  <el-tag v-if="p.applied" type="success" size="small">{{ t('cilium.applied') }}</el-tag>
                </div>
                <p class="preset-desc">{{ p.description }}</p>
                <div v-if="p.ports" class="preset-ports">{{ p.ports }}</div>
                <el-button size="small" type="primary" plain :disabled="!ciliumReady || p.applied" :loading="applying" @click="applyPreset(p.key)">
                  {{ p.applied ? t('cilium.applied') : t('cilium.oneClickApply') }}
                </el-button>
              </div>
            </el-col>
          </el-row>
        </el-card>

        <el-card shadow="never" class="section-card">
          <template #header><span>{{ t('cilium.checklist') }}</span></template>
          <el-table :data="dashboard?.checklist || []" size="small" stripe>
            <el-table-column prop="label" :label="t('cilium.checkItem')" />
            <el-table-column :label="t('cilium.checkStatus')" width="100">
              <template #default="{ row }">
                <el-tag :type="row.pass ? 'success' : 'warning'" size="small">{{ row.pass ? t('cilium.pass') : t('cilium.pending') }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="hint" :label="t('cilium.checkHint')" show-overflow-tooltip />
          </el-table>
        </el-card>
      </el-tab-pane>

      <!-- 策略 -->
      <el-tab-pane :label="t('cilium.tabPolicies')" name="policies">
        <el-alert :title="t('cilium.policiesHint')" type="info" :closable="false" show-icon class="mb-16" />
        <el-table :data="policies" stripe>
          <el-table-column prop="name" :label="t('common.name')" />
          <el-table-column prop="kind" :label="t('common.type')" width="240" />
          <el-table-column prop="namespace" :label="t('cilium.namespace')" width="120" />
          <el-table-column :label="t('common.actions')" width="100">
            <template #default="{ row }">
              <el-button type="danger" text size="small" @click="deletePolicy(row)">{{ t('common.delete') }}</el-button>
            </template>
          </el-table-column>
        </el-table>

        <el-collapse v-model="advancedOpen" class="advanced-collapse">
          <el-collapse-item :title="t('cilium.advancedYaml')" name="yaml">
            <el-input v-model="policyYaml" type="textarea" :rows="12" :placeholder="t('cilium.policyPlaceholder')" />
            <el-button type="primary" size="small" class="mt-8" :loading="applying" :disabled="!ciliumReady" @click="applyPolicyYaml">
              {{ t('cilium.applyPolicy') }}
            </el-button>
          </el-collapse-item>
        </el-collapse>
      </el-tab-pane>

      <!-- 设置 -->
      <el-tab-pane :label="t('cilium.tabSettings')" name="settings">
        <el-form label-width="200px" style="max-width: 680px">
          <el-form-item :label="t('cilium.hostFirewall')">
            <el-switch v-model="form.host_firewall_enabled" />
          </el-form-item>
          <el-form-item :label="t('cilium.hubble')">
            <el-switch v-model="form.hubble_enabled" />
          </el-form-item>
          <el-form-item :label="t('cilium.hubbleUI')">
            <el-switch v-model="form.hubble_ui_enabled" />
          </el-form-item>
          <el-form-item :label="t('cilium.auditMode')">
            <el-switch v-model="form.audit_mode" />
            <div class="form-hint">{{ t('cilium.auditModeHint') }}</div>
          </el-form-item>
          <el-form-item :label="t('cilium.networkDevice')">
            <el-input v-model="form.network_device" :placeholder="t('cilium.networkDevicePlaceholder')" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" :loading="applying" :disabled="!status?.k3s_running" @click="applyCilium">
              {{ t('cilium.applyHelm') }}
            </el-button>
            <el-button :disabled="!ciliumReady" @click="toggleAudit(true)">{{ t('cilium.enableAudit') }}</el-button>
            <el-button type="warning" :disabled="!ciliumReady" @click="toggleAudit(false)">{{ t('cilium.disableAudit') }}</el-button>
          </el-form-item>
        </el-form>
        <el-alert v-if="status?.hubble_ui_hint" :title="t('cilium.hubbleAccess')" type="info" :closable="false" show-icon>
          <code class="cmd">{{ status.hubble_ui_hint }}</code>
        </el-alert>
      </el-tab-pane>

      <!-- 教程 -->
      <el-tab-pane :label="t('cilium.tabGuide')" name="guide">
        <el-card shadow="never">
          <h4>{{ t('cilium.guideIntroTitle') }}</h4>
          <p>{{ t('cilium.guideIntro') }}</p>
          <el-steps direction="vertical" :active="5" class="guide-steps">
            <el-step :title="t('cilium.guideStep1Title')" :description="t('cilium.guideStep1')" />
            <el-step :title="t('cilium.guideStep2Title')" :description="t('cilium.guideStep2')" />
            <el-step :title="t('cilium.guideStep3Title')" :description="t('cilium.guideStep3')" />
            <el-step :title="t('cilium.guideStep4Title')" :description="t('cilium.guideStep4')" />
            <el-step :title="t('cilium.guideStep5Title')" :description="t('cilium.guideStep5')" />
          </el-steps>

          <h4 class="mt-24">{{ t('cilium.compareTitle') }}</h4>
          <el-table :data="compareRows" size="small" stripe class="compare-table">
            <el-table-column prop="aspect" :label="t('cilium.compareAspect')" width="120" />
            <el-table-column prop="ufw" :label="t('cilium.compareUfw')" />
            <el-table-column prop="cilium" :label="t('cilium.compareCiliumCol')" />
          </el-table>

          <h4 class="mt-24">{{ t('cilium.faqTitle') }}</h4>
          <ul class="faq-list">
            <li v-for="(item, i) in guideFaq" :key="i">{{ item }}</li>
          </ul>
          <p class="tutorial-note">{{ t('cilium.ufwNote') }}</p>
        </el-card>
      </el-tab-pane>
    </el-tabs>

    <SoftwareInstallLogDialog
      v-model="installDialog"
      :app-key="installAppKey"
      :app-name="installAppName"
      :trigger="installTrigger"
      @done="onInstallDone"
    />
  </div>
</template>

<style scoped>
.cilium-view {
  min-height: 200px;
}
.cilium-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}
.view-title {
  margin: 0 0 4px;
  font-size: 18px;
}
.view-sub {
  margin: 0;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}
.header-actions {
  display: flex;
  gap: 8px;
}
.cilium-tabs :deep(.el-tabs__content) {
  padding-top: 16px;
}
.dashboard-top {
  margin-bottom: 16px;
}
.health-card {
  text-align: center;
  height: 100%;
}
.health-ring {
  display: flex;
  justify-content: center;
  padding: 8px 0;
}
.health-num {
  display: block;
  font-size: 22px;
  font-weight: 700;
}
.health-label {
  font-size: 11px;
  color: var(--el-text-color-secondary);
}
.health-hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin: 8px 0 0;
}
.stat-pill {
  padding: 8px 4px;
  text-align: center;
}
.stat-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-bottom: 6px;
}
.section-card {
  margin-bottom: 16px;
}
.setup-steps {
  margin: 16px 0;
}
.step-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  justify-content: center;
  margin-top: 12px;
}
.card-header-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.preset-card {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 14px;
  margin-bottom: 12px;
  height: calc(100% - 12px);
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.preset-card.applied {
  border-color: var(--el-color-success-light-5);
  background: var(--el-color-success-light-9);
}
.preset-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.preset-desc {
  margin: 0;
  font-size: 13px;
  color: var(--el-text-color-secondary);
  flex: 1;
}
.preset-ports {
  font-size: 12px;
  font-family: monospace;
  color: var(--el-color-primary);
}
.form-hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
}
.mb-16 {
  margin-bottom: 16px;
}
.mt-8 {
  margin-top: 8px;
}
.mt-24 {
  margin-top: 24px;
}
.advanced-collapse {
  margin-top: 16px;
}
.guide-steps {
  margin-top: 16px;
}
.compare-table {
  margin-top: 12px;
}
.faq-list {
  line-height: 1.8;
  padding-left: 20px;
}
.tutorial-note {
  margin-top: 16px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.cmd {
  font-size: 12px;
  word-break: break-all;
}
</style>
