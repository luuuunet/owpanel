<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useLocaleStore } from '@/stores/locale'
import { LOCALE_OPTIONS, type LocaleCode } from '@/locales'

const localeStore = useLocaleStore()
const { t } = useI18n()

const options = LOCALE_OPTIONS

const currentShort = computed(() => {
  const opt = options.find((o) => o.value === localeStore.locale)
  return opt?.short ?? 'EN'
})

function setLocale(code: LocaleCode) {
  localeStore.setLocale(code)
}
</script>

<template>
  <div class="language-fab">
    <el-dropdown trigger="click" placement="top-end" @command="setLocale">
      <el-button class="fab-btn" circle type="primary" :title="t('common.language')">
        <span class="fab-label">{{ currentShort }}</span>
      </el-button>
      <template #dropdown>
        <el-dropdown-menu>
          <el-dropdown-item
            v-for="opt in options"
            :key="opt.value"
            :command="opt.value"
            :class="{ active: localeStore.locale === opt.value }"
          >
            {{ opt.label }}
          </el-dropdown-item>
        </el-dropdown-menu>
      </template>
    </el-dropdown>
  </div>
</template>

<style scoped>
.language-fab {
  position: fixed;
  right: 24px;
  bottom: 24px;
  z-index: 2000;
}

.fab-btn {
  width: 48px;
  height: 48px;
  font-size: 14px;
  font-weight: 700;
  box-shadow: 0 4px 16px rgba(246, 130, 31, 0.4);
}

.fab-label {
  line-height: 1;
}

:deep(.el-dropdown-menu__item.active) {
  color: var(--cf-orange);
  font-weight: 600;
}
</style>
