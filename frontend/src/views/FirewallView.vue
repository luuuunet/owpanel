<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import api from '@/api'
import { ElMessage, ElMessageBox } from 'element-plus'

const props = withDefaults(defineProps<{ embedded?: boolean }>(), { embedded: false })

const { t } = useI18n()

const rules = ref<any[]>([])
const status = ref<any>(null)
const dialogVisible = ref(false)
const form = ref({ port: 80, protocol: 'tcp', action: 'allow', source_ip: '', remark: '' })

async function load() {
  const [listRes, statusRes]: any[] = await Promise.all([
    api.get('/firewall'),
    api.get('/firewall/status'),
  ])
  rules.value = listRes.data || []
  status.value = statusRes.data
}

async function handleCreate() {
  await api.post('/firewall', form.value)
  ElMessage.success(t('firewall.created'))
  dialogVisible.value = false
  form.value = { port: 80, protocol: 'tcp', action: 'allow', source_ip: '', remark: '' }
  load()
}

async function handleDelete(id: number) {
  await ElMessageBox.confirm(t('firewall.deleteConfirm'), t('common.warning'), { type: 'warning' })
  await api.delete(`/firewall/${id}`)
  ElMessage.success(t('common.deleted'))
  load()
}

async function handleSync() {
  await api.post('/firewall/sync')
  ElMessage.success(t('firewall.synced'))
  load()
}

onMounted(load)
</script>

<template>
  <div>
    <div class="page-header" :class="{ 'page-header--embedded': props.embedded }">
      <h2 v-if="!props.embedded">{{ t('firewall.title') }}</h2>
      <div class="header-actions">
        <el-button @click="handleSync">{{ t('firewall.sync') }}</el-button>
        <el-button type="primary" @click="dialogVisible = true">{{ t('firewall.add') }}</el-button>
      </div>
    </div>

    <el-card v-if="status" class="status-card" shadow="never">
      <div class="status-grid">
        <div><span class="label">{{ t('firewall.backend') }}</span>{{ status.backend || 'none' }}</div>
        <div>
          <span class="label">{{ t('firewall.active') }}</span>
          <el-tag :type="status.active ? 'success' : 'info'" size="small">{{ status.active ? t('common.yes') : t('common.no') }}</el-tag>
        </div>
        <div><span class="label">{{ t('firewall.ruleCount') }}</span>{{ status.rule_count }}</div>
      </div>
      <p v-if="status.message" class="status-msg">{{ status.message }}</p>
    </el-card>

    <el-table :data="rules" stripe>
      <el-table-column prop="port" :label="t('firewall.port')" width="90" />
      <el-table-column prop="protocol" :label="t('firewall.protocol')" width="90" />
      <el-table-column prop="action" :label="t('firewall.action')" width="100">
        <template #default="{ row }">
          <el-tag :type="row.action === 'allow' ? 'success' : 'danger'">{{ row.action }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="source_ip" :label="t('firewall.sourceIp')" width="140">
        <template #default="{ row }">{{ row.source_ip || t('firewall.any') }}</template>
      </el-table-column>
      <el-table-column :label="t('firewall.applied')" width="100">
        <template #default="{ row }">
          <el-tag :type="row.applied ? 'success' : 'warning'" size="small">
            {{ row.applied ? t('common.yes') : t('common.no') }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="remark" :label="t('firewall.remark')" />
      <el-table-column :label="t('common.actions')" width="100" fixed="right">
        <template #default="{ row }">
          <el-button type="danger" text @click="handleDelete(row.id)">{{ t('common.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialogVisible" :title="t('firewall.addTitle')" width="440px">
      <el-form :model="form" label-width="80px">
        <el-form-item :label="t('firewall.port')"><el-input-number v-model="form.port" :min="1" :max="65535" /></el-form-item>
        <el-form-item :label="t('firewall.protocol')">
          <el-select v-model="form.protocol"><el-option label="TCP" value="tcp" /><el-option label="UDP" value="udp" /></el-select>
        </el-form-item>
        <el-form-item :label="t('firewall.action')">
          <el-select v-model="form.action"><el-option :label="t('firewall.allow')" value="allow" /><el-option :label="t('firewall.deny')" value="deny" /></el-select>
        </el-form-item>
        <el-form-item :label="t('firewall.sourceIp')">
          <el-input v-model="form.source_ip" :placeholder="t('firewall.sourceIpHint')" />
        </el-form-item>
        <el-form-item :label="t('firewall.remark')"><el-input v-model="form.remark" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="handleCreate">{{ t('common.add') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.header-actions { display: flex; gap: 8px; }
.status-card { margin-bottom: 16px; }
.status-grid { display: flex; gap: 24px; flex-wrap: wrap; }
.label { color: var(--el-text-color-secondary); margin-right: 8px; }
.status-msg { margin: 12px 0 0; font-size: 13px; color: var(--el-text-color-secondary); white-space: pre-wrap; }
</style>
