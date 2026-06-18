import { panelStaticPath } from '@/utils/panelBase'

export interface SoftwareIconMeta {
  bg: string
  label: string
}

/** Brand colors for common apps; others use deterministic hash color. */
export const softwareIconMeta: Record<string, SoftwareIconMeta> = {
  nginx: { bg: '#009639', label: 'N' },
  openresty: { bg: '#009639', label: 'OR' },
  apache: { bg: '#D22128', label: 'A' },
  openlitespeed: { bg: '#0066CC', label: 'LS' },
  mysql: { bg: '#00758F', label: 'My' },
  mariadb: { bg: '#C0765A', label: 'Ma' },
  postgresql: { bg: '#336791', label: 'Pg' },
  redis: { bg: '#DC382D', label: 'R' },
  mongodb: { bg: '#47A248', label: 'Mg' },
  php83: { bg: '#777BB4', label: '8.3' },
  php82: { bg: '#777BB4', label: '8.2' },
  php81: { bg: '#777BB4', label: '8.1' },
  php74: { bg: '#777BB4', label: '7.4' },
  php: { bg: '#777BB4', label: 'PHP' },
  nodejs: { bg: '#339933', label: 'JS' },
  nodejs20: { bg: '#339933', label: '20' },
  nodejs18: { bg: '#339933', label: '18' },
  python: { bg: '#3776AB', label: 'Py' },
  dotnet10: { bg: '#512BD4', label: '10' },
  dotnet8: { bg: '#512BD4', label: '8' },
  dotnet: { bg: '#512BD4', label: '.NET' },
  java21: { bg: '#5382A1', label: '21' },
  java17: { bg: '#5382A1', label: '17' },
  java11: { bg: '#5382A1', label: '11' },
  java8: { bg: '#5382A1', label: '8' },
  java: { bg: '#5382A1', label: 'Jv' },
  pureftpd: { bg: '#F5921E', label: 'FTP' },
  postfix: { bg: '#004499', label: 'Px' },
  dovecot: { bg: '#0066CC', label: 'Dv' },
  phpmyadmin: { bg: '#F5921E', label: 'PMA' },
  memcached: { bg: '#5A5A5A', label: 'MC' },
  docker: { bg: '#2496ED', label: 'D' },
  fail2ban: { bg: '#E74C3C', label: 'F2' },
  supervisor: { bg: '#4A90D9', label: 'SV' },
  ollama: { bg: '#1a1a2e', label: 'Ol' },
  'open-webui': { bg: '#2563eb', label: 'UI' },
  localai: { bg: '#10b981', label: 'LA' },
  dify: { bg: '#6366f1', label: 'Df' },
  jupyter: { bg: '#F37726', label: 'Ju' },
  'jupyter-notebook': { bg: '#F37726', label: 'Ju' },
  vllm: { bg: '#7c3aed', label: 'vL' },
  comfyui: { bg: '#0ea5e9', label: 'CU' },
  'sd-webui': { bg: '#a855f7', label: 'SD' },
  anythingllm: { bg: '#334155', label: 'AL' },
  fastgpt: { bg: '#059669', label: 'FG' },
  whisper: { bg: '#412991', label: 'Wh' },
  'huggingface-ai': { bg: '#FFD21E', label: 'HF' },
  chatchat: { bg: '#0284c7', label: 'CC' },
  pm2: { bg: '#2B037D', label: 'PM' },
  composer: { bg: '#885630', label: 'Cp' },
  certbot: { bg: '#2E8540', label: 'SSL' },
  tomcat9: { bg: '#F8DC75', label: 'T9' },
  tomcat10: { bg: '#F8DC75', label: 'T10' },
  tomcat: { bg: '#F8DC75', label: 'TC' },
  wordpress: { bg: '#21759B', label: 'WP' },
  gitea: { bg: '#609926', label: 'Gt' },
  gitlab: { bg: '#FC6D26', label: 'GL' },
  grafana: { bg: '#F46800', label: 'Gr' },
  prometheus: { bg: '#E6522C', label: 'Pr' },
  elasticsearch: { bg: '#005571', label: 'ES' },
  minio: { bg: '#C72C48', label: 'Mi' },
  portainer: { bg: '#13BEF9', label: 'Pt' },
  jenkins: { bg: '#D33833', label: 'Jk' },
  nextcloud: { bg: '#0082C9', label: 'NC' },
  jellyfin: { bg: '#AA5CC3', label: 'Jf' },
  'home-assistant': { bg: '#18BCF2', label: 'HA' },
  keycloak: { bg: '#4D4D4D', label: 'KC' },
  kafka: { bg: '#231F20', label: 'Kf' },
  k3s: { bg: '#FFC107', label: 'K3' },
  cilium: { bg: '#6376DD', label: 'Ci' },
  rabbitmq: { bg: '#FF6600', label: 'Rb' },
  traefik: { bg: '#24A1C1', label: 'Tr' },
  caddy: { bg: '#1F88C0', label: 'Cd' },
  n8n: { bg: '#EA4B71', label: 'n8' },
  vaultwarden: { bg: '#175DDC', label: 'VW' },
  mattermost: { bg: '#0058CC', label: 'MM' },
  clickhouse: { bg: '#FFCC01', label: 'CH' },
  influxdb: { bg: '#22ADF6', label: 'If' },
  ghost: { bg: '#15171A', label: 'Gh' },
  discourse: { bg: '#000000', label: 'Dc' },
  maxkb: { bg: '#3370FF', label: 'MK' },
}

