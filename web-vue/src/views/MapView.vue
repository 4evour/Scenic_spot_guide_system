<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { NInput, NButton, NTag, NEmpty, NSpin, NAlert, useMessage } from 'naive-ui';
import { useGeolocation } from '../composables/useGeolocation';
import { useProximityGuide } from '../composables/useProximityGuide';
import { useSeniorMode } from '../composables/useSeniorMode';
import { AudioPlaybackController } from '../services/audioPlayback';
import { streamTTS } from '../services/ttsApi';
import { apiFetch } from '../services/api';
import {
  SCENIC_ROUTES,
  SCENIC_SPOTS,
  SERVICE_REMINDERS,
  findStructuredSpot,
  type ScenicRoutePlan,
  type ScenicVisualType,
} from '../constants/scenicVisualization';

const { t } = useI18n();

declare const AMap: Record<string, unknown>;

type ScenicSpot = {
  id: string;
  name: string;
  area: string;
  category: string;
  visualType: ScenicVisualType;
  description: string;
  lng: number;
  lat: number;
  rating: number;
  price: number;
  imageUrl: string;
  thumbnail: string;
  parameters: string[];
  culture: string;
  highlights: string[];
  openInfo: string;
  showTimes: string[];
  routeTags: string[];
  geofenceEnabled: boolean;
  geofenceRadiusM: number;
  geofenceIntroText: string;
  geofenceCooldownMinutes: number;
  signalBlindSpot?: boolean;
};

const AMAP_KEY = import.meta.env.VITE_AMAP_KEY || '';
const AMAP_SECURITY = import.meta.env.VITE_AMAP_SECURITY || '';

const fallbackSpots: ScenicSpot[] = SCENIC_SPOTS.map(spot => ({ ...spot }));

function escapeHtml(str: string): string {
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;').replace(/'/g, '&#039;');
}

const categoryTagType: Record<string, 'success' | 'info' | 'warning' | 'error' | 'default'> = {
  '核心景点': 'success',
  '文化建筑': 'warning',
  '演艺体验': 'info',
  '服务设施': 'default',
  '地标建筑': 'warning',
  '文化休憩': 'success',
};

const visualTypeMeta: Record<ScenicVisualType, { label: string; color: string; dimColor: string }> = {
  landmark: { label: '地标建筑', color: '#f4c765', dimColor: 'rgba(244,199,101,0.45)' },
  experience: { label: '体验景点', color: '#52f0ee', dimColor: 'rgba(82,240,238,0.42)' },
  culture: { label: '休憩文化', color: '#63e2b7', dimColor: 'rgba(99,226,183,0.42)' },
};

const state = reactive({
  loading: false,
  error: '',
  spots: [] as ScenicSpot[],
  source: '',
  selectedSpot: null as ScenicSpot | null,
  mapReady: false,
  mapFallback: false,
  search: '',
  activeRouteId: 'family' as ScenicRoutePlan['id'],
});

const filteredSpots = computed(() => {
  const term = state.search.trim().toLowerCase();
  const activeIds = new Set(activeRoute.value.spotIds);
  const base = state.spots.slice().sort((a, b) => Number(!activeIds.has(a.id)) - Number(!activeIds.has(b.id)));
  if (!term) return base;
  return base.filter(s =>
    `${s.name}${s.category}${s.description}`.toLowerCase().includes(term),
  );
});

const activeRoute = computed(() => SCENIC_ROUTES.find(route => route.id === state.activeRouteId) || SCENIC_ROUTES[0]);
const activeRouteSpotIds = computed(() => new Set(activeRoute.value.spotIds));
const activeRouteSpots = computed(() => activeRoute.value.spotIds
  .map(id => state.spots.find(spot => spot.id === id))
  .filter((spot): spot is ScenicSpot => Boolean(spot)));
