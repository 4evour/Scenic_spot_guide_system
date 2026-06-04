<script setup lang="ts">
import { onMounted, onUnmounted, ref, shallowRef, nextTick } from 'vue';
import { NSpin } from 'naive-ui';
import * as echarts from 'echarts/core';
import { LineChart, PieChart } from 'echarts/charts';
import { GridComponent, TooltipComponent, LegendComponent, GraphicComponent } from 'echarts/components';
import { CanvasRenderer } from 'echarts/renderers';
import { apiFetch } from '../services/api';

echarts.use([LineChart, PieChart, GridComponent, TooltipComponent, LegendComponent, GraphicComponent, CanvasRenderer]);

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
const trendChartRef = ref<HTMLDivElement>();
const pieChartRef = ref<HTMLDivElement>();

const trendChart = shallowRef<echarts.ECharts | null>(null);
const pieChart = shallowRef<echarts.ECharts | null>(null);

const emotionColor: Record<string, string> = {
  joy: '#63e2b7', surprise: '#f8be52', neutral: '#7eb8da', sadness: '#e88080', fear: '#b8a0dc',
};
const emotionLabel: Record<string, string> = {
  joy: '😊 正面', surprise: '😮 惊喜', neutral: '😐 中性', sadness: '😢 负面', fear: '😨 恐惧',
};

