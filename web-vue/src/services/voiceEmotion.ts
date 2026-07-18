export type VoiceEmotionCategory = 'complaint' | 'anxiety' | 'excitement' | 'neutral'

export type VoiceAcousticFeatures = {
  duration_ms: number
  sample_count: number
  rms_mean: number
  rms_peak: number
  rms_variation: number
  pause_ratio: number
  pitch_mean_hz: number
  pitch_variation_hz: number
  speech_rate_chars_per_second: number
  repetition_ratio: number
}

export type VoiceEmotionAssessment = {
  category: VoiceEmotionCategory
  confidence: number
  evidence: string[]
  features: VoiceAcousticFeatures
}

export type VoiceAcousticFrame = {
  rms: number
  pitchHz: number
}

function round(value: number, digits = 4) {
  const scale = 10 ** digits
  return Math.round(value * scale) / scale
}

function mean(values: number[]) {
  return values.length > 0 ? values.reduce((sum, value) => sum + value, 0) / values.length : 0
}

function standardDeviation(values: number[]) {
  if (values.length < 2) return 0
  const average = mean(values)
  return Math.sqrt(mean(values.map((value) => (value - average) ** 2)))
}

function repetitionRatio(transcript: string) {
  const chars = Array.from(transcript.replace(/[\s，。！？、,.!?]/g, ''))
  if (chars.length < 2) return 0
  let repeated = 0
  for (let index = 1; index < chars.length; index += 1) {
    if (chars[index] === chars[index - 1]) repeated += 1
  }
  return repeated / chars.length
}

export function assessVoiceEmotion(
  frames: VoiceAcousticFrame[],
  durationMs: number,
  transcript: string,
): VoiceEmotionAssessment | null {
  if (frames.length < 5 || durationMs < 300) return null
  const rmsValues = frames.map((frame) => frame.rms)
  const pitchValues = frames.map((frame) => frame.pitchHz).filter((value) => value > 0)
  const nonPunctuationChars = Array.from(transcript.replace(/[\s，。！？、,.!?]/g, '')).length
  const durationSeconds = Math.max(0.3, durationMs / 1000)
  const features: VoiceAcousticFeatures = {
    duration_ms: Math.round(durationMs),
    sample_count: frames.length,
    rms_mean: round(mean(rmsValues)),
    rms_peak: round(Math.max(...rmsValues)),
    rms_variation: round(standardDeviation(rmsValues)),
    pause_ratio: round(rmsValues.filter((value) => value < 0.015).length / rmsValues.length),
    pitch_mean_hz: round(mean(pitchValues), 2),
    pitch_variation_hz: round(standardDeviation(pitchValues), 2),
    speech_rate_chars_per_second: round(nonPunctuationChars / durationSeconds, 2),
    repetition_ratio: round(repetitionRatio(transcript)),
  }

  const forceful = [
    features.rms_peak >= 0.25,
    features.rms_variation >= 0.07,
    features.speech_rate_chars_per_second >= 7.5 || features.repetition_ratio >= 0.15,
  ].filter(Boolean).length
  if (forceful >= 3) {
    return { category: 'complaint', confidence: 0.68, evidence: ['voice:forceful'], features }
  }

  const anxious = [
    features.pause_ratio >= 0.4,
    features.pitch_mean_hz >= 190 && features.pitch_variation_hz >= 55,
    features.repetition_ratio >= 0.12,
  ].filter(Boolean).length
  if (anxious >= 2) {
    return { category: 'anxiety', confidence: 0.64, evidence: ['voice:tension'], features }
  }

  const excited = [
    features.rms_peak >= 0.18,
    features.speech_rate_chars_per_second >= 6,
    features.pitch_variation_hz >= 45,
    features.pause_ratio <= 0.25,
  ].filter(Boolean).length
  if (excited >= 3) {
    return { category: 'excitement', confidence: 0.66, evidence: ['voice:high_energy'], features }
  }
  return { category: 'neutral', confidence: 0.6, evidence: ['voice:neutral'], features }
}

function estimatePitch(buffer: Float32Array, sampleRate: number) {
  let energy = 0
  for (const sample of buffer) energy += sample * sample
  if (Math.sqrt(energy / buffer.length) < 0.015) return 0

  const minLag = Math.max(1, Math.floor(sampleRate / 400))
  const maxLag = Math.min(buffer.length - 2, Math.floor(sampleRate / 70))
  let bestLag = 0
  let bestCorrelation = 0
  for (let lag = minLag; lag <= maxLag; lag += 2) {
    let correlation = 0
    let leftEnergy = 0
    let rightEnergy = 0
    for (let index = 0; index < buffer.length - lag; index += 1) {
      const left = buffer[index]
      const right = buffer[index + lag]
      correlation += left * right
      leftEnergy += left * left
      rightEnergy += right * right
    }
    const normalized = correlation / Math.sqrt(Math.max(Number.EPSILON, leftEnergy * rightEnergy))
    if (normalized > bestCorrelation) {
      bestCorrelation = normalized
      bestLag = lag
    }
  }
  return bestCorrelation >= 0.45 && bestLag > 0 ? sampleRate / bestLag : 0
}

export class VoiceEmotionCapture {
  private generation = 0
  private stream: MediaStream | null = null
  private context: AudioContext | null = null
  private analyser: AnalyserNode | null = null
  private timer = 0
  private startedAt = 0
  private frames: VoiceAcousticFrame[] = []
  private buffer = new Float32Array(1024)

  async start() {
    this.cancel()
    const generation = ++this.generation
    if (!navigator.mediaDevices?.getUserMedia) return false
    const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
    if (generation !== this.generation) {
      stream.getTracks().forEach((track) => track.stop())
      return false
    }
    const context = new AudioContext()
    const analyser = context.createAnalyser()
    analyser.fftSize = 1024
    analyser.smoothingTimeConstant = 0.2
    context.createMediaStreamSource(stream).connect(analyser)
    this.stream = stream
    this.context = context
    this.analyser = analyser
    this.startedAt = performance.now()
    this.frames = []
    this.timer = window.setInterval(() => this.sample(), 100)
    return true
  }

  stop(transcript: string) {
    this.generation += 1
    const durationMs = this.startedAt > 0 ? performance.now() - this.startedAt : 0
    this.sample()
    const assessment = assessVoiceEmotion(this.frames, durationMs, transcript)
    this.cleanup()
    return assessment
  }

  cancel() {
    this.generation += 1
    this.cleanup()
  }

  private sample() {
    if (!this.analyser || !this.context) return
    this.analyser.getFloatTimeDomainData(this.buffer)
    let energy = 0
    for (const sample of this.buffer) energy += sample * sample
    const rms = Math.sqrt(energy / this.buffer.length)
    this.frames.push({ rms, pitchHz: estimatePitch(this.buffer, this.context.sampleRate) })
  }

  private cleanup() {
    if (this.timer) window.clearInterval(this.timer)
    this.timer = 0
    this.stream?.getTracks().forEach((track) => track.stop())
    this.stream = null
    if (this.context) void this.context.close()
    this.context = null
    this.analyser = null
    this.startedAt = 0
  }
}
