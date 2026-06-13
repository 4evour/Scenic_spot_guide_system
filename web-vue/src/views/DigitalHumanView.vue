<script setup lang="ts">
import { computed, onMounted, onUnmounted, onErrorCaptured, reactive, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute } from 'vue-router';
import { AudioPlaybackController, type PlaybackCue } from '../services/audioPlayback';
import { VtuberSocketClient } from '../services/vtuberSocket';
import { streamTTS } from '../services/ttsApi';
import type { ChatMessage, ConversationState, EmotionToken, VtuberMessage, Live2DExpression } from '../types/digitalHuman';
import Live2DStage from '../components/Live2DStage.vue';
import MarkdownRenderer from '../components/MarkdownRenderer.vue';
import { useSessionStore } from '../stores/session';

const { t } = useI18n();
const route = useRoute();

const uid = (): string =>
  typeof crypto.randomUUID === 'function'
    ? crypto.randomUUID()
    : 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, c => {
        const r = (Math.random() * 16) | 0;
        return (c === 'x' ? r : (r & 0x3) | 0x8).toString(16);
      });

// 会话持久化
const sessionStore = useSessionStore();
const DH_SESSION_KEY = 'sg_dh_session_id';

function getOrCreateSessionId(): string {
  let sid = localStorage.getItem(DH_SESSION_KEY);
  if (!sid) {
    sid = uid();
    localStorage.setItem(DH_SESSION_KEY, sid);
  }
  return sid;
}

const input = ref('');
const mouthOpen = ref(0);
const transcriptBuffer = ref('');
const hasActiveTurn = ref(false);
const isPlaybackActive = ref(false);
const isVoiceListening = ref(false);
const isVoiceStarting = ref(false);
const searchQuery = ref('');
const showSearch = ref(false);
const showSessionDrawer = ref(false);
const isSearching = ref(false);
const searchResults = ref<{ text: string; time: string }[]>([]);
const typewriterStreaming = ref(false);
const mobileTab = ref<'avatar' | 'chat'>('avatar');
const isMobileView = ref(window.innerWidth < 768);
function onWindowResize() { isMobileView.value = window.innerWidth < 768; }

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
      time: new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }),
    },
  ] as ChatMessage[],
});

let socket: VtuberSocketClient | null = null;
let recognition: SpeechRecognition | null = null;
let conversationTurn = 0;
let interruptedTurn = -1;
let fallbackTimer = 0;
let blockIncomingPlayback = false;
let waitingForFreshServerTurn = false;
let lastAssistantSpeechText = '';
let searchDebounceTimer = 0;

const audio = new AudioPlaybackController({
  onStart: (text: string | undefined, cue: PlaybackCue | undefined) => {
    isPlaybackActive.value = true;
    hasActiveTurn.value = true;
    typewriterStreaming.value = true;
    if (text) showAssistantSpeech(text);
    state.conversation = 'speaking';
    state.expression = resolveSpeechExpression(cue?.expression, text);
  },
  onEnd: () => {
    isPlaybackActive.value = false;
    mouthOpen.value = 0;
    typewriterStreaming.value = false;
    if (state.conversation === 'speaking') {
      state.conversation = 'idle';
      state.expression = 'neutral';
    }
  },
  onVolume: (volume: number) => {
    mouthOpen.value = volume;
  },
});

const statusLabel = computed(() => {
  if (state.conversation === 'connecting') return t('dh.connecting');
  if (state.conversation === 'thinking') return t('dh.searching');
  if (state.conversation === 'speaking') return t('dh.speaking');
  if (state.conversation === 'listening') return t('dh.listening');
  if (state.conversation === 'interrupted') return t('dh.interruptedStatus');
  if (state.conversation === 'error') return t('dh.error');
  return state.connected ? t('dh.online') : t('dh.offline');
});

const canInterrupt = computed(
  () =>
    hasActiveTurn.value ||
    isPlaybackActive.value ||
    isVoiceListening.value ||
    isVoiceStarting.value ||
    ['speaking', 'thinking', 'listening'].includes(state.conversation),
);

function nowTime() {
  return new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });
}

const ALL_EMOTION_TOKENS = ['neutral', 'joy', 'sadness', 'surprise', 'anger', 'fear', 'disgust'];

function stripEmotionTags(text: string) {
  const pattern = new RegExp(`\\[(${ALL_EMOTION_TOKENS.join('|')})\\]\\s*`, 'gi');
  return text.replace(pattern, '').trim() || text;
}

function expressionFromText(text?: string): Live2DExpression | undefined {
  if (!text) return undefined;
  const pattern = new RegExp(`\\[(${ALL_EMOTION_TOKENS.join('|')})\\]`, 'i');
  const match = text.match(pattern);
  return match ? emotionTokenToExpression(match[1] as EmotionToken) : undefined;
}

