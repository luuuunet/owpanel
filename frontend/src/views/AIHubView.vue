<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import api from '@/api'
import { ElMessage, ElMessageBox } from 'element-plus'
import ModelIcon from '@/components/ModelIcon.vue'
import { defaultPickId, groupCatalogEntries, groupHubModels, type BrandGroup } from '@/utils/modelBrand'

interface CatalogEntry {
  id: string
  name: string
  hf_model_id: string
  ollama_model: string
  modality: string
  pipeline_tag: string
  deploy_via: string
  app_store_key: string
  hub_deployable: boolean
  category: string
  params: string
  size_hint: string
  min_vram_gb: number
  cpu_ok: boolean
  gated: boolean
  tgi: boolean
  ollama: boolean
  tags: string[]
  description: string
  featured: boolean
}

interface HubModel {
  id: string
  name: string
  author: string
  downloads: number
  gated: boolean
  pipeline_tag: string
  modality: string
  tags: string[]
  deployable: boolean
  runtime_hint: string
  deploy_via: string
  app_store_key: string
  deploy_note: string
  hub_url: string
}

interface HubTaskOption {
  id: string
  label: string
  label_en: string
  modality: string
  placeholder: string
}

const { t, locale } = useI18n()
const router = useRouter()
const tab = ref('huggingface')
const loading = ref(false)

const hfStatus = ref<any>({})
const gpuInfo = ref<any>({})
const agents = ref<any[]>([])
const catalog = ref<CatalogEntry[]>([])
const hubTasks = ref<HubTaskOption[]>([])
const catalogModality = ref('all')
const hubQuery = ref('')
const hubTask = ref('text-generation')
const hubResults = ref<HubModel[]>([])
const hubSearching = ref(false)
const selectedHubModel = ref<HubModel | null>(null)
const marketTab = ref('hub')
const tokenSaving = ref(false)
const tokenTesting = ref(false)
const installLogs = ref<any>({ lines: [], status: 'idle' })
let logTimer: ReturnType<typeof setInterval> | null = null
let hubSearchTimer: ReturnType<typeof setTimeout> | null = null

const hfForm = ref({
  catalog_id: '',
  model_id: '',
  hf_token: '',
  runtime: 'tgi',
  enable_chat_ui: true,
  use_gpu: true,
  auto_configure_panel: true,
})

const selectedCatalog = computed(() =>
  catalog.value.find(c => c.id === hfForm.value.catalog_id) || null
)

const isHubSelection = computed(() => !hfForm.value.catalog_id && !!selectedHubModel.value)
const modelReadonly = computed(() => !!selectedCatalog.value && !isHubSelection.value)

const hubBrandPick = ref<Record<string, string>>({})
const catalogBrandPick = ref<Record<string, string>>({})

const hubBrandGroups = computed(() => groupHubModels(hubResults.value))
const catalogBrandGroups = computed(() => groupCatalogEntries(catalog.value))

const currentHubTask = computed(() => hubTasks.value.find(x => x.id === hubTask.value))

const catalogModalityOptions = computed(() => [
  { value: 'all', label: t('aiHub.modalityAll') },
  { value: 'text', label: t('aiHub.modalityText') },
  { value: 'image', label: t('aiHub.modalityImage') },
  { value: 'audio', label: t('aiHub.modalityAudio') },
  { value: 'video', label: t('aiHub.modalityVideo') },
  { value: 'vision', label: t('aiHub.modalityVision') },
])

function hubTaskLabel(task: HubTaskOption) {
  return String(locale.value || '').startsWith('en') ? (task.label_en || task.label) : task.label
}

function deployActionLabel(model?: HubModel | null, catalog?: CatalogEntry | null) {
  if (model?.deployable || catalog?.hub_deployable) return t('aiHub.deployFromCard')
  const key = model?.app_store_key || catalog?.app_store_key
  if (key) return t('aiHub.installRuntime')
  return t('aiHub.hubUnsupported')
}

const installing = computed(() =>
  hfStatus.value?.install_status === 'installing' || hfStatus.value?.status === 'installing'
)

const statusTagType = computed(() => {
  const s = hfStatus.value?.status
  if (s === 'running') return 'success'
  if (s === 'installing') return 'warning'
  if (s === 'failed') return 'danger'
  return 'info'
})

const runtimeOptions = computed(() => {
  const entry = selectedCatalog.value
  const hub = selectedHubModel.value
  const opts: { value: string; label: string; disabled?: boolean }[] = []
  if (hub && !entry) {
    if (hub.runtime_hint === 'tgi' || hub.deployable) {
      opts.push({ value: 'tgi', label: t('aiHub.runtimeTGI') })
    }
    if (hub.runtime_hint === 'ollama') {
      opts.push({ value: 'ollama', label: t('aiHub.runtimeOllama') })
    }
    if (!opts.length) {
      opts.push({ value: 'tgi', label: t('aiHub.runtimeTGI'), disabled: !hub.deployable })
    }
    return opts
  }
  if (!entry || entry.tgi) {
    opts.push({ value: 'tgi', label: t('aiHub.runtimeTGI') })
  }
  if (!entry || entry.ollama) {
    opts.push({ value: 'ollama', label: t('aiHub.runtimeOllama') })
  }
  return opts.length ? opts : [
    { value: 'tgi', label: t('aiHub.runtimeTGI') },
    { value: 'ollama', label: t('aiHub.runtimeOllama') },
  ]
})

