import type { VoiceEmotionAssessment } from './voiceEmotion'

export const VOICE_LATENCY_EVENTS = [
  'mic_start',
  'speech_end',
  'asr_result',
  'answer_request',
  'answer_start',
  'answer_complete',
  'tts_first_byte',
  'audio_play_start',
  'audio_complete',
] as const

export type VoiceLatencyEvent = (typeof VOICE_LATENCY_EVENTS)[number]

export type VoiceLatencySnapshot = {
  trace_id: string
  started_at: string
  status: 'completed' | 'failed' | 'aborted'
  error_type?: string
  emotion_assessment?: VoiceEmotionAssessment
  events: Partial<Record<VoiceLatencyEvent, number>>
  durations_ms: {
    mic_to_asr?: number
    asr_to_answer_start?: number
    answer_generation?: number
    answer_to_tts_first_byte?: number
    tts_to_audio_play_start?: number
    audio_playback?: number
    voice_pipeline_total?: number
  }
}

const TRACE_STORAGE_KEY = 'sg_voice_latency_traces'
const MAX_STORED_TRACES = 50

function monotonicMs() {
  return typeof performance !== 'undefined' ? performance.now() : Date.now()
}

function uuid() {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') return crypto.randomUUID()
  return `voice-${Date.now()}-${Math.random().toString(16).slice(2)}`
}

function duration(start?: number, end?: number) {
  if (!Number.isFinite(start) || !Number.isFinite(end) || (end as number) < (start as number)) return undefined
  return Math.round((end as number) - (start as number))
}

export class VoiceLatencyTrace {
  readonly traceId = uuid()
  private readonly startedAtMs = monotonicMs()
  private readonly startedAtISO = new Date().toISOString()
  private readonly events: Partial<Record<VoiceLatencyEvent, number>> = {}
  private status: VoiceLatencySnapshot['status'] = 'completed'
  private errorType = ''
  private emotionAssessment: VoiceEmotionAssessment | undefined

  mark(event: VoiceLatencyEvent, timestamp = monotonicMs()) {
    if (!Number.isFinite(timestamp) || this.events[event] !== undefined) return
    this.events[event] = Math.max(0, timestamp - this.startedAtMs)
  }

  fail(errorType: string, status: 'failed' | 'aborted' = 'failed') {
    this.status = status
    this.errorType = errorType || 'unknown'
  }

  setEmotionAssessment(assessment: VoiceEmotionAssessment | null) {
    if (assessment) this.emotionAssessment = assessment
  }

  complete() {
    if (this.status === 'completed') return
    this.status = 'completed'
    this.errorType = ''
  }

  snapshot(): VoiceLatencySnapshot {
    const event = (name: VoiceLatencyEvent) => this.events[name]
    const durations = {
      mic_to_asr: duration(event('mic_start'), event('asr_result')),
      asr_to_answer_start: duration(event('asr_result'), event('answer_start')),
      answer_generation: duration(event('answer_request'), event('answer_complete')),
      answer_to_tts_first_byte: duration(event('answer_complete'), event('tts_first_byte')),
      tts_to_audio_play_start: duration(event('tts_first_byte'), event('audio_play_start')),
      audio_playback: duration(event('audio_play_start'), event('audio_complete')),
      voice_pipeline_total: duration(event('mic_start'), event('audio_complete')),
    }
    return {
      trace_id: this.traceId,
      started_at: this.startedAtISO,
      status: this.status,
      ...(this.errorType ? { error_type: this.errorType } : {}),
      ...(this.emotionAssessment ? { emotion_assessment: this.emotionAssessment } : {}),
      events: { ...this.events },
      durations_ms: durations,
    }
  }
}

export function persistVoiceLatencyTrace(snapshot: VoiceLatencySnapshot) {
  if (typeof window === 'undefined') return
  try {
    const stored = JSON.parse(window.localStorage.getItem(TRACE_STORAGE_KEY) || '[]')
    const traces = Array.isArray(stored) ? stored : []
    traces.push(snapshot)
    window.localStorage.setItem(TRACE_STORAGE_KEY, JSON.stringify(traces.slice(-MAX_STORED_TRACES)))
  } catch {
    // Local telemetry must never interrupt the visitor interaction.
  }
  window.dispatchEvent(new CustomEvent('scenic-guide:voice-latency', { detail: snapshot }))
}

export function exportVoiceLatencyTraces() {
  if (typeof window === 'undefined' || typeof document === 'undefined') return 0
  try {
    const raw = window.localStorage.getItem(TRACE_STORAGE_KEY) || '[]'
    const traces = JSON.parse(raw)
    if (!Array.isArray(traces) || traces.length === 0) return 0
    const blob = new Blob([JSON.stringify(traces, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `sg_voice_latency_traces_${new Date().toISOString().replace(/[:.]/g, '-')}.json`
    link.click()
    URL.revokeObjectURL(url)
    return traces.length
  } catch {
    return 0
  }
}
