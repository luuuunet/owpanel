<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import api, { resolveApiError } from '@/api'
import { ElMessage, ElMessageBox } from 'element-plus'

const { t } = useI18n()

interface ProviderMeta {
  type: string
  name: string
  name_en: string
  description: string
  fields: string[]
}

interface Provider {
  id: number
  name: string
  provider_type: string
  enabled: boolean
  is_default: boolean
  default_from: string
  default_from_name: string
  config?: Record<string, string>
}

interface Campaign {
  id: number
  name: string
  provider_id: number
  subject: string
  status: string
  total_recipients: number
  sent_count: number
  failed_count: number
  rate_per_minute: number
  last_error?: string
  created_at: string
}

const subTab = ref('providers')
const catalog = ref<ProviderMeta[]>([])
const providers = ref<Provider[]>([])
const campaigns = ref<Campaign[]>([])
const loading = ref(false)
const actionLoading = ref('')

const providerDialog = ref(false)
const editingProvider = ref<Provider | null>(null)
const providerForm = ref({
  name: '',
  provider_type: 'local',
  default_from: '',
  default_from_name: '',
  enabled: true,
  is_default: false,
  config: {} as Record<string, string>,
})

const testDialog = ref(false)
const testProviderId = ref(0)
const testTo = ref('')

const campaignDialog = ref(false)
const campaignForm = ref({
  name: '',
  provider_id: 0,
  from_email: '',
  from_name: '',
  reply_to: '',
  subject: '',
  body_html: '',
  body_text: '',
  recipients: '',
  rate_per_minute: 60,
})

const selectedMeta = computed(() =>
  catalog.value.find(c => c.type === providerForm.value.provider_type),
)

function providerLabel(type: string) {
  const c = catalog.value.find(x => x.type === type)
  return c?.name || type
}

async function loadAll() {
  loading.value = true
  try {
    const [cat, prov, camps]: any[] = await Promise.all([
      api.get('/mail/bulk/providers/catalog'),
      api.get('/mail/bulk/providers'),
      api.get('/mail/bulk/campaigns'),
    ])
    catalog.value = cat.data || []
    providers.value = prov.data || []
    campaigns.value = camps.data || []
    if (!campaignForm.value.provider_id && providers.value.length) {
      campaignForm.value.provider_id = providers.value[0].id
    }
  } finally {
    loading.value = false
  }
}

function openProviderDialog(row?: Provider) {
  editingProvider.value = row || null
  if (row) {
    providerForm.value = {
      name: row.name,
      provider_type: row.provider_type,
      default_from: row.default_from,
      default_from_name: row.default_from_name,
      enabled: row.enabled,
      is_default: row.is_default,
      config: { ...(row.config || {}) },
    }
  } else {
    providerForm.value = {
      name: '',
      provider_type: 'local',
      default_from: '',
      default_from_name: '',
      enabled: true,
      is_default: false,
      config: {},
    }
  }
  providerDialog.value = true
}

async function saveProvider() {
  actionLoading.value = 'provider'
  try {
    const payload = { ...providerForm.value }
    if (editingProvider.value) {
      await api.put(`/mail/bulk/providers/${editingProvider.value.id}`, payload)
    } else {
      await api.post('/mail/bulk/providers', payload)
    }
    ElMessage.success(t('common.saved'))
    providerDialog.value = false
    await loadAll()
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
  } finally {
    actionLoading.value = ''
  }
}

async function deleteProvider(row: Provider) {
  await ElMessageBox.confirm(t('mail.bulkDeleteProviderConfirm'), t('common.warning'), { type: 'warning' })
  await api.delete(`/mail/bulk/providers/${row.id}`)
  ElMessage.success(t('common.deleted'))
  await loadAll()
}

function openTest(row: Provider) {
  testProviderId.value = row.id
  testTo.value = ''
  testDialog.value = true
}

async function runTest() {
  actionLoading.value = 'test'
  try {
    await api.post(`/mail/bulk/providers/${testProviderId.value}/test`, { to: testTo.value })
    ElMessage.success(t('mail.bulkTestSent'))
    testDialog.value = false
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
  } finally {
    actionLoading.value = ''
  }
}

function openCampaignDialog() {
  const p = providers.value.find(x => x.id === campaignForm.value.provider_id) || providers.value[0]
  campaignForm.value = {
    name: '',
    provider_id: p?.id || 0,
    from_email: p?.default_from || '',
    from_name: p?.default_from_name || '',
    reply_to: '',
    subject: '',
    body_html: '',
    body_text: '',
    recipients: '',
    rate_per_minute: 60,
  }
  campaignDialog.value = true
}

