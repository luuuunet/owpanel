<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import AppLogo from '@/components/AppLogo.vue'
import { menuGroups, canAccessMenuItem, menuGroupForPath, type MenuGroup, type MenuItem } from '@/config/menu'
import { useExtensionsStore } from '@/stores/extensions'
import { useNavRecent } from '@/composables/useNavRecent'
import * as Icons from '@element-plus/icons-vue'
import { ArrowDown, CircleClose, Search, SwitchButton } from '@element-plus/icons-vue'

interface SearchHit {
  path: string
  titleKey: string
  groupTitleKey: string
  icon: string
  externalUrl?: string
}

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const extStore = useExtensionsStore()
const { t } = useI18n()
const { recentList, recordVisit, clearRecent } = useNavRecent()

const emit = defineEmits<{ logout: [] }>()

const expanded = ref(false)
const searchQuery = ref('')
const searchFocused = ref(false)
const searchInputRef = ref<HTMLInputElement | null>(null)
const openGroupKey = ref('')
const selectedIndex = ref(0)
let collapseTimer: ReturnType<typeof setTimeout> | null = null

function menuItemLabel(titleKey: string) {
  if (titleKey.startsWith('@ext:')) return titleKey.slice(5)
  return t(titleKey)
}

function buildExtensionGroups(): MenuGroup[] {
  const map = new Map<string, MenuGroup>()
  for (const m of extStore.menuItems) {
    const item: MenuItem = {
      path: m.path,
      titleKey: `@ext:${m.title}`,
      icon: m.icon || 'Box',
      admin: m.admin,
      perm: m.perm,
      externalUrl: m.external_url,
    }
    if (!canAccessMenuItem(item, auth.user?.role, auth.user?.permissions)) continue
    const gKey = m.group || 'extensions'
    const titleKey = m.group_title ? `@ext:${m.group_title}` : 'menuGroup.extensions'
    if (!map.has(gKey)) {
      map.set(gKey, { titleKey, items: [] })
    }
    map.get(gKey)!.items.push(item)
  }
  return [...map.values()]
}

const visibleGroups = computed(() => {
  const base = menuGroups
    .map((group) => ({
      ...group,
      items: group.items.filter((item) =>
        canAccessMenuItem(item, auth.user?.role, auth.user?.permissions)
      ),
    }))
    .filter((group) => group.items.length > 0)
  const ext = buildExtensionGroups().filter((g) => g.items.length > 0)
  return [...base, ...ext]
})

const allMenuHits = computed<SearchHit[]>(() =>
  visibleGroups.value.flatMap((group) =>
    group.items.map((item) => ({
      path: item.path,
      titleKey: item.titleKey,
      groupTitleKey: group.titleKey,
      icon: item.icon,
      externalUrl: item.externalUrl,
    }))
  )
)

function groupKey(group: MenuGroup) {
  return group.titleKey
}

function findGroupByPath(path: string) {
  const group = menuGroupForPath(path)
  if (!group) return undefined
  return visibleGroups.value.find((g) => g.titleKey === group.titleKey)
}

function findHit(path: string) {
  return allMenuHits.value.find((h) => h.path === path)
}

function syncOpenGroupFromRoute() {
  const match = findGroupByPath(route.path)
  if (match) openGroupKey.value = groupKey(match)
  else if (visibleGroups.value.length) openGroupKey.value = groupKey(visibleGroups.value[0])
}

watch(
  () => route.path,
  (path) => {
    syncOpenGroupFromRoute()
    const hit = findHit(path)
    if (hit) recordVisit(hit)
  },
  { immediate: true }
)

function itemMatchesQuery(item: MenuItem, group: MenuGroup, q: string) {
  const label = menuItemLabel(item.titleKey).toLowerCase()
  const groupLabel = menuItemLabel(group.titleKey).toLowerCase()
  const path = item.path.toLowerCase()
  return label.includes(q) || groupLabel.includes(q) || path.includes(q.replace(/^\//, ''))
}

const filteredGroups = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return visibleGroups.value
  return visibleGroups.value
    .map((group) => ({
      ...group,
      items: group.items.filter((item) => itemMatchesQuery(item, group, q)),
    }))
    .filter((group) => group.items.length > 0)
})

