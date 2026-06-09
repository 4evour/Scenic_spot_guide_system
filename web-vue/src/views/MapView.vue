<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue';
import { NInput, NButton, NTag, NEmpty, NSpin, NAlert } from 'naive-ui';

declare const AMap: Record<string, unknown>;

type ScenicSpot = {
  id: string;
  name: string;
  category: string;
  description: string;
  lng: number;
  lat: number;
  rating: number;
  price: number;
  imageUrl: string;
};

const AMAP_KEY = import.meta.env.VITE_AMAP_KEY || '';
const AMAP_SECURITY = import.meta.env.VITE_AMAP_SECURITY || '';

const fallbackSpots: ScenicSpot[] = [
  { id: 'gate', name: '景区入口', category: '服务设施', description: '游客服务中心，购票和咨询', lng: 120.4155, lat: 31.5720, rating: 4.5, price: 0, imageUrl: '' },
  { id: 'wall', name: '灵山大照壁', category: '文化建筑', description: '灵山胜境的文化序厅', lng: 120.4168, lat: 31.5705, rating: 4.6, price: 0, imageUrl: '' },
  { id: 'bridge', name: '五明桥', category: '景观', description: '通往景区核心的山水步道', lng: 120.4182, lat: 31.5690, rating: 4.5, price: 0, imageUrl: '' },
  { id: 'foot', name: '佛足坛', category: '核心景点', description: '释迦牟尼足印圣迹', lng: 120.4195, lat: 31.5678, rating: 4.7, price: 0, imageUrl: '' },
  { id: 'jiulong', name: '九龙灌浴', category: '演艺体验', description: '再现释迦牟尼诞生的动态音乐喷泉', lng: 120.4208, lat: 31.5665, rating: 4.8, price: 0, imageUrl: '' },
  { id: 'hand', name: '天下第一掌', category: '互动体验', description: '灵山大佛右手复制件，摸佛手增福添寿', lng: 120.4215, lat: 31.5655, rating: 4.7, price: 0, imageUrl: '' },
  { id: 'mile', name: '百子戏弥勒', category: '文化景观', description: '轻松亲和的铜雕群', lng: 120.4220, lat: 31.5648, rating: 4.6, price: 0, imageUrl: '' },
  { id: 'temple', name: '祥符禅寺', category: '宗教场所', description: '千年古刹，灵山佛教文化源头', lng: 120.4228, lat: 31.5638, rating: 4.7, price: 0, imageUrl: '' },
  { id: 'buddha', name: '灵山大佛', category: '核心景点', description: '高88米的青铜立佛，灵山标志性景点', lng: 120.4235, lat: 31.5625, rating: 4.9, price: 0, imageUrl: '' },
  { id: 'fangong', name: '灵山梵宫', category: '文化建筑', description: '汇集东阳木雕、琉璃等工艺的佛教艺术殿堂', lng: 120.4200, lat: 31.5658, rating: 4.9, price: 0, imageUrl: '' },
  { id: 'wuyin', name: '五印坛城', category: '文化建筑', description: '藏传佛教文化主题展示', lng: 120.4218, lat: 31.5640, rating: 4.7, price: 0, imageUrl: '' },
  { id: 'rest', name: '文创驿站', category: '服务设施', description: '文创商品、饮品和休憩服务', lng: 120.4190, lat: 31.5680, rating: 4.5, price: 0, imageUrl: '' },
];

function escapeHtml(str: string): string {
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;').replace(/'/g, '&#039;');
}

const categoryTagType: Record<string, 'success' | 'info' | 'warning' | 'error' | 'default'> = {
  '核心景点': 'success',
  '文化建筑': 'warning',
  '演艺体验': 'info',
  '服务设施': 'default',
};

const state = reactive({
  loading: false,
  error: '',
  spots: [] as ScenicSpot[],
  source: '演示数据',
  selectedSpot: null as ScenicSpot | null,
  mapReady: false,
  search: '',
});

const filteredSpots = computed(() => {
  const term = state.search.trim().toLowerCase();
  if (!term) return state.spots;
  return state.spots.filter(s =>
    `${s.name}${s.category}${s.description}`.toLowerCase().includes(term),
  );
});

const mapContainer = ref<HTMLDivElement>();
let map: unknown = null;
let markers: unknown[] = [];
let infoWindow: unknown = null;

