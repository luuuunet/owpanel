<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '@/api'
import { isChineseLocale } from '@/locales'

export interface CountryInfo {
  code: string
  name: string
  zh?: string
  count?: number
  bytes?: number
  percent?: number
}

const props = defineProps<{
  visible: boolean
  country: CountryInfo | null
  hours: number
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
}>()

const { t, locale } = useI18n()
const router = useRouter()

const loading = ref(false)
const detailLoading = ref(false)
const applying = ref(false)
const websites = ref<{ id: number; domain: string }[]>([])
const countryDomains = ref<{ host: string; website_id?: number; count: number; bytes: number }[]>([])
const selectedHost = ref('')
const selectedWebsiteId = ref<number | null>(null)
const details = ref<any>(null)
const policies = ref<any[]>([])
const redirectUrl = ref('')

const drawerVisible = computed({
  get: () => props.visible,
  set: (v) => emit('update:visible', v),
})

const countryLabel = computed(() => {
  if (!props.country) return ''
  return isChineseLocale(locale.value) ? (props.country.zh || props.country.name) : props.country.name
})

const domainOptions = computed(() => {
  const seen = new Set<string>()
  const opts: { label: string; value: string; websiteId?: number; pv?: number }[] = []

  for (const w of websites.value) {
    if (!seen.has(w.domain)) {
      seen.add(w.domain)
      opts.push({ label: w.domain, value: w.domain, websiteId: w.id })
    }
  }
  for (const d of countryDomains.value) {
    if (!seen.has(d.host)) {
      seen.add(d.host)
      opts.push({
        label: `${d.host} (${formatNum(d.count)} PV)`,
        value: d.host,
        websiteId: d.website_id,
        pv: d.count,
      })
    } else {
      const existing = opts.find(o => o.value === d.host)
      if (existing && !existing.websiteId && d.website_id) {
        existing.websiteId = d.website_id
      }
    }
  }
  return opts.sort((a, b) => a.label.localeCompare(b.label))
})

function formatNum(n: number) {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M'
  if (n >= 1_000) return (n / 1_000).toFixed(1) + 'K'
  return String(n)
}

