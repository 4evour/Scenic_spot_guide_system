<script setup lang="ts">
import { h, onMounted, onUnmounted, ref, shallowRef, nextTick, computed } from 'vue'
import {
  NSpin,
  NDataTable,
  NEmpty,
  NAlert,
  NSkeleton,
  NCard,
  NTag,
  NStatistic,
  type DataTableColumns,
} from 'naive-ui'
import { useI18n } from 'vue-i18n'
import * as echarts from 'echarts/core'
import { LineChart, PieChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent, GraphicComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import { apiFetch } from '../services/api'

echarts.use([
  LineChart,
  PieChart,
  GridComponent,
  TooltipComponent,
  LegendComponent,
  GraphicComponent,
  CanvasRenderer,
])

const { t } = useI18n()

/* ---- Types ---- */

interface Overview {
  total_visitors: string
  weekly_visitors: string
  total_chats: string
  weekly_chats: string
  satisfaction_rate: string
  avg_response_time: string
  visitors_trend: number
  chats_trend: number
  satisfaction_trend: number
  response_trend: number
}

interface CategoryItem {
  category: string
  count: number
  percent: number
}

interface Conversation {
  user_query: string
  ai_response: string
  emotion: string
  response_time: string
  time: string
}

interface TopQuestion {
  question: string
  count: number
}

interface SatisfactionTrendItem {
  date: string
  rate: number
  total: number
}

interface KnowledgeStats {
  total_count: number
}

interface ScenicSpot {
  id: number
  name: string
  category: string
  rating: number
  sort_order?: number
}

interface TourRoute {
  id: number
  name: string
  description: string
  spots: string
  duration: number
  difficulty: string
  rating: number
}

interface DigitalHumanConfig {
  name: string
  default_avatar_id: string
  allow_avatar_switch: boolean
  voice_type: string
  voice_tone: string
}

interface SpotRatingRankItem {
  spot_id: number
  spot_name: string
  count: number
  avg_overall: number
  negative_ratings: number
}

interface RoutePreferenceItem {
  route_name: string
  count: number
  avg_duration: number
}

interface InterestTagItem {
  tag: string
  count: number
}

interface VisitorExperienceSummary {
  period_days: number
  total_ratings: number
  avg_overall: number
  negative_ratings: number
  spot_ratings: SpotRatingRankItem[]
  route_preferences: RoutePreferenceItem[]
  interest_tags: InterestTagItem[]
}

interface ConsumptionAnalysis {
  summary: {
    sample_count: number
    unique_tourists: number
    total_cost: number
    average_total_cost: number
    median_total_cost: number
    average_satisfaction: number
  }
  category_breakdown: Array<{ category: string; total_cost: number; share_percent: number }>
  recommendations: string[]
  source_metadata: { source_file?: string; source_sha256?: string; source_row_count?: number }
}

interface ConsumptionAnalysisResponse {
  available: boolean
  scope: string
  period: string
  message?: string
  analysis?: ConsumptionAnalysis
}

/* ---- State ---- */

const loading = ref(true)
const error = ref<string | null>(null)
const overview = ref<Overview>({
  total_visitors: '0',
  weekly_visitors: '0',
  total_chats: '0',
  weekly_chats: '0',
  satisfaction_rate: '0%',
  avg_response_time: '0s',
  visitors_trend: 0,
  chats_trend: 0,
  satisfaction_trend: 0,
  response_trend: 0,
})
const recentConversations = ref<Conversation[]>([])
const topQuestions = ref<TopQuestion[]>([])
const satisfactionTrend = ref<SatisfactionTrendItem[]>([])
const knowledgeStats = ref<KnowledgeStats>({ total_count: 0 })
const hourlyTrend = ref<Array<{ hour: string; count: number }>>([])
const scenicSpots = ref<ScenicSpot[]>([])
const tourRoutes = ref<TourRoute[]>([])
const digitalHumanConfig = ref<DigitalHumanConfig | null>(null)
const visitorExperienceSummary = ref<VisitorExperienceSummary>({
  period_days: 7,
  total_ratings: 0,
  avg_overall: 0,
  negative_ratings: 0,
  spot_ratings: [],
  route_preferences: [],
  interest_tags: [],
})
const consumptionAll = ref<ConsumptionAnalysisResponse | null>(null)
const consumptionLingshan = ref<ConsumptionAnalysisResponse | null>(null)

const trendChartRef = ref<HTMLDivElement>()
const pieChartRef = ref<HTMLDivElement>()
const satisfactionChartRef = ref<HTMLDivElement>()

const trendChart = shallowRef<echarts.ECharts | null>(null)
const pieChart = shallowRef<echarts.ECharts | null>(null)
const satisfactionChart = shallowRef<echarts.ECharts | null>(null)

const emotionLabels = computed<Record<string, string>>(() => ({
  joy: t('dashboard.emotions.joy'),
  surprise: t('dashboard.emotions.surprise'),
  neutral: t('dashboard.emotions.neutral'),
  sadness: t('dashboard.emotions.sadness'),
  fear: t('dashboard.emotions.fear'),
}))
const emotionTagType: Record<string, 'success' | 'warning' | 'info' | 'error' | 'default'> = {
  joy: 'success',
  surprise: 'warning',
  neutral: 'info',
  sadness: 'error',
  fear: 'default',
}

/* ---- Recent conversations table columns ---- */

const conversationColumns = computed<DataTableColumns<Conversation>>(() => [
  {
    title: t('dashboard.columns.time'),
    key: 'time',
    width: 110,
    render: (row) =>
      h('span', { style: 'color:var(--sg-text-hint);font-size:12px;white-space:nowrap' }, row.time),
  },
  { title: t('dashboard.columns.visitorQuery'), key: 'user_query', ellipsis: { tooltip: true }, width: 220 },
  { title: t('dashboard.columns.aiResponse'), key: 'ai_response', ellipsis: { tooltip: true }, width: 280 },
  {
    title: t('dashboard.columns.emotion'),
    key: 'emotion',
    width: 100,
    render: (row) =>
      h(
        NTag,
        {
          type: emotionTagType[row.emotion] ?? 'default',
          size: 'small',
          bordered: false,
        },
        { default: () => emotionLabels.value[row.emotion] ?? row.emotion },
      ),
  },
])

const conversationPagination = { pageSize: 6 }

/* ---- Top questions table columns ---- */

const topQuestionColumns = computed<DataTableColumns<TopQuestion>>(() => [
  { title: t('dashboard.columns.hotQuestion'), key: 'question', ellipsis: { tooltip: true } },
  { title: t('dashboard.columns.count'), key: 'count', width: 80, sorter: (a, b) => a.count - b.count },
])

const topQuestionPagination = { pageSize: 10 }

/* ---- Data loading ---- */

async function loadData() {
  loading.value = true
  error.value = null
  try {
    const [ov, trend, cats, recent, tq, st, ks, spots, routes, avatarConfig, visitorExperience, consumption, consumptionLingshanData] =
      await Promise.all([
        apiFetch<Overview>('/admin/dashboard/overview'),
        apiFetch<Array<{ hour: string; count: number }>>('/admin/dashboard/hourly-trend'),
        apiFetch<CategoryItem[]>('/admin/dashboard/category-distribution'),
        apiFetch<Conversation[]>('/admin/dashboard/recent-conversations?limit=6'),
        apiFetch<TopQuestion[]>('/admin/dashboard/top-questions?limit=10').catch(
          () => null as TopQuestion[] | null,
        ),
        apiFetch<SatisfactionTrendItem[]>('/admin/dashboard/satisfaction-trend').catch(
          () => null as SatisfactionTrendItem[] | null,
        ),
        apiFetch<KnowledgeStats>('/admin/knowledge/stats').catch(() => null as KnowledgeStats | null),
        apiFetch<ScenicSpot[]>('/spots').catch(() => null as ScenicSpot[] | null),
        apiFetch<TourRoute[]>('/routes').catch(() => null as TourRoute[] | null),
        apiFetch<DigitalHumanConfig>('/admin/digital-human/config').catch(
          () => null as DigitalHumanConfig | null,
        ),
        apiFetch<VisitorExperienceSummary>('/admin/dashboard/visitor-experience?period=7d').catch(
          () => null as VisitorExperienceSummary | null,
        ),
        apiFetch<ConsumptionAnalysisResponse>('/admin/dashboard/consumption-analysis?scope=all&period=2025').catch(
          () => null as ConsumptionAnalysisResponse | null,
        ),
        apiFetch<ConsumptionAnalysisResponse>('/admin/dashboard/consumption-analysis?scope=lingshan&period=2025').catch(
          () => null as ConsumptionAnalysisResponse | null,
        ),
      ])

    if (ov) overview.value = ov
    hourlyTrend.value = trend || []
    recentConversations.value = recent || []
    if (tq) topQuestions.value = tq
    if (st) satisfactionTrend.value = st
    if (ks) knowledgeStats.value = ks
    if (spots) scenicSpots.value = spots
    if (routes) tourRoutes.value = routes
    if (avatarConfig) digitalHumanConfig.value = avatarConfig
    if (visitorExperience) visitorExperienceSummary.value = visitorExperience
    if (consumption) consumptionAll.value = consumption
    if (consumptionLingshanData) consumptionLingshan.value = consumptionLingshanData

    await nextTick()
    renderTrendChart(hourlyTrend.value)
    renderPieChart(cats || [])
    if (satisfactionTrend.value.length > 0) {
      renderSatisfactionChart(satisfactionTrend.value)
    }
  } catch (e) {
    const msg = e instanceof Error ? e.message : t('common.loadFailed')
    error.value = msg
  } finally {
    loading.value = false
  }
}

function formatMoney(value: number | undefined) {
  return Number(value || 0).toFixed(2)
}

/* ---- Chart rendering ---- */

function renderTrendChart(data: Array<{ hour: string; count: number }>) {
  const el = trendChartRef.value
  if (!el) return
  if (trendChart.value) trendChart.value.dispose()
  trendChart.value = echarts.init(el)
  trendChart.value.setOption({
    backgroundColor: 'transparent',
    grid: { top: 30, right: 16, bottom: 30, left: 50 },
    xAxis: {
      type: 'category',
      data: data.map((d) => d.hour),
      axisLine: { lineStyle: { color: 'rgba(255,255,255,0.08)' } },
      axisLabel: { color: 'var(--sg-text-hint)', fontSize: 11 },
    },
    yAxis: {
      type: 'value',
      splitLine: { lineStyle: { color: 'rgba(255,255,255,0.04)' } },
      axisLabel: { color: 'var(--sg-text-hint)', fontSize: 11 },
    },
    series: [
      {
        type: 'line',
        smooth: true,
        symbol: 'circle',
        symbolSize: 6,
        lineStyle: { color: 'var(--sg-jade-bright)', width: 2 },
        itemStyle: { color: 'var(--sg-jade-bright)' },
        areaStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(99,226,183,0.25)' },
            { offset: 1, color: 'rgba(99,226,183,0)' },
          ]),
        },
        data: data.map((d) => d.count),
      },
    ],
    tooltip: {
      trigger: 'axis',
      backgroundColor: 'rgba(20,20,30,0.95)',
      borderColor: 'rgba(255,255,255,0.08)',
      textStyle: { color: '#fff', fontSize: 12 },
    },
  })
}