function expressionFromToken(token?: string | number): Live2DExpression | undefined {
  if (token === undefined || token === null) return undefined;
  if (typeof token === 'number') {
    if (token === 0) return 'neutral';
    if (token === 1) return 'thinking';
    if (token === 2) return 'interrupted';
    if (token === 3) return 'happy';
    return undefined;
  }

  const normalized = token.toLowerCase().replace(/^\[|\]$/g, '');
  if (/^\d+$/.test(normalized)) return expressionFromToken(Number(normalized));
  if (normalized.startsWith('exp_')) {
    if (normalized === 'exp_01') return 'happy';
    if (normalized === 'exp_02') return 'neutral';
    if (normalized === 'exp_03') return 'angry';
    if (normalized === 'exp_04') return 'thinking';
    if (normalized === 'exp_05') return 'surprised';
    if (normalized === 'exp_06') return 'sad';
    if (normalized === 'exp_07') return 'interrupted';
    if (normalized === 'exp_08') return 'blush';
  }
  return emotionTokenToExpression(normalized as EmotionToken);
}

function emotionTokenToExpression(token?: string): Live2DExpression | undefined {
  if (!token) return undefined;
  const t = token.toLowerCase().replace(/^\[|\]$/g, '');
  if (t === 'joy' || t === 'happy') return 'happy';
  if (t === 'neutral') return 'neutral';
  if (t === 'anger' || t === 'angry') return 'angry';
  if (t === 'sadness' || t === 'sad') return 'sad';
  if (t === 'surprise' || t === 'surprised') return 'surprised';
  if (t === 'fear' || t === 'thinking') return 'thinking';
  if (t === 'disgust' || t === 'interrupted') return 'interrupted';
  return undefined;
}

function resolveSpeechExpression(cueExpression?: string, text?: string): Live2DExpression {
  return expressionFromToken(cueExpression) || expressionFromText(text) || 'happy';
}

/** Persist message to both localStorage and backend */
async function persistMessage(role: ChatMessage['role'], content: string) {
  // localStorage backup (always works offline)
  try {
    const key = `sg_dh_msgs_${getOrCreateSessionId()}`;
    const existing = JSON.parse(localStorage.getItem(key) || '[]');
    existing.push({ role, content, time: Date.now() });
    // Keep only last 200 messages
    if (existing.length > 200) existing.splice(0, existing.length - 200);
    localStorage.setItem(key, JSON.stringify(existing));

    // 清理 7 天前的其他 session 消息
    const MAX_AGE_MS = 7 * 24 * 60 * 60 * 1000;
    for (let i = localStorage.length - 1; i >= 0; i--) {
      const k = localStorage.key(i);
      if (k?.startsWith('sg_dh_msgs_') && k !== key) {
        try {
          const msgs = JSON.parse(localStorage.getItem(k) || '[]');
          if (msgs.length > 0 && Date.now() - msgs[msgs.length - 1].time > MAX_AGE_MS) {
            localStorage.removeItem(k);
          }
        } catch { localStorage.removeItem(k!); }
      }
    }
  } catch { /* ignore */ }

  // Backend persistence
  sessionStore.appendMessage(role, content);
  // Try to save to backend via session store
  try {
    await sessionStore.saveMessage(getOrCreateSessionId(), role, content);
  } catch { /* silent retry next time */ }
}

function addMessage(role: ChatMessage['role'], text: string) {
  const displayText = stripEmotionTags(text);
  state.messages.push({
    id: uid(),
    role,
    text: displayText,
    time: nowTime(),
  });
  state.subtitle = displayText;
  if (role === 'user' || role === 'assistant') {
    void persistMessage(role, displayText);
  }
}

function showAssistantSpeech(text: string) {
  const displayText = stripEmotionTags(text);
  state.subtitle = displayText;
  if (displayText === lastAssistantSpeechText) return;
  lastAssistantSpeechText = displayText;
  typewriterStreaming.value = true;
  state.messages.push({
    id: uid(),
    role: 'assistant',
    text: displayText,
    time: nowTime(),
  });
  void persistMessage('assistant', displayText);
}

function connectSocket() {
  state.conversation = 'connecting';
  socket = new VtuberSocketClient(undefined, {
    onOpen: () => {
      state.connected = true;
      state.conversation = 'idle';
      addMessage('system', t('dh.connected'));
    },
    onClose: () => {
      state.connected = false;
      if (state.conversation !== 'interrupted') state.conversation = 'idle';
    },
    onError: (message: string) => {
      addMessage('system', message);
      state.conversation = 'idle';
      state.expression = 'neutral';
    },
    onMessage: handleSocketMessage,
  });
  socket.connect();
}

