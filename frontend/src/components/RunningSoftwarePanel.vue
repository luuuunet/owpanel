<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import api from '@/api'
import SoftwareIcon from '@/components/SoftwareIcon.vue'
import { ElMessage } from 'element-plus'

export interface InstalledAppMetric {
  key: string
  name: string
  category?: string
  version?: string
  port?: number
  live_status: string
  cpu: number
  memory: number
}

const props = defineProps<{
  apps: InstalledAppMetric[]
}>()

const emit = defineEmits<{ refresh: [] }>()

const { t } = useI18n()
const actionKey = ref('')

const sortedApps = computed(() =>
  [...props.apps].sort((a, b) => {
    if (a.live_status === 'running' && b.live_status !== 'running') return -1
    if (b.live_status === 'running' && a.live_status !== 'running') return 1
    return a.name.localeCompare(b.name)
  })
)

function statusType(s: string) {
  if (s === 'running') return 'success'
  if (s === 'installing') return 'warning'
  if (s === 'failed') return 'danger'
  return 'info'
}

function statusLabel(s: string) {
  if (s === 'running') return t('common.running')
  if (s === 'stopped') return t('common.stopped')
  if (s === 'installing') return t('software.installing')
  if (s === 'failed') return t('software.failed')
  return s
}

async function doAction(app: InstalledAppMetric, action: 'start' | 'stop' | 'restart') {
  const key = `${app.key}:${action}`
  actionKey.value = key
  try {
    await api.post(`/software/${app.key}/${action}`)
    ElMessage.success(t('common.success'))
    emit('refresh')
  } catch (e: any) {
    ElMessage.error(e?.error || e?.message || t('common.failed'))
  } finally {
    actionKey.value = ''
  }
}

function isLoading(app: InstalledAppMetric, action: string) {
  return actionKey.value === `${app.key}:${action}`
}
</script>

<template>
  <div class="running-software">
    <div v-if="sortedApps.length" class="software-list">
      <div v-for="app in sortedApps" :key="app.key" class="software-row">
        <SoftwareIcon :app-key="app.key" :size="28" simple />
        <div class="row-meta">
          <span class="row-name">{{ app.name }}</span>
          <span v-if="app.version || app.port" class="row-sub">
            <template v-if="app.version">{{ app.version }}</template>
            <template v-if="app.port">:{{ app.port }}</template>
          </span>
        </div>
        <div class="row-actions">
          <el-tag :type="statusType(app.live_status)" size="small" effect="plain" class="row-status">
            {{ statusLabel(app.live_status) }}
          </el-tag>
          <el-button
            v-if="app.live_status === 'running'"
            size="small"
            type="warning"
            plain
            :loading="isLoading(app, 'restart')"
            @click="doAction(app, 'restart')"
          >
            {{ t('common.restart') }}
          </el-button>
          <el-button
            v-if="app.live_status === 'running'"
            size="small"
            type="danger"
            plain
            :loading="isLoading(app, 'stop')"
            @click="doAction(app, 'stop')"
          >
            {{ t('common.stop') }}
          </el-button>
          <el-button
            v-else
            size="small"
            type="primary"
            plain
            :loading="isLoading(app, 'start')"
            @click="doAction(app, 'start')"
          >
            {{ t('common.start') }}
          </el-button>
        </div>
      </div>
    </div>
    <el-empty v-else :description="t('dashboard.noInstalledApps')" :image-size="56" />
  </div>
</template>

<style scoped>
.running-software { width: 100%; }
.software-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 8px;
  padding: 0 8px 12px;
}
.software-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border: 1px solid var(--cf-border);
  border-radius: 8px;
  background: var(--el-fill-color-lighter);
  min-width: 0;
}
.row-meta {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.row-name {
  font-weight: 600;
  font-size: 13px;
  line-height: 1.3;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.row-sub {
  font-size: 11px;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.row-actions {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4px;
  flex-shrink: 0;
}
.row-status { flex-shrink: 0; }

@media (min-width: 1200px) {
  .software-list {
    grid-template-columns: repeat(4, 1fr);
  }
}

@media (min-width: 768px) and (max-width: 1199px) {
  .software-list {
    grid-template-columns: repeat(3, 1fr);
  }
}

@media (max-width: 767px) {
  .software-list {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 480px) {
  .software-list {
    grid-template-columns: 1fr;
  }
}
</style>
