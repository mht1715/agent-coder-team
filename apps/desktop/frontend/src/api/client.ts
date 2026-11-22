import axios, { type AxiosError, type InternalAxiosRequestConfig } from 'axios'
import { tokenStorage } from './token-storage'
import type { AuthResponse } from '../types/api'

export type HTTPMethod = 'GET' | 'POST' | 'PATCH' | 'PUT' | 'DELETE'

export interface RequestOptions {
  params?: Record<string, unknown>
}

export interface WailsRuntime {
  Call: {
    ByName<T = unknown>(methodName: string, ...args: unknown[]): Promise<T>
  }
  Events: {
    On(eventName: string, callback: (event: { data?: unknown }) => void): () => void
    Off(...eventNames: string[]): void
  }
}

interface WailsAPIResponse {
  status: number
  body?: string
}

export class ApiError extends Error {
  readonly status: number
  readonly code: string
  readonly details?: unknown

  constructor(status: number, body: unknown) {
    const payload = body as { code?: string; message?: string; details?: unknown } | undefined
    super(payload?.message || 'API request failed')
    this.name = 'ApiError'
    this.status = status
    this.code = payload?.code || 'api.error'
    this.details = payload?.details
  }
}

const http = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  timeout: 30_000,
  headers: {
    'Content-Type': 'application/json',
  },
})

http.interceptors.request.use((config: InternalAxiosRequestConfig) => {
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
  const response = await http.post<AuthResponse>('/auth/refresh', {
    refresh_token: refreshToken,
  })
  if (!response.data.tokens) {
    throw new Error('refresh response did not contain tokens')
  }
  tokenStorage.set(response.data.tokens.access_token, response.data.tokens.refresh_token)
  return response.data.tokens.access_token
}

http.interceptors.response.use(
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
      return http.request(config)
    } catch (refreshError) {
      tokenStorage.clear()
      if (!window.location.hash.startsWith('#/login')) {
        window.location.hash = '#/login'
      }
      return Promise.reject(refreshError)
    }
  },
)

let runtimePromise: Promise<WailsRuntime> | null = null
let runtimeDetected: boolean | null = null

function loadWailsRuntime(): Promise<WailsRuntime> {
  // @ts-expect-error Wails injects this module into the webview at runtime.
  runtimePromise ||= import(/* @vite-ignore */ '/wails/runtime.js').then(
    (runtime) => runtime as WailsRuntime,
  )
  return runtimePromise
}

export async function detectWailsRuntime(): Promise<boolean> {
  if (runtimeDetected !== null) return runtimeDetected
  if (typeof window !== 'undefined' && '_wails' in window) {
    runtimeDetected = true
    return true
  }
  try {
    await loadWailsRuntime()
    runtimeDetected = true
  } catch {
    runtimeDetected = false
  }
  return runtimeDetected
}

export function isWailsRuntime(): boolean {
  return runtimeDetected === true || (typeof window !== 'undefined' && '_wails' in window)
}

export async function getWailsRuntime(): Promise<WailsRuntime> {
  if (!(await detectWailsRuntime())) {
    throw new Error('Wails runtime is not available')
  }
  return loadWailsRuntime()
}

function normalizeParams(params?: Record<string, unknown>): Record<string, string> | undefined {
  if (!params) return undefined
  const result: Record<string, string> = {}
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== null && value !== '') {
      result[key] = String(value)
    }
  }
  return Object.keys(result).length > 0 ? result : undefined
}

async function wailsRequest<T>(
  method: HTTPMethod,
  path: string,
  body?: unknown,
  options?: RequestOptions,
): Promise<T> {
  const runtime = await getWailsRuntime()
  const response = await runtime.Call.ByName<WailsAPIResponse>('main.DesktopAPI.Request', {
    method,
    path,
    query: normalizeParams(options?.params),
    body: body === undefined ? '' : JSON.stringify(body),
  })

  let payload: unknown
  if (response.body) {
    payload = JSON.parse(response.body)
  }
  if (response.status >= 400) {
    throw new ApiError(response.status, payload)
  }
  return payload as T
}

async function httpRequest<T>(
  method: HTTPMethod,
  path: string,
  body?: unknown,
  options?: RequestOptions,
): Promise<T> {
  const response = await http.request<T>({
    method,
    url: path,
    data: body,
    params: options?.params,
  })
  return response.data
}

export async function apiRequest<T>(
  method: HTTPMethod,
  path: string,
  body?: unknown,
  options?: RequestOptions,
): Promise<T> {
  if (await detectWailsRuntime()) {
    return wailsRequest<T>(method, path, body, options)
  }
  return httpRequest<T>(method, path, body, options)
}

export default apiRequest
