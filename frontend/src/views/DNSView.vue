<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import api from '@/api'
import { ElMessage, ElMessageBox } from 'element-plus'

const { t } = useI18n()

const tab = ref('guide')
const guideProvider = ref('cloudflare')

const guideProviders = [
  { key: 'cloudflare', doc: 'https://developers.cloudflare.com/fundamentals/api/get-started/create-token/' },
  { key: 'alidns', doc: 'https://help.aliyun.com/document_detail/29739.html' },
  { key: 'dnspod', doc: 'https://docs.dnpod.cn/api/create-token/' },
]

function guideSteps(key: string) {
  const count = key === 'dnspod' ? 6 : 6
  return Array.from({ length: count }, (_, i) => t(`dnsPage.${key}Step${i + 1}`))
}

function guideTitle(key: string) {
  return t(`dnsPage.${key}GuideTitle`)
}

function guidePermissions(key: string) {
  return t(`dnsPage.${key}Permissions`)
}

function openProviderDialog(provider = 'cloudflare') {
  providerForm.value = {
    name: '', provider, api_token: '', access_key: '', secret_key: '', is_default: true,
  }
  guideProvider.value = provider
  providerDialog.value = true
}

function showFullGuide(provider: string) {
  guideProvider.value = provider
  providerDialog.value = false
  tab.value = 'guide'
}
const loading = ref(false)
const records = ref<any[]>([])
const providers = ref<any[]>([])
const supported = ref<any[]>([])
const zones = ref<any[]>([])
const detectItems = ref<any[]>([])
const serverIP = ref('')
const selectedDetect = ref<any[]>([])
const detectTableRef = ref<any>(null)

const providerDialog = ref(false)
const providerForm = ref({
  name: '', provider: 'cloudflare', api_token: '', access_key: '', secret_key: '', is_default: true,
})

const recordDialog = ref(false)
const editingRecord = ref<any>(null)
const recordForm = ref({
  domain: '', type: 'A', name: '@', value: '', ttl: 600, proxied: false, provider_id: 0,
})

const zoneFilter = ref('')
const filteredRecords = computed(() => {
  if (!zoneFilter.value) return records.value
  return records.value.filter(r => r.domain === zoneFilter.value)
})

async function loadAll() {
  loading.value = true
  try {
    const [rec, prov, sup, ip]: any[] = await Promise.all([
      api.get('/dns'),
      api.get('/dns/providers'),
      api.get('/dns/providers/supported'),
      api.get('/dns/server-ip'),
    ])
    records.value = rec.data || []
    providers.value = prov.data || []
    supported.value = sup.data || []
    serverIP.value = ip.data?.ip || ''
    if (providers.value.length) {
      const z: any = await api.get('/dns/zones')
      zones.value = z.data || []
    }
  } finally {
    loading.value = false
  }
}

function providerLabel(key: string) {
  return supported.value.find(p => p.key === key)?.name || key
}

function syncStatusTag(status: string) {
  if (status === 'synced') return 'success'
  if (status === 'error') return 'danger'
  if (status === 'local') return 'info'
  return 'warning'
}

function syncStatusLabel(status: string) {
  const map: Record<string, string> = {
    synced: t('dnsPage.synced'),
    local: t('dnsPage.local'),
    pending: t('dnsPage.pending'),
    error: t('dnsPage.error'),
  }
  return map[status] || status
}

async function addProvider() {
  await api.post('/dns/providers', providerForm.value)
  ElMessage.success(t('dnsPage.providerAdded'))
  providerDialog.value = false
  loadAll()
}

async function testProvider(id: number) {
  try {
    await api.post(`/dns/providers/${id}/test`)
    ElMessage.success(t('dnsPage.connectionOk'))
  } catch (e: any) {
    ElMessage.error(e?.message || String(e))
  }
}

async function syncZones(id: number) {
  const res: any = await api.post(`/dns/providers/${id}/sync-zones`)
  ElMessage.success(t('dnsPage.zonesSynced', { n: res.data?.synced ?? 0 }))
  loadAll()
}

async function deleteProvider(id: number) {
  await ElMessageBox.confirm(t('dnsPage.deleteProviderConfirm'), t('common.confirm'), { type: 'warning' })
  await api.delete(`/dns/providers/${id}`)
  ElMessage.success(t('dnsPage.providerDeleted'))
  loadAll()
}

async function pullZone(zone: any) {
  const res: any = await api.post('/dns/zones/pull', { provider_id: zone.provider_id, zone: zone.name })
  ElMessage.success(t('dnsPage.recordsImported', { n: res.data?.imported ?? 0 }))
  loadAll()
}

