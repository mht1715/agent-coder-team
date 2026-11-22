import api from './client'
import type { Conversation, Message } from '../types/api'

export async function listProjectConversations(projectId: string): Promise<Conversation[]> {
  const { data } = await api.get<{ items: Conversation[] }>(`/projects/${projectId}/conversations`)
  return data.items
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
  const { data } = await api.get<MessagePage>(`/conversations/${conversationId}/messages`, {
    params: { before, limit },
  })
  return data
}

export async function sendMessage(
  conversationId: string,
  content: string,
  contentFormat = 'markdown',
): Promise<Message> {
  const { data } = await api.post<Message>(`/conversations/${conversationId}/messages`, {
    content,
    content_format: contentFormat,
  })
  return data
}

export function conversationStreamUrl(conversationId: string): string {
  const base = import.meta.env.VITE_API_BASE_URL || '/api/v1'
  return `${base}/conversations/${conversationId}/stream`
}