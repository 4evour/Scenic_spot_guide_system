<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive } from 'vue';
import KpiCard from '../components/KpiCard.vue';
import TrendChart from '../components/TrendChart.vue';
import BarList from '../components/BarList.vue';
import DonutChart from '../components/DonutChart.vue';

type Overview = {
  total_visitors: string;
  weekly_visitors: string;
  total_chats: string;
  weekly_chats: string;
  satisfaction_rate: string;
  avg_response_time: string;
  visitors_trend: number;
  chats_trend: number;
  satisfaction_trend: number;
  response_trend: number;
};

type HourlyTrend = { hour: string; count: number };
type TopQuestion = { question: string; count: number };
type CategoryItem = { category: string; count: number; percent: number };
type ResponseTimeItem = { bucket: string; count: number; percent: number };
type SatisfactionTrend = { date: string; rate: number; total: number };
type Conversation = {
  user_query: string;
  ai_response: string;
  emotion: string;
  response_time: string;
  time: string;
};

const state = reactive({
  now: '',
  loading: false,
  error: '',
  overview: {
    total_visitors: '0',
    weekly_visitors: '0',
    total_chats: '0',
    weekly_chats: '0',
    satisfaction_rate: '0.0%',
    avg_response_time: '0.0s',
    visitors_trend: 0,
    chats_trend: 0,
    satisfaction_trend: 0,
    response_trend: 0,
  } as Overview,
  hourlyTrend: [] as HourlyTrend[],
  topQuestions: [] as TopQuestion[],
  categories: [] as CategoryItem[],
  responseTimes: [] as ResponseTimeItem[],
  satisfactionTrend: [] as SatisfactionTrend[],
  conversations: [] as Conversation[],
});

const categoryColors = ['#52f0ee', '#f4c765', '#7ef2a0', '#8aa4ff', '#ff8b8b', '#c792ea'];

const kpis = computed(() => [
  { label: '今日服务人次', value: state.overview.total_visitors, note: trendText(state.overview.visitors_trend, '较昨日'), tone: 'cyan' as const },
  { label: '本周服务人次', value: state.overview.weekly_visitors, note: '近 7 天去重会话', tone: 'gold' as const },
  { label: '今日问答次数', value: state.overview.total_chats, note: trendText(state.overview.chats_trend, '较昨日'), tone: 'gold' as const },
  { label: '游客满意度', value: state.overview.satisfaction_rate, note: trendText(state.overview.satisfaction_trend, '较昨日'), tone: 'green' as const },
  { label: '平均响应延迟', value: state.overview.avg_response_time, note: trendText(state.overview.response_trend, '响应改善'), tone: 'cyan' as const },
]);

const trendValues = computed(() => {
  const values = state.hourlyTrend.map(item => item.count);
  return values.length ? values : Array.from({ length: 24 }, () => 0);
});

const topQuestionBars = computed(() => {
  const max = Math.max(...state.topQuestions.map(item => item.count), 1);
  return state.topQuestions.map(item => ({
    label: item.question,
    value: Math.round(item.count / max * 100),
    suffix: ` / ${item.count}`,
  }));
});

const categoryDonut = computed(() => state.categories.map((item, index) => ({
  label: item.category,
  value: Math.round(item.percent),
  color: categoryColors[index % categoryColors.length],
})));

const responseBars = computed(() => state.responseTimes.map(item => ({
  label: item.bucket,
  value: Math.max(Math.round(item.percent), 2),
  raw: item.count,
})));

const satisfactionTrendValues = computed(() => {
  const values = state.satisfactionTrend.map(item => item.rate);
  return values.length ? values : Array.from({ length: 7 }, () => 0);
});

function trendText(value: number, label: string) {
  const sign = value > 0 ? '+' : '';
  return `${label} ${sign}${value.toFixed(1)}%`;
}