function openAddRecord() {
  editingRecord.value = null
  recordForm.value = {
    domain: zoneFilter.value || zones.value[0]?.name || '',
    type: 'A', name: '@', value: serverIP.value, ttl: 600, proxied: false,
    provider_id: providers.value.find(p => p.is_default)?.id || providers.value[0]?.id || 0,
  }
  recordDialog.value = true
}

function openEditRecord(row: any) {
  editingRecord.value = row
  recordForm.value = { ...row, provider_id: row.provider_id || 0 }
  recordDialog.value = true
}

async function saveRecord() {
  if (editingRecord.value) {
    await api.put(`/dns/${editingRecord.value.id}`, recordForm.value)
  } else {
    await api.post('/dns', recordForm.value)
  }
  ElMessage.success(t('dnsPage.recordSaved'))
  recordDialog.value = false
  loadAll()
}

async function deleteRecord(id: number) {
  await ElMessageBox.confirm(t('dnsPage.deleteConfirm'), t('common.confirm'), { type: 'warning' })
  await api.delete(`/dns/${id}`)
  ElMessage.success(t('dnsPage.recordDeleted'))
  loadAll()
}

async function runDetect() {
  loading.value = true
  try {
    const res: any = await api.get('/dns/detect')
    detectItems.value = res.data || []
    selectedDetect.value = detectItems.value.filter((d: any) => d.needs_fix)
    await nextTick()
    detectTableRef.value?.clearSelection()
    for (const row of selectedDetect.value) {
      detectTableRef.value?.toggleRowSelection(row, true)
    }
    ElMessage.success(t('dnsPage.detectDone'))
  } finally {
    loading.value = false
  }
}

async function applyFix(hosts?: string[]) {
  const list = hosts || selectedDetect.value.map(d => d.host)
  if (!list.length) return
  await api.post('/dns/apply', { hosts: list, ip: serverIP.value, proxied: false })
  ElMessage.success(t('dnsPage.applyDone'))
  runDetect()
  loadAll()
}

onMounted(loadAll)
</script>

