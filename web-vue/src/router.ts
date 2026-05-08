import { reactive } from 'vue';

export type AppRoute = 'dashboard' | 'admin' | 'digital-human';

export const appState = reactive({
  route: resolveRoute(),
});

export function resolveRoute(): AppRoute {
  const hash = window.location.hash.replace('#/', '').replace('#', '');
  if (hash === 'admin' || hash === 'digital-human' || hash === 'dashboard') return hash;
  return 'dashboard';
}

export function startHashRouter() {
  window.addEventListener('hashchange', () => {
    appState.route = resolveRoute();
  });
}
