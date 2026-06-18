<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { EditorView, keymap, lineNumbers, highlightActiveLine } from '@codemirror/view'
import { EditorState, Transaction } from '@codemirror/state'
import { defaultKeymap, history, historyKeymap, redo, undo, undoDepth, redoDepth } from '@codemirror/commands'
import {
  syntaxHighlighting,
  defaultHighlightStyle,
  indentOnInput,
  bracketMatching,
} from '@codemirror/language'
import { oneDark } from '@codemirror/theme-one-dark'
import { javascript } from '@codemirror/lang-javascript'
import { php } from '@codemirror/lang-php'
import { html } from '@codemirror/lang-html'
import { css } from '@codemirror/lang-css'
import { json } from '@codemirror/lang-json'
import { xml } from '@codemirror/lang-xml'
import { python } from '@codemirror/lang-python'
import { markdown } from '@codemirror/lang-markdown'

const props = defineProps<{ modelValue: string; path: string; readOnly?: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [string]; 'history-change': [] }>()

const host = ref<HTMLElement>()
let view: EditorView | null = null
let suppressEmit = false

function langExt(path: string) {
  const ext = path.split('.').pop()?.toLowerCase() || ''
  switch (ext) {
    case 'js':
    case 'mjs':
    case 'cjs':
      return javascript()
    case 'ts':
    case 'tsx':
      return javascript({ typescript: true })
    case 'jsx':
      return javascript({ jsx: true })
    case 'php':
      return php()
    case 'html':
    case 'htm':
      return html()
    case 'css':
    case 'scss':
    case 'less':
      return css()
    case 'json':
      return json()
    case 'xml':
    case 'svg':
    case 'conf':
    case 'nginx':
      return xml()
    case 'py':
      return python()
    case 'md':
      return markdown()
    default:
      return null
  }
}

function replaceAll(content: string, addToHistory = true) {
  if (!view) return
  suppressEmit = true
  view.dispatch({
    changes: { from: 0, to: view.state.doc.length, insert: content },
    ...(addToHistory ? {} : { annotations: Transaction.addToHistory.of(false) }),
  })
  suppressEmit = false
  emit('update:modelValue', content)
}

function createView() {
  if (!host.value) return
  view?.destroy()
  const lang = langExt(props.path)
  const extensions = [
    lineNumbers(),
    history(),
    highlightActiveLine(),
    indentOnInput(),
    bracketMatching(),
    syntaxHighlighting(defaultHighlightStyle),
    oneDark,
    keymap.of([...defaultKeymap, ...historyKeymap]),
    EditorView.updateListener.of((u) => {
      if (u.docChanged && !suppressEmit) {
        emit('update:modelValue', u.state.doc.toString())
      }
      if (u.docChanged || u.selectionSet) {
        emit('history-change')
      }
    }),
  ]
  if (lang) extensions.push(lang)
  if (props.readOnly) {
    extensions.push(EditorState.readOnly.of(true))
  }

  view = new EditorView({
    state: EditorState.create({ doc: props.modelValue, extensions }),
    parent: host.value,
  })
}

function undoEdit() {
  if (view) undo(view)
}

function redoEdit() {
  if (view) redo(view)
}

function canUndo() {
  return view ? undoDepth(view.state) > 0 : false
}

function canRedo() {
  return view ? redoDepth(view.state) > 0 : false
}

function focusEditor() {
  view?.focus()
}

defineExpose({ undoEdit, redoEdit, canUndo, canRedo, replaceAll, focusEditor })

watch(
  () => props.path,
  () => createView()
)

watch(
  () => props.readOnly,
  () => createView()
)

watch(
  () => props.modelValue,
  (v) => {
    if (view && v !== view.state.doc.toString()) {
      replaceAll(v, false)
    }
  }
)

onMounted(createView)
onUnmounted(() => view?.destroy())
</script>

<template>
  <div ref="host" class="code-editor" />
</template>

<style scoped>
.code-editor {
  width: 100%;
  display: block;
  min-height: 420px;
  max-height: 70vh;
  overflow: auto;
  border-radius: 6px;
  font-size: 13px;
}
.code-editor :deep(.cm-editor) {
  width: 100%;
  height: 100%;
  min-height: inherit;
}
.code-editor :deep(.cm-scroller) {
  font-family: Consolas, 'Courier New', monospace;
}
</style>
