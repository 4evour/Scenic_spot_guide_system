<script setup lang="ts">
import { computed } from 'vue';
import { appState, isAuthenticated, logout, openDigitalHuman, type AppRoute } from './router';
import LoginView from './views/LoginView.vue';
import DashboardView from './views/DashboardView.vue';
import AdminView from './views/AdminView.vue';
import DigitalHumanView from './views/DigitalHumanView.vue';
import MapView from './views/MapView.vue';

const showLogin = computed(() => !isAuthenticated() || appState.route === 'login');

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

function onLoginSuccess() {
  window.location.hash = '/dashboard';
}
</script>

<template>
  <LoginView v-if="showLogin" @login-success="onLoginSuccess" />
  <div v-else class="app-shell">
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
      <button class="nav-link logout-btn" @click="logout">退出登录</button>
    </nav>

    <component :is="currentView" />
  </div>
</template>

<style scoped>
.logout-btn {
  margin-left: auto;
  cursor: pointer;
  background: none;
  border: none;
  color: rgba(255, 255, 255, 0.5);
  font-size: 13px;
  padding: 6px 12px;
  border-radius: 6px;
  transition: color 0.2s;
}
.logout-btn:hover {
  color: #ff8b8b;
}
</style>