async function createCampaign() {
  actionLoading.value = 'campaign'
  try {
    const res: any = await api.post('/mail/bulk/campaigns', campaignForm.value)
    ElMessage.success(t('mail.bulkCampaignCreated'))
    campaignDialog.value = false
    await loadAll()
    await startCampaign(res.data.id)
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
  } finally {
    actionLoading.value = ''
  }
}

async function startCampaign(id: number) {
  try {
    await api.post(`/mail/bulk/campaigns/${id}/start`)
    ElMessage.success(t('mail.bulkCampaignStarted'))
    await loadAll()
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
  }
}

async function cancelCampaign(row: Campaign) {
  await api.post(`/mail/bulk/campaigns/${row.id}/cancel`)
  ElMessage.success(t('mail.bulkCampaignCancelled'))
  await loadAll()
}

async function deleteCampaign(row: Campaign) {
  await ElMessageBox.confirm(t('mail.bulkDeleteCampaignConfirm'), t('common.warning'), { type: 'warning' })
  await api.delete(`/mail/bulk/campaigns/${row.id}`)
  ElMessage.success(t('common.deleted'))
  await loadAll()
}

function statusTagType(s: string) {
  if (s === 'completed') return 'success'
  if (s === 'sending') return 'warning'
  if (s === 'failed') return 'danger'
  if (s === 'cancelled') return 'info'
  return ''
}

onMounted(loadAll)
</script>