function handleSocketMessage(message: VtuberMessage) {
  if (blockIncomingPlayback && ['audio', 'full-text', 'backend-synth-complete'].includes(message.type)) {
    return;
  }

  if (message.type === 'full-text' && message.text && message.text !== 'Thinking...') {
    transcriptBuffer.value = message.text;
    state.expression = expressionFromText(message.text) || state.expression;
  }

  if (message.type === 'user-input-transcription' && message.text) {
    addMessage('user', message.text);
  }

  if (message.type === 'control' && message.text === 'conversation-chain-start') {
    if (blockIncomingPlayback && !waitingForFreshServerTurn) return;
    conversationTurn += 1;
    interruptedTurn = -1;
    hasActiveTurn.value = true;
    blockIncomingPlayback = false;
    waitingForFreshServerTurn = false;
    audio.resume();
    state.conversation = 'thinking';
    state.expression = 'thinking';
    state.subtitle = t('dh.thinking');
  }

  if (message.type === 'control' && message.text === 'conversation-chain-end') {
    hasActiveTurn.value = isPlaybackActive.value;
    typewriterStreaming.value = false;
    if (state.conversation !== 'interrupted') {
      state.conversation = 'idle';
      state.expression = 'neutral';
    }
  }

  if (message.type === 'control' && message.text === 'interrupt') {
    interruptAnswer();
    return;
  }

  if (message.type === 'interrupt-signal') {
    audio.interrupt();
    mouthOpen.value = 0;
    hasActiveTurn.value = false;
    isPlaybackActive.value = false;
    typewriterStreaming.value = false;
    blockIncomingPlayback = true;
    waitingForFreshServerTurn = false;
    interruptedTurn = conversationTurn;
    state.conversation = 'interrupted';
    state.expression = 'interrupted';
    state.subtitle = t('dh.interruptReceived');
    return;
  }

  if (message.type === 'audio') {
    if (interruptedTurn === conversationTurn) return;
    const text = message.display_text?.text || transcriptBuffer.value || t('dh.fallbackSpeech');
    const expression = expressionFromToken(message.actions?.expressions?.[0]) || expressionFromText(text);
    const cue = {
      volumes: message.volumes,
      sliceLengthMs: message.slice_length,
      expression,
    };
    typewriterStreaming.value = true;
    if (message.audio) audio.enqueueBase64Wav(message.audio, text, cue);
    else void audio.playTextFallback(text, cue);
  }

  if (message.type === 'backend-synth-complete') {
    hasActiveTurn.value = isPlaybackActive.value;
    typewriterStreaming.value = false;
    if (state.conversation === 'thinking') {
      state.conversation = 'idle';
      state.expression = 'neutral';
    }
  }

  if (message.type === 'error') {
    const errMsg = message.message || ''
    if (errMsg.startsWith('Conversation error')) return;
    addMessage('system', errMsg || t('dh.speechError'));
    state.conversation = 'error';
  }
}

function sendText() {
  const text = input.value.trim();
  if (!text) return;

  window.clearTimeout(fallbackTimer);
  conversationTurn += 1;
  interruptedTurn = -1;
  hasActiveTurn.value = true;
  const shouldWaitForServerTurn = Boolean(blockIncomingPlayback && socket);
  addMessage('user', text);
  lastAssistantSpeechText = '';
  input.value = '';
  state.conversation = 'thinking';
  state.expression = 'thinking';
  state.subtitle = t('dh.thinking');
  transcriptBuffer.value = '';

  const sent = socket?.sendText(text);
  if (!shouldWaitForServerTurn || !sent) {
    blockIncomingPlayback = false;
    waitingForFreshServerTurn = false;
    audio.resume();
  } else {
    waitingForFreshServerTurn = true;
  }
  if (!sent) {
    const fallback = buildFallbackAnswer(text);
    const turn = conversationTurn;
    fallbackTimer = window.setTimeout(async () => {
      if (interruptedTurn === turn) return;
      showAssistantSpeech(fallback);
      const cue = { expression: expressionFromText(fallback) || 'happy' as const };

      try {
        const response = await streamTTS({ text: fallback, voice: 'female_xiaoxiao' });
        const streamed = await audio.enqueueStream(response, fallback, cue);
        if (!streamed) {
          void audio.playTextFallback(fallback, cue);
        }
      } catch {
        void audio.playTextFallback(fallback, cue);
      }
    }, 420);
  }
}

