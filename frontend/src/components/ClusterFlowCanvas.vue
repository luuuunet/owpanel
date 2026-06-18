<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { ZoomIn, ZoomOut, Refresh, FullScreen } from '@element-plus/icons-vue'

export interface FlowNode {
  id: string
  type: string
  label: string
  x: number
  y: number
  ref_id?: number
  config?: Record<string, unknown>
  status?: string
}

export interface FlowEdge {
  id: string
  from: string
  to: string
  kind: string
}

export interface FlowGraph {
  nodes: FlowNode[]
  edges: FlowEdge[]
}

const props = defineProps<{
  modelValue: FlowGraph
  clusterNodes: any[]
  runLog?: string
  workflowStatus?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [graph: FlowGraph]
  save: []
  run: []
  syncNodes: []
}>()

const { t } = useI18n()

const canvasRef = ref<HTMLElement>()
const selectedId = ref('')
const selectedEdgeId = ref('')
const connectFrom = ref('')
const connectDrag = ref<{ fromId: string; mx: number; my: number } | null>(null)
const dragging = ref<{ id: string; ox: number; oy: number; nx: number; ny: number } | null>(null)
const panning = ref<{ ox: number; oy: number; px: number; py: number } | null>(null)
const paletteDrag = ref('')
const panX = ref(0)
const panY = ref(0)
const zoom = ref(1)

const NODE_W = 160
const NODE_H = 72

const palette = computed(() => [
  { type: 'master', icon: '👑', color: '#f6821f' },
  { type: 'worker', icon: '🖥️', color: '#0051c3' },
  { type: 'lb', icon: '⚖️', color: '#22c55e' },
  { type: 'db_master', icon: '🗄️', color: '#8b5cf6' },
  { type: 'db_slave', icon: '📀', color: '#a855f7' },
  { type: 'web_sync', icon: '🌐', color: '#06b6d4' },
])

const algorithms = computed(() => [
  { value: 'round_robin', label: t('clusterPage.algoRoundRobin') },
  { value: 'least_conn', label: t('clusterPage.algoLeastConn') },
  { value: 'ip_hash', label: t('clusterPage.algoIpHash') },
  { value: 'random', label: t('clusterPage.algoRandom') },
])

const selectedNode = computed(() => props.modelValue.nodes.find((n) => n.id === selectedId.value))

const contentTransform = computed(() => `translate(${panX.value}px, ${panY.value}px) scale(${zoom.value})`)

function typeLabel(type: string) {
  const map: Record<string, string> = {
    master: t('clusterPage.flowTypeMaster'),
    worker: t('clusterPage.flowTypeWorker'),
    lb: t('clusterPage.flowTypeLb'),
    db_master: t('clusterPage.flowTypeDbMaster'),
    db_slave: t('clusterPage.flowTypeDbSlave'),
    web_sync: t('clusterPage.flowTypeWebSync'),
  }
  return map[type] || type
}

function nodeStyle(type: string) {
  const p = palette.value.find((x) => x.type === type)
  return { borderColor: p?.color || '#94a3b8', '--node-accent': p?.color || '#94a3b8' }
}

function hasOutputPort(type: string) {
  return type === 'lb' || type === 'db_master' || type === 'master'
}

function hasInputPort(type: string) {
  return type === 'worker' || type === 'db_slave' || type === 'master'
}

function inferEdgeKind(fromType: string, toType: string): string | null {
  if (fromType === 'lb' && (toType === 'worker' || toType === 'master')) return 'routes'
  if (fromType === 'db_master' && toType === 'db_slave') return 'replicates'
  if (fromType === 'master' && toType === 'worker') return 'manages'
  return null
}

function emitGraph(graph: FlowGraph) {
  emit('update:modelValue', graph)
}

