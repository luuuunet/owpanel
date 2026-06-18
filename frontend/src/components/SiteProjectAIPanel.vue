<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { MagicStick, Promotion, CircleCheck } from '@element-plus/icons-vue'
import api, { AI_REQUEST_TIMEOUT, resolveApiError } from '@/api'
import AiChatImageInput from '@/components/AiChatImageInput.vue'
import AiChatVoiceButton from '@/components/AiChatVoiceButton.vue'
import { appendSpeechTranscript } from '@/composables/useSpeechInput'
import { resolveChatInputText, chatHistoryForApi } from '@/utils/aiChat'
import type { FileWriteSpec } from '@/types/siteProject'

interface ChatMsg {
  role: 'user' | 'assistant'
  content: string
  images?: string[]
  fileWrites?: FileWriteSpec[]
  summary?: string
  applied?: boolean
}

const props = withDefaults(defineProps<{
  siteId: number
  domain?: string
  focusPath?: string
  height?: string
}>(), {
  domain: '',
  focusPath: '',
  height: 'calc(100vh - 120px)',
})

const { t } = useI18n()
const scope = ref<'project' | 'file'>('project')
const chatInput = ref('')
const pendingImages = ref<string[]>([])
const chatLoading = ref(false)
const applying = ref(false)
const messages = ref<ChatMsg[]>([])
const chatBoxRef = ref<HTMLElement>()
const imageInputRef = ref<InstanceType<typeof AiChatImageInput>>()

const canSend = computed(() => (!!chatInput.value.trim() || pendingImages.value.length > 0) && !chatLoading.value)
const quickPrompts = computed(() => [
  t('siteModify.projectAiPromptTheme'),
  t('siteModify.projectAiPromptDark'),
  t('siteModify.projectAiPromptHome'),
])

async function scrollChat() {
  await nextTick()
  if (chatBoxRef.value) chatBoxRef.value.scrollTop = chatBoxRef.value.scrollHeight
}

function onComposerPaste(e: ClipboardEvent) {
  imageInputRef.value?.onPaste(e)
}

function onVoiceTranscript(text: string) {
  chatInput.value = appendSpeechTranscript(chatInput.value, text)
}

async function sendChat(preset?: string) {
  const text = resolveChatInputText(preset, chatInput.value)
  const images = [...pendingImages.value]
  if ((!text && !images.length) || chatLoading.value) return

  messages.value.push({
    role: 'user',
    content: text || t('siteModify.projectAiImageOnly'),
    images: images.length ? images : undefined,
  })
  chatInput.value = ''
  pendingImages.value = []
  chatLoading.value = true
  await scrollChat()
  try {
    const history = chatHistoryForApi(messages.value.slice(0, -1))
    const res: any = await api.post(
      `/websites/${props.siteId}/project/ai/chat`,
      {
        message: text,
        images,
        scope: scope.value,
        focus_path: props.focusPath || '',
        history,
      },
      { timeout: AI_REQUEST_TIMEOUT },
    )
    const data = res.data || {}
    messages.value.push({
      role: 'assistant',
      content: data.reply || t('siteModify.projectAiEmptyReply'),
      fileWrites: data.file_writes?.length ? data.file_writes : undefined,
      summary: data.summary,
    })
  } catch (e: any) {
    const errMsg = resolveApiError(e, t('siteModify.projectAiFailed'), t('common.requestTimeout'))
    ElMessage.error(errMsg)
    messages.value.push({ role: 'assistant', content: errMsg })
  } finally {
    chatLoading.value = false
    await scrollChat()
  }
}

async function applyWrites(msg: ChatMsg, index: number) {
  if (!msg.fileWrites?.length || applying.value) return
  try {
    await ElMessageBox.confirm(
      t('siteModify.projectAiApplyConfirm', { n: msg.fileWrites.length }),
      t('common.warning'),
      { type: 'warning' },
    )
  } catch {
    return
  }
  applying.value = true
  try {
    const res: any = await api.post(`/websites/${props.siteId}/project/ai/apply`, {
      file_writes: msg.fileWrites,
    })
    const written = res.data?.files_written || []
    if (written.length) {
      messages.value[index].applied = true
      ElMessage.success(t('siteModify.projectAiApplySuccess', { n: written.length }))
    } else {
      ElMessage.warning(t('siteModify.projectAiApplyPartial'))
    }
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('siteModify.projectAiApplyFailed')))
  } finally {
    applying.value = false
  }
}

function clearChat() {
  if (chatLoading.value) return
  messages.value = []
  chatInput.value = ''
  pendingImages.value = []
}
</script>

