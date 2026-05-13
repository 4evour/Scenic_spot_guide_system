import { reactive } from 'vue';

export type AppRoute = 'dashboard' | 'admin' | 'digital-human' | 'map';

export const DIGITAL_HUMAN_URL = '/digital-human#/digital-human';

export function openDigitalHuman() {
  window.location.href = DIGITAL_HUMAN_URL;
}

export const appState = reactive({
  route: resolveRoute(),
});

export function resolveRoute(): AppRoute {
  const hash = window.location.hash.replace('#/', '').replace('#', '');
  if (hash === 'admin' || hash === 'digital-human' || hash === 'dashboard' || hash === 'map') return hash;
  return 'dashboard';
}

export function startHashRouter() {
  window.addEventListener('hashchange', () => {
    appState.route = resolveRoute();
  });
}
