<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';

type ScenicPoint = {
  id: string;
  name: string;
  subtitle: string;
  x: number;
  y: number;
  minutes: number;
  crowd: '舒适' | '适中' | '较多';
};

const fallbackPoints: ScenicPoint[] = [
  { id: 'gate', name: '景区入口', subtitle: '游客服务中心', x: 12, y: 78, minutes: 0, crowd: '舒适' },
  { id: 'wall', name: '灵山大照壁', subtitle: '文化序厅', x: 26, y: 62, minutes: 6, crowd: '舒适' },
  { id: 'bridge', name: '五明桥', subtitle: '山水步道', x: 42, y: 48, minutes: 10, crowd: '适中' },
  { id: 'palace', name: '灵山梵宫', subtitle: '佛教艺术殿堂', x: 62, y: 36, minutes: 16, crowd: '适中' },
  { id: 'buddha', name: '灵山大佛', subtitle: '核心朝圣点', x: 82, y: 22, minutes: 24, crowd: '较多' },
  { id: 'rest', name: '文创驿站', subtitle: '休憩与补给', x: 70, y: 68, minutes: 18, crowd: '舒适' },
];

const state = reactive({
  loading: false,
  error: '',
  points: [...fallbackPoints],
  source: '演示点位',
});

const currentPointId = ref('bridge');
const selectedPointId = ref('palace');

const currentPoint = computed(() => state.points.find(point => point.id === currentPointId.value) || state.points[0]);
const selectedPoint = computed(() => state.points.find(point => point.id === selectedPointId.value) || state.points[1] || state.points[0]);
const routePoints = computed(() => {
  const currentIndex = state.points.findIndex(point => point.id === currentPoint.value.id);
  const selectedIndex = state.points.findIndex(point => point.id === selectedPoint.value.id);
  const start = Math.min(currentIndex, selectedIndex);
  const end = Math.max(currentIndex, selectedIndex);
  return state.points.slice(start, end + 1);
});

const routePath = computed(() => state.points.map(point => `${point.x},${point.y}`).join(' '));
const activePath = computed(() => routePoints.value.map(point => `${point.x},${point.y}`).join(' '));
const walkingMinutes = computed(() => Math.max(Math.abs(selectedPoint.value.minutes - currentPoint.value.minutes), 4));
const nextSuggestions = computed(() => state.points.filter(point => point.id !== currentPoint.value.id).slice(2, 5));

function getField(item: Record<string, unknown>, key: string) {
  return item[key] ?? item[key.charAt(0).toUpperCase() + key.slice(1)] ?? '';
}

function normalizeSpot(raw: Record<string, unknown>, index: number, total: number): ScenicPoint {
  const id = String(getField(raw, 'id') || getField(raw, 'ID') || `spot-${index + 1}`);
  const rating = Number(getField(raw, 'rating') || getField(raw, 'Rating') || 4.5);
  const progress = total <= 1 ? 0 : index / (total - 1);
  return {
    id,
    name: String(getField(raw, 'name') || getField(raw, 'Name') || `景点 ${index + 1}`),
    subtitle: String(getField(raw, 'category') || getField(raw, 'location') || getField(raw, 'Description') || '景区点位'),
    x: Math.round(12 + progress * 70),
    y: Math.round(78 - progress * 56 + Math.sin(index) * 8),
    minutes: index * 6,
    crowd: rating >= 4.8 ? '较多' : rating >= 4.6 ? '适中' : '舒适',
  };
}

async function loadScenicPoints() {
  state.loading = true;
  state.error = '';
  try {
    const response = await fetch('/api/v1/spots');
    const payload = await response.json() as { code?: number; message?: string; data?: Array<Record<string, unknown>> };
    if (!response.ok || payload.code !== 0) {
      throw new Error(payload.message || `景点接口异常 (${response.status})`);
    }
    const spots = Array.isArray(payload.data) ? payload.data : [];
    if (spots.length > 0) {
      state.points = spots.map((spot, index) => normalizeSpot(spot, index, spots.length));
      currentPointId.value = state.points[0].id;
      selectedPointId.value = state.points[Math.min(1, state.points.length - 1)].id;
      state.source = '实时景点数据';
    }
  } catch (error) {
    state.error = error instanceof Error ? error.message : '景点数据加载失败，已使用演示点位';
    state.points = [...fallbackPoints];
    state.source = '演示点位';
  } finally {
    state.loading = false;
  }
}

