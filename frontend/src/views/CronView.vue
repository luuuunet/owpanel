<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import api, { resolveApiError } from '@/api'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Plus,
  RefreshRight,
  VideoPlay,
  Document,
  Delete,
  Edit,
  Timer,
  Search,
} from '@element-plus/icons-vue'

interface CronJob {
  id: number
  name: string
  schedule: string
  command: string
  enabled: boolean
  last_run_at?: string
  last_status?: string
  last_output?: string
  sync_status?: string
  sync_message?: string
  next_run_at?: string
  executor?: string
}

interface TaskTemplate {
  id: string
  name: string
  description: string
  schedule: string
  command: string
  icon: string
  color: string
}

const { t } = useI18n()

const jobs = ref<CronJob[]>([])
const taskTemplates = ref<TaskTemplate[]>([])
const schedulerMode = ref('')
const templatesExpanded = ref(true)
const loading = ref(false)
const search = ref('')
const viewMode = ref<'board' | 'table'>('board')

const dialogVisible = ref(false)
const editingId = ref<number | null>(null)
const saving = ref(false)
const scheduleMode = ref<'preset' | 'daily' | 'hourly' | 'weekly' | 'custom'>('preset')
const form = ref({ name: '', schedule: '0 2 * * *', command: '', enabled: true })
const dailyHour = ref(2)
const dailyMinute = ref(0)
const weeklyDay = ref(0)
const weeklyHour = ref(3)
const weeklyMinute = ref(0)

const drawerVisible = ref(false)
const activeJob = ref<CronJob | null>(null)

const logVisible = ref(false)
const logContent = ref('')
const logTitle = ref('')
const logLoading = ref(false)
const logTargetId = ref<number | null>(null)

const presets = [
  { label: 'cron.presetHourly', value: '0 * * * *', desc: 'cron.presetHourlyDesc' },
  { label: 'cron.presetDaily', value: '0 2 * * *', desc: 'cron.presetDailyDesc' },
  { label: 'cron.presetWeekly', value: '0 3 * * 0', desc: 'cron.presetWeeklyDesc' },
  { label: 'cron.presetMonthly', value: '0 4 1 * *', desc: 'cron.presetMonthlyDesc' },
]

async function loadTemplates() {
  try {
    const res: any = await api.get('/cron/templates')
    taskTemplates.value = res.data || []
  } catch {
    taskTemplates.value = []
  }
}

async function loadStatus() {
  try {
    const res: any = await api.get('/cron/status')
    schedulerMode.value = res.data?.mode || ''
  } catch {
    schedulerMode.value = ''
  }
}

const stats = computed(() => {
  const list = jobs.value
  return {
    total: list.length,
    enabled: list.filter((j) => j.enabled).length,
    disabled: list.filter((j) => !j.enabled).length,
    failed: list.filter((j) => j.last_status === 'failed').length,
  }
})

const filteredJobs = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return jobs.value
  return jobs.value.filter(
    (j) =>
      j.name.toLowerCase().includes(q)
      || j.schedule.includes(q)
      || j.command.toLowerCase().includes(q)
      || (j.last_status || '').includes(q),
  )
})

const enabledJobs = computed(() => filteredJobs.value.filter((j) => j.enabled))
const disabledJobs = computed(() => filteredJobs.value.filter((j) => !j.enabled))

watch(scheduleMode, (mode) => {
  if (mode === 'hourly') form.value.schedule = '0 * * * *'
  else if (mode === 'daily') form.value.schedule = `${dailyMinute.value} ${dailyHour.value} * * *`
  else if (mode === 'weekly') form.value.schedule = `${weeklyMinute.value} ${weeklyHour.value} * * ${weeklyDay.value}`
})

watch([dailyHour, dailyMinute], () => {
  if (scheduleMode.value === 'daily') form.value.schedule = `${dailyMinute.value} ${dailyHour.value} * * *`
})