<template>
  <div v-loading="loading">
    <el-alert type="info" show-icon :closable="false" class="hint">{{ t('mail.bulkHint') }}</el-alert>

    <el-tabs v-model="subTab" class="bulk-subtabs">
      <el-tab-pane :label="t('mail.bulkProviders')" name="providers">
        <div class="tab-toolbar">
          <el-button type="primary" @click="openProviderDialog()">{{ t('mail.bulkAddProvider') }}</el-button>
        </div>
        <el-table :data="providers" stripe>
          <el-table-column prop="name" :label="t('common.name')" />
          <el-table-column :label="t('mail.bulkProviderType')" width="180">
            <template #default="{ row }">{{ providerLabel(row.provider_type) }}</template>
          </el-table-column>
          <el-table-column prop="default_from" :label="t('mail.bulkDefaultFrom')" min-width="180" />
          <el-table-column :label="t('common.status')" width="90">
            <template #default="{ row }">
              <el-tag :type="row.enabled ? 'success' : 'info'" size="small">{{ row.enabled ? t('common.enabled') : t('common.disabled') }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column :label="t('common.actions')" width="220">
            <template #default="{ row }">
              <el-button text type="primary" @click="openProviderDialog(row)">{{ t('common.edit') }}</el-button>
              <el-button text @click="openTest(row)">{{ t('mail.bulkTest') }}</el-button>
              <el-button text type="danger" @click="deleteProvider(row)">{{ t('common.delete') }}</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane :label="t('mail.bulkCampaigns')" name="campaigns">
        <div class="tab-toolbar">
          <el-button type="primary" :disabled="!providers.length" @click="openCampaignDialog()">{{ t('mail.bulkNewCampaign') }}</el-button>
          <el-button @click="loadAll">{{ t('common.refresh') }}</el-button>
        </div>
        <el-table :data="campaigns" stripe>
          <el-table-column prop="name" :label="t('common.name')" min-width="140" />
          <el-table-column prop="subject" :label="t('mail.bulkSubject')" min-width="160" />
          <el-table-column :label="t('common.status')" width="110">
            <template #default="{ row }">
              <el-tag :type="statusTagType(row.status)" size="small">{{ row.status }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column :label="t('mail.bulkProgress')" width="140">
            <template #default="{ row }">{{ row.sent_count }}/{{ row.total_recipients }} ({{ t('mail.bulkFailed') }} {{ row.failed_count }})</template>
          </el-table-column>
          <el-table-column prop="created_at" :label="t('common.createdAt')" width="170" />
          <el-table-column :label="t('common.actions')" width="200">
            <template #default="{ row }">
              <el-button v-if="row.status === 'draft'" text type="primary" @click="startCampaign(row.id)">{{ t('mail.bulkStart') }}</el-button>
              <el-button v-if="row.status === 'sending'" text type="warning" @click="cancelCampaign(row)">{{ t('common.cancel') }}</el-button>
              <el-button v-if="row.status !== 'sending'" text type="danger" @click="deleteCampaign(row)">{{ t('common.delete') }}</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <el-dialog v-model="providerDialog" :title="editingProvider ? t('mail.bulkEditProvider') : t('mail.bulkAddProvider')" width="560px">
      <el-form label-width="120px">
        <el-form-item :label="t('common.name')"><el-input v-model="providerForm.name" /></el-form-item>
        <el-form-item :label="t('mail.bulkProviderType')">
          <el-select v-model="providerForm.provider_type" style="width:100%" :disabled="!!editingProvider">
            <el-option v-for="c in catalog" :key="c.type" :label="c.name" :value="c.type" />
          </el-select>
          <p v-if="selectedMeta" class="form-hint">{{ selectedMeta.description }}</p>
        </el-form-item>
        <el-form-item :label="t('mail.bulkDefaultFrom')"><el-input v-model="providerForm.default_from" placeholder="noreply@yourdomain.com" /></el-form-item>
        <el-form-item :label="t('mail.bulkFromName')"><el-input v-model="providerForm.default_from_name" /></el-form-item>
        <el-form-item v-for="f in selectedMeta?.fields || []" :key="f" :label="f">
          <el-input v-model="providerForm.config[f]" :type="f.includes('password') || f.includes('secret') || f.includes('token') || f === 'api_key' ? 'password' : 'text'" show-password />
        </el-form-item>
        <el-form-item :label="t('common.enabled')"><el-switch v-model="providerForm.enabled" /></el-form-item>
        <el-form-item :label="t('mail.bulkDefault')"><el-switch v-model="providerForm.is_default" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="providerDialog = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="actionLoading === 'provider'" @click="saveProvider">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="testDialog" :title="t('mail.bulkTest')" width="420px">
      <el-form label-width="100px">
        <el-form-item :label="t('mail.bulkTestTo')"><el-input v-model="testTo" placeholder="test@example.com" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="testDialog = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="actionLoading === 'test'" @click="runTest">{{ t('mail.bulkSendTest') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="campaignDialog" :title="t('mail.bulkNewCampaign')" width="680px">
      <el-form label-width="110px">
        <el-form-item :label="t('mail.bulkProvider')">
          <el-select v-model="campaignForm.provider_id" style="width:100%">
            <el-option v-for="p in providers" :key="p.id" :label="p.name" :value="p.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('common.name')"><el-input v-model="campaignForm.name" /></el-form-item>
        <el-form-item :label="t('mail.bulkFrom')"><el-input v-model="campaignForm.from_email" placeholder="noreply@lulunet.cc" /></el-form-item>
        <el-form-item :label="t('mail.bulkFromName')"><el-input v-model="campaignForm.from_name" /></el-form-item>
        <el-form-item :label="t('mail.bulkReplyTo')"><el-input v-model="campaignForm.reply_to" /></el-form-item>
        <el-form-item :label="t('mail.bulkSubject')"><el-input v-model="campaignForm.subject" /></el-form-item>
        <el-form-item :label="t('mail.bulkBodyHtml')">
          <el-input v-model="campaignForm.body_html" type="textarea" :rows="6" />
        </el-form-item>
        <el-form-item :label="t('mail.bulkBodyText')">
          <el-input v-model="campaignForm.body_text" type="textarea" :rows="3" />
        </el-form-item>
        <el-form-item :label="t('mail.bulkRecipients')">
          <el-input v-model="campaignForm.recipients" type="textarea" :rows="6" :placeholder="t('mail.bulkRecipientsHint')" />
        </el-form-item>
        <el-form-item :label="t('mail.bulkRate')">
          <el-input-number v-model="campaignForm.rate_per_minute" :min="1" :max="600" />
          <span class="form-hint-inline">{{ t('mail.bulkRateHint') }}</span>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="campaignDialog = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="actionLoading === 'campaign'" @click="createCampaign">{{ t('mail.bulkCreateAndSend') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.hint { margin-bottom: 16px; }
.tab-toolbar { margin-bottom: 12px; display: flex; gap: 8px; flex-wrap: wrap; }
.form-hint { font-size: 12px; color: var(--el-text-color-secondary); margin-top: 4px; }
.form-hint-inline { margin-left: 8px; font-size: 12px; color: var(--el-text-color-secondary); }
.bulk-subtabs { margin-top: 8px; }
</style>
