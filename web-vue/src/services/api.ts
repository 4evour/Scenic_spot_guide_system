import router from '../router';
import { useAuthStore } from '../stores/auth';

export async function apiFetch<T = unknown>(
  path: string,
  options?: RequestInit,
): Promise<T> {
  const headers: HeadersInit = {
    ...(options?.body instanceof FormData ? {} : { 'Content-Type': 'application/json' }),
    ...(options?.headers || {}),
  };
  const response = await fetch(`/api/v1${path}`, {
    ...options,
    headers,
    credentials: 'include',
  });

  if (response.status === 401) {
    useAuthStore().invalidateAuth();
    router.push('/login');
    throw new Error('未登录或登录已过期');
  }

  const raw = await response.text();
  let payload: { code?: number; message?: string; msg?: string; data?: unknown } = {};
  if (raw.trim()) {
    try {
      payload = JSON.parse(raw);
    } catch {
      throw new Error(`接口返回非 JSON 响应 (${response.status})`);
    }
  }
  if (!response.ok || payload.code !== 0) {
    throw new Error(
      payload.message || payload.msg || response.statusText || `请求失败 (${response.status})`,
    );
  }
  return payload.data as T;
}
