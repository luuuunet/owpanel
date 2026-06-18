<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { MagicStick } from '@element-plus/icons-vue'
import api, { AI_REQUEST_TIMEOUT, resolveApiError } from '@/api'
import { ElMessage } from 'element-plus'

const props = defineProps<{
  modelValue: boolean
  app: { key: string; name: string; category?: string } | null
}>()

const emit = defineEmits<{ 'update:modelValue': [boolean]; saved: [] }>()

const { t } = useI18n()

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

const loading = ref(false)
const saving = ref(false)
const activeTab = ref('common')

const configPath = ref('')
const resolvedPath = ref('')
const hasConfigFile = ref(false)
const isPhp = ref(false)
const capabilities = ref<any>({})

const configForm = ref<Record<string, string>>({})
const rawContent = ref('')
const disableFunctions = ref('')
const extensions = ref<any[]>([])
const canInstall = ref(false)
const installExtName = ref('')
const installingExt = ref(false)
const pgInstallingName = ref('')

const chatMessages = ref<{ role: string; content: string }[]>([])
const chatInput = ref('')
const chatLoading = ref(false)
const chatBoxRef = ref<HTMLElement | null>(null)
const aiSuggestion = ref<{ config?: Record<string, any>; raw?: string } | null>(null)

const commonFields = computed(() => {
  const skip = new Set(['disable_functions'])
  return Object.keys(configForm.value).filter(k => !skip.has(k))
})

const supportsExtensions = computed(() => capabilities.value?.supports_extensions ?? isPhp.value)
const supportsPgExtensions = computed(() => capabilities.value?.supports_pg_extensions ?? props.app?.key === 'postgresql')
const supportsDisable = computed(() => capabilities.value?.supports_disable_functions ?? isPhp.value)
const supportsAI = computed(() => capabilities.value?.supports_ai !== false)

const disablePresets = [
  { label: 'exec,passthru,shell_exec,system', value: 'exec,passthru,shell_exec,system,proc_open,popen' },
  { label: 'phpinfo,eval', value: 'phpinfo,eval,assert,create_function' },
  { label: t('software.phpConfig.clearDisabled'), value: '' },
]

const aiPrompts = computed(() => [
  t('software.phpConfig.aiPromptOptimize'),
  t('software.phpConfig.aiPromptSecurity'),
  t('software.phpConfig.aiPromptExplain'),
])

async function load() {
  if (!props.app) return
  loading.value = true
  aiSuggestion.value = null
  try {
    const res: any = await api.get(`/software/${props.app.key}/config`)
    const raw = res.data?.config || {}
    configForm.value = Object.fromEntries(
      Object.entries(raw).map(([k, v]) => [k, typeof v === 'object' ? JSON.stringify(v) : String(v ?? '')])
    )
    configPath.value = res.data?.config_path || ''
    resolvedPath.value = res.data?.resolved_config_path || ''
    hasConfigFile.value = !!res.data?.has_config_file
    isPhp.value = !!res.data?.is_php
    capabilities.value = res.data?.capabilities || {}
    disableFunctions.value = String(raw.disable_functions ?? configForm.value.disable_functions ?? '')

    try {
      const rawRes: any = await api.get(`/software/${props.app.key}/config/raw`)
      rawContent.value = rawRes.data?.content || ''
      if (rawRes.data?.path) resolvedPath.value = rawRes.data.path
    } catch {
      rawContent.value = ''
    }

    if (supportsExtensions.value) {
      const phpRes: any = await api.get(`/software/${props.app.key}/php/detail`)
      extensions.value = phpRes.data?.extensions || []
      canInstall.value = !!phpRes.data?.can_install
      if (phpRes.data?.disable_functions) {
        disableFunctions.value = phpRes.data.disable_functions
      }
      if (phpRes.data?.ini_path) resolvedPath.value = phpRes.data.ini_path
    } else if (supportsPgExtensions.value) {
      const pgRes: any = await api.get(`/software/${props.app.key}/pgsql/detail`)
      extensions.value = (pgRes.data?.extensions || []).map((e: any) => ({
        ...e,
        enabled: e.available,
        loaded: e.installed,
      }))
      canInstall.value = !!pgRes.data?.can_install
    } else {
      extensions.value = []
    }
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
  } finally {
    loading.value = false
  }
}

watch(() => props.modelValue, (v) => {
  if (v && props.app) {
    activeTab.value = 'common'
    chatMessages.value = []
    load()
  }
})

