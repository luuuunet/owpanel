<script setup lang="ts">
import { computed } from 'vue'
import { diffLines } from '@/utils/lineDiff'

const props = defineProps<{
  before: string
  after: string
}>()

const rows = computed(() => diffLines(props.before, props.after))
</script>

<template>
  <div class="code-diff">
    <pre class="diff-pre"><code><div
      v-for="(row, i) in rows"
      :key="i"
      class="diff-row"
      :class="row.type"
    ><span v-if="row.type === 'del'" class="sign">-</span><span v-else-if="row.type === 'add'" class="sign">+</span><span v-else class="sign"> </span><span class="nums"><span v-if="row.oldNum">{{ row.oldNum }}</span><span v-else class="gap" /><span v-if="row.newNum">{{ row.newNum }}</span><span v-else class="gap" /></span><span class="text">{{ row.text || ' ' }}</span></div></code></pre>
  </div>
</template>

<style scoped>
.code-diff {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  overflow: auto;
  max-height: 60vh;
  background: #1e1e1e;
  font-family: Consolas, 'Courier New', monospace;
  font-size: 12px;
  line-height: 1.45;
}
.diff-pre {
  margin: 0;
  padding: 0;
}
.diff-row {
  display: flex;
  gap: 8px;
  padding: 0 10px;
  white-space: pre;
}
.diff-row.same { background: #1e1e1e; color: #d4d4d4; }
.diff-row.add { background: rgba(46, 160, 67, 0.22); color: #aff5b4; }
.diff-row.del { background: rgba(248, 81, 73, 0.22); color: #ffb1af; }
.sign { width: 12px; flex-shrink: 0; user-select: none; opacity: 0.85; }
.nums {
  width: 72px;
  flex-shrink: 0;
  display: flex;
  justify-content: space-between;
  color: #6e7681;
  user-select: none;
}
.gap { display: inline-block; width: 28px; }
.text { flex: 1; min-width: 0; }
</style>
