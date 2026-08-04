import { ref } from 'vue'

const STORAGE_KEY = 'owpanel-nav-recent'
const MAX_RECENT = 10

export interface NavRecentEntry {
  id?: string
  path: string
  titleKey: string
  groupTitleKey: string
  icon: string
  query?: Record<string, string>
  visitedAt: number
}

function recentKey(entry: Pick<NavRecentEntry, 'path' | 'query' | 'id'>) {
  if (entry.id) return entry.id
  const q = entry.query ? JSON.stringify(entry.query) : ''
  return `${entry.path}?${q}`
}

function readRecent(): NavRecentEntry[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw)
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
}

function writeRecent(list: NavRecentEntry[]) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(list.slice(0, MAX_RECENT)))
}

const recentList = ref<NavRecentEntry[]>(readRecent())

export function useNavRecent() {
  function recordVisit(entry: Omit<NavRecentEntry, 'visitedAt'>) {
    const next: NavRecentEntry = { ...entry, visitedAt: Date.now() }
    const key = recentKey(next)
    const filtered = recentList.value.filter((r) => recentKey(r) !== key)
    recentList.value = [next, ...filtered].slice(0, MAX_RECENT)
    writeRecent(recentList.value)
  }

  function clearRecent() {
    recentList.value = []
    localStorage.removeItem(STORAGE_KEY)
  }

  return { recentList, recordVisit, clearRecent }
}