function renderPieChart(data: CategoryItem[]) {
  const el = pieChartRef.value
  if (!el) return
  if (pieChart.value) pieChart.value.dispose()
  pieChart.value = echarts.init(el)
  const colors = ['#63e2b7', '#f8be52', '#7eb8da', '#b8a0dc', '#e88080', '#a5d6a7']
  pieChart.value?.setOption({
    backgroundColor: 'transparent',
    series: [
      {
        type: 'pie',
        radius: ['42%', '70%'],
        center: ['50%', '50%'],
        label: { color: 'rgba(255,255,255,0.6)', fontSize: 11 },
        labelLine: { lineStyle: { color: 'rgba(255,255,255,0.15)' } },
        data: data.map((d, i) => ({
          value: d.count,
          name: d.category,
          itemStyle: { color: colors[i % colors.length] },
        })),
      },
    ],
    tooltip: {
      backgroundColor: 'rgba(20,20,30,0.95)',
      borderColor: 'rgba(255,255,255,0.08)',
      textStyle: { color: '#fff', fontSize: 12 },
    },
  })
}

function renderSatisfactionChart(data: SatisfactionTrendItem[]) {
  const el = satisfactionChartRef.value
  if (!el) return
  if (satisfactionChart.value) satisfactionChart.value.dispose()
  satisfactionChart.value = echarts.init(el)
  const dates = data.map((d) => d.date.slice(5))
  const rates = data.map((d) => d.rate)
  satisfactionChart.value.setOption({
    backgroundColor: 'transparent',
    grid: { top: 30, right: 16, bottom: 30, left: 50 },
    xAxis: {
      type: 'category',
      data: dates,
      axisLine: { lineStyle: { color: 'rgba(255,255,255,0.08)' } },
      axisLabel: { color: 'var(--sg-text-hint)', fontSize: 11 },
    },
    yAxis: {
      type: 'value',
      min: 0,
      max: 100,
      axisLabel: { color: 'var(--sg-text-hint)', fontSize: 11, formatter: '{value}%' },
      splitLine: { lineStyle: { color: 'rgba(255,255,255,0.04)' } },
    },
    series: [
      {
        type: 'line',
        smooth: true,
        symbol: 'circle',
        symbolSize: 6,
        lineStyle: { color: 'var(--sg-gold)', width: 2 },
        itemStyle: { color: 'var(--sg-gold)' },
        areaStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(244,199,101,0.25)' },
            { offset: 1, color: 'rgba(244,199,101,0)' },
          ]),
        },
        data: rates,
      },
    ],
    tooltip: {
      trigger: 'axis',
      backgroundColor: 'rgba(20,20,30,0.95)',
      borderColor: 'rgba(255,255,255,0.08)',
      textStyle: { color: '#fff', fontSize: 12 },
      formatter: (params: unknown) => {
        const p = Array.isArray(params) ? params[0] : null
        if (!p || typeof p !== 'object' || !('value' in p)) return ''
        const item = data[(p as { dataIndex: number }).dataIndex]
        return item
          ? `${item.date}<br/>${t('dashboard.tooltips.satisfaction')}: ${item.rate}%<br/>${t('dashboard.tooltips.conversations')}: ${item.total}`
          : ''
      },
    },
  })
}

