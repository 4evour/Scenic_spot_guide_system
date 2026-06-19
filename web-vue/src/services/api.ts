import router from '../router';
import { useAuthStore } from '../stores/auth';
import { getCSRFToken } from '../utils/csrf';
import i18n from '../i18n';

export async function apiFetch<T = unknown>(
  path: string,
  options?: RequestInit,
): Promise<T> {
  const csrfToken = getCSRFToken();
  const headers: HeadersInit = {
    ...(options?.body instanceof FormData ? {} : { 'Content-Type': 'application/json' }),
    ...(csrfToken ? { 'X-CSRF-Token': csrfToken } : {}),
    ...(options?.headers || {}),
  };
  const timeoutMs = Number(import.meta.env.VITE_API_TIMEOUT_MS) || 15000;
  const response = await fetch(`/api/v1${path}`, {
    signal: AbortSignal.timeout(timeoutMs),
    ...options,
    headers,
    credentials: 'include',
  });

  if (response.status === 401) {
    useAuthStore().invalidateAuth();
    router.push('/login');
    throw new Error(i18n.global.t('api.unauthorizedExpired'));
  }

  const raw = await response.text();
  let payload: { code?: number; message?: string; msg?: string; data?: unknown } = {};
  if (raw.trim()) {
    try {
      payload = JSON.parse(raw);
    } catch {
      throw new Error(i18n.global.t('api.nonJsonResponse', { status: response.status }));
    }
  }
  if (!response.ok || payload.code !== 0) {
    throw new Error(
      payload.message || payload.msg || response.statusText || i18n.global.t('api.requestFailed', { status: response.status }),
    );
  }
  return payload.data as T;
}

// ============ 游客认证 API ============

export const guestApi = {
  /** 游客自动登录 */
  async login(deviceFingerprint: string) {
    return apiFetch<{ id: number; username: string; display_name: string; role: string }>(
      '/auth/guest-login',
      {
        method: 'POST',
        body: JSON.stringify({ device_fingerprint: deviceFingerprint }),
      },
    );
  },

  /** 游客升级为正式账号 */
  async upgrade(data: { username: string; password: string; email?: string }) {
    return apiFetch<{ id: number; username: string; email: string; role: string }>(
      '/auth/upgrade-guest',
      {
        method: 'POST',
        body: JSON.stringify(data),
      },
    );
  },
};

// ============ 会话管理 API ============

export const sessionApi = {
  /** 获取当前用户的会话列表 */
  async list(page = 1, pageSize = 20) {
    return apiFetch<{ list: unknown[]; total: number; page: number; page_size: number }>(
      `/sessions?page=${page}&page_size=${pageSize}`,
    );
  },

  /** 获取会话消息历史 */
  async getMessages(sessionId: string, limit = 50, beforeId = 0) {
    return apiFetch<{ messages: unknown[] }>(
      `/sessions/${sessionId}/messages?limit=${limit}&before_id=${beforeId}`,
    );
  },

  /** 删除会话 */
  async delete(sessionId: string) {
    return apiFetch(`/sessions/${sessionId}`, { method: 'DELETE' });
  },

  /** 搜索历史消息 */
  async search(keyword: string, page = 1, pageSize = 20) {
    return apiFetch<{ list: unknown[]; total: number }>(
      `/sessions/search?keyword=${encodeURIComponent(keyword)}&page=${page}&page_size=${pageSize}`,
    );
  },
};
