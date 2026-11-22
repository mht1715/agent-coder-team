import { apiRequest } from './client'
import type { WorkspaceBrowseResult } from '../types/api'

export async function browseWorkspace(path?: string): Promise<WorkspaceBrowseResult> {
  return apiRequest<WorkspaceBrowseResult>('GET', '/workspaces/browse', undefined, {
    params: path ? { path } : undefined,
  })
}