<template>
  <div class="site-project-ai" :style="{ height }">
    <div class="scope-bar">
      <el-radio-group v-model="scope" size="small">
        <el-radio-button value="project">{{ t('siteModify.projectAiScopeProject') }}</el-radio-button>
        <el-radio-button value="file" :disabled="!focusPath">{{ t('siteModify.projectAiScopeFile') }}</el-radio-button>
      </el-radio-group>
      <span class="scope-hint">{{ scope === 'project' ? t('siteModify.projectAiScopeProjectHint') : t('siteModify.projectAiScopeFileHint') }}</span>
    </div>

    <div ref="chatBoxRef" class="chat-messages">
      <div v-if="!messages.length" class="chat-empty">
        <el-icon class="welcome-icon"><MagicStick /></el-icon>
        <p>{{ t('siteModify.projectAiWelcome') }}</p>
        <div class="quick-prompts">
          <button v-for="p in quickPrompts" :key="p" type="button" class="prompt-chip" @click="sendChat(p)">{{ p }}</button>
        </div>
      </div>
      <div v-for="(msg, i) in messages" :key="i" class="chat-msg" :class="msg.role">
        <div class="msg-role">{{ msg.role === 'user' ? t('files.aiYou') : t('files.aiBot') }}</div>
        <div v-if="msg.images?.length" class="msg-images">
          <img v-for="(img, j) in msg.images" :key="j" :src="img" alt="" class="msg-image" />
        </div>
        <div v-if="msg.content" class="msg-body">{{ msg.content }}</div>
        <div v-if="msg.fileWrites?.length" class="file-writes">
          <p class="writes-title">{{ msg.summary || t('siteModify.projectAiPendingFiles', { n: msg.fileWrites.length }) }}</p>
          <ul class="writes-list">
            <li v-for="fw in msg.fileWrites" :key="fw.relative_path">{{ fw.relative_path }}</li>
          </ul>
          <el-button
            v-if="!msg.applied"
            type="primary"
            size="small"
            :icon="CircleCheck"
            :loading="applying"
            @click="applyWrites(msg, i)"
          >
            {{ t('siteModify.projectAiApply') }}
          </el-button>
          <el-tag v-else type="success" size="small">{{ t('siteModify.projectAiApplied') }}</el-tag>
        </div>
      </div>
      <div v-if="chatLoading" class="chat-msg assistant">
        <div class="msg-role">{{ t('files.aiBot') }}</div>
        <div class="msg-body typing">{{ t('files.aiThinkingProject') }}</div>
      </div>
    </div>

    <div class="chat-input-row" @paste="onComposerPaste">
      <AiChatImageInput ref="imageInputRef" v-model="pendingImages" :disabled="chatLoading" />
      <el-input
        v-model="chatInput"
        type="textarea"
        :rows="3"
        :placeholder="t('siteModify.projectAiPlaceholder')"
        :disabled="chatLoading"
        @keydown.ctrl.enter="sendChat()"
      />
      <div class="chat-actions">
        <div class="chat-actions-left">
          <AiChatVoiceButton :disabled="chatLoading" @transcript="onVoiceTranscript" />
          <el-button size="small" :disabled="!messages.length" @click="clearChat">{{ t('files.clearChat') }}</el-button>
        </div>
        <el-button
          class="chat-send-btn"
          type="primary"
          :icon="Promotion"
          :loading="chatLoading"
          :disabled="!canSend"
          @click="sendChat()"
        >
          {{ t('files.aiSend') }}
        </el-button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.site-project-ai {
  display: flex;
  flex-direction: column;
  min-height: 320px;
}
.scope-bar {
  flex-shrink: 0;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}
.scope-hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  flex: 1;
  min-width: 160px;
}
.chat-messages {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 8px 4px;
  background: var(--el-fill-color-lighter);
  border-radius: 10px;
}
.chat-empty {
  text-align: center;
  padding: 20px 12px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}
.welcome-icon {
  font-size: 28px;
  color: var(--el-color-primary);
  margin-bottom: 8px;
}
.quick-prompts {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  justify-content: center;
  margin-top: 12px;
}
.prompt-chip {
  border: 1px solid var(--el-border-color);
  background: var(--el-fill-color-blank);
  border-radius: 999px;
  padding: 4px 12px;
  font-size: 12px;
  cursor: pointer;
  color: var(--el-text-color-regular);
}
.prompt-chip:hover {
  border-color: var(--el-color-primary);
  color: var(--el-color-primary);
}
.chat-msg {
  margin-bottom: 12px;
  padding: 0 6px;
}
.chat-msg.user .msg-body {
  background: var(--el-color-primary-light-9);
  border-radius: 8px;
  padding: 8px 10px;
}
.chat-msg.assistant .msg-body {
  padding: 4px 2px;
  white-space: pre-wrap;
  line-height: 1.55;
  font-size: 13px;
}
.msg-images {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 6px;
}
.msg-image {
  width: 96px;
  height: 96px;
  object-fit: cover;
  border-radius: 8px;
  border: 1px solid var(--el-border-color);
}
.msg-role {
  font-size: 11px;
  color: var(--el-text-color-secondary);
  margin-bottom: 4px;
}
.file-writes {
  margin-top: 10px;
  padding: 10px;
  border-radius: 8px;
  border: 1px solid var(--el-color-primary-light-5);
  background: var(--el-color-primary-light-9);
}
.writes-title {
  margin: 0 0 6px;
  font-size: 12px;
  font-weight: 600;
}
.writes-list {
  margin: 0 0 8px;
  padding-left: 18px;
  font-size: 12px;
  font-family: Consolas, monospace;
}
.typing {
  color: var(--el-text-color-secondary);
  font-style: italic;
}
.chat-input-row {
  flex-shrink: 0;
  margin-top: 10px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.chat-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
}
.chat-actions-left {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.chat-send-btn {
  min-width: 92px;
}
.chat-actions :deep(.chat-send-btn.el-button--primary) {
  --el-button-bg-color: var(--cf-orange, #f6821f);
  --el-button-border-color: var(--cf-orange, #f6821f);
  --el-button-text-color: #fff;
  background-color: var(--cf-orange, #f6821f) !important;
  border-color: var(--cf-orange, #f6821f) !important;
  color: #fff !important;
}
.chat-actions :deep(.chat-send-btn.el-button--primary.is-disabled) {
  background-color: var(--el-color-primary-light-5, #fbc78a) !important;
  border-color: var(--el-color-primary-light-5, #fbc78a) !important;
  color: #fff !important;
  opacity: 0.85;
}
</style>
