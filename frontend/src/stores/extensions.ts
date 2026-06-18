import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '@/api'

export interface ExtensionMenuItem {
  path: string
  title: string
  icon: string
  group: string
  group_title?: string
  admin?: boolean
  perm?: string
  embed_url?: string
  external_url?: string
  extension_id?: string
}

export interface ExtensionInfo {
  id: string
  name: string
  version: string
  description: string
  author: string
  enabled: boolean
  dir: string
  hooks: string[]
  catalog_count: number
}

export const useExtensionsStore = defineStore('extensions', () => {
  const menuItems = ref<ExtensionMenuItem[]>([])
  const loaded = ref(false)

  async function fetchMenu() {
    try {
      const res: any = await api.get('/extensions/menu')
      menuItems.value = res.data || []
    } catch {
      menuItems.value = []
    } finally {
      loaded.value = true
    }
  }

  return { menuItems, loaded, fetchMenu }
})
