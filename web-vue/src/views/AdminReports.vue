<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { NButton, NAlert } from 'naive-ui'
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
const period = ref<'7d' | '30d'>('7d')
const periodLabel = computed(() => period.value === '7d' ? '近 7 天' : '近 30 天')

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

const negativeReasonBars = computed(() => {
  const data = state.report.negative_reasons || []
  const max = Math.max(...data.map(item => item.value), 1)
  return data.map(item => ({ label: item.label, value: Math.round(item.value / max * 100), suffix: ` / ${item.value}%` }))
})

const wordCloudItems = computed(() => state.report.word_cloud || [])
const audienceProfiles = computed(() => state.report.audience_profiles || [])
const routeSatisfaction = computed(() => state.report.route_satisfaction || [])

async function loadVisitorReport() {
  state.loading = true; state.error = ''
  try {
    const data = await apiFetch<Partial<VisitorReport> & { summary?: Partial<VisitorReport['summary']> }>(`/admin/reports/visitor?period=${period.value}`)
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
      <p class="hint-line">基于{{ periodLabel }}数字人、语音和网页问答交互记录自动生成。</p>
    </div>
    <div class="report-actions">
      <div class="period-switch">
        <button :class="{ active: period === '7d' }" @click="period = '7d'; loadVisitorReport()">7天</button>
        <button :class="{ active: period === '30d' }" @click="period = '30d'; loadVisitorReport()">30天</button>
      </div>
      <NButton :loading="state.loading" @click="loadVisitorReport">刷新报告</NButton>
    </div>
    <NAlert v-if="state.error" type="error" closable style="margin-top: 8px;">{{ state.error }}</NAlert>
  </article>

  <section class="kpi-row">
    <KpiCard label="交互记录" :value="String(state.report.summary.total_interactions)" :note="`${periodLabel}累计`" />
    <KpiCard label="满意倾向" :value="`${Math.round(state.report.summary.satisfaction_rate)}%`" note="正向情绪占比" tone="green" />
    <KpiCard label="负面占比" :value="`${Math.round(state.report.summary.negative_rate)}%`" note="需重点复盘" tone="red" />
    <KpiCard label="高峰时段" :value="state.report.summary.peak_hour" :note="`关注点：${state.report.summary.top_concern}`" tone="gold" />
  </section>

  <article class="panel">
    <h2>游客关注点分析</h2>
    <div v-if="state.loading" class="muted-center">正在生成报告...</div>
    <div v-else class="attention-layout">
      <BarList :items="attentionBars" />
      <div class="word-cloud">
        <span v-if="wordCloudItems.length === 0" class="empty-inline">暂无真实词云数据</span>
        <span
          v-for="item in wordCloudItems"
          :key="item.label"
          :style="{ fontSize: `${Math.min(22, 11 + item.value / 2)}px` }"
        >
          {{ item.label }}
        </span>
      </div>
    </div>
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
      <h2>负面反馈原因下钻</h2>
      <BarList :items="negativeReasonBars" />
    </article>
    <article class="panel">
      <h2>人群画像与路线匹配</h2>
      <div class="profile-list">
        <div v-if="audienceProfiles.length === 0" class="empty-block">暂无真实人群画像数据</div>
        <div v-for="item in audienceProfiles" :key="item.label" class="profile-row">
          <strong>{{ item.label }}</strong>
          <span>{{ item.percent }}% · {{ item.route }}</span>
          <small>满意度 {{ item.satisfaction }}%</small>
        </div>
      </div>
    </article>
  </div>

  <div class="two-col">
    <article class="panel">
      <h2>路线点击率/满意度</h2>
      <div class="route-report-list">
        <div v-if="routeSatisfaction.length === 0" class="empty-block">暂无真实路线满意度数据</div>
        <div v-for="item in routeSatisfaction" :key="item.label" class="route-report-row">
          <div>
            <strong>{{ item.label }}</strong>
            <span>点击率 {{ item.clickRate }}%</span>
          </div>
          <i :style="{ width: `${item.satisfaction}%` }"></i>
          <small>满意度 {{ item.satisfaction }}%</small>
        </div>
      </div>
    </article>
    <article class="panel">
      <h2>热门时段</h2>
      <BarList :items="peakHourBars" />
    </article>
  </div>

  <article class="panel">
    <h2>自动化改进建议</h2>
    <ul class="clean-list suggestion-grid">
      <li v-for="item in state.report.suggestions" :key="item.content">{{ item.content }}</li>
      <li v-if="state.report.suggestions.length === 0">暂无真实交互数据，暂不能生成改进建议。</li>
    </ul>
  </article>
</template>

<style scoped>
.kpi-row { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 16px; margin-bottom: 24px; }
.two-col { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; margin-bottom: 24px; }
.panel { background: var(--sg-surface-card, rgba(255,255,255,0.03)); border: 1px solid var(--sg-border-soft, rgba(255,255,255,0.06)); border-radius: var(--sg-radius-xl, 14px); padding: 24px; margin-bottom: 16px; }
.panel h2 { font-size: 15px; font-weight: 600; color: var(--sg-text-body, rgba(255,255,255,0.88)); margin-bottom: 16px; }
.report-header { display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 12px; }
.report-header h2 { margin-bottom: 0; }
.report-actions { display: flex; align-items: center; gap: 10px; }
.period-switch { display: flex; gap: 4px; padding: 3px; background: rgba(255,255,255,0.04); border-radius: 8px; }
.period-switch button { border: 0; border-radius: 6px; padding: 6px 12px; background: transparent; color: rgba(255,255,255,0.5); cursor: pointer; }
.period-switch button.active { background: var(--sg-jade-bright, #63e2b7); color: #041213; font-weight: 700; }
.secondary-action { padding: 10px 20px; background: rgba(255,255,255,0.04); border: 1px solid rgba(255,255,255,0.1); border-radius: 8px; color: rgba(255,255,255,0.75); font-size: 13px; cursor: pointer; transition: all 0.2s; }
.secondary-action:hover { background: rgba(255,255,255,0.08); color: rgba(255,255,255,0.9); }
.hint-line { font-size: 12px; color: rgba(255,255,255,0.3); margin-top: 8px; }
.muted-center { text-align: center; color: rgba(255,255,255,0.3); padding: 32px; }
.clean-list { list-style: none; padding: 0; }
.clean-list li { padding: 10px 0; border-bottom: 1px solid rgba(255,255,255,0.04); font-size: 13px; color: rgba(255,255,255,0.6); }
.clean-list li:last-child { border-bottom: none; }
.attention-layout { display: grid; grid-template-columns: 1.2fr .8fr; gap: 18px; align-items: stretch; }
.word-cloud { display: flex; flex-wrap: wrap; align-content: center; gap: 8px; padding: 14px; border-radius: 10px; background: rgba(82,240,238,0.035); border: 1px solid rgba(82,240,238,0.09); min-height: 150px; }
.word-cloud span { color: var(--sg-cyan, #52f0ee); line-height: 1; }
.word-cloud .empty-inline,
.empty-block {
  color: var(--sg-text-hint, rgba(255,255,255,0.35));
  font-size: 12px;
}
.profile-list { display: grid; gap: 10px; }
.profile-row { display: grid; grid-template-columns: 1fr auto auto; gap: 10px; align-items: center; padding: 10px 0; border-bottom: 1px solid rgba(255,255,255,0.04); }
.profile-row strong, .route-report-row strong { color: var(--sg-text-body, rgba(255,255,255,0.88)); font-size: 13px; }
.profile-row span, .route-report-row span { color: var(--sg-text-hint, rgba(255,255,255,0.35)); font-size: 12px; }
.profile-row small, .route-report-row small { color: var(--sg-jade-bright, #63e2b7); font-size: 12px; }
.route-report-list { display: grid; gap: 12px; }
.route-report-row { display: grid; grid-template-columns: 1fr minmax(120px, 45%) auto; gap: 10px; align-items: center; }
.route-report-row div { display: grid; gap: 3px; }
.route-report-row i { display: block; height: 8px; border-radius: 999px; background: var(--sg-jade-bright, #63e2b7); }
.suggestion-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(260px, 1fr)); gap: 0 18px; }
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
@media (max-width: 860px) {
  .attention-layout { grid-template-columns: 1fr; }
  .profile-row, .route-report-row { grid-template-columns: 1fr; }
  .report-actions { width: 100%; justify-content: space-between; }
}
</style>
