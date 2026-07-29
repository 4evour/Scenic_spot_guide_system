import assert from 'node:assert/strict'
import {
  STREAMING_VOICE_OPENINGS,
  ChatSSEDecoder,
  SerialTaskQueue,
  SpeechSegmenter,
} from '../src/services/streamingChat.ts'

const decoder = new ChatSSEDecoder()
const decodedEvents = [
  ...decoder.push('data: {"token":"灵山","done":false}\r\n\r'),
  ...decoder.push('\n: heartbeat\n\ndata: {"token":"大佛","done":false}\n\n'),
  ...decoder.push(
    'data: {"token":"","done":true,"trace_id":"trace-1","sources":[{"id":"source-1"}]}\n\n' +
      'data: [DONE]\n\n',
  ),
  ...decoder.finish(),
]

assert.deepEqual(
  decodedEvents.map((event) => ({ token: event.token, done: event.done, streamEnd: event.streamEnd })),
  [
    { token: '灵山', done: false, streamEnd: undefined },
    { token: '大佛', done: false, streamEnd: undefined },
    { token: '', done: true, streamEnd: undefined },
    { token: undefined, done: undefined, streamEnd: true },
  ],
)
assert.equal(decodedEvents[2].trace_id, 'trace-1')
assert.equal(decodedEvents[2].sources?.[0]?.id, 'source-1')

const decimalSentence = new SpeechSegmenter()
assert.deepEqual(decimalSentence.push('阿育王柱通高16.9米，'), [])
assert.deepEqual(decimalSentence.push('直径1.8米。'), ['阿育王柱通高16.9米，直径1.8米。'])
assert.deepEqual(decimalSentence.flush(), [])

const exactSentence = new SpeechSegmenter()
const exactInput = '您好！灵山大佛通高88米。祝您游览愉快'
const exactOutput = [
  ...exactSentence.push('您好！灵山'),
  ...exactSentence.push('大佛通高88'),
  ...exactSentence.push('米。祝您'),
  ...exactSentence.push('游览愉快'),
  ...exactSentence.flush(),
]
assert.equal(exactOutput.join(''), exactInput)

const longClause = new SpeechSegmenter({ maxChars: 24, minSoftBoundaryChars: 12 })
assert.deepEqual(
  longClause.push('灵山胜境建筑群沿中轴线展开，游客可以依次参观多个核心景观并了解佛教文化'),
  ['灵山胜境建筑群沿中轴线展开，'],
)
assert.equal(longClause.flush().join(''), '游客可以依次参观多个核心景观并了解佛教文化')

assert.throws(
  () => decoder.push('data: {"error":"模型服务暂时不可用"}\n\n'),
  /模型服务暂时不可用/,
)

assert.ok(STREAMING_VOICE_OPENINGS.zh.length <= 8)
assert.equal((STREAMING_VOICE_OPENINGS.zh.match(/[。！？!?]/g) || []).length, 1)

const serialQueue = new SerialTaskQueue()
const taskOrder = []
void serialQueue.enqueue(async () => {
  await new Promise((resolve) => setTimeout(resolve, 30))
  taskOrder.push(1)
})
void serialQueue.enqueue(async () => {
  await new Promise((resolve) => setTimeout(resolve, 1))
  taskOrder.push(2)
})
void serialQueue.enqueue(async () => {
  taskOrder.push(3)
})
await serialQueue.drain()
assert.deepEqual(taskOrder, [1, 2, 3])

console.log('streaming chat tests passed')
