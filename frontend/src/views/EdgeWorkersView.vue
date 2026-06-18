<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { isChineseLocale } from '@/locales'
import api from '@/api'
import { ElMessage, ElMessageBox } from 'element-plus'
import FileCodeEditor from '@/components/FileCodeEditor.vue'

withDefaults(defineProps<{ embedded?: boolean }>(), { embedded: false })

const { t, locale } = useI18n()

const activeTab = ref('workers')
const guideOpen = ref(['guide'])
const loading = ref(false)
const applying = ref(false)
const preview = ref('')
const workers = ref<any[]>([])
const websites = ref<any[]>([])
const availableDomains = ref<any[]>([])
const templates = ref<any[]>([])
const runtime = ref<any>({})
const kvNamespaces = ref<any[]>([])
const d1Databases = ref<any[]>([])
const ossStorages = ref<any[]>([])
const dialogVisible = ref(false)
const templatePickerVisible = ref(false)
const editingId = ref<number | null>(null)

const kvDialogVisible = ref(false)
const kvEditingId = ref<number | null>(null)
const kvForm = reactive({ name: '', description: '' })
const selectedNsId = ref<number | null>(null)
const kvKeys = ref<any[]>([])
const kvKeyForm = reactive({ key: '', value: '', ttl: 0 })

const d1DialogVisible = ref(false)
const d1Form = reactive({ name: '', description: '' })
const selectedD1Id = ref<number | null>(null)
const d1Sql = ref('SELECT 1')
const d1Result = ref<any>(null)

const form = reactive({
  name: '',
  description: '',
  route_pattern: '/',
  script_type: 'lua',
  script: '',
  website_id: 0,
  domains_list: [] as string[],
  enabled: true,
  priority: 100,
  triggers: 'request',
  bindings: [] as any[],
})

const scriptPath = computed(() => {
  if (form.script_type === 'lua') return 'worker.lua'
  if (form.script_type === 'njs') return 'worker.js'
  return 'worker.conf'
})

const runtimeBannerType = computed(() => {
  if (runtime.value?.lua_available) return 'success'
  if (runtime.value?.njs_available) return 'warning'
  return 'error'
})

function resetForm() {
  editingId.value = null
  Object.assign(form, {
    name: '',
    description: '',
    route_pattern: '/',
    script_type: 'lua',
    script: '',
    website_id: 0,
    domains_list: [],
    enabled: true,
    priority: 100,
    triggers: 'request',
    bindings: [],
  })
}

function siteLabel(id: number) {
  if (!id) return t('edgeWorkers.allSites')
  const w = websites.value.find((s) => s.id === id)
  return w?.domain || `#${id}`
}

function domainsLabel(row: any) {
  const list = row.domains_list || []
  if (list.includes('*')) return t('edgeWorkers.domainAll')
  if (list.length) return list.join(', ')
  if (row.domains) return row.domains
  return siteLabel(row.website_id)
}

const domainGroups = computed(() => {
  const map = new Map<number, { website_id: number; label: string; domains: any[] }>()
  for (const d of availableDomains.value) {
    if (!map.has(d.website_id)) {
      const site = websites.value.find((s) => s.id === d.website_id)
      map.set(d.website_id, {
        website_id: d.website_id,
        label: site?.domain || `#${d.website_id}`,
        domains: [],
      })
    }
    map.get(d.website_id)!.domains.push(d)
  }
  return [...map.values()]
})

const routePreviews = computed(() => {
  const route = form.route_pattern || '/'
  if (!form.domains_list.length) return []
  return form.domains_list.map((d) => `${d}${route.startsWith('/') ? route : '/' + route}`)
})

const routeExamples = computed(() => [
  { pattern: '/', desc: t('edgeWorkers.guideRouteAll'), example: 'example.com/' },
  { pattern: '/api/', desc: t('edgeWorkers.guideRoutePrefix'), example: 'example.com/api/users' },
  { pattern: '~ ^/old-path', desc: t('edgeWorkers.guideRouteRegex'), example: 'example.com/old-path/x' },
  { pattern: '^~ /blog/', desc: t('edgeWorkers.guideRouteRewrite'), example: 'example.com/blog/post' },
])

