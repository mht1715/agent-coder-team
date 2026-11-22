import api from './client'
import type { WorkspaceBrowseResult } from '../types/api'

export async function browseWorkspace(path?: string): Promise<WorkspaceBrowseResult> {
  const { data } = await api.get<WorkspaceBrowseResult>('/workspaces/browse', {
    params: path ? { path } : undefined,
  })
  return data
}