function handleResize() {
  trendChart.value?.resize()
  pieChart.value?.resize()
  satisfactionChart.value?.resize()
}

const loadingSkeletonCount = computed(() => [1, 2, 3, 4])
const displayTopQuestions = computed(() => topQuestions.value.slice(0, 5))
const displayTopSpots = computed(() =>
  // The project targets ES2020; copy before sorting to keep the source list immutable.
  [...scenicSpots.value]
    // oxlint-disable-next-line unicorn/no-array-sort -- ES2020 browser compatibility.
    .sort((a, b) => b.rating - a.rating || (a.sort_order ?? 0) - (b.sort_order ?? 0) || a.id - b.id)
    .slice(0, 5),
)
const displayRoutes = computed(() => tourRoutes.value.slice(0, 3))
const displayRoutePreferences = computed(() => visitorExperienceSummary.value.route_preferences.slice(0, 3))
const displayInterestTags = computed(() => visitorExperienceSummary.value.interest_tags.slice(0, 5))
const displaySpotRatings = computed(() => visitorExperienceSummary.value.spot_ratings.slice(0, 3))
const peakHour = computed(() => {
  const activeHours = hourlyTrend.value.filter((item) => item.count > 0)
  if (activeHours.length === 0) return null
  return activeHours.reduce((max, item) => (item.count > max.count ? item : max), activeHours[0])
})
const todayInteractionCount = computed(() => hourlyTrend.value.reduce((sum, item) => sum + item.count, 0))
const terminalSummary = computed(() => {
  const config = digitalHumanConfig.value
  if (!config) return []
  return [
    { label: t('dashboard.labels.guideName'), value: config.name || '-' },
    { label: t('dashboard.labels.avatarModel'), value: config.default_avatar_id || '-' },
    { label: t('dashboard.labels.voiceType'), value: config.voice_type || '-' },
    {
      label: t('dashboard.labels.avatarSwitch'),
      value: config.allow_avatar_switch ? t('dashboard.common.enabled') : t('dashboard.common.disabled'),
    },
  ]
})

