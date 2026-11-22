import api from './client'
import type { ProjectProfile } from '../types/api'

export async function getProjectProfile(projectId: string): Promise<ProjectProfile> {
  const { data } = await api.get<ProjectProfile>(`/projects/${projectId}/profile`)
  return data
}

export async function scanProjectProfile(projectId: string): Promise<ProjectProfile> {
  const { data } = await api.post<ProjectProfile>(`/projects/${projectId}/profile/scan`)
  return data
}