const searchResults = computed<SearchHit[]>(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return []
  return allMenuHits.value.filter((hit) => {
    const group = visibleGroups.value.find((g) => g.titleKey === hit.groupTitleKey)
    const item = group?.items.find((i) => i.path === hit.path)
    return item && group && itemMatchesQuery(item, group, q)
  })
})

const recentHits = computed(() =>
  recentList.value.filter((r) => allMenuHits.value.some((h) => h.path === r.path))
)

const showSearchPanel = computed(
  () => expanded.value && (searchFocused.value || searchQuery.value.trim() !== '')
)

const panelItems = computed<SearchHit[]>(() => {
  if (searchQuery.value.trim()) return searchResults.value
  return recentHits.value
})

watch([searchQuery, panelItems], () => {
  selectedIndex.value = 0
})

function getIcon(name: string) {
  return (Icons as Record<string, unknown>)[name] || Icons.Monitor
}

function groupIcon(group: MenuGroup) {
  return getIcon(group.items[0]?.icon || 'Menu')
}

function isGroupOpen(group: MenuGroup) {
  return openGroupKey.value === groupKey(group)
}

function isGroupActive(group: MenuGroup) {
  return group.items.some((i) => route.path === i.path || route.path.startsWith(i.path + '/'))
}

function keepSidebarOpen() {
  if (collapseTimer) {
    clearTimeout(collapseTimer)
    collapseTimer = null
  }
  expanded.value = true
}

function toggleGroup(group: MenuGroup) {
  const key = groupKey(group)
  if (openGroupKey.value === key && isGroupActive(group)) return
  openGroupKey.value = openGroupKey.value === key ? '' : key
}

function openExternal(url: string) {
  window.open(url, '_blank', 'noopener,noreferrer')
}

function navigateItem(item: MenuItem, group: MenuGroup) {
  keepSidebarOpen()
  openGroupKey.value = groupKey(group)
  recordVisit({
    path: item.path,
    titleKey: item.titleKey,
    groupTitleKey: group.titleKey,
    icon: item.icon,
  })
  if (item.externalUrl) {
    openExternal(item.externalUrl)
    return
  }
  if (route.path !== item.path) {
    router.push(item.path)
  }
}

function onEnter() {
  if (collapseTimer) {
    clearTimeout(collapseTimer)
    collapseTimer = null
  }
  expanded.value = true
  syncOpenGroupFromRoute()
}

function onLeave() {
  if (searchFocused.value) return
  collapseTimer = setTimeout(() => {
    expanded.value = false
    searchQuery.value = ''
    searchFocused.value = false
  }, 180)
}

function focusSearch() {
  expanded.value = true
  setTimeout(() => {
    searchInputRef.value?.focus()
    searchFocused.value = true
  }, 220)
}

function closeSearch() {
  searchQuery.value = ''
  searchFocused.value = false
  searchInputRef.value?.blur()
}

function navigateTo(hit: SearchHit) {
  if (hit.externalUrl) {
    recordVisit(hit)
    openExternal(hit.externalUrl)
    closeSearch()
    return
  }
  recordVisit(hit)
  router.push(hit.path)
  closeSearch()
}

function onSearchFocus() {
  searchFocused.value = true
}

function onSearchBlur() {
  setTimeout(() => {
    searchFocused.value = false
  }, 150)
}

function onSearchKeydown(e: KeyboardEvent) {
  const items = panelItems.value
  if (e.key === 'Escape') {
    e.preventDefault()
    closeSearch()
    return
  }
  if (e.key === 'ArrowDown' && items.length) {
    e.preventDefault()
    selectedIndex.value = (selectedIndex.value + 1) % items.length
    return
  }
  if (e.key === 'ArrowUp' && items.length) {
    e.preventDefault()
    selectedIndex.value = (selectedIndex.value - 1 + items.length) % items.length
    return
  }
  if (e.key === 'Enter') {
    e.preventDefault()
    if (items.length) navigateTo(items[selectedIndex.value])
    return
  }
}

