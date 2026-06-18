<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import api, { AI_REQUEST_TIMEOUT } from '@/api'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Folder, Link, MagicStick } from '@element-plus/icons-vue'
import FileCodeEditor from '@/components/FileCodeEditor.vue'
import LogViewer from '@/components/LogViewer.vue'
import SiteLogAIPanel from '@/components/SiteLogAIPanel.vue'
import SiteProjectAIPanel from '@/components/SiteProjectAIPanel.vue'
import AIChatPanel, { type AIChatMessage } from '@/components/AIChatPanel.vue'
import { rewriteCategoryOrder, rewriteTemplates } from '@/config/rewriteTemplates'
import { siteVisitUrl } from '@/utils/siteUrl'

const props = defineProps<{
  visible: boolean
  siteId: number | null
  initialMenu?: string
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  updated: []
}>()

const { t } = useI18n()

const activeMenu = ref('domain')
const site = ref<any>(null)
const options = ref<any>({})
const loading = ref(false)
const saving = ref(false)

const domainsText = ref('')
const domainList = ref<any[]>([])
const selectedDomainIds = ref<number[]>([])

const settings = ref({
  root_path: '',
  php_version: '8.3',
  ssl: false,
  remark: '',
  expires_permanent: true,
  expires_at: '',
  ssl_san_domains: '',
  force_https: false,
  index_files: '',
  rewrite_rules: '',
  redirect_url: '',
  proxy_pass: '',
  cache_enabled: false,
  cache_dev_mode: false,
  cache_html_ttl: 0,
  cache_static_ttl: 0,
  access_auth_enabled: false,
  access_auth_user: '',
  access_auth_pass: '',
  access_allow_ips: '',
  access_deny_ips: '',
  traffic_limit_enabled: false,
  traffic_rate: '10r/s',
  traffic_burst: 20,
  hotlink_enabled: false,
  hotlink_allow_empty: true,
  hotlink_allow_domains: '',
})

const subdirs = ref<any[]>([])
const subdirDialog = ref(false)
const editingSubdir = ref<any>(null)
const subdirForm = ref({ prefix: '', root_path: '', remark: '' })

const selectedRewriteTemplate = ref('')
const rewriteTemplateGroups = computed(() =>
  rewriteCategoryOrder.map((cat) => ({
    category: cat,
    label: t(`siteModify.rewriteCategories.${cat}`),
    items: rewriteTemplates.filter((x) => x.category === cat),
  })).filter((g) => g.items.length > 0)
)

const nginxContent = ref('')
const logs = ref<any>({ access_tail: '', error_tail: '', access_log: '', error_log: '' })
const composerCmd = ref('install --no-interaction')
const composerOutput = ref('')
const composerRunning = ref(false)
const sslEmail = ref('')
const sslIssuing = ref(false)

const siteAiDrawer = ref(false)
const siteAiTab = ref('project')
const siteAiMessages = ref<AIChatMessage[]>([])

const pathPickerVisible = ref(false)
const pathEntries = ref<any[]>([])
const browsePath = ref('/')

const menuItems = computed(() => [
  { key: 'domain', label: t('siteModify.domainManage') },
  { key: 'subdir', label: t('siteModify.subdirBind') },
  { key: 'directory', label: t('siteModify.siteDirectory') },
  { key: 'access', label: t('siteModify.accessLimit') },
  { key: 'traffic', label: t('siteModify.trafficControl') },
  { key: 'rewrite', label: t('siteModify.urlRewrite') },
  { key: 'index', label: t('siteModify.defaultDoc') },
  { key: 'config', label: t('siteModify.config') },
  { key: 'ssl', label: t('siteModify.ssl') },
  { key: 'php', label: t('siteModify.phpVersion') },
  { key: 'composer', label: t('siteModify.composer') },
  { key: 'redirect', label: t('siteModify.redirect') },
  { key: 'proxy', label: t('siteModify.reverseProxy') },
  { key: 'cache', label: t('siteModify.cdnCache') },
  { key: 'hotlink', label: t('siteModify.hotlink') },
  { key: 'logs', label: t('siteModify.responseLog') },
])

const dialogVisible = computed({
  get: () => props.visible,
  set: (v: boolean) => emit('update:visible', v),
})

const title = computed(() => {
  if (!site.value) return t('siteModify.title')
  const created = site.value.created_at
    ? new Date(site.value.created_at).toLocaleString()
    : ''
  return t('siteModify.titleWithDomain', { domain: site.value.domain, time: created })
})

function visitDomainUrl(row: { domain: string; port?: number }) {
  return siteVisitUrl(row.domain, {
    port: row.port || 80,
    ssl: !!settings.value.ssl,
  })
}

watch(
  () => [props.visible, props.siteId, props.initialMenu] as const,
  async ([vis, id, menu]) => {
    if (vis && id) {
      activeMenu.value = menu || 'domain'
      selectedDomainIds.value = []
      domainsText.value = ''
      await loadSite(id)
    }
  }
)

