<script setup lang="ts">
import GlobalSider from './GlobalSider.vue'
import GlobalHeader from './GlobalHeader.vue'
import GlobalContent from './GlobalContent.vue'
</script>

<template>
  <div class="basic-layout">
    <GlobalSider />
    <div class="layout-right">
      <GlobalHeader />
      <GlobalContent />
    </div>
  </div>
</template>

<style scoped>
.basic-layout {
  display: flex;
  height: 100vh;
  overflow: hidden;
  background: var(--sg-bg-ink, #031012);
}
.layout-right {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
</style>
