<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import api from '@/api'
import { ElMessage, ElMessageBox } from 'element-plus'

const { t } = useI18n()

const users = ref<any[]>([])
const dialogVisible = ref(false)
const editVisible = ref(false)
const editingId = ref<number | null>(null)

const form = ref({
  username: '', password: '', role: 'user', disk_quota_mb: 0, remark: '',
  permissions: { websites: true, databases: true, files: true, docker: false, ftp: false, mail: false, backup: false, monitor: true, bastion: false },
})

const editForm = ref({
  role: 'subuser', disk_quota_mb: 0, remark: '',
  permissions: { websites: true, databases: false, files: false, docker: false, ftp: false, mail: false, backup: false, monitor: true, bastion: false },
})

const permKeys = ['websites', 'databases', 'files', 'docker', 'ftp', 'mail', 'backup', 'monitor', 'bastion'] as const

async function load() {
  const res: any = await api.get('/users')
  users.value = res.data || []
}

async function handleCreate() {
  await api.post('/users', {
    ...form.value,
    permissions: JSON.stringify(form.value.permissions),
  })
  ElMessage.success(t('users.created'))
  dialogVisible.value = false
  load()
}

function openEdit(row: any) {
  editingId.value = row.id
  let perms = { websites: false, databases: false, files: false, docker: false, ftp: false, mail: false, backup: false, monitor: false, bastion: false }
  try { perms = { ...perms, ...JSON.parse(row.permissions || '{}') } } catch { /* ignore */ }
  editForm.value = { role: row.role, disk_quota_mb: row.disk_quota_mb || 0, remark: row.remark || '', permissions: perms }
  editVisible.value = true
}

async function saveEdit() {
  if (!editingId.value) return
  await api.patch(`/users/${editingId.value}`, {
    role: editForm.value.role,
    disk_quota_mb: editForm.value.disk_quota_mb,
    remark: editForm.value.remark,
    permissions: JSON.stringify(editForm.value.permissions),
  })
  ElMessage.success(t('common.updated'))
  editVisible.value = false
  load()
}

async function handleDelete(row: any) {
  await ElMessageBox.confirm(t('users.deleteConfirm', { name: row.username }), t('common.warning'), { type: 'warning' })
  await api.delete(`/users/${row.id}`)
  ElMessage.success(t('common.deleted'))
  load()
}

onMounted(load)
</script>

<template>
  <div>
    <div class="page-header">
      <h2>{{ t('users.title') }}</h2>
      <el-button type="primary" @click="dialogVisible = true">{{ t('users.add') }}</el-button>
    </div>
    <el-alert type="info" :closable="false" show-icon class="hint">{{ t('users.hint') }}</el-alert>

    <el-table :data="users" stripe>
      <el-table-column prop="username" :label="t('common.username')" />
      <el-table-column prop="role" :label="t('users.role')" width="110">
        <template #default="{ row }"><el-tag :type="row.role === 'admin' ? 'danger' : 'info'">{{ row.role }}</el-tag></template>
      </el-table-column>
      <el-table-column :label="t('users.diskQuota')" width="140">
        <template #default="{ row }">
          <span v-if="row.disk_quota_mb">{{ row.disk_used_mb || 0 }} / {{ row.disk_quota_mb }} MB</span>
          <span v-else>—</span>
        </template>
      </el-table-column>
      <el-table-column prop="remark" :label="t('users.remark')" />
      <el-table-column prop="created_at" :label="t('users.createdAt')" width="170" />
      <el-table-column :label="t('common.actions')" width="160" fixed="right">
        <template #default="{ row }">
          <el-button text @click="openEdit(row)">{{ t('common.edit') }}</el-button>
          <el-button v-if="row.role !== 'admin'" text type="danger" @click="handleDelete(row)">{{ t('common.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialogVisible" :title="t('users.addTitle')" width="520px">
      <el-form :model="form" label-width="100px">
        <el-form-item :label="t('common.username')"><el-input v-model="form.username" /></el-form-item>
        <el-form-item :label="t('common.password')"><el-input v-model="form.password" type="password" show-password /></el-form-item>
        <el-form-item :label="t('users.role')">
          <el-select v-model="form.role">
            <el-option :label="t('users.roleAdmin')" value="admin" />
            <el-option :label="t('users.roleUser')" value="user" />
            <el-option :label="t('users.roleSubuser')" value="subuser" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('users.diskQuota')"><el-input-number v-model="form.disk_quota_mb" :min="0" /></el-form-item>
        <el-form-item v-if="form.role === 'subuser'" :label="t('users.permissions')">
          <el-checkbox v-for="k in permKeys" :key="k" v-model="form.permissions[k]">{{ t(`users.perm.${k}`) }}</el-checkbox>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="handleCreate">{{ t('common.create') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="editVisible" :title="t('users.editTitle')" width="520px">
      <el-form :model="editForm" label-width="100px">
        <el-form-item :label="t('users.role')">
          <el-select v-model="editForm.role">
            <el-option :label="t('users.roleAdmin')" value="admin" />
            <el-option :label="t('users.roleUser')" value="user" />
            <el-option :label="t('users.roleSubuser')" value="subuser" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('users.diskQuota')"><el-input-number v-model="editForm.disk_quota_mb" :min="0" /></el-form-item>
        <el-form-item :label="t('users.remark')"><el-input v-model="editForm.remark" /></el-form-item>
        <el-form-item v-if="editForm.role === 'subuser'" :label="t('users.permissions')">
          <el-checkbox v-for="k in permKeys" :key="k" v-model="editForm.permissions[k]">{{ t(`users.perm.${k}`) }}</el-checkbox>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="saveEdit">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.hint { margin-bottom: 16px; }
.el-checkbox { display: block; margin-left: 0; }
</style>
