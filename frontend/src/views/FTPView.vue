<script setup lang="ts">

import { onMounted, ref } from 'vue'

import { useI18n } from 'vue-i18n'

import api from '@/api'

import { ElMessage, ElMessageBox } from 'element-plus'



const { t } = useI18n()



const list = ref<any[]>([])

const dialogVisible = ref(false)

const form = ref({ username: '', password: '', path: '' })

const defaultPath = ref('')



async function load() {
  try {
    const boot: any = await api.get('/auth/bootstrap')
    const wp = boot.data?.website_path
    if (wp) defaultPath.value = wp.endsWith('/') || wp.endsWith('\\') ? wp : wp + '/'
  } catch { /* ignore */ }
  const res: any = await api.get('/ftp')
  list.value = res.data || []
}



function openCreate() {

  form.value = { username: '', password: '', path: defaultPath.value }

  dialogVisible.value = true

}



async function handleCreate() {

  await api.post('/ftp', form.value)

  ElMessage.success(t('ftpPage.created'))

  dialogVisible.value = false

  load()

}



async function handleDelete(id: number) {

  await ElMessageBox.confirm(t('common.confirmDelete'), t('common.warning'), { type: 'warning' })

  await api.delete(`/ftp/${id}`)

  ElMessage.success(t('common.deleted'))

  load()

}

async function handleSync() {

  await api.post('/ftp/sync')

  ElMessage.success(t('ftpPage.synced'))

  load()

}



onMounted(load)

</script>



<template>

  <div>

    <div class="page-header">

      <h2>{{ t('ftpPage.title') }}</h2>

      <div class="header-actions">
        <el-button @click="handleSync">{{ t('ftpPage.sync') }}</el-button>
        <el-button type="primary" @click="openCreate">{{ t('ftpPage.addAccount') }}</el-button>
      </div>

    </div>

    <el-table :data="list" stripe>

      <el-table-column prop="username" :label="t('common.username')" />

      <el-table-column prop="path" :label="t('ftpPage.rootPath')" />

      <el-table-column :label="t('ftpPage.syncedCol')" width="100">
        <template #default="{ row }">
          <el-tag :type="row.synced ? 'success' : 'warning'" size="small">{{ row.synced ? t('common.yes') : t('common.no') }}</el-tag>
        </template>
      </el-table-column>

      <el-table-column prop="status" :label="t('common.status')" width="100">

        <template #default="{ row }"><el-tag type="success">{{ row.status }}</el-tag></template>

      </el-table-column>

      <el-table-column :label="t('common.actions')" width="120">

        <template #default="{ row }">

          <el-button type="danger" text @click="handleDelete(row.id)">{{ t('common.delete') }}</el-button>

        </template>

      </el-table-column>

    </el-table>

    <el-dialog v-model="dialogVisible" :title="t('ftpPage.addAccount')" width="480px">

      <el-form :model="form" label-width="100px">

        <el-form-item :label="t('common.username')"><el-input v-model="form.username" /></el-form-item>

        <el-form-item :label="t('common.password')"><el-input v-model="form.password" type="password" show-password /></el-form-item>

        <el-form-item :label="t('ftpPage.rootPath')"><el-input v-model="form.path" /></el-form-item>

      </el-form>

      <template #footer>

        <el-button @click="dialogVisible = false">{{ t('common.cancel') }}</el-button>

        <el-button type="primary" @click="handleCreate">{{ t('ftpPage.create') }}</el-button>

      </template>

    </el-dialog>

  </div>

</template>

<style scoped>
.header-actions { display: flex; gap: 8px; }
</style>

