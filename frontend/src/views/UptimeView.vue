<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import api from '@/api'
import { ElMessage, ElMessageBox } from 'element-plus'

const { t } = useI18n()

const monitors = ref<any[]>([])
const websites = ref<any[]>([])
const loading = ref(false)
const importing = ref(false)
const dialogVisible = ref(false)
const form = ref({
  name: '',
  url: 'https://',
  method: 'GET',
  interval_sec: 60,
  timeout_sec: 10,
  expected_status: 200,
  keyword: '',
  notify_webhook: '',
  enabled: true,
})

const quickPresets = [
  { key: 'fast', label: 'uptime.presetFast', interval: 60, desc: 'uptime.presetFastDesc' },
  { key: 'normal', label: 'uptime.presetNormal', interval: 300, desc: 'uptime.presetNormalDesc' },
  { key: 'slow', label: 'uptime.presetSlow', interval: 900, desc: 'uptime.presetSlowDesc' },
]

async function load() {
  loading.value = true
  try {
    const [mon, sites]: any[] = await Promise.all([
      api.get('/uptime'),
      api.get('/websites').catch(() => ({ data: [] })),
    ])
    monitors.value = mon.data || []
    websites.value = (sites.data || []).filter((w: any) => w.status === 'running')
  } finally {
    loading.value = false
  }
}

async function handleCreate() {
  await api.post('/uptime', form.value)
  ElMessage.success(t('uptime.created'))
  dialogVisible.value = false
  form.value = {
    name: '', url: 'https://', method: 'GET', interval_sec: 60, timeout_sec: 10,
    expected_status: 200, keyword: '', notify_webhook: '', enabled: true,
  }
  load()
}

async function importWebsites(intervalSec = 300) {
  importing.value = true
  try {
    const res: any = await api.post('/uptime/import-websites', { interval_sec: intervalSec })
    const d = res.data || {}
    ElMessage.success(t('uptime.importDone', { created: d.created || 0, skipped: d.skipped || 0 }))
    await load()
  } catch (e: any) {
    ElMessage.error(e?.error || e?.message || t('common.failed'))
  } finally {
    importing.value = false
  }
}

function openWithPreset(preset: typeof quickPresets[0]) {
  form.value.interval_sec = preset.interval
  dialogVisible.value = true
}

async function handleDelete(id: number) {
  await ElMessageBox.confirm(t('uptime.deleteConfirm'), t('common.warning'), { type: 'warning' })
  await api.delete(`/uptime/${id}`)
  ElMessage.success(t('common.deleted'))
  load()
}

async function handleToggle(row: any) {
  await api.patch(`/uptime/${row.id}`, { enabled: !row.enabled })
  ElMessage.success(t('common.updated'))
  load()
}

async function handleCheck(row: any) {
  await api.post(`/uptime/${row.id}/check`)
  ElMessage.success(t('uptime.checked'))
  load()
}

function statusType(s: string) {
  if (s === 'up') return 'success'
  if (s === 'down') return 'danger'
  return 'info'
}

onMounted(load)
</script>

