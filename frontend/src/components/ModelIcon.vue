<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { getModelIconMeta, getModelLogoUrl, getModelIconMetaForModel, getModelLogoUrlForModel } from '@/config/modelIcons'

const props = withDefaults(defineProps<{
  catalogId: string
  modelId?: string
  size?: number
}>(), {
  size: 40,
})

const imgFailed = ref(false)
const meta = computed(() =>
  props.modelId
    ? getModelIconMetaForModel(props.modelId)
    : getModelIconMeta(props.catalogId)
)
const logoUrl = computed(() =>
  props.modelId
    ? getModelLogoUrlForModel(props.modelId)
    : getModelLogoUrl(props.catalogId)
)

watch(() => [props.catalogId, props.modelId], () => {
  imgFailed.value = false
})

function onImgError() {
  imgFailed.value = true
}
</script>

<template>
  <div
    class="model-icon"
    :style="{ width: `${size}px`, height: `${size}px` }"
    :title="meta.vendor"
  >
    <div
      v-if="imgFailed"
      class="model-icon-fallback"
      :style="{ background: meta.bg, fontSize: `${Math.max(10, Math.round(size * 0.32))}px` }"
    >
      {{ meta.label }}
    </div>
    <img
      v-else
      :src="logoUrl"
      :alt="catalogId"
      class="model-icon-img"
      @error="onImgError"
    />
  </div>
</template>

<style scoped>
.model-icon {
  flex-shrink: 0;
  border-radius: 10px;
  overflow: hidden;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.08);
  background: var(--el-fill-color-blank);
}

.model-icon-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.model-icon-fallback {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-weight: 700;
  letter-spacing: -0.02em;
}
</style>
