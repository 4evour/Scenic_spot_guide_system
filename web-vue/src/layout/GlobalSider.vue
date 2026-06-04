<script setup lang="ts">
import { h, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { NLayoutSider, NMenu, NIcon, NScrollbar } from 'naive-ui'
import type { MenuOption } from 'naive-ui'
import { useAppStore } from '../stores/app'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const route = useRoute()
const appStore = useAppStore()
const authStore = useAuthStore()

function renderIcon(icon: string) {
  return () => h('span', { style: { fontSize: '18px' } }, icon)
}

const menuOptions = computed<MenuOption[]>(() => {
  const items: MenuOption[] = [
    { label: '数据大屏', key: 'dashboard', icon: renderIcon('📊') },
    {
      label: '景区管理',
      key: 'scenic-group',
      icon: renderIcon('📋'),
      children: [
        { label: '景点管理', key: 'admin-spots', icon: renderIcon('🏛️') },
        { label: '路线管理', key: 'admin-routes', icon: renderIcon('🛤️') },
        { label: '讲解内容', key: 'admin-content', icon: renderIcon('📝') },
      ],
    },
    {
      label: '数字人中心',
      key: 'dh-group',
      icon: renderIcon('🤖'),
      children: [
        { label: '形象配置', key: 'admin-avatar', icon: renderIcon('👤') },
        { label: '感受度报告', key: 'admin-reports', icon: renderIcon('📈') },
      ],
    },
    { label: '知识库', key: 'admin-knowledge', icon: renderIcon('📚') },
  ]

  if (authStore.isAdmin) {
    items.push({
      label: '系统管理',
      key: 'system-group',
      icon: renderIcon('⚙️'),
      children: [
        { label: '用户管理', key: 'admin-users', icon: renderIcon('👥') },
        { label: '系统设置', key: 'admin-settings', icon: renderIcon('🔧') },
      ],
    })
  }

  // 添加游客端入口
  items.push(
    { label: '', key: 'divider', type: 'divider', icon: undefined },
    { label: '地图导览', key: 'map', icon: renderIcon('🗺️') },
    { label: '数字人交互', key: 'digital-human', icon: renderIcon('💬') },
  )

  return items
})

const activeKey = computed(() => route.name as string)

function handleMenuUpdate(key: string) {
  if (key === 'divider') return
  router.push({ name: key })
}
</script>

<template>
  <NLayoutSider
    bordered
    collapse-mode="width"
    :collapsed-width="64"
    :width="220"
    :collapsed="appStore.siderCollapsed"
    show-trigger
    @collapse="appStore.siderCollapsed = true"
    @expand="appStore.siderCollapsed = false"
    :style="{ background: 'var(--sg-surface-strong, rgba(5, 17, 20, 0.94))' }"
  >
    <div class="sider-logo">
      <svg width="28" height="28" viewBox="0 0 24 24" fill="none">
        <path d="M12 2L2 7l10 5 10-5-10-5z" fill="#63e2b7" opacity="0.9"/>
        <path d="M2 17l10 5 10-5M2 12l10 5 10-5" stroke="#63e2b7" stroke-width="1.5" fill="none" opacity="0.5"/>
      </svg>
      <span v-if="!appStore.siderCollapsed" class="logo-text">景区智能导览</span>
    </div>

    <NScrollbar style="flex: 1; overflow: auto;">
      <NMenu
        :collapsed="appStore.siderCollapsed"
        :collapsed-width="64"
        :collapsed-icon-size="20"
        :options="menuOptions"
        :value="activeKey"
        :default-expanded-keys="['scenic-group', 'dh-group', 'system-group']"
        @update:value="handleMenuUpdate"
      />
    </NScrollbar>
  </NLayoutSider>
</template>

<style scoped>
.sider-logo {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 16px;
  height: 52px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.04);
}
.logo-text {
  font-size: 15px;
  font-weight: 600;
  color: var(--sg-text-heading, rgba(255, 255, 255, 0.92));
  white-space: nowrap;
}
</style>