function interruptAnswer() {
  recognition?.stop();
  window.clearTimeout(fallbackTimer);
  interruptedTurn = conversationTurn;
  hasActiveTurn.value = false;
  isPlaybackActive.value = false;
  isVoiceListening.value = false;
  isVoiceStarting.value = false;
  typewriterStreaming.value = false;
  blockIncomingPlayback = true;
  waitingForFreshServerTurn = false;
  audio.interrupt();
  const serverInterrupted = socket?.interrupt(state.subtitle) || false;
  mouthOpen.value = 0;
  state.interruptCount += 1;
  state.conversation = 'interrupted';
  state.expression = 'interrupted';
  state.subtitle = serverInterrupted
    ? t('dh.interruptedServer')
    : t('dh.interruptedLocal');
  addMessage('system', state.subtitle);

  window.setTimeout(() => {
    if (state.conversation === 'interrupted') {
      state.conversation = 'idle';
      state.expression = 'neutral';
    }
  }, 1200);
}

function toggleVoice() {
  const Recognition = window.SpeechRecognition || window.webkitSpeechRecognition;
  if (!Recognition) {
    addMessage('system', t('dh.browserNotSupported'));
    return;
  }

  if (recognition && (isVoiceListening.value || isVoiceStarting.value)) {
    recognition.stop();
    return;
  }

  const nextRecognition = new Recognition();
  recognition = nextRecognition;
  nextRecognition.lang = 'zh-CN';
  nextRecognition.interimResults = false;
  nextRecognition.continuous = false;
  nextRecognition.onstart = () => {
    isVoiceStarting.value = false;
    isVoiceListening.value = true;
    state.conversation = 'listening';
    state.expression = 'surprised';
    state.subtitle = t('dh.listening');
  };
  nextRecognition.onresult = event => {
    const spokenText = event.results?.[0]?.[0]?.transcript?.trim();
    if (!spokenText) return;
    input.value = spokenText;
    sendText();
  };
  nextRecognition.onerror = event => {
    const detail = event.error === 'not-allowed' || event.error === 'service-not-allowed'
      ? t('dh.noMicPermission')
      : t('dh.micNotSupported');
    addMessage('system', detail);
    isVoiceStarting.value = false;
    isVoiceListening.value = false;
    if (state.conversation === 'listening') {
      state.conversation = 'idle';
      state.expression = 'neutral';
    }
  };
  nextRecognition.onend = () => {
    isVoiceStarting.value = false;
    isVoiceListening.value = false;
    if (recognition === nextRecognition) recognition = null;
    if (state.conversation === 'listening') {
      state.conversation = 'idle';
      state.expression = 'neutral';
    }
  };

  try {
    isVoiceStarting.value = true;
    nextRecognition.start();
  } catch {
    isVoiceStarting.value = false;
    isVoiceListening.value = false;
    recognition = null;
    addMessage('system', t('dh.micStartFailed'));
  }
}

// --- Voice button long-press support ---
let voicePressTimer = 0;
function onVoicePointerDown() {
  voicePressTimer = window.setTimeout(() => {
    toggleVoice();
  }, 500);
}
function onVoicePointerUp() {
  window.clearTimeout(voicePressTimer);
  // Short click still toggles voice
  if (!isVoiceListening.value && !isVoiceStarting.value) {
    toggleVoice();
  }
}

function onVoicePointerLeave() {
  window.clearTimeout(voicePressTimer);
}

// --- Context-aware quick ask ---
const quickAskBase = computed(() => [
  { label: t('dh.quickAsk.buddhaHeight'), query: t('dh.quickAsk.buddhaHeight') },
  { label: t('dh.quickAsk.withKids'), query: t('dh.quickAsk.withKids') },
  { label: t('dh.quickAsk.routeLabel'), query: t('dh.quickAsk.route') },
  { label: t('dh.quickAsk.hoursLabel'), query: t('dh.quickAsk.hours') },
]);

const followUpQuestions = computed(() => {
  const lastMsg = state.messages.filter(m => m.role === 'assistant').pop();
  if (!lastMsg) return [];
  const text = lastMsg.text;
  const questions: { label: string; query: string }[] = [];
  if (text.includes('路线') || text.includes('route')) {
    questions.push({ label: t('dh.quickAsk.routeDetail'), query: '这条路线有哪些主要景点？' });
    questions.push({ label: t('dh.quickAsk.routeTime'), query: '需要多长时间走完？' });
  }
  if (text.includes('历史') || text.includes('history')) {
    questions.push({ label: t('dh.quickAsk.historyDetail'), query: '能讲讲这里的历史故事吗？' });
  }
  if (text.includes('大佛') || text.includes('Buddha')) {
    questions.push({ label: t('dh.quickAsk.buddhaStory'), query: '灵山大佛有什么特别之处？' });
  }
  return questions.slice(0, 3);
});

const quickAskItems = computed(() => {
  const followUps = followUpQuestions.value;
  if (followUps.length > 0) return followUps;
  return quickAskBase.value;
});

function quickAsk(query: string) {
  input.value = query;
  sendText();
}