<template>
  <div v-loading="loading">
    <div class="page-header">
      <div>
        <h2>{{ t('uptime.title') }}</h2>
        <p class="subtitle">{{ t('uptime.subtitle') }}</p>
      </div>
      <div class="header-actions">
        <el-button :loading="importing" :disabled="!websites.length" @click="importWebsites(300)">
          {{ t('uptime.importWebsites') }}
        </el-button>
        <el-button type="primary" @click="dialogVisible = true">{{ t('uptime.add') }}</el-button>
      </div>
    </div>

    <el-alert type="info" :closable="false" show-icon class="hint">
      <template #title>{{ t('uptime.whatIsTitle') }}</template>
      <template #default>{{ t('uptime.whatIsBody') }}</template>
    </el-alert>

    <h3 class="section-title">{{ t('uptime.quickTitle') }}</h3>
    <div class="preset-grid">
      <el-card shadow="never" class="preset-card preset-card--action" @click="importWebsites(300)">
        <div class="preset-name">{{ t('uptime.importWebsites') }}</div>
        <p class="preset-desc">{{ t('uptime.importWebsitesDesc', { n: websites.length }) }}</p>
        <el-button type="primary" size="small" :loading="importing" :disabled="!websites.length">
          {{ t('uptime.importBtn') }}
        </el-button>
      </el-card>
      <el-card v-for="p in quickPresets" :key="p.key" shadow="never" class="preset-card" @click="openWithPreset(p)">
        <div class="preset-name">{{ t(p.label) }}</div>
        <p class="preset-desc">{{ t(p.desc) }}</p>
      </el-card>
    </div>

    <el-table :data="monitors" stripe>
      <el-table-column prop="name" :label="t('uptime.name')" width="140" />
      <el-table-column prop="url" :label="t('uptime.url')" show-overflow-tooltip />
      <el-table-column :label="t('uptime.status')" width="100">
        <template #default="{ row }">
          <el-tag :type="statusType(row.last_status)">{{ row.last_status || 'unknown' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="t('uptime.latency')" width="100">
        <template #default="{ row }">{{ row.last_latency_ms ? `${row.last_latency_ms} ms` : '—' }}</template>
      </el-table-column>
      <el-table-column :label="t('uptime.lastCheck')" width="170">
        <template #default="{ row }">
          <span v-if="row.last_check_at">{{ new Date(row.last_check_at).toLocaleString() }}</span>
          <span v-else>—</span>
        </template>
      </el-table-column>
      <el-table-column prop="fail_count" :label="t('uptime.failCount')" width="90" />
      <el-table-column :label="t('common.status')" width="90">
        <template #default="{ row }">
          <el-switch :model-value="row.enabled" @change="handleToggle(row)" />
        </template>
      </el-table-column>
      <el-table-column :label="t('common.actions')" width="160" fixed="right">
        <template #default="{ row }">
          <el-button text type="primary" @click="handleCheck(row)">{{ t('uptime.checkNow') }}</el-button>
          <el-button text type="danger" @click="handleDelete(row.id)">{{ t('common.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-empty v-if="!monitors.length" :description="t('uptime.empty')">
      <el-button type="primary" :disabled="!websites.length" @click="importWebsites(300)">{{ t('uptime.importWebsites') }}</el-button>
    </el-empty>

    <el-dialog v-model="dialogVisible" :title="t('uptime.addTitle')" width="560px">
      <el-form :model="form" label-width="120px">
        <el-form-item :label="t('uptime.name')"><el-input v-model="form.name" /></el-form-item>
        <el-form-item :label="t('uptime.url')"><el-input v-model="form.url" /></el-form-item>
        <el-form-item :label="t('uptime.method')">
          <el-select v-model="form.method"><el-option label="GET" value="GET" /><el-option label="HEAD" value="HEAD" /></el-select>
        </el-form-item>
        <el-form-item :label="t('uptime.interval')"><el-input-number v-model="form.interval_sec" :min="15" :max="3600" /></el-form-item>
        <el-form-item :label="t('uptime.expectedStatus')"><el-input-number v-model="form.expected_status" :min="100" :max="599" /></el-form-item>
        <el-form-item :label="t('uptime.keyword')"><el-input v-model="form.keyword" :placeholder="t('uptime.keywordHint')" /></el-form-item>
        <el-form-item :label="t('uptime.webhook')"><el-input v-model="form.notify_webhook" :placeholder="t('uptime.webhookHint')" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="handleCreate">{{ t('common.create') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.subtitle { margin: 4px 0 0; font-size: 13px; color: var(--el-text-color-secondary); }
.header-actions { display: flex; gap: 8px; flex-wrap: wrap; }
.hint { margin-bottom: 16px; }
.section-title { margin: 0 0 12px; font-size: 15px; font-weight: 600; }
.preset-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 10px; margin-bottom: 16px; }
.preset-card { cursor: pointer; border: 1px solid var(--el-border-color-lighter); transition: border-color 0.15s; }
.preset-card:hover { border-color: var(--el-color-primary-light-5); }
.preset-card--action { cursor: default; }
.preset-name { font-weight: 600; margin-bottom: 6px; }
.preset-desc { margin: 0 0 10px; font-size: 12px; color: var(--el-text-color-secondary); line-height: 1.5; }
</style>
