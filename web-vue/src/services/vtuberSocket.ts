import type { VtuberMessage } from '../types/digitalHuman';

type Handlers = {
  onOpen?: () => void;
  onClose?: () => void;
  onError?: (message: string) => void;
  onMessage?: (message: VtuberMessage) => void;
};

export class VtuberSocketClient {
  private ws: WebSocket | null = null;
  private handlers: Handlers;
  private url: string;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 10;
  private baseReconnectDelay = 1000; // 1s
  private intentionalClose = false;

  constructor(url = defaultVtuberWsUrl(), handlers: Handlers = {}) {
    this.url = url;
    this.handlers = handlers;
  }

  connect() {
    this.intentionalClose = false;
    this.doConnect();
  }

  private doConnect() {
    this.cleanup();
    this.ws = new WebSocket(this.url);
    this.ws.onopen = () => {
      this.reconnectAttempts = 0;
      this.handlers.onOpen?.();
      this.send({ type: 'fetch-configs' });
      this.send({ type: 'fetch-history-list' });
      this.send({ type: 'create-new-history' });
    };
    this.ws.onclose = () => {
      this.handlers.onClose?.();
      this.scheduleReconnect();
    };
    this.ws.onerror = () => {
      this.handlers.onError?.('数字人语音服务未连接。文字聊天仍可正常使用，如需语音功能请启动 Open-LLM-VTuber 服务。');
    };
    this.ws.onmessage = event => {
      try {
        this.handlers.onMessage?.(JSON.parse(event.data));
      } catch {
        this.handlers.onError?.('收到无法解析的数字人消息。');
      }
    };
  }

  private scheduleReconnect() {
    if (this.intentionalClose) return;
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      this.handlers.onError?.('数字人语音服务重连失败，请检查服务是否启动。');
      return;
    }
    // Exponential backoff with jitter
    const delay = Math.min(
      this.baseReconnectDelay * Math.pow(2, this.reconnectAttempts) + Math.random() * 500,
      30000,
    );
    this.reconnectAttempts++;
    this.reconnectTimer = setTimeout(() => this.doConnect(), delay);
  }

  private cleanup() {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    if (this.ws && this.ws.readyState < WebSocket.CLOSING) {
      this.ws.close();
    }
    this.ws = null;
  }

  disconnect() {
    this.intentionalClose = true;
    this.cleanup();
  }

  sendText(text: string) {
    return this.send({ type: 'text-input', text });
  }

  interrupt(heardResponse = '') {
    return this.send({ type: 'interrupt-signal', text: heardResponse });
  }

  send(payload: Record<string, unknown>) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return false;
    this.ws.send(JSON.stringify(payload));
    return true;
  }
}

function defaultVtuberWsUrl() {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${protocol}//${window.location.host}/vtuber-ws/client-ws`;
}