onMounted(() => {
  loadData()
  window.addEventListener('resize', handleResize)
})
onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  trendChart.value?.dispose()
  pieChart.value?.dispose()
  satisfactionChart.value?.dispose()
})
</script>

<template>
  <div class="dashboard">
    <header class="page-header">
      <div>
        <h1>{{ t('dashboard.title') }}</h1>
        <p>{{ t('dashboard.subtitle') }}</p>
      </div>
    </header>

    <!-- Error alert -->
    <NAlert v-if="error" type="error" closable style="margin-bottom: 16px" @close="error = null">
      {{ error }}
    </NAlert>

    <NSpin :show="loading">
      <!-- KPI skeleton -->
      <div v-if="loading && recentConversations.length === 0" class="kpi-grid">
        <NCard
          v-for="n in loadingSkeletonCount"
          :key="n"
          :bordered="false"
          size="small"
          class="kpi-card-skeleton"
        >
          <NSkeleton :width="80" :height="16" style="margin-bottom: 12px" />
          <NSkeleton :width="120" :height="28" />
        </NCard>
      </div>

      <!-- KPI cards -->
      <div v-else class="kpi-grid">
        <NCard :bordered="false" size="small" class="kpi-card">
          <div class="kpi-icon" style="color: var(--sg-jade-bright)">👥</div>
          <div class="kpi-body">
            <NStatistic :label="t('dashboard.kpis.todayVisitors')" :value="overview.total_visitors" />
            <span class="kpi-trend" :class="overview.visitors_trend >= 0 ? 'up' : 'down'">
              {{ overview.visitors_trend >= 0 ? '↑' : '↓' }} {{ Math.abs(overview.visitors_trend) }}%
            </span>
          </div>
        </NCard>
        <NCard :bordered="false" size="small" class="kpi-card">
          <div class="kpi-icon" style="color: var(--sg-cyan)">💬</div>
          <div class="kpi-body">
            <NStatistic :label="t('dashboard.kpis.weeklyChats')" :value="overview.weekly_chats" />
            <span class="kpi-trend" :class="overview.chats_trend >= 0 ? 'up' : 'down'">
              {{ overview.chats_trend >= 0 ? '↑' : '↓' }} {{ Math.abs(overview.chats_trend) }}%
            </span>
          </div>
        </NCard>
        <NCard :bordered="false" size="small" class="kpi-card">
          <div class="kpi-icon" style="color: var(--sg-jade)">⭐</div>
          <div class="kpi-body">
            <NStatistic :label="t('dashboard.kpis.satisfaction')">
              <template #default>
                <span style="color: var(--sg-jade-bright)">{{ overview.satisfaction_rate }}</span>
              </template>
            </NStatistic>
            <span class="kpi-trend" :class="overview.satisfaction_trend >= 0 ? 'up' : 'down'">
              {{ overview.satisfaction_trend >= 0 ? '↑' : '↓' }} {{ Math.abs(overview.satisfaction_trend) }}%
            </span>
          </div>
        </NCard>
        <NCard :bordered="false" size="small" class="kpi-card">
          <div class="kpi-icon" style="color: var(--sg-gold)">⚡</div>
          <div class="kpi-body">
            <NStatistic :label="t('dashboard.kpis.avgResponseTime')">
              <template #default>
                <span style="color: var(--sg-gold)">{{ overview.avg_response_time }}</span>
              </template>
            </NStatistic>
            <span class="kpi-trend" :class="overview.response_trend <= 0 ? 'up' : 'down'">
              {{ overview.response_trend <= 0 ? '↓' : '↑' }} {{ Math.abs(overview.response_trend) }}%
            </span>
          </div>
        </NCard>
        <!-- Knowledge KPI -->
        <NCard :bordered="false" size="small" class="kpi-card">
          <div class="kpi-icon" style="color: var(--sg-blue)">📚</div>
          <div class="kpi-body">
            <NStatistic :label="t('dashboard.kpis.knowledgeItems')" :value="knowledgeStats.total_count" />
          </div>
        </NCard>
      </div>

      <section class="ops-grid">
        <NCard :bordered="false" class="chart-card ops-card">
          <template #header>{{ t('dashboard.sections.hotSpots') }}</template>
          <div v-if="displayTopSpots.length > 0" class="ops-list">
            <div v-for="spot in displayTopSpots" :key="spot.id" class="ops-list-row">
              <div>
                <strong>{{ spot.name }}</strong>
                <span>{{ spot.category || t('dashboard.common.none') }}</span>
              </div>
              <NTag size="small" :bordered="false" type="success">{{ spot.rating.toFixed(1) }}</NTag>
            </div>
          </div>
          <div v-else class="ops-empty">
            <NEmpty :description="t('dashboard.empty.hotSpots')" />
          </div>
        </NCard>

        <NCard :bordered="false" class="chart-card ops-card">
          <template #header>{{ t('dashboard.sections.questionCloud') }}</template>
          <div v-if="displayTopQuestions.length > 0" class="question-cloud">
            <button
              v-for="item in displayTopQuestions"
              :key="item.question"
              :style="{ fontSize: `${Math.min(20, 11 + item.count / 24)}px` }"
            >
              {{ item.question }}
              <span>{{ t('dashboard.units.times', { count: item.count }) }}</span>
            </button>
          </div>
          <div v-else class="ops-empty">
            <NEmpty :description="t('dashboard.empty.questionCloud')" />
          </div>
        </NCard>

        <NCard :bordered="false" class="chart-card ops-card">
          <template #header>{{ t('dashboard.sections.crowdHeat') }}</template>
          <div class="ops-metric-grid">
            <div>
              <strong>{{ todayInteractionCount }}</strong>
              <span>{{ t('dashboard.labels.todayInteractions') }}</span>
            </div>
            <div>
              <strong>{{ peakHour ? peakHour.hour : t('dashboard.common.none') }}</strong>
              <span>{{ t('dashboard.labels.peakHour') }}</span>
            </div>
            <div>
              <strong>{{ peakHour ? peakHour.count : 0 }}</strong>
              <span>{{ t('dashboard.labels.peakInteractions') }}</span>
            </div>
          </div>
        </NCard>
      </section>

      <section class="ops-grid ops-grid-middle">
        <NCard :bordered="false" class="chart-card ops-card">
          <template #header>{{ t('dashboard.sections.activityStatus') }}</template>
          <div v-if="displayRoutes.length > 0" class="ops-list">
            <div v-for="route in displayRoutes" :key="route.id" class="ops-list-row">
              <div>
                <strong>{{ route.name }}</strong>
                <span>{{ route.spots }}</span>
              </div>
              <NTag size="small" :bordered="false">{{
                t('dashboard.units.minutes', { count: route.duration })
              }}</NTag>
            </div>
          </div>
          <div v-else class="ops-empty">
            <NEmpty :description="t('dashboard.empty.activityStatus')" />
          </div>
        </NCard>

        <NCard :bordered="false" class="chart-card ops-card">
          <template #header>{{ t('dashboard.sections.terminalStatus') }}</template>
          <div v-if="terminalSummary.length > 0" class="ops-list">
            <div v-for="item in terminalSummary" :key="item.label" class="ops-list-row compact-row">
              <span>{{ item.label }}</span>
              <strong>{{ item.value }}</strong>
            </div>
          </div>
          <div v-else class="ops-empty">
            <NEmpty :description="t('dashboard.empty.terminalStatus')" />
          </div>
        </NCard>

        <NCard :bordered="false" class="chart-card ops-card">
          <template #header>{{ t('dashboard.sections.knowledgeOps') }}</template>
          <div class="knowledge-preview">
            <div>
              <strong>{{ knowledgeStats.total_count }}</strong>
              <span>{{ t('dashboard.kpis.knowledgeItems') }}</span>
            </div>
            <div>
              <strong>{{ topQuestions.length }}</strong>
              <span>{{ t('dashboard.labels.trackedQuestions') }}</span>
            </div>
          </div>
          <div class="ops-note">
            {{
              topQuestions.length > 0
                ? t('dashboard.labels.knowledgeCoverageHint')
                : t('dashboard.empty.knowledgeGaps')
            }}
          </div>
          <div class="avatar-preview">
            <strong>{{ t('dashboard.labels.avatarPreview') }}</strong>
            <span
              >{{ digitalHumanConfig?.name || t('dashboard.common.none') }} /
              {{ digitalHumanConfig?.voice_type || t('dashboard.common.none') }}</span
            >
          </div>
        </NCard>
      </section>

      <section class="visitor-experience-grid">
        <NCard :bordered="false" class="chart-card ops-card visitor-experience-card">
          <template #header>{{ t('dashboard.sections.visitorExperience') }}</template>
          <div class="visitor-experience-content">
            <div class="visitor-experience-metrics">
              <div>
                <strong>{{ visitorExperienceSummary.total_ratings }}</strong>
                <span>{{ t('dashboard.labels.totalRatings') }}</span>
              </div>
              <div>
                <strong>{{ visitorExperienceSummary.negative_ratings }}</strong>
                <span>{{ t('dashboard.labels.negativeRatings') }}</span>
              </div>
              <div>
                <strong>{{ visitorExperienceSummary.avg_overall.toFixed(1) }}</strong>
                <span>{{ t('dashboard.labels.avgSpotRating') }}</span>
              </div>
            </div>

            <div class="visitor-experience-lists">
              <div>
                <h3>{{ t('dashboard.labels.routePreference') }}</h3>
                <div v-if="displayRoutePreferences.length > 0" class="ops-list compact-list">
                  <div
                    v-for="item in displayRoutePreferences"
                    :key="item.route_name"
                    class="ops-list-row compact-row"
                  >
                    <span>{{ item.route_name }}</span>
                    <strong>{{ t('dashboard.units.times', { count: item.count }) }}</strong>
                  </div>
                </div>
                <NEmpty v-else :description="t('dashboard.common.none')" />
              </div>

              <div>
                <h3>{{ t('dashboard.labels.interestPreference') }}</h3>
                <div v-if="displayInterestTags.length > 0" class="interest-tags">
                  <NTag
                    v-for="item in displayInterestTags"
                    :key="item.tag"
                    size="small"
                    :bordered="false"
                    type="success"
                  >
                    {{ item.tag }} · {{ item.count }}
                  </NTag>
                </div>
                <NEmpty v-else :description="t('dashboard.common.none')" />
              </div>

              <div>
                <h3>{{ t('dashboard.kpis.satisfaction') }}</h3>
                <div v-if="displaySpotRatings.length > 0" class="ops-list compact-list">
                  <div
                    v-for="item in displaySpotRatings"
                    :key="item.spot_id"
                    class="ops-list-row compact-row"
                  >
                    <span>{{ item.spot_name || item.spot_id }}</span>
                    <strong>{{ item.avg_overall.toFixed(1) }}</strong>
                  </div>
                </div>
                <NEmpty v-else :description="t('dashboard.common.none')" />
              </div>
            </div>
          </div>
        </NCard>
      </section>

      <section class="visitor-experience-grid">
        <NCard :bordered="false" class="chart-card ops-card visitor-experience-card">
          <template #header>{{ t('dashboard.sections.consumptionAnalysis') }}</template>
          <div v-if="consumptionAll?.available && consumptionLingshan?.available" class="visitor-experience-content">
            <div class="visitor-experience-metrics">
              <div>
                <strong>¥{{ formatMoney(consumptionAll.analysis?.summary.total_cost) }}</strong>
                <span>{{ t('dashboard.consumption.totalAll') }}</span>
              </div>
              <div>
                <strong>¥{{ formatMoney(consumptionLingshan.analysis?.summary.average_total_cost) }}</strong>
                <span>{{ t('dashboard.consumption.lingshanAverage') }}</span>
              </div>
              <div>
                <strong>{{ consumptionLingshan.analysis?.summary.sample_count || 0 }}</strong>
                <span>{{ t('dashboard.consumption.lingshanSamples') }}</span>
              </div>
            </div>

            <div class="visitor-experience-lists">
              <div>
                <h3>{{ t('dashboard.consumption.categoryTitle') }}</h3>
                <div class="ops-list compact-list">
                  <div
                    v-for="item in consumptionLingshan.analysis?.category_breakdown || []"
                    :key="item.category"
                    class="ops-list-row compact-row"
                  >
                    <span>{{ t(`dashboard.consumption.categories.${item.category}`) }}</span>
                    <strong>{{ item.share_percent.toFixed(1) }}%</strong>
                  </div>
                </div>
              </div>

              <div>
                <h3>{{ t('dashboard.consumption.recommendationTitle') }}</h3>
                <div class="ops-list compact-list">
                  <div
                    v-for="recommendation in consumptionLingshan.analysis?.recommendations || []"
                    :key="recommendation"
                    class="ops-list-row compact-row"
                  >
                    <span>{{ recommendation }}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <NEmpty v-else :description="consumptionLingshan?.message || t('dashboard.consumption.empty')" />
        </NCard>
      </section>

      <!-- Charts -->
      <div class="chart-grid">
        <NCard :bordered="false" class="chart-card chart-wide">
          <template #header>{{ t('dashboard.sections.hourlyTrend') }}</template>
          <div ref="trendChartRef" class="chart-container"></div>
        </NCard>
        <NCard :bordered="false" class="chart-card">
          <template #header>{{ t('dashboard.sections.categoryDistribution') }}</template>
          <div ref="pieChartRef" class="chart-container"></div>
        </NCard>
      </div>

      <!-- Satisfaction trend + top questions -->
      <div class="chart-grid">
        <NCard :bordered="false" class="chart-card chart-wide">
          <template #header>{{ t('dashboard.sections.satisfactionTrend') }}</template>
          <div v-if="satisfactionTrend.length > 0" ref="satisfactionChartRef" class="chart-container"></div>
          <NEmpty v-else :description="t('dashboard.empty.satisfactionTrend')" />
        </NCard>
        <NCard :bordered="false" class="chart-card">
          <template #header>{{ t('dashboard.sections.topQuestions') }}</template>
          <NDataTable
            v-if="topQuestions.length > 0"
            :columns="topQuestionColumns"
            :data="topQuestions"
            :pagination="topQuestionPagination"
            :bordered="false"
            size="small"
          />
          <NEmpty v-else :description="t('dashboard.empty.topQuestions')" />
        </NCard>
      </div>

      <!-- Recent conversations -->
      <NCard :bordered="false" class="chart-card" style="margin-top: 16px">
        <template #header>{{ t('dashboard.sections.recentConversations') }}</template>
        <NDataTable
          v-if="recentConversations.length > 0"
          :columns="conversationColumns"
          :data="recentConversations"
          :pagination="conversationPagination"
          :bordered="false"
          size="small"
        />
        <NEmpty v-else :description="t('dashboard.empty.recentConversations')" />
      </NCard>
    </NSpin>
  </div>
