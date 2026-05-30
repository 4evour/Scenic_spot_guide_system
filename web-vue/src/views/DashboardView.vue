<script setup lang="ts">
import { onMounted, onUnmounted, ref, nextTick } from 'vue';
import { NGrid, NGi, NCard, NSpin, NStatistic, NTbody, NTable, NTr, NTd, NTh, NTag } from 'naive-ui';
import * as echarts from 'echarts';

const token = localStorage.getItem('authToken');
const headers: HeadersInit = {
  'Content-Type': 'application/json',
  ...(token ? { Authorization: `Bearer ${token}` } : {}),
};

interface Overview {
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
}

interface CategoryItem {
  category: string;
  count: number;
  percent: number;
}

interface Conversation {
  user_query: string;
  ai_response: string;
  emotion: string;
  response_time: string;
  time: string;
}

const loading = ref(true);
const overview = ref<Overview>({
  total_visitors: '0', weekly_visitors: '0', total_chats: '0', weekly_chats: '0',
  satisfaction_rate: '0%', avg_response_time: '0s',
  visitors_trend: 0, chats_trend: 0, satisfaction_trend: 0, response_trend: 0,
});
const recentConversations = ref<Conversation[]>([]);

let trendChart: echarts.ECharts | null = null;
let pieChart: echarts.ECharts | null = null;

const emotionColor: Record<string, string> = {
  joy: '#63e2b7', surprise: '#f8be52', neutral: '#7eb8da', sadness: '#e88080', fear: '#b8a0dc',
};
const emotionLabel: Record<string, string> = {
  joy: '😊 正面', surprise: '😮 惊喜', neutral: '😐 中性', sadness: '😢 负面', fear: '😨 恐惧',
};

async function fetchJSON<T>(path: string): Promise<T> {
  const res = await fetch(`/api/v1${path}`, { headers });
  const data = await res.json();
  return data.data as T;
}

async function loadData() {
  loading.value = true;
  try {
    const [ov, trend, cats, recent] = await Promise.all([
      fetchJSON<Overview>('/admin/dashboard/overview'),
      fetchJSON<Array<{ hour: string; count: number }>>('/admin/dashboard/hourly-trend'),
      fetchJSON<CategoryItem[]>('/admin/dashboard/category-distribution'),
      fetchJSON<Conversation[]>('/admin/dashboard/recent-conversations?limit=6'),
    ]);
    if (ov) overview.value = ov;
    recentConversations.value = recent || [];

    await nextTick();
    renderTrendChart(trend || []);
    renderPieChart(cats || []);
  } catch (e) {
    console.error('Dashboard load error:', e);
  } finally {
    loading.value = false;
  }
}

function renderTrendChart(data: Array<{ hour: string; count: number }>) {
  const el = document.getElementById('trendChart');
  if (!el) return;
  if (trendChart) trendChart.dispose();
  trendChart = echarts.init(el);
  trendChart.setOption({
    backgroundColor: 'transparent',
    grid: { top: 30, right: 16, bottom: 30, left: 50 },
    xAxis: {
      type: 'category',
      data: data.map(d => d.hour),
      axisLine: { lineStyle: { color: 'rgba(255,255,255,0.08)' } },
      axisLabel: { color: 'rgba(255,255,255,0.35)', fontSize: 11 },
    },
    yAxis: {
      type: 'value',
      splitLine: { lineStyle: { color: 'rgba(255,255,255,0.04)' } },
      axisLabel: { color: 'rgba(255,255,255,0.35)', fontSize: 11 },
    },
    series: [{
      type: 'line', smooth: true, symbol: 'circle', symbolSize: 6,
      lineStyle: { color: '#63e2b7', width: 2 },
      itemStyle: { color: '#63e2b7' },
      areaStyle: { color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
        { offset: 0, color: 'rgba(99,226,183,0.25)' },
        { offset: 1, color: 'rgba(99,226,183,0)' },
      ])},
      data: data.map(d => d.count),
    }],
    tooltip: { trigger: 'axis', backgroundColor: 'rgba(20,20,30,0.95)', borderColor: 'rgba(255,255,255,0.08)', textStyle: { color: '#fff', fontSize: 12 } },
  });
}

function renderPieChart(data: CategoryItem[]) {
  const el = document.getElementById('pieChart');
  if (!el) return;
  if (pieChart) pieChart.dispose();
  pieChart = echarts.init(el);
  const colors = ['#63e2b7', '#f8be52', '#7eb8da', '#b8a0dc', '#e88080', '#a5d6a7'];
  pieChart.setOption({
    backgroundColor: 'transparent',
    series: [{
      type: 'pie', radius: ['42%', '70%'], center: ['50%', '50%'],
      label: { color: 'rgba(255,255,255,0.6)', fontSize: 11 },
      labelLine: { lineStyle: { color: 'rgba(255,255,255,0.15)' } },
      data: data.map((d, i) => ({ value: d.count, name: d.category, itemStyle: { color: colors[i % colors.length] } })),
    }],
    tooltip: { backgroundColor: 'rgba(20,20,30,0.95)', borderColor: 'rgba(255,255,255,0.08)', textStyle: { color: '#fff', fontSize: 12 } },
  });
}

