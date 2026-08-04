import { ref } from 'vue'
import api from '@/api'
import { apiContentLang } from '@/locales'

export type QuickSearchKind = 'page' | 'tab' | 'resource'

export interface QuickSearchHit {
  id: string
  kind: QuickSearchKind
  path: string
  query?: Record<string, string>
  /** Display title (already resolved) */
  title: string
  subtitle?: string
  /** Display category label */
  category: string
  icon: string
  keywords?: string[]
  externalUrl?: string
  /** For recent visits / i18n helpers */
  titleKey: string
  groupTitleKey: string
}

const CACHE_MS = 5 * 60 * 1000
const MAX_RESULTS = 40

function norm(s: string) {
  return s.toLowerCase().replace(/[\s._\-:/]+/g, '')
}

function rawKey(title: string) {
  return `@raw:${title}`
}

function asList(data: unknown): any[] {
  if (Array.isArray(data)) return data
  if (data && typeof data === 'object') {
    const o = data as Record<string, unknown>
    for (const k of ['items', 'list', 'data', 'rows', 'apps', 'containers']) {
      if (Array.isArray(o[k])) return o[k] as any[]
    }
  }
  return []
}

async function safeGet(path: string, params?: Record<string, unknown>) {
  try {
    const res: any = await api.get(path, params ? { params } : undefined)
    return asList(res.data)
  } catch {
    return []
  }
}

