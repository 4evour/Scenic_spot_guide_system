<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { NBreadcrumb, NBreadcrumbItem } from 'naive-ui'
import { useAuthStore } from '../stores/auth'
import { switchLocale } from '../i18n'

const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const breadcrumbs = computed(() => {
  const items: Array<{ label: string; to: string }> = [{ label: t('common.home'), to: '/dashboard' }]
  for (const r of route.matched) {
    if (r.meta?.title && !r.meta?.hideBreadcrumb) {
      items.push({ label: t(r.meta.title as string), to: r.path })
    }
  }
  return items
})

const nextLocale = computed(() => locale.value === 'zh-CN' ? 'en-US' : 'zh-CN')

function toggleLang() {
  switchLocale(nextLocale.value as 'zh-CN' | 'en-US')
}

async function handleLogout() {
  await authStore.logout()
  router.push('/login')
}
</script>

<template>
  <header class="global-header">
    <NBreadcrumb>
      <NBreadcrumbItem
        v-for="(item, i) in breadcrumbs"
        :key="i"
        :clickable="i < breadcrumbs.length - 1"
        @click="i < breadcrumbs.length - 1 && router.push(item.to)"
      >
        {{ item.label }}
      </NBreadcrumbItem>
    </NBreadcrumb>

    <div class="header-actions">
      <button class="lang-switch" @click="toggleLang">{{ $t('lang.switch') }}</button>
      <span v-if="authStore.user" class="header-user">{{ authStore.user.username }}</span>
      <button class="header-logout" @click="handleLogout">{{ $t('nav.logout') }}</button>
    </div>
  </header>
</template>

<style scoped>
.global-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  height: 48px;
  background: rgba(255, 255, 255, 0.02);
  border-bottom: 1px solid rgba(255, 255, 255, 0.04);
  flex-shrink: 0;
}
.header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}
.header-user {
  font-size: 12px;
  color: var(--sg-text-faint, rgba(255, 255, 255, 0.3));
}
.header-logout {
  padding: 4px 12px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 6px;
  color: rgba(255, 255, 255, 0.5);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
}
.header-logout:hover {
  background: rgba(232, 128, 128, 0.1);
  color: #e88080;
  border-color: rgba(232, 128, 128, 0.2);
}
.lang-switch {
  padding: 4px 12px;
  background: rgba(99, 226, 183, 0.06);
  border: 1px solid rgba(99, 226, 183, 0.2);
  border-radius: 6px;
  color: var(--sg-jade-bright, #63e2b7);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
}
.lang-switch:hover {
  background: rgba(99, 226, 183, 0.15);
}
</style>
