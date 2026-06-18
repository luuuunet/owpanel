<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Moon, Sunny, Monitor } from '@element-plus/icons-vue'
import { DARK_VARIANT_OPTIONS, THEME_MODE_OPTIONS } from '@/config/themes'
import { useThemeStore } from '@/stores/theme'

const themeStore = useThemeStore()
const { t } = useI18n()

const modeIcon = computed(() => {
  if (themeStore.mode === 'dark') return Moon
  if (themeStore.mode === 'light') return Sunny
  return Monitor
})

const modeLabel = computed(() => {
  const opt = THEME_MODE_OPTIONS.find((o) => o.value === themeStore.mode)
  return opt ? t(opt.labelKey) : ''
})

function setMode(mode: string) {
  themeStore.setMode(mode as 'system' | 'light' | 'dark')
}

function setVariant(variant: string) {
  themeStore.setDarkVariant(variant as 'navy' | 'charcoal' | 'midnight' | 'slate')
}
</script>

<template>
  <div class="theme-fab">
    <el-dropdown trigger="click" placement="top-end" @command="setMode">
      <el-button class="fab-btn" circle :title="t('theme.title')">
        <el-icon><component :is="modeIcon" /></el-icon>
      </el-button>
      <template #dropdown>
        <el-dropdown-menu>
          <el-dropdown-item
            v-for="opt in THEME_MODE_OPTIONS"
            :key="opt.value"
            :command="opt.value"
            :class="{ active: themeStore.mode === opt.value }"
          >
            {{ t(opt.labelKey) }}
          </el-dropdown-item>
        </el-dropdown-menu>
      </template>
    </el-dropdown>

    <el-popover v-if="themeStore.resolvedTheme === 'dark'" placement="top-end" :width="220" trigger="click">
      <template #reference>
        <el-button class="fab-btn fab-btn-variant" circle :title="t('theme.darkVariant')">
          <span class="variant-dot" :style="{ background: DARK_VARIANT_OPTIONS.find(v => v.value === themeStore.darkVariant)?.swatch }" />
        </el-button>
      </template>
      <div class="variant-panel">
        <p class="variant-title">{{ t('theme.darkVariant') }}</p>
        <div class="variant-grid">
          <button
            v-for="opt in DARK_VARIANT_OPTIONS"
            :key="opt.value"
            type="button"
            class="variant-chip"
            :class="{ active: themeStore.darkVariant === opt.value }"
            @click="setVariant(opt.value)"
          >
            <span class="variant-swatch" :style="{ background: opt.swatch }" />
            <span>{{ t(opt.labelKey) }}</span>
          </button>
        </div>
        <p class="variant-hint">{{ modeLabel }}</p>
      </div>
    </el-popover>
  </div>
</template>

<style scoped>
.theme-fab {
  position: fixed;
  right: 24px;
  bottom: 80px;
  z-index: 2000;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.fab-btn {
  width: 48px;
  height: 48px;
  font-size: 18px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.18);
  background: var(--cf-surface);
  border: 1px solid var(--cf-border);
  color: var(--cf-text);
}

.fab-btn-variant {
  font-size: 0;
}

.variant-dot {
  display: block;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  border: 2px solid rgba(255, 255, 255, 0.35);
}

.variant-panel {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.variant-title {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
  color: var(--cf-text);
}

.variant-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}

.variant-chip {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border: 1px solid var(--cf-border);
  border-radius: 10px;
  background: var(--cf-bg);
  color: var(--cf-text);
  cursor: pointer;
  font-size: 12px;
  transition: border-color 0.15s, background 0.15s;
}

.variant-chip:hover,
.variant-chip.active {
  border-color: var(--cf-orange);
  background: rgba(246, 130, 31, 0.08);
}

.variant-swatch {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  flex-shrink: 0;
  border: 1px solid rgba(255, 255, 255, 0.15);
}

.variant-hint {
  margin: 0;
  font-size: 11px;
  color: var(--cf-text-muted);
}

:deep(.el-dropdown-menu__item.active) {
  color: var(--cf-orange);
  font-weight: 600;
}
</style>
