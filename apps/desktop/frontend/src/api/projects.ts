import { apiRequest } from './client'
import type { Project } from '../types/api'

export interface CreateProjectInput {
  name: string
  description: string
  working_dir: string
  project_mode: 'new' | 'existing'
  output_language: string
}

export async function listProjects(): Promise<Project[]> {
  const response = await apiRequest<{ items: Project[] }>('GET', '/projects')
  return response.items
}

export async function createProject(input: CreateProjectInput): Promise<Project> {
  return apiRequest<Project>('POST', '/projects', input)
}

export async function getProject(id: string): Promise<Project> {
  return apiRequest<Project>('GET', `/projects/${id}`)
}
