<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { getSoftwareIconMeta, getSoftwareIconDataUrl, getSoftwareLogoFallback, getSoftwareLogoUrl } from '@/config/softwareIcons'

const props = withDefaults(defineProps<{
  appKey: string
  iconUrl?: string
  size?: number
  /** Letter badge only — no logo image (dashboard list) */
  simple?: boolean
}>(), {
  size: 48,
  simple: false,
})

const imgFailed = ref(false)
const useFallback = ref(false)
const meta = computed(() => getSoftwareIconMeta(props.appKey))
const logoUrl = computed(() => {
  if (props.iconUrl) {
    return props.iconUrl
  }
  if (useFallback.value) {
    return getSoftwareLogoFallback(props.appKey) || getSoftwareIconDataUrl(props.appKey)
  }
  return getSoftwareLogoUrl(props.appKey)
})

watch(() => props.appKey, () => {
  imgFailed.value = false
  useFallback.value = false
})

function onImgError() {
  if (!useFallback.value && getSoftwareLogoFallback(props.appKey)) {
    useFallback.value = true
    return
  }
  imgFailed.value = true
}
</script>

<template>
  <div
    class="software-icon"
    :class="{ simple }"
    :style="{ width: `${size}px`, height: `${size}px` }"
    :title="appKey"
  >
    <div
      v-if="simple || imgFailed"
      class="software-icon-fallback"
      :style="{ background: meta.bg, fontSize: `${Math.max(10, Math.round(size * 0.34))}px` }"
    >
      {{ meta.label }}
    </div>
    <img
      v-else
      :src="logoUrl"
      :alt="appKey"
      class="software-icon-img"
      @error="onImgError"
    />
  </div>
</template>

<style scoped>
.software-icon {
  flex-shrink: 0;
  border-radius: 10px;
  overflow: hidden;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}

.software-icon.simple {
  border-radius: 8px;
  box-shadow: none;
  border: 1px solid rgba(0, 0, 0, 0.06);
}

.software-icon-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.software-icon-fallback {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-weight: 700;
  letter-spacing: -0.5px;
}
</style>
