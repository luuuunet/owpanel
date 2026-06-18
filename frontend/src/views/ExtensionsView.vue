<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import api from '@/api'
import { ElMessage } from 'element-plus'
import type { ExtensionInfo } from '@/stores/extensions'
import { useExtensionsStore } from '@/stores/extensions'

const { t } = useI18n()
const extStore = useExtensionsStore()
const list = ref<ExtensionInfo[]>([])
const loading = ref(false)
const reloading = ref(false)
const extensionsDir = ref('')
const loadError = ref('')

async function load() {
  loading.value = true
  loadError.value = ''
  try {
    const res: any = await api.get('/extensions')
    const data = res.data
    if (Array.isArray(data)) {
      list.value = data
      extensionsDir.value = ''
    } else {
      list.value = data?.items || []
      extensionsDir.value = data?.dir || ''
    }
  } catch (e: any) {
    list.value = []
    loadError.value = e?.error || e?.message || t('extensions.loadFailed')
  } finally {
    loading.value = false
  }
}

async function reloadAll() {
  reloading.value = true
  try {
    const res: any = await api.post('/extensions/reload')
    ElMessage.success(res.data?.message || t('extensions.reloaded'))
    await load()
    await extStore.fetchMenu()
  } catch (e: any) {
    ElMessage.error(e?.error || t('extensions.reloadFailed'))
  } finally {
    reloading.value = false
  }
}

async function toggleEnabled(row: ExtensionInfo, enabled: boolean) {
  try {
    await api.patch(`/extensions/${row.id}/enabled`, { enabled })
    row.enabled = enabled
    ElMessage.success(enabled ? t('extensions.enabled') : t('extensions.disabled'))
    await extStore.fetchMenu()
  } catch (e: any) {
    ElMessage.error(e?.error || t('common.failed'))
    await load()
  }
}

const hookEvents = computed(() => [
  'panel.startup',
  'website.created',
  'website.deleted',
  'app.installed',
  'app.uninstalled',
  'backup.completed',
])

const useCases = computed(() => [
  t('extensions.useCase1'),
  t('extensions.useCase2'),
  t('extensions.useCase3'),
  t('extensions.useCase4'),
])

onMounted(load)
</script>

<template>
  <div class="extensions-page">
    <div class="page-header">
      <div>
        <h2>{{ t('extensions.title') }}</h2>
        <p class="subtitle">{{ t('extensions.subtitle') }}</p>
      </div>
      <el-button type="primary" :loading="reloading" @click="reloadAll">{{ t('extensions.reload') }}</el-button>
    </div>

    <el-alert v-if="loadError" :title="loadError" type="error" show-icon :closable="false" style="margin-bottom: 16px" />

    <el-alert :title="t('extensions.hint')" type="info" show-icon :closable="false" style="margin-bottom: 16px" />
    <el-alert v-if="extensionsDir" :title="t('extensions.dirPath', { path: extensionsDir })" type="success" show-icon :closable="false" style="margin-bottom: 16px" />

    <el-card shadow="never" style="margin-bottom: 16px">
      <template #header>{{ t('extensions.useCasesTitle') }}</template>
      <ul class="use-cases">
        <li v-for="(item, i) in useCases" :key="i">{{ item }}</li>
      </ul>
    </el-card>

    <el-card shadow="never" style="margin-bottom: 16px">
      <template #header>{{ t('extensions.manifestTitle') }}</template>
      <pre class="code-sample">{{ t('extensions.manifestSample') }}</pre>
    </el-card>

    <el-table v-loading="loading" :data="list" stripe>
      <el-table-column prop="name" :label="t('common.name')" min-width="140" />
      <el-table-column prop="id" label="ID" width="120" />
      <el-table-column prop="version" :label="t('common.version')" width="80" />
      <el-table-column prop="description" :label="t('common.description')" show-overflow-tooltip />
      <el-table-column :label="t('extensions.hooks')" width="160">
        <template #default="{ row }">
          <el-tag v-for="h in row.hooks" :key="h" size="small" style="margin: 2px">{{ h }}</el-tag>
          <span v-if="!row.hooks?.length" class="muted">—</span>
        </template>
      </el-table-column>
      <el-table-column prop="catalog_count" :label="t('extensions.catalog')" width="80" align="center" />
      <el-table-column :label="t('common.status')" width="100" align="center">
        <template #default="{ row }">
          <el-switch :model-value="row.enabled" @change="(v: boolean) => toggleEnabled(row, v)" />
        </template>
      </el-table-column>
      <el-table-column prop="dir" :label="t('extensions.dir')" show-overflow-tooltip min-width="200" />
    </el-table>

    <el-empty v-if="!loading && !list.length" :description="t('extensions.empty')" />

    <el-card shadow="never" style="margin-top: 16px">
      <template #header>{{ t('extensions.eventsTitle') }}</template>
      <el-tag v-for="ev in hookEvents" :key="ev" style="margin: 4px">{{ ev }}</el-tag>
    </el-card>
  </div>
</template>

<style scoped>
.extensions-page { min-height: 100%; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 16px; }
.subtitle { color: var(--el-text-color-secondary); margin: 4px 0 0; font-size: 13px; }
.code-sample { white-space: pre-wrap; font-size: 12px; margin: 0; color: var(--el-text-color-regular); }
.muted { color: var(--el-text-color-secondary); }
.use-cases { margin: 0; padding-left: 20px; font-size: 13px; line-height: 1.8; color: var(--el-text-color-regular); }
</style>