async function apiFetch<T>(path: string): Promise<T> {
  const token = localStorage.getItem('authToken');
  const response = await fetch(`/api/v1${path}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  });
  const raw = await response.text();
  let payload: { code?: number; message?: string; msg?: string; data?: unknown } = {};
  if (raw.trim()) {
    try {
      payload = JSON.parse(raw);
    } catch {
      throw new Error(`接口返回非 JSON 响应 (${response.status})`);
    }
  }
  if (!response.ok || payload.code !== 0) {
    throw new Error(payload.message || payload.msg || response.statusText || `请求失败 (${response.status})`);
  }
  return payload.data as T;
}

async function loadDashboard() {
  state.loading = true;
  state.error = '';
  try {
    const [overview, hourly, questions, categories, responseTimes, satisfactionTrend, conversations] = await Promise.all([
      apiFetch<Overview>('/admin/dashboard/overview'),
      apiFetch<HourlyTrend[]>('/admin/dashboard/hourly-trend'),
      apiFetch<TopQuestion[]>('/admin/dashboard/top-questions?limit=5'),
      apiFetch<CategoryItem[]>('/admin/dashboard/category-distribution'),
      apiFetch<ResponseTimeItem[]>('/admin/dashboard/response-time-distribution'),
      apiFetch<SatisfactionTrend[]>('/admin/dashboard/satisfaction-trend'),
      apiFetch<Conversation[]>('/admin/dashboard/recent-conversations?limit=6'),
    ]);
    state.overview = overview;
    state.hourlyTrend = hourly;
    state.topQuestions = questions;
    state.categories = categories;
    state.responseTimes = responseTimes;
    state.satisfactionTrend = satisfactionTrend;
    state.conversations = conversations;
  } catch (error) {
    state.error = error instanceof Error ? error.message : '数据大屏加载失败';
  } finally {
    state.loading = false;
  }
}

let clockTimer = 0;
let dashboardTimer = 0;

function tick() {
  state.now = new Date().toLocaleString('zh-CN');
}

onMounted(() => {
  tick();
  loadDashboard();
  clockTimer = window.setInterval(tick, 1000);
  dashboardTimer = window.setInterval(loadDashboard, 30000);
});

onUnmounted(() => {
  window.clearInterval(clockTimer);
  window.clearInterval(dashboardTimer);
});
</script>

<template>
  <main class="dashboard-view">
    <header class="hero-console">
      <div>
        <p class="eyebrow">LingShan Scenic AI Operation Center</p>
        <h1>景区导览 AI 数字人数据大屏</h1>
        <p>展示当日/本周服务人次、热门问答、满意度趋势与响应效率等核心运营数据。</p>
      </div>
      <div class="clock-chip">{{ state.now }}</div>
    </header>

    <div v-if="state.error" class="notice error">{{ state.error }}</div>

    <section class="kpi-grid">
      <KpiCard v-for="item in kpis" :key="item.label" v-bind="item" />
    </section>

    <section class="dashboard-grid">
      <article class="panel span-2">
        <h2>今日服务流量趋势</h2>
        <TrendChart :values="trendValues" />
      </article>

      <article class="panel">
        <h2>热门问答 Top5</h2>
        <BarList v-if="topQuestionBars.length" :items="topQuestionBars" />
        <p v-else class="muted-center">暂无问答记录</p>
      </article>

      <article class="panel">
        <h2>游客关注点分布</h2>
        <DonutChart v-if="categoryDonut.length" :items="categoryDonut" center="关注" />
        <p v-else class="muted-center">暂无分类数据</p>
      </article>

      <article class="panel">
        <h2>响应延迟分布</h2>
        <div class="response-bars">
          <div v-for="item in responseBars" :key="item.label" class="response-col" :style="{ height: `${Math.max(item.value, 8)}%` }">
            <strong>{{ item.value }}%</strong>
            <span>{{ item.label }}</span>
          </div>
        </div>
      </article>

      <article class="panel">
        <h2>满意度趋势</h2>
        <TrendChart :values="satisfactionTrendValues" />
      </article>

      <article class="panel span-2">
        <h2>实时交互片段</h2>
        <div class="chat-log">
          <div v-for="item in state.conversations" :key="`${item.time}-${item.user_query}`" class="log-line">
            <span>客</span>
            <div>
              <strong>{{ item.user_query }}</strong>
              <p>{{ item.ai_response }}</p>
            </div>
            <small>{{ item.response_time }} / {{ item.time }}</small>
          </div>
        </div>
        <p v-if="!state.conversations.length" class="muted-center">暂无实时交互记录</p>
      </article>
    </section>
  </main>
</template>
