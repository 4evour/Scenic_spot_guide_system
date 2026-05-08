<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue';
import Live2DStage from '../components/Live2DStage.vue';
import { AudioPlaybackController } from '../services/audioPlayback';
import { VtuberSocketClient } from '../services/vtuberSocket';
import type { ChatMessage, ConversationState, VtuberMessage } from '../types/digitalHuman';

type Expression = 'neutral' | 'happy' | 'thinking' | 'surprised' | 'interrupted';

const input = ref('');
const mouthOpen = ref(0);
const transcriptBuffer = ref('');

const state = reactive({
  conversation: 'idle' as ConversationState,
  expression: 'happy' as Expression,
  subtitle: '你好，我是灵山景区 AI 导游小灵。你可以问我景点历史、路线推荐、开放时间或服务设施。',
  connected: false,
  interruptCount: 0,
  messages: [
    {
      id: crypto.randomUUID(),
      role: 'assistant',
      text: '你好，我是灵山景区 AI 导游小灵。你可以问我景点历史、路线推荐、开放时间或服务设施。',
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

const audio = new AudioPlaybackController({
  onStart: (text, cue) => {
    if (text) showAssistantSpeech(text);
    state.conversation = 'speaking';
    state.expression = resolveSpeechExpression(cue?.expression, text);
  },
  onEnd: () => {
    mouthOpen.value = 0;
    if (state.conversation === 'speaking') {
      state.conversation = 'idle';
      state.expression = 'neutral';
    }
  },
  onVolume: volume => {
    mouthOpen.value = volume;
  },
});

const statusLabel = computed(() => {
  if (state.conversation === 'connecting') return '连接数字人服务中';
  if (state.conversation === 'thinking') return '正在检索知识库';
  if (state.conversation === 'speaking') return '正在讲解';
  if (state.conversation === 'listening') return '正在听你说话';
  if (state.conversation === 'interrupted') return '已打断回答';
  if (state.conversation === 'error') return '服务异常';
  return state.connected ? '数字人在线' : '离线演示模式';
});

const canInterrupt = computed(() => ['speaking', 'thinking', 'listening'].includes(state.conversation));

function nowTime() {
  return new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });
}

function stripEmotionTags(text: string) {
  return text.replace(/\[(neutral|joy|smirk|surprise|sadness|fear|anger|disgust)\]\s*/gi, '').trim() || text;
}

function expressionFromText(text?: string): Expression | undefined {
  if (!text) return undefined;
  const match = text.match(/\[(neutral|joy|smirk|surprise|sadness|fear|anger|disgust)\]/i);
  return match ? expressionFromToken(match[1]) : undefined;
}

function expressionFromToken(token?: string | number): Expression | undefined {
  if (token === undefined || token === null) return undefined;
  if (typeof token === 'number') {
    if (token === 0) return 'neutral';
    if (token === 1) return 'thinking';
    if (token === 2) return 'interrupted';
    if (token === 3) return 'happy';
    return undefined;
  }

  const normalized = token.toLowerCase().replace(/^\[|\]$/g, '');
  if (normalized === 'happy' || normalized === 'joy' || normalized === 'smirk' || normalized === 'exp_01') return 'happy';
  if (normalized === 'neutral' || normalized === 'exp_02') return 'neutral';
  if (normalized === 'thinking' || normalized === 'sadness' || normalized === 'fear' || normalized === 'exp_04') return 'thinking';
  if (normalized === 'surprised' || normalized === 'surprise' || normalized === 'exp_05') return 'surprised';
  if (normalized === 'interrupted' || normalized === 'anger' || normalized === 'disgust' || normalized === 'exp_07') return 'interrupted';
  if (/^\d+$/.test(normalized)) return expressionFromToken(Number(normalized));
  return undefined;
}

function resolveSpeechExpression(cueExpression?: string, text?: string): Expression {
  return expressionFromToken(cueExpression) || expressionFromText(text) || 'happy';
}

function addMessage(role: ChatMessage['role'], text: string) {
  const displayText = stripEmotionTags(text);
  state.messages.push({
    id: crypto.randomUUID(),
    role,
    text: displayText,
    time: nowTime(),
  });
  state.subtitle = displayText;
}

function showAssistantSpeech(text: string) {
  const displayText = stripEmotionTags(text);
  state.subtitle = displayText;
  if (displayText === lastAssistantSpeechText) return;
  lastAssistantSpeechText = displayText;
  state.messages.push({
    id: crypto.randomUUID(),
    role: 'assistant',
    text: displayText,
    time: nowTime(),
  });
}

function connectSocket() {
  state.conversation = 'connecting';
  socket = new VtuberSocketClient(undefined, {
    onOpen: () => {
      state.connected = true;
      state.conversation = 'idle';
      addMessage('system', '数字人服务已连接。');
    },
    onClose: () => {
      state.connected = false;
      if (state.conversation !== 'interrupted') state.conversation = 'idle';
    },
    onError: message => {
      addMessage('system', message);
      state.conversation = 'error';
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
    blockIncomingPlayback = false;
    waitingForFreshServerTurn = false;
    audio.resume();
    state.conversation = 'thinking';
    state.expression = 'thinking';
    state.subtitle = '正在为你整理讲解内容...';
  }

  if (message.type === 'control' && message.text === 'conversation-chain-end') {
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
    blockIncomingPlayback = true;
    waitingForFreshServerTurn = false;
    interruptedTurn = conversationTurn;
    state.conversation = 'interrupted';
    state.expression = 'interrupted';
    state.subtitle = '已收到数字人服务的中断确认，当前语音已停止。';
    return;
  }

  if (message.type === 'audio') {
    if (interruptedTurn === conversationTurn) return;
    const text = message.display_text?.text || transcriptBuffer.value || '我已经整理好讲解内容，请跟我一起看这条路线。';
    const expression = expressionFromToken(message.actions?.expressions?.[0]) || expressionFromText(text);
    const cue = {
      volumes: message.volumes,
      sliceLengthMs: message.slice_length,
      expression,
    };
    if (message.audio) audio.enqueueBase64Wav(message.audio, text, cue);
    else void audio.playTextFallback(text, cue);
  }

  if (message.type === 'backend-synth-complete') {
    if (state.conversation === 'thinking') {
      state.conversation = 'idle';
      state.expression = 'neutral';
    }
  }

  if (message.type === 'error') {
    addMessage('system', message.message || '数字人服务返回异常。');
    state.conversation = 'error';
  }
}

function sendText() {
  const text = input.value.trim();
  if (!text) return;

  window.clearTimeout(fallbackTimer);
  conversationTurn += 1;
  interruptedTurn = -1;
  const shouldWaitForServerTurn = Boolean(blockIncomingPlayback && socket);
  addMessage('user', text);
  lastAssistantSpeechText = '';
  input.value = '';
  state.conversation = 'thinking';
  state.expression = 'thinking';
  state.subtitle = '正在为你整理讲解内容...';
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
    fallbackTimer = window.setTimeout(() => {
      if (interruptedTurn === turn) return;
      showAssistantSpeech(fallback);
      void audio.playTextFallback(fallback, { expression: expressionFromText(fallback) || 'happy' });
    }, 420);
  }
}

function interruptAnswer() {
  recognition?.stop();
  window.clearTimeout(fallbackTimer);
  interruptedTurn = conversationTurn;
  blockIncomingPlayback = true;
  waitingForFreshServerTurn = false;
  audio.interrupt();
  const serverInterrupted = socket?.interrupt(state.subtitle) || false;
  mouthOpen.value = 0;
  state.interruptCount += 1;
  state.conversation = 'interrupted';
  state.expression = 'interrupted';
  state.subtitle = serverInterrupted
    ? '已打断当前讲解，并向数字人服务发送中断信号。'
    : '已停止本地播放。当前未连接后端，处于前端打断演示模式。';
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
    addMessage('system', '当前浏览器不支持 Web Speech 语音识别，请使用 Chrome 或 Edge。');
    return;
  }

  if (recognition && state.conversation === 'listening') {
    recognition.stop();
    return;
  }

  recognition = new Recognition();
  recognition.lang = 'zh-CN';
  recognition.interimResults = false;
  recognition.continuous = false;
  recognition.onstart = () => {
    state.conversation = 'listening';
    state.expression = 'surprised';
  };
  recognition.onresult = event => {
    input.value = event.results[0][0].transcript.trim();
    sendText();
  };
  recognition.onerror = () => {
    addMessage('system', '无法启动语音输入，请检查浏览器麦克风权限。');
    state.conversation = 'idle';
    state.expression = 'neutral';
  };
  recognition.onend = () => {
    if (state.conversation === 'listening') {
      state.conversation = 'idle';
      state.expression = 'neutral';
    }
  };
  recognition.start();
}

function quickAsk(text: string) {
  input.value = text;
  sendText();
}

function buildFallbackAnswer(text: string) {
  if (text.includes('路线') || text.includes('推荐')) {
    return '推荐你走“山门迎宾-文化长廊-核心景观-观景台-文创驿站”路线，全程约 90 分钟。喜欢历史的游客可以把讲解重点放在碑刻、建筑形制和地方传说上。';
  }
  if (text.includes('开放') || text.includes('时间')) {
    return '景区开放时间为 08:00 到 17:30，建议 16:30 前入园。节假日客流较高，可以优先选择上午或傍晚时段。';
  }
  if (text.includes('历史') || text.includes('文化')) {
    return '灵山景区以山水格局和地方历史文化为核心，沿线包含古道遗存、传统建筑和民俗展示点，适合用“自然景观加人文故事”的方式游览。';
  }
  return '我已根据你的问题生成一段演示讲解：这里会结合本地知识库、游客兴趣和实时客流，为你推荐更合适的景点顺序与讲解重点。';
}

onMounted(connectSocket);

onUnmounted(() => {
  window.clearTimeout(fallbackTimer);
  socket?.disconnect();
  recognition?.stop();
  audio.interrupt();
  mouthOpen.value = 0;
});
</script>

<template>
  <main class="digital-human-view">
    <section class="digital-main">
      <header class="hero-console compact">
        <div>
          <p class="eyebrow">Live2D Scenic Guide</p>
          <h1>灵山景区 AI 数字人导览</h1>
          <p>保留 Live2D 生动展示，语音播放、口型动画和打断控制全部由当前前端链路接管。</p>
        </div>
        <div class="status-badge" :class="{ connected: state.connected }">{{ statusLabel }}</div>
      </header>

      <Live2DStage :state="state.conversation" :mouth-open="mouthOpen" :expression="state.expression" />

      <div class="subtitle-panel">{{ state.subtitle }}</div>

      <div class="control-strip">
        <button class="danger-action" :disabled="!canInterrupt" @click="interruptAnswer">打断回答</button>
        <button class="secondary-action" @click="connectSocket">重连数字人服务</button>
        <button class="secondary-action" @click="quickAsk('请推荐一条适合历史爱好者的游览路线')">历史路线</button>
        <button class="secondary-action" @click="quickAsk('景区开放时间是什么时候')">开放时间</button>
        <span class="mini-stat">已打断 {{ state.interruptCount }} 次</span>
      </div>
    </section>

    <aside class="chat-console">
      <h2>游客交互记录</h2>
      <div class="message-list">
        <article v-for="message in state.messages" :key="message.id" :class="['message-card', message.role]">
          <strong>{{ message.role === 'user' ? '游客' : message.role === 'assistant' ? '小灵' : '系统' }}</strong>
          <p>{{ message.text }}</p>
          <small>{{ message.time }}</small>
        </article>
      </div>
      <div class="input-row">
        <input v-model="input" placeholder="请输入景点、路线、开放时间等问题..." @keydown.enter="sendText" />
        <button class="primary-action" @click="sendText">发送</button>
        <button class="voice-action" :class="{ active: state.conversation === 'listening' }" @click="toggleVoice">语音</button>
      </div>
    </aside>
  </main>
</template>
