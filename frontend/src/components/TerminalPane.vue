<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import { panelBase } from '@/utils/panelBase'
import { useAuthStore } from '@/stores/auth'
import type { TerminalSession, TerminalTarget } from '@/types/terminal'

const props = defineProps<{
  session: TerminalSession
  targets: TerminalTarget[]
  active: boolean
  fontSize: number
}>()

const emit = defineEmits<{
  update: [patch: Partial<TerminalSession>]
  activity: []
}>()

const { t } = useI18n()
const auth = useAuthStore()

const termEl = ref<HTMLElement>()
let term: Terminal | null = null
let fitAddon: FitAddon | null = null
let ws: WebSocket | null = null
let resizeObs: ResizeObserver | null = null

const currentTarget = () => props.targets.find((x) => x.id === props.session.selectedTarget)

function patch(p: Partial<TerminalSession>) {
  emit('update', p)
}

function wsURL(): string {
  const base = panelBase()
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const path = base.replace(/^\//, '').replace(/\/$/, '')
  const prefix = path ? `/${path}` : ''
  const token = encodeURIComponent(auth.token || localStorage.getItem('token') || '')
  return `${proto}//${location.host}${prefix}/api/v1/terminal/ws?token=${token}`
}

function attachKeyHandlers() {
  if (!term) return
  term.attachCustomKeyEventHandler((ev) => {
    if (ev.ctrlKey && ev.shiftKey && ev.key === 'C' && term?.hasSelection()) {
      const sel = term.getSelection()
      if (sel) navigator.clipboard.writeText(sel).catch(() => {})
      return false
    }
    if (ev.ctrlKey && ev.shiftKey && ev.key === 'V') {
      navigator.clipboard.readText().then((text) => {
        if (text && ws?.readyState === WebSocket.OPEN) ws.send(text)
      }).catch(() => {})
      return false
    }
    return true
  })
}

function initTerm() {
  if (!termEl.value || term) return
  term = new Terminal({
    cursorBlink: true,
    fontSize: props.fontSize,
    fontFamily: "'Cascadia Code', 'JetBrains Mono', Consolas, monospace",
    theme: { background: '#0d1117', foreground: '#c9d1d9', cursor: '#58a6ff', selectionBackground: '#264f78' },
    allowProposedApi: true,
    scrollback: 5000,
  })
  fitAddon = new FitAddon()
  term.loadAddon(fitAddon)
  term.open(termEl.value)
  fitAddon.fit()
  attachKeyHandlers()
  term.writeln('\x1b[1;36m' + t('terminalPage.welcome') + '\x1b[0m')
  term.writeln(`\x1b[90m${props.session.user}@${props.session.host}:${props.session.port}\x1b[0m`)

  resizeObs = new ResizeObserver(() => {
    if (props.active) {
      fitAddon?.fit()
      sendResize()
    }
  })
  resizeObs.observe(termEl.value)
  term.onData((data) => {
    if (ws?.readyState === WebSocket.OPEN) ws.send(data)
  })
}

function applyFontSize(size: number) {
  if (!term) return
  term.options.fontSize = size
  fitAddon?.fit()
  sendResize()
}

function writeOutput(data: string | Uint8Array) {
  if (!props.active) emit('activity')
  if (typeof data === 'string') term?.write(data)
  else term?.write(data)
}

function sendResize() {
  if (!term || !ws || ws.readyState !== WebSocket.OPEN) return
  ws.send(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows }))
}

function sendCommand(cmd: string) {
  if (!ws || ws.readyState !== WebSocket.OPEN) return false
  ws.send(cmd.endsWith('\n') ? cmd : cmd + '\n')
  return true
}

function clearScreen() {
  term?.clear()
}

function disconnect() {
  ws?.close()
  ws = null
  patch({ connected: false, connecting: false, connectedAt: undefined })
  term?.writeln('\r\n\x1b[33m' + t('terminalPage.disconnected') + '\x1b[0m')
}

