<script setup lang="ts">
import { computed } from 'vue';

const props = defineProps<{
  values: number[];
}>();

const width = 900;
const height = 320;
const pad = 28;

const chart = computed(() => {
  const max = Math.max(...props.values, 1);
  const points = props.values.map((value, index) => {
    const x = props.values.length <= 1 ? 0 : (index / (props.values.length - 1)) * width;
    const y = height - pad - (value / max) * (height - pad * 2);
    return [Number(x.toFixed(1)), Number(y.toFixed(1))];
  });
  const line = points.map((point, index) => `${index === 0 ? 'M' : 'L'} ${point[0]} ${point[1]}`).join(' ');
  return {
    line,
    area: `${line} L ${width} ${height - pad} L 0 ${height - pad} Z`,
    points: points.filter((_, index) => index % 4 === 0),
  };
});
</script>

<template>
  <div class="trend-chart">
    <svg :viewBox="`0 0 ${width} ${height}`" preserveAspectRatio="none">
      <defs>
        <linearGradient id="trendFillVue" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stop-color="#52f0ee" stop-opacity="0.36" />
          <stop offset="100%" stop-color="#52f0ee" stop-opacity="0" />
        </linearGradient>
      </defs>
      <path :d="chart.area" fill="url(#trendFillVue)" />
      <path :d="chart.line" fill="none" stroke="#52f0ee" stroke-width="4" />
      <circle v-for="point in chart.points" :key="point.join('-')" :cx="point[0]" :cy="point[1]" r="5" fill="#f4c765" />
    </svg>
    <div class="axis-labels">
      <span>00:00</span><span>04:00</span><span>08:00</span><span>12:00</span><span>16:00</span><span>20:00</span><span>24:00</span>
    </div>
  </div>
</template>
