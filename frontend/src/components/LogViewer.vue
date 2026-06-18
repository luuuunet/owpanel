<script setup lang="ts">
import { computed } from 'vue'
import { highlightLogText, type LogKind } from '@/utils/logHighlight'

const props = withDefaults(
  defineProps<{
    content: string
    kind?: LogKind
    emptyText?: string
    maxHeight?: string
  }>(),
  {
    kind: 'generic',
    emptyText: '',
    maxHeight: '200px',
  },
)

const html = computed(() => {
  const raw = (props.content || '').trim()
  if (!raw) return ''
  return highlightLogText(raw, props.kind)
})
</script>

<template>
  <pre
    class="log-viewer"
    :class="[`log-viewer--${kind}`]"
    :style="{ maxHeight }"
  >
    <code v-if="html" class="log-viewer-code" v-html="html" />
    <span v-else class="log-viewer-empty">{{ emptyText }}</span>
  </pre>
</template>

<style scoped>
.log-viewer {
  margin: 0;
  padding: 10px 12px;
  background: #1e1e1e;
  border-radius: 8px;
  font-size: 12px;
  line-height: 1.55;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-all;
  border: 1px solid rgba(255, 255, 255, 0.08);
}
.log-viewer-code {
  font-family: Consolas, 'Cascadia Code', Monaco, monospace;
  color: #d4d4d4;
}
.log-viewer-empty {
  color: var(--el-text-color-secondary);
  font-family: inherit;
}
.log-viewer :deep(.log-hl-ip) { color: #4fc3f7; }
.log-viewer :deep(.log-hl-date) { color: #888; }
.log-viewer :deep(.log-hl-method) { color: #c792ea; font-weight: 600; }
.log-viewer :deep(.log-hl-path) { color: #ffcb6b; }
.log-viewer :deep(.log-hl-status) { font-weight: 700; }
.log-viewer :deep(.log-hl-status-2xx) { color: #89ddff; }
.log-viewer :deep(.log-hl-status-3xx) { color: #82aaff; }
.log-viewer :deep(.log-hl-status-4xx) { color: #ff9800; }
.log-viewer :deep(.log-hl-status-5xx) { color: #f07178; }
.log-viewer :deep(.log-hl-status-other) { color: #d4d4d4; }
.log-viewer :deep(.log-hl-size) { color: #a6accd; }
.log-viewer :deep(.log-hl-quoted) { color: #9cdcfe; }
.log-viewer :deep(.log-hl-level) { font-weight: 700; }
.log-viewer :deep(.log-hl-level-error),
.log-viewer :deep(.log-hl-level-crit),
.log-viewer :deep(.log-hl-level-alert),
.log-viewer :deep(.log-hl-level-emerg) { color: #f07178; }
.log-viewer :deep(.log-hl-level-warn) { color: #ff9800; }
.log-viewer :deep(.log-hl-level-notice),
.log-viewer :deep(.log-hl-level-info) { color: #82aaff; }
.log-viewer :deep(.log-hl-error-msg) { color: #f07178; }
.log-viewer :deep(.log-hl-pid) { color: #676e95; }
</style>
