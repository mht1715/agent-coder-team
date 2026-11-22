import { apiRequest } from './client'
import type { Conversation, Message } from '../types/api'

export async function listProjectConversations(projectId: string): Promise<Conversation[]> {
  const response = await apiRequest<{ items: Conversation[] }>(
    'GET',
    `/projects/${projectId}/conversations`,
  )
  return response.items
}

export interface MessagePage {
  items: Message[]
  has_more: boolean
  next_before?: string
}

export async function listMessages(
  conversationId: string,
  before?: string,
  limit = 50,
): Promise<MessagePage> {
  return apiRequest<MessagePage>('GET', `/conversations/${conversationId}/messages`, undefined, {
    params: { before, limit },
  })
}

export async function sendMessage(
  conversationId: string,
  content: string,
  contentFormat = 'markdown',
): Promise<Message> {
  return apiRequest<Message>('POST', `/conversations/${conversationId}/messages`, {
    content,
    content_format: contentFormat,
  })
}

export function conversationStreamUrl(conversationId: string): string {
  const base = import.meta.env.VITE_API_BASE_URL || '/api/v1'
  return `${base}/conversations/${conversationId}/stream`
}