async function loadSpots() {
  state.loading = true;
  state.error = '';
  try {
    const response = await fetch('/api/v1/spots', { signal: AbortSignal.timeout(15000) });
    const payload = await response.json() as { code?: number; message?: string; data?: Array<Record<string, unknown>> };
    if (!response.ok || payload.code !== 0) throw new Error(payload.message || '加载失败');
    const spots = (payload.data || []).map((raw, i) => ({
      id: String(raw.id || raw.ID || `spot-${i}`),
      name: String(raw.name || raw.Name || `景点${i + 1}`),
      category: String(raw.category || raw.Category || ''),
      description: String(raw.description || raw.Description || ''),
      lng: Number(raw.longitude || raw.Longitude || 120.42 + (i * 0.002)),
      lat: Number(raw.latitude || raw.Latitude || 31.57 - (i * 0.002)),
      rating: Number(raw.rating || raw.Rating || 4.5),
      price: Number(raw.price || raw.Price || 0),
      imageUrl: String(raw.image_url || raw.ImageURL || ''),
    }));
    if (spots.length > 0 && spots.some(s => s.lng > 100)) {
      state.spots = spots;
      state.source = '实时数据';
    } else {
      state.spots = [...fallbackSpots];
      state.source = '演示数据（景点未配置经纬度）';
    }
  } catch {
    state.spots = [...fallbackSpots];
    state.source = '演示数据';
  } finally {
    state.loading = false;
  }
}

function loadAmapScript(): Promise<void> {
  return new Promise((resolve, reject) => {
    if (typeof AMap !== 'undefined') { resolve(); return; }
    if (!AMAP_KEY) { reject(new Error('地图配置缺失')); return; }
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    if (AMAP_SECURITY) {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      (window as any)._AMapSecurityConfig = { securityJsCode: AMAP_SECURITY };
    }
    const script = document.createElement('script');
    script.src = `https://webapi.amap.com/maps?v=2.0&key=${AMAP_KEY}&plugin=AMap.Scale,AMap.ToolBar,AMap.Walking,AMap.Geolocation`;
    script.onload = () => resolve();
    script.onerror = () => reject(new Error('高德地图加载失败'));
    document.head.appendChild(script);
  });
}

function initMap() {
  if (!mapContainer.value || typeof AMap === 'undefined') return;
  const AMapMap = AMap.Map as new (container: HTMLElement, opts: Record<string, unknown>) => unknown;
  const AMapMarker = AMap.Marker as new (opts: Record<string, unknown>) => unknown;
  const AMapInfoWindow = AMap.InfoWindow as new (opts: Record<string, unknown>) => unknown;
  const AMapPolyline = AMap.Polyline as new (opts: Record<string, unknown>) => unknown;

  map = new AMapMap(mapContainer.value, {
    zoom: 16,
    center: [120.4200, 31.5670],
    mapStyle: 'amap://styles/normal',
    features: ['bg', 'road', 'building', 'point'],
  });

  infoWindow = new AMapInfoWindow({
    isCustom: true,
    offset: new (AMap.Pixel as new (x: number, y: number) => unknown)(0, -40),
  });

  markers = [];
  const path: number[][] = [];
  for (const spot of state.spots) {
    const marker = new AMapMarker({
      position: [spot.lng, spot.lat],
      title: spot.name,
      content: `<div style="
        width: 32px; height: 32px; border-radius: 50%;
        background: ${spot.rating >= 4.8 ? '#f4c765' : '#52f0ee'};
        border: 2px solid rgba(255,255,255,0.9);
        display: flex; align-items: center; justify-content: center;
        font-size: 13px; color: #0a0a0f; font-weight: bold;
        box-shadow: 0 2px 12px rgba(0,0,0,0.3);
        cursor: pointer; transition: transform 0.2s;
      ">${escapeHtml(spot.name.charAt(0))}</div>`,
      extData: spot,
    });
    (marker as unknown as { on: (event: string, cb: () => void) => void }).on('click', () => showSpotInfo(spot));
    markers.push(marker);
    path.push([spot.lng, spot.lat]);
  }

  const polyline = new AMapPolyline({
    path,
    strokeColor: '#52f0ee',
    strokeWeight: 3,
    strokeOpacity: 0.6,
    lineJoin: 'round',
    lineCap: 'round',
    strokeStyle: 'dashed',
  });

  (map as { add: (overlays: unknown[]) => void }).add([...markers, polyline]);
  (map as { setFitView: () => void }).setFitView();
  state.mapReady = true;
}