watch([weeklyDay, weeklyHour, weeklyMinute], () => {
  if (scheduleMode.value === 'weekly') form.value.schedule = `${weeklyMinute.value} ${weeklyHour.value} * * ${weeklyDay.value}`
})

function describeSchedule(schedule: string) {
  const s = schedule.trim()
  const hit = presets.find((p) => p.value === s)
  if (hit) return t(hit.desc)
  if (s === `${dailyMinute.value} ${dailyHour.value} * * *` || /^\d+ \d+ \* \* \*$/.test(s)) {
    const parts = s.split(/\s+/)
    return t('cron.descDaily', { h: parts[1], m: parts[0] })
  }
  return s
}

function syncTagType(status?: string) {
  if (status === 'synced') return 'success'
  if (status === 'error') return 'danger'
  return 'info'
}

function statusTagType(status?: string) {
  if (status === 'success') return 'success'
  if (status === 'failed') return 'danger'
  if (status === 'running') return 'warning'
  return 'info'
}

function formatTime(iso?: string) {
  if (!iso) return '—'
  return new Date(iso).toLocaleString()
}

function truncate(text: string, max = 48) {
  if (!text || text.length <= max) return text
  return `${text.slice(0, max)}…`
}

function applyTemplate(tpl: TaskTemplate) {
  if (!form.value.name.trim()) form.value.name = tpl.name
  form.value.schedule = tpl.schedule
  form.value.command = tpl.command
  scheduleMode.value = 'custom'
}

async function load() {
  loading.value = true
  try {
    const res: any = await api.get('/cron')
    jobs.value = res.data || []
    await loadStatus()
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
  } finally {
    loading.value = false
  }
}

function resetForm() {
  form.value = { name: '', schedule: '0 2 * * *', command: '', enabled: true }
  scheduleMode.value = 'preset'
  dailyHour.value = 2
  dailyMinute.value = 0
  weeklyDay.value = 0
  weeklyHour.value = 3
  weeklyMinute.value = 0
  editingId.value = null
}

function openCreate(template?: TaskTemplate) {
  resetForm()
  if (template) {
    form.value.name = template.name
    form.value.schedule = template.schedule
    form.value.command = template.command
    scheduleMode.value = 'custom'
  }
  dialogVisible.value = true
}

function openEdit(job: CronJob) {
  editingId.value = job.id
  form.value = {
    name: job.name,
    schedule: job.schedule,
    command: job.command,
    enabled: job.enabled,
  }
  scheduleMode.value = presets.some((p) => p.value === job.schedule) ? 'preset' : 'custom'
  dialogVisible.value = true
}

function openDrawer(job: CronJob) {
  activeJob.value = job
  drawerVisible.value = true
}

async function handleSave() {
  if (!form.value.name.trim()) {
    ElMessage.warning(t('cron.nameRequired'))
    return
  }
  if (!form.value.schedule.trim()) {
    ElMessage.warning(t('cron.scheduleRequired'))
    return
  }
  if (!form.value.command.trim()) {
    ElMessage.warning(t('cron.commandRequired'))
    return
  }
  saving.value = true
  try {
    const payload = {
      name: form.value.name.trim(),
      schedule: form.value.schedule.trim(),
      command: form.value.command.trim(),
      enabled: form.value.enabled,
    }
    if (editingId.value) {
      await api.put(`/cron/${editingId.value}`, payload)
      ElMessage.success(t('common.updated'))
    } else {
      await api.post('/cron', payload)
      ElMessage.success(t('cron.created'))
    }
    dialogVisible.value = false
    resetForm()
    await load()
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
  } finally {
    saving.value = false
  }
}

async function handleDelete(job: CronJob) {
  try {
    await ElMessageBox.confirm(t('cron.deleteConfirm'), t('common.warning'), { type: 'warning' })
    await api.delete(`/cron/${job.id}`)
    ElMessage.success(t('common.deleted'))
    if (activeJob.value?.id === job.id) drawerVisible.value = false
    await load()
  } catch (e: any) {
    if (e !== 'cancel') ElMessage.error(resolveApiError(e, t('common.failed')))
  }
}

