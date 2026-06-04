<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NConfigProvider, NMessageProvider, NDialogProvider, darkTheme } from 'naive-ui'
import { useAuthStore } from './stores/auth'
import { adminThemeOverrides } from './theme'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const isDarkTheme = computed(() => route.name !== 'login')
const isFullscreen = computed(() => !!route.meta?.fullscreen)

async function handleLogout() {
  await authStore.logout()
  router.push('/login')
}
</script>

<template>
  <NConfigProvider :theme="isDarkTheme ? darkTheme : undefined" :theme-overrides="isDarkTheme ? adminThemeOverrides : undefined">
    <NMessageProvider>
      <NDialogProvider>
        <!-- 登录页 -->
        <template v-if="route.name === 'login'">
          <router-view />
        </template>

        <!-- 游客端全屏页面（地图/数字人）：简单顶部栏 + 全屏内容 -->
        <template v-else-if="isFullscreen">
          <div class="fullscreen-layout">
            <header class="fullscreen-header">
              <div class="fullscreen-brand" @click="router.push('/dashboard')">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
                  <path d="M12 2L2 7l10 5 10-5-10-5z" fill="#63e2b7" opacity="0.9"/>
                  <path d="M2 17l10 5 10-5M2 12l10 5 10-5" stroke="#63e2b7" stroke-width="1.5" fill="none" opacity="0.5"/>
                </svg>
                <span>景区智能导览</span>
              </div>
              <div class="fullscreen-nav">
                <button :class="{ active: route.name === 'map' }" @click="router.push('/map')">🗺️ 地图</button>
                <button :class="{ active: route.name === 'digital-human' }" @click="router.push('/digital-human')">💬 数字人</button>
                <button v-if="authStore.isAdmin" @click="router.push('/dashboard')">⚙️ 管理</button>
                <button @click="handleLogout">退出</button>
              </div>
            </header>
            <main class="fullscreen-main">
              <router-view />
            </main>
          </div>
        </template>

        <!-- 管理端：Layout 由路由中的 BasicLayout 处理，这里直接渲染 router-view -->
        <template v-else>
          <router-view />
        </template>
      </NDialogProvider>
    </NMessageProvider>
  </NConfigProvider>
</template>

<style scoped>
.fullscreen-layout {
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
  background: var(--sg-bg-ink, #031012);
}
.fullscreen-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px;
  height: 44px;
  background: rgba(255, 255, 255, 0.02);
  border-bottom: 1px solid rgba(255, 255, 255, 0.04);
  flex-shrink: 0;
}
.fullscreen-brand {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.88);
}
.fullscreen-nav {
  display: flex;
  gap: 4px;
}
.fullscreen-nav button {
  padding: 4px 12px;
  background: none;
  border: 1px solid transparent;
  border-radius: 6px;
  color: rgba(255, 255, 255, 0.45);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
}
.fullscreen-nav button:hover {
  color: rgba(255, 255, 255, 0.7);
  background: rgba(255, 255, 255, 0.04);
}
.fullscreen-nav button.active {
  color: #63e2b7;
  background: rgba(99, 226, 183, 0.08);
  border-color: rgba(99, 226, 183, 0.15);
}
.fullscreen-main {
  flex: 1;
  overflow: hidden;
}

@media (max-width: 768px) {
  .fullscreen-brand span { display: none; }
}
</style>
