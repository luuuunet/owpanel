<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import api, { AI_REQUEST_TIMEOUT, resolveApiError } from '@/api'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  MagicStick, Plus, Promotion, Close, MoreFilled,
  FullScreen, Fold, Expand, Connection, Star, StarFilled, Delete,
} from '@element-plus/icons-vue'
import TerminalPane from '@/components/TerminalPane.vue'
import TerminalPamPanel from '@/components/TerminalPamPanel.vue'
import { createSession, type TerminalSession, type TerminalTarget } from '@/types/terminal'
import {
  applySaved, loadPrefs, loadSavedConnections, persistPrefs, persistSavedConnections,
  sessionToSaved, type SavedConnection, type TerminalPrefs,
} from '@/composables/useTerminalStorage'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

type MainTab = 'shell' | 'pam'
const mainTab = ref<MainTab>('shell')

interface SSHKeyItem {
  id: number
  name: string
  has_private?: boolean
  remark?: string
}

const targets = ref<TerminalTarget[]>([])
const bastionTargets = ref<TerminalTarget[]>([])
const sshKeys = ref<SSHKeyItem[]>([])
const savedConnections = ref<SavedConnection[]>([])
const sessions = ref<TerminalSession[]>([])
const activeSessionId = ref('')
const paneRefs = ref<Record<string, InstanceType<typeof TerminalPane> | null>>({})
const prefs = ref<TerminalPrefs>(loadPrefs())
const fullscreen = ref(false)
const pageRef = ref<HTMLElement>()

const aiOpen = ref(false)
const keysDialog = ref(false)
const keyForm = ref({ name: '', public_key: '', private_key: '', remark: '' })
const chatInput = ref('')
const chatLoading = ref(false)
const chatMessages = ref<{ role: string; content: string; suggestedCommand?: string }[]>([])
const chatBoxRef = ref<HTMLElement>()
const nowTick = ref(Date.now())

const activeSession = computed(() => sessions.value.find((s) => s.id === activeSessionId.value))
const savedKeys = computed(() => sshKeys.value.filter((k) => k.has_private))
const connectedCount = computed(() => sessions.value.filter((s) => s.connected).length)
const quickTargets = computed(() => targets.value.filter((x) => x.id !== 'custom'))
const allTargets = computed(() => [...targets.value, ...bastionTargets.value])

const statusText = computed(() => {
  const s = activeSession.value
  if (!s) return ''
  if (s.connected && s.connectedAt) {
    const sec = Math.floor((nowTick.value - s.connectedAt) / 1000)
    const m = Math.floor(sec / 60)
    const h = Math.floor(m / 60)
    const dur = h > 0 ? `${h}h ${m % 60}m` : m > 0 ? `${m}m ${sec % 60}s` : `${sec}s`
    return `${s.user}@${s.host}:${s.port} · ${dur}`
  }
  return `${s.user}@${s.host}:${s.port} · ${t('terminalPage.notConnected')}`
})

let tickTimer: ReturnType<typeof setInterval> | null = null

function setPaneRef(id: string, el: unknown) {
  paneRefs.value[id] = (el as InstanceType<typeof TerminalPane>) || null
}

function savePrefs() {
  persistPrefs(prefs.value)
}

function updateSession(id: string, patch: Partial<TerminalSession>) {
  const s = sessions.value.find((x) => x.id === id)
  if (s) Object.assign(s, patch)
}

function onSessionActivity(id: string) {
  if (id !== activeSessionId.value) {
    updateSession(id, { unread: true })
  }
}

function onTargetChange(session: TerminalSession, targetId: string) {
  const tg = targets.value.find((x) => x.id === targetId) || bastionTargets.value.find((x) => x.id === targetId)
  if (!tg || tg.id === 'custom') return
  session.host = tg.host
  session.port = tg.port
  session.user = tg.user
  session.assetId = tg.asset_id || null
  session.accountId = tg.account_id || null
  if (!tg.has_password) session.password = ''
  if (!session.connected) session.title = tg.label
}

