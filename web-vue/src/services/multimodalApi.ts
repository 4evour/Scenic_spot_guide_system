import { apiFetch } from './api'

export interface MultimodalChatResult {
  response: string
  model: string
  modality: string
  trace_id: string
  degraded: boolean
  elapsed_ms: number
}

function uploadFieldFor(file: File) {
  if (file.type.startsWith('image/')) return 'image'
  if (file.type.startsWith('audio/')) return 'audio'
  if (file.type.startsWith('video/')) return 'video'
  throw new Error('unsupported multimodal file type')
}

export function submitMultimodalChat(file: File, message: string, sessionId: string) {
  const form = new FormData()
  form.append('message', message)
  form.append('session_id', sessionId)
  form.append(uploadFieldFor(file), file)

  return apiFetch<MultimodalChatResult>('/ai/multimodal/chat', {
    method: 'POST',
    body: form,
  })
}
