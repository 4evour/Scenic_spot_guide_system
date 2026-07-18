import assert from 'node:assert/strict'
import { assessVoiceEmotion } from '../src/services/voiceEmotion.ts'

const highEnergyFrames = Array.from({ length: 20 }, (_, index) => ({
  rms: index % 2 === 0 ? 0.2 : 0.08,
  pitchHz: index % 2 === 0 ? 250 : 170,
}))
const excited = assessVoiceEmotion(highEnergyFrames, 1800, '这个景色很特别真的很特别')
assert.equal(excited?.category, 'excitement')
assert.equal(excited?.features.sample_count, 20)
assert.ok((excited?.features.pitch_variation_hz || 0) >= 39)

const anxiousFrames = Array.from({ length: 20 }, (_, index) => ({
  rms: index < 10 ? 0.005 : 0.04,
  pitchHz: index % 2 === 0 ? 280 : 160,
}))
const anxious = assessVoiceEmotion(anxiousFrames, 4000, '我我有点担心')
assert.equal(anxious?.category, 'anxiety')

const neutralFrames = Array.from({ length: 12 }, () => ({ rms: 0.04, pitchHz: 150 }))
const neutral = assessVoiceEmotion(neutralFrames, 2400, '请介绍一下灵山大佛')
assert.equal(neutral?.category, 'neutral')
assert.equal(neutral?.features.sample_count, 12)

assert.equal(assessVoiceEmotion([{ rms: 0.1, pitchHz: 200 }], 100, '样本不足'), null)

console.log('voice emotion tests passed')