async function connect() {
  if (props.session.connecting || props.session.connected) return
  const s = props.session
  const tg = currentTarget()
  const viaAsset = !!(s.assetId || tg?.asset_id)
  if (!viaAsset && !s.host.trim()) throw new Error(t('terminalPage.needHost'))
  if (!viaAsset && s.authMethod === 'password') {
    const hasSaved = !!(tg?.has_password && tg?.node_id)
    if (!s.password.trim() && !hasSaved) throw new Error(t('terminalPage.needPassword'))
  } else if (!viaAsset && s.authMethod === 'key') {
    if (!s.keyId && !s.privateKeyPaste.trim()) throw new Error(t('terminalPage.needKey'))
  }

  initTerm()
  patch({ connecting: true, unread: false })
  term?.clear()
  term?.writeln('\x1b[90m' + t('terminalPage.connecting') + '\x1b[0m')

  ws = new WebSocket(wsURL())
  ws.binaryType = 'arraybuffer'

  ws.onopen = () => {
    const tg = currentTarget()
    const payload: Record<string, unknown> = {
      type: 'connect',
      host: s.host,
      port: s.port,
      user: s.user,
      auth_method: s.authMethod,
      cols: term?.cols || 120,
      rows: term?.rows || 32,
    }
    if (s.authMethod === 'password' && s.password.trim()) payload.password = s.password
    if (s.authMethod === 'key') {
      if (s.keyId) payload.key_id = s.keyId
      if (s.privateKeyPaste.trim()) payload.private_key = s.privateKeyPaste
      if (s.keyPassphrase.trim()) payload.key_passphrase = s.keyPassphrase
    }
    if (tg?.node_id) payload.node_id = tg.node_id
    const assetId = s.assetId || tg?.asset_id
    if (assetId) payload.asset_id = assetId
    const accountId = s.accountId || tg?.account_id
    if (accountId) payload.account_id = accountId
    ws!.send(JSON.stringify(payload))
  }

  ws.onmessage = (ev) => {
    if (typeof ev.data === 'string') {
      try {
        const msg = JSON.parse(ev.data)
        if (msg.type === 'connected') {
          const title = `${s.user}@${s.host}`
          patch({ connected: true, connecting: false, title, connectedAt: Date.now(), unread: false })
          fitAddon?.fit()
          sendResize()
          return
        }
        if (msg.type === 'error') {
          patch({ connecting: false })
          term?.writeln('\r\n\x1b[31m' + (msg.message || 'Error') + '\x1b[0m')
          disconnect()
          return
        }
        if (msg.type === 'policy') {
          term?.writeln('\r\n\x1b[33m' + (msg.message || '') + '\x1b[0m')
          return
        }
      } catch { /* pass */ }
      writeOutput(ev.data)
      return
    }
    if (ev.data instanceof ArrayBuffer) {
      writeOutput(new Uint8Array(ev.data))
    }
  }

  ws.onerror = () => {
    patch({ connecting: false })
    term?.writeln('\r\n\x1b[31m' + t('terminalPage.connectFail') + '\x1b[0m')
  }
  ws.onclose = () => {
    patch({ connected: false, connecting: false })
  }
}

async function reconnect() {
  disconnect()
  await nextTick()
  await connect()
}

watch(() => props.fontSize, (v) => applyFontSize(v))

watch(
  () => props.active,
  (v) => {
    if (v) {
      patch({ unread: false })
      nextTick(() => {
        if (!term) initTerm()
        fitAddon?.fit()
        sendResize()
        term?.focus()
      })
    }
  }
)

onMounted(() => {
  if (props.active) initTerm()
})

onBeforeUnmount(() => {
  disconnect()
  resizeObs?.disconnect()
  term?.dispose()
  term = null
})

defineExpose({ connect, disconnect, reconnect, sendCommand, clearScreen, focus: () => term?.focus() })
</script>

<template>
  <div ref="termEl" class="term-pane" />
</template>

<style scoped>
.term-pane {
  width: 100%;
  height: 100%;
  background: #0d1117;
  border-radius: 6px;
  padding: 6px 8px;
  overflow: hidden;
}
.term-pane :deep(.xterm) { height: 100%; }
.term-pane :deep(.xterm-viewport) { overflow-y: auto !important; }
</style>