function showSpotInfo(spot: ScenicSpot) {
  state.selectedSpot = spot;
  if (infoWindow && map) {
    const priceText = spot.price > 0 ? `¥${spot.price}` : '免费';
    const content = `<div style="
      background: rgba(6,16,18,0.96); color: rgba(255,255,255,0.88);
      padding: 16px 20px; border-radius: 12px; min-width: 240px; max-width: 320px;
      border: 1px solid rgba(82,240,238,0.15);
      font-family: 'Microsoft YaHei', sans-serif;
      box-shadow: 0 8px 32px rgba(0,0,0,0.4);
      backdrop-filter: blur(8px);
    ">
      <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:8px;">
        <h3 style="margin:0; color:#f4c765; font-size:16px; font-weight:700;">${escapeHtml(spot.name)}</h3>
        <span style="padding:2px 8px; border-radius:8px; font-size:10px; background:rgba(99,226,183,0.12); color:#63e2b7; border:1px solid rgba(99,226,183,0.2);">${escapeHtml(spot.category)}</span>
      </div>
      <p style="margin:0 0 10px; font-size:13px; line-height:1.6; color:rgba(255,255,255,0.65);">${escapeHtml(spot.description)}</p>
      <div style="display:flex; gap:16px; font-size:12px; color:rgba(255,255,255,0.5); padding-top:8px; border-top:1px solid rgba(255,255,255,0.06);">
        <span style="color:#f4c765;">⭐ ${spot.rating}</span>
        <span>${priceText}</span>
      </div>
    </div>`;
    (infoWindow as { setContent: (c: string) => void; open: (m: unknown, pos: number[]) => void })
      .setContent(content);
    (infoWindow as { open: (m: unknown, pos: number[]) => void })
      .open(map, [spot.lng, spot.lat]);
  }
}

function flyToSpot(spot: ScenicSpot) {
  showSpotInfo(spot);
  if (map) {
    (map as { setCenter: (pos: number[]) => void }).setCenter([spot.lng, spot.lat]);
    (map as { setZoom: (z: number) => void }).setZoom(17);
  }
}

function locateMe() {
  if (!map) return;
  const Geo = AMap.Geolocation as new (opts: Record<string, unknown>) => unknown;
  const geolocation = new Geo({ enableHighAccuracy: true, timeout: 10000 });
  (geolocation as { getCurrentPosition: (cb: (status: string, result: Record<string, unknown>) => void) => void })
    .getCurrentPosition((status: string, result: Record<string, unknown>) => {
      if (status === 'complete' && result.position) {
        const pos = result.position as { lng: number; lat: number };
        (map as { setCenter: (pos: number[]) => void }).setCenter([pos.lng, pos.lat]);
        (map as { setZoom: (z: number) => void }).setZoom(17);
      }
    });
}

onMounted(async () => {
  await loadSpots();
  try {
    await loadAmapScript();
    initMap();
  } catch (e) {
    state.error = e instanceof Error ? e.message : '地图加载失败';
  }
});

onUnmounted(() => {
  if (map) {
    (map as { destroy: () => void }).destroy();
    map = null;
  }
});
</script>

<template>
  <main class="map-view">
    <header class="map-header">
      <div>
        <h1>游客实时导览地图</h1>
        <p>点击查看景点详情，支持定位和步行导航</p>
      </div>
      <div class="map-status" :class="state.mapReady ? 'ready' : 'loading'">
        <span class="status-dot"></span>
        {{ state.mapReady ? '地图就绪' : '加载中' }}
        <small>{{ state.source }}</small>
      </div>
    </header>

    <section class="map-layout">
      <article class="map-stage">
        <NAlert v-if="state.error" type="error" :show-icon="true" style="margin: 8px 16px;">
          {{ state.error }}
        </NAlert>
        <div ref="mapContainer" class="amap-container"></div>
        <div class="map-toolbar">
          <NButton size="small" quaternary @click="locateMe">
            📍 我的位置
          </NButton>
          <NSpin v-if="state.loading" size="small" />
          <span v-if="state.loading" class="loading-text">加载景点数据...</span>
        </div>
      </article>

      <aside class="map-sidebar">
        <!-- 选中景点详情 -->
        <article v-if="state.selectedSpot" class="spot-card">
          <div class="spot-card-header">
            <h2>{{ state.selectedSpot.name }}</h2>
            <NTag :type="categoryTagType[state.selectedSpot.category] || 'default'" size="small" :bordered="false">
              {{ state.selectedSpot.category }}
            </NTag>
          </div>
          <p class="spot-desc">{{ state.selectedSpot.description }}</p>
          <div class="spot-meta">
            <span class="spot-rating">⭐ {{ state.selectedSpot.rating }}</span>
            <span class="spot-price">{{ state.selectedSpot.price > 0 ? '¥' + state.selectedSpot.price : '免费' }}</span>
          </div>
        </article>

        <!-- 搜索 + 景点列表 -->
        <article class="spot-list-card">
          <div class="spot-list-header">
            <h3>景点 <small>({{ filteredSpots.length }})</small></h3>
          </div>
          <NInput
            v-model:value="state.search"
            placeholder="搜索景点..."
            size="small"
            clearable
            style="margin-bottom: 10px;"
          />
          <div class="spot-list">
            <button
              v-for="spot in filteredSpots"
              :key="spot.id"
              class="spot-item"
              :class="{ active: state.selectedSpot?.id === spot.id }"
              @click="flyToSpot(spot)"
            >
              <span class="spot-item-name">{{ spot.name }}</span>
              <span class="spot-item-meta">{{ spot.category }} · ⭐{{ spot.rating }}</span>
            </button>
            <NEmpty v-if="filteredSpots.length === 0" description="没有匹配的景点" size="small" />
          </div>
        </article>
      </aside>
    </section>
  </main>
