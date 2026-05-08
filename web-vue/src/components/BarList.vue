<script setup lang="ts">
defineProps<{
  items: Array<{ label: string; value: number; suffix?: string }>;
}>();
</script>

<template>
  <div class="bar-list">
    <div v-for="item in items" :key="item.label" class="bar-row">
      <span>{{ item.label }}</span>
      <div class="bar-track">
        <div class="bar-fill" :style="{ width: `${item.value}%` }" />
      </div>
      <strong>{{ item.value }}{{ item.suffix || '%' }}</strong>
    </div>
  </div>
</template>
