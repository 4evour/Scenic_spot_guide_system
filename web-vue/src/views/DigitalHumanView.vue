<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, onErrorCaptured, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { useGeolocation, type GeolocationPosition } from '../composables/useGeolocation'
import { useProximityGuide, type SpotWithCoords } from '../composables/useProximityGuide'
import { selectClosestEligibleSpot } from '../utils/geolocation'
import { useSeniorMode } from '../composables/useSeniorMode'
import { AudioPlaybackController, type PlaybackCue } from '../services/audioPlayback'
import {
  VoiceLatencyTrace,
  exportVoiceLatencyTraces,
  persistVoiceLatencyTrace,
} from '../services/voiceTelemetry'
import {
  VoiceEmotionCapture,
  type VoiceAcousticFeatures,
  type VoiceEmotionAssessment,
} from '../services/voiceEmotion'
import { submitMultimodalChat } from '../services/multimodalApi'
import { VtuberSocketClient } from '../services/vtuberSocket'
import { streamTTS, synthesizeTTS } from '../services/ttsApi'
import {
  STREAMING_VOICE_OPENINGS,
  SerialTaskQueue,
  SpeechSegmenter,
  streamChat,
} from '../services/streamingChat'
import { apiFetch, visitorExperienceApi, type RecommendedRoute } from '../services/api'
import type {
  ChatMessage,
  ConversationState,
  EmotionToken,
  VtuberMessage,
  Live2DExpression,
  DigitalHumanAvatarOption,
  RAGSource,
} from '../types/digitalHuman'
import Live2DStage from '../components/Live2DStage.vue'
import MarkdownRenderer from '../components/MarkdownRenderer.vue'
import MultimodalInput from '../components/MultimodalInput.vue'
import { useSessionStore } from '../stores/session'
import { useAuthStore } from '../stores/auth'
import { getCSRFToken } from '../utils/csrf'
import { buildGuideInsight, type GuideInsight } from '../constants/scenicVisualization'

const { t, locale } = useI18n()
const geolocationMessages = {
  notSupported: () => t('map.gpsNotSupported'),
  denied: () => t('map.gpsDenied'),
  unavailable: () => t('map.gpsUnavailable'),
  timeout: () => t('map.gpsTimeout'),
  failed: (message: string) => t('map.gpsFailed', { message }),
}
const route = useRoute()
const authStore = useAuthStore()
const DEFAULT_AVATAR_ID = 'mao_pro'
const AUTO_GUIDE_KEY = 'sg_auto_geofence_enabled'
const QR_INTRO_STORAGE_KEY = 'sg_qr_intro_payload'
const autoGuideEnabled = ref(localStorage.getItem(AUTO_GUIDE_KEY) === 'true')
const audioUnlocked = ref(false)
const demoPosition = ref<GeolocationPosition | null>(null)
const geofenceSpots = ref<SpotWithCoords[]>([])
type VisitorProfileId = 'family' | 'culture' | 'senior' | 'photo'

interface VisitorRatingSpot {
  id: number
  name: string
  category: string
}

const visitorRatingSpots = ref<VisitorRatingSpot[]>([])
const { seniorModeEnabled, ttsRate, toggleSeniorMode } = useSeniorMode()
const {
  currentPosition,
  error: geoError,
  startWatch,
  stopWatch,
} = useGeolocation({
  enableHighAccuracy: true,
  maximumAge: 5000,
  timeout: 10000,
  messages: geolocationMessages,
})
const activePosition = computed(() => demoPosition.value || currentPosition.value)
const canTriggerAutoGuide = computed(() => autoGuideEnabled.value && audioUnlocked.value)
const locationMode = computed(() => {
  if (demoPosition.value) return 'demo'
  if (currentPosition.value) return 'gps'
  return 'idle'
})
const locationAccuracy = computed(() => demoPosition.value?.accuracy ?? currentPosition.value?.accuracy ?? null)
const locationStatus = computed(() => {
  if (locationMode.value === 'demo') return t('dh.controls.location.demoMode')
  if (currentPosition.value && currentPosition.value.accuracy <= 10) {
    return t('dh.controls.location.ready', { accuracy: Math.round(currentPosition.value.accuracy) })
  }
  if (currentPosition.value) {
    return t('dh.controls.location.waitingAccuracy', { accuracy: Math.round(currentPosition.value.accuracy) })
  }
  if (geoError.value) return geoError.value
  return t('dh.controls.location.idle')
})
const { nearbySpot, resetTriggered, setSpots } = useProximityGuide(activePosition, {
  triggerRadiusM: 100,
  maxAccuracyM: 10,
  canTrigger: canTriggerAutoGuide,
})

function currentLocationContextForQuestion(text: string) {
  if (!/(我在哪|当前位置|现在在哪|附近|离我最近|离我多远|到哪个景点|哪个景点附近)/.test(text)) return null
  const position = activePosition.value
  if (!position) return null
  const closest = selectClosestEligibleSpot(position, geofenceSpots.value, {
    maxAccuracyM: 10,
    defaultRadiusM: 300,
  })
  if (!closest) return null
  return {
    spot_name: closest.spot.name,
    distance_meters: Math.round(closest.distanceMeters),
    accuracy_meters: position.accuracy,
  }
}

// Guest upgrade state
const showUpgradeModal = ref(false)
const upgradeForm = reactive({ username: '', password: '', email: '' })
const upgradeLoading = ref(false)
const upgradeError = ref('')

function isPasswordPolicyValid(password: string) {
  return (
    password.length >= 8 &&
    password.length <= 128 &&
    /[A-Z]/.test(password) &&
    /[a-z]/.test(password) &&
    /\d/.test(password)
  )
}

async function handleUpgrade() {
  if (!upgradeForm.username || !upgradeForm.password) {
    upgradeError.value = t('auth.usernamePasswordRequired')
    return
  }
  if (!isPasswordPolicyValid(upgradeForm.password)) {
    upgradeError.value = t('auth.passwordPolicy')
    return
  }
  upgradeLoading.value = true
  upgradeError.value = ''
  try {
    const ok = await authStore.upgradeAccount(
      upgradeForm.username,
      upgradeForm.password,
      upgradeForm.email || undefined,
    )
    if (ok) {
      showUpgradeModal.value = false
      upgradeForm.username = ''
      upgradeForm.password = ''
      upgradeForm.email = ''
    } else {
      upgradeError.value = t('auth.upgradeFailed')
    }
  } catch {
    upgradeError.value = t('auth.upgradeFailed')
  } finally {
    upgradeLoading.value = false
  }
}

const uid = (): string =>
  typeof crypto.randomUUID === 'function'
    ? crypto.randomUUID()
    : 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
        const r = (Math.random() * 16) | 0
        return (c === 'x' ? r : (r & 0x3) | 0x8).toString(16)
      })

// 会话持久化
const sessionStore = useSessionStore()
const DH_SESSION_KEY = 'sg_dh_session_id'

function getOrCreateSessionId(): string {
  let sid = localStorage.getItem(DH_SESSION_KEY)
  if (!sid) {
    sid = uid()
    localStorage.setItem(DH_SESSION_KEY, sid)
  }
  return sid
}

const input = ref('')
const multimodalOpen = ref(false)
const multimodalLoading = ref(false)
const multimodalError = ref('')
const mouthOpen = ref(0)
const transcriptBuffer = ref('')
const topicEntities = ref<string[]>([])
const hasActiveTurn = ref(false)
const isPlaybackActive = ref(false)
const isVoiceListening = ref(false)
const isVoiceStarting = ref(false)
const searchQuery = ref('')
const showSearch = ref(false)
const showSessionDrawer = ref(false)
const isSearching = ref(false)
const searchResults = ref<
  Array<{
    id: string
    text: string
    time: string
    role?: ChatMessage['role']
    sessionId?: string
    sessionTitle?: string
  }>
>([])
const currentInsight = ref<GuideInsight | null>(null)
const typewriterStreaming = ref(false)
const mobileTab = ref<'avatar' | 'chat'>('avatar')
const isMobileView = ref(window.innerWidth < 768)
const visitorLoopExpanded = ref(window.innerWidth >= 768)
const avatarOptions = ref<DigitalHumanAvatarOption[]>([])
const selectedAvatarId = ref(DEFAULT_AVATAR_ID)
const avatarSaving = ref(false)
type DigitalHumanRuntimeConfig = {
  voice_id: string
  tts_rate: string
  volume: number
  default_emotion: string
  emotion_level: number
  default_avatar_id: string
  allow_avatar_switch: boolean
  runtime_version?: string
}
const runtimeConfig = reactive<DigitalHumanRuntimeConfig>({
  voice_id: 'female_xiaoxiao',
  tts_rate: '-20%',
  volume: 80,
  default_emotion: 'joy',
  emotion_level: 3,
  default_avatar_id: DEFAULT_AVATAR_ID,
  allow_avatar_switch: true,
})
const audioStatus = ref<'locked' | 'ready' | 'playing' | 'error'>('locked')
const audioNotice = ref(t('dh.audio.initialNotice'))
const storedChatWidth = Number(localStorage.getItem('sg_dh_chat_width') || 420)
const chatWidth = ref(Number.isFinite(storedChatWidth) ? storedChatWidth : 420)
const isChatResizing = ref(false)
function onWindowResize() {
  const mobile = window.innerWidth < 768
  if (mobile !== isMobileView.value) visitorLoopExpanded.value = !mobile
  isMobileView.value = mobile
}

const selectedVisitorProfile = ref<VisitorProfileId>('family')
const recommendedRoutes = ref<RecommendedRoute[]>([])
const routeRecommendationLoading = ref(false)
const routeRecommendationError = ref('')
const selectedRatingSpotId = ref<number | null>(null)
const spotRatingScore = ref(5)
const spotRatingComment = ref('')
const spotRatingLoading = ref(false)
const spotRatingMessage = ref('')

const state = reactive({
  conversation: 'idle' as ConversationState,
  expression: 'happy' as Live2DExpression,
  subtitle: t('dh.greeting'),
  connected: false,
  interruptCount: 0,
  messages: [
    {
      id: uid(),
      role: 'assistant',
      text: t('dh.greeting'),
      time: formatTime(),
    },
  ] as ChatMessage[],
})

let socket: VtuberSocketClient | null = null
let recognition: SpeechRecognition | null = null
let voiceLatencyTrace: VoiceLatencyTrace | null = null
const voiceEmotionCapture = new VoiceEmotionCapture()
let pendingVoiceEmotion: VoiceEmotionAssessment | null = null
let activeChatAbortController: AbortController | null = null
let activeTTSAbortController: AbortController | null = null
let conversationTurn = 0
let interruptedTurn = -1
let fallbackTimer = 0
const STREAMING_TTS_TIMEOUT_MS = 3200
let blockIncomingPlayback = false
let waitingForFreshServerTurn = false
let backendFallbackActive = false
let pendingSocketQuestion: { text: string; turn: number; voiceFeatures?: VoiceAcousticFeatures } | null = null
let lastAssistantSpeechText = ''
let activeAssistantMessageId = ''
let activeAssistantText = ''
let searchDebounceTimer = 0
let lastAudioNotice = ''

const audio = new AudioPlaybackController({
  onStart: (text: string | undefined, cue: PlaybackCue | undefined) => {
    voiceLatencyTrace?.mark('audio_play_start')
    if (cue?.speechKind === 'opening') {
      voiceLatencyTrace?.mark('opening_audio_play_start')
    } else {
      voiceLatencyTrace?.mark('answer_audio_play_start')
    }
    audioStatus.value = 'playing'
    audioNotice.value = t('dh.audio.playingNotice')
    isPlaybackActive.value = true
    hasActiveTurn.value = true
    typewriterStreaming.value = true
    if (text && cue?.showText !== false) showAssistantSpeech(text)
    state.conversation = 'speaking'
    state.expression = resolveSpeechExpression(cue?.expression, text)
  },
  onEnd: () => {
    if (audioStatus.value === 'playing') {
      audioStatus.value = 'ready'
      audioNotice.value = t('dh.audio.readyNotice')
    }
    isPlaybackActive.value = false
    mouthOpen.value = 0
    typewriterStreaming.value = false
    if (state.conversation === 'speaking') {
      state.conversation = 'idle'
      state.expression = 'neutral'
    }
  },
  onComplete: () => {
    voiceLatencyTrace?.mark('audio_complete')
    finishVoiceLatencyTrace()
  },
  onFirstByte: (_text: string | undefined, cue: PlaybackCue | undefined) => {
    if (cue?.speechKind !== 'opening') voiceLatencyTrace?.mark('tts_first_byte')
  },
  onVolume: (volume: number) => {
    mouthOpen.value = volume
  },
  onError: (message: string) => {
    finishVoiceLatencyTrace('failed', 'audio_playback_error')
    showAudioNotice('error', message)
  },
})

const statusLabel = computed(() => {
  if (state.conversation === 'connecting') return t('dh.connecting')
  if (state.conversation === 'thinking') return t('dh.searching')
  if (state.conversation === 'speaking') return t('dh.speaking')
  if (state.conversation === 'listening') return t('dh.listening')
  if (state.conversation === 'interrupted') return t('dh.interruptedStatus')
  if (state.conversation === 'error') return t('dh.error')
  return state.connected ? t('dh.online') : t('dh.offline')
})

const selectedAvatar = computed(() => {
  return (
    avatarOptions.value.find((item) => item.id === selectedAvatarId.value) ||
    avatarOptions.value.find((item) => item.id === runtimeConfig.default_avatar_id) ||
    avatarOptions.value.find((item) => item.id === DEFAULT_AVATAR_ID) ||
    null
  )
})

const canInterrupt = computed(
  () =>
    hasActiveTurn.value ||
    isPlaybackActive.value ||
    isVoiceListening.value ||
    isVoiceStarting.value ||
    ['speaking', 'thinking', 'listening'].includes(state.conversation),
)

const chatPanelStyle = computed(() => {
  if (isMobileView.value) return {}
  const width = Math.min(620, Math.max(320, chatWidth.value))
  return { width: `${width}px` }
})

const visitorProfiles = computed(() => [
  {
    id: 'family' as const,
    label: t('dh.visitorLoop.profiles.family'),
    hint: t('dh.visitorLoop.profileHints.family'),
    tags: ['亲子', '文化'],
    difficulty: 'easy',
  },
  {
    id: 'culture' as const,
    label: t('dh.visitorLoop.profiles.culture'),
    hint: t('dh.visitorLoop.profileHints.culture'),
    tags: ['文化', '历史'],
    difficulty: '',
  },
  {
    id: 'senior' as const,
    label: t('dh.visitorLoop.profiles.senior'),
    hint: t('dh.visitorLoop.profileHints.senior'),
    tags: ['轻松', '老人'],
    difficulty: 'easy',
  },
  {
    id: 'photo' as const,
    label: t('dh.visitorLoop.profiles.photo'),
    hint: t('dh.visitorLoop.profileHints.photo'),
    tags: ['拍照', '打卡'],
    difficulty: '',
  },
])

