<script setup lang="ts">
import { computed } from 'vue';
import { NConfigProvider, NMessageProvider, NDialogProvider, darkTheme } from 'naive-ui';
import { appState, isAuthenticated, logout, openDigitalHuman, type AppRoute } from './router';
import { adminThemeOverrides } from './theme';
import LoginView from './views/LoginView.vue';
import DashboardView from './views/DashboardView.vue';
import AdminView from './views/AdminView.vue';
import DigitalHumanView from './views/DigitalHumanView.vue';
import MapView from './views/MapView.vue';

const showLogin = computed(() => !isAuthenticated() || appState.route === 'login');
const isDarkRoute = computed(() => ['dashboard', 'admin'].includes(appState.route));

const currentView = computed(() => {
  if (appState.route === 'admin') return AdminView;
  if (appState.route === 'digital-human') return DigitalHumanView;
  if (appState.route === 'map') return MapView;
  return DashboardView;
});

const navItems: { key: AppRoute; label: string; icon: string }[] = [
  { key: 'dashboard', label: '数据大屏', icon: '📊' },
  { key: 'admin', label: '管理后台', icon: '⚙️' },
  { key: 'digital-human', label: '数字人导览', icon: '🤖' },
  { key: 'map', label: '游客地图', icon: '🗺️' },
];

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
  <NConfigProvider :theme="isDarkRoute ? darkTheme : undefined" :theme-overrides="isDarkRoute ? adminThemeOverrides : undefined">
    <NMessageProvider>
      <NDialogProvider>
        <LoginView v-if="showLogin" @login-success="onLoginSuccess" />

        <div v-else class="app-layout">
          <!-- 顶部导航 -->
          <header class="app-nav">
            <div class="nav-brand" @click="navigate('dashboard')">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
                <path d="M12 2L2 7l10 5 10-5-10-5z" fill="#63e2b7" opacity="0.8"/>
                <path d="M2 17l10 5 10-5M2 12l10 5 10-5" stroke="#63e2b7" stroke-width="1.5" fill="none" opacity="0.5"/>
              </svg>
              <span class="brand-text">Scenic Guide</span>
            </div>

            <nav class="nav-tabs">
              <button
                v-for="item in navItems"
                :key="item.key"
                class="nav-tab"
                :class="{ active: appState.route === item.key }"
                @click="navigate(item.key)"
              >
                <span class="tab-icon">{{ item.icon }}</span>
                <span class="tab-label">{{ item.label }}</span>
              </button>
            </nav>

            <div class="nav-actions">
              <a href="/" class="nav-link">返回游客端</a>
              <button class="nav-link logout" @click="logout">退出</button>
              <div class="nav-avatar">A</div>
            </div>
          </header>

          <!-- 主内容 -->
          <main class="app-main">
            <component :is="currentView" />
          </main>
        </div>
      </NDialogProvider>
    </NMessageProvider>
  </NConfigProvider>
</template>

<style>
/* 全局重置 */
* { margin: 0; padding: 0; box-sizing: border-box; }
html, body, #app { height: 100%; }
body { font-family: -apple-system, 'PingFang SC', 'Segoe UI', sans-serif; }

.app-layout {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: #0a0a0f;
}

/* 导航栏 */
.app-nav {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  height: 56px;
  background: rgba(255, 255, 255, 0.02);
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
  flex-shrink: 0;
}

.nav-brand {
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
}
.brand-text {
  font-size: 16px;
  font-weight: 600;
  background: linear-gradient(135deg, #63e2b7, #18a058);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.nav-tabs {
  display: flex;
  gap: 2px;
}
.nav-tab {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border: none;
  background: none;
  color: rgba(255, 255, 255, 0.45);
  font-size: 13px;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
}
.nav-tab:hover {
  color: rgba(255, 255, 255, 0.88);
  background: rgba(255, 255, 255, 0.04);
}
.nav-tab.active {
  color: #63e2b7;
  background: rgba(99, 226, 183, 0.08);
}
.tab-icon { font-size: 15px; }

.nav-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}
.nav-link {
  color: rgba(255, 255, 255, 0.45);
  text-decoration: none;
  font-size: 12px;
  padding: 6px 10px;
  border-radius: 6px;
  border: none;
  background: none;
  cursor: pointer;
  transition: all 0.2s;
}
.nav-link:hover { color: rgba(255, 255, 255, 0.88); background: rgba(255, 255, 255, 0.04); }
.nav-link.logout:hover { color: #e88080; }
.nav-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: linear-gradient(135deg, #63e2b7, #18a058);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 700;
  color: #0a0a0f;
}

/* 主内容 */
.app-main {
  flex: 1;
  overflow-y: auto;
}
</style>
