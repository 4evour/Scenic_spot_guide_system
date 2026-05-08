<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive } from 'vue';
import KpiCard from '../components/KpiCard.vue';
import TrendChart from '../components/TrendChart.vue';
import BarList from '../components/BarList.vue';
import DonutChart from '../components/DonutChart.vue';

const state = reactive({
  visitors: 12356,
  chats: 8923,
  satisfaction: 96.8,
  response: 2.1,
  online: 2345,
  now: '',
  trend: [420, 360, 310, 280, 390, 560, 920, 1360, 1880, 2140, 2360, 2510, 2420, 2300, 2260, 2180, 1980, 1760, 1430, 1180, 920, 760, 620, 510],
});

const questions = [
  { label: '灵山主峰多久能游完', value: 100, suffix: '' },
  { label: '历史文化路线推荐', value: 92, suffix: '' },
  { label: '停车场和卫生间位置', value: 85, suffix: '' },
  { label: '亲子游览怎么安排', value: 78, suffix: '' },
  { label: '雨天还能游览吗', value: 71, suffix: '' },
];

const categories = [
  { label: '服务设施', value: 42, color: '#52f0ee' },
  { label: '历史文化', value: 24, color: '#f4c765' },
  { label: '路线推荐', value: 18, color: '#7ef2a0' },
  { label: '票务交通', value: 16, color: '#8aa4ff' },
];

const responseBars = [
  { label: '<1s', value: 42 },
  { label: '1-3s', value: 38 },
  { label: '3-5s', value: 15 },
  { label: '>5s', value: 5 },
];

const conversations = [
  ['游客', '灵山主峰从哪个入口上去最省力？', '刚刚'],
  ['AI', '建议从北门进入，先走文化长廊，再乘接驳车到半山观景点。', '2.1s'],
  ['游客', '下午带老人来适合走完整路线吗？', '2分钟前'],
  ['AI', '建议选择 60 分钟舒缓路线，避开陡坡，把观景台和文创驿站作为重点。', '1.9s'],
];

const kpis = computed(() => [
  { label: '今日服务人次', value: state.visitors.toLocaleString('zh-CN'), note: '较昨日 +12.5%', tone: 'cyan' as const },
  { label: 'AI 问答次数', value: state.chats.toLocaleString('zh-CN'), note: '知识库命中 91.7%', tone: 'gold' as const },
  { label: '游客满意度', value: `${state.satisfaction.toFixed(1)}%`, note: '正向反馈持续上升', tone: 'green' as const },
  { label: '平均响应延迟', value: `${state.response.toFixed(1)}s`, note: '满足 5 秒指标', tone: 'cyan' as const },
  { label: '实时在线游客', value: state.online.toLocaleString('zh-CN'), note: 'App + 终端设备', tone: 'gold' as const },
]);

let timer = 0;

function tick() {
  state.now = new Date().toLocaleString('zh-CN');
  state.visitors += Math.floor(Math.random() * 9) + 1;
  state.chats += Math.floor(Math.random() * 5) + 1;
  state.online = Math.max(2100, state.online + Math.floor(Math.random() * 13) - 5);
}

onMounted(() => {
  tick();
  timer = window.setInterval(tick, 2500);
});

onUnmounted(() => window.clearInterval(timer));
</script>

<template>
  <main class="dashboard-view">
    <header class="hero-console">
      <div>
        <p class="eyebrow">LingShan Scenic AI Operation Center</p>
        <h1>景区导览 AI 数字人数据大屏</h1>
        <p>模拟展示服务人次、热门问答、满意度趋势与知识库命中情况，用于比赛演示和方案汇报。</p>
      </div>
      <div class="clock-chip">{{ state.now }}</div>
    </header>

    <section class="kpi-grid">
      <KpiCard v-for="item in kpis" :key="item.label" v-bind="item" />
    </section>

    <section class="dashboard-grid">
      <article class="panel span-2">
        <h2>全天服务流量趋势</h2>
        <TrendChart :values="state.trend" />
      </article>

      <article class="panel">
        <h2>热门问答 Top5</h2>
        <BarList :items="questions" />
      </article>

      <article class="panel">
        <h2>游客关注点分布</h2>
        <DonutChart :items="categories" center="RAG" />
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
        <h2>游客情绪感受度</h2>
        <div class="emotion-ring">
          <strong>96.8%</strong>
          <span>满意度</span>
        </div>
        <p class="muted-center">正向 78% / 中性 17% / 待跟进 5%</p>
      </article>

      <article class="panel span-2">
        <h2>实时交互片段</h2>
        <div class="chat-log">
          <div v-for="(item, index) in conversations" :key="index" class="log-line" :class="{ ai: item[0] === 'AI' }">
            <span>{{ item[0] === 'AI' ? 'AI' : '客' }}</span>
            <div>
              <strong>{{ item[0] }}</strong>
              <p>{{ item[1] }}</p>
            </div>
            <small>{{ item[2] }}</small>
          </div>
        </div>
      </article>
    </section>
  </main>
</template>