function formatBytes(bytes: number) {
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`
}

async function loadBase() {
  if (!props.country?.code) return
  loading.value = true
  try {
    const [wsRes, domRes]: any[] = await Promise.all([
      api.get('/analytics/traffic-map/websites'),
      api.get(`/analytics/traffic-map/countries/${props.country.code}/domains`, { params: { hours: props.hours } }),
    ])
    websites.value = wsRes.data || []
    countryDomains.value = domRes.data || []
  } finally {
    loading.value = false
  }
}

async function loadDetails() {
  if (!props.country?.code || !selectedHost.value) {
    details.value = null
    return
  }
  detailLoading.value = true
  try {
    const res: any = await api.get(
      `/analytics/traffic-map/countries/${props.country.code}/domains/${encodeURIComponent(selectedHost.value)}/details`,
      { params: { hours: props.hours } },
    )
    details.value = res.data
  } finally {
    detailLoading.value = false
  }
}

async function loadPolicies() {
  if (!selectedWebsiteId.value) {
    policies.value = []
    return
  }
  const res: any = await api.get('/analytics/geo-policies', { params: { website_id: selectedWebsiteId.value } })
  policies.value = res.data || []
}

function onDomainChange(host: string) {
  selectedHost.value = host
  const opt = domainOptions.value.find(o => o.value === host)
  selectedWebsiteId.value = opt?.websiteId ?? websites.value.find(w => w.domain === host)?.id ?? null
  loadDetails()
  loadPolicies()
}

async function createPolicy(action: 'block' | 'redirect') {
  if (!props.country?.code || !selectedWebsiteId.value) {
    ElMessage.warning(t('traffic.geo.selectDomainFirst'))
    return
  }
  const body: any = {
    website_id: selectedWebsiteId.value,
    country_code: props.country.code,
    country_name: props.country.name,
    action,
    enabled: true,
  }
  if (action === 'redirect') {
    const url = redirectUrl.value.trim()
    if (!url) {
      ElMessage.warning(t('traffic.geo.redirectRequired'))
      return
    }
    body.redirect_url = url
  }
  try {
    await api.post('/analytics/geo-policies', body)
    ElMessage.success(t('traffic.geo.policyCreated'))
    await loadPolicies()
  } catch (e: any) {
    ElMessage.error(e?.error || e?.message || t('traffic.geo.policyFailed'))
  }
}

async function applyPolicies() {
  if (!selectedWebsiteId.value) return
  applying.value = true
  try {
    await api.post(`/analytics/geo-policies/apply/${selectedWebsiteId.value}`)
    ElMessage.success(t('traffic.geo.applied'))
  } catch (e: any) {
    ElMessage.error(e?.error || e?.message || t('traffic.geo.applyFailed'))
  } finally {
    applying.value = false
  }
}

async function blockAndApply() {
  await createPolicy('block')
  await applyPolicies()
}

async function redirectAndApply() {
  await createPolicy('redirect')
  await applyPolicies()
}

async function togglePolicy(row: any) {
  await api.put(`/analytics/geo-policies/${row.id}`, { enabled: !row.enabled })
  await loadPolicies()
}

async function deletePolicy(row: any) {
  await ElMessageBox.confirm(t('traffic.geo.deleteConfirm'), { type: 'warning' })
  await api.delete(`/analytics/geo-policies/${row.id}`)
  ElMessage.success(t('traffic.geo.deleted'))
  await loadPolicies()
}

function goWafGeo() {
  router.push({ path: '/protection', query: { tab: 'waf' } })
}

const countryPolicies = computed(() =>
  policies.value.filter(p => p.country_code === props.country?.code),
)

watch(
  () => [props.visible, props.country?.code, props.hours] as const,
  ([vis, code]) => {
    if (vis && code) {
      selectedHost.value = ''
      selectedWebsiteId.value = null
      details.value = null
      policies.value = []
      redirectUrl.value = ''
      loadBase()
    }
  },
  { immediate: true },
)
</script>

<template>
  <el-drawer
    v-model="drawerVisible"
    :title="t('traffic.geo.drawerTitle', { country: countryLabel })"
    size="520px"
    destroy-on-close
    append-to-body
    class="traffic-geo-drawer"
  >
    <div v-if="country" v-loading="loading" class="geo-body">
      <div class="country-stats">
        <div class="stat">
          <span class="label">{{ t('traffic.pageViews') }}</span>
          <span class="value">{{ formatNum(country.count || 0) }}</span>
        </div>
        <div v-if="country.bytes" class="stat">
          <span class="label">{{ t('traffic.bandwidth') }}</span>
          <span class="value">{{ formatBytes(country.bytes) }}</span>
        </div>
        <div v-if="country.percent != null" class="stat">
          <span class="label">{{ t('traffic.share') }}</span>
          <span class="value">{{ country.percent }}%</span>
        </div>
      </div>

      <div class="section">
        <div class="section-title">{{ t('traffic.geo.selectDomain') }}</div>
        <el-select
          v-model="selectedHost"
          filterable
          clearable
          :placeholder="t('traffic.geo.domainPlaceholder')"
          style="width: 100%"
          @change="onDomainChange"
        >
          <el-option
            v-for="opt in domainOptions"
            :key="opt.value"
            :label="opt.label"
            :value="opt.value"
          />
        </el-select>
      </div>

      <template v-if="selectedHost">
        <div v-loading="detailLoading" class="section">
          <div class="section-title">{{ t('traffic.geo.accessDetails') }}</div>
          <div v-if="details" class="detail-summary">
            <span>{{ t('traffic.pageViews') }}: {{ formatNum(details.total_pv || 0) }}</span>
            <span>{{ t('traffic.bandwidth') }}: {{ formatBytes(details.total_bytes || 0) }}</span>
          </div>

          <div v-if="details?.top_paths?.length" class="detail-block">
            <div class="block-title">{{ t('traffic.geo.topPaths') }}</div>
            <el-table :data="details.top_paths.slice(0, 10)" size="small" stripe>
              <el-table-column prop="path" :label="t('traffic.geo.path')" min-width="160" show-overflow-tooltip />
              <el-table-column prop="count" :label="t('traffic.pageViews')" width="80" />
              <el-table-column :label="t('traffic.bandwidth')" width="90">
                <template #default="{ row }">{{ formatBytes(row.bytes) }}</template>
              </el-table-column>
            </el-table>
          </div>

          <div v-if="details?.top_referers?.length" class="detail-block">
            <div class="block-title">{{ t('traffic.geo.topReferers') }}</div>
            <el-table :data="details.top_referers.slice(0, 10)" size="small" stripe>
              <el-table-column prop="host" :label="t('traffic.geo.refererHost')" min-width="160" show-overflow-tooltip />
              <el-table-column prop="count" :label="t('traffic.pageViews')" width="80" />
            </el-table>
          </div>

          <div v-if="details?.top_ips?.length" class="detail-block">
            <div class="block-title">{{ t('traffic.geo.topIPs') }}</div>
            <el-table :data="details.top_ips.slice(0, 10)" size="small" stripe>
              <el-table-column prop="ip" label="IP" width="130" />
              <el-table-column prop="count" :label="t('traffic.pageViews')" width="80" />
            </el-table>
          </div>
        </div>

        <div class="section actions-panel">
          <div class="section-title">{{ t('traffic.geo.actions') }}</div>
          <p class="hint">{{ t('traffic.geo.actionsHint') }}</p>

          <div class="action-row">
            <el-button
              type="danger"
              :disabled="!selectedWebsiteId"
              :loading="applying"
              @click="blockAndApply"
            >
              {{ t('traffic.geo.blockAccess') }}
            </el-button>
          </div>

          <div class="action-row redirect-row">
            <el-input
              v-model="redirectUrl"
              :placeholder="t('traffic.geo.redirectPlaceholder')"
              :disabled="!selectedWebsiteId"
            />
            <el-button
              type="primary"
              :disabled="!selectedWebsiteId"
              :loading="applying"
              @click="redirectAndApply"
            >
              {{ t('traffic.geo.redirect301') }}
            </el-button>
          </div>

          <div v-if="countryPolicies.length" class="policy-list">
            <div class="block-title">{{ t('traffic.geo.existingPolicies') }}</div>
            <div v-for="p in countryPolicies" :key="p.id" class="policy-row">
              <el-tag :type="p.action === 'block' ? 'danger' : 'warning'" size="small">
                {{ p.action === 'block' ? t('traffic.geo.blockAccess') : t('traffic.geo.redirect301') }}
              </el-tag>
              <span class="policy-url" v-if="p.redirect_url">{{ p.redirect_url }}</span>
              <el-switch :model-value="p.enabled" size="small" @change="togglePolicy(p)" />
              <el-button link type="danger" size="small" @click="deletePolicy(p)">
                {{ t('common.delete') }}
              </el-button>
            </div>
            <el-button size="small" :loading="applying" @click="applyPolicies">
              {{ t('traffic.geo.reapply') }}
            </el-button>
          </div>

          <el-link type="primary" @click="goWafGeo">{{ t('traffic.geo.wafGeoLink') }}</el-link>
        </div>
      </template>
    </div>
  </el-drawer>
</template>

<style scoped>
.geo-body { display: flex; flex-direction: column; gap: 16px; }
.country-stats {
  display: flex;
  gap: 20px;
  flex-wrap: wrap;
  padding: 12px;
  background: var(--el-fill-color-light);
  border-radius: 8px;
}
.stat { display: flex; flex-direction: column; gap: 2px; }
.stat .label { font-size: 12px; color: var(--el-text-color-secondary); }
.stat .value { font-size: 18px; font-weight: 700; }
.section-title { font-weight: 600; margin-bottom: 8px; font-size: 14px; }
.detail-summary {
  display: flex;
  gap: 16px;
  font-size: 13px;
  margin-bottom: 12px;
  color: var(--el-text-color-secondary);
}
.detail-block { margin-bottom: 12px; }
.block-title { font-size: 13px; font-weight: 600; margin-bottom: 6px; }
.actions-panel .hint { font-size: 12px; color: var(--el-text-color-secondary); margin: 0 0 12px; }
.action-row { margin-bottom: 10px; }
.redirect-row { display: flex; gap: 8px; }
.policy-list { margin-top: 12px; padding-top: 12px; border-top: 1px solid var(--el-border-color-lighter); }
.policy-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  flex-wrap: wrap;
}
.policy-url { font-size: 12px; color: var(--el-text-color-secondary); flex: 1; overflow: hidden; text-overflow: ellipsis; }
</style>