function addSession(fromTarget?: TerminalTarget, saved?: SavedConnection) {
  const s = createSession(sessions.value.length + 1, fromTarget)
  if (saved) applySaved(s, saved)
  sessions.value.push(s)
  activeSessionId.value = s.id
  return s
}

async function quickConnect(target?: TerminalTarget, saved?: SavedConnection) {
  const s = addSession(target, saved)
  await nextTick()
  try {
    await paneRefs.value[s.id]?.connect()
  } catch (e: any) {
    ElMessage.warning(e?.message || t('common.failed'))
  }
}

async function closeSession(id: string) {
  const idx = sessions.value.findIndex((s) => s.id === id)
  if (idx < 0) return
  paneRefs.value[id]?.disconnect()
  sessions.value.splice(idx, 1)
  delete paneRefs.value[id]
  if (activeSessionId.value === id) {
    activeSessionId.value = sessions.value[Math.min(idx, sessions.value.length - 1)]?.id || ''
  }
  if (!sessions.value.length) addSession(targets.value.find((x) => x.id === 'local'))
}

async function closeOthers(id: string) {
  for (const oid of sessions.value.filter((s) => s.id !== id).map((s) => s.id)) {
    await closeSession(oid)
  }
}

async function closeAll() {
  for (const id of [...sessions.value.map((s) => s.id)]) await closeSession(id)
}

function duplicateSession(id: string) {
  const src = sessions.value.find((s) => s.id === id)
  if (!src) return
  const copy = createSession(sessions.value.length + 1)
  Object.assign(copy, { ...src, id: copy.id, title: src.title + t('terminalPage.copySuffix'), connected: false, connecting: false, unread: false, connectedAt: undefined })
  sessions.value.push(copy)
  activeSessionId.value = copy.id
}

async function renameSession(id: string) {
  const s = sessions.value.find((x) => x.id === id)
  if (!s) return
  const { value } = await ElMessageBox.prompt(t('terminalPage.renamePrompt'), t('terminalPage.rename'), {
    inputValue: s.title,
    confirmButtonText: t('common.confirm'),
    cancelButtonText: t('common.cancel'),
  })
  if (value?.trim()) s.title = value.trim()
}

async function handleTabMenu(cmd: string, id: string) {
  switch (cmd) {
    case 'close': await closeSession(id); break
    case 'closeOthers': await closeOthers(id); break
    case 'closeAll': await closeAll(); break
    case 'duplicate': duplicateSession(id); break
    case 'rename': await renameSession(id); break
    case 'clear': paneRefs.value[id]?.clearScreen(); break
    case 'reconnect':
      try { await paneRefs.value[id]?.reconnect() } catch (e: any) { ElMessage.warning(e?.message || t('common.failed')) }
      break
  }
}

async function connectActive() {
  const pane = paneRefs.value[activeSessionId.value]
  if (!pane) return
  try { await pane.connect() } catch (e: any) { ElMessage.warning(e?.message || t('common.failed')) }
}

function disconnectActive() {
  paneRefs.value[activeSessionId.value]?.disconnect()
}

function sendCommand(cmd: string) {
  if (!paneRefs.value[activeSessionId.value]?.sendCommand(cmd)) {
    ElMessage.warning(t('terminalPage.connectFirst'))
  }
}

function bookmarkActive() {
  const s = activeSession.value
  if (!s) return
  const item = sessionToSaved(s)
  savedConnections.value.unshift(item)
  persistSavedConnections(savedConnections.value)
  ElMessage.success(t('terminalPage.savedOk'))
}

function removeSaved(id: string) {
  savedConnections.value = savedConnections.value.filter((x) => x.id !== id)
  persistSavedConnections(savedConnections.value)
}

function isBookmarked(s: TerminalSession) {
  return savedConnections.value.some((x) => x.host === s.host && x.port === s.port && x.user === s.user)
}

function toggleFullscreen() {
  fullscreen.value = !fullscreen.value
}

function changeFontSize(delta: number) {
  prefs.value.fontSize = Math.min(22, Math.max(11, prefs.value.fontSize + delta))
  savePrefs()
}

