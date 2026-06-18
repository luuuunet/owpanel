import { panelStaticPath } from '@/utils/panelBase'

export type CrawlerIconKey =
  | 'google'
  | 'apple'
  | 'openai'
  | 'anthropic'
  | 'bing'
  | 'baidu'
  | 'meta'
  | 'twitter'
  | 'yandex'
  | 'scraper'

export interface CrawlerIconMeta {
  bg: string
  label: string
}

const crawlerMeta: Record<CrawlerIconKey, CrawlerIconMeta> = {
  google: { bg: '#4285F4', label: 'G' },
  apple: { bg: '#555555', label: 'A' },
  openai: { bg: '#10A37F', label: 'AI' },
  anthropic: { bg: '#D4A574', label: 'A' },
  bing: { bg: '#008373', label: 'B' },
  baidu: { bg: '#2932E1', label: 'B' },
  meta: { bg: '#0081FB', label: 'M' },
  twitter: { bg: '#000000', label: 'X' },
  yandex: { bg: '#FC3F1D', label: 'Y' },
  scraper: { bg: '#6366F1', label: 'Bot' },
}

export function normalizeCrawlerIconKey(icon?: string): CrawlerIconKey {
  const key = (icon || '').toLowerCase()
  if (key in crawlerMeta) return key as CrawlerIconKey
  return 'scraper'
}

export function getCrawlerIconMeta(icon?: string): CrawlerIconMeta {
  return crawlerMeta[normalizeCrawlerIconKey(icon)]
}

export function getCrawlerLogoUrl(icon?: string): string {
  const key = normalizeCrawlerIconKey(icon)
  return panelStaticPath(`/crawlers/${key}.svg`)
}
