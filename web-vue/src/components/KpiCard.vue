<script setup lang="ts">
defineProps<{
  label: string;
  value: string;
  note: string;
  tone?: 'cyan' | 'gold' | 'green' | 'red';
}>();
</script>

<template>
  <article class="kpi-card" :class="tone || 'cyan'">
    <span>{{ label }}</span>
    <strong>{{ value }}</strong>
    <small>{{ note }}</small>
  </article>
</template>