async function saveCommon() {
  if (!props.app) return
  saving.value = true
  try {
    const cfg: Record<string, any> = {}
    for (const [k, v] of Object.entries(configForm.value)) {
      if (k === 'disable_functions') continue
      try { cfg[k] = JSON.parse(v) } catch { cfg[k] = v }
    }
    if (supportsDisable.value) {
      cfg.disable_functions = disableFunctions.value
    }
    await api.put(`/software/${props.app.key}/config`, { config: cfg })
    ElMessage.success(t('software.saveConfig'))
    emit('saved')
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
  } finally {
    saving.value = false
  }
}

async function saveDisableFunctions() {
  if (!props.app || !supportsDisable.value) return
  saving.value = true
  try {
    await api.put(`/software/${props.app.key}/php/disable-functions`, { functions: disableFunctions.value })
    configForm.value.disable_functions = disableFunctions.value
    ElMessage.success(t('software.phpConfig.disableSaved'))
    emit('saved')
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
  } finally {
    saving.value = false
  }
}

async function toggleExtension(ext: any) {
  if (!props.app) return
  try {
    await api.put(`/software/${props.app.key}/php/extensions/${ext.name}`, { enabled: !ext.enabled })
    ext.enabled = !ext.enabled
    ElMessage.success(t('common.success'))
    emit('saved')
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
  }
}

async function installExtension() {
  if (!props.app || !installExtName.value.trim()) return
  installingExt.value = true
  try {
    await api.post(`/software/${props.app.key}/php/extensions/install`, { name: installExtName.value.trim() })
    ElMessage.success(t('software.phpConfig.installSuccess'))
    installExtName.value = ''
    await load()
    emit('saved')
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
  } finally {
    installingExt.value = false
  }
}

async function installPgExtension(row: any) {
  if (!props.app) return
  pgInstallingName.value = row.name
  try {
    await api.post(`/software/${props.app.key}/pgsql/extensions/install`, { name: row.name })
    ElMessage.success(t('software.pgsqlConfig.installSuccess'))
    await load()
    emit('saved')
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
  } finally {
    pgInstallingName.value = ''
  }
}

async function saveRaw() {
  if (!props.app) return
  saving.value = true
  try {
    await api.put(`/software/${props.app.key}/config/raw`, { content: rawContent.value })
    ElMessage.success(t('software.saveConfig'))
    emit('saved')
  } catch (e: any) {
    ElMessage.error(resolveApiError(e, t('common.failed')))
  } finally {
    saving.value = false
  }
}

function applyPreset(val: string) {
  disableFunctions.value = val
}

async function scrollChat() {
  await nextTick()
  if (chatBoxRef.value) chatBoxRef.value.scrollTop = chatBoxRef.value.scrollHeight
}

async function sendChat(preset?: string) {
  if (!props.app) return
  const msg = (preset || chatInput.value).trim()
  if (!msg || chatLoading.value) return
  const list = [...chatMessages.value, { role: 'user', content: msg }]
  chatMessages.value = list
  chatInput.value = ''
  chatLoading.value = true
  aiSuggestion.value = null
  await scrollChat()
  try {
    const cfg: Record<string, any> = {}
    for (const [k, v] of Object.entries(configForm.value)) {
      try { cfg[k] = JSON.parse(v) } catch { cfg[k] = v }
    }
    const res: any = await api.post(`/software/${props.app.key}/config/ai/chat`, {
      message: msg,
      app_name: props.app.name,
      category: props.app.category || '',
      config_kind: capabilities.value?.config_kind || '',
      config: cfg,
      raw_content: rawContent.value,
      history: list.slice(-10).map(m => ({ role: m.role, content: m.content })),
    }, { timeout: AI_REQUEST_TIMEOUT })
    chatMessages.value = [...list, { role: 'assistant', content: res.data?.reply || '' }]
    if (res.data?.suggested_config || res.data?.suggested_raw) {
      aiSuggestion.value = {
        config: res.data.suggested_config,
        raw: res.data.suggested_raw,
      }
    }
  } catch (e: any) {
    chatMessages.value = [...list, { role: 'assistant', content: resolveApiError(e, t('software.phpConfig.aiFailed'), t('common.requestTimeout')) }]
  } finally {
    chatLoading.value = false
    scrollChat()
  }
}

function applyAISuggestion() {
  if (!aiSuggestion.value) return
  if (aiSuggestion.value.config) {
    for (const [k, v] of Object.entries(aiSuggestion.value.config)) {
      configForm.value[k] = typeof v === 'object' ? JSON.stringify(v) : String(v ?? '')
    }
  }
  if (aiSuggestion.value.raw) {
    rawContent.value = aiSuggestion.value.raw
  }
  aiSuggestion.value = null
  ElMessage.success(t('software.phpConfig.aiApplied'))
}
</script>