</template>

<style scoped>
.dashboard {
  padding: 10px 14px 28px;
  background: transparent;
  min-height: 100%;
}

/* 页面头部 */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  margin-bottom: 18px;
  padding: 22px 24px;
  border: 1px solid rgba(82, 240, 238, 0.14);
  border-radius: 16px;
  background:
    linear-gradient(115deg, rgba(82, 240, 238, 0.08), transparent 38%),
    linear-gradient(300deg, rgba(244, 199, 101, 0.08), transparent 44%), rgba(5, 17, 20, 0.76);
  box-shadow: 0 18px 44px rgba(0, 0, 0, 0.2);
}
.page-header h1 {
  font-size: 26px;
  font-weight: 700;
  color: var(--sg-text-heading);
  margin-bottom: 4px;
}
.page-header p {
  font-size: 13px;
  color: var(--sg-text-muted);
}

/* KPI 网格 */
.kpi-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}
.kpi-card {
  display: flex;
  align-items: center;
  gap: 16px;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.045), rgba(255, 255, 255, 0.018)), rgba(5, 17, 20, 0.72) !important;
  border: 1px solid rgba(82, 240, 238, 0.13) !important;
  border-radius: var(--sg-radius-xl) !important;
  transition: all 0.2s;
  box-shadow: 0 14px 34px rgba(0, 0, 0, 0.16);
}
.kpi-card:hover {
  background: var(--sg-surface-hover) !important;
  border-color: var(--sg-jade-border) !important;
  transform: translateY(-1px);
}
.kpi-card-skeleton {
  display: flex;
  align-items: center;
  gap: 16px;
  background: var(--sg-surface-card) !important;
  border: 1px solid var(--sg-border-subtle) !important;
  border-radius: var(--sg-radius-xl) !important;
  min-height: 90px;
}
.kpi-icon {
  font-size: 28px;
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid rgba(82, 240, 238, 0.12);
  border-radius: var(--sg-radius-md);
  background: rgba(82, 240, 238, 0.06);
  flex-shrink: 0;
}
.kpi-body {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.kpi-trend {
  font-size: 11px;
  font-weight: 600;
  margin-top: 2px;
}
.kpi-trend.up {
  color: var(--sg-jade-bright);
}
.kpi-trend.down {
  color: var(--sg-red-bright);
}

.ops-grid {
  display: grid;
  grid-template-columns: 1.2fr 1fr 1.1fr;
  gap: 16px;
  margin-bottom: 16px;
}
.ops-grid-middle {
  grid-template-columns: 1.2fr 0.9fr 1.2fr;
  margin-bottom: 24px;
}
.visitor-experience-grid {
  margin-bottom: 24px;
}
.visitor-experience-card {
  min-height: 0;
}
.visitor-experience-content {
  display: grid;
  gap: 16px;
}
.visitor-experience-metrics {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}
.visitor-experience-metrics div {
  display: grid;
  gap: 4px;
  padding: 12px;
  border: 1px solid rgba(82, 240, 238, 0.1);
  border-radius: 8px;
  background: rgba(82, 240, 238, 0.045);
}
.visitor-experience-metrics strong {
  color: var(--sg-jade-bright);
  font-size: 20px;
}
.visitor-experience-metrics span {
  color: var(--sg-text-hint);
  font-size: 11px;
}
.visitor-experience-lists {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}
.visitor-experience-lists h3 {
  margin: 0 0 8px;
  color: var(--sg-text-body);
  font-size: 12px;
}
.compact-list {
  min-height: auto;
}
.interest-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.ops-card :deep(.n-card-header) {
  font-size: 14px;
}
.question-cloud {
  display: flex;
  flex-wrap: wrap;
  align-content: flex-start;
  gap: 8px;
  min-height: 168px;
}
.question-cloud button {
  border: 1px solid rgba(82, 240, 238, 0.12);
  border-radius: 8px;
  background: rgba(82, 240, 238, 0.055);
  color: var(--sg-cyan);
  padding: 8px 10px;
  cursor: default;
}
.question-cloud span {
  display: block;
  color: var(--sg-text-hint);
  font-size: 10px;
  margin-top: 3px;
}
.knowledge-preview {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
  padding: 12px;
  border-radius: 8px;
  background: rgba(99, 226, 183, 0.06);
}
.knowledge-preview strong {
  color: var(--sg-jade-bright);
  font-size: 24px;
}
.knowledge-preview span {
  color: var(--sg-text-hint);
  font-size: 11px;
}
.knowledge-preview div {
  display: grid;
  gap: 2px;
}
.ops-empty {
  display: grid;
  min-height: 168px;
  place-items: center;
}
.ops-empty-compact {
  min-height: 88px;
}
.ops-list {
  display: grid;
  gap: 10px;
  min-height: 168px;
  align-content: start;
}
.ops-list-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
  border: 1px solid rgba(99, 226, 183, 0.1);
  border-radius: 8px;
  background: rgba(99, 226, 183, 0.045);
}
.ops-list-row div {
  display: grid;
  gap: 4px;
  min-width: 0;
}
.ops-list-row strong,
.compact-row strong {
  color: var(--sg-text-body);
  font-size: 13px;
}
.ops-list-row span,
.compact-row span {
  color: var(--sg-text-hint);
  font-size: 11px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.compact-row {
  min-height: 42px;
}
.ops-metric-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
  min-height: 168px;
  align-content: center;
}
.ops-metric-grid div {
  display: grid;
  gap: 6px;
  padding: 16px 12px;
  border: 1px solid rgba(82, 240, 238, 0.1);
  border-radius: 8px;
  background: rgba(82, 240, 238, 0.045);
}
.ops-metric-grid strong {
  color: var(--sg-cyan);
  font-size: 22px;
}
.ops-metric-grid span {
  color: var(--sg-text-hint);
  font-size: 11px;
}
.ops-note {
  padding: 12px;
  border: 1px solid rgba(244, 199, 101, 0.1);
  border-radius: 8px;
  background: rgba(244, 199, 101, 0.045);
  color: var(--sg-text-hint);
  font-size: 12px;
  line-height: 1.6;
}
.avatar-preview {
  display: grid;
  gap: 4px;
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid rgba(255, 255, 255, 0.05);
}
.avatar-preview strong {
  color: var(--sg-text-body);
  font-size: 12px;
}
.avatar-preview span {
  color: var(--sg-text-hint);
  font-size: 11px;
}

