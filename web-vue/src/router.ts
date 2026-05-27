import { reactive } from 'vue';

export type AppRoute = 'login' | 'dashboard' | 'admin' | 'digital-human' | 'map';

export const DIGITAL_HUMAN_URL = '/digital-human#/digital-human';

export function openDigitalHuman() {
  window.location.href = DIGITAL_HUMAN_URL;
}

export function isAuthenticated(): boolean {
  return !!localStorage.getItem('authToken');
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
  if (hash === 'admin' || hash === 'digital-human' || hash === 'dashboard' || hash === 'map') return hash;
  return 'dashboard';
}

export function startHashRouter() {
  window.addEventListener('hashchange', () => {
    appState.route = resolveRoute();
  });
}
