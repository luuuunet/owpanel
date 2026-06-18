<script setup lang="ts">

import { computed } from 'vue'

import { useI18n } from 'vue-i18n'

import { useLocaleStore } from '@/stores/locale'

import { LOCALE_OPTIONS, type LocaleCode } from '@/locales'



const props = defineProps<{

  modelValue?: LocaleCode

}>()



const emit = defineEmits<{

  'update:modelValue': [LocaleCode]

}>()



const localeStore = useLocaleStore()

const { t } = useI18n()



const current = computed({

  get: () => props.modelValue ?? localeStore.locale,

  set: (v: LocaleCode) => {

    if (props.modelValue !== undefined) {

      emit('update:modelValue', v)

    } else {

      localeStore.setLocale(v)

    }

  },

})



const options = LOCALE_OPTIONS

</script>



<template>

  <el-select v-model="current" size="small" style="width: 132px" :placeholder="t('common.language')">

    <el-option v-for="opt in options" :key="opt.value" :label="opt.label" :value="opt.value" />

  </el-select>

</template>


