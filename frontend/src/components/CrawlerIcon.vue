<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { getCrawlerIconMeta, getCrawlerLogoUrl } from '@/config/crawlerIcons'

const props = withDefaults(defineProps<{
  icon?: string
  name?: string
  size?: number
}>(), {
  size: 26,
})

const imgFailed = ref(false)
const meta = computed(() => getCrawlerIconMeta(props.icon))
const logoUrl = computed(() => getCrawlerLogoUrl(props.icon))

watch(() => props.icon, () => {
  imgFailed.value = false
})

function onImgError() {
  imgFailed.value = true
}
</script>

<template>
  <div
    class="crawler-icon"
    :style="{ width: `${size}px`, height: `${size}px` }"
    :title="name"
  >
    <div
      v-if="imgFailed"
      class="crawler-icon-fallback"
      :style="{ background: meta.bg, fontSize: `${Math.max(9, Math.round(size * 0.34))}px` }"
    >
      {{ meta.label }}
    </div>
    <img
      v-else
      :src="logoUrl"
      :alt="name || icon || 'crawler'"
      class="crawler-icon-img"
      @error="onImgError"
    />
  </div>
</template>

<style scoped>
.crawler-icon {
  flex-shrink: 0;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.08);
  background: var(--el-fill-color-blank);
}

.crawler-icon-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.crawler-icon-fallback {
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