function hashIconColor(key: string): string {
  let h = 0
  for (let i = 0; i < key.length; i++) {
    h = (Math.imul(31, h) + key.charCodeAt(i)) >>> 0
  }
  return `hsl(${h % 360}, 58%, 46%)`
}

function iconLabelFromKey(key: string): string {
  if (/^php\d/.test(key)) return key.replace('php', '')
  if (key.startsWith('nodejs')) return key.replace('nodejs', '') || 'JS'
  if (key.startsWith('java')) return key.replace('java', '') || 'Jv'
  if (key.startsWith('dotnet')) return 'DN'
  if (key.startsWith('tomcat')) return key.includes('10') ? 'T10' : 'T9'
  const parts = key.split(/[-_]/)
  if (parts.length >= 2) {
    return (parts[0].slice(0, 1) + parts[1].slice(0, 1)).toUpperCase()
  }
  return key.slice(0, 2).toUpperCase()
}

export function getSoftwareIconMeta(key: string): SoftwareIconMeta {
  return softwareIconMeta[key] ?? { bg: hashIconColor(key), label: iconLabelFromKey(key) }
}

/** Inline SVG badge — used when static file is missing (no broken image). */
export function getSoftwareIconDataUrl(key: string): string {
  const meta = getSoftwareIconMeta(key)
  const label = meta.label.replace(/&/g, '&amp;').replace(/</g, '&lt;')
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64"><rect width="64" height="64" rx="12" fill="${meta.bg}"/><text x="32" y="38" text-anchor="middle" fill="#fff" font-family="system-ui,sans-serif" font-size="18" font-weight="700">${label}</text></svg>`
  return `data:image/svg+xml,${encodeURIComponent(svg)}`
}

/** Logo path under /software-icons/{key}.svg (served from backend/web/software). */
export function getSoftwareLogoUrl(key: string): string {
  return panelStaticPath(`/software-icons/${key}.svg`)
}

/** Fallback logo when primary SVG is missing */
export function getSoftwareLogoFallback(key: string): string | null {
  if (key.startsWith('nodejs') && key !== 'nodejs') {
    return panelStaticPath('/software-icons/nodejs.svg')
  }
  if (key.startsWith('java') && key !== 'java') {
    return panelStaticPath('/software-icons/java.svg')
  }
  if (key.startsWith('tomcat')) {
    return panelStaticPath('/software-icons/java.svg')
  }
  if (key.startsWith('dotnet')) {
    return panelStaticPath('/software-icons/python.svg')
  }
  return getSoftwareIconDataUrl(key)
}
