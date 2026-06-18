<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import api, { resolveApiError } from '@/api'
import { ElMessage, ElMessageBox } from 'element-plus'
import SoftwareInstallLogDialog from '@/components/SoftwareInstallLogDialog.vue'

withDefaults(defineProps<{ embedded?: boolean }>(), { embedded: false })

interface DbRow {
  id: number
  name: string
  type: string
  host: string
  port: number
}

interface StatusInfo {
  kafka_installed: boolean
  kafka_running: boolean
  broker_reachable: boolean
  container_name: string
  bootstrap_servers: string
  hint?: string
}

const { t } = useI18n()
const loading = ref(false)
const saving = ref(false)
const applying = ref(false)
const presetLoading = ref(false)
const status = ref<StatusInfo | null>(null)
const databases = ref<DbRow[]>([])
const topics = ref<string[]>([])
const topicsHint = ref('')
const expectedTopics = ref<string[]>([])
const tutorialOpen = ref(['tutorial'])

const installDialog = ref(false)
const installAppKey = ref('kafka')
const installAppName = ref('Kafka')
const installTrigger = ref(false)

const form = reactive({
  enabled: false,
  bootstrap_servers: '127.0.0.1:9092',
  topic_prefix: 'opanel.db',
  mode: 'write_async',
  consumer_group: 'opanel-db-accel',
  linked_database_ids: [] as number[],
  topic_partitions: 3,
  replication_factor: 1,
  retention_hours: 168,
  producer_batch_size: 32768,
  producer_linger_ms: 5,
  compression_type: 'lz4',
  fetch_min_bytes: 1,
})

const accelModes = [
  { value: 'write_async', labelKey: 'kafkaAccel.modeWriteAsync', descKey: 'kafkaAccel.modeWriteAsyncDesc' },
  { value: 'cache_invalidate', labelKey: 'kafkaAccel.modeCacheInvalidate', descKey: 'kafkaAccel.modeCacheInvalidateDesc' },
  { value: 'read_through', labelKey: 'kafkaAccel.modeReadThrough', descKey: 'kafkaAccel.modeReadThroughDesc' },
]

const compressionOptions = [
  { value: 'none', labelKey: 'kafkaAccel.compressionNone' },
  { value: 'gzip', labelKey: 'kafkaAccel.compressionGzip' },
  { value: 'snappy', labelKey: 'kafkaAccel.compressionSnappy' },
  { value: 'lz4', labelKey: 'kafkaAccel.compressionLz4' },
  { value: 'zstd', labelKey: 'kafkaAccel.compressionZstd' },
]

const kafkaPresets = [
  { key: 'high_throughput', labelKey: 'kafkaAccel.presetHighThroughput' },
  { key: 'low_latency', labelKey: 'kafkaAccel.presetLowLatency' },
]

const tutorialSteps = computed(() => [
  t('kafkaAccel.tutorialStep1'),
  t('kafkaAccel.tutorialStep2'),
  t('kafkaAccel.tutorialStep3'),
  t('kafkaAccel.tutorialStep4'),
  t('kafkaAccel.tutorialStep5'),
  t('kafkaAccel.tutorialStep6'),
  t('kafkaAccel.tutorialStep7'),
])

const faqItems = computed(() => [
  t('kafkaAccel.tutorialFaq1'),
  t('kafkaAccel.tutorialFaq2'),
  t('kafkaAccel.tutorialFaq3'),
  t('kafkaAccel.tutorialFaq4'),
])

const eligibleDatabases = computed(() =>
  databases.value.filter((d) => ['mysql', 'mariadb', 'postgresql', 'postgres'].includes((d.type || '').toLowerCase())),
)

const statusBannerType = computed(() => {
  if (status.value?.kafka_running && status.value?.broker_reachable) return 'success'
  if (status.value?.kafka_installed) return 'warning'
  return 'error'
})

const kafkaReady = computed(() => !!(status.value?.kafka_running && status.value?.broker_reachable))

