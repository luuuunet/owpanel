<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  MagicStick,
  Promotion,
  Delete,
  RefreshRight,
  VideoPause,
} from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { copyText, newChatId, renderMarkdown, revealTextStream, streamSSE } from '@/utils/aiChat'
import { appendSpeechTranscript } from '@/composables/useSpeechInput'
import AiChatVoiceButton from '@/components/AiChatVoiceButton.vue'
import { apiBaseURL } from '@/api'

export interface AIChatMessage {
  id: string
  role: 'user' | 'assistant'
  content: string
  streaming?: boolean
  error?: boolean
}

const props = withDefaults(
  defineProps<{
    welcome?: string
    placeholder?: string
    contextLabel?: string
    quickPrompts?: string[]
    height?: string
    streamUrl?: string
    streamBody?: (message: string, history: AIChatMessage[]) => Record<string, unknown>
    sendFallback?: (message: string, history: AIChatMessage[], signal: AbortSignal) => Promise<string>
    disabled?: boolean
    showApplyCode?: boolean
  }>(),
  {
    welcome: '',
    placeholder: '',
    quickPrompts: () => [],
    height: '100%',
    showApplyCode: true,
  },
)

const emit = defineEmits<{
  clear: []
  applyCode: [code: string, lang: string]
  submit: [message: string]
}>()

const { t } = useI18n()

const messages = defineModel<AIChatMessage[]>({ default: () => [] })
const chatInput = ref('')
const loading = ref(false)
const chatBoxRef = ref<HTMLElement>()
const composerRef = ref<HTMLTextAreaElement>()
let abortCtrl: AbortController | null = null

const canSend = computed(() => !loading.value && !props.disabled && chatInput.value.trim().length > 0)

async function scrollToBottom() {
  await nextTick()
  if (chatBoxRef.value) chatBoxRef.value.scrollTop = chatBoxRef.value.scrollHeight
}

async function streamReply(userText: string, assistant: AIChatMessage) {
  const url = props.streamUrl
  if (!url || !props.streamBody) return false

  let gotChunk = false
  await streamSSE({
    url: apiBaseURL() + url,
    body: props.streamBody(userText, messages.value),
    signal: abortCtrl?.signal,
    onChunk: (chunk) => {
      gotChunk = true
      assistant.content += chunk
      scrollToBottom()
    },
    onError: (err) => {
      assistant.error = true
      assistant.content = err
    },
  })
  return gotChunk || assistant.content.length > 0
}

async function sendMessage(preset?: string) {
  const text = (preset || chatInput.value).trim()
  if (!text || loading.value || props.disabled) return

  messages.value.push({ id: newChatId(), role: 'user', content: text })
  chatInput.value = ''
  emit('submit', text)

  const assistant: AIChatMessage = {
    id: newChatId(),
    role: 'assistant',
    content: '',
    streaming: true,
  }
  messages.value.push(assistant)
  loading.value = true
  abortCtrl = new AbortController()
  await scrollToBottom()

  try {
    let streamed = false
    if (props.streamUrl && props.streamBody) {
      streamed = await streamReply(text, assistant)
    }
    if (!streamed && props.sendFallback) {
      const full = await props.sendFallback(text, messages.value, abortCtrl.signal)
      if (abortCtrl.signal.aborted) return
      await revealTextStream(full, (partial) => {
        assistant.content = partial
        scrollToBottom()
      }, abortCtrl.signal)
    } else if (!streamed && !assistant.content) {
      assistant.content = t('aiChat.emptyReply')
      assistant.error = true
    }
  } catch (e: any) {
    if (!abortCtrl?.signal.aborted) {
      assistant.error = true
      assistant.content = e?.error || e?.message || t('aiChat.failed')
    }
  } finally {
    assistant.streaming = false
    loading.value = false
    abortCtrl = null
    scrollToBottom()
  }
}

function stopGeneration() {
  abortCtrl?.abort()
  loading.value = false
  const last = [...messages.value].reverse().find((m) => m.role === 'assistant' && m.streaming)
  if (last) last.streaming = false
}

function clearChat() {
  if (loading.value) stopGeneration()
  messages.value = []
  emit('clear')
}

async function regenerate() {
  const users = messages.value.filter((m) => m.role === 'user')
  const lastUser = users[users.length - 1]
  if (!lastUser || loading.value) return
  while (messages.value.length && messages.value[messages.value.length - 1].role !== 'user') {
    messages.value.pop()
  }
  if (messages.value[messages.value.length - 1]?.role === 'user') {
    messages.value.pop()
  }
  await sendMessage(lastUser.content)
}

function onComposerKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    sendMessage()
  }
}

function onVoiceTranscript(text: string) {
  chatInput.value = appendSpeechTranscript(chatInput.value, text)
}

function onMessageClick(e: MouseEvent) {
  const target = e.target as HTMLElement
  if (target.classList.contains('md-code-copy')) {
    const block = target.closest('.md-code-block')
    const code = block?.querySelector('pre code')?.textContent || ''
    void copyText(code).then((ok) => {
      ElMessage.success(ok ? t('aiChat.copied') : t('aiChat.copyFailed'))
    })
    return
  }
  if (target.classList.contains('md-code-apply')) {
    const block = target.closest('.md-code-block')
    const code = block?.querySelector('pre code')?.textContent || ''
    const lang = block?.getAttribute('data-lang') || 'text'
    emit('applyCode', code, lang)
  }
}

