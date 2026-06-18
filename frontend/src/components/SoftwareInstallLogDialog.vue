<script setup lang="ts">
import { computed, nextTick, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import api from '@/api'
import { ElMessage } from 'element-plus'

const props = withDefaults(defineProps<{
  modelValue: boolean
  appKey: string
  appName?: string
  version?: string
  /** Call POST /install when dialog opens */
  triggerInstall?: boolean
  installMode?: 'install' | 'upgrade'
  /** Override default /software/:key/install/logs */
  logsApiPath?: string
  /** Override default /software/:key/install */
  installApiPath?: string
  /** Custom JSON body for install POST */
  installPayload?: Record<string, unknown>
}>(), {
  triggerInstall: false,
  installMode: 'install',
})

const emit = defineEmits<{
  'update:modelValue': [boolean]
  done: [payload: { success: boolean; error?: string }]
}>()

const { t } = useI18n()

const visible = computed({
  get: () => props.modelValue,
  set: (v: boolean) => emit('update:modelValue', v),
})

const lines = ref<string[]>([])
type InstallStatus = 'idle' | 'installing' | 'success' | 'failed'

const status = ref<InstallStatus>('idle')
const installError = ref('')
const logRef = ref<HTMLElement | null>(null)
const starting = ref(false)
let pollTimer: ReturnType<typeof setInterval> | null = null

const title = computed(() =>
  props.installMode === 'upgrade'
    ? t('software.upgradeTitle', { name: props.appName || props.appKey })
    : t('software.installLogTitle', { name: props.appName || props.appKey }),
)

const statusLabel = computed(() => {
  switch (status.value) {
    case 'installing': return t('software.installing')
    case 'success': return t('software.installSuccessShort')
    case 'failed': return t('software.failed')
    default: return t('software.installLogIdle')
  }
})

const statusType = computed(() => {
  switch (status.value) {
    case 'installing': return 'warning'
    case 'success': return 'success'
    case 'failed': return 'danger'
    default: return 'info'
  }
})

const canClose = computed(() => status.value !== 'installing' && !starting.value)

async function scrollToBottom() {
  await nextTick()
  const el = logRef.value
  if (el) el.scrollTop = el.scrollHeight
}

async function fetchLogs(): Promise<InstallStatus> {
  if (!props.appKey && !props.logsApiPath) return 'idle'
  try {
    const url = props.logsApiPath || `/software/${props.appKey}/install/logs`
    const res: any = await api.get(url, { timeout: 15000 })
    const data = res.data || {}
    lines.value = data.lines || []
    const next = (data.status || 'idle') as InstallStatus
    status.value = next
    installError.value = data.install_error || ''
    await scrollToBottom()
    return next
  } catch (e: any) {
    const msg = e?.error || e?.message || t('software.installFailed')
    lines.value = [t('software.installLogApiError', { msg })]
    status.value = 'failed'
    installError.value = msg
    return 'failed'
  }
}

async function startInstall() {
  if (!props.appKey && !props.installApiPath) return
  starting.value = true
  lines.value = [props.installMode === 'upgrade' ? t('software.upgradeStarted') : t('software.installLogStarting')]
  status.value = 'installing'
  try {
    if (props.installApiPath) {
      await api.post(props.installApiPath, props.installPayload || {}, { timeout: 120000 })
    } else {
      const endpoint = props.installMode === 'upgrade' ? 'upgrade' : 'install'
      await api.post(`/software/${props.appKey}/${endpoint}`, { version: props.version || '' }, { timeout: 120000 })
    }
  } catch (e: any) {
    const msg = e?.error || e?.message || t('software.installFailed')
    if (String(msg).includes('already')) {
      lines.value.push(props.installMode === 'upgrade' ? t('software.alreadyOnVersion') : t('software.installAlreadyDone'))
      await fetchLogs()
    } else {
      lines.value.push(msg)
      ElMessage.warning(msg)
    }
  } finally {
    starting.value = false
  }
}

function stopPoll() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

function startPoll() {
  stopPoll()
  pollTimer = setInterval(async () => {
    try {
      await fetchLogs()
      if (status.value === 'success') {
        stopPoll()
        ElMessage.success(t('software.installSuccess', { name: props.appName || props.appKey }))
        emit('done', { success: true })
      } else if (status.value === 'failed') {
        stopPoll()
        ElMessage.error(installError.value || t('software.installFailed'))
        emit('done', { success: false, error: installError.value })
      }
    } catch { /* ignore transient */ }
  }, 1200)
}

async function onOpen() {
  lines.value = []
  status.value = 'idle'
  installError.value = ''
  const currentStatus = await fetchLogs()
  const skipSuccess = !!props.installApiPath
  if (props.triggerInstall && currentStatus !== 'installing' && (skipSuccess || currentStatus !== 'success')) {
    await startInstall()
    await fetchLogs()
  }
  startPoll()
}

function onClose() {
  stopPoll()
}

watch(visible, (v) => {
  if (v) onOpen()
  else onClose()
})

onUnmounted(stopPoll)
</script>

<template>
  <el-dialog
    v-model="visible"
    :title="title"
    width="680px"
    class="install-log-dialog"
    :close-on-click-modal="canClose"
    :close-on-press-escape="canClose"
    :show-close="canClose"
    destroy-on-close
    @closed="onClose"
  >
    <div class="log-toolbar">
      <el-tag :type="statusType" size="small">{{ statusLabel }}</el-tag>
      <span v-if="installError" class="log-error">{{ installError }}</span>
      <el-button
        v-if="status === 'installing'"
        size="small"
        text
        :loading="true"
      >
        {{ t('software.installLogRunning') }}
      </el-button>
    </div>
    <div ref="logRef" class="log-box">
      <div v-if="!lines.length" class="log-empty">{{ t('software.installLogWaiting') }}</div>
      <pre v-else class="log-pre">{{ lines.join('\n') }}</pre>
    </div>
    <template #footer>
      <el-button v-if="canClose" type="primary" @click="visible = false">{{ t('common.close') }}</el-button>
      <el-button v-else disabled>{{ t('software.installing') }}…</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.log-toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
  flex-wrap: wrap;
}
.log-error {
  font-size: 12px;
  color: var(--el-color-danger);
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.log-box {
  height: 360px;
  overflow: auto;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: #0f172a;
  padding: 12px;
}
.log-empty {
  color: #94a3b8;
  font-size: 13px;
  text-align: center;
  padding: 40px 0;
}
.log-pre {
  margin: 0;
  font-family: Consolas, Monaco, 'Courier New', monospace;
  font-size: 12px;
  line-height: 1.55;
  color: #e2e8f0;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