function switchTab(offset: number) {
  const idx = sessions.value.findIndex((s) => s.id === activeSessionId.value)
  if (idx < 0) return
  const next = sessions.value[(idx + offset + sessions.value.length) % sessions.value.length]
  if (next) activeSessionId.value = next.id
}

function onKeydown(ev: KeyboardEvent) {
  if (ev.ctrlKey && ev.shiftKey && ev.key === 'T') { ev.preventDefault(); addSession(); return }
  if (ev.ctrlKey && ev.shiftKey && ev.key === 'W') { ev.preventDefault(); closeSession(activeSessionId.value); return }
  if (ev.ctrlKey && ev.shiftKey && (ev.key === ']' || ev.key === '}')) { ev.preventDefault(); switchTab(1); return }
  if (ev.ctrlKey && ev.shiftKey && (ev.key === '[' || ev.key === '{')) { ev.preventDefault(); switchTab(-1); return }
  if (ev.key === 'F11') { ev.preventDefault(); toggleFullscreen(); return }
}

async function loadAll() {
  const [tRes, kRes, bRes]: any[] = await Promise.all([
    api.get('/terminal/targets').catch(() => ({ data: [] })),
    api.get('/terminal/keys').catch(() => ({ data: [] })),
    api.get('/bastion/connect-targets').catch(() => ({ data: [] })),
  ])
  targets.value = tRes.data || []
  if (!targets.value.some((x: TerminalTarget) => x.id === 'custom')) {
    targets.value.push({ id: 'custom', label: t('terminalPage.custom'), host: '', port: 22, user: 'root' })
  }
  sshKeys.value = kRes.data || []
  bastionTargets.value = (bRes.data || []).map((a: any) => ({
    id: a.account_id ? `asset-${a.asset_id}-acc-${a.account_id}` : `asset-${a.asset_id || a.id}`,
    label: a.label,
    host: a.host,
    port: a.port,
    user: a.user,
    asset_id: a.asset_id || a.id,
    account_id: a.account_id || undefined,
    has_password: a.has_password,
    permission: a.permission,
  }))
  savedConnections.value = loadSavedConnections()
}

async function connectFromQuery(query: typeof route.query) {
  const assetQ = query.asset_id
  const accountQ = query.account_id
  if (!assetQ) return false
  await loadAll()
  const tg = bastionTargets.value.find((x) => {
    if (accountQ) return String(x.asset_id) === String(assetQ) && String(x.account_id) === String(accountQ)
    return String(x.asset_id) === String(assetQ) && !x.account_id
  }) || bastionTargets.value.find((x) => String(x.asset_id) === String(assetQ))
  if (!tg) return false
  mainTab.value = 'shell'
  await quickConnect(tg)
  return true
}

async function onPamConnect(query: Record<string, string>) {
  mainTab.value = 'shell'
  await router.replace({ path: '/terminal', query: { ...query, tab: undefined } })
  await connectFromQuery(query)
}

watch(mainTab, (tab) => {
  const wantTab = tab === 'pam' ? 'pam' : undefined
  const curTab = typeof route.query.tab === 'string' ? route.query.tab : undefined
  if (wantTab === curTab) return
  const q: Record<string, string> = {}
  for (const [k, v] of Object.entries(route.query)) {
    if (typeof v === 'string') q[k] = v
  }
  if (tab === 'pam') q.tab = 'pam'
  else delete q.tab
  router.replace({ path: '/terminal', query: q })
})

async function saveKey() {
  await api.post('/terminal/keys', keyForm.value)
  ElMessage.success(t('common.success'))
  keysDialog.value = false
  keyForm.value = { name: '', public_key: '', private_key: '', remark: '' }
  loadAll()
}

async function deleteKey(id: number) {
  await api.delete(`/terminal/keys/${id}`)
  loadAll()
}

async function scrollChat() {
  await nextTick()
  if (chatBoxRef.value) chatBoxRef.value.scrollTop = chatBoxRef.value.scrollHeight
}

