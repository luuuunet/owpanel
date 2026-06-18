/** Canonical software store categories — keep in sync with backend appstore/categories.go */
export const STORE_CATEGORY_ORDER = [
  'Web服务器',
  '运行环境',
  '数据库',
  '中间件',
  '容器',
  'FTP',
  '邮件',
  '网站',
  '人工智能',
  '图形处理',
  '视频处理',
  '多媒体',
  'DevOps',
  '开发工具',
  'BI',
  'CRM',
  '安全',
  '云存储',
  '工具',
  '生活',
] as const

const CATEGORY_ALIASES: Record<string, string> = {
  邮件服务: '邮件',
  郵件服務: '邮件',
  郵件: '邮件',
  Email: '邮件',
  系统工具: '工具',
  系統工具: '工具',
  'System Tools': '工具',
  存储: '云存储',
  儲存: '云存储',
  雲儲存: '云存储',
  建站: '网站',
  網站: '网站',
  'Web Server': 'Web服务器',
  Web伺服器: 'Web服务器',
  Database: '数据库',
  資料庫: '数据库',
  Runtime: '运行环境',
  執行環境: '运行环境',
  Container: '容器',
  中間件: '中间件',
  開發工具: '开发工具',
  多媒體: '多媒体',
}

export function normalizeStoreCategory(category: string): string {
  return CATEGORY_ALIASES[category] || category
}

/** Drop categories that share the same display label (keeps first occurrence). */
export function dedupeCategoriesByLabel(
  categories: string[],
  labelFn: (category: string) => string,
): string[] {
  const seen = new Set<string>()
  const result: string[] = []
  for (const c of categories) {
    const label = labelFn(c)
    if (seen.has(label)) continue
    seen.add(label)
    result.push(c)
  }
  return result
}

export function orderedStoreCategories(
  rawCategories: string[],
  labelFn?: (category: string) => string,
): string[] {
  const set = new Set(rawCategories.map(normalizeStoreCategory))
  const ordered = STORE_CATEGORY_ORDER.filter(c => set.has(c))
  const rest = [...set].filter(c => !STORE_CATEGORY_ORDER.includes(c as typeof STORE_CATEGORY_ORDER[number])).sort()
  const combined = [...ordered, ...rest]
  return labelFn ? dedupeCategoriesByLabel(combined, labelFn) : combined
}
