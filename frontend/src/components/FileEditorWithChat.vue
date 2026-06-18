<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  ChatDotRound,
  Promotion,
  MagicStick,
  RefreshLeft,
  RefreshRight,
  View,
  CircleCheck,
  CircleClose,
  Clock,
} from '@element-plus/icons-vue'
import api, { AI_REQUEST_TIMEOUT, resolveApiError } from '@/api'
import FileCodeEditor from '@/components/FileCodeEditor.vue'
import CodeDiffView from '@/components/CodeDiffView.vue'
import type { FileWriteSpec } from '@/types/siteProject'
import { diffStats } from '@/utils/lineDiff'
import AiChatImageInput from '@/components/AiChatImageInput.vue'
import AiChatVoiceButton from '@/components/AiChatVoiceButton.vue'
import { appendSpeechTranscript } from '@/composables/useSpeechInput'
import { resolveChatInputText, chatHistoryForApi } from '@/utils/aiChat'

export interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
  images?: string[]
  suggestedContent?: string
  fileWrites?: FileWriteSpec[]
  summary?: string
  projectApplied?: boolean
  previewing?: boolean
  applied?: boolean
  rejected?: boolean
}

interface Checkpoint {
  id: string
  content: string
  label: string
  time: number
}

const props = defineProps<{
  modelValue: string
  path: string
  siteId?: number
  siteRoot?: string
  siteDomain?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [string]
  save: []
}>()

const { t } = useI18n()

const editorRef = ref<InstanceType<typeof FileCodeEditor>>()
const chatInput = ref('')
const pendingImages = ref<string[]>([])
const chatLoading = ref(false)
const applyingProject = ref(false)
const aiScope = ref<'file' | 'project'>('file')
const messages = ref<ChatMessage[]>([])
const chatBoxRef = ref<HTMLElement>()
const imageInputRef = ref<InstanceType<typeof AiChatImageInput>>()
const diffVisible = ref(false)
const canUndo = ref(false)
const canRedo = ref(false)

const checkpoints = ref<Checkpoint[]>([])
const openedContent = ref('')

/** AI 预览：before=应用前内容，after=建议内容 */
const pendingAI = ref<{ before: string; after: string; messageIndex: number } | null>(null)

const pendingStats = computed(() => {
  if (!pendingAI.value) return null
  return diffStats(pendingAI.value.before, pendingAI.value.after)
})

const hasPendingAI = computed(() => !!pendingAI.value)

const canSendChat = computed(() => {
  const hasText = !!chatInput.value.trim()
  const hasImages = pendingImages.value.length > 0
  if (props.siteId) return (hasText || hasImages) && !chatLoading.value
  return hasText && !chatLoading.value
})

