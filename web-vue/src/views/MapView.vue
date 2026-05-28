<script setup lang="ts">
import { onMounted, onUnmounted, reactive, ref } from 'vue';

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

const AMAP_KEY = 'f12f34639d4cdaba89a67b2b9f28d7ee';
const AMAP_SECURITY = '12f34639d4cdaba89a67b2b9f28d7ee';

// 灵山胜境景点坐标（真实经纬度）
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

const state = reactive({
  loading: false,
  error: '',
  spots: [] as ScenicSpot[],
  source: '演示数据',
  selectedSpot: null as ScenicSpot | null,
  mapReady: false,
});

const mapContainer = ref<HTMLDivElement>();
let map: unknown = null;
let markers: unknown[] = [];
let infoWindow: unknown = null;

async function loadSpots() {
  state.loading = true;
  state.error = '';
  try {
    const response = await fetch('/api/v1/spots');
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
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (window as any)._AMapSecurityConfig = { securityJsCode: AMAP_SECURITY };
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

  // 添加景点标记
  markers = [];
  const path: number[][] = [];
  for (const spot of state.spots) {
    const marker = new AMapMarker({
      position: [spot.lng, spot.lat],
      title: spot.name,
      content: `<div style="
        width: 28px; height: 28px; border-radius: 50%;
        background: ${spot.rating >= 4.8 ? '#f4c765' : '#52f0ee'};
        border: 2px solid rgba(255,255,255,0.8);
        display: flex; align-items: center; justify-content: center;
        font-size: 12px; color: #1a1a2e; font-weight: bold;
        box-shadow: 0 2px 8px rgba(0,0,0,0.3);
        cursor: pointer;
      ">${spot.name.charAt(0)}</div>`,
      extData: spot,
    });
    (marker as unknown as { on: (event: string, cb: () => void) => void }).on('click', () => showSpotInfo(spot));
    markers.push(marker);
    path.push([spot.lng, spot.lat]);
  }

  // 绘制路线
  const polyline = new AMapPolyline({
    path,
    strokeColor: '#52f0ee',
    strokeWeight: 4,
    strokeOpacity: 0.8,
    lineJoin: 'round',
    lineCap: 'round',
  });

  (map as { add: (overlays: unknown[]) => void }).add([...markers, polyline]);
  (map as { setFitView: () => void }).setFitView();

  state.mapReady = true;
}

function showSpotInfo(spot: ScenicSpot) {
  state.selectedSpot = spot;
  if (infoWindow && map) {
    const content = `<div style="
      background: rgba(6,16,18,0.95); color: rgba(255,255,255,0.88);
      padding: 16px; border-radius: 8px; min-width: 200px;
      border: 1px solid rgba(82,240,238,0.2); font-family: 'Microsoft YaHei', sans-serif;
    ">
      <h3 style="margin: 0 0 8px; color: #f4c765; font-size: 15px;">${spot.name}</h3>
      <p style="margin: 0 0 4px; color: rgba(255,255,255,0.6); font-size: 12px;">${spot.category}</p>
      <p style="margin: 0 0 8px; font-size: 13px; line-height: 1.5;">${spot.description}</p>
      <div style="display: flex; gap: 12px; font-size: 12px; color: rgba(255,255,255,0.5);">
        <span>⭐ ${spot.rating}</span>
        ${spot.price > 0 ? `<span>¥${spot.price}</span>` : '<span>免费</span>'}
      </div>
    </div>`;
    (infoWindow as { setContent: (c: string) => void; open: (m: unknown, pos: number[]) => void })
      .setContent(content);
    (infoWindow as { open: (m: unknown, pos: number[]) => void })
      .open(map, [spot.lng, spot.lat]);
  }
}

function locateMe() {
  if (!map) return;
  const Geo = AMap.Geolocation as new (opts: Record<string, unknown>) => unknown;
  const geolocation = new Geo({
    enableHighAccuracy: true,
    timeout: 10000,
  });
  (geolocation as { getCurrentPosition: (cb: (status: string, result: Record<string, unknown>) => void) => void })
    .getCurrentPosition((status: string, result: Record<string, unknown>) => {
      if (status === 'complete' && result.position) {
        const pos = result.position as { lng: number; lat: number };
        (map as { setCenter: (pos: number[]) => void }).setCenter([pos.lng, pos.lng]);
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
    <header class="hero-console">
      <div>
        <p class="eyebrow">Scenic Route Map · 高德地图</p>
        <h1>游客实时导览地图</h1>
        <p>点击查看景点详情，支持定位和步行导航。</p>
      </div>
      <div class="status-badge" :class="state.mapReady ? 'connected' : 'loading'">
        {{ state.mapReady ? '地图已就绪' : '加载中' }} · {{ state.source }}
      </div>
    </header>

    <section class="map-layout">
      <article class="map-stage panel">
        <div v-if="state.error" class="notice error">{{ state.error }}</div>
        <div ref="mapContainer" class="amap-container"></div>
        <div class="map-toolbar">
          <button class="tool-btn" @click="locateMe">📍 我的位置</button>
          <span v-if="state.loading" class="loading-text">加载景点数据...</span>
        </div>
      </article>

      <aside class="map-side">
        <article v-if="state.selectedSpot" class="panel map-card spot-detail">
          <h2>{{ state.selectedSpot.name }}</h2>
          <p class="spot-category">{{ state.selectedSpot.category }}</p>
          <p class="spot-desc">{{ state.selectedSpot.description }}</p>
          <div class="spot-meta">
            <span>⭐ {{ state.selectedSpot.rating }}</span>
            <span>{{ state.selectedSpot.price > 0 ? '¥' + state.selectedSpot.price : '免费' }}</span>
          </div>
        </article>

        <article class="panel map-card">
          <h2>景点列表</h2>
          <button
            v-for="spot in state.spots"
            :key="spot.id"
            class="route-option"
            :class="{ active: state.selectedSpot?.id === spot.id }"
            @click="showSpotInfo(spot)"
          >
            <span>{{ spot.name }}</span>
            <small>{{ spot.category }} · ⭐{{ spot.rating }}</small>
          </button>
        </article>
      </aside>
    </section>
  </main>
</template>

<style scoped>
.amap-container {
  width: 100%;
  height: 500px;
  border-radius: 8px;
  overflow: hidden;
  background: #0a1a1e;
}

.map-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 0;
}

.tool-btn {
  padding: 6px 14px;
  background: rgba(82, 240, 238, 0.1);
  border: 1px solid rgba(82, 240, 238, 0.3);
  border-radius: 6px;
  color: #52f0ee;
  font-size: 13px;
  cursor: pointer;
  transition: background 0.2s;
}
.tool-btn:hover {
  background: rgba(82, 240, 238, 0.2);
}

.loading-text {
  color: rgba(255, 255, 255, 0.4);
  font-size: 12px;
}

.spot-detail h2 {
  color: #f4c765;
  margin: 0 0 4px;
}
.spot-category {
  color: rgba(255, 255, 255, 0.5);
  font-size: 12px;
  margin: 0 0 8px;
}
.spot-desc {
  font-size: 13px;
  line-height: 1.6;
  color: rgba(255, 255, 255, 0.75);
  margin: 0 0 8px;
}
.spot-meta {
  display: flex;
  gap: 16px;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.5);
}
</style>
