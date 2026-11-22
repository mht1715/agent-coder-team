import { apiRequest } from './client'
import type { AgentRun, Approval, RequirementDocument, Task, TaskStatus } from '../types/api'

export async function getRequirement(projectId: string): Promise<RequirementDocument> {
  return apiRequest<RequirementDocument>('GET', `/projects/${projectId}/requirement`)
}

export async function generateRequirement(projectId: string): Promise<RequirementDocument> {
  return apiRequest<RequirementDocument>('POST', `/projects/${projectId}/requirement/generate`)
}

export async function updateRequirement(
  projectId: string,
  title: string,
  contentMD: string,
): Promise<RequirementDocument> {
  return apiRequest<RequirementDocument>('PATCH', `/projects/${projectId}/requirement`, {
    title,
    content_md: contentMD,
  })
}

export async function approveRequirement(projectId: string): Promise<RequirementDocument> {
  return apiRequest<RequirementDocument>('POST', `/projects/${projectId}/requirement/approve`)
}

export async function listTasks(projectId: string): Promise<Task[]> {
  const response = await apiRequest<{ items: Task[] }>('GET', `/projects/${projectId}/tasks`)
  return response.items
}

export interface GeneratePlanResult {
  tasks: Task[]
  approval?: Approval
}

export async function generatePlan(projectId: string, replace = false): Promise<GeneratePlanResult> {
  return apiRequest<GeneratePlanResult>('POST', `/projects/${projectId}/tasks/generate`, { replace })
}

export async function transitionTask(
  taskId: string,
  status: TaskStatus,
  note = '',
): Promise<Task> {
  return apiRequest<Task>('POST', `/tasks/${taskId}/transition`, { status, note })
}

export async function requestTaskExecution(
  taskId: string,
  command = '',
  image = '',
): Promise<Approval> {
  return apiRequest<Approval>('POST', `/tasks/${taskId}/request-execution`, { command, image })
}

export async function listApprovals(
  projectId: string,
  status?: 'pending' | 'approved' | 'rejected',
): Promise<Approval[]> {
  const response = await apiRequest<{ items: Approval[] }>(
    'GET',
    `/projects/${projectId}/approvals`,
    undefined,
    { params: { status } },
  )
  return response.items
}

export async function decideApproval(
  approvalId: string,
  decision: 'approve' | 'reject',
  note = '',
): Promise<Approval> {
  return apiRequest<Approval>('POST', `/approvals/${approvalId}/decision`, { decision, note })
}

export async function listRuns(projectId: string): Promise<AgentRun[]> {
  const response = await apiRequest<{ items: AgentRun[] }>('GET', `/projects/${projectId}/runs`)
  return response.items
}

export async function finishRun(
  runId: string,
  status: 'succeeded' | 'failed' | 'cancelled',
  outputSummary = '',
): Promise<AgentRun> {
  return apiRequest<AgentRun>('POST', `/runs/${runId}/finish`, {
    status,
    output_summary: outputSummary,
  })
}