async function loadSite(id: number) {
  loading.value = true
  try {
    const [siteRes, optRes, domainsRes]: any[] = await Promise.all([
      api.get(`/websites/${id}`),
      api.get('/websites/options'),
      api.get(`/websites/${id}/domains`),
    ])
    site.value = siteRes.data
    options.value = optRes.data || {}
    domainList.value = domainsRes.data || []
    syncSettings()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || e?.error || t('siteModify.loadFailed'))
  } finally {
    loading.value = false
  }
}

function syncSettings() {
  if (!site.value) return
  settings.value = {
    root_path: site.value.root_path || '',
    php_version: site.value.php_version || 'static',
    ssl: !!site.value.ssl,
    force_https: !!site.value.force_https,
    ssl_san_domains: '',
  remark: site.value.remark || '',
  expires_permanent: !site.value.expires_at,
  expires_at: site.value.expires_at ? String(site.value.expires_at).slice(0, 10) : '',
  index_files: site.value.index_files || '',
    rewrite_rules: site.value.rewrite_rules || '',
    redirect_url: site.value.redirect_url || '',
    proxy_pass: site.value.proxy_pass || '',
    cache_enabled: !!site.value.cache_enabled,
    cache_dev_mode: !!site.value.cache_dev_mode,
    cache_html_ttl: site.value.cache_html_ttl || 0,
    cache_static_ttl: site.value.cache_static_ttl || 0,
    access_auth_enabled: !!site.value.access_auth_enabled,
    access_auth_user: site.value.access_auth_user || '',
    access_auth_pass: '',
    access_allow_ips: site.value.access_allow_ips || '',
    access_deny_ips: site.value.access_deny_ips || '',
    traffic_limit_enabled: !!site.value.traffic_limit_enabled,
    traffic_rate: site.value.traffic_rate || '10r/s',
    traffic_burst: site.value.traffic_burst || 20,
    hotlink_enabled: !!site.value.cross_site_protect_enabled || !!site.value.hotlink_enabled,
    hotlink_allow_empty: site.value.hotlink_allow_empty !== false,
    hotlink_allow_domains: site.value.hotlink_allow_domains || '',
  }
  subdirs.value = site.value.subdirs || []
}

async function refreshDomains() {
  if (!props.siteId) return
  const res: any = await api.get(`/websites/${props.siteId}/domains`)
  domainList.value = res.data || []
}

async function handleAddDomains() {
  if (!props.siteId || !domainsText.value.trim()) return
  try {
    const checkRes: any = await api.post('/domains/check', {
      domains_text: domainsText.value,
      exclude_website_id: props.siteId,
    })
    if (!checkRes.data?.available) {
      const c = checkRes.data?.conflicts?.[0]
      ElMessage.error(c ? `${c.domain}: ${c.owner}` : t('websites.domainTaken'))
      return
    }
    await api.post(`/websites/${props.siteId}/domains`, { domains_text: domainsText.value })
    ElMessage.success(t('siteModify.domainAdded'))
    domainsText.value = ''
    await refreshDomains()
    emit('updated')
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || e?.error || t('siteModify.saveFailed'))
  }
}

async function handleBatchDeleteDomains() {
  if (!props.siteId || selectedDomainIds.value.length === 0) return
  try {
    await api.post(`/websites/${props.siteId}/domains/batch-delete`, {
      ids: selectedDomainIds.value,
    })
    ElMessage.success(t('siteModify.domainRemoved'))
    selectedDomainIds.value = []
    await refreshDomains()
    emit('updated')
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || e?.error || t('siteModify.saveFailed'))
  }
}

function onDomainSelection(rows: any[]) {
  selectedDomainIds.value = rows.filter((r) => r.type !== 'primary').map((r) => r.id)
}

async function patchSite(patch: Record<string, unknown>) {
  if (!props.siteId) return
  saving.value = true
  try {
    const res: any = await api.patch(`/websites/${props.siteId}`, patch)
    site.value = res.data
    syncSettings()
    ElMessage.success(t('siteModify.saved'))
    emit('updated')
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || e?.error || t('siteModify.saveFailed'))
  } finally {
    saving.value = false
  }
}

async function saveDirectory() {
  await patchSite({
    root_path: settings.value.root_path,
    remark: settings.value.remark,
    expires_at: settings.value.expires_permanent ? '' : settings.value.expires_at,
  })
}

async function savePhp() {
  await patchSite({ php_version: settings.value.php_version })
}

async function saveSsl() {
  await patchSite({ ssl: settings.value.ssl, force_https: settings.value.force_https })
}

async function saveIndex() {
  await patchSite({ index_files: settings.value.index_files })
}

async function saveRewrite() {
  await patchSite({ rewrite_rules: settings.value.rewrite_rules })
}

