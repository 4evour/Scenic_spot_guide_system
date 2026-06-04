<script setup lang="ts">
import { computed, onMounted, onUnmounted, onErrorCaptured, reactive, ref } from 'vue';
import Live2DStage from '../components/Live2DStage.vue';

const uid = (): string =>
  typeof crypto.randomUUID === 'function'
    ? crypto.randomUUID()
    : 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, c => {
        const r = (Math.random() * 16) | 0;
        return (c === 'x' ? r : (r & 0x3) | 0x8).toString(16);
      });
import { AudioPlaybackController } from '../services/audioPlayback';
import { VtuberSocketClient } from '../services/vtuberSocket';
import type { ChatMessage, ConversationState, VtuberMessage } from '../types/digitalHuman';

type Expression = 'neutral' | 'happy' | 'thinking' | 'surprised' | 'interrupted';

const input = ref('');
const mouthOpen = ref(0);
const transcriptBuffer = ref('');
const hasActiveTurn = ref(false);
const isPlaybackActive = ref(false);
const isVoiceListening = ref(false);
const isVoiceStarting = ref(false);

const state = reactive({
  conversation: 'idle' as ConversationState,
  expression: 'happy' as Expression,
  subtitle: '你好，我是灵山景区 AI 导游小灵。你可以问我景点历史、路线推荐、开放时间或服务设施。',
  connected: false,
  interruptCount: 0,
  messages: [
    {
      id: uid(),
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
    isPlaybackActive.value = true;
    hasActiveTurn.value = true;
    if (text) showAssistantSpeech(text);
    state.conversation = 'speaking';
    state.expression = resolveSpeechExpression(cue?.expression, text);
  },
  onEnd: () => {
    isPlaybackActive.value = false;
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
    id: uid(),
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
    id: uid(),
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
    hasActiveTurn.value = true;
    blockIncomingPlayback = false;
    waitingForFreshServerTurn = false;
    audio.resume();
    state.conversation = 'thinking';
    state.expression = 'thinking';
    state.subtitle = '正在为你整理讲解内容...';
  }

  if (message.type === 'control' && message.text === 'conversation-chain-end') {
    hasActiveTurn.value = isPlaybackActive.value;
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
    hasActiveTurn.value = isPlaybackActive.value;
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
  hasActiveTurn.value = true;
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
  hasActiveTurn.value = false;
  isPlaybackActive.value = false;
  isVoiceListening.value = false;
  isVoiceStarting.value = false;
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
    state.subtitle = '我在听，请说出你的问题。';
  };
  nextRecognition.onresult = event => {
    const spokenText = event.results?.[0]?.[0]?.transcript?.trim();
    if (!spokenText) return;
    input.value = spokenText;
    sendText();
  };
  nextRecognition.onerror = event => {
    const detail = event.error === 'not-allowed' || event.error === 'service-not-allowed'
      ? '浏览器没有麦克风权限，请允许后重试。'
      : '无法启动语音输入，请确认浏览器支持并且麦克风可用。';
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
    addMessage('system', '语音输入启动失败，请稍后重试。');
  }
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

onErrorCaptured((err) => {
  console.error('数字人组件错误:', err);
  state.conversation = 'error';
  state.subtitle = '发生了一个错误，请刷新页面重试。';
  return false; // 阻止错误向上传播
});

onMounted(connectSocket);

onUnmounted(() => {
  window.clearTimeout(fallbackTimer);
  socket?.disconnect();
  recognition?.stop();
  audio.interrupt();
  hasActiveTurn.value = false;
  isPlaybackActive.value = false;
  isVoiceListening.value = false;
  isVoiceStarting.value = false;
  mouthOpen.value = 0;
});
</script>

<template>
  <main class="dh-view">
    <!-- 左侧：数字人展示区 -->
    <section class="dh-stage">
      <div class="dh-status">
        <span class="status-dot" :class="{ online: state.connected }"></span>
        <span class="status-text">{{ statusLabel }}</span>
      </div>

      <Live2DStage :state="state.conversation" :mouth-open="mouthOpen" :expression="state.expression" />

      <!-- 表情指示器 -->
      <div class="emotion-bar">
        <button v-for="expr in (['happy', 'neutral', 'thinking', 'surprised'] as const)" :key="expr"
          class="emotion-btn" :class="{ active: state.expression === expr }"
          @click="state.expression = expr">
          {{ { happy: '😊', neutral: '😐', thinking: '🤔', surprised: '😮' }[expr] }}
        </button>
      </div>

      <!-- 字幕气泡 -->
      <div class="subtitle-bubble">
        {{ state.subtitle }}
      </div>

      <!-- 控制栏 -->
      <div class="dh-controls">
        <button class="ctrl-btn danger" :disabled="!canInterrupt" @click="interruptAnswer">打断</button>
        <button class="ctrl-btn" @click="connectSocket">重连</button>
        <span class="interrupt-count">打断 {{ state.interruptCount }} 次</span>
      </div>
    </section>

    <!-- 右侧：聊天面板 -->
    <aside class="dh-chat">
      <div class="chat-header">
        <h2>💬 与小灵对话</h2>
        <p>支持文字和语音输入</p>
      </div>

      <!-- 快捷提问 -->
      <div class="quick-asks">
        <button @click="quickAsk('灵山大佛有多高？')">灵山大佛有多高？</button>
        <button @click="quickAsk('带孩子怎么玩？')">带孩子怎么玩？</button>
        <button @click="quickAsk('请推荐一条适合历史爱好者的游览路线')">推荐路线</button>
        <button @click="quickAsk('景区开放时间是什么时候')">开放时间</button>
      </div>

      <!-- 消息列表 -->
      <div class="message-list">
        <article v-for="msg in state.messages" :key="msg.id" :class="['msg', msg.role]">
          <div class="msg-label">{{ msg.role === 'user' ? '游客' : msg.role === 'assistant' ? '🤖 小灵' : '⚙ 系统' }}</div>
          <div class="msg-bubble">{{ msg.text }}</div>
          <div class="msg-time">{{ msg.time }}</div>
        </article>
      </div>

      <!-- 输入区 -->
      <div class="chat-input">
        <button class="voice-btn" :class="{ recording: isVoiceListening || isVoiceStarting }" @click="toggleVoice">
          🎤
        </button>
        <input v-model="input" placeholder="输入您的问题..." @keydown.enter="sendText" />
        <button class="send-btn" @click="sendText">发送</button>
      </div>
    </aside>
  </main>
</template>

<style scoped>
.dh-view {
  display: flex;
  height: calc(100vh - 56px);
  background: #0a0a0f;
}

/* 左侧数字人区域 */
.dh-stage {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  position: relative;
  background:
    radial-gradient(ellipse at 50% 40%, rgba(99, 226, 183, 0.06) 0%, transparent 60%),
    #0a0a0f;
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
  background: #63e2b7;
  box-shadow: 0 0 8px rgba(99, 226, 183, 0.5);
}
.status-text {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.45);
}

.emotion-bar {
  display: flex;
  gap: 8px;
  margin-top: 16px;
}
.emotion-btn {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  border: 2px solid rgba(255, 255, 255, 0.1);
  background: rgba(255, 255, 255, 0.04);
  font-size: 18px;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
}
.emotion-btn:hover { border-color: #63e2b7; transform: scale(1.1); }
.emotion-btn.active { border-color: #63e2b7; background: rgba(99, 226, 183, 0.15); }

.subtitle-bubble {
  max-width: 400px;
  margin-top: 20px;
  padding: 14px 20px;
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 12px;
  font-size: 14px;
  line-height: 1.6;
  color: rgba(255, 255, 255, 0.75);
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
.ctrl-btn:hover { background: rgba(255, 255, 255, 0.08); color: rgba(255, 255, 255, 0.88); }
.ctrl-btn.danger { border-color: rgba(232, 128, 128, 0.3); color: #e88080; }
.ctrl-btn.danger:hover { background: rgba(232, 128, 128, 0.1); }
.ctrl-btn:disabled { opacity: 0.3; cursor: not-allowed; }
.interrupt-count { font-size: 11px; color: rgba(255, 255, 255, 0.25); }

/* 右侧聊天面板 */
.dh-chat {
  width: min(400px, 40vw);
  min-width: 320px;
  display: flex;
  flex-direction: column;
  background: rgba(255, 255, 255, 0.02);
  border-left: 1px solid rgba(99, 226, 183, 0.06);
}

.chat-header {
  padding: 16px 20px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}
.chat-header h2 { font-size: 15px; color: rgba(255, 255, 255, 0.88); margin-bottom: 2px; }
.chat-header p { font-size: 12px; color: rgba(255, 255, 255, 0.35); }

.quick-asks {
  display: flex;
  gap: 6px;
  padding: 10px 16px;
  overflow-x: auto;
  border-bottom: 1px solid rgba(255, 255, 255, 0.04);
}
.quick-asks button {
  white-space: nowrap;
  padding: 5px 12px;
  border-radius: 16px;
  border: 1px solid rgba(99, 226, 183, 0.2);
  background: rgba(99, 226, 183, 0.06);
  font-size: 11px;
  color: #63e2b7;
  cursor: pointer;
  transition: all 0.2s;
}
.quick-asks button:hover { background: rgba(99, 226, 183, 0.15); }

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
.msg-label { font-size: 11px; color: rgba(255, 255, 255, 0.3); margin-bottom: 4px; }
.msg-bubble {
  padding: 10px 14px;
  border-radius: 12px;
  font-size: 13px;
  line-height: 1.6;
}
.msg.user .msg-bubble {
  background: rgba(99, 226, 183, 0.12);
  border: 1px solid rgba(99, 226, 183, 0.2);
  color: rgba(255, 255, 255, 0.88);
  border-bottom-right-radius: 4px;
}
.msg.assistant .msg-bubble {
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.08);
  color: rgba(255, 255, 255, 0.75);
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
.msg-time { font-size: 10px; color: rgba(255, 255, 255, 0.2); margin-top: 4px; }

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
.chat-input input:focus { border-color: #63e2b7; }
.send-btn {
  padding: 10px 18px;
  background: linear-gradient(135deg, #63e2b7, #18a058);
  border: none;
  border-radius: 8px;
  color: #0a0a0f;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}
.voice-btn {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  border: 1px solid rgba(99, 226, 183, 0.3);
  background: rgba(99, 226, 183, 0.06);
  font-size: 16px;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
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
  }
  .subtitle-bubble { max-width: 90%; }
}
</style>
