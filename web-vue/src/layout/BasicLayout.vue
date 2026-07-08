<script setup lang="ts">
import { NLayout } from 'naive-ui'
import GlobalSider from './GlobalSider.vue'
import GlobalHeader from './GlobalHeader.vue'
import GlobalContent from './GlobalContent.vue'
</script>

<template>
  <NLayout has-sider class="basic-layout">
    <GlobalSider />
    <div class="layout-right">
      <GlobalHeader />
      <GlobalContent />
    </div>
  </NLayout>
</template>

<style scoped>
.basic-layout {
  display: flex;
  height: 100vh;
  overflow: hidden;
  background:
    linear-gradient(rgba(82, 240, 238, 0.04) 1px, transparent 1px),
    linear-gradient(90deg, rgba(82, 240, 238, 0.035) 1px, transparent 1px),
    radial-gradient(ellipse at 28% 0%, rgba(82, 240, 238, 0.11), transparent 36%),
    radial-gradient(ellipse at 88% 12%, rgba(244, 199, 101, 0.09), transparent 32%),
    var(--sg-bg-ink, #031012);
  background-size: 34px 34px, 34px 34px, auto, auto, auto;
}
.layout-right {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
</style>