const activeReminders = computed(() => SERVICE_REMINDERS.filter(item => activeRouteSpotIds.value.has(item.spotId)));
const mapStatusLabel = computed(() => {
  if (state.mapReady) return '地图就绪';
  if (state.mapFallback) return '离线示意图';
  return t('map.loading');
});
const mapStatusClass = computed(() => {
  if (state.mapReady) return 'ready';
  if (state.mapFallback) return 'fallback';
  return 'loading';
});
const arStatusText = computed(() => {
  if (autoGuideEnabled.value && geoError.value) return geoError.value;
  if (autoGuideEnabled.value && !currentPosition.value) return '自动导览已开启，等待浏览器定位授权；未授权时仍可点击景点查看离线路线。';
  if (geoError.value) return 'GPS 信号弱，已切换离线景点列表；后台可标注梵宫等弱信号区域用于现场疏导。';
  if (currentPosition.value) return `AR 指引已就绪，当前位置精度约 ${Math.round(currentPosition.value.accuracy)}m。`;
  return '开启到点讲解后，可基于定位显示方向提示；未授权时保留离线景点选择。';
});
const offlineMapPoints = computed(() => {
  if (state.spots.length === 0) return [];
  const lngs = state.spots.map(spot => spot.lng);
  const lats = state.spots.map(spot => spot.lat);
  const minLng = Math.min(...lngs);
  const maxLng = Math.max(...lngs);
  const minLat = Math.min(...lats);
  const maxLat = Math.max(...lats);
  const lngRange = Math.max(maxLng - minLng, 0.0001);
  const latRange = Math.max(maxLat - minLat, 0.0001);
  const routeIds = activeRouteSpotIds.value;
  return state.spots.map(spot => ({
    spot,
    x: 8 + ((spot.lng - minLng) / lngRange) * 84,
    y: 8 + ((maxLat - spot.lat) / latRange) * 84,
    inRoute: routeIds.has(spot.id),
  }));
});
const offlineRoutePoints = computed(() => activeRoute.value.spotIds
  .map(id => offlineMapPoints.value.find(point => point.spot.id === id))
  .filter((point): point is NonNullable<typeof point> => Boolean(point)));
const offlineRoutePolyline = computed(() => offlineRoutePoints.value
  .map(point => `${point.x},${point.y}`)
  .join(' '));

const mapContainer = ref<HTMLDivElement>();
let map: unknown = null;
let markers: unknown[] = [];
let infoWindow: unknown = null;
let routePolyline: unknown = null;

// === GPS 主动导览 ===
const message = useMessage();
const AUTO_GUIDE_KEY = 'sg_auto_geofence_enabled';
const autoGuideEnabled = ref(localStorage.getItem(AUTO_GUIDE_KEY) === 'true');
const { seniorModeEnabled, ttsRate, toggleSeniorMode } = useSeniorMode();

const {
  currentPosition,
  error: geoError,
  startWatch,
  stopWatch,
} = useGeolocation({
  enableHighAccuracy: true,
  maximumAge: 5000,
  timeout: 10000,
});

const {
  nearbySpot,
  resetTriggered,
  setSpots,
} = useProximityGuide(currentPosition, {
  triggerRadiusM: 100,
});

// 音频播放器（单例，页面生命周期内复用）
const audioPlayer = new AudioPlaybackController({
  onStart: (text) => {
    console.log('[AutoGuide] 开始播放:', text?.substring(0, 50));
  },
  onEnd: () => {
    console.log('[AutoGuide] 播放结束');
  },
});

async function loadSpots() {
  state.loading = true;
  state.error = '';
  try {
    const response = await fetch('/api/v1/spots', { signal: AbortSignal.timeout(15000) });
    const payload = await response.json() as { code?: number; message?: string; data?: Array<Record<string, unknown>> };
    if (!response.ok || payload.code !== 0) throw new Error(payload.message || '加载失败');
    const spots = (payload.data || []).map((raw, i) => enrichSpot(raw, i));
    if (spots.length > 0 && spots.some(s => s.lng > 100)) {
      state.spots = mergeCoreSpots(spots);
      state.source = `${t('map.liveData')} + 结构化导览`;
    } else {
      state.spots = [...fallbackSpots];
      state.source = t('map.demoDataNoCoord');
    }
    // 注入景点坐标到近场检测
    setSpots(
      state.spots.map(s => ({
        id: s.id,
        name: s.name,
        lat: s.lat,
        lng: s.lng,
        triggerEnabled: s.geofenceEnabled,
        triggerRadiusM: s.geofenceRadiusM,
        introText: s.geofenceIntroText,
        cooldownMinutes: s.geofenceCooldownMinutes,
      })),
    );
    state.selectedSpot = activeRouteSpots.value[0] || state.spots[0] || null;
    if (!state.mapReady) state.mapFallback = true;
  } catch {
    state.spots = [...fallbackSpots];
    state.source = t('map.demoData');
    state.selectedSpot = activeRouteSpots.value[0] || state.spots[0] || null;
    if (!state.mapReady) state.mapFallback = true;
  } finally {
    state.loading = false;
  }
}

