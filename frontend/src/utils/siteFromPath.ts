export interface SitePathMatch {
  siteId: number
  siteRoot: string
  domain: string
}

export interface SitePathCandidate {
  id: number
  root_path?: string
  domain?: string
}

export function normalizeFsPath(path: string) {
  return path.replace(/\\/g, '/').replace(/\/+$/, '')
}

/** 根据绝对路径匹配所属网站（最长 root_path 前缀） */
export function resolveSiteForPath(filePath: string, sites: SitePathCandidate[]): SitePathMatch | null {
  const file = normalizeFsPath(filePath)
  if (!file) return null
  let best: SitePathMatch | null = null
  let bestLen = 0
  for (const site of sites) {
    const root = normalizeFsPath(site.root_path || '')
    if (!root || !site.id) continue
    if (file !== root && !file.startsWith(`${root}/`)) continue
    if (root.length > bestLen) {
      bestLen = root.length
      best = {
        siteId: site.id,
        siteRoot: root,
        domain: site.domain || '',
      }
    }
  }
  return best
}
