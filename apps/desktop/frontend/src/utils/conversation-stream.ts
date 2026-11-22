import { getWailsRuntime, isWailsRuntime } from '../api/client'
import { conversationStreamUrl } from '../api/conversations'
import { tokenStorage } from '../api/token-storage'
import { connectSSE } from './sse'
import type { Message, MessageDelta } from '../types/api'

export interface ConversationStreamHandlers {
  onMessage?: (message: Message) => void
  onDelta?: (delta: MessageDelta) => void
  onThinking?: (active: boolean) => void
  onEvent?: (type: string, data: unknown) => void
  onError: (error?: unknown) => void
}

interface WailsConversationEvent {
  conversation_id: string
  event: string
  data?: string
}

export type ConversationStreamCleanup = () => Promise<void>

function dispatchEvent(
  eventName: string,
  rawData: string,
  handlers: ConversationStreamHandlers,
): void {
  if (eventName === 'stream.error') {
    handlers.onThinking?.(false)
    handlers.onError(new Error('conversation stream disconnected'))
    return
  }
  if (!rawData) return

  let data: unknown
  try {
    data = JSON.parse(rawData) as unknown
  } catch {
    handlers.onError(new Error('invalid conversation event'))
    return
  }

  switch (eventName) {
    case 'message.created':
      handlers.onMessage?.(data as Message)
      break
    case 'message.delta':
      handlers.onDelta?.(data as MessageDelta)
      break
    case 'agent.thinking':
      handlers.onThinking?.(Boolean((data as { active?: boolean }).active))
      break
    default:
      break
  }
  handlers.onEvent?.(eventName, data)
}

export async function subscribeConversationStream(
  conversationId: string,
  handlers: ConversationStreamHandlers,
): Promise<ConversationStreamCleanup> {
  if (isWailsRuntime()) {
    const runtime = await getWailsRuntime()
    const unsubscribe = runtime.Events.On('conversation.event', (event) => {
      const payload = event.data as WailsConversationEvent | undefined
      if (!payload || payload.conversation_id !== conversationId) return
      dispatchEvent(payload.event, payload.data || '', handlers)
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
      dispatchEvent(event.event || 'message', event.data || '', handlers)
    },
    onError: (error) => {
      handlers.onThinking?.(false)
      handlers.onError(error)
    },
  })

  return async () => {
    controller.abort()
  }
}
