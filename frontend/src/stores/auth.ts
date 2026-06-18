import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '@/api'

interface User {
  id: number
  username: string
  role: string
  must_change_password?: boolean
  permissions?: string
  disk_quota_mb?: number
  disk_used_mb?: number
  totp_enabled?: boolean
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || '')
  const user = ref<User | null>(null)

  async function login(username: string, password: string) {
    const res: any = await api.post('/auth/login', { username, password })
    if (res.data?.require_totp) {
      return { requireTotp: true, tempToken: res.data.temp_token as string }
    }
    token.value = res.data.token
    user.value = res.data.user
    localStorage.setItem('token', res.data.token)
    return { requireTotp: false }
  }

  async function loginTotp(tempToken: string, code: string) {
    const res: any = await api.post('/auth/totp/login', { temp_token: tempToken, code })
    token.value = res.data.token
    user.value = res.data.user
    localStorage.setItem('token', res.data.token)
  }

  async function fetchMe() {
    const res: any = await api.get('/auth/me')
    user.value = res.data
  }

  function logout() {
    token.value = ''
    user.value = null
    localStorage.removeItem('token')
  }

  return { token, user, login, loginTotp, fetchMe, logout }
})
