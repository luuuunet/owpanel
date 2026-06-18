<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import api from '@/api'
import SoftwareInstallLogDialog from '@/components/SoftwareInstallLogDialog.vue'
import { ElMessage, ElMessageBox } from 'element-plus'

const { t } = useI18n()

const certs = ref<any[]>([])
const status = ref<any>({})
const loading = ref(false)
const issueVisible = ref(false)
const uploadVisible = ref(false)

const issueForm = reactive({
  domain: '',
  san_domains: '',
  email: '',
  webroot: '',
  auto_renew: true,
  deploy: true,
})

const uploadForm = reactive({
  domain: '',
  email: '',
  fullchain: '',
  privkey: '',
  deploy: true,
})

const certbotHint = computed(() =>
  status.value?.certbot_installed === false ? t('sslPage.certbotMissing') : ''
)

const certbotInstallVisible = ref(false)
const certbotInstalling = ref(false)

function openCertbotInstall() {
  certbotInstallVisible.value = true
}

async function onCertbotInstallDone(payload: { success: boolean }) {
  certbotInstalling.value = false
  if (payload.success) {
    await load()
  }
}

async function load() {
  loading.value = true
  try {
    const [listRes, stRes]: any[] = await Promise.all([
      api.get('/ssl'),
      api.get('/ssl/status'),
    ])
    certs.value = listRes.data || []
    status.value = stRes.data || {}
  } finally {
    loading.value = false
  }
}

function statusTag(row: any) {
  if (row.status === 'simulated') return 'warning'
  if (row.status === 'active') {
    if (row.days_left != null && row.days_left <= 30) return 'warning'
    return 'success'
  }
  if (row.status === 'failed') return 'danger'
  return 'info'
}

function statusLabel(row: any) {
  if (row.status === 'simulated') return t('sslPage.simulated')
  if (row.status === 'active' && row.days_left != null && row.days_left <= 30) {
    return t('sslPage.expiringSoon')
  }
  return row.status
}

function formatExpiry(row: any) {
  if (!row.expires_at) return '-'
  return new Date(row.expires_at).toLocaleDateString()
}

async function submitIssue() {
  await api.post('/ssl', { ...issueForm })
  ElMessage.success(t('sslPage.issued'))
  issueVisible.value = false
  await load()
}

async function submitUpload() {
  await api.post('/ssl/upload', { ...uploadForm })
  ElMessage.success(t('sslPage.uploaded'))
  uploadVisible.value = false
  await load()
}

async function renewOne(row: any) {
  await api.post(`/ssl/${row.id}/renew`)
  ElMessage.success(t('sslPage.renewed'))
  await load()
}

async function renewAll() {
  await ElMessageBox.confirm(t('sslPage.renewAllConfirm'), t('common.confirm'), { type: 'warning' })
  const res: any = await api.post('/ssl/renew-all')
  const n = res.data?.renewed ?? 0
  ElMessage.success(t('sslPage.renewedCount', { n }))
  await load()
}

async function deployOne(row: any) {
  await api.post(`/ssl/${row.id}/deploy`)
  ElMessage.success(t('sslPage.deployed'))
}

async function handleDelete(row: any) {
  await ElMessageBox.confirm(t('sslPage.deleteConfirm'), t('common.confirm'), { type: 'warning' })
  await api.delete(`/ssl/${row.id}`)
  ElMessage.success(t('common.deleted'))
  await load()
}

onMounted(load)
</script>