const selectedVisitorProfileConfig = computed(
  () =>
    visitorProfiles.value.find((item) => item.id === selectedVisitorProfile.value) ||
    visitorProfiles.value[0],
)
const ratingScoreOptions = [1, 2, 3, 4, 5]
const selectedRatingSpot = computed(
  () => visitorRatingSpots.value.find((spot) => spot.id === selectedRatingSpotId.value) || null,
)

function nowTime() {
  return formatTime()
}

function formatTime(value: Date | number | string = new Date()) {
  return new Date(value).toLocaleTimeString(locale.value, { hour: '2-digit', minute: '2-digit' })
}

function formatSearchTime(value: Date | number | string) {
  return new Date(value).toLocaleString(locale.value, {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function showAudioNotice(status: typeof audioStatus.value, message: string) {
  audioStatus.value = status
  audioNotice.value = message
  if (message && message !== lastAudioNotice) {
    lastAudioNotice = message
    addMessage('system', message)
  }
}

const ALL_EMOTION_TOKENS = ['neutral', 'joy', 'sadness', 'surprise', 'anger', 'fear', 'disgust']

type QRIntroPayload = {
  spot?: string
  intro?: string
}

function readQRIntroPayload(): QRIntroPayload {
  try {
    const raw = sessionStorage.getItem(QR_INTRO_STORAGE_KEY)
    if (raw) {
      sessionStorage.removeItem(QR_INTRO_STORAGE_KEY)
      const parsed = JSON.parse(raw) as QRIntroPayload
      return {
        spot: typeof parsed.spot === 'string' ? parsed.spot.trim() : '',
        intro: typeof parsed.intro === 'string' ? parsed.intro.trim() : '',
      }
    }
  } catch {
    sessionStorage.removeItem(QR_INTRO_STORAGE_KEY)
  }
  return {
    spot: typeof route.query.qr_spot === 'string' ? route.query.qr_spot.trim() : '',
    intro: typeof route.query.qr_intro === 'string' ? route.query.qr_intro.trim() : '',
  }
}

type AvatarPreferenceResponse = {
  avatar_id?: string
}

function isKnownAvatar(id: string) {
  return avatarOptions.value.some((item) => item.id === id)
}

async function loadRuntimeConfig() {
  try {
    const data = await apiFetch<Partial<DigitalHumanRuntimeConfig>>('/digital-human/runtime-config')
    Object.assign(runtimeConfig, {
      ...runtimeConfig,
      ...data,
      volume: Math.min(100, Math.max(0, Number(data.volume ?? runtimeConfig.volume))),
    })
  } catch {
    // Keep safe defaults when runtime configuration is unavailable.
  }
  audio.setVolume(runtimeConfig.volume / 100)
}

async function loadAvatarOptionsAndPreference() {
  try {
    const options = await apiFetch<DigitalHumanAvatarOption[]>('/digital-human/avatar-options')
    avatarOptions.value = options
  } catch {
    avatarOptions.value = []
  }

  let nextAvatarId = authStore.user?.preferredAvatarId || runtimeConfig.default_avatar_id || DEFAULT_AVATAR_ID
  let savedAvatarId = nextAvatarId
  try {
    const preference = await apiFetch<AvatarPreferenceResponse>('/user/avatar-preference')
    nextAvatarId = preference.avatar_id || nextAvatarId
    savedAvatarId = nextAvatarId
  } catch {
    // 用户偏好读取失败时继续使用默认形象，文字问答不受影响。
  }

  if (!isKnownAvatar(nextAvatarId)) {
    nextAvatarId = avatarOptions.value[0]?.id || DEFAULT_AVATAR_ID
  }
  selectedAvatarId.value = nextAvatarId
  if (avatarOptions.value.length === 1 && nextAvatarId !== savedAvatarId) {
    void ensureCSRFToken()
      .then((ok) => {
        if (!ok) return
        return apiFetch('/user/avatar-preference', {
          method: 'PUT',
          body: JSON.stringify({ avatar_id: nextAvatarId }),
        })
      })
      .catch(() => {})
  }
}

function syncSocketAvatar() {
  const avatar = selectedAvatar.value
  if (!avatar?.config_file) return
  socket?.switchConfig(avatar.config_file)
}

async function selectAvatar(id: string) {
  if (id === selectedAvatarId.value || !isKnownAvatar(id)) return
  const previousAvatarId = selectedAvatarId.value
  selectedAvatarId.value = id
  syncSocketAvatar()
  avatarSaving.value = true
  try {
    if (!(await ensureCSRFToken())) {
      throw new Error('missing csrf token')
    }
    await apiFetch('/user/avatar-preference', {
      method: 'PUT',
      body: JSON.stringify({ avatar_id: id }),
    })
    authStore.invalidateAuth()
    void authStore.fetchUser()
  } catch (error) {
    selectedAvatarId.value = previousAvatarId
    syncSocketAvatar()
    addMessage('system', error instanceof Error ? error.message : t('dh.avatar.saveFailed'))
  } finally {
    avatarSaving.value = false
  }
}

function cleanEmotionTags(text: string) {
  const pattern = new RegExp(`\\[(${ALL_EMOTION_TOKENS.join('|')})\\]\\s*`, 'gi')
  return text.replace(pattern, '').trim()
}

function stripEmotionTags(text: string) {
  return cleanEmotionTags(text) || text
}

function formatRAGSource(source: RAGSource) {
  const title = source.title || source.source || source.id
  const origin = source.source && source.source !== title ? ` · ${source.source}` : ''
  return `${title}${origin}`
}

function expressionFromText(text?: string): Live2DExpression | undefined {
  if (!text) return undefined
  const pattern = new RegExp(`\\[(${ALL_EMOTION_TOKENS.join('|')})\\]`, 'i')
  const match = text.match(pattern)
  return match ? emotionTokenToExpression(match[1] as EmotionToken) : undefined
}

function expressionFromToken(token?: string | number): Live2DExpression | undefined {
  if (token === undefined || token === null) return undefined
  if (typeof token === 'number') {
    if (token === 0) return 'neutral'
    if (token === 1) return 'happy'
    if (token === 2) return 'sad'
    if (token === 3) return 'surprised'
    if (token === 4) return 'angry'
    if (token === 5) return 'thinking'
    if (token === 6) return 'blush'
    if (token === 7) return 'interrupted'
    return undefined
  }

  const normalized = token.toLowerCase().replace(/^\[|\]$/g, '')
  if (/^\d+$/.test(normalized)) return expressionFromToken(Number(normalized))
  if (normalized.startsWith('exp_')) {
    if (normalized === 'exp_01') return 'neutral'
    if (normalized === 'exp_02') return 'happy'
    if (normalized === 'exp_03') return 'sad'
    if (normalized === 'exp_04') return 'surprised'
    if (normalized === 'exp_05') return 'angry'
    if (normalized === 'exp_06') return 'thinking'
    if (normalized === 'exp_07') return 'blush'
    if (normalized === 'exp_08') return 'interrupted'
  }
  return emotionTokenToExpression(normalized as EmotionToken)
}

function emotionTokenToExpression(token?: string): Live2DExpression | undefined {
  if (!token) return undefined
  const normalizedToken = token.toLowerCase().replace(/^\[|\]$/g, '')
  if (normalizedToken === 'joy' || normalizedToken === 'happy' || normalizedToken === 'satisfaction') return 'happy'
  if (normalizedToken === 'neutral') return 'neutral'
  if (normalizedToken === 'anger' || normalizedToken === 'angry' || normalizedToken === 'complaint') return 'angry'
  if (normalizedToken === 'sadness' || normalizedToken === 'sad') return 'sad'
  if (normalizedToken === 'surprise' || normalizedToken === 'surprised' || normalizedToken === 'excitement') return 'surprised'
  if (normalizedToken === 'fear' || normalizedToken === 'thinking' || normalizedToken === 'anxiety' || normalizedToken === 'question') return 'thinking'
  if (normalizedToken === 'disgust' || normalizedToken === 'interrupted') return 'interrupted'
  return undefined
}

function resolveSpeechExpression(cueExpression?: string, text?: string): Live2DExpression {
  return expressionFromToken(cueExpression) || expressionFromText(text) || 'neutral'
}

function resetAssistantTurn() {
  activeAssistantMessageId = ''
  activeAssistantText = ''
  lastAssistantSpeechText = ''
}

function localMessagesKey(sessionId = getOrCreateSessionId()) {
  return `sg_dh_msgs_${sessionId}`
}

function saveLocalMessagesSnapshot(sessionId = getOrCreateSessionId()) {
  try {
    const snapshot = state.messages
      .filter((m) => m.role === 'user' || m.role === 'assistant')
      .map((m) => ({ role: m.role, content: m.text, time: Date.now() }))
    localStorage.setItem(localMessagesKey(sessionId), JSON.stringify(snapshot.slice(-200)))
  } catch {
    // localStorage 只是离线备份，写入失败不影响当前会话。
  }
}

function mergeAssistantText(existing: string, chunk: string) {
  const current = existing.trimEnd()
  const next = chunk.trim()
  if (!next) return current
  if (!current) return next
  if (current === next || current.endsWith(next) || current.includes(next)) return current
  if (next.startsWith(current)) return next

  const maxOverlap = Math.min(current.length, next.length)
  for (let size = maxOverlap; size >= 8; size -= 1) {
    if (current.endsWith(next.slice(0, size))) {
      return current + next.slice(size)
    }
  }

  const separator = /[。！？!?]$/.test(current) ? '\n\n' : ''
  return current + separator + next
}

/** Persist message to both localStorage and backend */
async function persistMessage(role: ChatMessage['role'], content: string) {
  // localStorage backup (always works offline)
  try {
    const key = localMessagesKey()
    const existing = JSON.parse(localStorage.getItem(key) || '[]')
    existing.push({ role, content, time: Date.now() })
    // Keep only last 200 messages
    if (existing.length > 200) existing.splice(0, existing.length - 200)
    localStorage.setItem(key, JSON.stringify(existing))

    // 清理 7 天前的其他 session 消息
    const MAX_AGE_MS = 7 * 24 * 60 * 60 * 1000
    for (let i = localStorage.length - 1; i >= 0; i--) {
      const k = localStorage.key(i)
      if (k?.startsWith('sg_dh_msgs_') && k !== key) {
        try {
          const msgs = JSON.parse(localStorage.getItem(k) || '[]')
          if (msgs.length > 0 && Date.now() - msgs[msgs.length - 1].time > MAX_AGE_MS) {
            localStorage.removeItem(k)
          }
        } catch {
          localStorage.removeItem(k!)
        }
      }
    }
  } catch {
    /* ignore */
  }

  sessionStore.appendMessage(role, content)
  await sessionStore.saveMessage(getOrCreateSessionId(), role, content)
}

function addMessage(role: ChatMessage['role'], text: string) {
  const displayText = stripEmotionTags(text)
  state.messages.push({
    id: uid(),
    role,
    text: displayText,
    time: nowTime(),
  })
  state.subtitle = displayText
  if (role === 'user' || role === 'assistant') {
    void persistMessage(role, displayText)
  }
}

function showAssistantSpeech(text: string, sources: RAGSource[] = []) {
  const displayText = stripEmotionTags(text)
  state.subtitle = displayText
  if (displayText === lastAssistantSpeechText) return
  currentInsight.value = buildGuideInsight(displayText, locale.value)
  resetAssistantTurn()
  lastAssistantSpeechText = displayText
  typewriterStreaming.value = true
  state.messages.push({
    id: uid(),
    role: 'assistant',
    text: displayText,
    time: nowTime(),
    sources,
  })
  void persistMessage('assistant', displayText)
}

function appendAssistantSpeechChunk(text: string) {
  const displayText = stripEmotionTags(text)
  if (!displayText || displayText === 'Thinking...') return

  activeAssistantText = mergeAssistantText(activeAssistantText, displayText)
  currentInsight.value = buildGuideInsight(activeAssistantText, locale.value)
  lastAssistantSpeechText = activeAssistantText
  state.subtitle = displayText
  typewriterStreaming.value = true

  if (!activeAssistantMessageId) {
    activeAssistantMessageId = uid()
    state.messages.push({
      id: activeAssistantMessageId,
      role: 'assistant',
      text: activeAssistantText,
      time: nowTime(),
    })
  } else {
    const msg = state.messages.find((m) => m.id === activeAssistantMessageId)
    if (msg) {
      msg.text = activeAssistantText
    } else {
      activeAssistantMessageId = uid()
      state.messages.push({
        id: activeAssistantMessageId,
        role: 'assistant',
        text: activeAssistantText,
        time: nowTime(),
      })
    }
  }

  saveLocalMessagesSnapshot()
}

function setAssistantStreamText(text: string) {
  const displayText = cleanEmotionTags(text)
  if (!displayText) return

  activeAssistantText = displayText
  currentInsight.value = buildGuideInsight(displayText, locale.value)
  lastAssistantSpeechText = displayText
  state.subtitle = displayText
  typewriterStreaming.value = true

  if (!activeAssistantMessageId) {
    activeAssistantMessageId = uid()
    state.messages.push({
      id: activeAssistantMessageId,
      role: 'assistant',
      text: displayText,
      time: nowTime(),
    })
    return
  }
  const message = state.messages.find((item) => item.id === activeAssistantMessageId)
  if (message) message.text = displayText
}

function connectSocket() {
  socket?.disconnect()
  backendFallbackActive = false
  window.clearTimeout(fallbackTimer)
  state.conversation = 'connecting'
  socket = new VtuberSocketClient(undefined, {
    onOpen: () => {
      state.connected = true
      state.conversation = 'idle'
      syncSocketAvatar()
      const connectedMessage = t('dh.connected')
      if (!state.messages.some((message) => message.role === 'system' && message.text === connectedMessage)) {
        addMessage('system', connectedMessage)
      }
    },
    onClose: () => {
      state.connected = false
      if (!backendFallbackActive && state.conversation !== 'interrupted') state.conversation = 'idle'
    },
    onError: (message: string) => {
      if (backendFallbackActive) return
      finishVoiceLatencyTrace('failed', 'socket_error')
      addMessage('system', message)
      state.conversation = 'idle'
      state.expression = 'neutral'
    },
    onMessage: handleSocketMessage,
  })
  socket.connect()
}

function handleSocketMessage(message: VtuberMessage) {
  if (message.type === 'config-switched' || message.type === 'set-model-and-conf') {
    return
  }

  if (message.type === 'error' || (message.type === 'full-text' && isVtuberGenerationError(message.text))) {
    if (fallbackToBackend()) return
  }

  if (backendFallbackActive && ['audio', 'control', 'error', 'full-text', 'backend-synth-complete'].includes(message.type)) {
    return
  }

  if (blockIncomingPlayback && ['audio', 'full-text', 'backend-synth-complete'].includes(message.type)) {
    return
  }

  if (message.type === 'full-text' && message.text && message.text !== 'Thinking...') {
    pendingSocketQuestion = null
    voiceLatencyTrace?.mark('answer_start')
    transcriptBuffer.value = message.text
    state.expression = expressionFromText(message.text) || state.expression
  }

  if (message.type === 'user-input-transcription' && message.text) {
    addMessage('user', message.text)
  }

  if (message.type === 'control' && message.text === 'conversation-chain-start') {
    if (blockIncomingPlayback && !waitingForFreshServerTurn) return
    window.clearTimeout(fallbackTimer)
    fallbackTimer = 0
    conversationTurn += 1
    interruptedTurn = -1
    resetAssistantTurn()
    hasActiveTurn.value = true
    if (pendingSocketQuestion) pendingSocketQuestion.turn = conversationTurn
    blockIncomingPlayback = false
    waitingForFreshServerTurn = false
    audio.resume()
    state.conversation = 'thinking'
    state.expression = 'thinking'
    state.subtitle = t('dh.thinking')
  }

  if (message.type === 'control' && message.text === 'conversation-chain-end') {
    window.clearTimeout(fallbackTimer)
    fallbackTimer = 0
    pendingSocketQuestion = null
    voiceLatencyTrace?.mark('answer_complete')
    hasActiveTurn.value = isPlaybackActive.value
    typewriterStreaming.value = false
    saveLocalMessagesSnapshot()
    if (state.conversation !== 'interrupted') {
      state.conversation = 'idle'
      state.expression = 'neutral'
    }
  }

  if (message.type === 'control' && message.text === 'interrupt') {
    interruptAnswer()
    return
  }

  if (message.type === 'interrupt-signal') {
    audio.interrupt()
    mouthOpen.value = 0
    hasActiveTurn.value = false
    isPlaybackActive.value = false
    typewriterStreaming.value = false
    blockIncomingPlayback = true
    waitingForFreshServerTurn = false
    interruptedTurn = conversationTurn
    resetAssistantTurn()
    state.conversation = 'interrupted'
    state.expression = 'interrupted'
    state.subtitle = t('dh.interruptReceived')
    return
  }

  if (message.type === 'audio') {
    if (interruptedTurn === conversationTurn) return
    voiceLatencyTrace?.mark('answer_start')
    const text = message.display_text?.text || transcriptBuffer.value || t('dh.fallbackSpeech')
    if (isVtuberGenerationError(text) && fallbackToBackend()) return
    pendingSocketQuestion = null
    const expression = expressionFromToken(message.actions?.expressions?.[0]) || expressionFromText(text)
    const cue = {
      volumes: message.volumes,
      sliceLengthMs: message.slice_length,
      expression,
      showText: false,
    }
    typewriterStreaming.value = true
    appendAssistantSpeechChunk(text)
    if (message.audio) audio.enqueueBase64Wav(message.audio, text, cue)
    else void audio.playTextFallback(text, cue)
  }

  if (message.type === 'backend-synth-complete') {
    voiceLatencyTrace?.mark('answer_complete')
    hasActiveTurn.value = isPlaybackActive.value
    typewriterStreaming.value = false
    saveLocalMessagesSnapshot()
    if (state.conversation === 'thinking') {
      state.conversation = 'idle'
      state.expression = 'neutral'
    }
  }

  if (message.type === 'error') {
    const errMsg = message.message || ''
    if (errMsg.startsWith('Conversation error')) return
    addMessage('system', errMsg || t('dh.speechError'))
    state.conversation = 'error'
  }
}

function startVoiceLatencyTrace() {
  finishVoiceLatencyTrace('aborted', 'superseded')
  voiceLatencyTrace = new VoiceLatencyTrace()
  voiceLatencyTrace.mark('mic_start')
}

function finishVoiceLatencyTrace(status: 'completed' | 'failed' | 'aborted' = 'completed', errorType = '') {
  if (!voiceLatencyTrace) return
  const trace = voiceLatencyTrace
  if (status !== 'completed') {
    trace.fail(errorType || status, status)
  } else if (trace.snapshot().status === 'completed') {
    trace.complete()
  }
  persistVoiceLatencyTrace(trace.snapshot())
  voiceLatencyTrace = null
}

function cancelPendingRequests() {
  activeChatAbortController?.abort()
  activeChatAbortController = null
  activeTTSAbortController?.abort()
  activeTTSAbortController = null
}

function isAbortError(error: unknown) {
  return error instanceof DOMException && error.name === 'AbortError'
}

function cancelCurrentOutputForNewInput(preserveVoiceTrace = false) {
  cancelPendingRequests()
  if (!preserveVoiceTrace) finishVoiceLatencyTrace('aborted', 'superseded')
  if (hasActiveTurn.value || isPlaybackActive.value || waitingForFreshServerTurn) {
    audio.interrupt()
    socket?.interrupt()
  }
}

function sendText() {
  sendTextWithSource('text')
}

function isVtuberGenerationError(text?: string) {
  return Boolean(text && /Error calling the chat endpoint|Error occurred while generating response/i.test(text))
}

function fallbackToBackend() {
  const pending = pendingSocketQuestion
  if (!pending || pending.turn !== conversationTurn || backendFallbackActive) return false
  pendingSocketQuestion = null
  backendFallbackActive = true
  window.clearTimeout(fallbackTimer)
  fallbackTimer = 0
  waitingForFreshServerTurn = false
  blockIncomingPlayback = true
  socket?.disconnect()
  state.connected = false
  state.conversation = 'thinking'
  void answerWithBackendText(pending.text, pending.turn, pending.voiceFeatures)
  return true
}

function sendTextWithSource(source: 'text' | 'voice') {
  const text = input.value.trim()
  if (!text) return

  const voiceEmotion = source === 'voice' ? pendingVoiceEmotion : null
  pendingVoiceEmotion = null

  cancelCurrentOutputForNewInput(source === 'voice')
  window.clearTimeout(fallbackTimer)
  fallbackTimer = 0
  pendingSocketQuestion = null
  if (multimodalLoading.value) {
    multimodalLoading.value = false
    multimodalError.value = ''
  }
  conversationTurn += 1
  interruptedTurn = -1
  hasActiveTurn.value = true
  addMessage('user', text)
  resetAssistantTurn()
  input.value = ''
  state.conversation = 'thinking'
  state.expression = 'thinking'
  state.subtitle = t('dh.thinking')
  transcriptBuffer.value = ''
  currentInsight.value = null

  if (source === 'voice') {
    voiceLatencyTrace?.setEmotionAssessment(voiceEmotion)
    voiceLatencyTrace?.mark('answer_request')
    const acousticExpression = emotionTokenToExpression(voiceEmotion?.category)
    if (acousticExpression) state.expression = acousticExpression
  }

  blockIncomingPlayback = true
  audio.resume()
  waitingForFreshServerTurn = true
  const locationContext = currentLocationContextForQuestion(text)
  waitingForFreshServerTurn = false
  const turn = conversationTurn
  void answerWithBackendText(text, turn, voiceEmotion?.features, locationContext)
}

function closeMultimodalInput() {
  if (multimodalLoading.value) return
  multimodalOpen.value = false
  multimodalError.value = ''
}

async function submitMultimodal(payload: { file: File; message: string }) {
  if (multimodalLoading.value) return

  cancelCurrentOutputForNewInput()
  window.clearTimeout(fallbackTimer)
  multimodalLoading.value = true
  multimodalError.value = ''
  conversationTurn += 1
  interruptedTurn = -1
  const turn = conversationTurn
  const prompt = payload.message.trim()
  const userMessage = prompt
    ? t('dh.multimodal.userAttachmentWithPrompt', { name: payload.file.name, prompt })
    : t('dh.multimodal.userAttachment', { name: payload.file.name })

  hasActiveTurn.value = true
  addMessage('user', userMessage)
  resetAssistantTurn()
  state.conversation = 'thinking'
  state.expression = 'thinking'
  state.subtitle = t('dh.thinking')
  transcriptBuffer.value = ''
  currentInsight.value = null
  blockIncomingPlayback = true
  waitingForFreshServerTurn = false
  audio.resume()

  try {
    const data = await submitMultimodalChat(payload.file, prompt, getOrCreateSessionId())
    if (turn !== conversationTurn || interruptedTurn === turn) return
    const answer = data.response?.trim()
    if (!answer) throw new Error(t('dh.multimodal.emptyResponse'))
    multimodalOpen.value = false
    showAssistantSpeech(answer)
    await playAnswerAudio(answer)
  } catch (error) {
    if (turn !== conversationTurn || interruptedTurn === turn) return
    const message = error instanceof Error && error.message ? error.message : t('dh.multimodal.requestFailed')
    multimodalError.value = message
    addMessage('system', message)
  } finally {
    if (turn === conversationTurn) {
      multimodalLoading.value = false
      blockIncomingPlayback = false
      if (interruptedTurn !== turn && !isPlaybackActive.value && state.conversation === 'thinking') {
        state.conversation = 'idle'
        state.expression = 'neutral'
        typewriterStreaming.value = false
        hasActiveTurn.value = false
      }
    }
  }
}

async function answerWithBackendText(
  text: string,
  turn: number,
  voiceFeatures?: VoiceAcousticFeatures,
  locationContext = currentLocationContextForQuestion(text),
) {
  const chatController = new AbortController()
  const ttsController = new AbortController()
  activeChatAbortController = chatController
  activeTTSAbortController = ttsController
  const segmenter = new SpeechSegmenter()
  const speechQueue = new SerialTaskQueue()
  const canSpeak = audioStatus.value !== 'locked'
  const opening = locale.value.startsWith('zh') ? STREAMING_VOICE_OPENINGS.zh : STREAMING_VOICE_OPENINGS.en
  const configuredExpression = emotionTokenToExpression(runtimeConfig.default_emotion) || ('neutral' as const)
  const rate = seniorModeEnabled.value ? ttsRate.value : runtimeConfig.tts_rate || ttsRate.value
  let answer = ''
  let sequence = 1
  let browserSpeechOnly = false
  let fallbackNotified = false
  let csrfReady = Boolean(getCSRFToken())

  if (canSpeak) {
    audio.beginTurn()
    void audio.playTextFallback(opening, {
      expression: 'thinking',
      showText: false,
      speechKind: 'opening',
      sequence: 0,
    })
  }

  const queueModelSpeech = (rawSegment: string) => {
    const segment = cleanEmotionTags(rawSegment)
    if (!segment || !canSpeak) return
    const segmentSequence = sequence
    sequence += 1
    void speechQueue.enqueue(async () => {
      if (turn !== conversationTurn || interruptedTurn === turn || ttsController.signal.aborted) return
      const cue: PlaybackCue = {
        expression: expressionFromText(segment) || configuredExpression,
        showText: false,
        speechKind: 'model',
        sequence: segmentSequence,
      }

      if (!browserSpeechOnly) {
        try {
          if (!csrfReady) csrfReady = await ensureCSRFToken()
          if (!csrfReady) throw new Error('missing csrf token')
          const audioURL = await synthesizeTTS({
            text: segment,
            voice: runtimeConfig.voice_id,
            rate,
            signal: ttsController.signal,
            timeoutMs: STREAMING_TTS_TIMEOUT_MS,
          })
          if (turn !== conversationTurn || interruptedTurn === turn || ttsController.signal.aborted) {
            URL.revokeObjectURL(audioURL)
            return
          }
          if (audio.enqueueUrl(audioURL, segment, cue, true)) return
          URL.revokeObjectURL(audioURL)
          throw new Error('audio queue unavailable')
        } catch (error) {
          if (ttsController.signal.aborted) return
          browserSpeechOnly = true
          if (!fallbackNotified) {
            fallbackNotified = true
            const message =
              !isAbortError(error) && error instanceof Error && error.message
                ? t('dh.audio.ttsFallbackWithMessage', { message: error.message })
                : t('dh.audio.ttsFallback')
            showAudioNotice('error', message)
          }
        }
      }

      const fallbackQueued = await audio.playTextFallback(segment, cue)
      if (!fallbackQueued) finishVoiceLatencyTrace('failed', 'audio_unavailable')
    })
  }

  try {
    const data = await streamChat(
      {
        session_id: getOrCreateSessionId(),
        message: text,
        ...(voiceFeatures ? { voice_features: voiceFeatures } : {}),
        ...(locationContext ? { location_context: locationContext } : {}),
      },
      {
        signal: chatController.signal,
        onToken: (token) => {
          if (turn !== conversationTurn || interruptedTurn === turn) return
          if (!answer) voiceLatencyTrace?.mark('answer_start')
          answer += token
          setAssistantStreamText(answer)
          for (const segment of segmenter.push(token)) queueModelSpeech(segment)
        },
      },
    )
    if (turn !== conversationTurn || interruptedTurn === turn) return
    for (const segment of segmenter.flush()) queueModelSpeech(segment)
    voiceLatencyTrace?.mark('answer_complete')
    const responseExpression = emotionTokenToExpression(
      data.emotion_category || data.emotion_token || data.emotion,
    )
    if (responseExpression) state.expression = responseExpression
    if (!answer.trim()) {
      answer = buildFallbackAnswer(text)
      setAssistantStreamText(answer)
      queueModelSpeech(answer)
    }
    const assistantMessage = state.messages.find((message) => message.id === activeAssistantMessageId)
    if (assistantMessage) assistantMessage.sources = data.sources || []
    typewriterStreaming.value = false
    saveLocalMessagesSnapshot()
    await speechQueue.drain()
  } catch (error) {
    if (turn !== conversationTurn || interruptedTurn === turn || isAbortError(error)) return
    voiceLatencyTrace?.mark('answer_complete')
    voiceLatencyTrace?.fail('answer_request_error')
    for (const segment of segmenter.flush()) queueModelSpeech(segment)
    if (!answer.trim()) {
      answer = buildFallbackAnswer(text)
      setAssistantStreamText(answer)
      queueModelSpeech(answer)
    } else {
      addMessage('system', error instanceof Error ? error.message : t('dh.speechError'))
    }
    await speechQueue.drain()
  } finally {
    if (canSpeak && turn === conversationTurn && interruptedTurn !== turn) audio.endTurn()
    if (activeChatAbortController === chatController) activeChatAbortController = null
    if (activeTTSAbortController === ttsController) activeTTSAbortController = null
    if (!backendFallbackActive) blockIncomingPlayback = false
    if (interruptedTurn !== turn && !isPlaybackActive.value && state.conversation === 'thinking') {
      state.conversation = 'idle'
      state.expression = 'neutral'
      typewriterStreaming.value = false
      hasActiveTurn.value = false
    }
  }
}

async function ensureCSRFToken() {
  if (getCSRFToken()) return true
  if (await authStore.fetchUser(true)) {
    return Boolean(getCSRFToken())
  }
  authStore.invalidateAuth()
  if (await authStore.ensureGuestSession()) {
    return Boolean(getCSRFToken())
  }
  return false
}

function downloadVoiceLatencyTraces() {
  const count = exportVoiceLatencyTraces()
  addMessage('system', count > 0 ? t('dh.traceExported', { count }) : t('dh.traceExportEmpty'))
}

async function playAnswerAudio(answer: string, responseExpression?: Live2DExpression) {
  const speechText = stripEmotionTags(answer)
  const configuredExpression = emotionTokenToExpression(runtimeConfig.default_emotion)
  const cue = {
    expression: responseExpression || expressionFromText(answer) || configuredExpression || ('neutral' as const),
  }
  if (audioStatus.value === 'locked') {
    showAudioNotice('locked', t('dh.audio.lockedNotice'))
    finishVoiceLatencyTrace('failed', 'audio_locked')
    return
  }
  activeTTSAbortController?.abort()
  const controller = new AbortController()
  activeTTSAbortController = controller
  try {
    if (!(await ensureCSRFToken())) {
      throw new Error('missing csrf token')
    }
    const rate = seniorModeEnabled.value ? ttsRate.value : runtimeConfig.tts_rate || ttsRate.value
    const response = await streamTTS({ text: speechText, voice: runtimeConfig.voice_id, rate, signal: controller.signal })
    const streamed = await audio.enqueueStream(response, speechText, cue)
    if (!streamed) {
      const fallbackPlayed = await audio.playTextFallback(speechText, cue)
      if (!fallbackPlayed) finishVoiceLatencyTrace('failed', 'audio_unavailable')
    }
  } catch (err) {
    if (isAbortError(err)) return
    voiceLatencyTrace?.fail('tts_request_error')
    const message =
      err instanceof Error && err.message
        ? t('dh.audio.ttsFallbackWithMessage', { message: err.message })
        : t('dh.audio.ttsFallback')
    showAudioNotice('error', message)
    const fallbackPlayed = await audio.playTextFallback(speechText, cue)
    if (!fallbackPlayed) finishVoiceLatencyTrace('failed', 'audio_unavailable')
  } finally {
    if (activeTTSAbortController === controller) activeTTSAbortController = null
  }
}

async function loadGeofenceSpots() {
  try {
    const data = await apiFetch<Array<Record<string, unknown>>>('/spots')
    const mappedSpots = data.map((raw, index) => ({
      id: String(raw.id || raw.ID || `spot-${index}`),
      numericId: Number(raw.id || raw.ID || 0),
      name: String(raw.name || raw.Name || t('dh.spotFallbackName', { id: index + 1 })),
      category: String(raw.category || raw.Category || ''),
      lat: Number(raw.latitude || raw.Latitude || 0),
      lng: Number(raw.longitude || raw.Longitude || 0),
      triggerEnabled: Boolean(raw.geofence_enabled || raw.GeofenceEnabled),
      triggerRadiusM: Number(raw.geofence_radius_m || raw.GeofenceRadiusM || 100),
      introText: String(raw.geofence_intro_text || raw.GeofenceIntroText || ''),
      cooldownMinutes: Number(raw.geofence_cooldown_minutes || raw.GeofenceCooldownMinutes || 1440),
    }))
    visitorRatingSpots.value = mappedSpots
      .filter((spot) => Number.isFinite(spot.numericId) && spot.numericId > 0)
      .map((spot) => ({ id: spot.numericId, name: spot.name, category: spot.category }))
    if (!selectedRatingSpotId.value && visitorRatingSpots.value.length > 0) {
      selectedRatingSpotId.value = visitorRatingSpots.value[0].id
    }
    geofenceSpots.value = mappedSpots.filter(
      (spot) => spot.lat !== 0 && spot.lng !== 0 && spot.triggerEnabled,
    )
    setSpots(geofenceSpots.value)
  } catch {
    geofenceSpots.value = []
  }
}

function parseRouteSpotIds(raw: string) {
  try {
    const parsed = JSON.parse(raw) as unknown
    if (Array.isArray(parsed)) {
      return parsed.map((value) => Number(value)).filter((value) => Number.isFinite(value) && value > 0)
    }
  } catch {
    // Fall back to delimiter parsing below.
  }
  return raw
    .split(/[,，|\s]+/)
    .map((value) => Number(value.trim()))
    .filter((value) => Number.isFinite(value) && value > 0)
}

function parseRouteSpotNames(raw: string) {
  let values: string[] = []
  try {
    const parsed = JSON.parse(raw) as unknown
    if (Array.isArray(parsed)) {
      values = parsed.map((value) => String(value))
    }
  } catch {
    values = raw.split(/[,，、|\s]+/)
  }
  return Array.from(
    new Set(
      values
        .map((value) => value.trim())
        .filter((value) => value && !Number.isFinite(Number(value))),
    ),
  )
}

function findRatingSpotByRouteName(name: string) {
  return visitorRatingSpots.value.find(
    (spot) => spot.name === name || spot.name.includes(name) || name.includes(spot.name),
  )
}

function resolveRouteSpotIds(raw: string) {
  const ids = parseRouteSpotIds(raw)
  const nameIds = parseRouteSpotNames(raw)
    .map((name) => findRatingSpotByRouteName(name)?.id)
    .filter((id): id is number => typeof id === 'number')
  return Array.from(new Set([...ids, ...nameIds]))
}

function formatRouteSpots(raw: string) {
  const ids = resolveRouteSpotIds(raw)
  const names = ids.map((id) => visitorRatingSpots.value.find((spot) => spot.id === id)?.name).filter(Boolean)
  if (names.length > 0) return names.join(' · ')
  const routeSpotNames = parseRouteSpotNames(raw)
  if (routeSpotNames.length > 0) return routeSpotNames.join(' · ')
  return raw || t('dh.visitorLoop.routeSpotsUnknown')
}

async function recommendVisitorRoutes() {
  if (isMobileView.value) visitorLoopExpanded.value = true
  const profile = selectedVisitorProfileConfig.value
  routeRecommendationLoading.value = true
  routeRecommendationError.value = ''
  spotRatingMessage.value = ''
  try {
    const result = await visitorExperienceApi.recommendRoutes({
      session_id: getOrCreateSessionId(),
      profile_type: profile.id,
      interest_tags: profile.tags,
      difficulty: profile.difficulty,
      limit: 3,
    })
    recommendedRoutes.value = result.routes || []
    const firstRouteSpot = recommendedRoutes.value
      .flatMap((item) => resolveRouteSpotIds(item.route.spots))
      .find((id) => visitorRatingSpots.value.some((spot) => spot.id === id))
    if (firstRouteSpot) selectedRatingSpotId.value = firstRouteSpot
    if (recommendedRoutes.value.length === 0) {
      routeRecommendationError.value = t('dh.visitorLoop.noRoutes')
    }
  } catch (err) {
    routeRecommendationError.value = err instanceof Error ? err.message : t('dh.visitorLoop.recommendFailed')
  } finally {
    routeRecommendationLoading.value = false
  }
}

function selectRatingSpotFromRoute(recommendedRoute: RecommendedRoute) {
  const routeSpotId = resolveRouteSpotIds(recommendedRoute.route.spots).find((id) =>
    visitorRatingSpots.value.some((spot) => spot.id === id),
  )
  if (routeSpotId) {
    selectedRatingSpotId.value = routeSpotId
  }
}

async function submitSpotRating() {
  if (!selectedRatingSpotId.value) {
    spotRatingMessage.value = t('dh.visitorLoop.ratingSpotRequired')
    return
  }
  spotRatingLoading.value = true
  spotRatingMessage.value = ''
  try {
    await visitorExperienceApi.submitSpotRating({
      session_id: getOrCreateSessionId(),
      spot_id: selectedRatingSpotId.value,
      overall_rating: spotRatingScore.value,
      comment: spotRatingComment.value,
      tags: selectedVisitorProfileConfig.value.tags,
    })
    spotRatingMessage.value = t('dh.visitorLoop.ratingSaved', {
      name: selectedRatingSpot.value?.name || t('dh.visitorLoop.currentSpot'),
    })
    spotRatingComment.value = ''
  } catch (err) {
    spotRatingMessage.value = err instanceof Error ? err.message : t('dh.visitorLoop.ratingFailed')
  } finally {
    spotRatingLoading.value = false
  }
}

function toggleAutoGuide() {
  autoGuideEnabled.value = !autoGuideEnabled.value
  localStorage.setItem(AUTO_GUIDE_KEY, String(autoGuideEnabled.value))
  if (autoGuideEnabled.value) {
    resetTriggered()
    setSpots(geofenceSpots.value)
    startWatch()
  } else {
    stopWatch()
    demoPosition.value = null
  }
}

async function simulateDemoLocation(spot: SpotWithCoords) {
  resetTriggered()
  stopWatch()
  if (!autoGuideEnabled.value) {
    autoGuideEnabled.value = true
    localStorage.setItem(AUTO_GUIDE_KEY, 'true')
    setSpots(geofenceSpots.value)
  }
  demoPosition.value = null
  await nextTick()
  const startedAt = Date.now()
  for (let index = 0; index < 3; index += 1) {
    demoPosition.value = {
      lat: spot.lat,
      lng: spot.lng,
      accuracy: 0,
      timestamp: startedAt + index,
    }
    await nextTick()
  }
}

function useRealLocation() {
  demoPosition.value = null
  if (autoGuideEnabled.value) startWatch()
}

watch(nearbySpot, async (spot) => {
  if (!spot || !autoGuideEnabled.value) return
  const spotId = Number(spot.id)
  if (Number.isFinite(spotId) && spotId > 0) {
    selectedRatingSpotId.value = spotId
  }
  const text = spot.introText || t('dh.autoGuideIntro', { name: spot.name })
  addMessage('system', t('dh.autoGuideTriggered', { name: spot.name }))
  showAssistantSpeech(text)
  await playAnswerAudio(text)
})

async function enableSound() {
  const ok = await audio.unlock()
  if (ok) {
    audioUnlocked.value = true
    audioStatus.value = 'ready'
    audioNotice.value = t('dh.audio.enabledNotice')
    lastAudioNotice = ''
  }
}

function startChatResize(event: PointerEvent) {
  if (isMobileView.value) return
  isChatResizing.value = true
  ;(event.currentTarget as HTMLElement).setPointerCapture?.(event.pointerId)
  window.addEventListener('pointermove', onChatResize)
  window.addEventListener('pointerup', stopChatResize, { once: true })
}

function onChatResize(event: PointerEvent) {
  if (!isChatResizing.value) return
  const nextWidth = window.innerWidth - event.clientX
  chatWidth.value = Math.min(620, Math.max(320, nextWidth))
}

function stopChatResize() {
  if (!isChatResizing.value) return
  isChatResizing.value = false
  localStorage.setItem('sg_dh_chat_width', String(Math.round(chatWidth.value)))
  window.removeEventListener('pointermove', onChatResize)
}

function interruptAnswer() {
  recognition?.stop()
  voiceEmotionCapture.cancel()
  pendingVoiceEmotion = null
  cancelPendingRequests()
  finishVoiceLatencyTrace('aborted', 'user_interrupt')
  window.clearTimeout(fallbackTimer)
  if (multimodalLoading.value) {
    multimodalLoading.value = false
    multimodalError.value = ''
  }
  interruptedTurn = conversationTurn
  hasActiveTurn.value = false
  isPlaybackActive.value = false
  isVoiceListening.value = false
  isVoiceStarting.value = false
  typewriterStreaming.value = false
  blockIncomingPlayback = true
  waitingForFreshServerTurn = false
  audio.interrupt()
  resetAssistantTurn()
  const serverInterrupted = socket?.interrupt(state.subtitle) || false
  mouthOpen.value = 0
  state.interruptCount += 1
  state.conversation = 'interrupted'
  state.expression = 'interrupted'
  state.subtitle = serverInterrupted ? t('dh.interruptedServer') : t('dh.interruptedLocal')
  addMessage('system', state.subtitle)

  window.setTimeout(() => {
    if (state.conversation === 'interrupted') {
      state.conversation = 'idle'
      state.expression = 'neutral'
    }
  }, 1200)
}

function toggleVoice() {
  const Recognition = window.SpeechRecognition || window.webkitSpeechRecognition
  if (!Recognition) {
    addMessage('system', t('dh.browserNotSupported'))
    return
  }

  if (recognition && (isVoiceListening.value || isVoiceStarting.value)) {
    recognition.stop()
    return
  }

  cancelCurrentOutputForNewInput()
  const nextRecognition = new Recognition()
  recognition = nextRecognition
  nextRecognition.lang = 'zh-CN'
  nextRecognition.interimResults = false
  nextRecognition.continuous = false
  nextRecognition.onstart = () => {
    startVoiceLatencyTrace()
    isVoiceStarting.value = false
    isVoiceListening.value = true
    state.conversation = 'listening'
    state.expression = 'surprised'
    state.subtitle = t('dh.listening')
    void voiceEmotionCapture.start().catch(() => {})
  }
  nextRecognition.addEventListener('result', (rawEvent) => {
    const event = rawEvent as SpeechRecognitionEvent
    const spokenText = event.results?.[0]?.[0]?.transcript?.trim()
    if (!spokenText) return
    voiceLatencyTrace?.mark('speech_end')
    voiceLatencyTrace?.mark('asr_result')
    pendingVoiceEmotion = voiceEmotionCapture.stop(spokenText)
    input.value = spokenText
    sendTextWithSource('voice')
  })
  nextRecognition.addEventListener('error', (rawEvent) => {
    const event = rawEvent as SpeechRecognitionErrorEvent
    const detail =
      event.error === 'not-allowed' || event.error === 'service-not-allowed'
        ? t('dh.noMicPermission')
        : t('dh.micNotSupported')
    addMessage('system', detail)
    isVoiceStarting.value = false
    isVoiceListening.value = false
    voiceEmotionCapture.cancel()
    if (!voiceLatencyTrace?.snapshot().events.answer_start) {
      finishVoiceLatencyTrace('failed', event.error || 'asr_error')
    }
    if (state.conversation === 'listening') {
      state.conversation = 'idle'
      state.expression = 'neutral'
    }
  })
  nextRecognition.addEventListener('end', () => {
    voiceLatencyTrace?.mark('speech_end')
    isVoiceStarting.value = false
    isVoiceListening.value = false
    voiceEmotionCapture.cancel()
    if (!voiceLatencyTrace?.snapshot().events.answer_start) {
      finishVoiceLatencyTrace('failed', 'no_asr_result')
    }
    if (recognition === nextRecognition) recognition = null
    if (state.conversation === 'listening') {
      state.conversation = 'idle'
      state.expression = 'neutral'
    }
  })

  try {
    isVoiceStarting.value = true
    nextRecognition.start()
  } catch {
    isVoiceStarting.value = false
    isVoiceListening.value = false
    recognition = null
    voiceEmotionCapture.cancel()
    addMessage('system', t('dh.micStartFailed'))
  }
}

// --- Voice button long-press support ---
let voicePressTimer = 0
function onVoicePointerDown() {
  voicePressTimer = window.setTimeout(() => {
    toggleVoice()
  }, 500)
}
function onVoicePointerUp() {
  window.clearTimeout(voicePressTimer)
  // Short click still toggles voice
  if (!isVoiceListening.value && !isVoiceStarting.value) {
    toggleVoice()
  }
}

function onVoicePointerLeave() {
  window.clearTimeout(voicePressTimer)
}

// --- Context-aware quick ask ---
const quickAskBase = computed(() => [
  { label: t('dh.quickAsk.buddhaHeight'), query: t('dh.quickAsk.buddhaHeight') },
  { label: t('dh.quickAsk.withKids'), query: t('dh.quickAsk.withKids') },
  { label: t('dh.quickAsk.routeLabel'), query: t('dh.quickAsk.route') },
  { label: t('dh.quickAsk.hoursLabel'), query: t('dh.quickAsk.hours') },
])

const followUpQuestions = computed(() => {
  const lastMsg = state.messages.filter((m) => m.role === 'assistant').pop()
  if (!lastMsg) return []
  const text = lastMsg.text
  const questions: { label: string; query: string }[] = []
  if (text.includes('路线') || text.includes('route')) {
    questions.push({ label: t('dh.quickAsk.routeDetail'), query: t('dh.quickAsk.routeDetailQuery') })
    questions.push({ label: t('dh.quickAsk.routeTime'), query: t('dh.quickAsk.routeTimeQuery') })
  }
  if (text.includes('历史') || text.includes('history')) {
    questions.push({ label: t('dh.quickAsk.historyDetail'), query: t('dh.quickAsk.historyDetailQuery') })
  }
  // 动态匹配景点实体（从 profile API 获取，而非硬编码）
  for (const entity of topicEntities.value) {
    if (text.includes(entity)) {
      questions.push({
        label: t('dh.quickAsk.entityDetailLabel', { name: entity }),
        query: t('dh.quickAsk.entityDetailQuery', { name: entity }),
      })
      break
    }
  }
  return questions.slice(0, 3)
})

const quickAskItems = computed(() => {
  const followUps = followUpQuestions.value
  if (followUps.length > 0) return followUps
  return quickAskBase.value
})

function quickAsk(query: string) {
  input.value = query
  sendText()
}

// --- Search ---
function toggleSearch() {
  showSearch.value = !showSearch.value
  if (!showSearch.value) {
    searchQuery.value = ''
    searchResults.value = []
  }
}

function clearSearch() {
  searchQuery.value = ''
  searchResults.value = []
}

function onSearchInput() {
  window.clearTimeout(searchDebounceTimer)
  const q = searchQuery.value.trim()
  if (!q) {
    searchResults.value = []
    isSearching.value = false
    return
  }
  searchDebounceTimer = window.setTimeout(() => {
    void performSearch(q)
  }, 300)
}

async function performSearch(keyword: string) {
  const q = keyword.trim()
  if (!q) return
  try {
    isSearching.value = true
    const [historyResults, localResults] = await Promise.all([
      sessionStore.searchMessages(q, 1, 20),
      Promise.resolve(
        state.messages
          .filter((m) => m.text.toLowerCase().includes(q.toLowerCase()))
          .map((m) => ({
            id: `local-${m.id}`,
            text: m.text,
            time: m.time,
            role: m.role,
            sessionId: sessionStore.currentSessionId || getOrCreateSessionId(),
            sessionTitle: t('dh.currentSession'),
          })),
      ),
    ])

    const results = historyResults.list.map((item) => ({
      id: `history-${item.id}`,
      text: item.content,
      time: formatSearchTime(item.created_at),
      role: item.role as ChatMessage['role'],
      sessionId: item.session_id,
      sessionTitle: item.session_title || t('dh.sessionDefaultTitle'),
    }))
    const seen = new Set(results.map((item) => `${item.sessionId}:${item.text}`))
    for (const item of localResults) {
      const key = `${item.sessionId}:${item.text}`
      if (!seen.has(key)) {
        seen.add(key)
        results.push(item)
      }
    }
    searchResults.value = results.slice(0, 20)
  } finally {
    isSearching.value = false
  }
}

async function openSearchResult(result: { sessionId?: string }) {
  if (!result.sessionId) return
  await switchSession(result.sessionId)
  showSearch.value = false
  clearSearch()
}

// --- Session drawer ---
function toggleSessionDrawer() {
  showSessionDrawer.value = !showSessionDrawer.value
  if (showSessionDrawer.value) {
    void sessionStore.loadSessions()
  }
}

async function switchSession(sessionId: string) {
  sessionStore.setCurrentSession(sessionId)
  try {
    await sessionStore.loadMessages(sessionId, 50)
    if (sessionStore.messages.length > 0) {
      state.messages = sessionStore.messages.map((m) => ({
        id: `hist-${m.id}`,
        role: m.role as ChatMessage['role'],
        text: m.content,
        time: formatTime(m.created_at),
      }))
    } else {
      const localMsgs = loadLocalMessages(sessionId)
      if (localMsgs.length > 0) state.messages = localMsgs
    }
  } catch {
    const localMsgs = loadLocalMessages(sessionId)
    state.messages =
      localMsgs.length > 0
        ? localMsgs
        : [
            {
              id: uid(),
              role: 'assistant',
              text: t('dh.greeting'),
              time: nowTime(),
            },
          ]
  }
  showSessionDrawer.value = false
}

async function deleteSessionAndClose(sessionId: string) {
  await sessionStore.deleteSession(sessionId)
}

// --- Hit area handlers ---
function onHeadClick() {
  // Random expression on head click
  const exprs: Live2DExpression[] = ['happy', 'surprised', 'blush']
  const random = exprs[Math.floor(Math.random() * exprs.length)]
  state.expression = random
  window.setTimeout(() => {
    if (state.expression === random) state.expression = 'neutral'
  }, 2000)
}

function onBodyClick() {
  // Playful feedback on body click
  addMessage(
    'system',
    t('dh.avatar.poked', { name: selectedAvatar.value?.name || t('dh.avatar.fallbackName') }),
  )
}

// --- LocalStorage recovery ---
function loadLocalMessages(sessionId = getOrCreateSessionId()): ChatMessage[] {
  try {
    const raw = localStorage.getItem(localMessagesKey(sessionId))
    if (!raw) return []
    const items = JSON.parse(raw) as Array<{ role: string; content: string; time: number }>
    return items.map((item) => ({
      id: uid(),
      role: item.role as ChatMessage['role'],
      text: item.content,
      time: formatTime(item.time),
    }))
  } catch {
    return []
  }
}

function buildFallbackAnswer(_text: string) {
  // 通用兜底：当 WebSocket 不可用时引导用户使用文字聊天
  // 具体景区信息已通过 RAG 知识库 + ScenicProfile 配置化，不再硬编码景区内容
  return t('dh.fallbackGeneric')
}

onErrorCaptured((err) => {
  console.error('数字人组件错误:', err)
  state.conversation = 'error'
  state.subtitle = t('dh.componentError')
  return false
})

onMounted(async () => {
  const sessionId = getOrCreateSessionId()
  sessionStore.setCurrentSession(sessionId)
  try {
    await sessionStore.loadMessages(sessionId, 50)
    if (sessionStore.messages.length > 0) {
      state.messages = sessionStore.messages.map((m) => ({
        id: `hist-${m.id}`,
        role: m.role as ChatMessage['role'],
        text: m.content,
        time: formatTime(m.created_at),
      }))
    } else {
      const localMsgs = loadLocalMessages(sessionId)
      if (localMsgs.length > 0) {
        state.messages = localMsgs
      }
    }
  } catch {
    // Try localStorage fallback
    const localMsgs = loadLocalMessages(sessionId)
    if (localMsgs.length > 0) {
      state.messages = localMsgs
    }
  }
  await loadRuntimeConfig()
  await loadAvatarOptionsAndPreference()
  // 加载景区配置（topic_entities 用于动态关键词匹配）
  try {
    const profile = await apiFetch<{ topic_entities?: string[] }>('/scenic/profile')
    topicEntities.value = profile.topic_entities || []
  } catch {
    // 静默失败，topicEntities 保持空数组
  }

  connectSocket()
  await loadGeofenceSpots()
  if (autoGuideEnabled.value) startWatch()
  window.addEventListener('resize', onWindowResize)

  // QR 扫码后直接播放接口返回的讲解词，避免再次触发 RAG 推理。
  const qrPayload = readQRIntroPayload()
  if (qrPayload.intro) {
    if (qrPayload.spot) {
      addMessage('system', t('dh.scanSuccess', { name: qrPayload.spot }))
    }
    showAssistantSpeech(qrPayload.intro)
    await playAnswerAudio(qrPayload.intro)
    return
  }

  // QR 扫码追问仍保留自动提问能力。
  const autoAsk = route.query.auto_ask as string
  if (autoAsk) {
    let autoAsked = false
    const waitForSocket = setInterval(() => {
      if (socket && state.connected) {
        clearInterval(waitForSocket)
        autoAsked = true
        input.value = autoAsk
        sendText()
      }
    }, 200)
    setTimeout(() => {
      clearInterval(waitForSocket)
      if (!autoAsked) {
        input.value = autoAsk
        sendText()
      }
    }, 1500)
  }
})

onUnmounted(() => {
  window.clearTimeout(fallbackTimer)
  window.clearTimeout(voicePressTimer)
  window.clearTimeout(searchDebounceTimer)
  socket?.disconnect()
  recognition?.stop()
  voiceEmotionCapture.cancel()
  pendingVoiceEmotion = null
  cancelPendingRequests()
  finishVoiceLatencyTrace('aborted', 'component_unmounted')
  stopWatch()
  audio.interrupt()
  hasActiveTurn.value = false
  isPlaybackActive.value = false
  isVoiceListening.value = false
  isVoiceStarting.value = false
  mouthOpen.value = 0
  window.removeEventListener('resize', onWindowResize)
  window.removeEventListener('pointermove', onChatResize)
  window.removeEventListener('pointerup', stopChatResize)
})
</script>

<template>
  <main class="dh-view" :class="{ 'senior-mode-page': seniorModeEnabled }">
    <!-- 左侧：数字人展示区 -->
    <section v-show="mobileTab === 'avatar' || !isMobileView" class="dh-stage">
      <div class="dh-status">
        <span class="status-dot" :class="{ online: state.connected }"></span>
        <span class="status-text">{{ statusLabel }}</span>
      </div>

      <div
        v-if="avatarOptions.length > 0"
        class="avatar-switcher"
        :class="{ locked: avatarOptions.length === 1 }"
        :aria-label="$t('dh.avatar.ariaLabel')"
      >
        <span v-if="avatarOptions.length === 1" class="avatar-lock-label">{{
          $t('dh.avatar.lockedLabel')
        }}</span>
        <button
          v-for="avatar in avatarOptions"
          :key="avatar.id"
          class="avatar-choice"
          :class="{ active: selectedAvatarId === avatar.id }"
          :disabled="avatarSaving"
          @click="selectAvatar(avatar.id)"
        >
          <span class="avatar-dot" :style="{ background: avatar.preview_color }">{{
            avatar.fallback_label
          }}</span>
          <span>{{ avatar.name }}</span>
        </button>
      </div>

      <Live2DStage
        :state="state.conversation"
        :mouth-open="mouthOpen"
        :expression="state.expression"
        :model-url="selectedAvatar?.model_url"
        @head-click="onHeadClick"
        @body-click="onBodyClick"
      />

      <!-- 字幕气泡 -->
      <div v-if="isMobileView" class="subtitle-bubble">
        {{ state.subtitle }}
      </div>

      <div class="stage-tools">
        <div v-if="autoGuideEnabled || locationMode === 'demo'" class="location-panel">
          <span
            class="location-status"
            :class="{ ready: locationMode === 'gps' && locationAccuracy !== null && locationAccuracy <= 10 }"
          >
            {{ locationStatus }}
          </span>
          <button class="location-action" @click="useRealLocation">{{ $t('dh.controls.location.useGps') }}</button>
          <button
            v-for="spot in geofenceSpots"
            :key="spot.id"
            class="location-action demo"
            @click="simulateDemoLocation(spot)"
          >
            {{ spot.name }}
          </button>
        </div>
        <div
          class="audio-hint"
          :class="{
            error: audioStatus === 'error',
            ready: audioStatus === 'ready' || audioStatus === 'playing',
          }"
        >
          {{ audioNotice }}
        </div>
        <div class="dh-controls">
          <button
            class="ctrl-btn sound"
            :class="{
              ready: audioStatus === 'ready' || audioStatus === 'playing',
              error: audioStatus === 'error',
            }"
            @click="enableSound"
          >
            {{
              audioStatus === 'playing'
                ? $t('dh.controls.soundPlaying')
                : audioStatus === 'ready'
                  ? $t('dh.controls.soundReady')
                  : $t('dh.controls.soundEnable')
            }}
          </button>
          <button class="ctrl-btn danger" :disabled="!canInterrupt" @click="interruptAnswer">
            {{ $t('dh.interrupt') }}
          </button>
          <button class="ctrl-btn" @click="connectSocket">{{ $t('dh.reconnect') }}</button>
          <button class="ctrl-btn" :class="{ ready: autoGuideEnabled }" @click="toggleAutoGuide">
            {{ autoGuideEnabled ? $t('dh.controls.autoGuideOn') : $t('dh.controls.autoGuideOff') }}
          </button>
          <button class="ctrl-btn" :class="{ ready: seniorModeEnabled }" @click="toggleSeniorMode">
            {{ seniorModeEnabled ? $t('dh.controls.exitSeniorMode') : $t('dh.controls.seniorMode') }}
          </button>
          <span class="interrupt-count">{{ $t('dh.interruptCount', { count: state.interruptCount }) }}</span>
        </div>
        <div class="emotion-bar">
          <button
            v-for="expr in ['happy', 'neutral', 'thinking', 'surprised', 'sad', 'angry', 'blush'] as const"
            :key="expr"
            class="emotion-btn"
            :class="{ active: state.expression === expr }"
            @click="state.expression = expr"
          >
            {{
              {
                happy: '😊',
                neutral: '😐',
                thinking: '🤔',
                surprised: '😮',
                sad: '😢',
                angry: '😠',
                blush: '😳',
              }[expr]
            }}
          </button>
        </div>
      </div>
    </section>

    <div
      v-if="!isMobileView"
      class="chat-resizer"
      :class="{ resizing: isChatResizing }"
      :title="$t('dh.controls.chatResizeTitle')"
      @pointerdown.prevent="startChatResize"
    ></div>

    <!-- 右侧：聊天面板 -->
    <aside v-show="mobileTab === 'chat' || !isMobileView" class="dh-chat" :style="chatPanelStyle">
      <div class="chat-header">
        <div class="chat-header-top">
          <h2>{{ $t('dh.title') }}</h2>
          <div class="header-actions">
            <button
              class="icon-btn"
              :class="{ active: showSearch }"
              :title="$t('dh.actions.search')"
              @click="toggleSearch"
            >
              🔍
            </button>
            <button class="icon-btn" :title="$t('dh.actions.history')" @click="toggleSessionDrawer">
              📋
            </button>
            <button
              class="icon-btn"
              :title="$t('dh.actions.exportVoiceTraces')"
              @click="downloadVoiceLatencyTraces"
            >
              ↓
            </button>
            <button
              v-if="authStore.isGuest"
              class="icon-btn upgrade-btn"
              :title="$t('dh.actions.register')"
              @click="showUpgradeModal = true"
            >
              👤
            </button>
          </div>
        </div>
        <p>{{ $t('dh.subtitle') }}</p>

        <!-- 搜索栏 -->
        <div v-if="showSearch" class="search-bar">
          <input
            v-model="searchQuery"
            autofocus
            :placeholder="$t('dh.searchPlaceholder')"
            @input="onSearchInput"
          />
          <button v-if="searchQuery" class="search-clear" @click="clearSearch">✕</button>
        </div>
      </div>

      <!-- 快捷提问 -->
      <div class="quick-asks">
        <button v-for="item in quickAskItems" :key="item.label" @click="quickAsk(item.query)">
          {{ item.label }}
        </button>
        <button
          v-if="followUpQuestions.length > 0"
          class="refresh-btn"
          :title="$t('dh.actions.refresh')"
          @click="state.expression = 'neutral'"
        >
          🔄
        </button>
      </div>

      <section v-if="currentInsight" class="answer-visual-card">
        <div class="answer-visual-thumb">{{ currentInsight.image }}</div>
        <div class="answer-visual-body">
          <div class="answer-visual-header">
            <strong>{{ currentInsight.title }}</strong>
            <span v-for="tag in currentInsight.tags" :key="tag">{{ tag }}</span>
          </div>
          <div class="answer-visual-points">
            <i v-for="point in currentInsight.points" :key="point">{{ point }}</i>
          </div>
        </div>
      </section>

      <section class="visitor-loop-panel" aria-labelledby="visitor-loop-title">
        <div class="visitor-loop-header">
          <div>
            <strong id="visitor-loop-title">{{ $t('dh.visitorLoop.title') }}</strong>
            <span>{{ $t('dh.visitorLoop.subtitle') }}</span>
          </div>
          <div class="visitor-loop-actions">
            <button
              v-if="isMobileView"
              class="visitor-loop-toggle"
              :aria-label="
                visitorLoopExpanded ? $t('dh.visitorLoop.collapse') : $t('dh.visitorLoop.expand')
              "
              @click="visitorLoopExpanded = !visitorLoopExpanded"
            >
              {{ visitorLoopExpanded ? '−' : '＋' }}
            </button>
            <button
              class="loop-primary-btn"
              :disabled="routeRecommendationLoading"
              @click="recommendVisitorRoutes"
            >
              {{
                routeRecommendationLoading
                  ? $t('dh.visitorLoop.recommending')
                  : $t('dh.visitorLoop.recommendRoutes')
              }}
            </button>
          </div>
        </div>

        <div v-if="!isMobileView || visitorLoopExpanded" class="visitor-loop-content">
          <div class="visitor-profile-row" :aria-label="$t('dh.visitorLoop.profileTitle')">
            <button
              v-for="profile in visitorProfiles"
              :key="profile.id"
              class="visitor-profile-option"
              :class="{ active: selectedVisitorProfile === profile.id }"
              @click="selectedVisitorProfile = profile.id"
            >
              <strong>{{ profile.label }}</strong>
              <span>{{ profile.hint }}</span>
            </button>
          </div>

          <p v-if="routeRecommendationError" class="visitor-loop-message error">
            {{ routeRecommendationError }}
          </p>

          <div v-if="recommendedRoutes.length > 0" class="route-recommend-list">
            <article v-for="item in recommendedRoutes" :key="item.route.id" class="route-recommend-card">
              <div class="route-recommend-main">
                <strong>{{ item.route.name }}</strong>
                <span>{{ item.reason }}</span>
              </div>
              <div class="route-recommend-meta">
                <span>{{ formatRouteSpots(item.route.spots) }}</span>
                <span>{{ $t('dh.visitorLoop.durationMinutes', { count: item.route.duration }) }}</span>
                <span>{{ $t('dh.visitorLoop.score', { score: item.score.toFixed(1) }) }}</span>
              </div>
              <button class="route-rate-btn" @click="selectRatingSpotFromRoute(item)">
                {{ $t('dh.visitorLoop.rateThisRoute') }}
              </button>
            </article>
          </div>

          <div class="spot-rating-box">
            <div class="spot-rating-header">
              <strong>{{ $t('dh.visitorLoop.ratingTitle') }}</strong>
              <span>{{ selectedRatingSpot?.name || $t('dh.visitorLoop.currentSpot') }}</span>
            </div>
            <select
              v-model="selectedRatingSpotId"
              :disabled="visitorRatingSpots.length === 0"
              :aria-label="$t('dh.visitorLoop.spotSelect')"
            >
              <option v-for="spot in visitorRatingSpots" :key="spot.id" :value="spot.id">
                {{ spot.name }}{{ spot.category ? ' / ' + spot.category : '' }}
              </option>
            </select>
            <div class="rating-stars" :aria-label="$t('dh.visitorLoop.ratingScore')">
              <button
                v-for="score in ratingScoreOptions"
                :key="score"
                :class="{ active: score <= spotRatingScore }"
                :aria-pressed="score <= spotRatingScore"
                @click="spotRatingScore = score"
              >
                {{ score <= spotRatingScore ? '★' : '☆' }}
              </button>
            </div>
            <textarea
              v-model="spotRatingComment"
              rows="2"
              :placeholder="$t('dh.visitorLoop.ratingCommentPlaceholder')"
            ></textarea>
            <button
              class="loop-primary-btn full"
              :disabled="spotRatingLoading || !selectedRatingSpotId"
              @click="submitSpotRating"
            >
              {{
                spotRatingLoading ? $t('dh.visitorLoop.ratingSubmitting') : $t('dh.visitorLoop.ratingSubmit')
              }}
            </button>
            <p v-if="spotRatingMessage" class="visitor-loop-message">
              {{ spotRatingMessage }}
            </p>
          </div>
        </div>
      </section>

      <!-- 搜索结果 -->
      <div v-if="showSearch && searchResults.length > 0" class="search-results">
        <button
          v-for="(result, i) in searchResults"
          :key="i"
          class="search-result-item"
          @click="openSearchResult(result)"
        >
          <span class="search-result-meta">
            {{ result.sessionTitle || $t('dh.sessionDefaultTitle') }} · {{ result.time }}
          </span>
          <!-- eslint-disable vue/no-v-html -- highlightMatch escapes the search query before injecting mark tags -->
          <span class="search-result-text" v-html="highlightMatch(result.text, searchQuery)" />
          <!-- eslint-enable vue/no-v-html -->
        </button>
      </div>
      <div v-else-if="showSearch && searchQuery && !isSearching" class="search-empty">
        {{ $t('dh.searchNoResults') }}
      </div>

      <!-- 消息列表 -->
      <div v-if="!showSearch || !searchQuery" class="message-list">
        <article v-for="msg in state.messages" :key="msg.id" :class="['msg', msg.role]">
          <div class="msg-label">
            {{
              msg.role === 'user'
                ? $t('dh.user')
                : msg.role === 'assistant'
                  ? $t('dh.assistant')
                  : $t('dh.system')
            }}
          </div>
          <div class="msg-bubble">
            <MarkdownRenderer
              v-if="msg.role === 'assistant'"
              :text="msg.text"
              :streaming="typewriterStreaming && msg.id === state.messages[state.messages.length - 1]?.id"
            />
            <span v-else>{{ msg.text }}</span>
          </div>
          <div v-if="msg.role === 'assistant' && msg.sources?.length" class="msg-sources">
            <span class="msg-sources-label">{{ $t('dh.sourcesLabel') }}</span>
            <span
              v-for="source in msg.sources"
              :key="source.id || source.title"
              class="msg-source"
              :title="source.preview || formatRAGSource(source)"
            >
              {{ formatRAGSource(source) }}
            </span>
          </div>
          <div class="msg-time">{{ msg.time }}</div>
        </article>
      </div>

      <!-- 多模态输入区 -->
      <MultimodalInput
        v-if="multimodalOpen"
        :loading="multimodalLoading"
        :error="multimodalError"
        @submit="submitMultimodal"
        @close="closeMultimodalInput"
      />

      <!-- 输入区 -->
      <div class="chat-input">
        <button
          type="button"
          class="multimodal-btn"
          :class="{ active: multimodalOpen }"
          :disabled="multimodalLoading"
          :aria-label="t('dh.multimodal.open')"
          :aria-expanded="multimodalOpen"
          :title="t('dh.multimodal.open')"
          @click="multimodalOpen = !multimodalOpen"
        >
          ＋
        </button>
        <button
          type="button"
          class="voice-btn"
          :class="{ recording: isVoiceListening || isVoiceStarting }"
          @pointerdown="onVoicePointerDown"
          @pointerup="onVoicePointerUp"
          @pointerleave="onVoicePointerLeave"
        >
          🎤
        </button>
        <input v-model="input" :placeholder="$t('dh.placeholder')" @keydown.enter="sendText" />
        <button type="button" class="send-btn" @click="sendText">{{ $t('dh.send') }}</button>
      </div>
    </aside>

    <!-- 会话抽屉 -->
    <Teleport to="body">
      <div v-if="showSessionDrawer" class="drawer-overlay" @click.self="showSessionDrawer = false">
        <div class="session-drawer">
          <div class="drawer-header">
            <h3>📋 {{ $t('dh.historySessions') }}</h3>
            <button class="drawer-close" @click="showSessionDrawer = false">✕</button>
          </div>
          <div class="drawer-body">
            <div v-if="sessionStore.sessions.length === 0" class="drawer-empty">
              {{ $t('dh.noSessions') }}
            </div>
            <div
              v-for="sess in sessionStore.sessions"
              :key="sess.id"
              class="session-item"
              :class="{ active: sess.session_id === getOrCreateSessionId() }"
              @click="switchSession(sess.session_id)"
            >
              <div class="session-info">
                <span class="session-title">{{ sess.title || $t('dh.sessionDefaultTitle') }}</span>
                <span class="session-meta"
                  >{{ $t('dh.sessionMessageCount', { count: sess.message_count }) }} ·
                  {{ new Date(sess.last_active_at).toLocaleDateString(locale) }}</span
                >
              </div>
              <button class="session-delete" @click.stop="deleteSessionAndClose(sess.session_id)">🗑</button>
            </div>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- 游客升级弹窗 -->
    <Teleport to="body">
      <div v-if="showUpgradeModal" class="drawer-overlay" @click.self="showUpgradeModal = false">
        <div class="upgrade-modal">
          <div class="drawer-header">
            <h3>📝 {{ $t('auth.upgradeTitle') }}</h3>
            <button class="drawer-close" @click="showUpgradeModal = false">✕</button>
          </div>
          <div class="upgrade-form">
            <p class="upgrade-hint">{{ $t('auth.upgradeHint') }}</p>
            <input
              v-model="upgradeForm.username"
              :placeholder="$t('auth.usernamePlaceholder')"
              autocomplete="username"
            />
            <input
              v-model="upgradeForm.password"
              type="password"
              :placeholder="$t('auth.passwordPlaceholder')"
              autocomplete="new-password"
            />
            <input
              v-model="upgradeForm.email"
              type="email"
              :placeholder="$t('auth.emailPlaceholder')"
              autocomplete="email"
            />
            <p v-if="upgradeError" class="upgrade-error">{{ upgradeError }}</p>
            <button class="upgrade-submit" :disabled="upgradeLoading" @click="handleUpgrade">
              {{ upgradeLoading ? $t('auth.upgradeLoading') : $t('auth.upgradeButton') }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>
    <!-- Mobile 底部 Tab 切换 -->
    <nav v-if="isMobileView" class="mobile-tabs">
      <button :class="{ active: mobileTab === 'avatar' }" @click="mobileTab = 'avatar'">
        🤖 {{ $t('dh.tabAvatar') }}
      </button>
      <button :class="{ active: mobileTab === 'chat' }" @click="mobileTab = 'chat'">
        💬 {{ $t('dh.tabChat') }}
      </button>
    </nav>
  </main>
</template>

<script lang="ts">
// highlightMatch helper (used in template via v-html)
function highlightMatch(text: string, query: string): string {
  if (!query) return text
  const escaped = query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const regex = new RegExp(`(${escaped})`, 'gi')
  return text.replace(
    regex,
    '<mark style="background:rgba(99,226,183,.3);border-radius:2px;padding:0 2px">$1</mark>',
  )
}
</script>

<style scoped>
.dh-view {
  display: flex;
  height: calc(100vh - 44px);
  background: var(--sg-bg-ink, #0a0a0f);
}

/* 左侧数字人区域 */
.dh-stage {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: stretch;
  justify-content: center;
  position: relative;
  background:
    radial-gradient(ellipse at 50% 40%, var(--sg-jade-bg, rgba(99, 226, 183, 0.06)) 0%, transparent 60%),
    var(--sg-bg-ink, #0a0a0f);
}

.dh-status {
  position: absolute;
  top: 20px;
  left: 24px;
  display: flex;
  align-items: center;
  gap: 8px;
}
.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #666;
}
.status-dot.online {
  background: var(--sg-jade-bright, #63e2b7);
  box-shadow: 0 0 8px rgba(99, 226, 183, 0.5);
}
.status-text {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.45);
}

.avatar-switcher {
  position: absolute;
  top: 16px;
  right: 20px;
  display: flex;
  gap: 8px;
  padding: 6px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 8px;
  background: rgba(10, 10, 15, 0.52);
  backdrop-filter: blur(12px);
  z-index: 2;
}

.avatar-lock-label {
  display: inline-flex;
  align-items: center;
  padding: 0 6px;
  color: rgba(255, 255, 255, 0.42);
  font-size: 11px;
  white-space: nowrap;
}

.avatar-choice {
  min-width: 78px;
  height: 34px;
  padding: 0 10px 0 6px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border: 1px solid transparent;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.04);
  color: rgba(255, 255, 255, 0.68);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
}

.avatar-choice:hover,
.avatar-choice.active {
  border-color: rgba(99, 226, 183, 0.4);
  background: rgba(99, 226, 183, 0.12);
  color: rgba(255, 255, 255, 0.92);
}

.avatar-choice:disabled {
  cursor: wait;
  opacity: 0.72;
}

.avatar-dot {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  display: inline-grid;
  place-items: center;
  color: #081014;
  font-size: 11px;
  font-weight: 700;
  flex: none;
}

.emotion-bar {
  display: flex;
  gap: 6px;
  margin-top: 8px;
  flex-wrap: wrap;
  justify-content: center;
}
.stage-tools {
  position: absolute;
  left: 18px;
  right: 18px;
  bottom: 16px;
  z-index: 3;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
}
.stage-tools .emotion-bar {
  margin-top: 0;
}
.emotion-btn {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  border: 2px solid rgba(255, 255, 255, 0.1);
  background: rgba(255, 255, 255, 0.04);
  font-size: 15px;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
}
.emotion-btn:hover {
  border-color: var(--sg-jade-bright, #63e2b7);
  transform: scale(1.1);
}
.emotion-btn.active {
  border-color: var(--sg-jade-bright, #63e2b7);
  background: rgba(99, 226, 183, 0.15);
}

.subtitle-bubble {
  max-width: 400px;
  margin-top: 16px;
  padding: 14px 20px;
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 12px;
  font-size: 14px;
  line-height: 1.6;
  color: var(--sg-text-secondary, rgba(255, 255, 255, 0.75));
  text-align: center;
  backdrop-filter: blur(10px);
}

.dh-controls {
  position: static;
  width: 100%;
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  justify-content: center;
}
.ctrl-btn {
  padding: 6px 14px;
  border-radius: 6px;
  border: 1px solid rgba(255, 255, 255, 0.12);
  background: rgba(255, 255, 255, 0.04);
  color: rgba(255, 255, 255, 0.6);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
}
.ctrl-btn:hover {
  background: rgba(255, 255, 255, 0.08);
  color: var(--sg-text-body, rgba(255, 255, 255, 0.88));
}
.ctrl-btn.danger {
  border-color: rgba(232, 128, 128, 0.3);
  color: var(--sg-red-bright, #e88080);
}
.ctrl-btn.danger:hover {
  background: rgba(232, 128, 128, 0.1);
}
.ctrl-btn.sound {
  border-color: rgba(99, 226, 183, 0.28);
  color: var(--sg-jade-bright, #63e2b7);
}
.ctrl-btn.sound.ready {
  background: rgba(99, 226, 183, 0.12);
}
.ctrl-btn.sound.error {
  border-color: rgba(248, 190, 82, 0.36);
  color: #f8be52;
}
.ctrl-btn:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}
.interrupt-count {
  font-size: 11px;
  color: rgba(255, 255, 255, 0.25);
}
.audio-hint {
  position: static;
  width: 100%;
  min-height: 20px;
  color: rgba(255, 255, 255, 0.48);
  font-size: 12px;
  line-height: 1.5;
  text-align: center;
}
.audio-hint.ready {
  color: rgba(99, 226, 183, 0.72);
}
.audio-hint.error {
  color: rgba(248, 190, 82, 0.86);
}
.location-panel {
  position: static;
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  flex-wrap: wrap;
  padding: 6px 8px;
  border: 1px solid rgba(99, 226, 183, 0.12);
  background: rgba(6, 16, 18, 0.72);
  color: rgba(255, 255, 255, 0.58);
  font-size: 11px;
}
.location-status.ready {
  color: var(--sg-jade-bright, #63e2b7);
}
.location-action {
  padding: 4px 8px;
  border: 1px solid rgba(255, 255, 255, 0.14);
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.04);
  color: rgba(255, 255, 255, 0.72);
  font-size: 11px;
  cursor: pointer;
}
.location-action:hover,
.location-action.demo {
  border-color: rgba(99, 226, 183, 0.28);
}

.chat-resizer {
  width: 10px;
  cursor: col-resize;
  background: transparent;
  position: relative;
  flex: none;
}
.chat-resizer::after {
  content: '';
  position: absolute;
  top: 18px;
  bottom: 18px;
  left: 4px;
  width: 1px;
  background: rgba(99, 226, 183, 0.16);
}
.chat-resizer:hover::after,
.chat-resizer.resizing::after {
  left: 3px;
  width: 3px;
  background: rgba(99, 226, 183, 0.55);
}

/* 右侧聊天面板 */
.dh-chat {
  width: min(420px, 40vw);
  min-width: 320px;
  max-width: 620px;
  display: flex;
  flex-direction: column;
  background: rgba(255, 255, 255, 0.02);
  border-left: 1px solid var(--sg-jade-bg, rgba(99, 226, 183, 0.06));
}

.chat-header {
  padding: 16px 20px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}
.chat-header-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.chat-header-top h2 {
  font-size: 15px;
  color: var(--sg-text-body, rgba(255, 255, 255, 0.88));
  margin-bottom: 2px;
}
.chat-header p {
  font-size: 12px;
  color: var(--sg-text-hint, rgba(255, 255, 255, 0.35));
}

.header-actions {
  display: flex;
  gap: 6px;
}
.icon-btn {
  width: 28px;
  height: 28px;
  border-radius: 6px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(255, 255, 255, 0.04);
  font-size: 13px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}
.icon-btn:hover {
  background: rgba(99, 226, 183, 0.1);
  border-color: var(--sg-jade-border, rgba(99, 226, 183, 0.3));
}
.icon-btn.active {
  background: rgba(99, 226, 183, 0.15);
  border-color: var(--sg-jade-bright, #63e2b7);
}

.search-bar {
  display: flex;
  align-items: center;
  margin-top: 10px;
  gap: 6px;
}
.search-bar input {
  flex: 1;
  padding: 6px 10px;
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 6px;
  color: white;
  font-size: 12px;
  outline: none;
}
.search-bar input:focus {
  border-color: var(--sg-jade-bright, #63e2b7);
}
.search-clear {
  background: none;
  border: none;
  color: rgba(255, 255, 255, 0.4);
  cursor: pointer;
  font-size: 14px;
}

.search-results {
  max-height: 200px;
  overflow-y: auto;
  padding: 8px 16px;
}
.search-result-item {
  display: grid;
  width: 100%;
  gap: 4px;
  padding: 8px 0;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.6);
  text-align: left;
  background: transparent;
  border: none;
  border-bottom: 1px solid rgba(255, 255, 255, 0.04);
  line-height: 1.5;
  cursor: pointer;
}
.search-result-item:hover {
  color: rgba(255, 255, 255, 0.86);
}
.search-result-meta {
  color: var(--sg-text-hint, rgba(255, 255, 255, 0.35));
  font-size: 10px;
}
.search-result-text {
  color: inherit;
}
.search-empty {
  padding: 20px;
  text-align: center;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.25);
}

.quick-asks {
  display: flex;
  gap: 6px;
  padding: 10px 16px;
  flex-wrap: wrap;
  overflow-x: hidden;
  border-bottom: 1px solid rgba(255, 255, 255, 0.04);
  align-items: center;
}
.quick-asks button {
  flex: 1 1 120px;
  white-space: normal;
  line-height: 1.35;
  padding: 5px 12px;
  border-radius: 16px;
  border: 1px solid var(--sg-jade-border, rgba(99, 226, 183, 0.2));
  background: var(--sg-jade-bg, rgba(99, 226, 183, 0.06));
  font-size: 11px;
  color: var(--sg-jade-bright, #63e2b7);
  cursor: pointer;
  transition: all 0.2s;
}
.quick-asks button:hover {
  background: rgba(99, 226, 183, 0.15);
}
.quick-asks .refresh-btn {
  margin-left: auto;
  padding: 5px 8px;
  font-size: 14px;
}

.answer-visual-card {
  display: grid;
  grid-template-columns: 52px 1fr;
  gap: 12px;
  margin: 10px 16px 0;
  padding: 12px;
  border: 1px solid rgba(82, 240, 238, 0.14);
  border-radius: 10px;
  background: rgba(82, 240, 238, 0.045);
}
.answer-visual-thumb {
  width: 52px;
  height: 52px;
  border-radius: 8px;
  display: grid;
  place-items: center;
  color: #051214;
  background: var(--sg-gold, #f4c765);
  font-size: 20px;
  font-weight: 800;
}
.answer-visual-body {
  display: grid;
  gap: 8px;
  min-width: 0;
}
.answer-visual-header {
  display: flex;
  gap: 6px;
  align-items: center;
  flex-wrap: wrap;
}
.answer-visual-header strong {
  color: var(--sg-text-heading, rgba(255, 255, 255, 0.92));
  font-size: 13px;
}
.answer-visual-header span {
  color: var(--sg-jade-bright, #63e2b7);
  background: rgba(99, 226, 183, 0.08);
  border: 1px solid rgba(99, 226, 183, 0.14);
  border-radius: 999px;
  padding: 2px 7px;
  font-size: 10px;
}
.answer-visual-points {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}
.answer-visual-points i {
  font-style: normal;
  color: rgba(255, 255, 255, 0.68);
  background: rgba(255, 255, 255, 0.045);
  border-radius: 6px;
  padding: 4px 7px;
  font-size: 11px;
  line-height: 1.35;
}

.visitor-loop-panel {
  display: grid;
  gap: 10px;
  margin: 10px 16px 0;
  padding: 12px;
  max-height: 310px;
  overflow-y: auto;
  border: 1px solid rgba(82, 240, 238, 0.14);
  border-radius: 10px;
  background: rgba(82, 240, 238, 0.04);
}

.visitor-loop-header,
.spot-rating-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.visitor-loop-header > div:first-child {
  display: grid;
  gap: 3px;
  min-width: 0;
}

.visitor-loop-actions {
  display: flex;
  align-items: center;
  gap: 6px;
}

.visitor-loop-toggle {
  width: 30px;
  height: 30px;
  border: 1px solid rgba(99, 226, 183, 0.3);
  border-radius: 8px;
  color: var(--sg-jade-bright, #63e2b7);
  background: rgba(99, 226, 183, 0.08);
  cursor: pointer;
}

.visitor-loop-content {
  display: grid;
  gap: 10px;
}

.visitor-loop-header strong,
.spot-rating-header strong,
.route-recommend-main strong {
  color: var(--sg-text-body, rgba(255, 255, 255, 0.88));
  font-size: 13px;
}

.visitor-loop-header span,
.spot-rating-header span,
.route-recommend-main span,
.route-recommend-meta,
.visitor-loop-message {
  color: var(--sg-text-hint, rgba(255, 255, 255, 0.35));
  font-size: 11px;
}

.visitor-profile-row {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.visitor-profile-option {
  display: grid;
  gap: 3px;
  min-height: 54px;
  padding: 8px;
  border: 1px solid rgba(99, 226, 183, 0.16);
  border-radius: 8px;
  color: var(--sg-text-secondary, rgba(255, 255, 255, 0.75));
  text-align: left;
  background: rgba(255, 255, 255, 0.035);
  cursor: pointer;
}

.visitor-profile-option strong {
  font-size: 12px;
}

.visitor-profile-option span {
  color: var(--sg-text-hint, rgba(255, 255, 255, 0.35));
  font-size: 10px;
  line-height: 1.4;
}

.visitor-profile-option.active {
  border-color: var(--sg-jade-bright, #63e2b7);
  background: rgba(99, 226, 183, 0.12);
}

.loop-primary-btn,
.route-rate-btn {
  min-height: 30px;
  padding: 6px 10px;
  border: 1px solid rgba(99, 226, 183, 0.3);
  border-radius: 8px;
  color: var(--sg-bg-ink, #0a0a0f);
  font-size: 12px;
  font-weight: 700;
  background: var(--sg-jade-bright, #63e2b7);
  cursor: pointer;
}

.loop-primary-btn:disabled,
.route-rate-btn:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.loop-primary-btn.full {
  width: 100%;
}

.route-recommend-list,
.spot-rating-box {
  display: grid;
  gap: 8px;
}

.route-recommend-card {
  display: grid;
  gap: 7px;
  padding: 10px;
  border: 1px solid rgba(99, 226, 183, 0.12);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.035);
}

.route-recommend-main {
  display: grid;
  gap: 3px;
}

.route-recommend-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.route-recommend-meta span {
  padding: 3px 6px;
  border: 1px solid rgba(99, 226, 183, 0.12);
  border-radius: 999px;
  background: rgba(99, 226, 183, 0.06);
}

.route-rate-btn {
  justify-self: start;
  color: var(--sg-jade-bright, #63e2b7);
  background: rgba(99, 226, 183, 0.08);
}

.spot-rating-box {
  padding-top: 8px;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
}

.spot-rating-box select,
.spot-rating-box textarea {
  width: 100%;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 8px;
  color: var(--sg-text-body, rgba(255, 255, 255, 0.88));
  background: rgba(255, 255, 255, 0.06);
  outline: none;
}

.spot-rating-box select {
  height: 34px;
  padding: 0 10px;
}

.spot-rating-box textarea {
  resize: vertical;
  min-height: 48px;
  padding: 8px 10px;
  font-size: 12px;
}

.rating-stars {
  display: flex;
  gap: 4px;
}

.rating-stars button {
  width: 30px;
  height: 30px;
  border: 1px solid rgba(244, 199, 101, 0.22);
  border-radius: 8px;
  color: rgba(244, 199, 101, 0.55);
  background: rgba(244, 199, 101, 0.05);
  cursor: pointer;
}

.rating-stars button.active {
  color: var(--sg-gold, #f4c765);
  background: rgba(244, 199, 101, 0.12);
}

.visitor-loop-message.error {
  color: var(--sg-red-bright, #e88080);
}

.message-list {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.msg {
  max-width: 85%;
}
.msg.user {
  align-self: flex-end;
}
.msg.assistant {
  align-self: flex-start;
}
.msg.system {
  align-self: center;
  max-width: 100%;
}
.msg-label {
  font-size: 11px;
  color: var(--sg-text-faint, rgba(255, 255, 255, 0.3));
  margin-bottom: 4px;
}
.msg-bubble {
  padding: 10px 14px;
  border-radius: 12px;
  font-size: 13px;
  line-height: 1.6;
}
.msg.user .msg-bubble {
  background: rgba(99, 226, 183, 0.12);
  border: 1px solid var(--sg-jade-border, rgba(99, 226, 183, 0.2));
  color: var(--sg-text-body, rgba(255, 255, 255, 0.88));
  border-bottom-right-radius: 4px;
}
.msg.assistant .msg-bubble {
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.08);
  color: var(--sg-text-secondary, rgba(255, 255, 255, 0.75));
  border-bottom-left-radius: 4px;
}
.msg.system .msg-bubble {
  background: rgba(248, 190, 82, 0.06);
  border: 1px solid rgba(248, 190, 82, 0.15);
  color: rgba(248, 190, 82, 0.7);
  font-size: 12px;
  text-align: center;
  border-radius: 8px;
}
.msg-sources {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
  margin-top: 6px;
  font-size: 11px;
  color: var(--sg-text-faint, rgba(255, 255, 255, 0.34));
}
.msg-source {
  max-width: 220px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  padding: 2px 7px;
  border: 1px solid rgba(99, 226, 183, 0.18);
  border-radius: 8px;
  color: rgba(99, 226, 183, 0.78);
  background: rgba(99, 226, 183, 0.06);
}
.msg-time {
  font-size: 10px;
  color: var(--sg-text-ghost, rgba(255, 255, 255, 0.2));
  margin-top: 4px;
}

.chat-input {
  display: flex;
  gap: 8px;
  padding: 12px 16px;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
}
.multimodal-btn {
  width: 40px;
  height: 40px;
  flex: 0 0 40px;
  border: 1px solid rgba(99, 226, 183, 0.3);
  border-radius: 50%;
  background: var(--sg-jade-bg, rgba(99, 226, 183, 0.06));
  color: var(--sg-jade-bright, #63e2b7);
  cursor: pointer;
  font-size: 22px;
  line-height: 1;
  transition: all 0.2s;
}
.multimodal-btn:hover,
.multimodal-btn.active {
  background: rgba(99, 226, 183, 0.15);
  border-color: var(--sg-jade-bright, #63e2b7);
}
.multimodal-btn:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}
.chat-input input {
  flex: 1;
  padding: 10px 14px;
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 8px;
  color: white;
  font-size: 13px;
  outline: none;
  transition: border-color 0.2s;
}
.chat-input input:focus {
  border-color: var(--sg-jade-bright, #63e2b7);
}
.send-btn {
  padding: 10px 18px;
  background: linear-gradient(135deg, var(--sg-jade-bright, #63e2b7), #18a058);
  border: none;
  border-radius: 8px;
  color: var(--sg-bg-ink, #0a0a0f);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}
.voice-btn {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  border: 1px solid rgba(99, 226, 183, 0.3);
  background: var(--sg-jade-bg, rgba(99, 226, 183, 0.06));
  font-size: 16px;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
  touch-action: none;
}
.voice-btn:hover {
  background: rgba(99, 226, 183, 0.15);
}
.voice-btn.recording {
  background: rgba(232, 128, 128, 0.15);
  border-color: rgba(232, 128, 128, 0.4);
  animation: pulse 1s infinite;
}
@keyframes pulse {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.6;
  }
}

/* 会话抽屉 */
.drawer-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  z-index: 100;
  display: flex;
  justify-content: flex-end;
}
.session-drawer {
  width: 340px;
  max-width: 85vw;
  height: 100%;
  background: #12121a;
  border-left: 1px solid rgba(255, 255, 255, 0.08);
  display: flex;
  flex-direction: column;
}
.drawer-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}
.drawer-header h3 {
  font-size: 14px;
  color: var(--sg-text-body, rgba(255, 255, 255, 0.88));
}
.drawer-close {
  background: none;
  border: none;
  color: rgba(255, 255, 255, 0.4);
  font-size: 16px;
  cursor: pointer;
}
.drawer-body {
  flex: 1;
  overflow-y: auto;
  padding: 12px;
}
.drawer-empty {
  text-align: center;
  color: rgba(255, 255, 255, 0.25);
  font-size: 13px;
  padding: 40px 0;
}
.session-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.15s;
  margin-bottom: 4px;
}
.session-item:hover {
  background: rgba(255, 255, 255, 0.04);
}
.session-item.active {
  background: rgba(99, 226, 183, 0.08);
  border: 1px solid rgba(99, 226, 183, 0.15);
}
.session-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.session-title {
  font-size: 13px;
  color: var(--sg-text-body, rgba(255, 255, 255, 0.88));
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.session-meta {
  font-size: 11px;
  color: rgba(255, 255, 255, 0.3);
}
.session-delete {
  background: none;
  border: none;
  font-size: 14px;
  cursor: pointer;
  opacity: 0.3;
  transition: opacity 0.2s;
  padding: 4px;
}
.session-delete:hover {
  opacity: 1;
}

@media (max-width: 1023px) {
  .dh-chat {
    width: min(360px, 45vw);
    min-width: 280px;
  }
}

@media (max-width: 768px) {
  .dh-view {
    flex-direction: column;
    height: calc(100dvh - 44px);
    padding-bottom: calc(42px + env(safe-area-inset-bottom, 0));
  }
  .dh-stage {
    flex: 1;
    height: auto;
    min-height: 0;
    padding-bottom: 220px;
    box-sizing: border-box;
  }
  .avatar-switcher {
    top: 48px;
    right: 12px;
    max-width: calc(100% - 24px);
    overflow-x: auto;
  }
  .avatar-choice {
    min-width: 68px;
    height: 32px;
    padding-right: 8px;
  }
  .dh-chat {
    width: 100%;
    max-width: none;
    min-width: unset;
    flex: 1;
    border-left: none;
    border-top: 1px solid rgba(255, 255, 255, 0.06);
    min-height: 0;
  }
  .dh-controls {
    width: 100%;
  }
  .audio-hint {
    width: 100%;
  }
  .location-panel {
    width: 100%;
    max-height: 64px;
    overflow: auto;
  }
  .stage-tools {
    left: 12px;
    right: 12px;
    bottom: 10px;
    max-height: 210px;
    overflow-y: auto;
  }
  .subtitle-bubble {
    max-width: 90%;
  }
  .emotion-bar {
    gap: 4px;
  }
  .emotion-btn {
    width: 28px;
    height: 28px;
    font-size: 13px;
  }
  .session-drawer {
    width: 100%;
    max-width: 100vw;
  }
}

/* Mobile bottom tab bar */
.mobile-tabs {
  display: flex;
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  z-index: 50;
  background: rgba(18, 18, 26, 0.95);
  backdrop-filter: blur(12px);
  border-top: 1px solid rgba(255, 255, 255, 0.08);
  padding-bottom: env(safe-area-inset-bottom, 0);
}
.mobile-tabs button {
  flex: 1;
  padding: 10px 0;
  background: none;
  border: none;
  color: rgba(255, 255, 255, 0.45);
  font-size: 13px;
  cursor: pointer;
  transition:
    color 0.2s,
    background 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
}
.mobile-tabs button.active {
  color: var(--sg-jade-bright, #63e2b7);
  background: rgba(99, 226, 183, 0.06);
}

/* Upgrade modal */
.upgrade-modal {
  position: fixed;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: min(400px, 90vw);
  background: var(--sg-bg-card, #1a1a2e);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 12px;
  overflow: hidden;
}
.upgrade-form {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.upgrade-hint {
  font-size: 13px;
  color: rgba(255, 255, 255, 0.5);
  margin: 0;
}
.upgrade-form input {
  padding: 10px 14px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 8px;
  color: rgba(255, 255, 255, 0.88);
  font-size: 14px;
  outline: none;
}
.upgrade-form input:focus {
  border-color: var(--sg-jade-bright, #63e2b7);
}
.upgrade-error {
  font-size: 12px;
  color: #e88080;
  margin: 0;
}
.upgrade-submit {
  padding: 10px;
  background: var(--sg-jade-bright, #63e2b7);
  border: none;
  border-radius: 8px;
  color: #1a1a2e;
  font-weight: 600;
  font-size: 14px;
  cursor: pointer;
  transition: opacity 0.2s;
}
.upgrade-submit:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.upgrade-btn {
  animation: pulse 2s infinite;
}
@keyframes pulse {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.6;
  }
}

/* 游客端自然景区服务风 */
.dh-view {
  height: calc(100vh - 58px);
  color: var(--visitor-ink);
  background:
    radial-gradient(ellipse at 22% 8%, rgba(232, 220, 199, 0.36), transparent 36%),
    radial-gradient(ellipse at 78% 90%, rgba(198, 107, 61, 0.18), transparent 34%),
    linear-gradient(135deg, var(--visitor-sage), var(--visitor-moss));
}

.dh-stage {
  margin: 16px 0 16px 18px;
  border: 1px solid rgba(232, 220, 199, 0.24);
  border-radius: 30px;
  background:
    radial-gradient(ellipse at 50% 42%, rgba(232, 220, 199, 0.36), transparent 48%), rgba(232, 220, 199, 0.18);
  box-shadow: var(--visitor-shadow);
  overflow: hidden;
}

.dh-stage::before {
  position: absolute;
  inset: 0;
  pointer-events: none;
  content: '';
  opacity: 0.12;
  background:
    radial-gradient(circle at 20% 20%, rgba(232, 220, 199, 0.85) 0 1px, transparent 1px),
    radial-gradient(circle at 72% 68%, rgba(38, 51, 31, 0.62) 0 1px, transparent 1px);
  background-size:
    18px 18px,
    26px 26px;
}

.dh-status {
  z-index: 3;
  padding: 8px 12px;
  border: 1px solid rgba(232, 220, 199, 0.34);
  border-radius: 999px;
  background: rgba(232, 220, 199, 0.72);
}

.status-dot.online {
  background: var(--visitor-moss);
  box-shadow: none;
}

.status-text {
  color: var(--visitor-ink);
  font-weight: 700;
}

.avatar-switcher {
  z-index: 3;
  border-color: rgba(232, 220, 199, 0.34);
  border-radius: 18px;
  background: rgba(232, 220, 199, 0.72);
}

.avatar-lock-label,
.avatar-choice {
  color: var(--visitor-muted);
}

.avatar-choice {
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.24);
}

.avatar-choice:hover,
.avatar-choice.active {
  color: var(--visitor-sand);
  border-color: var(--visitor-moss);
  background: var(--visitor-moss);
}

.subtitle-bubble {
  position: relative;
  z-index: 3;
  max-width: min(520px, 86%);
  border-color: var(--visitor-line);
  border-radius: 24px;
  color: var(--visitor-ink);
  background: var(--visitor-sand);
  box-shadow: 0 16px 34px rgba(38, 51, 31, 0.18);
}

.dh-controls {
  z-index: 3;
}

.ctrl-btn {
  min-height: 38px;
  border-color: rgba(232, 220, 199, 0.34);
  border-radius: 999px;
  color: var(--visitor-ink);
  background: rgba(232, 220, 199, 0.72);
}

.ctrl-btn:hover,
.ctrl-btn.sound.ready {
  color: var(--visitor-sand);
  background: var(--visitor-moss);
}

.ctrl-btn.sound,
.ctrl-btn.danger,
.ctrl-btn.sound.error {
  color: var(--visitor-ink);
  border-color: var(--visitor-line);
}

.audio-hint {
  z-index: 3;
  color: rgba(232, 220, 199, 0.9);
}

.audio-hint.ready,
.audio-hint.error {
  color: var(--visitor-sand);
}

.chat-resizer::after {
  background: rgba(232, 220, 199, 0.42);
}

.dh-chat {
  margin: 16px 18px 16px 0;
  border: 1px solid var(--visitor-line);
  border-radius: 28px;
  background: var(--visitor-sand);
  box-shadow: var(--visitor-shadow);
  overflow: hidden;
}

.chat-header,
.quick-asks,
.chat-input {
  border-color: var(--visitor-line);
}

.chat-header {
  background: rgba(255, 255, 255, 0.18);
}

.chat-header-top h2 {
  color: var(--visitor-ink);
  font-family: Georgia, 'Times New Roman', serif;
  font-size: 20px;
}

.chat-header p,
.msg-label,
.msg-time,
.search-result-meta,
.search-empty {
  color: var(--visitor-muted);
}

.icon-btn,
.quick-asks button,
.search-bar input,
.chat-input input,
.multimodal-btn,
.voice-btn {
  border-color: var(--visitor-line);
  border-radius: 999px;
  color: var(--visitor-ink);
  background: rgba(255, 255, 255, 0.28);
}

.icon-btn:hover,
.icon-btn.active,
.quick-asks button:hover,
.multimodal-btn:hover,
.multimodal-btn.active,
.voice-btn:hover {
  border-color: var(--visitor-line-strong);
  background: rgba(96, 108, 56, 0.12);
}

.search-bar input,
.chat-input input,
.multimodal-btn {
  color: var(--visitor-ink);
}

.answer-visual-card,
.visitor-loop-panel,
.route-recommend-card,
.spot-rating-box,
.msg-bubble,
.search-result-item {
  border-color: var(--visitor-line);
}

.answer-visual-card,
.visitor-loop-panel,
.route-recommend-card {
  border-radius: 20px;
  background: rgba(255, 255, 255, 0.22);
}

.answer-visual-thumb {
  color: var(--visitor-sand);
  background: var(--visitor-terracotta);
}

.answer-visual-header strong,
.visitor-loop-header strong,
.spot-rating-header strong,
.route-recommend-main strong,
.session-title {
  color: var(--visitor-ink);
}

.answer-visual-header span,
.visitor-loop-header span,
.spot-rating-header span,
.route-recommend-main span,
.route-recommend-meta span,
.visitor-loop-message,
.msg-source {
  color: var(--visitor-moss);
  border-color: var(--visitor-line);
  background: rgba(96, 108, 56, 0.1);
}

.answer-visual-points i {
  color: var(--visitor-muted);
  background: rgba(255, 255, 255, 0.24);
}

.visitor-profile-option,
.spot-rating-box select,
.spot-rating-box textarea,
.rating-stars button {
  color: var(--visitor-ink);
  border-color: var(--visitor-line);
  background: rgba(255, 255, 255, 0.28);
}

.visitor-profile-option.active,
.rating-stars button.active {
  border-color: var(--visitor-moss);
  background: rgba(96, 108, 56, 0.12);
}

.visitor-profile-option span {
  color: var(--visitor-muted);
}

.loop-primary-btn {
  color: var(--visitor-sand);
  border-color: var(--visitor-moss);
  background: var(--visitor-moss);
}

.route-rate-btn {
  color: var(--visitor-moss);
  border-color: var(--visitor-line);
  background: rgba(96, 108, 56, 0.1);
}

.msg.user .msg-bubble {
  color: var(--visitor-sand);
  border-color: var(--visitor-moss);
  background: var(--visitor-moss);
}

.msg.assistant .msg-bubble {
  color: var(--visitor-ink);
  background: rgba(255, 255, 255, 0.34);
}

.msg.system .msg-bubble {
  color: var(--visitor-ink);
  border-color: rgba(198, 107, 61, 0.22);
  background: rgba(198, 107, 61, 0.12);
}

.send-btn {
  border-radius: 999px;
  color: var(--visitor-sand);
  background: var(--visitor-moss);
}

.drawer-overlay {
  background: rgba(38, 51, 31, 0.48);
}

.session-drawer,
.upgrade-modal {
  color: var(--visitor-ink);
  border-color: var(--visitor-line);
  background: var(--visitor-sand);
}

.drawer-header {
  border-color: var(--visitor-line);
}

.drawer-header h3,
.upgrade-hint,
.upgrade-form input {
  color: var(--visitor-ink);
}

.drawer-close {
  color: var(--visitor-muted);
}

.session-item:hover,
.session-item.active {
  background: rgba(96, 108, 56, 0.1);
}

.session-meta,
.drawer-empty {
  color: var(--visitor-muted);
}

.upgrade-form input {
  border-color: var(--visitor-line);
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.28);
}

.upgrade-submit {
  border-radius: 999px;
  color: var(--visitor-sand);
  background: var(--visitor-moss);
}

.mobile-tabs {
  border-top-color: var(--visitor-line);
  background: var(--visitor-sand);
}

.mobile-tabs button {
  color: var(--visitor-muted);
}

.mobile-tabs button.active {
  color: var(--visitor-sand);
  background: var(--visitor-moss);
}

@media (max-width: 768px) {
  .dh-view {
    height: calc(100dvh - 58px);
  }

  .dh-stage {
    margin: 10px;
    border-radius: 24px;
  }

  .dh-chat {
    margin: 0 10px 10px;
    border-radius: 24px;
  }
}
</style>
