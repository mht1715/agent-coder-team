import axios, { type AxiosError, type InternalAxiosRequestConfig } from 'axios'
import { tokenStorage } from './token-storage'
import type { AuthResponse } from '../types/api'

const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  timeout: 30_000,
  headers: {
    'Content-Type': 'application/json',
  },
})

api.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const token = tokenStorage.getAccess()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

interface RetryConfig extends InternalAxiosRequestConfig {
  _retry?: boolean
}

let refreshPromise: Promise<string> | null = null

async function refreshAccessToken(): Promise<string> {
  const refreshToken = tokenStorage.getRefresh()
  if (!refreshToken) {
    throw new Error('no refresh token')
  }
  const response = await axios.post<AuthResponse>(
    `${api.defaults.baseURL}/auth/refresh`,
    { refresh_token: refreshToken },
    { headers: { 'Content-Type': 'application/json' } },
  )
  tokenStorage.set(response.data.tokens.access_token, response.data.tokens.refresh_token)
  return response.data.tokens.access_token
}

api.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const config = error.config as RetryConfig | undefined
    if (!config || error.response?.status !== 401 || config._retry) {
      return Promise.reject(error)
    }
    if (config.url?.includes('/auth/login') || config.url?.includes('/auth/register')) {
      return Promise.reject(error)
    }
    config._retry = true
    try {
      refreshPromise ||= refreshAccessToken().finally(() => {
        refreshPromise = null
      })
      const token = await refreshPromise
      config.headers.Authorization = `Bearer ${token}`
      return api.request(config)
    } catch (refreshError) {
      tokenStorage.clear()
      if (!window.location.hash.startsWith('#/login')) {
        window.location.hash = '#/login'
      }
      return Promise.reject(refreshError)
    }
  },
)

export default api