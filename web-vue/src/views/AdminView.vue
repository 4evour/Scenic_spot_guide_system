<script setup lang="ts">
import { reactive } from 'vue'
import AdminKnowledge from './AdminKnowledge.vue'
import AdminAvatar from './AdminAvatar.vue'
import AdminReports from './AdminReports.vue'
import AdminSettings from './AdminSettings.vue'

const state = reactive({ tab: 'knowledge' })

const tabs = [
  ['knowledge', '知识库管理'],
  ['avatar', '数字人形象'],
  ['reports', '感受度报告'],
  ['settings', '系统设置'],
]

const tabComponents: Record<string, unknown> = {
  knowledge: AdminKnowledge,
  avatar: AdminAvatar,
  reports: AdminReports,
  settings: AdminSettings,
}
</script>

<template>
  <div class="admin-view">
    <header class="admin-header">
      <div>
        <h1>管理后台</h1>
        <p>维护景区知识、数字人配置和系统设置</p>
      </div>
    </header>

    <nav class="admin-tabs">
      <button
        v-for="[key, label] in tabs"
        :key="key"
        class="admin-tab"
        :class="{ active: state.tab === key }"
        @click="state.tab = key"
      >
        {{ label }}
      </button>
    </nav>

    <div class="admin-content">
      <KeepAlive>
        <component :is="tabComponents[state.tab]" />
      </KeepAlive>
    </div>
  </div>
</template>

<style scoped>
.admin-view {
  padding: 28px 32px;
  background: var(--sg-bg-ink, #0a0a0f);
  min-height: 100%;
}
.admin-header { margin-bottom: 20px; }
.admin-header h1 {
  font-size: 22px;
  font-weight: 700;
  color: rgba(255, 255, 255, 0.92);
  margin-bottom: 4px;
}
.admin-header p {
  font-size: 13px;
  color: rgba(255, 255, 255, 0.35);
}

.admin-tabs {
  display: flex;
  gap: 4px;
  margin-bottom: 24px;
  padding: 4px;
  background: rgba(255, 255, 255, 0.02);
  border-radius: 10px;
  border: 1px solid rgba(255, 255, 255, 0.04);
  width: fit-content;
}
.admin-tab {
  padding: 8px 20px;
  border: none;
  background: none;
  color: rgba(255, 255, 255, 0.4);
  font-size: 13px;
  font-weight: 500;
  border-radius: 7px;
  cursor: pointer;
  transition: all 0.2s;
}
.admin-tab:hover { color: rgba(255, 255, 255, 0.7); }
.admin-tab.active {
  color: var(--sg-jade, #63e2b7);
  background: rgba(99, 226, 183, 0.1);
}

.admin-content { margin-top: 0; }

@media (max-width: 768px) {
  .admin-view { padding: 16px; }
  .admin-tabs { width: 100%; overflow-x: auto; }
}
</style>
