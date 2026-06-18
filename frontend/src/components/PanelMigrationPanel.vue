<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import api, { apiBaseURL } from '@/api'
import { ElMessage, ElMessageBox } from 'element-plus'
import { UploadFilled } from '@element-plus/icons-vue'

const { t } = useI18n()

const loading = ref(false)
const exporting = ref(false)
const importing = ref(false)
const preview = ref<any>(null)
const includeLogs = ref(false)
const importMode = ref('replace')
const importFile = ref<File | null>(null)
const lastExport = ref<{ filename: string; size: number } | null>(null)

const countRows = computed(() => {
  const counts = preview.value?.manifest?.counts || {}
  return [
    { key: 'users', label: t('settings.migrationCountUsers') },
    { key: 'websites', label: t('settings.migrationCountWebsites') },
    { key: 'databases', label: t('settings.migrationCountDatabases') },
    { key: 'ssl_certificates', label: t('settings.migrationCountSSL') },
    { key: 'apps', label: t('settings.migrationCountApps') },
    { key: 'ftp_accounts', label: t('settings.migrationCountFTP') },
    { key: 'cron_jobs', label: t('settings.migrationCountCron') },
    { key: 'mail_domains', label: t('settings.migrationCountMailDomains') },
    { key: 'mailboxes', label: t('settings.migrationCountMailboxes') },
    { key: 'wordpress_sites', label: t('settings.migrationCountWordPress') },
    { key: 'extensions', label: t('settings.migrationCountExtensions') },
  ].map((row) => ({ ...row, value: counts[row.key] ?? 0 }))
})

async function loadPreview() {
  loading.value = true
  try {
    const res: any = await api.get('/settings/migration/preview')
    preview.value = res.data
  } catch (e: any) {
    ElMessage.error(e?.error || e?.message || t('settings.migrationPreviewFailed'))
  } finally {
    loading.value = false
  }
}

async function runExport() {
  exporting.value = true
  try {
    const res: any = await api.post(
      '/settings/migration/export',
      { include_logs: includeLogs.value, include_secrets: true },
      { timeout: 600000 }
    )
    lastExport.value = { filename: res.data?.filename, size: res.data?.size || 0 }
    ElMessage.success(t('settings.migrationExportDone'))
    await downloadBundle(res.data?.filename)
  } catch (e: any) {
    ElMessage.error(e?.error || e?.message || t('settings.migrationExportFailed'))
  } finally {
    exporting.value = false
  }
}

async function downloadBundle(filename?: string) {
  const name = filename || lastExport.value?.filename
  if (!name) return
  const token = localStorage.getItem('token')
  const url = `${apiBaseURL()}/settings/migration/download?file=${encodeURIComponent(name)}`
  const res = await fetch(url, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  })
  if (!res.ok) {
    ElMessage.error(t('settings.migrationDownloadFailed'))
    return
  }
  const blob = await res.blob()
  const a = document.createElement('a')
  a.href = URL.createObjectURL(blob)
  a.download = name
  a.click()
  URL.revokeObjectURL(a.href)
}

function onImportSelect(file: File) {
  importFile.value = file
  return false
}

async function runImport() {
  if (!importFile.value) {
    ElMessage.warning(t('settings.migrationImportNeedFile'))
    return
  }
  try {
    await ElMessageBox.confirm(
      importMode.value === 'replace'
        ? t('settings.migrationImportReplaceConfirm')
        : t('settings.migrationImportMergeConfirm'),
      t('common.warning'),
      { type: 'warning' }
    )
  } catch {
    return
  }
  importing.value = true
  try {
    const form = new FormData()
    form.append('file', importFile.value)
    form.append('mode', importMode.value)
    const token = localStorage.getItem('token')
    const res = await fetch(`${apiBaseURL()}/settings/migration/import`, {
      method: 'POST',
      headers: token ? { Authorization: `Bearer ${token}` } : {},
      body: form,
    })
    const body = await res.json()
    if (!res.ok) {
      throw new Error(body?.error || body?.message || t('settings.migrationImportFailed'))
    }
    ElMessage.success(t('settings.migrationImportDone'))
    if (body?.data?.requires_restart) {
      ElMessage.warning(t('settings.migrationRestartHint'))
    }
    importFile.value = null
    await loadPreview()
  } catch (e: any) {
    ElMessage.error(e?.message || t('settings.migrationImportFailed'))
  } finally {
    importing.value = false
  }
}

onMounted(loadPreview)
</script>

<template>
  <el-card shadow="hover" class="settings-card" v-loading="loading">
    <template #header>{{ t('settings.migrationSection') }}</template>

    <el-alert type="info" show-icon :closable="false" :title="t('settings.migrationIntro')" class="migration-alert" />

    <div class="migration-actions">
      <el-checkbox v-model="includeLogs">{{ t('settings.migrationIncludeLogs') }}</el-checkbox>
      <el-button type="primary" :loading="exporting" @click="runExport">
        {{ t('settings.migrationExport') }}
      </el-button>
      <el-button v-if="lastExport?.filename" :disabled="exporting" @click="downloadBundle()">
        {{ t('settings.migrationDownloadAgain') }}
      </el-button>
    </div>

    <div v-if="lastExport" class="hint">
      {{ t('settings.migrationLastExport', { name: lastExport.filename, size: lastExport.size }) }}
    </div>

    <el-descriptions v-if="preview?.manifest" :column="2" border size="small" class="migration-counts">
      <el-descriptions-item v-for="row in countRows" :key="row.key" :label="row.label">
        {{ row.value }}
      </el-descriptions-item>
    </el-descriptions>

    <el-divider>{{ t('settings.migrationImportTitle') }}</el-divider>
    <p class="hint">{{ t('settings.migrationImportHint') }}</p>

    <el-form label-width="120px" class="import-form">
      <el-form-item :label="t('settings.migrationImportMode')">
        <el-radio-group v-model="importMode">
          <el-radio value="replace">{{ t('settings.migrationImportReplace') }}</el-radio>
          <el-radio value="merge">{{ t('settings.migrationImportMerge') }}</el-radio>
        </el-radio-group>
      </el-form-item>
      <el-form-item :label="t('settings.migrationImportFile')">
        <el-upload drag :auto-upload="false" :show-file-list="true" :limit="1" accept=".tar.gz,.tgz" :before-upload="onImportSelect">
          <el-icon class="upload-icon"><UploadFilled /></el-icon>
          <div>{{ t('settings.migrationImportDrop') }}</div>
        </el-upload>
      </el-form-item>
      <el-form-item>
        <el-button type="warning" :loading="importing" @click="runImport">
          {{ t('settings.migrationImport') }}
        </el-button>
      </el-form-item>
    </el-form>

    <el-alert type="warning" show-icon :closable="false" :title="t('settings.migrationLimitations')" class="migration-alert" />
  </el-card>
</template>

<style scoped>
.migration-alert {
  margin-bottom: 12px;
}

.migration-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.migration-counts {
  margin-top: 12px;
}

.import-form {
  margin-top: 8px;
}

.upload-icon {
  font-size: 42px;
  color: var(--cf-text-muted);
}

.hint {
  margin-top: 6px;
  color: var(--cf-text-muted);
  font-size: 12px;
}
</style>
