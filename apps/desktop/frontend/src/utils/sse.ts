export interface SSEEvent {
  id?: string
  event?: string
  data: string
}

export interface ConnectSSEOptions {
  url: string
  token: string
  signal: AbortSignal
  onEvent: (event: SSEEvent) => void
  onError?: (error: unknown) => void
}

export async function connectSSE(options: ConnectSSEOptions): Promise<void> {
  const response = await fetch(options.url, {
    method: 'GET',
    headers: {
      Accept: 'text/event-stream',
      Authorization: `Bearer ${options.token}`,
    },
    signal: options.signal,
  })
  if (!response.ok || !response.body) {
    throw new Error(`SSE connection failed: ${response.status}`)
  }

  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''

  try {
    while (true) {
      const { value, done } = await reader.read()
      if (done) break
      buffer += decoder.decode(value, { stream: true })
      buffer = buffer.replace(/\r\n/g, '\n')

      let boundary = buffer.indexOf('\n\n')
      while (boundary >= 0) {
        const block = buffer.slice(0, boundary)
        buffer = buffer.slice(boundary + 2)
        parseBlock(block, options.onEvent)
        boundary = buffer.indexOf('\n\n')
      }
    }
  } catch (error) {
    if (!options.signal.aborted) {
      options.onError?.(error)
    }
  } finally {
    reader.releaseLock()
  }
}

function parseBlock(block: string, onEvent: (event: SSEEvent) => void) {
  if (!block || block.startsWith(':')) return

  const event: SSEEvent = { data: '' }
  const data: string[] = []
  for (const line of block.split('\n')) {
    if (!line || line.startsWith(':')) continue
    const separator = line.indexOf(':')
    const field = separator >= 0 ? line.slice(0, separator) : line
    const value = separator >= 0 ? line.slice(separator + 1).replace(/^ /, '') : ''
    if (field === 'id') event.id = value
    if (field === 'event') event.event = value
    if (field === 'data') data.push(value)
  }
  if (data.length) {
    event.data = data.join('\n')
    onEvent(event)
  }
}