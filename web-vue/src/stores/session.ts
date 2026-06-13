import { defineStore } from 'pinia'
import { ref } from 'vue'

const API_TIMEOUT_MS = Number(import.meta.env.VITE_API_TIMEOUT_MS) || 15000;

export interface ChatSession {
  id: number
  user_id: number
  session_id: string
  title: string
  source: string
  message_count: number
  last_active_at: string
  created_at: string
}

export interface ChatMessage {
  id: number
  chat_session_id: number
  user_id: number
  role: 'user' | 'assistant' | 'system'
  content: string
  emotion?: string
  response_time_ms?: number
  created_at: string
}

/** 从 cookie 中读取 csrf_token */
function getCSRFToken(): string {
  const match = document.cookie.match(/(?:^|;\s*)csrf_token=([^;]*)/);
  return match ? decodeURIComponent(match[1]) : '';
}

async function apiFetch<T>(path: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`/api/v1${path}`, {
    signal: AbortSignal.timeout(API_TIMEOUT_MS),
    credentials: 'include',
    ...options,
    headers: {
      'Content-Type': 'application/json',
      'X-CSRF-Token': getCSRFToken(),
      ...options?.headers,
    },
  });
  const payload = await response.json();
  if (!response.ok || payload.code !== 0) throw new Error(payload.message || 'API error');
  return payload.data as T;
}

export const useSessionStore = defineStore('session', () => {
  const sessions = ref<ChatSession[]>([])
  const currentSessionId = ref<string | null>(null)
  const messages = ref<ChatMessage[]>([])
  const isLoading = ref(false)
  const hasMore = ref(true)
  const totalSessions = ref(0)

  /** 加载会话列表 */
  async function loadSessions(page = 1, pageSize = 20) {
    try {
      const data = await apiFetch<{ list: ChatSession[]; total: number }>(
        `/sessions?page=${page}&page_size=${pageSize}`,
      );
      sessions.value = data.list || [];
      totalSessions.value = data.total || 0;
    } catch {
      sessions.value = [];
    }
  }

  /** 切换/创建会话 */
  function setCurrentSession(sessionId: string) {
    currentSessionId.value = sessionId;
    messages.value = [];
    hasMore.value = true;
  }

  /** 加载消息历史（分页） */
  async function loadMessages(sessionId: string, limit = 50) {
    if (isLoading.value) return;
    isLoading.value = true;
    try {
      const beforeId = messages.value.length > 0
        ? messages.value[0].id
        : 0;
      const data = await apiFetch<{ messages: ChatMessage[] }>(
        `/sessions/${sessionId}/messages?limit=${limit}&before_id=${beforeId}`,
      );
      const newMessages = data.messages || [];
      if (newMessages.length < limit) hasMore.value = false;
      messages.value = [...newMessages, ...messages.value];
    } catch {
      // 静默失败
    } finally {
      isLoading.value = false;
    }
  }

  /** 追加消息（实时） */
  function appendMessage(role: ChatMessage['role'], content: string) {
    messages.value.push({
      id: Date.now(),
      chat_session_id: 0,
      user_id: 0,
      role,
      content,
      created_at: new Date().toISOString(),
    });
  }

  /** 删除会话 */
  async function deleteSession(sessionId: string) {
    try {
      await apiFetch(`/sessions/${sessionId}`, { method: 'DELETE' });
      sessions.value = sessions.value.filter(s => s.session_id !== sessionId);
      if (currentSessionId.value === sessionId) {
        currentSessionId.value = null;
        messages.value = [];
      }
    } catch {
      // 静默失败
    }
  }

  /** 保存单条消息到后端 */
  async function saveMessage(sessionId: string, role: ChatMessage['role'], content: string) {
    try {
      await apiFetch(`/sessions/${sessionId}/messages`, {
        method: 'POST',
        body: JSON.stringify({ role, content }),
      });
    } catch {
      // 静默失败 — 消息已在 localStorage 作为备份
    }
  }

  /** 搜索历史消息 */
  async function searchMessages(keyword: string, page = 1, pageSize = 20) {
    try {
      return await apiFetch<{ list: ChatMessage[]; total: number }>(
        `/sessions/search?keyword=${encodeURIComponent(keyword)}&page=${page}&page_size=${pageSize}`,
      );
    } catch {
      return { list: [], total: 0 };
    }
  }

  return {
    sessions, currentSessionId, messages, isLoading, hasMore, totalSessions,
    loadSessions, setCurrentSession, loadMessages, appendMessage, saveMessage, deleteSession, searchMessages,
  }
})
