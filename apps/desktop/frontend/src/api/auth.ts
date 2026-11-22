import { apiRequest } from './client'
import type { AuthResponse, User } from '../types/api'

export async function login(username: string, password: string): Promise<AuthResponse> {
  return apiRequest<AuthResponse>('POST', '/auth/login', { username, password })
}

export async function register(
  username: string,
  password: string,
  displayName: string,
): Promise<AuthResponse> {
  return apiRequest<AuthResponse>('POST', '/auth/register', {
    username,
    password,
    display_name: displayName,
  })
}

export async function getBootstrapStatus(): Promise<{ registration_open: boolean }> {
  return apiRequest<{ registration_open: boolean }>('GET', '/auth/bootstrap')
}

export async function getMe(): Promise<User> {
  return apiRequest<User>('GET', '/me')
}

export async function logout(): Promise<void> {
  await apiRequest<void>('POST', '/auth/logout')
}
