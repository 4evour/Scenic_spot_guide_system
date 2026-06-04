<script setup lang="ts">
import { NLayoutContent } from 'naive-ui'
</script>

<template>
  <NLayoutContent
    content-style="padding: 20px;"
    :native-scrollbar="false"
    style="flex: 1; overflow: auto;"
  >
    <router-view v-slot="{ Component }">
      <Transition name="fade" mode="out-in">
        <component :is="Component" />
      </Transition>
    </router-view>
  </NLayoutContent>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.15s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