function enrichSpot(raw: Record<string, unknown>, i: number): ScenicSpot {
  const rawID = String(raw.id || raw.ID || `spot-${i}`);
  const name = String(raw.name || raw.Name || `景点${i + 1}`);
  const structured = findStructuredSpot(rawID) || findStructuredSpot(name);
  const id = structured?.id || rawID;
  return {
    id,
    name,
    area: String(raw.area || structured?.area || '灵山胜境'),
    category: String(raw.category || raw.Category || structured?.category || '文化休憩'),
    visualType: normalizeVisualType(raw.type || raw.visualType || structured?.visualType),
    description: String(raw.description || raw.Description || structured?.description || ''),
    lng: Number(raw.longitude || raw.Longitude || structured?.lng || 120.42 + (i * 0.002)),
    lat: Number(raw.latitude || raw.Latitude || structured?.lat || 31.57 - (i * 0.002)),
    rating: Number(raw.rating || raw.Rating || structured?.rating || 4.5),
    price: Number(raw.price || raw.Price || structured?.price || 0),
    imageUrl: String(raw.image_url || raw.ImageURL || structured?.imageUrl || ''),
    thumbnail: String(raw.thumbnail || structured?.thumbnail || name.charAt(0)),
    parameters: asStringList(raw.parameters || structured?.parameters),
    culture: String(raw.culture || structured?.culture || ''),
    highlights: asStringList(raw.highlights || structured?.highlights),
    openInfo: String(raw.openInfo || raw.open_info || structured?.openInfo || ''),
    showTimes: asStringList(raw.showTimes || raw.show_times || structured?.showTimes),
    routeTags: asStringList(raw.routeTags || raw.route_tags || structured?.routeTags),
    geofenceEnabled: Boolean(raw.geofence_enabled || raw.GeofenceEnabled || structured?.geofenceEnabled),
    geofenceRadiusM: Number(raw.geofence_radius_m || raw.GeofenceRadiusM || structured?.geofenceRadiusM || 100),
    geofenceIntroText: String(raw.geofence_intro_text || raw.GeofenceIntroText || structured?.geofenceIntroText || ''),
    geofenceCooldownMinutes: Number(raw.geofence_cooldown_minutes || raw.GeofenceCooldownMinutes || structured?.geofenceCooldownMinutes || 1440),
    signalBlindSpot: Boolean(raw.signalBlindSpot || raw.signal_blind_spot || structured?.signalBlindSpot),
  };
}

function mergeCoreSpots(spots: ScenicSpot[]): ScenicSpot[] {
  const exists = new Set(spots.flatMap(spot => [spot.id, spot.name]));
  const missing = fallbackSpots.filter(spot => !exists.has(spot.id) && !exists.has(spot.name));
  return [...spots, ...missing];
}

function normalizeVisualType(value: unknown): ScenicVisualType {
  if (value === 'landmark' || value === 'experience' || value === 'culture') return value;
  const text = String(value || '');
  if (text.includes('地标') || text.includes('大佛') || text.includes('梵宫')) return 'landmark';
  if (text.includes('体验') || text.includes('演艺') || text.includes('湖')) return 'experience';
  return 'culture';
}

function asStringList(value: unknown): string[] {
  if (Array.isArray(value)) return value.map(item => String(item)).filter(Boolean);
  if (typeof value === 'string' && value.trim()) return value.split(/[、,，]/).map(item => item.trim()).filter(Boolean);
  return [];
}

function loadAmapScript(): Promise<void> {
  return new Promise((resolve, reject) => {
    if (typeof AMap !== 'undefined') { resolve(); return; }
    if (!AMAP_KEY) { reject(new Error(t('map.configMissing'))); return; }
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    if (AMAP_SECURITY) {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      (window as any)._AMapSecurityConfig = { securityJsCode: AMAP_SECURITY };
    }
    const script = document.createElement('script');
    const timer = window.setTimeout(() => reject(new Error(t('map.amapLoadFailed'))), 8000);
    script.src = `https://webapi.amap.com/maps?v=2.0&key=${AMAP_KEY}&plugin=AMap.Scale,AMap.ToolBar,AMap.Walking,AMap.Geolocation`;
    script.onload = () => {
      window.clearTimeout(timer);
      resolve();
    };
    script.onerror = () => {
      window.clearTimeout(timer);
      reject(new Error(t('map.amapLoadFailed')));
    };
    document.head.appendChild(script);
  });
}

function initMap() {
  if (!mapContainer.value || typeof AMap === 'undefined') return;
  const AMapMap = AMap.Map as new (container: HTMLElement, opts: Record<string, unknown>) => unknown;
  const AMapInfoWindow = AMap.InfoWindow as new (opts: Record<string, unknown>) => unknown;

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

  renderMapOverlays();
  state.mapReady = true;
  state.mapFallback = false;
}

