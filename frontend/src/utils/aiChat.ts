export interface CodeBlock {
  lang: string
  code: string
}

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

function inlineFormat(s: string): string {
  let out = s
  out = out.replace(/`([^`]+)`/g, '<code class="md-inline-code">$1</code>')
  out = out.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
  out = out.replace(/\*([^*]+)\*/g, '<em>$1</em>')
  return out
}

export function extractCodeBlocks(text: string): CodeBlock[] {
  const blocks: CodeBlock[] = []
  const re = /```([a-zA-Z0-9_-]*)\n([\s\S]*?)```/g
  let m: RegExpExecArray | null
  while ((m = re.exec(text)) !== null) {
    blocks.push({ lang: m[1] || 'text', code: m[2].replace(/\n$/, '') })
  }
  return blocks
}

export function renderMarkdown(text: string): string {
  if (!text.trim()) return ''

  const parts: string[] = []
  const fenceRe = /```([a-zA-Z0-9_-]*)\n([\s\S]*?)```/g
  let last = 0
  let m: RegExpExecArray | null

  while ((m = fenceRe.exec(text)) !== null) {
    if (m.index > last) {
      parts.push(renderMarkdownBlocks(text.slice(last, m.index)))
    }
    const lang = escapeHtml(m[1] || 'text')
    const code = escapeHtml(m[2].replace(/\n$/, ''))
    parts.push(
      `<div class="md-code-block" data-lang="${lang}">` +
        `<div class="md-code-head"><span class="md-code-lang">${lang}</span>` +
        `<button type="button" class="md-code-copy" data-copy>Copy</button></div>` +
        `<pre><code>${code}</code></pre></div>`,
    )
    last = m.index + m[0].length
  }
  if (last < text.length) {
    parts.push(renderMarkdownBlocks(text.slice(last)))
  }
  return parts.join('')
}

function renderMarkdownBlocks(raw: string): string {
  const lines = raw.split('\n')
  const out: string[] = []
  let inUl = false
  let inOl = false

  const closeLists = () => {
    if (inUl) {
      out.push('</ul>')
      inUl = false
    }
    if (inOl) {
      out.push('</ol>')
      inOl = false
    }
  }

  for (const line of lines) {
    const trimmed = line.trim()
    if (!trimmed) {
      closeLists()
      continue
    }
    if (/^#{1,3}\s+/.test(trimmed)) {
      closeLists()
      const level = trimmed.match(/^#+/)?.[0].length || 2
      const tag = level === 1 ? 'h3' : level === 2 ? 'h4' : 'h5'
      out.push(`<${tag} class="md-heading">${inlineFormat(escapeHtml(trimmed.replace(/^#+\s+/, '')))}</${tag}>`)
      continue
    }
    if (/^[-*]\s+/.test(trimmed)) {
      if (!inUl) {
        closeLists()
        out.push('<ul class="md-list">')
        inUl = true
      }
      out.push(`<li>${inlineFormat(escapeHtml(trimmed.replace(/^[-*]\s+/, '')))}</li>`)
      continue
    }
    if (/^\d+\.\s+/.test(trimmed)) {
      if (!inOl) {
        closeLists()
        out.push('<ol class="md-list">')
        inOl = true
      }
      out.push(`<li>${inlineFormat(escapeHtml(trimmed.replace(/^\d+\.\s+/, '')))}</li>`)
      continue
    }
    closeLists()
    out.push(`<p class="md-p">${inlineFormat(escapeHtml(trimmed))}</p>`)
  }
  closeLists()
  return out.join('')
}

export async function copyText(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch {
    return false
  }
}

export async function revealTextStream(
  full: string,
  onUpdate: (partial: string) => void,
  signal?: AbortSignal,
): Promise<void> {
  if (!full) {
    onUpdate('')
    return
  }
  const step = Math.max(2, Math.ceil(full.length / 120))
  for (let i = 0; i < full.length; i += step) {
    if (signal?.aborted) return
    onUpdate(full.slice(0, Math.min(i + step, full.length)))
    await new Promise((r) => setTimeout(r, 12))
  }
  onUpdate(full)
}

export interface SSEStreamOptions {
  url: string
  body: Record<string, unknown>
  signal?: AbortSignal
  onChunk: (text: string) => void
  onError?: (msg: string) => void
  onDone?: () => void
}

export async function streamSSE(opts: SSEStreamOptions): Promise<void> {
  const token = localStorage.getItem('token')
  const res = await fetch(opts.url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify(opts.body),
    signal: opts.signal,
  })
  if (!res.ok) {
    const errText = await res.text()
    opts.onError?.(errText || `HTTP ${res.status}`)
    return
  }
  const reader = res.body?.getReader()
  if (!reader) {
    opts.onError?.('No response body')
    return
  }
  const decoder = new TextDecoder()
  let buffer = ''
  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    const lines = buffer.split('\n')
    buffer = lines.pop() || ''
    for (const line of lines) {
      const trimmed = line.trim()
      if (!trimmed.startsWith('data:')) continue
      const json = trimmed.slice(5).trim()
      if (!json) continue
      try {
        const ev = JSON.parse(json) as { content?: string; done?: boolean; error?: string }
        if (ev.error) opts.onError?.(ev.error)
        if (ev.content) opts.onChunk(ev.content)
        if (ev.done) opts.onDone?.()
      } catch {
        /* skip malformed */
      }
    }
  }
  opts.onDone?.()
}

export function newChatId(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

/** Vue @click 会把 Event 当作第一个参数传入，需过滤 */
export function resolveChatInputText(preset: unknown, input: string): string {
  if (typeof preset === 'string') return preset.trim()
  return input.trim()
}

/** 发给 API 的对话历史不含旧图片，避免重复上传 base64 */
export function chatHistoryForApi(messages: Array<{ role: string; content: string; images?: string[] }>) {
  return messages.map((m) => ({ role: m.role, content: m.content }))
}