function renderContent(msg: AIChatMessage) {
  if (msg.role === 'user') return msg.content
  return renderMarkdown(msg.content).replace(/data-copy>Copy<\/button>/g, `data-copy>${t('aiChat.copy') || 'Copy'}</button>`)
}

watch(() => messages.value.length, () => scrollToBottom())

onMounted(() => scrollToBottom())
</script>

<template>
  <div class="ai-chat-panel" :style="{ height }">
    <header class="ai-chat-header">
      <div class="ai-chat-brand">
        <span class="ai-chat-logo"><el-icon><MagicStick /></el-icon></span>
        <div>
          <div class="ai-chat-title">{{ t('aiChat.title') }}</div>
          <div v-if="contextLabel" class="ai-chat-context">{{ contextLabel }}</div>
        </div>
      </div>
      <div class="ai-chat-header-actions">
        <el-tooltip :content="t('aiChat.regenerate')">
          <el-button text :icon="RefreshRight" :disabled="loading || !messages.some((m) => m.role === 'user')" @click="regenerate" />
        </el-tooltip>
        <el-tooltip :content="t('aiChat.clear')">
          <el-button text :icon="Delete" :disabled="!messages.length" @click="clearChat" />
        </el-tooltip>
      </div>
    </header>

    <div ref="chatBoxRef" class="ai-chat-messages" @click="onMessageClick">
      <div v-if="!messages.length" class="ai-chat-welcome">
        <el-icon class="welcome-icon"><MagicStick /></el-icon>
        <p>{{ welcome || t('aiChat.welcome') }}</p>
      </div>

      <div
        v-for="msg in messages"
        :key="msg.id"
        class="ai-chat-row"
        :class="[msg.role, { error: msg.error, streaming: msg.streaming }]"
      >
        <div class="ai-chat-avatar">{{ msg.role === 'user' ? t('aiChat.you') : 'AI' }}</div>
        <div class="ai-chat-bubble">
          <div v-if="msg.role === 'user'" class="ai-chat-text user-text">{{ msg.content }}</div>
          <div v-else-if="msg.streaming && !msg.content" class="ai-chat-typing">
            <span /><span /><span />
          </div>
          <div v-else class="ai-chat-md" v-html="renderContent(msg)" />
          <div v-if="msg.streaming && msg.content" class="ai-stream-cursor" />
        </div>
      </div>
    </div>

    <div v-if="quickPrompts.length" class="ai-chat-prompts">
      <button
        v-for="p in quickPrompts"
        :key="p"
        type="button"
        class="prompt-chip"
        :disabled="loading || disabled"
        @click="sendMessage(p)"
      >
        {{ p }}
      </button>
    </div>

    <slot name="toolbar" />

    <footer class="ai-chat-composer">
      <div class="composer-box">
        <textarea
          ref="composerRef"
          v-model="chatInput"
          class="composer-input"
          rows="1"
          :placeholder="placeholder || t('aiChat.placeholder')"
          :disabled="loading || disabled"
          @keydown="onComposerKeydown"
        />
        <div class="composer-actions">
          <AiChatVoiceButton :disabled="loading || disabled" @transcript="onVoiceTranscript" />
          <el-button
            v-if="loading"
            circle
            :icon="VideoPause"
            title="Stop"
            @click="stopGeneration"
          />
          <el-button
            v-else
            type="primary"
            circle
            :icon="Promotion"
            :disabled="!canSend"
            @click="sendMessage()"
          />
        </div>
      </div>
      <p class="composer-hint">{{ t('aiChat.hint') }}</p>
    </footer>
  </div>
</template>