function renderMapOverlays() {
  if (!map || typeof AMap === 'undefined') return;
  const AMapMarker = AMap.Marker as new (opts: Record<string, unknown>) => unknown;
  const AMapPolyline = AMap.Polyline as new (opts: Record<string, unknown>) => unknown;
  if (markers.length || routePolyline) {
    (map as { remove?: (overlays: unknown[]) => void }).remove?.([...markers, routePolyline].filter(Boolean));
  }
  markers = [];
  routePolyline = null;
  const routeIds = activeRouteSpotIds.value;
  for (const spot of state.spots) {
    const meta = visualTypeMeta[spot.visualType] || visualTypeMeta.culture;
    const isInRoute = routeIds.has(spot.id);
    const marker = new AMapMarker({
      position: [spot.lng, spot.lat],
      title: spot.name,
      content: `<div style="
        width: 32px; height: 32px; border-radius: 50%;
        background: ${isInRoute ? meta.color : meta.dimColor};
        border: 2px solid rgba(255,255,255,0.9);
        display: flex; align-items: center; justify-content: center;
        font-size: 13px; color: #0a0a0f; font-weight: bold;
        box-shadow: 0 2px 12px rgba(0,0,0,0.3);
        cursor: pointer; transition: transform 0.2s; opacity:${isInRoute ? 1 : 0.48};
      ">${escapeHtml(spot.name.charAt(0))}</div>`,
      extData: spot,
    });
    (marker as unknown as { on: (event: string, cb: () => void) => void }).on('click', () => showSpotInfo(spot));
    markers.push(marker);
  }

  const path = activeRouteSpots.value.map(spot => [spot.lng, spot.lat]);
  routePolyline = new AMapPolyline({
    path,
    strokeColor: '#52f0ee',
    strokeWeight: 5,
    strokeOpacity: 0.82,
    lineJoin: 'round',
    lineCap: 'round',
    showDir: true,
  });

  (map as { add: (overlays: unknown[]) => void }).add([...markers, routePolyline]);
  (map as { setFitView: () => void }).setFitView();
}

