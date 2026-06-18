<script setup lang="ts">
import { nextTick, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Promotion, MagicStick } from '@element-plus/icons-vue'
import api, { AI_REQUEST_TIMEOUT, resolveApiError } from '@/api'

export interface RuleDraft {
  name: string
  pattern: string
  action: string
  ttl_minutes: number
  website_id: number
  priority: number
  enabled: boolean
}

interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
  suggestedRule?: RuleDraft
}

const props = defineProps<{
  sites: any[]
  rules: any[]
  config?: Record<string, unknown>
  compact?: boolean
}>()

const emit = defineEmits<{
  apply: [rule: RuleDraft]
}>()

const { t } = useI18n()

const chatInput = ref('')
const chatLoading = ref(false)
const messages = ref<ChatMessage[]>([])
const chatBoxRef = ref<HTMLElement>()

const templates = [
  { key: 'admin', icon: '🔒' },
  { key: 'api', icon: '⚡' },
  { key: 'login', icon: '👤' },
  { key: 'cart', icon: '🛒' },
  { key: 'search', icon: '🔍' },
  { key: 'json', icon: '{ }' },
]

async function scrollChat() {
  await nextTick()
  if (chatBoxRef.value) {
    chatBoxRef.value.scrollTop = chatBoxRef.value.scrollHeight
  }
}

function buildContext(draft?: RuleDraft) {
  return {
    sites: props.sites.map((s) => ({ id: s.id, domain: s.domain })),
    existing_rules: props.rules.map((r) => ({
      name: r.name,
      pattern: r.pattern,
      action: r.action,
      ttl_minutes: r.ttl_minutes || 0,
      website_id: r.website_id || 0,
      priority: r.priority ?? 100,
      enabled: r.enabled !== false,
    })),
    form_draft: draft,
    global_config: props.config ? {
      enabled: props.config.enabled,
      dev_mode: props.config.dev_mode,
      bypass_paths: props.config.bypass_paths,
      bypass_cookies: props.config.bypass_cookies,
    } : undefined,
  }
}

function templateRule(key: string): RuleDraft {
  const map: Record<string, RuleDraft> = {
    admin: { name: t('cache.tplAdminName'), pattern: '/admin|/wp-admin|/user/login', action: 'bypass', ttl_minutes: 0, website_id: 0, priority: 10, enabled: true },
    api: { name: t('cache.tplApiName'), pattern: '^/api/|/graphql|/webhook', action: 'bypass', ttl_minutes: 0, website_id: 0, priority: 20, enabled: true },
    login: { name: t('cache.tplLoginName'), pattern: '/login|/register|/signin|/auth/', action: 'bypass', ttl_minutes: 0, website_id: 0, priority: 30, enabled: true },
    cart: { name: t('cache.tplCartName'), pattern: '/cart|/checkout|/order|/payment', action: 'bypass', ttl_minutes: 0, website_id: 0, priority: 40, enabled: true },
    search: { name: t('cache.tplSearchName'), pattern: '/search|/s=|\\?s=', action: 'bypass', ttl_minutes: 0, website_id: 0, priority: 60, enabled: true },
    json: { name: t('cache.tplJsonName'), pattern: '\\.json$|/feed/|/sitemap', action: 'bypass', ttl_minutes: 0, website_id: 0, priority: 50, enabled: true },
  }
  return map[key] || map.api
}

function applyTemplate(key: string) {
  const rule = templateRule(key)
  emit('apply', rule)
  messages.value.push({
    role: 'assistant',
    content: t('cache.tplApplied', { name: rule.name }),
    suggestedRule: rule,
  })
  scrollChat()
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
    const res: any = await api.post('/cache/rules/ai/suggest', {
      message: text,
      history,
      context: buildContext(),
    }, { timeout: AI_REQUEST_TIMEOUT })
    const reply = res.data?.reply || ''
    const suggested = res.data?.suggested_rule
    messages.value.push({
      role: 'assistant',
      content: reply,
      suggestedRule: suggested || undefined,
    })
  } catch (e: any) {
    const errMsg = resolveApiError(e, t('cache.aiFailed'), t('common.requestTimeout'))
    ElMessage.error(errMsg)
    messages.value.push({ role: 'assistant', content: errMsg })
  } finally {
    chatLoading.value = false
    await scrollChat()
  }
}

function applySuggestion(rule?: RuleDraft) {
  if (!rule) return
  emit('apply', { ...rule })
  ElMessage.success(t('cache.ruleAppliedToForm'))
}

function askExample(promptKey: string) {
  chatInput.value = t(`cache.aiPrompt${promptKey}`)
  sendChat()
}

defineExpose({ clearChat: () => { messages.value = [] } })
</script>