<style scoped>
.ai-chat-panel {
  display: flex;
  flex-direction: column;
  min-height: 320px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 12px;
  overflow: hidden;
}
.ai-chat-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  background: linear-gradient(180deg, rgba(124, 58, 237, 0.08), transparent);
}
.ai-chat-brand {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.ai-chat-logo {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #7c3aed, #2563eb);
  color: #fff;
  font-size: 16px;
}
.ai-chat-title {
  font-weight: 700;
  font-size: 14px;
}
.ai-chat-context {
  font-size: 11px;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 220px;
}
.ai-chat-messages {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 12px;
  scroll-behavior: smooth;
}
.ai-chat-welcome {
  text-align: center;
  padding: 24px 12px;
  color: var(--el-text-color-secondary);
}
.welcome-icon {
  font-size: 28px;
  color: #7c3aed;
  margin-bottom: 8px;
}
.ai-chat-welcome p {
  margin: 0;
  font-size: 13px;
  line-height: 1.6;
}
.ai-chat-row {
  display: flex;
  gap: 10px;
  margin-bottom: 14px;
}
.ai-chat-row.user {
  flex-direction: row-reverse;
}
.ai-chat-avatar {
  flex-shrink: 0;
  width: 28px;
  height: 28px;
  border-radius: 6px;
  font-size: 10px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--el-fill-color);
  color: var(--el-text-color-secondary);
}
.ai-chat-row.user .ai-chat-avatar {
  background: rgba(124, 58, 237, 0.15);
  color: #7c3aed;
}
.ai-chat-row.assistant .ai-chat-avatar {
  background: linear-gradient(135deg, #7c3aed, #2563eb);
  color: #fff;
}
.ai-chat-bubble {
  max-width: calc(100% - 40px);
  min-width: 0;
}
.user-text {
  background: rgba(124, 58, 237, 0.12);
  border: 1px solid rgba(124, 58, 237, 0.2);
  padding: 8px 12px;
  border-radius: 12px 12px 4px 12px;
  font-size: 13px;
  line-height: 1.55;
  white-space: pre-wrap;
  word-break: break-word;
}
.ai-chat-row.user .ai-chat-bubble {
  text-align: right;
}
.ai-chat-md {
  font-size: 13px;
  line-height: 1.6;
  color: var(--el-text-color-primary);
}
.ai-chat-md :deep(.md-p) {
  margin: 0 0 8px;
}
.ai-chat-md :deep(.md-heading) {
  margin: 10px 0 6px;
  font-weight: 600;
}
.ai-chat-md :deep(.md-list) {
  margin: 0 0 8px 18px;
  padding: 0;
}
.ai-chat-md :deep(.md-inline-code) {
  background: var(--el-fill-color);
  padding: 1px 5px;
  border-radius: 4px;
  font-family: Consolas, monospace;
  font-size: 12px;
}
.ai-chat-md :deep(.md-code-block) {
  margin: 8px 0;
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid rgba(255, 255, 255, 0.08);
  background: #1e1e1e;
}
.ai-chat-md :deep(.md-code-head) {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 10px;
  background: #2d2d2d;
  font-size: 11px;
}
.ai-chat-md :deep(.md-code-lang) {
  color: #888;
  text-transform: lowercase;
}
.ai-chat-md :deep(.md-code-copy) {
  border: none;
  background: transparent;
  color: #aaa;
  cursor: pointer;
  font-size: 11px;
  padding: 2px 6px;
  border-radius: 4px;
}
.ai-chat-md :deep(.md-code-copy:hover) {
  background: rgba(255, 255, 255, 0.08);
  color: #fff;
}
.ai-chat-md :deep(pre) {
  margin: 0;
  padding: 10px 12px;
  overflow-x: auto;
}
.ai-chat-md :deep(pre code) {
  font-family: Consolas, 'Cascadia Code', monospace;
  font-size: 12px;
  color: #d4d4d4;
  white-space: pre;
}
.ai-chat-row.error .ai-chat-md {
  color: var(--el-color-danger);
}
.ai-chat-typing {
  display: flex;
  gap: 4px;
  padding: 8px 0;
}
.ai-chat-typing span {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #7c3aed;
  animation: ai-dot 1.2s infinite ease-in-out;
}
.ai-chat-typing span:nth-child(2) { animation-delay: 0.15s; }
.ai-chat-typing span:nth-child(3) { animation-delay: 0.3s; }
@keyframes ai-dot {
  0%, 80%, 100% { opacity: 0.3; transform: scale(0.8); }
  40% { opacity: 1; transform: scale(1); }
}
.ai-stream-cursor {
  display: inline-block;
  width: 2px;
  height: 14px;
  background: #7c3aed;
  margin-left: 2px;
  vertical-align: text-bottom;
  animation: ai-blink 1s step-end infinite;
}
@keyframes ai-blink {
  50% { opacity: 0; }
}
.ai-chat-prompts {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  padding: 0 12px 8px;
}
.prompt-chip {
  border: 1px solid var(--el-border-color);
  background: var(--el-fill-color-blank);
  border-radius: 999px;
  padding: 4px 10px;
  font-size: 12px;
  cursor: pointer;
  color: var(--el-text-color-regular);
}
.prompt-chip:hover:not(:disabled) {
  border-color: #7c3aed;
  color: #7c3aed;
}
.prompt-chip:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.ai-chat-composer {
  padding: 10px 12px 12px;
  border-top: 1px solid var(--el-border-color-lighter);
  background: var(--el-fill-color-blank);
}
.composer-box {
  display: flex;
  align-items: flex-end;
  gap: 8px;
  padding: 8px 10px;
  border-radius: 12px;
  border: 1px solid var(--el-border-color);
  background: var(--el-bg-color);
  transition: border-color 0.15s, box-shadow 0.15s;
}
.composer-box:focus-within {
  border-color: #7c3aed;
  box-shadow: 0 0 0 2px rgba(124, 58, 237, 0.15);
}
.composer-input {
  flex: 1;
  border: none;
  outline: none;
  resize: none;
  min-height: 22px;
  max-height: 120px;
  font-size: 13px;
  line-height: 1.5;
  font-family: inherit;
  background: transparent;
  color: var(--el-text-color-primary);
}
.composer-hint {
  margin: 6px 0 0;
  font-size: 11px;
  color: var(--el-text-color-placeholder);
  text-align: center;
}
</style>