async function sendChat() {
  const msg = chatInput.value.trim()
  if (!msg || chatLoading.value) return
  const s = activeSession.value
  chatMessages.value.push({ role: 'user', content: msg })
  chatInput.value = ''
  chatLoading.value = true
  await scrollChat()
  try {
    const res: any = await api.post('/terminal/ai/chat', {
      message: msg,
      host: s?.host || '',
      user: s?.user || '',
      history: chatMessages.value.slice(-10).map((m) => ({ role: m.role, content: m.content })),
    }, { timeout: AI_REQUEST_TIMEOUT })
    chatMessages.value.push({ role: 'assistant', content: res.data?.reply || '', suggestedCommand: res.data?.suggested_command })
  } catch (e: any) {
    chatMessages.value.push({ role: 'assistant', content: resolveApiError(e, t('terminalPage.aiFailed'), t('common.requestTimeout')) })
  } finally {
    chatLoading.value = false
    scrollChat()
  }
}

watch(() => prefs.value.sidebarCollapsed, savePrefs)
watch(() => prefs.value.connBarCollapsed, savePrefs)

onMounted(async () => {
  window.addEventListener('keydown', onKeydown)
  tickTimer = setInterval(() => { nowTick.value = Date.now() }, 1000)
  if (route.query.tab === 'pam') mainTab.value = 'pam'
  await loadAll()
  if (await connectFromQuery(route.query)) return
  if (mainTab.value === 'shell') addSession(targets.value.find((x) => x.id === 'local'))
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown)
  if (tickTimer) clearInterval(tickTimer)
})
</script>

