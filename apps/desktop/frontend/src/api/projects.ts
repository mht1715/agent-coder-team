import api from './client'
import type { Project } from '../types/api'

export interface CreateProjectInput {
  name: string
  description: string
  working_dir: string
  project_mode: 'new' | 'existing'
  output_language: string
}

export async function listProjects(): Promise<Project[]> {
  const { data } = await api.get<{ items: Project[] }>('/projects')
  return data.items
}

export async function createProject(input: CreateProjectInput): Promise<Project> {
  const { data } = await api.post<Project>('/projects', input)
  return data
}

export async function getProject(id: string): Promise<Project> {
  const { data } = await api.get<Project>(`/projects/${id}`)
  return data
}