const focusRelativePath = computed(() => {
  if (!props.siteRoot) return ''
  const root = props.siteRoot.replace(/\\/g, '/').replace(/\/+$/, '')
  const file = props.path.replace(/\\/g, '/')
  if (!file.startsWith(root + '/') && file !== root) return ''
  const rel = file.slice(root.length).replace(/^\//, '')
  return rel || ''
})

const showProjectScope = computed(() => !!props.siteId)

const chatPlaceholder = computed(() => {
  if (showProjectScope.value && aiScope.value === 'project') {
    return t('files.aiPlaceholderProject')
  }
  return t('files.aiPlaceholder')
})

const chatThinkingText = computed(() => {
  if (showProjectScope.value && aiScope.value === 'project') {
    return t('files.aiThinkingProject')
  }
  return t('files.aiThinking')
})

const quickPrompts = computed(() => {
  if (!showProjectScope.value || aiScope.value !== 'project') return []
  return [
    t('siteModify.projectAiPromptTheme'),
    t('siteModify.projectAiPromptDark'),
    t('siteModify.projectAiPromptHome'),
  ]
})

function sendQuickPrompt(text: string) {
  void sendChat(text)
}

function syncHistoryState() {
  canUndo.value = editorRef.value?.canUndo() ?? false
  canRedo.value = editorRef.value?.canRedo() ?? false
}

function pushCheckpoint(content: string, label: string) {
  checkpoints.value.push({
    id: `${Date.now()}-${Math.random().toString(36).slice(2, 7)}`,
    content,
    label,
    time: Date.now(),
  })
  if (checkpoints.value.length > 30) {
    checkpoints.value.shift()
  }
}

function resetSession(content: string) {
  messages.value = []
  chatInput.value = ''
  pendingImages.value = []
  pendingAI.value = null
  diffVisible.value = false
  openedContent.value = content
  checkpoints.value = [{
    id: 'open',
    content,
    label: t('files.cpOpened'),
    time: Date.now(),
  }]
}

watch(
  () => props.path,
  () => {
    resetSession(props.modelValue)
  }
)

watch(
  () => props.modelValue,
  (v, old) => {
    if (!openedContent.value && v) {
      resetSession(v)
      return
    }
    if (pendingAI.value && v !== pendingAI.value.after && v !== pendingAI.value.before) {
      pendingAI.value = null
      messages.value.forEach((m) => { m.previewing = false })
    }
    if (old !== undefined && v !== old && !pendingAI.value) {
      syncHistoryState()
    }
  },
  { immediate: true }
)

async function scrollChat() {
  await nextTick()
  if (chatBoxRef.value) {
    chatBoxRef.value.scrollTop = chatBoxRef.value.scrollHeight
  }
}

async function sendChat(preset?: string) {
  const text = resolveChatInputText(preset, chatInput.value)
  const images = [...pendingImages.value]
  if (props.siteId) {
    if ((!text && !images.length) || chatLoading.value) return
  } else if (!text || chatLoading.value) {
    return
  }

  if (pendingAI.value) {
    try {
      await ElMessageBox.confirm(t('files.aiPendingConfirm'), t('common.warning'), { type: 'warning' })
      rejectAI(false)
    } catch {
      return
    }
  }

  messages.value.push({
    role: 'user',
    content: text || (images.length ? t('siteModify.projectAiImageOnly') : text),
    images: images.length ? images : undefined,
  })
  chatInput.value = ''
  pendingImages.value = []
  chatLoading.value = true
  await scrollChat()

  try {
    const history = chatHistoryForApi(messages.value.slice(0, -1))
    if (props.siteId) {
      const scope = aiScope.value === 'project' ? 'project' : 'file'
      const res: any = await api.post(
        `/websites/${props.siteId}/project/ai/chat`,
        {
          message: text,
          images,
          scope,
          focus_path: focusRelativePath.value,
          history,
        },
        { timeout: AI_REQUEST_TIMEOUT },
      )
      const data = res.data || {}
      const msg: ChatMessage = {
        role: 'assistant',
        content: data.reply || t('siteModify.projectAiEmptyReply'),
        fileWrites: data.file_writes?.length ? data.file_writes : undefined,
        summary: data.summary,
      }
      messages.value.push(msg)
      const currentWrite = data.file_writes?.find((fw: FileWriteSpec) =>
        fw.relative_path === focusRelativePath.value
        || props.path.replace(/\\/g, '/').endsWith('/' + fw.relative_path),
      )
      if (currentWrite?.content) {
        startPreview(messages.value.length - 1, currentWrite.content)
      }
    } else {
      const res: any = await api.post('/files/ai/chat', {
        path: props.path,
        content: props.modelValue,
        message: text,
        history,
      }, { timeout: AI_REQUEST_TIMEOUT })
      const reply = res.data?.reply || ''
      const suggested = res.data?.suggested_content || ''
      const msg: ChatMessage = {
        role: 'assistant',
        content: reply,
        suggestedContent: suggested || undefined,
      }
      messages.value.push(msg)
      if (suggested) {
        startPreview(messages.value.length - 1, suggested)
      }
    }
  } catch (e: any) {
    const errMsg = resolveApiError(e, t('files.aiFailed'), t('common.requestTimeout'))
    ElMessage.error(errMsg)
    messages.value.push({ role: 'assistant', content: errMsg })
  } finally {
    chatLoading.value = false
    await scrollChat()
  }
}

function startPreview(messageIndex: number, suggested: string) {
  const before = props.modelValue
  pendingAI.value = { before, after: suggested, messageIndex }
  messages.value.forEach((m, i) => {
    m.previewing = i === messageIndex
    if (i === messageIndex) {
      m.applied = false
      m.rejected = false
    }
  })
  emit('update:modelValue', suggested)
  nextTick(syncHistoryState)
}

function acceptAI(showToast = true) {
  if (!pendingAI.value) return
  const { after, messageIndex } = pendingAI.value
  pushCheckpoint(after, t('files.cpAI'))
  if (messages.value[messageIndex]) {
    messages.value[messageIndex].previewing = false
    messages.value[messageIndex].applied = true
  }
  pendingAI.value = null
  emit('update:modelValue', after)
  if (showToast) ElMessage.success(t('files.aiAccepted'))
  nextTick(syncHistoryState)
}

function rejectAI(showToast = true) {
  if (!pendingAI.value) return
  const { before, messageIndex } = pendingAI.value
  if (messages.value[messageIndex]) {
    messages.value[messageIndex].previewing = false
    messages.value[messageIndex].rejected = true
  }
  pendingAI.value = null
  editorRef.value?.replaceAll(before, true)
  if (showToast) ElMessage.info(t('files.aiRejected'))
  nextTick(syncHistoryState)
}

function previewFromMessage(index: number) {
  const msg = messages.value[index]
  if (!msg?.suggestedContent) return
  if (pendingAI.value) rejectAI(false)
  startPreview(index, msg.suggestedContent)
}

function restoreCheckpoint(cp: Checkpoint) {
  if (pendingAI.value) rejectAI(false)
  editorRef.value?.replaceAll(cp.content, true)
  pushCheckpoint(cp.content, t('files.cpRestore', { label: cp.label }))
  ElMessage.success(t('files.cpRestored'))
}

function revertToOpened() {
  if (!openedContent.value) return
  if (pendingAI.value) rejectAI(false)
  editorRef.value?.replaceAll(openedContent.value, true)
  pushCheckpoint(openedContent.value, t('files.cpRevertOpen'))
  ElMessage.success(t('files.revertedToOpen'))
}

function undoEdit() {
  editorRef.value?.undoEdit()
  editorRef.value?.focusEditor()
  nextTick(syncHistoryState)
}

function redoEdit() {
  editorRef.value?.redoEdit()
  editorRef.value?.focusEditor()
  nextTick(syncHistoryState)
}

function openDiff() {
  if (!pendingAI.value) return
  diffVisible.value = true
}

function acceptAndCloseDiff() {
  acceptAI()
  diffVisible.value = false
}

function clearChat() {
  if (chatLoading.value) return
  if (messages.value.length === 0) return
  if (pendingAI.value) {
    rejectAI(false)
  }
  messages.value = []
  chatInput.value = ''
  pendingImages.value = []
}

function onComposerPaste(e: ClipboardEvent) {
  if (!showProjectScope.value) return
  imageInputRef.value?.onPaste(e)
}

function onVoiceTranscript(text: string) {
  chatInput.value = appendSpeechTranscript(chatInput.value, text)
}

async function applyProjectWrites(msg: ChatMessage, index: number) {
  if (!props.siteId || !msg.fileWrites?.length || applyingProject.value) return
  try {
    await ElMessageBox.confirm(
      t('siteModify.projectAiApplyConfirm', { n: msg.fileWrites.length }),
      t('common.warning'),
      { type: 'warning' },
    )
  } catch {
    return
  }
  applyingProject.value = true
  try {
    const res: any = await api.post(`/websites/${props.siteId}/project/ai/apply`, {
      file_writes: msg.fileWrites,
    })
    const written = res.data?.files_written || []
    if (written.length) {
      messages.value[index].projectApplied = true
      ElMessage.success(t('siteModify.projectAiApplySuccess', { n: written.length }))
      const currentWrite = msg.fileWrites.find((fw) =>
        fw.relative_path === focusRelativePath.value
        || props.path.replace(/\\/g, '/').endsWith('/' + fw.relative_path),
      )
      if (currentWrite?.content) {
        emit('update:modelValue', currentWrite.content)
      }
    } else {
      ElMessage.warning(t('siteModify.projectAiApplyPartial'))
    }
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('siteModify.projectAiApplyFailed')))
  } finally {
    applyingProject.value = false
  }
}
</script>

