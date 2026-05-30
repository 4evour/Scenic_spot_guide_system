import { reactive } from 'vue';

export type AppRoute = 'login' | 'dashboard' | 'admin' | 'digital-human' | 'map';

export const DIGITAL_HUMAN_URL = '/digital-human#/digital-human';

export function openDigitalHuman() {
  window.location.href = DIGITAL_HUMAN_URL;
}

export function isAuthenticated(): boolean {
  return !!localStorage.getItem('authToken');
}

export function getCurrentUserRole(): string {
  try {
    const raw = localStorage.getItem('user');
    if (!raw) return '';
    const user = JSON.parse(raw);
    return user?.role || '';
  } catch {
    return '';
  }
}

export function isAdmin(): boolean {
  return getCurrentUserRole() === 'admin';
}

export function logout() {
  localStorage.removeItem('authToken');
  localStorage.removeItem('user');
  window.location.hash = '/login';
}

export const appState = reactive({
  route: resolveRoute(),
});

export function resolveRoute(): AppRoute {
  const hash = window.location.hash.replace('#/', '').replace('#', '');

  if (hash === 'login') return 'login';

  // Admin-only routes: gate by role
  if (hash === 'dashboard' || hash === 'admin') {
    if (!isAuthenticated()) return 'login';
    if (!isAdmin()) return 'map';
    return hash;
  }

  if (hash === 'digital-human' || hash === 'map') return hash;

  // Default route
  if (!isAuthenticated()) return 'login';
  if (isAdmin()) return 'dashboard';
  return 'map';
}

export function startHashRouter() {
  window.addEventListener('hashchange', () => {
    appState.route = resolveRoute();
  });
}