<template>
  <el-dialog
    v-model="visible"
    :title="t('software.configTitle', { name: app?.name })"
    width="820px"
    destroy-on-close
    class="software-config-dialog"
  >
    <el-alert
      v-if="resolvedPath || configPath"
      :title="`${t('software.configFile')}: ${resolvedPath || configPath}`"
      type="info"
      :closable="false"
      style="margin-bottom: 12px"
    />

    <div class="cap-tags" v-if="capabilities">
      <el-tag v-if="supportsExtensions" size="small" type="success">{{ t('software.phpConfig.capExtensions') }}</el-tag>
      <el-tag v-if="supportsPgExtensions" size="small" type="success">{{ t('software.pgsqlConfig.capExtensions') }}</el-tag>
      <el-tag v-if="supportsDisable" size="small" type="warning">{{ t('software.phpConfig.capDisable') }}</el-tag>
      <el-tag v-if="capabilities.supports_raw_edit" size="small">{{ t('software.phpConfig.capRaw') }}</el-tag>
      <el-tag v-if="capabilities.is_docker_app" size="small" type="info">{{ t('software.phpConfig.capDocker') }}</el-tag>
      <el-tag v-if="supportsAI" size="small" type="primary">{{ t('software.phpConfig.capAI') }}</el-tag>
    </div>

    <div v-loading="loading">
      <el-tabs v-model="activeTab">
        <el-tab-pane :label="t('software.phpConfig.tabCommon')" name="common">
          <el-form label-width="180px" style="max-height: 420px; overflow-y: auto">
            <el-form-item v-for="key in commonFields" :key="key" :label="key">
              <el-input v-model="configForm[key]" />
            </el-form-item>
            <el-empty v-if="!commonFields.length" :description="t('software.phpConfig.noCommonFields')" />
          </el-form>
          <div class="footer-actions">
            <el-button type="primary" :loading="saving" @click="saveCommon">{{ t('software.saveConfig') }}</el-button>
          </div>
        </el-tab-pane>

        <el-tab-pane v-if="supportsDisable" :label="t('software.phpConfig.tabDisable')" name="disable">
          <p class="hint">{{ t('software.phpConfig.disableHint') }}</p>
          <el-input
            v-model="disableFunctions"
            type="textarea"
            :rows="5"
            :placeholder="t('software.phpConfig.disablePlaceholder')"
          />
          <div class="preset-row">
            <span>{{ t('software.phpConfig.presets') }}:</span>
            <el-button v-for="p in disablePresets" :key="p.label" size="small" @click="applyPreset(p.value)">
              {{ p.label }}
            </el-button>
          </div>
          <div class="footer-actions">
            <el-button type="primary" :loading="saving" @click="saveDisableFunctions">{{ t('common.save') }}</el-button>
          </div>
        </el-tab-pane>

        <el-tab-pane v-if="supportsExtensions" :label="t('software.phpConfig.tabExtensions')" name="extensions">
          <el-table :data="extensions" stripe max-height="360" size="small">
            <el-table-column prop="name" :label="t('software.phpConfig.extName')" width="140" />
            <el-table-column prop="file" :label="t('software.phpConfig.extFile')" show-overflow-tooltip />
            <el-table-column :label="t('common.status')" width="100">
              <template #default="{ row }">
                <el-tag v-if="row.loaded" type="success" size="small">{{ t('software.phpConfig.loaded') }}</el-tag>
                <el-tag v-else-if="row.enabled" type="warning" size="small">{{ t('software.phpConfig.enabled') }}</el-tag>
                <el-tag v-else type="info" size="small">{{ t('software.phpConfig.disabled') }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column :label="t('common.actions')" width="100">
              <template #default="{ row }">
                <el-button
                  v-if="!row.builtin"
                  text
                  :type="row.enabled ? 'danger' : 'success'"
                  size="small"
                  @click="toggleExtension(row)"
                >
                  {{ row.enabled ? t('software.phpConfig.disableExt') : t('software.phpConfig.enableExt') }}
                </el-button>
              </template>
            </el-table-column>
          </el-table>
          <div v-if="canInstall" class="install-row">
            <el-input v-model="installExtName" :placeholder="t('software.phpConfig.installPlaceholder')" style="width: 220px" />
            <el-button type="primary" :loading="installingExt" @click="installExtension">{{ t('software.phpConfig.installExt') }}</el-button>
          </div>
          <p v-else class="hint">{{ t('software.phpConfig.winExtHint') }}</p>
        </el-tab-pane>

        <el-tab-pane v-if="supportsPgExtensions" :label="t('software.pgsqlConfig.tabExtensions')" name="pg-extensions">
          <p class="hint">{{ t('software.pgsqlConfig.selectDatabase') }}</p>
          <el-table :data="extensions" stripe max-height="360" size="small">
            <el-table-column prop="name" :label="t('software.phpConfig.extName')" width="150" />
            <el-table-column prop="description" :label="t('databases.extDescription')" show-overflow-tooltip />
            <el-table-column :label="t('common.status')" width="120">
              <template #default="{ row }">
                <el-tag v-if="row.available" type="success" size="small">{{ t('software.pgsqlConfig.available') }}</el-tag>
                <el-tag v-else type="info" size="small">{{ t('software.pgsqlConfig.notAvailable') }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column :label="t('common.actions')" width="120">
              <template #default="{ row }">
                <el-button
                  v-if="!row.available && row.can_install"
                  text
                  type="primary"
                  size="small"
                  :loading="pgInstallingName === row.name"
                  @click="installPgExtension(row)"
                >
                  {{ t('software.pgsqlConfig.installPackage') }}
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane :label="t('software.phpConfig.tabRaw')" name="raw">
          <p class="hint">{{ hasConfigFile ? t('software.phpConfig.rawEditHint') : t('software.phpConfig.noRawFile') }}</p>
          <el-input
            v-model="rawContent"
            type="textarea"
            :rows="18"
            :placeholder="t('software.phpConfig.rawPlaceholder')"
            class="raw-editor"
          />
          <div class="footer-actions">
            <el-button type="primary" :loading="saving" @click="saveRaw">
              {{ t('software.saveConfig') }}
            </el-button>
          </div>
        </el-tab-pane>

        <el-tab-pane v-if="supportsAI" :label="t('software.phpConfig.tabAI')" name="ai">
          <p class="hint">{{ t('software.phpConfig.aiWelcome') }}</p>
          <div ref="chatBoxRef" class="ai-chat-box">
            <div v-for="(m, i) in chatMessages" :key="i" class="ai-msg" :class="m.role">
              <span class="ai-role">{{ m.role === 'user' ? t('cache.you') : 'AI' }}</span>
              <pre class="ai-text">{{ m.content }}</pre>
            </div>
            <div v-if="!chatMessages.length" class="ai-empty">{{ t('software.phpConfig.aiEmpty') }}</div>
          </div>
          <div v-if="aiSuggestion" class="ai-suggestion">
            <span>{{ t('software.phpConfig.aiSuggestion') }}</span>
            <el-button type="primary" size="small" @click="applyAISuggestion">{{ t('software.phpConfig.aiApply') }}</el-button>
          </div>
          <div class="ai-prompts">
            <el-button v-for="p in aiPrompts" :key="p" size="small" :disabled="chatLoading" @click="sendChat(p)">{{ p }}</el-button>
          </div>
          <div class="ai-input-row">
            <el-input
              v-model="chatInput"
              type="textarea"
              :rows="2"
              :placeholder="t('software.phpConfig.aiPlaceholder')"
              @keydown.ctrl.enter="sendChat()"
            />
            <el-button type="primary" :icon="MagicStick" :loading="chatLoading" @click="sendChat()">
              {{ t('software.phpConfig.aiSend') }}
            </el-button>
          </div>
        </el-tab-pane>
      </el-tabs>
    </div>
  </el-dialog>
</template>

<style scoped>
.hint {
  color: var(--el-text-color-secondary);
  font-size: 13px;
  margin-bottom: 12px;
}
.cap-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 12px;
}
.preset-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
}
.install-row {
  display: flex;
  gap: 10px;
  margin-top: 14px;
  align-items: center;
}
.footer-actions {
  margin-top: 16px;
  text-align: right;
}
.raw-editor :deep(textarea) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 13px;
}
.ai-chat-box {
  max-height: 280px;
  overflow-y: auto;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  padding: 10px;
  margin-bottom: 10px;
  background: var(--el-fill-color-blank);
}
.ai-msg {
  margin-bottom: 10px;
}
.ai-msg.user .ai-text {
  background: var(--el-color-primary-light-9);
}
.ai-role {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  display: block;
  margin-bottom: 4px;
}
.ai-text {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 13px;
  padding: 8px;
  border-radius: 4px;
  background: var(--el-fill-color-light);
  font-family: inherit;
}
.ai-empty {
  color: var(--el-text-color-placeholder);
  font-size: 13px;
  text-align: center;
  padding: 24px;
}
.ai-suggestion {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
  font-size: 13px;
}
.ai-prompts {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 10px;
}
.ai-input-row {
  display: flex;
  gap: 10px;
  align-items: flex-end;
}
.ai-input-row .el-button {
  flex-shrink: 0;
}
</style>
