<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { Medal } from '@element-plus/icons-vue'
import api from '@/api'
import { apiContentLang } from '@/locales'

withDefaults(defineProps<{ embedded?: boolean; compact?: boolean }>(), { embedded: false, compact: false })

const { t, locale } = useI18n()
const router = useRouter()

const health = ref<any>(null)
const loading = ref(true)

const healthColor = computed(() => {
  const s = health.value?.score ?? 0
  if (s >= 80) return '#22c55e'
  if (s >= 60) return '#f59e0b'
  return '#ef4444'
})

const gradeTagType = computed(() => {
  const s = health.value?.score ?? 0
  if (s >= 80) return 'success'
  if (s >= 60) return 'warning'
  return 'danger'
})

async function load() {
  loading.value = true
  try {
    const res: any = await api.get('/dashboard/health', {
      params: { lang: apiContentLang(locale.value) },
    })
    health.value = res.data
  } catch {
    // silent on dashboard
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <div
    v-if="embedded"
    v-loading="loading"
    class="health-gauge-card gauge-card"
    :class="{ compact, horizontal: compact }"
    role="button"
    tabindex="0"
    @click="router.push('/toolbox')"
    @keydown.enter="router.push('/toolbox')"
  >
    <div class="health-ring">
      <el-progress
        v-if="health"
        type="circle"
        :percentage="health.score"
        :color="healthColor"
        :width="compact ? 64 : 88"
        :stroke-width="compact ? 5 : 6"
        class="health-progress"
      >
        <template #default>
          <span class="score">{{ health.score }}</span>
        </template>
      </el-progress>
      <div v-else class="health-ring-placeholder" />
    </div>
    <div class="health-copy">
      <div class="health-title-row">
        <div class="gauge-icon health">
          <el-icon :size="16"><Medal /></el-icon>
        </div>
        <span class="gauge-label">{{ t('toolboxPage.healthScore') }}</span>
        <el-tag v-if="health?.grade" :type="gradeTagType" size="small" effect="light" round class="grade-tag">
          {{ health.grade }}
        </el-tag>
      </div>
      <p v-if="health" class="gauge-sub">{{ health.summary }}</p>
      <span class="health-link">{{ t('toolboxPage.openToolbox') }} →</span>
    </div>
  </div>

  <el-card v-else v-loading="loading" shadow="hover" class="health-card">
    <template #header>
      <div class="card-head">
        <span>{{ t('toolboxPage.healthScore') }}</span>
        <el-button link type="primary" size="small" @click="router.push('/toolbox')">{{ t('toolboxPage.openToolbox') }}</el-button>
      </div>
    </template>
    <div v-if="health" class="health-inner">
      <el-progress type="dashboard" :percentage="health.score" :color="healthColor" :width="100">
        <template #default>
          <span class="score">{{ health.score }}</span>
          <span class="grade">{{ health.grade }}</span>
        </template>
      </el-progress>
      <p class="summary">{{ health.summary }}</p>
    </div>
  </el-card>
</template>

<style scoped>
.health-card {
  height: 100%;
}
.health-card :deep(.el-card__body) {
  padding: 12px 16px;
  height: calc(100% - 56px);
  display: flex;
  align-items: center;
}
.card-head { display: flex; justify-content: space-between; align-items: center; }
.health-inner {
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: 16px;
  width: 100%;
}
.score { display: block; font-size: 20px; font-weight: 700; }
.grade { font-size: 11px; color: var(--el-text-color-secondary); }
.summary {
  flex: 1;
  min-width: 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  text-align: left;
  margin: 0;
  line-height: 1.5;
}
@media (max-width: 768px) {
  .health-inner {
    flex-direction: column;
    text-align: center;
  }
  .summary { text-align: center; }
}
.health-gauge-card {
  position: relative;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 14px;
  padding: 14px 16px;
  background: linear-gradient(135deg, var(--el-fill-color-blank) 0%, var(--el-fill-color-light) 100%);
  transition: box-shadow 0.18s, border-color 0.18s, transform 0.18s;
  min-height: 100%;
  cursor: pointer;
}
.health-gauge-card:hover {
  border-color: var(--el-color-primary-light-5);
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.06);
  transform: translateY(-1px);
}
.health-gauge-card.horizontal {
  display: flex;
  align-items: center;
  gap: 14px;
  text-align: left;
}
.health-ring {
  flex-shrink: 0;
}
.health-ring-placeholder {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  background: var(--el-fill-color);
}
.health-copy {
  flex: 1;
  min-width: 0;
}
.health-title-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 4px;
}
.health-gauge-card .gauge-icon.health {
  width: 28px;
  height: 28px;
  border-radius: 8px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: rgba(34, 197, 94, 0.12);
  color: #22c55e;
}
.health-gauge-card .gauge-label {
  font-size: 14px;
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.grade-tag {
  margin-left: auto;
}
.health-gauge-card .gauge-sub {
  margin: 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.45;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.health-progress { margin: 0; }
.health-gauge-card .score {
  display: block;
  font-size: 18px;
  font-weight: 800;
  font-variant-numeric: tabular-nums;
  line-height: 1;
}
.health-link {
  display: inline-block;
  margin-top: 6px;
  font-size: 11px;
  color: var(--el-color-primary);
  opacity: 0.85;
}
</style>