function applyConfigFromResponse(res: any) {
  const cfg = res.data?.config || {}
  form.enabled = !!cfg.enabled
  form.bootstrap_servers = cfg.bootstrap_servers || '127.0.0.1:9092'
  form.topic_prefix = cfg.topic_prefix || 'opanel.db'
  form.mode = cfg.mode || 'write_async'
  form.consumer_group = cfg.consumer_group || 'opanel-db-accel'
  form.linked_database_ids = res.data?.linked_database_ids || []
  form.topic_partitions = cfg.topic_partitions || 3
  form.replication_factor = cfg.replication_factor || 1
  form.retention_hours = cfg.retention_hours || 168
  form.producer_batch_size = cfg.producer_batch_size || 32768
  form.producer_linger_ms = cfg.producer_linger_ms ?? 5
  form.compression_type = cfg.compression_type || 'lz4'
  form.fetch_min_bytes = cfg.fetch_min_bytes || 1
  expectedTopics.value = res.data?.expected_topics || []
}

async function loadStatus() {
  try {
    const res: any = await api.get('/kafka-accel/status')
    status.value = res.data || null
  } catch {
    status.value = null
  }
}

async function loadTopics() {
  try {
    const res: any = await api.get('/kafka-accel/topics')
    topics.value = res.data?.topics || []
    topicsHint.value = res.data?.hint || ''
  } catch {
    topics.value = []
    topicsHint.value = ''
  }
}

async function loadConfig() {
  const res: any = await api.get('/kafka-accel/config')
  applyConfigFromResponse(res)
}

async function loadDatabases() {
  try {
    const res: any = await api.get('/databases')
    databases.value = res.data || []
  } catch {
    databases.value = []
  }
}

async function loadAll() {
  loading.value = true
  try {
    await Promise.all([loadStatus(), loadConfig(), loadDatabases(), loadTopics()])
  } finally {
    loading.value = false
  }
}

function buildPatchPayload() {
  return {
    enabled: form.enabled,
    bootstrap_servers: form.bootstrap_servers,
    topic_prefix: form.topic_prefix,
    mode: form.mode,
    consumer_group: form.consumer_group,
    linked_database_ids: form.linked_database_ids,
    topic_partitions: form.topic_partitions,
    replication_factor: form.replication_factor,
    retention_hours: form.retention_hours,
    producer_batch_size: form.producer_batch_size,
    producer_linger_ms: form.producer_linger_ms,
    compression_type: form.compression_type,
    fetch_min_bytes: form.fetch_min_bytes,
  }
}

async function saveConfig() {
  saving.value = true
  try {
    const res: any = await api.patch('/kafka-accel/config', buildPatchPayload())
    applyConfigFromResponse(res)
    ElMessage.success(t('kafkaAccel.saved'))
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
  } finally {
    saving.value = false
  }
}

async function applyTopics() {
  applying.value = true
  try {
    await saveConfig()
    const res: any = await api.post('/kafka-accel/apply')
    ElMessage.success(res.data?.message || t('kafkaAccel.applySuccess'))
    await loadTopics()
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('kafkaAccel.applyFailed')))
  } finally {
    applying.value = false
  }
}

async function applyPreset(key: string, label: string) {
  await ElMessageBox.confirm(t('kafkaAccel.presetConfirm', { name: label }), t('common.confirm'), { type: 'info' })
  presetLoading.value = true
  try {
    const res: any = await api.post(`/kafka-accel/presets/${key}`)
    await loadConfig()
    ElMessage.success(res.data?.message || t('kafkaAccel.presetApplied', { name: label }))
    await loadStatus()
    await loadTopics()
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
  } finally {
    presetLoading.value = false
  }
}

async function optimizeOneClick() {
  await ElMessageBox.confirm(t('kafkaAccel.optimizeConfirm'), t('common.confirm'), { type: 'info' })
  presetLoading.value = true
  try {
    const res: any = await api.post('/kafka-accel/presets/optimize')
    await loadConfig()
    ElMessage.success(res.data?.message || t('kafkaAccel.optimizeSuccess'))
    await loadStatus()
    await loadTopics()
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
  } finally {
    presetLoading.value = false
  }
}

function openInstall() {
  installTrigger.value = true
  installDialog.value = true
}

function onInstallDone(payload: { success: boolean }) {
  if (payload.success) loadAll()
}

onMounted(loadAll)
</script>

