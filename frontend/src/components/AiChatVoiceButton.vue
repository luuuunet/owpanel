<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Microphone } from '@element-plus/icons-vue'
import { useSpeechInput } from '@/composables/useSpeechInput'

const props = withDefaults(defineProps<{
  disabled?: boolean
}>(), {
  disabled: false,
})

const emit = defineEmits<{
  transcript: [string]
}>()

const { t, locale } = useI18n()
const { supported, listening, interimText, start, stop } = useSpeechInput(() => locale.value)

function onError(key: string) {
  if (key === 'unsupported') ElMessage.warning(t('aiChat.voiceUnsupported'))
  else if (key === 'denied') ElMessage.warning(t('aiChat.voiceDenied'))
  else if (key === 'noSpeech') ElMessage.info(t('aiChat.voiceNoSpeech'))
  else ElMessage.error(t('aiChat.voiceFailed'))
}

function toggle() {
  if (props.disabled || !supported.value) return
  if (listening.value) {
    stop()
    return
  }
  start((text) => emit('transcript', text), onError)
}
</script>

<template>
  <div class="ai-chat-voice">
    <el-tooltip :content="listening ? t('aiChat.voiceStop') : t('aiChat.voiceStart')">
      <el-button
        class="voice-btn"
        :class="{ listening }"
        :type="listening ? 'danger' : 'default'"
        size="small"
        :icon="Microphone"
        :disabled="disabled || !supported"
        @click="toggle"
      />
    </el-tooltip>
    <span v-if="listening" class="voice-status">
      {{ interimText || t('aiChat.voiceListening') }}
    </span>
  </div>
</template>

<style scoped>
.ai-chat-voice {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.voice-btn.listening {
  animation: voice-pulse 1.2s ease-in-out infinite;
}
.voice-status {
  font-size: 11px;
  color: var(--el-color-danger);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 180px;
}
@keyframes voice-pulse {
  0%, 100% { box-shadow: 0 0 0 0 rgba(245, 108, 108, 0.35); }
  50% { box-shadow: 0 0 0 6px rgba(245, 108, 108, 0); }
}
</style>