function handleResize() { trendChart?.resize(); pieChart?.resize(); }

onMounted(() => { loadData(); window.addEventListener('resize', handleResize); });
onUnmounted(() => { window.removeEventListener('resize', handleResize); trendChart?.dispose(); pieChart?.dispose(); });
</script>

<template>
  <div class="dashboard">
    <div class="page-header">
      <h1>数据大屏</h1>
      <p>实时运营数据概览</p>
    </div>

    <NSpin :show="loading">
      <NGrid :cols="4" :x-gap="16" :y-gap="16" style="margin-bottom: 24px;">
        <NGi>
          <NCard size="small" class="kpi-card">
            <NStatistic label="今日服务人次" :value="overview.total_visitors">
              <template #suffix><span style="font-size:12px;color:rgba(255,255,255,0.35);">人次</span></template>
            </NStatistic>
          </NCard>
        </NGi>
        <NGi>
          <NCard size="small" class="kpi-card">
            <NStatistic label="本周问答次数" :value="overview.weekly_chats">
              <template #suffix><span style="font-size:12px;color:rgba(255,255,255,0.35);">次</span></template>
            </NStatistic>
          </NCard>
        </NGi>
        <NGi>
          <NCard size="small" class="kpi-card">
            <NStatistic label="用户满意度">
              <template #default><span style="font-size:28px;font-weight:700;color:#63e2b7;">{{ overview.satisfaction_rate }}</span></template>
            </NStatistic>
          </NCard>
        </NGi>
        <NGi>
          <NCard size="small" class="kpi-card">
            <NStatistic label="平均响应延迟">
              <template #default><span style="font-size:28px;font-weight:700;color:#f8be52;">{{ overview.avg_response_time }}</span></template>
            </NStatistic>
          </NCard>
        </NGi>
      </NGrid>

      <NGrid :cols="3" :x-gap="16" :y-gap="16" style="margin-bottom: 24px;">
        <NGi :span="2">
          <NCard size="small" class="chart-card">
            <template #header><span class="card-title">📈 24 小时流量趋势</span></template>
            <div id="trendChart" style="height: 300px;"></div>
          </NCard>
        </NGi>
        <NGi>
          <NCard size="small" class="chart-card">
            <template #header><span class="card-title">🎯 关注点分布</span></template>
            <div id="pieChart" style="height: 300px;"></div>
          </NCard>
        </NGi>
      </NGrid>

      <NCard size="small" class="chart-card">
        <template #header><span class="card-title">💬 最近对话</span></template>
        <NTable :bordered="false" size="small">
          <thead>
            <NTr>
              <NTh>时间</NTh><NTh>游客问题</NTh><NTh>AI 回答</NTh><NTh>情绪</NTh>
            </NTr>
          </thead>
          <NTbody>
            <NTr v-for="(item, idx) in recentConversations" :key="idx">
              <NTd style="color:rgba(255,255,255,0.35);font-size:12px;white-space:nowrap;">{{ item.time }}</NTd>
              <NTd style="max-width:200px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;">{{ item.user_query }}</NTd>
              <NTd style="max-width:300px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;">{{ item.ai_response }}</NTd>
              <NTd><NTag :color="{ color: (emotionColor[item.emotion] || '#7eb8da') + '20', textColor: emotionColor[item.emotion] || '#7eb8da', borderColor: (emotionColor[item.emotion] || '#7eb8da') + '40' }" size="small" :bordered="true">{{ emotionLabel[item.emotion] || item.emotion }}</NTag></NTd>
            </NTr>
          </NTbody>
        </NTable>
      </NCard>
    </NSpin>
  </div>
</template>

<style scoped>
.dashboard { padding: 24px; background: #0a0a0f; min-height: 100%; }
.page-header { margin-bottom: 24px; }
.page-header h1 { font-size: 20px; font-weight: 600; color: rgba(255,255,255,0.88); margin-bottom: 4px; }
.page-header p { font-size: 13px; color: rgba(255,255,255,0.35); }
.kpi-card { background: rgba(255,255,255,0.04) !important; border: 1px solid rgba(255,255,255,0.08) !important; border-radius: 12px !important; }
.chart-card { background: rgba(255,255,255,0.04) !important; border: 1px solid rgba(255,255,255,0.08) !important; border-radius: 12px !important; }
.card-title { font-size: 14px; color: rgba(255,255,255,0.65); }
</style>
