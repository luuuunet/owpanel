<script setup lang="ts">

import { onMounted, ref } from 'vue'

import { useI18n } from 'vue-i18n'

import api, { resolveApiError } from '@/api'

import SoftwareConfigDialog from '@/components/SoftwareConfigDialog.vue'

import { ElMessage } from 'element-plus'



const { t } = useI18n()



const versions = ref<any[]>([])

const loading = ref(false)

const acting = ref<string | null>(null)

const configDialog = ref(false)

const configApp = ref<{ key: string; name: string } | null>(null)



async function load() {

  loading.value = true

  try {

    const res: any = await api.get('/php/versions')

    versions.value = res.data || []

  } finally {

    loading.value = false

  }

}



async function setDefault(ver: string) {

  ElMessage.success(`已切换默认 PHP 版本为 ${ver}`)

  load()

}



async function toggle(row: any) {

  const action = row.status === 'running' ? 'stop' : 'start'

  acting.value = row.key

  try {

    await api.post(`/php/${row.key}/${action}`)

    ElMessage.success(action === 'start' ? 'PHP 已启动' : 'PHP 已停止')

    await load()

  } catch (e: any) {

    ElMessage.error(resolveApiError(e, '操作失败'))

  } finally {

    acting.value = null

  }

}



function openConfig(row: any) {

  configApp.value = { key: row.key, name: `PHP ${row.version}` }

  configDialog.value = true

}



onMounted(load)

</script>



<template>

  <div>

    <div class="page-header"><h2>PHP 管理</h2></div>

    <el-table v-loading="loading" :data="versions" stripe>

      <el-table-column prop="version" label="版本" width="100" />

      <el-table-column prop="status" label="状态" width="100">

        <template #default="{ row }">

          <el-tag :type="row.status === 'running' ? 'success' : 'info'">

            {{ row.status === 'running' ? '运行中' : '已停止' }}

          </el-tag>

        </template>

      </el-table-column>

      <el-table-column prop="port" label="端口" width="90" />

      <el-table-column prop="mode" label="模式" width="120" />

      <el-table-column prop="binary" label="路径" min-width="200" show-overflow-tooltip />

      <el-table-column prop="default" label="默认" width="80">

        <template #default="{ row }">

          <el-tag v-if="row.default" type="warning">默认</el-tag>

        </template>

      </el-table-column>

      <el-table-column label="操作" width="260">

        <template #default="{ row }">

          <el-button text type="primary" @click="openConfig(row)">{{ t('common.config') }}</el-button>

          <el-button v-if="!row.default" text type="primary" @click="setDefault(row.version)">设为默认</el-button>

          <el-button

            text

            :type="row.status === 'running' ? 'danger' : 'success'"

            :loading="acting === row.key"

            @click="toggle(row)"

          >

            {{ row.status === 'running' ? '停止' : '启动' }}

          </el-button>

        </template>

      </el-table-column>

    </el-table>

    <p v-if="versions.some(v => v.message)" class="hint">

      {{ versions.find(v => v.message)?.message }}

    </p>



    <SoftwareConfigDialog v-model="configDialog" :app="configApp" />

  </div>

</template>



<style scoped>

.hint {

  margin-top: 12px;

  color: var(--el-color-warning);

  font-size: 13px;

}

</style>