<template>
  <div class="editor-with-chat" :class="{ 'has-ai-preview': hasPendingAI }">
    <div class="editor-pane">
      <div class="pane-toolbar">
        <span class="file-path">{{ path }}</span>
        <div class="toolbar-actions">
          <el-button-group size="small">
            <el-button :icon="RefreshLeft" :disabled="!canUndo" @click="undoEdit">{{ t('files.editorUndo') }}</el-button>
            <el-button :icon="RefreshRight" :disabled="!canRedo" @click="redoEdit">{{ t('files.editorRedo') }}</el-button>
          </el-button-group>
          <el-dropdown trigger="click" @command="restoreCheckpoint">
            <el-button size="small" :icon="Clock">{{ t('files.editorHistory') }}</el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item v-for="cp in [...checkpoints].reverse()" :key="cp.id" :command="cp">
                  {{ cp.label }}
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
          <el-button size="small" @click="revertToOpened">{{ t('files.revertOpen') }}</el-button>
          <el-button type="primary" size="small" @click="emit('save')">{{ t('common.save') }}</el-button>
        </div>
      </div>

      <div v-if="hasPendingAI && pendingStats" class="ai-preview-bar">
        <div class="preview-meta">
          <el-tag type="warning" size="small">{{ t('files.aiPreviewMode') }}</el-tag>
          <span class="preview-stats">
            +{{ pendingStats.added }} / -{{ pendingStats.removed }}
          </span>
        </div>
        <div class="preview-actions">
          <el-button size="small" :icon="View" @click="openDiff">{{ t('files.viewDiff') }}</el-button>
          <el-button size="small" type="danger" plain :icon="CircleClose" @click="rejectAI()">{{ t('files.aiReject') }}</el-button>
          <el-button size="small" type="success" :icon="CircleCheck" @click="acceptAI()">{{ t('files.aiAccept') }}</el-button>
        </div>
      </div>

      <FileCodeEditor
        ref="editorRef"
        :model-value="modelValue"
        :path="path"
        @update:model-value="emit('update:modelValue', $event)"
        @history-change="syncHistoryState"
      />
    </div>

    <div class="chat-pane">
      <div class="chat-header">
        <div class="chat-header-main">
          <el-icon><ChatDotRound /></el-icon>
          <span>{{ t('files.aiAssistant') }}</span>
          <el-tag v-if="siteDomain && showProjectScope" size="small" type="info" class="site-tag">
            {{ t('files.aiSiteBound', { domain: siteDomain }) }}
          </el-tag>
        </div>
        <el-tag v-if="hasPendingAI" size="small" type="warning">{{ t('files.aiPending') }}</el-tag>
      </div>

      <div class="scope-section">
        <div class="scope-label">{{ t('files.aiScopeTitle') }}</div>
        <template v-if="showProjectScope">
          <el-radio-group v-model="aiScope" size="small" class="scope-toggle">
            <el-radio-button value="file">{{ t('siteModify.projectAiScopeFile') }}</el-radio-button>
            <el-radio-button value="project">{{ t('siteModify.projectAiScopeProject') }}</el-radio-button>
          </el-radio-group>
          <p class="scope-hint">
            {{ aiScope === 'project' ? t('siteModify.projectAiScopeProjectHint') : t('siteModify.projectAiScopeFileHint') }}
          </p>
          <p v-if="aiScope === 'project'" class="scope-speed-hint">{{ t('files.aiScopeSpeedHint') }}</p>
        </template>
        <el-alert v-else type="info" :closable="false" show-icon class="scope-alert">
          {{ t('files.aiNoSiteContext') }}
        </el-alert>
      </div>

      <div ref="chatBoxRef" class="chat-messages">
        <div v-if="messages.length === 0" class="chat-empty">
          <el-icon v-if="showProjectScope && aiScope === 'project'" class="welcome-icon"><MagicStick /></el-icon>
          <p class="empty-text">
            {{ showProjectScope && aiScope === 'project' ? t('siteModify.projectAiWelcome') : t('files.aiHint') }}
          </p>
          <div v-if="quickPrompts.length" class="quick-prompts">
            <button
              v-for="p in quickPrompts"
              :key="p"
              type="button"
              class="prompt-chip"
              @click="sendQuickPrompt(p)"
            >
              {{ p }}
            </button>
          </div>
        </div>
        <div v-for="(msg, i) in messages" :key="i" class="chat-msg" :class="[msg.role, { previewing: msg.previewing }]">
          <div class="msg-role">{{ msg.role === 'user' ? t('files.aiYou') : t('files.aiBot') }}</div>
          <div v-if="msg.images?.length" class="msg-images">
            <img v-for="(img, j) in msg.images" :key="j" :src="img" alt="" class="msg-image" />
          </div>
          <div class="msg-body">{{ msg.content }}</div>
          <div v-if="msg.fileWrites?.length" class="msg-project-writes">
            <p class="writes-title">{{ msg.summary || t('siteModify.projectAiPendingFiles', { n: msg.fileWrites.length }) }}</p>
            <ul class="writes-list">
              <li v-for="fw in msg.fileWrites" :key="fw.relative_path">{{ fw.relative_path }}</li>
            </ul>
            <el-button
              v-if="!msg.projectApplied"
              type="primary"
              size="small"
              :icon="CircleCheck"
              :loading="applyingProject"
              @click="applyProjectWrites(msg, i)"
            >
              {{ t('siteModify.projectAiApply') }}
            </el-button>
            <el-tag v-else type="success" size="small">{{ t('siteModify.projectAiApplied') }}</el-tag>
          </div>
          <div v-if="msg.suggestedContent" class="msg-actions">
            <el-tag v-if="msg.applied" size="small" type="success">{{ t('files.aiAppliedTag') }}</el-tag>
            <el-tag v-else-if="msg.rejected" size="small" type="info">{{ t('files.aiRejectedTag') }}</el-tag>
            <el-tag v-else-if="msg.previewing" size="small" type="warning">{{ t('files.aiPreviewingTag') }}</el-tag>
            <template v-if="!msg.applied && !msg.rejected && !msg.previewing">
              <el-button size="small" type="primary" plain :icon="MagicStick" @click="previewFromMessage(i)">{{ t('files.aiPreview') }}</el-button>
              <el-button size="small" type="success" plain :icon="CircleCheck" @click="startPreview(i, msg.suggestedContent!); acceptAI()">{{ t('files.aiApply') }}</el-button>
            </template>
            <el-button v-if="msg.previewing" size="small" :icon="View" @click="openDiff">{{ t('files.viewDiff') }}</el-button>
          </div>
        </div>
        <div v-if="chatLoading" class="chat-msg assistant">
          <div class="msg-role">{{ t('files.aiBot') }}</div>
          <div class="msg-body typing">{{ chatThinkingText }}</div>
        </div>
      </div>

      <div class="chat-input-row">
        <div class="chat-op-bar">
          <el-button size="small" :icon="RefreshLeft" :disabled="!canUndo" @click="undoEdit">{{ t('files.editorUndo') }}</el-button>
          <el-button size="small" :icon="RefreshRight" :disabled="!canRedo" @click="redoEdit">{{ t('files.editorRedo') }}</el-button>
          <template v-if="hasPendingAI">
            <el-button size="small" type="success" :icon="CircleCheck" @click="acceptAI()">{{ t('files.aiAccept') }}</el-button>
            <el-button size="small" type="danger" plain :icon="CircleClose" @click="rejectAI()">{{ t('files.aiReject') }}</el-button>
          </template>
          <el-button size="small" :disabled="!messages.length" @click="clearChat">{{ t('files.clearChat') }}</el-button>
        </div>
        <div class="chat-composer" @paste="onComposerPaste">
          <AiChatImageInput v-if="showProjectScope" ref="imageInputRef" v-model="pendingImages" :disabled="chatLoading" />
          <el-input
            v-model="chatInput"
            type="textarea"
            :rows="4"
            :placeholder="chatPlaceholder"
            :disabled="chatLoading"
            @keydown.ctrl.enter="sendChat()"
          />
          <div class="composer-footer">
            <AiChatVoiceButton :disabled="chatLoading" @transcript="onVoiceTranscript" />
            <el-button
              class="chat-send-btn"
              type="primary"
              :icon="Promotion"
              :loading="chatLoading"
              :disabled="!canSendChat"
              @click="sendChat()"
            >
              {{ t('files.aiSend') }}
            </el-button>
          </div>
        </div>
      </div>
    </div>

    <el-dialog v-model="diffVisible" :title="t('files.diffTitle')" width="85%" top="4vh" destroy-on-close>
      <CodeDiffView v-if="pendingAI" :before="pendingAI.before" :after="pendingAI.after" />
      <template #footer>
        <el-button @click="diffVisible = false">{{ t('common.close') }}</el-button>
        <el-button type="danger" plain @click="rejectAI(); diffVisible = false">{{ t('files.aiReject') }}</el-button>
        <el-button type="success" @click="acceptAndCloseDiff">{{ t('files.aiAccept') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.editor-with-chat {
  display: grid;
  grid-template-columns: 1fr minmax(380px, 420px);
  gap: 0;
  height: 100%;
  min-height: 480px;
  max-height: 100%;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  overflow: hidden;
}
.editor-with-chat.has-ai-preview {
  border-color: var(--el-color-warning);
  box-shadow: 0 0 0 1px var(--el-color-warning-light-7);
}

.editor-pane {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  border-right: 1px solid var(--el-border-color-light);
}

.pane-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 8px 12px;
  background: var(--el-fill-color-light);
  border-bottom: 1px solid var(--el-border-color-lighter);
  flex-wrap: wrap;
}

.file-path {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  word-break: break-all;
  flex: 1;
  min-width: 120px;
}

.toolbar-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.ai-preview-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 8px 12px;
  background: var(--el-color-warning-light-9);
  border-bottom: 1px solid var(--el-color-warning-light-5);
  flex-wrap: wrap;
}

