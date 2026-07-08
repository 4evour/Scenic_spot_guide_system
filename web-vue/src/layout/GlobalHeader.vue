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
  margin: 14px 18px 0;
  padding: 0 18px;
  height: 54px;
  background: rgba(5, 17, 20, 0.78);
  border: 1px solid rgba(82, 240, 238, 0.12);
  border-radius: 14px;
  box-shadow: 0 14px 36px rgba(0, 0, 0, 0.18);
  backdrop-filter: blur(16px);
  flex-shrink: 0;
}
.header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}
.header-user {
  padding: 5px 10px;
  border: 1px solid rgba(82, 240, 238, 0.12);
  border-radius: 999px;
  font-size: 12px;
  color: var(--sg-text-soft, rgba(255, 255, 255, 0.3));
  background: rgba(82, 240, 238, 0.045);
}
.header-logout {
  min-height: 32px;
  padding: 0 12px;
  background: rgba(232, 128, 128, 0.08);
  border: 1px solid rgba(232, 128, 128, 0.18);
  border-radius: 999px;
  color: #e88080;
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
  min-height: 32px;
  padding: 0 12px;
  background: rgba(99, 226, 183, 0.08);
  border: 1px solid rgba(99, 226, 183, 0.22);
  border-radius: 999px;
  color: var(--sg-jade-bright, #63e2b7);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
}
.lang-switch:hover {
  background: rgba(99, 226, 183, 0.15);
}

:deep(.n-breadcrumb .n-breadcrumb-item__link) {
  color: var(--sg-text-muted);
}

:deep(.n-breadcrumb .n-breadcrumb-item:last-child .n-breadcrumb-item__link) {
  color: var(--sg-text-heading);
}
</style>
