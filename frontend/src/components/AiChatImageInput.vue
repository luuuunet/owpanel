<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Picture, Close } from '@element-plus/icons-vue'
import {
  AI_CHAT_MAX_IMAGES,
  extractClipboardImages,
  filesToAiChatImages,
} from '@/utils/aiImages'

const props = withDefaults(defineProps<{
  modelValue: string[]
  disabled?: boolean
}>(), {
  disabled: false,
})

const emit = defineEmits<{
  'update:modelValue': [string[]]
}>()

const { t } = useI18n()
const fileInputRef = ref<HTMLInputElement>()

function setImages(next: string[]) {
  emit('update:modelValue', next)
}

async function addFiles(files: FileList | File[]) {
  if (props.disabled) return
  try {
    const added = await filesToAiChatImages(files, props.modelValue.length)
    setImages([...props.modelValue, ...added])
  } catch (e: any) {
    const key = e?.message
    if (key === 'tooMany') ElMessage.warning(t('aiChat.imageTooMany', { n: AI_CHAT_MAX_IMAGES }))
    else if (key === 'tooLarge') ElMessage.warning(t('aiChat.imageTooLarge'))
    else if (key === 'unsupportedType') ElMessage.warning(t('aiChat.imageUnsupported'))
    else ElMessage.error(t('aiChat.imageReadFailed'))
  }
}

function onPickClick() {
  if (props.disabled) return
  fileInputRef.value?.click()
}

function onFileChange(e: Event) {
  const input = e.target as HTMLInputElement
  if (input.files?.length) {
    void addFiles(input.files)
  }
  input.value = ''
}

function removeAt(index: number) {
  setImages(props.modelValue.filter((_, i) => i !== index))
}

async function onPaste(e: ClipboardEvent) {
  if (props.disabled) return
  const files = extractClipboardImages(e)
  if (!files.length) return
  e.preventDefault()
  await addFiles(files)
}

defineExpose({ onPaste })
</script>

<template>
  <div class="ai-chat-image-input" @paste="onPaste">
    <div v-if="modelValue.length" class="image-strip">
      <div v-for="(img, i) in modelValue" :key="i" class="image-thumb">
        <img :src="img" alt="" />
        <button type="button" class="remove-btn" :disabled="disabled" @click="removeAt(i)">
          <el-icon><Close /></el-icon>
        </button>
      </div>
    </div>
    <div class="image-actions">
      <el-button size="small" :icon="Picture" :disabled="disabled || modelValue.length >= AI_CHAT_MAX_IMAGES" @click="onPickClick">
        {{ t('aiChat.addImage') }}
      </el-button>
      <span class="hint">{{ t('aiChat.imageHint', { n: AI_CHAT_MAX_IMAGES }) }}</span>
    </div>
    <input
      ref="fileInputRef"
      type="file"
      accept="image/jpeg,image/png,image/gif,image/webp"
      multiple
      class="hidden-input"
      @change="onFileChange"
    />
  </div>
</template>

<style scoped>
.ai-chat-image-input {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.image-strip {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.image-thumb {
  position: relative;
  width: 72px;
  height: 72px;
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid var(--el-border-color);
  background: var(--el-fill-color-light);
}
.image-thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.remove-btn {
  position: absolute;
  top: 2px;
  right: 2px;
  width: 20px;
  height: 20px;
  border: none;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.55);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  padding: 0;
}
.image-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.hint {
  font-size: 11px;
  color: var(--el-text-color-secondary);
}
.hidden-input {
  display: none;
}
</style>
