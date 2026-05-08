<script setup lang="ts">
import { computed } from 'vue';

const props = defineProps<{
  items: Array<{ label: string; value: number; color: string }>;
  center: string;
}>();

const radius = 82;
const circumference = Math.PI * 2 * radius;
const segments = computed(() => {
  let offset = 0;
  return props.items.map(item => {
    const dash = circumference * item.value / 100;
    const segment = { ...item, dash, gap: circumference - dash, offset };
    offset += dash;
    return segment;
  });
});
</script>

<template>
  <div class="donut-panel">
    <div class="donut-wrap">
      <svg viewBox="0 0 220 220">
        <circle
          v-for="item in segments"
          :key="item.label"
          cx="110"
          cy="110"
          :r="radius"
          fill="none"
          :stroke="item.color"
          stroke-width="24"
          :stroke-dasharray="`${item.dash} ${item.gap}`"
          :stroke-dashoffset="-item.offset"
          transform="rotate(-90 110 110)"
        />
        <circle cx="110" cy="110" r="54" fill="rgba(5,17,20,.96)" />
      </svg>
      <strong>{{ center }}</strong>
    </div>
    <div class="legend-list">
      <div v-for="item in items" :key="item.label" class="legend-item">
        <i :style="{ background: item.color }" />
        <span>{{ item.label }}</span>
        <strong>{{ item.value }}%</strong>
      </div>
    </div>
  </div>
</template>