function onGlobalKeydown(e: KeyboardEvent) {
  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
    e.preventDefault()
    focusSearch()
  }
}

onMounted(() => {
  window.addEventListener('keydown', onGlobalKeydown)
  void extStore.fetchMenu()
})
onUnmounted(() => {
  window.removeEventListener('keydown', onGlobalKeydown)
  if (collapseTimer) clearTimeout(collapseTimer)
})
</script>

<template>
  <aside
    class="cf-sidebar"
    :class="{ expanded }"
    @mouseenter="onEnter"
    @mouseleave="onLeave"
  >
    <div class="cf-sidebar-inner">
      <div class="cf-brand">
        <div class="cf-logo">
          <AppLogo :size="36" />
        </div>
        <span class="cf-brand-text">{{ menuItemLabel('common.appName') }}</span>
      </div>

      <button type="button" class="cf-search-trigger" :title="menuItemLabel('nav.search')" @click="focusSearch">
        <el-icon><Search /></el-icon>
      </button>

      <div class="cf-search-wrap">
        <el-icon class="cf-search-icon"><Search /></el-icon>
        <input
          ref="searchInputRef"
          v-model="searchQuery"
          type="search"
          class="cf-search-input"
          :placeholder="menuItemLabel('nav.searchPlaceholder')"
          @focus="onSearchFocus"
          @blur="onSearchBlur"
          @keydown="onSearchKeydown"
        />
        <kbd v-if="!searchQuery" class="cf-search-kbd">Ctrl K</kbd>
        <button
          v-else
          type="button"
          class="cf-search-clear"
          :title="menuItemLabel('nav.clearRecent')"
          @mousedown.prevent
          @click="searchQuery = ''"
        >
          <el-icon><CircleClose /></el-icon>
        </button>
      </div>

      <!-- 搜索面板：最近访问 / 搜索结果 -->
      <div v-if="showSearchPanel" class="cf-search-panel">
        <div class="cf-search-panel-head">
          <span>{{ searchQuery.trim() ? menuItemLabel('nav.results') : menuItemLabel('nav.recent') }}</span>
          <button
            v-if="!searchQuery.trim() && recentHits.length"
            type="button"
            class="cf-search-panel-clear"
            @mousedown.prevent
            @click="clearRecent"
          >
            {{ menuItemLabel('nav.clearRecent') }}
          </button>
        </div>
        <div v-if="panelItems.length" class="cf-search-list">
          <button
            v-for="(hit, idx) in panelItems"
            :key="hit.path"
            type="button"
            class="cf-search-hit"
            :class="{ active: idx === selectedIndex, current: route.path === hit.path }"
            @mousedown.prevent
            @click="navigateTo(hit)"
            @mouseenter="selectedIndex = idx"
          >
            <el-icon class="cf-search-hit-icon"><component :is="getIcon(hit.icon)" /></el-icon>
            <span class="cf-search-hit-main">
              <span class="cf-search-hit-title">{{ menuItemLabel(hit.titleKey) }}</span>
              <span class="cf-search-hit-group">{{ menuItemLabel(hit.groupTitleKey) }}</span>
            </span>
          </button>
        </div>
        <div v-else class="cf-empty">{{ searchQuery.trim() ? menuItemLabel('nav.noResults') : menuItemLabel('nav.noRecent') }}</div>
        <p class="cf-search-hint">{{ menuItemLabel('nav.searchHint') }}</p>
      </div>

      <nav v-else class="cf-nav">
        <div v-for="group in filteredGroups" :key="group.titleKey" class="cf-accordion">
          <template v-if="!expanded && group.items.length === 1">
            <a
              href="#"
              class="cf-nav-icon"
              :class="{ active: route.path === group.items[0].path }"
              :title="menuItemLabel(group.items[0].titleKey)"
              @mousedown.prevent="keepSidebarOpen"
              @click.prevent="navigateItem(group.items[0], group)"
            >
              <el-icon><component :is="getIcon(group.items[0].icon)" /></el-icon>
            </a>
          </template>

          <template v-else-if="!expanded">
            <button
              type="button"
              class="cf-nav-icon"
              :class="{ active: isGroupActive(group) }"
              :title="menuItemLabel(group.titleKey)"
              @mouseenter="onEnter"
            >
              <el-icon><component :is="groupIcon(group)" /></el-icon>
            </button>
          </template>

          <template v-else>
            <button
              type="button"
              class="cf-accordion-head"
              :class="{ open: isGroupOpen(group), active: isGroupActive(group) }"
              @click="toggleGroup(group)"
            >
              <el-icon class="cf-accordion-icon"><component :is="groupIcon(group)" /></el-icon>
              <span class="cf-accordion-title">{{ menuItemLabel(group.titleKey) }}</span>
              <el-icon v-if="group.items.length > 1" class="cf-accordion-arrow" :class="{ open: isGroupOpen(group) }">
                <ArrowDown />
              </el-icon>
            </button>

            <div v-if="group.items.length === 1" class="cf-accordion-body cf-accordion-body--single">
              <a
                href="#"
                class="cf-nav-link"
                :class="{ active: route.path === group.items[0].path }"
                @mousedown.prevent="keepSidebarOpen"
                @click.prevent="navigateItem(group.items[0], group)"
              >
                {{ menuItemLabel(group.items[0].titleKey) }}
              </a>
            </div>

            <Transition v-else name="cf-acc">
              <div v-show="isGroupOpen(group)" class="cf-accordion-body">
                <a
                  v-for="item in group.items"
                  :key="item.path"
                  href="#"
                  class="cf-nav-link"
                  :class="{ active: route.path === item.path }"
                  @mousedown.prevent="keepSidebarOpen"
                  @click.prevent="navigateItem(item, group)"
                >
                  {{ menuItemLabel(item.titleKey) }}
                </a>
              </div>
            </Transition>
          </template>
        </div>
      </nav>

      <div class="cf-footer">
        <button type="button" class="cf-logout" :title="menuItemLabel('common.logout')" @click="emit('logout')">
          <el-icon><SwitchButton /></el-icon>
          <span class="cf-logout-label">{{ menuItemLabel('common.logout') }}</span>
        </button>
        <div v-if="expanded" class="cf-user">{{ auth.user?.username }}</div>
      </div>
    </div>
  </aside>
