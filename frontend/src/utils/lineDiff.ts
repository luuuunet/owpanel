export interface DiffLine {
  type: 'add' | 'del' | 'same'
  text: string
  oldNum?: number
  newNum?: number
}

export interface DiffStats {
  added: number
  removed: number
  changed: boolean
}

/** Simple line diff for AI preview (Myers-style LCS on lines). */
export function diffLines(before: string, after: string): DiffLine[] {
  const a = before.replace(/\r\n/g, '\n').split('\n')
  const b = after.replace(/\r\n/g, '\n').split('\n')
  const n = a.length
  const m = b.length
  const dp: number[][] = Array.from({ length: n + 1 }, () => Array(m + 1).fill(0))
  for (let i = n - 1; i >= 0; i--) {
    for (let j = m - 1; j >= 0; j--) {
      dp[i][j] = a[i] === b[j] ? dp[i + 1][j + 1] + 1 : Math.max(dp[i + 1][j], dp[i][j + 1])
    }
  }

  const out: DiffLine[] = []
  let i = 0
  let j = 0
  let oldNum = 1
  let newNum = 1
  while (i < n && j < m) {
    if (a[i] === b[j]) {
      out.push({ type: 'same', text: a[i], oldNum, newNum })
      i++
      j++
      oldNum++
      newNum++
    } else if (dp[i + 1][j] >= dp[i][j + 1]) {
      out.push({ type: 'del', text: a[i], oldNum })
      i++
      oldNum++
    } else {
      out.push({ type: 'add', text: b[j], newNum })
      j++
      newNum++
    }
  }
  while (i < n) {
    out.push({ type: 'del', text: a[i], oldNum })
    i++
    oldNum++
  }
  while (j < m) {
    out.push({ type: 'add', text: b[j], newNum })
    j++
    newNum++
  }
  return out
}

export function diffStats(before: string, after: string): DiffStats {
  const lines = diffLines(before, after)
  let added = 0
  let removed = 0
  for (const l of lines) {
    if (l.type === 'add') added++
    if (l.type === 'del') removed++
  }
  return { added, removed, changed: added > 0 || removed > 0 }
}