function applyRewriteTemplate() {
  const tpl = rewriteTemplates.find((x) => x.id === selectedRewriteTemplate.value)
  if (!tpl) return
  const name = t(`siteModify.rewriteTemplates.${tpl.id}`)
  const apply = () => {
    settings.value.rewrite_rules = tpl.rules
    ElMessage.success(t('siteModify.rewriteApplied', { name }))
  }
  if (settings.value.rewrite_rules.trim()) {
    ElMessageBox.confirm(t('siteModify.rewriteReplaceConfirm', { name }), t('common.confirm'), { type: 'warning' })
      .then(apply)
      .catch(() => {})
  } else {
    apply()
  }
}

function clearRewrite() {
  settings.value.rewrite_rules = ''
}

async function saveRedirect() {
  await patchSite({ redirect_url: settings.value.redirect_url })
}

async function saveProxy() {
  await patchSite({ proxy_pass: settings.value.proxy_pass })
}

async function saveCache() {
  await patchSite({
    cache_enabled: settings.value.cache_enabled,
    cache_dev_mode: settings.value.cache_dev_mode,
    cache_html_ttl: settings.value.cache_html_ttl,
    cache_static_ttl: settings.value.cache_static_ttl,
  })
}

async function saveAccess() {
  const patch: Record<string, unknown> = {
    access_auth_enabled: settings.value.access_auth_enabled,
    access_auth_user: settings.value.access_auth_user,
    access_allow_ips: settings.value.access_allow_ips,
    access_deny_ips: settings.value.access_deny_ips,
  }
  if (settings.value.access_auth_pass) {
    patch.access_auth_pass = settings.value.access_auth_pass
  }
  await patchSite(patch)
}

async function saveTraffic() {
  await patchSite({
    traffic_limit_enabled: settings.value.traffic_limit_enabled,
    traffic_rate: settings.value.traffic_rate,
    traffic_burst: settings.value.traffic_burst,
  })
}

async function saveHotlink() {
  await patchSite({
    hotlink_enabled: settings.value.hotlink_enabled,
    hotlink_allow_empty: settings.value.hotlink_allow_empty,
    hotlink_allow_domains: settings.value.hotlink_allow_domains,
    cross_site_protect_enabled: settings.value.hotlink_enabled,
  })
}

async function loadSubdirs() {
  if (!props.siteId) return
  const res: any = await api.get(`/websites/${props.siteId}/subdirs`)
  subdirs.value = res.data || []
}

function openSubdirDialog(row?: any) {
  editingSubdir.value = row || null
  if (row) {
    subdirForm.value = { prefix: row.prefix, root_path: row.root_path, remark: row.remark || '' }
  } else {
    subdirForm.value = { prefix: '', root_path: settings.value.root_path, remark: '' }
  }
  subdirDialog.value = true
}

async function saveSubdir() {
  if (!props.siteId) return
  saving.value = true
  try {
    if (editingSubdir.value?.id) {
      await api.put(`/websites/${props.siteId}/subdirs/${editingSubdir.value.id}`, subdirForm.value)
    } else {
      await api.post(`/websites/${props.siteId}/subdirs`, subdirForm.value)
    }
    ElMessage.success(t('siteModify.saved'))
    subdirDialog.value = false
    await loadSubdirs()
    emit('updated')
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || e?.error || t('siteModify.saveFailed'))
  } finally {
    saving.value = false
  }
}

async function deleteSubdir(row: any) {
  if (!props.siteId) return
  await api.delete(`/websites/${props.siteId}/subdirs/${row.id}`)
  ElMessage.success(t('siteModify.deleted'))
  await loadSubdirs()
  emit('updated')
}

async function loadNginx() {
  if (!props.siteId) return
  const res: any = await api.get(`/websites/${props.siteId}/nginx`)
  nginxContent.value = res.data?.content || ''
}

async function saveNginx() {
  if (!props.siteId) return
  saving.value = true
  try {
    await api.put(`/websites/${props.siteId}/nginx`, { content: nginxContent.value })
    ElMessage.success(t('siteModify.saved'))
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || e?.error || t('siteModify.saveFailed'))
  } finally {
    saving.value = false
  }
}

async function loadLogs() {
  if (!props.siteId) return
  const res: any = await api.get(`/websites/${props.siteId}/logs`, { params: { lines: 200 } })
  logs.value = res.data || {}
}

function siteAiStreamBody(message: string, history: AIChatMessage[]) {
  return {
    message: `[${activeMenuLabel.value}] ${message}`,
    access_log: logs.value.access_tail || '',
    error_log: logs.value.error_tail || '',
    access_path: logs.value.access_log || '',
    error_path: logs.value.error_log || '',
    history: history.filter((m) => !m.streaming).slice(-10).map((m) => ({ role: m.role, content: m.content })),
  }
}