.preview-meta {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
}

.preview-stats {
  font-family: monospace;
  color: var(--el-text-color-secondary);
}

.preview-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.editor-pane :deep(.code-editor) {
  flex: 1;
  min-height: 0;
  max-height: none;
  border-radius: 0;
}

.editor-pane :deep(.code-editor .cm-editor) {
  min-height: 100%;
  height: 100%;
}

.chat-pane {
  display: flex;
  flex-direction: column;
  background: var(--el-bg-color);
  min-width: 0;
  min-height: 0;
  overflow: hidden;
}

.chat-header {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 12px 14px;
  font-weight: 600;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.chat-header-main {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  flex-wrap: wrap;
}
.site-tag {
  font-weight: 400;
}

.scope-section {
  flex-shrink: 0;
  padding: 10px 12px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  background: var(--el-fill-color-lighter);
}
.scope-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  margin-bottom: 8px;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.scope-toggle {
  width: 100%;
  display: flex;
}
.scope-toggle :deep(.el-radio-button) {
  flex: 1;
}
.scope-toggle :deep(.el-radio-button__inner) {
  width: 100%;
}
.scope-hint {
  margin: 8px 0 0;
  font-size: 12px;
  line-height: 1.45;
  color: var(--el-text-color-secondary);
}
.scope-speed-hint {
  margin: 4px 0 0;
  font-size: 11px;
  line-height: 1.4;
  color: var(--el-color-warning);
}
.scope-alert {
  margin: 0;
}
.scope-alert :deep(.el-alert__content) {
  font-size: 12px;
  line-height: 1.45;
}

.chat-messages {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  background: var(--el-fill-color-blank);
}

.chat-empty {
  text-align: center;
  padding: 16px 8px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
  line-height: 1.55;
}
.welcome-icon {
  font-size: 28px;
  color: var(--el-color-primary);
  margin-bottom: 8px;
}
.empty-text {
  margin: 0;
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

.chat-header-old-remove {
  display: none;
}

.msg-project-writes {
  margin-top: 8px;
  padding: 8px;
  border-radius: 6px;
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
  font-size: 11px;
  font-family: Consolas, monospace;
}

.pending-tag { margin-left: auto; }

.chat-msg {
  padding: 10px 12px;
  border-radius: 8px;
  font-size: 13px;
  line-height: 1.55;
}

.chat-msg.user {
  background: var(--el-color-primary-light-9);
  align-self: flex-end;
  max-width: 95%;
}

.chat-msg.assistant {
  background: var(--el-fill-color-light);
  align-self: stretch;
}

.chat-msg.previewing {
  outline: 2px solid var(--el-color-warning);
}

.msg-role {
  font-size: 11px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  margin-bottom: 4px;
}

.msg-body {
  white-space: pre-wrap;
  word-break: break-word;
}

.msg-body.typing {
  color: var(--el-text-color-secondary);
  font-style: italic;
}

.msg-images {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 6px;
}
.msg-image {
  width: 72px;
  height: 72px;
  object-fit: cover;
  border-radius: 6px;
  border: 1px solid var(--el-border-color);
}

.msg-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  margin-top: 8px;
}

.chat-input-row {
  flex-shrink: 0;
  padding: 12px;
  border-top: 1px solid var(--el-border-color-lighter);
  display: flex;
  flex-direction: column;
  gap: 10px;
  background: var(--el-bg-color);
}

.chat-op-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.op-hint {
  margin: 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.chat-composer {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px 12px;
  border-top: 1px solid var(--el-border-color-lighter);
  background: var(--el-fill-color-lighter);
}

.composer-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.chat-composer :deep(.el-textarea__inner) {
  min-height: 88px;
}

.chat-send-btn {
  flex-shrink: 0;
  min-width: 92px;
  height: 36px;
}

.chat-actions :deep(.chat-send-btn.el-button--primary),
.chat-composer :deep(.chat-send-btn.el-button--primary) {
  --el-button-bg-color: var(--cf-orange, #f6821f);
  --el-button-border-color: var(--cf-orange, #f6821f);
  --el-button-text-color: #fff;
  --el-button-hover-bg-color: var(--cf-orange-hover, #e56f10);
  --el-button-hover-border-color: var(--cf-orange-hover, #e56f10);
  --el-button-hover-text-color: #fff;
  --el-button-active-bg-color: #d96a0a;
  --el-button-active-border-color: #d96a0a;
  --el-button-active-text-color: #fff;
  background-color: var(--cf-orange, #f6821f) !important;
  border-color: var(--cf-orange, #f6821f) !important;
  color: #fff !important;
}

.chat-actions :deep(.chat-send-btn.el-button--primary.is-disabled),
.chat-composer :deep(.chat-send-btn.el-button--primary.is-disabled) {
  background-color: var(--el-color-primary-light-5, #fbc78a) !important;
  border-color: var(--el-color-primary-light-5, #fbc78a) !important;
  color: #fff !important;
  opacity: 0.85;
}

@media (max-width: 960px) {
  .editor-with-chat {
    grid-template-columns: 1fr;
    grid-template-rows: 1fr 320px;
    height: auto;
  }
  .editor-pane {
    border-right: none;
    border-bottom: 1px solid var(--el-border-color-light);
  }
}
</style>