// --- Search ---
function toggleSearch() {
  showSearch.value = !showSearch.value;
  if (!showSearch.value) {
    searchQuery.value = '';
    searchResults.value = [];
  }
}

function onSearchInput() {
  window.clearTimeout(searchDebounceTimer);
  const q = searchQuery.value.trim();
  if (!q) {
    searchResults.value = [];
    return;
  }
  searchDebounceTimer = window.setTimeout(() => {
    isSearching.value = true;
    // Search in local messages
    const results = state.messages
      .filter(m => m.text.toLowerCase().includes(q.toLowerCase()))
      .map(m => ({ text: m.text, time: m.time }));
    searchResults.value = results.slice(0, 20);
    isSearching.value = false;
  }, 300);
}

// --- Session drawer ---
function toggleSessionDrawer() {
  showSessionDrawer.value = !showSessionDrawer.value;
  if (showSessionDrawer.value) {
    void sessionStore.loadSessions();
  }
}

async function switchSession(sessionId: string) {
  sessionStore.setCurrentSession(sessionId);
  try {
    await sessionStore.loadMessages(sessionId, 50);
    if (sessionStore.messages.length > 0) {
      state.messages = sessionStore.messages.map(m => ({
        id: `hist-${m.id}`,
        role: m.role as ChatMessage['role'],
        text: m.content,
        time: new Date(m.created_at).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }),
      }));
    }
  } catch {
    state.messages = [{
      id: uid(),
      role: 'assistant',
      text: t('dh.greeting'),
      time: nowTime(),
    }];
  }
  showSessionDrawer.value = false;
}

async function deleteSessionAndClose(sessionId: string) {
  await sessionStore.deleteSession(sessionId);
}

// --- Hit area handlers ---
function onHeadClick() {
  // Random expression on head click
  const exprs: Live2DExpression[] = ['happy', 'surprised', 'blush'];
  const random = exprs[Math.floor(Math.random() * exprs.length)];
  state.expression = random;
  window.setTimeout(() => {
    if (state.expression === random) state.expression = 'neutral';
  }, 2000);
}

function onBodyClick() {
  // Playful feedback on body click
  addMessage('system', '👋 你戳了戳小灵');
}

// --- LocalStorage recovery ---
function loadLocalMessages(): ChatMessage[] {
  try {
    const key = `sg_dh_msgs_${getOrCreateSessionId()}`;
    const raw = localStorage.getItem(key);
    if (!raw) return [];
    const items = JSON.parse(raw) as Array<{ role: string; content: string; time: number }>;
    return items.map(item => ({
      id: uid(),
      role: item.role as ChatMessage['role'],
      text: item.content,
      time: new Date(item.time).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }),
    }));
  } catch {
    return [];
  }
}

function buildFallbackAnswer(text: string) {
  const lower = text.toLowerCase();
  const isEn = lower.match(/[a-z]/) && !/[一-鿿]/.test(lower);

  if (isEn) {
    if (lower.includes('route') || lower.includes('recommend')) {
      return 'I recommend the "Gate – Cultural Corridor – Core Landscapes – Viewing Platform – Creative Station" route, about 90 minutes. History lovers should focus on inscriptions, architecture, and local legends.';
    }
    if (lower.includes('hour') || lower.includes('open') || lower.includes('time')) {
      return 'The scenic area is open from 8:00 AM to 5:30 PM. Last entry at 4:30 PM. Morning and evening visits are recommended on holidays.';
    }
    if (lower.includes('history') || lower.includes('culture')) {
      return 'Lingshan Scenic Area features ancient pathways, traditional architecture, and folk culture exhibits. Best explored as "nature + cultural stories."';
    }
    return 'I\'ve prepared a demo narration based on your question. It combines local knowledge, visitor interests, and real-time conditions to recommend the best route and focus.';
  }

  if (text.includes('路线') || text.includes('推荐')) {
    return '推荐你走"山门迎宾-文化长廊-核心景观-观景台-文创驿站"路线，全程约 90 分钟。喜欢历史的游客可以把讲解重点放在碑刻、建筑形制和地方传说上。';
  }
  if (text.includes('开放') || text.includes('时间')) {
    return '景区开放时间为 08:00 到 17:30，建议 16:30 前入园。节假日客流较高，可以优先选择上午或傍晚时段。';
  }
  if (text.includes('历史') || text.includes('文化')) {
    return '灵山景区以山水格局和地方历史文化为核心，沿线包含古道遗存、传统建筑和民俗展示点，适合用"自然景观加人文故事"的方式游览。';
  }
  return '我已根据你的问题生成一段演示讲解：这里会结合本地知识库、游客兴趣和实时客流，为你推荐更合适的景点顺序与讲解重点。';
}

