<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import SystemMonitorPanel from '@/components/SystemMonitorPanel.vue'
import TrafficMap from '@/components/TrafficMap.vue'
import HealthScoreCard from '@/components/HealthScoreCard.vue'
import AlertCenter from '@/components/AlertCenter.vue'

const { t } = useI18n()
const monitorStats = ref<any>(null)

function onMonitorStats(stats: any) {
  monitorStats.value = stats
}
</script>

<template>
  <div class="dashboard">
    <div class="dash-grid">
      <SystemMonitorPanel layout="dashboard" hide-health @stats="onMonitorStats">
        <template #overview-top>
          <div class="overview-top-row">
            <HealthScoreCard embedded compact class="overview-health" />
            <AlertCenter :stats="monitorStats" compact :poll-sec="15" class="overview-alerts" />
          </div>
        </template>
      </SystemMonitorPanel>

      <el-card shadow="hover" class="dash-traffic">
        <template #header>{{ t('traffic.title') }}</template>
        <TrafficMap compact dashboard />
      </el-card>
    </div>
  </div>
</template>

<style scoped>
.dashboard {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.dash-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(360px, 0.95fr);
  gap: 16px;
  grid-template-areas:
    "overview traffic"
    "trend traffic"
    "apps apps";
  align-items: stretch;
}

.overview-top-row {
  display: flex;
  align-items: stretch;
  gap: 12px;
  min-height: 88px;
}

.overview-top-row :deep(.overview-health) {
  flex: 1 1 260px;
  min-width: 0;
}

.overview-top-row :deep(.overview-alerts) {
  flex: 1 1 200px;
  min-width: 0;
}

.overview-top-row :deep(.health-gauge-card) {
  border: none;
  border-radius: 10px;
  background: var(--el-fill-color-lighter);
  min-height: 100%;
  padding: 12px 14px;
}

.overview-top-row :deep(.health-gauge-card:hover) {
  transform: none;
  box-shadow: none;
  border-color: transparent;
  background: var(--el-fill-color);
}

.dash-traffic {
  grid-area: traffic;
  min-width: 0;
  align-self: stretch;
  display: flex;
  flex-direction: column;
}

.dash-traffic :deep(.el-card__body) {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  padding: 0;
}

.dashboard :deep(.system-monitor.dashboard) {
  display: contents;
}

.dashboard :deep(.dash-unified-card) {
  grid-area: overview;
  min-width: 0;
}

.dashboard :deep(.dash-unified-card .el-card__body) {
  padding: 12px 14px 14px;
}

.dashboard :deep(.dash-trend-card) {
  grid-area: trend;
}

.dashboard :deep(.dash-apps-card) {
  grid-area: apps;
}

.dashboard :deep(.dash-trend-card .el-card__body),
.dashboard :deep(.dash-apps-card .el-card__body) {
  padding: 12px 14px 14px;
}

@media (max-width: 1100px) {
  .overview-top-row {
    flex-direction: column;
    min-height: 0;
  }

  .dash-grid {
    grid-template-columns: 1fr;
    grid-template-areas:
      "overview"
      "traffic"
      "trend"
      "apps";
  }
}
</style>
