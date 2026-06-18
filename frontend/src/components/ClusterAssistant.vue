<script setup lang="ts">
import { nextTick, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Promotion, MagicStick } from '@element-plus/icons-vue'
import api, { AI_REQUEST_TIMEOUT, resolveApiError } from '@/api'
import type { FlowGraph } from '@/components/ClusterFlowCanvas.vue'

const props = defineProps<{
  nodes: any[]
  graph: FlowGraph
  balancers?: any[]
}>()

const emit = defineEmits<{
  applyGraph: [graph: FlowGraph]
}>()

const { t } = useI18n()
const chatInput = ref('')
const chatLoading = ref(false)
const messages = ref<{ role: 'user' | 'assistant'; content: string; graph?: FlowGraph }[]>([])
const chatBoxRef = ref<HTMLElement>()

async function scrollChat() {
  await nextTick()
  if (chatBoxRef.value) chatBoxRef.value.scrollTop = chatBoxRef.value.scrollHeight
}

async function sendChat() {
  const text = chatInput.value.trim()
  if (!text || chatLoading.value) return
  messages.value.push({ role: 'user', content: text })
  chatInput.value = ''
  chatLoading.value = true
  await scrollChat()
  try {
    const history = messages.value.slice(0, -1).map((m) => ({ role: m.role, content: m.content }))
    const res: any = await api.post('/cluster/workflow/ai/suggest', {
      message: text,
      history,
      context: { nodes: props.nodes, graph: props.graph, balancers: props.balancers || [] },
    }, { timeout: AI_REQUEST_TIMEOUT })
    const g = res.data?.suggested_graph
    messages.value.push({
      role: 'assistant',
      content: res.data?.reply || '',
      graph: g?.nodes ? (g as FlowGraph) : undefined,
    })
  } catch (e: any) {
    const errMsg = resolveApiError(e, t('clusterPage.aiFailed'), t('common.requestTimeout'))
    ElMessage.error(errMsg)
    messages.value.push({ role: 'assistant', content: errMsg })
  } finally {
    chatLoading.value = false
    await scrollChat()
  }
}

function applyGraph(g?: FlowGraph) {
  if (!g?.nodes) return
  emit('applyGraph', g)
  ElMessage.success(t('clusterPage.flowApplied'))
}

function quickPrompt(key: string) {
  chatInput.value = t(`clusterPage.aiFlowPrompt${key}`)
  sendChat()
}
</script>

<template>
  <div class="cluster-assistant">
    <div class="chat-header"><el-icon><MagicStick /></el-icon><span>{{ t('clusterPage.flowAssistant') }}</span></div>
    <div ref="chatBoxRef" class="chat-box">
      <div v-if="!messages.length" class="chat-empty">
        <p>{{ t('clusterPage.aiFlowWelcome') }}</p>
        <el-button size="small" text type="primary" @click="quickPrompt('Ha')">{{ t('clusterPage.aiFlowQuickHa') }}</el-button>
        <el-button size="small" text type="primary" @click="quickPrompt('Wp')">{{ t('clusterPage.aiFlowQuickWp') }}</el-button>
      </div>
      <div v-for="(msg, i) in messages" :key="i" class="chat-msg" :class="msg.role">
        <div class="msg-body">{{ msg.content }}</div>
        <el-button v-if="msg.graph" size="small" type="primary" plain @click="applyGraph(msg.graph)">{{ t('clusterPage.flowApplyGraph') }}</el-button>
      </div>
      <div v-if="chatLoading" class="chat-msg assistant"><div class="msg-body">{{ t('clusterPage.aiThinking') }}</div></div>
    </div>
    <div class="chat-input-row">
      <el-input v-model="chatInput" type="textarea" :rows="3" :placeholder="t('clusterPage.aiFlowPlaceholder')" @keydown.ctrl.enter="sendChat" />
      <el-button type="primary" :loading="chatLoading" :icon="Promotion" @click="sendChat">{{ t('clusterPage.aiSend') }}</el-button>
    </div>
    <el-alert type="info" :closable="false" show-icon :title="t('clusterPage.aiNoteTitle')">{{ t('clusterPage.aiNoteBody') }}</el-alert>
  </div>
</template>

<style scoped>
.cluster-assistant { display: flex; flex-direction: column; gap: 10px; height: 100%; min-height: 400px; overflow: hidden; }
.chat-header { flex-shrink: 0; display: flex; align-items: center; gap: 6px; font-weight: 600; padding-bottom: 8px; border-bottom: 1px solid var(--el-border-color-lighter); }
.chat-box { flex: 1; min-height: 0; overflow-y: auto; padding: 8px; background: var(--el-fill-color-lighter); border-radius: 8px; }
.chat-empty { font-size: 13px; color: var(--el-text-color-secondary); }
.chat-msg { margin-bottom: 12px; }
.chat-msg.user .msg-body { background: var(--el-color-primary-light-9); padding: 8px; border-radius: 8px; }
.msg-body { white-space: pre-wrap; font-size: 13px; line-height: 1.5; margin-bottom: 6px; }
.chat-input-row { flex-shrink: 0; display: flex; flex-direction: column; gap: 8px; }
</style>