/* 图表网格 */
.chart-grid {
  display: grid;
  grid-template-columns: 2fr 1fr;
  gap: 16px;
  margin-bottom: 24px;
}
.chart-wide {
  grid-column: 1;
}

.chart-card {
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.038), rgba(255, 255, 255, 0.014)), rgba(5, 17, 20, 0.72) !important;
  border: 1px solid rgba(82, 240, 238, 0.13) !important;
  border-radius: var(--sg-radius-xl) !important;
  box-shadow: 0 14px 34px rgba(0, 0, 0, 0.16);
}
.chart-container {
  height: 300px;
}

.ops-list-row,
.question-cloud button,
.ops-metric-grid div,
.knowledge-preview,
.ops-note,
.avatar-preview {
  border-color: rgba(82, 240, 238, 0.12);
  background: rgba(82, 240, 238, 0.04);
}

.ops-list-row strong,
.compact-row strong {
  color: var(--sg-text-heading);
}

.question-cloud button {
  border-radius: 999px;
}

:deep(.n-card-header) {
  color: var(--sg-text-heading);
}

:deep(.n-data-table) {
  --n-merged-th-color: rgba(82, 240, 238, 0.06);
  --n-merged-td-color: transparent;
}

@media (max-width: 1200px) {
  .chart-grid {
    grid-template-columns: 1fr;
  }
  .ops-grid,
  .ops-grid-middle,
  .visitor-experience-lists {
    grid-template-columns: 1fr;
  }
}
@media (max-width: 768px) {
  .dashboard {
    padding: 16px;
  }
  .kpi-grid {
    grid-template-columns: 1fr 1fr;
  }
  .chart-container {
    height: 220px;
  }
  .visitor-experience-metrics {
    grid-template-columns: 1fr;
  }
}
@media (max-width: 480px) {
  .kpi-grid {
    grid-template-columns: 1fr;
  }
}
</style>