</template>

<style scoped>
.map-view {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 56px);
  background: var(--sg-bg-ink, #031012);
}

.map-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 28px;
  border-bottom: 1px solid var(--sg-border-subtle, rgba(255,255,255,0.04));
  flex-shrink: 0;
}
.map-header h1 {
  font-size: 18px;
  font-weight: 700;
  color: var(--sg-text-heading, rgba(255,255,255,0.92));
  margin-bottom: 2px;
}
.map-header p {
  font-size: 12px;
  color: var(--sg-text-hint, rgba(255,255,255,0.35));
}

.map-status {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 14px;
  border-radius: 20px;
  font-size: 12px;
  color: var(--sg-text-secondary, rgba(255,255,255,0.5));
  background: var(--sg-surface-card, rgba(255,255,255,0.03));
  border: 1px solid var(--sg-border-soft, rgba(255,255,255,0.06));
}
.map-status small { color: var(--sg-text-faint, rgba(255,255,255,0.25)); }
.status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--sg-red-bright, #e88080);
}
.map-status.ready .status-dot { background: var(--sg-jade-bright, #63e2b7); }

.map-layout {
  display: grid;
  grid-template-columns: 1fr 360px;
  flex: 1;
  min-height: 0;
}

.map-stage {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.amap-container {
  flex: 1;
  min-height: 300px;
  background: #0a1a1e;
}

.map-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 16px;
  border-top: 1px solid var(--sg-border-subtle, rgba(255,255,255,0.04));
}
.loading-text { color: var(--sg-text-faint, rgba(255,255,255,0.3)); font-size: 12px; }

.map-sidebar {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px;
  overflow-y: auto;
  border-left: 1px solid var(--sg-border-subtle, rgba(255,255,255,0.04));
  background: rgba(255,255,255,0.01);
}

.spot-card {
  background: var(--sg-surface-card, rgba(255,255,255,0.03));
  border: 1px solid var(--sg-jade-border, rgba(99,226,183,0.12));
  border-radius: 14px;
  padding: 20px;
}
.spot-card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 10px;
}
.spot-card-header h2 {
  font-size: 16px;
  font-weight: 700;
  color: var(--sg-gold, #f4c765);
  margin: 0;
}
.spot-desc {
  font-size: 13px;
  line-height: 1.6;
  color: var(--sg-text-secondary, rgba(255,255,255,0.6));
  margin: 0 0 12px;
}
.spot-meta {
  display: flex;
  gap: 16px;
  font-size: 13px;
}
.spot-rating { color: var(--sg-gold, #f4c765); }
.spot-price { color: var(--sg-text-secondary, rgba(255,255,255,0.5)); }

.spot-list-card {
  background: var(--sg-surface-card, rgba(255,255,255,0.03));
  border: 1px solid var(--sg-border-soft, rgba(255,255,255,0.06));
  border-radius: 14px;
  padding: 16px;
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.spot-list-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}
.spot-list-header h3 {
  font-size: 14px;
  font-weight: 600;
  color: var(--sg-text-body, rgba(255,255,255,0.88));
  margin: 0;
}
.spot-list-header h3 small { color: var(--sg-text-faint, rgba(255,255,255,0.3)); font-weight: 400; }
.spot-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  overflow-y: auto;
  flex: 1;
}
.spot-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 14px;
  background: none;
  border: 1px solid transparent;
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.15s;
  text-align: left;
}
.spot-item:hover {
  background: var(--sg-surface-hover, rgba(255,255,255,0.03));
  border-color: var(--sg-border-soft, rgba(255,255,255,0.06));
}
.spot-item.active {
  background: var(--sg-jade-bg, rgba(99,226,183,0.06));
  border-color: var(--sg-jade-border, rgba(99,226,183,0.15));
}
.spot-item-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--sg-text-body, rgba(255,255,255,0.8));
}
.spot-item.active .spot-item-name { color: var(--sg-jade-bright, #63e2b7); }
.spot-item-meta {
  font-size: 11px;
  color: var(--sg-text-faint, rgba(255,255,255,0.3));
}

@media (max-width: 1200px) {
  .map-layout { grid-template-columns: 1fr 300px; }
}
@media (max-width: 768px) {
  .map-layout { grid-template-columns: 1fr; }
  .map-sidebar { border-left: none; border-top: 1px solid var(--sg-border-subtle, rgba(255,255,255,0.04)); max-height: 40vh; }
}
</style>
