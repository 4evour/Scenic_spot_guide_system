<script setup lang="ts">
import { computed } from 'vue';
import { appState, openDigitalHuman, type AppRoute } from './router';
import DashboardView from './views/DashboardView.vue';
import AdminView from './views/AdminView.vue';
import DigitalHumanView from './views/DigitalHumanView.vue';
import MapView from './views/MapView.vue';

const currentView = computed(() => {
  if (appState.route === 'admin') return AdminView;
  if (appState.route === 'digital-human') return DigitalHumanView;
  if (appState.route === 'map') return MapView;
  return DashboardView;
});

function navigate(route: AppRoute) {
  if (route === 'digital-human') {
    openDigitalHuman();
    return;
  }

  window.location.hash = `/${route}`;
}
</script>

<template>
  <div class="app-shell">
    <nav class="top-nav" aria-label="主导航">
      <button :class="{ active: appState.route === 'dashboard' }" @click="navigate('dashboard')">
        <span class="nav-icon">▣</span>
        数据大屏
      </button>
      <button :class="{ active: appState.route === 'admin' }" @click="navigate('admin')">
        <span class="nav-icon">▤</span>
        管理后台
      </button>
      <button :class="{ active: appState.route === 'digital-human' }" @click="navigate('digital-human')">
        <span class="nav-icon">◉</span>
        数字人导览
      </button>
      <button :class="{ active: appState.route === 'map' }" @click="navigate('map')">
        <span class="nav-icon">⌖</span>
        游客地图
      </button>
      <a href="/" class="nav-link">返回游客端</a>
    </nav>

    <component :is="currentView" />
  </div>
</template>