<template>
  <div ref="pageRef" class="terminal-page" :class="{ fullscreen }">
    <!-- 顶栏 -->
    <div class="page-header">
      <div>
        <div class="main-tabs">
          <el-radio-group v-model="mainTab" size="default">
            <el-radio-button value="shell">{{ t('terminalPage.tabShell') }}</el-radio-button>
            <el-radio-button value="pam">{{ t('terminalPage.tabPam') }}</el-radio-button>
          </el-radio-group>
        </div>
        <h2 v-if="mainTab === 'shell'">{{ t('terminalPage.title') }}</h2>
        <p v-if="mainTab === 'shell'" class="hint">{{ t('terminalPage.subtitleMulti', { n: connectedCount, total: sessions.length }) }}</p>
      </div>
      <div v-if="mainTab === 'shell'" class="header-actions">
        <el-button-group>
          <el-button size="small" @click="changeFontSize(-1)">A-</el-button>
          <el-button size="small" disabled>{{ prefs.fontSize }}</el-button>
          <el-button size="small" @click="changeFontSize(1)">A+</el-button>
        </el-button-group>
        <el-button :icon="fullscreen ? Fold : FullScreen" @click="toggleFullscreen">
          {{ fullscreen ? t('terminalPage.exitFullscreen') : t('terminalPage.fullscreen') }}
        </el-button>
        <el-button @click="keysDialog = true">{{ t('terminalPage.manageKeys') }}</el-button>
        <el-button type="primary" plain :icon="MagicStick" @click="aiOpen = !aiOpen">{{ t('terminalPage.aiAssistant') }}</el-button>
      </div>
    </div>

    <TerminalPamPanel
      v-if="mainTab === 'pam'"
      @connect="onPamConnect"
      @assets-changed="loadAll"
    />

    <div v-show="mainTab === 'shell'" class="workspace">
      <!-- 左侧连接面板 -->
      <aside v-show="!prefs.sidebarCollapsed" class="sidebar">
        <div v-if="bastionTargets.length" class="sidebar-section">
          <div class="sidebar-title">{{ t('terminalPage.bastionAssets') }}</div>
          <div
            v-for="tg in bastionTargets"
            :key="tg.id"
            class="sidebar-item bastion-item"
            @click="quickConnect(tg)"
            @dblclick="quickConnect(tg)"
          >
            <el-icon><Connection /></el-icon>
            <span class="item-label">{{ tg.label }}</span>
            <el-button text size="small" class="item-action" @click.stop="addSession(tg)">+</el-button>
          </div>
        </div>
        <div class="sidebar-section">
          <div class="sidebar-title">{{ t('terminalPage.quickConnect') }}</div>
          <div
            v-for="tg in quickTargets"
            :key="tg.id"
            class="sidebar-item"
            @click="quickConnect(tg)"
            @dblclick="quickConnect(tg)"
          >
            <el-icon><Connection /></el-icon>
            <span class="item-label">{{ tg.label }}</span>
            <el-button text size="small" class="item-action" @click.stop="addSession(tg)">+</el-button>
          </div>
        </div>
        <div v-if="savedConnections.length" class="sidebar-section">
          <div class="sidebar-title">{{ t('terminalPage.savedConnections') }}</div>
          <div
            v-for="sc in savedConnections"
            :key="sc.id"
            class="sidebar-item"
            @click="quickConnect(undefined, sc)"
          >
            <el-icon><StarFilled /></el-icon>
            <span class="item-label">{{ sc.name }}</span>
            <el-button text size="small" type="danger" class="item-action" @click.stop="removeSaved(sc.id)">
              <el-icon><Delete /></el-icon>
            </el-button>
          </div>
        </div>
        <p class="sidebar-tip">{{ t('terminalPage.sidebarTip') }}</p>
      </aside>

      <div class="main-panel">
        <!-- 标签栏 -->
        <div class="session-tabs">
          <el-button text class="sidebar-toggle" @click="prefs.sidebarCollapsed = !prefs.sidebarCollapsed">
            <el-icon><component :is="prefs.sidebarCollapsed ? Expand : Fold" /></el-icon>
          </el-button>
          <div
            v-for="s in sessions"
            :key="s.id"
            class="session-tab"
            :class="{ active: s.id === activeSessionId, connected: s.connected, unread: s.unread }"
            @click="activeSessionId = s.id"
          >
            <span class="tab-dot" :class="{ on: s.connected }" />
            <span class="tab-title">{{ s.title }}</span>
            <span v-if="s.unread" class="tab-badge" />
            <el-dropdown trigger="click" @command="(cmd: string) => handleTabMenu(cmd, s.id)" @click.stop>
              <el-icon class="tab-menu"><MoreFilled /></el-icon>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="reconnect">{{ t('terminalPage.reconnect') }}</el-dropdown-item>
                  <el-dropdown-item command="clear">{{ t('terminalPage.clearScreen') }}</el-dropdown-item>
                  <el-dropdown-item command="duplicate">{{ t('terminalPage.duplicate') }}</el-dropdown-item>
                  <el-dropdown-item command="rename">{{ t('terminalPage.rename') }}</el-dropdown-item>
                  <el-dropdown-item divided command="close">{{ t('terminalPage.closeTab') }}</el-dropdown-item>
                  <el-dropdown-item command="closeOthers" :disabled="sessions.length <= 1">{{ t('terminalPage.closeOthers') }}</el-dropdown-item>
                  <el-dropdown-item command="closeAll">{{ t('terminalPage.closeAll') }}</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
            <el-icon v-if="sessions.length > 1" class="tab-close" @click.stop="closeSession(s.id)"><Close /></el-icon>
          </div>
          <button type="button" class="tab-add" :title="t('terminalPage.newSession')" @click="addSession()">
            <el-icon><Plus /></el-icon>
          </button>
        </div>

        <!-- 连接栏（可折叠） -->
        <div v-if="activeSession" class="conn-wrap">
          <div class="conn-toggle" @click="prefs.connBarCollapsed = !prefs.connBarCollapsed">
            <el-icon><component :is="prefs.connBarCollapsed ? Expand : Fold" /></el-icon>
            <span>{{ t('terminalPage.connectionSettings') }}</span>
            <span class="conn-summary">{{ activeSession.user }}@{{ activeSession.host }}:{{ activeSession.port }}</span>
          </div>
          <el-collapse-transition>
            <div v-show="!prefs.connBarCollapsed" class="conn-bar">
              <el-form inline size="small">
                <el-form-item :label="t('terminalPage.target')">
                  <el-select v-model="activeSession.selectedTarget" style="width:180px" :disabled="activeSession.connected" @change="(v: string) => onTargetChange(activeSession!, v)">
                    <el-option v-for="tg in targets" :key="tg.id" :label="tg.label" :value="tg.id" />
                  </el-select>
                </el-form-item>
                <el-form-item :label="t('terminalPage.host')">
                  <el-input v-model="activeSession.host" style="width:120px" :disabled="activeSession.connected || (activeSession.selectedTarget !== 'custom' && activeSession.selectedTarget !== 'local' && !targets.find(x => x.id === activeSession!.selectedTarget)?.is_local)" />
                </el-form-item>
                <el-form-item :label="t('common.port')">
                  <el-input-number v-model="activeSession.port" :min="1" :max="65535" :disabled="activeSession.connected" controls-position="right" style="width:100px" />
                </el-form-item>
                <el-form-item :label="t('terminalPage.user')">
                  <el-input v-model="activeSession.user" style="width:80px" :disabled="activeSession.connected" />
                </el-form-item>
                <el-form-item :label="t('terminalPage.auth')">
                  <el-radio-group v-model="activeSession.authMethod" :disabled="activeSession.connected">
                    <el-radio-button value="password">{{ t('terminalPage.authPassword') }}</el-radio-button>
                    <el-radio-button value="key">{{ t('terminalPage.authKey') }}</el-radio-button>
                  </el-radio-group>
                </el-form-item>
                <el-form-item v-if="activeSession.authMethod === 'password'" :label="t('terminalPage.password')">
                  <el-input v-model="activeSession.password" type="password" show-password style="width:130px" :disabled="activeSession.connected" :placeholder="targets.find(x => x.id === activeSession!.selectedTarget)?.has_password ? t('terminalPage.passwordOptional') : ''" />
                </el-form-item>
                <el-form-item v-if="activeSession.authMethod === 'key'" :label="t('terminalPage.savedKey')">
                  <el-select v-model="activeSession.keyId" clearable style="width:140px" :disabled="activeSession.connected" :placeholder="t('terminalPage.pickKey')">
                    <el-option v-for="k in savedKeys" :key="k.id" :label="k.name" :value="k.id" />
                  </el-select>
                </el-form-item>
                <el-form-item>
                  <el-button v-if="!activeSession.connected" type="primary" :loading="activeSession.connecting" @click="connectActive">{{ t('terminalPage.connect') }}</el-button>
                  <el-button v-else type="danger" @click="disconnectActive">{{ t('terminalPage.disconnect') }}</el-button>
                  <el-button v-if="activeSession.connected" @click="paneRefs[activeSessionId]?.clearScreen()">{{ t('terminalPage.clearScreen') }}</el-button>
                  <el-button :icon="isBookmarked(activeSession) ? StarFilled : Star" @click="bookmarkActive">{{ t('terminalPage.saveConnection') }}</el-button>
                </el-form-item>
              </el-form>
            </div>
          </el-collapse-transition>
        </div>

        <!-- 终端区 -->
        <div class="main-row">
          <div class="term-area" :class="{ 'with-ai': aiOpen }">
            <div
              v-for="s in sessions"
              :key="s.id"
              class="term-layer"
              :class="{ active: s.id === activeSessionId }"
            >
              <TerminalPane
                :ref="(el) => setPaneRef(s.id, el)"
                :session="s"
                :targets="allTargets"
                :active="s.id === activeSessionId"
                :font-size="prefs.fontSize"
                @update="(p) => updateSession(s.id, p)"
                @activity="onSessionActivity(s.id)"
              />
            </div>
          </div>
          <aside v-show="aiOpen" class="ai-panel">
            <div class="ai-head"><el-icon><MagicStick /></el-icon> {{ t('terminalPage.aiAssistant') }}</div>
            <div ref="chatBoxRef" class="ai-messages">
              <p v-if="!chatMessages.length" class="ai-empty">{{ t('terminalPage.aiWelcome') }}</p>
              <div v-for="(m, i) in chatMessages" :key="i" class="ai-msg" :class="m.role">
                <div class="msg-body">{{ m.content }}</div>
                <el-button v-if="m.suggestedCommand" size="small" type="primary" plain @click="sendCommand(m.suggestedCommand!)">{{ t('terminalPage.runCommand') }}</el-button>
              </div>
              <div v-if="chatLoading" class="ai-msg assistant">{{ t('terminalPage.aiThinking') }}</div>
            </div>
            <div class="ai-input">
              <el-input v-model="chatInput" type="textarea" :rows="2" :placeholder="t('terminalPage.aiPlaceholder')" @keydown.ctrl.enter="sendChat" />
              <el-button type="primary" :icon="Promotion" :loading="chatLoading" @click="sendChat" />
            </div>
          </aside>
        </div>

        <!-- 状态栏 -->
        <div class="status-bar">
          <span>{{ statusText }}</span>
          <span class="status-right">{{ t('terminalPage.shortcuts') }}</span>
        </div>
      </div>
    </div>

    <el-dialog v-model="keysDialog" :title="t('terminalPage.manageKeys')" width="600px">
      <el-form label-width="80px">
        <el-form-item :label="t('common.name')"><el-input v-model="keyForm.name" /></el-form-item>
        <el-form-item :label="t('terminalPage.publicKey')"><el-input v-model="keyForm.public_key" type="textarea" :rows="2" /></el-form-item>
        <el-form-item :label="t('terminalPage.privateKey')"><el-input v-model="keyForm.private_key" type="textarea" :rows="4" :placeholder="t('terminalPage.privateKeyHint')" /></el-form-item>
        <el-form-item :label="t('common.description')"><el-input v-model="keyForm.remark" /></el-form-item>
      </el-form>
      <el-button type="primary" @click="saveKey">{{ t('terminalPage.addKey') }}</el-button>
      <el-table :data="sshKeys" size="small" style="margin-top:16px">
        <el-table-column prop="name" :label="t('common.name')" width="120" />
        <el-table-column :label="t('terminalPage.keyType')" width="100">
          <template #default="{ row }">{{ row.has_private ? t('terminalPage.keyPrivate') : t('terminalPage.keyPublicOnly') }}</template>
        </el-table-column>
        <el-table-column prop="remark" :label="t('common.description')" show-overflow-tooltip />
        <el-table-column :label="t('common.actions')" width="80">
          <template #default="{ row }"><el-button text type="danger" @click="deleteKey(row.id)">{{ t('common.delete') }}</el-button></template>
        </el-table-column>
      </el-table>
    </el-dialog>
  </div>