</template>

<style scoped>
.cf-sidebar {
  --cf-sidebar-collapsed: 56px;
  --cf-sidebar-expanded: 260px;
  width: var(--cf-sidebar-collapsed);
  height: 100%;
  height: 100dvh;
  flex-shrink: 0;
  background: var(--apple-glass, rgba(255, 255, 255, 0.82));
  backdrop-filter: var(--apple-glass-blur, saturate(180%) blur(20px));
  -webkit-backdrop-filter: var(--apple-glass-blur, saturate(180%) blur(20px));
  border-right: 1px solid var(--apple-glass-border, rgba(0, 0, 0, 0.06));
  transition: width 0.32s var(--apple-ease, cubic-bezier(0.25, 0.1, 0.25, 1));
  overflow: hidden;
  z-index: 100;
}

.cf-sidebar.expanded {
  width: var(--cf-sidebar-expanded);
  box-shadow: var(--apple-shadow-md, 0 4px 16px rgba(0, 0, 0, 0.06));
}

.cf-sidebar-inner {
  width: var(--cf-sidebar-collapsed);
  height: 100%;
  display: flex;
  flex-direction: column;
  padding: 8px 0;
  transition: width 0.22s cubic-bezier(0.4, 0, 0.2, 1);
}

.cf-sidebar.expanded .cf-sidebar-inner {
  width: var(--cf-sidebar-expanded);
}

