<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import CacheView from '@/views/CacheView.vue'
import FirewallView from '@/views/FirewallView.vue'
import NginxView from '@/views/NginxView.vue'
import WAFView from '@/views/WAFView.vue'
import SecurityView from '@/views/SecurityView.vue'
import EdgeWorkersView from '@/views/EdgeWorkersView.vue'
import KafkaAccelView from '@/views/KafkaAccelView.vue'
import CiliumView from '@/views/CiliumView.vue'

const TAB_KEYS = ['cache', 'firewall', 'nginx', 'waf', 'workers', 'kafka', 'cilium', 'security'] as const
type TabKey = (typeof TAB_KEYS)[number]

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const auth = useAuthStore()

function isAdminTab(tab: TabKey) {
  return tab === 'firewall' || tab === 'waf' || tab === 'workers' || tab === 'kafka' || tab === 'cilium' || tab === 'security'
}

const visibleTabs = computed(() => {
  const role = auth.user?.role
  if (!role || role === 'admin') return [...TAB_KEYS]
  if (role === 'user') return TAB_KEYS.filter(tab => !isAdminTab(tab))
  if (role === 'subuser') {
    try {
      const p = JSON.parse(auth.user?.permissions || '{}')
      if (p.websites) return TAB_KEYS.filter(tab => !isAdminTab(tab))
    } catch {
      /* ignore */
    }
  }
  return []
})

function normalizeTab(raw: unknown): TabKey {
  const tab = typeof raw === 'string' ? raw : ''
  if (visibleTabs.value.includes(tab as TabKey)) return tab as TabKey
  return visibleTabs.value[0] ?? 'cache'
}

const activeTab = ref<TabKey>(normalizeTab(route.query.tab))

watch(
  () => route.query.tab,
  (tab) => {
    activeTab.value = normalizeTab(tab)
  }
)

watch(activeTab, (tab) => {
  if (route.query.tab === tab) return
  router.replace({ query: { ...route.query, tab } })
})

watch(visibleTabs, (tabs) => {
  if (!tabs.includes(activeTab.value)) {
    activeTab.value = normalizeTab(tabs[0])
  }
})

const tabLabels: Record<TabKey, string> = {
  cache: 'protection.tabs.cache',
  firewall: 'protection.tabs.firewall',
  nginx: 'protection.tabs.nginx',
  waf: 'protection.tabs.waf',
  workers: 'protection.tabs.workers',
  kafka: 'protection.tabs.kafka',
  cilium: 'protection.tabs.cilium',
  security: 'protection.tabs.security',
}
</script>

<template>
  <div class="protection-center">
    <div class="page-header">
      <div>
        <h2>{{ t('protection.title') }}</h2>
        <p class="subtitle">{{ t('protection.subtitle') }}</p>
      </div>
    </div>

    <el-tabs v-model="activeTab" type="border-card" class="protection-tabs">
      <el-tab-pane
        v-for="tab in visibleTabs"
        :key="tab"
        :name="tab"
        :label="t(tabLabels[tab])"
      >
        <CacheView v-if="tab === 'cache'" embedded />
        <FirewallView v-else-if="tab === 'firewall'" embedded />
        <NginxView v-else-if="tab === 'nginx'" embedded />
        <WAFView v-else-if="tab === 'waf'" embedded />
        <EdgeWorkersView v-else-if="tab === 'workers'" embedded />
        <KafkaAccelView v-else-if="tab === 'kafka'" embedded />
        <CiliumView v-else-if="tab === 'cilium'" embedded />
        <SecurityView v-else-if="tab === 'security'" embedded />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<style scoped>
.protection-center .page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 16px;
  flex-wrap: wrap;
  gap: 12px;
}

.protection-center .page-header h2 {
  margin: 0 0 4px;
}

.protection-center .subtitle {
  margin: 0;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.protection-tabs :deep(.el-tabs__content) {
  padding: 16px 0 0;
}
</style>
