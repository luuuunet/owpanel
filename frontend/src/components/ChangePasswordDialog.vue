<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import api from '@/api'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'

const props = defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [v: boolean]; done: [] }>()

const { t } = useI18n()
const auth = useAuthStore()
const loading = ref(false)
const form = ref({ password: '', confirm: '' })

async function submit() {
  if (form.value.password.length < 8) {
    ElMessage.warning(t('changePassword.minLength'))
    return
  }
  if (form.value.password !== form.value.confirm) {
    ElMessage.warning(t('changePassword.mismatch'))
    return
  }
  loading.value = true
  try {
    await api.post('/auth/change-password', { password: form.value.password })
    if (auth.user) auth.user.must_change_password = false
    ElMessage.success(t('changePassword.success'))
    emit('update:modelValue', false)
    emit('done')
    form.value = { password: '', confirm: '' }
  } catch (e: any) {
    ElMessage.error(e?.error || e?.message || t('common.failed'))
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <el-dialog
    :model-value="props.modelValue"
    :title="t('changePassword.title')"
    width="440px"
    :close-on-click-modal="false"
    :show-close="!auth.user?.must_change_password"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <el-alert
      v-if="auth.user?.must_change_password"
      type="warning"
      :closable="false"
      show-icon
      :title="t('changePassword.requiredHint')"
      style="margin-bottom: 16px"
    />
    <el-form label-width="100px">
      <el-form-item :label="t('changePassword.newPassword')">
        <el-input v-model="form.password" type="password" show-password />
      </el-form-item>
      <el-form-item :label="t('changePassword.confirmPassword')">
        <el-input v-model="form.confirm" type="password" show-password />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button v-if="!auth.user?.must_change_password" @click="emit('update:modelValue', false)">
        {{ t('common.cancel') }}
      </el-button>
      <el-button type="primary" :loading="loading" @click="submit">{{ t('common.save') }}</el-button>
    </template>
  </el-dialog>
</template>
