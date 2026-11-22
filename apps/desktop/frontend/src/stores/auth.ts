import { defineStore } from 'pinia'
import type { User } from '../types/api'
import { getMe, login as loginApi, logout as logoutApi, register as registerApi } from '../api/auth'
import { tokenStorage } from '../api/token-storage'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    user: null as User | null,
    initialized: false,
  }),
  getters: {
    isAuthenticated: () => Boolean(tokenStorage.getAccess()),
    displayName: (state) => state.user?.display_name || state.user?.username || '',
  },
  actions: {
    async login(username: string, password: string) {
      const response = await loginApi(username, password)
      tokenStorage.set(response.tokens.access_token, response.tokens.refresh_token)
      this.user = response.user
      this.initialized = true
    },
    async register(username: string, password: string, displayName: string) {
      const response = await registerApi(username, password, displayName)
      tokenStorage.set(response.tokens.access_token, response.tokens.refresh_token)
      this.user = response.user
      this.initialized = true
    },
    async loadMe() {
      if (!tokenStorage.getAccess()) {
        this.user = null
        this.initialized = true
        return
      }
      try {
        this.user = await getMe()
      } catch {
        tokenStorage.clear()
        this.user = null
      } finally {
        this.initialized = true
      }
    },
    async logout() {
      try {
        await logoutApi()
      } finally {
        tokenStorage.clear()
        this.user = null
        this.initialized = true
      }
    },
  },
})