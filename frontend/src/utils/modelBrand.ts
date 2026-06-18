import { getModelVendor, inferVendorFromModelId, type ModelVendor } from '@/config/modelIcons'

export interface BrandGroup<T> {
  key: string
  label: string
  vendor: ModelVendor
  iconModelId: string
  models: T[]
  totalDownloads: number
  hasGated: boolean
  deployableCount: number
}

const vendorLabels: Record<ModelVendor, string> = {
  qwen: 'Qwen',
  meta: 'Meta Llama',
  mistral: 'Mistral',
  google: 'Google Gemma',
  microsoft: 'Microsoft Phi',
  deepseek: 'DeepSeek',
  huggingface: 'Hugging Face',
}

export function brandLabelForKey(key: string, sampleModelId = ''): string {
  if (key.startsWith('org:')) {
    const org = key.slice(4)
    return org.split(/[-_]/).map(w => w.charAt(0).toUpperCase() + w.slice(1)).join(' ')
  }
  if (key in vendorLabels) {
    return vendorLabels[key as ModelVendor]
  }
  const vendor = inferVendorFromModelId(sampleModelId)
  if (vendor !== 'huggingface') return vendorLabels[vendor]
  const org = sampleModelId.split('/')[0]
  return org || key
}

export function hubBrandKey(modelId: string, author?: string): string {
  const vendor = inferVendorFromModelId(modelId)
  if (vendor !== 'huggingface') return vendor
  const org = (author || modelId.split('/')[0] || 'other').toLowerCase().replace(/\s+/g, '-')
  return `org:${org}`
}

export function catalogBrandKey(catalogId: string, hfModelId: string, ollamaModel: string): string {
  const sample = hfModelId || ollamaModel || catalogId
  const vendor = getModelVendor(catalogId)
  if (vendor !== 'huggingface') return vendor
  return hubBrandKey(sample)
}

export function groupHubModels<T extends { id: string; author?: string; downloads?: number; gated?: boolean; deployable?: boolean }>(
  models: T[],
): BrandGroup<T>[] {
  const map = new Map<string, BrandGroup<T>>()
  for (const m of models) {
    const key = hubBrandKey(m.id, m.author)
    let g = map.get(key)
    if (!g) {
      g = {
        key,
        label: brandLabelForKey(key, m.id),
        vendor: inferVendorFromModelId(m.id),
        iconModelId: m.id,
        models: [],
        totalDownloads: 0,
        hasGated: false,
        deployableCount: 0,
      }
      map.set(key, g)
    }
    g.models.push(m)
    g.totalDownloads += m.downloads || 0
    if (m.gated) g.hasGated = true
    if (m.deployable) g.deployableCount++
  }
  const groups = [...map.values()]
  for (const g of groups) {
    g.models.sort((a, b) => (b.downloads || 0) - (a.downloads || 0))
    g.iconModelId = g.models[0]?.id || g.iconModelId
  }
  groups.sort((a, b) => b.totalDownloads - a.totalDownloads)
  return groups
}

export function groupCatalogEntries<T extends {
  id: string
  hf_model_id: string
  ollama_model: string
  params?: string
  featured?: boolean
  gated?: boolean
}>(
  entries: T[],
): BrandGroup<T>[] {
  const map = new Map<string, BrandGroup<T>>()
  for (const e of entries) {
    const sample = e.hf_model_id || e.ollama_model || e.id
    const key = catalogBrandKey(e.id, e.hf_model_id, e.ollama_model)
    let g = map.get(key)
    if (!g) {
      g = {
        key,
        label: brandLabelForKey(key, sample),
        vendor: getModelVendor(e.id),
        iconModelId: sample,
        models: [],
        totalDownloads: 0,
        hasGated: false,
        deployableCount: 0,
      }
      map.set(key, g)
    }
    g.models.push(e)
    if ((e as { gated?: boolean }).gated) g.hasGated = true
    g.deployableCount++
  }
  const groups = [...map.values()]
  for (const g of groups) {
    g.models.sort((a, b) => {
      const af = (a as { featured?: boolean }).featured ? 1 : 0
      const bf = (b as { featured?: boolean }).featured ? 1 : 0
      if (af !== bf) return bf - af
      return String((a as { params?: string }).params || '').localeCompare(String((b as { params?: string }).params || ''))
    })
    const first = g.models[0]
    g.iconModelId = (first as { hf_model_id?: string }).hf_model_id
      || (first as { ollama_model?: string }).ollama_model
      || first.id
  }
  groups.sort((a, b) => b.models.length - a.models.length)
  return groups
}

export function defaultPickId<T extends { id: string; deployable?: boolean }>(models: T[]): string {
  const deployable = models.find(m => m.deployable !== false)
  return (deployable || models[0])?.id || ''
}