async function handleToggle(job: CronJob, enabled: boolean) {
  try {
    await api.patch(`/cron/${job.id}/toggle`, { enabled })
    ElMessage.success(t('common.updated'))
    await load()
    if (activeJob.value?.id === job.id) {
      activeJob.value = jobs.value.find((j) => j.id === job.id) || null
    }
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
    await load()
  }
}

async function handleRun(job: CronJob) {
  try {
    await api.post(`/cron/${job.id}/run`)
    ElMessage.success(t('cron.runStarted'))
    setTimeout(load, 1500)
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
  }
}

async function handleLogs(job: CronJob) {
  logTargetId.value = job.id
  logTitle.value = job.name
  logVisible.value = true
  await fetchLogs()
}

async function fetchLogs() {
  if (!logTargetId.value) return
  logLoading.value = true
  try {
    const res: any = await api.get(`/cron/${logTargetId.value}/logs`)
    logContent.value = res.data?.log || t('cron.noLog')
  } catch (e: any) {
    logContent.value = resolveApiError(e, t('common.failed'))
  } finally {
    logLoading.value = false
  }
}

async function handleReload() {
  try {
    await api.post('/cron/reload')
    ElMessage.success(t('cron.reloaded'))
    await load()
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
  }
}

function applyPreset(value: string) {
  form.value.schedule = value
  scheduleMode.value = 'preset'
}

onMounted(() => {
  loadTemplates()
  load()
})
</script>