function selectPoint(point: ScenicPoint) {
  selectedPointId.value = point.id;
}

function setCurrentPoint(point: ScenicPoint) {
  currentPointId.value = point.id;
  if (selectedPointId.value === point.id) {
    selectedPointId.value = state.points.find(item => item.id !== point.id)?.id || point.id;
  }
}

onMounted(loadScenicPoints);
</script>

<template>
  <main class="map-view">
    <header class="hero-console">
      <div>
        <p class="eyebrow">LingShan Scenic Route Map</p>
        <h1>游客实时导览地图</h1>
        <p>以当前位置为起点，展示下一站、步行时间、沿途景点和客流状态。</p>
      </div>
      <div class="status-badge connected">当前位置：{{ currentPoint.name }}</div>
    </header>

    <section class="map-layout">
      <article class="map-stage panel">
        <div v-if="state.error" class="notice error">{{ state.error }}</div>
        <div class="map-toolbar">
          <div>
            <span>推荐下一站</span>
            <strong>{{ selectedPoint.name }}</strong>
          </div>
          <div>
            <span>预计步行</span>
            <strong>{{ walkingMinutes }} 分钟</strong>
          </div>
          <div>
            <span>客流状态</span>
            <strong>{{ selectedPoint.crowd }}</strong>
          </div>
          <div>
            <span>数据来源</span>
            <strong>{{ state.loading ? '加载中' : state.source }}</strong>
          </div>
        </div>

        <svg class="scenic-map" viewBox="0 0 100 100" role="img" aria-label="灵山胜境简易路线地图">
          <defs>
            <linearGradient id="routeLine" x1="0" x2="1" y1="0" y2="0">
              <stop offset="0" stop-color="#52f0ee" />
              <stop offset="1" stop-color="#f4c765" />
            </linearGradient>
          </defs>
          <path class="terrain-shape lake" d="M10 75 C26 58 35 68 50 48 C64 30 76 37 91 15 L96 95 L8 95 Z" />
          <path class="terrain-shape hill" d="M5 35 C20 18 38 30 48 14 C65 1 82 12 94 4 L94 36 C70 30 54 44 35 40 C22 37 15 47 5 45 Z" />
          <polyline class="route-base" :points="routePath" />
          <polyline class="route-active" :points="activePath" />
          <g v-for="point in state.points" :key="point.id" class="map-point" :class="{ current: point.id === currentPoint.id, selected: point.id === selectedPoint.id }" @click="selectPoint(point)">
            <circle :cx="point.x" :cy="point.y" r="3.5" />
            <text :x="point.x" :y="point.y - 6">{{ point.name }}</text>
          </g>
        </svg>
      </article>

      <aside class="map-side">
        <article class="panel map-card">
          <h2>当前位置</h2>
          <strong>{{ currentPoint.name }}</strong>
          <p>{{ currentPoint.subtitle }}，可向 {{ selectedPoint.name }} 前进。</p>
          <div class="button-row">
            <button class="primary-action" @click="setCurrentPoint(selectedPoint)">到达此处</button>
            <button class="secondary-action" @click="selectPoint(state.points[state.points.length - 1])">查看末站</button>
          </div>
        </article>

        <article class="panel map-card">
          <h2>景点列表</h2>
          <button
            v-for="point in state.points"
            :key="point.id"
            class="route-option"
            :class="{ active: point.id === selectedPoint.id }"
            @click="selectPoint(point)"
          >
            <span>{{ point.name }}</span>
            <small>{{ point.minutes }} 分钟 / {{ point.crowd }}</small>
          </button>
        </article>

        <article class="panel map-card">
          <h2>接下来可去</h2>
          <div class="next-list">
            <button v-for="point in nextSuggestions" :key="point.id" @click="selectPoint(point)">
              <strong>{{ point.name }}</strong>
              <span>{{ point.subtitle }}</span>
            </button>
          </div>
        </article>
      </aside>
    </section>
  </main>
</template>