.cf-brand {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 4px 10px 12px;
  flex-shrink: 0;
  border-bottom: 1px solid var(--cf-border, #e2e8f0);
  margin-bottom: 4px;
}

.cf-logo {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.cf-logo :deep(.app-logo) {
  width: 36px;
  height: 36px;
  border-radius: 9px;
  box-shadow: 0 2px 12px rgba(246, 130, 31, 0.22);
}

.cf-brand-text {
  font-size: 16px;
  font-weight: 600;
  color: var(--cf-navy, #1d2433);
  letter-spacing: -0.02em;
  white-space: nowrap;
  opacity: 0;
  max-width: 0;
  overflow: hidden;
  transition: opacity 0.15s, max-width 0.22s;
}

.cf-sidebar.expanded .cf-brand-text {
  opacity: 1;
  max-width: 180px;
}

.cf-search-trigger {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  margin: 4px auto 8px;
  padding: 0;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--cf-text-muted, #64748b);
  cursor: pointer;
  font-size: 18px;
  transition: background 0.15s, color 0.15s;
}

.cf-sidebar.expanded .cf-search-trigger {
  display: none;
}

.cf-search-trigger:hover {
  background: var(--cf-bg, #f4f5f7);
  color: var(--cf-text, #1f2937);
}

.cf-search-wrap {
  position: relative;
  margin: 0 12px 8px;
  flex-shrink: 0;
  display: none;
}

.cf-sidebar.expanded .cf-search-wrap {
  display: block;
}

.cf-search-icon {
  position: absolute;
  left: 10px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--cf-text-muted, #64748b);
  font-size: 14px;
  pointer-events: none;
}

.cf-search-input {
  width: 100%;
  height: 36px;
  padding: 0 52px 0 34px;
  border-radius: var(--apple-radius-sm, 10px);
  border: 1px solid var(--apple-glass-border, rgba(0, 0, 0, 0.06));
  background: rgba(0, 0, 0, 0.03);
  color: var(--cf-text, #1f2937);
  font-size: 13px;
  outline: none;
  transition: border-color 0.2s, box-shadow 0.2s, background 0.2s;
}

.cf-search-input::placeholder {
  color: var(--cf-text-muted, #64748b);
}

.cf-search-input:focus {
  border-color: var(--cf-orange, #f6821f);
  background: var(--cf-surface);
  box-shadow: 0 0 0 3px rgba(246, 130, 31, 0.12);
}

.cf-search-kbd {
  position: absolute;
  right: 8px;
  top: 50%;
  transform: translateY(-50%);
  font-size: 10px;
  color: var(--cf-text-muted, #64748b);
  border: 1px solid var(--cf-border, #e2e8f0);
  border-radius: 4px;
  padding: 1px 5px;
  font-family: inherit;
  pointer-events: none;
  background: var(--el-fill-color-light);
}

.cf-search-clear {
  position: absolute;
  right: 6px;
  top: 50%;
  transform: translateY(-50%);
  border: none;
  background: transparent;
  color: var(--cf-text-muted, #64748b);
  cursor: pointer;
  font-size: 14px;
  padding: 4px;
  display: flex;
  border-radius: 4px;
}

.cf-search-clear:hover {
  color: var(--cf-orange, #f6821f);
}

.cf-search-panel {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  padding: 0 8px 4px;
  min-height: 0;
}

.cf-search-panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 8px 6px;
  font-size: 11px;
  font-weight: 600;
  color: var(--cf-text-muted, #64748b);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.cf-search-panel-clear {
  border: none;
  background: transparent;
  color: var(--cf-orange, #f6821f);
  font-size: 11px;
  cursor: pointer;
  padding: 2px 4px;
  border-radius: 4px;
}

.cf-search-panel-clear:hover {
  background: var(--cf-orange-light, #fdebd7);
}

.cf-search-list {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.cf-search-hit {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 8px 10px;
  border: none;
  border-radius: 8px;
  background: transparent;
  cursor: pointer;
  text-align: left;
  transition: background 0.12s;
}

.cf-search-hit:hover,
.cf-search-hit.active {
  background: var(--cf-bg, #f4f5f7);
}

.cf-search-hit.current .cf-search-hit-title {
  color: var(--cf-orange, #f6821f);
}

.cf-search-hit-icon {
  font-size: 16px;
  color: var(--cf-text-muted, #64748b);
  flex-shrink: 0;
}

.cf-search-hit-main {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.cf-search-hit-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--cf-text, #1f2937);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.cf-search-hit-group {
  font-size: 11px;
  color: var(--cf-text-muted, #64748b);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.cf-search-hint {
  flex-shrink: 0;
  margin: 6px 8px 0;
  font-size: 10px;
  color: var(--cf-text-muted, #64748b);
  opacity: 0.85;
}

.cf-nav {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 0 8px;
}

.cf-nav-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  margin: 2px auto;
  border-radius: var(--apple-radius-sm, 10px);
  color: var(--cf-text-muted, #64748b);
  text-decoration: none;
  border: none;
  background: transparent;
  cursor: pointer;
  font-size: 18px;
  transition: background 0.2s var(--apple-ease, ease), color 0.2s, transform 0.2s;
}

.cf-nav-icon:hover {
  background: rgba(0, 0, 0, 0.04);
  color: var(--cf-text, #1f2937);
}

.cf-nav-icon.active {
  background: var(--cf-orange-light, #fdebd7);
  color: var(--cf-orange, #f6821f);
}

.cf-accordion {
  margin-bottom: 2px;
}

.cf-accordion-head {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 8px 10px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--cf-text-muted, #64748b);
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  text-align: left;
  transition: background 0.15s, color 0.15s;
}

.cf-accordion-head:hover,
.cf-accordion-head.open,
.cf-accordion-head.active {
  background: var(--cf-bg, #f4f5f7);
  color: var(--cf-text, #1f2937);
}

.cf-accordion-icon {
  font-size: 16px;
  flex-shrink: 0;
}

.cf-accordion-title {
  flex: 1;
  white-space: nowrap;
}

.cf-accordion-arrow {
  font-size: 12px;
  opacity: 0.5;
  transition: transform 0.2s;
}

.cf-accordion-arrow.open {
  transform: rotate(180deg);
}

.cf-accordion-body {
  overflow: hidden;
  padding: 2px 0 4px 8px;
}

.cf-accordion-body--single {
  padding-left: 36px;
}

.cf-nav-link {
  display: block;
  padding: 7px 12px;
  border-radius: var(--apple-radius-sm, 10px);
  color: var(--cf-text-muted, #64748b);
  text-decoration: none;
  font-size: 13px;
  font-weight: 500;
  letter-spacing: -0.01em;
  white-space: nowrap;
  transition: background 0.2s, color 0.2s;
}

.cf-nav-link:hover {
  background: rgba(0, 0, 0, 0.04);
  color: var(--cf-text, #1f2937);
}

.cf-nav-link.active {
  background: var(--cf-orange-light, #fdebd7);
  color: var(--cf-orange, #f6821f);
  font-weight: 600;
}

.cf-acc-enter-active,
.cf-acc-leave-active {
  transition: max-height 0.22s ease, opacity 0.18s ease;
}

.cf-acc-enter-from,
.cf-acc-leave-to {
  max-height: 0;
  opacity: 0;
}

.cf-acc-enter-to,
.cf-acc-leave-from {
  max-height: 320px;
  opacity: 1;
}

.cf-empty {
  padding: 12px;
  font-size: 13px;
  color: var(--cf-text-muted, #64748b);
}

.cf-footer {
  flex-shrink: 0;
  padding: 8px;
  border-top: 1px solid var(--cf-border, #e2e8f0);
}

.cf-logout {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 40px;
  height: 40px;
  margin: 0 auto;
  padding: 0;
  justify-content: center;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--cf-text-muted, #64748b);
  cursor: pointer;
  font-size: 18px;
  transition: background 0.15s, color 0.15s;
}

.cf-sidebar.expanded .cf-logout {
  width: 100%;
  height: auto;
  margin: 0;
  padding: 8px 10px;
  justify-content: flex-start;
}

.cf-logout:hover {
  background: var(--cf-bg, #f4f5f7);
  color: var(--cf-orange, #f6821f);
}

.cf-logout-label {
  font-size: 13px;
  font-weight: 500;
  white-space: nowrap;
  opacity: 0;
  max-width: 0;
  overflow: hidden;
  transition: opacity 0.15s, max-width 0.22s;
}

.cf-sidebar.expanded .cf-logout-label {
  opacity: 1;
  max-width: 120px;
}

.cf-user {
  font-size: 11px;
  color: var(--cf-text-muted, #64748b);
  padding: 4px 10px 0;
  white-space: nowrap;
}
</style>
