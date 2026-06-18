<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import api, { resolveApiError } from '@/api'
import { ElMessage } from 'element-plus'

const props = defineProps<{
  modelValue: boolean
  databases: any[]
}>()

const emit = defineEmits<{ 'update:modelValue': [boolean] }>()

const { t } = useI18n()

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

const loading = ref(false)
const installingName = ref('')
const togglingName = ref('')
const selectedDbId = ref<number | null>(null)
const extensions = ref<any[]>([])
const canInstall = ref(false)
const serverVersion = ref('')

const pgDatabases = computed(() =>
  props.databases.filter((d) => ['postgresql', 'postgres'].includes((d.type || '').toLowerCase())),
)

const selectedDb = computed(() => pgDatabases.value.find((d) => d.id === selectedDbId.value))

watch(
  () => props.modelValue,
  (v) => {
    if (v) {
      if (!selectedDbId.value && pgDatabases.value.length) {
        selectedDbId.value = pgDatabases.value[0].id
      }
      load()
    }
  },
)

watch(selectedDbId, () => {
  if (visible.value) load()
})

async function load() {
  loading.value = true
  try {
    const dbName = selectedDb.value?.name || ''
    const res: any = await api.get('/databases/pgsql/extensions', {
      params: dbName ? { database: dbName } : {},
    })
    extensions.value = res.data?.extensions || []
    canInstall.value = !!res.data?.can_install
    serverVersion.value = res.data?.server_version || ''
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
  } finally {
    loading.value = false
  }
}

async function installPackage(row: any) {
  installingName.value = row.name
  try {
    const dbName = selectedDb.value?.name || ''
    await api.post(`/databases/pgsql/extensions/${encodeURIComponent(row.name)}/install`, null, {
      params: dbName ? { database: dbName } : {},
    })
    ElMessage.success(t('databases.installExtPackageSuccess'))
    await load()
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
  } finally {
    installingName.value = ''
  }
}

async function toggleDbExtension(row: any) {
  if (!selectedDb.value) return
  togglingName.value = row.name
  try {
    await api.put(
      `/databases/${selectedDb.value.id}/pgsql/extensions/${encodeURIComponent(row.name)}`,
      { enabled: !row.installed },
    )
    row.installed = !row.installed
    ElMessage.success(t('common.success'))
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
  } finally {
    togglingName.value = ''
  }
}
</script>

<template>
  <el-dialog
    v-model="visible"
    :title="t('databases.extensionsTitle')"
    width="780px"
    destroy-on-close
    class="pgsql-ext-dialog"
  >
    <div class="ext-toolbar">
      <el-select
        v-model="selectedDbId"
        :placeholder="t('databases.extensionsDbPlaceholder')"
        style="width: 260px"
        filterable
      >
        <el-option
          v-for="db in pgDatabases"
          :key="db.id"
          :label="db.name"
          :value="db.id"
        />
      </el-select>
      <span v-if="serverVersion" class="ver-hint">PostgreSQL {{ serverVersion }}</span>
    </div>

    <el-table v-loading="loading" :data="extensions" stripe max-height="420" size="small">
      <el-table-column prop="name" :label="t('software.phpConfig.extName')" width="150" />
      <el-table-column prop="description" :label="t('databases.extDescription')" show-overflow-tooltip />
      <el-table-column :label="t('common.status')" width="200">
        <template #default="{ row }">
          <el-tag v-if="row.available" type="success" size="small" effect="plain">
            {{ t('databases.extAvailable') }}
          </el-tag>
          <el-tag v-else type="info" size="small" effect="plain">
            {{ t('databases.extNotAvailable') }}
          </el-tag>
          <el-tag
            v-if="selectedDb"
            :type="row.installed ? 'success' : 'warning'"
            size="small"
            effect="plain"
            class="db-tag"
          >
            {{ row.installed ? t('databases.extDbEnabled') : t('databases.extDbDisabled') }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="t('common.actions')" width="200" fixed="right">
        <template #default="{ row }">
          <el-button
            v-if="!row.available && row.can_install"
            text
            type="primary"
            size="small"
            :loading="installingName === row.name"
            @click="installPackage(row)"
          >
            {{ t('databases.installExtPackage') }}
          </el-button>
          <el-button
            v-if="selectedDb && row.available"
            text
            :type="row.installed ? 'danger' : 'success'"
            size="small"
            :loading="togglingName === row.name"
            @click="toggleDbExtension(row)"
          >
            {{ row.installed ? t('databases.disableExt') : t('databases.enableExt') }}
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-empty v-if="!pgDatabases.length" :description="t('databases.empty')" />
  </el-dialog>
</template>

<style scoped>
.ext-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}
.ver-hint {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}
.db-tag {
  margin-left: 6px;
}
</style>
