export interface User {
  id: string
  username: string
  display_name: string
  status: string
  created_at: string
}

export interface TokenPair {
  access_token: string
  refresh_token: string
  access_expires_at: string
  refresh_expires_at: string
}

export interface AuthResponse {
  user: User
  tokens?: TokenPair
}

export interface Project {
  id: string
  name: string
  description: string
  working_dir: string
  project_mode: 'new' | 'existing'
  status: string
  output_language: string
  created_at: string
  updated_at: string
}

export interface Conversation {
  id: string
  project_id: string
  type: string
  title: string
  last_message_at?: string
  created_at: string
  updated_at: string
}

export interface Message {
  id: string
  conversation_id: string
  sender_type: 'user' | 'main_agent' | 'worker_agent' | 'system'
  sender_id?: string
  content: string
  content_format: string
  summary?: string
  metadata?: Record<string, unknown>
  created_at: string
}

export interface ProjectProfile {
  project_id: string
  primary_language: string
  languages: string[]
  frameworks: string[]
  package_manager: string
  build_command: string
  test_command: string
  format_command: string
  file_count: number
  top_level: string[]
  truncated: boolean
  detected_at: string
  updated_at: string
}
export interface WorkspaceDirectoryEntry {
  name: string
  path: string
}

export interface WorkspaceBrowseResult {
  path: string
  parent?: string
  root: string
  roots: string[]
  entries: WorkspaceDirectoryEntry[]
  truncated: boolean
}

export interface RequirementDocument {
  id: string
  project_id: string
  title: string
  content_md: string
  status: 'draft' | 'approved'
  version: number
  generated_by: string
  approved_at?: string
  created_at: string
  updated_at: string
}

export type TaskStatus =
  | 'backlog'
  | 'ready'
  | 'in_progress'
  | 'review'
  | 'done'
  | 'blocked'
  | 'cancelled'

export interface Task {
  id: string
  project_id: string
  task_key: string
  title: string
  description: string
  acceptance_criteria: string
  agent_role: string
  priority: string
  status: TaskStatus
  position: number
  dependencies: string[]
  started_at?: string
  completed_at?: string
  created_at: string
  updated_at: string
}

export interface Approval {
  id: string
  project_id: string
  task_id?: string
  run_id?: string
  kind: 'plan_review' | 'task_execution'
  status: 'pending' | 'approved' | 'rejected'
  title: string
  description: string
  requested_by_type: string
  decided_by?: string
  decision_note?: string
  decided_at?: string
  created_at: string
  updated_at: string
}

export interface RunLog {
  at: string
  level: string
  message: string
}

export interface AgentRun {
  id: string
  project_id: string
  task_id?: string
  conversation_id?: string
  kind: 'requirement' | 'planning' | 'execution'
  status: 'queued' | 'running' | 'succeeded' | 'failed' | 'cancelled'
  title: string
  input_summary?: string
  output_summary?: string
  logs: RunLog[]
  started_at?: string
  finished_at?: string
  created_at: string
  updated_at: string
}

export interface MessageDelta {
  message_id: string
  conversation_id: string
  sender_type: Message['sender_type']
  delta: string
  created_at: string
}