<template>
  <div class="kafka-accel" v-loading="loading">
    <el-alert
      :title="t('kafkaAccel.statusTitle')"
      :type="statusBannerType"
      show-icon
      :closable="false"
      class="status-banner"
    >
      <template #default>
        <p>{{ t('kafkaAccel.statusHint') }}</p>
        <p>
          <strong>{{ t('kafkaAccel.installed') }}:</strong>
          {{ status?.kafka_installed ? t('common.yes') : t('common.no') }}
          · {{ t('kafkaAccel.running') }}:
          {{ status?.kafka_running ? t('common.yes') : t('common.no') }}
          · {{ t('kafkaAccel.broker') }}:
          {{ status?.broker_reachable ? t('common.yes') : t('common.no') }}
        </p>
        <p v-if="status?.hint" class="status-msg">{{ status.hint }}</p>
        <el-button
          v-if="!kafkaReady"
          type="primary"
          size="small"
          style="margin-top: 8px"
          @click="openInstall"
        >
          {{ status?.kafka_installed ? t('kafkaAccel.reinstallOneClick') : t('kafkaAccel.installOneClick') }}
        </el-button>
      </template>
    </el-alert>

    <el-collapse v-model="tutorialOpen" class="tutorial-collapse">
      <el-collapse-item :title="t('kafkaAccel.tutorialTitle')" name="tutorial">
        <ol class="tutorial-steps">
          <li v-for="(step, i) in tutorialSteps" :key="i">{{ step }}</li>
        </ol>
        <div class="tutorial-notes">
          <p><strong>{{ t('kafkaAccel.tutorialFaqTitle') }}</strong></p>
          <ul>
            <li v-for="(item, i) in faqItems" :key="'f' + i">{{ item }}</li>
          </ul>
        </div>
      </el-collapse-item>
    </el-collapse>

    <el-card shadow="never" class="section-card">
      <template #header>
        <div class="config-header">
          <span>{{ t('kafkaAccel.configTitle') }}</span>
          <div class="config-toolbar">
            <el-button type="primary" :loading="presetLoading" @click="optimizeOneClick">
              {{ t('kafkaAccel.optimizeOneClick') }}
            </el-button>
          </div>
        </div>
      </template>

      <div class="preset-row">
        <span class="preset-label">{{ t('kafkaAccel.applyPreset') }}</span>
        <el-button
          v-for="p in kafkaPresets"
          :key="p.key"
          size="small"
          :loading="presetLoading"
          @click="applyPreset(p.key, t(p.labelKey))"
        >
          {{ t(p.labelKey) }}
        </el-button>
      </div>

      <el-form label-width="140px" @submit.prevent>
        <el-form-item :label="t('kafkaAccel.enabled')">
          <el-switch v-model="form.enabled" />
        </el-form-item>
        <el-form-item :label="t('kafkaAccel.bootstrap')">
          <el-input v-model="form.bootstrap_servers" style="max-width: 360px" />
        </el-form-item>
        <el-form-item :label="t('kafkaAccel.topicPrefix')">
          <el-input v-model="form.topic_prefix" style="max-width: 360px" />
        </el-form-item>
        <el-form-item :label="t('kafkaAccel.mode')">
          <el-select v-model="form.mode" style="max-width: 360px">
            <el-option
              v-for="m in accelModes"
              :key="m.value"
              :value="m.value"
              :label="t(m.labelKey)"
            />
          </el-select>
          <p class="form-hint">
            {{ t(accelModes.find((x) => x.value === form.mode)?.descKey || 'kafkaAccel.modeWriteAsyncDesc') }}
          </p>
        </el-form-item>
        <el-form-item :label="t('kafkaAccel.databases')">
          <el-select
            v-model="form.linked_database_ids"
            multiple
            filterable
            style="width: 100%; max-width: 520px"
            :placeholder="t('kafkaAccel.databasesPlaceholder')"
          >
            <el-option
              v-for="db in eligibleDatabases"
              :key="db.id"
              :value="db.id"
              :label="`${db.name} (${db.type})`"
            />
          </el-select>
        </el-form-item>

        <el-collapse class="advanced-collapse">
          <el-collapse-item :title="t('kafkaAccel.advancedTitle')" name="advanced">
            <el-form-item :label="t('kafkaAccel.topicPartitions')">
              <el-input-number v-model="form.topic_partitions" :min="1" :max="100" />
            </el-form-item>
            <el-form-item :label="t('kafkaAccel.replicationFactor')">
              <el-input-number v-model="form.replication_factor" :min="1" :max="10" />
            </el-form-item>
            <el-form-item :label="t('kafkaAccel.retentionHours')">
              <el-input-number v-model="form.retention_hours" :min="1" :max="8760" />
            </el-form-item>
            <el-form-item :label="t('kafkaAccel.producerBatchSize')">
              <el-input-number v-model="form.producer_batch_size" :min="1024" :max="1048576" :step="1024" />
            </el-form-item>
            <el-form-item :label="t('kafkaAccel.producerLingerMs')">
              <el-input-number v-model="form.producer_linger_ms" :min="0" :max="1000" />
            </el-form-item>
            <el-form-item :label="t('kafkaAccel.compressionType')">
              <el-select v-model="form.compression_type" style="max-width: 200px">
                <el-option
                  v-for="c in compressionOptions"
                  :key="c.value"
                  :value="c.value"
                  :label="t(c.labelKey)"
                />
              </el-select>
            </el-form-item>
            <el-form-item :label="t('kafkaAccel.consumerGroup')">
              <el-input v-model="form.consumer_group" style="max-width: 360px" />
            </el-form-item>
            <el-form-item :label="t('kafkaAccel.fetchMinBytes')">
              <el-input-number v-model="form.fetch_min_bytes" :min="1" :max="1048576" />
            </el-form-item>
          </el-collapse-item>
        </el-collapse>

        <el-form-item>
          <el-button type="primary" :loading="saving" @click="saveConfig">{{ t('common.save') }}</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card shadow="never" class="section-card">
      <template #header>
        <div class="topics-header">
          <span>{{ t('kafkaAccel.topicsTitle') }}</span>
          <el-button type="success" :loading="applying" :disabled="!kafkaReady" @click="applyTopics">
            {{ t('kafkaAccel.applyTopics') }}
          </el-button>
        </div>
      </template>
      <el-alert
        :title="t('kafkaAccel.comboHint')"
        type="info"
        show-icon
        :closable="false"
        style="margin-bottom: 12px"
      />
      <p v-if="expectedTopics.length" class="expected-label">{{ t('kafkaAccel.expectedTopics') }}</p>
      <div v-if="expectedTopics.length" class="topic-tags">
        <el-tag v-for="tp in expectedTopics" :key="tp" size="small">{{ tp }}</el-tag>
      </div>
      <p v-if="topicsHint && !topics.length" class="form-hint">{{ topicsHint }}</p>
      <el-table v-if="topics.length" :data="topics.map((x) => ({ name: x }))" stripe size="small">
        <el-table-column prop="name" :label="t('kafkaAccel.topicName')" />
      </el-table>
      <p v-else-if="!topicsHint" class="muted">{{ t('kafkaAccel.noTopics') }}</p>
    </el-card>

    <SoftwareInstallLogDialog
      v-model="installDialog"
      :app-key="installAppKey"
      :app-name="installAppName"
      :trigger-install="installTrigger"
      @done="onInstallDone"
    />
  </div>
