<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import api, { resolveApiError } from '@/api'
import { ElMessage, ElMessageBox } from 'element-plus'

export type ProcessSort = 'cpu' | 'memory'

const props = defineProps<{
  visible: boolean
  sort: ProcessSort
  isAdmin?: boolean
}>()

const emit = defineEmits<{ 'update:visible': [boolean]; refresh: [] }>()

const { t } = useI18n()

const loading = ref(false)
const processes = ref<any[]>([])
const killingPid = ref<number | null>(null)

const drawerTitle = () =>
  props.sort === 'memory' ? t('dashboard.topMemoryProcesses') : t('dashboard.topCpuProcesses')

async function loadProcesses() {
  loading.value = true
  try {
    const res: any = await api.get('/dashboard/processes', {
      params: { sort: props.sort, limit: 15 },
    })
    processes.value = res.data || []
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
    processes.value = []
  } finally {
    loading.value = false
  }
}

async function killProcess(row: { pid: number; name?: string; command?: string }) {
  try {
    await ElMessageBox.confirm(
      t('dashboard.killProcessConfirm', { name: row.name || row.pid, pid: row.pid }),
      t('dashboard.killProcess'),
      {
        type: 'warning',
        confirmButtonText: t('dashboard.killProcess'),
        cancelButtonText: t('common.cancel'),
      },
    )
  } catch {
    return
  }
  killingPid.value = row.pid
  try {
    await api.post(`/toolbox/system/processes/${row.pid}/kill`)
    ElMessage.success(t('dashboard.killProcessSuccess'))
    await loadProcesses()
    emit('refresh')
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('dashboard.killProcessFailed')))
  } finally {
    killingPid.value = null
  }
}

function onVisibleChange(open: boolean) {
  emit('update:visible', open)
}

watch(
  () => [props.visible, props.sort] as const,
  ([open]) => {
    if (open) loadProcesses()
  },
)
</script>

<template>
  <el-drawer
    :model-value="visible"
    :title="drawerTitle()"
    direction="rtl"
    size="min(560px, 92vw)"
    destroy-on-close
    @update:model-value="onVisibleChange"
  >
    <el-table v-loading="loading" :data="processes" size="small" stripe max-height="calc(100vh - 120px)">
      <el-table-column prop="pid" label="PID" width="70" />
      <el-table-column prop="name" :label="t('dashboard.procName')" width="120" show-overflow-tooltip />
      <el-table-column prop="user" :label="t('toolboxPage.user')" width="90" show-overflow-tooltip />
      <el-table-column :label="t('dashboard.cpuUsage')" width="72">
        <template #default="{ row }">{{ row.cpu?.toFixed?.(1) ?? row.cpu }}%</template>
      </el-table-column>
      <el-table-column :label="t('dashboard.memoryUsage')" width="72">
        <template #default="{ row }">{{ row.memory?.toFixed?.(1) ?? row.memory }}%</template>
      </el-table-column>
      <el-table-column prop="command" :label="t('dashboard.procCmd')" show-overflow-tooltip />
      <el-table-column v-if="isAdmin" :label="t('common.actions')" width="100" fixed="right">
        <template #default="{ row }">
          <el-button
            link
            type="danger"
            size="small"
            :loading="killingPid === row.pid"
            :disabled="row.pid <= 1"
            @click="killProcess(row)"
          >
            {{ t('dashboard.killProcess') }}
          </el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-empty v-if="!loading && !processes.length" :description="t('dashboard.noProcesses')" :image-size="64" />
  </el-drawer>
</template>