async function loadData() {
  loading.value = true;
  try {
    const [ov, trend, cats, recent] = await Promise.all([
      apiFetch<Overview>('/admin/dashboard/overview'),
      apiFetch<Array<{ hour: string; count: number }>>('/admin/dashboard/hourly-trend'),
      apiFetch<CategoryItem[]>('/admin/dashboard/category-distribution'),
      apiFetch<Conversation[]>('/admin/dashboard/recent-conversations?limit=6'),
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
  const el = trendChartRef.value;
  if (!el) return;
  if (trendChart.value) trendChart.value.dispose();
  trendChart.value = echarts.init(el);
  trendChart.value.setOption({
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
  const el = pieChartRef.value;
  if (!el) return;
  if (pieChart.value) pieChart.value.dispose();
  pieChart.value = echarts.init(el);
  const colors = ['#63e2b7', '#f8be52', '#7eb8da', '#b8a0dc', '#e88080', '#a5d6a7'];
  pieChart.value?.setOption({
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

function handleResize() { trendChart.value?.resize(); pieChart.value?.resize(); }

onMounted(() => { loadData(); window.addEventListener('resize', handleResize); });
onUnmounted(() => { window.removeEventListener('resize', handleResize); trendChart.value?.dispose(); pieChart.value?.dispose(); });
</script>

<template>
  <div class="dashboard">
    <header class="page-header">
      <div>
        <h1>数据大屏</h1>
        <p>实时运营数据概览</p>
      </div>
    </header>

    <NSpin :show="loading">
      <!-- KPI 卡片 -->
      <div class="kpi-grid">
        <div class="kpi-card">
          <div class="kpi-icon" style="color: #63e2b7;">👥</div>
          <div class="kpi-body">
            <span class="kpi-label">今日服务人次</span>
            <span class="kpi-value">{{ overview.total_visitors }}</span>
            <span class="kpi-trend" :class="overview.visitors_trend >= 0 ? 'up' : 'down'">
              {{ overview.visitors_trend >= 0 ? '↑' : '↓' }} {{ Math.abs(overview.visitors_trend) }}%
            </span>
          </div>
        </div>
        <div class="kpi-card">
          <div class="kpi-icon" style="color: #52f0ee;">💬</div>
          <div class="kpi-body">
            <span class="kpi-label">本周问答次数</span>
            <span class="kpi-value">{{ overview.weekly_chats }}</span>
            <span class="kpi-trend" :class="overview.chats_trend >= 0 ? 'up' : 'down'">
              {{ overview.chats_trend >= 0 ? '↑' : '↓' }} {{ Math.abs(overview.chats_trend) }}%
            </span>
          </div>
        </div>
        <div class="kpi-card">
          <div class="kpi-icon" style="color: #7ef2a0;">⭐</div>
          <div class="kpi-body">
            <span class="kpi-label">用户满意度</span>
            <span class="kpi-value highlight-green">{{ overview.satisfaction_rate }}</span>
            <span class="kpi-trend" :class="overview.satisfaction_trend >= 0 ? 'up' : 'down'">
              {{ overview.satisfaction_trend >= 0 ? '↑' : '↓' }} {{ Math.abs(overview.satisfaction_trend) }}%
            </span>
          </div>
        </div>
        <div class="kpi-card">
          <div class="kpi-icon" style="color: #f4c765;">⚡</div>
          <div class="kpi-body">
            <span class="kpi-label">平均响应延迟</span>
            <span class="kpi-value highlight-gold">{{ overview.avg_response_time }}</span>
            <span class="kpi-trend" :class="overview.response_trend <= 0 ? 'up' : 'down'">
              {{ overview.response_trend <= 0 ? '↓' : '↑' }} {{ Math.abs(overview.response_trend) }}%
            </span>
          </div>
        </div>
      </div>

      <!-- 图表区域 -->
      <div class="chart-grid">
        <div class="panel chart-card chart-wide">
          <h3 class="panel-title">📈 24 小时流量趋势</h3>
          <div ref="trendChartRef" class="chart-container"></div>
        </div>
        <div class="panel chart-card">
          <h3 class="panel-title">🎯 关注点分布</h3>
          <div ref="pieChartRef" class="chart-container"></div>
        </div>
      </div>

      <!-- 最近对话 -->
      <div class="panel chart-card">
        <h3 class="panel-title">💬 最近对话</h3>
        <div class="table-wrap">
          <table class="data-table">
            <thead>
              <tr>
                <th>时间</th><th>游客问题</th><th>AI 回答</th><th>情绪</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(item, idx) in recentConversations" :key="idx">
                <td class="col-time">{{ item.time }}</td>
                <td class="col-text">{{ item.user_query }}</td>
                <td class="col-text">{{ item.ai_response }}</td>
                <td>
                  <span class="emotion-tag" :style="{ color: emotionColor[item.emotion] || '#7eb8da', borderColor: (emotionColor[item.emotion] || '#7eb8da') + '40', background: (emotionColor[item.emotion] || '#7eb8da') + '15' }">
                    {{ emotionLabel[item.emotion] || item.emotion }}
                  </span>
                </td>
              </tr>
              <tr v-if="recentConversations.length === 0">
                <td colspan="4" class="empty-row">暂无对话记录</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </NSpin>
  </div>
</template>

<style scoped>
.dashboard {
  padding: 28px 32px;
  background: #0a0a0f;
  min-height: 100%;
}

/* 页面头部 */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  margin-bottom: 28px;
}
.page-header h1 {
  font-size: 22px;
  font-weight: 700;
  color: rgba(255,255,255,0.92);
  margin-bottom: 4px;
}
.page-header p {
  font-size: 13px;
  color: rgba(255,255,255,0.35);
}

/* KPI 网格 */
.kpi-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}
.kpi-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px 24px;
  background: rgba(255,255,255,0.03);
  border: 1px solid rgba(255,255,255,0.06);
  border-radius: 14px;
  transition: all 0.2s;
}
.kpi-card:hover {
  background: rgba(255,255,255,0.05);
  border-color: rgba(99, 226, 183, 0.12);
  transform: translateY(-1px);
}
.kpi-icon {
  font-size: 28px;
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
  background: rgba(255,255,255,0.03);
  flex-shrink: 0;
}
.kpi-body {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.kpi-label {
  font-size: 12px;
  color: rgba(255,255,255,0.4);
  margin-bottom: 4px;
}
.kpi-value {
  font-size: 26px;
  font-weight: 700;
  color: rgba(255,255,255,0.92);
  line-height: 1.2;
}
.kpi-value.highlight-green { color: #63e2b7; }
.kpi-value.highlight-gold { color: #f4c765; }
.kpi-trend {
  font-size: 11px;
  font-weight: 600;
  margin-top: 2px;
}
.kpi-trend.up { color: #63e2b7; }
.kpi-trend.down { color: #e88080; }

/* 图表网格 */
.chart-grid {
  display: grid;
  grid-template-columns: 2fr 1fr;
  gap: 16px;
  margin-bottom: 24px;
}
.chart-wide { grid-column: 1; }

.panel {
  background: rgba(255,255,255,0.03);
  border: 1px solid rgba(255,255,255,0.06);
  border-radius: 14px;
  padding: 20px 24px;
}
.panel-title {
  font-size: 14px;
  font-weight: 600;
  color: rgba(255,255,255,0.65);
  margin-bottom: 16px;
}
.chart-container { height: 300px; }

/* 数据表格 */
.table-wrap { overflow-x: auto; }
.data-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}
.data-table th {
  text-align: left;
  padding: 10px 12px;
  color: rgba(255,255,255,0.4);
  font-weight: 500;
  font-size: 12px;
  border-bottom: 1px solid rgba(255,255,255,0.06);
}
.data-table td {
  padding: 10px 12px;
  color: rgba(255,255,255,0.7);
  border-bottom: 1px solid rgba(255,255,255,0.03);
}
.data-table tr:hover td {
  background: rgba(255,255,255,0.02);
}
.col-time {
  color: rgba(255,255,255,0.35) !important;
  font-size: 12px;
  white-space: nowrap;
}
.col-text {
  max-width: 280px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.emotion-tag {
  display: inline-block;
  padding: 2px 10px;
  border-radius: 10px;
  font-size: 11px;
  font-weight: 500;
  border: 1px solid;
}
.empty-row {
  text-align: center;
  color: rgba(255,255,255,0.25);
  padding: 32px !important;
}

@media (max-width: 1200px) {
  .chart-grid { grid-template-columns: 1fr; }
}
@media (max-width: 768px) {
  .dashboard { padding: 16px; }
  .kpi-grid { grid-template-columns: 1fr 1fr; }
  .chart-container { height: 220px; }
}
@media (max-width: 480px) {
  .kpi-grid { grid-template-columns: 1fr; }
}
</style>