<template>
  <div class="dns-page" v-loading="loading">
    <div class="page-header">
      <div>
        <h2>{{ t('dnsPage.title') }}</h2>
        <p class="hint">{{ t('dnsPage.supportedProviders') }}</p>
      </div>
      <div class="header-actions">
        <el-tag type="info">{{ t('dnsPage.serverIP') }}: {{ serverIP || '—' }}</el-tag>
        <el-button text type="primary" @click="tab = 'guide'">{{ t('dnsPage.guide') }}</el-button>
      </div>
    </div>

    <el-tabs v-model="tab">
      <el-tab-pane :label="t('dnsPage.guide')" name="guide">
        <el-alert type="info" :closable="false" show-icon class="guide-intro">
          {{ t('dnsPage.guideIntro') }}
        </el-alert>

        <el-card shadow="never" class="guide-card">
          <template #header>{{ t('dnsPage.guideWorkflowTitle') }}</template>
          <ol class="guide-list">
            <li>{{ t('dnsPage.guideWorkflow1') }}</li>
            <li>{{ t('dnsPage.guideWorkflow2') }}</li>
            <li>{{ t('dnsPage.guideWorkflow3') }}</li>
            <li>{{ t('dnsPage.guideWorkflow4') }}</li>
          </ol>
        </el-card>

        <el-collapse v-model="guideProvider" accordion class="provider-guides">
          <el-collapse-item v-for="p in guideProviders" :key="p.key" :name="p.key">
            <template #title>
              <span class="guide-collapse-title">{{ guideTitle(p.key) }}</span>
            </template>
            <ol class="guide-list">
              <li v-for="(step, idx) in guideSteps(p.key)" :key="idx">{{ step }}</li>
            </ol>
            <div class="guide-meta">
              <el-tag type="warning" size="small">{{ t('dnsPage.guidePermissions') }}: {{ guidePermissions(p.key) }}</el-tag>
              <el-link type="primary" :href="p.doc" target="_blank" rel="noopener">
                {{ t('dnsPage.guideDocLink') }} ↗
              </el-link>
              <el-button size="small" type="primary" @click="openProviderDialog(p.key)">
                {{ t('dnsPage.addProvider') }}
              </el-button>
            </div>
          </el-collapse-item>
        </el-collapse>
      </el-tab-pane>

      <el-tab-pane :label="t('dnsPage.records')" name="records">
        <div class="toolbar">
          <el-select v-model="zoneFilter" clearable :placeholder="t('dnsPage.zoneName')" style="width: 200px">
            <el-option v-for="z in zones" :key="z.id" :label="z.name" :value="z.name" />
          </el-select>
          <el-button type="primary" @click="openAddRecord">{{ t('dnsPage.addRecord') }}</el-button>
        </div>
        <el-table :data="filteredRecords" stripe>
          <el-table-column prop="domain" :label="t('dnsPage.zoneName')" width="160" />
          <el-table-column prop="name" :label="t('dnsPage.host')" width="100" />
          <el-table-column prop="type" :label="t('dnsPage.recordType')" width="80" />
          <el-table-column prop="value" :label="t('dnsPage.recordValue')" show-overflow-tooltip />
          <el-table-column prop="ttl" :label="t('dnsPage.ttl')" width="80" />
          <el-table-column :label="t('dnsPage.syncStatus')" width="100">
            <template #default="{ row }">
              <el-tag :type="syncStatusTag(row.sync_status)" size="small">
                {{ syncStatusLabel(row.sync_status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column :label="t('common.actions')" width="140">
            <template #default="{ row }">
              <el-button text type="primary" @click="openEditRecord(row)">{{ t('common.edit') }}</el-button>
              <el-button text type="danger" @click="deleteRecord(row.id)">{{ t('common.delete') }}</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane :label="t('dnsPage.providers')" name="providers">
        <div class="toolbar">
          <el-button type="primary" @click="openProviderDialog()">{{ t('dnsPage.addProvider') }}</el-button>
          <el-button @click="tab = 'guide'">{{ t('dnsPage.guide') }}</el-button>
        </div>
        <el-empty v-if="!providers.length" :description="t('dnsPage.noProvider')">
          <el-button type="primary" @click="tab = 'guide'">{{ t('dnsPage.guide') }}</el-button>
        </el-empty>
        <el-table v-else :data="providers" stripe>
          <el-table-column prop="name" :label="t('dnsPage.providerName')" />
          <el-table-column :label="t('dnsPage.providerType')" width="140">
            <template #default="{ row }">{{ row.provider_name || providerLabel(row.provider) }}</template>
          </el-table-column>
          <el-table-column :label="t('dnsPage.isDefault')" width="80">
            <template #default="{ row }">
              <el-tag v-if="row.is_default" type="success" size="small">{{ t('dnsPage.isDefault') }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column :label="t('common.actions')" width="280">
            <template #default="{ row }">
              <el-button text @click="testProvider(row.id)">{{ t('dnsPage.testConnection') }}</el-button>
              <el-button text type="primary" @click="syncZones(row.id)">{{ t('dnsPage.syncZones') }}</el-button>
              <el-button text type="danger" @click="deleteProvider(row.id)">{{ t('common.delete') }}</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane :label="t('dnsPage.zones')" name="zones">
        <el-table :data="zones" stripe>
          <el-table-column prop="name" :label="t('dnsPage.zoneName')" />
          <el-table-column prop="status" :label="t('dnsPage.zoneStatus')" width="100" />
          <el-table-column :label="t('common.actions')" width="160">
            <template #default="{ row }">
              <el-button text type="primary" @click="pullZone(row)">{{ t('dnsPage.pullRecords') }}</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane :label="t('dnsPage.detect')" name="detect">
        <div class="toolbar">
          <el-button type="primary" @click="runDetect">{{ t('dnsPage.detectNow') }}</el-button>
          <el-button type="success" :disabled="!selectedDetect.length" @click="applyFix()">
            {{ t('dnsPage.applySelected') }}
          </el-button>
          <el-button :disabled="!detectItems.some(d => d.needs_fix)" @click="applyFix(detectItems.filter(d => d.needs_fix).map(d => d.host))">
            {{ t('dnsPage.applyFix') }}
          </el-button>
        </div>
        <el-table ref="detectTableRef" :data="detectItems" stripe @selection-change="(v: any[]) => selectedDetect = v">
          <el-table-column type="selection" width="48" />
          <el-table-column prop="host" label="Host" min-width="180" />
          <el-table-column prop="source" :label="t('dnsPage.source')" width="100" />
          <el-table-column :label="t('dnsPage.zoneFound')" width="100">
            <template #default="{ row }">
              <el-tag :type="row.zone_found ? 'success' : 'warning'" size="small">
                {{ row.zone_found ? row.zone_name : '—' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="current_value" :label="t('dnsPage.currentValue')" width="140" />
          <el-table-column prop="expected_ip" :label="t('dnsPage.expectedIP')" width="140" />
          <el-table-column :label="t('dnsPage.needsFix')" width="90">
            <template #default="{ row }">
              <el-tag :type="row.needs_fix ? 'danger' : 'success'" size="small">
                {{ row.needs_fix ? '!' : 'OK' }}
              </el-tag>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <el-dialog v-model="providerDialog" :title="t('dnsPage.addProvider')" width="560px">
      <el-form label-width="100px">
        <el-form-item :label="t('dnsPage.providerName')">
          <el-input v-model="providerForm.name" placeholder="My Cloudflare" />
        </el-form-item>
        <el-form-item :label="t('dnsPage.providerType')">
          <el-select v-model="providerForm.provider" style="width: 100%" @change="guideProvider = providerForm.provider">
            <el-option v-for="p in supported" :key="p.key" :label="p.name" :value="p.key" />
          </el-select>
        </el-form-item>

        <div class="dialog-guide">
          <div class="dialog-guide-head">
            <span>{{ t('dnsPage.guideViewInDialog') }}</span>
            <el-button text type="primary" size="small" @click="showFullGuide(providerForm.provider)">
              {{ t('dnsPage.guide') }} →
            </el-button>
          </div>
          <ol class="guide-list compact">
            <li v-for="(step, idx) in guideSteps(providerForm.provider)" :key="idx">{{ step }}</li>
          </ol>
        </div>

        <el-form-item v-if="providerForm.provider === 'cloudflare'" :label="t('dnsPage.apiToken')">
          <el-input v-model="providerForm.api_token" type="password" show-password />
        </el-form-item>
        <template v-if="providerForm.provider === 'alidns'">
          <el-form-item :label="t('dnsPage.accessKey')">
            <el-input v-model="providerForm.access_key" />
          </el-form-item>
          <el-form-item :label="t('dnsPage.secretKey')">
            <el-input v-model="providerForm.secret_key" type="password" show-password />
          </el-form-item>
        </template>
        <template v-if="providerForm.provider === 'dnspod'">
          <el-form-item :label="t('dnsPage.apiToken')">
            <el-input v-model="providerForm.api_token" :placeholder="t('dnsPage.tokenHint')" />
          </el-form-item>
        </template>
        <el-form-item :label="t('dnsPage.isDefault')">
          <el-switch v-model="providerForm.is_default" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="providerDialog = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="addProvider">{{ t('common.confirm') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="recordDialog" :title="editingRecord ? t('dnsPage.editRecord') : t('dnsPage.addRecord')" width="520px">
      <el-form label-width="100px">
        <el-form-item :label="t('dnsPage.zoneName')">
          <el-select v-model="recordForm.domain" filterable allow-create style="width: 100%">
            <el-option v-for="z in zones" :key="z.id" :label="z.name" :value="z.name" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('dnsPage.recordType')">
          <el-select v-model="recordForm.type" style="width: 100%">
            <el-option label="A" value="A" />
            <el-option label="AAAA" value="AAAA" />
            <el-option label="CNAME" value="CNAME" />
            <el-option label="MX" value="MX" />
            <el-option label="TXT" value="TXT" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('dnsPage.host')">
          <el-input v-model="recordForm.name" placeholder="@" />
        </el-form-item>
        <el-form-item :label="t('dnsPage.recordValue')">
          <el-input v-model="recordForm.value" />
        </el-form-item>
        <el-form-item :label="t('dnsPage.ttl')">
          <el-input-number v-model="recordForm.ttl" :min="60" :max="86400" />
        </el-form-item>
        <el-form-item v-if="recordForm.type === 'A'" :label="t('dnsPage.proxied')">
          <el-switch v-model="recordForm.proxied" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="recordDialog = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="saveRecord">{{ t('common.confirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.dns-page { width: 100%; }
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 16px;
}
.header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}
.hint { margin: 0; font-size: 13px; color: var(--el-text-color-secondary); }
.toolbar { display: flex; gap: 12px; margin-bottom: 12px; align-items: center; }
.guide-intro { margin-bottom: 16px; }
.guide-card { margin-bottom: 16px; }
.guide-list {
  margin: 0;
  padding-left: 20px;
  line-height: 1.8;
  font-size: 14px;
  color: var(--el-text-color-regular);
}
.guide-list.compact { font-size: 13px; line-height: 1.6; }
.guide-list li { margin-bottom: 6px; }
.provider-guides { margin-top: 8px; }
.guide-collapse-title { font-weight: 600; }
.guide-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.dialog-guide {
  margin: 0 0 16px 100px;
  padding: 12px 14px;
  background: var(--el-fill-color-light);
  border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter);
}
.dialog-guide-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
@media (max-width: 640px) {
  .dialog-guide { margin-left: 0; }
}
</style>
