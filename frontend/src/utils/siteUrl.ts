export function siteVisitUrl(domain: string, opts?: { port?: number; ssl?: boolean }) {
  const host = String(domain || '').trim()
  if (!host) return '#'
  const port = opts?.port || 80
  const ssl = !!opts?.ssl
  const scheme = ssl ? 'https' : 'http'
  const defaultPort = ssl ? 443 : 80
  if (port === defaultPort) return `${scheme}://${host}`
  return `${scheme}://${host}:${port}`
}

export function openSiteVisit(domain: string, opts?: { port?: number; ssl?: boolean }) {
  const url = siteVisitUrl(domain, opts)
  if (url === '#') return
  window.open(url, '_blank', 'noopener,noreferrer')
}
