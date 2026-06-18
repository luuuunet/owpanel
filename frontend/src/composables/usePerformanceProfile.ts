import { computed, ref } from 'vue'
import api from '@/api'

export interface PerformanceProfile {
  enabled: boolean
  dashboard_live?: boolean
  collect_sec: number
  idle_collect_sec?: number
  monitor_lite_sec: number
  monitor_full_sec: number
  traffic_poll_sec: number
  traffic_map_sec: number
  cluster_sync_sec: number
  uptime_scan_sec: number
}

const profile = ref<PerformanceProfile | null>(null)
let loading: Promise<void> | null = null

export function usePerformanceProfile() {
  const powerSaveEnabled = computed(() => profile.value?.enabled === true)

  const liteIntervalSec = computed(() => profile.value?.monitor_lite_sec ?? 15)
  const fullIntervalSec = computed(() => profile.value?.monitor_full_sec ?? 60)
  const trafficMapSec = computed(() => profile.value?.traffic_map_sec ?? 90)

  async function load(force = false) {
    if (!force && profile.value) return
    if (loading) return loading
    loading = (async () => {
      const res: any = await api.get('/dashboard/performance')
      profile.value = res.data || null
    })().finally(() => {
      loading = null
    })
    return loading
  }

  async function setEnabled(enabled: boolean) {
    const res: any = await api.put('/dashboard/performance', { enabled })
    profile.value = res.data || null
  }

  return {
    profile,
    powerSaveEnabled,
    liteIntervalSec,
    fullIntervalSec,
    trafficMapSec,
    load,
    setEnabled,
  }
}
