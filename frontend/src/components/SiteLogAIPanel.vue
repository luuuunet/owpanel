<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { MagicStick } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import api, { AI_REQUEST_TIMEOUT, resolveApiError } from '@/api'
import AIChatPanel, { type AIChatMessage } from '@/components/AIChatPanel.vue'

const props = defineProps<{
  siteId: number
  domain: string
  accessLog: string
  errorLog: string
  accessPath: string
  errorPath: string
}>()

const emit = defineEmits<{ repaired: [] }>()

const { t } = useI18n()

const messages = ref<AIChatMessage[]>([])
const repairLoading = ref(false)
const repairLogs = ref<string[]>([])

function hasLogContent() {
  return !!(props.accessLog.trim() || props.errorLog.trim())
}

function streamBody(message: string, history: AIChatMessage[]) {
  return {
    message,
    access_log: props.accessLog,
    error_log: props.errorLog,
    access_path: props.accessPath,
    error_path: props.errorPath,
    history: history.filter((m) => !m.streaming).slice(-10).map((m) => ({ role: m.role, content: m.content })),
  }
}

async function sendFallback(message: string, history: AIChatMessage[], signal: AbortSignal) {
  if (!hasLogContent()) {
    throw new Error(t('siteModify.logAiNoContent'))
  }
  const res: any = await api.post(
    `/websites/${props.siteId}/logs/ai/chat`,
    streamBody(message, history),
    { timeout: AI_REQUEST_TIMEOUT, signal } as any,
  )
  return res.data?.reply || ''
}

async function runAutoRepair() {
  if (!hasLogContent()) {
    ElMessage.warning(t('siteModify.logAiNoContent'))
    return
  }
  repairLoading.value = true
  repairLogs.value = []
  try {
    const res: any = await api.post(
      `/websites/${props.siteId}/logs/ai/repair`,
      { access_log: props.accessLog, error_log: props.errorLog },
      { timeout: AI_REQUEST_TIMEOUT },
    )
    const data = res.data || {}
    repairLogs.value = data.logs || []
    if (data.fixed) {
      ElMessage.success(t('siteModify.logAiRepairSuccess'))
      emit('repaired')
    } else if (data.summary) {
      ElMessage.warning(t('siteModify.logAiRepairPartial'))
    }
    if (data.summary) {
      messages.value.push({
        id: `${Date.now()}-repair`,
        role: 'assistant',
        content: `**${data.summary}**\n\n${data.diagnosis || ''}`.trim(),
      })
    }
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('siteModify.logAiRepairFailed'), t('common.requestTimeout')))
  } finally {
    repairLoading.value = false
  }
}

function clearAll() {
  repairLogs.value = []
}
</script>

<template>
  <div class="site-log-ai-wrap">
    <AIChatPanel
      v-model="messages"
      :welcome="t('siteModify.logAiWelcome')"
      :placeholder="t('siteModify.logAiPlaceholder')"
      :context-label="domain"
      :quick-prompts="[t('siteModify.logAiPromptErrors'), t('siteModify.logAiPromptFix')]"
      :stream-url="`/websites/${siteId}/logs/ai/chat/stream`"
      :stream-body="streamBody"
      :send-fallback="sendFallback"
      height="420px"
      @clear="clearAll"
    >
      <template #toolbar>
        <div class="site-log-toolbar">
          <el-button type="primary" class="ai-repair-btn" :loading="repairLoading" @click="runAutoRepair">
            <el-icon v-if="!repairLoading"><MagicStick /></el-icon>
            {{ t('siteModify.logAiAutoFix') }}
          </el-button>
          <p class="toolbar-hint">{{ t('siteModify.logAiHint') }}</p>
          <div v-if="repairLogs.length" class="repair-log">
            <div v-for="(line, i) in repairLogs" :key="i">{{ line }}</div>
          </div>
        </div>
      </template>
    </AIChatPanel>
  </div>
</template>

<style scoped>
.site-log-ai-wrap {
  min-width: 340px;
  max-width: 400px;
  flex-shrink: 0;
}
.site-log-toolbar {
  padding: 0 12px 8px;
}
.ai-repair-btn {
  width: 100%;
  background: linear-gradient(135deg, #7c3aed, #2563eb) !important;
  border: none !important;
  font-weight: 600;
}
.toolbar-hint {
  margin: 8px 0 0;
  font-size: 11px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}
.repair-log {
  margin-top: 8px;
  padding: 8px;
  border-radius: 6px;
  background: var(--el-fill-color-light);
  font-size: 11px;
  font-family: Consolas, monospace;
  max-height: 100px;
  overflow-y: auto;
}
</style>
