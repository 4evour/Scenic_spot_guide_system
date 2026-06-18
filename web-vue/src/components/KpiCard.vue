<script setup lang="ts">
import { computed } from 'vue'
import { NCard, NSkeleton } from 'naive-ui'

const props = defineProps<{
  label: string
  value: string | number
  note?: string
  tone?: 'cyan' | 'gold' | 'green' | 'red'
  loading?: boolean
  trend?: 'up' | 'down' | 'flat'
  trendValue?: string
}>()

const toneColor = computed(() => {
  const colors = { cyan: '#52f0ee', gold: '#f4c765', green: '#63e2b7', red: '#e88080' }
  return colors[props.tone || 'cyan']
})

const trendIcon = computed(() => {
  if (props.trend === 'up') return '↑'
  if (props.trend === 'down') return '↓'
  return '→'
})

</script>

<template>
  <NCard
    :bordered="false"
    size="small"
    class="kpi-card"
    :style="{ '--kpi-tone': toneColor }"
  >
    <NSkeleton v-if="loading" :width="100" :height="48" :repeat="2" />

    <template v-else>
      <div class="kpi-header">
        <span class="kpi-label">{{ label }}</span>
        <span v-if="trend && trendValue" class="kpi-trend" :class="trend">
          {{ trendIcon }} {{ trendValue }}
        </span>
      </div>

      <div class="kpi-value">
        {{ value }}
      </div>

      <div v-if="note" class="kpi-note">{{ note }}</div>
    </template>
  </NCard>
</template>

<style scoped>
.kpi-card {
  background: rgba(255, 255, 255, 0.03) !important;
  border: 1px solid rgba(255, 255, 255, 0.06) !important;
  border-radius: 12px !important;
  border-left: 3px solid var(--kpi-tone, #52f0ee) !important;
  transition: all 0.2s;
}
.kpi-card:hover {
  background: rgba(255, 255, 255, 0.04) !important;
  border-color: rgba(255, 255, 255, 0.08) !important;
  transform: translateY(-1px);
}

.kpi-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}
.kpi-label {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.45);
  font-weight: 500;
}

.kpi-trend {
  font-size: 11px;
  font-weight: 600;
  padding: 1px 6px;
  border-radius: 4px;
}
.kpi-trend.up {
  color: #63e2b7;
  background: rgba(99, 226, 183, 0.1);
}
.kpi-trend.down {
  color: #e88080;
  background: rgba(232, 128, 128, 0.1);
}
.kpi-trend.flat {
  color: rgba(255, 255, 255, 0.4);
  background: rgba(255, 255, 255, 0.04);
}

.kpi-value {
  font-size: 28px;
  font-weight: 700;
  color: var(--kpi-tone, #52f0ee);
  line-height: 1.2;
  margin-bottom: 4px;
  font-variant-numeric: tabular-nums;
}

.kpi-note {
  font-size: 11px;
  color: rgba(255, 255, 255, 0.3);
}
</style>