function newId(prefix: string) {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2, 6)}`
}

function screenToCanvas(clientX: number, clientY: number) {
  const rect = canvasRef.value!.getBoundingClientRect()
  return {
    x: (clientX - rect.left - panX.value) / zoom.value,
    y: (clientY - rect.top - panY.value) / zoom.value,
  }
}

function portPos(node: FlowNode, side: 'in' | 'out') {
  const y = node.y + NODE_H / 2
  return side === 'out' ? { x: node.x + NODE_W, y } : { x: node.x, y }
}

function onPaletteDragStart(type: string, e: DragEvent) {
  paletteDrag.value = type
  e.dataTransfer?.setData('text/plain', type)
}

function onCanvasDrop(e: DragEvent) {
  e.preventDefault()
  const type = paletteDrag.value || e.dataTransfer?.getData('text/plain')
  if (!type || !canvasRef.value) return
  const pt = screenToCanvas(e.clientX, e.clientY)
  const x = pt.x - NODE_W / 2
  const y = pt.y - NODE_H / 2
  const node: FlowNode = { id: newId(type), type, label: typeLabel(type), x: Math.max(20, x), y: Math.max(20, y) }
  if (type === 'lb') node.config = { domain: 'app.example.com', listen_port: 80, algorithm: 'round_robin' }
  if (type === 'db_master') node.config = { db_type: 'mysql', repl_user: 'repl', db_name: 'app_db' }
  if (type === 'worker' && props.clusterNodes.length) {
    const w = props.clusterNodes.find((n) => !n.is_local) || props.clusterNodes[0]
    node.ref_id = w.id
    node.label = w.name
    node.status = w.status
  }
  if (type === 'master') {
    const m = props.clusterNodes.find((n) => n.is_local) || props.clusterNodes[0]
    if (m) { node.ref_id = m.id; node.label = m.name; node.status = m.status }
  }
  if (type === 'db_slave' && props.clusterNodes.length) {
    const s = props.clusterNodes.find((n) => !n.is_local && n.provision_role === 'db_slave')
      || props.clusterNodes.find((n) => !n.is_local)
    if (s) { node.ref_id = s.id; node.label = s.name; node.status = s.status }
  }
  emitGraph({ ...props.modelValue, nodes: [...props.modelValue.nodes, node] })
  paletteDrag.value = ''
  selectedId.value = node.id
  selectedEdgeId.value = ''
}

function onCanvasDragOver(e: DragEvent) { e.preventDefault() }

function startPan(e: MouseEvent) {
  if ((e.target as HTMLElement).closest('.flow-node') || (e.target as HTMLElement).closest('.canvas-toolbar')) return
  panning.value = { ox: e.clientX, oy: e.clientY, px: panX.value, py: panY.value }
  selectedId.value = ''
  selectedEdgeId.value = ''
  connectFrom.value = ''
  connectDrag.value = null
}

function startDrag(node: FlowNode, e: MouseEvent) {
  if ((e.target as HTMLElement).closest('.port')) return
  dragging.value = { id: node.id, ox: e.clientX, oy: e.clientY, nx: node.x, ny: node.y }
  selectedId.value = node.id
  selectedEdgeId.value = ''
}

function onMouseMove(e: MouseEvent) {
  if (panning.value) {
    panX.value = panning.value.px + (e.clientX - panning.value.ox)
    panY.value = panning.value.py + (e.clientY - panning.value.oy)
    return
  }
  if (connectDrag.value) {
    const pt = screenToCanvas(e.clientX, e.clientY)
    connectDrag.value = { ...connectDrag.value, mx: pt.x, my: pt.y }
    return
  }
  if (!dragging.value) return
  const dx = (e.clientX - dragging.value.ox) / zoom.value
  const dy = (e.clientY - dragging.value.oy) / zoom.value
  const nodes = props.modelValue.nodes.map((n) =>
    n.id === dragging.value!.id ? { ...n, x: dragging.value!.nx + dx, y: dragging.value!.ny + dy } : n,
  )
  emitGraph({ ...props.modelValue, nodes })
}

function onMouseUp() {
  dragging.value = null
  panning.value = null
  connectDrag.value = null
}

function tryConnect(fromId: string, toId: string) {
  const from = props.modelValue.nodes.find((n) => n.id === fromId)
  const to = props.modelValue.nodes.find((n) => n.id === toId)
  if (!from || !to) return
  const kind = inferEdgeKind(from.type, to.type)
  if (!kind) {
    ElMessage.warning(t('clusterPage.flowInvalidConnect'))
    return
  }
  const dup = props.modelValue.edges.some((e) => e.from === fromId && e.to === toId && e.kind === kind)
  if (dup) return
  emitGraph({
    ...props.modelValue,
    edges: [...props.modelValue.edges, { id: newId('e'), from: fromId, to: toId, kind }],
  })
}

function onOutputPortDown(nodeId: string, e: MouseEvent) {
  e.stopPropagation()
  const node = props.modelValue.nodes.find((n) => n.id === nodeId)
  if (!node) return
  const p = portPos(node, 'out')
  connectFrom.value = nodeId
  connectDrag.value = { fromId: nodeId, mx: p.x, my: p.y }
  selectedId.value = nodeId
  selectedEdgeId.value = ''
}

function onOutputPortClick(nodeId: string, e: MouseEvent) {
  e.stopPropagation()
  if (connectDrag.value) return
  connectFrom.value = connectFrom.value === nodeId ? '' : nodeId
  selectedId.value = nodeId
  selectedEdgeId.value = ''
}

function onInputPortClick(nodeId: string, e: MouseEvent) {
  e.stopPropagation()
  if (!connectFrom.value) {
    selectedId.value = nodeId
    return
  }
  tryConnect(connectFrom.value, nodeId)
  connectFrom.value = ''
  connectDrag.value = null
}

function onInputPortUp(nodeId: string) {
  if (connectDrag.value && connectFrom.value) {
    tryConnect(connectFrom.value, nodeId)
    connectFrom.value = ''
    connectDrag.value = null
  }
}

function onWheel(e: WheelEvent) {
  e.preventDefault()
  const delta = e.deltaY > 0 ? -0.08 : 0.08
  zoom.value = Math.min(1.5, Math.max(0.5, +(zoom.value + delta).toFixed(2)))
}

function zoomIn() { zoom.value = Math.min(1.5, +(zoom.value + 0.1).toFixed(2)) }
function zoomOut() { zoom.value = Math.max(0.5, +(zoom.value - 0.1).toFixed(2)) }
function zoomReset() { zoom.value = 1; panX.value = 0; panY.value = 0 }

function fitView() {
  const nodes = props.modelValue.nodes
  if (!nodes.length || !canvasRef.value) { zoomReset(); return }
  let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity
  for (const n of nodes) {
    minX = Math.min(minX, n.x)
    minY = Math.min(minY, n.y)
    maxX = Math.max(maxX, n.x + NODE_W)
    maxY = Math.max(maxY, n.y + NODE_H)
  }
  const pad = 48
  const cw = canvasRef.value.clientWidth
  const ch = canvasRef.value.clientHeight
  const gw = maxX - minX + pad * 2
  const gh = maxY - minY + pad * 2
  const scale = Math.min(1.5, Math.max(0.5, Math.min(cw / gw, ch / gh)))
  zoom.value = scale
  panX.value = (cw - gw * scale) / 2 + pad * scale - minX * scale
  panY.value = (ch - gh * scale) / 2 + pad * scale - minY * scale
}

function removeSelected() {
  if (selectedEdgeId.value) {
    emitGraph({ ...props.modelValue, edges: props.modelValue.edges.filter((e) => e.id !== selectedEdgeId.value) })
    selectedEdgeId.value = ''
    return
  }
  if (!selectedId.value) return
  const id = selectedId.value
  emitGraph({
    nodes: props.modelValue.nodes.filter((n) => n.id !== id),
    edges: props.modelValue.edges.filter((e) => e.from !== id && e.to !== id),
  })
  selectedId.value = ''
}

function selectEdge(edgeId: string, e: MouseEvent) {
  e.stopPropagation()
  selectedEdgeId.value = edgeId
  selectedId.value = ''
  connectFrom.value = ''
}

function onKeyDown(e: KeyboardEvent) {
  if (e.key === 'Delete' || e.key === 'Backspace') {
    if ((e.target as HTMLElement).closest('input, textarea, .el-input, .el-textarea')) return
    if (selectedEdgeId.value || selectedId.value) {
      e.preventDefault()
      removeSelected()
    }
  }
}

function edgePath(from: FlowNode, to: FlowNode) {
  const p1 = portPos(from, 'out')
  const p2 = portPos(to, 'in')
  const mx = (p1.x + p2.x) / 2
  return `M ${p1.x} ${p1.y} C ${mx} ${p1.y}, ${mx} ${p2.y}, ${p2.x} ${p2.y}`
}

function tempEdgePath() {
  if (!connectDrag.value) return ''
  const from = props.modelValue.nodes.find((n) => n.id === connectDrag.value!.fromId)
  if (!from) return ''
  const p1 = portPos(from, 'out')
  const { mx, my } = connectDrag.value
  const mid = (p1.x + mx) / 2
  return `M ${p1.x} ${p1.y} C ${mid} ${p1.y}, ${mid} ${my}, ${mx} ${my}`
}

function edgeColor(kind: string, selected: boolean) {
  if (selected) return '#f59e0b'
  if (kind === 'replicates') return '#8b5cf6'
  if (kind === 'manages') return '#f6821f'
  return '#22c55e'
}

function updateSelected(patch: Partial<FlowNode>) {
  if (!selectedId.value) return
  emitGraph({ ...props.modelValue, nodes: props.modelValue.nodes.map((n) => (n.id === selectedId.value ? { ...n, ...patch } : n)) })
}

function updateConfig(key: string, val: unknown) {
  if (!selectedNode.value) return
  updateSelected({ config: { ...(selectedNode.value.config || {}), [key]: val } })
}

onMounted(() => window.addEventListener('keydown', onKeyDown))
onUnmounted(() => window.removeEventListener('keydown', onKeyDown))
</script>

<template>
  <div class="flow-editor" @mousemove="onMouseMove" @mouseup="onMouseUp" @mouseleave="onMouseUp">
    <aside class="palette">
      <div class="palette-title">{{ t('clusterPage.flowPalette') }}</div>
      <div v-for="p in palette" :key="p.type" class="palette-item" draggable="true" :style="{ borderLeftColor: p.color }" @dragstart="onPaletteDragStart(p.type, $event)">
        <span>{{ p.icon }}</span><span>{{ typeLabel(p.type) }}</span>
      </div>
      <p class="palette-hint">{{ t('clusterPage.flowDragHint') }}</p>
      <div class="palette-actions">
        <el-button size="small" @click="emit('syncNodes')">{{ t('clusterPage.flowSyncNodes') }}</el-button>
        <el-button size="small" type="primary" @click="emit('save')">{{ t('common.save') }}</el-button>
        <el-button size="small" type="success" @click="emit('run')">{{ t('clusterPage.flowRun') }}</el-button>
      </div>
      <div v-if="connectFrom && !connectDrag" class="connect-hint">{{ t('clusterPage.flowConnectTo') }}</div>
    </aside>
    <div class="canvas-wrap">
      <div ref="canvasRef" class="canvas" @drop="onCanvasDrop" @dragover="onCanvasDragOver" @mousedown="startPan" @wheel.prevent="onWheel">
        <div class="canvas-toolbar" @mousedown.stop>
          <el-button-group size="small">
            <el-button :icon="ZoomIn" @click="zoomIn" />
            <el-button :icon="ZoomOut" @click="zoomOut" />
            <el-button :icon="Refresh" @click="zoomReset" />
            <el-button :icon="FullScreen" @click="fitView" />
          </el-button-group>
          <span class="zoom-label">{{ Math.round(zoom * 100) }}%</span>
        </div>
        <div class="canvas-content" :style="{ transform: contentTransform }">
          <svg class="edges-layer">
            <defs><marker id="flow-arrow" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0,0 L0,6 L6,3 z" fill="#64748b" /></marker></defs>
            <path v-for="e in modelValue.edges" :key="e.id"
              v-show="modelValue.nodes.find(n => n.id === e.from) && modelValue.nodes.find(n => n.id === e.to)"
              :d="edgePath(modelValue.nodes.find(n => n.id === e.from)!, modelValue.nodes.find(n => n.id === e.to)!)"
              fill="none"
              :stroke="edgeColor(e.kind, selectedEdgeId === e.id)"
              :stroke-width="selectedEdgeId === e.id ? 3 : 2"
              marker-end="url(#flow-arrow)"
              class="edge-path"
              @mousedown.stop="selectEdge(e.id, $event)" />
            <path v-if="connectDrag" :d="tempEdgePath()" fill="none" stroke="#94a3b8" stroke-width="2" stroke-dasharray="6 4" pointer-events="none" />
          </svg>
          <div v-for="node in modelValue.nodes" :key="node.id" class="flow-node"
            :class="{ selected: selectedId === node.id, connecting: connectFrom === node.id }"
            :style="{ left: node.x + 'px', top: node.y + 'px', ...nodeStyle(node.type) }"
            @mousedown.stop="startDrag(node, $event)" @click.stop="selectedId = node.id; selectedEdgeId = ''">
            <button v-if="hasInputPort(node.type)" type="button" class="port port-in"
              :class="{ active: connectFrom && connectFrom !== node.id }"
              @mouseup.stop="onInputPortUp(node.id)" @click.stop="onInputPortClick(node.id, $event)">●</button>
            <div class="node-head"><span class="node-type">{{ typeLabel(node.type) }}</span>
              <el-tag v-if="node.status" size="small">{{ node.status }}</el-tag></div>
            <div class="node-label">{{ node.label }}</div>
            <button v-if="hasOutputPort(node.type)" type="button" class="port port-out"
              @mousedown.stop="onOutputPortDown(node.id, $event)" @click.stop="onOutputPortClick(node.id, $event)">●</button>
          </div>
        </div>
      </div>
      <div v-if="selectedNode" class="node-inspector">
        <div class="inspector-head"><strong>{{ t('clusterPage.flowInspector') }}</strong>
          <el-button text type="danger" size="small" @click="removeSelected">{{ t('common.delete') }}</el-button></div>
        <el-form size="small" label-width="90px">
          <el-form-item :label="t('common.name')"><el-input :model-value="selectedNode.label" @update:model-value="(v: string) => updateSelected({ label: v })" /></el-form-item>
          <el-form-item v-if="selectedNode.type === 'worker' || selectedNode.type === 'master' || selectedNode.type === 'db_slave'" :label="t('clusterPage.flowLinkNode')">
            <el-select :model-value="selectedNode.ref_id || 0" style="width:100%" @update:model-value="(v: number) => { const n = clusterNodes.find(x => x.id === v); updateSelected({ ref_id: v, label: n?.name || selectedNode!.label, status: n?.status }) }">
              <el-option v-for="n in clusterNodes" :key="n.id" :value="n.id" :label="n.name" /></el-select></el-form-item>
          <template v-if="selectedNode.type === 'lb'">
            <el-form-item label="Domain"><el-input :model-value="String(selectedNode.config?.domain || '')" @update:model-value="(v: string) => updateConfig('domain', v)" /></el-form-item>
            <el-form-item :label="t('common.port')"><el-input-number :model-value="Number(selectedNode.config?.listen_port || 80)" :min="1" :max="65535" @update:model-value="(v: number) => updateConfig('listen_port', v)" /></el-form-item>
            <el-form-item :label="t('clusterPage.algorithm')">
              <el-select :model-value="String(selectedNode.config?.algorithm || 'round_robin')" style="width:100%" @update:model-value="(v: string) => updateConfig('algorithm', v)">
                <el-option v-for="a in algorithms" :key="a.value" :label="a.label" :value="a.value" />
              </el-select>
            </el-form-item>
          </template>
          <template v-if="selectedNode.type === 'db_master'">
            <el-form-item :label="t('clusterPage.flowDbName')"><el-input :model-value="String(selectedNode.config?.db_name || 'app_db')" @update:model-value="(v: string) => updateConfig('db_name', v)" /></el-form-item>
            <el-form-item :label="t('clusterPage.flowReplUser')"><el-input :model-value="String(selectedNode.config?.repl_user || 'repl')" @update:model-value="(v: string) => updateConfig('repl_user', v)" /></el-form-item>
          </template>
        </el-form>
      </div>
      <div v-if="runLog" class="run-log"><div class="log-head">{{ t('clusterPage.flowRunLog') }} <el-tag size="small">{{ workflowStatus }}</el-tag></div><pre>{{ runLog }}</pre></div>
    </div>
  </div>
</template>

<style scoped>
.flow-editor { display: grid; grid-template-columns: 200px 1fr; gap: 12px; min-height: 560px; }
.palette { background: var(--el-bg-color); border: 1px solid var(--el-border-color-lighter); border-radius: 8px; padding: 12px; }
.palette-title { font-weight: 600; font-size: 13px; margin-bottom: 10px; }
.palette-item { display: flex; align-items: center; gap: 8px; padding: 8px 10px; margin-bottom: 6px; border: 1px solid var(--el-border-color-lighter); border-left-width: 3px; border-radius: 6px; cursor: grab; font-size: 13px; }
.palette-hint { font-size: 11px; color: var(--el-text-color-secondary); line-height: 1.5; }
.palette-actions { display: flex; flex-direction: column; gap: 6px; margin-top: 8px; }
.connect-hint { margin-top: 8px; font-size: 12px; color: var(--el-color-primary); }
.canvas-wrap { display: flex; flex-direction: column; gap: 10px; min-height: 560px; }
.canvas {
  position: relative; flex: 1; min-height: 560px; border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter); overflow: hidden; cursor: grab;
  background-color: var(--el-fill-color-lighter);
  background-image: radial-gradient(circle, var(--el-border-color) 1px, transparent 1px);
  background-size: 20px 20px;
}
.canvas:active { cursor: grabbing; }
.canvas-toolbar {
  position: absolute; top: 10px; left: 10px; z-index: 10;
  display: flex; align-items: center; gap: 8px;
  background: var(--el-bg-color); border-radius: 6px; padding: 4px 8px;
  border: 1px solid var(--el-border-color-lighter); box-shadow: 0 1px 4px rgba(0,0,0,.06);
}
.zoom-label { font-size: 12px; color: var(--el-text-color-secondary); min-width: 36px; }
.canvas-content { position: absolute; inset: 0; transform-origin: 0 0; }
.edges-layer { position: absolute; inset: 0; width: 100%; height: 100%; overflow: visible; pointer-events: none; }
.edge-path { pointer-events: stroke; cursor: pointer; }
.flow-node {
  position: absolute; width: 160px; min-height: 72px; padding: 10px 16px 14px;
  background: var(--el-bg-color); border: 2px solid var(--el-border-color); border-radius: 10px;
  cursor: move; z-index: 2; box-sizing: border-box;
}
.flow-node.selected { box-shadow: 0 0 0 2px var(--node-accent); }
.flow-node.connecting { box-shadow: 0 0 0 2px var(--el-color-primary); }
.node-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 4px; }
.node-type { font-size: 10px; color: var(--el-text-color-secondary); font-weight: 600; }
.node-label { font-size: 13px; font-weight: 600; word-break: break-all; }
.port {
  position: absolute; top: 50%; transform: translateY(-50%);
  width: 14px; height: 14px; border-radius: 50%;
  border: 2px solid var(--node-accent); background: #fff; cursor: crosshair; padding: 0; font-size: 0; line-height: 0;
}
.port-in { left: -7px; }
.port-out { right: -7px; }
.port.active { background: var(--el-color-primary); border-color: var(--el-color-primary); }
.node-inspector { padding: 12px; border: 1px solid var(--el-border-color-lighter); border-radius: 8px; }
.inspector-head { display: flex; justify-content: space-between; margin-bottom: 8px; }
.run-log { border-radius: 8px; padding: 10px; background: #0f172a; color: #e2e8f0; }
.run-log pre { margin: 8px 0 0; font-size: 11px; max-height: 120px; overflow: auto; white-space: pre-wrap; }
@media (max-width: 900px) { .flow-editor { grid-template-columns: 1fr; } }
</style>