const recommendedRuntime = computed(() => {
  const entry = selectedCatalog.value
  if (!entry) return hfForm.value.runtime
  if (entry.ollama && !entry.tgi) return 'ollama'
  if (entry.tgi && !entry.ollama) return 'tgi'
  if (entry.category === 'reasoning' || entry.ollama) return 'ollama'
  return 'tgi'
})

const tutorialSteps = computed(() => [
  t('aiHub.tutorialStep1'),
  t('aiHub.tutorialStep2'),
  t('aiHub.tutorialStep3'),
  t('aiHub.tutorialStep4'),
  t('aiHub.tutorialStep5'),
  t('aiHub.tutorialStep6'),
  t('aiHub.tutorialStep7'),
])

const afterDeployItems = computed(() => [
  t('aiHub.tutorialAfter1'),
  t('aiHub.tutorialAfter2'),
  t('aiHub.tutorialAfter3'),
])

const faqItems = computed(() => [
  t('aiHub.tutorialFaq1'),
  t('aiHub.tutorialFaq2'),
  t('aiHub.tutorialFaq3'),
  t('aiHub.tutorialFaq4'),
])

function syncHubBrandPicks() {
  const next = { ...hubBrandPick.value }
  for (const g of hubBrandGroups.value) {
    if (!next[g.key] || !g.models.some(m => m.id === next[g.key])) {
      const pool = g.models.filter(m => m.deployable)
      next[g.key] = defaultPickId(pool.length ? pool : g.models)
    }
  }
  hubBrandPick.value = next
}

function syncCatalogBrandPicks() {
  const next = { ...catalogBrandPick.value }
  for (const g of catalogBrandGroups.value) {
    if (!next[g.key] || !g.models.some(m => m.id === next[g.key])) {
      next[g.key] = g.models[0]?.id || ''
    }
  }
  catalogBrandPick.value = next
}

watch(hubResults, syncHubBrandPicks, { deep: true })
watch(catalog, syncCatalogBrandPicks, { deep: true })

function hubPickModel(group: BrandGroup<HubModel>): HubModel | undefined {
  const id = hubBrandPick.value[group.key]
  return group.models.find(m => m.id === id) || group.models.find(m => m.deployable) || group.models[0]
}

function isHubBrandActive(group: BrandGroup<HubModel>) {
  return !!selectedHubModel.value && group.models.some(m => m.id === selectedHubModel.value?.id)
}

function onHubBrandSelect(key: string, modelId: string) {
  hubBrandPick.value = { ...hubBrandPick.value, [key]: modelId }
  const model = hubResults.value.find(m => m.id === modelId)
  if (model?.deployable) selectHubModel(model)
}

function selectHubBrandGroup(group: BrandGroup<HubModel>) {
  const model = hubPickModel(group)
  if (model?.deployable) selectHubModel(model)
}

async function deployHubBrand(group: BrandGroup<HubModel>) {
  const model = hubPickModel(group)
  if (!model) return
  if (!model.deployable) {
    if (model.app_store_key) {
      selectHubModel(model)
      goSoftwareApp(model.app_store_key)
      return
    }
    ElMessage.warning(t('aiHub.hubNotDeployable'))
    return
  }
  selectHubModel(model)
  await setupHuggingFace()
}

function onCatalogBrandSelect(key: string, catalogId: string) {
  catalogBrandPick.value = { ...catalogBrandPick.value, [key]: catalogId }
  const entry = catalog.value.find(c => c.id === catalogId)
  if (entry) selectCatalogEntry(entry)
}

function isCatalogBrandActive(group: BrandGroup<CatalogEntry>) {
  return !!hfForm.value.catalog_id && group.models.some(m => m.id === hfForm.value.catalog_id)
}

function catalogPickEntry(group: BrandGroup<CatalogEntry>): CatalogEntry | undefined {
  const id = catalogBrandPick.value[group.key]
  return group.models.find(m => m.id === id) || group.models[0]
}

async function deployCatalogBrand(group: BrandGroup<CatalogEntry>) {
  const entry = catalogPickEntry(group)
  if (!entry) return
  if (!entry.hub_deployable) {
    if (entry.app_store_key) {
      selectCatalogEntry(entry)
      goSoftwareApp(entry.app_store_key)
      return
    }
    ElMessage.warning(t('aiHub.hubNotDeployable'))
    return
  }
  selectCatalogEntry(entry)
  await setupHuggingFace()
}

function goSoftwareApp(appKey: string) {
  router.push({ path: '/software', query: { tab: 'store', category: '人工智能', q: appKey } })
}

function hubOptionLabel(m: HubModel) {
  const dl = formatDownloads(m.downloads)
  return m.gated ? `${m.name} (${dl} · Gated)` : `${m.name} (${dl})`
}

function catalogOptionLabel(entry: CatalogEntry) {
  return `${entry.name} · ${entry.params}${entry.size_hint ? ` · ${entry.size_hint}` : ''}`
}

function categoryLabel(cat: string) {
  const key = `aiHub.category.${cat}` as const
  const translated = t(key)
  return translated !== key ? translated : cat
}

