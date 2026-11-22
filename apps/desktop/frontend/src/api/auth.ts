import api from './client'
import type { AuthResponse, User } from '../types/api'

export async function login(username: string, password: string): Promise<AuthResponse> {
  const { data } = await api.post<AuthResponse>('/auth/login', { username, password })
  return data
}

export async function register(
  username: string,
  password: string,
  displayName: string,
): Promise<AuthResponse> {
  const { data } = await api.post<AuthResponse>('/auth/register', {
    username,
    password,
    display_name: displayName,
  })
  return data
}

export async function getBootstrapStatus(): Promise<{ registration_open: boolean }> {
  const { data } = await api.get<{ registration_open: boolean }>('/auth/bootstrap')
  return data
}

export async function getMe(): Promise<User> {
  const { data } = await api.get<User>('/me')
  return data
}

export async function logout(): Promise<void> {
  await api.post('/auth/logout')
}