</template>

<style scoped>
.kafka-accel .status-banner {
  margin-bottom: 16px;
}
.kafka-accel .status-banner p {
  margin: 4px 0;
  font-size: 13px;
}
.tutorial-collapse {
  margin-bottom: 16px;
}
.tutorial-collapse :deep(.el-collapse-item__header) {
  font-weight: 600;
}
.tutorial-steps {
  margin: 0 0 12px;
  padding-left: 20px;
  line-height: 1.8;
}
.tutorial-notes {
  color: var(--el-text-color-secondary);
  font-size: 13px;
  line-height: 1.7;
}
.tutorial-notes ul {
  margin: 4px 0 0;
  padding-left: 20px;
}
.section-card {
  margin-bottom: 16px;
}
.config-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.config-toolbar {
  display: flex;
  gap: 8px;
}
.preset-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 16px;
}
.preset-label {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  margin-right: 4px;
}
.advanced-collapse {
  margin-bottom: 8px;
  border: none;
}
.advanced-collapse :deep(.el-collapse-item__header) {
  font-weight: 600;
  border-bottom: none;
}
.topics-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.form-hint {
  margin: 6px 0 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.expected-label {
  margin: 0 0 8px;
  font-size: 13px;
  font-weight: 600;
}
.topic-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 12px;
}
.muted {
  color: var(--el-text-color-placeholder);
  font-size: 13px;
}
</style>