function selectCatalogEntry(entry: CatalogEntry) {
  selectedHubModel.value = null
  hfForm.value.catalog_id = entry.id
  hfForm.value.runtime = recommendedRuntimeFor(entry)
  hfForm.value.model_id = resolveModelID(entry, hfForm.value.runtime)
}

function selectHubModel(entry: HubModel) {
  hfForm.value.catalog_id = ''
  selectedHubModel.value = entry
  hfForm.value.model_id = entry.id
  hfForm.value.runtime = entry.runtime_hint || 'tgi'
}

function onCustomModelInput() {
  hfForm.value.catalog_id = ''
  selectedHubModel.value = null
}

function formatDownloads(n: number) {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`
  return String(n || 0)
}

async function searchHubModels() {
  hubSearching.value = true
  try {
    const res: any = await api.get('/ai/huggingface/search', {
      params: {
        q: hubQuery.value.trim(),
        task: hubTask.value,
        limit: 40,
        hf_token: hfForm.value.hf_token || undefined,
      },
    })
    hubResults.value = res.data || []
    syncHubBrandPicks()
  } catch (e: any) {
    ElMessage.error(e?.error || t('aiHub.hubSearchFailed'))
  } finally {
    hubSearching.value = false
  }
}

function scheduleHubSearch() {
  if (hubSearchTimer) clearTimeout(hubSearchTimer)
  hubSearchTimer = setTimeout(() => {
    searchHubModels()
  }, 400)
}

async function saveHFToken() {
  if (!hfForm.value.hf_token.trim()) {
    ElMessage.warning(t('aiHub.hfTokenRequired'))
    return
  }
  tokenSaving.value = true
  try {
    await api.put('/ai/huggingface/token', { hf_token: hfForm.value.hf_token })
    ElMessage.success(t('aiHub.hfTokenSaved'))
    await loadAll()
  } catch (e: any) {
    ElMessage.error(e?.error || t('common.failed'))
  } finally {
    tokenSaving.value = false
  }
}

async function testHFToken() {
  tokenTesting.value = true
  try {
    const res: any = await api.post('/ai/huggingface/token/test', {
      hf_token: hfForm.value.hf_token || undefined,
    })
    ElMessage.success(t('aiHub.hfTokenOk', { name: res.data?.name || res.data?.type || 'OK' }))
  } catch (e: any) {
    ElMessage.error(e?.error || t('aiHub.hfTokenInvalid'))
  } finally {
    tokenTesting.value = false
  }
}

function openHubSite() {
  window.open(hfStatus.value?.hub_url || 'https://huggingface.co/models', '_blank')
}

function openHubModel(url: string) {
  window.open(url, '_blank')
}

function recommendedRuntimeFor(entry: CatalogEntry) {
  if (entry.ollama && !entry.tgi) return 'ollama'
  if (entry.tgi && !entry.ollama) return 'tgi'
  if (entry.category === 'reasoning') return 'ollama'
  return gpuInfo.value.available ? 'tgi' : 'ollama'
}

function resolveModelID(entry: CatalogEntry, runtime: string) {
  if (runtime === 'ollama' && entry.ollama_model) return entry.ollama_model
  if (entry.hf_model_id) return entry.hf_model_id
  return entry.ollama_model
}

function onRuntimeChange() {
  const entry = selectedCatalog.value
  if (entry) {
    hfForm.value.model_id = resolveModelID(entry, hfForm.value.runtime)
  }
}

async function loadCatalog() {
  const params: Record<string, string> = {}
  if (catalogModality.value && catalogModality.value !== 'all') {
    params.modality = catalogModality.value
  }
  const cat: any = await api.get('/ai/huggingface/catalog', { params })
  catalog.value = cat.data || []
  syncCatalogBrandPicks()
}

async function loadAll() {
  const [st, gpu, tasks, agentList]: any[] = await Promise.all([
    api.get('/ai/huggingface/status'),
    api.get('/ai/gpu'),
    api.get('/ai/huggingface/tasks'),
    api.get('/ai/agents'),
  ])
  hfStatus.value = st.data || {}
  gpuInfo.value = gpu.data || {}
  hubTasks.value = tasks.data?.length ? tasks.data : defaultHubTasks()
  agents.value = agentList.data || []
  await loadCatalog()

  if (!hfForm.value.catalog_id && !selectedHubModel.value && catalog.value.length && !hfForm.value.model_id) {
    const featured = catalog.value.find(c => c.featured) || catalog.value[0]
    selectCatalogEntry(featured)
  }
  if (hfStatus.value.model_id && !hfForm.value.catalog_id) {
    hfForm.value.model_id = hfStatus.value.model_id
  }
  if (hfStatus.value.runtime) {
    hfForm.value.runtime = hfStatus.value.runtime
  }
  hfForm.value.use_gpu = !!gpuInfo.value.available
  await loadInstallLogs()
}

function defaultHubTasks(): HubTaskOption[] {
  return [
    { id: 'text-generation', label: '文本生成 / 对话', label_en: 'Text generation', modality: 'text', placeholder: 'Qwen, Llama…' },
    { id: 'text-to-image', label: '文生图', label_en: 'Text to image', modality: 'image', placeholder: 'SDXL, FLUX…' },
    { id: 'image-to-text', label: '图像理解', label_en: 'Image to text', modality: 'vision', placeholder: 'BLIP, Qwen-VL…' },
    { id: 'automatic-speech-recognition', label: '语音识别 ASR', label_en: 'Speech recognition', modality: 'audio', placeholder: 'Whisper…' },
    { id: 'text-to-speech', label: '语音合成 TTS', label_en: 'Text to speech', modality: 'audio', placeholder: 'Bark, XTTS…' },
    { id: 'text-to-video', label: '文生视频', label_en: 'Text to video', modality: 'video', placeholder: 'HunyuanVideo…' },
    { id: 'audio-to-audio', label: '音频处理', label_en: 'Audio processing', modality: 'audio', placeholder: 'MusicGen…' },
    { id: 'all', label: '全部类型', label_en: 'All types', modality: '', placeholder: '' },
  ]
}

watch(catalogModality, () => {
  loadCatalog()
})

watch(hubTask, () => {
  searchHubModels()
})

async function loadInstallLogs() {
  const res: any = await api.get('/ai/huggingface/install/logs')
  installLogs.value = res.data || { lines: [], status: 'idle' }
}

async function setupHuggingFace() {
  loading.value = true
  try {
    await api.post('/ai/huggingface/setup', hfForm.value)
    ElMessage.success(t('aiHub.setupStarted'))
    startLogPolling()
    await loadAll()
  } catch (e: any) {
    ElMessage.error(e?.error || t('common.failed'))
  } finally {
    loading.value = false
  }
}

async function uninstallHuggingFace() {
  try {
    await ElMessageBox.confirm(t('aiHub.uninstallConfirm'), t('common.confirm'), { type: 'warning' })
  } catch {
    return
  }
  loading.value = true
  try {
    await api.post('/ai/huggingface/uninstall')
    ElMessage.success(t('aiHub.uninstalled'))
    await loadAll()
  } catch (e: any) {
    ElMessage.error(e?.error || t('common.failed'))
  } finally {
    loading.value = false
  }
}

function openChatUI() {
  if (hfStatus.value?.chat_url) window.open(hfStatus.value.chat_url, '_blank')
}

async function copyText(text: string, okMsg: string) {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success(okMsg)
  } catch {
    ElMessage.error(t('common.failed'))
  }
}

function goSettings() {
  router.push('/settings')
}

function goSoftware() {
  router.push({ path: '/software', query: { tab: 'store', category: '人工智能' } })
}

function startLogPolling() {
  stopLogPolling()
  logTimer = setInterval(async () => {
    await loadInstallLogs()
    if (installLogs.value.status !== 'installing') {
      await loadAll()
      stopLogPolling()
    }
  }, 2000)
}

function stopLogPolling() {
  if (logTimer) {
    clearInterval(logTimer)
    logTimer = null
  }
}

onMounted(async () => {
  loading.value = true
  try {
    await loadAll()
    await searchHubModels()
    if (installing.value) startLogPolling()
  } finally {
    loading.value = false
  }
})

onUnmounted(() => {
  stopLogPolling()
  if (hubSearchTimer) clearTimeout(hubSearchTimer)
})
</script>

<template>
  <div class="ai-hub-view" v-loading="loading">
    <el-tabs v-model="tab">
      <el-tab-pane :label="t('aiHub.tabHuggingFace')" name="huggingface">
        <el-row :gutter="16">
          <el-col :xs="24" :lg="14">
            <el-card shadow="never" class="mb-16">
              <template #header>
                <div class="card-head">
                  <span>{{ t('aiHub.marketplaceTitle') }}</span>
                  <el-tag :type="statusTagType">{{ hfStatus.status || 'stopped' }}</el-tag>
                </div>
              </template>

              <el-alert type="info" :closable="false" show-icon class="mb-16">
                {{ t('aiHub.hfDesc') }}
                <el-link type="primary" class="hub-link" @click="openHubSite">huggingface.co</el-link>
              </el-alert>

              <div class="hf-token-bar mb-16">
                <el-input
                  v-model="hfForm.hf_token"
                  type="password"
                  show-password
                  :placeholder="t('aiHub.hfTokenHint')"
                  class="token-input"
                />
                <el-button :loading="tokenSaving" @click="saveHFToken">{{ t('aiHub.saveToken') }}</el-button>
                <el-button :loading="tokenTesting" @click="testHFToken">{{ t('aiHub.testToken') }}</el-button>
                <el-tag v-if="hfStatus.hf_token_configured" type="success" size="small">{{ t('aiHub.tokenConfigured') }}</el-tag>
              </div>

              <el-tabs v-model="marketTab" class="market-tabs mb-16">
                <el-tab-pane :label="t('aiHub.hubSearchTab')" name="hub">
                  <div class="hub-toolbar">
                    <el-input
                      v-model="hubQuery"
                      clearable
                      :placeholder="currentHubTask?.placeholder || t('aiHub.hubSearchPlaceholder')"
                      @input="scheduleHubSearch"
                      @keyup.enter="searchHubModels"
                    />
                    <el-select v-model="hubTask" style="width: 200px">
                      <el-option
                        v-for="task in hubTasks"
                        :key="task.id"
                        :label="hubTaskLabel(task)"
                        :value="task.id"
                      />
                    </el-select>
                    <el-button type="primary" :loading="hubSearching" @click="searchHubModels">{{ t('aiHub.hubSearch') }}</el-button>
                    <el-button @click="openHubSite">{{ t('aiHub.openHub') }}</el-button>
                  </div>
                  <el-alert
                    v-if="hubTask === 'image-to-text'"
                    type="info"
                    :closable="false"
                    show-icon
                    class="mb-16"
                    :title="t('aiHub.visionDeployHint')"
                  />
                  <div v-loading="hubSearching" class="model-grid hub-grid">
                    <div
                      v-for="group in hubBrandGroups"
                      :key="group.key"
                      class="model-card brand-card"
                      :class="{ active: isHubBrandActive(group) }"
                      @click="selectHubBrandGroup(group)"
                    >
                      <div class="model-card-head">
                        <div class="model-title-row">
                          <ModelIcon catalog-id="" :model-id="group.iconModelId" :size="40" />
                          <div class="model-title-text">
                            <strong>{{ group.label }}</strong>
                            <span class="model-author">
                              {{ t('aiHub.brandVariants', { n: group.models.length }) }}
                              · {{ formatDownloads(group.totalDownloads) }} ↓
                            </span>
                          </div>
                        </div>
                        <div class="card-badges">
                          <el-tag v-if="group.hasGated" size="small" type="warning">{{ t('aiHub.gated') }}</el-tag>
                          <el-tag v-if="!hubPickModel(group)?.deployable && !hubPickModel(group)?.app_store_key" size="small" type="info">
                            {{ t('aiHub.hubUnsupported') }}
                          </el-tag>
                        </div>
                      </div>
                      <div class="brand-variant-row" @click.stop>
                        <span class="variant-label">{{ t('aiHub.selectVariant') }}</span>
                        <el-select
                          :model-value="hubBrandPick[group.key]"
                          filterable
                          style="width: 100%"
                          @update:model-value="onHubBrandSelect(group.key, $event)"
                        >
                          <el-option
                            v-for="m in group.models"
                            :key="m.id"
                            :label="hubOptionLabel(m)"
                            :value="m.id"
                            :disabled="!m.deployable"
                          />
                        </el-select>
                      </div>
                      <div v-if="hubPickModel(group)?.deploy_note && !hubPickModel(group)?.deployable" class="deploy-note">
                        {{ hubPickModel(group)?.deploy_note }}
                      </div>
                      <div class="hub-card-actions">
                        <el-button
                          link
                          type="primary"
                          size="small"
                          @click.stop="openHubModel(hubPickModel(group)?.hub_url || '')"
                        >
                          {{ t('aiHub.viewOnHub') }}
                        </el-button>
                        <el-button
                          type="primary"
                          size="small"
                          :disabled="!hubPickModel(group)?.deployable && !hubPickModel(group)?.app_store_key"
                          :loading="installing && isHubBrandActive(group)"
                          @click.stop="deployHubBrand(group)"
                        >
                          {{ deployActionLabel(hubPickModel(group)) }}
                        </el-button>
                      </div>
                    </div>
                    <el-empty v-if="!hubSearching && !hubBrandGroups.length" :description="t('aiHub.hubEmpty')" />
                  </div>
                </el-tab-pane>

                <el-tab-pane :label="t('aiHub.catalogTab')" name="catalog">
                  <div class="modality-bar">
                    <el-radio-group v-model="catalogModality" size="small">
                      <el-radio-button
                        v-for="opt in catalogModalityOptions"
                        :key="opt.value"
                        :value="opt.value"
                      >
                        {{ opt.label }}
                      </el-radio-button>
                    </el-radio-group>
                  </div>
                  <div class="catalog-section">
                    <div class="model-grid">
                      <div
                        v-for="group in catalogBrandGroups"
                        :key="group.key"
                        class="model-card brand-card"
                        :class="{ active: isCatalogBrandActive(group) }"
                        @click="onCatalogBrandSelect(group.key, catalogBrandPick[group.key] || group.models[0]?.id)"
                      >
                        <div class="model-card-head">
                          <div class="model-title-row">
                            <ModelIcon :catalog-id="catalogPickEntry(group)?.id || ''" :model-id="group.iconModelId" :size="40" />
                            <div class="model-title-text">
                              <strong>{{ group.label }}</strong>
                              <span class="model-author">{{ t('aiHub.brandVariants', { n: group.models.length }) }}</span>
                            </div>
                          </div>
                          <el-tag v-if="group.hasGated" size="small" type="warning">{{ t('aiHub.gated') }}</el-tag>
                        </div>
                        <div class="brand-variant-row" @click.stop>
                          <span class="variant-label">{{ t('aiHub.selectVariant') }}</span>
                          <el-select
                            :model-value="catalogBrandPick[group.key]"
                            style="width: 100%"
                            @update:model-value="onCatalogBrandSelect(group.key, $event)"
                          >
                            <el-option
                              v-for="entry in group.models"
                              :key="entry.id"
                              :label="catalogOptionLabel(entry)"
                              :value="entry.id"
                            />
                          </el-select>
                        </div>
                        <p v-if="catalogPickEntry(group)?.description" class="model-desc">
                          {{ catalogPickEntry(group)?.description }}
                        </p>
                        <div class="model-hints">
                          <template v-if="catalogPickEntry(group)">
                            <el-tag size="small" type="info">{{ categoryLabel(catalogPickEntry(group)!.category) }}</el-tag>
                            <span v-if="catalogPickEntry(group)!.cpu_ok">{{ t('aiHub.cpuOk') }}</span>
                            <span v-if="catalogPickEntry(group)!.min_vram_gb > 0">{{ t('aiHub.gpuHint', { gb: catalogPickEntry(group)!.min_vram_gb }) }}</span>
                          </template>
                        </div>
                        <el-button
                          type="primary"
                          size="small"
                          :loading="installing && isCatalogBrandActive(group)"
                          @click.stop="deployCatalogBrand(group)"
                        >
                          {{ deployActionLabel(null, catalogPickEntry(group)) }}
                        </el-button>
                      </div>
                    </div>
                  </div>
                </el-tab-pane>
              </el-tabs>
            </el-card>

            <el-card shadow="never">
              <template #header>
                <span>{{ t('aiHub.hfTitle') }}</span>
              </template>

              <el-collapse class="tutorial-collapse mb-16">
                <el-collapse-item :title="t('aiHub.tutorialTitle')" name="tutorial">
                  <ol class="tutorial-steps">
                    <li v-for="(step, i) in tutorialSteps" :key="i">{{ step }}</li>
                  </ol>
                  <div class="tutorial-notes">
                    <p><strong>{{ t('aiHub.tutorialAfterTitle') }}</strong></p>
                    <ul>
                      <li v-for="(item, i) in afterDeployItems" :key="'a' + i">{{ item }}</li>
                    </ul>
                    <p><strong>{{ t('aiHub.tutorialFaqTitle') }}</strong></p>
                    <ul>
                      <li v-for="(item, i) in faqItems" :key="'f' + i">{{ item }}</li>
                    </ul>
                  </div>
                </el-collapse-item>
              </el-collapse>

              <el-form label-width="140px" label-position="left">
                <el-form-item :label="t('aiHub.runtime')">
                  <el-radio-group v-model="hfForm.runtime" @change="onRuntimeChange">
                    <el-radio
                      v-for="opt in runtimeOptions"
                      :key="opt.value"
                      :value="opt.value"
                      :disabled="opt.disabled"
                    >
                      {{ opt.label }}
                    </el-radio>
                  </el-radio-group>
                  <div v-if="selectedCatalog" class="runtime-hint">
                    {{ t('aiHub.runtimeRecommend', { runtime: recommendedRuntime === 'ollama' ? t('aiHub.runtimeOllama') : t('aiHub.runtimeTGI') }) }}
                  </div>
                </el-form-item>
                <el-form-item :label="t('aiHub.model')">
                  <div v-if="selectedCatalog || selectedHubModel" class="selected-model-row">
                    <ModelIcon
                      :catalog-id="selectedCatalog?.id || ''"
                      :model-id="selectedHubModel?.id || hfForm.model_id"
                      :size="32"
                    />
                    <el-input
                      v-model="hfForm.model_id"
                      :readonly="modelReadonly"
                      :placeholder="t('aiHub.customModelPlaceholder')"
                      @input="onCustomModelInput"
                    />
                  </div>
                  <el-input
                    v-else
                    v-model="hfForm.model_id"
                    :placeholder="t('aiHub.customModelPlaceholder')"
                    @input="onCustomModelInput"
                  />
                  <div v-if="selectedHubModel" class="field-hint">
                    <el-link type="primary" @click="openHubModel(selectedHubModel.hub_url)">{{ t('aiHub.viewOnHub') }}</el-link>
                  </div>
                </el-form-item>
                <el-form-item :label="t('aiHub.hfToken')">
                  <el-input v-model="hfForm.hf_token" type="password" show-password :placeholder="t('aiHub.hfTokenHint')" />
                  <div v-if="selectedCatalog?.gated || selectedHubModel?.gated" class="field-hint">{{ t('aiHub.gatedTokenHint') }}</div>
                </el-form-item>
                <el-form-item :label="t('aiHub.options')">
                  <el-checkbox v-model="hfForm.enable_chat_ui">{{ t('aiHub.enableChatUI') }}</el-checkbox>
                  <el-checkbox v-model="hfForm.use_gpu" :disabled="!gpuInfo.available">{{ t('aiHub.useGPU') }}</el-checkbox>
                  <el-checkbox v-model="hfForm.auto_configure_panel">{{ t('aiHub.autoConfigurePanel') }}</el-checkbox>
                </el-form-item>
                <el-form-item>
                  <el-button type="primary" :loading="installing" @click="setupHuggingFace">
                    {{ hfStatus.installed ? t('aiHub.redeploy') : t('aiHub.oneClickSetup') }}
                  </el-button>
                  <el-button v-if="hfStatus.webui_running" @click="openChatUI">{{ t('aiHub.openChatUI') }}</el-button>
                  <el-button v-if="hfStatus.installed" type="danger" plain @click="uninstallHuggingFace">{{ t('aiHub.uninstall') }}</el-button>
                </el-form-item>
              </el-form>

              <el-card v-if="hfStatus.installed || hfStatus.api_base_url" shadow="never" class="api-section mt-16">
                <template #header>{{ t('aiHub.apiSectionTitle') }}</template>
                <el-descriptions :column="1" border size="small">
                  <el-descriptions-item v-if="hfStatus.public_ip" :label="t('aiHub.publicIP')">
                    <span class="mono">{{ hfStatus.public_ip }}</span>
                  </el-descriptions-item>
                  <el-descriptions-item v-if="hfStatus.api_base_url_public" :label="t('aiHub.apiBasePublic')">
                    <span class="mono">{{ hfStatus.api_base_url_public }}</span>
                    <el-button link type="primary" size="small" @click="copyText(hfStatus.api_base_url_public, t('aiHub.copied'))">
                      {{ t('aiHub.copy') }}
                    </el-button>
                  </el-descriptions-item>
                  <el-descriptions-item :label="t('aiHub.apiBaseLocal')">
                    <span class="mono">{{ hfStatus.api_base_url_local || hfStatus.api_base_url }}</span>
                    <el-button link type="primary" size="small" @click="copyText(hfStatus.api_base_url_local || hfStatus.api_base_url, t('aiHub.copied'))">
                      {{ t('aiHub.copy') }}
                    </el-button>
                  </el-descriptions-item>
                  <el-descriptions-item :label="t('aiHub.apiKey')">
                    <span class="mono">{{ hfStatus.api_key }}</span>
                    <el-button link type="primary" size="small" @click="copyText(hfStatus.api_key, t('aiHub.copied'))">
                      {{ t('aiHub.copy') }}
                    </el-button>
                  </el-descriptions-item>
                  <el-descriptions-item v-if="hfStatus.chat_url_public" :label="t('aiHub.chatUrlPublic')">
                    <span class="mono">{{ hfStatus.chat_url_public }}</span>
                    <el-button link type="primary" size="small" @click="copyText(hfStatus.chat_url_public, t('aiHub.copied'))">
                      {{ t('aiHub.copy') }}
                    </el-button>
                  </el-descriptions-item>
                  <el-descriptions-item :label="t('aiHub.chatUrlLocal')">
                    <span class="mono">{{ hfStatus.chat_url_local || hfStatus.chat_url }}</span>
                  </el-descriptions-item>
                  <el-descriptions-item :label="t('aiHub.panelLinked')">
                    <el-tag :type="hfStatus.panel_configured ? 'success' : 'info'" size="small">
                      {{ hfStatus.panel_configured ? t('aiHub.yes') : t('aiHub.no') }}
                    </el-tag>
                  </el-descriptions-item>
                  <el-descriptions-item v-if="hfStatus.openai_compat" :label="t('aiHub.openaiCompat')">
                    {{ t('aiHub.openaiCompatNote') }}
                  </el-descriptions-item>
                  <el-descriptions-item v-if="hfStatus.api_base_url_public" :label="t('aiHub.firewallHint')">
                    {{ t('aiHub.firewallHintText', { api: hfStatus.tgi_port, web: hfStatus.webui_port }) }}
                  </el-descriptions-item>
                </el-descriptions>
                <div v-if="hfStatus.api_sample_curl" class="curl-block">
                  <div class="curl-head">
                    <span>{{ hfStatus.api_base_url_public ? t('aiHub.sampleCurlPublic') : t('aiHub.sampleCurl') }}</span>
                    <el-button link type="primary" size="small" @click="copyText(hfStatus.api_sample_curl, t('aiHub.copiedCurl'))">
                      {{ t('aiHub.copyCurl') }}
                    </el-button>
                  </div>
                  <pre>{{ hfStatus.api_sample_curl }}</pre>
                </div>
                <div v-if="hfStatus.api_sample_curl_local && hfStatus.api_base_url_public" class="curl-block">
                  <div class="curl-head">
                    <span>{{ t('aiHub.sampleCurlLocal') }}</span>
                    <el-button link type="primary" size="small" @click="copyText(hfStatus.api_sample_curl_local, t('aiHub.copiedCurl'))">
                      {{ t('aiHub.copyCurl') }}
                    </el-button>
                  </div>
                  <pre>{{ hfStatus.api_sample_curl_local }}</pre>
                </div>
              </el-card>
            </el-card>
          </el-col>

          <el-col :xs="24" :lg="10">
            <el-card shadow="never" :header="t('aiHub.installLog')">
              <div class="install-log">
                <div v-if="!installLogs.lines?.length" class="log-empty">{{ t('aiHub.noLogs') }}</div>
                <div v-for="(line, i) in installLogs.lines" :key="i" class="log-line">{{ line }}</div>
              </div>
            </el-card>
          </el-col>
        </el-row>
      </el-tab-pane>

      <el-tab-pane :label="t('aiHub.tabAgents')" name="agents">
        <div class="tab-toolbar">
          <el-button type="primary" @click="goSoftware">{{ t('aiHub.installFromStore') }}</el-button>
        </div>
        <el-table :data="agents" stripe>
          <el-table-column prop="name" :label="t('aiHub.colName')" min-width="140" />
          <el-table-column prop="key" :label="t('aiHub.colKey')" min-width="120" />
          <el-table-column prop="status" :label="t('aiHub.colStatus')" width="100">
            <template #default="{ row }">
              <el-tag :type="row.status === 'running' ? 'success' : 'info'" size="small">{{ row.status }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="port" :label="t('aiHub.colPort')" width="80" />
          <el-table-column prop="version" :label="t('aiHub.colVersion')" width="100" />
        </el-table>
        <el-empty v-if="!agents.length" :description="t('aiHub.noAgents')" />
      </el-tab-pane>

      <el-tab-pane :label="t('aiHub.tabModels')" name="models">
        <el-card shadow="never">
          <p class="models-hint">{{ t('aiHub.modelsHint') }}</p>
          <el-button type="primary" @click="goSettings">{{ t('aiHub.openSettings') }}</el-button>
        </el-card>
      </el-tab-pane>

      <el-tab-pane :label="t('aiHub.tabGPU')" name="gpu">
        <el-card shadow="never">
          <el-result v-if="!gpuInfo.available" icon="warning" :title="t('aiHub.gpuNotFound')" :sub-title="gpuInfo.message" />
          <template v-else>
            <el-alert type="success" :closable="false" show-icon :title="gpuInfo.message" class="mb-16" />
            <el-table :data="gpuInfo.devices?.map((d: string) => ({ device: d }))" stripe>
              <el-table-column prop="device" :label="t('aiHub.gpuDevice')" />
            </el-table>
          </template>
        </el-card>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<style scoped>
.ai-hub-view { padding: 4px 0; }
.card-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.mb-16 { margin-bottom: 16px; }
.mt-16 { margin-top: 16px; }
.tab-toolbar { margin-bottom: 12px; }
.section-title { margin: 0 0 12px; font-size: 14px; color: var(--el-text-color-primary); }
.catalog-section { margin-bottom: 20px; }
.hub-link { margin-left: 6px; vertical-align: baseline; }
.hf-token-bar { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; }
.token-input { flex: 1; min-width: 200px; }
.hub-toolbar { display: flex; flex-wrap: wrap; gap: 8px; margin-bottom: 12px; }
.hub-toolbar .el-input { flex: 1; min-width: 180px; }
.hub-grid { min-height: 120px; }
.model-card.disabled { opacity: 0.72; cursor: not-allowed; }
.model-title-text { display: flex; flex-direction: column; min-width: 0; }
.model-author { font-size: 11px; color: var(--el-text-color-secondary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.card-badges { display: flex; flex-direction: column; gap: 4px; align-items: flex-end; }
.brand-card { cursor: pointer; }
.brand-variant-row { margin-bottom: 10px; }
.variant-label { display: block; font-size: 12px; color: var(--el-text-color-secondary); margin-bottom: 6px; }
.deploy-note { margin: 0 0 8px; font-size: 12px; line-height: 1.5; color: var(--el-text-color-secondary); }
.hub-card-actions { display: flex; align-items: center; justify-content: space-between; gap: 8px; margin-top: 4px; }
.modality-bar { margin-bottom: 12px; overflow-x: auto; }
.modality-bar :deep(.el-radio-group) { flex-wrap: wrap; }
.model-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 12px;
}
.model-card {
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  padding: 12px;
  cursor: pointer;
  transition: border-color 0.2s, box-shadow 0.2s;
}
.model-card:hover, .model-card.active {
  border-color: var(--el-color-primary);
  box-shadow: 0 0 0 1px var(--el-color-primary-light-7);
}
.model-card-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 8px; margin-bottom: 8px; }
.model-title-row { display: flex; align-items: center; gap: 10px; min-width: 0; }
.model-title-row strong { line-height: 1.3; }
.selected-model-row { display: flex; align-items: center; gap: 10px; width: 100%; }
.selected-model-row .el-input { flex: 1; }
.model-meta, .model-hints, .model-tags { display: flex; flex-wrap: wrap; gap: 6px; align-items: center; margin-bottom: 8px; font-size: 12px; color: var(--el-text-color-secondary); }
.model-desc { margin: 0 0 10px; font-size: 12px; line-height: 1.6; color: var(--el-text-color-secondary); min-height: 38px; }
.runtime-hint, .field-hint { margin-top: 6px; font-size: 12px; color: var(--el-text-color-secondary); }
.api-section :deep(.el-card__header) { font-weight: 600; }
.mono { font-family: ui-monospace, monospace; }
.curl-block { margin-top: 12px; }
.curl-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 8px; font-size: 13px; font-weight: 600; }
.curl-block pre {
  margin: 0;
  padding: 12px;
  border-radius: 6px;
  background: var(--el-fill-color-light);
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
}
.install-log {
  max-height: 420px;
  overflow: auto;
  font-family: ui-monospace, monospace;
  font-size: 12px;
  line-height: 1.6;
  background: var(--el-fill-color-light);
  border-radius: 6px;
  padding: 12px;
}
.log-line { white-space: pre-wrap; word-break: break-all; }
.log-empty { color: var(--el-text-color-secondary); }
.models-hint { margin: 0 0 16px; color: var(--el-text-color-secondary); }
.tutorial-collapse :deep(.el-collapse-item__header) { font-weight: 600; }
.tutorial-steps { margin: 0 0 12px; padding-left: 20px; line-height: 1.8; }
.tutorial-notes { color: var(--el-text-color-secondary); font-size: 13px; line-height: 1.7; }
.tutorial-notes ul { margin: 4px 0 12px; padding-left: 20px; }
.tutorial-notes p { margin: 8px 0 4px; color: var(--el-text-color-primary); }
</style>
