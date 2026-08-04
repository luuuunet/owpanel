<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import { panelBase } from '@/utils/panelBase'
import { useAuthStore } from '@/stores/auth'

const props = defineProps<{
  containerId: string
  active: boolean
  shell?: string
}>()

const { t } = useI18n()
const auth = useAuthStore()
const termEl = ref<HTMLElement>()
let term: Terminal | null = null
let fitAddon: FitAddon | null = null
let ws: WebSocket | null = null
let resizeObs: ResizeObserver | null = null
let started = false

function wsURL(): string {
  const base = panelBase()
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const path = base.replace(/^\//, '').replace(/\/$/, '')
  const prefix = path ? `/${path}` : ''
  const token = encodeURIComponent(auth.token || localStorage.getItem('token') || '')
  const id = encodeURIComponent(props.containerId)
  return `${proto}//${location.host}${prefix}/api/v1/docker/containers/${id}/exec/ws?token=${token}`
}

function initTerm() {
  if (!termEl.value || term) return
  term = new Terminal({
    cursorBlink: true,
    fontSize: 13,
    fontFamily: "'Cascadia Code', 'JetBrains Mono', Consolas, monospace",
    theme: { background: '#0d1117', foreground: '#c9d1d9', cursor: '#58a6ff', selectionBackground: '#264f78' },
    scrollback: 4000,
  })
  fitAddon = new FitAddon()
  term.loadAddon(fitAddon)
  term.open(termEl.value)
  fitAddon.fit()
  term.onData((data) => {
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: 'stdin', data }))
    }
  })
  resizeObs = new ResizeObserver(() => {
    if (props.active) fitAddon?.fit()
  })
  resizeObs.observe(termEl.value)
}

function connect() {
  if (!props.containerId || !term) return
  disconnect()
  started = false
  term?.clear()
  term?.writeln(`\x1b[1;36m${t('docker.consoleConnecting')}\x1b[0m`)
  ws = new WebSocket(wsURL())
  ws.binaryType = 'arraybuffer'
  ws.onopen = () => {
    ws?.send(JSON.stringify({ type: 'start', shell: props.shell || '/bin/sh' }))
  }
  ws.onmessage = (ev) => {
    if (typeof ev.data === 'string') {
      try {
        const msg = JSON.parse(ev.data)
        if (msg.type === 'ready') {
          started = true
          term?.writeln(`\x1b[90m${t('docker.consoleReady')}\x1b[0m`)
          return
        }
        if (msg.type === 'error') {
          term?.writeln(`\x1b[31m${msg.data || t('common.failed')}\x1b[0m`)
          return
        }
        if (msg.type === 'exit') {
          term?.writeln(`\x1b[90m[${msg.data || 'exit'}]\x1b[0m`)
          return
        }
      } catch {
        term?.write(ev.data)
      }
      return
    }
    const bytes = new Uint8Array(ev.data as ArrayBuffer)
    term?.write(bytes)
  }
  ws.onclose = () => {
    if (started) term?.writeln(`\x1b[90m${t('docker.consoleClosed')}\x1b[0m`)
  }
  ws.onerror = () => {
    term?.writeln(`\x1b[31m${t('docker.consoleError')}\x1b[0m`)
  }
}

function disconnect() {
  if (ws) {
    try { ws.close() } catch { /* ignore */ }
    ws = null
  }
  started = false
}

function reconnect() {
  connect()
}

watch(() => props.active, async (v) => {
  if (v) {
    await nextTick()
    fitAddon?.fit()
    if (!ws || ws.readyState === WebSocket.CLOSED) connect()
  }
})

watch(() => props.containerId, () => {
  if (props.active) connect()
})

onMounted(async () => {
  initTerm()
  await nextTick()
  if (props.active) connect()
})

onBeforeUnmount(() => {
  disconnect()
  resizeObs?.disconnect()
  term?.dispose()
  term = null
})

defineExpose({ reconnect })
</script>

<template>
  <div class="docker-exec">
    <div class="docker-exec-bar">
      <el-button size="small" @click="reconnect">{{ t('docker.consoleReconnect') }}</el-button>
      <span class="hint">{{ t('docker.consoleHint') }}</span>
    </div>
    <div ref="termEl" class="docker-exec-term" />
  </div>
</template>

<style scoped>
.docker-exec { display: flex; flex-direction: column; height: 100%; min-height: 360px; }
.docker-exec-bar { display: flex; align-items: center; gap: 10px; margin-bottom: 8px; }
.docker-exec-bar .hint { font-size: 12px; color: var(--el-text-color-secondary); }
.docker-exec-term { flex: 1; min-height: 320px; border-radius: 8px; overflow: hidden; background: #0d1117; padding: 8px; }
</style>
