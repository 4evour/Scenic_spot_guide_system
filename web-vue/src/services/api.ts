import router from '../router';
import { useAuthStore } from '../stores/auth';
import { getCSRFToken } from '../utils/csrf';
import i18n from '../i18n';

function mergeAbortSignals(...signals: Array<AbortSignal | null | undefined>) {
  const activeSignals = signals.filter((signal): signal is AbortSignal => Boolean(signal));
  if (activeSignals.length <= 1) return activeSignals[0];
  if (typeof AbortSignal.any === 'function') return AbortSignal.any(activeSignals);

  const controller = new AbortController();
  const abort = () => controller.abort();
  for (const signal of activeSignals) {
    if (signal.aborted) {
      abort();
      break;
    }
    signal.addEventListener('abort', abort, { once: true });
  }
  return controller.signal;
}

export async function apiFetch<T = unknown>(
  path: string,
  options?: RequestInit,
): Promise<T> {
  const csrfToken = getCSRFToken();
  const headers: HeadersInit = {
    ...(options?.body instanceof FormData ? {} : { 'Content-Type': 'application/json' }),
    ...(csrfToken ? { 'X-CSRF-Token': csrfToken } : {}),
    ...options?.headers,
  };
  const timeoutMs = Number(import.meta.env.VITE_API_TIMEOUT_MS) || 15000;
  const response = await fetch(`/api/v1${path}`, {
    ...options,
    signal: mergeAbortSignals(options?.signal, AbortSignal.timeout(timeoutMs)),
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

// ============ 游客体验闭环 API ============

export interface TourRoute {
  id: number;
  name: string;
  description: string;
  spots: string;
  duration: number;
  difficulty: string;
  rating: number;
}

export interface RecommendedRoute {
  route: TourRoute;
  score: number;
  reason: string;
  matched_tags: string[];
}

export interface RouteRecommendationResult {
  routes: RecommendedRoute[];
}

export interface RouteRecommendationInput {
  session_id: string;
  profile_type: string;
  interest_tags: string[];
  difficulty?: string;
  limit?: number;
}

export interface SpotRatingInput {
  session_id: string;
  spot_id: number;
  overall_rating: number;
  culture_rating?: number;
  photo_rating?: number;
  facility_rating?: number;
  comment?: string;
  tags?: string[];
}

export interface SpotRatingStats {
  spot_id: number;
  count: number;
  avg_overall: number;
  avg_culture: number;
  avg_photo: number;
  avg_facility: number;
  negative_ratings: number;
}

export const visitorExperienceApi = {
  async recommendRoutes(input: RouteRecommendationInput) {
    return apiFetch<RouteRecommendationResult>('/visitor/routes/recommend', {
      method: 'POST',
      body: JSON.stringify(input),
    });
  },

  async submitSpotRating(input: SpotRatingInput) {
    return apiFetch('/visitor/ratings', {
      method: 'POST',
      body: JSON.stringify(input),
    });
  },

  async getSpotRatingStats(spotId: number) {
    return apiFetch<SpotRatingStats>(`/visitor/spots/${spotId}/ratings/stats`);
  },
};