<template>
  <div class="cron-page">
    <div class="page-header">
      <div>
        <h2>{{ t('cron.title') }}</h2>
        <p class="page-sub">{{ t('cron.subtitle') }}</p>
      </div>
      <div class="header-actions">
        <el-button :icon="RefreshRight" :loading="loading" @click="handleReload">{{ t('cron.reload') }}</el-button>
        <el-button type="primary" :icon="Plus" @click="openCreate()">{{ t('cron.add') }}</el-button>
      </div>
    </div>

    <el-alert type="info" :closable="false" show-icon class="hint">
      {{ t('cron.hint') }}
      <span v-if="schedulerMode" class="scheduler-badge">
        {{ schedulerMode === 'system_crontab' ? t('cron.modeSystem') : t('cron.modeInternal') }}
      </span>
    </el-alert>

    <div v-if="taskTemplates.length" class="template-section">
      <div class="section-head clickable" @click="templatesExpanded = !templatesExpanded">
        <span>{{ t('cron.quickTemplates') }}</span>
        <el-tag size="small" type="info" effect="plain">{{ taskTemplates.length }}</el-tag>
        <span class="expand-hint">{{ templatesExpanded ? '▾' : '▸' }}</span>
      </div>
      <div v-show="templatesExpanded" class="template-grid">
        <button
          v-for="tpl in taskTemplates"
          :key="tpl.id"
          type="button"
          class="template-card"
          @click="openCreate(tpl)"
        >
          <span class="template-icon" :style="{ background: tpl.color }">{{ tpl.icon }}</span>
          <span class="template-name">{{ tpl.name }}</span>
          <span class="template-desc">{{ tpl.description }}</span>
        </button>
      </div>
    </div>

    <div v-if="stats.total > 0" class="stats-row">
      <div class="stat-card">
        <span class="stat-value">{{ stats.total }}</span>
        <span class="stat-label">{{ t('cron.statTotal') }}</span>
      </div>
      <div class="stat-card enabled">
        <span class="stat-value">{{ stats.enabled }}</span>
        <span class="stat-label">{{ t('cron.statEnabled') }}</span>
      </div>
      <div class="stat-card disabled">
        <span class="stat-value">{{ stats.disabled }}</span>
        <span class="stat-label">{{ t('cron.statDisabled') }}</span>
      </div>
      <div class="stat-card failed">
        <span class="stat-value">{{ stats.failed }}</span>
        <span class="stat-label">{{ t('cron.statFailed') }}</span>
      </div>
    </div>

    <div v-if="jobs.length" class="toolbar">
      <el-input v-model="search" :prefix-icon="Search" clearable :placeholder="t('cron.searchPlaceholder')" class="search-input" />
      <el-radio-group v-model="viewMode" class="view-toggle">
        <el-radio-button value="board">{{ t('cron.viewBoard') }}</el-radio-button>
        <el-radio-button value="table">{{ t('cron.viewTable') }}</el-radio-button>
      </el-radio-group>
    </div>

    <!-- Empty state -->
    <el-empty v-if="!jobs.length && !loading" :description="t('cron.empty')" class="empty-only">
      <el-button type="primary" @click="openCreate()">{{ t('cron.add') }}</el-button>
    </el-empty>

    <!-- Board view -->
    <div v-else-if="viewMode === 'board'" v-loading="loading" class="job-board">
      <section v-if="enabledJobs.length" class="job-section">
        <div class="section-head">
          <span>{{ t('cron.sectionEnabled') }}</span>
          <el-tag size="small" type="success" effect="plain">{{ enabledJobs.length }}</el-tag>
        </div>
        <div class="job-grid">
          <article
            v-for="job in enabledJobs"
            :key="job.id"
            class="job-card"
            :class="{ failed: job.last_status === 'failed' }"
            @click="openDrawer(job)"
          >
            <div class="job-card-top">
              <el-icon class="job-icon"><Timer /></el-icon>
              <span class="job-name">{{ job.name }}</span>
              <el-switch :model-value="job.enabled" size="small" @click.stop @change="(v: boolean) => handleToggle(job, v)" />
            </div>
            <div class="job-schedule">{{ describeSchedule(job.schedule) }}</div>
            <div v-if="job.next_run_at" class="job-next">{{ t('cron.nextRun') }}: {{ formatTime(job.next_run_at) }}</div>
            <code class="job-command">{{ truncate(job.command, 56) }}</code>
            <div class="job-meta">
              <el-tag size="small" :type="syncTagType(job.sync_status)" effect="plain">{{ job.sync_status || 'pending' }}</el-tag>
              <el-tag v-if="job.last_status" size="small" :type="statusTagType(job.last_status)" effect="plain">{{ job.last_status }}</el-tag>
              <span class="job-last">{{ formatTime(job.last_run_at) }}</span>
            </div>
            <div class="job-actions" @click.stop>
              <el-button size="small" :icon="VideoPlay" @click="handleRun(job)">{{ t('cron.runNow') }}</el-button>
              <el-button size="small" :icon="Document" @click="handleLogs(job)">{{ t('cron.logs') }}</el-button>
              <el-button size="small" :icon="Edit" @click="openEdit(job)">{{ t('common.edit') }}</el-button>
            </div>
          </article>
        </div>
      </section>

      <section v-if="disabledJobs.length" class="job-section">
        <div class="section-head">
          <span>{{ t('cron.sectionDisabled') }}</span>
          <el-tag size="small" type="info" effect="plain">{{ disabledJobs.length }}</el-tag>
        </div>
        <div class="job-grid">
          <article
            v-for="job in disabledJobs"
            :key="job.id"
            class="job-card muted"
            @click="openDrawer(job)"
          >
            <div class="job-card-top">
              <el-icon class="job-icon"><Timer /></el-icon>
              <span class="job-name">{{ job.name }}</span>
              <el-switch :model-value="job.enabled" size="small" @click.stop @change="(v: boolean) => handleToggle(job, v)" />
            </div>
            <div class="job-schedule">{{ describeSchedule(job.schedule) }}</div>
            <div v-if="job.next_run_at" class="job-next">{{ t('cron.nextRun') }}: {{ formatTime(job.next_run_at) }}</div>
            <code class="job-command">{{ truncate(job.command, 56) }}</code>
            <div class="job-actions" @click.stop>
              <el-button size="small" :icon="Edit" @click="openEdit(job)">{{ t('common.edit') }}</el-button>
              <el-button size="small" type="danger" plain :icon="Delete" @click="handleDelete(job)">{{ t('common.delete') }}</el-button>
            </div>
          </article>
        </div>
      </section>

      <el-empty v-if="!filteredJobs.length && !loading" :description="t('cron.noMatch')" />
    </div>

    <!-- Table view -->
    <el-table v-else v-loading="loading" :data="filteredJobs" stripe>
      <el-table-column prop="name" :label="t('cron.name')" width="140" />
      <el-table-column :label="t('cron.schedule')" width="160">
        <template #default="{ row }">
          <div>{{ row.schedule }}</div>
          <div class="table-sub">{{ describeSchedule(row.schedule) }}</div>
        </template>
      </el-table-column>
      <el-table-column :label="t('cron.nextRun')" width="150">
        <template #default="{ row }">{{ formatTime(row.next_run_at) }}</template>
      </el-table-column>
      <el-table-column prop="command" :label="t('cron.command')" min-width="200" show-overflow-tooltip />
      <el-table-column :label="t('cron.syncStatus')" width="100">
        <template #default="{ row }">
          <el-tag size="small" :type="syncTagType(row.sync_status)">{{ row.sync_status || 'pending' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="t('cron.lastRun')" width="150">
        <template #default="{ row }">{{ formatTime(row.last_run_at) }}</template>
      </el-table-column>
      <el-table-column :label="t('cron.lastStatus')" width="90">
        <template #default="{ row }">
          <el-tag v-if="row.last_status" size="small" :type="statusTagType(row.last_status)">{{ row.last_status }}</el-tag>
          <span v-else>—</span>
        </template>
      </el-table-column>
      <el-table-column :label="t('common.status')" width="80">
        <template #default="{ row }">
          <el-switch :model-value="row.enabled" @change="(v: boolean) => handleToggle(row, v)" />
        </template>
      </el-table-column>
      <el-table-column :label="t('common.actions')" width="240" fixed="right">
        <template #default="{ row }">
          <el-button text type="primary" @click="handleRun(row)">{{ t('cron.runNow') }}</el-button>
          <el-button text @click="handleLogs(row)">{{ t('cron.logs') }}</el-button>
          <el-button text @click="openEdit(row)">{{ t('common.edit') }}</el-button>
          <el-button text type="danger" @click="handleDelete(row)">{{ t('common.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- Create / Edit dialog -->
    <el-dialog
      v-model="dialogVisible"
      :title="editingId ? t('cron.editTitle') : t('cron.addTitle')"
      width="680px"
      destroy-on-close
      @closed="resetForm"
    >
      <el-form :model="form" label-width="100px">
        <el-form-item :label="t('cron.name')" required>
          <el-input v-model="form.name" :placeholder="t('cron.namePlaceholder')" />
        </el-form-item>

        <el-form-item :label="t('cron.scheduleMode')">
          <el-radio-group v-model="scheduleMode">
            <el-radio-button value="preset">{{ t('cron.modePreset') }}</el-radio-button>
            <el-radio-button value="hourly">{{ t('cron.modeHourly') }}</el-radio-button>
            <el-radio-button value="daily">{{ t('cron.modeDaily') }}</el-radio-button>
            <el-radio-button value="weekly">{{ t('cron.modeWeekly') }}</el-radio-button>
            <el-radio-button value="custom">{{ t('cron.modeCustom') }}</el-radio-button>
          </el-radio-group>
        </el-form-item>

        <el-form-item v-if="scheduleMode === 'preset'" :label="t('cron.schedule')">
          <div class="preset-grid">
            <button
              v-for="p in presets"
              :key="p.value"
              type="button"
              class="preset-btn"
              :class="{ active: form.schedule === p.value }"
              @click="applyPreset(p.value)"
            >
              <span class="preset-label">{{ t(p.label) }}</span>
              <span class="preset-cron">{{ p.value }}</span>
            </button>
          </div>
        </el-form-item>

        <el-form-item v-else-if="scheduleMode === 'daily'" :label="t('cron.runAt')">
          <div class="time-pickers">
            <el-input-number v-model="dailyHour" :min="0" :max="23" />
            <span>:</span>
            <el-input-number v-model="dailyMinute" :min="0" :max="59" />
          </div>
        </el-form-item>

        <el-form-item v-else-if="scheduleMode === 'weekly'" :label="t('cron.runAt')">
          <div class="weekly-pickers">
            <el-select v-model="weeklyDay" style="width: 120px">
              <el-option v-for="(label, idx) in ['cron.weekSun','cron.weekMon','cron.weekTue','cron.weekWed','cron.weekThu','cron.weekFri','cron.weekSat']" :key="idx" :label="t(label)" :value="idx" />
            </el-select>
            <el-input-number v-model="weeklyHour" :min="0" :max="23" />
            <span>:</span>
            <el-input-number v-model="weeklyMinute" :min="0" :max="59" />
          </div>
        </el-form-item>

        <el-form-item v-if="scheduleMode === 'custom' || scheduleMode === 'hourly'" :label="t('cron.schedule')">
          <el-input v-model="form.schedule" placeholder="0 2 * * *" />
          <p class="form-hint">{{ t('cron.scheduleHint') }}</p>
        </el-form-item>

        <el-form-item v-if="scheduleMode !== 'custom'" :label="t('cron.expression')">
          <code class="expr-preview">{{ form.schedule }}</code>
        </el-form-item>

        <el-form-item :label="t('cron.command')" required>
          <el-input v-model="form.command" type="textarea" :rows="4" :placeholder="t('cron.commandPlaceholder')" />
        </el-form-item>

        <el-form-item v-if="!editingId" :label="t('cron.quickTemplates')">
          <div class="tpl-chips">
            <el-button v-for="tpl in taskTemplates" :key="tpl.id" size="small" @click="applyTemplate(tpl)">
              {{ tpl.name }}
            </el-button>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="saving" @click="handleSave">{{ editingId ? t('common.save') : t('common.create') }}</el-button>
      </template>
    </el-dialog>

    <!-- Detail drawer -->
    <el-drawer v-model="drawerVisible" :title="activeJob?.name || t('cron.title')" size="520px" destroy-on-close>
      <template v-if="activeJob">
        <div class="drawer-tags">
          <el-switch :model-value="activeJob.enabled" @change="(v: boolean) => handleToggle(activeJob!, v)" />
          <el-tag :type="syncTagType(activeJob.sync_status)" effect="plain">{{ activeJob.sync_status || 'pending' }}</el-tag>
          <el-tag v-if="activeJob.last_status" :type="statusTagType(activeJob.last_status)" effect="plain">{{ activeJob.last_status }}</el-tag>
        </div>
        <el-descriptions :column="1" border size="small" class="job-desc">
          <el-descriptions-item :label="t('cron.schedule')">{{ activeJob.schedule }}</el-descriptions-item>
          <el-descriptions-item :label="t('cron.scheduleDesc')">{{ describeSchedule(activeJob.schedule) }}</el-descriptions-item>
          <el-descriptions-item :label="t('cron.lastRun')">{{ formatTime(activeJob.last_run_at) }}</el-descriptions-item>
          <el-descriptions-item v-if="activeJob.next_run_at" :label="t('cron.nextRun')">{{ formatTime(activeJob.next_run_at) }}</el-descriptions-item>
          <el-descriptions-item v-if="activeJob.executor" :label="t('cron.executor')">{{ activeJob.executor === 'system_crontab' ? t('cron.modeSystem') : t('cron.modeInternal') }}</el-descriptions-item>
          <el-descriptions-item v-if="activeJob.sync_message" :label="t('cron.syncStatus')">{{ activeJob.sync_message }}</el-descriptions-item>
        </el-descriptions>
        <div class="drawer-section">
          <div class="section-label">{{ t('cron.command') }}</div>
          <pre class="command-box">{{ activeJob.command }}</pre>
        </div>
        <div v-if="activeJob.last_output" class="drawer-section">
          <div class="section-label">{{ t('cron.lastOutput') }}</div>
          <pre class="output-box">{{ activeJob.last_output }}</pre>
        </div>
        <div class="drawer-actions">
          <el-button type="primary" :icon="VideoPlay" @click="handleRun(activeJob)">{{ t('cron.runNow') }}</el-button>
          <el-button :icon="Document" @click="handleLogs(activeJob)">{{ t('cron.logs') }}</el-button>
          <el-button :icon="Edit" @click="openEdit(activeJob)">{{ t('common.edit') }}</el-button>
          <el-button type="danger" plain :icon="Delete" @click="handleDelete(activeJob)">{{ t('common.delete') }}</el-button>
        </div>
      </template>
    </el-drawer>

    <!-- Logs -->
    <el-dialog v-model="logVisible" :title="t('cron.logTitle', { name: logTitle })" width="760px" destroy-on-close>
      <div class="log-toolbar">
        <el-button size="small" :icon="RefreshRight" :loading="logLoading" @click="fetchLogs">{{ t('common.refresh') }}</el-button>
      </div>
      <pre v-loading="logLoading" class="log-box">{{ logContent }}</pre>
    </el-dialog>
  </div>
</template>

<style scoped>
.cron-page { padding-bottom: 24px; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; margin-bottom: 16px; flex-wrap: wrap; }
.page-header h2 { margin: 0 0 4px; }
.page-sub { margin: 0; font-size: 13px; color: var(--el-text-color-secondary); }
.header-actions { display: flex; gap: 8px; flex-shrink: 0; }
.hint { margin-bottom: 16px; }

.stats-row { display: grid; grid-template-columns: repeat(4, 1fr); gap: 12px; margin-bottom: 16px; max-width: 640px; }
.stat-card { background: var(--el-fill-color-light); border-radius: 10px; padding: 12px 16px; display: flex; flex-direction: column; gap: 2px; }
.stat-card.enabled { border-left: 3px solid var(--el-color-success); }
.stat-card.disabled { border-left: 3px solid var(--el-text-color-placeholder); }
.stat-card.failed { border-left: 3px solid var(--el-color-danger); }
.stat-value { font-size: 22px; font-weight: 600; }
.stat-label { font-size: 12px; color: var(--el-text-color-secondary); }

.toolbar { display: flex; align-items: center; gap: 12px; margin-bottom: 16px; flex-wrap: wrap; }
.search-input { max-width: 280px; }
.view-toggle { margin-left: auto; }

.template-section { margin-bottom: 20px; }
.section-head.clickable { cursor: pointer; user-select: none; }
.expand-hint { margin-left: auto; color: var(--el-text-color-secondary); font-size: 12px; }
.scheduler-badge { margin-left: 8px; font-size: 12px; color: var(--el-color-primary); }
.empty-only { padding: 24px 0; }
.job-next { font-size: 12px; color: var(--el-text-color-secondary); margin-bottom: 6px; }
.template-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 12px;
  width: 100%;
}
.template-card {
  display: flex; flex-direction: column; align-items: center; gap: 6px; padding: 14px 10px;
  border: 1px solid var(--el-border-color-lighter); border-radius: 12px; background: var(--el-bg-color);
  cursor: pointer; transition: border-color .2s, box-shadow .2s; text-align: center;
  width: 100%; min-width: 0; font: inherit; color: inherit;
}
.template-card:hover { border-color: var(--el-color-primary); box-shadow: 0 4px 12px rgba(0,0,0,.06); }
.template-icon { width: 36px; height: 36px; border-radius: 8px; color: #fff; font-size: 11px; font-weight: 700; display: flex; align-items: center; justify-content: center; }
.template-name { font-weight: 600; font-size: 13px; }
.template-desc { font-size: 11px; color: var(--el-text-color-secondary); line-height: 1.3; }

.job-board { display: flex; flex-direction: column; gap: 24px; }
.job-section { display: flex; flex-direction: column; gap: 12px; }
.section-head { display: flex; align-items: center; gap: 8px; font-weight: 600; font-size: 14px; }
.job-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 14px; }
.job-card {
  border: 1px solid var(--el-border-color-lighter); border-radius: 12px; padding: 14px;
  background: var(--el-bg-color); cursor: pointer; transition: border-color .2s, box-shadow .2s;
}
.job-card:hover { border-color: var(--el-color-primary-light-5); box-shadow: 0 4px 14px rgba(0,0,0,.06); }
.job-card.failed { border-left: 3px solid var(--el-color-danger); }
.job-card.muted { opacity: .85; }
.job-card-top { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; }
.job-icon { color: var(--el-color-primary); flex-shrink: 0; }
.job-name { font-weight: 600; font-size: 15px; flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.job-schedule { font-size: 13px; color: var(--el-color-primary); margin-bottom: 6px; }
.job-command { display: block; font-size: 11px; color: var(--el-text-color-secondary); background: var(--el-fill-color-light); padding: 6px 8px; border-radius: 6px; margin-bottom: 8px; word-break: break-all; }
.job-meta { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; margin-bottom: 10px; }
.job-last { font-size: 11px; color: var(--el-text-color-placeholder); margin-left: auto; }
.job-actions { display: flex; gap: 6px; flex-wrap: wrap; }

.table-sub { font-size: 11px; color: var(--el-text-color-secondary); margin-top: 2px; }

.preset-grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 8px; width: 100%; }
.preset-btn {
  text-align: left; padding: 10px 12px; border: 1px solid var(--el-border-color); border-radius: 8px;
  background: var(--el-fill-color-blank); cursor: pointer; transition: border-color .2s;
}
.preset-btn.active { border-color: var(--el-color-primary); background: var(--el-color-primary-light-9); }
.preset-label { display: block; font-weight: 600; font-size: 13px; }
.preset-cron { display: block; font-size: 11px; color: var(--el-text-color-secondary); font-family: monospace; margin-top: 2px; }
.time-pickers, .weekly-pickers { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.expr-preview { font-size: 13px; background: var(--el-fill-color-light); padding: 4px 8px; border-radius: 4px; }
.form-hint { margin: 6px 0 0; font-size: 12px; color: var(--el-text-color-secondary); }
.tpl-chips { display: flex; flex-wrap: wrap; gap: 6px; }

.drawer-tags { display: flex; align-items: center; gap: 10px; margin-bottom: 16px; flex-wrap: wrap; }
.job-desc { margin-bottom: 16px; }
.drawer-section { margin-bottom: 16px; }
.section-label { font-size: 12px; font-weight: 600; color: var(--el-text-color-secondary); margin-bottom: 8px; text-transform: uppercase; letter-spacing: .04em; }
.command-box, .output-box {
  font-size: 12px; background: var(--el-fill-color-light); padding: 10px; border-radius: 6px;
  white-space: pre-wrap; word-break: break-all; margin: 0; max-height: 160px; overflow: auto;
}
.drawer-actions { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 8px; }

.log-toolbar { margin-bottom: 8px; }
.log-box {
  max-height: 420px; overflow: auto; background: #1e1e1e; color: #d4d4d4;
  padding: 12px; border-radius: 6px; font-size: 12px; line-height: 1.5; white-space: pre-wrap; word-break: break-all; margin: 0;
}

@media (max-width: 768px) {
  .stats-row { grid-template-columns: repeat(2, 1fr); max-width: none; }
  .template-grid { grid-template-columns: repeat(auto-fill, minmax(140px, 1fr)); }
  .job-grid { grid-template-columns: 1fr; }
  .preset-grid { grid-template-columns: 1fr; }
  .search-input { width: 100%; max-width: none; }
  .view-toggle { margin-left: 0; }
}
</style>