/** Static deep-link tabs & aliases (no API). */
export function buildTabHits(
  canAccess: (path: string) => boolean,
  t: (key: string) => string,
): QuickSearchHit[] {
  const hits: QuickSearchHit[] = []

  const add = (hit: Omit<QuickSearchHit, 'titleKey' | 'groupTitleKey' | 'kind'> & { kind?: QuickSearchKind }) => {
    if (!canAccess(hit.path)) return
    hits.push({
      ...hit,
      kind: hit.kind || 'tab',
      titleKey: rawKey(hit.title),
      groupTitleKey: rawKey(hit.category),
    })
  }

  if (canAccess('/protection')) {
    const tabs: Array<{ tab: string; titleKey: string; keywords: string[]; icon: string }> = [
      { tab: 'cache', titleKey: 'protection.tabs.cache', keywords: ['cdn', '缓存', 'cache', 'purge'], icon: 'Lightning' },
      { tab: 'firewall', titleKey: 'protection.tabs.firewall', keywords: ['防火墙', 'firewall', 'ufw', 'iptables', '端口放行'], icon: 'Lock' },
      { tab: 'nginx', titleKey: 'protection.tabs.nginx', keywords: ['nginx', 'openresty', 'web服务器', '反代'], icon: 'Monitor' },
      { tab: 'waf', titleKey: 'protection.tabs.waf', keywords: ['waf', '防火墙', '攻击防护'], icon: 'Warning' },
      { tab: 'workers', titleKey: 'protection.tabs.workers', keywords: ['edge', 'worker', '边缘'], icon: 'Cpu' },
      { tab: 'kafka', titleKey: 'protection.tabs.kafka', keywords: ['kafka', '加速'], icon: 'Connection' },
      { tab: 'cilium', titleKey: 'protection.tabs.cilium', keywords: ['cilium', 'ebpf', '网络'], icon: 'Share' },
      { tab: 'security', titleKey: 'protection.tabs.security', keywords: ['安全检测', 'security', '扫描'], icon: 'CircleCheck' },
    ]
    for (const x of tabs) {
      const title = t(x.titleKey)
      add({
        id: `tab:protection:${x.tab}`,
        path: '/protection',
        query: { tab: x.tab },
        title,
        subtitle: t('menu.protection'),
        category: t('nav.catTabs'),
        icon: x.icon,
        keywords: x.keywords,
      })
    }
  }

  if (canAccess('/software')) {
    add({
      id: 'tab:software:store',
      path: '/software',
      query: { tab: 'store' },
      title: t('software.storeTab'),
      subtitle: t('menu.software'),
      category: t('nav.catTabs'),
      icon: 'ShoppingCart',
      keywords: ['应用商店', '软件商店', 'store', 'app store', '安装软件'],
    })
    add({
      id: 'tab:software:installed',
      path: '/software',
      query: { tab: 'installed' },
      title: t('software.installedTab'),
      subtitle: t('menu.software'),
      category: t('nav.catTabs'),
      icon: 'Box',
      keywords: ['已安装', 'installed', '我的应用', '已搭建'],
    })
  }

  if (canAccess('/toolbox')) {
    const tabs = [
      { tab: 'system', titleKey: 'toolboxPage.tabSystem', kw: ['系统', '健康', '进程', '负载'] },
      { tab: 'bench', titleKey: 'toolboxPage.tabBench', kw: ['性能测试', '测速', 'benchmark', 'cpu', '磁盘'] },
      { tab: 'ports', titleKey: 'toolboxPage.tabPorts', kw: ['端口', '监听', 'listen'] },
      { tab: 'snippets', titleKey: 'toolboxPage.tabSnippets', kw: ['命令片段', 'snippet', '脚本'] },
      { tab: 'network', titleKey: 'toolboxPage.tabNetwork', kw: ['ping', 'dns', 'traceroute', '网络诊断'] },
    ]
    for (const x of tabs) {
      add({
        id: `tab:toolbox:${x.tab}`,
        path: '/toolbox',
        query: { tab: x.tab },
        title: t(x.titleKey),
        subtitle: t('menu.toolbox'),
        category: t('nav.catTabs'),
        icon: 'Tools',
        keywords: x.kw,
      })
    }
  }

  if (canAccess('/infra-hub')) {
    const tabs = [
      { tab: 'overview', titleKey: 'infraHub.tabOverview' },
      { tab: 'llmops', titleKey: 'infraHub.tabLLMOps', kw: ['llm', '模型', 'ollama'] },
      { tab: 'dataops', titleKey: 'infraHub.tabDataOps', kw: ['向量', 'vector', '数据'] },
      { tab: 'aiops', titleKey: 'infraHub.tabAIOps', kw: ['aiops', '运维'] },
      { tab: 'secops', titleKey: 'infraHub.tabSecOps', kw: ['安全运营', 'secops'] },
      { tab: 'orchestration', titleKey: 'infraHub.tabOrchestration', kw: ['编排', 'k8s'] },
    ]
    for (const x of tabs) {
      add({
        id: `tab:infra:${x.tab}`,
        path: '/infra-hub',
        query: { tab: x.tab },
        title: t(x.titleKey),
        subtitle: t('menu.infraHub'),
        category: t('nav.catTabs'),
        icon: 'Platform',
        keywords: x.kw || [],
      })
    }
  }

  if (canAccess('/runtimes')) {
    for (const tab of ['php', 'java', 'nodejs', 'go', 'rust', 'python', 'dotnet']) {
      const title = t(`runtime.tab.${tab}`)
      add({
        id: `tab:runtime:${tab}`,
        path: '/runtimes',
        query: { tab },
        title,
        subtitle: t('menu.runtimes'),
        category: t('nav.catTabs'),
        icon: 'Platform',
        keywords: [tab, title],
      })
    }
  }

  // Common aliases for top menu pages (helps Chinese/English mixed queries)
  const pageAliases: Array<{ path: string; keywords: string[]; icon?: string }> = [
    { path: '/websites', keywords: ['网站', '站点', '域名', 'site', 'website', '建站'] },
    { path: '/databases', keywords: ['数据库', 'mysql', 'postgres', 'mongodb', 'redis', 'maria'] },
    { path: '/ssl', keywords: ['证书', 'https', 'letsencrypt', 'acme', 'ssl'] },
    { path: '/docker', keywords: ['容器', 'container', '镜像'] },
    { path: '/compose', keywords: ['编排', 'stack', '一键部署'] },
    { path: '/ftp', keywords: ['ftp', 'sftp', '文件传输'] },
    { path: '/mail', keywords: ['邮件', '邮箱', 'smtp', 'postfix', 'webmail'] },
    { path: '/firewall', keywords: ['防火墙'], icon: 'Lock' },
    { path: '/cron', keywords: ['计划任务', 'crontab', '定时'] },
    { path: '/backup', keywords: ['备份', '快照', 'backup'] },
    { path: '/files', keywords: ['文件管理', 'file manager'] },
    { path: '/dns', keywords: ['dns', '解析', 'cloudflare'] },
    { path: '/wordpress', keywords: ['wordpress', 'wp', '建站'] },
    { path: '/terminal', keywords: ['ssh', '终端', '堡垒机', 'shell'] },
    { path: '/php', keywords: ['php', 'fpm'] },
    { path: '/settings', keywords: ['面板设置', '配置', 'totp', '二步验证'] },
  ]
  for (const a of pageAliases) {
    if (!canAccess(a.path) && a.path !== '/firewall') continue
    // firewall redirect lives under protection — still useful as alias hit
    if (a.path === '/firewall' && canAccess('/protection')) {
      add({
        id: 'alias:firewall',
        kind: 'tab',
        path: '/protection',
        query: { tab: 'firewall' },
        title: t('protection.tabs.firewall'),
        subtitle: t('menu.protection'),
        category: t('nav.catTabs'),
        icon: a.icon || 'Lock',
        keywords: a.keywords,
      })
    }
  }

  return hits
}