function onWebsitePick(id: number) {
  form.website_id = id
  if (!id) return
  const siteDomains = availableDomains.value
    .filter((d) => d.website_id === id)
    .map((d) => d.domain)
  form.domains_list = [...new Set([...form.domains_list, ...siteDomains])]
}

function normalizeDomainInput(raw: string) {
  return raw.trim().toLowerCase().replace(/^https?:\/\//, '').split('/')[0].split(':')[0]
}

function onDomainChange(vals: string[]) {
  form.domains_list = [...new Set(vals.map(normalizeDomainInput).filter(Boolean))]
}

function templateName(tpl: any) {
  return isChineseLocale(locale.value) ? tpl.name_zh || tpl.name : tpl.name
}

function templateDesc(tpl: any) {
  return isChineseLocale(locale.value) ? tpl.description_zh || tpl.description : tpl.description
}

function addBinding() {
  form.bindings.push({ binding_type: 'kv', binding_name: 'MY_KV', resource_id: null, resource_key: '' })
}

function removeBinding(idx: number) {
  form.bindings.splice(idx, 1)
}

async function loadKV() {
  const res: any = await api.get('/edge-workers/kv/namespaces')
  kvNamespaces.value = res.data || []
  if (selectedNsId.value) await loadKVKeys()
}

async function loadKVKeys() {
  if (!selectedNsId.value) return
  const res: any = await api.get(`/edge-workers/kv/namespaces/${selectedNsId.value}/keys`)
  kvKeys.value = res.data || []
}

async function loadD1() {
  const res: any = await api.get('/edge-workers/d1/databases')
  d1Databases.value = res.data || []
}

async function loadAll() {
  loading.value = true
  try {
    const [listRes, rtRes, tplRes, sitesRes, domainsRes, ossRes]: any[] = await Promise.all([
      api.get('/edge-workers'),
      api.get('/edge-workers/runtime'),
      api.get('/edge-workers/templates'),
      api.get('/websites'),
      api.get('/edge-workers/available-domains'),
      api.get('/oss/storages').catch(() => ({ data: [] })),
    ])
    workers.value = listRes.data || []
    runtime.value = rtRes.data || {}
    templates.value = tplRes.data || []
    websites.value = sitesRes.data || []
    availableDomains.value = domainsRes.data || []
    ossStorages.value = ossRes.data || []
    await Promise.all([loadKV(), loadD1()])
  } finally {
    loading.value = false
  }
}

function openCreate() {
  resetForm()
  dialogVisible.value = true
}

function openEdit(row: any) {
  editingId.value = row.id
  Object.assign(form, {
    name: row.name,
    description: row.description || '',
    route_pattern: row.route_pattern || '/',
    script_type: row.script_type || 'lua',
    script: row.script || '',
    website_id: row.website_id || 0,
    domains_list: row.domains_list?.length ? [...row.domains_list] : (row.domains ? row.domains.split(',').map((d: string) => d.trim()).filter(Boolean) : []),
    enabled: row.enabled !== false,
    priority: row.priority || 100,
    triggers: row.triggers || 'request',
    bindings: (row.bindings || []).map((b: any) => ({
      binding_type: b.binding_type,
      binding_name: b.binding_name,
      resource_id: b.resource_id || null,
      resource_key: b.resource_key || '',
    })),
  })
  dialogVisible.value = true
}

function applyTemplate(tpl: any) {
  Object.assign(form, {
    name: templateName(tpl),
    description: templateDesc(tpl),
    route_pattern: tpl.route_pattern,
    script_type: tpl.script_type,
    script: tpl.script,
    triggers: tpl.triggers,
    bindings: tpl.id === 'counter-kv' ? [{ binding_type: 'kv', binding_name: 'MY_KV', resource_id: null, resource_key: '' }] : [],
  })
  templatePickerVisible.value = false
  dialogVisible.value = true
}

async function saveWorker() {
  if (!form.domains_list.length && !form.website_id) {
    ElMessage.error(t('edgeWorkers.domainRequired'))
    return
  }
  const payload = {
    ...form,
    domains: form.domains_list.join(','),
    bindings: form.bindings,
  }
  delete (payload as any).domains_list
  try {
    if (editingId.value) {
      await api.patch(`/edge-workers/${editingId.value}`, payload)
    } else {
      await api.post('/edge-workers', payload)
    }
    ElMessage.success(t('common.success'))
    dialogVisible.value = false
    await loadAll()
  } catch (e: any) {
    ElMessage.error(e?.error || e?.message || t('common.failed'))
  }
}

async function toggleWorker(row: any) {
  await api.patch(`/edge-workers/${row.id}/toggle`, { enabled: !row.enabled })
  await loadAll()
}

async function deleteWorker(row: any) {
  await ElMessageBox.confirm(t('common.confirmDelete'), t('common.warning'), { type: 'warning' })
  await api.delete(`/edge-workers/${row.id}`)
  ElMessage.success(t('common.success'))
  await loadAll()
}

async function applyWorkers() {
  applying.value = true
  try {
    const res: any = await api.post('/edge-workers/apply')
    ElMessage.success(res.data?.message || res.message || t('edgeWorkers.applied'))
    if (res.data?.preview) preview.value = res.data.preview
  } catch (e: any) {
    ElMessage.error(e?.error || t('edgeWorkers.applyFailed'))
  } finally {
    applying.value = false
  }
}

async function loadPreview() {
  const res: any = await api.get('/edge-workers/preview')
  preview.value = res.data?.preview || ''
}

function openKVCreate() {
  kvEditingId.value = null
  Object.assign(kvForm, { name: '', description: '' })
  kvDialogVisible.value = true
}

async function saveKVNamespace() {
  try {
    if (kvEditingId.value) {
      await api.patch(`/edge-workers/kv/namespaces/${kvEditingId.value}`, kvForm)
    } else {
      await api.post('/edge-workers/kv/namespaces', kvForm)
    }
    kvDialogVisible.value = false
    await loadKV()
  } catch (e: any) {
    ElMessage.error(e?.error || t('common.failed'))
  }
}

async function deleteKVNamespace(row: any) {
  await ElMessageBox.confirm(t('common.confirmDelete'), t('common.warning'), { type: 'warning' })
  await api.delete(`/edge-workers/kv/namespaces/${row.id}`)
  if (selectedNsId.value === row.id) selectedNsId.value = null
  await loadKV()
}

async function saveKVKey() {
  if (!selectedNsId.value || !kvKeyForm.key) return
  await api.put(`/edge-workers/kv/namespaces/${selectedNsId.value}/keys/${encodeURIComponent(kvKeyForm.key)}`, {
    value: kvKeyForm.value,
    ttl: kvKeyForm.ttl || 0,
  })
  kvKeyForm.key = ''
  kvKeyForm.value = ''
  await loadKVKeys()
}

async function deleteKVKey(row: any) {
  if (!selectedNsId.value) return
  await api.delete(`/edge-workers/kv/namespaces/${selectedNsId.value}/keys/${encodeURIComponent(row.key)}`)
  await loadKVKeys()
}

async function exportKV() {
  if (!selectedNsId.value) return
  const res: any = await api.get(`/edge-workers/kv/namespaces/${selectedNsId.value}/export`)
  const blob = new Blob([JSON.stringify(res.data, null, 2)], { type: 'application/json' })
  const a = document.createElement('a')
  a.href = URL.createObjectURL(blob)
  a.download = `kv-ns-${selectedNsId.value}.json`
  a.click()
}

function openD1Create() {
  Object.assign(d1Form, { name: '', description: '' })
  d1DialogVisible.value = true
}

async function saveD1Database() {
  await api.post('/edge-workers/d1/databases', d1Form)
  d1DialogVisible.value = false
  await loadD1()
}

async function deleteD1Database(row: any) {
  await ElMessageBox.confirm(t('common.confirmDelete'), t('common.warning'), { type: 'warning' })
  await api.delete(`/edge-workers/d1/databases/${row.id}`)
  if (selectedD1Id.value === row.id) selectedD1Id.value = null
  await loadD1()
}

async function runD1Query() {
  if (!selectedD1Id.value) return
  try {
    const res: any = await api.post(`/edge-workers/d1/databases/${selectedD1Id.value}/query`, { sql: d1Sql.value })
    d1Result.value = res.data
  } catch (e: any) {
    ElMessage.error(e?.error || t('common.failed'))
  }
}

onMounted(loadAll)
</script>

<template>
  <div class="edge-workers" v-loading="loading">
    <el-alert
      :title="t('edgeWorkers.runtimeTitle')"
      :type="runtimeBannerType"
      show-icon
      :closable="false"
      class="runtime-banner"
    >
      <template #default>
        <p>{{ t('edgeWorkers.runtimeHint') }}</p>
        <p>
          <strong>{{ t('edgeWorkers.runtimeActive') }}:</strong>
          {{ runtime.runtime || '—' }}
          · Lua: {{ runtime.lua_available ? '✓' : '✗' }}
          · njs: {{ runtime.njs_available ? '✓' : '✗' }}
        </p>
        <p v-if="runtime.message" class="runtime-msg">{{ runtime.message }}</p>
      </template>
    </el-alert>

    <el-tabs v-model="activeTab" class="main-tabs">
      <el-tab-pane :label="t('edgeWorkers.tabWorkers')" name="workers">
        <el-collapse v-model="guideOpen" class="worker-guide">
          <el-collapse-item :title="t('edgeWorkers.guideTitle')" name="guide">
            <p class="guide-intro">{{ t('edgeWorkers.guideIntro') }}</p>
            <ol class="guide-steps">
              <li v-for="i in 6" :key="i">{{ t(`edgeWorkers.guideStep${i}`) }}</li>
            </ol>

            <h4 class="guide-subtitle">{{ t('edgeWorkers.guideRoutesTitle') }}</h4>
            <el-table :data="routeExamples" size="small" stripe class="guide-table">
              <el-table-column prop="pattern" :label="t('edgeWorkers.route')" width="140">
                <template #default="{ row }"><code>{{ row.pattern }}</code></template>
              </el-table-column>
              <el-table-column prop="desc" :label="t('edgeWorkers.guideRouteDesc')" min-width="160" />
              <el-table-column prop="example" :label="t('edgeWorkers.guideRouteExample')" min-width="180">
                <template #default="{ row }"><code>{{ row.example }}</code></template>
              </el-table-column>
            </el-table>

            <h4 class="guide-subtitle">{{ t('edgeWorkers.guideBindingsTitle') }}</h4>
            <p class="guide-intro">{{ t('edgeWorkers.guideBindingsIntro') }}</p>
            <ul class="guide-list">
              <li><strong>KV</strong> — {{ t('edgeWorkers.guideBindingKv') }}</li>
              <li><strong>D1</strong> — {{ t('edgeWorkers.guideBindingD1') }}</li>
              <li><strong>Redis</strong> — {{ t('edgeWorkers.guideBindingRedis') }}</li>
            </ul>

            <h4 class="guide-subtitle">{{ t('edgeWorkers.guideTemplatesTitle') }}</h4>
            <p class="guide-intro">{{ t('edgeWorkers.guideTemplatesIntro') }}</p>
            <el-table :data="templates" size="small" stripe class="guide-table">
              <el-table-column :label="t('common.name')" min-width="140">
                <template #default="{ row }">{{ templateName(row) }}</template>
              </el-table-column>
              <el-table-column prop="route_pattern" :label="t('edgeWorkers.route')" width="120">
                <template #default="{ row }"><code>{{ row.route_pattern }}</code></template>
              </el-table-column>
              <el-table-column prop="script_type" :label="t('edgeWorkers.type')" width="90" />
              <el-table-column :label="t('common.description')" min-width="180" show-overflow-tooltip>
                <template #default="{ row }">{{ templateDesc(row) }}</template>
              </el-table-column>
              <el-table-column :label="t('common.actions')" width="100">
                <template #default="{ row }">
                  <el-button text type="primary" size="small" @click="applyTemplate(row)">{{ t('edgeWorkers.guideUseTemplate') }}</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-collapse-item>
        </el-collapse>

        <div class="toolbar">
          <el-button type="primary" @click="openCreate">{{ t('edgeWorkers.create') }}</el-button>
          <el-button @click="templatePickerVisible = true">{{ t('edgeWorkers.fromTemplate') }}</el-button>
          <el-button @click="loadPreview">{{ t('edgeWorkers.preview') }}</el-button>
          <el-button type="success" :loading="applying" @click="applyWorkers">{{ t('edgeWorkers.deploy') }}</el-button>
        </div>

        <el-table :data="workers" stripe empty-text="—" style="width: 100%">
          <el-table-column prop="name" :label="t('edgeWorkers.name')" min-width="140" />
          <el-table-column prop="route_pattern" :label="t('edgeWorkers.route')" min-width="120" />
          <el-table-column :label="t('edgeWorkers.domains')" min-width="180" show-overflow-tooltip>
            <template #default="{ row }">{{ domainsLabel(row) }}</template>
          </el-table-column>
          <el-table-column prop="script_type" :label="t('edgeWorkers.type')" width="90" />
          <el-table-column :label="t('edgeWorkers.bindings')" width="90">
            <template #default="{ row }">{{ (row.bindings || []).length }}</template>
          </el-table-column>
          <el-table-column :label="t('common.enabled')" width="90">
            <template #default="{ row }">
              <el-switch :model-value="row.enabled" @change="toggleWorker(row)" />
            </template>
          </el-table-column>
          <el-table-column :label="t('common.actions')" width="160" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" @click="openEdit(row)">{{ t('common.edit') }}</el-button>
              <el-button link type="danger" @click="deleteWorker(row)">{{ t('common.delete') }}</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane :label="t('edgeWorkers.tabKV')" name="kv">
        <el-alert type="info" :closable="false" show-icon class="tab-guide" :title="t('edgeWorkers.guideKvTitle')">
          <template #default>
            <ol class="guide-steps compact">
              <li v-for="i in 4" :key="i">{{ t(`edgeWorkers.guideKvStep${i}`) }}</li>
            </ol>
          </template>
        </el-alert>
        <div class="toolbar">
          <el-button type="primary" @click="openKVCreate">{{ t('edgeWorkers.kvCreate') }}</el-button>
          <el-button :disabled="!selectedNsId" @click="exportKV">{{ t('edgeWorkers.kvExport') }}</el-button>
        </div>
        <el-row :gutter="16">
          <el-col :span="8">
            <el-table :data="kvNamespaces" highlight-current-row @current-change="(r: any) => { selectedNsId = r?.id; loadKVKeys() }">
              <el-table-column prop="name" :label="t('edgeWorkers.kvNamespace')" />
              <el-table-column width="80">
                <template #default="{ row }">
                  <el-button link type="danger" @click.stop="deleteKVNamespace(row)">{{ t('common.delete') }}</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-col>
          <el-col :span="16">
            <div v-if="selectedNsId" class="kv-editor">
              <el-form inline>
                <el-form-item :label="t('edgeWorkers.kvKey')"><el-input v-model="kvKeyForm.key" /></el-form-item>
                <el-form-item :label="t('edgeWorkers.kvValue')"><el-input v-model="kvKeyForm.value" /></el-form-item>
                <el-button type="primary" @click="saveKVKey">{{ t('common.save') }}</el-button>
              </el-form>
              <el-table :data="kvKeys" size="small" stripe>
                <el-table-column prop="key" label="Key" />
                <el-table-column prop="value" label="Value" show-overflow-tooltip />
                <el-table-column width="80">
                  <template #default="{ row }">
                    <el-button link type="danger" @click="deleteKVKey(row)">{{ t('common.delete') }}</el-button>
                  </template>
                </el-table-column>
              </el-table>
            </div>
            <el-empty v-else :description="t('edgeWorkers.kvNamespace')" />
          </el-col>
        </el-row>
      </el-tab-pane>

      <el-tab-pane :label="t('edgeWorkers.tabD1')" name="d1">
        <el-alert type="info" :closable="false" show-icon class="tab-guide" :title="t('edgeWorkers.guideD1Title')">
          <template #default>
            <ol class="guide-steps compact">
              <li v-for="i in 4" :key="i">{{ t(`edgeWorkers.guideD1Step${i}`) }}</li>
            </ol>
          </template>
        </el-alert>
        <div class="toolbar">
          <el-button type="primary" @click="openD1Create">{{ t('edgeWorkers.d1Create') }}</el-button>
        </div>
        <el-row :gutter="16">
          <el-col :span="8">
            <el-table :data="d1Databases" highlight-current-row @current-change="(r: any) => { selectedD1Id = r?.id; d1Result = null }">
              <el-table-column prop="name" label="Name" />
              <el-table-column width="80">
                <template #default="{ row }">
                  <el-button link type="danger" @click.stop="deleteD1Database(row)">{{ t('common.delete') }}</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-col>
          <el-col :span="16">
            <div v-if="selectedD1Id">
              <p class="section-label">{{ t('edgeWorkers.d1Query') }}</p>
              <FileCodeEditor v-model="d1Sql" path="query.sql" style="min-height: 120px" />
              <el-button type="primary" class="mt-8" @click="runD1Query">{{ t('edgeWorkers.d1Run') }}</el-button>
              <pre v-if="d1Result" class="preview-code">{{ JSON.stringify(d1Result, null, 2) }}</pre>
            </div>
            <el-empty v-else description="D1" />
          </el-col>
        </el-row>
      </el-tab-pane>
    </el-tabs>

    <el-collapse v-if="preview" class="preview-box">
      <el-collapse-item :title="t('edgeWorkers.preview')" name="preview">
        <pre class="preview-code">{{ preview }}</pre>
      </el-collapse-item>
    </el-collapse>

    <el-dialog v-model="dialogVisible" :title="editingId ? t('edgeWorkers.edit') : t('edgeWorkers.create')" width="780px" destroy-on-close>
      <el-form label-width="120px">
        <el-form-item :label="t('edgeWorkers.name')"><el-input v-model="form.name" /></el-form-item>
        <el-form-item :label="t('edgeWorkers.route')"><el-input v-model="form.route_pattern" /></el-form-item>
        <el-form-item :label="t('edgeWorkers.pickWebsite')">
          <el-select v-model="form.website_id" style="width: 100%" clearable @change="onWebsitePick">
            <el-option :value="0" :label="t('edgeWorkers.allSites')" />
            <el-option v-for="s in websites" :key="s.id" :value="s.id" :label="s.domain" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('edgeWorkers.domains')" required>
          <el-select
            v-model="form.domains_list"
            multiple
            filterable
            allow-create
            default-first-option
            :placeholder="t('edgeWorkers.customDomain')"
            style="width: 100%"
            @change="onDomainChange"
          >
            <el-option :value="'*'" :label="t('edgeWorkers.domainAll')" />
            <el-option-group v-for="g in domainGroups" :key="g.website_id" :label="g.label">
              <el-option v-for="d in g.domains" :key="d.domain" :value="d.domain" :label="d.domain + (d.is_primary ? ' (primary)' : '')" />
            </el-option-group>
          </el-select>
          <p class="hint">{{ t('edgeWorkers.domainHint') }}</p>
          <p v-if="routePreviews.length" class="hint">
            <strong>{{ t('edgeWorkers.domainPreview') }}:</strong>
            {{ routePreviews.join(' · ') }}
          </p>
        </el-form-item>
        <el-form-item :label="t('edgeWorkers.type')">
          <el-select v-model="form.script_type" style="width: 100%">
            <el-option value="lua" label="Lua (OpenResty)" />
            <el-option value="njs" label="njs" />
            <el-option value="template" label="Nginx template" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('edgeWorkers.bindings')">
          <div class="bindings-list">
            <div v-for="(b, idx) in form.bindings" :key="idx" class="binding-row">
              <el-select v-model="b.binding_type" style="width: 110px">
                <el-option value="kv" label="KV" />
                <el-option value="d1" label="D1" />
                <el-option value="redis" label="Redis" />
                <el-option value="oss" label="OSS/R2" />
              </el-select>
              <el-input v-model="b.binding_name" :placeholder="t('edgeWorkers.bindingName')" style="width: 120px" />
              <el-select v-if="b.binding_type === 'kv'" v-model="b.resource_id" :placeholder="t('edgeWorkers.kvNamespace')" style="width: 160px">
                <el-option v-for="ns in kvNamespaces" :key="ns.id" :value="ns.id" :label="ns.name" />
              </el-select>
              <el-select v-else-if="b.binding_type === 'd1'" v-model="b.resource_id" style="width: 160px">
                <el-option v-for="db in d1Databases" :key="db.id" :value="db.id" :label="db.name" />
              </el-select>
              <el-select v-else-if="b.binding_type === 'oss'" v-model="b.resource_id" style="width: 160px">
                <el-option v-for="o in ossStorages" :key="o.id" :value="o.id" :label="o.name" />
              </el-select>
              <el-input v-else-if="b.binding_type === 'redis'" v-model="b.resource_key" :placeholder="t('edgeWorkers.redisBindingHint')" style="width: 220px" />
              <el-button link type="danger" @click="removeBinding(idx)">{{ t('common.delete') }}</el-button>
            </div>
            <el-button size="small" @click="addBinding">{{ t('edgeWorkers.addBinding') }}</el-button>
            <p v-if="form.bindings.some((x: any) => x.binding_type === 'oss')" class="hint">{{ t('edgeWorkers.ossBindingHint') }}</p>
          </div>
        </el-form-item>
        <el-form-item :label="t('edgeWorkers.script')" class="script-form-item">
          <FileCodeEditor v-model="form.script" :path="scriptPath" />
        </el-form-item>
        <el-form-item :label="t('common.enabled')"><el-switch v-model="form.enabled" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="saveWorker">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="kvDialogVisible" :title="t('edgeWorkers.kvCreate')" width="480px">
      <el-form label-width="100px">
        <el-form-item :label="t('edgeWorkers.name')"><el-input v-model="kvForm.name" /></el-form-item>
        <el-form-item :label="t('edgeWorkers.description')"><el-input v-model="kvForm.description" type="textarea" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="kvDialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="saveKVNamespace">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="d1DialogVisible" :title="t('edgeWorkers.d1Create')" width="480px">
      <el-form label-width="100px">
        <el-form-item :label="t('edgeWorkers.name')"><el-input v-model="d1Form.name" /></el-form-item>
        <el-form-item :label="t('edgeWorkers.description')"><el-input v-model="d1Form.description" type="textarea" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="d1DialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="saveD1Database">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="templatePickerVisible" :title="t('edgeWorkers.fromTemplate')" width="640px">
      <div class="template-grid">
        <el-card v-for="tpl in templates" :key="tpl.id" shadow="hover" class="template-card" @click="applyTemplate(tpl)">
          <h4>{{ templateName(tpl) }}</h4>
          <p>{{ templateDesc(tpl) }}</p>
          <el-tag size="small">{{ tpl.script_type }}</el-tag>
        </el-card>
      </div>
    </el-dialog>
  </div>
</template>

<style scoped>
.edge-workers .runtime-banner { margin-bottom: 16px; }
.edge-workers .runtime-banner p { margin: 4px 0; font-size: 13px; }
.worker-guide { margin-bottom: 16px; }
.guide-intro { margin: 0 0 10px; font-size: 13px; color: var(--el-text-color-secondary); line-height: 1.6; }
.guide-steps { margin: 0 0 14px; padding-left: 20px; line-height: 1.8; font-size: 13px; }
.guide-steps.compact { margin-bottom: 0; }
.guide-subtitle { margin: 16px 0 8px; font-size: 14px; font-weight: 600; }
.guide-list { margin: 0 0 12px; padding-left: 20px; font-size: 13px; line-height: 1.7; }
.guide-table { margin-bottom: 8px; }
.guide-table code { font-size: 12px; }
.tab-guide { margin-bottom: 12px; }
.edge-workers .toolbar { display: flex; flex-wrap: wrap; gap: 8px; margin-bottom: 16px; }
.main-tabs { margin-top: 8px; }
.preview-box { margin-top: 16px; }
.preview-code {
  max-height: 360px; overflow: auto; font-size: 12px;
  background: var(--el-fill-color-light); padding: 12px; border-radius: 4px; white-space: pre-wrap;
}
.template-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(260px, 1fr)); gap: 12px; }
.template-card { cursor: pointer; }
.template-card h4 { margin: 0 0 8px; }
.template-card p { margin: 0 0 8px; font-size: 13px; color: var(--el-text-color-secondary); }
.bindings-list { width: 100%; }
.script-form-item :deep(.el-form-item__content) {
  width: 100%;
  max-width: 100%;
}
.script-form-item :deep(.code-editor) {
  min-height: 280px;
}
.binding-row { display: flex; flex-wrap: wrap; gap: 8px; margin-bottom: 8px; align-items: center; }
.hint { font-size: 12px; color: var(--el-text-color-secondary); margin-top: 8px; }
.mt-8 { margin-top: 8px; }
.section-label { font-weight: 600; margin-bottom: 8px; }
</style>