onErrorCaptured((err) => {
  console.error('数字人组件错误:', err);
  state.conversation = 'error';
  state.subtitle = t('dh.componentError');
  return false;
});

onMounted(async () => {
  const sessionId = getOrCreateSessionId();
  sessionStore.setCurrentSession(sessionId);
  try {
    await sessionStore.loadMessages(sessionId, 50);
    if (sessionStore.messages.length > 0) {
      state.messages = sessionStore.messages.map(m => ({
        id: `hist-${m.id}`,
        role: m.role as ChatMessage['role'],
        text: m.content,
        time: new Date(m.created_at).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }),
      }));
    }
  } catch {
    // Try localStorage fallback
    const localMsgs = loadLocalMessages();
    if (localMsgs.length > 0) {
      state.messages = localMsgs;
    }
  }
  connectSocket();
  window.addEventListener('resize', onWindowResize);

  // QR 扫码后自动提问
  const autoAsk = route.query.auto_ask as string;
  if (autoAsk) {
    // 等待 socket 连接后自动发送
    const waitForSocket = setInterval(() => {
      if (socket && state.connected) {
        clearInterval(waitForSocket);
        input.value = autoAsk;
        sendText();
      }
    }, 200);
    // 最多等 5 秒
    setTimeout(() => clearInterval(waitForSocket), 5000);
  }
});

onUnmounted(() => {
  window.clearTimeout(fallbackTimer);
  window.clearTimeout(voicePressTimer);
  window.clearTimeout(searchDebounceTimer);
  socket?.disconnect();
  recognition?.stop();
  audio.interrupt();
  hasActiveTurn.value = false;
  isPlaybackActive.value = false;
  isVoiceListening.value = false;
  isVoiceStarting.value = false;
  mouthOpen.value = 0;
  window.removeEventListener('resize', onWindowResize);
});
</script>

<template>
  <main class="dh-view">
    <!-- 左侧：数字人展示区 -->
    <section class="dh-stage" v-show="mobileTab === 'avatar' || !isMobileView">
      <div class="dh-status">
        <span class="status-dot" :class="{ online: state.connected }"></span>
        <span class="status-text">{{ statusLabel }}</span>
      </div>

      <Live2DStage
        :state="state.conversation"
        :mouth-open="mouthOpen"
        :expression="state.expression"
        @head-click="onHeadClick"
        @body-click="onBodyClick"
      />

      <!-- 表情指示器 -->
      <div class="emotion-bar">
        <button
          v-for="expr in (['happy', 'neutral', 'thinking', 'surprised', 'sad', 'angry', 'blush'] as const)"
          :key="expr"
          class="emotion-btn"
          :class="{ active: state.expression === expr }"
          @click="state.expression = expr"
        >
          {{ { happy: '😊', neutral: '😐', thinking: '🤔', surprised: '😮', sad: '😢', angry: '😠', blush: '😳' }[expr] }}
        </button>
      </div>

      <!-- 字幕气泡 -->
      <div class="subtitle-bubble">
        {{ state.subtitle }}
      </div>

      <!-- 控制栏 -->
      <div class="dh-controls">
        <button class="ctrl-btn danger" :disabled="!canInterrupt" @click="interruptAnswer">
          {{ $t('dh.interrupt') }}
        </button>
        <button class="ctrl-btn" @click="connectSocket">{{ $t('dh.reconnect') }}</button>
        <span class="interrupt-count">{{ $t('dh.interruptCount', { count: state.interruptCount }) }}</span>
      </div>
    </section>

    <!-- 右侧：聊天面板 -->
    <aside class="dh-chat" v-show="mobileTab === 'chat' || !isMobileView">
      <div class="chat-header">
        <div class="chat-header-top">
          <h2>{{ $t('dh.title') }}</h2>
          <div class="header-actions">
            <button class="icon-btn" :class="{ active: showSearch }" @click="toggleSearch" title="搜索">
              🔍
            </button>
            <button class="icon-btn" @click="toggleSessionDrawer" title="历史会话">
              📋
            </button>
          </div>
        </div>
        <p>{{ $t('dh.subtitle') }}</p>

        <!-- 搜索栏 -->
        <div v-if="showSearch" class="search-bar">
          <input
            v-model="searchQuery"
            :placeholder="$t('dh.searchPlaceholder')"
            @input="onSearchInput"
            autofocus
          />
          <button v-if="searchQuery" class="search-clear" @click="searchQuery = ''; searchResults = []">✕</button>
        </div>
      </div>

      <!-- 快捷提问 -->
      <div class="quick-asks">
        <button
          v-for="item in quickAskItems"
          :key="item.label"
          @click="quickAsk(item.query)"
        >
          {{ item.label }}
        </button>
        <button
          v-if="followUpQuestions.length > 0"
          class="refresh-btn"
          @click="state.expression = 'neutral'"
          title="刷新"
        >
          🔄
        </button>
      </div>

      <!-- 搜索结果 -->
      <div v-if="showSearch && searchResults.length > 0" class="search-results">
        <div
          v-for="(result, i) in searchResults"
          :key="i"
          class="search-result-item"
          v-html="highlightMatch(result.text, searchQuery)"
        />
      </div>
      <div v-else-if="showSearch && searchQuery && !isSearching" class="search-empty">
        {{ $t('dh.searchNoResults') }}
      </div>

      <!-- 消息列表 -->
      <div v-if="!showSearch || !searchQuery" class="message-list" @scroll.passive>
        <article v-for="msg in state.messages" :key="msg.id" :class="['msg', msg.role]">
          <div class="msg-label">
            {{ msg.role === 'user' ? $t('dh.user') : msg.role === 'assistant' ? $t('dh.assistant') : $t('dh.system') }}
          </div>
          <div class="msg-bubble">
            <MarkdownRenderer
              v-if="msg.role === 'assistant'"
              :text="msg.text"
              :streaming="typewriterStreaming && msg.id === state.messages[state.messages.length - 1]?.id"
            />
            <span v-else>{{ msg.text }}</span>
          </div>
          <div class="msg-time">{{ msg.time }}</div>
        </article>
      </div>

      <!-- 输入区 -->
      <div class="chat-input">
        <button
          class="voice-btn"
          :class="{ recording: isVoiceListening || isVoiceStarting }"
          @pointerdown="onVoicePointerDown"
          @pointerup="onVoicePointerUp"
          @pointerleave="onVoicePointerLeave"
        >
          🎤
        </button>
        <input
          v-model="input"
          :placeholder="$t('dh.placeholder')"
          @keydown.enter="sendText"
        />
        <button class="send-btn" @click="sendText">{{ $t('dh.send') }}</button>
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
                <span class="session-meta">{{ sess.message_count }} 条消息 · {{ new Date(sess.last_active_at).toLocaleDateString('zh-CN') }}</span>
              </div>
              <button
                class="session-delete"
                @click.stop="deleteSessionAndClose(sess.session_id)"
              >
                🗑
              </button>
            </div>
          </div>
        </div>
      </div>
    </Teleport>
    <!-- Mobile 底部 Tab 切换 -->
    <nav v-if="isMobileView" class="mobile-tabs">
      <button
        :class="{ active: mobileTab === 'avatar' }"
        @click="mobileTab = 'avatar'"
      >
        🤖 {{ $t('dh.tabAvatar') }}
      </button>
      <button
        :class="{ active: mobileTab === 'chat' }"
        @click="mobileTab = 'chat'"
      >
        💬 {{ $t('dh.tabChat') }}
      </button>
    </nav>
  </main>