async function siteAiSendFallback(message: string, history: AIChatMessage[], signal: AbortSignal) {
  if (!props.siteId) return ''
  if (!logs.value.access_tail && !logs.value.error_tail) {
    await loadLogs()
  }
  const res: any = await api.post(
    `/websites/${props.siteId}/logs/ai/chat`,
    siteAiStreamBody(message, history),
    { timeout: AI_REQUEST_TIMEOUT, signal } as any,
  )
  return res.data?.reply || ''
}

async function openSiteAiDrawer() {
  siteAiDrawer.value = true
  siteAiTab.value = 'project'
  if (props.siteId && !logs.value.access_tail && !logs.value.error_tail) {
    await loadLogs()
  }
}

const activeMenuLabel = computed(() => menuItems.value.find((m) => m.key === activeMenu.value)?.label || '')

async function onMenuSelect(key: string) {
  activeMenu.value = key
  if (key === 'config') await loadNginx()
  if (key === 'logs') await loadLogs()
  if (key === 'subdir') await loadSubdirs()
}

async function openPathPicker() {
  pathPickerVisible.value = true
  browsePath.value = settings.value.root_path
  await loadPathDir(browsePath.value)
}

async function loadPathDir(path: string) {
  try {
    const res: any = await api.get('/files', { params: { path } })
    pathEntries.value = res.data || []
    browsePath.value = path
  } catch {
    pathEntries.value = []
  }
}

function selectPath(path: string) {
  settings.value.root_path = path
  pathPickerVisible.value = false
}

async function runComposer() {
  if (!props.siteId) return
  composerRunning.value = true
  composerOutput.value = ''
  try {
    const res: any = await api.post(`/websites/${props.siteId}/composer`, { command: composerCmd.value })
    composerOutput.value = res.data?.output || ''
    ElMessage.success(res.data?.ok ? t('siteModify.composerDone') : t('siteModify.composerFailed'))
  } catch (e: any) {
    composerOutput.value = e?.response?.data?.error || e?.error || ''
    ElMessage.error(t('siteModify.composerFailed'))
  } finally {
    composerRunning.value = false
  }
}

async function issueSSL() {
  if (!props.siteId) return
  sslIssuing.value = true
  try {
    await api.post(`/websites/${props.siteId}/ssl/issue`, {
      email: sslEmail.value,
      san_domains: settings.value.ssl_san_domains,
      deploy: true,
    })
    ElMessage.success(t('siteModify.sslIssued'))
    settings.value.ssl = true
    settings.value.force_https = true
    await loadSite(props.siteId)
    emit('updated')
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || e?.error || t('siteModify.saveFailed'))
  } finally {
    sslIssuing.value = false
  }
}
</script>

