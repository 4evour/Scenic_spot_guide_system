<script setup lang="ts">
import { h, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { NLayoutSider, NMenu, NScrollbar } from 'naive-ui'
import type { MenuOption } from 'naive-ui'
import { useAppStore } from '../stores/app'
import { useAuthStore } from '../stores/auth'

const { t } = useI18n()
const router = useRouter()
const route = useRoute()
const appStore = useAppStore()
const authStore = useAuthStore()

function renderIcon(icon: string) {
  return () => h('span', { style: { fontSize: '18px' } }, icon)
}

const menuOptions = computed<MenuOption[]>(() => {
  const items: MenuOption[] = [
    { label: t('nav.dashboard'), key: 'dashboard', icon: renderIcon('📊') },
    {
      label: t('nav.scenicMgmt'),
      key: 'scenic-group',
      icon: renderIcon('📋'),
      children: [
        { label: t('nav.spots'), key: 'admin-spots', icon: renderIcon('🏛️') },
        { label: t('nav.routes'), key: 'admin-routes', icon: renderIcon('🛤️') },
        { label: t('nav.content'), key: 'admin-content', icon: renderIcon('📝') },
        { label: t('nav.qrcode'), key: 'admin-qrcode', icon: renderIcon('▦') },
      ],
    },
    {
      label: t('nav.digitalCenter'),
      key: 'dh-group',
      icon: renderIcon('🤖'),
      children: [
        { label: t('nav.avatar'), key: 'admin-avatar', icon: renderIcon('👤') },
        { label: t('nav.reports'), key: 'admin-reports', icon: renderIcon('📈') },
        { label: t('nav.queries'), key: 'admin-queries', icon: renderIcon('❓') },
      ],
    },
    { label: t('nav.knowledge'), key: 'admin-knowledge', icon: renderIcon('📚') },
  ]

  if (authStore.isAdmin) {
    items.push({
      label: t('nav.systemMgmt'),
      key: 'system-group',
      icon: renderIcon('⚙️'),
      children: [
        { label: t('nav.users'), key: 'admin-users', icon: renderIcon('👥') },
        { label: t('nav.settings'), key: 'admin-settings', icon: renderIcon('🔧') },
      ],
    })
  }

  // 添加游客端入口
  items.push(
    { label: '', key: 'divider', type: 'divider', icon: undefined },
    { label: t('map.title'), key: 'map', icon: renderIcon('🗺️') },
    { label: t('dh.title'), key: 'digital-human', icon: renderIcon('💬') },
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
    :style="{ background: 'var(--sg-surface-strong, rgba(5, 17, 20, 0.94))' }"
    @collapse="appStore.siderCollapsed = true"
    @expand="appStore.siderCollapsed = false"
  >
    <div class="sider-logo">
      <svg width="28" height="28" viewBox="0 0 24 24" fill="none">
        <path d="M12 2L2 7l10 5 10-5-10-5z" fill="#63e2b7" opacity="0.9"/>
        <path d="M2 17l10 5 10-5M2 12l10 5 10-5" stroke="#63e2b7" stroke-width="1.5" fill="none" opacity="0.5"/>
      </svg>
      <span v-if="!appStore.siderCollapsed" class="logo-text">{{ $t('nav.brand') }}</span>
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
