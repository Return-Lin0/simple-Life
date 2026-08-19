// 认证状态：Access Token 存内存，用户信息存 Pinia
import { defineStore } from 'pinia'
import { authApi } from '@/api/modules'
import { refreshSession } from '@/api/client'
import { clearAccessToken, getAccessToken, setAccessToken } from '@/utils/token'
import type { UserInfo } from '@/types'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    user: null as UserInfo | null,
    ready: false, // 启动时是否已完成会话探测
  }),
  getters: {
    authenticated: (s) => s.user !== null,
  },
  actions: {
    getToken() {
      return getAccessToken()
    },
    async login(username: string, password: string) {
      const resp = await authApi.login({ username, password })
      setAccessToken(resp.access_token)
      this.user = resp.user
      this.ready = true
    },
    async register(username: string, password: string, nickname: string) {
      return authApi.register({ username, password, nickname })
    },
    async logout() {
      try {
        await authApi.logout()
      } catch {
        // 登出失败不阻塞本地清理
      }
      this.clearSession()
    },
    // 启动时探测会话：Refresh Token 有效则静默续期
    async bootstrap() {
      if (this.ready) return
      const ok = await refreshSession()
      if (ok) {
        try {
          this.user = await authApi.me()
        } catch {
          this.user = null
        }
      }
      this.ready = true
    },
    clearSession() {
      clearAccessToken()
      this.user = null
    },
  },
})