<template>
  <el-dialog
    v-model="dialogVisible"
    :title="title"
    :width="activeMenu === 'logs' ? '1080px' : '920px'"
    top="4vh"
    destroy-on-close
    class="site-modify-dialog"
  >
    <div v-loading="loading" class="site-modify-body">
      <el-menu
        :default-active="activeMenu"
        class="site-modify-menu"
        @select="onMenuSelect"
      >
        <el-menu-item v-for="item in menuItems" :key="item.key" :index="item.key">
          <span class="menu-item-label">
            {{ item.label }}
            <el-icon v-if="item.key === 'logs'" class="menu-ai-icon"><MagicStick /></el-icon>
          </span>
        </el-menu-item>
      </el-menu>

      <div class="site-modify-panel">
        <!-- 域名管理 -->
        <div v-show="activeMenu === 'domain'" class="panel-section">
          <el-input
            v-model="domainsText"
            type="textarea"
            :rows="6"
            :placeholder="`${t('websites.domainHint')}\n${t('websites.domainWildcard')}\n${t('websites.domainPort')}`"
          />
          <div class="panel-actions">
            <el-button type="primary" @click="handleAddDomains">{{ t('siteModify.add') }}</el-button>
          </div>
          <el-table
            :data="domainList"
            stripe
            size="small"
            class="domain-table"
            @selection-change="onDomainSelection"
          >
            <el-table-column type="selection" width="48" :selectable="(row: any) => row.type !== 'primary'" />
            <el-table-column prop="domain" :label="t('websites.domain')" min-width="200">
              <template #default="{ row }">
                <a
                  class="domain-visit-link"
                  :href="visitDomainUrl(row)"
                  target="_blank"
                  rel="noopener noreferrer"
                  :title="t('siteModify.visitSite')"
                  @click.stop
                >
                  {{ row.domain }}
                  <el-icon class="domain-visit-icon"><Link /></el-icon>
                </a>
              </template>
            </el-table-column>
            <el-table-column prop="port" :label="t('websites.port')" width="80" />
            <el-table-column :label="t('common.actions')" width="120">
              <template #default="{ row }">
                <span v-if="row.type === 'primary'" class="muted">{{ t('siteModify.primaryLocked') }}</span>
              </template>
            </el-table-column>
          </el-table>
          <div class="panel-actions">
            <el-button
              type="danger"
              plain
              :disabled="selectedDomainIds.length === 0"
              @click="handleBatchDeleteDomains"
            >
              {{ t('siteModify.batchDelete') }}
            </el-button>
          </div>
        </div>

        <!-- 站点目录 -->
        <div v-show="activeMenu === 'directory'" class="panel-section">
          <el-form label-width="100px">
            <el-form-item :label="t('websites.rootPath')">
              <el-input v-model="settings.root_path">
                <template #append>
                  <el-button :icon="Folder" @click="openPathPicker">{{ t('websites.selectPath') }}</el-button>
                </template>
              </el-input>
            </el-form-item>
            <el-form-item :label="t('websites.remark')">
              <el-input v-model="settings.remark" type="textarea" :rows="2" />
            </el-form-item>
            <el-form-item :label="t('websites.expires')">
              <div class="expires-form">
                <el-checkbox v-model="settings.expires_permanent">{{ t('websites.permanent') }}</el-checkbox>
                <el-date-picker
                  v-if="!settings.expires_permanent"
                  v-model="settings.expires_at"
                  type="date"
                  value-format="YYYY-MM-DD"
                  :placeholder="t('websites.expiresDatePlaceholder')"
                  style="width: 100%"
                />
              </div>
              <p v-if="!settings.expires_permanent" class="panel-hint">{{ t('websites.expiresAutoStopHint') }}</p>
            </el-form-item>
          </el-form>
          <el-button type="primary" :loading="saving" @click="saveDirectory">{{ t('siteModify.save') }}</el-button>
        </div>

        <!-- 默认文档 -->
        <div v-show="activeMenu === 'index'" class="panel-section">
          <p class="panel-hint">{{ t('siteModify.indexHint') }}</p>
          <el-input v-model="settings.index_files" placeholder="index.php index.html index.htm" />
          <div class="panel-actions">
            <el-button type="primary" :loading="saving" @click="saveIndex">{{ t('siteModify.save') }}</el-button>
          </div>
        </div>

        <!-- URL 重写 -->
        <div v-show="activeMenu === 'rewrite'" class="panel-section">
          <p class="panel-hint">{{ t('siteModify.rewriteHint') }}</p>
          <div class="rewrite-toolbar">
            <el-select
              v-model="selectedRewriteTemplate"
              filterable
              :placeholder="t('siteModify.rewriteSelectPlaceholder')"
              style="width: 280px"
            >
              <el-option-group
                v-for="group in rewriteTemplateGroups"
                :key="group.category"
                :label="group.label"
              >
                <el-option
                  v-for="item in group.items"
                  :key="item.id"
                  :label="t(`siteModify.rewriteTemplates.${item.id}`)"
                  :value="item.id"
                />
              </el-option-group>
            </el-select>
            <el-button type="primary" plain :disabled="!selectedRewriteTemplate" @click="applyRewriteTemplate">
              {{ t('siteModify.rewriteApply') }}
            </el-button>
            <el-button @click="clearRewrite">{{ t('siteModify.rewriteClear') }}</el-button>
          </div>
          <el-input v-model="settings.rewrite_rules" type="textarea" :rows="14" class="mono-input" />
          <div class="panel-actions">
            <el-button type="primary" :loading="saving" @click="saveRewrite">{{ t('siteModify.save') }}</el-button>
          </div>
        </div>

        <!-- 配置 -->
        <div v-show="activeMenu === 'config'" class="panel-section panel-config">
          <FileCodeEditor v-model="nginxContent" path="nginx.conf" />
          <div class="panel-actions">
            <el-button @click="loadNginx">{{ t('siteModify.reload') }}</el-button>
            <el-button type="primary" :loading="saving" @click="saveNginx">{{ t('siteModify.save') }}</el-button>
          </div>
        </div>

        <!-- SSL -->
        <div v-show="activeMenu === 'ssl'" class="panel-section">
          <el-form label-width="140px">
            <el-form-item :label="t('siteModify.sslEnable')">
              <el-switch v-model="settings.ssl" />
            </el-form-item>
            <el-form-item :label="t('siteModify.sslForceHTTPS')">
              <el-switch v-model="settings.force_https" :disabled="!settings.ssl" />
              <span class="panel-hint-inline">{{ t('siteModify.sslForceHTTPSHint') }}</span>
            </el-form-item>
            <el-form-item :label="t('siteModify.sslEmail')">
              <el-input v-model="sslEmail" placeholder="admin@example.com" />
            </el-form-item>
            <el-form-item :label="t('siteModify.sslSanDomains')">
              <el-input v-model="settings.ssl_san_domains" type="textarea" :rows="2" :placeholder="t('siteModify.sslSanPlaceholder')" />
            </el-form-item>
          </el-form>
          <p class="panel-hint">{{ t('siteModify.sslHint') }}</p>
          <div class="panel-actions">
            <el-button type="success" :loading="sslIssuing" @click="issueSSL">{{ t('siteModify.sslApply') }}</el-button>
            <el-button type="primary" :loading="saving" @click="saveSsl">{{ t('siteModify.save') }}</el-button>
          </div>
        </div>

        <!-- PHP -->
        <div v-show="activeMenu === 'php'" class="panel-section">
          <el-form label-width="100px">
            <el-form-item :label="t('websites.phpVersion')">
              <el-select v-model="settings.php_version" style="width: 240px">
                <el-option
                  v-for="p in options.php_versions || []"
                  :key="p.value"
                  :label="p.label"
                  :value="p.value"
                />
              </el-select>
            </el-form-item>
          </el-form>
          <el-button type="primary" :loading="saving" @click="savePhp">{{ t('siteModify.save') }}</el-button>
        </div>

        <!-- 重定向 -->
        <div v-show="activeMenu === 'redirect'" class="panel-section">
          <p class="panel-hint">{{ t('siteModify.redirectHint') }}</p>
          <el-input v-model="settings.redirect_url" placeholder="https://example.com/" />
          <div class="panel-actions">
            <el-button type="primary" :loading="saving" @click="saveRedirect">{{ t('siteModify.save') }}</el-button>
          </div>
        </div>

        <!-- 反向代理 -->
        <div v-show="activeMenu === 'proxy'" class="panel-section">
          <p class="panel-hint">{{ t('siteModify.proxyHint') }}</p>
          <el-input v-model="settings.proxy_pass" placeholder="http://127.0.0.1:3000" />
          <div class="panel-actions">
            <el-button type="primary" :loading="saving" @click="saveProxy">{{ t('siteModify.save') }}</el-button>
          </div>
        </div>

        <!-- CDN 缓存 -->
        <div v-show="activeMenu === 'cache'" class="panel-section">
          <p class="panel-hint">{{ t('siteModify.cacheHint') }}</p>
          <el-form label-width="200px">
            <el-form-item :label="t('siteModify.cacheEnable')">
              <el-switch v-model="settings.cache_enabled" />
            </el-form-item>
            <el-form-item :label="t('siteModify.cacheDevMode')">
              <el-switch v-model="settings.cache_dev_mode" :disabled="!settings.cache_enabled" />
              <span class="panel-hint-inline">{{ t('siteModify.cacheDevModeHint') }}</span>
            </el-form-item>
            <el-form-item :label="t('siteModify.cacheHtmlTTL')">
              <el-input-number v-model="settings.cache_html_ttl" :min="0" :max="1440" />
            </el-form-item>
            <el-form-item :label="t('siteModify.cacheStaticTTL')">
              <el-input-number v-model="settings.cache_static_ttl" :min="0" :max="8760" />
            </el-form-item>
          </el-form>
          <el-button type="primary" :loading="saving" @click="saveCache">{{ t('siteModify.save') }}</el-button>
        </div>

        <!-- 响应日志 -->
        <div v-show="activeMenu === 'logs'" class="panel-section logs-panel">
          <div class="logs-layout">
            <div class="logs-main">
              <div class="log-block">
                <div class="log-head">
                  <span>{{ t('siteModify.accessLog') }}</span>
                  <code>{{ logs.access_log }}</code>
                </div>
                <LogViewer
                  :content="logs.access_tail"
                  kind="access"
                  :empty-text="t('siteModify.noLog')"
                  max-height="180px"
                />
              </div>
              <div class="log-block">
                <div class="log-head">
                  <span>{{ t('siteModify.errorLog') }}</span>
                  <code>{{ logs.error_log }}</code>
                </div>
                <LogViewer
                  :content="logs.error_tail"
                  kind="error"
                  :empty-text="t('siteModify.noLog')"
                  max-height="180px"
                />
              </div>
              <el-button @click="loadLogs">{{ t('siteModify.reload') }}</el-button>
            </div>
            <SiteLogAIPanel
              v-if="siteId"
              :site-id="siteId"
              :domain="site?.domain || ''"
              :access-log="logs.access_tail || ''"
              :error-log="logs.error_tail || ''"
              :access-path="logs.access_log || ''"
              :error-path="logs.error_log || ''"
              @repaired="loadLogs(); emit('updated')"
            />
          </div>
        </div>

        <!-- 子目录绑定 -->
        <div v-show="activeMenu === 'subdir'" class="panel-section">
          <p class="panel-hint">{{ t('siteModify.subdirHint') }}</p>
          <el-button type="primary" @click="openSubdirDialog()">{{ t('siteModify.subdirAdd') }}</el-button>
          <el-table :data="subdirs" stripe size="small">
            <el-table-column prop="prefix" :label="t('siteModify.subdirPrefix')" width="120" />
            <el-table-column prop="root_path" :label="t('websites.rootPath')" min-width="200" />
            <el-table-column prop="remark" :label="t('common.description')" min-width="120" />
            <el-table-column :label="t('common.actions')" width="140">
              <template #default="{ row }">
                <el-button text type="primary" @click="openSubdirDialog(row)">{{ t('common.edit') }}</el-button>
                <el-button text type="danger" @click="deleteSubdir(row)">{{ t('common.delete') }}</el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>

        <!-- 限制访问 -->
        <div v-show="activeMenu === 'access'" class="panel-section">
          <p class="panel-hint">{{ t('siteModify.accessHint') }}</p>
          <el-form label-width="140px">
            <el-form-item :label="t('siteModify.accessAuth')">
              <el-switch v-model="settings.access_auth_enabled" />
            </el-form-item>
            <template v-if="settings.access_auth_enabled">
              <el-form-item :label="t('common.username')">
                <el-input v-model="settings.access_auth_user" />
              </el-form-item>
              <el-form-item :label="t('common.password')">
                <el-input v-model="settings.access_auth_pass" type="password" show-password :placeholder="t('siteModify.passwordKeep')" />
              </el-form-item>
            </template>
            <el-form-item :label="t('siteModify.accessAllow')">
              <el-input v-model="settings.access_allow_ips" type="textarea" :rows="4" :placeholder="t('siteModify.ipPlaceholder')" />
            </el-form-item>
            <el-form-item :label="t('siteModify.accessDeny')">
              <el-input v-model="settings.access_deny_ips" type="textarea" :rows="3" :placeholder="t('siteModify.ipPlaceholder')" />
            </el-form-item>
          </el-form>
          <el-button type="primary" :loading="saving" @click="saveAccess">{{ t('siteModify.save') }}</el-button>
        </div>

        <!-- 流量控制 -->
        <div v-show="activeMenu === 'traffic'" class="panel-section">
          <p class="panel-hint">{{ t('siteModify.trafficHint') }}</p>
          <el-alert type="info" :closable="false" class="hint-inline">{{ t('siteModify.trafficLimitsHint') }}</el-alert>
          <el-form label-width="140px">
            <el-form-item :label="t('siteModify.trafficEnable')">
              <el-switch v-model="settings.traffic_limit_enabled" />
            </el-form-item>
            <el-form-item :label="t('siteModify.trafficRate')">
              <el-input v-model="settings.traffic_rate" placeholder="10r/s" style="width: 160px" />
            </el-form-item>
            <el-form-item :label="t('siteModify.trafficBurst')">
              <el-input-number v-model="settings.traffic_burst" :min="1" :max="1000" />
            </el-form-item>
          </el-form>
          <el-button type="primary" :loading="saving" @click="saveTraffic">{{ t('siteModify.save') }}</el-button>
        </div>

        <!-- 防盗链 -->
        <div v-show="activeMenu === 'hotlink'" class="panel-section">
          <p class="panel-hint">{{ t('siteModify.hotlinkHint') }}</p>
          <el-form label-width="160px">
            <el-form-item :label="t('siteModify.hotlinkEnable')">
              <el-switch v-model="settings.hotlink_enabled" />
            </el-form-item>
            <el-form-item :label="t('siteModify.hotlinkAllowEmpty')">
              <el-switch v-model="settings.hotlink_allow_empty" />
            </el-form-item>
            <el-form-item :label="t('siteModify.hotlinkAllowDomains')">
              <el-input v-model="settings.hotlink_allow_domains" type="textarea" :rows="3" :placeholder="t('siteModify.hotlinkDomainsPlaceholder')" />
            </el-form-item>
          </el-form>
          <el-button type="primary" :loading="saving" @click="saveHotlink">{{ t('siteModify.save') }}</el-button>
        </div>

        <!-- Composer -->
        <div v-show="activeMenu === 'composer'" class="panel-section">
          <p class="panel-hint">{{ t('siteModify.composerHint') }}</p>
          <el-input v-model="composerCmd" placeholder="install / update / require vendor/package" />
          <div class="panel-actions">
            <el-button type="primary" :loading="composerRunning" @click="runComposer">{{ t('siteModify.composerRun') }}</el-button>
          </div>
          <pre v-if="composerOutput" class="log-pre">{{ composerOutput }}</pre>
        </div>
      </div>
    </div>

    <div class="site-modify-ai-bar">
      <el-button type="primary" plain :icon="MagicStick" @click="openSiteAiDrawer">
        {{ t('siteModify.siteAiAssist') }}
      </el-button>
      <span class="site-modify-ai-hint">{{ t('siteModify.siteAiAssistHint') }}</span>
    </div>
  </el-dialog>

  <el-drawer v-model="siteAiDrawer" :title="t('siteModify.siteAiAssist')" size="520px" destroy-on-close class="site-ai-drawer">
    <el-tabs v-if="siteId" v-model="siteAiTab" class="site-ai-tabs">
      <el-tab-pane :label="t('siteModify.projectAiTab')" name="project">
        <SiteProjectAIPanel
          :site-id="siteId"
          :domain="site?.domain || ''"
          height="calc(100vh - 160px)"
        />
      </el-tab-pane>
      <el-tab-pane :label="t('siteModify.logAiTab')" name="logs">
        <AIChatPanel
          v-model="siteAiMessages"
          :welcome="t('siteModify.siteAiDrawerHint', { tab: activeMenuLabel })"
          :placeholder="t('siteModify.siteAiPlaceholder')"
          :context-label="site?.domain || ''"
          :stream-url="`/websites/${siteId}/logs/ai/chat/stream`"
          :stream-body="siteAiStreamBody"
          :send-fallback="siteAiSendFallback"
          height="calc(100vh - 160px)"
        />
      </el-tab-pane>
    </el-tabs>
  </el-drawer>

  <el-dialog v-model="subdirDialog" :title="editingSubdir ? t('siteModify.subdirEdit') : t('siteModify.subdirAdd')" width="480px" append-to-body>
    <el-form label-width="100px">
      <el-form-item :label="t('siteModify.subdirPrefix')">
        <el-input v-model="subdirForm.prefix" placeholder="/blog" />
      </el-form-item>
      <el-form-item :label="t('websites.rootPath')">
        <el-input v-model="subdirForm.root_path" />
      </el-form-item>
      <el-form-item :label="t('common.description')">
        <el-input v-model="subdirForm.remark" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="subdirDialog = false">{{ t('common.cancel') }}</el-button>
      <el-button type="primary" :loading="saving" @click="saveSubdir">{{ t('siteModify.save') }}</el-button>
    </template>
  </el-dialog>

  <el-dialog v-model="pathPickerVisible" :title="t('websites.selectPath')" width="520px" append-to-body>
    <div class="path-bar">{{ browsePath }}</div>
    <el-table
      :data="pathEntries"
      stripe
      max-height="320"
      @row-dblclick="(row: any) => row.is_dir && loadPathDir(row.path)"
    >
      <el-table-column prop="name" label="Name" />
      <el-table-column width="100">
        <template #default="{ row }">
          <el-button v-if="row.is_dir" text type="primary" @click="loadPathDir(row.path)">Open</el-button>
          <el-button v-else text type="success" @click="selectPath(row.path)">Select</el-button>
        </template>
      </el-table-column>
    </el-table>
    <template #footer>
      <el-button type="primary" @click="selectPath(browsePath)">{{ t('siteModify.save') }}</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.site-modify-body {
  display: flex;
  min-height: 480px;
  gap: 0;
}
.site-modify-menu {
  width: 168px;
  flex-shrink: 0;
  border-right: 1px solid var(--el-border-color-light);
  max-height: 56vh;
  overflow-y: auto;
}
.site-modify-menu :deep(.el-menu-item) {
  height: 40px;
  line-height: 40px;
  font-size: 13px;
  padding-left: 16px !important;
}
.site-modify-panel {
  flex: 1;
  padding: 0 20px 8px;
  max-height: 56vh;
  overflow-y: auto;
}
.panel-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.panel-actions {
  margin-top: 4px;
}
.panel-hint {
  margin: 0;
  font-size: 13px;
  color: var(--el-text-color-secondary);
  line-height: 1.6;
}
.domain-table {
  margin-top: 8px;
}
.domain-visit-link {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: var(--el-color-primary);
  text-decoration: none;
  font-weight: 500;
}
.domain-visit-link:hover {
  text-decoration: underline;
}
.domain-visit-icon {
  font-size: 12px;
  opacity: 0.75;
}
.mono-input :deep(textarea) {
  font-family: Consolas, Monaco, monospace;
  font-size: 13px;
}
.muted {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
.panel-hint-inline {
  margin-left: 10px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.log-block {
  margin-bottom: 12px;
}
.logs-panel {
  padding-right: 0;
}
.logs-layout {
  display: flex;
  gap: 16px;
  align-items: flex-start;
}
.logs-main {
  flex: 1;
  min-width: 0;
}
.menu-item-label {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.menu-ai-icon {
  font-size: 14px;
  color: #7c3aed;
}
.site-modify-ai-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.site-modify-ai-hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.site-ai-drawer :deep(.el-drawer__body) {
  padding: 12px;
}
.site-ai-tabs :deep(.el-tabs__content) {
  padding-top: 8px;
}
.log-head {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-bottom: 6px;
  font-size: 13px;
}
.log-head code {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  word-break: break-all;
}
.log-pre {
  margin: 0;
  padding: 10px 12px;
  background: var(--el-fill-color-light);
  border-radius: 6px;
  font-size: 12px;
  line-height: 1.5;
  max-height: 160px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-all;
}
.rewrite-toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  margin-bottom: 4px;
}
.hint-inline {
  margin-bottom: 8px;
}
.path-bar {
  margin-bottom: 12px;
  padding: 8px 12px;
  background: var(--el-fill-color-light);
  border-radius: 6px;
  font-family: monospace;
  font-size: 13px;
  word-break: break-all;
}
</style>