<template>
  <div class="rule-assistant" :class="{ compact }">
    <div v-if="!compact" class="tpl-section">
      <div class="tpl-title">{{ t('cache.smartTemplates') }}</div>
      <div class="tpl-grid">
        <button v-for="tpl in templates" :key="tpl.key" type="button" class="tpl-chip" @click="applyTemplate(tpl.key)">
          <span class="tpl-icon">{{ tpl.icon }}</span>
          <span>{{ t(`cache.tpl${tpl.key.charAt(0).toUpperCase()}${tpl.key.slice(1)}`) }}</span>
        </button>
      </div>
    </div>

    <div class="chat-header">
      <el-icon><MagicStick /></el-icon>
      <span>{{ t('cache.ruleAssistant') }}</span>
    </div>

    <div ref="chatBoxRef" class="chat-box">
      <div v-if="messages.length === 0" class="chat-empty">
        <p>{{ t('cache.aiWelcome') }}</p>
        <div class="quick-prompts">
          <el-button size="small" text type="primary" @click="askExample('Admin')">{{ t('cache.aiQuickAdmin') }}</el-button>
          <el-button size="small" text type="primary" @click="askExample('Wordpress')">{{ t('cache.aiQuickWordpress') }}</el-button>
          <el-button size="small" text type="primary" @click="askExample('Ecommerce')">{{ t('cache.aiQuickEcommerce') }}</el-button>
        </div>
      </div>
      <div v-for="(msg, idx) in messages" :key="idx" class="chat-msg" :class="msg.role">
        <div class="msg-role">{{ msg.role === 'user' ? t('cache.you') : t('cache.assistant') }}</div>
        <div class="msg-body">{{ msg.content }}</div>
        <el-button
          v-if="msg.suggestedRule"
          size="small"
          type="primary"
          plain
          class="apply-btn"
          @click="applySuggestion(msg.suggestedRule)"
        >
          {{ t('cache.applyToForm') }}
        </el-button>
      </div>
      <div v-if="chatLoading" class="chat-msg assistant">
        <div class="msg-role">{{ t('cache.assistant') }}</div>
        <div class="msg-body typing">{{ t('cache.aiThinking') }}</div>
      </div>
    </div>

    <div class="chat-input-row">
      <el-input
        v-model="chatInput"
        type="textarea"
        :rows="compact ? 2 : 3"
        :placeholder="t('cache.aiInputPlaceholder')"
        @keydown.ctrl.enter="sendChat"
      />
      <div class="chat-actions">
        <span class="hint">{{ t('cache.aiCtrlEnter') }}</span>
        <el-button type="primary" :loading="chatLoading" :icon="Promotion" @click="sendChat">
          {{ t('cache.aiSend') }}
        </el-button>
      </div>
    </div>

    <el-alert type="info" :closable="false" show-icon class="ai-note">
      <template #title>{{ t('cache.aiNoteTitle') }}</template>
      {{ t('cache.aiNoteBody') }}
    </el-alert>
  </div>
</template>

<style scoped>
.rule-assistant {
  display: flex;
  flex-direction: column;
  gap: 10px;
  height: 100%;
  min-height: 320px;
  overflow: hidden;
}
.rule-assistant.compact { min-height: 280px; }
.tpl-section { margin-bottom: 4px; }
.tpl-title { font-size: 12px; font-weight: 600; color: var(--el-text-color-secondary); margin-bottom: 8px; }
.tpl-grid { display: flex; flex-wrap: wrap; gap: 8px; }
.tpl-chip {
  display: inline-flex; align-items: center; gap: 6px;
  padding: 6px 10px; border: 1px solid var(--el-border-color); border-radius: 999px;
  background: var(--el-fill-color-blank); font-size: 12px; cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
}
.tpl-chip:hover { border-color: var(--el-color-primary); background: var(--el-color-primary-light-9); }
.tpl-icon { font-size: 14px; }
.chat-header {
  display: flex; align-items: center; gap: 6px; font-weight: 600; font-size: 14px;
  padding-bottom: 6px; border-bottom: 1px solid var(--el-border-color-lighter);
}
.chat-box {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 8px 4px;
  background: var(--el-fill-color-lighter);
  border-radius: 8px;
}
.chat-empty { font-size: 13px; color: var(--el-text-color-secondary); padding: 8px; }
.quick-prompts { display: flex; flex-wrap: wrap; gap: 4px; margin-top: 8px; }
.chat-msg { margin-bottom: 12px; padding: 0 6px; }
.chat-msg.user .msg-body { background: var(--el-color-primary-light-9); border-radius: 8px; padding: 8px 10px; }
.chat-msg.assistant .msg-body { padding: 4px 2px; white-space: pre-wrap; line-height: 1.5; font-size: 13px; }
.msg-role { font-size: 11px; color: var(--el-text-color-secondary); margin-bottom: 4px; }
.apply-btn { margin-top: 6px; }
.typing { color: var(--el-text-color-secondary); font-style: italic; }
.chat-input-row { flex-shrink: 0; display: flex; flex-direction: column; gap: 8px; }
.chat-actions { display: flex; justify-content: space-between; align-items: center; }
.hint { font-size: 11px; color: var(--el-text-color-placeholder); }
.ai-note { margin-top: 4px; }
.ai-note :deep(.el-alert__title) { font-size: 12px; }
</style>
