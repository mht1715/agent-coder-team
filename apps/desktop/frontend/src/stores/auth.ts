import { defineStore } from 'pinia'
import type { User } from '../types/api'
import { getMe, login as loginApi, logout as logoutApi, register as registerApi } from '../api/auth'
import { tokenStorage } from '../api/token-storage'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    user: null as User | null,
    authenticated: false,
    initialized: false,
  }),
  getters: {
    isAuthenticated: (state) => state.authenticated,
    displayName: (state) => state.user?.display_name || state.user?.username || '',
  },
  actions: {
    async login(username: string, password: string) {
      const response = await loginApi(username, password)
      if (response.tokens) {
        tokenStorage.set(response.tokens.access_token, response.tokens.refresh_token)
      }
      this.user = response.user
      this.authenticated = true
      this.initialized = true
    },
    async register(username: string, password: string, displayName: string) {
      const response = await registerApi(username, password, displayName)
      if (response.tokens) {
        tokenStorage.set(response.tokens.access_token, response.tokens.refresh_token)
      }
      this.user = response.user
      this.authenticated = true
      this.initialized = true
    },
    async loadMe() {
      try {
        this.user = await getMe()
        this.authenticated = true
      } catch {
        tokenStorage.clear()
        this.user = null
        this.authenticated = false
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
        this.authenticated = false
        this.initialized = true
      }
    },
  },
})