</template>

<script lang="ts">
// highlightMatch helper (used in template via v-html)
function highlightMatch(text: string, query: string): string {
  if (!query) return text;
  const escaped = query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const regex = new RegExp(`(${escaped})`, 'gi');
  return text.replace(regex, '<mark style="background:rgba(99,226,183,.3);border-radius:2px;padding:0 2px">$1</mark>');
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

.emotion-bar {
  display: flex;
  gap: 6px;
  margin-top: 8px;
  flex-wrap: wrap;
  justify-content: center;
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
.emotion-btn:hover { border-color: var(--sg-jade-bright, #63e2b7); transform: scale(1.1); }
.emotion-btn.active { border-color: var(--sg-jade-bright, #63e2b7); background: rgba(99, 226, 183, 0.15); }

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
  position: absolute;
  bottom: 24px;
  display: flex;
  align-items: center;
  gap: 10px;
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
.ctrl-btn:hover { background: rgba(255, 255, 255, 0.08); color: var(--sg-text-body, rgba(255, 255, 255, 0.88)); }
.ctrl-btn.danger { border-color: rgba(232, 128, 128, 0.3); color: var(--sg-red-bright, #e88080); }
.ctrl-btn.danger:hover { background: rgba(232, 128, 128, 0.1); }
.ctrl-btn:disabled { opacity: 0.3; cursor: not-allowed; }
.interrupt-count { font-size: 11px; color: rgba(255, 255, 255, 0.25); }

/* 右侧聊天面板 */
.dh-chat {
  width: min(420px, 40vw);
  min-width: 320px;
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
.chat-header-top h2 { font-size: 15px; color: var(--sg-text-body, rgba(255, 255, 255, 0.88)); margin-bottom: 2px; }
.chat-header p { font-size: 12px; color: var(--sg-text-hint, rgba(255, 255, 255, 0.35)); }

.header-actions { display: flex; gap: 6px; }
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
.icon-btn:hover { background: rgba(99, 226, 183, 0.1); border-color: var(--sg-jade-border, rgba(99, 226, 183, 0.3)); }
.icon-btn.active { background: rgba(99, 226, 183, 0.15); border-color: var(--sg-jade-bright, #63e2b7); }

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
.search-bar input:focus { border-color: var(--sg-jade-bright, #63e2b7); }
.search-clear {
  background: none;
  border: none;
  color: rgba(255,255,255,.4);
  cursor: pointer;
  font-size: 14px;
}

.search-results {
  max-height: 200px;
  overflow-y: auto;
  padding: 8px 16px;
}
.search-result-item {
  padding: 6px 0;
  font-size: 12px;
  color: rgba(255,255,255,.6);
  border-bottom: 1px solid rgba(255,255,255,.04);
  line-height: 1.5;
}
.search-empty {
  padding: 20px;
  text-align: center;
  font-size: 12px;
  color: rgba(255,255,255,.25);
}

.quick-asks {
  display: flex;
  gap: 6px;
  padding: 10px 16px;
  overflow-x: auto;
  border-bottom: 1px solid rgba(255, 255, 255, 0.04);
  align-items: center;
}
.quick-asks button {
  white-space: nowrap;
  padding: 5px 12px;
  border-radius: 16px;
  border: 1px solid var(--sg-jade-border, rgba(99, 226, 183, 0.2));
  background: var(--sg-jade-bg, rgba(99, 226, 183, 0.06));
  font-size: 11px;
  color: var(--sg-jade-bright, #63e2b7);
  cursor: pointer;
  transition: all 0.2s;
}
.quick-asks button:hover { background: rgba(99, 226, 183, 0.15); }
.quick-asks .refresh-btn {
  margin-left: auto;
  padding: 5px 8px;
  font-size: 14px;
}

.message-list {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.msg { max-width: 85%; }
.msg.user { align-self: flex-end; }
.msg.assistant { align-self: flex-start; }
.msg.system { align-self: center; max-width: 100%; }
.msg-label { font-size: 11px; color: var(--sg-text-faint, rgba(255, 255, 255, 0.3)); margin-bottom: 4px; }
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
.msg-time { font-size: 10px; color: var(--sg-text-ghost, rgba(255, 255, 255, 0.2)); margin-top: 4px; }

.chat-input {
  display: flex;
  gap: 8px;
  padding: 12px 16px;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
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
.chat-input input:focus { border-color: var(--sg-jade-bright, #63e2b7); }
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
.voice-btn:hover { background: rgba(99, 226, 183, 0.15); }
.voice-btn.recording {
  background: rgba(232, 128, 128, 0.15);
  border-color: rgba(232, 128, 128, 0.4);
  animation: pulse 1s infinite;
}
@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.6; }
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
.drawer-header h3 { font-size: 14px; color: var(--sg-text-body, rgba(255, 255, 255, 0.88)); }
.drawer-close {
  background: none;
  border: none;
  color: rgba(255,255,255,.4);
  font-size: 16px;
  cursor: pointer;
}
.drawer-body { flex: 1; overflow-y: auto; padding: 12px; }
.drawer-empty {
  text-align: center;
  color: rgba(255,255,255,.25);
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
.session-item:hover { background: rgba(255, 255, 255, 0.04); }
.session-item.active { background: rgba(99, 226, 183, 0.08); border: 1px solid rgba(99, 226, 183, 0.15); }
.session-info { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.session-title { font-size: 13px; color: var(--sg-text-body, rgba(255, 255, 255, 0.88)); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.session-meta { font-size: 11px; color: rgba(255,255,255,.3); }
.session-delete {
  background: none;
  border: none;
  font-size: 14px;
  cursor: pointer;
  opacity: 0.3;
  transition: opacity 0.2s;
  padding: 4px;
}
.session-delete:hover { opacity: 1; }

@media (max-width: 1023px) {
  .dh-chat { width: min(360px, 45vw); min-width: 280px; }
}

@media (max-width: 768px) {
  .dh-view {
    flex-direction: column;
  }
  .dh-stage {
    flex: none;
    height: 45vh;
  }
  .dh-chat {
    width: 100%;
    min-width: unset;
    flex: 1;
    border-left: none;
    border-top: 1px solid rgba(255, 255, 255, 0.06);
    min-height: 0;
  }
  .subtitle-bubble { max-width: 90%; }
  .emotion-bar { gap: 4px; }
  .emotion-btn { width: 28px; height: 28px; font-size: 13px; }
  .session-drawer { width: 100%; max-width: 100vw; }
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
  transition: color 0.2s, background 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
}
.mobile-tabs button.active {
  color: var(--sg-jade-bright, #63e2b7);
  background: rgba(99, 226, 183, 0.06);
}
</style>