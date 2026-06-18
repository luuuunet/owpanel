<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import api from '@/api'
import type { ExtensionInfo } from '@/stores/extensions'
import { useExtensionsStore } from '@/stores/extensions'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const extStore = useExtensionsStore()

const embedURL = ref('')
const title = ref('')
const detail = ref<ExtensionInfo | null>(null)
const loading = ref(true)
const error = ref('')

const extId = computed(() => String(route.params.id || ''))

async function resolveEmbed() {
  loading.value = true
  error.value = ''
  embedURL.value = ''
  detail.value = null
  try {
    await extStore.fetchMenu()
    const hit = extStore.menuItems.find(m => m.extension_id === extId.value || m.path === `/ext/${extId.value}`)
    if (hit?.external_url) {
      window.open(hit.external_url, '_blank', 'noopener,noreferrer')
      error.value = t('extensions.openedExternal')
      return
    }
    if (hit?.embed_url && /^https?:\/\//i.test(hit.embed_url)) {
      embedURL.value = hit.embed_url
      title.value = hit.title
      return
    }
    const res: any = await api.get(`/extensions/embed/${extId.value}`)
    const data = res.data || {}
    title.value = data.title || hit?.title || extId.value
    if (data.embed_url && /^https?:\/\//i.test(data.embed_url)) {
      embedURL.value = data.embed_url
      return
    }
    if (data.detail) {
      detail.value = data.detail
      return
    }
    const detailRes: any = await api.get(`/extensions/detail/${extId.value}`)
    detail.value = detailRes.data || null
  } catch {
    error.value = t('extensions.embedMissing')
  } finally {
    loading.value = false
  }
}

function goManage() {
  router.push('/extensions')
}

watch(extId, resolveEmbed)
onMounted(resolveEmbed)
</script>

<template>
  <div v-loading="loading" class="embed-page">
    <el-alert v-if="error" :title="error" type="warning" show-icon :closable="false" />

    <iframe v-else-if="embedURL" :src="embedURL" :title="title" class="embed-frame" />

    <el-card v-else-if="detail" shadow="never" class="detail-card">
      <template #header>
        <div class="detail-head">
          <div>
            <h2>{{ detail.name }}</h2>
            <p class="detail-sub">{{ detail.id }} · v{{ detail.version || '—' }}</p>
          </div>
          <el-button type="primary" link @click="goManage">{{ t('extensions.goManage') }}</el-button>
        </div>
      </template>

      <p v-if="detail.description" class="detail-desc">{{ detail.description }}</p>
      <p v-else class="detail-desc muted">{{ t('extensions.detailNoDesc') }}</p>

      <el-descriptions :column="1" border size="small" style="margin-top: 16px">
        <el-descriptions-item :label="t('common.status')">
          <el-tag :type="detail.enabled ? 'success' : 'info'" size="small">
            {{ detail.enabled ? t('common.enabled') : t('common.disabled') }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item v-if="detail.author" :label="t('extensions.author')">{{ detail.author }}</el-descriptions-item>
        <el-descriptions-item :label="t('extensions.hooks')">
          <el-tag v-for="h in detail.hooks" :key="h" size="small" style="margin: 2px">{{ h }}</el-tag>
          <span v-if="!detail.hooks?.length" class="muted">—</span>
        </el-descriptions-item>
        <el-descriptions-item :label="t('extensions.catalog')">{{ detail.catalog_count ?? 0 }}</el-descriptions-item>
        <el-descriptions-item :label="t('extensions.dir')">{{ detail.dir }}</el-descriptions-item>
      </el-descriptions>

      <el-alert
        :title="t('extensions.detailHint')"
        type="info"
        show-icon
        :closable="false"
        style="margin-top: 16px"
      />
    </el-card>

    <el-empty v-else :description="t('extensions.embedMissing')" />
  </div>
</template>

<style scoped>
.embed-page { height: calc(100vh - 120px); min-height: 400px; display: flex; flex-direction: column; }
.embed-frame { flex: 1; width: 100%; border: none; border-radius: 8px; background: #fff; }
.detail-card { flex: 1; }
.detail-head { display: flex; justify-content: space-between; align-items: flex-start; gap: 12px; }
.detail-head h2 { margin: 0; font-size: 20px; }
.detail-sub { margin: 4px 0 0; font-size: 13px; color: var(--el-text-color-secondary); }
.detail-desc { margin: 0; line-height: 1.6; color: var(--el-text-color-regular); }
.muted { color: var(--el-text-color-secondary); }
</style>
