import { panelStaticPath } from '@/utils/panelBase'

export type ModelVendor = 'qwen' | 'meta' | 'mistral' | 'google' | 'microsoft' | 'deepseek' | 'huggingface'

export interface ModelIconMeta {
  bg: string
  label: string
  vendor: ModelVendor
}

const catalogVendor: Record<string, ModelVendor> = {
  'qwen2.5-0.5b': 'qwen',
  'qwen2.5-1.5b': 'qwen',
  'qwen2.5-7b': 'qwen',
  'qwen2.5-coder-1.5b': 'qwen',
  'llama3.2-3b': 'meta',
  'llama3.1-8b': 'meta',
  'codellama-7b': 'meta',
  'mistral-7b': 'mistral',
  'gemma2-2b': 'google',
  'phi3-mini': 'microsoft',
  'deepseek-r1-7b': 'deepseek',
  'deepseek-v3': 'deepseek',
  'smollm2-360m': 'huggingface',
}

const vendorMeta: Record<ModelVendor, ModelIconMeta> = {
  qwen: { bg: '#615EFF', label: 'Q', vendor: 'qwen' },
  meta: { bg: '#0668E1', label: 'Ll', vendor: 'meta' },
  mistral: { bg: '#FF7000', label: 'M', vendor: 'mistral' },
  google: { bg: '#4285F4', label: 'G', vendor: 'google' },
  microsoft: { bg: '#0078D4', label: 'Φ', vendor: 'microsoft' },
  deepseek: { bg: '#4D6BFE', label: 'DS', vendor: 'deepseek' },
  huggingface: { bg: '#FFD21E', label: 'HF', vendor: 'huggingface' },
}

export function getModelVendor(catalogId: string): ModelVendor {
  return catalogVendor[catalogId] ?? inferVendorFromModelId(catalogId)
}

export function inferVendorFromModelId(modelId: string): ModelVendor {
  const id = modelId.toLowerCase()
  if (id.includes('qwen')) return 'qwen'
  if (id.includes('llama') || id.includes('codellama') || id.startsWith('meta-')) return 'meta'
  if (id.includes('mistral')) return 'mistral'
  if (id.includes('gemma') || id.startsWith('google/')) return 'google'
  if (id.includes('phi-') || id.includes('phi3') || id.startsWith('microsoft/')) return 'microsoft'
  if (id.includes('deepseek')) return 'deepseek'
  if (id.includes('stable-diffusion') || id.includes('stabilityai') || id.includes('flux') || id.includes('black-forest')) return 'huggingface'
  if (id.includes('whisper') || id.startsWith('openai/')) return 'microsoft'
  return 'huggingface'
}

export function getModelLogoUrlForModel(modelOrCatalogId: string): string {
  const vendor = catalogVendor[modelOrCatalogId] ?? inferVendorFromModelId(modelOrCatalogId)
  return panelStaticPath(`/models/${vendor}.svg`)
}

export function getModelIconMetaForModel(modelOrCatalogId: string): ModelIconMeta {
  const vendor = catalogVendor[modelOrCatalogId] ?? inferVendorFromModelId(modelOrCatalogId)
  return vendorMeta[vendor]
}

export function getModelIconMeta(catalogId: string): ModelIconMeta {
  return vendorMeta[getModelVendor(catalogId)]
}

export function getModelLogoUrl(catalogId: string): string {
  const vendor = getModelVendor(catalogId)
  return panelStaticPath(`/models/${vendor}.svg`)
}

export function getModelIconDataUrl(catalogId: string): string {
  const meta = getModelIconMeta(catalogId)
  const label = meta.label.replace(/&/g, '&amp;').replace(/</g, '&lt;')
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64"><rect width="64" height="64" rx="12" fill="${meta.bg}"/><text x="32" y="38" text-anchor="middle" fill="#fff" font-family="system-ui,sans-serif" font-size="18" font-weight="700">${label}</text></svg>`
  return `data:image/svg+xml,${encodeURIComponent(svg)}`
}
