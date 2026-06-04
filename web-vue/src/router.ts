import { reactive } from 'vue';

export type AppRoute = 'login' | 'dashboard' | 'admin' | 'digital-human' | 'map';

export function openDigitalHuman() {
  window.location.hash = '#/digital-human';
}

interface CachedAuth {
  valid: boolean;
  role: string;
  userId: number;
  username: string;
  checkedAt: number;
}

let authCache: CachedAuth | null = null;
const AUTH_CACHE_TTL = 5 * 60 * 1000;

async function fetchAuth(): Promise<CachedAuth> {
  if (authCache && Date.now() - authCache.checkedAt < AUTH_CACHE_TTL) {
    return authCache;
  }
  try {
    const res = await fetch('/api/v1/user/me', { credentials: 'include' });
    if (res.ok) {
      const data = await res.json();
      if (data.code === 0 && data.data) {
        authCache = {
          valid: true,
          role: data.data.role || '',
          userId: data.data.id || 0,
          username: data.data.username || '',
          checkedAt: Date.now(),
        };
        return authCache;
      }
    }
  } catch {
    // network error
  }
  authCache = { valid: false, role: '', userId: 0, username: '', checkedAt: Date.now() };
  return authCache;
}

export async function isAuthenticated(): Promise<boolean> {
  const auth = await fetchAuth();
  return auth.valid;
}

export async function getCurrentUserRole(): Promise<string> {
  const auth = await fetchAuth();
  return auth.role;
}

export async function isAdmin(): Promise<boolean> {
  const role = await getCurrentUserRole();
  return role === 'admin';
}

export function invalidateAuth() {
  authCache = null;
}

export async function logout() {
  try {
    await fetch('/api/v1/logout', { method: 'POST', credentials: 'include' });
  } catch {
    // ignore
  }
  authCache = null;
  window.location.hash = '/login';
}

export const appState = reactive({
  route: 'login' as AppRoute,
});

export async function resolveRoute(): Promise<AppRoute> {
  const hash = window.location.hash.replace('#/', '').replace('#', '');

  if (hash === 'login') return 'login';

  if (hash === 'dashboard' || hash === 'admin') {
    if (!(await isAuthenticated())) return 'login';
    if (!(await isAdmin())) return 'map';
    return hash;
  }

  if (hash === 'digital-human' || hash === 'map') return hash;

  // Default route
  if (!(await isAuthenticated())) return 'login';
  if (await isAdmin()) return 'dashboard';
  return 'map';
}

let routeSeq = 0;

export function startHashRouter() {
  // Initial route resolution
  resolveRoute().then((route) => {
    appState.route = route;
  });

  window.addEventListener('hashchange', () => {
    const seq = ++routeSeq;
    resolveRoute().then((route) => {
      if (seq === routeSeq) {
        appState.route = route;
      }
    });
  });
}