<template>
  <div v-loading="loading">
    <div class="page-header">
      <h2>{{ t('sslPage.title') }}</h2>
      <div class="header-actions">
        <el-button @click="renewAll">{{ t('sslPage.renewAll') }}</el-button>
        <el-button @click="uploadVisible = true">{{ t('sslPage.upload') }}</el-button>
        <el-button type="primary" @click="issueVisible = true">{{ t('sslPage.issue') }}</el-button>
      </div>
    </div>

    <el-alert v-if="certs.some(c => c.status === 'simulated')" type="warning" :closable="false" show-icon class="hint-alert" :title="t('sslPage.simulatedHint')" />

    <el-alert v-if="certbotHint" type="warning" :closable="false" show-icon class="hint-alert">
      <div class="certbot-hint">
        <span>{{ certbotHint }}</span>
        <el-button type="primary" size="small" :loading="certbotInstalling" @click="openCertbotInstall">
          {{ t('sslPage.installCertbot') }}
        </el-button>
      </div>
    </el-alert>

    <SoftwareInstallLogDialog
      v-model="certbotInstallVisible"
      app-key="certbot"
      app-name="Certbot"
      trigger-install
      @done="onCertbotInstallDone"
    />

    <el-row :gutter="16" class="stat-row">
      <el-col :span="4">
        <el-card shadow="hover"><div class="stat-label">{{ t('sslPage.total') }}</div><div class="stat-value">{{ status.total ?? 0 }}</div></el-card>
      </el-col>
      <el-col :span="4">
        <el-card shadow="hover"><div class="stat-label">{{ t('sslPage.active') }}</div><div class="stat-value ok">{{ status.active ?? 0 }}</div></el-card>
      </el-col>
      <el-col :span="4">
        <el-card shadow="hover"><div class="stat-label">{{ t('sslPage.expiringSoonCount') }}</div><div class="stat-value warn">{{ status.expiring_soon ?? 0 }}</div></el-card>
      </el-col>
      <el-col :span="4">
        <el-card shadow="hover"><div class="stat-label">{{ t('sslPage.expired') }}</div><div class="stat-value bad">{{ status.expired ?? 0 }}</div></el-card>
      </el-col>
      <el-col :span="4">
        <el-card shadow="hover"><div class="stat-label">{{ t('sslPage.failed') }}</div><div class="stat-value bad">{{ status.failed ?? 0 }}</div></el-card>
      </el-col>
    </el-row>

    <el-table :data="certs" stripe>
      <el-table-column prop="domain" :label="t('sslPage.domain')" min-width="160" />
      <el-table-column prop="provider" :label="t('sslPage.provider')" width="120" />
      <el-table-column :label="t('common.status')" width="120">
        <template #default="{ row }">
          <el-tag :type="statusTag(row)" size="small">{{ statusLabel(row) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="t('sslPage.expiresAt')" width="120">
        <template #default="{ row }">{{ formatExpiry(row) }}</template>
      </el-table-column>
      <el-table-column :label="t('sslPage.daysLeft')" width="100">
        <template #default="{ row }">
          <span v-if="row.days_left != null" :class="{ warn: row.days_left <= 30, bad: row.days_left < 0 }">
            {{ row.days_left }}d
          </span>
          <span v-else>-</span>
        </template>
      </el-table-column>
      <el-table-column prop="auto_renew" :label="t('sslPage.autoRenew')" width="90">
        <template #default="{ row }">
          <el-tag :type="row.auto_renew ? 'success' : 'info'" size="small">
            {{ row.auto_renew ? t('common.yes') : t('common.no') }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="t('common.actions')" width="220" fixed="right">
        <template #default="{ row }">
          <el-button text type="primary" @click="renewOne(row)">{{ t('sslPage.renew') }}</el-button>
          <el-button text type="success" @click="deployOne(row)">{{ t('sslPage.deploy') }}</el-button>
          <el-button text type="danger" @click="handleDelete(row)">{{ t('common.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="issueVisible" :title="t('sslPage.issueTitle')" width="520px">
      <el-form label-width="120px">
        <el-form-item :label="t('sslPage.domain')"><el-input v-model="issueForm.domain" placeholder="example.com" /></el-form-item>
        <el-form-item :label="t('sslPage.sanDomains')">
          <el-input v-model="issueForm.san_domains" type="textarea" :rows="2" :placeholder="t('sslPage.sanPlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('sslPage.email')"><el-input v-model="issueForm.email" placeholder="admin@example.com" /></el-form-item>
        <el-form-item :label="t('sslPage.webroot')">
          <el-input v-model="issueForm.webroot" :placeholder="t('sslPage.webrootPlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('sslPage.autoRenew')"><el-switch v-model="issueForm.auto_renew" /></el-form-item>
        <el-form-item :label="t('sslPage.deploySite')"><el-switch v-model="issueForm.deploy" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="issueVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="submitIssue">{{ t('sslPage.issue') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="uploadVisible" :title="t('sslPage.uploadTitle')" width="560px">
      <el-form label-width="100px">
        <el-form-item :label="t('sslPage.domain')"><el-input v-model="uploadForm.domain" /></el-form-item>
        <el-form-item :label="t('sslPage.email')"><el-input v-model="uploadForm.email" /></el-form-item>
        <el-form-item :label="t('sslPage.fullchain')">
          <el-input v-model="uploadForm.fullchain" type="textarea" :rows="6" class="mono-input" placeholder="-----BEGIN CERTIFICATE-----" />
        </el-form-item>
        <el-form-item :label="t('sslPage.privkey')">
          <el-input v-model="uploadForm.privkey" type="textarea" :rows="4" class="mono-input" placeholder="-----BEGIN PRIVATE KEY-----" />
        </el-form-item>
        <el-form-item :label="t('sslPage.deploySite')"><el-switch v-model="uploadForm.deploy" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="uploadVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="submitUpload">{{ t('sslPage.upload') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}
.header-actions {
  display: flex;
  gap: 8px;
}
.hint-alert {
  margin-bottom: 16px;
}
.certbot-hint {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  width: 100%;
}
.stat-row {
  margin-bottom: 16px;
}
.stat-label {
  color: #909399;
  font-size: 13px;
  margin-bottom: 6px;
}
.stat-value {
  font-size: 22px;
  font-weight: 700;
}
.stat-value.ok { color: var(--el-color-success); }
.stat-value.warn { color: var(--el-color-warning); }
.stat-value.bad { color: var(--el-color-danger); }
.warn { color: var(--el-color-warning); font-weight: 600; }
.bad { color: var(--el-color-danger); font-weight: 600; }
.mono-input :deep(textarea) {
  font-family: Consolas, Monaco, monospace;
  font-size: 12px;
}
</style>
