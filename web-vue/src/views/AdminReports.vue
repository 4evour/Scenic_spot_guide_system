<script setup lang="ts">
import { computed, onMounted, reactive } from 'vue'
import { NButton, NAlert, useMessage } from 'naive-ui'
import KpiCard from '../components/KpiCard.vue'
import BarList from '../components/BarList.vue'
import DonutChart from '../components/DonutChart.vue'
import { apiFetch } from '../services/api'
import type { VisitorReport } from '../types/admin'
import { defaultVisitorReport } from '../types/admin'

const state = reactive({
  loading: false,
  error: '',
  report: { ...defaultVisitorReport } as VisitorReport,
})

const attentionBars = computed(() => state.report.attention_analysis.map(item => ({
  label: item.label, value: Math.round(item.value),
})))

const emotionDonutItems = computed(() => {
  const colors: Record<string, string> = { '正面': '#7ef2a0', '中性': '#52f0ee', '负面': '#ff8b8b' }
  return state.report.emotion_distribution.map(item => ({
    label: item.label, value: Math.round(item.percent), color: colors[item.label] || '#f4c765',
  }))
})

const peakHourBars = computed(() => {
  const max = Math.max(...state.report.peak_hours.map(item => item.count), 1)
  return state.report.peak_hours.map(item => ({
    label: item.hour, value: Math.round(item.count / max * 100), suffix: ` / ${item.count}`,
  }))
})

async function loadVisitorReport() {
  state.loading = true; state.error = ''
  try {
    const data = await apiFetch<Partial<VisitorReport> & { summary?: Partial<VisitorReport['summary']> }>('/admin/reports/visitor')
    state.report = { ...defaultVisitorReport, ...data, summary: { ...defaultVisitorReport.summary, ...(data.summary || {}) } }
  } catch (error) { state.error = error instanceof Error ? error.message : '感受度报告加载失败' }
  finally { state.loading = false }
}

onMounted(loadVisitorReport)
</script>

<template>
  <article class="panel report-header">
    <div>
      <h2>游客感受度报告</h2>
      <p class="hint-line">基于近 7 天数字人、语音和网页问答交互记录自动生成。</p>
    </div>
    <NButton :loading="state.loading" @click="loadVisitorReport">刷新报告</NButton>
    <NAlert v-if="state.error" type="error" closable style="margin-top: 8px;">{{ state.error }}</NAlert>
  </article>

  <section class="kpi-row">
    <KpiCard label="交互记录" :value="String(state.report.summary.total_interactions)" note="近 7 天累计" />
    <KpiCard label="满意倾向" :value="`${Math.round(state.report.summary.satisfaction_rate)}%`" note="正向情绪占比" tone="green" />
    <KpiCard label="负面占比" :value="`${Math.round(state.report.summary.negative_rate)}%`" note="需重点复盘" tone="red" />
    <KpiCard label="高峰时段" :value="state.report.summary.peak_hour" :note="`关注点：${state.report.summary.top_concern}`" tone="gold" />
  </section>

  <article class="panel">
    <h2>游客关注点分析</h2>
    <div v-if="state.loading" class="muted-center">正在生成报告...</div>
    <BarList v-else :items="attentionBars" />
  </article>

  <div class="two-col">
    <article class="panel">
      <h2>情绪分布</h2>
      <DonutChart :items="emotionDonutItems" :center="`${Math.round(state.report.summary.satisfaction_rate)}%`" />
    </article>
    <article class="panel">
      <h2>情感趋势</h2>
      <div class="emotion-trend">
        <div v-for="item in state.report.emotion_trend" :key="item.date" class="trend-day">
          <span>{{ item.date.slice(5) }}</span>
          <div class="trend-stack">
            <i class="positive" :style="{ height: `${item.positive_rate}%` }" />
            <i class="negative" :style="{ height: `${item.negative_rate}%` }" />
          </div>
          <small>{{ item.total }}</small>
        </div>
      </div>
    </article>
  </div>

  <div class="two-col">
    <article class="panel">
      <h2>热门时段</h2>
      <BarList :items="peakHourBars" />
    </article>
    <article class="panel">
      <h2>服务建议</h2>
      <ul class="clean-list">
        <li v-for="item in state.report.suggestions" :key="item.content">{{ item.content }}</li>
      </ul>
    </article>
  </div>
</template>

<style scoped>
.kpi-row { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 16px; margin-bottom: 24px; }
.two-col { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; margin-bottom: 24px; }
.panel { background: var(--sg-surface-card, rgba(255,255,255,0.03)); border: 1px solid var(--sg-border-soft, rgba(255,255,255,0.06)); border-radius: var(--sg-radius-xl, 14px); padding: 24px; margin-bottom: 16px; }
.panel h2 { font-size: 15px; font-weight: 600; color: var(--sg-text-body, rgba(255,255,255,0.88)); margin-bottom: 16px; }
.report-header { display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 12px; }
.report-header h2 { margin-bottom: 0; }
.secondary-action { padding: 10px 20px; background: rgba(255,255,255,0.04); border: 1px solid rgba(255,255,255,0.1); border-radius: 8px; color: rgba(255,255,255,0.75); font-size: 13px; cursor: pointer; transition: all 0.2s; }
.secondary-action:hover { background: rgba(255,255,255,0.08); color: rgba(255,255,255,0.9); }
.hint-line { font-size: 12px; color: rgba(255,255,255,0.3); margin-top: 8px; }
.muted-center { text-align: center; color: rgba(255,255,255,0.3); padding: 32px; }
.clean-list { list-style: none; padding: 0; }
.clean-list li { padding: 10px 0; border-bottom: 1px solid rgba(255,255,255,0.04); font-size: 13px; color: rgba(255,255,255,0.6); }
.clean-list li:last-child { border-bottom: none; }
.emotion-trend { display: flex; gap: 6px; align-items: flex-end; height: 120px; }
.trend-day { flex: 1; display: flex; flex-direction: column; align-items: center; gap: 4px; }
.trend-day span { font-size: 10px; color: rgba(255,255,255,0.3); }
.trend-day small { font-size: 10px; color: rgba(255,255,255,0.25); }
.trend-stack { width: 100%; height: 80px; display: flex; flex-direction: column; justify-content: flex-end; border-radius: 4px; overflow: hidden; }
.trend-stack .positive { background: var(--sg-jade-bright, #63e2b7); min-height: 2px; }
.trend-stack .negative { background: var(--sg-red-bright, #e88080); min-height: 2px; }
.notice { padding: 8px 12px; border-radius: 6px; font-size: 12px; margin-top: 8px; }
.notice.error { background: rgba(232,128,128,0.08); color: #e88080; }
@media (max-width: 1200px) { .two-col { grid-template-columns: 1fr; } }
</style>