export function pageHitFromMenu(opts: {
  path: string
  title: string
  category: string
  icon: string
  titleKey: string
  groupTitleKey: string
  externalUrl?: string
  keywords?: string[]
}): QuickSearchHit {
  return {
    id: `page:${opts.path}:${opts.externalUrl || ''}`,
    kind: 'page',
    path: opts.path,
    title: opts.title,
    category: opts.category,
    icon: opts.icon,
    titleKey: opts.titleKey,
    groupTitleKey: opts.groupTitleKey,
    externalUrl: opts.externalUrl,
    keywords: opts.keywords || [],
  }
}

function scoreHit(hit: QuickSearchHit, q: string, nq: string): number {
  const title = hit.title.toLowerCase()
  const subtitle = (hit.subtitle || '').toLowerCase()
  const category = hit.category.toLowerCase()
  const path = hit.path.toLowerCase()
  const kw = (hit.keywords || []).map((k) => k.toLowerCase())
  const blob = [title, subtitle, category, path, ...kw].join(' ')
  const nblob = norm(blob)

  if (title === q) return 120
  if (title.startsWith(q)) return 100
  if (title.includes(q)) return 85
  if (kw.some((k) => k === q || k.startsWith(q))) return 80
  if (kw.some((k) => k.includes(q))) return 70
  if (subtitle.includes(q) || category.includes(q)) return 60
  if (path.includes(q.replace(/^\//, ''))) return 55
  if (nq.length >= 2 && nblob.includes(nq)) return 50
  if (blob.includes(q)) return 40
  return 0
}

export function rankQuickSearch(hits: QuickSearchHit[], query: string): QuickSearchHit[] {
  const q = query.trim().toLowerCase()
  if (!q) return []
  const nq = norm(q)
  const scored = hits
    .map((h) => ({ h, s: scoreHit(h, q, nq) }))
    .filter((x) => x.s > 0)
    .sort((a, b) => {
      if (b.s !== a.s) return b.s - a.s
      const kindOrder = { page: 0, tab: 1, resource: 2 } as const
      return kindOrder[a.h.kind] - kindOrder[b.h.kind]
    })
  return scored.slice(0, MAX_RESULTS).map((x) => x.h)
}

export function useQuickSearch() {
  const resourceHits = ref<QuickSearchHit[]>([])
  const loading = ref(false)
  const loadedAt = ref(0)
  let inflight: Promise<void> | null = null

  async function ensureResourceIndex(
    canAccess: (path: string) => boolean,
    t: (key: string) => string,
    lang?: string,
  ) {
    if (Date.now() - loadedAt.value < CACHE_MS && resourceHits.value.length) return
    if (inflight) return inflight

    inflight = (async () => {
      loading.value = true
      const out: QuickSearchHit[] = []
      const tasks: Promise<void>[] = []

      const push = (hit: Omit<QuickSearchHit, 'titleKey' | 'groupTitleKey' | 'kind'> & { kind?: QuickSearchKind }) => {
        out.push({
          ...hit,
          kind: hit.kind || 'resource',
          titleKey: rawKey(hit.title),
          groupTitleKey: rawKey(hit.category),
        })
      }

      if (canAccess('/websites')) {
        tasks.push((async () => {
          const lists = await Promise.all([
            safeGet('/websites/projects', { type: 'php' }),
            safeGet('/websites/projects', { type: 'node' }),
            safeGet('/websites/projects', { type: 'java' }),
          ])
          for (const rows of lists) {
            for (const row of rows) {
              const domain = String(row.domain || '').trim()
              if (!domain) continue
              push({
                id: `site:${row.source || row.project_type || 'web'}:${row.id}`,
                path: '/websites',
                query: { q: domain, type: row.project_type || 'php' },
                title: domain,
                subtitle: [row.project_type, row.remark, row.php_version].filter(Boolean).join(' · ') || t('menu.website'),
                category: t('nav.catSites'),
                icon: 'Link',
                keywords: [domain, row.remark, row.root_path, '网站', '站点'].filter(Boolean),
              })
            }
          }
        })())
      }

      if (canAccess('/software')) {
        tasks.push((async () => {
          const [installed, store] = await Promise.all([
            safeGet('/software/installed'),
            safeGet('/software/store'),
          ])
          const seen = new Set<string>()
          for (const app of installed) {
            const key = String(app.key || '')
            const name = String(app.name || key)
            if (!key && !name) continue
            seen.add(key)
            push({
              id: `app:installed:${key}:${app.version || ''}`,
              path: '/software',
              query: { tab: 'installed', q: name },
              title: name,
              subtitle: [app.version && `v${app.version}`, app.bind_domain, t('software.installedTab')].filter(Boolean).join(' · '),
              category: t('nav.catApps'),
              icon: 'Box',
              keywords: [key, name, app.category, app.bind_domain, '已安装', 'installed'].filter(Boolean),
            })
          }
          for (const app of store) {
            const key = String(app.key || app.family_key || '')
            const name = String(app.name || key)
            if (!key || seen.has(key)) continue
            // Prefer installed entries; still index store for discoverability
            push({
              id: `app:store:${key}`,
              path: '/software',
              query: { tab: 'store', q: name },
              title: name,
              subtitle: [app.category, app.installed ? t('common.installed') : t('software.storeTab')].filter(Boolean).join(' · '),
              category: t('nav.catApps'),
              icon: 'ShoppingCart',
              keywords: [key, name, app.category, app.description, '软件', '应用'].filter(Boolean),
            })
          }
        })())
      }

      if (canAccess('/databases')) {
        tasks.push((async () => {
          const rows = await safeGet('/databases')
          for (const row of rows) {
            const name = String(row.name || row.db_name || '').trim()
            if (!name) continue
            push({
              id: `db:${row.id}`,
              path: '/databases',
              query: { q: name },
              title: name,
              subtitle: [row.engine || row.type, row.username].filter(Boolean).join(' · ') || t('menu.database'),
              category: t('nav.catDatabases'),
              icon: 'Coin',
              keywords: [name, row.engine, row.type, '数据库', 'mysql'].filter(Boolean),
            })
          }
        })())
      }

      if (canAccess('/ssl')) {
        tasks.push((async () => {
          const rows = await safeGet('/ssl')
          for (const row of rows) {
            const domain = String(row.domain || row.domains || row.name || '').trim()
            if (!domain) continue
            push({
              id: `ssl:${row.id || domain}`,
              path: '/ssl',
              query: { q: domain },
              title: domain,
              subtitle: [row.issuer, row.status].filter(Boolean).join(' · ') || t('menu.ssl'),
              category: t('nav.catSSL'),
              icon: 'Lock',
              keywords: [domain, 'ssl', '证书', 'https'].filter(Boolean),
            })
          }
        })())
      }

      if (canAccess('/docker')) {
        tasks.push((async () => {
          const rows = await safeGet('/docker/containers')
          for (const row of rows) {
            const name = String(row.name || row.Names || row.names || '').replace(/^\//, '').trim()
            const id = String(row.id || row.Id || '').slice(0, 12)
            if (!name && !id) continue
            push({
              id: `docker:${id || name}`,
              path: '/docker',
              query: { q: name || id },
              title: name || id,
              subtitle: [row.image || row.Image, row.state || row.State || row.status].filter(Boolean).join(' · ') || t('menu.docker'),
              category: t('nav.catDocker'),
              icon: 'Box',
              keywords: [name, id, row.image, '容器', 'docker'].filter(Boolean),
            })
          }
        })())
      }

      if (canAccess('/compose')) {
        tasks.push((async () => {
          const rows = await safeGet('/compose')
          for (const row of rows) {
            const name = String(row.name || row.title || '').trim()
            if (!name) continue
            push({
              id: `compose:${row.id}`,
              path: '/compose',
              query: { q: name },
              title: name,
              subtitle: [row.status, row.path].filter(Boolean).join(' · ') || t('menu.compose'),
              category: t('nav.catDocker'),
              icon: 'Grid',
              keywords: [name, 'compose', '编排', 'stack'].filter(Boolean),
            })
          }
        })())
      }

      if (canAccess('/ftp')) {
        tasks.push((async () => {
          const rows = await safeGet('/ftp')
          for (const row of rows) {
            const user = String(row.username || row.user || row.name || '').trim()
            if (!user) continue
            push({
              id: `ftp:${row.id || user}`,
              path: '/ftp',
              query: { q: user },
              title: user,
              subtitle: row.path || t('menu.ftp'),
              category: t('nav.catFTP'),
              icon: 'Upload',
              keywords: [user, 'ftp', row.path].filter(Boolean),
            })
          }
        })())
      }

      if (canAccess('/wordpress')) {
        tasks.push((async () => {
          const rows = await safeGet('/wordpress')
          for (const row of rows) {
            const domain = String(row.domain || row.name || '').trim()
            if (!domain) continue
            push({
              id: `wp:${row.id || domain}`,
              path: '/wordpress',
              query: { q: domain },
              title: domain,
              subtitle: [row.version, 'WordPress'].filter(Boolean).join(' · '),
              category: t('nav.catSites'),
              icon: 'Reading',
              keywords: [domain, 'wordpress', 'wp'].filter(Boolean),
            })
          }
        })())
      }

      if (canAccess('/cron')) {
        tasks.push((async () => {
          const rows = await safeGet('/cron')
          for (const row of rows) {
            const name = String(row.name || row.command || '').trim()
            if (!name) continue
            push({
              id: `cron:${row.id}`,
              path: '/cron',
              query: { q: name },
              title: name.length > 48 ? `${name.slice(0, 48)}…` : name,
              subtitle: row.schedule || t('menu.cron'),
              category: t('nav.catAutomation'),
              icon: 'Timer',
              keywords: [name, row.command, row.schedule, '计划任务', 'cron'].filter(Boolean),
            })
          }
        })())
      }

      if (canAccess('/backup')) {
        tasks.push((async () => {
          const rows = await safeGet('/backup')
          for (const row of rows) {
            const name = String(row.name || row.target || '').trim()
            if (!name) continue
            push({
              id: `backup:${row.id}`,
              path: '/backup',
              query: { q: name },
              title: name,
              subtitle: [row.type, row.schedule].filter(Boolean).join(' · ') || t('menu.backup'),
              category: t('nav.catAutomation'),
              icon: 'FolderOpened',
              keywords: [name, '备份', 'backup'].filter(Boolean),
            })
          }
        })())
      }

      if (canAccess('/mail')) {
        tasks.push((async () => {
          const [domains, boxes] = await Promise.all([
            safeGet('/mail/domains'),
            safeGet('/mail/mailboxes'),
          ])
          for (const row of domains) {
            const domain = String(row.domain || row.name || '').trim()
            if (!domain) continue
            push({
              id: `mail-domain:${row.id || domain}`,
              path: '/mail',
              query: { q: domain },
              title: domain,
              subtitle: t('menu.mail'),
              category: t('nav.catMail'),
              icon: 'Message',
              keywords: [domain, '邮件域名', 'mail'].filter(Boolean),
            })
          }
          for (const row of boxes) {
            const email = String(row.email || row.address || `${row.local_part || ''}@${row.domain || ''}`).trim()
            if (!email || email === '@') continue
            push({
              id: `mailbox:${row.id || email}`,
              path: '/mail',
              query: { q: email },
              title: email,
              subtitle: t('nav.catMail'),
              category: t('nav.catMail'),
              icon: 'Message',
              keywords: [email, '邮箱', 'mailbox'].filter(Boolean),
            })
          }
        })())
      }

      if (canAccess('/toolbox')) {
        tasks.push((async () => {
          const rows = await safeGet('/toolbox/snippets', { lang: apiContentLang(lang || 'zh-CN') })
          for (const row of rows) {
            const name = String(row.name || '').trim()
            if (!name) continue
            push({
              id: `snippet:${row.id}`,
              path: '/toolbox',
              query: { tab: 'snippets', q: name },
              title: name,
              subtitle: row.category || t('toolboxPage.tabSnippets'),
              category: t('nav.catTools'),
              icon: 'Tools',
              keywords: [name, row.command, row.remark, '命令片段'].filter(Boolean),
            })
          }
        })())
      }

      if (canAccess('/dns')) {
        tasks.push((async () => {
          const rows = await safeGet('/dns/zones')
          for (const row of rows) {
            const name = String(row.name || row.zone || row.domain || '').trim()
            if (!name) continue
            push({
              id: `dns:${row.id || name}`,
              path: '/dns',
              query: { q: name },
              title: name,
              subtitle: t('menu.dns'),
              category: t('nav.catDomains'),
              icon: 'Compass',
              keywords: [name, 'dns', '解析'].filter(Boolean),
            })
          }
        })())
      }

      if (canAccess('/users')) {
        tasks.push((async () => {
          const rows = await safeGet('/users')
          for (const row of rows) {
            const name = String(row.username || row.name || '').trim()
            if (!name) continue
            push({
              id: `user:${row.id || name}`,
              path: '/users',
              query: { q: name },
              title: name,
              subtitle: row.role || t('menu.users'),
              category: t('nav.catUsers'),
              icon: 'User',
              keywords: [name, row.email, '用户'].filter(Boolean),
            })
          }
        })())
      }

      await Promise.all(tasks)
      resourceHits.value = out
      loadedAt.value = Date.now()
      loading.value = false
      inflight = null
    })()

    return inflight
  }

  function invalidate() {
    loadedAt.value = 0
  }

  return {
    resourceHits,
    loading,
    ensureResourceIndex,
    invalidate,
  }
}