</template>

<style scoped>
.terminal-page { max-width: 100%; display: flex; flex-direction: column; height: calc(100vh - 80px); }
.terminal-page.fullscreen {
  position: fixed; inset: 0; z-index: 2000;
  height: 100vh; background: var(--el-bg-color);
  padding: 12px; box-sizing: border-box;
}
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 8px; flex-wrap: wrap; gap: 8px; flex-shrink: 0; }
.main-tabs { margin-bottom: 8px; }
.page-header h2 { margin: 0 0 4px; font-size: 18px; }
.hint { margin: 0; font-size: 12px; color: var(--el-text-color-secondary); }
.header-actions { display: flex; gap: 8px; flex-wrap: wrap; align-items: center; }

.workspace { display: flex; flex: 1; min-height: 0; gap: 10px; }
.sidebar {
  width: 220px; flex-shrink: 0;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px; padding: 10px;
  overflow-y: auto; background: var(--el-fill-color-blank);
}
.sidebar-section { margin-bottom: 12px; }
.sidebar-title { font-size: 11px; font-weight: 600; text-transform: uppercase; color: var(--el-text-color-secondary); margin-bottom: 6px; letter-spacing: .04em; }
.sidebar-item {
  display: flex; align-items: center; gap: 8px;
  padding: 7px 8px; border-radius: 6px; cursor: pointer; font-size: 13px;
}
.sidebar-item:hover { background: var(--el-fill-color-light); }
.item-label { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.item-action { flex-shrink: 0; padding: 2px !important; }
.sidebar-tip { font-size: 11px; color: var(--el-text-color-placeholder); margin: 8px 0 0; line-height: 1.4; }

.main-panel { flex: 1; min-width: 0; display: flex; flex-direction: column; min-height: 0; }

.session-tabs {
  display: flex; align-items: center; gap: 2px;
  margin-bottom: 6px; padding: 3px;
  background: var(--el-fill-color-lighter); border-radius: 8px;
  overflow-x: auto; flex-shrink: 0;
}
.sidebar-toggle { flex-shrink: 0; padding: 4px 8px !important; }
.session-tab {
  display: flex; align-items: center; gap: 5px;
  padding: 5px 8px; border-radius: 6px; cursor: pointer;
  font-size: 12px; white-space: nowrap; user-select: none;
  border: 1px solid transparent; max-width: 200px; position: relative;
}
.session-tab:hover { background: var(--el-fill-color); }
.session-tab.active { background: var(--el-bg-color); border-color: var(--el-border-color); }
.session-tab.unread .tab-title { font-weight: 600; }
.tab-dot { width: 7px; height: 7px; border-radius: 50%; background: var(--el-text-color-placeholder); flex-shrink: 0; }
.tab-dot.on { background: var(--el-color-success); }
.tab-badge { width: 6px; height: 6px; border-radius: 50%; background: var(--el-color-danger); flex-shrink: 0; }
.tab-title { overflow: hidden; text-overflow: ellipsis; }
.tab-menu, .tab-close { font-size: 13px; color: var(--el-text-color-secondary); flex-shrink: 0; }
.tab-add {
  display: flex; align-items: center; justify-content: center;
  width: 28px; height: 28px; border: none; border-radius: 6px;
  background: transparent; cursor: pointer; color: var(--el-text-color-secondary); flex-shrink: 0;
}
.tab-add:hover { background: var(--el-fill-color); color: var(--el-color-primary); }

.conn-wrap { flex-shrink: 0; margin-bottom: 6px; border: 1px solid var(--el-border-color-lighter); border-radius: 8px; overflow: hidden; }
.conn-toggle {
  display: flex; align-items: center; gap: 8px; padding: 6px 10px;
  cursor: pointer; font-size: 12px; background: var(--el-fill-color-lighter);
  user-select: none;
}
.conn-toggle:hover { background: var(--el-fill-color); }
.conn-summary { margin-left: auto; color: var(--el-text-color-secondary); font-family: monospace; font-size: 11px; }
.conn-bar { padding: 8px 10px 2px; }
.conn-bar :deep(.el-form-item) { margin-bottom: 6px; margin-right: 12px; }

.main-row { display: flex; gap: 10px; flex: 1; min-height: 0; }
.term-area { flex: 1; min-width: 0; position: relative; min-height: 200px; }
.term-layer {
  position: absolute; inset: 0;
  visibility: hidden; pointer-events: none; z-index: 0;
}
.term-layer.active { visibility: visible; pointer-events: auto; z-index: 1; }

.ai-panel {
  width: 320px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-bg-color);
  min-height: 0;
  overflow: hidden;
}
.ai-head { flex-shrink: 0; padding: 8px 12px; font-weight: 600; font-size: 13px; border-bottom: 1px solid var(--el-border-color-lighter); display: flex; align-items: center; gap: 6px; }
.ai-messages { flex: 1; min-height: 0; overflow-y: auto; padding: 10px; font-size: 13px; }
.ai-empty { color: var(--el-text-color-secondary); margin: 0; }
.ai-msg { margin-bottom: 10px; }
.ai-msg.user .msg-body { background: var(--el-fill-color-light); padding: 8px; border-radius: 6px; }
.ai-msg.assistant .msg-body { line-height: 1.5; white-space: pre-wrap; margin-bottom: 6px; }
.ai-input { flex-shrink: 0; padding: 8px; border-top: 1px solid var(--el-border-color-lighter); display: flex; gap: 8px; align-items: flex-end; background: var(--el-bg-color); }
.ai-input .el-textarea { flex: 1; }

.status-bar {
  display: flex; justify-content: space-between; align-items: center;
  padding: 4px 10px; margin-top: 6px; flex-shrink: 0;
  font-size: 11px; font-family: monospace;
  background: var(--el-fill-color-lighter); border-radius: 6px;
  color: var(--el-text-color-secondary);
}
.status-right { opacity: .7; }
</style>
