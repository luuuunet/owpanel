<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { resolveApiError } from '@/api'
import { useAuthStore } from '@/stores/auth'
import { usePerformanceProfile } from '@/composables/usePerformanceProfile'

const emit = defineEmits<{ changed: [] }>()

const { t } = useI18n()
const auth = useAuthStore()
const { profile, load, setEnabled } = usePerformanceProfile()
const saving = ref(false)
const expanded = ref(false)

const isAdmin = computed(() => !auth.user?.role || auth.user.role === 'admin')

const intervalRows = computed(() => {
  const p = profile.value
  if (!p) return []
  return [
    { key: 'collect', label: t('dashboard.perfCollect'), value: p.collect_sec },
    { key: 'lite', label: t('dashboard.perfMonitorLite'), value: p.monitor_lite_sec },
    { key: 'full', label: t('dashboard.perfMonitorFull'), value: p.monitor_full_sec },
    { key: 'traffic', label: t('dashboard.perfTrafficPoll'), value: p.traffic_poll_sec },
    { key: 'map', label: t('dashboard.perfTrafficMap'), value: p.traffic_map_sec },
    { key: 'cluster', label: t('dashboard.perfClusterSync'), value: p.cluster_sync_sec },
    { key: 'uptime', label: t('dashboard.perfUptimeScan'), value: p.uptime_scan_sec },
  ]
})

async function onToggle(enabled: boolean) {
  if (!isAdmin.value) return
  saving.value = true
  try {
    await setEnabled(enabled)
    ElMessage.success(enabled ? t('dashboard.perfEnabledSuccess') : t('dashboard.perfDisabledSuccess'))
    emit('changed')
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('dashboard.perfSaveFailed')))
    await load(true)
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  load(true)
  setInterval(() => load(true), 30000)
})
</script>

<template>
  <el-card shadow="hover" class="perf-card">
    <div class="perf-head">
      <div class="perf-title-wrap">
        <h3 class="perf-title">{{ t('dashboard.perfTitle') }}</h3>
        <p class="perf-hint">{{ t('dashboard.perfHint') }}</p>
      </div>
      <div class="perf-actions">
        <el-tag :type="profile?.enabled ? 'warning' : 'success'" size="small">
          {{ profile?.enabled ? t('dashboard.perfModeOn') : t('dashboard.perfModeOff') }}
        </el-tag>
        <el-tag v-if="profile?.dashboard_live" type="success" size="small">{{ t('dashboard.perfLive') }}</el-tag>
        <el-tag v-else-if="profile?.idle_collect_sec" type="info" size="small">
          {{ t('dashboard.perfIdle', { sec: profile.idle_collect_sec }) }}
        </el-tag>
        <el-switch
          v-if="isAdmin"
          :model-value="!!profile?.enabled"
          :loading="saving"
          inline-prompt
          :active-text="t('dashboard.perfSwitchOn')"
          :inactive-text="t('dashboard.perfSwitchOff')"
          @change="onToggle"
        />
        <span v-else class="readonly-hint">{{ t('dashboard.perfAdminOnly') }}</span>
        <el-button text size="small" @click="expanded = !expanded">
          {{ expanded ? t('dashboard.perfCollapse') : t('dashboard.perfExpand') }}
        </el-button>
      </div>
    </div>
    <el-collapse-transition>
      <div v-show="expanded" class="perf-body">
        <el-table :data="intervalRows" size="small" stripe class="perf-table">
          <el-table-column prop="label" :label="t('dashboard.perfItem')" min-width="180" />
          <el-table-column :label="t('dashboard.perfInterval')" width="120" align="right">
            <template #default="{ row }">{{ row.value }}s</template>
          </el-table-column>
        </el-table>
        <p class="perf-note">{{ t('dashboard.perfNote') }}</p>
      </div>
    </el-collapse-transition>
  </el-card>
</template>

<style scoped>
.perf-card { margin-bottom: 0; }
.perf-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
}
.perf-title { margin: 0; font-size: 15px; font-weight: 600; }
.perf-hint { margin: 4px 0 0; font-size: 12px; color: var(--el-text-color-secondary); max-width: 640px; }
.perf-actions { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
.readonly-hint { font-size: 12px; color: var(--el-text-color-secondary); }
.perf-body { margin-top: 12px; padding-top: 12px; border-top: 1px solid var(--el-border-color-lighter); }
.perf-table { max-width: 480px; }
.perf-note { margin: 10px 0 0; font-size: 12px; color: var(--el-text-color-secondary); }
</style>
