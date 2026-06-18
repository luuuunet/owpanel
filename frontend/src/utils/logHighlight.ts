export type LogKind = 'access' | 'error' | 'generic'

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
}

function statusClass(code: string): string {
  const n = parseInt(code, 10)
  if (n >= 500) return 'log-status-5xx'
  if (n >= 400) return 'log-status-4xx'
  if (n >= 300) return 'log-status-3xx'
  if (n >= 200) return 'log-status-2xx'
  return 'log-status-other'
}

function highlightAccessLine(line: string): string {
  let s = escapeHtml(line)
  s = s.replace(/^(\d+\.\d+\.\d+\.\d+)/, '<span class="log-hl-ip">$1</span>')
  s = s.replace(/\[([^\]]+)\]/, '[<span class="log-hl-date">$1</span>]')
  s = s.replace(
    /"([A-Z]+) ([^"]*?) HTTP\/[\d.]+"/,
    '"<span class="log-hl-method">$1</span> <span class="log-hl-path">$2</span> HTTP/..."',
  )
  s = s.replace(/" (\d{3}) (\d+)/, (_m, code: string, size: string) =>
    `" <span class="log-hl-status ${statusClass(code)}">${code}</span> <span class="log-hl-size">${size}</span>`,
  )
  s = s.replace(/"([^"]*?)"/g, (m, inner: string) => {
    if (m.includes('log-hl-')) return m
    return `"<span class="log-hl-quoted">${inner}</span>"`
  })
  return s
}

function highlightErrorLine(line: string): string {
  let s = escapeHtml(line)
  s = s.replace(/^(\d{4}\/\d{2}\/\d{2}\s+\d{2}:\d{2}:\d{2})/, '<span class="log-hl-date">$1</span>')
  s = s.replace(
    /\[(error|warn|notice|info|debug|crit|alert|emerg)\]/gi,
    (_m, lvl: string) => `[<span class="log-hl-level log-hl-level-${lvl.toLowerCase()}">${lvl}</span>]`,
  )
  s = s.replace(/open\(\s*"([^"]+)"/g, 'open("<span class="log-hl-path">$1</span>"')
  s = s.replace(/failed \([^)]+\)/g, (m) => `<span class="log-hl-error-msg">${m}</span>`)
  s = s.replace(/(\d+#\d+:\*\d+)/g, '<span class="log-hl-pid">$1</span>')
  return s
}

export function highlightLogLine(line: string, kind: LogKind): string {
  if (!line.trim()) return ''
  if (kind === 'access') return highlightAccessLine(line)
  if (kind === 'error') return highlightErrorLine(line)
  return escapeHtml(line)
}

export function highlightLogText(text: string, kind: LogKind): string {
  if (!text.trim()) return ''
  return text.split('\n').map((line) => highlightLogLine(line, kind)).join('\n')
}
