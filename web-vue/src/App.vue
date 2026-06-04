<script setup lang="ts">
import { computed, defineAsyncComponent, onMounted, ref } from 'vue';
import { NConfigProvider, NMessageProvider, NDialogProvider, darkTheme } from 'naive-ui';
import { appState, isAuthenticated, isAdmin, logout, type AppRoute } from './router';
import { adminThemeOverrides } from './theme';
import LoginView from './views/LoginView.vue';

const DashboardView = defineAsyncComponent(() => import('./views/DashboardView.vue'));
const AdminView = defineAsyncComponent(() => import('./views/AdminView.vue'));
const DigitalHumanView = defineAsyncComponent(() => import('./views/DigitalHumanView.vue'));
const MapView = defineAsyncComponent(() => import('./views/MapView.vue'));

const authState = ref(false);
const adminState = ref(false);
const showLogin = computed(() => !authState.value || appState.route === 'login');
const isDarkRoute = computed(() => ['dashboard', 'admin', 'digital-human', 'map'].includes(appState.route));

const currentView = computed(() => {
  if (appState.route === 'admin') return AdminView;
  if (appState.route === 'digital-human') return DigitalHumanView;
  if (appState.route === 'map') return MapView;
  return DashboardView;
});

const allNavItems: { key: AppRoute; label: string; icon: string; adminOnly?: boolean }[] = [
  { key: 'dashboard', label: '数据大屏', icon: '📊', adminOnly: true },
  { key: 'admin', label: '管理后台', icon: '⚙️', adminOnly: true },
  { key: 'digital-human', label: '数字人导览', icon: '🤖' },
  { key: 'map', label: '游客地图', icon: '🗺️' },
];

const navItems = computed(() =>
  allNavItems.filter(item => !item.adminOnly || adminState.value)
);

async function refreshAuth() {
  authState.value = await isAuthenticated();
  adminState.value = await isAdmin();
}

onMounted(() => {
  refreshAuth();
});

function navigate(route: AppRoute) {
  window.location.hash = `/${route}`;
}

async function onLoginSuccess() {
  await refreshAuth();
  window.location.hash = adminState.value ? '/dashboard' : '/map';
}

function handleLogout() {
  authState.value = false;
  adminState.value = false;
  logout();
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
            <div class="nav-brand" @click="navigate(adminState ? 'dashboard' : 'map')">
              <div class="brand-icon">
                <svg width="28" height="28" viewBox="0 0 24 24" fill="none">
                  <path d="M12 2L2 7l10 5 10-5-10-5z" fill="#63e2b7" opacity="0.9"/>
                  <path d="M2 17l10 5 10-5M2 12l10 5 10-5" stroke="#63e2b7" stroke-width="1.5" fill="none" opacity="0.5"/>
                </svg>
              </div>
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
              <a href="/" class="nav-link">游客端</a>
              <div class="nav-user">
                <div class="nav-avatar">A</div>
                <button class="nav-link logout" @click="handleLogout">退出</button>
              </div>
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
</style>

<style scoped>
.app-layout {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: #0a0a0f;
}

.app-nav {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 28px;
  height: 56px;
  background: rgba(10, 10, 15, 0.75);
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
  border-bottom: 1px solid rgba(99, 226, 183, 0.1);
  flex-shrink: 0;
  position: sticky;
  top: 0;
  z-index: 100;
}

.nav-brand {
  display: flex;
  align-items: center;
  gap: 12px;
  cursor: pointer;
  flex-shrink: 0;
}
.brand-icon {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  background: rgba(99, 226, 183, 0.08);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.2s;
}
.nav-brand:hover .brand-icon {
  background: rgba(99, 226, 183, 0.15);
}
.brand-text {
  font-size: 17px;
  font-weight: 700;
  background: linear-gradient(135deg, #63e2b7 30%, #52f0ee 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  letter-spacing: 0.5px;
}

.nav-tabs {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
}
.nav-tab {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 18px;
  border: none;
  background: none;
  color: rgba(255, 255, 255, 0.4);
  font-size: 13px;
  font-weight: 500;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
  position: relative;
}
.nav-tab:hover {
  color: rgba(255, 255, 255, 0.85);
  background: rgba(255, 255, 255, 0.04);
}
.nav-tab.active {
  color: #63e2b7;
  background: rgba(99, 226, 183, 0.06);
}
.nav-tab.active::after {
  content: '';
  position: absolute;
  bottom: -1px;
  left: 50%;
  transform: translateX(-50%);
  width: 24px;
  height: 2px;
  background: linear-gradient(90deg, #63e2b7, #52f0ee);
  border-radius: 1px;
}
.tab-icon { font-size: 15px; }

.nav-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}
.nav-link {
  color: rgba(255, 255, 255, 0.4);
  text-decoration: none;
  font-size: 12px;
  padding: 6px 12px;
  border-radius: 6px;
  border: 1px solid rgba(255, 255, 255, 0.06);
  background: none;
  cursor: pointer;
  transition: all 0.2s;
  white-space: nowrap;
}
.nav-link:hover {
  color: rgba(255, 255, 255, 0.88);
  border-color: rgba(255, 255, 255, 0.12);
  background: rgba(255, 255, 255, 0.04);
}

.nav-user {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 8px 4px 4px;
  border-radius: 20px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.06);
  transition: all 0.2s;
}
.nav-user:hover {
  background: rgba(255, 255, 255, 0.06);
}
.nav-link.logout {
  border: none;
  padding: 4px 8px;
  font-size: 11px;
}
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
  flex-shrink: 0;
}

.app-main {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

@media (max-width: 768px) {
  .app-nav { padding: 0 16px; }
  .brand-text { display: none; }
  .nav-link { display: none; }
  .nav-tab { padding: 6px 12px; font-size: 12px; }
  .tab-label { display: none; }
}
</style>