function showSpotInfo(spot: ScenicSpot) {
  state.selectedSpot = spot;
  if (infoWindow && map) {
    const priceText = spot.price > 0 ? `¥${spot.price}` : t('map.free');
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
      <div style="display:grid; gap:6px; font-size:12px; color:rgba(255,255,255,0.68); padding-top:8px; border-top:1px solid rgba(255,255,255,0.06);">
        <span style="color:#f4c765;">${escapeHtml(spot.parameters.join(' / ') || `评分 ${spot.rating}`)}</span>
        <span>${escapeHtml(spot.openInfo || priceText)}</span>
        <span>${escapeHtml(spot.highlights.slice(0, 2).join(' · '))}</span>
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

function switchRoute(routeId: ScenicRoutePlan['id']) {
  state.activeRouteId = routeId;
  if (!state.selectedSpot || !activeRouteSpotIds.value.has(state.selectedSpot.id)) {
    state.selectedSpot = activeRouteSpots.value[0] || state.selectedSpot;
  }
  renderMapOverlays();
}

function locateMe() {
  if (!map || typeof AMap === 'undefined') {
    startWatch();
    message.info('已请求定位权限，离线示意图会保留景点和路线。', { duration: 3000 });
    return;
  }
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

function toggleAutoGuide() {
  autoGuideEnabled.value = !autoGuideEnabled.value;
  localStorage.setItem(AUTO_GUIDE_KEY, String(autoGuideEnabled.value));
  if (autoGuideEnabled.value) {
    resetTriggered();
    setSpots(
      state.spots.map(s => ({
        id: s.id,
        name: s.name,
        lat: s.lat,
        lng: s.lng,
        triggerEnabled: s.geofenceEnabled,
        triggerRadiusM: s.geofenceRadiusM,
        introText: s.geofenceIntroText,
        cooldownMinutes: s.geofenceCooldownMinutes,
      })),
    );
    startWatch();
    message.success('自动导览已开启，靠近景点后会自动讲解。', { duration: 3000 });
  } else {
    stopWatch();
    audioPlayer.interrupt();
    message.info('自动导览已关闭。', { duration: 2500 });
  }
}

function toggleSeniorGuideMode() {
  toggleSeniorMode();
  message.info(seniorModeEnabled.value ? '老年模式已开启，文字和按钮会放大。' : '老年模式已关闭。', { duration: 2500 });
}

// 监听近场检测结果，自动触发讲解
watch(nearbySpot, async (spot) => {
  if (!spot || !autoGuideEnabled.value) return;

  // 1. 弹出通知
  message.info(t('map.autoGuideArrived', { name: spot.name }), { duration: 4000 });

  // 2. 地图定位到该景点
  if (map) {
    (map as { setCenter: (pos: number[]) => void }).setCenter([spot.lng, spot.lat]);
    (map as { setZoom: (z: number) => void }).setZoom(17);
  }

  // 3. 获取讲解内容
  try {
    const contents = await apiFetch<Array<{ id: number; title: string; content: string }>>(
      `/contents/spot/${spot.id}`,
    );

    if (!contents || contents.length === 0) {
      console.warn('[AutoGuide] 该景点无讲解内容:', spot.name);
      return;
    }

    const guideText = spot.introText || contents[0]?.content || '';
    if (!guideText) {
      console.warn('[AutoGuide] 讲解内容为空:', spot.name);
      return;
    }

    // 4. 打断当前播放
    audioPlayer.interrupt();

    // 5. 调用流式 TTS 并播放
    try {
      const ttsResponse = await streamTTS({
        text: guideText,
        voice: 'female_xiaoxiao',
        rate: ttsRate.value,
      });
      const enqueued = await audioPlayer.enqueueStream(ttsResponse, guideText, {});
      if (!enqueued) {
        audioPlayer.playTextFallback(guideText, {});
      }
    } catch (ttsErr) {
      console.warn('[AutoGuide] TTS 失败，回退到浏览器语音合成:', ttsErr);
      audioPlayer.playTextFallback(guideText, {});
    }
  } catch (contentErr) {
    console.error('[AutoGuide] 获取讲解内容失败:', contentErr);
  }
});

onMounted(async () => {
  await loadSpots();
  if (autoGuideEnabled.value) startWatch();
  try {
    await loadAmapScript();
    initMap();
  } catch (e) {
    state.mapFallback = true;
    state.mapReady = false;
    state.error = '';
    state.source = `${state.source || t('map.demoData')} · 离线示意图`;
    console.warn('[Map] AMap unavailable, using offline scenic map.', e);
  }
});

onUnmounted(() => {
  stopWatch();
  audioPlayer.interrupt();
  if (map) {
    (map as { destroy: () => void }).destroy();
    map = null;
  }
});
</script>

<template>
  <main class="map-view" :class="{ 'senior-mode-page': seniorModeEnabled }">
    <header class="map-header">
      <div>
        <h1>{{ $t('map.title') }}</h1>
        <p>{{ $t('map.subtitle') }}</p>
      </div>
      <div class="map-status" :class="mapStatusClass">
        <span class="status-dot"></span>
        {{ mapStatusLabel }}
        <small>{{ state.source }}</small>
      </div>
    </header>

    <section class="map-layout">
      <article class="map-stage">
        <NAlert v-if="state.error" type="error" :show-icon="true" style="margin: 8px 16px;">
          {{ state.error }}
        </NAlert>
        <div class="map-canvas-shell">
          <div ref="mapContainer" class="amap-container" :class="{ hidden: state.mapFallback }"></div>
          <div class="offline-map" :class="{ overlay: state.mapReady && !state.mapFallback }" aria-label="景区路线图">
            <svg viewBox="0 0 100 100" preserveAspectRatio="none" class="offline-map-svg">
              <defs>
                <linearGradient id="routeGlow" x1="0" x2="1" y1="0" y2="1">
                  <stop offset="0%" stop-color="#52f0ee" />
                  <stop offset="100%" stop-color="#63e2b7" />
                </linearGradient>
              </defs>
              <path d="M8 84 C24 64 27 35 48 47 S69 83 92 20" class="offline-river" />
              <polyline
                v-if="offlineRoutePolyline"
                :points="offlineRoutePolyline"
                class="offline-route-line"
              />
            </svg>
            <button
              v-for="point in offlineMapPoints"
              :key="point.spot.id"
              class="offline-spot-marker"
              :class="{ route: point.inRoute, active: state.selectedSpot?.id === point.spot.id }"
              :style="{
                left: `${point.x}%`,
                top: `${point.y}%`,
                '--spot-color': visualTypeMeta[point.spot.visualType].color,
              }"
              @click="flyToSpot(point.spot)"
            >
              <span>{{ point.spot.thumbnail || point.spot.name.charAt(0) }}</span>
              <strong>{{ point.spot.name }}</strong>
            </button>
          </div>
        </div>
        <div class="map-toolbar">
          <NButton size="small" quaternary @click="locateMe">
            {{ $t('map.myLocation') }}
          </NButton>
          <NButton
            size="small"
            :type="autoGuideEnabled ? 'primary' : 'default'"
            @click="toggleAutoGuide"
          >
            {{ autoGuideEnabled ? '⏸ ' + $t('map.autoGuideOff') : '▶ ' + $t('map.autoGuideOn') }}
          </NButton>
          <NButton
            size="small"
            :type="seniorModeEnabled ? 'primary' : 'default'"
            @click="toggleSeniorGuideMode"
          >
            {{ seniorModeEnabled ? '退出老年模式' : '老年模式' }}
          </NButton>
          <span
            v-if="autoGuideEnabled && currentPosition"
            class="gps-indicator"
          >
            {{ $t('map.gpsLabel', { accuracy: Math.round(currentPosition.accuracy) }) }}
          </span>
          <span v-if="autoGuideEnabled && geoError" class="gps-error">
            {{ geoError }}
          </span>
          <NSpin v-if="state.loading" size="small" />
          <span v-if="state.loading" class="loading-text">{{ $t('map.loadingSpots') }}</span>
        </div>
        <div class="ar-guide-panel" :class="{ offline: Boolean(geoError) }">
          <strong>{{ geoError ? '离线导览模式' : 'AR 导航提示' }}</strong>
          <span>{{ arStatusText }}</span>
        </div>
      </article>

      <aside class="map-sidebar">
        <article class="route-card">
          <div class="route-card-header">
            <div>
              <h2>个性化路线</h2>
              <p>{{ activeRoute.duration }} · {{ activeRoute.summary }}</p>
            </div>
          </div>
          <div class="route-tabs">
            <button
              v-for="route in SCENIC_ROUTES"
              :key="route.id"
              :class="{ active: route.id === state.activeRouteId }"
              @click="switchRoute(route.id)"
            >
              {{ route.name }}
            </button>
          </div>
          <ol class="route-nodes">
            <li v-for="spot in activeRouteSpots" :key="spot.id" @click="flyToSpot(spot)">
              <span>{{ spot.name }}</span>
              <small>{{ activeRoute.nodeHighlights[spot.id] }}</small>
            </li>
          </ol>
        </article>

        <article class="reminder-card">
          <h2>服务提醒</h2>
          <div v-for="item in activeReminders" :key="item.title" class="reminder-item" :class="item.priority">
            <strong>{{ item.title }}</strong>
            <span>{{ item.startTime }} · 提前{{ item.advanceMinutes }}分钟</span>
            <p>{{ item.message }}</p>
          </div>
        </article>

        <!-- 选中景点详情 -->
        <article v-if="state.selectedSpot" class="spot-card">
          <div class="spot-card-header">
            <h2>{{ state.selectedSpot.name }}</h2>
            <NTag :type="categoryTagType[state.selectedSpot.category] || 'default'" size="small" :bordered="false">
              {{ state.selectedSpot.category }}
            </NTag>
          </div>
          <p class="spot-desc">{{ state.selectedSpot.description }}</p>
          <div class="spot-section">
            <strong>建筑参数</strong>
            <span>{{ state.selectedSpot.parameters.join(' / ') || '暂无参数' }}</span>
          </div>
          <div class="spot-section">
            <strong>文化内涵</strong>
            <span>{{ state.selectedSpot.culture || '暂无说明' }}</span>
          </div>
          <div class="spot-section">
            <strong>游玩亮点</strong>
            <span>{{ state.selectedSpot.highlights.join(' / ') || '暂无亮点' }}</span>
          </div>
          <div class="spot-section">
            <strong>开放/演出</strong>
            <span>{{ state.selectedSpot.openInfo || state.selectedSpot.showTimes.join(' / ') || '随景区开放' }}</span>
          </div>
          <div class="spot-meta">
            <span class="spot-rating">⭐ {{ state.selectedSpot.rating }}</span>
            <span class="spot-price">{{ state.selectedSpot.price > 0 ? '¥' + state.selectedSpot.price : $t('map.free') }}</span>
            <span v-if="state.selectedSpot.signalBlindSpot" class="spot-signal">弱信号</span>
          </div>
        </article>

        <!-- 搜索 + 景点列表 -->
        <article class="spot-list-card">
          <div class="spot-list-header">
            <h3>{{ $t('map.spots') }} <small>({{ filteredSpots.length }})</small></h3>
          </div>
          <NInput
            v-model:value="state.search"
            :placeholder="$t('map.searchPlaceholder')"
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
              <span class="spot-item-main">
                <span class="spot-type-dot" :style="{ background: visualTypeMeta[spot.visualType].color }"></span>
                <span class="spot-item-name">{{ spot.name }}</span>
              </span>
              <span class="spot-item-meta">{{ spot.area }} · {{ visualTypeMeta[spot.visualType].label }}</span>
            </button>
            <NEmpty v-if="filteredSpots.length === 0" :description="$t('map.noResults')" size="small" />
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
.map-status.fallback .status-dot { background: var(--sg-gold, #f4c765); }

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

.map-canvas-shell {
  position: relative;
  flex: 1;
  min-height: 300px;
  background: #0a1a1e;
  overflow: hidden;
}

.amap-container {
  width: 100%;
  height: 100%;
  min-height: 300px;
}

.amap-container.hidden {
  visibility: hidden;
}

.offline-map {
  position: absolute;
  inset: 0;
  background:
    linear-gradient(rgba(82, 240, 238, 0.035) 1px, transparent 1px),
    linear-gradient(90deg, rgba(82, 240, 238, 0.035) 1px, transparent 1px),
    radial-gradient(circle at 45% 55%, rgba(99, 226, 183, 0.1), transparent 44%),
    #081b1e;
  background-size: 40px 40px, 40px 40px, auto, auto;
}

.offline-map.overlay {
  pointer-events: none;
  background: transparent;
}

.offline-map.overlay::before,
.offline-map.overlay .offline-map-svg {
  display: none;
}

.offline-map.overlay .offline-spot-marker {
  pointer-events: auto;
}

.offline-map::before {
  content: "灵山胜境 / 拈花湾离线导览图";
  position: absolute;
  left: 18px;
  top: 16px;
  z-index: 1;
  color: rgba(255, 255, 255, 0.62);
  font-size: 12px;
}

.offline-map-svg {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
}

.offline-river {
  fill: none;
  stroke: rgba(82, 240, 238, 0.12);
  stroke-width: 6;
  stroke-linecap: round;
}

.offline-route-line {
  fill: none;
  stroke: url(#routeGlow);
  stroke-width: 1.8;
  stroke-linecap: round;
  stroke-linejoin: round;
  filter: drop-shadow(0 0 6px rgba(82, 240, 238, 0.45));
}

.offline-spot-marker {
  position: absolute;
  transform: translate(-50%, -50%);
  display: grid;
  justify-items: center;
  gap: 4px;
  min-width: 64px;
  border: none;
  background: transparent;
  color: rgba(255, 255, 255, 0.72);
  cursor: pointer;
  z-index: 2;
}

.offline-spot-marker span {
  width: 30px;
  height: 30px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  background: color-mix(in srgb, var(--spot-color) 45%, #0a1a1e);
  border: 1px solid rgba(255, 255, 255, 0.42);
  color: #031012;
  font-size: 12px;
  font-weight: 800;
  box-shadow: 0 6px 18px rgba(0, 0, 0, 0.32);
}

.offline-spot-marker strong {
  max-width: 86px;
  padding: 2px 6px;
  border-radius: 999px;
  background: rgba(3, 16, 18, 0.68);
  color: rgba(255, 255, 255, 0.72);
  font-size: 11px;
  font-weight: 600;
  white-space: nowrap;
}

.offline-spot-marker.route span,
.offline-spot-marker.active span {
  background: var(--spot-color);
  border-color: rgba(255, 255, 255, 0.9);
}

.offline-spot-marker.active strong {
  color: #031012;
  background: var(--sg-jade-bright, #63e2b7);
}

.map-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 16px;
  border-top: 1px solid var(--sg-border-subtle, rgba(255,255,255,0.04));
  flex-wrap: wrap;
}
.loading-text { color: var(--sg-text-faint, rgba(255,255,255,0.3)); font-size: 12px; }

.gps-indicator {
  font-size: 11px;
  color: var(--sg-jade-bright, #63e2b7);
  padding: 2px 8px;
  border-radius: 10px;
  background: var(--sg-jade-bg, rgba(99,226,183,0.06));
  border: 1px solid var(--sg-jade-border, rgba(99,226,183,0.15));
}
.gps-error {
  font-size: 11px;
  color: var(--sg-red-bright, #e88080);
  padding: 2px 8px;
  border-radius: 10px;
  background: rgba(232,128,128,0.06);
  border: 1px solid rgba(232,128,128,0.15);
}

.ar-guide-panel {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 16px;
  border-top: 1px solid var(--sg-border-subtle, rgba(255,255,255,0.04));
  background: rgba(82, 240, 238, 0.035);
  color: var(--sg-text-secondary, rgba(255,255,255,0.6));
  font-size: 12px;
}
.ar-guide-panel strong {
  color: var(--sg-cyan, #52f0ee);
  white-space: nowrap;
}
.ar-guide-panel.offline {
  background: rgba(244, 199, 101, 0.055);
}
.ar-guide-panel.offline strong {
  color: var(--sg-gold, #f4c765);
}

.map-sidebar {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px;
  overflow-y: auto;
  border-left: 1px solid var(--sg-border-subtle, rgba(255,255,255,0.04));
  background: rgba(255,255,255,0.01);
}

.route-card,
.reminder-card {
  background: var(--sg-surface-card, rgba(255,255,255,0.03));
  border: 1px solid var(--sg-border-soft, rgba(255,255,255,0.06));
  border-radius: 14px;
  padding: 16px;
}
.route-card-header h2,
.reminder-card h2 {
  font-size: 15px;
  color: var(--sg-text-heading, rgba(255,255,255,0.92));
  margin: 0 0 6px;
}
.route-card-header p {
  font-size: 12px;
  line-height: 1.5;
  color: var(--sg-text-hint, rgba(255,255,255,0.35));
  margin: 0 0 12px;
}
.route-tabs {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 6px;
  margin-bottom: 12px;
}
.route-tabs button {
  min-height: 34px;
  border: 1px solid var(--sg-border-soft, rgba(255,255,255,0.06));
  border-radius: 8px;
  background: rgba(255,255,255,0.025);
  color: var(--sg-text-secondary, rgba(255,255,255,0.6));
  font-size: 12px;
  cursor: pointer;
}
.route-tabs button.active {
  color: #041213;
  background: var(--sg-jade-bright, #63e2b7);
  border-color: var(--sg-jade-bright, #63e2b7);
  font-weight: 700;
}
.route-nodes {
  display: grid;
  gap: 8px;
  margin: 0;
  padding: 0;
  list-style: none;
}
.route-nodes li {
  display: grid;
  gap: 3px;
  padding: 8px 10px;
  border-radius: 8px;
  background: rgba(255,255,255,0.025);
  cursor: pointer;
}
.route-nodes span {
  font-size: 13px;
  color: var(--sg-text-body, rgba(255,255,255,0.88));
  font-weight: 600;
}
.route-nodes small {
  font-size: 11px;
  line-height: 1.4;
  color: var(--sg-text-hint, rgba(255,255,255,0.35));
}
.reminder-card {
  display: grid;
  gap: 8px;
}
.reminder-item {
  border-left: 3px solid var(--sg-cyan, #52f0ee);
  padding: 8px 10px;
  border-radius: 8px;
  background: rgba(255,255,255,0.025);
}
.reminder-item.high { border-left-color: var(--sg-gold, #f4c765); }
.reminder-item.medium { border-left-color: var(--sg-jade-bright, #63e2b7); }
.reminder-item strong {
  display: block;
  font-size: 12px;
  color: var(--sg-text-body, rgba(255,255,255,0.88));
}
.reminder-item span,
.reminder-item p {
  display: block;
  margin: 3px 0 0;
  font-size: 11px;
  line-height: 1.45;
  color: var(--sg-text-hint, rgba(255,255,255,0.35));
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
.spot-section {
  display: grid;
  gap: 4px;
  padding: 8px 0;
  border-top: 1px solid var(--sg-border-subtle, rgba(255,255,255,0.04));
}
.spot-section strong {
  font-size: 11px;
  color: var(--sg-jade-bright, #63e2b7);
}
.spot-section span {
  font-size: 12px;
  line-height: 1.55;
  color: var(--sg-text-secondary, rgba(255,255,255,0.6));
}
.spot-meta {
  display: flex;
  gap: 16px;
  font-size: 13px;
  flex-wrap: wrap;
}
.spot-rating { color: var(--sg-gold, #f4c765); }
.spot-price { color: var(--sg-text-secondary, rgba(255,255,255,0.5)); }
.spot-signal {
  color: var(--sg-gold, #f4c765);
  padding: 0 8px;
  border-radius: 999px;
  background: rgba(244,199,101,0.09);
}

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
.spot-item-main {
  display: inline-flex;
  align-items: center;
  min-width: 0;
  gap: 8px;
}
.spot-type-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex: 0 0 auto;
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
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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
  .map-header { align-items: flex-start; gap: 10px; flex-direction: column; }
  .ar-guide-panel { align-items: flex-start; flex-direction: column; gap: 4px; }
}
</style>
