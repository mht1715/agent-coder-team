import { apiRequest } from './client'
import type { ProjectProfile } from '../types/api'

export async function getProjectProfile(projectId: string): Promise<ProjectProfile> {
  return apiRequest<ProjectProfile>('GET', `/projects/${projectId}/profile`)
}

export async function scanProjectProfile(projectId: string): Promise<ProjectProfile> {
  return apiRequest<ProjectProfile>('POST', `/projects/${projectId}/profile/scan`)
}
