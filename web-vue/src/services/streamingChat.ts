import type { RAGSource } from '../types/digitalHuman'

export type ChatStreamEvent = {
  token?: string
  done?: boolean
  streamEnd?: boolean
  trace_id?: string
  sources?: RAGSource[]
  confidence?: number
  should_abstain?: boolean
  emotion?: string
  emotion_token?: string
  emotion_category?: string
  emotion_confidence?: number
  recommend_human_service?: boolean
  emotion_modality?: string
  acoustic_confidence?: number
  emotion_evidence?: string[]
  error?: string
}

export type StreamChatRequest = {
  session_id: string
  message: string
  voice_features?: unknown
  location_context?: unknown
}

export type StreamChatOptions = {
  signal?: AbortSignal
  timeoutMs?: number
  onToken?: (token: string) => void
}

export const STREAMING_VOICE_OPENINGS = {
  zh: '好的，请稍等。',
  en: 'One moment, please.',
} as const

function mergeAbortSignals(...signals: Array<AbortSignal | undefined>) {
  const activeSignals = signals.filter((signal): signal is AbortSignal => Boolean(signal))
  if (activeSignals.length <= 1) return activeSignals[0]
  if (typeof AbortSignal.any === 'function') return AbortSignal.any(activeSignals)
  const controller = new AbortController()
  const abort = () => controller.abort()
  for (const signal of activeSignals) {
    if (signal.aborted) {
      abort()
      break
    }
    signal.addEventListener('abort', abort, { once: true })
  }
  return controller.signal
}

function readCSRFToken() {
  if (typeof document === 'undefined') return ''
  const match = document.cookie.match(/(?:^|;\s*)csrf_token=([^;]*)/)
  return match ? decodeURIComponent(match[1]) : ''
}

export class ChatSSEDecoder {
  private buffer = ''

  push(chunk: string): ChatStreamEvent[] {
    this.buffer += chunk
    const events: ChatStreamEvent[] = []

    while (true) {
      const boundary = this.buffer.match(/\r?\n\r?\n/)
      if (!boundary || boundary.index === undefined) break
      const frame = this.buffer.slice(0, boundary.index)
      this.buffer = this.buffer.slice(boundary.index + boundary[0].length)
      events.push(...this.parseFrame(frame))
    }

    return events
  }

  finish(): ChatStreamEvent[] {
    const frame = this.buffer
    this.buffer = ''
    return frame.trim() ? this.parseFrame(frame) : []
  }

  private parseFrame(frame: string): ChatStreamEvent[] {
    const data = frame
      .split(/\r?\n/)
      .filter((line) => line.startsWith('data:'))
      .map((line) => line.slice(5).trimStart())
      .join('\n')
      .trim()

    if (!data) return []
    if (data === '[DONE]') return [{ streamEnd: true }]

    let event: ChatStreamEvent
    try {
      event = JSON.parse(data) as ChatStreamEvent
    } catch {
      throw new Error('模型流式响应格式错误')
    }
    if (event.error) throw new Error(event.error)
    return [event]
  }
}

export async function streamChat(
  request: StreamChatRequest,
  options: StreamChatOptions = {},
): Promise<ChatStreamEvent> {
  const csrfToken = readCSRFToken()
  const timeoutMs = options.timeoutMs ?? 60_000
  const response = await fetch('/api/v1/ai/chat', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(csrfToken ? { 'X-CSRF-Token': csrfToken } : {}),
    },
    body: JSON.stringify({ ...request, stream: true }),
    credentials: 'include',
    signal: mergeAbortSignals(options.signal, AbortSignal.timeout(timeoutMs)),
  })

  if (!response.ok) {
    const message = await response.text().catch(() => '')
    throw new Error(message || `模型请求失败 (${response.status})`)
  }
  if (!response.body) throw new Error('模型没有返回流式响应')

  const reader = response.body.getReader()
  const textDecoder = new TextDecoder()
  const sseDecoder = new ChatSSEDecoder()
  let finalEvent: ChatStreamEvent | null = null
  let streamEnded = false

  const consume = (events: ChatStreamEvent[]) => {
    for (const event of events) {
      if (event.streamEnd) {
        streamEnded = true
        continue
      }
      if (event.token) options.onToken?.(event.token)
      if (event.done) finalEvent = event
    }
  }

  while (!streamEnded) {
    const { done, value } = await reader.read()
    if (done) break
    consume(sseDecoder.push(textDecoder.decode(value, { stream: true })))
  }
  consume(sseDecoder.push(textDecoder.decode()))
  consume(sseDecoder.finish())

  if (!finalEvent) throw new Error('模型流式响应提前结束')
  return finalEvent
}

export type SpeechSegmenterOptions = {
  maxChars?: number
  minSoftBoundaryChars?: number
}

export class SpeechSegmenter {
  private buffer = ''
  private readonly maxChars: number
  private readonly minSoftBoundaryChars: number

  constructor(options: SpeechSegmenterOptions = {}) {
    this.maxChars = Math.max(16, options.maxChars ?? 48)
    this.minSoftBoundaryChars = Math.max(6, options.minSoftBoundaryChars ?? 18)
  }

  push(fragment: string): string[] {
    if (!fragment) return []
    this.buffer += fragment
    return this.drain()
  }

  flush(): string[] {
    const tail = this.buffer
    this.buffer = ''
    return tail.trim() ? [tail] : []
  }

  private drain(): string[] {
    const segments: string[] = []
    while (this.buffer) {
      const strongBoundary = this.buffer.search(/[。！？!?；;\n]/u)
      if (strongBoundary >= 0) {
        const end = strongBoundary + 1
        segments.push(this.buffer.slice(0, end))
        this.buffer = this.buffer.slice(end)
        continue
      }

      if (this.buffer.length < this.maxChars) break
      const searchArea = this.buffer.slice(0, this.maxChars)
      const softBoundary = Math.max(
        searchArea.lastIndexOf('，'),
        searchArea.lastIndexOf(','),
        searchArea.lastIndexOf('：'),
        searchArea.lastIndexOf(':'),
      )
      if (softBoundary >= this.minSoftBoundaryChars) {
        const end = softBoundary + 1
        segments.push(this.buffer.slice(0, end))
        this.buffer = this.buffer.slice(end)
        continue
      }

      if (this.buffer.length < this.maxChars * 2) break
      segments.push(this.buffer.slice(0, this.maxChars))
      this.buffer = this.buffer.slice(this.maxChars)
    }
    return segments
  }
}

export class SerialTaskQueue {
  private tail: Promise<void> = Promise.resolve()

  enqueue(task: () => Promise<void>): Promise<void> {
    const result = this.tail.then(task)
    this.tail = result.catch(() => {})
    return result
  }

  drain(): Promise<void> {
    return this.tail
  }
}
