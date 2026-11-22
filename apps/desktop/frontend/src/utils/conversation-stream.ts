import { getWailsRuntime, isWailsRuntime } from '../api/client'
import { conversationStreamUrl } from '../api/conversations'
import { tokenStorage } from '../api/token-storage'
import { connectSSE } from './sse'
import type { Message } from '../types/api'

export interface ConversationStreamHandlers {
  onMessage: (message: Message) => void
  onError: (error?: unknown) => void
}

interface WailsConversationEvent {
  conversation_id: string
  event: string
  data?: string
}

export type ConversationStreamCleanup = () => Promise<void>

export async function subscribeConversationStream(
  conversationId: string,
  handlers: ConversationStreamHandlers,
): Promise<ConversationStreamCleanup> {
  if (isWailsRuntime()) {
    const runtime = await getWailsRuntime()
    const unsubscribe = runtime.Events.On('conversation.event', (event) => {
      const payload = event.data as WailsConversationEvent | undefined
      if (!payload || payload.conversation_id !== conversationId) return
      if (payload.event === 'message.created' && payload.data) {
        try {
          handlers.onMessage(JSON.parse(payload.data) as Message)
        } catch {
          handlers.onError(new Error('invalid conversation event'))
        }
      }
      if (payload.event === 'stream.error') {
        handlers.onError(new Error('conversation stream disconnected'))
      }
    })

    try {
      await runtime.Call.ByName('main.DesktopAPI.WatchConversation', conversationId)
    } catch (error) {
      unsubscribe()
      throw error
    }

    return async () => {
      unsubscribe()
      try {
        await runtime.Call.ByName('main.DesktopAPI.UnwatchConversation', conversationId)
      } catch {
        // The app may already be shutting down.
      }
    }
  }

  const token = tokenStorage.getAccess()
  if (!token) {
    throw new Error('missing access token for SSE')
  }

  const controller = new AbortController()
  void connectSSE({
    url: conversationStreamUrl(conversationId),
    token,
    signal: controller.signal,
    onEvent: (event) => {
      if (event.event !== 'message.created') return
      try {
        handlers.onMessage(JSON.parse(event.data) as Message)
      } catch {
        handlers.onError(new Error('invalid conversation event'))
      }
    },
    onError: handlers.onError,
  })

  return async () => {
    controller.abort()
  }
}
